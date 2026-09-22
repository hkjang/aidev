package server

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestPostgresKnowledgeProposalCanonicalMergeCASRollbackAndPolicy(t *testing.T) {
	s, c, ctx, _, wid := jobTestFixture(t)
	source := testJSONObject(t, c.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "운영 변경 기준", "markdown": "# 운영 기준\n\n- [ ] 기준 확인"}, 200))
	did := str(source, "id")
	create := func(version int, markdown string) string {
		return str(testJSONObject(t, c.request("POST", "/api/v1/documents/"+did+"/proposals", map[string]any{"base_version": version, "markdown": markdown, "reason": "근거를 확인한 변경 초안", "provenance": "human", "consent": true}, 201)), "id")
	}
	id := create(1, "# 운영 기준\n\n- [x] 기준 확인\n- [ ] 후속 검증 SECRET_PROPOSAL_SENTINEL")
	record := testJSONObject(t, c.request("GET", "/api/v1/knowledge/proposals/"+id, nil, 200))
	if boolean(record, "stale") || !strings.Contains(str(record, "markdown"), "SECRET_PROPOSAL_SENTINEL") {
		t.Fatal(record)
	}
	var cipher string
	if err := s.DB.QueryRow(ctx, `SELECT ciphertext FROM knowledge_proposals WHERE id=$1`, id).Scan(&cipher); err != nil || strings.Contains(cipher, "SECRET_PROPOSAL_SENTINEL") {
		t.Fatal(err, cipher)
	}
	c.request("POST", "/api/v1/knowledge/proposals/"+id+"/merge", map[string]any{"revision": 1}, 400)
	c.request("PUT", "/api/v1/documents/"+did, map[string]any{"version": 1, "markdown": "동료가 먼저 변경"}, 200)
	c.request("POST", "/api/v1/knowledge/proposals/"+id+"/merge", map[string]any{"revision": 1, "consent": true}, 409)
	record = testJSONObject(t, c.request("GET", "/api/v1/knowledge/proposals/"+id, nil, 200))
	if !boolean(record, "stale") {
		t.Fatal(record)
	}
	c.request("POST", "/api/v1/knowledge/proposals/"+id+"/decision", map[string]any{"revision": 1, "status": "withdrawn", "note": "새 기준으로 다시 작성"}, 200)
	proposal := create(2, "---\ntags: [operations]\n---\n# 변경 기준\n\n- [x] 기준 확인\n")
	merged := testJSONObject(t, c.request("POST", "/api/v1/knowledge/proposals/"+proposal+"/merge", map[string]any{"revision": 1, "consent": true, "note": "차이를 확인하고 반영"}, 200))
	if number(merged, "version", 0) != 3 || !strings.Contains(string(jsonValue(merged["tags"])), "operations") {
		t.Fatal("canonical front matter/version lost", merged)
	}
	c.request("POST", "/api/v1/knowledge/proposals/"+proposal+"/merge", map[string]any{"revision": 1, "consent": true}, 200)
	var count int
	if err := s.DB.QueryRow(ctx, `SELECT count(*) FROM document_versions WHERE document_id=$1`, did).Scan(&count); err != nil || count != 3 {
		t.Fatal("replay created extra version", count, err)
	}
	pending := create(3, "# 다음 변경\n\n원자적 반영 시험")
	if _, err := s.DB.Exec(ctx, `CREATE FUNCTION madi_test_proposal_rollback() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'fixture rollback'; END $$; CREATE TRIGGER madi_test_proposal_rollback BEFORE INSERT ON knowledge_proposal_events FOR EACH ROW EXECUTE FUNCTION madi_test_proposal_rollback()`); err != nil {
		t.Fatal(err)
	}
	c.request("POST", "/api/v1/knowledge/proposals/"+pending+"/merge", map[string]any{"revision": 1, "consent": true}, 500)
	document := testJSONObject(t, c.request("GET", "/api/v1/documents/"+did, nil, 200))
	if number(document, "version", 0) != 3 {
		t.Fatal("failed proposal event committed document", document)
	}
	record = testJSONObject(t, c.request("GET", "/api/v1/knowledge/proposals/"+pending, nil, 200))
	if str(record, "status") != "open" || number(record, "revision", 0) != 1 {
		t.Fatal("failed document partially merged proposal", record)
	}
	if _, err := s.DB.Exec(ctx, `DROP TRIGGER madi_test_proposal_rollback ON knowledge_proposal_events`); err != nil {
		t.Fatal(err)
	}
	c.request("POST", "/api/v1/knowledge/proposals/"+pending+"/merge", map[string]any{"revision": 1, "consent": true}, 200)
	rejected := create(4, "# 반려 후보")
	c.request("POST", "/api/v1/knowledge/proposals/"+rejected+"/decision", map[string]any{"revision": 1, "status": "rejected", "note": "정책 비활성시 제외"}, 403)
	c.request("PUT", "/api/v1/admin/settings", map[string]any{"approval_enabled": true}, 200)
	c.request("POST", "/api/v1/knowledge/proposals/"+rejected+"/decision", map[string]any{"revision": 1, "status": "rejected", "note": "검토 근거 부족"}, 200)
	c.request("POST", "/api/v1/knowledge/proposals/"+rejected+"/merge", map[string]any{"revision": 2, "consent": true}, 409)
	// Publication remains governed by the ordinary save/approval policy.
	c.request("PUT", "/api/v1/admin/settings", map[string]any{"approval_enabled": false}, 200)
	c.request("PUT", "/api/v1/documents/"+did, map[string]any{"version": 4, "status": "published"}, 200)
	publishChange := create(5, "# 게시 내용 변경")
	c.request("PUT", "/api/v1/admin/settings", map[string]any{"approval_enabled": true}, 200)
	out := testJSONObject(t, c.request("POST", "/api/v1/knowledge/proposals/"+publishChange+"/merge", map[string]any{"revision": 1, "consent": true}, 200))
	if str(out, "status") != "draft" || number(out, "version", 0) != 6 {
		t.Fatal("proposal bypassed publish approval", out)
	}
}

func TestPostgresKnowledgeProposalMCPWorkspaceScopeAndExplicitConsent(t *testing.T) {
	_, c, _, _, wid := jobTestFixture(t)
	doc := testJSONObject(t, c.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "Agent 변경안", "markdown": "기존 기준"}, 200))
	did := str(doc, "id")
	key := testJSONObject(t, c.request("POST", "/api/v1/keys", map[string]any{"name": "변경안 Agent", "workspace_id": wid, "scopes": []string{"document:read", "document:write"}}, 201))
	c.token = str(key, "token")
	call := func(name string, args map[string]any) map[string]any {
		out := testJSONObject(t, c.request("POST", "/api/v1/mcp", map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{"name": name, "arguments": args}}, 200))
		return out["result"].(map[string]any)
	}
	args := map[string]any{"document_id": did, "base_version": 1, "markdown": "변경 근거 MCP_PROPOSAL_SENTINEL", "reason": "Agent가 제안", "provenance": "ai_assisted", "consent": false}
	if !boolean(call("create_document_proposal", args), "isError") {
		t.Fatal("MCP accepted no sharing consent")
	}
	args["consent"] = true
	made := call("create_document_proposal", args)
	if boolean(made, "isError") {
		t.Fatal(made)
	}
	// REST list is shared with MCP; the resource is not an unconstrained agent store.
	list := c.request("GET", "/api/v1/knowledge/proposals?workspace_id="+wid, nil, 200)
	var entries []map[string]any
	if json.Unmarshal(list, &entries) != nil || len(entries) != 1 {
		t.Fatal(string(list))
	}
	id := str(entries[0], "id")
	inspected := call("get_document_proposal", map[string]any{"proposal_id": id})
	if boolean(inspected, "isError") || !strings.Contains(string(jsonValue(inspected)), "MCP_PROPOSAL_SENTINEL") {
		t.Fatal(inspected)
	}
	merged := call("merge_document_proposal", map[string]any{"proposal_id": id, "revision": 1, "consent": true})
	if boolean(merged, "isError") {
		t.Fatal(merged)
	}
	if boolean(call("get_document_impact", map[string]any{"document_id": did, "from_version": 1, "depth": 1}), "isError") {
		t.Fatal("MCP impact failed")
	}
	c.token = ""
	other := str(testJSONObject(t, c.request("POST", "/api/v1/workspaces", map[string]any{"name": "다른 범위"}, 200)), "id")
	key = testJSONObject(t, c.request("POST", "/api/v1/keys", map[string]any{"name": "다른 범위키", "workspace_id": other, "scopes": []string{"document:read", "document:write"}}, 201))
	c.token = str(key, "token")
	c.request("GET", "/api/v1/knowledge/proposals/"+id, nil, 404)
	if !boolean(call("get_document_proposal", map[string]any{"proposal_id": id}), "isError") {
		t.Fatal("cross workspace MCP proposal leak")
	}
}
