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
)

// "Not found" and "could not" are different things to be told.
//
// Deleting a webhook reported 404 for any failure at all, so a database error
// during the delete told the person their webhook was already gone. That is the
// worst possible thing to be wrong about here: they stop trying, and the notes
// keep going to whoever was on the other end.
//
// The failure is produced rather than imagined — the delete is made to fail for
// real by taking the cascade off the deliveries, in this test's own schema, so
// the statement the handler runs hits a foreign key it cannot satisfy.
func TestWebhookDeletionSaysItFailedRatherThanNotFoundIntegration(t *testing.T) {
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		t.Skip("POSTGRES_DSN is not configured")
	}
	ctx := context.Background()
	db := isolatedHTTPStore(t, dsn)

	ownerID, subscriptionID := uuid.New(), uuid.New()
	username := "webhook_fail_" + strings.ReplaceAll(ownerID.String(), "-", "")
	if _, err := db.Pool.Exec(ctx, `INSERT INTO users(id,username,display_name) VALUES($1,$2::citext,$2::text)`, ownerID, username); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Pool.Exec(ctx, `
		INSERT INTO webhook_subscriptions(id,owner_id,name,url,secret_ciphertext,events)
		VALUES($1,$2,'끊을 웹훅','https://example.com/webhook','enc:x',ARRAY['note.created'])`,
		subscriptionID, ownerID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Pool.Exec(ctx, `
		INSERT INTO webhook_deliveries(subscription_id,event_id,event_type,payload,status)
		VALUES($1,$2,'note.created','{}','queued')`, subscriptionID, uuid.New()); err != nil {
		t.Fatal(err)
	}

	authService := &auth.Service{Store: db}
	server := &Server{Store: db, Auth: authService}
	router := chi.NewRouter()
	router.Delete("/webhooks/{webhookID}", server.deleteWebhook)
	handler := authService.Middleware(auth.Require(router))

	call := func(id uuid.UUID) *httptest.ResponseRecorder {
		session, err := authService.CreateSession(ctx, ownerID, auth.SessionOrigin{UserAgent: "integration-test", ClientIP: "127.0.0.1"})
		if err != nil {
			t.Fatal(err)
		}
		request := httptest.NewRequest(http.MethodDelete, "/webhooks/"+id.String(), nil)
		request.AddCookie(&http.Cookie{Name: auth.CookieName, Value: session})
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		return response
	}

	// A webhook that genuinely is not there still says so.
	if response := call(uuid.New()); response.Code != http.StatusNotFound {
		t.Errorf("deleting a webhook that does not exist returned %d, want 404", response.Code)
	}

	// Now make the delete fail for real.
	var constraint string
	if err := db.Pool.QueryRow(ctx, `
		SELECT conname FROM pg_constraint
		WHERE conrelid='webhook_deliveries'::regclass AND contype='f'
		  AND confrelid='webhook_subscriptions'::regclass`).Scan(&constraint); err != nil {
		t.Fatal(err)
	}
	identifier := `"` + strings.ReplaceAll(constraint, `"`, `""`) + `"`
	if _, err := db.Pool.Exec(ctx, `ALTER TABLE webhook_deliveries DROP CONSTRAINT `+identifier); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Pool.Exec(ctx, `ALTER TABLE webhook_deliveries ADD CONSTRAINT `+identifier+
		` FOREIGN KEY (subscription_id) REFERENCES webhook_subscriptions(id) ON DELETE NO ACTION`); err != nil {
		t.Fatal(err)
	}

	response := call(subscriptionID)
	if response.Code != http.StatusInternalServerError {
		t.Errorf("a delete that failed returned %d, want 500: %s", response.Code, response.Body.String())
	}
	if strings.Contains(response.Body.String(), "찾을 수 없습니다") {
		t.Errorf("a delete that failed told the person the webhook does not exist: %s", response.Body.String())
	}

	// And it is still there, which is the point: saying it is gone when it is
	// not is what stops someone trying again.
	var alive bool
	if err := db.Pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM webhook_subscriptions WHERE id=$1)`, subscriptionID).Scan(&alive); err != nil {
		t.Fatal(err)
	}
	if !alive {
		t.Error("the webhook was deleted after all; this test no longer produces the failure it describes")
	}
}
