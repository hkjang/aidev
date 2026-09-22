package server

import (
	"strings"
	"testing"
)

func TestPostgresSettingsRevisionCompareAndSwap(t *testing.T) {
	s, ts := integrationTestServer(t)
	c := newIntegrationTestClient(t, ts.URL)
	c.request("POST", "/api/v1/auth/login", map[string]any{"email": "admin@example.test", "password": "Integration-Test-Password-2026!"}, 200)
	initial := testJSONObject(t, c.request("GET", "/api/v1/admin/settings", nil, 200))
	rev := str(initial, "settings_revision")
	if len(rev) != 64 {
		t.Fatal("missing settings revision")
	}
	updated := testJSONObject(t, c.request("PUT", "/api/v1/admin/settings", map[string]any{"expected_settings_revision": rev, "settings_revision": rev, "feature_flags": map[string]any{"canvas": false}, "otel_auth_token": "test-otel-secret"}, 200))
	fresh := testJSONObject(t, c.request("GET", "/api/v1/admin/settings", nil, 200))
	if str(updated, "settings_revision") == rev || str(updated, "settings_revision") != str(fresh, "settings_revision") || str(fresh, "otel_auth_token") != "" {
		t.Fatal("revision canonicalization or secret redaction failed")
	}
	c.request("PUT", "/api/v1/admin/settings", map[string]any{"expected_settings_revision": rev, "feature_flags": map[string]any{"canvas": true}}, 409)
	c.request("PUT", "/api/v1/admin/settings", map[string]any{"expected_settings_revision": 1}, 400)
	c.request("PUT", "/api/v1/admin/settings", map[string]any{"site_name": "independent sparse setting"}, 200)
	var raw string
	if e := s.DB.QueryRow(t.Context(), "SELECT data::text FROM settings WHERE id=1").Scan(&raw); e != nil {
		t.Fatal(e)
	}
	if strings.Contains(raw, "settings_revision") || strings.Contains(raw, "test-otel-secret") || !strings.Contains(raw, `"canvas": false`) {
		t.Fatal("metadata persisted, secret plaintext, or false flag overwritten")
	}
}
