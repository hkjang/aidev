package httpapi

import (
	"encoding/csv"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/hkjang/invenqor/server/internal/storagetest"
)

// Both CSV exports read a bounded number of rows and used to write whatever
// they got as if it were the whole answer. An inventory of 12,000 hosts came
// back as a 10,000-row file called invenqor-assets.csv, and nothing in it - not
// a count, not the name, not a header - said the other 2,000 were missing. The
// file is asked for precisely so it can be read away from the console, where
// there is nothing left to compare it against.
func TestCSVExportsSayWhenTheRowLimitCutThemShort(t *testing.T) {
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
	// The export writes an audit entry of its own, so the audit rows this test
	// counts are picked out by an action no export uses.
	for index := 0; index < 5; index++ {
		insertAuditEvent(
			t, runtime, "asset.update", "recent.actor", "success",
			time.Now().UTC().Add(-time.Duration(index)*time.Minute),
		)
	}

	for _, export := range []struct {
		name string
		path string
		file string
	}{
		{"assets", "/api/v1/assets.csv?", "invenqor-assets"},
		{
			"audit", "/api/v1/admin/audit.csv?action=asset.update&",
			"invenqor-audit",
		},
	} {
		t.Run(export.name, func(t *testing.T) {
			cut := exportCSV(t, server, cookie, csrf, export.path+"limit=3")
			if len(cut.rows) != 3 {
				t.Fatalf("limit=3 wrote %d rows, want 3", len(cut.rows))
			}
			if cut.header("X-Invenqor-Truncated") != "true" {
				t.Errorf(
					"an extract missing rows was not marked truncated: %q",
					cut.header("X-Invenqor-Truncated"),
				)
			}
			if cut.header("X-Invenqor-Row-Limit") != "3" {
				t.Errorf(
					"row limit header = %q, want 3",
					cut.header("X-Invenqor-Row-Limit"),
				)
			}
			// The name is what a reader still has once the headers are gone.
			if want := export.file + "-partial.csv"; !strings.Contains(
				cut.header("Content-Disposition"), want,
			) {
				t.Errorf(
					"a truncated extract was named %q, want %s",
					cut.header("Content-Disposition"), want,
				)
			}

			// A result that ends exactly on the limit is complete. Reporting it
			// as cut short would send an operator hunting for rows that are
			// already in the file.
			exact := exportCSV(t, server, cookie, csrf, export.path+"limit=5")
			if len(exact.rows) != 5 {
				t.Fatalf("limit=5 wrote %d rows, want 5", len(exact.rows))
			}
			if exact.header("X-Invenqor-Truncated") != "" {
				t.Errorf("a complete extract was marked truncated")
			}
			if want := export.file + ".csv"; !strings.Contains(
				exact.header("Content-Disposition"), want,
			) {
				t.Errorf(
					"a complete extract was named %q, want %s",
					exact.header("Content-Disposition"), want,
				)
			}
		})
	}

	// The audit trail of an incomplete extract has to record that it was
	// incomplete, or the entry stands as evidence that the whole log was taken.
	rows, err := runtime.DB().Query(
		`SELECT after_json FROM audit_logs WHERE action='audit.export'
		 ORDER BY occurred_at`,
	)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	recorded := make([]bool, 0, 2)
	for rows.Next() {
		var after string
		if err := rows.Scan(&after); err != nil {
			t.Fatal(err)
		}
		var entry struct {
			Truncated bool `json:"truncated"`
			Rows      int  `json:"rows"`
		}
		if err := json.Unmarshal([]byte(after), &entry); err != nil {
			t.Fatalf("audit metadata %q: %v", after, err)
		}
		if entry.Rows == 0 {
			t.Errorf("audit entry recorded no rows: %s", after)
		}
		recorded = append(recorded, entry.Truncated)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if len(recorded) != 2 || !recorded[0] || recorded[1] {
		t.Errorf(
			"audit entries recorded truncation as %v, want [true false]",
			recorded,
		)
	}
}

type csvExport struct {
	rows     [][]string
	recorder *httptest.ResponseRecorder
}

func (export csvExport) header(name string) string {
	return export.recorder.Result().Header.Get(name)
}

// exportCSV fetches an export and returns its data rows, without the header
// row and without the byte order mark that precedes it.
func exportCSV(
	t *testing.T,
	server *Server,
	cookie *http.Cookie,
	csrf string,
	path string,
) csvExport {
	t.Helper()
	recorder := performAuthenticatedJSON(
		t, server, http.MethodGet, path, nil, cookie, csrf,
	)
	if recorder.Code != http.StatusOK {
		t.Fatalf("%s status = %d body = %s", path, recorder.Code, recorder.Body)
	}
	records, err := csv.NewReader(
		strings.NewReader(strings.TrimPrefix(recorder.Body.String(), "\ufeff")),
	).ReadAll()
	if err != nil {
		t.Fatalf("%s wrote unreadable CSV: %v", path, err)
	}
	if len(records) == 0 {
		t.Fatalf("%s wrote no header row", path)
	}
	return csvExport{rows: records[1:], recorder: recorder}
}
