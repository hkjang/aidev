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

func TestBrowserUXStructure(t *testing.T) {
	if os.Getenv("MADI_BROWSER_UX") != "1" {
		t.Skip("set MADI_BROWSER_UX=1 after web build")
	}
	seed, _ := integrationTestServer(t)
	app, err := New(context.Background(), seed.DB, seed.EncryptionKey, "0.1.0", "admin@example.test", "Integration-Test-Password-2026!", os.DirFS("../../web/dist"))
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(app)
	defer server.Close()
	ctx, cancel := context.WithTimeout(t.Context(), 210*time.Second)
	defer cancel()
	app.StartJobs(ctx)
	screenshotRoot, err := filepath.Abs("../../test-results/ux-structure")
	if err != nil {
		t.Fatal(err)
	}
	command := exec.CommandContext(ctx, "node", "../../tests/ux-structure.mjs")
	command.Env = append(os.Environ(), "MADI_BASE_URL="+server.URL, "MADI_ADMIN_PASSWORD=Integration-Test-Password-2026!", "MADI_SCREENSHOT_DIR="+screenshotRoot)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("UX browser %v\n%s", err, output)
	}
	t.Log(string(output))
}
