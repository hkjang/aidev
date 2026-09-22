package server

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestGraphProjectionAmbiguityAndSourceSemantics(t *testing.T) {
	links, tags := projectGraphReferences("---\ntags: [운영]\nlinks: '[[숨은 YAML]]'\n---\n# 제목\n[[운영 문서#절차|표시]] [[운영 문서]] #지식\n`[[코드]]` \\[[이스케이프]]\n\n```md\n[[예제]]\n```\n")
	if len(links) != 1 || links[0] != "운영 문서" || len(tags) != 2 {
		t.Fatal(links, tags)
	}
	id, a, b := newID(), newID(), newID()
	edges, unresolved := resolveGraphReferences([]map[string]any{
		{"id": id, "title": "원본", "links": []string{"동명", a, "없음"}, "parent_id": b},
		{"id": a, "title": "동명"}, {"id": b, "title": "동명"},
	})
	if len(edges) != 2 || len(unresolved) != 2 || str(unresolved[0], "reason") != "ambiguous" {
		t.Fatal(edges, unresolved)
	}
}

func TestPostgresKnowledgeGraphCurrentACLRelationsAndBacklinks(t *testing.T) {
	s, admin, ctx, _, wid := jobTestFixture(t)
	create := func(title, md, visibility string) map[string]any {
		t.Helper()
		return testJSONObject(t, admin.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": title, "markdown": md, "visibility": visibility}, 200))
	}
	target := create("연결 대상", "# 기준\n", "workspace")
	private := create("PRIVATE_GRAPH_TARGET_SENTINEL", "# 비공개", "private")
	source := create("연결 원본", "[[연결 대상#기준|확인]] [[없는 문서]] #연결\n\n`[[숨은 예제]]`\n", "workspace")
	id, tid, pid := str(source, "id"), str(target, "id"), str(private, "id")
	admin.request("POST", "/api/v1/documents/"+id+"/relations", map[string]any{"target_id": tid, "type": "related", "expected_version": 1}, 200)
	admin.request("POST", "/api/v1/documents/"+id+"/relations", map[string]any{"target_id": tid, "type": "related", "expected_version": 1}, 200)
	admin.request("POST", "/api/v1/documents/"+id+"/relations", map[string]any{"target_id": pid, "type": "reference", "expected_version": 1}, 200)
	admin.request("POST", "/api/v1/documents/"+id+"/relations", map[string]any{"target_id": id, "type": "related", "expected_version": 1}, 400)
	admin.request("POST", "/api/v1/documents/"+id+"/relations", map[string]any{"target_id": tid, "type": "parent", "expected_version": 1}, 400)
	graph := func(client *integrationTestClient) map[string]any {
		return testJSONObject(t, client.request("GET", "/api/v1/graph?workspace_id="+wid, nil, 200))
	}
	fresh := graph(admin)
	if !strings.Contains(string(jsonValue(fresh)), `"origin":"wiki"`) || strings.Contains(string(jsonValue(fresh)), "숨은 예제") {
		t.Fatal("fresh graph semantic projection", fresh)
	}
	for _, doc := range []string{id, tid, pid} {
		if _, e := s.indexSearchDocument(ctx, doc); e != nil {
			t.Fatal(e)
		}
	}
	indexed := graph(admin)
	if !strings.Contains(string(jsonValue(indexed)), `"indexed":true`) {
		t.Fatal("projection not used", indexed)
	}
	user := testJSONObject(t, admin.request("POST", "/api/v1/admin/users", map[string]any{"email": "graph-reader@example.test", "name": "그래프 열람", "role": "viewer", "password": "Graph-reader-password-2026!"}, 200))
	admin.request("PUT", "/api/v1/workspaces/"+wid+"/members", map[string]any{"email": user["email"], "role": "viewer"}, 200)
	reader := newIntegrationTestClient(t, admin.base)
	reader.request("POST", "/api/v1/auth/login", map[string]any{"email": user["email"], "password": "Graph-reader-password-2026!"}, 200)
	visible := string(jsonValue(graph(reader)))
	if strings.Contains(visible, "PRIVATE_GRAPH_TARGET_SENTINEL") || strings.Contains(visible, pid) {
		t.Fatal("private relation target leaked", visible)
	}
	reader.request("POST", "/api/v1/documents/"+id+"/relations", map[string]any{"target_id": tid, "type": "related", "expected_version": 1}, 403)
	admin.request("PUT", "/api/v1/documents/"+tid, map[string]any{"version": 1, "title": "변경된 대상"}, 200)
	backlinks := admin.request("GET", "/api/v1/documents/"+tid+"/backlinks", nil, 200)
	if !strings.Contains(string(backlinks), "연결 원본") {
		t.Fatal("rename alias backlink lost", string(backlinks))
	}
	admin.request("DELETE", "/api/v1/documents/"+id+"/relations/"+tid+"?type=related&expected_version=2", nil, 409)
	admin.request("DELETE", "/api/v1/documents/"+id+"/relations/"+tid+"?type=related&expected_version=1", nil, 200)
	create("연결 대상", "# 동명", "workspace")
	if !strings.Contains(string(jsonValue(graph(admin))), `"reason":"ambiguous"`) {
		t.Fatal("ambiguous alias resolved arbitrarily")
	}
	var list []map[string]any
	if e := json.Unmarshal(admin.request("GET", "/api/v1/documents/"+tid+"/backlinks", nil, 200), &list); e != nil {
		t.Fatal(e)
	}
	for _, row := range list {
		if str(row, "id") == id {
			t.Fatal("ambiguous name accepted as target backlink", row)
		}
	}
	admin.request("PUT", "/api/v1/documents/"+id, map[string]any{"version": 1, "markdown": "[[" + tid + "]]"}, 200)
	if !strings.Contains(string(admin.request("GET", "/api/v1/documents/"+tid+"/backlinks", nil, 200)), "연결 원본") {
		t.Fatal("stable ID backlink missing while index stale")
	}
}
