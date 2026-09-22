package server

import (
	"context"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

func TestBrowserRAGGenerations(t *testing.T) {
	if os.Getenv("MADI_BROWSER_RAG_GENERATIONS") != "1" {
		t.Skip("set MADI_BROWSER_RAG_GENERATIONS=1 after web build")
	}
	seed, admin, wid, _ := ragTestSetup(t)
	provider, _, _ := ragTestProvider(t, nil)
	ragTestConfigure(t, admin, provider.URL)
	doc := str(testJSONObject(t, admin.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "GPU 세대 검증 문서", "markdown": "# GPU 운영\n\nGPU 장애 대응 근거입니다. 저장된 원문과 권한을 보존합니다.", "visibility": "private"}, 200)), "id")
	initial := ragTestConsent(t, admin, doc, 1, false, false)
	ragTestRun(t, seed, str(initial, "job_id"))
	var installed bool
	if e := seed.DB.QueryRow(t.Context(), `SELECT EXISTS(SELECT 1 FROM pg_extension WHERE extname='vector')`).Scan(&installed); e != nil {
		t.Fatal(e)
	}
	app, e := New(context.Background(), seed.DB, seed.EncryptionKey, "0.1.0", "admin@example.test", "Integration-Test-Password-2026!", os.DirFS("../../web/dist"))
	if e != nil {
		t.Fatal(e)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 240*time.Second)
	defer cancel()
	defer app.CloseCollaboration()
	server := httptest.NewServer(app)
	defer server.Close()
	app.StartJobs(ctx)
	output, _ := filepath.Abs("../../test-results/rag-generations")
	command := exec.CommandContext(ctx, "node", "../../tests/rag-generations.mjs")
	command.Env = append(os.Environ(), "MADI_BASE_URL="+server.URL, "MADI_WORKSPACE_ID="+wid, "MADI_DOCUMENT_ID="+doc, "MADI_HAS_PGVECTOR="+strconv.FormatBool(installed), "MADI_SCREENSHOT_DIR="+output)
	raw, e := command.CombinedOutput()
	if e != nil {
		t.Fatalf("RAG generations browser: %v\n%s", e, raw)
	}
	t.Log(string(raw))
}
