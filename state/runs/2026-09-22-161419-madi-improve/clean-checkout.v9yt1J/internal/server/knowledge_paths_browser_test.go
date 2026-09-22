package server

import (
	"context"
	"net/http/httptest"
	"os"
	"os/exec"
	"testing"
	"time"
)

func TestBrowserLearningPaths(t *testing.T) {
	if os.Getenv("MADI_BROWSER_LEARNING_PATHS") != "1" {
		t.Skip("MADI_BROWSER_LEARNING_PATHS=1 after web build")
	}
	seed, _ := integrationTestServer(t)
	app, e := New(t.Context(), seed.DB, seed.EncryptionKey, "0.1.0", "admin@example.test", "Integration-Test-Password-2026!", os.DirFS("../../web/dist"))
	if e != nil {
		t.Fatal(e)
	}
	endpoint := httptest.NewServer(app)
	defer endpoint.Close()
	defer app.CloseCollaboration()
	ctx, cancel := context.WithTimeout(t.Context(), 4*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "node", "../../tests/learning-paths-browser.mjs")
	cmd.Env = append(os.Environ(), "MADI_BASE_URL="+endpoint.URL, "MADI_TEST_PASSWORD=Integration-Test-Password-2026!")
	output, e := cmd.CombinedOutput()
	if e != nil {
		t.Fatalf("learning browser: %v\n%s", e, output)
	}
	t.Log(string(output))
}
