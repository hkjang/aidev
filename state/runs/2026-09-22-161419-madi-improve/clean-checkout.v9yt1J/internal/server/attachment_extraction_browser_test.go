package server

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"testing"
	"time"
)

func TestBrowserAttachmentExtraction(t *testing.T) {
	if os.Getenv("MADI_BROWSER_ATTACHMENT_EXTRACTION") != "1" {
		t.Skip("MADI_BROWSER_ATTACHMENT_EXTRACTION=1 after web build")
	}
	seed, _ := integrationTestServer(t)
	app, e := New(context.Background(), seed.DB, seed.EncryptionKey, "0.1.0", "admin@example.test", "Integration-Test-Password-2026!", os.DirFS("../../web/dist"))
	if e != nil {
		t.Fatal(e)
	}
	endpoint := httptest.NewServer(app)
	defer endpoint.Close()
	defer app.CloseCollaboration()
	ctx, cancel := context.WithTimeout(t.Context(), 4*time.Minute)
	defer cancel()
	app.StartJobs(ctx)
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"첨부에 기록된 쿠버네티스 운영 근거입니다. [1]\"}}]}\n\ndata: [DONE]\n\n")
	}))
	defer provider.Close()
	command := exec.CommandContext(ctx, "node", "../../tests/attachment-browser.mjs")
	command.Env = append(os.Environ(), "MADI_BASE_URL="+endpoint.URL, "MADI_TEST_PASSWORD=Integration-Test-Password-2026!", "MADI_TEST_AI_URL="+provider.URL)
	output, e := command.CombinedOutput()
	if e != nil {
		t.Fatalf("attachment browser: %v\n%s", e, output)
	}
	t.Log(string(output))
}
