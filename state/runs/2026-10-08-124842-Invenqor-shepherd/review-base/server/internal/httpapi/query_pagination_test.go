package httpapi

import (
	"testing"
	"time"

	"github.com/hkjang/invenqor/server/internal/storagetest"
)

// The Query DSL is the only way to ask most inventory questions, and its result
// used to stop at the limit with no way past it: an expression matching 1,200
// stale hosts answered with at most 500 and the caller's only move was to
// narrow the expression until it fitted. An API key script has no console to
// narrow anything in. Paging by offset is the same walk /api/v1/assets already
// supports, and total says how far the walk goes.
func TestQueryExecutePagesPastTheLimit(t *testing.T) {
	runtime := storagetest.Open(t)
	defer runtime.Close()
	server := testServer(t, runtime)
	cookie, csrf := authenticateInitialAdmin(t, server, runtime)

	const assets = 5
	for index := 0; index < assets; index++ {
		insertAssetSeenAt(
			t, server, "host-"+string(rune('a'+index)),
			time.Now().UTC().Add(-time.Duration(index)*time.Minute),
		)
	}

	first := executeQueryResult(t, server, cookie, csrf, map[string]any{
		"query": `type = "host"`, "limit": 2,
	})
	if len(first.Items) != 2 || first.Offset != 0 {
		t.Fatalf("first page returned %d items at offset %d, want 2 at 0",
			len(first.Items), first.Offset)
	}
	if first.Total != assets {
		t.Errorf("total = %d, want %d", first.Total, assets)
	}
	if !first.HasMore || first.NextOffset != 2 {
		t.Fatalf("has_more = %v, next_offset = %d, want true and 2",
			first.HasMore, first.NextOffset)
	}

	// Walking next_offset has to reach every match exactly once. A lookahead row
	// that leaked into a page would show up here as a duplicate.
	seen := map[string]int{}
	offset, pages := 0, 0
	for {
		page := executeQueryResult(t, server, cookie, csrf, map[string]any{
			"query": `type = "host"`, "limit": 2, "offset": offset,
		})
		if page.Offset != offset {
			t.Fatalf("asked for offset %d, response said %d", offset, page.Offset)
		}
		if page.Total != assets {
			t.Errorf("page at offset %d reported total %d, want %d",
				offset, page.Total, assets)
		}
		for _, item := range page.Items {
			seen[item.Name]++
		}
		pages++
		if pages > assets+2 {
			t.Fatalf("paging did not finish after %d pages: %v", pages, seen)
		}
		if !page.HasMore {
			if page.NextOffset != assets {
				t.Errorf("last page next_offset = %d, want %d",
					page.NextOffset, assets)
			}
			break
		}
		offset = page.NextOffset
	}
	if len(seen) != assets {
		t.Fatalf("the walk saw %d distinct assets, want %d: %v",
			len(seen), assets, seen)
	}
	for name, count := range seen {
		if count != 1 {
			t.Errorf("%s appeared %d times in the walk", name, count)
		}
	}

	// Past the end is an empty page, not an error, and total still answers how
	// large the result was.
	beyond := executeQueryResult(t, server, cookie, csrf, map[string]any{
		"query": `type = "host"`, "limit": 2, "offset": assets,
	})
	if len(beyond.Items) != 0 || beyond.HasMore {
		t.Fatalf("offset past the end returned %d items, has_more = %v",
			len(beyond.Items), beyond.HasMore)
	}
	if beyond.Total != assets {
		t.Errorf("offset past the end reported total %d, want %d",
			beyond.Total, assets)
	}
}

// Assets ingested in one batch share last_seen_at to the second. Ordering by
// that column alone leaves their relative order to the engine, and a paged walk
// over an unstable order hands back some rows twice while never showing others.
func TestQueryExecutePagesTiedTimestampsWithoutRepeating(t *testing.T) {
	runtime := storagetest.Open(t)
	defer runtime.Close()
	server := testServer(t, runtime)
	cookie, csrf := authenticateInitialAdmin(t, server, runtime)

	const assets = 8
	batch := time.Now().UTC().Truncate(time.Second)
	for index := 0; index < assets; index++ {
		insertAssetSeenAt(t, server, "tied-"+string(rune('a'+index)), batch)
	}

	seen := map[string]int{}
	for offset := 0; offset < assets; offset += 3 {
		page := executeQueryResult(t, server, cookie, csrf, map[string]any{
			"query": `type = "host"`, "limit": 3, "offset": offset,
		})
		for _, item := range page.Items {
			seen[item.Name]++
		}
	}
	if len(seen) != assets {
		t.Fatalf("paging tied timestamps saw %d distinct assets, want %d: %v",
			len(seen), assets, seen)
	}
	for name, count := range seen {
		if count != 1 {
			t.Errorf("%s appeared %d times across pages", name, count)
		}
	}
}
