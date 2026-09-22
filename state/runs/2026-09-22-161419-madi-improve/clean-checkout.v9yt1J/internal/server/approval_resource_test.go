package server

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
)

func TestPostgresApprovalGenericAtomicCallbackAndConsume(t *testing.T) {
	s, admin, reviewer, wid, _ := collaborationTestSetup(t)
	owner := testJSONObject(t, admin.request("GET", "/api/v1/auth/me", nil, 200))
	reviewUser := testJSONObject(t, reviewer.request("GET", "/api/v1/auth/me", nil, 200))
	p := &Principal{ID: str(owner, "id"), Kind: "user", Role: "admin"}
	ctx := context.Background()
	_, e := s.DB.Exec(ctx, `CREATE TABLE approval_fixture_resources(id uuid PRIMARY KEY,workspace_id uuid,owner_id uuid,version bigint,payload jsonb,completed boolean NOT NULL DEFAULT false)`)
	if e != nil {
		t.Fatal(e)
	}
	rid := newID()
	if _, e = s.DB.Exec(ctx, `INSERT INTO approval_fixture_resources(id,workspace_id,owner_id,version,payload) VALUES($1,$2,$3,1,'{"command":"approved-command","arguments":["safe-value"]}')`, rid, wid, p.ID); e != nil {
		t.Fatal(e)
	}
	failComplete := false
	delete(s.approvalAdapters, "runbook") // Replace only this isolated server's real adapter with the atomicity fixture.
	s.RegisterApprovalAdapter("runbook", ApprovalAdapter{
		Lock: func(ctx context.Context, tx pgx.Tx, actor *Principal, id string, write bool) (ApprovalResource, error) {
			var r ApprovalResource
			var raw []byte
			e := tx.QueryRow(ctx, `SELECT id::text,workspace_id::text,owner_id::text,version,payload FROM approval_fixture_resources WHERE id=$1 FOR UPDATE`, id).Scan(&r.ID, &r.WorkspaceID, &r.OwnerID, &r.Version, &raw)
			r.Kind = "runbook"
			r.Title = "격리 실행 검토"
			json.Unmarshal(raw, &r.Snapshot)
			return r, e
		},
		CanReview: func(ctx context.Context, tx pgx.Tx, uid string, r ApprovalResource) (bool, error) {
			return uid == str(reviewUser, "id"), nil
		},
		Complete: func(ctx context.Context, tx pgx.Tx, p *Principal, r ApprovalRequest, resource ApprovalResource) error {
			if _, e := tx.Exec(ctx, `UPDATE approval_fixture_resources SET completed=true WHERE id=$1`, resource.ID); e != nil {
				return e
			}
			if failComplete {
				return errors.New("test callback fails before commit")
			}
			return nil
		},
	})
	admin.request("PUT", "/api/v1/admin/settings", map[string]any{"approval_enabled": true}, 200)
	start := func() ApprovalRequest {
		t.Helper()
		tx, e := s.DB.Begin(ctx)
		if e != nil {
			t.Fatal(e)
		}
		defer tx.Rollback(ctx)
		r, e := s.StartApprovalTx(ctx, tx, p, "runbook", rid, "검토 후 정확한 명령만 실행")
		if e != nil {
			t.Fatal(e)
		}
		if e = tx.Commit(ctx); e != nil {
			t.Fatal(e)
		}
		return r
	}
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		t.Fatal(e)
	}
	_, e = s.StartApprovalTx(ctx, tx, p, "runbook", rid, "")
	tx.Rollback(ctx)
	var failure *approvalFailure
	if !errors.As(e, &failure) || failure.Status != 409 {
		t.Fatalf("runbook without explicit policy: %v", e)
	}
	admin.request("POST", "/api/v1/admin/approval/policies", map[string]any{"name": "격리 실행 승인", "workspace_id": wid, "resource_kind": "runbook", "stages": []any{map[string]any{"name": "실행 담당자 검토", "mode": "all", "gates": []any{map[string]any{"name": "별도 검토자", "kind": "user", "id": str(reviewUser, "id")}}}}}, 200)
	r := start()
	failComplete = true
	reviewer.request("POST", "/api/v1/approvals/requests/"+r.ID+"/decisions", map[string]any{"action": "approve", "request_version": r.Version}, 500)
	var status string
	var completed bool
	var count int
	if e = s.DB.QueryRow(ctx, `SELECT status FROM approval_requests WHERE id=$1`, r.ID).Scan(&status); e != nil {
		t.Fatal(e)
	}
	if e = s.DB.QueryRow(ctx, `SELECT completed FROM approval_fixture_resources WHERE id=$1`, rid).Scan(&completed); e != nil {
		t.Fatal(e)
	}
	if e = s.DB.QueryRow(ctx, `SELECT count(*) FROM approval_decisions WHERE request_id=$1`, r.ID).Scan(&count); e != nil {
		t.Fatal(e)
	}
	if status != "pending" || completed || count != 0 {
		t.Fatal("callback failure did not rollback decision and effect")
	}
	failComplete = false
	result := testJSONObject(t, reviewer.request("POST", "/api/v1/approvals/requests/"+r.ID+"/decisions", map[string]any{"action": "approve", "request_version": r.Version}, 200))
	if str(result, "status") != "approved" {
		t.Fatal("generic request not approved")
	}
	consume := func(commit bool) error {
		tx, e := s.DB.Begin(ctx)
		if e != nil {
			return e
		}
		defer tx.Rollback(ctx)
		_, e = s.ConsumeApprovalTx(ctx, tx, p, "runbook", rid, r.ID, int64(number(result, "version", 0)))
		if e != nil {
			return e
		}
		if commit {
			return tx.Commit(ctx)
		}
		return nil
	}
	// A job enqueue failure would roll back the consume together with the job.
	if e = consume(false); e != nil {
		t.Fatal(e)
	}
	if e = consume(true); e != nil {
		t.Fatal(e)
	}
	if e = consume(true); !errors.As(e, &failure) || failure.Status != 409 {
		t.Fatalf("reused execution approval: %v", e)
	}
	// Exact approved payload is mandatory, even if a buggy external adapter
	// forgot to increment its source version.
	r = start()
	result = testJSONObject(t, reviewer.request("POST", "/api/v1/approvals/requests/"+r.ID+"/decisions", map[string]any{"action": "approve", "request_version": r.Version}, 200))
	if _, e = s.DB.Exec(ctx, `UPDATE approval_fixture_resources SET payload='{"command":"different-command"}' WHERE id=$1`, rid); e != nil {
		t.Fatal(e)
	}
	if e = consume(true); !errors.As(e, &failure) || failure.Status != 409 {
		t.Fatalf("modified execution payload: %v", e)
	}
}
