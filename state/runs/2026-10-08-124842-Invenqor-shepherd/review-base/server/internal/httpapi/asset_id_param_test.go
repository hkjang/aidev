package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"
)

// openapi.yaml declares {assetId} as `format: uuid` and promises 404 as the only
// failure for GET, PATCH, DELETE and restore on /api/v1/assets/{assetId}, while
// the history and relations collections promise 200 and nothing else. The five
// handlers put chi.URLParam(request, "assetID") straight into `WHERE id = $1`.
// assets.id, asset_changes.asset_id and asset_relations.source/target_asset_id
// are UUID columns on PostgreSQL and TEXT on the SQLite fallback, so the same
// request diverged by dialect: a spelling PostgreSQL's uuid input rejects failed
// the query with SQLSTATE 22P02 and surfaced as a 500 openapi never declares,
// and an upper-case id — the declared form — was folded by PostgreSQL into a hit
// where SQLite's comparison matched nothing and answered 404.
//
// These tests run through the real router against the real Runtime, and look at
// the rows the handlers write rather than at the response code alone.

// assetIDResponse calls one of the five routes for the given raw path parameter
// and reports the status with the error code, if any.
func assetIDResponse(
	t *testing.T, server *Server, cookie *http.Cookie, csrf, method, path string,
	body any,
) (int, string, string) {
	t.Helper()
	response := performAuthenticatedJSON(t, server, method, path, body, cookie, csrf)
	var payload struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	_ = json.Unmarshal(response.Body.Bytes(), &payload)
	return response.Code, payload.Error.Code, response.Body.String()
}

// assetIDSpellings are the path parameters uuid.Parse accepts, or does not,
// without being the 36-character hyphenated form openapi declares. The first is
// not a UUID in any reading; the rest are alternative spellings of an id that
// does name an asset, and are the ones whose handling diverged by dialect.
func assetIDSpellings(canonical string) map[string]string {
	return map[string]string{
		"not a uuid at all": "not-a-uuid",
		"urn prefixed":      "urn:uuid:" + canonical,
		"brace wrapped":     "{" + canonical + "}",
		"unhyphenated":      strings.ReplaceAll(canonical, "-", ""),
	}
}

// assetRowState reads back the two columns a delete and a restore move, so a
// handler that answered 404 after having already written is still caught.
func assetRowState(t *testing.T, server *Server, id string) (string, bool) {
	t.Helper()
	var status string
	var deletedAt any
	if err := server.database.DB().QueryRow(
		`SELECT status, deleted_at FROM assets WHERE id=$1`, id,
	).Scan(&status, &deletedAt); err != nil {
		t.Fatalf("read asset %s: %v", id, err)
	}
	return status, deletedAt != nil
}

// A path parameter that is not the declared canonical UUID has to be refused
// before any query runs, with the 404 openapi declares, on both dialects.
func TestAssetRoutesAnswer404ForNonCanonicalAssetID(t *testing.T) {
	runtime := newRuntime(t)
	server := testServer(t, runtime)
	cookie, csrf := authenticateInitialAdmin(t, server, runtime)
	now := time.Now().UTC()
	assetID := insertSoftwareTestAsset(t, server, "param-shape",
		"param-shape", "host", `{}`, 1, now)

	routes := []struct {
		name   string
		method string
		suffix string
		body   any
	}{
		{"get", http.MethodGet, "", nil},
		{"patch", http.MethodPatch, "", map[string]any{"name": "renamed by a bad id"}},
		{"delete", http.MethodDelete, "", nil},
		{"restore", http.MethodPost, "/restore", nil},
	}
	for _, route := range routes {
		for label, spelling := range assetIDSpellings(assetID) {
			status, code, body := assetIDResponse(t, server, cookie, csrf,
				route.method, "/api/v1/assets/"+spelling+route.suffix, route.body)
			if status != http.StatusNotFound || code != "ASSET_NOT_FOUND" {
				t.Errorf("%s with a %s asset id (%q) = %d %q, want 404 ASSET_NOT_FOUND: %s",
					route.name, label, spelling, status, code, body)
			}
		}
	}
	// None of those requests may have touched the asset they spell loosely.
	if status, deleted := assetRowState(t, server, assetID); status != "active" || deleted {
		t.Errorf("asset after the refused requests = status %q deleted %v, want active false",
			status, deleted)
	}
	if changes := countRows(t, server,
		`SELECT COUNT(*) FROM asset_changes WHERE asset_id=$1`, assetID,
	); changes != 0 {
		t.Errorf("asset_changes rows written by the refused requests = %d, want 0", changes)
	}
}

// history and relations declare 200 and no 404 at all, so a path parameter they
// cannot look up has to answer an empty collection rather than a 500 on
// PostgreSQL and an empty one on the fallback.
func TestAssetCollectionsAnswerEmptyForNonCanonicalAssetID(t *testing.T) {
	runtime := newRuntime(t)
	server := testServer(t, runtime)
	cookie, csrf := authenticateInitialAdmin(t, server, runtime)
	now := time.Now().UTC()
	assetID := insertSoftwareTestAsset(t, server, "collection-shape",
		"collection-shape", "host", `{}`, 1, now)
	peerID := insertSoftwareTestAsset(t, server, "collection-peer",
		"collection-peer", "software_product", `{}`, 1, now)
	insertRunsOn(t, server, peerID, assetID)

	for _, suffix := range []string{"/history", "/relations"} {
		for label, spelling := range assetIDSpellings(assetID) {
			response := performAuthenticatedJSON(t, server, http.MethodGet,
				"/api/v1/assets/"+spelling+suffix, nil, cookie, csrf)
			if response.Code != http.StatusOK {
				t.Errorf("%s with a %s asset id (%q) = %d, want 200: %s",
					suffix, label, spelling, response.Code, response.Body.String())
				continue
			}
			var payload struct {
				Items []map[string]any `json:"items"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
				t.Errorf("%s with a %s asset id: decode %s: %v",
					suffix, label, response.Body.String(), err)
				continue
			}
			if len(payload.Items) != 0 {
				t.Errorf("%s with a %s asset id returned %d items, want 0",
					suffix, label, len(payload.Items))
			}
		}
	}
}

// The upper-case 36-character form is the spelling openapi declares, and
// PostgreSQL folded it into a hit where the fallback's TEXT comparison did not.
// Every route has to act on the asset itself, which is checked here on the rows
// the handlers write rather than on the response codes.
func TestAssetRoutesAcceptUpperCaseAssetID(t *testing.T) {
	runtime := newRuntime(t)
	server := testServer(t, runtime)
	cookie, csrf := authenticateInitialAdmin(t, server, runtime)
	now := time.Now().UTC()
	assetID := insertSoftwareTestAsset(t, server, "upper-case-shape",
		"upper-case-shape", "host", `{}`, 1, now)
	peerID := insertSoftwareTestAsset(t, server, "upper-case-peer",
		"upper-case-peer", "software_product", `{}`, 1, now)
	insertRunsOn(t, server, peerID, assetID)
	shouting := strings.ToUpper(assetID)
	base := "/api/v1/assets/" + shouting

	// GET has to find it and report the canonical id back.
	getStatus, getCode, getBody := assetIDResponse(t, server, cookie, csrf,
		http.MethodGet, base, nil)
	if getStatus != http.StatusOK {
		t.Fatalf("get with an upper-case id = %d %q, want 200: %s",
			getStatus, getCode, getBody)
	}
	var read struct {
		Asset struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"asset"`
	}
	if err := json.Unmarshal([]byte(getBody), &read); err != nil {
		t.Fatalf("decode %s: %v", getBody, err)
	}
	if read.Asset.ID != assetID {
		t.Errorf("get returned id %q, want the canonical %q", read.Asset.ID, assetID)
	}

	// PATCH has to land on the stored row, not just answer 200.
	patchStatus, patchCode, patchBody := assetIDResponse(t, server, cookie, csrf,
		http.MethodPatch, base, map[string]any{
			"name": "renamed through a shouting id", "reason": "same asset",
		})
	if patchStatus != http.StatusOK {
		t.Fatalf("patch with an upper-case id = %d %q, want 200: %s",
			patchStatus, patchCode, patchBody)
	}
	var stored string
	if err := server.database.DB().QueryRow(
		`SELECT name FROM assets WHERE id=$1`, assetID,
	).Scan(&stored); err != nil {
		t.Fatalf("read the patched name: %v", err)
	}
	if stored != "renamed through a shouting id" {
		t.Errorf("assets.name = %q, want the patched name", stored)
	}

	// DELETE and restore move deleted_at and status on that same row.
	if status, code, body := assetIDResponse(t, server, cookie, csrf,
		http.MethodDelete, base, nil); status != http.StatusOK {
		t.Fatalf("delete with an upper-case id = %d %q, want 200: %s", status, code, body)
	}
	if status, deleted := assetRowState(t, server, assetID); status != "deleted" || !deleted {
		t.Errorf("asset after the delete = status %q deleted %v, want deleted true",
			status, deleted)
	}
	if status, code, body := assetIDResponse(t, server, cookie, csrf,
		http.MethodPost, base+"/restore", nil); status != http.StatusOK {
		t.Fatalf("restore with an upper-case id = %d %q, want 200: %s", status, code, body)
	}
	if status, deleted := assetRowState(t, server, assetID); status != "active" || deleted {
		t.Errorf("asset after the restore = status %q deleted %v, want active false",
			status, deleted)
	}

	// The change rows have to be attached to the canonical id, because that is
	// what every later lookup spells.
	for changeType, want := range map[string]int{"updated": 1, "deleted": 1, "restored": 1} {
		if got := countRows(t, server,
			`SELECT COUNT(*) FROM asset_changes WHERE asset_id=$1 AND change_type=$2`,
			assetID, changeType,
		); got != want {
			t.Errorf("asset_changes %q rows for the canonical id = %d, want %d",
				changeType, got, want)
		}
	}

	// And the two collections have to report that history and the relation.
	historyResponse := performAuthenticatedJSON(t, server, http.MethodGet,
		base+"/history", nil, cookie, csrf)
	var history struct {
		Items []map[string]any `json:"items"`
	}
	if err := json.Unmarshal(historyResponse.Body.Bytes(), &history); err != nil {
		t.Fatalf("decode history %s: %v", historyResponse.Body.String(), err)
	}
	if historyResponse.Code != http.StatusOK || len(history.Items) != 3 {
		t.Errorf("history with an upper-case id = %d with %d items, want 200 with 3",
			historyResponse.Code, len(history.Items))
	}
	relationsResponse := performAuthenticatedJSON(t, server, http.MethodGet,
		base+"/relations", nil, cookie, csrf)
	var relations struct {
		Items []map[string]any `json:"items"`
	}
	if err := json.Unmarshal(relationsResponse.Body.Bytes(), &relations); err != nil {
		t.Fatalf("decode relations %s: %v", relationsResponse.Body.String(), err)
	}
	if relationsResponse.Code != http.StatusOK || len(relations.Items) != 1 {
		t.Errorf("relations with an upper-case id = %d with %d items, want 200 with 1",
			relationsResponse.Code, len(relations.Items))
	}
}
