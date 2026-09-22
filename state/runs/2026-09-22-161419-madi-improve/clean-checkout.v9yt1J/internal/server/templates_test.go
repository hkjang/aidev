package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestTemplateValidationAndLiteralDates(t *testing.T) {
	in := templateInput{WorkspaceID: newID(), Name: "회의", Markdown: "---\ntags: [팀, 팀]\n---\n# 안건\n\n"}
	if err := normalizeTemplate(&in); err != nil {
		t.Fatal(err)
	}
	if in.Visibility != "private" || len(in.Tags) != 1 || !strings.HasSuffix(in.Markdown, "\n\n") {
		t.Fatal("defaults/frontmatter/bytes", in)
	}
	for _, change := range []func(*templateInput){func(v *templateInput) { v.Kind = "script" }, func(v *templateInput) { v.Visibility = "space" }, func(v *templateInput) { v.Name = "" }, func(v *templateInput) { v.Tags = []string{strings.Repeat("a", 201)}; v.Markdown = "plain" }, func(v *templateInput) { v.Markdown = "---\nbad: [\n---\n" }, func(v *templateInput) { v.Description = "\x00" }} {
		invalid := in
		change(&invalid)
		if normalizeTemplate(&invalid) == nil {
			t.Fatal("accepted invalid template", invalid)
		}
	}
	now := time.Date(2026, 9, 8, 23, 30, 0, 0, time.UTC)
	result := expandTemplateDates("{{date}} {{datetime}} {{exec}} ${process.env} $(id)", now, "Asia/Seoul")
	if result != "2026-09-09 2026-09-09T08:30:00+09:00 {{exec}} ${process.env} $(id)" {
		t.Fatal(result)
	}
	for _, builtin := range builtinTemplates() {
		v := templateFromMap(builtin)
		v.WorkspaceID = newID()
		if err := normalizeTemplate(&v); err != nil {
			t.Fatal(str(builtin, "id"), err)
		}
	}
}

func TestPostgresTemplatesACLVersionIsolationAndDerivedDocument(t *testing.T) {
	s, ts := integrationTestServer(t)
	ctx := context.Background()
	admin := newIntegrationTestClient(t, ts.URL)
	admin.request("POST", "/api/v1/auth/login", map[string]any{"email": "admin@example.test", "password": "Integration-Test-Password-2026!"}, 200)
	var workspaces []map[string]any
	json.Unmarshal(admin.request("GET", "/api/v1/workspaces", nil, 200), &workspaces)
	wid := str(workspaces[0], "id")
	u := testJSONObject(t, admin.request("POST", "/api/v1/admin/users", map[string]any{"email": "template-editor@example.test", "name": "템플릿 작성자", "role": "editor", "password": "Template-editor-password-2026!"}, 200))
	admin.request("PUT", "/api/v1/workspaces/"+wid+"/members", map[string]any{"email": u["email"], "role": "editor"}, 200)
	editor := newIntegrationTestClient(t, ts.URL)
	editor.request("POST", "/api/v1/auth/login", map[string]any{"email": u["email"], "password": "Template-editor-password-2026!"}, 200)
	create := func(client *integrationTestClient, name, visibility, space string) map[string]any {
		return testJSONObject(t, client.request("POST", "/api/v1/templates", map[string]any{"workspace_id": wid, "name": name, "description": "설명", "category": "운영", "markdown": "---\ntags: [운영]\n---\n# 원문\n\n", "visibility": visibility, "space_id": space, "kind": "runbook"}, 200))
	}
	private := create(editor, "PRIVATE_TEMPLATE_SENTINEL", "private", "")
	pid := str(private, "id")
	admin.request("GET", "/api/v1/templates/"+pid, nil, 404)
	listed := admin.request("GET", "/api/v1/templates?workspace_id="+wid, nil, 200)
	if strings.Contains(string(listed), "PRIVATE_TEMPLATE_SENTINEL") {
		t.Fatal("admin read private template")
	}
	admin.request("POST", "/api/v1/templates/"+pid+"/duplicate", map[string]any{"workspace_id": wid, "expected_version": 1}, 404)
	admin.request("POST", "/api/v1/templates/"+pid+"/documents", map[string]any{"workspace_id": wid, "expected_version": 1}, 404)
	shared := create(admin, "공유 런북", "workspace", "")
	id := str(shared, "id")
	path := "/api/v1/templates/" + id
	read := testJSONObject(t, editor.request("GET", path, nil, 200))
	if boolean(read, "can_write") {
		t.Fatal("reader can edit other owner's shared template")
	}
	editor.request("PUT", path, shared, 403)
	editor.request("GET", path+"/versions", nil, 403)
	duplicate := testJSONObject(t, editor.request("POST", path+"/duplicate", map[string]any{"workspace_id": wid, "expected_version": 1}, 200))
	if str(duplicate, "visibility") != "private" || str(duplicate, "owner_id") != str(u, "id") {
		t.Fatal("copy ownership/default privacy", duplicate)
	}
	other := testJSONObject(t, admin.request("POST", "/api/v1/workspaces", map[string]any{"name": "다른 워크스페이스"}, 200))
	otherID := str(other, "id")
	admin.request("POST", path+"/documents", map[string]any{"workspace_id": otherID, "expected_version": 1}, 404)
	admin.request("POST", path+"/duplicate", map[string]any{"workspace_id": otherID, "expected_version": 1}, 404)
	shared["name"] = "바뀐 런북"
	updated := testJSONObject(t, admin.request("PUT", path, shared, 200))
	if number(updated, "version", 0) != 2 {
		t.Fatal(updated)
	}
	admin.request("PUT", path, shared, 409)
	admin.request("POST", path+"/documents", map[string]any{"workspace_id": wid, "expected_version": 1}, 409)
	created := testJSONObject(t, editor.request("POST", path+"/documents", map[string]any{"workspace_id": wid, "expected_version": 2, "title": "실제 런북"}, 200))
	if str(created, "markdown") != str(shared, "markdown") || str(created, "visibility") != "private" || str(created, "status") != "draft" {
		t.Fatal("canonical creation lost bytes/privacy/draft", created)
	}
	knowledge := testJSONObject(t, editor.request("GET", "/api/v1/documents/"+str(created, "id")+"/knowledge", nil, 200))
	if str(knowledge, "kind") != "runbook" {
		t.Fatal("kind hook missing", knowledge)
	}
	admin.request("GET", "/api/v1/documents/"+str(created, "id"), nil, 404)
	history := admin.request("GET", path+"/versions", nil, 200)
	if !strings.Contains(string(history), "공유 런북") {
		t.Fatal("history missing")
	}
	restored := testJSONObject(t, admin.request("POST", path+"/versions/1/restore", map[string]any{"version": 2}, 200))
	if str(restored, "name") != "공유 런북" || number(restored, "version", 0) != 3 {
		t.Fatal(restored)
	}
	admin.request("DELETE", path+"?version=2", nil, 409)
	admin.request("DELETE", path+"?version=3", nil, 200)
	admin.request("POST", path+"/documents", map[string]any{"workspace_id": wid, "expected_version": 4}, 404)
	admin.request("POST", path+"/restore", map[string]any{"version": 3}, 409)
	restored = testJSONObject(t, admin.request("POST", path+"/restore", map[string]any{"version": 4}, 200))
	if restored["deleted_at"] != nil || number(restored, "version", 0) != 5 {
		t.Fatal(restored)
	}
	space := testJSONObject(t, admin.request("POST", "/api/v1/spaces", map[string]any{"workspace_id": wid, "name": "제한 공간", "visibility": "restricted"}, 200))
	sid := str(space, "id")
	restricted := create(admin, "SPACE_TEMPLATE_SENTINEL", "space", sid)
	restrictedID := str(restricted, "id")
	editor.request("GET", "/api/v1/templates/"+restrictedID, nil, 404)
	admin.request("PUT", "/api/v1/spaces/"+sid+"/members", map[string]any{"user_id": u["id"], "role": "viewer"}, 200)
	editor.request("GET", "/api/v1/templates/"+restrictedID, nil, 200)
	editor.request("POST", "/api/v1/templates", map[string]any{"workspace_id": wid, "space_id": sid, "visibility": "space", "name": "읽기 공간에 쓰기"}, 403)
	editor.request("POST", "/api/v1/templates/"+restrictedID+"/documents", map[string]any{"workspace_id": wid, "expected_version": 1, "space_id": sid}, 403)
	issued := testJSONObject(t, admin.request("POST", "/api/v1/keys", map[string]any{"workspace_id": wid, "name": "템플릿 조회", "scopes": []string{"document:read"}}, 201))
	key := newIntegrationTestClient(t, ts.URL)
	key.token = str(issued, "token")
	key.request("GET", path, nil, 200)
	key.request("POST", path+"/duplicate", map[string]any{"workspace_id": wid, "expected_version": 5}, 403)
	key.request("POST", path+"/documents", map[string]any{"workspace_id": wid, "expected_version": 5}, 403)
	admin.request("PUT", "/api/v1/workspaces/"+wid+"/members", map[string]any{"email": u["email"], "role": "viewer"}, 200)
	editor.request("POST", "/api/v1/templates/"+pid+"/documents", map[string]any{"workspace_id": wid, "expected_version": 1}, 403)
	var count int
	if e := s.DB.QueryRow(ctx, "SELECT count(*) FROM document_template_versions WHERE template_id=$1", id).Scan(&count); e != nil || count != 5 {
		t.Fatal("version snapshots", count, e)
	}
}

func TestPostgresTemplatesProtectAllFieldsAndHistory(t *testing.T) {
	s, admin, ctx, _, wid := jobTestFixture(t)
	policy := defaultProtectionSettings()
	policy["enabled"] = true
	policy["mode"] = "mask"
	policy["custom_terms"] = []string{"템플릿기밀"}
	if _, err := s.DB.Exec(ctx, "UPDATE protection_settings SET data=$1", jsonValue(policy)); err != nil {
		t.Fatal(err)
	}
	payload := map[string]any{"workspace_id": wid, "name": "템플릿기밀", "description": "owner@example.test", "category": "템플릿기밀", "markdown": "---\ntags: [owner@example.test]\n---\n# 템플릿기밀\n\n010-1234-5678\n", "tags": []string{"owner@example.test"}}
	v := testJSONObject(t, admin.request("POST", "/api/v1/templates", payload, 200))
	id := str(v, "id")
	for _, raw := range []string{"템플릿기밀", "owner@example.test", "010-1234-5678"} {
		if strings.Contains(string(jsonValue(v)), raw) {
			t.Fatal("raw response leak", raw)
		}
	}
	var snapshot string
	if e := s.DB.QueryRow(ctx, "SELECT data::text FROM document_template_versions WHERE template_id=$1", id).Scan(&snapshot); e != nil {
		t.Fatal(e)
	}
	if strings.Contains(snapshot, "owner@example.test") || strings.Contains(snapshot, "템플릿기밀") {
		t.Fatal("raw version persisted")
	}
	policy["mode"] = "block"
	s.DB.Exec(ctx, "UPDATE protection_settings SET data=$1", jsonValue(policy))
	admin.request("POST", "/api/v1/templates", payload, 422)
	var count int
	s.DB.QueryRow(ctx, "SELECT count(*) FROM document_templates WHERE workspace_id=$1", wid).Scan(&count)
	if count != 1 {
		t.Fatal("blocked template persisted", count)
	}
	// Builtin expansion still passes through the current canonical protection API.
	admin.request("POST", "/api/v1/templates/builtin-meeting/documents", map[string]any{"workspace_id": wid, "expected_version": 1, "title": "템플릿기밀"}, 422)
	admin.request("POST", "/api/v1/templates/builtin-meeting/documents", map[string]any{"workspace_id": wid, "expected_version": 1, "title": "깨끗한 회의"}, 200)
	if !strings.Contains(fmt.Sprint(v["protection"]), "true") {
		t.Fatal("mask notice missing")
	}
}

func TestPostgresTemplateSourceRecheckRollsBackDerivedDocument(t *testing.T) {
	s, admin, ctx, p, wid := jobTestFixture(t)
	input := map[string]any{"workspace_id": wid, "name": "원본", "markdown": "first source"}
	template := testJSONObject(t, admin.request("POST", "/api/v1/templates", input, 200))
	id := str(template, "id")
	template["markdown"] = "changed source"
	admin.request("PUT", "/api/v1/templates/"+id, template, 200)
	// Simulate a source changing after the handler read it but before the shared
	// createDocument transaction. The hook must roll back the entire new document.
	r := httptest.NewRequest("POST", "/api/v1/documents", strings.NewReader(string(jsonValue(map[string]any{"workspace_id": wid, "title": "MUST_NOT_COMMIT", "markdown": "first source"}))))
	r = r.WithContext(context.WithValue(context.WithValue(ctx, principalKey, p), templateCreationContext{}, templateCreationContext{Kind: "page", SourceID: id, WorkspaceID: wid, Version: 1}))
	w := httptest.NewRecorder()
	s.createDocument(w, r)
	if w.Code != 409 {
		t.Fatalf("stale source status %d: %s", w.Code, w.Body.String())
	}
	var count int
	if err := s.DB.QueryRow(ctx, "SELECT count(*) FROM documents WHERE workspace_id=$1 AND title='MUST_NOT_COMMIT'", wid).Scan(&count); err != nil || count != 0 {
		t.Fatal("stale template source committed", count, err)
	}
}
