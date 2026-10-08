package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func insertAssetWithID(t *testing.T, server *Server, id string, name string) {
	t.Helper()
	if _, err := server.database.DB().Exec(
		`INSERT INTO assets(
			id, asset_key, name, type, status, criticality, environment,
			confidence, attributes_json, custom_fields_json, source,
			first_seen_at, last_seen_at, created_at, updated_at
		 ) VALUES($1,$2,$3,'host','active','normal','other',
		          1.0,'{}','{}','manual',
		          CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,
		          CURRENT_TIMESTAMP)`,
		id, "identified-"+name, name,
	); err != nil {
		t.Fatal(err)
	}
}

// The id column holds a UUID, and a value that is not one can never name an
// asset. Handing it to the database gave neither mode a usable answer:
// PostgreSQL failed the statement on the type, so the operator saw HTTP 500
// with nothing naming the value, and the SQLite fallback compared it as text
// and answered 200 with an empty list as though the asset simply did not exist.
func TestIdentifierClauseThatIsNotAUUIDIsRejectedWithTheValue(t *testing.T) {
	runtime := newRuntime(t)
	server := testServer(t, runtime)
	cookie, csrf := authenticateInitialAdmin(t, server, runtime)

	for _, badValue := range []string{"web-01", "0d0f", "host:web-01"} {
		response := performAuthenticatedJSON(
			t, server, http.MethodPost, "/api/v1/query/execute",
			map[string]any{"query": `id = "` + badValue + `"`}, cookie, csrf,
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
		if payload.Error.Code != "INVALID_QUERY" ||
			!strings.Contains(payload.Error.Message, badValue) {
			t.Fatalf("%q error = %+v", badValue, payload.Error)
		}
	}

	// Validation has to give the same answer, or the console reports an
	// expression as writable and then fails it on execution.
	validate := performAuthenticatedJSON(
		t, server, http.MethodPost, "/api/v1/query/validate",
		map[string]any{"query": `id = "web-01"`}, cookie, csrf,
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
	if verdict.Valid || !strings.Contains(verdict.Error, "web-01") {
		t.Fatalf("validate verdict = %+v", verdict)
	}
}

// A UUID names the same asset however it is spelled. PostgreSQL reads the
// upper-cased spelling as the same value, so an operator who pasted an
// identifier out of a report that upper-cases it got the asset in production
// and an empty list from the SQLite fallback's text comparison.
func TestIdentifierClauseNamesTheAssetWhateverTheSpelling(t *testing.T) {
	runtime := newRuntime(t)
	server := testServer(t, runtime)
	cookie, csrf := authenticateInitialAdmin(t, server, runtime)

	id := uuid.NewString()
	insertAssetWithID(t, server, id, "wanted")
	insertAssetWithID(t, server, uuid.NewString(), "other")

	for _, spelling := range []string{
		id,
		strings.ToUpper(id),
		strings.ReplaceAll(id, "-", ""),
	} {
		found := executeQueryNames(
			t, server, cookie, csrf, `id = "`+spelling+`"`,
		)
		if len(found) != 1 || found[0] != "wanted" {
			t.Fatalf("id = %q returned %v", spelling, found)
		}
	}
}
