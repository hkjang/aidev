package store

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

// The tiles on the landing page are counts, so they have to count.
//
// They used to report the length of lists the queries capped at five to eight,
// so a workspace with two thousand unconnected thoughts said six. That is not a
// sample size on the page — it sits beside the words "waiting to be connected",
// and reading it as the total is the only way to read it.
func TestTodayCountsAreTotalsNotPreviewLengthsIntegration(t *testing.T) {
	db, userID, spaceID := retrievalSpace(t)
	ctx := context.Background()

	// Comfortably past every list cap.
	const unconnected = 40
	if _, err := db.Pool.Exec(ctx, `
		INSERT INTO notes(space_id,author_id,content,updated_at)
		SELECT $1,$2,'연결되지 않은 생각 ' || g, now() - interval '30 days'
		FROM generate_series(1,$3) g`, spaceID, userID, unconnected); err != nil {
		t.Fatal(err)
	}

	digest, err := db.TodayReview(ctx, userID)
	if err != nil {
		t.Fatal(err)
	}

	if digest.Counts["orphans"] != unconnected {
		t.Errorf("orphans tile says %d with %d unconnected thoughts", digest.Counts["orphans"], unconnected)
	}
	// The list stays a preview, and looks like one.
	if len(digest.Orphans) >= unconnected {
		t.Errorf("the previewed list returned %d rows; it is meant to stay capped", len(digest.Orphans))
	}
	if digest.Counts["orphans"] <= len(digest.Orphans) {
		t.Error("the count did not exceed the preview, so it is still reporting the cap")
	}

	// Nothing was written that should count as needing review the same way, but
	// the tile must still agree with its own list rather than drift from it.
	if digest.Counts["review"] < len(digest.Review) {
		t.Errorf("review tile %d is smaller than the %d rows beneath it",
			digest.Counts["review"], len(digest.Review))
	}
	for _, key := range []string{"review", "orphans", "dreams", "activity"} {
		if _, ok := digest.Counts[key]; !ok {
			t.Errorf("the %s tile has no count at all", key)
		}
	}
	_ = uuid.Nil
}
