package server

import (
	"strings"
	"testing"
)

const htmlTasksFixture = `<table><tr><td colspan="2"><ul data-type="taskList"><li data-type="taskItem" data-checked="false"><p>첫 점검 <strong>중요</strong></p><ul data-type="taskList"><li data-type="taskItem" data-checked="true"><p>하위 점검</p></li></ul></li><li data-type="taskItem" data-checked="false"><p>두 번째 점검</p></li></ul></td></tr></table>`

func TestHTMLTaskSourceOffsetsAndRoundTrip(t *testing.T) {
	md := "---\ntitle: 할 일\n---\n" + htmlTasksFixture + "\n\n```html\n" + htmlTasksFixture + "\n```\n\n<pre>" + htmlTasksFixture + "</pre>\n"
	idx := indexMarkdown(md)
	if len(idx.Tasks) != 3 {
		t.Fatalf("expected three actual HTML tasks, got %#v", idx.Tasks)
	}
	for i, task := range idx.Tasks {
		if task.Line != 3 || task.HTML == nil || (i > 0 && task.Start <= idx.Tasks[i-1].Start) {
			t.Fatal(idx.Tasks)
		}
	}
	if idx.Tasks[0].Text != "첫 점검 중요" || idx.Tasks[1].Text != "하위 점검" || !idx.Tasks[1].Done {
		t.Fatal(idx.Tasks)
	}
	id := newID()
	updated := updateHTMLTask(md, idx.Tasks[2], id, true, false)
	rows := indexMarkdown(updated).Tasks
	if len(rows) != 3 || rows[2].ID != id || !rows[2].Done || rows[0].Done || rows[2].Text != "두 번째 점검" {
		t.Fatal(rows)
	}
	if !strings.Contains(updated, `colspan="2"`) || !strings.HasSuffix(updated, "```\n\n<pre>"+htmlTasksFixture+"</pre>\n") {
		t.Fatal("unrelated HTML changed", updated)
	}
	duplicate := strings.Replace(updated, "<p>첫 점검", `<p><a href="/app/tasks?task=`+id+`">작업</a>첫 점검`, 1)
	rows = indexMarkdown(duplicate).Tasks
	if !rows[0].Ambiguous || !rows[2].Ambiguous {
		t.Fatal(rows)
	}
	newID := newID()
	separate := updateHTMLTask(duplicate, rows[2], newID, false, true)
	rows = indexMarkdown(separate).Tasks
	if rows[2].ID != newID || rows[2].Done || rows[0].Ambiguous || rows[2].Ambiguous {
		t.Fatal(rows)
	}
}

func TestPostgresHTMLTaskMutationExactTarget(t *testing.T) {
	s, ts := integrationTestServer(t)
	c := newIntegrationTestClient(t, ts.URL)
	c.request("POST", "/api/v1/auth/login", map[string]any{"email": "admin@example.test", "password": "Integration-Test-Password-2026!"}, 200)
	w := testJSONObject(t, c.request("POST", "/api/v1/workspaces", map[string]any{"name": "표 안 할 일"}, 200))
	d := testJSONObject(t, c.request("POST", "/api/v1/documents", map[string]any{"workspace_id": w["id"], "title": "표 점검", "markdown": htmlTasksFixture}, 200))
	idx := indexMarkdown(htmlTasksFixture)
	in := map[string]any{"document_id": d["id"], "version": 1, "line": 0, "status": "done", "priority": "normal"}
	c.request("PUT", "/api/v1/tasks/details", in, 409)
	in["source_start"] = idx.Tasks[2].Start
	updated := testJSONObject(t, c.request("PUT", "/api/v1/tasks/details", in, 200))
	rows := indexMarkdown(str(updated, "markdown")).Tasks
	if rows[0].Done || !rows[2].Done || !validID(rows[2].ID) {
		t.Fatal(rows)
	}
	var count int
	if e := s.DB.QueryRow(t.Context(), "SELECT count(*) FROM automation_events WHERE type='task.completed' AND resource_id=$1", d["id"]).Scan(&count); e != nil || count != 1 {
		t.Fatal("completion outbox", count, e)
	}
	in["version"] = 1
	c.request("PUT", "/api/v1/tasks/details", in, 409)
	board := testJSONObject(t, c.request("GET", "/api/v1/tasks/board?workspace_id="+str(w, "id"), nil, 200))
	if len(board["items"].([]any)) != 3 {
		t.Fatal(board)
	}
}
