package server

import (
	"context"

	"github.com/jackc/pgx/v5"
)

// ApprovalResource is a server-built immutable review target, never decoded
// directly from a user request. Snapshot must contain everything being approved.
// DocumentID is optional for non-document resources; WorkspaceID is mandatory.
type ApprovalResource struct {
	Kind, ID, WorkspaceID, SpaceID, OwnerID, DocumentID, Title string
	Version                                                    int64
	Snapshot                                                   map[string]any
}

// Each callback must use only tx.Query/Exec, never s.DB or remote network calls.
// Lock obtains the resource's serialization lock and validates the actor's
// current read/write ACL. CanReview checks each candidate's CURRENT resource
// ACL. Submit may transition resource state/version and returns the final
// immutable review snapshot. Complete commits approved/rejected/cancelled state
// in the SAME transaction; returning an error rolls back all review decisions.
type ApprovalAdapter struct {
	Lock      func(context.Context, pgx.Tx, *Principal, string, bool) (ApprovalResource, error)
	CanReview func(context.Context, pgx.Tx, string, ApprovalResource) (bool, error)
	Submit    func(context.Context, pgx.Tx, *Principal, ApprovalResource) (ApprovalResource, error)
	Complete  func(context.Context, pgx.Tx, *Principal, ApprovalRequest, ApprovalResource) error
}

type ApprovalRequest = approvalRequest

// RegisterApprovalAdapter is startup-only. The server must not be serving yet.
func (s *Server) RegisterApprovalAdapter(kind string, adapter ApprovalAdapter) {
	if !oneOf(kind, "document", "runbook", "sql_query_plan", "impact_exception", "knowledge_distribution", "learning_step") || adapter.Lock == nil || adapter.CanReview == nil {
		panic("invalid approval adapter")
	}
	if s.approvalAdapters == nil {
		s.approvalAdapters = map[string]ApprovalAdapter{}
	}
	if _, exists := s.approvalAdapters[kind]; exists {
		panic("duplicate approval adapter")
	}
	s.approvalAdapters[kind] = adapter
}

// ConsumeApprovalTx is the one-time execution authorization boundary for a
// runbook adapter. The adapter's Complete callback must not mutate the approved
// immutable payload/version. The caller queues its exact execution in this same
// transaction, then executes outside the transaction with its own sandbox and
// idempotency controls. A restored, changed, cancelled or reused approval fails.
func (s *Server) ConsumeApprovalTx(ctx context.Context, tx pgx.Tx, p *Principal, kind, resourceID, requestID string, expectedVersion int64) (ApprovalRequest, error) {
	var out ApprovalRequest
	if kind != "runbook" || p == nil || p.Kind != "user" || p.TokenID != "" || p.ScopeRestricted {
		return out, approvalProblem(403, "실행 승인은 실제 요청자의 로그인 세션에서만 소비할 수 있습니다")
	}
	adapter, ok := s.approvalAdapters[kind]
	if !ok || !validID(requestID) || expectedVersion < 1 {
		return out, approvalProblem(400, "실행 승인 대상과 요청 버전을 확인하세요")
	}
	resource, e := adapter.Lock(ctx, tx, p, resourceID, true)
	if e != nil {
		return out, e
	}
	out, e = approvalReadRequestTx(ctx, tx, requestID, false)
	if e != nil {
		return out, e
	}
	if out.Status != "approved" || out.Version != expectedVersion || out.ResourceKind != kind || out.ResourceID != resourceID || out.WorkspaceID != resource.WorkspaceID || p.ID != out.RequesterID {
		return out, approvalProblem(409, "현재 요청자가 한 번만 사용할 수 있는 실행 승인이 아닙니다")
	}
	valid, reason, e := approvalCurrentTx(ctx, tx, out, resource)
	if e != nil {
		return out, e
	}
	if !valid {
		return out, approvalProblem(409, reason)
	}
	locked, e := approvalReadRequestTx(ctx, tx, requestID, true)
	if e != nil {
		return out, e
	}
	if locked.Status != "approved" || locked.Version != expectedVersion {
		return out, approvalProblem(409, "실행 승인이 이미 사용되거나 변경되었습니다")
	}
	_, e = tx.Exec(ctx, `UPDATE approval_requests SET status='consumed',version=version+1,updated_at=now() WHERE id=$1`, requestID)
	out.Status = "consumed"
	out.Version++
	return out, e
}
