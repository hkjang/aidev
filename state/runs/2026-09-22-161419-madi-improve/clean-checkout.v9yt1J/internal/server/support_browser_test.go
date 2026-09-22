package server

import (
	"context"
	"net/http/httptest"
	"os"
	"os/exec"
	"testing"
	"time"
)

func TestBrowserSupportDiagnostics(t *testing.T) {
	if os.Getenv("MADI_BROWSER_SUPPORT") != "1" {
		t.Skip("set MADI_BROWSER_SUPPORT=1 after web build")
	}
	s, c, target, wid, uid, tid := supportTestSetup(t)
	c.request("PUT", "/api/v1/admin/settings", map[string]any{"support_enabled": true, "support_operator_ids": []string{uid}, "support_max_minutes": 15}, 200)
	doc := testJSONObject(t, c.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "공통 열람 운영 문서", "markdown": "# 공통 열람 운영 문서\n\n현재 양쪽에 허용된 문서만 진단합니다.\n\n- **문서 원문 보존**\n- 다운로드 및 외부 전송 없음\n\n![실행하지 않는 외부 이미지](https://example.invalid/support-secret.png)\n\n<script>throw new Error('지원에서 실행하면 안 됩니다')</script>", "visibility": "selected"}, 200))
	did := str(doc, "id")
	c.request("PUT", "/api/v1/documents/"+did+"/shares", map[string]any{"user_id": tid, "permission": "read"}, 200)
	target.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "TARGET_PRIVATE_BROWSER_SECRET", "markdown": "PRIVATE_BODY_NEVER_VISIBLE", "visibility": "private"}, 200)
	app, e := New(context.Background(), s.DB, s.EncryptionKey, "support-test", "admin@example.test", "Integration-Test-Password-2026!", os.DirFS("../../web/dist"))
	if e != nil {
		t.Fatal(e)
	}
	server := httptest.NewServer(app)
	defer server.Close()
	ctx, cancel := context.WithTimeout(t.Context(), 150*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, "node", "../../tests/support.mjs")
	command.Env = append(os.Environ(), "MADI_BASE_URL="+server.URL, "MADI_SUPPORT_WORKSPACE="+wid, "MADI_SUPPORT_DOCUMENT="+did, "MADI_SUPPORT_TARGET="+tid)
	output, e := command.CombinedOutput()
	if e != nil {
		t.Fatalf("Support browser %v\n%s", e, output)
	}
	t.Log(string(output))
}
