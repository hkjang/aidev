package server

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
)

//go:embed knowledge_evidence.sql
var knowledgeEvidenceSchema string

func (s *Server) migrateKnowledgeEvidence(ctx context.Context) error {
	_, err := s.DB.Exec(ctx, knowledgeEvidenceSchema)
	return err
}
func (s *Server) registerKnowledgeEvidence() {
	s.admin("GET /api/v1/admin/evidence-policy", s.getEvidencePolicy)
	s.admin("PUT /api/v1/admin/evidence-policy", s.putEvidencePolicy)
	s.admin("GET /api/v1/admin/evidence-policy/history", func(w http.ResponseWriter, r *http.Request) {
		v, err := s.rows(r.Context(), `SELECT to_jsonb(h)-'actor_id' FROM knowledge_evidence_policy_history h ORDER BY version DESC LIMIT 100`)
		respond(w, v, err)
	})
	s.handle("GET /api/v1/ai/evidence/policy", s.getEvidencePolicy)
	s.handle("POST /api/v1/ai/evidence", s.createEvidence)
	s.handle("GET /api/v1/ai/evidence", s.listEvidence)
	s.handle("GET /api/v1/ai/evidence/{id}", s.getEvidence)
	s.handle("POST /api/v1/ai/evidence/{id}/reviews", s.reviewEvidence)
	s.handle("DELETE /api/v1/ai/evidence/{id}", s.deleteEvidence)
}

type evidenceQuote struct {
	aiSource
	Text string `json:"text"`
}
type evidencePayload struct {
	Format              string          `json:"format"`
	Question            string          `json:"question"`
	Answer              string          `json:"answer"`
	Model               string          `json:"model"`
	MaxTokens           int             `json:"max_tokens"`
	SettingsFingerprint string          `json:"settings_fingerprint"`
	Action              string          `json:"action"`
	Quotes              []evidenceQuote `json:"quotes"`
}

func (s *Server) getEvidencePolicy(w http.ResponseWriter, r *http.Request) {
	v, err := s.one(r.Context(), `SELECT to_jsonb(p) FROM knowledge_evidence_policy p WHERE id=1`)
	respond(w, v, err)
}
func (s *Server) putEvidencePolicy(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Enabled       *bool `json:"enabled"`
		RetentionDays int   `json:"retention_days"`
		Version       int   `json:"version"`
	}
	if decode(r, &in) != nil || in.Enabled == nil || in.RetentionDays < 1 || in.RetentionDays > 3650 || in.Version < 1 || in.Version > 2147483646 {
		apiError(w, 400, "활성 여부·보존기간(1~3650일)·현재 설정 버전을 확인하세요")
		return
	}
	tx, err := s.DB.Begin(r.Context())
	if err != nil {
		respond(w, nil, err)
		return
	}
	defer tx.Rollback(r.Context())
	result, err := tx.Exec(r.Context(), `UPDATE knowledge_evidence_policy SET enabled=$1,retention_days=$2,version=version+1,updated_at=now() WHERE id=1 AND version=$3`, *in.Enabled, in.RetentionDays, in.Version)
	if err != nil {
		respond(w, nil, err)
		return
	}
	if result.RowsAffected() != 1 {
		apiError(w, 409, "보관 정책이 변경되었습니다. 현재 설정을 다시 확인하세요")
		return
	}
	// Increasing retention never resurrects an expired snapshot. Shortening it
	// applies to existing copies; legal hold prevents physical deletion, not access.
	_, err = tx.Exec(r.Context(), `UPDATE knowledge_evidence SET expires_at=LEAST(expires_at,created_at+make_interval(days=>$1))`, in.RetentionDays)
	if err == nil {
		_, err = tx.Exec(r.Context(), `INSERT INTO knowledge_evidence_policy_history(version,actor_id,enabled,retention_days) VALUES($1,$2,$3,$4)`, in.Version+1, current(r).ID, *in.Enabled, in.RetentionDays)
	}
	if err == nil {
		err = tx.Commit(r.Context())
	}
	if err != nil {
		respond(w, nil, err)
		return
	}
	s.audit(r, "EVIDENCE_POLICY_UPDATE", "", map[string]any{"enabled": *in.Enabled, "retention_days": in.RetentionDays})
	s.getEvidencePolicy(w, r)
}

func (s *Server) createEvidence(w http.ResponseWriter, r *http.Request) {
	p := current(r)
	var in struct {
		Ticket  string `json:"ticket"`
		Consent bool   `json:"consent"`
	}
	if !personalAIHistory(p) {
		apiError(w, 403, "개인 브라우저 세션에서만 AI 근거를 보관할 수 있습니다")
		return
	}
	if decode(r, &in) != nil || !in.Consent {
		apiError(w, 400, "답변과 당시 원문 구간의 별도 보관에 동의하세요")
		return
	}
	t, err := s.openAIHistory(in.Ticket, p)
	if err != nil {
		apiError(w, 400, err.Error())
		return
	}
	if len(t.Sources) == 0 {
		apiError(w, 422, "문서 출처가 있는 완료된 답변만 보관할 수 있습니다")
		return
	}
	cfg, err := s.effectiveSettings(r.Context(), t.WorkspaceID)
	if err != nil || aiHistoryProvider(cfg) != t.Provider || s.validateAIStream(r, p, t.WorkspaceID, t.Sources, cfg) != nil {
		apiError(w, 409, "근거·권한·공급자 설정이 변경되었습니다. 현재 자료로 답변을 다시 생성하세요")
		return
	}
	ctx := r.Context()
	tx, err := s.DB.Begin(ctx)
	if err != nil {
		respond(w, nil, err)
		return
	}
	defer tx.Rollback(ctx)
	var enabled bool
	var days int
	if err = tx.QueryRow(ctx, `SELECT enabled,retention_days FROM knowledge_evidence_policy WHERE id=1 FOR SHARE`).Scan(&enabled, &days); err != nil {
		respond(w, nil, err)
		return
	}
	if !enabled {
		apiError(w, 403, "관리자가 AI 근거 보관 정책을 활성화해야 합니다")
		return
	}
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,193))`, t.ID); err != nil {
		respond(w, nil, err)
		return
	}
	var exists bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM knowledge_evidence WHERE id=$1 AND owner_id=$2)`, t.ID, p.ID).Scan(&exists); err != nil {
		respond(w, nil, err)
		return
	}
	if exists {
		jsonResponse(w, 200, map[string]any{"id": t.ID, "replayed": true})
		return
	}
	payload := evidencePayload{Format: "madi-evidence-v1", Question: t.Question, Answer: t.Answer, Model: t.Model, MaxTokens: number(cfg, "ai_max_tokens", 4096), SettingsFingerprint: t.Provider, Action: t.Action, Quotes: []evidenceQuote{}}
	// Deterministic row-lock ordering matches other document mutation paths.
	ordered := append([]aiSource{}, t.Sources...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].ID < ordered[j].ID })
	quotes := map[string]evidenceQuote{}
	for _, src := range ordered {
		if src.StartByte < 0 || src.EndByte <= src.StartByte || src.EndByte-src.StartByte > 8192 || len(src.ContentHash) != 64 {
			apiError(w, 409, "검증할 수 없는 근거 구간입니다")
			return
		}
		var version int
		var fragment []byte
		var title string
		if src.AttachmentID != "" {
			var value attachmentCitationCurrent
			value, err = s.attachmentCitationTx(ctx, tx, p, src, true)
			version, title, fragment = value.Version, value.Title, []byte(value.Text)
		} else {
			err = tx.QueryRow(ctx, `SELECT d.version,d.title,substring(convert_to(d.markdown,'UTF8') from $4 for $5) FROM documents d WHERE d.id=$1 AND d.workspace_id=$2 AND d.deleted_at IS NULL AND madi_document_allowed($3,d.id,false) FOR SHARE`, src.ID, t.WorkspaceID, p.ID, src.StartByte+1, src.EndByte-src.StartByte).Scan(&version, &title, &fragment)
		}
		if err != nil || version != src.Version || !utf8.Valid(fragment) || digest(string(fragment)) != src.ContentHash || ragActorTx(ctx, tx, p, src.ID, t.WorkspaceID, false) != nil {
			apiError(w, 409, "근거 원문 또는 현재 접근 권한이 변경되었습니다")
			return
		}
		if src.RAGGrantID != "" {
			g, e := ragGrantTx(ctx, tx, src.ID, true)
			if e != nil || !g.Active || g.ID != src.RAGGrantID || g.Revision != src.RAGGrantRevision || g.Version != src.Version || g.Provider != ragProviderFingerprint(cfg) {
				apiError(w, 409, "AI 색인 동의가 변경되었습니다")
				return
			}
			actor := &Principal{ID: g.ActorID, TokenID: g.TokenID, WorkspaceID: g.WorkspaceID, PluginID: g.Constraints.PluginID, ScopeRestricted: g.Constraints.Restricted || g.TokenID != "", Scopes: g.Constraints.Scopes}
			if ragActorTx(ctx, tx, actor, src.ID, t.WorkspaceID, true) != nil {
				apiError(w, 409, "색인 실행자의 현재 권한을 확인하세요")
				return
			}
		}
		src.Title = title
		quotes[src.CitationID] = evidenceQuote{src, string(fragment)}
	}
	for _, src := range t.Sources {
		payload.Quotes = append(payload.Quotes, quotes[src.CitationID])
	}
	fresh, err := s.ragSettingsTx(ctx, tx, t.WorkspaceID)
	if err != nil || aiHistoryProvider(fresh) != t.Provider {
		apiError(w, 409, "AI 설정이 변경되었습니다")
		return
	}
	if err = s.knowledgeActorTx(r, tx, t.WorkspaceID, "document:read", "ai:execute"); err != nil {
		apiError(w, 403, "현재 로그인 세션이 필요합니다")
		return
	}
	if err = s.checkEvidenceProtection(ctx, tx, p, t.WorkspaceID, payload); err != nil {
		apiError(w, 422, err.Error())
		return
	}
	plain := string(jsonValue(payload))
	if len(plain) > 2<<20 {
		apiError(w, 422, "근거 보관은 답변과 원문 합계 2MiB 이하여야 합니다")
		return
	}
	sealed, err := s.encrypt(plain)
	if err == nil {
		_, err = tx.Exec(ctx, `INSERT INTO knowledge_evidence(id,owner_id,workspace_id,source_refs,ciphertext,payload_hash,expires_at) VALUES($1,$2,$3,$4,$5,$6,now()+make_interval(days=>$7))`, t.ID, p.ID, t.WorkspaceID, jsonValue(t.Sources), sealed, digest(plain), days)
	}
	if err == nil {
		err = tx.Commit(ctx)
	}
	if err != nil {
		respond(w, nil, err)
		return
	}
	s.audit(r, "EVIDENCE_CREATE", t.ID, map[string]any{"source_count": len(t.Sources)})
	jsonResponse(w, 201, map[string]any{"id": t.ID, "version": 1})
}

func (s *Server) checkEvidenceProtection(ctx context.Context, tx pgx.Tx, p *Principal, wid string, value any) error {
	check, err := s.ProtectDocumentMetadataTx(ctx, tx, p, "", wid, value)
	if err != nil || check.Changed {
		return errors.New("현재 민감정보 정책에서 이 근거 사본을 보관하거나 표시할 수 없습니다. 원문을 수정해 새 답변을 생성하세요")
	}
	return nil
}

func (s *Server) listEvidence(w http.ResponseWriter, r *http.Request) {
	p := current(r)
	wid := r.URL.Query().Get("workspace_id")
	if !personalAIHistory(p) || !s.canWorkspace(r.Context(), p, wid, false) {
		apiError(w, 403, "개인 근거 보관함 접근 권한이 없습니다")
		return
	}
	items, err := s.rows(r.Context(), `SELECT jsonb_build_object('id',e.id,'created_at',e.created_at,'expires_at',e.expires_at,'review_status',e.review_status,'version',e.version,'source_count',jsonb_array_length(e.source_refs)) FROM knowledge_evidence e WHERE e.owner_id=$1 AND e.workspace_id=$2 AND madi_evidence_allowed($1,e.id) ORDER BY e.created_at DESC,e.id LIMIT 100`, p.ID, wid)
	respond(w, items, err)
}

func (s *Server) readEvidence(r *http.Request, tx pgx.Tx) (map[string]any, error) {
	p := current(r)
	id := r.PathValue("id")
	if !personalAIHistory(p) || !validID(id) {
		return nil, pgx.ErrNoRows
	}
	var sealed, hash, wid, status string
	var revision int
	var created, expires time.Time
	err := tx.QueryRow(r.Context(), `SELECT ciphertext,payload_hash,workspace_id::text,review_status,version,created_at,expires_at FROM knowledge_evidence WHERE id=$1 AND owner_id=$2 AND madi_evidence_allowed($2,id) FOR SHARE`, id, p.ID).Scan(&sealed, &hash, &wid, &status, &revision, &created, &expires)
	if err != nil {
		return nil, err
	}
	plain, err := s.decrypt(sealed)
	var payload evidencePayload
	if err != nil || digest(plain) != hash || json.Unmarshal([]byte(plain), &payload) != nil || payload.Format != "madi-evidence-v1" {
		return nil, errors.New("근거 사본의 무결성을 확인할 수 없습니다")
	}
	if err = s.checkEvidenceProtection(r.Context(), tx, p, wid, payload); err != nil {
		return nil, err
	}
	sources := []map[string]any{}
	for _, quote := range payload.Quotes {
		var title, body string
		var version int
		if err = tx.QueryRow(r.Context(), `SELECT title,markdown,version FROM documents WHERE id=$1 AND workspace_id=$2 AND deleted_at IS NULL AND madi_document_allowed($3,id,false) FOR SHARE`, quote.ID, wid, p.ID).Scan(&title, &body, &version); err != nil {
			return nil, pgx.ErrNoRows
		}
		integrity := quote.EndByte-quote.StartByte == len(quote.Text) && digest(quote.Text) == quote.ContentHash
		if !integrity {
			return nil, errors.New("보관한 인용 구간의 무결성이 일치하지 않습니다")
		}
		currentText := ""
		// Return the same coordinates for comparison, never label them as a
		// matching quote solely because the document revision still exists.
		if quote.StartByte >= 0 && quote.EndByte <= len(body) && utf8.ValidString(body[quote.StartByte:quote.EndByte]) {
			currentText = body[quote.StartByte:quote.EndByte]
		}
		freshness := "changed"
		if version == quote.Version {
			freshness = "current"
		}
		if quote.AttachmentID != "" {
			value, e := s.attachmentCitationTx(r.Context(), tx, p, quote.aiSource, false)
			if e != nil {
				return nil, pgx.ErrNoRows
			}
			title, version, currentText = value.Title, value.Version, value.Text
			freshness = "changed"
			if value.Fresh {
				freshness = "current"
			}
		}
		if err = s.checkEvidenceProtection(r.Context(), tx, p, wid, map[string]any{"title": title, "text": currentText}); err != nil {
			return nil, err
		}
		sources = append(sources, map[string]any{"source": quote.aiSource, "text": quote.Text, "integrity": "match", "freshness": freshness, "current_version": version, "current_title": title, "current_text": currentText, "same_span": digest(currentText) == quote.ContentHash})
	}
	rows, err := tx.Query(r.Context(), `SELECT status,note_ciphertext,created_at,version FROM knowledge_evidence_reviews WHERE evidence_id=$1 ORDER BY version DESC LIMIT 100`, id)
	if err != nil {
		return nil, err
	}
	reviews := []map[string]any{}
	for rows.Next() {
		var state, note string
		var at time.Time
		var v int
		if err = rows.Scan(&state, &note, &at, &v); err != nil {
			break
		}
		var text string
		text, err = s.decrypt(note)
		if err != nil {
			break
		}
		reviews = append(reviews, map[string]any{"status": state, "note": text, "created_at": at, "version": v})
	}
	if err == nil {
		err = rows.Err()
	}
	rows.Close()
	if err != nil {
		return nil, err
	}
	if err = s.checkEvidenceProtection(r.Context(), tx, p, wid, reviews); err != nil {
		return nil, err
	}
	if err = s.knowledgeActorTx(r, tx, wid, "document:read"); err != nil {
		return nil, err
	}
	if !expires.After(time.Now()) {
		return nil, pgx.ErrNoRows
	}
	return map[string]any{"id": id, "workspace_id": wid, "version": revision, "created_at": created, "expires_at": expires, "question": payload.Question, "answer": payload.Answer, "model": payload.Model, "max_tokens": payload.MaxTokens, "settings_fingerprint": payload.SettingsFingerprint, "sources": sources, "review_status": status, "reviews": reviews, "notice": "무결성은 당시 보관 원문과의 일치입니다. 최신성·AI 답변의 사실성·주장과 근거의 연결은 별도로 검토해야 합니다."}, nil
}

func (s *Server) getEvidence(w http.ResponseWriter, r *http.Request) {
	tx, err := s.DB.Begin(r.Context())
	if err != nil {
		respond(w, nil, err)
		return
	}
	defer tx.Rollback(r.Context())
	v, err := s.readEvidence(r, tx)
	if errors.Is(err, pgx.ErrNoRows) {
		apiError(w, 404, "현재 접근할 수 있는 근거 기록이 없습니다")
		return
	}
	if err != nil {
		apiError(w, 409, err.Error())
		return
	}
	respond(w, v, nil)
}

func (s *Server) reviewEvidence(w http.ResponseWriter, r *http.Request) {
	if !personalAIHistory(current(r)) || !validID(r.PathValue("id")) {
		apiError(w, 404, "현재 접근할 수 있는 근거 기록이 없습니다")
		return
	}
	var in struct {
		Version int    `json:"version"`
		Status  string `json:"status"`
		Note    string `json:"note"`
	}
	if decode(r, &in) != nil || in.Version < 1 || !oneOf(in.Status, "unreviewed", "supported", "insufficient", "misinterpreted") || len(in.Note) > 8000 {
		apiError(w, 400, "검토 상태·현재 버전·의견(8KiB 이하)을 확인하세요")
		return
	}
	tx, err := s.DB.Begin(r.Context())
	if err != nil {
		respond(w, nil, err)
		return
	}
	defer tx.Rollback(r.Context())
	// Lock before readEvidence's shared lock to avoid two concurrent reviews
	// upgrading a shared lock and deadlocking.
	if _, err = tx.Exec(r.Context(), `SELECT id FROM knowledge_evidence WHERE id=$1 AND owner_id=$2 FOR UPDATE`, r.PathValue("id"), current(r).ID); err != nil {
		respond(w, nil, err)
		return
	}
	v, err := s.readEvidence(r, tx)
	if err != nil {
		apiError(w, 404, "현재 접근할 수 있는 근거 기록이 없습니다")
		return
	}
	if number(v, "version", 0) != in.Version {
		apiError(w, 409, "다른 창에서 검토가 변경되었습니다")
		return
	}
	if err = s.checkEvidenceProtection(r.Context(), tx, current(r), str(v, "workspace_id"), in.Note); err != nil {
		apiError(w, 422, err.Error())
		return
	}
	sealed, err := s.encrypt(in.Note)
	if err == nil {
		_, err = tx.Exec(r.Context(), `UPDATE knowledge_evidence SET version=version+1,review_status=$2 WHERE id=$1`, r.PathValue("id"), in.Status)
	}
	if err == nil {
		_, err = tx.Exec(r.Context(), `INSERT INTO knowledge_evidence_reviews(id,evidence_id,actor_id,version,status,note_ciphertext) VALUES($1,$2,$3,$4,$5,$6)`, newID(), r.PathValue("id"), current(r).ID, in.Version+1, in.Status, sealed)
	}
	if err == nil {
		err = tx.Commit(r.Context())
	}
	if err != nil {
		respond(w, nil, err)
		return
	}
	s.audit(r, "EVIDENCE_REVIEW", r.PathValue("id"), map[string]any{"status": in.Status})
	jsonResponse(w, 200, map[string]any{"version": in.Version + 1, "review_status": in.Status})
}

func (s *Server) deleteEvidence(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Confirmation string `json:"confirmation"`
		Version      int    `json:"version"`
	}
	if !personalAIHistory(current(r)) || !validID(r.PathValue("id")) {
		apiError(w, 403, "본인의 개인 근거만 삭제할 수 있습니다")
		return
	}
	if decode(r, &in) != nil || in.Confirmation != "DELETE" || in.Version < 1 {
		apiError(w, 400, "현재 버전과 삭제 확인이 필요합니다")
		return
	}
	result, err := s.DB.Exec(r.Context(), `DELETE FROM knowledge_evidence WHERE id=$1 AND owner_id=$2 AND version=$3 AND NOT madi_evidence_held(id)`, r.PathValue("id"), current(r).ID, in.Version)
	if err != nil {
		respond(w, nil, err)
		return
	}
	if result.RowsAffected() != 1 {
		apiError(w, 409, "기록이 변경·삭제되었거나 원문 보존 정책으로 삭제할 수 없습니다")
		return
	}
	s.audit(r, "EVIDENCE_DELETE", r.PathValue("id"), nil)
	jsonResponse(w, 200, map[string]any{"deleted": true})
}

func (s *Server) purgeEvidence(ctx context.Context) error {
	_, err := s.DB.Exec(ctx, `DELETE FROM knowledge_evidence WHERE id IN (SELECT id FROM knowledge_evidence WHERE expires_at<=now() AND NOT madi_evidence_held(id) ORDER BY expires_at LIMIT 500)`)
	if err != nil {
		return fmt.Errorf("purge evidence: %w", err)
	}
	return nil
}
