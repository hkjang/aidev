package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hkjang/umm/internal/auth"
)

// The audit log has to answer a question, not just be readable.
//
// It recorded everything worth recording and could only be read newest-first,
// all of it. "Who took this person out of that space" meant scrolling until you
// found it, on a log that only grows.
func TestAuditLogAnswersAQuestionIntegration(t *testing.T) {
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		t.Skip("POSTGRES_DSN is not configured")
	}
	ctx := context.Background()
	db := isolatedHTTPStore(t, dsn)

	alice, bob := uuid.New(), uuid.New()
	aliceName := "audit_a_" + strings.ReplaceAll(alice.String(), "-", "")
	bobName := "audit_b_" + strings.ReplaceAll(bob.String(), "-", "")
	if _, err := db.Pool.Exec(ctx, `
		INSERT INTO users(id,username,display_name,role) VALUES($1,$2::citext,$2::text,'admin'),($3,$4::citext,$4::text,'user')`,
		alice, aliceName, bob, bobName); err != nil {
		t.Fatal(err)
	}
	space := uuid.New().String()
	db.Audit(ctx, &alice, "space.unshare", "space", space, map[string]any{"userId": bob.String()})
	db.Audit(ctx, &alice, "api_key.rotate", "api_key", uuid.New().String(), map[string]any{})
	db.Audit(ctx, &bob, "note.delete", "note", uuid.New().String(), map[string]any{})
	db.Audit(ctx, nil, "dream.run", "dream", "manual", map[string]any{})

	authService := &auth.Service{Store: db}
	server := &Server{Store: db, Auth: authService}
	router := chi.NewRouter()
	router.Get("/admin/audit", server.adminAudit)
	handler := authService.Middleware(auth.Require(router))
	session, err := authService.CreateSession(ctx, alice, auth.SessionOrigin{UserAgent: "integration-test", ClientIP: "127.0.0.1"})
	if err != nil {
		t.Fatal(err)
	}

	ask := func(query string) []map[string]any {
		t.Helper()
		request := httptest.NewRequest(http.MethodGet, "/admin/audit?"+query, nil)
		request.AddCookie(&http.Cookie{Name: auth.CookieName, Value: session})
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Fatalf("audit?%s returned %d: %s", query, response.Code, response.Body.String())
		}
		var body struct {
			Audit   []map[string]any `json:"audit"`
			Actions []string         `json:"actions"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		return body.Audit
	}

	if all := ask(""); len(all) < 4 {
		t.Fatalf("no filter returned %d rows, want at least the 4 just written", len(all))
	}

	// The question this exists for.
	unshares := ask("action=space.unshare&resourceId=" + space)
	if len(unshares) != 1 {
		t.Fatalf("asking who unshared that space returned %d rows, want 1", len(unshares))
	}
	if unshares[0]["actor"] != aliceName {
		t.Errorf("the answer names %v, want %s", unshares[0]["actor"], aliceName)
	}

	// One person's actions, whoever else was busy.
	byBob := ask("actor=" + bobName)
	if len(byBob) != 1 || byBob[0]["action"] != "note.delete" {
		t.Errorf("filtering by actor returned %v", byBob)
	}

	// Filters combine rather than replacing one another.
	if combined := ask("actor=" + aliceName + "&action=api_key.rotate"); len(combined) != 1 {
		t.Errorf("two filters together returned %d rows, want 1", len(combined))
	}

	// An action nobody performed is empty, not everything: a filter that
	// silently fails to apply makes the page look like an answer.
	if none := ask("action=space.obliterate"); len(none) != 0 {
		t.Errorf("an action nobody performed returned %d rows", len(none))
	}
	if none := ask("actor=nobody_at_all"); len(none) != 0 {
		t.Errorf("an actor nobody is returned %d rows", len(none))
	}

	// What the listing calls 'system' has to be askable by that name.
	if system := ask("actor=system"); len(system) != 1 || system[0]["action"] != "dream.run" {
		t.Errorf("asking for system returned %v", system)
	}
}
