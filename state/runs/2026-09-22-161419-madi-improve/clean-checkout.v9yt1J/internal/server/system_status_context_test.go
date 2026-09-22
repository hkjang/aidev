package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
)

func systemStatusInstallTest(t *testing.T, s *Server) {
	t.Helper()
	if e := s.migrateSystemStatus(t.Context()); e != nil {
		t.Fatal(e)
	}
	if !slices.Contains(s.apiRoutes, "GET /api/v1/documents/{id}/system-status") {
		s.registerSystemStatus()
	}
}
func TestPostgresSystemStatusRealRunbookContextNotObservation(t *testing.T) {
	s, admin, _, _, _, doc, _ := runbookTestSetup(t)
	systemStatusInstallTest(t, s)
	systemStatusTestEnable(t, admin)
	me := testJSONObject(t, admin.request("GET", "/api/v1/auth/me", nil, 200))
	owner := str(me, "id")
	if owner == "" {
		t.Fatal(me)
	}
	admin.request("PUT", "/api/v1/documents/"+doc+"/system-status", systemStatusTestCard(owner, []string{owner}), 200)
	plan := runbookPrepare(t, admin, doc, "validate")
	queued := runbookConfirm(t, admin, plan)
	job := runbookJobForTest(t, s, str(queued, "job_id"))
	if _, e := s.executeRunbookJob(context.Background(), job); e != nil {
		t.Fatal(e)
	}
	state := testJSONObject(t, admin.request("GET", "/api/v1/documents/"+doc+"/system-status", nil, 200))
	runbook := state["context"].(map[string]any)["runbook"].(map[string]any)
	if str(runbook, "status") != "succeeded" || str(state["summary"].(map[string]any), "state") != "unobserved" || state["latest_report"] != nil {
		t.Fatal("runbook execution became deployment observation", state)
	}
	if strings.Contains(string(jsonValue(state)), "test-runner-secret") || runbook["snapshot"] != nil || runbook["last_error"] != nil {
		t.Fatal("runner context exposed payload")
	}
}
func TestPostgresSystemStatusRealConnectorContextNotObservation(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer connection-secret-not-public" {
			w.WriteHeader(401)
			return
		}
		json.NewEncoder(w).Encode(map[string]any{"items": []map[string]any{{"id": "ops-system", "title": "원격 운영 명세", "content": "# 수집된 원격 명세\n실제 시스템 버전 확인은 별도 보고입니다."}}})
	}))
	defer upstream.Close()
	s, c, _, _, input := connectorTestSetup(t, upstream.URL)
	systemStatusInstallTest(t, s)
	systemStatusTestEnable(t, c)
	connector := testJSONObject(t, c.request("POST", "/api/v1/connectors", input, 200))
	id := str(connector, "id")
	connectorRunResult(t, s, c, id, "run")
	var doc string
	var version int
	if e := s.DB.QueryRow(t.Context(), `SELECT d.id::text,d.version FROM documents d JOIN connector_records x ON x.document_id=d.id WHERE x.connector_id=$1`, id).Scan(&doc, &version); e != nil {
		t.Fatal(e)
	}
	// Imported documents can be owned by a service account. The operational
	// responsible user is explicitly selected, not inferred from that ownership.
	owner := str(testJSONObject(t, c.request("GET", "/api/v1/auth/me", nil, 200)), "id")
	card := systemStatusTestCard(owner, []string{owner})
	card["document_version"] = version
	c.request("PUT", "/api/v1/documents/"+doc+"/system-status", card, 200)
	state := testJSONObject(t, c.request("GET", "/api/v1/documents/"+doc+"/system-status", nil, 200))
	context := state["context"].(map[string]any)
	records := context["connector_records"].([]any)
	if len(records) != 1 || records[0].(map[string]any)["last_seen_at"] == nil || state["latest_report"] != nil || str(state["summary"].(map[string]any), "state") != "unobserved" {
		t.Fatal("connector collection became deployment observation", state)
	}
	for _, secret := range []string{"connection-secret-not-public", upstream.URL, "source_url", "원격 명세\n실제"} {
		if strings.Contains(string(jsonValue(state)), secret) {
			t.Fatal("connector context leaked", secret)
		}
	}
	if _, e := s.DB.Exec(t.Context(), `UPDATE users SET disabled=true WHERE id=$1`, input["service_account_id"]); e != nil {
		t.Fatal(e)
	}
	state = testJSONObject(t, c.request("GET", "/api/v1/documents/"+doc+"/system-status", nil, 200))
	if len(state["context"].(map[string]any)["connector_records"].([]any)) != 0 {
		t.Fatal("revoked connector context remained")
	}
}
