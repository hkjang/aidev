package server

import (
	"strings"
	"testing"
)

func TestPostgresLegacyAndMCPSearchAuditDoesNotRecordRawQueries(t *testing.T) {
	s, c, wid, _ := agentTestSetup(t)
	const query = "PRIVATE_LEGACY_SEARCH_SENTINEL_secret@example.test"
	c.request("GET", "/api/v1/documents?workspace_id="+wid+"&q="+query, nil, 200)
	issued := testJSONObject(t, c.request("POST", "/api/v1/keys", map[string]any{"name": "검색 감사 검증", "workspace_id": wid, "scopes": []string{"search:read"}, "expires_in_days": 1, "rate_limit": 300}, 201))
	key := newIntegrationTestClient(t, c.base)
	key.token = str(issued, "token")
	for _, tool := range []string{"search_documents", "search_workspace"} {
		result := key.request("POST", "/mcp", map[string]any{"jsonrpc": "2.0", "id": newID(), "method": "tools/call", "params": map[string]any{"name": tool, "arguments": map[string]any{"query": query, "workspace_id": wid}}}, 200)
		if strings.Contains(string(result), `"isError":true`) {
			t.Fatal(string(result))
		}
	}
	var n, safe int
	if e := s.DB.QueryRow(t.Context(), `SELECT count(*) FROM audit_logs WHERE details::text LIKE $1`, "%"+query+"%").Scan(&n); e != nil || n != 0 {
		t.Fatalf("search query leaked to shared audit %d %v", n, e)
	}
	if e := s.DB.QueryRow(t.Context(), `SELECT count(*) FROM audit_logs WHERE action='SEARCH' AND resource=$1 AND NOT (details ? 'query') AND (((details->>'query_bytes')::int=$2 AND details ? 'results') OR details ? 'result_count')`, wid, len(query)).Scan(&safe); e != nil || safe != 3 {
		t.Fatalf("redacted search counts missing %d %v", safe, e)
	}
	if e := s.DB.QueryRow(t.Context(), `SELECT count(*) FROM search_history_entries WHERE query=$1`, query).Scan(&n); e != nil || n != 0 {
		t.Fatalf("query stored without personal consent %d %v", n, e)
	}
}

func TestPostgresMCPSearchScopeCannotReadRawDocumentsOrOtherResources(t *testing.T) {
	s, c, wid, _ := agentTestSetup(t)
	const tail = "RAW_BODY_TAIL_UNAVAILABLE"
	doc := testJSONObject(t, c.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "needle 운영 기준", "markdown": "needle " + strings.Repeat("safe text ", 120) + tail, "visibility": "private", "block_metadata": map[string]any{"sensitive": "RAW_BLOCK_METADATA_UNAVAILABLE"}}, 200))
	did := str(doc, "id")
	space := testJSONObject(t, c.request("POST", "/api/v1/spaces", map[string]any{"workspace_id": wid, "name": "문서 범위 검증 공간", "visibility": "workspace"}, 200))
	sid := str(space, "id")
	if _, e := s.DB.Exec(t.Context(), `UPDATE documents SET space_id=$2 WHERE id=$1`, did, sid); e != nil {
		t.Fatal(e)
	}
	db := testJSONObject(t, c.request("POST", "/api/v1/databases", map[string]any{"workspace_id": wid, "name": "needle DATABASE_SCOPE_UNAVAILABLE"}, 200))
	c.request("POST", "/api/v1/databases/"+str(db, "id")+"/rows", map[string]any{"values": map[string]any{"title": "needle ROW_SCOPE_UNAVAILABLE"}}, 200)
	other := testJSONObject(t, c.request("POST", "/api/v1/admin/users", map[string]any{"email": "mcp-private@example.test", "name": "다른 소유자", "password": "Private-Actor-Password-2026!", "role": "editor"}, 200))
	c.request("PUT", "/api/v1/workspaces/"+wid+"/members", map[string]any{"email": "mcp-private@example.test", "role": "editor"}, 200)
	private := testJSONObject(t, c.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "needle OTHER_PRIVATE_UNAVAILABLE", "markdown": "needle private context", "visibility": "private"}, 200))
	if _, e := s.DB.Exec(t.Context(), `UPDATE documents SET owner_id=$2 WHERE id=$1`, str(private, "id"), str(other, "id")); e != nil {
		t.Fatal(e)
	}
	issue := func(scopes []string) *integrationTestClient {
		t.Helper()
		v := testJSONObject(t, c.request("POST", "/api/v1/keys", map[string]any{"name": "MCP 범위 검증", "workspace_id": wid, "scopes": scopes, "expires_in_days": 1, "rate_limit": 300}, 201))
		key := newIntegrationTestClient(t, c.base)
		key.token = str(v, "token")
		return key
	}
	search := issue([]string{"search:read"})
	for _, suffix := range []string{"", "?q=needle", "?q=%20"} {
		search.request("GET", "/api/v1/documents"+suffix, nil, 403)
	}
	search.request("GET", "/api/v1/documents/"+did, nil, 403)
	search.request("GET", "/api/v1/spaces/"+sid+"/documents", nil, 403)
	database := issue([]string{"database:read"})
	database.request("GET", "/api/v1/spaces?workspace_id="+wid, nil, 200)
	database.request("GET", "/api/v1/spaces/"+sid+"/documents", nil, 403)
	for _, tool := range []string{"search_documents", "search_workspace"} {
		result := search.request("POST", "/mcp", map[string]any{"jsonrpc": "2.0", "id": newID(), "method": "tools/call", "params": map[string]any{"name": tool, "arguments": map[string]any{"query": "needle", "workspace_id": wid}}}, 200)
		encoded := string(result)
		if strings.Contains(encoded, `"isError":true`) || !strings.Contains(encoded, "needle 운영 기준") || !strings.Contains(encoded, "snippet") {
			t.Fatalf("search result contract %s", result)
		}
		for _, secret := range []string{tail, "RAW_BLOCK_METADATA_UNAVAILABLE", "DATABASE_SCOPE_UNAVAILABLE", "ROW_SCOPE_UNAVAILABLE", "OTHER_PRIVATE_UNAVAILABLE"} {
			if strings.Contains(encoded, secret) {
				t.Fatalf("search scope leaked %s", secret)
			}
		}
	}
	read := issue([]string{"document:read"})
	for _, client := range []*integrationTestClient{read, c} {
		raw := client.request("GET", "/api/v1/spaces/"+sid+"/documents", nil, 200)
		if !strings.Contains(string(raw), "needle 운영 기준") || strings.Contains(string(raw), tail) || strings.Contains(string(raw), `"markdown":`) || strings.Contains(string(raw), `"block_metadata":`) {
			t.Fatalf("space document summary contract %s", raw)
		}
		if client == read && strings.Contains(string(raw), `"can_write":true`) {
			t.Fatal("read-only key advertised write capability")
		}
	}
	if raw := read.request("GET", "/api/v1/documents/"+did, nil, 200); !strings.Contains(string(raw), tail) {
		t.Fatal("document:read detail contract broken")
	}
	if raw := c.request("GET", "/api/v1/documents?workspace_id="+wid+"&q=%20", nil, 200); !strings.Contains(string(raw), "needle 운영 기준") || strings.Contains(string(raw), tail) {
		t.Fatal("normal cookie bounded document listing broken")
	}
}
