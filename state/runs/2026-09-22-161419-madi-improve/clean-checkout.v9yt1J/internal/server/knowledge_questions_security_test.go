package server

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestPostgresKnowledgeQuestionsCurrentApprovalOwnerRecoveryAndMCP(t *testing.T) {
	s, c, reviewer, wid, _ := collaborationTestSetup(t)
	ctx := t.Context()
	uid := str(testJSONObject(t, c.request("GET", "/api/v1/auth/me", nil, 200)), "id")
	rid := str(testJSONObject(t, reviewer.request("GET", "/api/v1/auth/me", nil, 200)), "id")
	c.request("PUT", "/api/v1/admin/settings", map[string]any{"approval_enabled": true}, 200)
	c.request("POST", "/api/v1/admin/approval/policies", map[string]any{"name": "공식 답변 게시 정책", "workspace_id": wid, "resource_kind": "document", "enabled": true, "stages": []any{map[string]any{"name": "독립 확인", "mode": "all", "gates": []any{map[string]any{"name": "팀 검토", "kind": "user", "id": rid}}}}}, 200)
	source := testJSONObject(t, c.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "공통 운영 근거", "markdown": "확인된 근거", "visibility": "workspace"}, 200))
	doc := testJSONObject(t, c.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "팀의 공식 답변", "markdown": "근거에 따른 절차", "visibility": "workspace"}, 200))
	did, sid := str(doc, "id"), str(source, "id")
	c.request("POST", "/api/v1/documents/"+did+"/approval", map[string]any{"action": "submit"}, 200)
	approval := testJSONObject(t, c.request("GET", "/api/v1/documents/"+did+"/approval", nil, 200))["request"].(map[string]any)
	reviewer.request("POST", "/api/v1/approvals/requests/"+str(approval, "id")+"/decisions", map[string]any{"action": "approve", "request_version": approval["version"]}, 200)
	doc = testJSONObject(t, c.request("GET", "/api/v1/documents/"+did, nil, 200))
	input := map[string]any{"document_id": did, "version": doc["version"], "owner_id": rid, "question": "공식 운영 순서는?", "reason": "담당자에게 연결", "review_due": "2099-01-01", "sources": []questionRef{{ID: sid, Version: 1}}, "consent": true}
	q := testJSONObject(t, c.request("POST", "/api/v1/knowledge/questions", input, 201))
	path := "/api/v1/knowledge/questions/" + str(q, "id")
	view := testJSONObject(t, c.request("GET", path, nil, 200))
	if !boolean(view, "can_manage") || boolean(view, "can_confirm") {
		t.Fatal("document owner must only manage, not attest for assignee", view)
	}
	c.request("POST", path+"/decision", map[string]any{"revision": 1, "state": "confirmed", "note": "소유자가 담당자를 대신함", "consent": true}, 404)
	reviewer.request("POST", path+"/decision", map[string]any{"revision": 1, "state": "confirmed", "note": "현재 승인과 근거를 담당자로 확인", "consent": true}, 200)
	view = testJSONObject(t, c.request("GET", path, nil, 200))
	if !boolean(view, "official") {
		t.Fatal("current independent publication approval not recognized", view)
	}
	// An absent assignee cannot make the question impossible to maintain. The
	// canonical document owner can reassign, without a service-admin ACL bypass.
	if _, e := s.DB.Exec(ctx, `UPDATE users SET disabled=true WHERE id=$1`, rid); e != nil {
		t.Fatal(e)
	}
	view = testJSONObject(t, c.request("GET", path, nil, 200))
	if boolean(view, "owner_valid") || boolean(view, "official") || !boolean(view, "can_manage") {
		t.Fatal(view)
	}
	input["revision"], input["owner_id"] = 2, uid
	c.request("PUT", path, input, 200)
	view = testJSONObject(t, c.request("GET", path, nil, 200))
	if !boolean(view, "can_confirm") || !boolean(view, "owner_valid") || str(view, "state") != "proposed" {
		t.Fatal(view)
	}
	key := testJSONObject(t, c.request("POST", "/api/v1/keys", map[string]any{"name": "관리 질문 읽기", "workspace_id": wid, "scopes": []string{"document:read"}}, 201))
	c.token = str(key, "token")
	for _, tool := range []struct {
		name string
		args map[string]any
	}{{"get_managed_question", map[string]any{"question_id": q["id"]}}, {"list_managed_questions", map[string]any{}}} {
		response := testJSONObject(t, c.request("POST", "/api/v1/mcp", map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{"name": tool.name, "arguments": tool.args}}, 200))
		result, ok := response["result"].(map[string]any)
		if !ok || boolean(result, "isError") || !strings.Contains(string(jsonValue(result)), "공식 운영 순서는") {
			t.Fatal(response)
		}
	}
	c.token = ""
	if _, e := s.DB.Exec(ctx, `UPDATE knowledge_questions SET review_due='2000-01-01' WHERE id=$1`, q["id"]); e != nil {
		t.Fatal(e)
	}
	c.request("POST", path+"/decision", map[string]any{"revision": 3, "state": "confirmed", "note": "만료된 검토일", "consent": true}, 409)
	setting := defaultProtectionSettings()
	setting["enabled"], setting["mode"], setting["custom_terms"] = true, "mask", []string{"공식 운영 순서는"}
	if _, e := s.DB.Exec(ctx, `UPDATE protection_settings SET data=$1`, jsonValue(setting)); e != nil {
		t.Fatal(e)
	}
	c.request("GET", path, nil, 409)
	list := c.request("GET", "/api/v1/knowledge/questions?workspace_id="+wid, nil, 200)
	if strings.Contains(string(list), str(q, "id")) {
		t.Fatal("PII-protected question title appeared in list")
	}
}

func TestPostgresKnowledgeQuestionDraftReceiptFailureRollsBackCanonicalCreate(t *testing.T) {
	s, c, ctx, p, wid := jobTestFixture(t)
	source := testJSONObject(t, c.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "원자 초안 근거", "markdown": "근거"}, 200))
	cid, mid := newID(), newID()
	if _, e := s.DB.Exec(ctx, `INSERT INTO ai_conversations(id,workspace_id,owner_id,title) VALUES($1,$2,$3,'원자 시험')`, cid, wid, p.ID); e != nil {
		t.Fatal(e)
	}
	if _, e := s.DB.Exec(ctx, `INSERT INTO ai_messages(id,conversation_id,ordinal,question,answer,action,sources,provider_fingerprint,model) VALUES($1,$2,1,'원자 시험 질문','원자 시험 답변','ask',$3,'fixture','fixture')`, mid, cid, jsonValue([]aiSource{{ID: str(source, "id"), Version: 1}})); e != nil {
		t.Fatal(e)
	}
	preview := testJSONObject(t, c.request("GET", "/api/v1/ai/messages/"+mid+"/question-preview", nil, 200))
	var before int
	if e := s.DB.QueryRow(ctx, `SELECT count(*) FROM documents WHERE workspace_id=$1`, wid).Scan(&before); e != nil {
		t.Fatal(e)
	}
	if _, e := s.DB.Exec(ctx, `CREATE FUNCTION madi_question_receipt_failure() RETURNS trigger LANGUAGE plpgsql AS $$BEGIN RAISE EXCEPTION 'receipt fixture';END$$; CREATE TRIGGER madi_question_receipt_failure BEFORE INSERT ON knowledge_question_draft_receipts FOR EACH ROW EXECUTE FUNCTION madi_question_receipt_failure()`); e != nil {
		t.Fatal(e)
	}
	input := map[string]any{"preview_hash": preview["preview_hash"], "consent": true}
	url := "/api/v1/ai/messages/" + mid + "/question-draft"
	c.request("POST", url, input, 500)
	var after, versions, receipts int
	s.DB.QueryRow(ctx, `SELECT count(*) FROM documents WHERE workspace_id=$1`, wid).Scan(&after)
	s.DB.QueryRow(ctx, `SELECT count(*) FROM document_versions WHERE document_id NOT IN(SELECT id FROM documents)`).Scan(&versions)
	s.DB.QueryRow(ctx, `SELECT count(*) FROM knowledge_question_draft_receipts`).Scan(&receipts)
	if before != after || versions != 0 || receipts != 0 {
		t.Fatal("partial draft persisted", before, after, versions, receipts)
	}
	if _, e := s.DB.Exec(ctx, `DROP TRIGGER madi_question_receipt_failure ON knowledge_question_draft_receipts`); e != nil {
		t.Fatal(e)
	}
	d := testJSONObject(t, c.request("POST", url, input, 200))
	if str(d, "visibility") != "private" {
		t.Fatal(d)
	}
	// Runtime OpenAPI includes the actual request contract and every durable table.
	var spec map[string]any
	if json.Unmarshal(c.request("GET", "/api/v1/openapi.json", nil, 200), &spec) != nil {
		t.Fatal("OpenAPI parse")
	}
	paths := spec["paths"].(map[string]any)
	if paths["/ai/messages/{id}/question-draft"] == nil || paths["/knowledge/questions"] == nil {
		t.Fatal("new route absent from OpenAPI")
	}
}
