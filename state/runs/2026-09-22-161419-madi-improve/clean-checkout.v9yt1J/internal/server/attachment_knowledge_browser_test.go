package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestBrowserAttachmentKnowledgeHandoff(t *testing.T) {
	if os.Getenv("MADI_BROWSER_ATTACHMENT_KNOWLEDGE") != "1" {
		t.Skip("MADI_BROWSER_ATTACHMENT_KNOWLEDGE=1 after web build")
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
	app.StartJobs(ctx)
	var slow atomic.Bool
	var cancelled atomic.Int64
	var mu sync.Mutex
	payloads := []string{}
	// Control endpoints exist only in this disposable test fixture. The service
	// has no attachment DELETE endpoint; deleting the exact fixture attachment
	// below exercises current API/UI revocation without inventing such a feature.
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/fixture/slow":
			slow.Store(r.Method == "POST")
			jsonResponse(w, 200, map[string]any{"slow": slow.Load()})
			return
		case "/fixture/state":
			mu.Lock()
			defer mu.Unlock()
			jsonResponse(w, 200, map[string]any{"payloads": payloads, "cancelled": cancelled.Load()})
			return
		case "/fixture/delete-attachment":
			var v struct {
				ID         string `json:"id"`
				DocumentID string `json:"document_id"`
			}
			if r.Method != "POST" || json.NewDecoder(io.LimitReader(r.Body, 4096)).Decode(&v) != nil || !validID(v.ID) || !validID(v.DocumentID) {
				http.Error(w, "invalid fixture target", 400)
				return
			}
			tag, err := seed.DB.Exec(r.Context(), `DELETE FROM attachments WHERE id=$1 AND document_id=$2`, v.ID, v.DocumentID)
			if err != nil || tag.RowsAffected() != 1 {
				http.Error(w, "fixture target missing", 400)
				return
			}
			jsonResponse(w, 200, map[string]any{"deleted": true})
			return
		}
		if !strings.HasSuffix(r.URL.Path, "/chat/completions") {
			http.NotFound(w, r)
			return
		}
		raw, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil {
			http.Error(w, "read", 400)
			return
		}
		mu.Lock()
		payloads = append(payloads, string(raw))
		mu.Unlock()
		w.Header().Set("Content-Type", "text/event-stream")
		if slow.Load() {
			fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"RETRACT_ATTACHMENT_PENDING\"}}]}\n\n")
			w.(http.Flusher).Flush()
			select {
			case <-r.Context().Done():
				cancelled.Add(1)
				return
			case <-ctx.Done():
				return
			}
		}
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"선택한 첨부 원문으로 운영 근거를 확인했습니다. [1]\"}}]}\n\ndata: [DONE]\n\n")
	}))
	defer func() {
		cancel()
		provider.Close()
	}()
	cmd := exec.CommandContext(ctx, "node", "../../tests/attachment-knowledge-browser.mjs")
	cmd.Env = append(os.Environ(), "MADI_BASE_URL="+endpoint.URL, "MADI_TEST_PASSWORD=Integration-Test-Password-2026!", "MADI_TEST_AI_URL="+provider.URL)
	output, e := cmd.CombinedOutput()
	if e != nil {
		t.Fatalf("attachment knowledge handoff: %v\n%s", e, output)
	}
	t.Log(string(output))
}
