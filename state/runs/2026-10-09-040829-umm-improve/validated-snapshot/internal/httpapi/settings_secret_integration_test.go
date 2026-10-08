package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hkjang/umm/internal/auth"
	"github.com/hkjang/umm/internal/cryptoutil"
	"github.com/hkjang/umm/internal/store"
)

// A saved secret has to survive the next save of its section.
//
// The settings screen never sends a stored key back — it shows a mask — so the
// write path drops the field and relies on the store merging the ciphertext it
// already has. That merge was driven by a list of sections, and ptium was not
// on it, so saving the section again wrote the object without the key. An
// administrator configured Ptium, came back, and had to type it in again.
func TestSavingASectionAgainKeepsItsSecretIntegration(t *testing.T) {
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		t.Skip("POSTGRES_DSN is not configured")
	}
	ctx := context.Background()
	db := isolatedHTTPStore(t, dsn)

	adminID := uuid.New()
	username := "secret_keep_" + strings.ReplaceAll(adminID.String(), "-", "")
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
	server := &Server{Store: db, Cipher: cipher}
	router := chi.NewRouter()
	router.Put("/settings/{section}", server.putAdminSetting)
	handler := authService.Middleware(auth.Require(auth.RequireAdmin(router)))

	save := func(section, body string) int {
		t.Helper()
		request := httptest.NewRequest(http.MethodPut, "/settings/"+section, strings.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		request.AddCookie(&http.Cookie{Name: auth.CookieName, Value: session})
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		return response.Code
	}
	stored := func(section, field string) string {
		t.Helper()
		var value map[string]any
		if err := db.GetSetting(ctx, section, &value); err != nil {
			t.Fatal(err)
		}
		text, _ := value[field].(string)
		return text
	}

	// Every section that keeps a secret, so a new one cannot be added to the
	// list in one place and forgotten in the other.
	for _, tc := range []struct {
		section string
		field   string
		first   string
		again   string
	}{
		{"ptium", "api_key", `{"enabled":true,"base_url":"https://ptium.example","api_key":"ptium-secret","timeout_seconds":30}`,
			`{"enabled":true,"base_url":"https://ptium.example","api_key":"` + secretMask + `","timeout_seconds":45}`},
	} {
		if code := save(tc.section, tc.first); code != http.StatusOK {
			t.Fatalf("%s: first save=%d", tc.section, code)
		}
		saved := stored(tc.section, tc.field)
		if !strings.HasPrefix(saved, "enc:") {
			t.Fatalf("%s: the secret was not stored encrypted: %q", tc.section, saved)
		}
		if code := save(tc.section, tc.again); code != http.StatusOK {
			t.Fatalf("%s: second save=%d", tc.section, code)
		}
		if after := stored(tc.section, tc.field); after != saved {
			t.Errorf("%s: saving the section again lost the stored secret (%q -> %q)", tc.section, saved, after)
		}
		// The rest of the section still updates; preserving the key must not
		// freeze everything beside it.
		var value map[string]any
		if err := db.GetSetting(ctx, tc.section, &value); err != nil {
			t.Fatal(err)
		}
		if timeout, _ := value["timeout_seconds"].(float64); timeout != 45 {
			t.Errorf("%s: the other fields did not update: timeout=%v", tc.section, timeout)
		}
	}

	// And a real new key still replaces the old one.
	before := stored("ptium", "api_key")
	if code := save("ptium", `{"enabled":true,"base_url":"https://ptium.example","api_key":"a-different-secret","timeout_seconds":45}`); code != http.StatusOK {
		t.Fatalf("replacing the key=%d", code)
	}
	if after := stored("ptium", "api_key"); after == before {
		t.Error("entering a new key did not replace the stored one")
	}
}

// Every section that masks a secret must also preserve it, and the two lists
// are now one. This holds them together.
func TestEverySecretSectionPreservesItsFields(t *testing.T) {
	for _, section := range []string{"oidc", "ai_gateway", "ptium"} {
		secrets := store.SecretSettingFields(section)
		if len(secrets) == 0 {
			t.Fatalf("%s: no secret fields", section)
		}
		preserved := store.PreservedSettingFields(section)
		for _, field := range secrets {
			found := false
			for _, kept := range preserved {
				if kept == field {
					found = true
				}
			}
			if !found {
				t.Errorf("%s: %q is masked on read but not preserved on write, so saving the section erases it", section, field)
			}
		}
	}
}
