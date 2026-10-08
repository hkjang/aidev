package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hkjang/umm/internal/auth"
)

func TestLegacySecuritySavePreservesAbuseGuardsIntegration(t *testing.T) {
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		t.Skip("POSTGRES_DSN is not configured")
	}
	ctx := context.Background()
	db := isolatedHTTPStore(t, dsn)

	adminID := uuid.New()
	username := "security_compat_" + strings.ReplaceAll(adminID.String(), "-", "")
	if _, err := db.Pool.Exec(ctx, `INSERT INTO users(id,username,display_name,role) VALUES($1,$2::citext,$2::text,'admin')`, adminID, username); err != nil {
		t.Fatal(err)
	}
	seed := map[string]any{
		"api_key_scopes":         []string{"notes:read", "notes:write"},
		"default_key_days":       45,
		"rotation_overlap_hours": 12,
		"login_max_failures":     17,
		"login_lockout_minutes":  73,
		"api_rate_per_minute":    4321,
		"ai_rate_per_minute":     27,
		"ai_daily_limit":         321,
	}
	if err := db.PutSetting(ctx, "security", seed, adminID); err != nil {
		t.Fatal(err)
	}

	authService := &auth.Service{Store: db}
	session, err := authService.CreateSession(ctx, adminID, auth.SessionOrigin{UserAgent: "integration-test", ClientIP: "127.0.0.1"})
	if err != nil {
		t.Fatal(err)
	}
	server := &Server{Store: db}
	router := chi.NewRouter()
	router.Put("/settings/{section}", server.putAdminSetting)
	handler := authService.Middleware(auth.Require(auth.RequireAdmin(router)))
	save := func(body string) *httptest.ResponseRecorder {
		t.Helper()
		request := httptest.NewRequest(http.MethodPut, "/settings/security", strings.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		request.AddCookie(&http.Cookie{Name: auth.CookieName, Value: session})
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		return response
	}
	read := func() map[string]any {
		t.Helper()
		var value map[string]any
		if err := db.GetSetting(ctx, "security", &value); err != nil {
			t.Fatal(err)
		}
		return value
	}

	legacy := `{"api_key_scopes":["notes:read"],"default_key_days":90,"rotation_overlap_hours":24}`
	if response := save(legacy); response.Code != http.StatusOK {
		t.Fatalf("legacy security save=%d body=%s", response.Code, response.Body.String())
	}
	afterLegacy := read()
	for field, expected := range map[string]float64{
		"login_max_failures":    17,
		"login_lockout_minutes": 73,
		"api_rate_per_minute":   4321,
		"ai_rate_per_minute":    27,
		"ai_daily_limit":        321,
	} {
		if afterLegacy[field] != expected {
			t.Fatalf("legacy save changed %s: got=%v want=%v settings=%#v", field, afterLegacy[field], expected, afterLegacy)
		}
	}

	// Hold the same transaction lock while another administrator commits a
	// newer guard value. A legacy save must wait, then merge from that latest
	// committed row instead of applying a stale snapshot.
	lockTx, err := db.Pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer lockTx.Rollback(context.Background())
	if _, err = lockTx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, "umm:app-setting:security"); err != nil {
		t.Fatal(err)
	}
	if _, err = lockTx.Exec(ctx, `UPDATE app_settings SET value=jsonb_set(value,'{login_max_failures}','19'::jsonb) WHERE key='security'`); err != nil {
		t.Fatal(err)
	}
	concurrentSave := make(chan error, 1)
	go func() {
		concurrentSave <- db.PutSetting(context.Background(), "security", map[string]any{
			"api_key_scopes":         []string{"notes:read"},
			"default_key_days":       120,
			"rotation_overlap_hours": 48,
		}, adminID)
	}()
	select {
	case saveErr := <-concurrentSave:
		t.Fatalf("legacy save did not wait for the current setting transaction: %v", saveErr)
	case <-time.After(100 * time.Millisecond):
	}
	if err = lockTx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	select {
	case saveErr := <-concurrentSave:
		if saveErr != nil {
			t.Fatal(saveErr)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("legacy save did not resume after the setting transaction committed")
	}
	afterConcurrent := read()
	if afterConcurrent["login_max_failures"] != float64(19) {
		t.Fatalf("legacy save lost the latest committed guard: %#v", afterConcurrent)
	}

	explicitZero := `{"api_key_scopes":["notes:read"],"default_key_days":90,"rotation_overlap_hours":24,"ai_daily_limit":0}`
	if response := save(explicitZero); response.Code != http.StatusOK {
		t.Fatalf("explicit zero security save=%d body=%s", response.Code, response.Body.String())
	}
	afterZero := read()
	if afterZero["ai_daily_limit"] != float64(0) {
		t.Fatalf("explicit zero daily limit was not stored: %#v", afterZero)
	}
	if afterZero["login_max_failures"] != float64(19) || afterZero["api_rate_per_minute"] != float64(4321) {
		t.Fatalf("partial modern save changed omitted guards: %#v", afterZero)
	}

	invalidNull := `{"api_key_scopes":["notes:read"],"login_max_failures":null}`
	if response := save(invalidNull); response.Code != http.StatusBadRequest {
		t.Fatalf("explicit null security save=%d body=%s", response.Code, response.Body.String())
	}
	afterInvalid := read()
	encodedBefore, _ := json.Marshal(afterZero)
	encodedAfter, _ := json.Marshal(afterInvalid)
	if string(encodedAfter) != string(encodedBefore) {
		t.Fatalf("rejected security save changed settings: before=%s after=%s", encodedBefore, encodedAfter)
	}
}
