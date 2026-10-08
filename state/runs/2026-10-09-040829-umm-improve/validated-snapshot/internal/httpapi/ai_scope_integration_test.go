package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/hkjang/umm/internal/auth"
)

func TestAIAssistRequiresDedicatedScopeIntegration(t *testing.T) {
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		t.Skip("POSTGRES_DSN is not configured")
	}
	ctx := context.Background()
	db := isolatedHTTPStore(t, dsn)
	server := &Server{Store: db}
	config := server.getSecurityConfig(httptest.NewRequest(http.MethodGet, "/", nil))
	if !slices.Contains(config.APIKeyScopes, "ai:assist") || !validateScopes([]string{"ai:assist"}, config.APIKeyScopes) {
		t.Fatalf("migration did not make ai:assist issuable: %#v", config.APIKeyScopes)
	}

	userID := uuid.New()
	username := "ai_scope_" + strings.ReplaceAll(userID.String(), "-", "")
	if _, err := db.Pool.Exec(ctx, `INSERT INTO users(id,username,display_name) VALUES($1,$2::citext,$2::text)`, userID, username); err != nil {
		t.Fatal(err)
	}
	authService := &auth.Service{Store: db}
	_, notesSecret, err := authService.CreateKey(ctx, userID, "read only", []string{"notes:read"}, 1)
	if err != nil {
		t.Fatal(err)
	}
	_, assistSecret, err := authService.CreateKey(ctx, userID, "assist only", []string{"ai:assist"}, 1)
	if err != nil {
		t.Fatal(err)
	}
	handler := authService.Middleware(auth.Require(http.HandlerFunc(server.aiAssist)))
	request := func(secret, body string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/ai/assist", strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+secret)
		req.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, req)
		return response
	}

	denied := request(notesSecret, `{"noteIds":[],"mode":"summarize"}`)
	if denied.Code != http.StatusForbidden || !strings.Contains(denied.Body.String(), "ai:assist") {
		t.Fatalf("notes:read key reached AI Assist: status=%d body=%s", denied.Code, denied.Body.String())
	}
	allowed := request(assistSecret, `{`)
	if allowed.Code != http.StatusBadRequest {
		t.Fatalf("ai:assist key did not pass scope enforcement: status=%d body=%s", allowed.Code, allowed.Body.String())
	}
}
