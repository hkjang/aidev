package server

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestPostgresLimitedAgentAndSupportGraphs(t *testing.T) {
	s, c, target, wid, uid, tid := supportTestSetup(t)
	source, destination := newID(), newID()
	aliases := []string{"연결 별칭"}
	for i := 0; i < 40; i++ {
		aliases = append(aliases, strings.Repeat("a", i+1))
	}
	aliases = append(aliases, strings.Repeat("OVERSIZE_ALIAS", 100000))
	markdown := "[[연결 대상]]\n\n" + strings.Repeat("원", 20000) + "\n\n[[연결 대상]]\n"
	if _, e := s.DB.Exec(t.Context(), `INSERT INTO documents(id,workspace_id,owner_id,title,markdown,aliases) VALUES($1,$2,$3,'연결 시작',$4,$5),($6,$2,$3,'연결 대상','본문','[]')`, source, wid, uid, markdown, jsonValue(aliases), destination); e != nil {
		t.Fatal(e)
	}
	secret := testJSONObject(t, target.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "OTHER_PRIVATE_GRAPH_SENTINEL", "visibility": "private", "markdown": "[[연결 대상]]"}, 200))
	rows, e := s.rows(t.Context(), "SELECT "+limitedGraphDocumentJSON+" FROM documents d WHERE id=$1", source)
	if e != nil || len(rows) != 1 {
		t.Fatalf("limited graph query %v", e)
	}
	if len(listStrings(rows[0]["aliases"])) != 32 || !boolean(rows[0], "aliases_truncated") || len(jsonValue(rows)) > 128<<10 {
		t.Fatal("alias projection did not bound source bytes")
	}
	// A long body is clipped independently of alias omission.
	if _, e := s.DB.Exec(t.Context(), `UPDATE documents SET markdown=$2 WHERE id=$1`, source, markdown+strings.Repeat("길", 16000)); e != nil {
		t.Fatal(e)
	}
	p := &Principal{ID: uid, Role: "admin", WorkspaceID: wid}
	a := workspaceAgent{WorkspaceID: wid, DocumentIDs: []string{source, destination, str(secret, "id")}, Tools: []string{"get_graph"}}
	var call agentToolCall
	if e := json.Unmarshal([]byte(`{"id":"graph","type":"function","function":{"name":"get_graph","arguments":"{}"}}`), &call); e != nil {
		t.Fatal(e)
	}
	value, sources, e := s.agentReadTool(t.Context(), p, a, call)
	if e != nil {
		t.Fatal(e)
	}
	result := value.(map[string]any)
	if !boolean(result, "truncated") || number(result, "alias_limit", 0) != 32 || number(result, "source_character_limit", 0) != 32000 || len(sources) != 2 || strings.Contains(string(jsonValue(result)), "OTHER_PRIVATE_GRAPH_SENTINEL") {
		t.Fatalf("Agent graph diagnostics or ACL %s", jsonValue(result))
	}
	c.request("PUT", "/api/v1/admin/settings", map[string]any{"support_enabled": true, "support_operator_ids": []string{uid}}, 200)
	session := supportTestStart(t, c, tid)
	result = testJSONObject(t, c.request("GET", "/api/v1/support/sessions/"+session+"/graph?workspace_id="+wid, nil, 200))
	if !boolean(result, "truncated") || number(result, "alias_limit", 0) != 32 || strings.Contains(string(jsonValue(result)), "OTHER_PRIVATE_GRAPH_SENTINEL") {
		t.Fatalf("support graph diagnostics or ACL %s", jsonValue(result))
	}
}
