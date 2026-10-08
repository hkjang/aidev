package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// A canvas of two thousand thoughts is about a megabyte of JSON, and the
// documented install exposes umm directly rather than behind a proxy that would
// compress for it. Measured on a real space: 1,017,041 bytes became 65,424.
func TestLargeJSONResponsesAreCompressed(t *testing.T) {
	server := &Server{}
	body := strings.Repeat(`{"id":"a","content":"생각","x":0,"y":0},`, 2000)
	handler := server.compressResponses(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("[" + body + "]"))
	}))

	request := httptest.NewRequest(http.MethodGet, "/api/v1/spaces/x/notes", nil)
	request.Header.Set("Accept-Encoding", "gzip")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if got := response.Header().Get("Content-Encoding"); got != "gzip" {
		t.Fatalf("a large JSON response was not compressed: encoding=%q size=%d", got, response.Body.Len())
	}
	if response.Body.Len() >= len(body) {
		t.Errorf("compressed body is %d bytes against %d uncompressed", response.Body.Len(), len(body))
	}
}

// The bundle is served as text/javascript, not application/javascript.
//
// Listing only the latter left the 435 kB script uncompressed while the
// stylesheet next to it shrank — the kind of gap that looks fine in the code and
// only shows up in a response. Each type here is one the running server actually
// returns.
func TestTheContentTypesTheServerActuallyReturnsAreCompressed(t *testing.T) {
	server := &Server{}
	payload := strings.Repeat("export const thought = '생각';\n", 3000)
	for _, contentType := range []string{
		"text/javascript; charset=utf-8",
		"text/css; charset=utf-8",
		"application/json; charset=utf-8",
		"text/html; charset=utf-8",
		"image/svg+xml",
	} {
		handler := server.compressResponses(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", contentType)
			_, _ = w.Write([]byte(payload))
		}))
		request := httptest.NewRequest(http.MethodGet, "/assets/index-abc123.js", nil)
		request.Header.Set("Accept-Encoding", "gzip")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if got := response.Header().Get("Content-Encoding"); got != "gzip" {
			t.Errorf("%s was not compressed (encoding=%q, %d bytes)", contentType, got, response.Body.Len())
		}
	}
}

// A client that does not ask for compression must still get a readable answer.
func TestClientsThatDoNotAskAreNotCompressed(t *testing.T) {
	server := &Server{}
	handler := server.compressResponses(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))

	request := httptest.NewRequest(http.MethodGet, "/api/v1/spaces", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if got := response.Header().Get("Content-Encoding"); got != "" {
		t.Errorf("a client that asked for nothing got %q", got)
	}
	if response.Body.String() != `{"ok":true}` {
		t.Errorf("body = %q", response.Body.String())
	}
}

// Compressing a response whose length an attacker can watch while varying what
// it echoes is the shape of BREACH. umm's CSRF defence is origin-based rather
// than a token in the body, so the endpoints that mint a secret are the only
// ones where a secret appears in a response at all — and they stay uncompressed.
func TestResponsesCarryingAFreshSecretAreNotCompressed(t *testing.T) {
	server := &Server{}
	secret := strings.Repeat("umm_key_a1b2c3d4_", 400)
	handler := server.compressResponses(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"secret":"` + secret + `"}`))
	}))

	for _, path := range []string{
		"/api/v1/api-keys",
		"/api/v1/api-keys/2b0d/rotate",
		"/api/v1/webhooks",
		"/api/v1/webhooks/2b0d/rotate-secret",
	} {
		request := httptest.NewRequest(http.MethodPost, path, nil)
		request.Header.Set("Accept-Encoding", "gzip")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if got := response.Header().Get("Content-Encoding"); got != "" {
			t.Errorf("%s compressed a response carrying a fresh secret: %q", path, got)
		}
	}
}
