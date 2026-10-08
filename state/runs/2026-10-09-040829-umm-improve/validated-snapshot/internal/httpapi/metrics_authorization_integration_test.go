package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/hkjang/umm/internal/auth"
	"github.com/hkjang/umm/internal/observability"
)

func TestPrometheusMetricsRequiresAdminSessionOrDedicatedKeyIntegration(t *testing.T) {
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		t.Skip("POSTGRES_DSN is not configured")
	}
	ctx := context.Background()
	db := isolatedHTTPStore(t, dsn)
	userID, adminID := uuid.New(), uuid.New()
	userName := "metrics_user_" + strings.ReplaceAll(userID.String(), "-", "")
	adminName := "metrics_admin_" + strings.ReplaceAll(adminID.String(), "-", "")
	if _, err := db.Pool.Exec(ctx, `
		INSERT INTO users(id,username,display_name,role) VALUES
		($1,$2::citext,$2::text,'user'),($3,$4::citext,$4::text,'admin')`,
		userID, userName, adminID, adminName); err != nil {
		t.Fatal(err)
	}
	authService := &auth.Service{Store: db}
	userSession, err := authService.CreateSession(ctx, userID, auth.SessionOrigin{UserAgent: "metrics-test", ClientIP: "127.0.0.1"})
	if err != nil {
		t.Fatal(err)
	}
	adminSession, err := authService.CreateSession(ctx, adminID, auth.SessionOrigin{UserAgent: "metrics-test", ClientIP: "127.0.0.1"})
	if err != nil {
		t.Fatal(err)
	}
	_, metricsKey, err := authService.CreateKey(ctx, userID, "metrics", []string{"metrics:read"}, 1)
	if err != nil {
		t.Fatal(err)
	}
	_, notesKey, err := authService.CreateKey(ctx, userID, "notes", []string{"notes:read"}, 1)
	if err != nil {
		t.Fatal(err)
	}
	_, adminNotesKey, err := authService.CreateKey(ctx, adminID, "admin notes", []string{"notes:read"}, 1)
	if err != nil {
		t.Fatal(err)
	}

	server := &Server{Store: db, Metrics: observability.NewRegistry(), Version: "metrics-test"}
	handler := authService.Middleware(auth.Require(http.HandlerFunc(server.prometheusMetrics)))
	request := func(session, key string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/metrics", nil)
		if session != "" {
			req.AddCookie(&http.Cookie{Name: auth.CookieName, Value: session})
		}
		if key != "" {
			req.Header.Set("Authorization", "Bearer "+key)
		}
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, req)
		return response
	}

	for _, test := range []struct {
		name       string
		session    string
		key        string
		wantStatus int
	}{
		{name: "regular wildcard session", session: userSession, wantStatus: http.StatusForbidden},
		{name: "administrator session", session: adminSession, wantStatus: http.StatusOK},
		{name: "metrics key", key: metricsKey, wantStatus: http.StatusOK},
		{name: "ordinary key", key: notesKey, wantStatus: http.StatusForbidden},
		{name: "admin ordinary key", key: adminNotesKey, wantStatus: http.StatusForbidden},
	} {
		t.Run(test.name, func(t *testing.T) {
			response := request(test.session, test.key)
			if response.Code != test.wantStatus {
				t.Fatalf("status=%d want=%d body=%s", response.Code, test.wantStatus, response.Body.String())
			}
			if test.wantStatus == http.StatusOK && !strings.Contains(response.Body.String(), "umm_build_info") {
				t.Fatalf("successful response omitted Prometheus metrics: %s", response.Body.String())
			}
		})
	}
}
