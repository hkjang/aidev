package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func insertAssetSeenAt(t *testing.T, server *Server, name string, seen time.Time) {
	t.Helper()
	if _, err := server.database.DB().Exec(
		`INSERT INTO assets(
			id, asset_key, name, type, status, criticality, environment,
			confidence, attributes_json, custom_fields_json, source,
			first_seen_at, last_seen_at, created_at, updated_at
		 ) VALUES($1,$2,$3,'host','active','normal','other',
		          1.0,'{}','{}','agent',$4,$4,$4,$4)`,
		uuid.NewString(), "seen-"+name, name, seen,
	); err != nil {
		t.Fatal(err)
	}
}

func executeQueryNames(
	t *testing.T, server *Server, cookie *http.Cookie, csrf string, dsl string,
) []string {
	t.Helper()
	response := performAuthenticatedJSON(
		t, server, http.MethodPost, "/api/v1/query/execute",
		map[string]any{"query": dsl}, cookie, csrf,
	)
	if response.Code != http.StatusOK {
		t.Fatalf("query %q status = %d body = %s",
			dsl, response.Code, response.Body.String())
	}
	var payload struct {
		Items []struct {
			Name string `json:"name"`
		} `json:"items"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	names := make([]string, 0, len(payload.Items))
	for _, item := range payload.Items {
		names = append(names, item.Name)
	}
	return names
}

// A query naming an absolute instant has to select by that instant in both
// storage modes. The value used to reach PostgreSQL as text, which failed the
// statement, and reached the SQLite fallback as a string comparison against a
// column written in a different layout, which returned rows on both sides of
// the bound.
func TestAbsoluteTimeClauseSelectsByTheInstant(t *testing.T) {
	runtime := newRuntime(t)
	server := testServer(t, runtime)
	cookie, csrf := authenticateInitialAdmin(t, server, runtime)

	bound := time.Date(2026, 1, 31, 9, 0, 0, 0, time.UTC)
	// Both sit on the bound's own day, where the SQLite fallback's text
	// comparison used to put every stored value on the same side of the bound:
	// "2026-01-31 06:00:00 +0000 UTC" against "2026-01-31T09:00:00Z" turns on
	// a space against a "T".
	insertAssetSeenAt(t, server, "before-the-bound", bound.Add(-3*time.Hour))
	insertAssetSeenAt(t, server, "after-the-bound", bound.Add(3*time.Hour))

	older := executeQueryNames(
		t, server, cookie, csrf, `last_seen_at < "2026-01-31T09:00:00Z"`,
	)
	if len(older) != 1 || older[0] != "before-the-bound" {
		t.Fatalf("older than the bound = %v", older)
	}
	newer := executeQueryNames(
		t, server, cookie, csrf, `last_seen_at >= "2026-01-31T09:00:00Z"`,
	)
	if len(newer) != 1 || newer[0] != "after-the-bound" {
		t.Fatalf("newer than the bound = %v", newer)
	}
}

// A time the server cannot read is the caller's mistake, and it has to be named
// back to the caller. Handing it to the database gave neither mode a usable
// answer: PostgreSQL failed the statement, so the operator saw HTTP 500 with
// nothing naming the value, and the SQLite fallback compared it as text and
// answered 200 with a row set nobody asked for.
func TestUnreadableTimeClauseIsRejectedWithTheValue(t *testing.T) {
	runtime := newRuntime(t)
	server := testServer(t, runtime)
	cookie, csrf := authenticateInitialAdmin(t, server, runtime)

	for _, badValue := range []string{"yesterday", "2026-13-45", "now - two days"} {
		response := performAuthenticatedJSON(
			t, server, http.MethodPost, "/api/v1/query/execute",
			map[string]any{"query": `last_seen_at < "` + badValue + `"`},
			cookie, csrf,
		)
		if response.Code != http.StatusBadRequest {
			t.Fatalf("%q status = %d body = %s",
				badValue, response.Code, response.Body.String())
		}
		var payload struct {
			Error struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
			t.Fatal(err)
		}
		if payload.Error.Code != "INVALID_QUERY" {
			t.Fatalf("%q error = %+v", badValue, payload.Error)
		}
	}

	// Validation has to give the same answer, or the console reports an
	// expression as writable and then fails it on execution.
	validate := performAuthenticatedJSON(
		t, server, http.MethodPost, "/api/v1/query/validate",
		map[string]any{"query": `last_seen_at < "yesterday"`}, cookie, csrf,
	)
	if validate.Code != http.StatusOK {
		t.Fatalf("validate status = %d body = %s",
			validate.Code, validate.Body.String())
	}
	var verdict struct {
		Valid bool   `json:"valid"`
		Error string `json:"error"`
	}
	if err := json.Unmarshal(validate.Body.Bytes(), &verdict); err != nil {
		t.Fatal(err)
	}
	if verdict.Valid || !strings.Contains(verdict.Error, "yesterday") {
		t.Fatalf("validate verdict = %+v", verdict)
	}
}
