package server

import (
	"bytes"
	"testing"
	"time"
)

func TestPostgresKnowledgeQuestionsPublicationFreshnessCASAndPrivateACL(t *testing.T) {
	s, owner, reader, wid, _ := collaborationTestSetup(t)
	ctx := t.Context()
	uid := str(testJSONObject(t, owner.request("GET", "/api/v1/auth/me", nil, 200)), "id")
	doc := testJSONObject(t, owner.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "공식 운영 답변", "markdown": "서비스를 먼저 확인합니다", "visibility": "workspace"}, 200))
	did := str(doc, "id")
	source := testJSONObject(t, owner.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "운영 근거", "markdown": "QUESTION_SOURCE_PRIVATE_SENTINEL 기준 2개", "visibility": "workspace"}, 200))
	sid := str(source, "id")
	input := map[string]any{"document_id": did, "version": 1, "owner_id": uid, "question": "QUESTION_TEXT_SECRET 어떻게 확인하나요?", "reason": "담당자의 검증된 답변 관리", "review_due": time.Now().UTC().AddDate(0, 0, 7).Format("2006-01-02"), "sources": []questionRef{{ID: sid, Version: 1}}, "consent": true}
	made := testJSONObject(t, owner.request("POST", "/api/v1/knowledge/questions", input, 201))
	id := str(made, "id")
	path := "/api/v1/knowledge/questions/" + id
	owner.request("POST", "/api/v1/knowledge/questions", input, 409)
	var cipher string
	if e := s.DB.QueryRow(ctx, `SELECT ciphertext FROM knowledge_questions WHERE id=$1`, id).Scan(&cipher); e != nil || bytes.Contains([]byte(cipher), []byte("QUESTION_TEXT_SECRET")) {
		t.Fatal("question plaintext", e)
	}
	view := testJSONObject(t, reader.request("GET", path, nil, 200))
	if boolean(view, "official") || !boolean(view, "fresh") || boolean(view, "published_current") {
		t.Fatal(view)
	}
	decision := map[string]any{"revision": 1, "state": "confirmed", "note": "근거와 답변을 확인했습니다", "consent": true}
	owner.request("POST", path+"/decision", decision, 409)
	owner.request("PUT", "/api/v1/documents/"+did, map[string]any{"version": 1, "status": "published"}, 200)
	// Publication changes the canonical revision. It cannot silently refresh a
	// registered answer's attestation.
	owner.request("POST", path+"/decision", decision, 409)
	input["revision"], input["version"] = 1, 2
	owner.request("PUT", path, input, 200)
	owner.request("PUT", path, input, 409)
	decision["revision"] = 2
	reader.request("POST", path+"/decision", decision, 404)
	owner.request("POST", path+"/decision", decision, 200)
	view = testJSONObject(t, reader.request("GET", path, nil, 200))
	if !boolean(view, "official") || number(view, "revision", 0) != 3 || len(view["events"].([]any)) != 2 {
		t.Fatal(view)
	}
	owner.request("PUT", "/api/v1/documents/"+sid, map[string]any{"version": 1, "markdown": "기준 3개"}, 200)
	view = testJSONObject(t, reader.request("GET", path, nil, 200))
	if boolean(view, "official") || boolean(view, "fresh") || str(view, "state") != "confirmed" {
		t.Fatal("history/latest collapsed", view)
	}
	input["revision"] = 3
	input["sources"] = []questionRef{{ID: sid, Version: 2}}
	owner.request("PUT", path, input, 200)
	decision["revision"] = 4
	owner.request("POST", path+"/decision", decision, 200)
	owner.request("PUT", "/api/v1/documents/"+sid, map[string]any{"version": 2, "visibility": "private"}, 200)
	reader.request("GET", path, nil, 404)
	if out := reader.request("GET", "/api/v1/knowledge/questions?workspace_id="+wid, nil, 200); bytes.Contains(out, []byte(id)) || bytes.Contains(out, []byte("QUESTION_TEXT_SECRET")) {
		t.Fatal("hidden source/question existence leaked", string(out))
	}
	// Enabling a publication approval policy invalidates an earlier unmanaged
	// official label without changing the source answer or silently approving it.
	owner.request("PUT", "/api/v1/admin/settings", map[string]any{"approval_enabled": true}, 200)
	view = testJSONObject(t, owner.request("GET", path, nil, 200))
	if boolean(view, "official") || boolean(view, "published_current") || !boolean(view, "approval_enabled") {
		t.Fatal(view)
	}
	if _, e := s.DB.Exec(ctx, `DELETE FROM documents WHERE id=$1`, sid); e != nil {
		t.Fatal(e)
	}
	var n int
	if e := s.DB.QueryRow(ctx, `SELECT count(*) FROM knowledge_questions WHERE id=$1`, id).Scan(&n); e != nil || n != 0 {
		t.Fatal("deleted source retained copied question", n, e)
	}
}

func TestPostgresKnowledgeQuestionsPersonalPromotionIsExplicitPrivateAndBounded(t *testing.T) {
	s, owner, other, wid, _ := collaborationTestSetup(t)
	ctx := t.Context()
	uid := str(testJSONObject(t, owner.request("GET", "/api/v1/auth/me", nil, 200)), "id")
	source := testJSONObject(t, owner.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "개인 정리 근거", "markdown": "현재 확인 기준", "visibility": "workspace"}, 200))
	sid := str(source, "id")
	cid, mid := newID(), newID()
	_, e := s.DB.Exec(ctx, `INSERT INTO ai_conversations(id,workspace_id,owner_id,title) VALUES($1,$2,$3,'개인 대화')`, cid, wid, uid)
	if e == nil {
		_, e = s.DB.Exec(ctx, `INSERT INTO ai_messages(id,conversation_id,ordinal,question,answer,action,sources,provider_fingerprint,model) VALUES($1,$2,1,'PERSONAL_QUESTION_SENTINEL 질문','개인 AI 답변','ask',$3,'fixture','fixture')`, mid, cid, jsonValue([]aiSource{{ID: sid, Version: 1}}))
	}
	if e != nil {
		t.Fatal(e)
	}
	previewPath := "/api/v1/ai/messages/" + mid + "/question-preview"
	preview := testJSONObject(t, owner.request("GET", previewPath, nil, 200))
	if str(preview, "question") != "PERSONAL_QUESTION_SENTINEL 질문" {
		t.Fatal(preview)
	}
	other.request("GET", previewPath, nil, 404)
	draftInput := map[string]any{"preview_hash": preview["preview_hash"], "consent": true}
	draft := testJSONObject(t, owner.request("POST", "/api/v1/ai/messages/"+mid+"/question-draft", draftInput, 200))
	if str(draft, "visibility") != "private" || str(draft, "markdown") != str(preview, "answer") {
		t.Fatal("copy was shared or changed", draft)
	}
	if replay := testJSONObject(t, owner.request("POST", "/api/v1/ai/messages/"+mid+"/question-draft", draftInput, 200)); str(replay, "id") != str(draft, "id") {
		t.Fatal("replay duplicated draft", replay)
	}
	var n int
	s.DB.QueryRow(ctx, `SELECT count(*) FROM knowledge_questions`).Scan(&n)
	if n != 0 {
		t.Fatal("preview implicitly promoted a question")
	}
	doc := testJSONObject(t, owner.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "정리한 답변", "markdown": preview["answer"], "visibility": "workspace"}, 200))
	did := str(doc, "id")
	in := map[string]any{"document_id": did, "version": 1, "owner_id": uid, "question": preview["question"], "reason": "이 질문과 답변만 명시적으로 정리", "review_due": "2099-01-01", "sources": preview["sources"], "message_id": mid, "consent": false}
	owner.request("POST", "/api/v1/knowledge/questions", in, 400)
	in["consent"] = true
	owner.request("POST", "/api/v1/knowledge/questions", in, 409)
	owner.request("PUT", "/api/v1/documents/"+did, map[string]any{"version": 1, "visibility": "private"}, 200)
	in["version"] = 2
	in["question"] = "다른 개인 질문으로 위조"
	owner.request("POST", "/api/v1/knowledge/questions", in, 409)
	in["question"] = preview["question"]
	made := testJSONObject(t, owner.request("POST", "/api/v1/knowledge/questions", in, 201))
	id := str(made, "id")
	view := testJSONObject(t, owner.request("GET", "/api/v1/knowledge/questions/"+id, nil, 200))
	if str(view, "origin") != "personal_ai_question" || view["origin_hash"] != nil || bytes.Contains(jsonValue(view), []byte(mid)) {
		t.Fatal("personal provenance private ID leaked", view)
	}
	other.request("GET", "/api/v1/knowledge/questions/"+id, nil, 404)
	owner.request("PUT", "/api/v1/documents/"+sid, map[string]any{"version": 1, "markdown": "수정된 기준"}, 200)
	owner.request("GET", previewPath, nil, 404)
	owner.request("POST", "/api/v1/ai/messages/"+mid+"/question-draft", draftInput, 409)
	key := testJSONObject(t, owner.request("POST", "/api/v1/keys", map[string]any{"name": "조회 전용 질문 연동", "workspace_id": wid, "scopes": []string{"document:read", "document:write"}}, 201))
	owner.token = str(key, "token")
	owner.request("GET", "/api/v1/knowledge/questions/"+id, nil, 200)
	owner.request("GET", previewPath, nil, 403)
	owner.request("POST", "/api/v1/knowledge/questions", in, 403)
	owner.request("POST", "/api/v1/knowledge/questions/"+id+"/decision", map[string]any{"revision": 1, "state": "confirmed", "note": "agent cannot attest", "consent": true}, 403)
}
