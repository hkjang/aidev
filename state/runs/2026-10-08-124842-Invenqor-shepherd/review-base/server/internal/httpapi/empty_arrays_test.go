package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// A fresh installation is the first thing anyone sees, and it is the state with
// the least test coverage everywhere.
//
// Go marshals a nil slice as null, not []. The console maps over these arrays
// without guarding - data.matrix.cells.map(...), data.freshness.map(...) - so a
// single nil slice throws inside render and the whole page goes blank. There is
// no error message; the user sees nothing and has nothing to report.
//
// This walks the real responses of the views a new installation lands on and
// fails on any null where an array belongs, naming the path.
func TestEmptyInstallationReturnsArraysNotNull(t *testing.T) {
	runtime := newRuntime(t)
	server := testServer(t, runtime)
	cookie, _ := authenticateInitialAdmin(t, server, runtime)

	for _, path := range []string{
		"/api/v1/assets/visualization?days=14",
		"/api/v1/assets/visualization?days=1",
		"/api/v1/assets/software-products",
		"/api/v1/assets",
		"/api/v1/admin/agents",
		// The console calls .slice() straight onto these two without a guard, so
		// a nil here throws in render rather than showing an empty section.
		"/api/v1/admin/agent-updates",
		"/api/v1/admin/diagnostics/enrollment",
	} {
		request := httptest.NewRequest(http.MethodGet, path, nil)
		request.AddCookie(cookie)
		response := httptest.NewRecorder()
		server.Handler().ServeHTTP(response, request)
		// A test Server has no update directory, and the console handles that
		// answer with a catch. Anything else that is not 200 is a real problem.
		if response.Code == http.StatusServiceUnavailable &&
			strings.Contains(response.Body.String(), "UPDATE_STORE_UNAVAILABLE") {
			continue
		}
		if response.Code != http.StatusOK {
			t.Fatalf("%s = %d body = %s", path, response.Code, response.Body.String())
		}
		var decoded any
		if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil {
			t.Fatalf("%s returned undecodable JSON: %v", path, err)
		}
		for _, where := range nullsUnderPluralKeys(decoded, "") {
			t.Errorf(
				"%s returned null at %s where the console maps over an array;\n"+
					"rendering that throws and the page goes blank with no message",
				path, where,
			)
		}
	}
}

// nullsUnderPluralKeys reports every null reached by a key that names a
// collection. Naming the key rather than inspecting a schema keeps this honest
// about what it checks: it cannot know every array, but a null under "cells",
// "rows", "items" or "edges" is one the console will map over.
func nullsUnderPluralKeys(value any, path string) []string {
	var found []string
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			at := path + "/" + key
			if child == nil && looksLikeCollection(key) {
				found = append(found, at)
				continue
			}
			found = append(found, nullsUnderPluralKeys(child, at)...)
		}
	case []any:
		for index, child := range typed {
			found = append(found, nullsUnderPluralKeys(child, path+"/[]")...)
			_ = index
		}
	}
	return found
}

func looksLikeCollection(key string) bool {
	for _, name := range []string{
		"cells", "rows", "columns", "items", "edges", "nodes", "children",
		"buckets", "series", "points", "records", "errors", "evidence",
		"agents", "assets", "products", "history", "roles", "scopes",
		"releases", "agent_versions", "awaiting_inventory", "by_event_code",
	} {
		if key == name {
			return true
		}
	}
	return strings.HasSuffix(key, "_list")
}
