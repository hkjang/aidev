package server

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestPostgresSCIMScopesProvisioningGroupsAndIsolation(t *testing.T) {
	s, admin, editor, wid, _ := collaborationTestSetup(t)
	issue := func(workspace, email string) *integrationTestClient {
		user := testJSONObject(t, admin.request("POST", "/api/v1/admin/users", map[string]any{"email": email, "name": "디렉터리 동기화", "role": "viewer", "kind": "service"}, 200))
		admin.request("PUT", "/api/v1/workspaces/"+workspace+"/members", map[string]any{"email": email, "role": "editor"}, 200)
		key := testJSONObject(t, admin.request("POST", "/api/v1/keys", map[string]any{"user_id": user["id"], "name": "SCIM", "workspace_id": workspace, "scopes": []string{scimProvisionScope}, "rate_limit": 1000}, 201))
		client := newIntegrationTestClient(t, admin.base)
		client.token = str(key, "token")
		return client
	}
	admin.request("POST", "/api/v1/keys", map[string]any{"name": "개인 상승", "workspace_id": wid, "scopes": []string{scimProvisionScope}}, 400)
	scopes := append(append([]string{}, keyScopes...), scimProvisionScope)
	admin.request("PUT", "/api/v1/admin/settings", map[string]any{"scim_enabled": true, "allowed_key_scopes": scopes, "identity_group_mappings": []any{map[string]any{"provider": "scim", "group": "Directory Editors", "workspace_id": wid, "role": "editor"}}}, 200)
	admin.request("POST", "/api/v1/keys", map[string]any{"name": "개인 상승", "workspace_id": wid, "scopes": []string{scimProvisionScope}}, 403)
	editor.request("POST", "/api/v1/keys", map[string]any{"name": "일반 상승", "workspace_id": wid, "scopes": []string{scimProvisionScope}}, 403)
	client := issue(wid, "scim-sync@example.test")
	otherWorkspace := testJSONObject(t, admin.request("POST", "/api/v1/workspaces", map[string]any{"name": "격리 SCIM"}, 200))
	otherWid := str(otherWorkspace, "id")
	other := issue(otherWid, "scim-other@example.test")
	client.request("GET", "/api/v1/scim/v2/ServiceProviderConfig", nil, 200)
	client.request("GET", "/api/v1/admin/users", nil, 403)
	create := func(username, email string) map[string]any {
		return map[string]any{"schemas": []string{scimUserSchema}, "userName": username, "displayName": "자동 동기화 사용자", "emails": []any{map[string]any{"value": email, "primary": true}}, "active": true}
	}
	client.request("POST", "/api/v1/scim/v2/Users", create("admin", "admin@example.test"), 409)
	created := testJSONObject(t, client.request("POST", "/api/v1/scim/v2/Users", create("Directory.Worker", "directory.worker@example.test"), 201))
	uid := str(created, "id")
	client.request("POST", "/api/v1/scim/v2/Users", create("directory.worker", "different.worker@example.test"), 409)
	other.request("GET", "/api/v1/scim/v2/Users/"+uid, nil, 404)
	other.request("PATCH", "/api/v1/scim/v2/Users/"+uid, map[string]any{}, 404)
	other.request("DELETE", "/api/v1/scim/v2/Users/"+uid, nil, 404)
	var role string
	if e := s.DB.QueryRow(context.Background(), `SELECT role FROM workspace_members WHERE workspace_id=$1 AND user_id=$2`, wid, uid).Scan(&role); e != nil || role != "viewer" {
		t.Fatalf("SCIM baseline: %q %v", role, e)
	}
	groupInput := map[string]any{"schemas": []string{scimGroupSchema}, "displayName": "Directory Editors", "members": []any{map[string]any{"value": uid}}}
	other.request("POST", "/api/v1/scim/v2/Groups", groupInput, 400)
	group := testJSONObject(t, client.request("POST", "/api/v1/scim/v2/Groups", groupInput, 201))
	gid := str(group, "id")
	if e := s.DB.QueryRow(context.Background(), `SELECT role FROM workspace_members WHERE workspace_id=$1 AND user_id=$2`, wid, uid).Scan(&role); e != nil || role != "editor" {
		t.Fatalf("SCIM group grant: %q %v", role, e)
	}
	patch := func(path string, value any) map[string]any {
		return map[string]any{"schemas": []string{scimPatchSchema}, "Operations": []any{map[string]any{"op": "replace", "path": path, "value": value}}}
	}
	client.request("PATCH", "/api/v1/scim/v2/Users/"+uid, patch("roles", []any{"admin"}), 400)
	client.request("PATCH", "/api/v1/scim/v2/Users/"+uid, patch("active", false), 200)
	var disabled bool
	if e := s.DB.QueryRow(context.Background(), `SELECT disabled FROM users WHERE id=$1`, uid).Scan(&disabled); e != nil || !disabled {
		t.Fatal("deactivation did not disable login")
	}
	var membership bool
	s.DB.QueryRow(context.Background(), `SELECT EXISTS(SELECT 1 FROM workspace_members WHERE workspace_id=$1 AND user_id=$2)`, wid, uid).Scan(&membership)
	if membership {
		t.Fatal("inactive SCIM grants retained")
	}
	client.request("PATCH", "/api/v1/scim/v2/Users/"+uid, patch("active", true), 200)
	client.request("PATCH", "/api/v1/scim/v2/Groups/"+gid, map[string]any{"schemas": []string{scimPatchSchema}, "Operations": []any{map[string]any{"op": "remove", "path": `members[value eq "` + uid + `"]`}}}, 200)
	if e := s.DB.QueryRow(context.Background(), `SELECT role FROM workspace_members WHERE workspace_id=$1 AND user_id=$2`, wid, uid).Scan(&role); e != nil || role != "viewer" {
		t.Fatalf("group removal: %q %v", role, e)
	}
	listed := testJSONObject(t, client.request("GET", "/api/v1/scim/v2/Users?filter=userName%20eq%20%22directory.worker%22&count=1", nil, 200))
	if number(listed, "totalResults", 0) != 1 {
		t.Fatalf("SCIM filter: %v", listed)
	}
	client.request("GET", "/api/v1/scim/v2/Users?filter=userName%20pr", nil, 400)
	// Optimistic concurrency rejects stale IdP writes without changing the resource.
	body, _ := json.Marshal(patch("displayName", "오래된 편집"))
	request, _ := http.NewRequest("PATCH", client.base+"/api/v1/scim/v2/Users/"+uid, strings.NewReader(string(body)))
	request.Header.Set("Authorization", "Bearer "+client.token)
	request.Header.Set("If-Match", `W/"1"`)
	response, e := client.client.Do(request)
	if e != nil {
		t.Fatal(e)
	}
	response.Body.Close()
	if response.StatusCode != 412 {
		t.Fatalf("stale ETag accepted: %d", response.StatusCode)
	}
	// A manually promoted administrator can no longer be changed by scoped SCIM.
	admin.request("PUT", "/api/v1/admin/users/"+uid, map[string]any{"role": "admin"}, 200)
	client.request("PATCH", "/api/v1/scim/v2/Users/"+uid, patch("active", false), 403)
	client.request("DELETE", "/api/v1/scim/v2/Users/"+uid, nil, 403)
	admin.request("PUT", "/api/v1/admin/users/"+uid, map[string]any{"role": "viewer"}, 200)
	client.request("DELETE", "/api/v1/scim/v2/Users/"+uid, nil, 204)
	client.request("GET", "/api/v1/scim/v2/Users/"+uid, nil, 404)
	client.request("DELETE", "/api/v1/scim/v2/Groups/"+gid, nil, 204)
	admin.request("PUT", "/api/v1/admin/settings", map[string]any{"allowed_key_scopes": keyScopes}, 200)
	client.request("GET", "/api/v1/scim/v2/Users", nil, 403)
}
