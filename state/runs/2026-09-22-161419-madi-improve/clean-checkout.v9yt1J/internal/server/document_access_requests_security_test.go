package server

import (
	"context"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestPostgresDocumentAccessRequestPIIAndDormantGrants(t *testing.T) {
	s, owner, ctx, _, wid := jobTestFixture(t)
	u := testJSONObject(t, owner.request("POST", "/api/v1/admin/users", map[string]any{"email": "request-security@example.test", "name": "접근 요청자", "role": "editor", "password": "Access-Request-Password-2026!"}, 200))
	owner.request("PUT", "/api/v1/workspaces/"+wid+"/members", map[string]any{"email": u["email"], "role": "editor"}, 200)
	requester := newIntegrationTestClient(t, owner.base)
	requester.request("POST", "/api/v1/auth/login", map[string]any{"email": u["email"], "password": "Access-Request-Password-2026!"}, 200)
	create := func(title string) string {
		return str(testJSONObject(t, owner.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": title, "markdown": "원문", "visibility": "private"}, 200)), "id")
	}
	requestID := func(doc string) string {
		var id string
		if e := s.DB.QueryRow(ctx, "SELECT id::text FROM document_access_requests WHERE document_id=$1 AND requester_id=$2", doc, u["id"]).Scan(&id); e != nil {
			t.Fatal(e)
		}
		return id
	}
	policy := defaultProtectionSettings()
	policy["enabled"], policy["mode"] = true, "mask"
	policy["custom_terms"] = []string{"REQUEST_SECRET_SENTINEL"}
	if _, e := s.DB.Exec(ctx, "UPDATE protection_settings SET data=$1", jsonValue(policy)); e != nil {
		t.Fatal(e)
	}
	id := create("마스킹 사유")
	requester.request("POST", "/api/v1/documents/"+id+"/access-requests", map[string]any{"workspace_id": wid, "permission": "read", "reason": "업무 REQUEST_SECRET_SENTINEL requester-secret@example.test"}, 202)
	rid := requestID(id)
	var sealed string
	if e := s.DB.QueryRow(ctx, "SELECT reason_ciphertext FROM document_access_requests WHERE id=$1", rid).Scan(&sealed); e != nil {
		t.Fatal(e)
	}
	plain, e := s.decrypt(sealed)
	if e != nil || strings.Contains(plain, "REQUEST_SECRET_SENTINEL") || strings.Contains(plain, "requester-secret@example.test") || !strings.Contains(plain, "업무") {
		t.Fatal("reason masking before encryption", plain, e)
	}
	incoming := owner.request("GET", "/api/v1/access-requests?workspace_id="+wid, nil, 200)
	if strings.Contains(string(incoming), "REQUEST_SECRET_SENTINEL") || strings.Contains(string(incoming), "requester-secret@example.test") || strings.Contains(string(incoming), sealed) {
		t.Fatal("owner reason projection leaked original or ciphertext")
	}
	policy["mode"] = "block"
	if _, e = s.DB.Exec(ctx, "UPDATE protection_settings SET data=$1", jsonValue(policy)); e != nil {
		t.Fatal(e)
	}
	unknown := newID()
	requester.request("POST", "/api/v1/documents/"+unknown+"/access-requests", map[string]any{"workspace_id": wid, "permission": "read", "reason": "REQUEST_SECRET_SENTINEL"}, 422)
	var count int
	if e = s.DB.QueryRow(ctx, "SELECT count(*) FROM document_access_requests WHERE requested_document_id=$1", unknown).Scan(&count); e != nil || count != 0 {
		t.Fatal("blocked PII receipt persisted", count, e)
	}
	policy["enabled"] = false
	if _, e = s.DB.Exec(ctx, "UPDATE protection_settings SET data=$1", jsonValue(policy)); e != nil {
		t.Fatal(e)
	}
	dormant := create("비활성 공유")
	if _, e = s.DB.Exec(ctx, "INSERT INTO document_shares(document_id,user_id,permission) VALUES($1,$2,'write')", dormant, u["id"]); e != nil {
		t.Fatal(e)
	}
	requester.request("POST", "/api/v1/documents/"+dormant+"/access-requests", map[string]any{"workspace_id": wid, "permission": "read"}, 202)
	owner.request("PUT", "/api/v1/access-requests/"+requestID(dormant), map[string]any{"revision": 1, "action": "grant", "expected_document_version": 1}, 409)
	var visibility, permission string
	var version int
	if e = s.DB.QueryRow(ctx, "SELECT d.visibility,d.version,s.permission FROM documents d JOIN document_shares s ON s.document_id=d.id WHERE d.id=$1", dormant).Scan(&visibility, &version, &permission); e != nil || visibility != "private" || version != 1 || permission != "write" {
		t.Fatal("dormant shares were reactivated or deleted", visibility, version, permission, e)
	}
	downgraded := create("요청 후 역할 변경")
	requester.request("POST", "/api/v1/documents/"+downgraded+"/access-requests", map[string]any{"workspace_id": wid, "permission": "write"}, 202)
	if _, e = s.DB.Exec(ctx, "UPDATE workspace_members SET role='viewer' WHERE workspace_id=$1 AND user_id=$2", wid, u["id"]); e != nil {
		t.Fatal(e)
	}
	owner.request("PUT", "/api/v1/access-requests/"+requestID(downgraded), map[string]any{"revision": 1, "action": "grant", "expected_document_version": 1}, 409)
	if e = s.DB.QueryRow(ctx, "SELECT count(*) FROM document_shares WHERE document_id=$1", downgraded).Scan(&count); e != nil || count != 0 {
		t.Fatal("stale requester role created grant", count, e)
	}
}

func TestPostgresDocumentAccessActorTransactionRevalidation(t *testing.T) {
	s, admin, ctx, p, wid := jobTestFixture(t)
	base, e := url.Parse(admin.base)
	if e != nil {
		t.Fatal(e)
	}
	r := httptest.NewRequest("POST", "/", nil).WithContext(context.WithValue(ctx, principalKey, p))
	for _, cookie := range admin.client.Jar.Cookies(base) {
		r.AddCookie(cookie)
	}
	cookie, e := r.Cookie("madi_session")
	if e != nil {
		t.Fatal(e)
	}
	check := func(want bool) {
		t.Helper()
		tx, e := s.DB.Begin(ctx)
		if e != nil {
			t.Fatal(e)
		}
		defer tx.Rollback(ctx)
		err := documentAccessActorTx(r, tx, wid, true)
		if (err == nil) != want {
			t.Fatal("current transaction guard", err, want)
		}
	}
	check(true)
	// Start a transaction while the cookie is valid, then expire it at a newer
	// wall-clock instant. PostgreSQL now() stays frozen at the earlier TX start.
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		t.Fatal(e)
	}
	var started, timeExpired time.Time
	if e = tx.QueryRow(ctx, "SELECT now()").Scan(&started); e != nil {
		t.Fatal(e)
	}
	if e = s.DB.QueryRow(ctx, "UPDATE sessions SET expires_at=clock_timestamp() WHERE token_hash=$1 RETURNING expires_at", digest(cookie.Value)).Scan(&timeExpired); e != nil {
		t.Fatal(e)
	}
	if !timeExpired.After(started) {
		t.Fatal("fixture did not cross transaction-start clock")
	}
	if documentAccessActorTx(r, tx, wid, true) == nil {
		t.Fatal("cookie expired during TX accepted from frozen now()")
	}
	tx.Rollback(ctx)
	if _, e = s.DB.Exec(ctx, "UPDATE sessions SET expires_at=clock_timestamp()+interval'1 day' WHERE token_hash=$1", digest(cookie.Value)); e != nil {
		t.Fatal(e)
	}
	check(true)
	if _, e = s.DB.Exec(ctx, "UPDATE workspace_members SET role='viewer' WHERE workspace_id=$1 AND user_id=$2", wid, p.ID); e != nil {
		t.Fatal(e)
	}
	check(false)
	if _, e = s.DB.Exec(ctx, "DELETE FROM workspace_members WHERE workspace_id=$1 AND user_id=$2", wid, p.ID); e != nil {
		t.Fatal(e)
	}
	check(false)
	// A stale in-memory admin principal never substitutes for current membership.
	if _, e = s.DB.Exec(ctx, "INSERT INTO workspace_members(workspace_id,user_id,role) VALUES($1,$2,'owner')", wid, p.ID); e != nil {
		t.Fatal(e)
	}
	if _, e = s.DB.Exec(ctx, "DELETE FROM sessions WHERE token_hash=$1", digest(cookie.Value)); e != nil {
		t.Fatal(e)
	}
	check(false)
}
