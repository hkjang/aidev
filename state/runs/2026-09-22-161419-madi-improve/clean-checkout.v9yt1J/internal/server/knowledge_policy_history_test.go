package server

import "testing"

func TestPostgresKnowledgePolicyHistoryCASRestoresNewRevision(t *testing.T) {
	_, c, _, _, _ := jobTestFixture(t)
	c.request("PUT", "/api/v1/admin/knowledge-packages/policy", map[string]any{"enabled": false, "retention_hours": 2, "token_counter": "responses", "allow_http": true, "version": 1}, 200)
	c.request("POST", "/api/v1/admin/knowledge-packages/policy/history/1/restore", map[string]any{"version": 2}, 400)
	got := testJSONObject(t, c.request("POST", "/api/v1/admin/knowledge-packages/policy/history/1/restore", map[string]any{"version": 2, "consent": true}, 200))
	if number(got, "version", 0) != 3 || !boolean(got, "enabled") || str(got, "token_counter") != "estimate" || boolean(got, "allow_http") || number(got, "retention_hours", 0) != 24 {
		t.Fatal(got)
	}
	c.request("POST", "/api/v1/admin/knowledge-packages/policy/history/1/restore", map[string]any{"version": 2, "consent": true}, 409)
	c.request("GET", "/api/v1/admin/knowledge-packages/policy/history", nil, 200)
	c.request("PUT", "/api/v1/admin/evidence-policy", map[string]any{"enabled": true, "retention_days": 1, "version": 1}, 200)
	got = testJSONObject(t, c.request("POST", "/api/v1/admin/evidence-policy/history/1/restore", map[string]any{"version": 2, "consent": true}, 200))
	if number(got, "version", 0) != 3 || boolean(got, "enabled") || number(got, "retention_days", 0) != 90 {
		t.Fatal(got)
	}
	c.request("POST", "/api/v1/admin/evidence-policy/history/1/restore", map[string]any{"version": 2, "consent": true}, 409)
}
