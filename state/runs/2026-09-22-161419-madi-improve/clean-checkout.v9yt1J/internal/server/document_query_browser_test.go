package server

import (
	"context"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestBrowserDocumentQuery(t *testing.T) {
	if os.Getenv("MADI_BROWSER_DOCUMENT_QUERY") != "1" {
		t.Skip("set MADI_BROWSER_DOCUMENT_QUERY=1 after web build")
	}
	s, c, _, _, wid := jobTestFixture(t)
	u := testJSONObject(t, c.request("POST", "/api/v1/admin/users", map[string]any{"email": "browser-query@example.test", "name": "지식 조회자", "role": "editor", "password": "Query-Browser-Password-2026!"}, 200))
	c.request("PUT", "/api/v1/workspaces/"+wid+"/members", map[string]any{"email": u["email"], "role": "editor"}, 200)
	public := documentQueryCreate(t, c, wid, "운영 점검 가이드", "---\n담당일: 2026-09-09\n---\n# 운영 점검\n\n담당자가 점검 결과를 기록합니다.\n- [ ] 점검 상태 확인", "workspace")
	documentQueryCreate(t, c, wid, "운영 PRIVATE_BROWSER_SENTINEL", "개인 원문", "private")
	parent := documentQueryCreate(t, c, wid, "운영 현황 조회", "# 운영 현황 조회\n\n저장된 선언형 정의를 직접 실행해 현재 권한 안의 표를 확인하세요.\n\n```madi-query\n"+documentQueryExample+"\n```\n", "workspace")
	builder := documentQueryCreate(t, c, wid, "조회 정의 편집", "# 조회 정의\n\n여기에 조회 정의를 넣습니다.\n", "workspace")
	app, e := New(context.Background(), s.DB, s.EncryptionKey, "0.1.0", "admin@example.test", "Integration-Test-Password-2026!", os.DirFS("../../web/dist"))
	if e != nil {
		t.Fatal(e)
	}
	server := httptest.NewServer(app)
	defer server.Close()
	defer app.CloseCollaboration()
	ctx, cancel := context.WithTimeout(t.Context(), 180*time.Second)
	defer cancel()
	out, _ := filepath.Abs("../../test-results/document-query")
	command := exec.CommandContext(ctx, "node", "../../tests/document-query.mjs")
	command.Env = append(os.Environ(), "MADI_BASE_URL="+server.URL, "MADI_WORKSPACE_ID="+wid, "MADI_DOCUMENT_ID="+parent, "MADI_SOURCE_ID="+public, "MADI_BUILDER_ID="+builder, "MADI_SCREENSHOT_DIR="+out)
	raw, e := command.CombinedOutput()
	if e != nil {
		t.Fatalf("document query browser: %v\n%s", e, raw)
	}
	t.Log(string(raw))
}
