package server

import (
	"encoding/json"
	"strings"
	"testing"
)

const documentQueryExample = `{"version":1,"source":"documents","columns":[{"field":"title","label":"문서"},{"field":"property.담당일"}],"properties":{"담당일":"date"},"filters":[{"field":"title","op":"contains","value":{"parameter":"검색어"}}],"parameters":{"검색어":{"type":"string","required":true}},"order_by":[{"field":"updated_at","direction":"desc"}],"limit":20}`

func TestDocumentQueryTypedGrammarAndMarkdownIsolation(t *testing.T) {
	q, e := parseDocumentQuery(documentQueryExample)
	if e != nil {
		t.Fatal(e)
	}
	params, e := documentQueryParameters(q, map[string]json.RawMessage{"검색어": json.RawMessage(`"운영"`)})
	if e != nil || !documentQueryMatches(q, map[string]any{"title": "운영 안내"}, params) || documentQueryMatches(q, map[string]any{"title": "별도 문서"}, params) {
		t.Fatal(params, e)
	}
	for _, raw := range []string{
		strings.Replace(documentQueryExample, `"version":1`, `"version":1,"version":1`, 1),
		strings.Replace(documentQueryExample, `"version":1`, `"Version":1,"version":1`, 1),
		strings.Replace(documentQueryExample, `"field":"title"`, `"Field":"title"`, 1),
		strings.Replace(documentQueryExample, `"limit":20`, `"limit":101`, 1),
		strings.Replace(documentQueryExample, `"source":"documents"`, `"source":"sql"`, 1),
		strings.Replace(documentQueryExample, `"contains"`, `"eval"`, 1),
		strings.Replace(documentQueryExample, `"field":"title"`, `"field":"markdown"`, 1),
		strings.Replace(documentQueryExample, `"limit":20`, `"limit":20,"sql":"SELECT 1"`, 1),
		strings.Replace(documentQueryExample, `"type":"string"`, `"type":"number"`, 1),
		strings.Replace(documentQueryExample, `"parameter":"검색어"`, `"parameter":"검색어","eval":"process.env"`, 1),
		documentQueryExample + `{}`, strings.Repeat(" ", 5121) + documentQueryExample,
	} {
		if _, e := parseDocumentQuery(raw); e == nil {
			t.Fatalf("unsafe grammar accepted %s", raw)
		}
	}
	if _, e = documentQueryParameters(q, map[string]json.RawMessage{"검색어": json.RawMessage(`42`)}); e == nil {
		t.Fatal("wrong parameter type accepted")
	}
	md := "---\nexample: |\n  ```madi-query\n  {}\n  ```\n---\n\n````text\n```madi-query\n{}\n```\n````\n\n```madi-query\n" + documentQueryExample + "\n```\n"
	before := md
	fences, limited := documentQueryFences(md)
	if limited || len(fences) != 1 || fences[0].Definition == nil || fences[0].Hash != digest(documentQueryExample) || md != before {
		t.Fatal(fences, limited)
	}
	props, bad := documentQueryProperties("---\n담당일: 2026-09-09\n숫자: 42\n위험: &anchor [a, b]\n별칭: *anchor\n---\n본문", map[string]string{"담당일": "date", "숫자": "number", "위험": "string", "별칭": "string"})
	if !bad || props["property.담당일"] != "2026-09-09" || props["property.숫자"] != float64(42) || props["property.위험"] != nil {
		t.Fatal(props, bad)
	}
	q.Filters = []documentQueryFilter{{Field: "tags", Op: "exists"}}
	if !documentQueryMatches(q, map[string]any{"tags": []any{"태그"}}, nil) {
		t.Fatal("array exists rejected")
	}
}
func documentQueryCreate(t *testing.T, c *integrationTestClient, wid, title, md, visibility string) string {
	t.Helper()
	return str(testJSONObject(t, c.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": title, "markdown": md, "visibility": visibility}, 200)), "id")
}
func documentQueryInput(t *testing.T, c *integrationTestClient, id string, params map[string]any) map[string]any {
	t.Helper()
	meta := testJSONObject(t, c.request("GET", "/api/v1/documents/"+id+"/queries", nil, 200))
	items := meta["queries"].([]any)
	if len(items) == 0 {
		t.Fatal(meta)
	}
	f := items[0].(map[string]any)
	return map[string]any{"document_version": meta["document_version"], "query_hash": f["hash"], "parameters": params}
}
func TestPostgresDocumentQueryCurrentACLPropertiesTokenAndNoMutation(t *testing.T) {
	s, c, ctx, _, wid := jobTestFixture(t)
	viewer := testJSONObject(t, c.request("POST", "/api/v1/admin/users", map[string]any{"email": "query-viewer@example.test", "name": "조회 사용자", "role": "viewer", "password": "Query-Viewer-Password-2026!"}, 200))
	c.request("PUT", "/api/v1/workspaces/"+wid+"/members", map[string]any{"email": viewer["email"], "role": "viewer"}, 200)
	v := newIntegrationTestClient(t, c.base)
	v.request("POST", "/api/v1/auth/login", map[string]any{"email": viewer["email"], "password": "Query-Viewer-Password-2026!"}, 200)
	public := documentQueryCreate(t, c, wid, "운영 가이드", "---\n담당일: 2026-09-09\n---\n본문 원문 SENTINEL_NOT_A_COLUMN", "workspace")
	secret := documentQueryCreate(t, c, wid, "운영 PRIVATE_SENTINEL", "---\n담당일: 2026-09-10\n---\n기밀", "private")
	parent := documentQueryCreate(t, c, wid, "조회 대시보드", "# 조회\n\n```madi-query\n"+documentQueryExample+"\n```\n", "workspace")
	in := documentQueryInput(t, v, parent, map[string]any{"검색어": "운영"})
	out := testJSONObject(t, v.request("POST", "/api/v1/documents/"+parent+"/queries/execute", in, 200))
	raw, _ := json.Marshal(out)
	if strings.Contains(string(raw), "PRIVATE_SENTINEL") || strings.Contains(string(raw), "SENTINEL_NOT_A_COLUMN") || len(out["rows"].([]any)) != 1 || out["rows"].([]any)[0].(map[string]any)["values"].(map[string]any)["property.담당일"] != "2026-09-09" {
		t.Fatal(string(raw))
	}
	check := map[string]any{"validation_token": out["validation_token"]}
	v.request("POST", "/api/v1/documents/"+parent+"/queries/check", check, 200)
	c.request("POST", "/api/v1/documents/"+parent+"/queries/check", check, 409)
	var count, version int
	var md string
	if e := s.DB.QueryRow(ctx, `SELECT version,markdown,(SELECT count(*) FROM document_versions WHERE document_id=$1) FROM documents WHERE id=$1`, parent).Scan(&version, &md, &count); e != nil || version != 1 || count != 1 || !strings.Contains(md, documentQueryExample) {
		t.Fatal("read query mutated canonical source", e, version, count)
	}
	issued := testJSONObject(t, c.request("POST", "/api/v1/keys", map[string]any{"user_id": viewer["id"], "name": "조회 전용", "workspace_id": wid, "scopes": []string{"document:read"}, "expires_in_days": 1}, 201))
	key := newIntegrationTestClient(t, c.base)
	key.token = str(issued, "token")
	key.request("POST", "/api/v1/documents/"+parent+"/queries/execute", in, 200)
	mcp := key.request("POST", "/api/v1/mcp", map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{"name": "execute_document_query", "arguments": map[string]any{"document_id": parent, "document_version": in["document_version"], "query_hash": in["query_hash"], "parameters": in["parameters"]}}}, 200)
	if !strings.Contains(string(mcp), "운영 가이드") || strings.Contains(string(mcp), "PRIVATE_SENTINEL") || strings.Contains(string(mcp), `"isError":true`) {
		t.Fatal(string(mcp))
	}
	bad := map[string]any{"document_version": 1, "query_hash": in["query_hash"], "parameters": in["parameters"], "definition": map[string]any{"sql": "SELECT password_hash FROM users"}}
	v.request("POST", "/api/v1/documents/"+parent+"/queries/execute", bad, 400)
	if _, e := s.DB.Exec(ctx, "UPDATE documents SET visibility='private' WHERE id=$1", public); e != nil {
		t.Fatal(e)
	}
	v.request("POST", "/api/v1/documents/"+parent+"/queries/check", check, 409)
	out = testJSONObject(t, v.request("POST", "/api/v1/documents/"+parent+"/queries/execute", in, 200))
	if len(out["rows"].([]any)) != 0 {
		t.Fatal(out)
	}
	key.request("GET", "/api/v1/documents/"+secret+"/queries", nil, 404)
	c.request("PUT", "/api/v1/admin/settings", map[string]any{"feature_flags": map[string]any{"document-queries": false}}, 200)
	v.request("POST", "/api/v1/documents/"+parent+"/queries/execute", in, 403)
	keyID := str(issued["key"].(map[string]any), "id")
	c.request("DELETE", "/api/v1/keys/"+keyID, nil, 200)
	key.request("GET", "/api/v1/documents/"+parent+"/queries", nil, 401)
}
func TestPostgresDocumentQueryTasksRelationsIntermediateACLAndStale(t *testing.T) {
	s, c, ctx, _, wid := jobTestFixture(t)
	taskID := newID()
	tasks := documentQueryCreate(t, c, wid, "실행 할 일", "- [ ] [상태 점검](/app/tasks?task="+taskID+")\n- [x] 완료 점검\n\n```text\n- [ ] CODE_SENTINEL\n```", "workspace")
	q := `{"version":1,"source":"tasks","document_ids":["` + tasks + `"],"columns":[{"field":"text"},{"field":"done"},{"field":"priority"}],"filters":[{"field":"done","op":"eq","value":false}],"limit":20}`
	parent := documentQueryCreate(t, c, wid, "할 일 조회", "```madi-query\n"+q+"\n```", "workspace")
	in := documentQueryInput(t, c, parent, nil)
	out := testJSONObject(t, c.request("POST", "/api/v1/documents/"+parent+"/queries/execute", in, 200))
	if len(out["rows"].([]any)) != 1 || strings.Contains(string(jsonValue(out)), "CODE_SENTINEL") {
		t.Fatal(out)
	}
	if _, e := s.DB.Exec(ctx, "INSERT INTO task_details(document_id,task_id,status,priority,updated_by) SELECT $1,$2,'doing','urgent',owner_id FROM documents WHERE id=$1", tasks, taskID); e != nil {
		t.Fatal(e)
	}
	c.request("POST", "/api/v1/documents/"+parent+"/queries/check", map[string]any{"validation_token": out["validation_token"]}, 409)
	viewer := testJSONObject(t, c.request("POST", "/api/v1/admin/users", map[string]any{"email": "query-path@example.test", "name": "경로 조회자", "role": "viewer", "password": "Query-Path-Password-2026!"}, 200))
	c.request("PUT", "/api/v1/workspaces/"+wid+"/members", map[string]any{"email": viewer["email"], "role": "viewer"}, 200)
	v := newIntegrationTestClient(t, c.base)
	v.request("POST", "/api/v1/auth/login", map[string]any{"email": viewer["email"], "password": "Query-Path-Password-2026!"}, 200)
	root := documentQueryCreate(t, c, wid, "출발 문서", "```madi-query\n{\"version\":1,\"source\":\"relations\",\"columns\":[{\"field\":\"target_title\"},{\"field\":\"depth\"}],\"depth\":3,\"limit\":20}\n```", "workspace")
	middle := documentQueryCreate(t, c, wid, "경로 내부", "원문", "workspace")
	end := documentQueryCreate(t, c, wid, "최종 대상", "원문", "workspace")
	for _, edge := range [][2]string{{root, middle}, {middle, end}} {
		c.request("POST", "/api/v1/documents/"+edge[0]+"/relations", map[string]any{"target_id": edge[1], "type": "reference", "expected_version": 1}, 200)
	}
	in = documentQueryInput(t, v, root, nil)
	out = testJSONObject(t, v.request("POST", "/api/v1/documents/"+root+"/queries/execute", in, 200))
	if len(out["rows"].([]any)) != 2 {
		t.Fatal(out)
	}
	v.request("POST", "/api/v1/documents/"+root+"/queries/check", map[string]any{"validation_token": out["validation_token"]}, 200)
	if _, e := s.DB.Exec(ctx, "UPDATE documents SET visibility='private' WHERE id=$1", middle); e != nil {
		t.Fatal(e)
	}
	v.request("POST", "/api/v1/documents/"+root+"/queries/check", map[string]any{"validation_token": out["validation_token"]}, 409)
	out = testJSONObject(t, v.request("POST", "/api/v1/documents/"+root+"/queries/execute", in, 200))
	if len(out["rows"].([]any)) != 0 {
		t.Fatal("hidden intermediate path expanded", out)
	}
}
