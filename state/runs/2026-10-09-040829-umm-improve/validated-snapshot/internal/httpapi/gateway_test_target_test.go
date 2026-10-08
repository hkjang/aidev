package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hkjang/umm/internal/auth"
	"github.com/hkjang/umm/internal/store"
)

// The connection test has to check the server embeddings are actually sent to.
//
// It read only the chat address at first, so once the two could differ it
// reported on a server nothing embeds against — a green result that means
// nothing, which is worse than a red one.
func TestGatewayTestUsesTheEmbeddingAddressIntegration(t *testing.T) {
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		t.Skip("POSTGRES_DSN is not configured")
	}
	ctx := context.Background()
	db := isolatedHTTPStore(t, dsn)

	var mu sync.Mutex
	var chatHits, embedHits int
	var sawAuth string
	chat := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		chatHits++
		sawAuth = r.Header.Get("Authorization")
		mu.Unlock()
		http.Error(w, "the chat gateway should not be asked to embed", http.StatusTeapot)
	}))
	defer chat.Close()
	embed := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		embedHits++
		sawAuth = r.Header.Get("Authorization")
		mu.Unlock()
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{"embedding": []float64{0.1, 0.2, 0.3}, "index": 0},
				{"embedding": []float64{0.2, 0.1, 0.4}, "index": 1},
			},
		})
	}))
	defer embed.Close()

	userID := uuid.New()
	username := "gwtest_" + strings.ReplaceAll(userID.String(), "-", "")
	if _, err := db.Pool.Exec(ctx, `INSERT INTO users(id,username,display_name,role) VALUES($1,$2::citext,$2::text,'admin')`,
		userID, username); err != nil {
		t.Fatal(err)
	}
	authService := &auth.Service{Store: db}
	session, err := authService.CreateSession(ctx, userID, auth.SessionOrigin{UserAgent: "integration-test", ClientIP: "127.0.0.1"})
	if err != nil {
		t.Fatal(err)
	}
	server := &Server{Store: db}
	router := chi.NewRouter()
	router.Post("/admin/ai-gateway/test", server.testEmbeddingGateway)
	handler := authService.Middleware(auth.Require(router))

	body, _ := json.Marshal(map[string]any{
		"base_url":           chat.URL,
		"api_key":            "chat-secret-must-not-travel",
		"embedding_base_url": embed.URL,
		"embedding_api_key":  "",
		"embedding_model":    "bge-m3",
	})
	request := httptest.NewRequest(http.MethodPost, "/admin/ai-gateway/test", strings.NewReader(string(body)))
	request.Header.Set("Content-Type", "application/json")
	request.AddCookie(&http.Cookie{Name: auth.CookieName, Value: session})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("test returned %d: %s", response.Code, response.Body.String())
	}
	var result struct {
		OK     bool   `json:"ok"`
		Detail string `json:"detail"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if !result.OK {
		t.Fatalf("the embedding server answered but the test failed: %s", result.Detail)
	}

	mu.Lock()
	defer mu.Unlock()
	if embedHits == 0 {
		t.Error("the embedding server was never asked")
	}
	if chatHits != 0 {
		t.Errorf("the chat gateway was asked to embed %d times", chatHits)
	}
	// The rule that makes a separate address safe holds here too.
	if strings.Contains(sawAuth, "chat-secret-must-not-travel") {
		t.Errorf("the chat gateway's key was sent to the embedding server: %q", sawAuth)
	}
}

// With no separate address the test still checks the chat gateway, which is
// where embeddings go in that case.
func TestGatewayTestFallsBackToTheChatAddressIntegration(t *testing.T) {
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		t.Skip("POSTGRES_DSN is not configured")
	}
	ctx := context.Background()
	db := isolatedHTTPStore(t, dsn)

	var mu sync.Mutex
	var hits int
	gateway := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		mu.Lock()
		hits++
		mu.Unlock()
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{"embedding": []float64{0.1, 0.2}, "index": 0},
				{"embedding": []float64{0.3, 0.4}, "index": 1},
			},
		})
	}))
	defer gateway.Close()

	userID := uuid.New()
	username := "gwfall_" + strings.ReplaceAll(userID.String(), "-", "")
	if _, err := db.Pool.Exec(ctx, `INSERT INTO users(id,username,display_name,role) VALUES($1,$2::citext,$2::text,'admin')`,
		userID, username); err != nil {
		t.Fatal(err)
	}
	authService := &auth.Service{Store: db}
	session, err := authService.CreateSession(ctx, userID, auth.SessionOrigin{UserAgent: "integration-test", ClientIP: "127.0.0.1"})
	if err != nil {
		t.Fatal(err)
	}
	server := &Server{Store: db}
	router := chi.NewRouter()
	router.Post("/admin/ai-gateway/test", server.testEmbeddingGateway)
	handler := authService.Middleware(auth.Require(router))

	body, _ := json.Marshal(map[string]any{
		"base_url": gateway.URL, "embedding_model": "bge-m3", "embedding_base_url": "",
	})
	request := httptest.NewRequest(http.MethodPost, "/admin/ai-gateway/test", strings.NewReader(string(body)))
	request.Header.Set("Content-Type", "application/json")
	request.AddCookie(&http.Cookie{Name: auth.CookieName, Value: session})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("test returned %d: %s", response.Code, response.Body.String())
	}
	mu.Lock()
	defer mu.Unlock()
	if hits == 0 {
		t.Error("with no embedding address the chat gateway should have been tested")
	}
	_ = store.ResolveEmbeddingEndpoint
}
