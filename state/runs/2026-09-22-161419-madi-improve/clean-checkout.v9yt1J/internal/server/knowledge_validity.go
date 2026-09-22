package server

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

//go:embed knowledge_validity.sql
var knowledgeValiditySchema string

func (s *Server) migrateKnowledgeValidity(ctx context.Context) error {
	_, err := s.DB.Exec(ctx, knowledgeValiditySchema)
	return err
}
func (s *Server) registerKnowledgeValidity() {
	s.handle("GET /api/v1/documents/{id}/validity", s.getKnowledgeValidity)
	s.handle("PUT /api/v1/documents/{id}/validity", s.putKnowledgeValidity)
	s.handle("GET /api/v1/documents/{id}/validity/check", s.checkKnowledgeValidity)
	s.handle("GET /api/v1/documents/{id}/valid-at", s.getKnowledgeAtDate)
	s.handle("GET /api/v1/knowledge/time-search", s.searchKnowledgeAtDate)
	s.handle("POST /api/v1/knowledge/time-check", s.checkKnowledgeTimeResults)
}

type knowledgeValidPeriod struct {
	Version int    `json:"version"`
	From    string `json:"from"`
	Until   string `json:"until"`
}

const validityNotice = "업무 유효기간은 작성자가 명시한 날짜 범위이며 사실성·법적 효력·게시 승인 인증이 아닙니다. 시작일 포함·종료일 제외이며 종료일 공란은 무기한입니다. 수정 시각과 별개이고 미등록 기간은 유효하다고 추정하지 않습니다."

func validKnowledgeDate(value string) bool {
	if len(value) != 10 || value < "0001-01-01" || value > "9999-12-31" {
		return false
	}
	t, err := time.Parse("2006-01-02", value)
	return err == nil && t.Format("2006-01-02") == value
}
func normalizeValidPeriods(in []knowledgeValidPeriod) ([]knowledgeValidPeriod, error) {
	if len(in) > 100 {
		return nil, errors.New("문서별 유효기간은 최대 100개입니다")
	}
	out := append([]knowledgeValidPeriod{}, in...)
	sort.Slice(out, func(i, j int) bool { return out[i].From < out[j].From })
	for i, p := range out {
		if p.Version < 1 || p.Version > 2147483647 || !validKnowledgeDate(p.From) || p.Until != "" && (!validKnowledgeDate(p.Until) || p.Until <= p.From) {
			return nil, errors.New("실제 날짜와 문서 버전을 확인하세요. 종료일은 시작일 이후입니다")
		}
		if i > 0 && (out[i-1].Until == "" || out[i-1].Until > p.From) {
			return nil, errors.New("문서의 유효기간은 서로 겹칠 수 없습니다")
		}
	}
	return out, nil
}

// A publication label is not inferred from a historical revision: versions
// intentionally preserve Markdown, not a retroactive approval assertion.
func (s *Server) validityDocumentTx(r *http.Request, tx pgx.Tx, write bool) (string, int, error) {
	if !validID(r.PathValue("id")) {
		return "", 0, pgx.ErrNoRows
	}
	lock := " FOR SHARE"
	if write {
		lock = " FOR UPDATE"
	}
	var wid string
	var version int
	err := tx.QueryRow(r.Context(), `SELECT workspace_id::text,version FROM documents WHERE id=$1 AND deleted_at IS NULL AND madi_document_allowed($2,id,$3) AND ($4='' OR workspace_id::text=$4)`+lock, r.PathValue("id"), current(r).ID, write, current(r).WorkspaceID).Scan(&wid, &version)
	if err != nil {
		return "", 0, err
	}
	scopes := []string{"document:read"}
	if write {
		scopes = append(scopes, "document:write")
	}
	if err = s.knowledgeActorTx(r, tx, wid, scopes...); err != nil {
		return "", 0, err
	}
	return wid, version, nil
}

func (s *Server) putKnowledgeValidity(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Revision        int                    `json:"revision"`
		DocumentVersion int                    `json:"document_version"`
		Periods         []knowledgeValidPeriod `json:"periods"`
		Reason          string                 `json:"reason"`
		Consent         bool                   `json:"consent"`
	}
	if decode(r, &in) != nil || in.Revision < 0 || in.Revision > 2147483646 || in.DocumentVersion < 1 || in.DocumentVersion > 2147483647 || !in.Consent || len(strings.TrimSpace(in.Reason)) == 0 || len(in.Reason) > 4000 {
		apiError(w, 400, "현재 문서·기간 revision, 변경 이유와 열람자 공유 확인이 필요합니다")
		return
	}
	periods, err := normalizeValidPeriods(in.Periods)
	if err != nil {
		apiError(w, 400, err.Error())
		return
	}
	tx, err := s.DB.Begin(r.Context())
	if err != nil {
		respond(w, nil, err)
		return
	}
	defer tx.Rollback(r.Context())
	wid, version, err := s.validityDocumentTx(r, tx, true)
	if err != nil {
		apiError(w, 404, "현재 수정 가능한 문서가 없습니다")
		return
	}
	if version != in.DocumentVersion {
		apiError(w, 409, "문서가 변경되었습니다. 현재 원문과 유효기간을 다시 확인하세요")
		return
	}
	if _, err = tx.Exec(r.Context(), `INSERT INTO knowledge_validity(document_id) VALUES($1) ON CONFLICT DO NOTHING`, r.PathValue("id")); err != nil {
		respond(w, nil, err)
		return
	}
	var revision int
	if err = tx.QueryRow(r.Context(), `SELECT revision FROM knowledge_validity WHERE document_id=$1 FOR UPDATE`, r.PathValue("id")).Scan(&revision); err != nil {
		respond(w, nil, err)
		return
	}
	if revision != in.Revision {
		apiError(w, 409, "유효기간 등록 내용이 변경되었습니다. 다시 확인하세요")
		return
	}
	for _, p := range periods {
		var exists bool
		if err = tx.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM document_versions WHERE document_id=$1 AND version=$2)`, r.PathValue("id"), p.Version).Scan(&exists); err != nil {
			respond(w, nil, err)
			return
		}
		if !exists {
			apiError(w, 400, "선택한 문서 버전이 없습니다")
			return
		}
	}
	if err = s.checkEvidenceProtection(r.Context(), tx, current(r), wid, in.Reason); err != nil {
		apiError(w, 422, "변경 이유를 현재 정보 보호 정책에 맞게 수정하세요")
		return
	}
	sealed, err := s.encrypt(in.Reason)
	if err != nil {
		respond(w, nil, err)
		return
	}
	_, err = tx.Exec(r.Context(), `DELETE FROM knowledge_validity_periods WHERE document_id=$1`, r.PathValue("id"))
	for i, p := range periods {
		if err != nil {
			break
		}
		_, err = tx.Exec(r.Context(), `INSERT INTO knowledge_validity_periods(document_id,ordinal,document_version,valid_from,valid_until) VALUES($1,$2,$3,$4::text::date,NULLIF($5,'')::date)`, r.PathValue("id"), i, p.Version, p.From, p.Until)
	}
	if err == nil {
		_, err = tx.Exec(r.Context(), `UPDATE knowledge_validity SET revision=revision+1,updated_by=$2,updated_at=now() WHERE document_id=$1`, r.PathValue("id"), current(r).ID)
	}
	if err == nil {
		_, err = tx.Exec(r.Context(), `INSERT INTO knowledge_validity_history(document_id,revision,document_revision,periods,actor_id,reason_ciphertext) VALUES($1,$2,$3,$4,$5,$6)`, r.PathValue("id"), revision+1, version, jsonValue(periods), current(r).ID, sealed)
	}
	if err == nil {
		err = tx.Commit(r.Context())
	}
	if err != nil {
		respond(w, nil, err)
		return
	}
	s.audit(r, "DOCUMENT_VALIDITY_UPDATE", r.PathValue("id"), map[string]any{"revision": revision + 1, "period_count": len(periods)})
	jsonResponse(w, 200, map[string]any{"revision": revision + 1, "periods": periods, "notice": validityNotice})
}

func (s *Server) getKnowledgeValidity(w http.ResponseWriter, r *http.Request) {
	tx, err := s.DB.Begin(r.Context())
	if err != nil {
		respond(w, nil, err)
		return
	}
	defer tx.Rollback(r.Context())
	wid, version, err := s.validityDocumentTx(r, tx, false)
	if err != nil {
		apiError(w, 404, "현재 접근 가능한 문서가 없습니다")
		return
	}
	var revision int
	var updated any
	err = tx.QueryRow(r.Context(), `SELECT revision,updated_at FROM knowledge_validity WHERE document_id=$1`, r.PathValue("id")).Scan(&revision, &updated)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		respond(w, nil, err)
		return
	}
	periods := []knowledgeValidPeriod{}
	rows, err := tx.Query(r.Context(), `SELECT document_version,valid_from::text,coalesce(valid_until::text,'') FROM knowledge_validity_periods WHERE document_id=$1 ORDER BY ordinal`, r.PathValue("id"))
	if err != nil {
		respond(w, nil, err)
		return
	}
	for rows.Next() {
		var p knowledgeValidPeriod
		if err = rows.Scan(&p.Version, &p.From, &p.Until); err != nil {
			break
		}
		periods = append(periods, p)
	}
	if err == nil {
		err = rows.Err()
	}
	rows.Close()
	if err != nil {
		respond(w, nil, err)
		return
	}
	history := []map[string]any{}
	rows, err = tx.Query(r.Context(), `SELECT revision,document_revision,periods,reason_ciphertext,recorded_at FROM knowledge_validity_history WHERE document_id=$1 ORDER BY revision DESC LIMIT 50`, r.PathValue("id"))
	if err != nil {
		respond(w, nil, err)
		return
	}
	for rows.Next() {
		var rev, docRev int
		var body []byte
		var sealed string
		var recorded time.Time
		if err = rows.Scan(&rev, &docRev, &body, &sealed, &recorded); err != nil {
			break
		}
		var ps []knowledgeValidPeriod
		var reason string
		if err = json.Unmarshal(body, &ps); err != nil {
			break
		}
		if reason, err = s.decrypt(sealed); err != nil {
			break
		}
		history = append(history, map[string]any{"revision": rev, "document_version": docRev, "periods": ps, "reason": reason, "recorded_at": recorded})
	}
	if err == nil {
		err = rows.Err()
	}
	rows.Close()
	if err != nil {
		respond(w, nil, err)
		return
	}
	if err = s.checkEvidenceProtection(r.Context(), tx, current(r), wid, history); err != nil {
		apiError(w, 422, "현재 정보 보호 정책에서 과거 등록 이유를 표시할 수 없습니다")
		return
	}
	var canWrite bool
	err = tx.QueryRow(r.Context(), `SELECT madi_document_allowed($1,$2,true)`, current(r).ID, r.PathValue("id")).Scan(&canWrite)
	if err != nil {
		respond(w, nil, err)
		return
	}
	var protection int
	if err = tx.QueryRow(r.Context(), `SELECT revision FROM protection_settings WHERE id=1`).Scan(&protection); err != nil {
		respond(w, nil, err)
		return
	}
	jsonResponse(w, 200, map[string]any{"document_id": r.PathValue("id"), "workspace_id": wid, "document_version": version, "revision": revision, "updated_at": updated, "periods": periods, "history": history, "history_limit": 50, "can_write": canWrite && hasIntegrationScope(current(r), "document:write"), "notice": validityNotice, "protection_revision": protection})
}

func (s *Server) checkKnowledgeValidity(w http.ResponseWriter, r *http.Request) {
	tx, err := s.DB.Begin(r.Context())
	if err != nil {
		respond(w, nil, err)
		return
	}
	defer tx.Rollback(r.Context())
	_, version, err := s.validityDocumentTx(r, tx, false)
	if err != nil {
		apiError(w, 404, "현재 접근 가능한 문서가 없습니다")
		return
	}
	var revision, protection int
	err = tx.QueryRow(r.Context(), `SELECT coalesce((SELECT revision FROM knowledge_validity WHERE document_id=$1),0),revision FROM protection_settings WHERE id=1`, r.PathValue("id")).Scan(&revision, &protection)
	respond(w, map[string]any{"revision": revision, "document_version": version, "protection_revision": protection}, err)
}

func (s *Server) getKnowledgeAtDate(w http.ResponseWriter, r *http.Request) {
	date := r.URL.Query().Get("date")
	if !validKnowledgeDate(date) {
		apiError(w, 400, "업무 기준일은 YYYY-MM-DD 실제 날짜로 입력하세요")
		return
	}
	tx, err := s.DB.Begin(r.Context())
	if err != nil {
		respond(w, nil, err)
		return
	}
	defer tx.Rollback(r.Context())
	wid, version, err := s.validityDocumentTx(r, tx, false)
	if err != nil {
		apiError(w, 404, "현재 접근 가능한 문서가 없습니다")
		return
	}
	var raw []byte
	err = tx.QueryRow(r.Context(), `SELECT jsonb_build_object('document_id',d.id,'workspace_id',d.workspace_id,'version',v.version,'current_version',d.version,'title',v.title,'markdown',v.markdown,'tags',v.tags,'recorded_at',v.created_at,'valid_from',p.valid_from,'valid_until',p.valid_until,'validity_revision',c.revision) FROM documents d JOIN knowledge_validity c ON c.document_id=d.id JOIN knowledge_validity_periods p ON p.document_id=d.id JOIN document_versions v ON v.document_id=d.id AND v.version=p.document_version WHERE d.id=$1 AND daterange(p.valid_from,p.valid_until,'[)') @> $2::text::date`, r.PathValue("id"), date).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		apiError(w, 404, "이 날짜에 명시적으로 등록된 문서 버전이 없습니다")
		return
	}
	if err != nil {
		respond(w, nil, err)
		return
	}
	var out map[string]any
	if err = json.Unmarshal(raw, &out); err != nil {
		respond(w, nil, err)
		return
	}
	if expected := r.URL.Query().Get("revision"); expected != "" && expected != strconv.Itoa(number(out, "validity_revision", 0)) {
		apiError(w, 409, "검색 후 유효기간 등록이 변경되었습니다. 다시 검색하세요")
		return
	}
	if err = s.checkEvidenceProtection(r.Context(), tx, current(r), wid, out); err != nil {
		apiError(w, 422, "현재 정보 보호 정책에서 과거 원문을 표시할 수 없습니다")
		return
	}
	out["date"] = date
	out["notice"] = validityNotice
	out["content_hash"] = digest(str(out, "markdown"))
	out["historical"] = number(out, "version", 0) != version
	var protection int
	if err = tx.QueryRow(r.Context(), `SELECT revision FROM protection_settings WHERE id=1`).Scan(&protection); err != nil {
		respond(w, nil, err)
		return
	}
	out["protection_revision"] = protection
	jsonResponse(w, 200, out)
}
