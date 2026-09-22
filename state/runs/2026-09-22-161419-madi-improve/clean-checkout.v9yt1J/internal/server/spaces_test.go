package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"
)

func TestPostgresSpaceHierarchyACL(t *testing.T) {
	s, ts := integrationTestServer(t)
	admin := newIntegrationTestClient(t, ts.URL)
	admin.request("POST", "/api/v1/auth/login", map[string]any{"email": "admin@example.test", "password": "Integration-Test-Password-2026!"}, 200)
	var ws []map[string]any
	json.Unmarshal(admin.request("GET", "/api/v1/workspaces", nil, 200), &ws)
	wid := str(ws[0], "id")
	u := testJSONObject(t, admin.request("POST", "/api/v1/admin/users", map[string]any{"email": "space-user@example.test", "name": "공간 담당자", "password": "Space-password-2026!", "role": "editor"}, 200))
	admin.request("PUT", "/api/v1/workspaces/"+wid+"/members", map[string]any{"email": u["email"], "role": "editor"}, 200)
	reader := newIntegrationTestClient(t, ts.URL)
	reader.request("POST", "/api/v1/auth/login", map[string]any{"email": u["email"], "password": "Space-password-2026!"}, 200)
	createSpace := func(name, parent, visibility string) map[string]any {
		return testJSONObject(t, admin.request("POST", "/api/v1/spaces", map[string]any{"workspace_id": wid, "name": name, "parent_id": parent, "visibility": visibility}, 200))
	}
	root := createSpace("경영 공간", "", "restricted")
	sid := str(root, "id")
	child := createSpace("하위 공간", sid, "workspace")
	cid := str(child, "id")
	doc := testJSONObject(t, admin.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "space_id": cid, "title": "SPACE_SECRET", "markdown": "SPACE_SECRET confidential source"}, 200))
	did := str(doc, "id")
	db := testJSONObject(t, admin.request("POST", "/api/v1/databases", map[string]any{"workspace_id": wid, "space_id": cid, "name": "SPACE_SECRET"}, 200))
	dbid := str(db, "id")
	reader.request("GET", "/api/v1/documents/"+did, nil, 404)
	reader.request("GET", "/api/v1/databases/"+dbid, nil, 404)
	for _, path := range []string{"/api/v1/documents?q=SPACE_SECRET", "/api/v1/databases?workspace_id=" + wid, "/api/v1/spaces?workspace_id=" + wid, "/api/v1/graph?workspace_id=" + wid} {
		body := reader.request("GET", path, nil, 200)
		if strings.Contains(string(body), did) || strings.Contains(string(body), "SPACE_SECRET") {
			t.Fatalf("ACL leak from %s: %s", path, body)
		}
	}
	p := &Principal{ID: str(u, "id"), Role: "editor"}
	r := httptest.NewRequest("POST", "/", nil)
	r = r.WithContext(context.WithValue(r.Context(), principalKey, p))
	ctx, err := s.aiContext(r, p, "", wid, "SPACE_SECRET")
	if err != nil || len(ctx) != 0 {
		t.Fatalf("AI ACL: %v %#v", err, ctx)
	}
	admin.request("PUT", "/api/v1/spaces/"+sid+"/members", map[string]any{"user_id": u["id"], "role": "viewer"}, 200)
	reader.request("GET", "/api/v1/documents/"+did, nil, 200)
	reader.request("PUT", "/api/v1/documents/"+did, map[string]any{"version": 1, "title": "no"}, 403)
	reader.request("POST", "/api/v1/databases/"+dbid+"/rows", map[string]any{"values": map[string]any{}}, 403)
	var canComment bool
	if err = s.DB.QueryRow(context.Background(), "SELECT madi_document_comment_allowed($1,$2)", p.ID, did).Scan(&canComment); err != nil || canComment {
		t.Fatalf("space viewer comment permission %v %v", canComment, err)
	}
	admin.request("PUT", "/api/v1/spaces/"+sid+"/members", map[string]any{"user_id": u["id"], "role": "editor"}, 200)
	reader.request("PUT", "/api/v1/documents/"+did, map[string]any{"version": 1, "title": "SPACE_SECRET updated"}, 200)
	reader.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "space_id": cid, "title": "나의 노트"}, 200)
	admin.request("PUT", "/api/v1/spaces/"+sid, map[string]any{"parent_id": cid}, 400)
	admin.request("DELETE", "/api/v1/spaces/"+sid, nil, 409)
	admin.request("PUT", "/api/v1/spaces/"+sid+"/members", map[string]any{"user_id": u["id"], "role": "remove"}, 200)
	reader.request("GET", "/api/v1/documents/"+did, nil, 404)
	// A public child does not override a private parent document.
	parent := testJSONObject(t, admin.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "비공개 상위", "visibility": "private"}, 200))
	under := testJSONObject(t, admin.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "parent_id": parent["id"], "title": "공개 하위", "visibility": "workspace"}, 200))
	reader.request("GET", "/api/v1/documents/"+str(under, "id"), nil, 404)
	reader.request("POST", "/api/v1/spaces", map[string]any{"workspace_id": wid, "name": "비인가 생성"}, 403)
}

func TestPostgresWorkspaceOverridesAndOrganizations(t *testing.T) {
	s, ts := integrationTestServer(t)
	admin := newIntegrationTestClient(t, ts.URL)
	admin.request("POST", "/api/v1/auth/login", map[string]any{"email": "admin@example.test", "password": "Integration-Test-Password-2026!"}, 200)
	var ws []map[string]any
	json.Unmarshal(admin.request("GET", "/api/v1/workspaces", nil, 200), &ws)
	wid := str(ws[0], "id")
	path := "/api/v1/workspaces/" + wid + "/settings"
	initial := testJSONObject(t, admin.request("GET", path, nil, 200))
	if number(initial, "version", -1) != 0 {
		t.Fatal(initial)
	}
	v := testJSONObject(t, admin.request("PUT", path, map[string]any{"version": 0, "data": map[string]any{"ai_enabled": true, "ai_base_url": "http://local-llm:8000/v1", "ai_model": "local-model", "ai_api_key": "workspace-secret", "ai_max_tokens": 262144, "feature_flags": map[string]any{"canvas": true}}}, 200))
	if strings.Contains(string(jsonValue(v)), "workspace-secret") || !boolean(v["data"].(map[string]any), "ai_api_key_configured") {
		t.Fatal("secret response", v)
	}
	var raw string
	s.DB.QueryRow(context.Background(), "SELECT data->>'ai_api_key' FROM workspace_settings WHERE workspace_id=$1", wid).Scan(&raw)
	if !strings.HasPrefix(raw, "enc:") {
		t.Fatal("secret stored in plaintext")
	}
	effective, e := s.effectiveSettings(context.Background(), wid)
	if e != nil || str(effective, "ai_api_key") != "workspace-secret" || number(effective, "ai_max_tokens", 0) != 262144 {
		t.Fatal(e, effective)
	}
	global, e := s.settings(context.Background())
	if e != nil || str(global, "ai_api_key") == "workspace-secret" {
		t.Fatal("global setting modified")
	}
	admin.request("PUT", path, map[string]any{"version": 0, "data": map[string]any{"site_name": "덮어쓰기 공격"}}, 409)
	admin.request("PUT", path, map[string]any{"version": v["version"], "data": map[string]any{"session_hours": 999}}, 400)
	admin.request("PUT", path, map[string]any{"version": v["version"], "data": map[string]any{"ai_max_tokens": 262145}}, 400)
	admin.request("PUT", path, map[string]any{"version": v["version"], "data": map[string]any{"theme_primary": "url(javascript:alert(1))"}}, 400)
	next := testJSONObject(t, admin.request("PUT", path, map[string]any{"version": v["version"], "data": map[string]any{"site_name": "팀 이름", "ai_api_key": ""}}, 200))
	var history []map[string]any
	json.Unmarshal(admin.request("GET", path+"/history", nil, 200), &history)
	admin.request("POST", path+"/history/"+str(history[0], "id")+"/restore", map[string]any{"version": next["version"]}, 200)
	effective, e = s.effectiveSettings(context.Background(), wid)
	if e != nil || str(effective, "ai_api_key") != "workspace-secret" || str(effective, "site_name") == "팀 이름" {
		t.Fatal("history restore", e, effective)
	}
	org := testJSONObject(t, admin.request("POST", "/api/v1/organizations", map[string]any{"name": "우리 조직", "slug": "our-org"}, 200))
	admin.request("PUT", "/api/v1/workspaces/"+wid+"/organization", map[string]any{"organization_id": org["id"]}, 200)
	u := testJSONObject(t, admin.request("POST", "/api/v1/admin/users", map[string]any{"name": "조직 멤버", "email": "org@example.test", "password": "Organization-password-2026!", "role": "editor"}, 200))
	admin.request("PUT", "/api/v1/organizations/"+str(org, "id")+"/members", map[string]any{"user_id": u["id"], "role": "member"}, 200)
	member := newIntegrationTestClient(t, ts.URL)
	member.request("POST", "/api/v1/auth/login", map[string]any{"email": u["email"], "password": "Organization-password-2026!"}, 200)
	member.request("GET", path, nil, 403)
	// Organization membership never implicitly grants document workspace access.
	if s.canWorkspace(context.Background(), &Principal{ID: str(u, "id"), Role: "editor"}, wid, false) {
		t.Fatal("organization member gained workspace access")
	}
}

func TestPostgresBackupCatalogueCompleteness(t *testing.T) {
	s, _ := integrationTestServer(t)
	rows, e := s.DB.Query(context.Background(), "SELECT tablename FROM pg_tables WHERE schemaname=current_schema()")
	if e != nil {
		t.Fatal(e)
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		if e = rows.Scan(&name); e != nil {
			t.Fatal(e)
		}
		if name != "schema_migrations" && !slices.Contains(backupTables, name) && !slices.Contains(ephemeralTables, name) {
			t.Errorf("durable table missing from backup catalogue: %s", name)
		}
	}
	if rows.Err() != nil {
		t.Fatal(rows.Err())
	}
}

func TestPostgresConcurrentTreeMoveCannotCreateCycle(t *testing.T) {
	s, ts := integrationTestServer(t)
	admin := newIntegrationTestClient(t, ts.URL)
	admin.request("POST", "/api/v1/auth/login", map[string]any{"email": "admin@example.test", "password": "Integration-Test-Password-2026!"}, 200)
	var ws []map[string]any
	json.Unmarshal(admin.request("GET", "/api/v1/workspaces", nil, 200), &ws)
	wid := str(ws[0], "id")
	create := func(name string) string {
		return str(testJSONObject(t, admin.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": name}, 200)), "id")
	}
	a, b := create("상위 A"), create("상위 B")
	var wg sync.WaitGroup
	statuses := make(chan int, 2)
	for _, move := range [][2]string{{a, b}, {b, a}} {
		wg.Add(1)
		go func(move [2]string) {
			defer wg.Done()
			req, _ := http.NewRequest("PUT", ts.URL+"/api/v1/documents/"+move[0], strings.NewReader(string(jsonValue(map[string]any{"version": 1, "parent_id": move[1]}))))
			req.Header.Set("X-Madi-Request", "1")
			req.Header.Set("Content-Type", "application/json")
			res, e := admin.client.Do(req)
			if e != nil {
				statuses <- 0
				return
			}
			defer res.Body.Close()
			statuses <- res.StatusCode
		}(move)
	}
	wg.Wait()
	close(statuses)
	success := 0
	for status := range statuses {
		if status == 200 {
			success++
		} else if status != 400 && status != 409 {
			t.Fatalf("unexpected status %d", status)
		}
	}
	if success != 1 {
		t.Fatalf("expected exactly one move, got %d", success)
	}
	var cycle bool
	if e := s.DB.QueryRow(context.Background(), "SELECT EXISTS(SELECT 1 FROM documents a JOIN documents b ON a.parent_id=b.id AND b.parent_id=a.id)").Scan(&cycle); e != nil || cycle {
		t.Fatalf("cycle=%v error=%v", cycle, e)
	}
}
