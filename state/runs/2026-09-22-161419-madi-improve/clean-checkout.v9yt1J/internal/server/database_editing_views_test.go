package server

import (
	"encoding/json"
	"testing"
)

func TestPostgresDatabaseEditingViewsPrivacyCASAndDefaults(t *testing.T) {
	_, owner, _, _, wid := jobTestFixture(t)
	db := testJSONObject(t, owner.request("POST", "/api/v1/databases", map[string]any{"workspace_id": wid, "name": "보기 분리"}, 200))
	id := str(db, "id")
	createUser := func(email, role string) *integrationTestClient {
		owner.request("POST", "/api/v1/admin/users", map[string]any{"email": email, "name": "보기 사용자", "role": role, "password": "View-Editing-Test-2026!"}, 200)
		owner.request("PUT", "/api/v1/workspaces/"+wid+"/members", map[string]any{"email": email, "role": role}, 200)
		client := newIntegrationTestClient(t, owner.base)
		client.request("POST", "/api/v1/auth/login", map[string]any{"email": email, "password": "View-Editing-Test-2026!"}, 200)
		return client
	}
	reader, editor := createUser("views-reader@example.test", "viewer"), createUser("views-editor@example.test", "editor")
	data := map[string]any{"view": "table", "filters": []any{}, "sorts": []any{}, "columns": []any{"title", "status"}, "board_property_id": "status", "date_property_id": ""}
	private := testJSONObject(t, reader.request("POST", "/api/v1/databases/"+id+"/views", map[string]any{"name": "개인 전용", "visibility": "private", "data": data}, 200))
	pid := str(private, "id")
	list := func(client *integrationTestClient) map[string]any {
		return testJSONObject(t, client.request("GET", "/api/v1/databases/"+id+"/views", nil, 200))
	}
	if len(list(owner)["views"].([]any)) != 0 || len(list(editor)["views"].([]any)) != 0 {
		t.Fatal("private view exposed to admin/editor")
	}
	owner.request("PUT", "/api/v1/databases/"+id+"/views/"+pid, map[string]any{"name": "관리자 우회", "visibility": "private", "data": data, "expected_version": 1}, 404)
	reader.request("POST", "/api/v1/databases/"+id+"/views", map[string]any{"name": "권한 없는 팀 보기", "visibility": "workspace", "data": data, "share_consent": true}, 403)
	editor.request("POST", "/api/v1/databases/"+id+"/views", map[string]any{"name": "동의 없는 팀 보기", "visibility": "workspace", "data": data}, 400)
	shared := testJSONObject(t, editor.request("POST", "/api/v1/databases/"+id+"/views", map[string]any{"name": "공유 현황", "visibility": "workspace", "data": data, "share_consent": true}, 200))
	sid := str(shared, "id")
	if len(list(reader)["views"].([]any)) != 2 {
		t.Fatal("readable team view missing")
	}
	reader.request("PUT", "/api/v1/databases/"+id+"/view-preference", map[string]any{"view_id": sid}, 200)
	if str(list(reader), "default_view_id") != sid || str(list(owner), "default_view_id") != "" {
		t.Fatal("personal default affected another user")
	}
	changed := testJSONObject(t, editor.request("PUT", "/api/v1/databases/"+id+"/views/"+sid, map[string]any{"name": "공유 현황 갱신", "visibility": "workspace", "data": data, "share_consent": true, "expected_version": 1}, 200))
	if number(changed, "version", 0) != 2 {
		t.Fatal("view CAS revision missing")
	}
	editor.request("PUT", "/api/v1/databases/"+id+"/views/"+sid, map[string]any{"name": "오래된 갱신", "visibility": "workspace", "data": data, "share_consent": true, "expected_version": 1}, 409)
	editor.request("DELETE", "/api/v1/databases/"+id+"/views/"+sid, map[string]any{"expected_version": 1}, 409)
	editor.request("PUT", "/api/v1/databases/"+id+"/views/"+sid, map[string]any{"name": "공유 종료", "visibility": "private", "data": data, "expected_version": 2}, 200)
	if str(list(reader), "default_view_id") != "" || len(list(reader)["views"].([]any)) != 1 {
		t.Fatal("private change left shared/default access")
	}
	reader.request("PUT", "/api/v1/databases/"+id+"/view-preference", map[string]any{"view_id": sid}, 404)
	reader.request("PUT", "/api/v1/databases/"+id+"/view-preference", map[string]any{"view_id": pid}, 200)
	reader.request("DELETE", "/api/v1/databases/"+id+"/views/"+pid, map[string]any{"expected_version": 1}, 200)
	if str(list(reader), "default_view_id") != "" {
		t.Fatal("deleted view left personal default")
	}
	// Removal of DB membership immediately applies to view configuration and changes.
	var users []map[string]any
	if e := json.Unmarshal(owner.request("GET", "/api/v1/admin/users", nil, 200), &users); e != nil {
		t.Fatal(e)
	}
	var editorID string
	for _, u := range users {
		if str(u, "email") == "views-editor@example.test" {
			editorID = str(u, "id")
		}
	}
	owner.request("DELETE", "/api/v1/workspaces/"+wid+"/members/"+editorID, nil, 200)
	editor.request("GET", "/api/v1/databases/"+id+"/views", nil, 403)
	editor.request("PUT", "/api/v1/databases/"+id+"/views/"+sid, map[string]any{"name": "철회 후", "visibility": "private", "data": data, "expected_version": 3}, 403)
}
