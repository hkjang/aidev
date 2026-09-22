package server

import (
	"context"
	"net/http/httptest"
	"os"
	"os/exec"
	"testing"
	"time"
)

func TestBrowserKnowledgeDistribution(t *testing.T) {
	if os.Getenv("MADI_BROWSER_KNOWLEDGE_DISTRIBUTION") != "1" {
		t.Skip("set MADI_BROWSER_KNOWLEDGE_DISTRIBUTION=1 after web build")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 240*time.Second)
	defer cancel()
	newEndpoint := func() string {
		seed, _ := integrationTestServer(t)
		app, e := New(context.Background(), seed.DB, seed.EncryptionKey, "0.1.0", "admin@example.test", "Integration-Test-Password-2026!", os.DirFS("../../web/dist"))
		if e != nil {
			t.Fatal(e)
		}
		endpoint := httptest.NewServer(app)
		t.Cleanup(endpoint.Close)
		t.Cleanup(app.CloseCollaboration)
		c := newIntegrationTestClient(t, endpoint.URL)
		c.request("POST", "/api/v1/auth/login", map[string]any{"email": "admin@example.test", "password": "Integration-Test-Password-2026!"}, 200)
		c.request("PUT", "/api/v1/admin/settings", map[string]any{"storage_path": t.TempDir()}, 200)
		app.StartJobs(ctx)
		return endpoint.URL
	}
	sender, receiver := newEndpoint(), newEndpoint()
	command := exec.CommandContext(ctx, "node", "../../tests/knowledge-distribution.mjs")
	command.Env = append(os.Environ(), "MADI_BASE_URL="+sender, "MADI_RECEIVER_URL="+receiver, "MADI_ADMIN_PASSWORD=Integration-Test-Password-2026!")
	out, e := command.CombinedOutput()
	if e != nil {
		t.Fatalf("distribution browser: %v\n%s", e, out)
	}
	t.Log(string(out))
}
