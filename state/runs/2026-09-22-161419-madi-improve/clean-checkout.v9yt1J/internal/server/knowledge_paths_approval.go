package server

import (
	"context"
	"github.com/jackc/pgx/v5"
	"net/http"
)

func (s *Server) learningResourceTx(ctx context.Context, tx pgx.Tx, p *Principal, id string, write bool) (ApprovalResource, error) {
	var empty ApprovalResource
	if p == nil || p.Kind != "user" || p.PluginID != "" || !validID(id) || !hasIntegrationScope(p, "document:read") || write && (p.TokenID != "" || p.ScopeRestricted) {
		return empty, pgx.ErrNoRows
	}
	if e := learningACLTx(ctx, tx); e != nil {
		return empty, e
	}
	var pathID string
	if e := tx.QueryRow(ctx, `SELECT st.path_id::text FROM knowledge_path_progress g JOIN knowledge_path_steps st ON st.id=g.step_id WHERE g.id=$1`, id).Scan(&pathID); e != nil {
		return empty, e
	}
	path, e := knowledgePathReadTx(ctx, tx, p, pathID, false)
	if e != nil {
		return empty, e
	}
	g, e := scanLearningProgress(tx.QueryRow(ctx, "SELECT "+learningProgressSelect+" FROM knowledge_path_progress g WHERE id=$1 FOR UPDATE", id))
	if e != nil {
		return empty, e
	}
	if write && g.UserID != p.ID {
		return empty, pgx.ErrNoRows
	}
	if g.UserID != p.ID {
		var assigned bool
		e = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM approval_assignments aa JOIN approval_requests a ON a.id=aa.request_id WHERE aa.user_id=$1 AND a.resource_kind='learning_step' AND a.resource_id=$2 AND a.resource_version=$3)`, p.ID, id, g.Revision).Scan(&assigned)
		if e != nil || !assigned {
			return empty, pgx.ErrNoRows
		}
	}
	steps, e := knowledgePathStepsTx(ctx, tx, path.ID)
	if e != nil {
		return empty, e
	}
	prefix := []knowledgePathStep{}
	var selected knowledgePathStep
	for _, st := range steps {
		prefix = append(prefix, st)
		if st.ID == g.StepID {
			selected = st
			break
		}
	}
	if selected.ID == "" || selected.Kind != "review" {
		return empty, pgx.ErrNoRows
	}
	versions, e := learningDocumentsTx(ctx, tx, p, path.WorkspaceID, prefix)
	if e != nil {
		return empty, e
	}
	// A review freezes the exact earlier read/practice records and their current
	// source versions. No plaintext practice content is duplicated in snapshots.
	refs := []map[string]any{}
	for _, st := range prefix {
		if st.ID == selected.ID {
			break
		}
		if st.Kind == "review" {
			continue
		}
		prior, e := scanLearningProgress(tx.QueryRow(ctx, "SELECT "+learningProgressSelect+" FROM knowledge_path_progress g WHERE step_id=$1 AND user_id=$2 FOR SHARE", st.ID, g.UserID))
		if e != nil {
			return empty, approvalProblem(409, "선행 읽기·실습 기록을 다시 확인하세요")
		}
		refs = append(refs, map[string]any{"id": prior.ID, "step_id": st.ID, "document_id": st.DocumentID, "revision": prior.Revision, "path_revision": prior.PathRevision, "document_version": prior.DocumentVersion, "current_document_version": versions[st.DocumentID], "state": prior.State, "proof_hash": digest(prior.Proof)})
		if prior.Proof != "" {
			proof, e := s.decrypt(prior.Proof)
			if e != nil {
				return empty, e
			}
			if e = s.checkEvidenceProtection(ctx, tx, p, path.WorkspaceID, map[string]any{"proof": proof}); e != nil {
				return empty, approvalProblem(422, "현재 정보보호 정책으로 제출한 실습 원문을 표시할 수 없습니다. 정제한 기록을 다시 제출하세요")
			}
		}
	}
	if g.Proof != "" {
		proof, e := s.decrypt(g.Proof)
		if e != nil {
			return empty, e
		}
		if e = s.checkEvidenceProtection(ctx, tx, p, path.WorkspaceID, map[string]any{"proof": proof}); e != nil {
			return empty, approvalProblem(422, "현재 정보보호 정책으로 제출한 실습 원문을 표시할 수 없습니다. 정제한 기록을 다시 제출하세요")
		}
	}
	return ApprovalResource{Kind: "learning_step", ID: id, WorkspaceID: path.WorkspaceID, SpaceID: path.SpaceID, OwnerID: g.UserID, Title: "지식 경로의 독립 실습 검토", Version: g.Revision, Snapshot: map[string]any{"path_id": path.ID, "step_id": g.StepID, "path_revision": g.PathRevision, "current_path_revision": path.Revision, "path_archived": path.Archived, "document_id": selected.DocumentID, "document_version": g.DocumentVersion, "current_document_version": versions[selected.DocumentID], "proof_hash": digest(g.Proof), "preceding_records": refs, "review_url": "/app/knowledge-paths?path=" + path.ID + "&review=" + g.ID}}, nil
}
func (s *Server) learningApprovalAdapter() ApprovalAdapter {
	return ApprovalAdapter{
		Lock: s.learningResourceTx,
		CanReview: func(ctx context.Context, tx pgx.Tx, uid string, res ApprovalResource) (bool, error) {
			var allowed bool
			e := tx.QueryRow(ctx, `SELECT madi_knowledge_path_allowed($1,$2,false) AND NOT EXISTS(SELECT 1 FROM knowledge_path_steps st WHERE st.path_id=$2 AND st.active AND st.ordinal<=(SELECT ordinal FROM knowledge_path_steps WHERE id=$3) AND NOT madi_document_allowed($1,st.document_id,false))`, uid, str(res.Snapshot, "path_id"), str(res.Snapshot, "step_id")).Scan(&allowed)
			return allowed, e
		},
		Submit: func(ctx context.Context, tx pgx.Tx, p *Principal, res ApprovalResource) (ApprovalResource, error) {
			var state string
			if e := tx.QueryRow(ctx, `SELECT state FROM knowledge_path_progress WHERE id=$1`, res.ID).Scan(&state); e != nil {
				return res, e
			}
			if state != "pending_review" {
				return res, approvalProblem(409, "명시적으로 제출한 현재 실습 기록만 검토할 수 있습니다")
			}
			if res.Snapshot["path_archived"] == true || res.Snapshot["path_revision"] != res.Snapshot["current_path_revision"] || res.Snapshot["document_version"] != res.Snapshot["current_document_version"] {
				return res, approvalProblem(409, "현재 경로와 원문 버전을 다시 확인하세요")
			}
			return res, nil
		},
		Complete: func(ctx context.Context, tx pgx.Tx, p *Principal, a ApprovalRequest, res ApprovalResource) error {
			state := "needs_recheck"
			if a.Status == "approved" {
				state = "approved"
			} else if a.Status == "rejected" {
				state = "rejected"
			}
			_, e := tx.Exec(ctx, `UPDATE knowledge_path_progress SET state=$2,updated_at=clock_timestamp() WHERE id=$1`, res.ID, state)
			if e == nil {
				_, e = tx.Exec(ctx, `INSERT INTO knowledge_path_events(id,path_id,step_id,user_id,action,metadata) VALUES($1,$2,$3,$4,$5,$6)`, newID(), str(res.Snapshot, "path_id"), str(res.Snapshot, "step_id"), res.OwnerID, state, jsonValue(map[string]any{"progress_id": res.ID, "revision": res.Version, "approval_id": a.ID, "reviewer_id": p.ID}))
			}
			return e
		}}
}
func (s *Server) getLearningReview(w http.ResponseWriter, r *http.Request) {
	if !learningCookie(r) {
		apiError(w, 403, "실습 원문 검토는 사람의 로그인 화면에서 가능합니다")
		return
	}
	tx, e := s.learningTx(r, true)
	if e != nil {
		learningRespondError(w, e)
		return
	}
	defer tx.Rollback(r.Context())
	res, e := s.learningResourceTx(r.Context(), tx, current(r), r.PathValue("id"), false)
	if e != nil {
		learningRespondError(w, e)
		return
	}
	g, e := scanLearningProgress(tx.QueryRow(r.Context(), "SELECT "+learningProgressSelect+" FROM knowledge_path_progress g WHERE id=$1", res.ID))
	if e != nil {
		learningRespondError(w, e)
		return
	}
	a, e := approvalReadRequestTx(r.Context(), tx, g.ApprovalID, false)
	if e != nil {
		learningRespondError(w, e)
		return
	}
	valid, reason, e := approvalCurrentTx(r.Context(), tx, a, res)
	if e != nil {
		learningRespondError(w, e)
		return
	}
	if !valid {
		apiError(w, 409, "제출 이후 경로·원문·선행 기록이 변경되었습니다. 새 실습 기록은 재제출 후 검토할 수 있습니다: "+reason)
		return
	}
	proof, e := s.decrypt(g.Proof)
	if e != nil {
		learningRespondError(w, e)
		return
	}
	refs := []map[string]any{}
	for _, ref := range res.Snapshot["preceding_records"].([]map[string]any) {
		var cipher string
		var title string
		e = tx.QueryRow(r.Context(), `SELECT g.proof,st.title FROM knowledge_path_progress g JOIN knowledge_path_steps st ON st.id=g.step_id WHERE g.id=$1`, ref["id"]).Scan(&cipher, &title)
		if e != nil {
			learningRespondError(w, e)
			return
		}
		plain := ""
		if cipher != "" {
			plain, e = s.decrypt(cipher)
			if e != nil {
				learningRespondError(w, e)
				return
			}
		}
		refs = append(refs, map[string]any{"step_id": ref["step_id"], "document_id": ref["document_id"], "document_version": ref["document_version"], "title": title, "proof": plain})
	}
	if e == nil {
		e = s.learningActorTx(r, tx, res.WorkspaceID, "document:read")
	}
	if e == nil {
		e = tx.Commit(r.Context())
	}
	if e != nil {
		learningRespondError(w, e)
		return
	}
	jsonResponse(w, 200, map[string]any{"progress": g, "proof": proof, "preceding_records": refs, "current": valid, "reason": reason, "approval_id": g.ApprovalID, "path_id": str(res.Snapshot, "path_id"), "notice": "현재 근거 접근 권한과 제출한 기록을 확인합니다. 승인해도 문서 게시·실제 명령 실행·추가 접근 권한은 부여되지 않습니다."})
}
