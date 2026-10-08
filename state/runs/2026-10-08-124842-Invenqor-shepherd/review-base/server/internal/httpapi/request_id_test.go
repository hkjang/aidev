package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// The request ID is the audit trail's correlation identifier. It is written
// into every audit record and diagnostic event, the console filters by it, and
// this product tells an operator to search the diagnostic log for the one it
// showed them.
//
// chi's RequestID takes an inbound X-Request-Id verbatim when one is present,
// so without this the subject of an audit record chooses its own identifier -
// and can set it to one already used by an administrator's action, so that
// filtering by that identifier returns both. Nothing needs the inbound form:
// the Agent and the console only read this header off a response.
func TestAClientCannotChooseTheRequestIDRecordedAgainstIt(t *testing.T) {
	runtime := newRuntime(t)
	server := testServer(t, runtime)

	forged := "forged-by-the-caller-000001"
	request := httptest.NewRequest(http.MethodGet, "/health/live", nil)
	request.Header.Set("X-Request-Id", forged)
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, request)

	assigned := response.Header().Get("X-Request-Id")
	if assigned == "" {
		t.Fatal("no request ID was assigned, so nothing correlates")
	}
	if assigned == forged || strings.Contains(assigned, "forged") {
		t.Fatalf(
			"the caller's own identifier was adopted and recorded: %q",
			assigned,
		)
	}
}

// Two requests must be told apart, or correlation is worthless.
func TestEachRequestGetsItsOwnIdentifier(t *testing.T) {
	runtime := newRuntime(t)
	server := testServer(t, runtime)

	seen := map[string]bool{}
	for range 3 {
		response := httptest.NewRecorder()
		server.Handler().ServeHTTP(
			response,
			httptest.NewRequest(http.MethodGet, "/health/live", nil),
		)
		id := response.Header().Get("X-Request-Id")
		if id == "" || seen[id] {
			t.Fatalf("request identifier %q is missing or repeated", id)
		}
		seen[id] = true
	}
}
