package server

import (
	"context"
	"net/http/httptest"
	"os"
	"os/exec"
	"testing"
	"time"
)

func TestBrowserRAGIndexConsent(t *testing.T) {
	if os.Getenv("MADI_BROWSER_RAG") != "1" {
		t.Skip("set MADI_BROWSER_RAG=1 after web build")
	}
	s, c, wid, _ := ragTestSetup(t)
	provider, _, _ := ragTestProvider(t, nil)
	ragTestConfigure(t, c, provider.URL)
	doc := str(testJSONObject(t, c.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "GPU 운영 지식 · 검색 AI", "markdown": "# GPU 운영 지식\n\n사내 GPU 운영 절차를 문서로 연결합니다.\n\n## 안전한 색인\n\n비공개 문서는 작성자 동의와 현재 권한으로만 색인합니다.\n\n- 명시적 공급자 동의\n- 원문 범위와 버전 검증\n- 키 회수 시 후속 처리 중단\n", "visibility": "private"}, 200)), "id")
	app, e := New(context.Background(), s.DB, s.EncryptionKey, "rag-test", "admin@example.test", "Integration-Test-Password-2026!", os.DirFS("../../web/dist"))
	if e != nil {
		t.Fatal(e)
	}
	server := httptest.NewServer(app)
	defer server.Close()
	ctx, cancel := context.WithTimeout(t.Context(), 150*time.Second)
	defer cancel()
	app.StartJobs(ctx)
	app.StartRAGIndex(ctx)
	command := exec.CommandContext(ctx, "node", "../../tests/rag-index.mjs")
	command.Env = append(os.Environ(), "MADI_BASE_URL="+server.URL, "MADI_RAG_DOCUMENT="+doc, "MADI_RAG_WORKSPACE="+wid)
	output, e := command.CombinedOutput()
	if e != nil {
		t.Fatalf("RAG browser %v\n%s", e, output)
	}
	t.Log(string(output))
}
