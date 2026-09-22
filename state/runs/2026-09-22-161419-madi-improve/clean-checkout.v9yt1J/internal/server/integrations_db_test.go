package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"testing/fstest"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Each run creates an isolated schema, so this test never mutates an existing madi service.
func integrationTestServer(t *testing.T) (*Server, *httptest.Server) {
	t.Helper()
	dsn := os.Getenv("MADI_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("MADI_TEST_POSTGRES_DSN이 설정된 경우 실제 PostgreSQL 통합 검증을 실행합니다.")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	t.Cleanup(cancel)
	admin, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(admin.Close)
	schema := "madi_test_" + strings.ReplaceAll(newID(), "-", "")
	if _, err = admin.Exec(ctx, "CREATE SCHEMA "+pgx.Identifier{schema}.Sanitize()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = admin.Exec(context.Background(), "DROP SCHEMA "+pgx.Identifier{schema}.Sanitize()+" CASCADE")
	})
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	config.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	s, err := New(ctx, pool, bytes.Repeat([]byte{7}, 32), "test", "admin@example.test", "Integration-Test-Password-2026!", fstest.MapFS{"index.html": &fstest.MapFile{Data: []byte("<html>madi</html>")}})
	if err != nil {
		t.Fatal(err)
	}
	// Keep every fixture's file writes as isolated as its database schema. A
	// non-root CI runner must never depend on /var/lib/madi being writable, and
	// tests must not create attachments in a developer's configured service root.
	if _, err = pool.Exec(ctx, "UPDATE settings SET data=jsonb_set(data,'{storage_path}',to_jsonb($1::text)) WHERE id=1", t.TempDir()); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(s)
	t.Cleanup(server.Close)
	return s, server
}

type integrationTestClient struct {
	t           *testing.T
	client      *http.Client
	base, token string
}

func newIntegrationTestClient(t *testing.T, base string) *integrationTestClient {
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	return &integrationTestClient{t: t, client: &http.Client{Jar: jar, Timeout: 10 * time.Second}, base: base}
}

func (c *integrationTestClient) request(method, path string, body any, want int) []byte {
	c.t.Helper()
	data, _ := json.Marshal(body)
	r, err := http.NewRequest(method, c.base+path, bytes.NewReader(data))
	if err != nil {
		c.t.Fatal(err)
	}
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-Madi-Request", "1")
	if c.token != "" {
		r.Header.Set("Authorization", "Bearer "+c.token)
	}
	response, err := c.client.Do(r)
	if err != nil {
		c.t.Fatal(err)
	}
	defer response.Body.Close()
	out, err := io.ReadAll(response.Body)
	if err != nil {
		c.t.Fatal(err)
	}
	if want == 0 {
		if response.StatusCode >= 300 {
			c.t.Fatalf("%s %s: %d %s", method, path, response.StatusCode, out)
		}
	} else if response.StatusCode != want {
		c.t.Fatalf("%s %s: got %d want %d: %s", method, path, response.StatusCode, want, out)
	}
	return out
}

func testJSONObject(t *testing.T, data []byte) map[string]any {
	t.Helper()
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestPostgresKeysMCPAndAIACL(t *testing.T) {
	s, server := integrationTestServer(t)
	admin := newIntegrationTestClient(t, server.URL)
	admin.request("POST", "/api/v1/auth/login", map[string]any{"email": "admin@example.test", "password": "Integration-Test-Password-2026!"}, 200)
	var workspaces []map[string]any
	if err := json.Unmarshal(admin.request("GET", "/api/v1/workspaces", nil, 200), &workspaces); err != nil {
		t.Fatal(err)
	}
	wid := str(workspaces[0], "id")
	public := testJSONObject(t, admin.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "운영 지침", "markdown": "PUBLIC_GUIDE 운영 지침을 확인하세요."}, 0))
	private := testJSONObject(t, admin.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "운영 기밀", "markdown": "PRIVATE_SENTINEL_947 운영 기밀", "visibility": "private"}, 0))
	editor := testJSONObject(t, admin.request("POST", "/api/v1/admin/users", map[string]any{"email": "editor@example.test", "name": "편집자", "password": "Integration-Editor-Password!", "role": "editor"}, 0))
	admin.request("PUT", "/api/v1/workspaces/"+wid+"/members", map[string]any{"email": "editor@example.test", "role": "editor"}, 0)
	issued := testJSONObject(t, admin.request("POST", "/api/v1/keys", map[string]any{"user_id": editor["id"], "name": "통합 테스트", "workspace_id": wid, "scopes": keyScopes, "expires_in_days": 30, "rate_limit": 500}, 201))
	key := issued["key"].(map[string]any)
	keyID := str(key, "id")
	keyClient := newIntegrationTestClient(t, server.URL)
	keyClient.token = str(issued, "token")
	keyClient.request("GET", "/api/v1/documents/"+str(public, "id"), nil, 200)
	keyClient.request("GET", "/api/v1/documents/"+str(private, "id"), nil, 404)
	keyClient.request("GET", "/api/v1/documents?workspace_id="+newID(), nil, 403)
	keyClient.request("POST", "/api/v1/keys", map[string]any{}, 403)
	mcp := testJSONObject(t, keyClient.request("POST", "/api/v1/mcp", map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{"name": "search_documents", "arguments": map[string]any{"query": "운영"}}}, 200))
	serialized, _ := json.Marshal(mcp)
	if strings.Contains(string(serialized), "PRIVATE_SENTINEL") || !strings.Contains(string(serialized), "PUBLIC_GUIDE") {
		t.Fatalf("MCP ACL failed: %s", serialized)
	}
	var calls atomic.Int32
	var mu sync.Mutex
	var payload map[string]any
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		mu.Lock()
		_ = json.NewDecoder(r.Body).Decode(&payload)
		mu.Unlock()
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("wrong AI endpoint: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"운영 지침입니다 [1].\"}}]}\n\ndata: [DONE]\n\n")
	}))
	defer provider.Close()
	admin.request("PUT", "/api/v1/admin/settings", map[string]any{"ai_enabled": true, "ai_base_url": provider.URL + "/v1", "ai_model": "test-model", "ai_api_key": "test-private-provider-key", "ai_max_tokens": 262144}, 200)
	settings := admin.request("GET", "/api/v1/admin/settings", nil, 200)
	if strings.Contains(string(settings), "test-private-provider-key") {
		t.Fatal("admin API disclosed saved AI secret")
	}
	var stored string
	if err := s.DB.QueryRow(context.Background(), "SELECT data->>'ai_api_key' FROM settings WHERE id=1").Scan(&stored); err != nil || !strings.HasPrefix(stored, "enc:") {
		t.Fatalf("secret not encrypted: %v", err)
	}
	keyClient.request("POST", "/api/v1/ai/chat", map[string]any{"prompt": "운영", "document_id": private["id"]}, 403)
	if calls.Load() != 0 {
		t.Fatal("unauthorized document reached AI provider")
	}
	stream := keyClient.request("POST", "/api/v1/ai/chat", map[string]any{"prompt": "운영", "workspace_id": wid}, 200)
	if !strings.Contains(string(stream), `"sources"`) || !strings.Contains(string(stream), "[DONE]") {
		t.Fatalf("invalid SSE: %s", stream)
	}
	mu.Lock()
	serialized, _ = json.Marshal(payload)
	mu.Unlock()
	if strings.Contains(string(serialized), "PRIVATE_SENTINEL") || !strings.Contains(string(serialized), "PUBLIC_GUIDE") || !strings.Contains(string(serialized), `"max_tokens":262144`) || !strings.Contains(string(serialized), `"stream":true`) {
		t.Fatalf("AI provider ACL/stream contract failed: %s", serialized)
	}
	rotated := testJSONObject(t, admin.request("POST", "/api/v1/keys/"+keyID+"/rotate", map[string]any{}, 200))
	keyClient.request("GET", "/api/v1/documents/"+str(public, "id"), nil, 401)
	keyClient.token = str(rotated, "token")
	keyClient.request("GET", "/api/v1/documents/"+str(public, "id"), nil, 200)
	admin.request("PUT", "/api/v1/keys/"+keyID, map[string]any{"scopes": []string{"document:read"}}, 200)
	keyClient.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "denied"}, 403)
	admin.request("PUT", "/api/v1/keys/"+keyID, map[string]any{"ip_allowlist": []string{"192.0.2.0/24"}}, 200)
	keyClient.request("GET", "/api/v1/documents", nil, 403)
	admin.request("PUT", "/api/v1/keys/"+keyID, map[string]any{"ip_allowlist": []string{}, "rate_limit": 1}, 200)
	rotated = testJSONObject(t, admin.request("POST", "/api/v1/keys/"+keyID+"/rotate", map[string]any{}, 200))
	keyClient.token = str(rotated, "token")
	keyClient.request("GET", "/api/v1/documents", nil, 200)
	keyClient.request("GET", "/api/v1/documents", nil, 429)
	admin.request("DELETE", "/api/v1/keys/"+keyID, nil, 200)
	keyClient.request("GET", "/api/v1/documents", nil, 401)
}
