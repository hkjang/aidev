package server

import (
	"context"
	"fmt"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestBrowserSearchOperations(t *testing.T) {
	if os.Getenv("MADI_BROWSER_SEARCH_OPERATIONS") != "1" {
		t.Skip("set MADI_BROWSER_SEARCH_OPERATIONS=1 after web build")
	}
	seed, admin, _, wid, _ := collaborationTestSetup(t)
	ids := []string{}
	for i := 0; i < 45; i++ {
		doc := testJSONObject(t, admin.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": fmt.Sprintf("쿠버네티스 운영 %02d", i+1), "markdown": "# 쿠버네티스 운영\n\n장애 대응 근거를 확인합니다.\n\n- [ ] 쿠버네티스 운영 점검\n\n```text\n쿠버네티스 운영 코드\n```", "tags": []string{"운영검증"}}, 200))
		id := str(doc, "id")
		ids = append(ids, id)
		if _, e := seed.indexSearchDocument(t.Context(), id); e != nil {
			t.Fatal(e)
		}
	}
	app, e := New(context.Background(), seed.DB, seed.EncryptionKey, "0.1.0", "admin@example.test", "Integration-Test-Password-2026!", os.DirFS("../../web/dist"))
	if e != nil {
		t.Fatal(e)
	}
	server := httptest.NewServer(app)
	defer server.Close()
	defer app.CloseCollaboration()
	ctx, cancel := context.WithTimeout(t.Context(), 180*time.Second)
	defer cancel()
	output, _ := filepath.Abs("../../test-results/search-operations")
	command := exec.CommandContext(ctx, "node", "../../tests/search-operations.mjs")
	command.Env = append(os.Environ(), "MADI_BASE_URL="+server.URL, "MADI_WORKSPACE_ID="+wid, "MADI_DOCUMENT_ID="+ids[0], "MADI_SCREENSHOT_DIR="+output)
	raw, e := command.CombinedOutput()
	if e != nil {
		t.Fatalf("search browser: %v\n%s", e, raw)
	}
	t.Log(string(raw))
}
