package httpapi

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hkjang/invenqor/server/internal/storagetest"

	"github.com/google/uuid"
	"github.com/hkjang/invenqor/server/internal/auth"
	"github.com/hkjang/invenqor/server/internal/bootstrap"
	"github.com/hkjang/invenqor/server/internal/storage"
)

func TestAutomaticAgentEnrollmentSupportsOpenAndProtectedModes(t *testing.T) {
	runtime := storagetest.Open(t)
	defer runtime.Close()
	server := testServer(t, runtime)
	openBody := map[string]string{
		"agent_id":    uuid.NewString(),
		"hostname":    "zero-touch-host",
		"claim_token": "ivq_ec_0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
	}
	open := performJSON(
		t, server, http.MethodPost, "/v1/agent/enroll", openBody, nil,
	)
	if open.Code != http.StatusCreated {
		t.Fatalf(
			"open enrollment status = %d body = %s",
			open.Code,
			open.Body.String(),
		)
	}

	fleetToken := "ivq_et_0123456789abcdef0123456789abcdef0123456789abcdef"
	fleetHash := sha256.Sum256([]byte(fleetToken))
	if _, _, err := server.updateAgentEnrollmentPolicy(
		context.Background(),
		"test",
		func(policy *agentEnrollmentPolicy) error {
			policy.Enabled = true
			policy.RequireToken = true
			policy.TokenHash = hex.EncodeToString(fleetHash[:])
			return nil
		},
	); err != nil {
		t.Fatal(err)
	}
	protectedBody := map[string]string{
		"agent_id":    uuid.NewString(),
		"hostname":    "protected-host",
		"claim_token": "ivq_ec_abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789",
	}
	unauthorized := performJSON(
		t,
		server,
		http.MethodPost,
		"/v1/agent/enroll",
		protectedBody,
		map[string]string{"X-Invenqor-Enrollment-Token": "wrong"},
	)
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized enrollment status = %d", unauthorized.Code)
	}
	enrolled := performJSON(
		t,
		server,
		http.MethodPost,
		"/v1/agent/enroll",
		protectedBody,
		map[string]string{"X-Invenqor-Enrollment-Token": fleetToken},
	)
	if enrolled.Code != http.StatusCreated {
		t.Fatalf(
			"enrollment status = %d body = %s",
			enrolled.Code,
			enrolled.Body.String(),
		)
	}
	var result struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(enrolled.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(result.Token, "ivq_at_") {
		t.Fatal("enrollment response omitted a device bearer token")
	}

	if _, _, err := server.updateAgentEnrollmentPolicy(
		context.Background(),
		"test",
		func(policy *agentEnrollmentPolicy) error {
			policy.Enabled = false
			return nil
		},
	); err != nil {
		t.Fatal(err)
	}
	disabledBody := map[string]string{
		"agent_id":    uuid.NewString(),
		"hostname":    "disabled-host",
		"claim_token": "ivq_ec_1111111111111111111111111111111111111111111111111111111111111111",
	}
	disabled := performJSON(
		t, server, http.MethodPost, "/v1/agent/enroll", disabledBody, nil,
	)
	if disabled.Code != http.StatusForbidden {
		t.Fatalf("disabled enrollment status = %d", disabled.Code)
	}
}

func TestDashboardStatisticsUseAuthoritativeTotals(t *testing.T) {
	runtime := storagetest.Open(t)
	defer runtime.Close()
	now := time.Now().UTC()
	agentInternalID := uuid.NewString()
	if _, err := runtime.DB().Exec(
		`INSERT INTO agents(
		 id,agent_id,hostname,status,last_seen_at
		 ) VALUES($1,$2,'stats-host','active',$3)`,
		agentInternalID, uuid.NewString(), now,
	); err != nil {
		t.Fatal(err)
	}
	if _, err := runtime.DB().Exec(
		`INSERT INTO assets(
		 id,asset_key,name,type,status,criticality,environment,source,
		 first_seen_at,last_seen_at
		 ) VALUES($1,$2,'stats-host','host','active','critical',
		 'production','agent',$3,$3)`,
		uuid.NewString(), uuid.NewString(), now,
	); err != nil {
		t.Fatal(err)
	}
	if _, err := runtime.DB().Exec(
		`INSERT INTO agent_events(
		 id,agent_id,event_id,schema_version,kind,raw_event,
		 received_at,processing_status
		 ) VALUES($1,$2,$3,1,'heartbeat','{}',$4,'processed')`,
		uuid.NewString(), agentInternalID, uuid.NewString(), now,
	); err != nil {
		t.Fatal(err)
	}
	server := testServer(t, runtime)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/statistics", nil)
	response := httptest.NewRecorder()
	server.dashboardStatistics(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("statistics status = %d body = %s", response.Code, response.Body.String())
	}
	var payload struct {
		Assets struct {
			Total int64 `json:"total"`
		} `json:"assets"`
		Agents struct {
			Healthy int64 `json:"healthy"`
		} `json:"agents"`
		Collection struct {
			Events24h int64 `json:"events_24h"`
			Daily     []struct {
				Date   string `json:"date"`
				Events int64  `json:"events"`
				Failed int64  `json:"failed"`
			} `json:"daily"`
		} `json:"collection"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Assets.Total != 1 || payload.Agents.Healthy != 1 ||
		payload.Collection.Events24h != 1 || len(payload.Collection.Daily) != 7 {
		t.Fatalf("unexpected statistics: %+v", payload)
	}
	// A seven-slot array of zeros used to satisfy this test while the trend chart
	// was empty in SQLite mode, so assert the event actually lands on its day.
	var plotted int64
	for _, day := range payload.Collection.Daily {
		plotted += day.Events
		if day.Date == "" {
			t.Fatalf("daily bucket without a date: %+v", payload.Collection.Daily)
		}
	}
	if plotted != 1 {
		t.Fatalf(
			"daily series plotted %d events, want 1: %+v",
			plotted, payload.Collection.Daily,
		)
	}
	if payload.Collection.Daily[6].Events != 1 {
		t.Fatalf("today's bucket = %+v", payload.Collection.Daily[6])
	}
}

func TestHealthEndpointsReportSQLiteFallback(t *testing.T) {
	runtime := storagetest.OpenSQLite(t)
	defer runtime.Close()
	server := testServer(t, runtime)

	for _, path := range []string{"/health/live", "/health/ready"} {
		request := httptest.NewRequest(http.MethodGet, path, nil)
		response := httptest.NewRecorder()
		server.Handler().ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Errorf("%s status = %d, want 200", path, response.Code)
		}
	}

	unauthenticated := httptest.NewRequest(http.MethodGet, "/health/database", nil)
	unauthenticatedResponse := httptest.NewRecorder()
	server.Handler().ServeHTTP(unauthenticatedResponse, unauthenticated)
	if unauthenticatedResponse.Code != http.StatusUnauthorized {
		t.Fatalf(
			"unauthenticated database health status = %d, want 401",
			unauthenticatedResponse.Code,
		)
	}
	adminCookie, _ := authenticateInitialAdmin(t, server, runtime)
	request := httptest.NewRequest(http.MethodGet, "/health/database", nil)
	request.AddCookie(adminCookie)
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("database health status = %d body = %s", response.Code, response.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response error = %v", err)
	}
	if payload["mode"] != string(storage.ModeSQLiteFallback) {
		t.Fatalf("mode = %v, want %s", payload["mode"], storage.ModeSQLiteFallback)
	}
}

func TestPublicSystemInfoDoesNotExposeAdministrativeRuntimeDetails(t *testing.T) {
	runtime := storagetest.Open(t)
	defer runtime.Close()
	server := testServer(t, runtime)

	public := httptest.NewRecorder()
	server.Handler().ServeHTTP(
		public,
		httptest.NewRequest(http.MethodGet, "/api/v1/system/info", nil),
	)
	var publicPayload map[string]any
	if public.Code != http.StatusOK ||
		json.Unmarshal(public.Body.Bytes(), &publicPayload) != nil {
		t.Fatalf("public system info = %d/%s", public.Code, public.Body.String())
	}
	publicVersion, versionPresent := publicPayload["server_version"].(string)
	if !versionPresent || publicVersion == "" || publicPayload["database_mode"] != nil ||
		publicPayload["listen_address"] != nil || publicPayload["agent_enrollment_mode"] != nil {
		t.Fatalf("public system info exposed administrative details: %#v", publicPayload)
	}
	var accessLogs int
	if err := runtime.DB().QueryRow(
		// CAST because details_json is JSONB on PostgreSQL, which has no LIKE.
		`SELECT COUNT(*) FROM diagnostic_logs
		  WHERE event_code='HTTP_REQUEST'
		    AND CAST(details_json AS TEXT) LIKE '%/api/v1/system/info%'`,
	).Scan(&accessLogs); err != nil || accessLogs != 1 {
		t.Fatalf("persisted public system access logs = %d/%v, want 1", accessLogs, err)
	}

	adminCookie, _ := authenticateInitialAdmin(t, server, runtime)
	adminRequest := httptest.NewRequest(http.MethodGet, "/api/v1/admin/system/info", nil)
	adminRequest.AddCookie(adminCookie)
	admin := httptest.NewRecorder()
	server.Handler().ServeHTTP(admin, adminRequest)
	var adminPayload map[string]any
	if admin.Code != http.StatusOK ||
		json.Unmarshal(admin.Body.Bytes(), &adminPayload) != nil {
		t.Fatalf("admin system info = %d/%s", admin.Code, admin.Body.String())
	}
	// Whichever engine the suite is running against - the point is that the admin
	// endpoint reports the mode at all, where the public one must not.
	if adminPayload["database_mode"] != string(runtime.Mode()) ||
		adminPayload["agent_enrollment_mode"] == nil {
		t.Fatalf("admin system info omitted runtime details: %#v", adminPayload)
	}
}

func TestRequestLogRetentionSkipsNoiseButKeepsFailures(t *testing.T) {
	tests := []struct {
		path   string
		status int
		want   bool
	}{
		{"/health/live", http.StatusOK, false},
		{"/health/ready", http.StatusOK, false},
		{"/assets/index.js", http.StatusOK, false},
		{"/", http.StatusOK, false},
		{"/api/v1/assets", http.StatusOK, true},
		{"/v1/agent/events", http.StatusAccepted, true},
		{"/health/live", http.StatusServiceUnavailable, true},
		{"/assets/missing.js", http.StatusNotFound, true},
	}
	for _, test := range tests {
		if got := shouldPersistRequestLog(test.path, test.status); got != test.want {
			t.Errorf("shouldPersistRequestLog(%q, %d) = %t, want %t", test.path, test.status, got, test.want)
		}
	}
}

func TestExplicitInvalidPostgresDSNFailsClosedWithoutExposingPassword(t *testing.T) {
	secret := "do-not-expose-this"
	runtime, err := storage.Open(context.Background(), storage.Options{
		PostgresDSN: "postgres://user:" + secret + "@%zz/invenqor",
		SQLitePath:  filepath.Join(t.TempDir(), "invenqor.db"),
	})
	if runtime != nil || err == nil {
		if runtime != nil {
			_ = runtime.Close()
		}
		t.Fatalf("storage.Open() = %#v, %v, want fail-closed error", runtime, err)
	}
	if strings.Contains(err.Error(), secret) {
		t.Fatal("database startup error exposed the PostgreSQL password")
	}
	if !strings.Contains(err.Error(), "INVALID_DSN") {
		t.Fatalf("database startup error = %s, want INVALID_DSN", err)
	}
}

func TestSecurityHeadersAreApplied(t *testing.T) {
	runtime := storagetest.Open(t)
	defer runtime.Close()
	server := testServer(t, runtime)
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, request)
	for _, header := range []string{
		"Content-Security-Policy",
		"X-Content-Type-Options",
		"X-Frame-Options",
	} {
		if response.Header().Get(header) == "" {
			t.Errorf("%s header is missing", header)
		}
	}
}

// newRuntime is the name these tests have always used for the store.
func newRuntime(t *testing.T) *storage.Runtime {
	t.Helper()
	return storagetest.Open(t)
}

func testServer(t *testing.T, runtime *storage.Runtime) *Server {
	t.Helper()
	stateDir := storagetest.StateDir(t, runtime)
	bootstrapManager := auth.NewBootstrapManager(runtime.DB(), stateDir)
	if _, err := bootstrapManager.Ensure(context.Background()); err != nil {
		t.Fatalf("BootstrapManager.Ensure() error = %v", err)
	}
	bootstrapStore, err := bootstrap.Open(stateDir)
	if err != nil {
		t.Fatalf("bootstrap.Open() error = %v", err)
	}
	totpService := auth.NewTOTPService(runtime.DB(), bootstrapStore)
	authOptions := auth.DefaultServiceOptions()
	authOptions.TOTP = totpService
	authService, err := auth.NewService(runtime.DB(), authOptions)
	if err != nil {
		t.Fatalf("auth.NewService() error = %v", err)
	}
	oidcService := auth.NewOIDCService(runtime.DB(), bootstrapStore, authService)
	return New(Options{
		Database:            runtime,
		AuthService:         authService,
		OIDCService:         oidcService,
		TOTPService:         totpService,
		BootstrapManager:    bootstrapManager,
		BootstrapStore:      bootstrapStore,
		AgentAutoEnrollment: true,
		Logger:              slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
}
