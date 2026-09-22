package server

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestMarkdownTaskReferencesAreExplicitAndStable(t *testing.T) {
	id := newID()
	ref := "[작업](/app/tasks?task=" + id + ")"
	idx := indexMarkdown("- [ ] 운영 점검 " + ref + "\n- [x] 또 다른 점검\n\n```md\n- [ ] 예제 " + ref + "\n```\n")
	if len(idx.Tasks) != 2 || idx.Tasks[0].ID != id || idx.Tasks[0].Ambiguous || strings.Contains(idx.Tasks[0].Text, "/app/tasks") {
		t.Fatal(idx.Tasks)
	}
	idx = indexMarkdown("- [ ] 운영 점검 " + ref + "\n- [ ] 복제한 점검 " + ref)
	if len(idx.Tasks) != 2 || !idx.Tasks[0].Ambiguous || !idx.Tasks[1].Ambiguous {
		t.Fatal(idx.Tasks)
	}
	idx = indexMarkdown("- [ ] 코드 `" + ref + "`\n")
	if len(idx.Tasks) != 1 || idx.Tasks[0].ID != "" {
		t.Fatal("code became task identity", idx.Tasks)
	}
}
func TestPostgresTaskDetailsIdentityAssignmentAndACL(t *testing.T) {
	_, ts := integrationTestServer(t)
	c := newIntegrationTestClient(t, ts.URL)
	me := testJSONObject(t, c.request("POST", "/api/v1/auth/login", map[string]any{"email": "admin@example.test", "password": "Integration-Test-Password-2026!"}, 200))
	ws := testJSONObject(t, c.request("POST", "/api/v1/workspaces", map[string]any{"name": "할 일 운영 검증"}, 200))
	wid := str(ws, "id")
	d := testJSONObject(t, c.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "운영 점검", "markdown": "- [ ] 첫 번째 점검\n\n```md\n- [ ] 예제\n```\n", "visibility": "private"}, 200))
	id := str(d, "id")
	details := map[string]any{"document_id": id, "version": 1, "line": 0, "status": "doing", "priority": "high", "due_date": "2026-09-20", "assignee_id": me["id"]}
	updated := testJSONObject(t, c.request("PUT", "/api/v1/tasks/details", details, 200))
	if !strings.Contains(str(updated, "markdown"), "[작업](/app/tasks?task=") {
		t.Fatal(updated)
	}
	board := func() []map[string]any {
		t.Helper()
		data := testJSONObject(t, c.request("GET", "/api/v1/tasks/board?workspace_id="+wid, nil, 200))
		var rows []map[string]any
		raw, _ := json.Marshal(data["items"])
		json.Unmarshal(raw, &rows)
		return rows
	}
	rows := board()
	if len(rows) != 1 || str(rows[0], "status") != "doing" || str(rows[0], "due_date") != "2026-09-20" || rows[0]["assignee_id"] != me["id"] {
		t.Fatal(rows)
	}
	tid := str(rows[0], "task_id")
	details["task_id"] = tid
	details["version"] = 1
	c.request("PUT", "/api/v1/tasks/details", details, 409)
	moved := strings.Replace("# 점검 계획\n\n"+str(updated, "markdown"), "첫 번째 점검", "이름을 변경한 점검", 1)
	updated = testJSONObject(t, c.request("PUT", "/api/v1/documents/"+id, map[string]any{"version": 2, "markdown": moved}, 200))
	rows = board()
	if str(rows[0], "task_id") != tid || number(rows[0], "line", -1) != 2 || str(rows[0], "status") != "doing" {
		t.Fatal("reorder lost metadata", rows)
	}
	c.request("PUT", "/api/v1/tasks", map[string]any{"document_id": id, "version": 2, "line": 0, "done": true}, 409)
	details["version"] = 3
	details["line"] = 2
	details["status"] = "done"
	updated = testJSONObject(t, c.request("PUT", "/api/v1/tasks/details", details, 200))
	if !strings.Contains(str(updated, "markdown"), "- [x] 이름을 변경한 점검") {
		t.Fatal(updated)
	}
	rows = board()
	if !boolean(rows[0], "done") || str(rows[0], "status") != "done" {
		t.Fatal(rows)
	}
	duplicate := str(updated, "markdown") + "\n- [ ] 복제한 점검 [작업](/app/tasks?task=" + tid + ")\n"
	c.request("PUT", "/api/v1/documents/"+id, map[string]any{"version": 4, "markdown": duplicate}, 200)
	rows = board()
	if len(rows) != 2 || !boolean(rows[1], "ambiguous") {
		t.Fatal(rows)
	}
	details["version"] = 5
	details["line"] = rows[1]["line"]
	details["status"] = "todo"
	c.request("PUT", "/api/v1/tasks/details", details, 409)
	details["separate"] = true
	c.request("PUT", "/api/v1/tasks/details", details, 200)
	rows = board()
	if str(rows[1], "task_id") == tid || boolean(rows[0], "ambiguous") || boolean(rows[1], "ambiguous") {
		t.Fatal(rows)
	}
	u := testJSONObject(t, c.request("POST", "/api/v1/admin/users", map[string]any{"email": "task-observer@example.test", "name": "비인가 담당자", "password": "Tasks-password-2026!", "role": "editor"}, 200))
	c.request("PUT", "/api/v1/workspaces/"+wid+"/members", map[string]any{"email": u["email"], "role": "editor"}, 200)
	details["task_id"] = rows[1]["task_id"]
	details["version"] = 6
	details["separate"] = false
	details["assignee_id"] = u["id"]
	c.request("PUT", "/api/v1/tasks/details", details, 400)
	got := testJSONObject(t, c.request("GET", "/api/v1/documents/"+id, nil, 200))
	if number(got, "version", 0) != 6 {
		t.Fatal("rejected assignment changed document")
	}
	member := newIntegrationTestClient(t, ts.URL)
	member.request("POST", "/api/v1/auth/login", map[string]any{"email": u["email"], "password": "Tasks-password-2026!"}, 200)
	if strings.Contains(string(member.request("GET", "/api/v1/tasks/board?workspace_id="+wid, nil, 200)), id) {
		t.Fatal("private task leaked")
	}
	member.request("PUT", "/api/v1/tasks/details", details, 403)
}
