package server

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestPostgresSystemStatusServiceKeyMCPAndRevocation(t *testing.T) {
	s, c, wid, uid, doc := systemStatusTestSetup(t)
	systemStatusTestEnable(t, c)
	service := testJSONObject(t, c.request("POST", "/api/v1/admin/users", map[string]any{"email": "status-reporter@example.test", "name": "운영 관측 보고기", "role": "editor", "kind": "service"}, 200))
	c.request("PUT", "/api/v1/workspaces/"+wid+"/members", map[string]any{"email": service["email"], "role": "editor"}, 200)
	path := "/api/v1/documents/" + doc + "/system-status"
	card := systemStatusTestCard(uid, []string{str(service, "id")})
	c.request("PUT", path, card, 200)
	issue := func(scopes []string) *integrationTestClient {
		issued := testJSONObject(t, c.request("POST", "/api/v1/keys", map[string]any{"user_id": service["id"], "name": "명시 관측 보고", "workspace_id": wid, "scopes": scopes}, 201))
		client := newIntegrationTestClient(t, c.base)
		client.token = str(issued, "token")
		return client
	}
	read := issue([]string{"document:read"})
	rw := issue([]string{"document:read", "document:write"})
	write := issue([]string{"document:write"})
	report := systemStatusTestReport(t, rw, doc)
	read.request("POST", path+"/reports", report, 403)
	write.request("POST", path+"/reports", report, 403)
	rw.request("PUT", path, card, 403)
	get := testJSONObject(t, rw.request("POST", "/mcp", map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{"name": "get_system_status", "arguments": map[string]any{"document_id": doc}}}, 200))
	if boolean(get["result"].(map[string]any), "isError") {
		t.Fatal(get)
	}
	args := map[string]any{"document_id": doc}
	for k, v := range report {
		args[k] = v
	}
	out := testJSONObject(t, rw.request("POST", "/mcp", map[string]any{"jsonrpc": "2.0", "id": 2, "method": "tools/call", "params": map[string]any{"name": "report_system_status", "arguments": args}}, 200))
	if boolean(out["result"].(map[string]any), "isError") {
		t.Fatal(out)
	}
	state := testJSONObject(t, rw.request("GET", path, nil, 200))
	latest := state["latest_report"].(map[string]any)
	if str(latest, "actor_kind") != "service_account" || latest["actor_id"] != service["id"] {
		t.Fatal("report attribution", latest)
	}
	card["revision"] = 1
	card["reporter_ids"] = []string{}
	c.request("PUT", path, card, 200)
	rw.request("POST", path+"/reports", report, 403)
	card["revision"] = 2
	card["reporter_ids"] = []string{str(service, "id")}
	c.request("PUT", path, card, 200)
	newReport := systemStatusTestReport(t, rw, doc)
	c.request("PUT", "/api/v1/documents/"+doc, map[string]any{"version": 1, "visibility": "private"}, 200)
	rw.request("GET", path, nil, 404)
	rw.request("POST", path+"/reports", newReport, 404)
	privateAdmin := testJSONObject(t, c.request("POST", "/api/v1/admin/users", map[string]any{"email": "status-admin@example.test", "name": "다른 관리자", "role": "admin", "password": "Status-Admin-Password-2026!"}, 200))
	c.request("PUT", "/api/v1/workspaces/"+wid+"/members", map[string]any{"email": privateAdmin["email"], "role": "admin"}, 200)
	other := newIntegrationTestClient(t, c.base)
	other.request("POST", "/api/v1/auth/login", map[string]any{"email": privateAdmin["email"], "password": "Status-Admin-Password-2026!"}, 200)
	other.request("GET", path, nil, 404)
	var count int
	s.DB.QueryRow(t.Context(), `SELECT count(*) FROM system_status_reports`).Scan(&count)
	if count != 1 {
		t.Fatal("unauthorized report persisted", count)
	}
}

func TestPostgresSystemStatusLockWaitSessionExpiry(t *testing.T) {
	s, c, _, uid, doc := systemStatusTestSetup(t)
	systemStatusTestEnable(t, c)
	path := "/api/v1/documents/" + doc + "/system-status"
	c.request("PUT", path, systemStatusTestCard(uid, []string{uid}), 200)
	report := systemStatusTestReport(t, c, doc)
	blocker, e := s.DB.Begin(t.Context())
	if e != nil {
		t.Fatal(e)
	}
	defer blocker.Rollback(t.Context())
	if _, e = blocker.Exec(t.Context(), `SELECT id FROM documents WHERE id=$1 FOR UPDATE`, doc); e != nil {
		t.Fatal(e)
	}
	result := make(chan int, 1)
	failure := make(chan error, 1)
	go func() {
		body, _ := json.Marshal(report)
		req, _ := http.NewRequest("POST", c.base+path+"/reports", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Madi-Request", "1")
		response, e := c.client.Do(req)
		if e != nil {
			failure <- e
			return
		}
		defer response.Body.Close()
		io.Copy(io.Discard, response.Body)
		result <- response.StatusCode
	}()
	deadline := time.Now().Add(3 * time.Second)
	waiting := false
	for time.Now().Before(deadline) {
		if e = s.DB.QueryRow(t.Context(), `SELECT EXISTS(SELECT 1 FROM pg_stat_activity WHERE pid<>pg_backend_pid() AND wait_event_type='Lock' AND query LIKE 'SELECT workspace_id::text,version FROM documents WHERE id=$1%')`).Scan(&waiting); e != nil {
			t.Fatal(e)
		}
		if waiting {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !waiting {
		t.Fatal("report never waited on actual document lock")
	}
	if _, e = s.DB.Exec(t.Context(), `UPDATE sessions SET expires_at=clock_timestamp()-interval '1 second' WHERE user_id=$1`, uid); e != nil {
		t.Fatal(e)
	}
	if e = blocker.Commit(t.Context()); e != nil {
		t.Fatal(e)
	}
	select {
	case e := <-failure:
		t.Fatal(e)
	case code := <-result:
		if code != 403 {
			t.Fatal("expired session survived row-lock wait", code)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("report did not finish")
	}
	var count int
	s.DB.QueryRow(t.Context(), `SELECT count(*) FROM system_status_reports`).Scan(&count)
	if count != 0 {
		t.Fatal("expired report persisted")
	}
}

func TestPostgresSystemStatusDeletedCardEpochAndUppercaseIDs(t *testing.T) {
	_, c, _, uid, doc := systemStatusTestSetup(t)
	systemStatusTestEnable(t, c)
	path := "/api/v1/documents/" + doc + "/system-status"
	card := systemStatusTestCard(strings.ToUpper(uid), []string{strings.ToUpper(uid)})
	c.request("PUT", path, card, 200)
	report := systemStatusTestReport(t, c, doc)
	c.request("POST", path+"/reports", report, 201)
	c.request("DELETE", path, map[string]any{"revision": 1, "document_version": 1, "confirm": true}, 200)
	c.request("PUT", path, card, 200)
	c.request("POST", path+"/reports", report, 409)
	current := systemStatusTestReport(t, c, doc)
	if current["verification_epoch"] == report["verification_epoch"] {
		t.Fatal("recreated card shares offline epoch")
	}
	card["revision"] = 1
	card["reporter_ids"] = []string{uid, strings.ToUpper(uid)}
	c.request("PUT", path, card, 400)
}

func TestPostgresSystemStatusProtectionAndAuditRedaction(t *testing.T) {
	s, c, _, uid, doc := systemStatusTestSetup(t)
	systemStatusTestEnable(t, c)
	path := "/api/v1/documents/" + doc + "/system-status"
	c.request("PUT", path, systemStatusTestCard(uid, []string{uid}), 200)
	report := systemStatusTestReport(t, c, doc)
	report["observation"].(map[string]any)["note"] = "OPERATIONS_PROTECTED_SENTINEL"
	c.request("POST", path+"/reports", report, 201)
	next := systemStatusTestReport(t, c, doc)
	next["observation"].(map[string]any)["note"] = "OPERATIONS_PROTECTED_SENTINEL"
	policy := defaultProtectionSettings()
	policy["enabled"] = true
	policy["custom_terms"] = []string{"OPERATIONS_PROTECTED_SENTINEL"}
	for _, mode := range []string{"mask", "block"} {
		policy["mode"] = mode
		if _, e := s.DB.Exec(t.Context(), `UPDATE protection_settings SET data=$1,revision=revision+1`, jsonValue(policy)); e != nil {
			t.Fatal(e)
		}
		body := c.request("GET", path, nil, 422)
		if strings.Contains(string(body), "OPERATIONS_PROTECTED_SENTINEL") {
			t.Fatal("protected report disclosed")
		}
		c.request("POST", path+"/reports", next, 422)
	}
	var count int
	if e := s.DB.QueryRow(t.Context(), `SELECT count(*) FROM system_status_reports`).Scan(&count); e != nil || count != 1 {
		t.Fatal("masked/blocked report silently changed or persisted", e, count)
	}
	var leaked bool
	if e := s.DB.QueryRow(t.Context(), `SELECT EXISTS(SELECT 1 FROM audit_logs WHERE action LIKE 'SYSTEM_STATUS%' AND details::text LIKE '%OPERATIONS_PROTECTED_SENTINEL%')`).Scan(&leaked); e != nil || leaked {
		t.Fatal("raw report in audit", e, leaked)
	}
}
