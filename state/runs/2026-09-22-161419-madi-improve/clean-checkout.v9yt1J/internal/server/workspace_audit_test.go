package server

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestWorkspaceAuditCurrentACLAndSafeProjection(t *testing.T) {
	s, admin, ctx, p, wid := jobTestFixture(t)
	path := "/api/v1/workspaces/" + wid + "/audit"
	other := testJSONObject(t, admin.request("POST", "/api/v1/admin/users", map[string]any{"email": "audit-editor@example.test", "name": "다른 작성자", "role": "editor", "password": "Audit-editor-password-2026!"}, 200))
	admin.request("PUT", "/api/v1/workspaces/"+wid+"/members", map[string]any{"email": "audit-editor@example.test", "role": "admin"}, 200)
	member := newIntegrationTestClient(t, admin.base)
	member.request("POST", "/api/v1/auth/login", map[string]any{"email": "audit-editor@example.test", "password": "Audit-editor-password-2026!"}, 200)
	private := testJSONObject(t, member.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "AUDIT_PRIVATE_TITLE", "markdown": "AUDIT_PRIVATE_BODY", "visibility": "private"}, 200))
	shared := testJSONObject(t, member.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "공개되는 팀 문서", "markdown": "AUDIT_BODY_SECRET", "visibility": "workspace"}, 200))
	space := testJSONObject(t, member.request("POST", "/api/v1/spaces", map[string]any{"workspace_id": wid, "name": "AUDIT_RESTRICTED_SPACE", "visibility": "restricted"}, 200))
	member.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "space_id": space["id"], "title": "AUDIT_RESTRICTED_DOC", "markdown": "숨긴 원문", "visibility": "workspace"}, 200)
	database := testJSONObject(t, member.request("POST", "/api/v1/databases", map[string]any{"workspace_id": wid, "space_id": space["id"], "name": "AUDIT_RESTRICTED_DB"}, 200))
	member.request("POST", "/api/v1/databases/"+str(database, "id")+"/rows", map[string]any{"values": map[string]any{"title": "AUDIT_RESTRICTED_ROW"}}, 200)
	member.request("POST", "/api/v1/canvases", map[string]any{"workspace_id": wid, "title": "AUDIT_PRIVATE_CANVAS", "visibility": "private", "data": canvasData{}}, 200)
	member.request("POST", "/api/v1/templates", map[string]any{"workspace_id": wid, "name": "AUDIT_PRIVATE_TEMPLATE", "markdown": "숨긴 템플릿", "visibility": "private"}, 200)
	details := map[string]any{"before": "AUDIT_BEFORE_SECRET", "after": "AUDIT_AFTER_SECRET", "api_key": "AUDIT_KEY_SECRET", "version": 2, "enabled": false, "rows": 3, "status": "published"}
	for _, id := range []string{str(private, "id"), str(shared, "id")} {
		if _, err := s.DB.Exec(ctx, "INSERT INTO audit_logs(id,user_id,action,resource,ip,details) VALUES($1,$2,'DOCUMENT_UPDATE',$3,'192.0.2.99',$4)", newID(), str(other, "id"), id, jsonValue(details)); err != nil {
			t.Fatal(err)
		}
	}
	raw := string(admin.request("GET", path, nil, 200))
	for _, secret := range []string{"AUDIT_PRIVATE", "AUDIT_RESTRICTED", "AUDIT_BODY", "AUDIT_BEFORE", "AUDIT_AFTER", "AUDIT_KEY", "192.0.2.99", str(private, "id")} {
		if strings.Contains(raw, secret) {
			t.Fatalf("audit leaked %s: %s", secret, raw)
		}
	}
	if !strings.Contains(raw, str(shared, "id")) || !strings.Contains(raw, `"enabled":false`) {
		t.Fatalf("safe metadata absent: %s", raw)
	}
	member.request("GET", path, nil, 200)
	// Resource becoming private immediately removes its past events, including for service admin.
	_, err := s.DB.Exec(ctx, "UPDATE documents SET visibility='private' WHERE id=$1", str(shared, "id"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(admin.request("GET", path, nil, 200)), str(shared, "id")) {
		t.Fatal("revoked content metadata remains visible")
	}
	// Workspace membership is mandatory even for a service administrator.
	isolated := testJSONObject(t, admin.request("POST", "/api/v1/workspaces", map[string]any{"name": "격리된 감사"}, 200))
	member.request("GET", "/api/v1/workspaces/"+str(isolated, "id")+"/audit", nil, 403)
	for i := 0; i < 110; i++ {
		_, err = s.DB.Exec(ctx, "INSERT INTO audit_logs(id,user_id,action,resource,details) VALUES($1,$2,'WORKSPACE_SETTINGS_UPDATE',$3,'{}')", newID(), p.ID, wid)
		if err != nil {
			t.Fatal(err)
		}
	}
	first := testJSONObject(t, admin.request("GET", path+"?action=WORKSPACE_SETTINGS_UPDATE", nil, 200))
	if len(first["items"].([]any)) != 100 || str(first, "next_cursor") == "" {
		t.Fatal("first page cursor missing")
	}
	second := testJSONObject(t, admin.request("GET", path+"?action=WORKSPACE_SETTINGS_UPDATE&cursor="+str(first, "next_cursor"), nil, 200))
	if len(second["items"].([]any)) != 10 || str(second, "next_cursor") != "" {
		t.Fatalf("bad next page %v", second)
	}
	seen := map[string]bool{}
	for _, page := range []map[string]any{first, second} {
		for _, v := range page["items"].([]any) {
			id := str(v.(map[string]any), "id")
			if seen[id] {
				t.Fatal("duplicate audit row")
			}
			seen[id] = true
		}
	}
	for _, q := range []string{"?days=0", "?days=366", "?action=%25", "?cursor=bad"} {
		admin.request("GET", path+q, nil, 400)
	}
	_, err = s.DB.Exec(ctx, "UPDATE workspace_members SET role='viewer' WHERE workspace_id=$1 AND user_id=$2", wid, str(other, "id"))
	if err != nil {
		t.Fatal(err)
	}
	member.request("GET", path, nil, 403)
	var items map[string]any
	if json.Unmarshal([]byte(raw), &items) != nil {
		t.Fatal("invalid audit JSON")
	}
}
