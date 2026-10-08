package mcp_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/hkjang/umm/internal/auth"
	"github.com/hkjang/umm/internal/mcp"
	"github.com/hkjang/umm/internal/store"
)

// A browser session must not reach MCP.
//
// The guard is one comparison — the principal's AuthType has to be an API key —
// and nothing held it. What it holds back is the whole point of the key model:
// a session principal carries Scopes {"*": true}, every scope there is. If that
// comparison were ever dropped or renamed, a signed-in browser would reach
// every tool with unlimited scope and the keys people carefully narrow would
// stop meaning anything.
//
// Both directions are checked, so the test cannot pass by refusing everyone.
func TestMCPTakesAKeyAndNotASessionIntegration(t *testing.T) {
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		t.Skip("POSTGRES_DSN is not configured")
	}
	ctx := context.Background()
	db, err := store.Open(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Pool.Close() })
	if err = db.Migrate(ctx); err != nil {
		t.Fatal(err)
	}

	userID := uuid.New()
	username := "mcp_auth_" + strings.ReplaceAll(userID.String(), "-", "")
	if _, err = db.Pool.Exec(ctx, `INSERT INTO users(id,username,display_name) VALUES($1,$2::citext,$2::text)`, userID, username); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Pool.Exec(context.Background(), `DELETE FROM users WHERE id=$1`, userID) })

	authService := &auth.Service{Store: db}
	handler := authService.Middleware(&mcp.Handler{Store: db, Version: "test"})

	// Trimmed: the tool list is four kilobytes of JSON, and a failure nobody
	// can read is a failure nobody acts on.
	short := func(body string) string {
		if len(body) > 220 {
			return body[:220] + "…"
		}
		return body
	}
	call := func(apply func(*http.Request)) (int, string) {
		body, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/list"})
		request := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		apply(request)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		return response.Code, response.Body.String()
	}

	// A signed-in browser is refused, and told why.
	session, err := authService.CreateSession(ctx, userID, auth.SessionOrigin{UserAgent: "integration-test", ClientIP: "127.0.0.1"})
	if err != nil {
		t.Fatal(err)
	}
	code, body := call(func(r *http.Request) { r.AddCookie(&http.Cookie{Name: auth.CookieName, Value: session}) })
	if code != http.StatusUnauthorized {
		t.Errorf("a browser session reached MCP with %d: %s", code, short(body))
	}
	if !strings.Contains(body, "Bearer API key required") {
		t.Errorf("the refusal did not say a key is needed: %s", short(body))
	}
	if strings.Contains(body, "capture_thought") {
		t.Errorf("a browser session was handed the tool list: %s", short(body))
	}

	// And a key is not refused, so the check above is about sessions rather
	// than about everything.
	_, secret, err := authService.CreateKey(ctx, userID, "mcp", []string{"notes:read"}, 30)
	if err != nil {
		t.Fatal(err)
	}
	code, body = call(func(r *http.Request) { r.Header.Set("Authorization", "Bearer "+secret) })
	if code != http.StatusOK {
		t.Fatalf("an API key was refused with %d: %s", code, short(body))
	}
	if !strings.Contains(body, "capture_thought") {
		t.Errorf("the tool list did not come back for a key: %s", short(body))
	}
}
