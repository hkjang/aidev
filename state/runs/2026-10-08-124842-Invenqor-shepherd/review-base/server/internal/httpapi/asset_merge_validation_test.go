package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

// openapi.yaml promises `400 Primary or secondary assets missing` for
// POST /api/v1/assets/merge, but the handler never looked the ids up. A
// secondary that is not an asset updated no rows and still answered 200 while
// writing a "merged" asset_changes row and a successful audit entry; a primary
// that is not an asset only surfaced as the asset_changes foreign key failing,
// i.e. a 500. Both are checked here through the real router.

// mergeAssetsResponse posts to the production merge route.
func mergeAssetsResponse(
	t *testing.T, server *Server, cookie *http.Cookie, csrf, primaryID string,
	secondaryIDs ...string,
) (int, string) {
	t.Helper()
	response := performAuthenticatedJSON(t, server, http.MethodPost,
		"/api/v1/assets/merge",
		map[string]any{
			"primary_id": primaryID, "secondary_ids": secondaryIDs,
			"reason": "duplicate host",
		}, cookie, csrf)
	var payload struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	_ = json.Unmarshal(response.Body.Bytes(), &payload)
	return response.Code, payload.Error.Code
}

func countRows(t *testing.T, server *Server, query string, args ...any) int {
	t.Helper()
	var count int
	if err := server.database.DB().QueryRow(query, args...).Scan(&count); err != nil {
		t.Fatalf("count %q: %v", query, err)
	}
	return count
}

// assertMergeWroteNothing checks the two tables a merge writes to, because a
// response code alone would not catch a rolled-back handler that had already
// recorded a successful merge.
func assertMergeWroteNothing(t *testing.T, server *Server) {
	t.Helper()
	if changes := countRows(t, server,
		`SELECT COUNT(*) FROM asset_changes WHERE change_type='merged'`,
	); changes != 0 {
		t.Fatalf("asset_changes 'merged' rows = %d, want 0", changes)
	}
	if entries := countRows(t, server,
		`SELECT COUNT(*) FROM audit_logs WHERE action='asset.merge'`,
	); entries != 0 {
		t.Fatalf("audit_logs 'asset.merge' rows = %d, want 0", entries)
	}
}

func assetStatus(t *testing.T, server *Server, assetID string) string {
	t.Helper()
	var status string
	if err := server.database.DB().QueryRow(
		`SELECT status FROM assets WHERE id=$1`, assetID,
	).Scan(&status); err != nil {
		t.Fatalf("read status of %s: %v", assetID, err)
	}
	return status
}

func TestMergeAssetsRejectsPrimaryThatIsNotAnAsset(t *testing.T) {
	runtime := newRuntime(t)
	server := testServer(t, runtime)
	cookie, csrf := authenticateInitialAdmin(t, server, runtime)
	now := time.Now().UTC()
	secondaryID := insertSoftwareTestAsset(t, server, "orphan-secondary",
		"orphan-secondary", "host", `{}`, 1, now)

	status, code := mergeAssetsResponse(t, server, cookie, csrf, uuid.NewString(),
		secondaryID)
	if status != http.StatusBadRequest || code != "INVALID_MERGE" {
		t.Fatalf("merge into a missing primary = %d %q, want 400 INVALID_MERGE",
			status, code)
	}
	if got := assetStatus(t, server, secondaryID); got != "active" {
		t.Fatalf("secondary status = %q, want it untouched (active)", got)
	}
	assertMergeWroteNothing(t, server)
}

// The whole request rolls back: a secondary that exists alongside one that does
// not stays unmerged, rather than half the request succeeding.
func TestMergeAssetsRejectsMissingSecondaryWithoutMergingTheOthers(t *testing.T) {
	runtime := newRuntime(t)
	server := testServer(t, runtime)
	cookie, csrf := authenticateInitialAdmin(t, server, runtime)
	now := time.Now().UTC()
	primaryID := insertSoftwareTestAsset(t, server, "keep-primary", "keep-primary",
		"host", `{}`, 1, now)
	realSecondaryID := insertSoftwareTestAsset(t, server, "keep-secondary",
		"keep-secondary", "host", `{}`, 1, now.Add(-time.Minute))

	status, code := mergeAssetsResponse(t, server, cookie, csrf, primaryID,
		realSecondaryID, uuid.NewString())
	if status != http.StatusBadRequest || code != "INVALID_MERGE" {
		t.Fatalf("merge with a missing secondary = %d %q, want 400 INVALID_MERGE",
			status, code)
	}
	if got := assetStatus(t, server, realSecondaryID); got != "active" {
		t.Fatalf("the existing secondary was merged anyway: status = %q", got)
	}
	if got := assetStatus(t, server, primaryID); got != "active" {
		t.Fatalf("primary status = %q, want active", got)
	}
	assertMergeWroteNothing(t, server)
}

// assets.id is a UUID column on PostgreSQL and TEXT on the SQLite fallback, so
// an id that is not a UUID would make the lookup itself fail on one dialect
// only. Both must answer the same code for the same input.
func TestMergeAssetsRejectsIdsThatAreNotUUIDs(t *testing.T) {
	runtime := newRuntime(t)
	server := testServer(t, runtime)
	cookie, csrf := authenticateInitialAdmin(t, server, runtime)
	now := time.Now().UTC()
	primaryID := insertSoftwareTestAsset(t, server, "shape-primary",
		"shape-primary", "host", `{}`, 1, now)

	status, code := mergeAssetsResponse(t, server, cookie, csrf, primaryID,
		"not-a-uuid")
	if status != http.StatusBadRequest || code != "INVALID_MERGE" {
		t.Fatalf("merge with a malformed secondary id = %d %q, want 400 INVALID_MERGE",
			status, code)
	}
	assertMergeWroteNothing(t, server)
}

// primary_id reaches assets.id and asset_sources.asset_id, UUID columns on
// PostgreSQL and TEXT on the SQLite fallback. Every spelling uuid.Parse accepts
// but openapi does not declare (`format: uuid`) has to be refused before any
// query runs: PostgreSQL rejects the urn: one with SQLSTATE 22P02, surfacing as
// a 500 openapi never promises, and silently folds the brace and unhyphenated
// ones so the merge went through on rows the fallback matched not at all.
// nonCanonicalSpellings is shared with the split tests in
// asset_split_validation_test.go.
func TestMergeAssetsRejectsNonCanonicalPrimaryID(t *testing.T) {
	for name, rewrite := range nonCanonicalSpellings {
		t.Run(name, func(t *testing.T) {
			runtime := newRuntime(t)
			server := testServer(t, runtime)
			cookie, csrf := authenticateInitialAdmin(t, server, runtime)
			now := time.Now().UTC()
			primaryID := insertSoftwareTestAsset(t, server, "shape-merge-primary",
				"shape-merge-primary", "host", `{}`, 1, now)
			secondaryID := insertSoftwareTestAsset(t, server, "shape-merge-secondary",
				"shape-merge-secondary", "host", `{}`, 1, now.Add(-time.Minute))

			// Rewriting an id that really is an asset is what makes this bite: on
			// PostgreSQL the brace and unhyphenated forms matched the real row.
			spelling := rewrite(primaryID)
			status, code := mergeAssetsResponse(t, server, cookie, csrf, spelling,
				secondaryID)
			if status != http.StatusBadRequest || code != "INVALID_MERGE" {
				t.Fatalf("merge into primary %q = %d %q, want 400 INVALID_MERGE",
					spelling, status, code)
			}
			if got := assetStatus(t, server, secondaryID); got != "active" {
				t.Fatalf("secondary status = %q, want it untouched (active)", got)
			}
			assertMergeWroteNothing(t, server)
		})
	}
}

// Same for every secondary_ids element: it names the asset whose sources move
// and whose status flips, so a folded spelling carried the merge out on
// PostgreSQL alone.
func TestMergeAssetsRejectsNonCanonicalSecondaryIDs(t *testing.T) {
	for name, rewrite := range nonCanonicalSpellings {
		t.Run(name, func(t *testing.T) {
			runtime := newRuntime(t)
			server := testServer(t, runtime)
			cookie, csrf := authenticateInitialAdmin(t, server, runtime)
			now := time.Now().UTC()
			primaryID := insertSoftwareTestAsset(t, server, "shape-sec-primary",
				"shape-sec-primary", "host", `{}`, 1, now)
			goodSecondaryID := insertSoftwareTestAsset(t, server, "shape-sec-good",
				"shape-sec-good", "host", `{}`, 1, now.Add(-time.Minute))
			mergeableID := insertSoftwareTestAsset(t, server, "shape-sec-mergeable",
				"shape-sec-mergeable", "host", `{}`, 1, now.Add(-2*time.Minute))

			spelling := rewrite(mergeableID)
			status, code := mergeAssetsResponse(t, server, cookie, csrf, primaryID,
				goodSecondaryID, spelling)
			if status != http.StatusBadRequest || code != "INVALID_MERGE" {
				t.Fatalf("merge with secondary %q = %d %q, want 400 INVALID_MERGE",
					spelling, status, code)
			}
			if got := assetStatus(t, server, goodSecondaryID); got != "active" {
				t.Fatalf("the well-formed secondary was merged anyway: status = %q", got)
			}
			if got := assetStatus(t, server, mergeableID); got != "active" {
				t.Fatalf("the asset named by %q was merged: status = %q", spelling, got)
			}
			assertMergeWroteNothing(t, server)
		})
	}
}

// Upper-case hex is the one spelling openapi declares, just not folded:
// PostgreSQL's uuid input folds it and the fallback's TEXT comparison does not.
// The handler has to fold it itself so the merge happens on both dialects, and
// so asset_changes.after_json records the canonical secondary - the MCP
// merged_into lookup (mcp.go) matches that list by exact text on both engines,
// so an upper-case id stored there merges the asset without ever pointing a
// client at its primary.
func TestMergeAssetsFoldsUpperCaseIDsToCanonicalForm(t *testing.T) {
	runtime := newRuntime(t)
	server := testServer(t, runtime)
	cookie, csrf := authenticateInitialAdmin(t, server, runtime)
	now := time.Now().UTC()
	primaryID := insertSoftwareTestAsset(t, server, "upper-merge-primary",
		"upper-merge-primary", "host", `{}`, 1, now)
	secondaryID := insertSoftwareTestAsset(t, server, "upper-merge-secondary",
		"upper-merge-secondary", "host", `{}`, 1, now.Add(-time.Minute))
	movedSourceID := insertTestAssetSource(t, server, secondaryID, "upper-merge-source")

	upperSecondary := strings.ToUpper(secondaryID)
	status, code := mergeAssetsResponse(t, server, cookie, csrf,
		strings.ToUpper(primaryID), upperSecondary)
	if status != http.StatusOK {
		t.Fatalf("merge with upper-case ids = %d %q, want 200", status, code)
	}
	if got := assetStatus(t, server, secondaryID); got != "merged" {
		t.Fatalf("secondary status = %q, want merged", got)
	}
	if got := assetStatus(t, server, primaryID); got != "active" {
		t.Fatalf("primary status = %q, want active", got)
	}
	if got := sourceOwner(t, server, movedSourceID); got != primaryID {
		t.Fatalf("source owner = %q, want the canonical primary %q", got, primaryID)
	}
	if changes := countRows(t, server,
		`SELECT COUNT(*) FROM asset_changes WHERE change_type='merged' AND asset_id=$1`,
		primaryID,
	); changes != 1 {
		t.Fatalf("asset_changes 'merged' rows for the canonical primary = %d, want 1",
			changes)
	}
	if entries := countRows(t, server,
		`SELECT COUNT(*) FROM audit_logs WHERE action='asset.merge' AND resource_id=$1`,
		primaryID,
	); entries != 1 {
		t.Fatalf("audit_logs 'asset.merge' rows for the canonical primary = %d, want 1",
			entries)
	}
	var afterJSON string
	if err := server.database.DB().QueryRow(
		`SELECT after_json FROM asset_changes WHERE change_type='merged' AND asset_id=$1`,
		primaryID,
	).Scan(&afterJSON); err != nil {
		t.Fatalf("read the recorded merge of %s: %v", primaryID, err)
	}
	if !strings.Contains(afterJSON, secondaryID) || strings.Contains(afterJSON, upperSecondary) {
		t.Fatalf("asset_changes.after_json = %s, want the canonical %q and not %q",
			afterJSON, secondaryID, upperSecondary)
	}
}

// The point of recording the canonical secondary: MCP asset_get has to be able
// to tell a client holding the merged id where its asset went.
func TestMergeAssetsWithUpperCaseIDsStillAnswersMergedInto(t *testing.T) {
	runtime := newRuntime(t)
	server := testServer(t, runtime)
	cookie, csrf := authenticateInitialAdmin(t, server, runtime)
	now := time.Now().UTC()
	primaryID := insertSoftwareTestAsset(t, server, "hint-primary", "hint-primary",
		"host", `{}`, 1, now)
	secondaryID := insertSoftwareTestAsset(t, server, "hint-secondary",
		"hint-secondary", "host", `{}`, 1, now.Add(-time.Minute))

	status, code := mergeAssetsResponse(t, server, cookie, csrf,
		strings.ToUpper(primaryID), strings.ToUpper(secondaryID))
	if status != http.StatusOK {
		t.Fatalf("merge with upper-case ids = %d %q, want 200", status, code)
	}
	merged, err := mcpAssetGetReplyFor(t, server, secondaryID)
	if err != nil {
		t.Fatalf("asset_get on the merged asset returned an error: %v", err)
	}
	if merged.MergedInto != primaryID {
		t.Fatalf("merged_into = %q, want the primary %q", merged.MergedInto, primaryID)
	}
}
