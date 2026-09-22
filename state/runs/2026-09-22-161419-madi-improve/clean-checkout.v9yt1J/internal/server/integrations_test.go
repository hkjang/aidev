package server

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestIntegrationScopeBoundaries(t *testing.T) {
	tests := []struct {
		name, method, path string
		scopes             []string
		allowed            bool
	}{
		{"read page", "GET", "/api/v1/documents/abc", []string{"document:read"}, true},
		{"write cannot read", "GET", "/api/v1/documents/abc", []string{"document:write"}, false},
		{"read cannot write", "PUT", "/api/v1/documents/abc", []string{"document:read"}, false},
		{"document listing query requires document scope", "GET", "/api/v1/documents?q=hello", []string{"document:read"}, true},
		{"search-only cannot use document list", "GET", "/api/v1/documents?q=hello", []string{"search:read"}, false},
		{"database-only may navigate spaces", "GET", "/api/v1/spaces", []string{"database:read"}, true},
		{"database-only cannot read space documents", "GET", "/api/v1/spaces/x/documents", []string{"database:read"}, false},
		{"document scope may read space summaries", "GET", "/api/v1/spaces/x/documents", []string{"document:read"}, true},
		{"keys cannot mint keys", "POST", "/api/v1/keys", keyScopes, false},
		{"keys cannot change password", "PUT", "/api/v1/profile", keyScopes, false},
		{"admin key cannot change settings", "PUT", "/api/v1/admin/settings", keyScopes, false},
		{"keys cannot change memberships", "PUT", "/api/v1/workspaces/x/members", keyScopes, false},
		{"database write", "POST", "/api/v1/databases/x/rows", []string{"database:write"}, true},
		{"AI execute", "POST", "/api/v1/ai/chat", []string{"ai:execute"}, true},
		{"MCP delegates", "POST", "/mcp", []string{}, true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			p := &Principal{Role: "admin", TokenID: "key", Scopes: test.scopes}
			if got := integrationScopeAllowed(p, httptest.NewRequest(test.method, test.path, nil)); got != test.allowed {
				t.Fatalf("got %v want %v", got, test.allowed)
			}
		})
	}
}

func TestAPIKeyIPRestrictions(t *testing.T) {
	for _, test := range []struct {
		ip    string
		allow []string
		want  bool
	}{
		{"192.168.5.7", nil, true},
		{"192.168.5.7", []string{"192.168.5.0/24"}, true},
		{"192.168.6.7", []string{"192.168.5.0/24"}, false},
		{"::ffff:192.168.5.7", []string{"192.168.5.7"}, true},
		{"2001:db8::4", []string{"2001:db8::/32"}, true},
		{"bad", []string{"0.0.0.0/0"}, false},
	} {
		if got := integrationIPAllowed(test.ip, test.allow); got != test.want {
			t.Errorf("%s %v: got %v", test.ip, test.allow, got)
		}
	}
	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "192.168.1.5:4000"
	r.Header.Set("X-Forwarded-For", "127.0.0.1")
	if got := integrationClientIP(r); got != "192.168.1.5" {
		t.Fatalf("trusted spoofable header: %s", got)
	}
}

func TestKeyPolicyValidation(t *testing.T) {
	valid := keyInput{Name: "개인 연동", WorkspaceID: "workspace", Scopes: []string{"document:read"}}
	if err := validateKeyInput(&valid, map[string]any{"default_key_days": 30}); err != nil {
		t.Fatal(err)
	}
	if valid.ExpiresInDays != 30 || valid.RateLimit != 120 || valid.IPAllowlist == nil {
		t.Fatalf("defaults not applied: %+v", valid)
	}
	if err := validateKeyInput(&valid, map[string]any{"allowed_key_scopes": []string{}}); err == nil {
		t.Fatal("admin-disallowed scopes must be refused")
	}
	valid.IPAllowlist = []string{"not-an-ip"}
	if err := validateKeyInput(&valid, map[string]any{}); err == nil {
		t.Fatal("invalid IP accepted")
	}
}

func TestAIEndpoint(t *testing.T) {
	for _, test := range []struct{ base, want string }{
		{"http://localhost:11434/v1", "http://localhost:11434/v1/chat/completions"},
		{"https://llm.local/v1/", "https://llm.local/v1/chat/completions"},
		{"https://gateway.local/chat/completions?api-version=2024-10-21", "https://gateway.local/chat/completions?api-version=2024-10-21"},
	} {
		got, err := aiEndpoint(test.base)
		if err != nil || got != test.want {
			t.Errorf("%s: %s, %v", test.base, got, err)
		}
	}
	for _, base := range []string{"file:///tmp/key", "https://user:password@host/v1", "not-a-url"} {
		if _, err := aiEndpoint(base); err == nil {
			t.Errorf("accepted invalid URL %s", base)
		}
	}
}

func TestAIStreaming(t *testing.T) {
	good := ": keepalive\n\ndata: {\"choices\":[{\"delta\":{\"content\":\"안녕\"}}]}\n\ndata: {\"choices\":[{\"delta\":{\"content\":\"하세요\"}}]}\n\ndata: [DONE]\n\n"
	var result strings.Builder
	if err := streamAIResponse(context.Background(), strings.NewReader(good), func(text string) error { result.WriteString(text); return nil }); err != nil {
		t.Fatal(err)
	}
	if result.String() != "안녕하세요" {
		t.Fatalf("bad stream: %s", result.String())
	}
	for _, bad := range []string{"data: {\"choices\":[{\"delta\":{\"content\":\"partial\"}}]}\n\n", "data: invalid\n\n", "data: {\"error\":{\"message\":\"bad provider\"}}\n\n"} {
		if err := streamAIResponse(context.Background(), strings.NewReader(bad), func(string) error { return nil }); err == nil {
			t.Fatal("broken stream must fail")
		}
	}
	err := streamAIResponse(context.Background(), strings.NewReader(good), func(string) error { return io.ErrClosedPipe })
	if !errors.Is(err, io.ErrClosedPipe) {
		t.Fatalf("downstream cancellation not propagated: %v", err)
	}
}

func TestMCPProtocolAndScopeDiscovery(t *testing.T) {
	s := &Server{Version: "test"}
	call := func(body string, p *Principal) *httptest.ResponseRecorder {
		r := httptest.NewRequest("POST", "/api/v1/mcp", strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Accept", "application/json, text/event-stream")
		r = r.WithContext(context.WithValue(r.Context(), principalKey, p))
		w := httptest.NewRecorder()
		s.mcp(w, r)
		return w
	}
	p := &Principal{TokenID: "key", Scopes: []string{"search:read"}}
	w := call(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"future"}}`, p)
	if w.Code != 200 || !strings.Contains(w.Body.String(), mcpVersions[0]) {
		t.Fatalf("initialize: %s", w.Body)
	}
	w = call(`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`, p)
	if !strings.Contains(w.Body.String(), "search_documents") || strings.Contains(w.Body.String(), "create_document") {
		t.Fatalf("scope discovery: %s", w.Body)
	}
	w = call(`{"jsonrpc":"2.0","method":"notifications/initialized"}`, p)
	if w.Code != 202 || w.Body.Len() != 0 {
		t.Fatalf("notification response: %d %s", w.Code, w.Body)
	}
	w = call(`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"create_document","arguments":{"title":"x"}}}`, p)
	if !strings.Contains(w.Body.String(), `"isError":true`) {
		t.Fatalf("scope bypass: %s", w.Body)
	}
	w = call(`{"jsonrpc":"2.0","id":4,"method":"initialize"}`, &Principal{})
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("session MCP must be refused, got %d", w.Code)
	}
}

func TestMCPDispatchRejectsPathInjection(t *testing.T) {
	for _, id := range []string{"../admin/settings", "x?workspace_id=other", "", "123"} {
		if _, _, _, err := mcpDispatch("get_document", map[string]any{"document_id": id}, ""); err == nil {
			t.Fatalf("unsafe ID accepted: %q", id)
		}
	}
	wid := "00000000-0000-4000-8000-000000000001"
	method, path, body, err := mcpDispatch("create_document", map[string]any{"title": "새 문서"}, wid)
	if err != nil || method != "POST" || path != "/api/v1/documents" || body["workspace_id"] != wid {
		t.Fatalf("wrong create dispatch: %s %s %v %v", method, path, body, err)
	}
	for _, tool := range mcpTools() {
		if _, err := json.Marshal(tool); err != nil {
			t.Fatal(err)
		}
	}
}
