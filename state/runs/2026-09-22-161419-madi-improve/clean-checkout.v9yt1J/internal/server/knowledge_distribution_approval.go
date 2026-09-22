package server

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
)

func distributionNeedsApproval(m distributionManifest) bool {
	for _, f := range m.Files {
		if f.Approval.Required {
			return true
		}
	}
	return false
}

// The decision binds the complete immutable manifest, including every file's
// hash, destination, signing key and deadline. It never contains file bytes or
// plaintext titles; the guarded review endpoint resolves those separately.
func distributionApprovalResource(v distributionExport, m distributionManifest) ApprovalResource {
	m.BundleApproval = nil
	var size int64
	for _, f := range m.Files {
		size += f.Bytes
	}
	return ApprovalResource{Kind: "knowledge_distribution", ID: v.ID, WorkspaceID: v.WorkspaceID, OwnerID: v.OwnerID, Title: "망별 지식 배포", Version: 1, Snapshot: map[string]any{
		"bundle_id": v.ID, "manifest_sha256": digest(string(jsonValue(m))), "receiver_instance": m.ReceiverInstance,
		"signing_key_id": m.KeyID, "created_at": m.CreatedAt, "expires_at": m.ExpiresAt, "file_count": len(m.Files), "total_bytes": size,
		"review_url": "/app/knowledge-distribution?review=" + v.ID,
	}}
}

func (s *Server) distributionReviewTx(ctx context.Context, tx pgx.Tx, p *Principal, id string, write bool) (distributionExport, exportRun, distributionManifest, error) {
	var v distributionExport
	var m distributionManifest
	var run exportRun
	if !validID(id) || p == nil || p.Kind != "user" || p.TokenID != "" || p.ScopeRestricted || p.PluginID != "" {
		return v, run, m, pgx.ErrNoRows
	}
	var sealed, hash string
	e := tx.QueryRow(ctx, `SELECT id::text,workspace_id::text,owner_id::text,coalesce(export_id::text,''),signing_key_id::text,receiver_instance::text,status,policy_revision,created_at,expires_at,manifest_ciphertext,manifest_hash FROM knowledge_distribution_exports WHERE id=$1`, id).Scan(&v.ID, &v.WorkspaceID, &v.OwnerID, &v.ExportID, &v.KeyID, &v.Receiver, &v.Status, &v.PolicyRevision, &v.CreatedAt, &v.ExpiresAt, &sealed, &hash)
	if e != nil || !oneOf(v.Status, "awaiting_review", "queued", "ready") || !v.ExpiresAt.After(time.Now()) || write && p.ID != v.OwnerID {
		return v, run, m, pgx.ErrNoRows
	}
	raw, e := s.decrypt(sealed)
	if e != nil || digest(raw) != hash {
		return v, run, m, approvalProblem(409, "배포 검토 자료의 무결성을 확인하세요")
	}
	m, e = parseDistributionManifest([]byte(raw))
	if e != nil || m.BundleID != v.ID || m.SourceWorkspace != v.WorkspaceID || !distributionNeedsApproval(m) {
		return v, run, m, approvalProblem(409, "승인용 배포 자료를 다시 준비하세요")
	}
	run, e = scanExportRun(tx.QueryRow(ctx, "SELECT "+exportRunSelect+" FROM export_runs WHERE id=$1", v.ExportID))
	if e != nil || run.OwnerID != v.OwnerID || run.WorkspaceID != v.WorkspaceID {
		return v, run, m, pgx.ErrNoRows
	}
	if e = s.distributionSourcesTx(ctx, tx, p, run, &m, true, nil); e != nil {
		return v, run, m, approvalProblem(409, e.Error())
	}
	if _, _, e = s.distributionSigningTx(ctx, tx, v); e != nil {
		return v, run, m, approvalProblem(409, e.Error())
	}
	var current string
	if e = tx.QueryRow(ctx, `SELECT status FROM knowledge_distribution_exports WHERE id=$1 FOR UPDATE`, id).Scan(&current); e != nil || current != v.Status {
		return v, run, m, approvalProblem(409, "배포 상태가 변경됐습니다")
	}
	if e = s.checkEvidenceProtection(ctx, tx, p, v.WorkspaceID, m); e != nil {
		return v, run, m, approvalProblem(422, "현재 정보 보호 정책에서 검토 자료를 표시할 수 없습니다")
	}
	if !v.ExpiresAt.After(time.Now()) || distributionSessionTx(ctx, tx, run) != nil {
		return v, run, m, approvalProblem(409, "검토 자료 또는 원신청자 세션이 만료됐습니다")
	}
	return v, run, m, nil
}

func (s *Server) distributionApprovalAdapter() ApprovalAdapter {
	return ApprovalAdapter{
		Lock: func(ctx context.Context, tx pgx.Tx, p *Principal, id string, write bool) (ApprovalResource, error) {
			v, _, m, e := s.distributionReviewTx(ctx, tx, p, id, write)
			return distributionApprovalResource(v, m), e
		},
		CanReview: func(ctx context.Context, tx pgx.Tx, uid string, resource ApprovalResource) (bool, error) {
			var allowed bool
			e := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM knowledge_distribution_exports e JOIN workspace_members wm ON wm.workspace_id=e.workspace_id AND wm.user_id=$2 JOIN users u ON u.id=$2 AND NOT u.disabled AND u.kind='user' WHERE e.id=$1 AND NOT EXISTS(SELECT 1 FROM jsonb_to_recordset(e.document_refs) ref(id uuid) WHERE NOT madi_document_allowed($2,ref.id,false)))`, resource.ID, uid).Scan(&allowed)
			return allowed, e
		},
		Submit: func(ctx context.Context, tx pgx.Tx, p *Principal, r ApprovalResource) (ApprovalResource, error) {
			var state string
			if e := tx.QueryRow(ctx, `SELECT status FROM knowledge_distribution_exports WHERE id=$1`, r.ID).Scan(&state); e != nil {
				return r, e
			}
			if state != "awaiting_review" {
				return r, approvalProblem(409, "검토 대기 중인 배포만 제출할 수 있습니다")
			}
			var existing string
			if e := tx.QueryRow(ctx, `SELECT coalesce(approval_id::text,'') FROM knowledge_distribution_exports WHERE id=$1`, r.ID).Scan(&existing); e != nil {
				return r, e
			}
			if existing != "" {
				previous, e := approvalReadRequestTx(ctx, tx, existing, false)
				if e != nil {
					return r, e
				}
				if oneOf(previous.Status, "pending", "approved") {
					valid, _, e := approvalCurrentTx(ctx, tx, previous, r)
					if e != nil {
						return r, e
					}
					if valid {
						return r, approvalProblem(409, "현재 동일한 배포 검토가 이미 진행 또는 완료됐습니다")
					}
				}
			}
			return r, nil
		},
		// A decision never signs or transmits files. The original requester must
		// explicitly revalidate and queue signing after all gates are approved.
		Complete: func(ctx context.Context, tx pgx.Tx, p *Principal, a ApprovalRequest, r ApprovalResource) error {
			run, e := scanExportRun(tx.QueryRow(ctx, "SELECT "+exportRunSelect+" FROM export_runs WHERE id=(SELECT export_id FROM knowledge_distribution_exports WHERE id=$1)", r.ID))
			if e != nil {
				return e
			}
			if deadline, ok := r.Snapshot["expires_at"].(int64); !ok || time.Now().Unix() >= deadline {
				return approvalProblem(409, "배포 검토 유효기간이 지났습니다")
			}
			return distributionSessionTx(ctx, tx, run)
		},
	}
}

func (s *Server) distributionBundleApprovedTx(ctx context.Context, tx pgx.Tx, v distributionExport, m *distributionManifest, check bool) error {
	var id string
	if e := tx.QueryRow(ctx, `SELECT coalesce(approval_id::text,'') FROM knowledge_distribution_exports WHERE id=$1 FOR SHARE`, v.ID).Scan(&id); e != nil {
		return e
	}
	if !distributionNeedsApproval(*m) {
		if id != "" || m.BundleApproval != nil {
			return approvalProblem(409, "승인 정책이 변경됐습니다. 새 배포를 준비하세요")
		}
		return nil
	}
	if !validID(id) {
		return approvalProblem(409, "문서와 첨부 전체·수신망·유효기간의 배포 승인을 완료하세요")
	}
	a, e := approvalReadRequestTx(ctx, tx, id, true)
	resource := distributionApprovalResource(v, *m)
	if e != nil || a.Status != "approved" || a.ResourceKind != resource.Kind || a.ResourceID != v.ID {
		return approvalProblem(409, "현재 배포의 승인을 완료해야 합니다")
	}
	valid, reason, e := approvalCurrentTx(ctx, tx, a, resource)
	if e != nil {
		return e
	}
	if !valid {
		return approvalProblem(409, reason)
	}
	expected := distributionApproval{Required: true, RequestID: a.ID, ResourceHash: a.ResourceHash}
	if check && (m.BundleApproval == nil || *m.BundleApproval != expected) {
		return approvalProblem(409, "서명에 연결한 배포 승인 근거가 변경됐습니다")
	}
	m.BundleApproval = &expected
	if !v.ExpiresAt.After(time.Now()) {
		return approvalProblem(409, "배포 승인 유효기간이 지났습니다")
	}
	return nil
}

// Return true only after the unsigned manifest was durably staged for review.
func (s *Server) prepareDistributionReview(ctx context.Context, p *Principal, run exportRun, v distributionExport, m *distributionManifest) (bool, error) {
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return false, e
	}
	defer tx.Rollback(ctx)
	if e = s.distributionSourcesTx(ctx, tx, p, run, m, true, nil); e != nil {
		return false, e
	}
	if _, _, e = s.distributionSigningTx(ctx, tx, v); e != nil {
		return false, e
	}
	var id, hash, state string
	if e = tx.QueryRow(ctx, `SELECT coalesce(approval_id::text,''),manifest_hash,status FROM knowledge_distribution_exports WHERE id=$1 FOR UPDATE`, v.ID).Scan(&id, &hash, &state); e != nil {
		return false, e
	}
	if state != "queued" {
		return false, errors.New("배포 준비 상태가 변경됐습니다")
	}
	if distributionNeedsApproval(*m) && id == "" && hash == "" {
		raw := string(jsonValue(*m))
		cipher, e := s.encrypt(raw)
		if e != nil {
			return false, e
		}
		_, e = tx.Exec(ctx, `UPDATE knowledge_distribution_exports SET status='awaiting_review',manifest_ciphertext=$2,manifest_hash=$3 WHERE id=$1`, v.ID, cipher, digest(raw))
		if e == nil {
			e = distributionSessionTx(ctx, tx, run)
		}
		if e == nil {
			e = tx.Commit(ctx)
		}
		return e == nil, e
	}
	if e = s.distributionBundleApprovedTx(ctx, tx, v, m, false); e != nil {
		return false, e
	}
	return false, nil
}

func (s *Server) getDistributionReview(w http.ResponseWriter, r *http.Request) {
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	v, _, m, e := s.distributionReviewTx(r.Context(), tx, current(r), r.PathValue("id"), false)
	if e != nil {
		approvalRespondError(w, e)
		return
	}
	status, e := s.approvalStatusTx(r.Context(), tx, current(r), distributionApprovalResource(v, m), false)
	if e == nil {
		e = s.knowledgeActorTx(r, tx, v.WorkspaceID, "document:read")
	}
	if e != nil {
		approvalRespondError(w, e)
		return
	}
	jsonResponse(w, 200, map[string]any{"id": v.ID, "owner_id": v.OwnerID, "status": v.Status, "manifest": m, "approval": status, "manifest_sha256": distributionApprovalResource(v, m).Snapshot["manifest_sha256"], "notice": "아래 문서·첨부 전체의 정확한 해시, 수신망, 유효기간을 검토합니다. 승인만으로 서명·전송하지 않습니다. 원문 확인과 별개로 내용의 사실성은 검토자가 판단해야 합니다."})
}

func (s *Server) submitDistributionReview(w http.ResponseWriter, r *http.Request) {
	var in struct {
		ManifestHash string `json:"manifest_sha256"`
		Consent      bool   `json:"consent"`
	}
	if decode(r, &in) != nil || !in.Consent || !migrationHexHash(in.ManifestHash) {
		apiError(w, 400, "배포 매니페스트 해시와 검토 제출 동의를 확인하세요")
		return
	}
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	v, _, m, e := s.distributionReviewTx(r.Context(), tx, current(r), r.PathValue("id"), true)
	if e != nil {
		approvalRespondError(w, e)
		return
	}
	if distributionApprovalResource(v, m).Snapshot["manifest_sha256"] != in.ManifestHash {
		apiError(w, 409, "확인한 배포 목록이 변경됐습니다")
		return
	}
	a, e := s.StartApprovalTx(r.Context(), tx, current(r), "knowledge_distribution", v.ID, "")
	if e == nil {
		_, e = tx.Exec(r.Context(), `UPDATE knowledge_distribution_exports SET approval_id=$2 WHERE id=$1`, v.ID, a.ID)
	}
	if e == nil {
		e = s.knowledgeActorTx(r, tx, v.WorkspaceID, "document:read")
	}
	if e == nil {
		e = tx.Commit(r.Context())
	}
	if e != nil {
		approvalRespondError(w, e)
		return
	}
	s.audit(r, "DISTRIBUTION_APPROVAL_SUBMIT", v.ID, map[string]any{"request_id": a.ID})
	jsonResponse(w, 201, approvalSummary(a))
}

func (s *Server) signReviewedDistribution(w http.ResponseWriter, r *http.Request) {
	var in struct {
		RequestID      string `json:"request_id"`
		RequestVersion int64  `json:"request_version"`
		Consent        bool   `json:"consent"`
	}
	if decode(r, &in) != nil || !in.Consent || !validID(in.RequestID) || in.RequestVersion < 1 {
		apiError(w, 400, "현재 승인 요청·버전과 서명 동의를 확인하세요")
		return
	}
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	v, run, m, e := s.distributionReviewTx(r.Context(), tx, current(r), r.PathValue("id"), true)
	if e != nil {
		approvalRespondError(w, e)
		return
	}
	cookie, ce := r.Cookie("madi_session")
	if ce != nil || run.SessionHash != digest(cookie.Value) || v.Status != "awaiting_review" {
		apiError(w, 409, "원래 로그인 세션의 검토 대기 배포만 서명할 수 있습니다")
		return
	}
	if e = s.distributionBundleApprovedTx(r.Context(), tx, v, &m, false); e != nil {
		approvalRespondError(w, e)
		return
	}
	a, e := approvalReadRequestTx(r.Context(), tx, in.RequestID, true)
	if e != nil || m.BundleApproval.RequestID != in.RequestID || a.Version != in.RequestVersion {
		apiError(w, 409, "승인 요청이 변경됐습니다")
		return
	}
	jobID, e := s.EnqueueJob(r.Context(), tx, "knowledge.distribution", v.OwnerID, v.WorkspaceID, map[string]any{"distribution_id": v.ID})
	if e == nil {
		_, e = tx.Exec(r.Context(), `UPDATE knowledge_distribution_exports SET status='queued',job_id=$2 WHERE id=$1`, v.ID, jobID)
	}
	if e == nil {
		_, e = tx.Exec(r.Context(), `UPDATE automation_jobs SET timeout_seconds=300,max_attempts=1 WHERE id=$1`, jobID)
	}
	if e == nil {
		e = s.knowledgeActorTx(r, tx, v.WorkspaceID, "document:read")
	}
	if e == nil {
		e = tx.Commit(r.Context())
	}
	if e != nil {
		approvalRespondError(w, e)
		return
	}
	s.audit(r, "DISTRIBUTION_APPROVED_SIGN_REQUEST", v.ID, map[string]any{"request_id": in.RequestID, "job_id": jobID})
	jsonResponse(w, 202, map[string]any{"id": v.ID, "job_id": jobID, "status": "queued"})
}

// Kept explicit to prevent a signed document-publication receipt being
// mistaken for approval of newly attached bytes or a different destination.
func validateReceivedDistributionApproval(m distributionManifest) error {
	if distributionNeedsApproval(m) && (m.BundleApproval == nil || !m.BundleApproval.Required) {
		return errors.New("첨부·수신망·유효기간을 포함한 전체 배포 승인 근거가 없습니다")
	}
	return nil
}
