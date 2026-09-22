package server

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestPostgresIntegratedCalendarDatesAndACL(t *testing.T) {
	_, ts := integrationTestServer(t)
	c := newIntegrationTestClient(t, ts.URL)
	c.request("POST", "/api/v1/auth/login", map[string]any{"email": "admin@example.test", "password": "Integration-Test-Password-2026!"}, 200)
	w := testJSONObject(t, c.request("POST", "/api/v1/workspaces", map[string]any{"name": "통합 일정"}, 200))
	wid := str(w, "id")
	d := testJSONObject(t, c.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "2026-09-09", "markdown": "---\ndate: 2026-09-09\n---\n- [ ] 점검 2026-09-11\n", "visibility": "private"}, 200))
	id := str(d, "id")
	db := testJSONObject(t, c.request("POST", "/api/v1/databases", map[string]any{"workspace_id": wid, "name": "배포 일정", "properties": []any{map[string]any{"id": "title", "name": "제목", "type": "text"}, map[string]any{"id": "due", "name": "예정일", "type": "date"}}}, 200))
	c.request("POST", "/api/v1/databases/"+str(db, "id")+"/rows", map[string]any{"values": map[string]any{"title": "다음 배포", "due": "2026-09-12"}}, 200)
	event := map[string]any{"workspace_id": wid, "title": "연결된 비공개 회의", "kind": "meeting", "document_id": id, "start_date": "2026-09-08", "end_date": "2026-09-10", "visibility": "workspace"}
	saved := testJSONObject(t, c.request("POST", "/api/v1/tasks/calendar/events", event, 200))
	eid := str(saved, "id")
	calendar := testJSONObject(t, c.request("GET", "/api/v1/tasks/calendar?workspace_id="+wid+"&month=2026-09", nil, 200))
	events := calendar["events"].([]any)
	if len(events) != 4 {
		t.Fatal(calendar)
	}
	independent := map[string]any{"workspace_id": wid, "title": "문서와 연결하지 않은 개인 일정", "kind": "meeting", "start_date": "2026-09-10", "end_date": "2026-09-10", "visibility": "private"}
	c.request("POST", "/api/v1/tasks/calendar/events", independent, 200)
	filtered := testJSONObject(t, c.request("GET", "/api/v1/tasks/calendar?workspace_id="+wid+"&month=2026-09&document_id="+id, nil, 200))
	if len(filtered["events"].([]any)) != 3 {
		t.Fatal("document scope must contain only source date, source task and explicitly linked meeting", filtered)
	}
	for _, raw := range filtered["events"].([]any) {
		item := raw.(map[string]any)
		if str(item, "document_id") != id || str(item, "kind") == "database" {
			t.Fatal("independent resource was implicitly associated", item)
		}
	}
	c.request("GET", "/api/v1/tasks/calendar?workspace_id="+wid+"&month=2026-09&document_id=invalid", nil, 400)
	key := testJSONObject(t, c.request("POST", "/api/v1/keys", map[string]any{"name": "문서 일정만 읽기", "workspace_id": wid, "scopes": []string{"document:read"}, "expires_in_days": 7}, 201))
	reader := newIntegrationTestClient(t, ts.URL)
	reader.token = str(key, "token")
	limited := reader.request("GET", "/api/v1/tasks/calendar?workspace_id="+wid+"&month=2026-09", nil, 200)
	if strings.Contains(string(limited), str(db, "id")) || strings.Contains(string(limited), "다음 배포") {
		t.Fatal("database calendar bypassed integration scope", string(limited))
	}
	event["version"] = 0
	c.request("PUT", "/api/v1/tasks/calendar/events/"+eid, event, 409)
	event["version"] = 1
	event["title"] = "회의 일정 변경"
	c.request("PUT", "/api/v1/tasks/calendar/events/"+eid, event, 200)
	u := testJSONObject(t, c.request("POST", "/api/v1/admin/users", map[string]any{"email": "calendar@example.test", "name": "일정 조회자", "role": "editor", "password": "Calendar-password-2026!"}, 200))
	c.request("PUT", "/api/v1/workspaces/"+wid+"/members", map[string]any{"email": u["email"], "role": "viewer"}, 200)
	viewer := newIntegrationTestClient(t, ts.URL)
	viewer.request("POST", "/api/v1/auth/login", map[string]any{"email": u["email"], "password": "Calendar-password-2026!"}, 200)
	raw := viewer.request("GET", "/api/v1/tasks/calendar?workspace_id="+wid+"&month=2026-09", nil, 200)
	if strings.Contains(string(raw), id) || strings.Contains(string(raw), eid) {
		t.Fatal("private linked meeting leaked", string(raw))
	}
	json.Unmarshal(raw, &calendar)
	if len(calendar["events"].([]any)) != 1 {
		t.Fatal(calendar)
	}
	privateScope := testJSONObject(t, viewer.request("GET", "/api/v1/tasks/calendar?workspace_id="+wid+"&month=2026-09&document_id="+id, nil, 200))
	if len(privateScope["events"].([]any)) != 0 || boolean(privateScope, "truncated") {
		t.Fatal("document-filter calendar leaked private existence or events", privateScope)
	}
	viewer.request("POST", "/api/v1/tasks/calendar/events", event, 403)
	viewer.request("DELETE", "/api/v1/tasks/calendar/events/"+eid, nil, 404)
	c.request("GET", "/api/v1/tasks/calendar?workspace_id="+wid+"&month=bad", nil, 400)
	event["start_date"] = "2026-02-31"
	c.request("POST", "/api/v1/tasks/calendar/events", event, 400)
	c.request("DELETE", "/api/v1/tasks/calendar/events/"+eid, nil, 200)
}
