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

// Rotating a key that cannot be rotated, and rotating one that fails.
//
// The rotate endpoint answered "there is no active key to rotate" for every
// error. Only the first lookup can honestly say that; the new key, the overlap
// and the commit all fail differently, and reporting those as "no such key"
// sends someone looking for a key that is sitting in their list.
//
// The failure is produced rather than imagined: the api_keys table is made
// unwritable inside this test's own schema, so the insert the rotation performs
// really cannot succeed while the lookup before it still can.
func TestKeyRotationSaysItFailedRatherThanMissingIntegration(t *testing.T) {
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		t.Skip("POSTGRES_DSN is not configured")
	}
	ctx := context.Background()
	db := isolatedHTTPStore(t, dsn)

	userID := uuid.New()
	username := "rotate_fail_" + strings.ReplaceAll(userID.String(), "-", "")
	if _, err := db.Pool.Exec(ctx, `INSERT INTO users(id,username,display_name) VALUES($1,$2::citext,$2::text)`, userID, username); err != nil {
		t.Fatal(err)
	}
	authService := &auth.Service{Store: db}
	created, _, err := authService.CreateKey(ctx, userID, "회전할 키", []string{"notes:read"}, 30)
	if err != nil {
		t.Fatal(err)
	}

	server := &Server{Store: db, Auth: authService}
	router := chi.NewRouter()
	router.Post("/api-keys/{keyID}/rotate", server.rotateAPIKey)
	handler := authService.Middleware(auth.Require(router))

	call := func(id uuid.UUID) *httptest.ResponseRecorder {
		session, sessionErr := authService.CreateSession(ctx, userID, auth.SessionOrigin{UserAgent: "integration-test", ClientIP: "127.0.0.1"})
		if sessionErr != nil {
			t.Fatal(sessionErr)
		}
		request := httptest.NewRequest(http.MethodPost, "/api-keys/"+id.String()+"/rotate", nil)
		request.AddCookie(&http.Cookie{Name: auth.CookieName, Value: session})
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		return response
	}

	// A key that genuinely is not there still says so.
	if response := call(uuid.New()); response.Code != http.StatusNotFound {
		t.Errorf("rotating a key that does not exist returned %d, want 404", response.Code)
	}

	// Now make the rotation fail for real, after its lookup has succeeded.
	if _, err = db.Pool.Exec(ctx, `
		CREATE OR REPLACE FUNCTION refuse_api_key_insert() RETURNS trigger AS $$
		BEGIN RAISE EXCEPTION 'api key storage unavailable'; END $$ LANGUAGE plpgsql`); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Pool.Exec(ctx, `
		CREATE TRIGGER refuse_rotation BEFORE INSERT ON api_keys
		FOR EACH ROW EXECUTE FUNCTION refuse_api_key_insert()`); err != nil {
		t.Fatal(err)
	}

	response := call(created.ID)
	if response.Code != http.StatusInternalServerError {
		t.Errorf("a rotation that failed returned %d, want 500: %s", response.Code, response.Body.String())
	}
	if strings.Contains(response.Body.String(), "찾을 수 없습니다") {
		t.Errorf("a rotation that failed said the key does not exist: %s", response.Body.String())
	}

	// And the original key is untouched, which is the point: being told it is
	// gone is what stops someone rotating it again.
	var status string
	if err = db.Pool.QueryRow(ctx, `SELECT status FROM api_keys WHERE id=$1`, created.ID).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "active" {
		t.Errorf("the key was left %s after a failed rotation, want active", status)
	}
}
