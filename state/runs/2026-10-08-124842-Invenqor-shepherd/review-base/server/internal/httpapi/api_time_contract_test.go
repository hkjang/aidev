package httpapi

import (
	"encoding/json"
	"net/http"
	"regexp"
	"testing"
	"time"

	"github.com/hkjang/invenqor/server/internal/storagetest"
)

// Every `*_at` field the console reads must be RFC 3339 with an explicit zone.
// The SQLite fallback used to hand out "2026-07-29 19:17:12", which a browser
// reads as local time - so audit entries, logins and key rotations all appeared
// shifted by the viewer's UTC offset - and Go's "… +0000 UTC" layout, which no
// specification obliges a browser to parse.
//
// This walks real responses rather than trusting the field types, because a new
// endpoint that builds a map by hand is exactly how the problem returns.
func TestAPIResponsesAlwaysCarryZonedTimestamps(t *testing.T) {
	runtime := storagetest.Open(t)
	defer runtime.Close()
	server := testServer(t, runtime)
	cookie, csrf := authenticateInitialAdmin(t, server, runtime)

	// Create rows through the API so the timestamps are written the way the
	// running server writes them.
	if response := performAuthenticatedJSON(
		t, server, http.MethodPost, "/api/v1/admin/api-keys",
		map[string]any{"name": "timestamp-probe", "scopes": []string{"assets.read"}},
		cookie, csrf,
	); response.Code != http.StatusCreated {
		t.Fatalf("create api key status = %d body = %s", response.Code, response.Body)
	}
	if response := performAuthenticatedJSON(
		t, server, http.MethodPost, "/api/v1/admin/agents",
		map[string]any{
			"agent_id": "11111111-2222-3333-4444-555555555555",
			"hostname": "timestamp-probe",
		},
		cookie, csrf,
	); response.Code != http.StatusCreated {
		t.Fatalf("provision agent status = %d body = %s", response.Code, response.Body)
	}
	if response := performAuthenticatedJSON(
		t, server, http.MethodPost, "/api/v1/assets",
		map[string]any{
			"name": "timestamp-probe-host", "type": "host",
			"reason": "timestamp contract",
		},
		cookie, csrf,
	); response.Code != http.StatusCreated {
		t.Fatalf("create asset status = %d body = %s", response.Code, response.Body)
	}

	paths := []string{
		"/api/v1/admin/api-keys",
		"/api/v1/admin/agents",
		"/api/v1/admin/users",
		"/api/v1/admin/audit?limit=50",
		"/api/v1/admin/diagnostics/logs",
		"/api/v1/admin/diagnostics/enrollment",
		"/api/v1/admin/settings",
		"/api/v1/admin/settings/history",
		"/api/v1/admin/settings/agent-enrollment",
		"/api/v1/admin/settings/classification",
		"/api/v1/assets?limit=50",
		"/api/v1/assets/software-products?limit=50",
		"/api/v1/dashboard/statistics",
		"/api/v1/auth/me",
	}
	for _, path := range paths {
		response := performAuthenticatedJSON(
			t, server, http.MethodGet, path, nil, cookie, csrf,
		)
		if response.Code != http.StatusOK {
			t.Fatalf("GET %s status = %d body = %s", path, response.Code, response.Body)
		}
		var payload any
		if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
			t.Fatalf("GET %s: decode body: %v", path, err)
		}
		for _, finding := range unzonedTimestamps(payload, path) {
			t.Errorf("%s is not a zoned RFC 3339 timestamp", finding)
		}
	}
}

var timestampField = regexp.MustCompile(`(_at|_time)$`)

// unzonedTimestamps walks a decoded response and reports every field whose name
// marks it as a timestamp but whose value a browser cannot read unambiguously.
func unzonedTimestamps(value any, path string) []string {
	var findings []string
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			childPath := path + "." + key
			text, isText := child.(string)
			if timestampField.MatchString(key) && isText && text != "" {
				if _, err := time.Parse(time.RFC3339, text); err != nil {
					findings = append(
						findings,
						childPath+" = "+text,
					)
				}
				continue
			}
			findings = append(findings, unzonedTimestamps(child, childPath)...)
		}
	case []any:
		for index, child := range typed {
			findings = append(
				findings,
				unzonedTimestamps(child, path+"["+itoa(index)+"]")...,
			)
		}
	}
	return findings
}
