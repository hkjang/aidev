package server

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"
)

//go:embed knowledge_proposals.sql
var knowledgeProposalsSchema string

func (s *Server) migrateKnowledgeProposals(ctx context.Context) error {
	_, err := s.DB.Exec(ctx, knowledgeProposalsSchema)
	return err
}
func (s *Server) registerKnowledgeProposals() {
	s.handle("POST /api/v1/documents/{id}/proposals", s.createKnowledgeProposal)
	s.handle("GET /api/v1/documents/{id}/proposal-context", s.knowledgeProposalContext)
	s.handle("GET /api/v1/knowledge/proposals", s.listKnowledgeProposals)
	s.handle("GET /api/v1/knowledge/proposals/{id}", s.getKnowledgeProposal)
	s.handle("GET /api/v1/knowledge/proposals/{id}/check", s.checkKnowledgeProposal)
	s.handle("POST /api/v1/knowledge/proposals/{id}/merge", s.mergeKnowledgeProposal)
	s.handle("POST /api/v1/knowledge/proposals/{id}/decision", s.decideKnowledgeProposal)
}

func (s *Server) knowledgeProposalContext(w http.ResponseWriter, r *http.Request) {
	if !validID(r.PathValue("id")) {
		apiError(w, 400, "문서 형식을 확인하세요")
		return
	}
	tx, err := s.DB.Begin(r.Context())
	if err != nil {
		respond(w, nil, err)
		return
	}
	defer tx.Rollback(r.Context())
	var wid string
	var version, protection int
	err = tx.QueryRow(r.Context(), `SELECT workspace_id::text,version FROM documents WHERE id=$1 AND deleted_at IS NULL AND madi_document_allowed($2,id,true) AND ($3='' OR workspace_id::text=$3) FOR SHARE`, r.PathValue("id"), current(r).ID, current(r).WorkspaceID).Scan(&wid, &version)
	if err != nil {
		apiError(w, 404, "현재 변경안을 작성할 수 있는 문서가 없습니다")
		return
	}
	if err = s.knowledgeActorTx(r, tx, wid, "document:read", "document:write"); err != nil {
		apiError(w, 403, err.Error())
		return
	}
	err = tx.QueryRow(r.Context(), `SELECT revision FROM protection_settings WHERE id=1`).Scan(&protection)
	respond(w, map[string]any{"version": version, "protection_revision": protection}, err)
}

func (s *Server) checkKnowledgeProposal(w http.ResponseWriter, r *http.Request) {
	if !validID(r.PathValue("id")) {
		apiError(w, 404, "현재 접근 가능한 변경안이 없습니다")
		return
	}
	tx, err := s.DB.Begin(r.Context())
	if err != nil {
		respond(w, nil, err)
		return
	}
	defer tx.Rollback(r.Context())
	origin, err := s.proposalConflictOriginTx(r, tx, r.PathValue("id"))
	if err != nil {
		apiError(w, 404, "현재 접근 가능한 변경안 출처가 없습니다")
		return
	}
	var wid string
	var revision, version, protection int
	err = tx.QueryRow(r.Context(), `SELECT d.workspace_id::text,p.revision,d.version,c.revision FROM knowledge_proposals p JOIN documents d ON d.id=p.document_id CROSS JOIN protection_settings c WHERE p.id=$1 AND c.id=1 AND d.deleted_at IS NULL AND madi_document_allowed($2,d.id,false) AND madi_proposal_origin_allowed($2,p.id)`, r.PathValue("id"), current(r).ID).Scan(&wid, &revision, &version, &protection)
	if err != nil {
		apiError(w, 404, "현재 접근 가능한 변경안이 없습니다")
		return
	}
	if err = s.knowledgeActorTx(r, tx, wid, "document:read"); err != nil {
		apiError(w, 403, "현재 접근 권한이 변경되었습니다")
		return
	}
	jsonResponse(w, 200, map[string]any{"revision": revision, "current_version": version, "protection_revision": protection, "origin_stale": boolean(origin, "stale")})
}

type knowledgeProposalBody struct {
	Markdown string `json:"markdown"`
	Reason   string `json:"reason"`
}

const proposalSelect = `SELECT jsonb_build_object('id',p.id,'document_id',p.document_id,'document_title',d.title,'workspace_id',d.workspace_id,'owner_id',p.owner_id,'owner_name',u.name,'base_version',p.base_version,'current_version',d.version,'base_hash',p.base_hash,'markdown_hash',p.markdown_hash,'ciphertext',p.ciphertext,'provenance',p.provenance,'status',p.status,'revision',p.revision,'merged_version',p.merged_version,'created_at',p.created_at,'can_write',madi_document_allowed($1,d.id,true)) FROM knowledge_proposals p JOIN documents d ON d.id=p.document_id JOIN users u ON u.id=p.owner_id WHERE d.deleted_at IS NULL AND madi_document_allowed($1,d.id,false) AND madi_proposal_origin_allowed($1,p.id) AND ($2='' OR d.workspace_id::text=$2)`

func (s *Server) createKnowledgeProposal(w http.ResponseWriter, r *http.Request) {
	var in struct {
		BaseVersion      int    `json:"base_version"`
		Markdown         string `json:"markdown"`
		Reason           string `json:"reason"`
		Provenance       string `json:"provenance"`
		Consent          bool   `json:"consent"`
		ConflictID       string `json:"conflict_id"`
		ConflictRevision int    `json:"conflict_revision"`
		ConflictConsent  bool   `json:"conflict_consent"`
	}
	id := r.PathValue("id")
	if decode(r, &in) != nil || !validID(id) || in.BaseVersion < 1 || len(in.Markdown) > 2<<20 || len(strings.TrimSpace(in.Reason)) == 0 || len(in.Reason) > 4000 || !oneOf(in.Provenance, "human", "ai_assisted") || !in.Consent {
		apiError(w, 400, "기준 버전·2MiB 이하 원문 제안·작성 이유·작성 방식·문서 열람자에게 제안 공유 동의를 확인하세요")
		return
	}
	if _, err := parseFrontMatter(in.Markdown); err != nil {
		apiError(w, 400, err.Error())
		return
	}
	p := current(r)
	if !hasIntegrationScope(p, "document:write") || !hasIntegrationScope(p, "document:read") {
		apiError(w, 403, "현재 문서 조회·작성 권한이 필요합니다")
		return
	}
	tx, err := s.DB.Begin(r.Context())
	if err != nil {
		respond(w, nil, err)
		return
	}
	defer tx.Rollback(r.Context())
	origin, err := s.validateProposalConflictCreationTx(r, tx, id, in.ConflictID, in.ConflictRevision, in.ConflictConsent)
	if err != nil {
		approvalRespondError(w, err)
		return
	}
	var wid, base string
	err = tx.QueryRow(r.Context(), `SELECT workspace_id::text,markdown FROM documents WHERE id=$1 AND version=$2 AND deleted_at IS NULL AND madi_document_allowed($3,id,true) FOR SHARE`, id, in.BaseVersion, p.ID).Scan(&wid, &base)
	if err != nil {
		apiError(w, 409, "원문 또는 현재 작성 권한이 변경되었습니다. 최신 원문으로 비교하세요")
		return
	}
	if base == in.Markdown {
		apiError(w, 400, "원문과 다른 변경 내용을 작성하세요")
		return
	}
	if len(jsonValue(map[string]any{"base": base, "proposed": in.Markdown})) > 2<<20 {
		apiError(w, 400, "변경안 비교용 기준·제안 원문 합계는 JSON 기준 2MiB 이하여야 합니다")
		return
	}
	if err = s.knowledgeActorTx(r, tx, wid, "document:read", "document:write"); err != nil {
		apiError(w, 403, err.Error())
		return
	}
	body := knowledgeProposalBody{in.Markdown, in.Reason}
	if err = s.checkEvidenceProtection(r.Context(), tx, p, wid, body); err != nil {
		apiError(w, 422, err.Error())
		return
	}
	// Serialize this author's creations to keep the documented open limit exact.
	if _, err = tx.Exec(r.Context(), `SELECT pg_advisory_xact_lock(hashtextextended($1,731))`, p.ID); err != nil {
		respond(w, nil, err)
		return
	}
	var count int
	if err = tx.QueryRow(r.Context(), `SELECT count(*) FROM knowledge_proposals WHERE owner_id=$1 AND status='open'`, p.ID).Scan(&count); err != nil {
		respond(w, nil, err)
		return
	}
	if count >= 100 {
		apiError(w, 409, "열린 개인 변경안은 최대 100개입니다. 기존 변경안을 반영하거나 철회하세요")
		return
	}
	sealed, err := s.encrypt(string(jsonValue(body)))
	if err != nil {
		respond(w, nil, err)
		return
	}
	var proposalID string
	err = tx.QueryRow(r.Context(), `INSERT INTO knowledge_proposals(document_id,owner_id,base_version,base_hash,markdown_hash,ciphertext,provenance) VALUES($1,$2,$3,$4,$5,$6,$7) ON CONFLICT DO NOTHING RETURNING id::text`, id, p.ID, in.BaseVersion, digest(base), digest(in.Markdown), sealed, in.Provenance).Scan(&proposalID)
	if errors.Is(err, pgx.ErrNoRows) {
		apiError(w, 409, "같은 기준 버전과 원문의 열린 변경안이 이미 있습니다")
		return
	}
	if err == nil {
		err = insertProposalConflictOriginTx(r, tx, proposalID, origin)
	}
	if err == nil {
		err = s.proposalConflictFinalACLTx(r, tx, proposalID)
	}
	if err == nil {
		err = tx.Commit(r.Context())
	}
	if err != nil {
		respond(w, nil, err)
		return
	}
	s.audit(r, "KNOWLEDGE_PROPOSAL_CREATE", id, map[string]any{"proposal_id": proposalID, "base_version": in.BaseVersion, "provenance": in.Provenance})
	jsonResponse(w, 201, map[string]any{"id": proposalID, "status": "open", "revision": 1})
}

func (s *Server) listKnowledgeProposals(w http.ResponseWriter, r *http.Request) {
	wid := r.URL.Query().Get("workspace_id")
	if _, err := s.packagePrincipal(r, wid); err != nil {
		apiError(w, 403, "현재 워크스페이스 조회 권한이 필요합니다")
		return
	}
	rows, err := s.rows(r.Context(), proposalSelect+` AND d.workspace_id=$3 ORDER BY p.created_at DESC,p.id LIMIT 200`, current(r).ID, current(r).WorkspaceID, wid)
	for _, row := range rows {
		delete(row, "ciphertext")
		delete(row, "base_hash")
		delete(row, "markdown_hash")
		row["can_write"] = boolean(row, "can_write") && hasIntegrationScope(current(r), "document:write")
	}
	respond(w, rows, err)
}

func (s *Server) readKnowledgeProposal(r *http.Request, tx pgx.Tx) (map[string]any, knowledgeProposalBody, error) {
	out := knowledgeProposalBody{}
	id := r.PathValue("id")
	if !validID(id) || !hasIntegrationScope(current(r), "document:read") {
		return nil, out, pgx.ErrNoRows
	}
	origin, originError := s.proposalConflictOriginTx(r, tx, id)
	if originError != nil {
		return nil, out, originError
	}
	var documentID string
	if err := tx.QueryRow(r.Context(), `SELECT d.id::text FROM knowledge_proposals p JOIN documents d ON d.id=p.document_id WHERE p.id=$1 AND d.deleted_at IS NULL AND madi_document_allowed($2,d.id,false) AND ($3='' OR d.workspace_id::text=$3) FOR SHARE OF d`, id, current(r).ID, current(r).WorkspaceID).Scan(&documentID); err != nil {
		return nil, out, err
	}
	var raw []byte
	err := tx.QueryRow(r.Context(), proposalSelect+` AND p.id=$3 FOR SHARE OF p`, current(r).ID, current(r).WorkspaceID, id).Scan(&raw)
	if err != nil {
		return nil, out, err
	}
	var row map[string]any
	if err = json.Unmarshal(raw, &row); err != nil {
		return nil, out, err
	}
	plain, err := s.decrypt(str(row, "ciphertext"))
	if err != nil || json.Unmarshal([]byte(plain), &out) != nil || digest(out.Markdown) != str(row, "markdown_hash") {
		return nil, out, errors.New("변경안 원문의 무결성을 확인할 수 없습니다")
	}
	wid := str(row, "workspace_id")
	if err = s.knowledgeActorTx(r, tx, wid, "document:read"); err != nil {
		return nil, out, err
	}
	if err = s.checkEvidenceProtection(r.Context(), tx, current(r), wid, out); err != nil {
		return nil, out, err
	}
	delete(row, "ciphertext")
	if origin != nil {
		row["origin"] = origin
		row["origin_stale"] = boolean(origin, "stale")
	}
	row["can_write"] = boolean(row, "can_write") && hasIntegrationScope(current(r), "document:write")
	return row, out, nil
}

func (s *Server) getKnowledgeProposal(w http.ResponseWriter, r *http.Request) {
	tx, err := s.DB.Begin(r.Context())
	if err != nil {
		respond(w, nil, err)
		return
	}
	defer tx.Rollback(r.Context())
	row, body, err := s.readKnowledgeProposal(r, tx)
	if errors.Is(err, pgx.ErrNoRows) {
		apiError(w, 404, "현재 열람 가능한 변경안이 없습니다")
		return
	}
	if err != nil {
		var failure *approvalFailure
		if errors.As(err, &failure) {
			approvalRespondError(w, err)
			return
		}
		apiError(w, 409, err.Error())
		return
	}
	var base, currentMD string
	err = tx.QueryRow(r.Context(), `SELECT v.markdown,d.markdown FROM documents d JOIN document_versions v ON v.document_id=d.id AND v.version=$2 WHERE d.id=$1 AND d.deleted_at IS NULL AND madi_document_allowed($3,d.id,false) FOR SHARE OF d`, str(row, "document_id"), number(row, "base_version", 0), current(r).ID).Scan(&base, &currentMD)
	if err != nil || digest(base) != str(row, "base_hash") {
		apiError(w, 409, "기준 원문 이력을 확인할 수 없습니다")
		return
	}
	if err = s.checkEvidenceProtection(r.Context(), tx, current(r), str(row, "workspace_id"), map[string]any{"base": base, "current": currentMD}); err != nil {
		apiError(w, 409, err.Error())
		return
	}
	history, err := tx.Query(r.Context(), `SELECT revision,status,note_ciphertext FROM knowledge_proposal_events WHERE proposal_id=$1 ORDER BY revision DESC LIMIT 100`, r.PathValue("id"))
	if err != nil {
		respond(w, nil, err)
		return
	}
	events := []map[string]any{}
	for history.Next() {
		var rev int
		var status, sealed string
		if err = history.Scan(&rev, &status, &sealed); err != nil {
			break
		}
		note, e := s.decrypt(sealed)
		if e != nil {
			err = e
			break
		}
		events = append(events, map[string]any{"revision": rev, "status": status, "note": note})
	}
	history.Close()
	if err == nil {
		err = history.Err()
	}
	if err == nil {
		err = s.checkEvidenceProtection(r.Context(), tx, current(r), str(row, "workspace_id"), events)
	}
	if err != nil {
		apiError(w, 409, "현재 정책에서 변경안 검토 이력을 표시할 수 없습니다")
		return
	}
	row["markdown"], row["reason"], row["base_markdown"], row["current_markdown"] = body.Markdown, body.Reason, base, currentMD
	row["diff"], row["stale"], row["events"] = boundedDocumentDiff(base, body.Markdown), number(row, "base_version", 0) != number(row, "current_version", 0) || boolean(row, "origin_stale"), events
	var protectionRevision int
	if err = tx.QueryRow(r.Context(), `SELECT revision FROM protection_settings WHERE id=1 FOR SHARE`).Scan(&protectionRevision); err != nil {
		respond(w, nil, err)
		return
	}
	row["protection_revision"] = protectionRevision
	row["notice"] = "제안 본문은 문서와 별도로 보관합니다. AI 도움 표시는 작성자의 분류이며 모델의 사실성 인증이 아닙니다. 변경안 반영은 문서 게시 승인이 아닙니다. 현재 기준 버전이 바뀌면 비교 후 새 변경안이 필요합니다."
	if err = s.proposalConflictFinalACLTx(r, tx, r.PathValue("id")); err != nil {
		approvalRespondError(w, err)
		return
	}
	jsonResponse(w, 200, row)
}

type documentProposalMergeKey struct{}
type documentProposalMerge struct {
	ID       string
	Revision int
	Note     string
}

func (s *Server) mergeKnowledgeProposal(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Revision int    `json:"revision"`
		Consent  bool   `json:"consent"`
		Note     string `json:"note"`
	}
	if decode(r, &in) != nil || in.Revision < 1 || !in.Consent || len(in.Note) > 4000 {
		apiError(w, 400, "변경안 버전·비교 후 반영 동의를 확인하세요")
		return
	}
	if !hasIntegrationScope(current(r), "document:write") {
		apiError(w, 403, "문서 작성 권한이 필요합니다")
		return
	}
	tx, err := s.DB.Begin(r.Context())
	if err != nil {
		respond(w, nil, err)
		return
	}
	defer tx.Rollback(r.Context())
	row, body, err := s.readKnowledgeProposal(r, tx)
	if err != nil {
		apiError(w, 404, "현재 접근 가능한 변경안이 없습니다")
		return
	}
	_ = tx.Rollback(r.Context())
	if str(row, "status") == "merged" {
		jsonResponse(w, 200, map[string]any{"id": str(row, "document_id"), "proposal_id": str(row, "id"), "merged_version": row["merged_version"], "replayed": true})
		return
	}
	if boolean(row, "origin_stale") {
		apiError(w, 409, "비교 출처 또는 판단 기록이 변경되었습니다. 새 분석으로 변경안을 작성하세요")
		return
	}
	request := r.Clone(context.WithValue(r.Context(), documentProposalMergeKey{}, documentProposalMerge{str(row, "id"), in.Revision, in.Note}))
	s.saveDocument(w, request, str(row, "document_id"), map[string]any{"version": number(row, "base_version", 0), "markdown": body.Markdown})
}

// Trusted intent is verified under the document lock used by the ordinary
// REST/MCP mutation. Proposal and document changes commit or roll back together.
func (s *Server) validateProposalMergeTx(r *http.Request, tx pgx.Tx, id string, version int, oldMarkdown, newMarkdown string) error {
	intent, ok := r.Context().Value(documentProposalMergeKey{}).(documentProposalMerge)
	if !ok {
		return nil
	}
	origin, originError := s.proposalConflictOriginTx(r, tx, intent.ID)
	if originError != nil {
		return originError
	}
	if boolean(origin, "stale") {
		return errors.New("비교 출처 또는 판단 기록이 변경되었습니다. 새 분석으로 확인하세요")
	}
	var revision, baseVersion int
	var status, baseHash, newHash, wid string
	err := tx.QueryRow(r.Context(), `SELECT p.revision,p.base_version,p.status,p.base_hash,p.markdown_hash,d.workspace_id::text FROM knowledge_proposals p JOIN documents d ON d.id=p.document_id WHERE p.id=$1 AND p.document_id=$2 AND d.deleted_at IS NULL AND madi_document_allowed($3,d.id,true) FOR UPDATE OF p`, intent.ID, id, current(r).ID).Scan(&revision, &baseVersion, &status, &baseHash, &newHash, &wid)
	if err != nil || revision != intent.Revision || status != "open" || baseVersion != version || baseHash != digest(oldMarkdown) || newHash != digest(newMarkdown) {
		return errors.New("변경안·기준 원문·현재 권한이 달라졌습니다. 자동 병합하지 않았습니다")
	}
	if err = s.knowledgeActorTx(r, tx, wid, "document:read", "document:write"); err != nil {
		return err
	}
	return s.checkEvidenceProtection(r.Context(), tx, current(r), wid, intent.Note)
}
func (s *Server) finishProposalMergeTx(r *http.Request, tx pgx.Tx, id, markdown string, version int) error {
	intent, ok := r.Context().Value(documentProposalMergeKey{}).(documentProposalMerge)
	if !ok {
		return nil
	}
	var expected string
	if err := tx.QueryRow(r.Context(), `SELECT markdown_hash FROM knowledge_proposals WHERE id=$1 AND document_id=$2`, intent.ID, id).Scan(&expected); err != nil {
		return err
	}
	if expected != digest(markdown) {
		return errors.New("보호 정책으로 변경안 원문이 바뀌었습니다. 새 비교를 확인해야 합니다")
	}
	sealed, err := s.encrypt(intent.Note)
	if err != nil {
		return err
	}
	result, err := tx.Exec(r.Context(), `UPDATE knowledge_proposals SET status='merged',merged_version=$2,revision=revision+1,updated_at=now() WHERE id=$1 AND status='open' AND revision=$3`, intent.ID, version, intent.Revision)
	if err != nil {
		return err
	}
	if result.RowsAffected() != 1 {
		return errors.New("변경안이 먼저 처리되었습니다")
	}
	_, err = tx.Exec(r.Context(), `INSERT INTO knowledge_proposal_events(proposal_id,actor_id,revision,status,note_ciphertext) VALUES($1,$2,$3,'merged',$4)`, intent.ID, current(r).ID, intent.Revision+1, sealed)
	if err == nil {
		err = s.proposalConflictFinalACLTx(r, tx, intent.ID)
	}
	return err
}

func (s *Server) decideKnowledgeProposal(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Revision int    `json:"revision"`
		Status   string `json:"status"`
		Note     string `json:"note"`
	}
	if decode(r, &in) != nil || !validID(r.PathValue("id")) || in.Revision < 1 || !oneOf(in.Status, "rejected", "withdrawn") || len(strings.TrimSpace(in.Note)) == 0 || len(in.Note) > 4000 {
		apiError(w, 400, "변경안 revision·철회/반려 상태·의견을 확인하세요")
		return
	}
	tx, err := s.DB.Begin(r.Context())
	if err != nil {
		respond(w, nil, err)
		return
	}
	defer tx.Rollback(r.Context())
	var wid, owner, doc string
	var revision int
	var status string
	if _, err = s.proposalConflictOriginTx(r, tx, r.PathValue("id")); err != nil {
		apiError(w, 404, "현재 처리할 수 있는 변경안 출처가 없습니다")
		return
	}
	// Resource-before-proposal order matches merging and prevents inversion.
	if err = tx.QueryRow(r.Context(), `SELECT d.id::text FROM documents d JOIN knowledge_proposals p ON p.document_id=d.id WHERE p.id=$1 AND d.deleted_at IS NULL AND madi_document_allowed($2,d.id,true) FOR SHARE OF d`, r.PathValue("id"), current(r).ID).Scan(&doc); err != nil {
		apiError(w, 404, "현재 처리할 수 있는 변경안이 없습니다")
		return
	}
	err = tx.QueryRow(r.Context(), `SELECT d.workspace_id::text,p.owner_id::text,p.revision,p.status FROM knowledge_proposals p JOIN documents d ON d.id=p.document_id WHERE p.id=$1 FOR UPDATE OF p`, r.PathValue("id")).Scan(&wid, &owner, &revision, &status)
	if err != nil {
		respond(w, nil, err)
		return
	}
	if revision != in.Revision || status != "open" {
		apiError(w, 409, "이미 처리되었거나 변경된 제안입니다")
		return
	}
	if in.Status == "withdrawn" && owner != current(r).ID {
		apiError(w, 403, "작성자만 변경안을 철회할 수 있습니다")
		return
	}
	if in.Status == "rejected" && !impactWorkflowTx(r.Context(), tx) {
		apiError(w, 403, "관리자가 검토·승인 기능을 활성화한 경우에만 반려 절차를 사용합니다")
		return
	}
	if err = s.knowledgeActorTx(r, tx, wid, "document:read", "document:write"); err != nil {
		apiError(w, 403, err.Error())
		return
	}
	if err = s.checkEvidenceProtection(r.Context(), tx, current(r), wid, in.Note); err != nil {
		apiError(w, 422, err.Error())
		return
	}
	sealed, err := s.encrypt(in.Note)
	if err == nil {
		_, err = tx.Exec(r.Context(), `UPDATE knowledge_proposals SET status=$2,revision=revision+1,updated_at=now() WHERE id=$1`, r.PathValue("id"), in.Status)
	}
	if err == nil {
		_, err = tx.Exec(r.Context(), `INSERT INTO knowledge_proposal_events(proposal_id,actor_id,revision,status,note_ciphertext) VALUES($1,$2,$3,$4,$5)`, r.PathValue("id"), current(r).ID, revision+1, in.Status, sealed)
	}
	if err == nil {
		err = tx.Commit(r.Context())
	}
	if err != nil {
		respond(w, nil, err)
		return
	}
	s.audit(r, "KNOWLEDGE_PROPOSAL_DECISION", doc, map[string]any{"proposal_id": r.PathValue("id"), "status": in.Status})
	jsonResponse(w, 200, map[string]any{"revision": revision + 1, "status": in.Status})
}
