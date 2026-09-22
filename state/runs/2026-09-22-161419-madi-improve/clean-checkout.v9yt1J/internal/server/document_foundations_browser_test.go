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

func TestBrowserDocumentFoundations(t *testing.T) {
	if os.Getenv("MADI_BROWSER_FOUNDATIONS") != "1" {
		t.Skip("set MADI_BROWSER_FOUNDATIONS=1 after web build")
	}
	seed, _ := integrationTestServer(t)
	var storageRoot string
	if err := seed.DB.QueryRow(t.Context(), "SELECT data->>'storage_path' FROM settings WHERE id=1").Scan(&storageRoot); err != nil {
		t.Fatal(err)
	}
	if !filepath.IsAbs(storageRoot) || storageRoot == str(defaultSettings(), "storage_path") {
		t.Fatal("browser fixture must use its own temporary attachment root")
	}
	app, err := New(context.Background(), seed.DB, seed.EncryptionKey, "0.1.0", "admin@example.test", "Integration-Test-Password-2026!", os.DirFS("../../web/dist"))
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(app)
	defer server.Close()
	ctx, cancel := context.WithTimeout(t.Context(), 180*time.Second)
	defer cancel()
	app.StartJobs(ctx)
	command := exec.CommandContext(ctx, "node", "../../tests/document-foundations.mjs")
	command.Env = append(os.Environ(), "MADI_BASE_URL="+server.URL, "MADI_ADMIN_PASSWORD=Integration-Test-Password-2026!", "MADI_SCREENSHOT_DIR="+t.TempDir())
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("foundations browser %v\n%s", err, output)
	}
	t.Log(string(output))
	var attachmentPath string
	if err := seed.DB.QueryRow(t.Context(), "SELECT path FROM attachments WHERE name='한국어 첨부.txt' ORDER BY created_at DESC LIMIT 1").Scan(&attachmentPath); err != nil {
		t.Fatal(err)
	}
	rel, err := filepath.Rel(storageRoot, attachmentPath)
	if err != nil || !filepath.IsLocal(rel) {
		t.Fatal("browser attachment escaped fixture root", err)
	}
	data, err := os.ReadFile(attachmentPath)
	if err != nil || string(data) != "첨부파일 원본\n" {
		t.Fatal("browser attachment bytes not preserved in fixture root", err)
	}
}
