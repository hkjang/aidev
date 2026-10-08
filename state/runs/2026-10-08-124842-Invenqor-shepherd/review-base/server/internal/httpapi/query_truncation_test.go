package httpapi

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/hkjang/invenqor/server/internal/storagetest"
)

type queryResult struct {
	Items []struct {
		Name string `json:"name"`
	} `json:"items"`
	Count      int   `json:"count"`
	Limit      int   `json:"limit"`
	Offset     int   `json:"offset"`
	Total      int64 `json:"total"`
	HasMore    bool  `json:"has_more"`
	NextOffset int   `json:"next_offset"`
	Truncated  bool  `json:"truncated"`
}

func executeQueryResult(
	t *testing.T,
	server *Server,
	cookie *http.Cookie,
	csrf string,
	body map[string]any,
) queryResult {
	t.Helper()
	response := performAuthenticatedJSON(
		t, server, http.MethodPost, "/api/v1/query/execute",
		body, cookie, csrf,
	)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", response.Code, response.Body.String())
	}
	var payload queryResult
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	return payload
}

// A Query DSL result is bounded and used to be returned as though it were the
// whole answer. An operator asking which hosts have gone quiet got the first
// page and a count that agreed with it, and there is no offset on this
// endpoint, so nothing anywhere said the question had more answers. The same
// endpoint is reachable with an API key, where a script has no console to
// notice the row count landing exactly on the limit it asked for.
func TestQueryExecuteSaysWhenTheLimitCutTheResultShort(t *testing.T) {
	runtime := storagetest.Open(t)
	defer runtime.Close()
	server := testServer(t, runtime)
	cookie, csrf := authenticateInitialAdmin(t, server, runtime)

	for index := 0; index < 5; index++ {
		insertAssetSeenAt(
			t, server, "host-"+string(rune('a'+index)),
			time.Now().UTC().Add(-time.Duration(index)*time.Minute),
		)
	}

	cut := executeQueryResult(t, server, cookie, csrf, map[string]any{
		"query": `type = "host"`, "limit": 3,
	})
	if len(cut.Items) != 3 || cut.Count != 3 {
		t.Fatalf("limit=3 returned %d items, count %d, want 3 and 3",
			len(cut.Items), cut.Count)
	}
	if !cut.Truncated {
		t.Errorf("a result missing rows was not marked truncated")
	}
	if cut.Limit != 3 {
		t.Errorf("limit = %d, want 3", cut.Limit)
	}

	// A result that ends exactly on the limit is complete. Reporting it as
	// truncated would teach an operator to ignore the flag.
	exact := executeQueryResult(t, server, cookie, csrf, map[string]any{
		"query": `type = "host"`, "limit": 5,
	})
	if len(exact.Items) != 5 || exact.Truncated {
		t.Fatalf("limit=5 returned %d items, truncated = %v, want 5 and false",
			len(exact.Items), exact.Truncated)
	}

	// The default bound applies when the caller names none, and it has to be
	// reported as the one that was enforced.
	whole := executeQueryResult(t, server, cookie, csrf, map[string]any{
		"query": `type = "host"`,
	})
	if len(whole.Items) != 5 || whole.Truncated || whole.Limit != 100 {
		t.Fatalf("unbounded query returned %d items, truncated = %v, limit = %d",
			len(whole.Items), whole.Truncated, whole.Limit)
	}

	// The audit entry is the record of what an operator was actually shown, so
	// a partial answer must not be filed as though it were the whole inventory.
	rows, err := runtime.DB().Query(
		`SELECT after_json FROM audit_logs WHERE action = 'query.execute'`,
	)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	// The three runs are told apart by the rows they returned rather than by
	// their timestamps, which are written from the same clock and can tie.
	recorded := map[int]bool{}
	for rows.Next() {
		var after string
		if err := rows.Scan(&after); err != nil {
			t.Fatal(err)
		}
		var entry struct {
			ResultCount int  `json:"result_count"`
			Truncated   bool `json:"truncated"`
		}
		if err := json.Unmarshal([]byte(after), &entry); err != nil {
			t.Fatal(err)
		}
		if seen, ok := recorded[entry.ResultCount]; ok && seen != entry.Truncated {
			t.Fatalf("two entries returning %d rows disagree on truncated",
				entry.ResultCount)
		}
		recorded[entry.ResultCount] = entry.Truncated
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if !recorded[3] {
		t.Errorf("the partial run was filed as a whole answer: %v", recorded)
	}
	if recorded[5] {
		t.Errorf("a complete run was filed as truncated: %v", recorded)
	}
	if len(recorded) != 2 {
		t.Fatalf("audit recorded %v, want entries for 3 and 5 rows", recorded)
	}
}
