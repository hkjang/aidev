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
	"github.com/hkjang/umm/internal/cryptoutil"
)

// Testing the Ptium connection asks it for the template list, and that used to
// be the whole verdict: anything the list could not do was reported as the
// connection failing. Templates are the optional half — a deck can be made
// without one — so a Ptium that keeps them elsewhere, or answers that endpoint
// in a shape this version does not know, is still a Ptium umm can talk to.
//
// Being told the connection is broken when it works sends someone to check the
// address and the key, which are the two things that are fine.
func TestPtiumConnectionTestIntegration(t *testing.T) {
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		t.Skip("POSTGRES_DSN is not configured")
	}
	ctx := context.Background()
	db := isolatedHTTPStore(t, dsn)

	const savedKey = "ptium-secret-key-123"
	adminID := uuid.New()
	username := "ptium_test_" + strings.ReplaceAll(adminID.String(), "-", "")
	if _, err := db.Pool.Exec(ctx, `INSERT INTO users(id,username,display_name,role) VALUES($1,$2::citext,$2::text,'admin')`, adminID, username); err != nil {
		t.Fatal(err)
	}
	authService := &auth.Service{Store: db}
	session, err := authService.CreateSession(ctx, adminID, auth.SessionOrigin{UserAgent: "integration-test", ClientIP: "127.0.0.1"})
	if err != nil {
		t.Fatal(err)
	}
	cipher, err := cryptoutil.New([]byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatal(err)
	}
	db.Cipher = cipher
	server := &Server{Store: db, Cipher: cipher}
	router := chi.NewRouter()
	router.Post("/admin/ptium/test", server.testPtium)
	router.Put("/settings/{section}", server.putAdminSetting)
	handler := authService.Middleware(auth.Require(auth.RequireAdmin(router)))

	// One Ptium whose template endpoint can be told how to misbehave.
	mode := "ok"
	ptium := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+savedKey {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"detail":"invalid api key"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		switch mode {
		case "notfound":
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"detail":"no such endpoint"}`))
		case "shape":
			_, _ = w.Write([]byte(`["not","the","documented","shape"]`))
		case "error":
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"detail":"boom"}`))
		default:
			_, _ = w.Write([]byte(`{"data":[{"id":"t1","name":"basic"}]}`))
		}
	}))
	defer ptium.Close()

	post := func(path, body string) (int, map[string]any) {
		t.Helper()
		request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
		if strings.HasPrefix(path, "/settings") {
			request = httptest.NewRequest(http.MethodPut, path, strings.NewReader(body))
		}
		request.Header.Set("Content-Type", "application/json")
		request.AddCookie(&http.Cookie{Name: auth.CookieName, Value: session})
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		var payload map[string]any
		if response.Body.Len() > 0 {
			_ = json.Unmarshal(response.Body.Bytes(), &payload)
		}
		return response.Code, payload
	}
	test := func(key string) (int, map[string]any) {
		return post("/admin/ptium/test", `{"base_url":"`+ptium.URL+`","api_key":"`+key+`","timeout_seconds":10}`)
	}

	// Save the key first, so the masked path has something to fall back to.
	if code, _ := post("/settings/ptium", `{"enabled":true,"base_url":"`+ptium.URL+`","api_key":"`+savedKey+`","timeout_seconds":10}`); code != http.StatusOK {
		t.Fatalf("saving the settings=%d", code)
	}

	for _, tc := range []struct {
		mode      string
		wantCode  int
		wantInMsg string
		note      string
	}{
		{"ok", http.StatusOK, "템플릿 1개", "templates listed"},
		{"notfound", http.StatusOK, "템플릿 목록은 가져오지 못했습니다", "no template endpoint"},
		{"shape", http.StatusOK, "템플릿 목록은 가져오지 못했습니다", "an unfamiliar template shape"},
		{"error", http.StatusOK, "템플릿 목록은 가져오지 못했습니다", "templates erroring"},
	} {
		mode = tc.mode
		code, payload := test(secretMask)
		if code != tc.wantCode {
			t.Errorf("%s: HTTP %d, want %d (%v)", tc.note, code, tc.wantCode, payload)
			continue
		}
		message, _ := payload["message"].(string)
		if !strings.Contains(message, tc.wantInMsg) {
			t.Errorf("%s: message %q does not mention %q", tc.note, message, tc.wantInMsg)
		}
	}

	// A key Ptium refuses is not a working integration, and must not be softened
	// into "connected".
	mode = "ok"
	code, payload := test("the-wrong-key")
	if code != http.StatusBadRequest {
		t.Errorf("a rejected key gave HTTP %d, want 400 (%v)", code, payload)
	}
	if detail, _ := payload["detail"].(string); !strings.Contains(detail, "API 키를 거부") {
		t.Errorf("a rejected key was not reported as a key problem: %q", detail)
	}

	// The reported sequence, acted out. Something wrong is saved, the right key
	// is typed in and tested — it passes — and then the page is reloaded, at
	// which point the saved key is used again and the connection stops working
	// with nothing having changed on screen.
	if code, _ := post("/settings/ptium", `{"enabled":true,"base_url":"`+ptium.URL+`","api_key":"a-stale-key","timeout_seconds":10}`); code != http.StatusOK {
		t.Fatalf("saving the stale key=%d", code)
	}

	code, payload = test(savedKey)
	if code != http.StatusOK {
		t.Fatalf("testing the corrected key=%d (%v)", code, payload)
	}
	if payload["unsaved"] != true {
		t.Errorf("a key that is not the saved one was not reported as unsaved: %v", payload)
	}
	if message, _ := payload["message"].(string); !strings.Contains(message, "저장되지 않았습니다") {
		t.Errorf("the message does not say the key is unsaved: %q", message)
	}

	// The reload. Same screen, same button, and now it fails — which is the
	// symptom, and the reason the message above has to exist.
	if code, payload = test(secretMask); code != http.StatusBadRequest {
		t.Errorf("after the reload the stale key still passed: HTTP %d (%v)", code, payload)
	}

	// Saving what was tested is what makes it stick.
	if code, _ := post("/settings/ptium", `{"enabled":true,"base_url":"`+ptium.URL+`","api_key":"`+savedKey+`","timeout_seconds":10}`); code != http.StatusOK {
		t.Fatalf("saving the corrected key=%d", code)
	}
	code, payload = test(secretMask)
	if code != http.StatusOK {
		t.Errorf("after saving, the reload test still fails: HTTP %d (%v)", code, payload)
	}
	if payload["unsaved"] != false {
		t.Errorf("the saved key was reported as unsaved: %v", payload)
	}
}
