package server

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestNormalizeValidPeriodsBoundariesAndOverlap(t *testing.T) {
	for _, in := range [][]knowledgeValidPeriod{
		{{2147483648, "2026-01-01", ""}},
		{{1, "2026-02-29", ""}}, {{0, "2026-01-01", ""}}, {{1, "2026-01-02", "2026-01-01"}}, {{1, "0000-01-01", ""}}, {{1, "2026-01-01", ""}, {2, "2026-02-01", ""}}, {{1, "2026-01-01", "2026-03-01"}, {2, "2026-02-28", ""}},
	} {
		if _, err := normalizeValidPeriods(in); err == nil {
			t.Fatal("accepted invalid periods", in)
		}
	}
	out, err := normalizeValidPeriods([]knowledgeValidPeriod{{2, "2026-03-01", ""}, {1, "2024-02-29", "2026-03-01"}})
	if err != nil || out[0].Version != 1 {
		t.Fatal(out, err)
	}
	if out, err = normalizeValidPeriods(nil); err != nil || out == nil {
		t.Fatal(out, err)
	}
}

func TestPostgresKnowledgeValidityHistoricalSearchCASPrivacyAndRollback(t *testing.T) {
	s, c, ctx, _, wid := jobTestFixture(t)
	doc := testJSONObject(t, c.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "운영 유효 기준", "markdown": "쿠버네티스 장애 대응 OLD_VALIDITY_SENTINEL"}, 200))
	did := str(doc, "id")
	c.request("PUT", "/api/v1/documents/"+did, map[string]any{"version": 1, "markdown": "쿠버네티스 운영 정책 NEW_VALIDITY_SENTINEL"}, 200)
	periods := []knowledgeValidPeriod{{1, "2026-01-01", "2026-07-01"}, {2, "2026-07-01", ""}}
	write := func(rev, version int, ps []knowledgeValidPeriod, status int) {
		c.request("PUT", "/api/v1/documents/"+did+"/validity", map[string]any{"revision": rev, "document_version": version, "periods": ps, "reason": "변경 이유 SECRET_VALIDITY_REASON", "consent": true}, status)
	}
	initial := testJSONObject(t, c.request("GET", "/api/v1/documents/"+did+"/validity", nil, 200))
	if number(initial, "revision", -1) != 0 {
		t.Fatal(initial)
	}
	write(0, 2, periods, 200)
	check := func(documents any, want bool, status int) {
		out := c.request("POST", "/api/v1/knowledge/time-check", map[string]any{"workspace_id": wid, "protection_revision": number(initial, "protection_revision", 1), "documents": documents}, status)
		if status == 200 && boolean(testJSONObject(t, out), "valid") != want {
			t.Fatal("unexpected batch validity", string(out))
		}
	}
	check(nil, false, 400)
	check([]any{}, true, 200)
	check([]map[string]any{{"id": did, "version": 2, "revision": 1}}, true, 200)
	check([]map[string]any{{"id": did, "version": 1, "revision": 1}}, false, 200)
	check([]map[string]any{{"id": did, "version": int64(2147483648), "revision": 1}}, false, 400)
	check([]map[string]any{{"id": did, "version": 2, "revision": 1}, {"id": did, "version": 2, "revision": 1}}, false, 400)
	write(0, 2, periods, 409)
	write(1, 1, periods, 409)
	write(1, 2, []knowledgeValidPeriod{{99, "2026-01-01", ""}}, 400)
	c.request("PUT", "/api/v1/documents/"+did+"/validity", map[string]any{"revision": 1, "document_version": 2, "periods": periods, "reason": "미동의"}, 400)
	var cipher string
	if err := s.DB.QueryRow(ctx, `SELECT reason_ciphertext FROM knowledge_validity_history WHERE document_id=$1 AND revision=1`, did).Scan(&cipher); err != nil || strings.Contains(cipher, "SECRET_VALIDITY_REASON") {
		t.Fatal(err, cipher)
	}
	get := func(date string) map[string]any {
		return testJSONObject(t, c.request("GET", "/api/v1/documents/"+did+"/valid-at?date="+date+"&revision=1", nil, 200))
	}
	old := get("2026-06-30")
	if number(old, "version", 0) != 1 || !boolean(old, "historical") || !strings.Contains(str(old, "markdown"), "OLD_VALIDITY_SENTINEL") {
		t.Fatal(old)
	}
	if next := get("2026-07-01"); number(next, "version", 0) != 2 || boolean(next, "historical") {
		t.Fatal(next)
	}
	c.request("GET", "/api/v1/documents/"+did+"/valid-at?date=2025-12-31", nil, 404)
	c.request("GET", "/api/v1/documents/"+did+"/valid-at?date=2026-02-29", nil, 400)
	c.request("GET", "/api/v1/documents/"+did+"/valid-at?date=2026-06-30&revision=9", nil, 409)
	c.request("PUT", "/api/v1/workspaces/"+wid+"/search-dictionary", map[string]any{"revision": 0, "confirm_shared": true, "entries": []map[string]any{{"canonical": "쿠버네티스", "aliases": []string{"K8s", "Kubernetes"}}}}, 200)
	search := func(date, q string) map[string]any {
		return testJSONObject(t, c.request("GET", "/api/v1/knowledge/time-search?workspace_id="+wid+"&date="+date+"&q="+q, nil, 200))
	}
	result := search("2026-06-30", "K8s")
	if len(result["results"].([]any)) != 1 || !strings.Contains(string(jsonValue(result)), "OLD_VALIDITY_SENTINEL") {
		t.Fatal(result)
	}
	if result = search("2025-12-31", ""); str(result, "outcome") != "no_match_in_declared_periods" {
		t.Fatal(result)
	}
	// An event failure rolls back removal and replacement of the old periods.
	if _, err := s.DB.Exec(ctx, `CREATE FUNCTION validity_fail_fixture() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'fixture failure'; END $$; CREATE TRIGGER validity_fail_fixture BEFORE INSERT ON knowledge_validity_history FOR EACH ROW EXECUTE FUNCTION validity_fail_fixture()`); err != nil {
		t.Fatal(err)
	}
	write(1, 2, []knowledgeValidPeriod{}, 500)
	get("2026-06-30")
	if _, err := s.DB.Exec(ctx, `DROP TRIGGER validity_fail_fixture ON knowledge_validity_history`); err != nil {
		t.Fatal(err)
	}
	if _, err := s.DB.Exec(ctx, `INSERT INTO knowledge_validity_periods(document_id,ordinal,document_version,valid_from) VALUES($1,4,2,'2026-06-01')`, did); err == nil {
		t.Fatal("database accepted overlapping validity")
	}
	// Current read scope is required, even when search:read exists.
	key := testJSONObject(t, c.request("POST", "/api/v1/keys", map[string]any{"name": "시점 검색", "workspace_id": wid, "scopes": []string{"document:read"}}, 201))
	c.token = str(key, "token")
	get("2026-06-30")
	write(1, 2, periods, 403)
	out := testJSONObject(t, c.request("POST", "/api/v1/mcp", map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{"name": "search_knowledge_at_date", "arguments": map[string]any{"date": "2026-06-30", "query": "K8s"}}}, 200))
	if boolean(out["result"].(map[string]any), "isError") {
		t.Fatal(out)
	}
	c.token = ""
	other := newID()
	if _, err := s.DB.Exec(ctx, `WITH u AS(INSERT INTO users(id,email,name,password_hash,role) VALUES($1,'validity-owner@example.test','개인 소유자','','editor') RETURNING id), m AS(INSERT INTO workspace_members(workspace_id,user_id,role) SELECT $2,id,'editor' FROM u) UPDATE documents SET owner_id=$1,visibility='private' WHERE id=$3`, other, wid, did); err != nil {
		t.Fatal(err)
	}
	result = search("2026-06-30", "")
	encoded, _ := json.Marshal(result)
	if strings.Contains(string(encoded), "VALIDITY_SENTINEL") || strings.Contains(string(encoded), did) || len(result["results"].([]any)) != 0 {
		t.Fatal("historical ACL leak", result)
	}
	c.request("GET", "/api/v1/documents/"+did+"/validity", nil, 404)
	c.request("GET", "/api/v1/documents/"+did+"/valid-at?date=2026-06-30", nil, 404)
}
