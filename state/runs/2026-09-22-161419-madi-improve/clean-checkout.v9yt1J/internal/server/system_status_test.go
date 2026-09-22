package server

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"slices"
	"sort"
	"sync"
	"testing"
	"time"
)

func systemStatusTestSetup(t *testing.T) (*Server, *integrationTestClient, string, string, string) {
	t.Helper()
	s, ts := integrationTestServer(t)
	if e := s.migrateSystemStatus(t.Context()); e != nil {
		t.Fatal(e)
	}
	if !slices.Contains(s.apiRoutes, "GET /api/v1/documents/{id}/system-status") {
		s.registerSystemStatus()
	}
	c := newIntegrationTestClient(t, ts.URL)
	me := testJSONObject(t, c.request("POST", "/api/v1/auth/login", map[string]any{"email": "admin@example.test", "password": "Integration-Test-Password-2026!"}, 200))
	wid := str(testJSONObject(t, c.request("POST", "/api/v1/workspaces", map[string]any{"name": "운영 관측 검증"}, 200)), "id")
	doc := str(testJSONObject(t, c.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "GPU 운영 기준", "markdown": "# 변하지 않는 정본\n\n기대 배포 정책", "visibility": "workspace"}, 200)), "id")
	return s, c, wid, str(me, "id"), doc
}
func systemStatusTestEnable(t *testing.T, c *integrationTestClient) {
	t.Helper()
	state := testJSONObject(t, c.request("GET", "/api/v1/admin/system-status/policy", nil, 200))
	policy := state["policy"].(map[string]any)
	policy["enabled"] = true
	policy["confirm"] = true
	c.request("PUT", "/api/v1/admin/system-status/policy", policy, 200)
}
func systemStatusTestCard(uid string, reporters []string) map[string]any {
	return map[string]any{"revision": 0, "document_version": 1, "owner_id": uid, "reporter_ids": reporters, "ttl_seconds": 300, "expected": map[string]any{"name": "GPU 지식 서비스", "environment": "폐쇄망 검증", "version": "v1.0.0", "deployment": "release-blue", "note": "기대값일 뿐 배포 확인이 아닙니다"}, "confirm": true}
}
func systemStatusTestReport(t *testing.T, c *integrationTestClient, doc string) map[string]any {
	t.Helper()
	state := testJSONObject(t, c.request("GET", "/api/v1/documents/"+doc+"/system-status", nil, 200))
	card := state["card"].(map[string]any)
	return map[string]any{"request_id": newID(), "card_revision": card["revision"], "report_revision": card["report_revision"], "verification_epoch": card["verification_epoch"], "document_version": card["document_version"], "observed_at": time.Now().UTC().Add(-2 * time.Second).Format(time.RFC3339Nano), "observation": map[string]any{"version": "v0.9.0", "deployment": "release-green", "health": "degraded", "note": "명시 보고 주체가 직접 관측한 합성 자료"}, "confirm": true}
}
func TestSystemStatusFreshness(t *testing.T) {
	now := time.Date(2026, 9, 8, 9, 0, 0, 0, time.UTC)
	p := systemStatusPolicy{Enabled: true, MaxTTL: 300}
	c := systemStatusCard{Revision: 1, Epoch: 2, DocumentVersion: 7, TTL: 300, Expected: systemStatusExpected{Version: "v1", Deployment: "blue"}}
	r := systemStatusReport{CardRevision: 1, Epoch: 2, DocumentVersion: 7, Observed: now.Add(-290 * time.Second), Received: now.Add(-time.Second), Observation: systemStatusObservation{Version: "v1", Deployment: "blue", Health: "unknown"}}
	got := systemStatusFreshness(c, p, 7, &r, now)
	if str(got, "state") != "reported_match" || boolean(got, "independently_verified") {
		t.Fatal(got)
	}
	if got["expires_at"].(time.Time) != now.Add(10*time.Second) {
		t.Fatal("receipt extended observation TTL", got)
	}
	if str(systemStatusFreshness(c, p, 7, &r, now.Add(11*time.Second)), "state") != "stale" {
		t.Fatal("stale observation trusted")
	}
	r.Observed = now.Add(30 * time.Second)
	r.Received = now.Add(-299 * time.Second)
	if systemStatusFreshness(c, p, 7, &r, now)["expires_at"].(time.Time) != now.Add(time.Second) {
		t.Fatal("future skew extended received TTL")
	}
	r.Epoch = 1
	if str(systemStatusFreshness(c, p, 7, &r, now), "state") != "baseline_changed" {
		t.Fatal("restored observation trusted")
	}
	if str(systemStatusFreshness(c, p, 8, &r, now), "state") != "document_changed" {
		t.Fatal("changed document trusted")
	}
}
func TestPostgresSystemStatusReportReplayDriftAndRestore(t *testing.T) {
	s, c, _, uid, doc := systemStatusTestSetup(t)
	path := "/api/v1/documents/" + doc + "/system-status"
	card := systemStatusTestCard(uid, []string{uid})
	c.request("PUT", path, card, 409)
	systemStatusTestEnable(t, c)
	c.request("PUT", path, card, 200)
	state := testJSONObject(t, c.request("GET", path, nil, 200))
	if str(state["summary"].(map[string]any), "state") != "unobserved" {
		t.Fatal("registered expectation became observed", state)
	}
	report := systemStatusTestReport(t, c, doc)
	created := testJSONObject(t, c.request("POST", path+"/reports", report, 201))
	again := testJSONObject(t, c.request("POST", path+"/reports", report, 200))
	if !boolean(again, "replayed") || again["id"] != created["id"] {
		t.Fatal("retry changed report", again)
	}
	state = testJSONObject(t, c.request("GET", path, nil, 200))
	summary := state["summary"].(map[string]any)
	if str(summary, "state") != "drift" || !boolean(summary, "fresh") || len(summary["drift_fields"].([]any)) != 2 || boolean(summary, "independently_verified") {
		t.Fatal(summary)
	}
	if len(state["reports"].([]any)) != 1 {
		t.Fatal("duplicate report")
	}
	var raw, markdown string
	var version int
	if e := s.DB.QueryRow(t.Context(), `SELECT ciphertext FROM system_status_reports WHERE id=$1`, created["id"]).Scan(&raw); e != nil {
		t.Fatal(e)
	}
	if bytes.Contains([]byte(raw), []byte("release-green")) {
		t.Fatal("report at rest not encrypted")
	}
	s.DB.QueryRow(t.Context(), `SELECT version,markdown FROM documents WHERE id=$1`, doc).Scan(&version, &markdown)
	if version != 1 || markdown != "# 변하지 않는 정본\n\n기대 배포 정책" {
		t.Fatal("observation modified canonical document")
	}
	future := systemStatusTestReport(t, c, doc)
	future["observed_at"] = time.Now().Add(2 * time.Minute).Format(time.RFC3339)
	c.request("POST", path+"/reports", future, 400)
	older := systemStatusTestReport(t, c, doc)
	older["observed_at"] = time.Now().Add(-time.Hour).Format(time.RFC3339)
	c.request("POST", path+"/reports", older, 409)
	card["revision"] = 1
	c.request("PUT", path, card, 200)
	c.request("POST", path+"/reports", report, 409)
	state = testJSONObject(t, c.request("GET", path, nil, 200))
	if str(state["summary"].(map[string]any), "state") != "baseline_changed" {
		t.Fatal(state)
	}
	report = systemStatusTestReport(t, c, doc)
	c.request("POST", path+"/reports", report, 201)
	tx, e := s.DB.Begin(t.Context())
	if e != nil {
		t.Fatal(e)
	}
	defer tx.Rollback(t.Context())
	if e = s.invalidateSystemStatusRestoreTx(t.Context(), tx); e != nil {
		t.Fatal(e)
	}
	if e = tx.Commit(t.Context()); e != nil {
		t.Fatal(e)
	}
	c.request("POST", path+"/reports", report, 409)
	systemStatusTestEnable(t, c)
	c.request("POST", path+"/reports", report, 409)
	state = testJSONObject(t, c.request("GET", path, nil, 200))
	if str(state["summary"].(map[string]any), "state") != "baseline_changed" {
		t.Fatal("restored reports became fresh", state)
	}
	c.request("PUT", "/api/v1/documents/"+doc, map[string]any{"version": 1, "markdown": "변경된 실제 원문"}, 200)
	state = testJSONObject(t, c.request("GET", path, nil, 200))
	if str(state["summary"].(map[string]any), "state") != "document_changed" {
		t.Fatal(state)
	}
	if _, e = s.DB.Exec(t.Context(), `UPDATE system_status_reports SET received_at=now()-interval '100 days'`); e != nil {
		t.Fatal(e)
	}
	removed, e := s.expireSystemStatusReports(t.Context())
	if e != nil || removed != 2 {
		t.Fatal(removed, e)
	}
}
func TestPostgresSystemStatusConcurrentReportCAS(t *testing.T) {
	s, c, _, uid, doc := systemStatusTestSetup(t)
	systemStatusTestEnable(t, c)
	path := "/api/v1/documents/" + doc + "/system-status"
	c.request("PUT", path, systemStatusTestCard(uid, []string{uid}), 200)
	report := systemStatusTestReport(t, c, doc)
	statuses := make(chan int, 2)
	failures := make(chan error, 2)
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			input := map[string]any{}
			for k, v := range report {
				input[k] = v
			}
			input["request_id"] = newID()
			body, _ := json.Marshal(input)
			r, _ := http.NewRequest("POST", c.base+path+"/reports", bytes.NewReader(body))
			r.Header.Set("Content-Type", "application/json")
			r.Header.Set("X-Madi-Request", "1")
			response, e := c.client.Do(r)
			if e != nil {
				failures <- e
				return
			}
			defer response.Body.Close()
			io.Copy(io.Discard, response.Body)
			statuses <- response.StatusCode
		}()
	}
	wg.Wait()
	close(statuses)
	close(failures)
	for e := range failures {
		t.Fatal(e)
	}
	got := []int{}
	for code := range statuses {
		got = append(got, code)
	}
	sort.Ints(got)
	if len(got) != 2 || got[0] != 201 || got[1] != 409 {
		t.Fatal(got)
	}
	var count int
	s.DB.QueryRow(t.Context(), `SELECT count(*) FROM system_status_reports`).Scan(&count)
	if count != 1 {
		t.Fatal("concurrent reports not atomic", count)
	}
}
