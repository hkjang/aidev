package server

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

func (s *Server) impactExceptionJSONTx(ctx context.Context, tx pgx.Tx, p *Principal, ex impactException, review impactExceptionReview) (map[string]any, error) {
	reason, err := s.decrypt(ex.ReasonCiphertext)
	if err != nil {
		return nil, err
	}
	if err = s.checkEvidenceProtection(ctx, tx, p, review.WorkspaceID, map[string]any{"reason": reason, "source_title": review.SourceTitle, "target_title": review.TargetTitle}); err != nil {
		return nil, approvalProblem(422, "현재 보호 정책으로 예외 근거를 표시할 수 없습니다")
	}
	valid := impactExceptionCurrent(ex, review) && oneOf(ex.Status, "pending", "approved")
	message := "검토·문서 버전·직접 관계와 유효기간을 확인했습니다. 서버가 예외의 적절성을 자동 판단한 것은 아닙니다."
	if !valid {
		message = "문서·검토·관계 또는 유효기간이 변경되었습니다. 과거 결정 이력이며 현재 적용 가능한 예외가 아닙니다."
	}
	var approval map[string]any
	if ex.ApprovalID != "" {
		req, e := approvalReadRequestTx(ctx, tx, ex.ApprovalID, false)
		if e != nil {
			return nil, e
		}
		ok, why, e := approvalCurrentTx(ctx, tx, req, impactExceptionResource(ex, review))
		if e != nil {
			var failure *approvalFailure
			if !errors.As(e, &failure) || (failure.Status != 404 && failure.Status != 409) {
				return nil, e
			}
			ok = false
			why = failure.Message
		}
		if !ok {
			valid = false
			message = why
		}
		approval = approvalSummary(req)
		valid = valid && (req.Status == "pending" || req.Status == "approved")
	}
	return map[string]any{"id": ex.ID, "review_id": ex.ReviewID, "requester_id": ex.RequesterID, "review_revision": ex.ReviewRevision, "source_id": review.SourceID, "source_title": review.SourceTitle, "source_version": ex.SourceVersion, "current_source_version": review.CurrentSourceVersion, "target_id": review.TargetID, "target_title": review.TargetTitle, "target_version": ex.TargetVersion, "current_target_version": review.CurrentTargetVersion, "reason": reason, "reason_hash": ex.ReasonHash, "valid_until": ex.ValidUntil, "created_at": ex.CreatedAt, "status": ex.Status, "approval_id": ex.ApprovalID, "approval": approval, "current": valid, "effective": valid && ex.Status == "approved", "notice": message}, nil
}

func (s *Server) impactExceptionApprovalContextTx(ctx context.Context, tx pgx.Tx, p *Principal, id string) (map[string]any, error) {
	ex, err := impactExceptionReadTx(ctx, tx, id)
	if err != nil {
		return nil, err
	}
	review, err := impactExceptionReviewTx(ctx, tx, p, ex.ReviewID, false)
	if err != nil {
		return nil, err
	}
	return s.impactExceptionJSONTx(ctx, tx, p, ex, review)
}
func (s *Server) listImpactExceptions(w http.ResponseWriter, r *http.Request) {
	tx, err := s.DB.Begin(r.Context())
	if err != nil {
		respond(w, nil, err)
		return
	}
	defer tx.Rollback(r.Context())
	review, err := impactExceptionReviewTx(r.Context(), tx, current(r), r.PathValue("id"), false)
	if err != nil {
		approvalRespondError(w, err)
		return
	}
	if !impactWorkflowTx(r.Context(), tx) {
		apiError(w, 404, "검토·승인 기능이 비활성화되어 있습니다")
		return
	}
	rows, err := tx.Query(r.Context(), `SELECT id::text FROM knowledge_impact_exceptions WHERE review_id=$1 ORDER BY created_at DESC,id LIMIT 20`, review.ID)
	if err != nil {
		respond(w, nil, err)
		return
	}
	ids := []string{}
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			break
		}
		ids = append(ids, id)
	}
	rows.Close()
	if err == nil {
		err = rows.Err()
	}
	if err != nil {
		respond(w, nil, err)
		return
	}
	out := []map[string]any{}
	for _, id := range ids {
		ex, e := impactExceptionReadTx(r.Context(), tx, id)
		if e == nil {
			var value map[string]any
			value, e = s.impactExceptionJSONTx(r.Context(), tx, current(r), ex, review)
			if e == nil {
				out = append(out, value)
			}
		}
		if e != nil {
			approvalRespondError(w, e)
			return
		}
	}
	cfg, _, err := approvalConfigurationTx(r.Context(), tx)
	if err != nil {
		approvalRespondError(w, err)
		return
	}
	_, policyErr := approvalPolicyTx(r.Context(), tx, cfg, review.WorkspaceID, review.SpaceID, "impact_exception")
	var canWrite bool
	err = tx.QueryRow(r.Context(), `SELECT madi_document_allowed($1,$2,true) AND ($1::uuid=$3 OR EXISTS(SELECT 1 FROM workspace_members WHERE user_id=$1 AND workspace_id=$4 AND role IN ('owner','admin')))`, current(r).ID, review.TargetID, review.OwnerID, review.WorkspaceID).Scan(&canWrite)
	if err == nil {
		err = s.impactExceptionActorTx(r, tx, review, false)
	}
	if err == nil {
		err = tx.Commit(r.Context())
	}
	if err != nil {
		approvalRespondError(w, err)
		return
	}
	jsonResponse(w, 200, map[string]any{"items": out, "review_revision": review.Revision, "source_version": review.CurrentSourceVersion, "target_version": review.CurrentTargetVersion, "can_request": canWrite && impactExceptionHuman(r) && hasIntegrationScope(current(r), "document:write") && policyErr == nil && review.Status == "exception_requested" && review.Related && review.SourceVersion == review.CurrentSourceVersion && review.TargetVersion == review.CurrentTargetVersion, "policy_configured": policyErr == nil, "notice": "예외 요청은 실제 승인 완료가 아닙니다. 관리자가 변경 영향 예외 승인 정책을 별도로 설정해야 하며, 승인도 원문·게시·실행 권한을 변경하지 않습니다. 최근 20개 요청만 표시합니다."})
}
func (s *Server) getImpactException(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !validID(id) {
		apiError(w, 404, "접근 가능한 예외 요청이 없습니다")
		return
	}
	tx, err := s.DB.Begin(r.Context())
	if err != nil {
		respond(w, nil, err)
		return
	}
	defer tx.Rollback(r.Context())
	var rid string
	if tx.QueryRow(r.Context(), `SELECT review_id::text FROM knowledge_impact_exceptions WHERE id=$1 AND madi_impact_allowed($2,review_id)`, id, current(r).ID).Scan(&rid) != nil {
		apiError(w, 404, "접근 가능한 예외 요청이 없습니다")
		return
	}
	review, err := impactExceptionReviewTx(r.Context(), tx, current(r), rid, false)
	if err != nil {
		approvalRespondError(w, err)
		return
	}
	if !impactWorkflowTx(r.Context(), tx) {
		apiError(w, 404, "검토·승인 기능이 비활성화되어 있습니다")
		return
	}
	ex, err := impactExceptionReadTx(r.Context(), tx, id)
	var out map[string]any
	if err == nil {
		out, err = s.impactExceptionJSONTx(r.Context(), tx, current(r), ex, review)
	}
	if err == nil {
		err = s.impactExceptionActorTx(r, tx, review, false)
	}
	if err == nil {
		err = tx.Commit(r.Context())
	}
	if err != nil {
		approvalRespondError(w, err)
		return
	}
	jsonResponse(w, 200, out)
}
func (s *Server) createImpactException(w http.ResponseWriter, r *http.Request) {
	if !impactExceptionHuman(r) || !hasIntegrationScope(current(r), "document:read") || !hasIntegrationScope(current(r), "document:write") {
		apiError(w, 403, "담당자·관리자의 실제 로그인 세션과 문서 읽기·작성 권한이 필요합니다")
		return
	}
	var in struct {
		ReviewRevision int       `json:"review_revision"`
		SourceVersion  int       `json:"source_version"`
		TargetVersion  int       `json:"target_version"`
		Reason         string    `json:"reason"`
		ValidUntil     time.Time `json:"valid_until"`
		Confirm        bool      `json:"confirm"`
	}
	if decode(r, &in) != nil || !in.Confirm || in.ReviewRevision < 1 || in.SourceVersion < 1 || in.TargetVersion < 1 || len(strings.TrimSpace(in.Reason)) == 0 || len(in.Reason) > 4000 || !in.ValidUntil.After(time.Now()) || in.ValidUntil.After(time.Now().Add(90*24*time.Hour)) {
		apiError(w, 400, "현재 검토·문서 버전, 예외 근거(1~4000바이트), 90일 이내 유효기간과 명시 확인이 필요합니다")
		return
	}
	tx, err := s.DB.Begin(r.Context())
	if err != nil {
		respond(w, nil, err)
		return
	}
	defer tx.Rollback(r.Context())
	review, err := impactExceptionReviewTx(r.Context(), tx, current(r), r.PathValue("id"), true)
	if err != nil {
		approvalRespondError(w, err)
		return
	}
	if review.Revision != in.ReviewRevision || review.SourceVersion != in.SourceVersion || review.CurrentSourceVersion != in.SourceVersion || review.TargetVersion != in.TargetVersion || review.CurrentTargetVersion != in.TargetVersion || review.Status != "exception_requested" || !review.Related {
		apiError(w, 409, "예외 요청 상태·현재 버전·관계를 다시 확인하세요")
		return
	}
	// Configuration locks precede pending approval locks, matching the engine.
	cfg, _, err := approvalConfigurationTx(r.Context(), tx)
	if err == nil {
		_, err = approvalPolicyTx(r.Context(), tx, cfg, review.WorkspaceID, review.SpaceID, "impact_exception")
	}
	if err != nil {
		approvalRespondError(w, err)
		return
	}
	var pending string
	err = tx.QueryRow(r.Context(), `SELECT id::text FROM knowledge_impact_exceptions WHERE review_id=$1 AND status='pending'`, review.ID).Scan(&pending)
	if err != nil && err != pgx.ErrNoRows {
		respond(w, nil, err)
		return
	}
	if pending != "" {
		old, e := impactExceptionReadTx(r.Context(), tx, pending)
		if e != nil {
			respond(w, nil, e)
			return
		}
		info, e := s.impactExceptionJSONTx(r.Context(), tx, current(r), old, review)
		if e != nil {
			approvalRespondError(w, e)
			return
		}
		if boolean(info, "current") {
			apiError(w, 409, "처리 중인 예외 요청이 있습니다. 기존 요청을 확인하거나 취소하세요")
			return
		}
		if _, e = tx.Exec(r.Context(), `UPDATE knowledge_impact_exceptions SET status='cancelled',updated_at=clock_timestamp() WHERE id=$1`, old.ID); e == nil && old.ApprovalID != "" {
			_, e = tx.Exec(r.Context(), `UPDATE approval_requests SET status='superseded',version=version+1,updated_at=clock_timestamp() WHERE id=$1 AND status='pending'`, old.ApprovalID)
		}
		if e != nil {
			respond(w, nil, e)
			return
		}
	}
	if err = s.checkEvidenceProtection(r.Context(), tx, current(r), review.WorkspaceID, in.Reason); err != nil {
		apiError(w, 422, err.Error())
		return
	}
	sealed, err := s.encrypt(in.Reason)
	if err != nil {
		respond(w, nil, err)
		return
	}
	id := newID()
	_, err = tx.Exec(r.Context(), `INSERT INTO knowledge_impact_exceptions(id,review_id,requester_id,review_revision,source_version,target_version,reason_ciphertext,reason_hash,valid_until) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`, id, review.ID, current(r).ID, in.ReviewRevision, in.SourceVersion, in.TargetVersion, sealed, digest(in.Reason), in.ValidUntil)
	var request ApprovalRequest
	if err == nil {
		request, err = s.StartApprovalTx(r.Context(), tx, current(r), "impact_exception", id, "")
	}
	if err == nil {
		_, err = tx.Exec(r.Context(), `UPDATE knowledge_impact_exceptions SET approval_id=$2 WHERE id=$1`, id, request.ID)
	}
	if err == nil {
		err = s.impactExceptionActorTx(r, tx, review, true)
	}
	if err == nil {
		err = tx.Commit(r.Context())
	}
	if err != nil {
		approvalRespondError(w, err)
		return
	}
	s.audit(r, "KNOWLEDGE_IMPACT_EXCEPTION_REQUEST", id, map[string]any{"review_id": review.ID, "approval_id": request.ID, "source_version": in.SourceVersion, "target_version": in.TargetVersion})
	jsonResponse(w, 201, map[string]any{"id": id, "approval_id": request.ID, "status": "pending"})
}
