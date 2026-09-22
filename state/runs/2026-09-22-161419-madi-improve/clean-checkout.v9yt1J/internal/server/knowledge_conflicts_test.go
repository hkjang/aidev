package server

import (
	"bytes"
	"io"
	"net/http"
	"sort"
	"strings"
	"testing"
)

func TestPostgresKnowledgeConflictsPrivateReviewCASAndProtection(t *testing.T) {
	s, owner, ctx, _, wid := jobTestFixture(t)
	owner.request("POST", "/api/v1/admin/users", map[string]any{"email": "conflict-reader@example.test", "name": "비교 열람자", "role": "viewer", "password": "Conflict-Test-Password-2026!"}, 200)
	owner.request("PUT", "/api/v1/workspaces/"+wid+"/members", map[string]any{"email": "conflict-reader@example.test", "role": "viewer"}, 200)
	reader := newIntegrationTestClient(t, owner.base)
	reader.request("POST", "/api/v1/auth/login", map[string]any{"email": "conflict-reader@example.test", "password": "Conflict-Test-Password-2026!"}, 200)
	docs := []map[string]any{}
	for _, n := range []string{"30", "90"} {
		docs = append(docs, testJSONObject(t, owner.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "정책 " + n, "markdown": "# 기록 보존 정책\n\n검증 기록의 기본 보존 기간은 " + n + "일입니다.\n", "visibility": "workspace"}, 200)))
	}
	input := map[string]any{"workspace_id": wid, "documents": []any{map[string]any{"id": docs[0]["id"], "version": 1}, map[string]any{"id": docs[1]["id"], "version": 1}}, "consent": true}
	report := testJSONObject(t, reader.request("POST", "/api/v1/knowledge/conflicts", input, 201))
	id := str(report, "id")
	if number(report, "candidates", 0) != 1 {
		t.Fatal(report)
	}
	owner.request("GET", "/api/v1/knowledge/conflicts/"+id, nil, 404)
	data := testJSONObject(t, reader.request("GET", "/api/v1/knowledge/conflicts/"+id, nil, 200))
	candidate := data["candidates"].([]any)[0].(map[string]any)
	cid := str(candidate, "id")
	var cipher string
	if e := s.DB.QueryRow(ctx, `SELECT ciphertext FROM knowledge_conflict_candidates WHERE id=$1`, cid).Scan(&cipher); e != nil || strings.Contains(cipher, "기본 보존") {
		t.Fatal("plaintext snapshot", e)
	}
	for _, side := range []string{"left", "right"} {
		part := candidate[side].(map[string]any)
		if len(str(part, "document_hash")) != 64 || digest(str(part, "text")) != str(part, "excerpt_hash") || number(part, "line", 0) != 3 {
			t.Fatal("exact source contract", part)
		}
	}
	review := map[string]any{"revision": 1, "decision": "conflict", "note": "두 정책의 적용 대상이 같은지 담당자가 확인 필요", "consent": true}
	reader.request("POST", "/api/v1/knowledge/conflict-candidates/"+cid+"/reviews", review, 200)
	reader.request("POST", "/api/v1/knowledge/conflict-candidates/"+cid+"/reviews", review, 409)
	detail := testJSONObject(t, reader.request("GET", "/api/v1/knowledge/conflict-candidates/"+cid, nil, 200))
	if len(detail["history"].([]any)) != 1 || number(detail, "revision", 0) != 2 {
		t.Fatal(detail)
	}
	for _, doc := range docs {
		current := testJSONObject(t, owner.request("GET", "/api/v1/documents/"+str(doc, "id"), nil, 200))
		if number(current, "version", 0) != 1 || current["markdown"] != doc["markdown"] {
			t.Fatal("judgment wrote source", current)
		}
	}
	owner.request("PUT", "/api/v1/documents/"+str(docs[0], "id"), map[string]any{"version": 1, "markdown": "# 기록 보존 정책\n\n검증 기록의 기본 보존 기간은 60일입니다.\n"}, 200)
	detail = testJSONObject(t, reader.request("GET", "/api/v1/knowledge/conflict-candidates/"+cid, nil, 200))
	if !boolean(detail, "stale") {
		t.Fatal("changed source considered current", detail)
	}
	review["revision"] = 2
	reader.request("POST", "/api/v1/knowledge/conflict-candidates/"+cid+"/reviews", review, 409)
	owner.request("PUT", "/api/v1/documents/"+str(docs[0], "id"), map[string]any{"version": 2, "visibility": "private"}, 200)
	raw := reader.request("GET", "/api/v1/knowledge/conflicts/"+id, nil, 200)
	if strings.Contains(string(raw), "기본 보존") || strings.Contains(string(raw), str(docs[0], "id")) {
		t.Fatal("revoked content leaked", string(raw))
	}
	reader.request("GET", "/api/v1/knowledge/conflict-candidates/"+cid, nil, 404)
	// An existing slot may be marked unavailable, but no new private title or ID appears.
	if !strings.Contains(string(raw), `"unavailable":true`) {
		t.Fatal(string(raw))
	}
	input["documents"] = []any{map[string]any{"id": docs[0]["id"], "version": 3}, map[string]any{"id": docs[1]["id"], "version": 1}}
	reader.request("POST", "/api/v1/knowledge/conflicts", input, 404)
	reader.request("DELETE", "/api/v1/knowledge/conflicts/"+id, nil, 200)
	// Current policy blocks recording even if the underlying old Markdown predates it.
	if _, e := s.DB.Exec(ctx, `UPDATE protection_settings SET data=data||'{"enabled":true,"mode":"mask","detectors":[],"custom_terms":["기본 보존"]}'::jsonb,revision=revision+1 WHERE id=1`); e != nil {
		t.Fatal(e)
	}
	owner.request("POST", "/api/v1/knowledge/conflicts", input, 422)
}

func TestPostgresKnowledgeConflictsConcurrentOppositeMerges(t *testing.T) {
	_, owner, _, _, wid := jobTestFixture(t)
	docs := []map[string]any{}
	refs := []any{}
	for _, value := range []string{"30", "90"} {
		doc := testJSONObject(t, owner.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "동시 정합성", "markdown": "# 같은 보관 기준\n\n서비스 이력의 보관 기한은 " + value + "일입니다.\n"}, 200))
		docs = append(docs, doc)
		refs = append(refs, map[string]any{"id": doc["id"], "version": 1})
	}
	report := testJSONObject(t, owner.request("POST", "/api/v1/knowledge/conflicts", map[string]any{"workspace_id": wid, "documents": refs, "consent": true}, 201))
	view := testJSONObject(t, owner.request("GET", "/api/v1/knowledge/conflicts/"+str(report, "id"), nil, 200))
	cid := str(view["candidates"].([]any)[0].(map[string]any), "id")
	proposals := []string{}
	for _, doc := range docs {
		p := testJSONObject(t, owner.request("POST", "/api/v1/documents/"+str(doc, "id")+"/proposals", map[string]any{"base_version": 1, "markdown": "# 같은 보관 기준\n\n서비스 이력의 보관 기한은 60일입니다.\n", "reason": "두 원문을 대조한 변경 제안", "provenance": "human", "consent": true, "conflict_id": cid, "conflict_revision": 1, "conflict_consent": true}, 201))
		proposals = append(proposals, str(p, "id"))
	}
	type result struct {
		status int
		body   string
		err    error
	}
	results := make(chan result, 2)
	start := make(chan struct{})
	for _, id := range proposals {
		go func(id string) {
			<-start
			req, e := http.NewRequest("POST", owner.base+"/api/v1/knowledge/proposals/"+id+"/merge", bytes.NewReader(jsonValue(map[string]any{"revision": 1, "consent": true})))
			if e != nil {
				results <- result{err: e}
				return
			}
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Madi-Request", "1")
			response, e := owner.client.Do(req)
			if e != nil {
				results <- result{err: e}
				return
			}
			defer response.Body.Close()
			raw, e := io.ReadAll(response.Body)
			results <- result{response.StatusCode, string(raw), e}
		}(id)
	}
	close(start)
	statuses := []int{}
	for range 2 {
		r := <-results
		if r.err != nil {
			t.Fatal(r.err)
		}
		if r.status != 200 && r.status != 409 {
			t.Fatalf("opposite merge deadlock/ACL failure: %d %s", r.status, r.body)
		}
		statuses = append(statuses, r.status)
	}
	sort.Ints(statuses)
	if statuses[0] != 200 || statuses[1] != 409 {
		t.Fatal("both source-stale plans merged", statuses)
	}
	total := 0
	for _, doc := range docs {
		got := testJSONObject(t, owner.request("GET", "/api/v1/documents/"+str(doc, "id"), nil, 200))
		total += number(got, "version", 0)
	}
	if total != 3 {
		t.Fatal("more than one canonical mutation", total)
	}
}

func TestPostgresKnowledgeConflictsMCPReadIsolationAndSourceCAS(t *testing.T) {
	_, owner, _, _, wid := jobTestFixture(t)
	refs := []any{}
	for _, value := range []string{"허용", "금지"} {
		doc := testJSONObject(t, owner.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "접근 정책", "markdown": "# 접근 정책\n\n외부 단말기의 서비스 접근은 " + value + "입니다.\n", "visibility": "private"}, 200))
		refs = append(refs, map[string]any{"id": doc["id"], "version": 1})
	}
	in := map[string]any{"workspace_id": wid, "documents": refs, "consent": false}
	owner.request("POST", "/api/v1/knowledge/conflicts", in, 400)
	in["consent"] = true
	refs[0].(map[string]any)["version"] = 2
	owner.request("POST", "/api/v1/knowledge/conflicts", in, 409)
	refs[0].(map[string]any)["version"] = 1
	report := testJSONObject(t, owner.request("POST", "/api/v1/knowledge/conflicts", in, 201))
	id := str(report, "id")
	key := testJSONObject(t, owner.request("POST", "/api/v1/keys", map[string]any{"name": "비교 읽기", "workspace_id": wid, "scopes": []string{"document:read"}}, 201))
	owner.token = str(key, "token")
	owner.request("GET", "/api/v1/knowledge/conflicts/"+id, nil, 200)
	for _, call := range []map[string]any{{"name": "list_knowledge_conflicts", "arguments": map[string]any{"workspace_id": wid}}, {"name": "get_knowledge_conflict", "arguments": map[string]any{"run_id": id}}} {
		response := testJSONObject(t, owner.request("POST", "/api/v1/mcp", map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": call}, 200))
		if response["error"] != nil || boolean(response["result"].(map[string]any), "isError") {
			t.Fatal(response)
		}
	}
	owner.request("POST", "/api/v1/knowledge/conflicts", in, 403)
	list := testJSONObject(t, owner.request("GET", "/api/v1/knowledge/conflicts?workspace_id="+wid, nil, 200))
	if raw := string(jsonValue(list)); strings.Contains(raw, "외부 단말기") {
		t.Fatal("list plaintext", raw)
	}
	owner.token = ""
	other := testJSONObject(t, owner.request("POST", "/api/v1/workspaces", map[string]any{"name": "다른 비교 영역"}, 200))
	in["workspace_id"] = other["id"]
	owner.request("POST", "/api/v1/knowledge/conflicts", in, 404)
}

func TestPostgresKnowledgeConflictsProposalOriginACLStaleAndAtomicity(t *testing.T) {
	s, owner, ctx, _, wid := jobTestFixture(t)
	owner.request("POST", "/api/v1/admin/users", map[string]any{"email": "conflict-editor@example.test", "name": "변경안 열람자", "role": "editor", "password": "Conflict-Editor-Password-2026!"}, 200)
	owner.request("PUT", "/api/v1/workspaces/"+wid+"/members", map[string]any{"email": "conflict-editor@example.test", "role": "editor"}, 200)
	editor := newIntegrationTestClient(t, owner.base)
	editor.request("POST", "/api/v1/auth/login", map[string]any{"email": "conflict-editor@example.test", "password": "Conflict-Editor-Password-2026!"}, 200)
	docs := []map[string]any{}
	for _, n := range []string{"30", "90"} {
		docs = append(docs, testJSONObject(t, owner.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "공통 기록 정책 " + n, "markdown": "# 보관 정책\n\n검증 원본의 보관 기간은 " + n + "일입니다.\n", "visibility": "workspace"}, 200)))
	}
	report := testJSONObject(t, owner.request("POST", "/api/v1/knowledge/conflicts", map[string]any{"workspace_id": wid, "consent": true, "documents": []any{map[string]any{"id": docs[0]["id"], "version": 1}, map[string]any{"id": docs[1]["id"], "version": 1}}}, 201))
	view := testJSONObject(t, owner.request("GET", "/api/v1/knowledge/conflicts/"+str(report, "id"), nil, 200))
	candidate := view["candidates"].([]any)[0].(map[string]any)
	cid := str(candidate, "id")
	target := str(docs[0], "id")
	body := map[string]any{"base_version": 1, "markdown": "# 보관 정책\n\n검증 원본의 보관 기간은 60일입니다.\n", "reason": "담당자가 적용 대상을 확인한 변경 초안", "provenance": "human", "consent": true, "conflict_id": cid, "conflict_revision": 1, "conflict_consent": true}
	// A different report reader cannot use the personal candidate as an origin.
	editor.request("POST", "/api/v1/documents/"+target+"/proposals", body, 404)
	body["conflict_consent"] = false
	owner.request("POST", "/api/v1/documents/"+target+"/proposals", body, 400)
	body["conflict_consent"] = true
	if _, e := s.DB.Exec(ctx, `CREATE FUNCTION madi_conflict_origin_fail() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'rollback test'; END $$; CREATE TRIGGER madi_conflict_origin_fail BEFORE INSERT ON knowledge_proposal_conflict_origins FOR EACH ROW EXECUTE FUNCTION madi_conflict_origin_fail()`); e != nil {
		t.Fatal(e)
	}
	owner.request("POST", "/api/v1/documents/"+target+"/proposals", body, 500)
	var count int
	if e := s.DB.QueryRow(ctx, `SELECT count(*) FROM knowledge_proposals WHERE document_id=$1`, target).Scan(&count); e != nil || count != 0 {
		t.Fatal("proposal escaped receipt transaction", count, e)
	}
	if _, e := s.DB.Exec(ctx, `DROP TRIGGER madi_conflict_origin_fail ON knowledge_proposal_conflict_origins`); e != nil {
		t.Fatal(e)
	}
	proposal := testJSONObject(t, owner.request("POST", "/api/v1/documents/"+target+"/proposals", body, 201))
	pid := str(proposal, "id")
	shared := testJSONObject(t, editor.request("GET", "/api/v1/knowledge/proposals/"+pid, nil, 200))
	origin := shared["origin"].(map[string]any)
	if origin["run_id"] != nil || origin["candidate_id"] != nil || boolean(shared, "stale") {
		t.Fatal("private report ownership leaked", origin)
	}
	owner.request("DELETE", "/api/v1/knowledge/conflicts/"+str(report, "id"), nil, 409)
	// Revoking only the OTHER document removes the target proposal too.
	other := str(docs[1], "id")
	owner.request("PUT", "/api/v1/documents/"+other, map[string]any{"version": 1, "visibility": "private"}, 200)
	editor.request("GET", "/api/v1/knowledge/proposals/"+pid, nil, 404)
	editor.request("GET", "/api/v1/knowledge/proposals/"+pid+"/check", nil, 404)
	if raw := editor.request("GET", "/api/v1/knowledge/proposals?workspace_id="+wid, nil, 200); strings.Contains(string(raw), pid) {
		t.Fatal("origin denied proposal listed", string(raw))
	}
	owner.request("POST", "/api/v1/knowledge/proposals/"+pid+"/merge", map[string]any{"revision": 1, "consent": true}, 409)
	current := testJSONObject(t, owner.request("GET", "/api/v1/documents/"+target, nil, 200))
	if current["markdown"] != docs[0]["markdown"] || number(current, "version", 0) != 1 {
		t.Fatal("stale origin merged", current)
	}
	// A fresh independent report/proposal merges through the canonical command.
	if _, e := s.DB.Exec(ctx, `UPDATE knowledge_conflict_runs SET created_at=now()-interval '5 seconds' WHERE owner_id=$1`, current["owner_id"]); e != nil {
		t.Fatal(e)
	}
	report = testJSONObject(t, owner.request("POST", "/api/v1/knowledge/conflicts", map[string]any{"workspace_id": wid, "consent": true, "documents": []any{map[string]any{"id": target, "version": 1}, map[string]any{"id": other, "version": 2}}}, 201))
	view = testJSONObject(t, owner.request("GET", "/api/v1/knowledge/conflicts/"+str(report, "id"), nil, 200))
	candidate = view["candidates"].([]any)[0].(map[string]any)
	body["conflict_id"] = candidate["id"]
	body["markdown"] = "# 보관 정책\n\n검증 원본의 보관 기간은 45일입니다.\n"
	made := testJSONObject(t, owner.request("POST", "/api/v1/documents/"+target+"/proposals", body, 201))
	merged := testJSONObject(t, owner.request("POST", "/api/v1/knowledge/proposals/"+str(made, "id")+"/merge", map[string]any{"revision": 1, "consent": true}, 200))
	if number(merged, "version", 0) != 2 {
		t.Fatal(merged)
	}
	owner.request("POST", "/api/v1/knowledge/proposals/"+str(made, "id")+"/merge", map[string]any{"revision": 1, "consent": true}, 200)
}
