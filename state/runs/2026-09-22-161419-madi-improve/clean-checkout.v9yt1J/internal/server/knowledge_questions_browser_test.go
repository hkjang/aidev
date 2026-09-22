package server

import (
	"context"
	"net/http/httptest"
	"os"
	"os/exec"
	"testing"
	"time"
)

func TestBrowserKnowledgeQuestions(t *testing.T) {
	if os.Getenv("MADI_BROWSER_KNOWLEDGE_QUESTIONS") != "1" {
		t.Skip("set MADI_BROWSER_KNOWLEDGE_QUESTIONS=1 after web build")
	}
	seed, _ := integrationTestServer(t)
	app, e := New(t.Context(), seed.DB, seed.EncryptionKey, "0.1.0", "admin@example.test", "Integration-Test-Password-2026!", os.DirFS("../../web/dist"))
	if e != nil {
		t.Fatal(e)
	}
	endpoint := httptest.NewServer(app)
	defer endpoint.Close()
	defer app.CloseCollaboration()
	c := newIntegrationTestClient(t, endpoint.URL)
	c.request("POST", "/api/v1/auth/login", map[string]any{"email": "admin@example.test", "password": "Integration-Test-Password-2026!"}, 200)
	wid := str(testJSONObject(t, c.request("POST", "/api/v1/workspaces", map[string]any{"name": "질문과 검증된 답변"}, 200)), "id")
	uid := str(testJSONObject(t, c.request("GET", "/api/v1/auth/me", nil, 200)), "id")
	sid := str(testJSONObject(t, c.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "GPU 운영 확인 기준", "markdown": "# GPU 운영\n\n작업 전 GPU 사용량을 확인합니다.", "visibility": "workspace"}, 200)), "id")
	cid, mid := newID(), newID()
	_, e = app.DB.Exec(t.Context(), `INSERT INTO ai_conversations(id,workspace_id,owner_id,title) VALUES($1,$2,$3,'개인 GPU 질문')`, cid, wid, uid)
	if e != nil {
		t.Fatal(e)
	}
	_, e = app.DB.Exec(t.Context(), `INSERT INTO ai_messages(id,conversation_id,ordinal,question,answer,action,sources,provider_fingerprint,model) VALUES($1,$2,1,'GPU 작업 전에 무엇을 확인하나요?','# 확인 절차

GPU 사용량을 확인하고 변경 기록을 남깁니다.','ask',$3,'isolated-fixture','격리 시험 답변')`, mid, cid, jsonValue([]aiSource{{ID: sid, Version: 1}}))
	if e != nil {
		t.Fatal(e)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 180*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "node", "../../tests/knowledge-questions.mjs")
	cmd.Env = append(os.Environ(), "MADI_BASE_URL="+endpoint.URL, "MADI_ADMIN_PASSWORD=Integration-Test-Password-2026!", "MADI_QUESTION_WORKSPACE="+wid, "MADI_QUESTION_SOURCE="+sid, "MADI_QUESTION_MESSAGE="+mid, "MADI_QUESTION_CONVERSATION="+cid)
	out, e := cmd.CombinedOutput()
	if e != nil {
		t.Fatalf("questions browser: %v\n%s", e, out)
	}
	t.Log(string(out))
}
