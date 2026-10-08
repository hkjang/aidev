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

// Which webhook is failing, whose it is, and with what.
//
// The metrics screen carried one number — deliveries failed in the last day —
// and every detail behind it was recorded and never shown. The number said
// something was wrong and nothing about where to look.
func TestAdminSeesWhichWebhookIsFailingIntegration(t *testing.T) {
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		t.Skip("POSTGRES_DSN is not configured")
	}
	ctx := context.Background()
	db := isolatedHTTPStore(t, dsn)

	admin, owner := uuid.New(), uuid.New()
	name := func(id uuid.UUID) string { return "wh_" + strings.ReplaceAll(id.String(), "-", "") }
	if _, err := db.Pool.Exec(ctx, `
		INSERT INTO users(id,username,display_name,role) VALUES($1,$2::citext,$2::text,'admin'),($3,$4::citext,$4::text,'user')`,
		admin, name(admin), owner, name(owner)); err != nil {
		t.Fatal(err)
	}
	healthy, broken := uuid.New(), uuid.New()
	if _, err := db.Pool.Exec(ctx, `
		INSERT INTO webhook_subscriptions(id,owner_id,name,url,secret_ciphertext,events,failure_count,last_error) VALUES
		($1,$3,'조용한 웹훅','https://example.com/quiet','enc:x',ARRAY['note.created'],0,''),
		($2,$3,'깨진 웹훅','https://hooks.example.com/services/T000/B000/SUPER-SECRET-TOKEN','enc:x',ARRAY['note.created'],7,'dial tcp: connection refused')`,
		healthy, broken, owner); err != nil {
		t.Fatal(err)
	}

	authService := &auth.Service{Store: db}
	server := &Server{Store: db, Auth: authService}
	router := chi.NewRouter()
	router.Get("/admin/webhooks", server.adminWebhooks)
	router.Post("/admin/webhooks/{webhookID}/pause", server.pauseWebhook)
	handler := authService.Middleware(auth.Require(router))
	session, err := authService.CreateSession(ctx, admin, auth.SessionOrigin{UserAgent: "integration-test", ClientIP: "127.0.0.1"})
	if err != nil {
		t.Fatal(err)
	}
	send := func(method, path string) (int, string) {
		request := httptest.NewRequest(method, path, nil)
		request.AddCookie(&http.Cookie{Name: auth.CookieName, Value: session})
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		return response.Code, response.Body.String()
	}

	code, body := send(http.MethodGet, "/admin/webhooks")
	if code != http.StatusOK {
		t.Fatalf("listing webhooks returned %d: %s", code, body)
	}

	// The URL is often the credential — a hook address with a token in its
	// path is the secret. An administrator gets the destination, not the key.
	if strings.Contains(body, "SUPER-SECRET-TOKEN") {
		t.Error("the webhook's secret path was handed to the admin screen")
	}
	if !strings.Contains(body, "https://hooks.example.com/…") {
		t.Errorf("the destination is not identifiable: %s", body)
	}

	// The failing one is named, with whose it is and why it failed.
	var listed struct {
		Webhooks []map[string]any `json:"webhooks"`
	}
	if err = json.Unmarshal([]byte(body), &listed); err != nil {
		t.Fatal(err)
	}
	if len(listed.Webhooks) != 2 {
		t.Fatalf("listed %d webhooks, want 2", len(listed.Webhooks))
	}
	// Worst first: an administrator opens this to find what is wrong.
	if listed.Webhooks[0]["name"] != "깨진 웹훅" {
		t.Errorf("the failing webhook is not first: %v", listed.Webhooks[0]["name"])
	}
	if listed.Webhooks[0]["lastError"] != "dial tcp: connection refused" {
		t.Errorf("the reason it failed is missing: %v", listed.Webhooks[0]["lastError"])
	}
	if listed.Webhooks[0]["owner"] != name(owner) {
		t.Errorf("whose webhook it is was not said: %v", listed.Webhooks[0]["owner"])
	}

	// Narrowed to the ones in trouble.
	code, body = send(http.MethodGet, "/admin/webhooks?failing=true")
	if code != http.StatusOK {
		t.Fatalf("filtering returned %d", code)
	}
	if strings.Contains(body, "조용한 웹훅") {
		t.Error("a webhook with no failures was listed among the failing")
	}

	// Pausing stops the deliveries and keeps the configuration.
	if code, body = send(http.MethodPost, "/admin/webhooks/"+broken.String()+"/pause"); code != http.StatusOK {
		t.Fatalf("pausing returned %d: %s", code, body)
	}
	var active bool
	var url, secret string
	if err = db.Pool.QueryRow(ctx, `SELECT active, url, secret_ciphertext FROM webhook_subscriptions WHERE id=$1`, broken).Scan(&active, &url, &secret); err != nil {
		t.Fatal(err)
	}
	if active {
		t.Error("the webhook is still delivering after being paused")
	}
	if url == "" || secret == "" {
		t.Error("pausing threw away the owner's configuration")
	}

	// Pausing twice is not a failure: there is simply nothing left to stop.
	if code, body = send(http.MethodPost, "/admin/webhooks/"+broken.String()+"/pause"); code != http.StatusOK {
		t.Errorf("pausing an already paused webhook returned %d: %s", code, body)
	}
	if code, _ = send(http.MethodPost, "/admin/webhooks/"+uuid.New().String()+"/pause"); code != http.StatusNotFound {
		t.Errorf("pausing a webhook that does not exist returned %d, want 404", code)
	}
}
