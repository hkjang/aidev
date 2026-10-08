package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

// Merging asset A into B leaves A with status 'merged' and a deleted_at, and
// asset_get filtered it out with the same `asset not found` it gives a UUID
// that never existed. A model still holding A's id - from an earlier turn, an
// asset_search page, a relation edge - asked again and concluded the asset was
// gone, when the console and REST GET both still knew it as merged.

type mcpAssetGetReply struct {
	Asset      *assetView `json:"asset"`
	MergedInto string     `json:"merged_into"`
	MergedAt   string     `json:"merged_at"`
	Message    string     `json:"message"`
}

func mcpAssetGetReplyFor(t *testing.T, server *Server, assetID string) (mcpAssetGetReply, error) {
	t.Helper()
	result, err := server.mcpAssetGet(
		httptest.NewRequest("POST", "/mcp", nil),
		mcpArgumentsForTest("asset_get", []byte(`{"asset_id":"`+assetID+`"}`)),
	)
	if err != nil {
		return mcpAssetGetReply{}, err
	}
	encoded, _ := json.Marshal(result)
	var reply mcpAssetGetReply
	if err := json.Unmarshal(encoded, &reply); err != nil {
		t.Fatal(err)
	}
	return reply, nil
}

// mergeAssetsThroughREST merges through the production handler so the test
// reads the asset_changes row the server actually writes, not one shaped by
// hand.
func mergeAssetsThroughREST(
	t *testing.T, server *Server, cookie *http.Cookie, csrf, primaryID string,
	secondaryIDs ...string,
) {
	t.Helper()
	response := performAuthenticatedJSON(t, server, http.MethodPost,
		"/api/v1/assets/merge",
		map[string]any{
			"primary_id": primaryID, "secondary_ids": secondaryIDs,
			"reason": "duplicate host",
		}, cookie, csrf)
	if response.Code != http.StatusOK {
		t.Fatalf("merge status = %d body = %s", response.Code, response.Body.String())
	}
}

func TestMCPAssetGetPointsAMergedAssetAtItsPrimary(t *testing.T) {
	runtime := newRuntime(t)
	server := testServer(t, runtime)
	cookie, csrf := authenticateInitialAdmin(t, server, runtime)
	now := time.Now().UTC()
	primaryID := insertSoftwareTestAsset(t, server, "merge-primary", "merge-primary",
		"host", `{}`, 1, now)
	secondaryID := insertSoftwareTestAsset(t, server, "merge-secondary", "merge-secondary",
		"host", `{}`, 1, now.Add(-time.Minute))
	mergeAssetsThroughREST(t, server, cookie, csrf, primaryID, secondaryID)

	merged, err := mcpAssetGetReplyFor(t, server, secondaryID)
	if err != nil {
		t.Fatalf("asset_get on the merged asset returned an error: %v", err)
	}
	if merged.MergedInto != primaryID {
		t.Fatalf("merged_into = %q, want the primary %q", merged.MergedInto, primaryID)
	}
	if merged.Asset != nil {
		t.Fatalf("a merged asset must not be returned as if it were live: %+v", merged.Asset)
	}
	// merged_at is the deleted_at the merge wrote, normalised the way every
	// other timestamp leaves this API, whichever driver scanned it.
	if _, err := time.Parse(time.RFC3339Nano, merged.MergedAt); err != nil {
		t.Fatalf("merged_at = %q is not RFC 3339: %v", merged.MergedAt, err)
	}
	if !strings.Contains(merged.Message, primaryID) ||
		!strings.Contains(merged.Message, "asset_get") {
		t.Fatalf("message = %q must name the primary and the call to make", merged.Message)
	}

	// The primary is unchanged: a live asset, no merge hint.
	live, err := mcpAssetGetReplyFor(t, server, primaryID)
	if err != nil {
		t.Fatal(err)
	}
	if live.Asset == nil || live.Asset.ID != primaryID || live.MergedInto != "" {
		t.Fatalf("primary after merge = %+v", live)
	}

	// An id that never existed keeps the exact error clients already match on.
	if _, err := mcpAssetGetReplyFor(t, server, uuid.NewString()); err == nil ||
		err.Error() != "asset not found" {
		t.Fatalf("unknown asset error = %v, want %q", err, "asset not found")
	}
}

// A->B then B->C answers one hop at a time: asking for A names B, asking for B
// names C. The next call gets its own hint, so the chain needs no walking and
// no cycle guard.
func TestMCPAssetGetFollowsOneMergeHopOnly(t *testing.T) {
	runtime := newRuntime(t)
	server := testServer(t, runtime)
	cookie, csrf := authenticateInitialAdmin(t, server, runtime)
	now := time.Now().UTC()
	a := insertSoftwareTestAsset(t, server, "chain-a", "chain-a", "host", `{}`, 1, now)
	b := insertSoftwareTestAsset(t, server, "chain-b", "chain-b", "host", `{}`, 1, now)
	c := insertSoftwareTestAsset(t, server, "chain-c", "chain-c", "host", `{}`, 1, now)
	mergeAssetsThroughREST(t, server, cookie, csrf, b, a)
	mergeAssetsThroughREST(t, server, cookie, csrf, c, b)

	first, err := mcpAssetGetReplyFor(t, server, a)
	if err != nil {
		t.Fatal(err)
	}
	if first.MergedInto != b {
		t.Fatalf("A merged_into = %q, want B %q (one hop, not C)", first.MergedInto, b)
	}
	second, err := mcpAssetGetReplyFor(t, server, b)
	if err != nil {
		t.Fatal(err)
	}
	if second.MergedInto != c {
		t.Fatalf("B merged_into = %q, want C %q", second.MergedInto, c)
	}
	last, err := mcpAssetGetReplyFor(t, server, c)
	if err != nil || last.Asset == nil || last.Asset.ID != c {
		t.Fatalf("C = %+v err = %v", last, err)
	}
}
