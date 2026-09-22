package server

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func impactExceptionFixture(t *testing.T) (*Server, *integrationTestClient, *integrationTestClient, string, string, string, string) {
	t.Helper()
	s, c, _, _, wid := jobTestFixture(t)
	reviewer := testJSONObject(t, c.request("POST", "/api/v1/admin/users", map[string]any{"email": "exception-reviewer@example.test", "name": "예외 검토자", "role": "editor", "password": "Exception-Password-2026!"}, 200))
	c.request("PUT", "/api/v1/workspaces/"+wid+"/members", map[string]any{"email": reviewer["email"], "role": "viewer"}, 200)
	rc := newIntegrationTestClient(t, c.base)
	rc.request("POST", "/api/v1/auth/login", map[string]any{"email": reviewer["email"], "password": "Exception-Password-2026!"}, 200)
	create := func(title string) string {
		return str(testJSONObject(t, c.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": title, "markdown": "고정된 운영 기준", "visibility": "workspace"}, 200)), "id")
	}
	source, target := create("운영 정책"), create("설치 절차")
	c.request("POST", "/api/v1/documents/"+target+"/relations", map[string]any{"target_id": source, "type": "policy", "expected_version": 1}, 200)
	c.request("PUT", "/api/v1/admin/settings", map[string]any{"approval_enabled": true}, 200)
	review := str(testJSONObject(t, c.request("POST", "/api/v1/documents/"+source+"/impact-reviews", map[string]any{"source_version": 1, "target_id": target, "target_version": 1, "relation_type": "policy"}, 201)), "id")
	c.request("PUT", "/api/v1/knowledge/impact-reviews/"+review, map[string]any{"revision": 1, "target_version": 1, "status": "exception_requested", "note": "정해진 기한 동안 대체 절차 검토 요청"}, 200)
	return s, c, rc, wid, source, target, review
}
func impactExceptionPolicy(t *testing.T, c *integrationTestClient, wid string) map[string]any {
	t.Helper()
	users := []map[string]any{}
	json.Unmarshal(c.request("GET", "/api/v1/admin/users", nil, 200), &users)
	var uid string
	for _, u := range users {
		if str(u, "email") == "exception-reviewer@example.test" {
			uid = str(u, "id")
		}
	}
	return testJSONObject(t, c.request("POST", "/api/v1/admin/approval/policies", map[string]any{"workspace_id": wid, "resource_kind": "impact_exception", "name": "변경 영향 예외 승인", "enabled": true, "stages": []any{map[string]any{"name": "독립 검토", "mode": "all", "gates": []any{map[string]any{"name": "지정 검토자", "kind": "user", "id": uid}}}}}, 200))
}
func impactExceptionInput() map[string]any {
	return map[string]any{"review_revision": 2, "source_version": 1, "target_version": 1, "reason": "대체 절차를 적용하고 기한 내 정상화합니다. EXCEPTION_PRIVATE_REASON", "valid_until": time.Now().Add(24 * time.Hour).UTC().Format(time.RFC3339Nano), "confirm": true}
}
func TestPostgresImpactExceptionExplicitPolicyImmutableDecisionAndExpiry(t *testing.T) {
	s, c, rc, wid, source, target, review := impactExceptionFixture(t)
	path := "/api/v1/knowledge/impact-reviews/" + review + "/exceptions"
	in := impactExceptionInput()
	c.request("POST", path, in, 409)
	impactExceptionPolicy(t, c, wid)
	created := testJSONObject(t, c.request("POST", path, in, 201))
	id, aid := str(created, "id"), str(created, "approval_id")
	c.request("POST", path, in, 409)
	var stored string
	var snapshots []byte
	if e := s.DB.QueryRow(t.Context(), `SELECT e.reason_ciphertext,a.snapshot FROM knowledge_impact_exceptions e JOIN approval_requests a ON a.id=e.approval_id WHERE e.id=$1`, id).Scan(&stored, &snapshots); e != nil || strings.Contains(stored, "EXCEPTION_PRIVATE_REASON") || strings.Contains(string(snapshots), "EXCEPTION_PRIVATE_REASON") {
		t.Fatal("raw reason persisted outside ciphertext", e)
	}
	detail := testJSONObject(t, rc.request("GET", "/api/v1/knowledge/impact-exceptions/"+id, nil, 200))
	if boolean(detail, "effective") || str(detail, "reason") != in["reason"] {
		t.Fatal(detail)
	}
	decide := func(client *integrationTestClient, action string, want int) {
		client.request("POST", "/api/v1/approvals/requests/"+aid+"/decisions", map[string]any{"action": action, "request_version": 1, "comment": "현재 버전과 기한 확인"}, want)
	}
	decide(c, "approve", 403)
	decide(rc, "approve", 200)
	decide(rc, "approve", 409)
	detail = testJSONObject(t, c.request("GET", "/api/v1/knowledge/impact-exceptions/"+id, nil, 200))
	if !boolean(detail, "effective") || str(detail, "status") != "approved" {
		t.Fatal(detail)
	}
	upper := testJSONObject(t, c.request("GET", "/api/v1/knowledge/impact-reviews/"+strings.ToUpper(review)+"/exceptions", nil, 200))
	if !boolean(upper["items"].([]any)[0].(map[string]any), "effective") {
		t.Fatal("UUID spelling changed immutable snapshot", upper)
	}
	for _, doc := range []string{source, target} {
		v := testJSONObject(t, c.request("GET", "/api/v1/documents/"+doc, nil, 200))
		if number(v, "version", 0) != 1 || str(v, "markdown") != "고정된 운영 기준" {
			t.Fatal("exception changed canonical document", v)
		}
	}
	// Relation resurrection does not resurrect an old exception authorization.
	if _, e := s.DB.Exec(t.Context(), `UPDATE document_relations SET created_at=created_at+interval '1 second' WHERE source_id=$1 AND target_id=$2`, target, source); e != nil {
		t.Fatal(e)
	}
	detail = testJSONObject(t, c.request("GET", "/api/v1/knowledge/impact-exceptions/"+id, nil, 200))
	if boolean(detail, "effective") {
		t.Fatal("changed relation resurrected exception")
	}
	c.request("PUT", "/api/v1/admin/settings", map[string]any{"approval_enabled": false}, 200)
	rc.request("GET", "/api/v1/knowledge/impact-exceptions/"+id, nil, 404)
	c.request("PUT", "/api/v1/admin/settings", map[string]any{"approval_enabled": true}, 200)
	detail = testJSONObject(t, c.request("GET", "/api/v1/knowledge/impact-exceptions/"+id, nil, 200))
	if boolean(detail, "effective") {
		t.Fatal("settings off/on resurrected exception")
	}
}
func TestPostgresImpactExceptionRejectionResubmitCASPrivateAndRestore(t *testing.T) {
	s, c, rc, wid, source, target, review := impactExceptionFixture(t)
	impactExceptionPolicy(t, c, wid)
	path := "/api/v1/knowledge/impact-reviews/" + review + "/exceptions"
	first := testJSONObject(t, c.request("POST", path, impactExceptionInput(), 201))
	rc.request("POST", "/api/v1/approvals/requests/"+str(first, "approval_id")+"/decisions", map[string]any{"action": "reject", "request_version": 1, "comment": "대체 절차 보강 필요"}, 200)
	second := testJSONObject(t, c.request("POST", path, impactExceptionInput(), 201))
	c.request("PUT", "/api/v1/documents/"+target, map[string]any{"version": 1, "markdown": "변경된 절차"}, 200)
	rc.request("POST", "/api/v1/approvals/requests/"+str(second, "approval_id")+"/decisions", map[string]any{"action": "approve", "request_version": 1}, 409)
	c.request("PUT", "/api/v1/knowledge/impact-reviews/"+review, map[string]any{"revision": 2, "target_version": 2, "status": "exception_requested", "note": "수정된 대체 절차 검토"}, 200)
	next := impactExceptionInput()
	next["review_revision"], next["target_version"] = 3, 2
	third := testJSONObject(t, c.request("POST", path, next, 201))
	rc.request("POST", "/api/v1/approvals/requests/"+str(third, "approval_id")+"/decisions", map[string]any{"action": "approve", "request_version": 1}, 200)
	tx, e := s.DB.Begin(t.Context())
	if e != nil {
		t.Fatal(e)
	}
	if e = invalidateImpactExceptionsRestoreTx(t.Context(), tx); e != nil {
		t.Fatal(e)
	}
	if e = tx.Commit(t.Context()); e != nil {
		t.Fatal(e)
	}
	value := testJSONObject(t, c.request("GET", "/api/v1/knowledge/impact-exceptions/"+str(third, "id"), nil, 200))
	if boolean(value, "effective") || str(value, "status") != "cancelled" {
		t.Fatal("restore did not invalidate", value)
	}
	c.request("PUT", "/api/v1/documents/"+source, map[string]any{"version": 1, "visibility": "private"}, 200)
	rc.request("GET", path, nil, 404)
	rc.request("GET", "/api/v1/knowledge/impact-exceptions/"+str(third, "id"), nil, 404)
	rc.request("GET", "/api/v1/approvals/requests/"+str(third, "approval_id"), nil, 404)
}
func TestPostgresImpactExceptionConcurrentRequestSingleReceipt(t *testing.T) {
	s, c, _, wid, _, _, review := impactExceptionFixture(t)
	impactExceptionPolicy(t, c, wid)
	config := s.DB.Config()
	config.MaxConns = 4
	pool, e := pgxpool.NewWithConfig(t.Context(), config)
	if e != nil {
		t.Fatal(e)
	}
	original := s.DB
	s.DB = pool
	defer func() { s.DB = original; pool.Close() }()
	body, _ := json.Marshal(impactExceptionInput())
	var wg sync.WaitGroup
	codes := make(chan int, 6)
	for range 6 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req, _ := http.NewRequest("POST", c.base+"/api/v1/knowledge/impact-reviews/"+review+"/exceptions", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Madi-Request", "1")
			res, e := c.client.Do(req)
			if e != nil {
				codes <- 0
				return
			}
			io.Copy(io.Discard, res.Body)
			res.Body.Close()
			codes <- res.StatusCode
		}()
	}
	wg.Wait()
	close(codes)
	success := 0
	for code := range codes {
		if code == 201 {
			success++
		} else if code != 409 {
			t.Fatalf("unexpected concurrent status %d", code)
		}
	}
	if success != 1 {
		t.Fatal(success)
	}
	var exceptions, requests int
	if e := s.DB.QueryRow(t.Context(), `SELECT (SELECT count(*) FROM knowledge_impact_exceptions),(SELECT count(*) FROM approval_requests WHERE resource_kind='impact_exception')`).Scan(&exceptions, &requests); e != nil || exceptions != 1 || requests != 1 {
		t.Fatal(e, exceptions, requests)
	}
}

func TestPostgresImpactExceptionCurrentProtectionKeysAndTimeExpiry(t *testing.T) {
	s, c, rc, wid, _, _, review := impactExceptionFixture(t)
	impactExceptionPolicy(t, c, wid)
	path := "/api/v1/knowledge/impact-reviews/" + review + "/exceptions"
	in := impactExceptionInput()
	in["reason"] = "예외 근거에 보호표현을 포함합니다"
	in["valid_until"] = time.Now().Add(2 * time.Second).UTC().Format(time.RFC3339Nano)
	created := testJSONObject(t, c.request("POST", path, in, 201))
	id, aid := str(created, "id"), str(created, "approval_id")
	issued := testJSONObject(t, c.request("POST", "/api/v1/keys", map[string]any{"name": "예외 읽기 전용", "workspace_id": wid, "scopes": []string{"document:read", "document:write"}}, 201))
	token := newIntegrationTestClient(t, c.base)
	token.token = str(issued, "token")
	token.request("GET", "/api/v1/knowledge/impact-exceptions/"+id, nil, 200)
	token.request("POST", path, in, 403)
	token.request("POST", "/api/v1/approvals/requests/"+aid+"/decisions", map[string]any{"action": "approve", "request_version": 1}, 403)
	mcp := testJSONObject(t, token.request("POST", "/mcp", map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{"name": "get_impact_exception", "arguments": map[string]any{"exception_id": id}}}, 200))
	if boolean(mcp["result"].(map[string]any), "isError") {
		t.Fatal(mcp)
	}
	expires, _ := time.Parse(time.RFC3339Nano, in["valid_until"].(string))
	if delay := time.Until(expires) + 20*time.Millisecond; delay > 0 {
		time.Sleep(delay)
	}
	value := testJSONObject(t, rc.request("GET", "/api/v1/knowledge/impact-exceptions/"+id, nil, 200))
	if boolean(value, "current") || boolean(value, "effective") {
		t.Fatal("expired exception remains usable", value)
	}
	// Current candidate validation removes expired approval gates even though the
	// immutable content hash itself does not depend on wall-clock time.
	detail := testJSONObject(t, rc.request("GET", "/api/v1/approvals/requests/"+aid, nil, 200))
	if len(detail["eligible_gates"].([]any)) != 0 || !boolean(detail, "stale") {
		t.Fatal("expired request still had eligible gates")
	}
	status := testJSONObject(t, rc.request("GET", "/api/v1/approvals/resources/impact_exception/"+id, nil, 200))
	if !boolean(status, "stale") {
		t.Fatal("expired generic resource status claimed current", status)
	}
	rc.request("POST", "/api/v1/approvals/requests/"+aid+"/decisions", map[string]any{"action": "approve", "request_version": 1}, 403)
	policy := defaultProtectionSettings()
	policy["enabled"], policy["mode"], policy["custom_terms"] = true, "block", []string{"보호표현"}
	if _, e := s.DB.Exec(t.Context(), `UPDATE protection_settings SET data=$1,revision=revision+1`, jsonValue(policy)); e != nil {
		t.Fatal(e)
	}
	for _, url := range []string{path, "/api/v1/knowledge/impact-exceptions/" + id, "/api/v1/approvals/requests/" + aid} {
		raw := c.request("GET", url, nil, 422)
		if strings.Contains(string(raw), "보호표현") {
			t.Fatal("PII error leaked original")
		}
	}
	var audit string
	if e := s.DB.QueryRow(t.Context(), `SELECT coalesce(jsonb_agg(details),'[]')::text FROM audit_logs WHERE action LIKE '%EXCEPTION%'`).Scan(&audit); e != nil || strings.Contains(audit, "보호표현") {
		t.Fatal("reason leaked into audit", e, audit)
	}
}
