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

func TestBrowserImpactException(t *testing.T) {
	if os.Getenv("MADI_BROWSER_IMPACT_EXCEPTION") != "1" {
		t.Skip("set MADI_BROWSER_IMPACT_EXCEPTION=1 after web build")
	}
	seed, c, _, wid, source, target, review := impactExceptionFixture(t)
	impactExceptionPolicy(t, c, wid)
	app, e := New(context.Background(), seed.DB, seed.EncryptionKey, "0.1.0", "admin@example.test", "Integration-Test-Password-2026!", os.DirFS("../../web/dist"))
	if e != nil {
		t.Fatal(e)
	}
	server := httptest.NewServer(app)
	defer server.Close()
	defer app.CloseCollaboration()
	ctx, cancel := context.WithTimeout(t.Context(), 180*time.Second)
	defer cancel()
	out, _ := filepath.Abs("../../test-results/impact-exception")
	command := exec.CommandContext(ctx, "node", "../../tests/impact-exception.mjs")
	command.Env = append(os.Environ(), "MADI_BASE_URL="+server.URL, "MADI_WORKSPACE_ID="+wid, "MADI_SOURCE_ID="+source, "MADI_TARGET_ID="+target, "MADI_REVIEW_ID="+review, "MADI_SCREENSHOT_DIR="+out)
	raw, e := command.CombinedOutput()
	if e != nil {
		t.Fatalf("exception browser: %v\n%s", e, raw)
	}
	t.Log(string(raw))
}
