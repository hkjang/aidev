package server

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestPostgresDocumentSplitAtomicPrivacyReplayAndTasks(t *testing.T) {
	s, owner, ctx, p, wid := jobTestFixture(t)
	taskID := newID()
	selected := "## 분리할 단락\n\n| 항목 | 값 |\n|---|---|\n| 대상 | 서버 |\n\n- [ ] [확인 작업](/app/tasks?task=" + taskID + ")\n\n```go\nfmt.Println(\"원문\")\n```\n"
	before := "---\ntags: [원본태그]\naliases: [원본별칭]\n---\n\n# 원본\n\n"
	after := "\n마지막 문장은 보존합니다.\n"
	md := before + selected + after
	source := testJSONObject(t, owner.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "개인 지식 분리", "markdown": md, "visibility": "private"}, 200))
	id := str(source, "id")
	if _, e := s.DB.Exec(ctx, "INSERT INTO task_details(document_id,task_id,assignee_id,due_date,status,priority,updated_by) VALUES($1,$2,$3,'2026-09-20','doing','high',$3)", id, taskID, p.ID); e != nil {
		t.Fatal(e)
	}
	preview := testJSONObject(t, owner.request("POST", "/api/v1/documents/"+id+"/split-preview", map[string]any{"expected_version": 1, "start_byte": len(before), "end_byte": len(before) + len(selected), "selected_text": selected, "title": "새 작업 지침"}, 200))
	if str(preview, "selected_markdown") != selected || strings.Contains(string(jsonValue(preview)), "마지막 문장") || !strings.Contains(str(preview, "notice"), "나만 보기") {
		t.Fatal("split preview source scope", preview)
	}
	request := map[string]any{"ticket": preview["ticket"], "client_request_id": newID(), "consent": true}
	// Inject an actual SQL failure after both document changes, before commit.
	if _, e := s.DB.Exec(ctx, `CREATE FUNCTION madi_split_fixture_fail() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'fixture split rollback';END $$;CREATE TRIGGER madi_split_fixture_fail BEFORE INSERT ON document_split_receipts FOR EACH ROW EXECUTE FUNCTION madi_split_fixture_fail()`); e != nil {
		t.Fatal(e)
	}
	owner.request("POST", "/api/v1/documents/"+id+"/split", request, 500)
	var docs, versions, events int
	if e := s.DB.QueryRow(ctx, "SELECT count(*) FROM documents WHERE id=$1", preview["child_id"]).Scan(&docs); e != nil || docs != 0 {
		t.Fatal("rollback left child", docs, e)
	}
	if e := s.DB.QueryRow(ctx, "SELECT count(*) FROM document_versions WHERE document_id=$1", id).Scan(&versions); e != nil || versions != 1 {
		t.Fatal("rollback left source version", versions, e)
	}
	if e := s.DB.QueryRow(ctx, "SELECT count(*) FROM automation_events WHERE resource_id=$1 AND type='document.updated'", id).Scan(&events); e != nil || events != 0 {
		t.Fatal("rollback left outbox", events, e)
	}
	if got := testJSONObject(t, owner.request("GET", "/api/v1/documents/"+id, nil, 200)); str(got, "markdown") != md || number(got, "version", 0) != 1 {
		t.Fatal("rollback changed canonical source")
	}
	if _, e := s.DB.Exec(ctx, "DROP TRIGGER madi_split_fixture_fail ON document_split_receipts"); e != nil {
		t.Fatal(e)
	}
	// Two real concurrent HTTP calls with the same request identifier create one child.
	type response struct {
		status int
		body   []byte
		err    error
	}
	results := make(chan response, 2)
	for i := 0; i < 2; i++ {
		go func() {
			r, e := http.NewRequestWithContext(ctx, "POST", owner.base+"/api/v1/documents/"+id+"/split", bytes.NewReader(jsonValue(request)))
			if e != nil {
				results <- response{err: e}
				return
			}
			r.Header.Set("X-Madi-Request", "1")
			r.Header.Set("Content-Type", "application/json")
			res, e := owner.client.Do(r)
			if e != nil {
				results <- response{err: e}
				return
			}
			defer res.Body.Close()
			raw, e := io.ReadAll(res.Body)
			results <- response{res.StatusCode, raw, e}
		}()
	}
	var result map[string]any
	for i := 0; i < 2; i++ {
		res := <-results
		if res.err != nil || res.status != 200 {
			t.Fatal("idempotent concurrent split", res.status, string(res.body), res.err)
		}
		result = testJSONObject(t, res.body)
	}
	child := result["child"].(map[string]any)
	parent := result["source"].(map[string]any)
	cid := str(child, "id")
	if str(child, "parent_id") != id || str(child, "visibility") != "workspace" || str(child, "status") != "draft" || str(child, "markdown") != selected || number(parent, "version", 0) != 2 || str(parent, "markdown") != before+str(preview, "replacement_markdown")+after {
		t.Fatal("split canonical mismatch", result)
	}
	if strings.Join(listStrings(parent["tags"]), ",") != "원본태그" || strings.Join(listStrings(parent["aliases"]), ",") != "원본별칭" {
		t.Fatal("source front matter lost")
	}
	var taskDoc string
	if e := s.DB.QueryRow(ctx, "SELECT document_id::text FROM task_details WHERE task_id=$1", taskID).Scan(&taskDoc); e != nil || taskDoc != cid {
		t.Fatal("task metadata did not follow selected block", taskDoc, e)
	}
	u := testJSONObject(t, owner.request("POST", "/api/v1/admin/users", map[string]any{"email": "split-reader@example.test", "name": "다른 사용자", "role": "editor", "password": "Split-Reader-Password-2026!"}, 200))
	owner.request("PUT", "/api/v1/workspaces/"+wid+"/members", map[string]any{"email": u["email"], "role": "editor"}, 200)
	reader := newIntegrationTestClient(t, owner.base)
	reader.request("POST", "/api/v1/auth/login", map[string]any{"email": u["email"], "password": "Split-Reader-Password-2026!"}, 200)
	reader.request("GET", "/api/v1/documents/"+cid, nil, 404)
	reader.request("POST", "/api/v1/documents/"+id+"/split", request, 409)
	owner.request("POST", "/api/v1/documents/"+id+"/split", request, 200)
	if e := s.DB.QueryRow(ctx, "SELECT count(*) FROM document_split_receipts WHERE source_id=$1", id).Scan(&docs); e != nil || docs != 1 {
		t.Fatal("duplicate receipt", docs, e)
	}
	if e := s.DB.QueryRow(ctx, "SELECT count(*) FROM document_versions WHERE document_id=$1", id).Scan(&versions); e != nil || versions != 2 {
		t.Fatal("replay added source version", versions, e)
	}
	owner.request("DELETE", "/api/v1/documents/"+cid, map[string]any{"expected_version": 1}, 200)
	owner.request("POST", "/api/v1/documents/"+id+"/split", request, 410)
}

func TestPostgresDocumentSplitRetiresCollaborationEpoch(t *testing.T) {
	s, admin, _, wid, _ := collaborationTestSetup(t)
	md := "선택한 전체 문단"
	doc := testJSONObject(t, admin.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "분리와 공동 편집", "markdown": md}, 200))
	id := str(doc, "id")
	session := collaborationTestSession(t, admin)
	initial, e := s.collaborationState(t.Context(), id, session, nil)
	if e != nil {
		t.Fatal(e)
	}
	seed := collaborationTestDoc(md)
	defer seed.Destroy()
	state, e := s.collaborationState(t.Context(), id, session, &collaborationMessage{Type: "seed", Schema: collaborationSchemaID, Epoch: initial.Epoch, Version: initial.Version, State: seed.EncodeStateAsUpdate()})
	if e != nil {
		t.Fatal(e)
	}
	stale := collaborationTestClone(t, state.State)
	collaborationTestInsert(stale, " 오프라인 늦은 변경")
	preview := testJSONObject(t, admin.request("POST", "/api/v1/documents/"+id+"/split-preview", map[string]any{"expected_version": 1, "start_byte": 0, "end_byte": len(md), "selected_text": md, "title": "분리 문서"}, 200))
	result := testJSONObject(t, admin.request("POST", "/api/v1/documents/"+id+"/split", map[string]any{"ticket": preview["ticket"], "client_request_id": newID(), "consent": true}, 200))
	after := str(result["source"].(map[string]any), "markdown")
	recovered, e := s.collaborationState(t.Context(), id, session, &collaborationMessage{Type: "update", Schema: collaborationSchemaID, Epoch: initial.Epoch, State: stale.EncodeStateAsUpdate()})
	if e != nil || recovered.Type != "reset" || recovered.Epoch == initial.Epoch || recovered.Markdown != after {
		t.Fatal("stale CRDT overwrote split", e, recovered.Type)
	}
	if actual := testJSONObject(t, admin.request("GET", "/api/v1/documents/"+id, nil, 200)); str(actual, "markdown") != after || number(actual, "version", 0) != 2 {
		t.Fatal("retired epoch changed split canonical")
	}
}
