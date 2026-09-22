package server

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
)

// Save and approval decisions take locks in the same order: resource, settings,
// policy, request. A concurrent administrator cannot enable approvals between
// the status check and a direct publication.
func approvalSaveStatusTx(ctx context.Context, tx pgx.Tx, id string, before, changes map[string]any, status string) (string, error) {
	var locked string
	if e := tx.QueryRow(ctx, `SELECT id::text FROM documents WHERE id=$1 AND deleted_at IS NULL FOR UPDATE`, id).Scan(&locked); e != nil {
		return status, e
	}
	var raw []byte
	if e := tx.QueryRow(ctx, `SELECT data FROM settings WHERE id=1 FOR SHARE`).Scan(&raw); e != nil {
		return status, e
	}
	cfg := defaultSettings()
	if e := json.Unmarshal(raw, &cfg); e != nil {
		return status, e
	}
	if !boolean(cfg, "approval_enabled") {
		return status, nil
	}
	if status == "published" && str(before, "status") != "published" {
		return status, approvalProblem(403, "검토 승인 후 게시할 수 있습니다")
	}
	after := approvalDocumentSnapshot(before)
	for key, value := range changes {
		// PostgreSQL uses null, while mutation inputs represent no parent/space
		// as an empty string. Normalize before comparing the immutable snapshot.
		if (key == "space_id" || key == "parent_id") && value == "" {
			value = nil
		}
		after[key] = value
	}
	if approvalSnapshotHash(after) != approvalSnapshotHash(approvalDocumentSnapshot(before)) {
		status = "draft"
	}
	return status, nil
}

func (s *Server) canApproveDocument(ctx context.Context, p *Principal, id string) (bool, error) {
	if p.TokenID != "" || p.ScopeRestricted || p.Kind != "user" {
		return false, nil
	}
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return false, e
	}
	defer tx.Rollback(ctx)
	resource, e := s.approvalAdapters["document"].Lock(ctx, tx, p, id, false)
	if errors.Is(e, pgx.ErrNoRows) {
		return false, nil
	}
	if e != nil {
		return false, e
	}
	var requestID string
	e = tx.QueryRow(ctx, `SELECT id::text FROM approval_requests WHERE resource_kind='document' AND resource_id=$1 AND status='pending'`, id).Scan(&requestID)
	if errors.Is(e, pgx.ErrNoRows) {
		return false, nil
	}
	if e != nil {
		return false, e
	}
	request, e := approvalReadRequestTx(ctx, tx, requestID, false)
	if e != nil {
		return false, e
	}
	valid, _, e := approvalCurrentTx(ctx, tx, request, resource)
	var denied *approvalFailure
	if errors.As(e, &denied) && denied.Status == 404 {
		return false, nil
	}
	if e != nil || !valid {
		return false, e
	}
	gates, e := s.approvalEligibleGatesTx(ctx, tx, request, p, resource)
	return len(gates) > 0, e
}
