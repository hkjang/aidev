package server

import (
	"net/url"
	"strings"
	"testing"
)

func TestPostgresUniversalSearchKindsCurrentACLAndScopes(t *testing.T) {
	s, ts := integrationTestServer(t)
	ctx := t.Context()
	admin := newIntegrationTestClient(t, ts.URL)
	me := testJSONObject(t, admin.request("POST", "/api/v1/auth/login", map[string]any{"email": "admin@example.test", "password": "Integration-Test-Password-2026!"}, 200))
	_ = me
	wid := str(testJSONObject(t, admin.request("POST", "/api/v1/workspaces", map[string]any{"name": "통합 검색 시험"}, 200)), "id")
	var uid string
	if e := s.DB.QueryRow(ctx, "SELECT id::text FROM users WHERE email='admin@example.test'").Scan(&uid); e != nil {
		t.Fatal(e)
	}
	user := testJSONObject(t, admin.request("POST", "/api/v1/admin/users", map[string]any{"email": "universal@example.test", "name": "검색 유니버설 구성원", "role": "editor", "password": "Search-Password-2026!"}, 200))
	admin.request("PUT", "/api/v1/workspaces/"+wid+"/members", map[string]any{"email": user["email"], "role": "viewer"}, 200)
	doc := testJSONObject(t, admin.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "검색 유니버설 운영 가이드", "markdown": "# 검색 유니버설 운영\n\n검색 유니버설 설명입니다.\n\n- [ ] 검색 유니버설 점검\n\n```go\nvar universal = `검색 유니버설 코드`\n```\n", "tags": []string{"검색유니버설"}}, 200))
	id := str(doc, "id")
	hidden := testJSONObject(t, admin.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "검색 유니버설 SECRET", "markdown": "검색 유니버설 HIDDEN_BODY", "visibility": "private", "tags": []string{"HIDDEN_TAG"}}, 200))
	hid := str(hidden, "id")
	for _, d := range []string{id, hid} {
		if _, e := s.indexSearchDocument(ctx, d); e != nil {
			t.Fatal(e)
		}
	}
	db := testJSONObject(t, admin.request("POST", "/api/v1/databases", map[string]any{"workspace_id": wid, "name": "검색 유니버설 DB", "properties": []any{map[string]any{"id": "title", "name": "이름", "type": "text"}}}, 200))
	admin.request("POST", "/api/v1/databases/"+str(db, "id")+"/rows", map[string]any{"values": map[string]any{"title": "검색 유니버설 행"}}, 200)
	for _, d := range []string{id, hid} {
		if _, e := s.DB.Exec(ctx, "INSERT INTO comments(id,document_id,user_id,body) VALUES($1,$2,$3,'검색 유니버설 댓글')", newID(), d, uid); e != nil {
			t.Fatal(e)
		}
		if _, e := s.DB.Exec(ctx, "INSERT INTO attachments(id,document_id,user_id,name,content_type,size,path) VALUES($1,$2,$3,'검색 유니버설.pdf','application/pdf',12,'test-only-not-read')", newID(), d, uid); e != nil {
			t.Fatal(e)
		}
	}
	viewer := newIntegrationTestClient(t, ts.URL)
	viewer.request("POST", "/api/v1/auth/login", map[string]any{"email": user["email"], "password": "Search-Password-2026!"}, 200)
	conversation := newID()
	if _, e := s.DB.Exec(ctx, "INSERT INTO ai_conversations(id,workspace_id,owner_id,title) VALUES($1,$2,$3,'검색 유니버설 개인 대화')", conversation, wid, str(user, "id")); e != nil {
		t.Fatal(e)
	}
	if _, e := s.DB.Exec(ctx, `INSERT INTO ai_messages(id,conversation_id,ordinal,question,answer,action,sources,provider_fingerprint,model) VALUES($1,$2,1,'검색 유니버설 질문','검색 유니버설 답변','ask',$3,'fixture','fixture')`, newID(), conversation, jsonValue([]aiSource{{ID: id, Version: 1}})); e != nil {
		t.Fatal(e)
	}
	base := "/api/v1/search?workspace_id=" + wid + "&q=" + url.QueryEscape("검색")
	all := string(viewer.request("GET", base, nil, 200))
	if strings.Contains(all, hid) || strings.Contains(all, "HIDDEN") || strings.Contains(all, "SECRET") {
		t.Fatal("private search result leak", all)
	}
	for _, kind := range universalSearchKinds {
		raw := viewer.request("GET", base+"&type="+kind, nil, 200)
		v := testJSONObject(t, raw)
		rows := v["results"].([]any)
		if len(rows) == 0 {
			t.Fatalf("missing kind %s: %s", kind, raw)
		}
		for _, row := range rows {
			if str(row.(map[string]any), "kind") != kind {
				t.Fatal("kind filter", row)
			}
		}
	}
	for _, filter := range []string{"&tag=missing", "&status=archived", "&author_id=" + str(user, "id"), "&from=2099-01-01", "&to=2000-01-01"} {
		v := testJSONObject(t, viewer.request("GET", base+filter, nil, 200))
		if len(v["results"].([]any)) != 0 {
			t.Fatal("filter ignored", filter, v)
		}
	}
	page := testJSONObject(t, viewer.request("GET", base+"&limit=2", nil, 200))
	if !boolean(page, "has_more") || len(page["results"].([]any)) != 2 {
		t.Fatal("pagination", page)
	}
	for _, sort := range []string{"newest", "oldest", "title", "relevance"} {
		viewer.request("GET", base+"&sort="+sort, nil, 200)
	}
	for _, bad := range []string{"&sort=sql", "&type=unknown", "&limit=101", "&offset=-1", "&from=invalid", "&from=2026-09-10&to=2026-09-01", "&author_id=bad"} {
		viewer.request("GET", base+bad, nil, 400)
	}
	// Literal SQL wildcard characters are search data, never match-all patterns.
	wildcard := testJSONObject(t, viewer.request("GET", "/api/v1/search?workspace_id="+wid+"&q=%25&type=document", nil, 200))
	if len(wildcard["results"].([]any)) != 0 {
		t.Fatal("wildcard expanded", wildcard)
	}
	for _, scope := range []string{"search:read", "document:read", "database:read"} {
		key := testJSONObject(t, admin.request("POST", "/api/v1/keys", map[string]any{"name": "검색 범위", "workspace_id": wid, "scopes": []string{scope}}, 201))
		client := newIntegrationTestClient(t, ts.URL)
		client.token = str(key, "token")
		v := testJSONObject(t, client.request("GET", base, nil, 200))
		for _, item := range v["results"].([]any) {
			kind := str(item.(map[string]any), "kind")
			if kind == "user" || kind == "ai_conversation" || (scope == "database:read" && kind != "database" && kind != "row") || (scope != "database:read" && (kind == "database" || kind == "row")) {
				t.Fatal("scope bypass", scope, kind)
			}
		}
		if scope == "database:read" {
			client.request("POST", "/api/v1/search/reindex", map[string]any{"document_id": id}, 403)
		} else {
			client.request("POST", "/api/v1/search/reindex", map[string]any{"document_id": id}, 202)
		}
	}
	// Permission changes need no reindex to remove all derived results.
	admin.request("PUT", "/api/v1/documents/"+id, map[string]any{"version": 1, "visibility": "private"}, 200)
	for _, kind := range []string{"document", "block", "code", "task", "file", "comment", "tag", "ai_conversation"} {
		v := testJSONObject(t, viewer.request("GET", base+"&type="+kind, nil, 200))
		if len(v["results"].([]any)) != 0 {
			t.Fatal("revoked indexed content", kind, v)
		}
	}
	// Authorized readers never receive old-version snippets while the worker is behind.
	v := testJSONObject(t, admin.request("GET", base+"&type=code", nil, 200))
	if len(v["results"].([]any)) != 0 {
		t.Fatal("stale fragment exposed", v)
	}
}

func TestSearchFragmentsPreserveSourceAndCodeBoundaries(t *testing.T) {
	md := "---\ntitle: ignored\n---\n# 섹션\n\n- [ ] 진짜 작업\n\n```markdown\n- [ ] 예제일 뿐\n```\n\n<ul data-type=\"taskList\"><li data-type=\"taskItem\" data-checked=\"true\"><p>HTML 작업</p></li></ul>\n"
	fragments := projectSearchFragments(md)
	tasks, codes := 0, 0
	for _, f := range fragments {
		if f.Content != md[f.Start:f.End] {
			t.Fatal("non-source projection", f)
		}
		if f.Kind == "task" {
			tasks++
		}
		if f.Kind == "code" {
			codes++
		}
	}
	if tasks != 2 || codes != 1 {
		t.Fatal(tasks, codes, fragments)
	}
}

func TestPostgresSearchPopularityBoundedDistinctReaders(t *testing.T) {
	s, ts := integrationTestServer(t)
	c := newIntegrationTestClient(t, ts.URL)
	me := testJSONObject(t, c.request("POST", "/api/v1/auth/login", map[string]any{"email": "admin@example.test", "password": "Integration-Test-Password-2026!"}, 200))
	wid := str(testJSONObject(t, c.request("POST", "/api/v1/workspaces", map[string]any{"name": "Ranking verification"}, 200)), "id")
	d := testJSONObject(t, c.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "RANKING", "markdown": "RANKING content"}, 200))
	did := str(d, "id")
	if _, e := s.DB.Exec(t.Context(), "UPDATE documents SET updated_at=now()-interval '1 day' WHERE id=$1", did); e != nil {
		t.Fatal(e)
	}
	score := func() float64 {
		v := testJSONObject(t, c.request("GET", "/api/v1/search?workspace_id="+wid+"&type=document&q=RANKING", nil, 200))
		return v["results"].([]any)[0].(map[string]any)["score"].(float64)
	}
	before := score()
	for i := 0; i < 3; i++ {
		if _, e := s.DB.Exec(t.Context(), "INSERT INTO audit_logs(id,user_id,action,resource) VALUES($1,$2,'DOCUMENT_READ',$3)", newID(), str(me, "id"), did); e != nil {
			t.Fatal(e)
		}
	}
	after := score()
	if after-before < 0.0124 || after-before > 0.0126 {
		t.Fatal("same reader counted repeatedly or popularity missing", before, after)
	}
	if _, e := s.DB.Exec(t.Context(), "UPDATE audit_logs SET created_at=now()-interval '31 days' WHERE action='DOCUMENT_READ' AND resource=$1", did); e != nil {
		t.Fatal(e)
	}
	expired := score()
	if expired > before+0.0001 {
		t.Fatal("old readership still weighted", before, expired)
	}
}
