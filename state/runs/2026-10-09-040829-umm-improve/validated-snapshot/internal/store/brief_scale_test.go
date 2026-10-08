package store

import (
	"context"
	"strconv"
	"testing"
)

// Finding duplicates means comparing every pair, so the work grows with the
// square of the space. Measured at 1024 dimensions: 1,000 thoughts take 229ms,
// 2,000 take 916ms and 10,000 would take about 23 seconds — per space, every
// morning. A brief that times out silently is worse than one that says what it
// left out.
// Finding duplicates means comparing every pair, so the work grows with the
// square of the space. Measured at 1024 dimensions: 1,000 thoughts take 229ms,
// 2,000 take 916ms and 10,000 would take about 23 seconds — per space, every
// morning. A brief that times out silently is worse than one that says what it
// left out.
//
// Which end is kept is the part that can be wrong: a duplicate is almost always
// recent, so keeping the oldest thousand would examine the least likely part of
// the space and nothing about the result would look wrong.
func TestDuplicateScanReadsTheRecentEndIntegration(t *testing.T) {
	db, userID, spaceID := retrievalSpace(t)
	ctx := context.Background()

	total := maxDuplicateScanNotes + 50
	if _, err := db.Pool.Exec(ctx, `
		INSERT INTO notes(space_id,author_id,content,created_at)
		SELECT $1,$2,'순서 확인 ' || g, now() - make_interval(secs => $3 - g)
		FROM generate_series(1,$3) g`, spaceID, userID, total); err != nil {
		t.Fatal(err)
	}

	notes, trimmed, err := db.recentNotesForDuplicateScan(ctx, userID, spaceID)
	if err != nil {
		t.Fatal(err)
	}
	if !trimmed {
		t.Fatal("a space past the bound did not report that there were more")
	}
	if len(notes) != maxDuplicateScanNotes {
		t.Fatalf("read %d notes, want %d", len(notes), maxDuplicateScanNotes)
	}
	newest := "순서 확인 " + strconv.Itoa(total)
	found := false
	for _, note := range notes {
		if note.Content == newest {
			found = true
		}
		if note.Content == "순서 확인 1" {
			t.Error("the oldest thought was read; the bound must keep the recent end")
		}
	}
	if !found {
		t.Errorf("the newest thought %q was not read", newest)
	}
}

// Under the bound nothing is trimmed, so a small workspace does not read as
// partly examined.
func TestDuplicateScanLeavesSmallSpacesWholeIntegration(t *testing.T) {
	db, userID, spaceID := retrievalSpace(t)
	ctx := context.Background()

	if _, err := db.Pool.Exec(ctx, `
		INSERT INTO notes(space_id,author_id,content)
		SELECT $1,$2,'작은 공간 ' || g FROM generate_series(1,5) g`, spaceID, userID); err != nil {
		t.Fatal(err)
	}
	notes, trimmed, err := db.recentNotesForDuplicateScan(ctx, userID, spaceID)
	if err != nil {
		t.Fatal(err)
	}
	if trimmed {
		t.Error("a five-thought space was reported as trimmed")
	}
	if len(notes) != 5 {
		t.Errorf("read %d notes, want 5", len(notes))
	}
}

// A pair where one side sits in a line that was decided against is not the same
// news as a pair someone simply wrote twice, and only the first carries a reason
// the person needs. Sorting by similarity alone let a workspace full of
// near-identical thoughts fill every slot and drown it out — seen on real data.
func TestSetAsidePairsOutrankOrdinaryDuplicates(t *testing.T) {
	ordinary := func(score float64) DuplicatePair { return DuplicatePair{Score: score} }
	repeated := func(score float64) DuplicatePair {
		return DuplicatePair{Score: score, SetAside: &BranchRef{Name: "젠킨스로 이전", Status: BranchAbandoned}}
	}

	// Highest similarity last, so score alone would leave it out of the top three.
	pairs := []DuplicatePair{ordinary(0.99), ordinary(0.98), ordinary(0.97), repeated(0.93)}
	sortDuplicatesForBrief(pairs)

	if pairs[0].SetAside == nil {
		t.Fatalf("the repeated decision did not come first: %+v", pairs[0])
	}
	// Everything else keeps its similarity order; the label promotes, it does not
	// reshuffle.
	for index := 1; index < len(pairs); index++ {
		if pairs[index].SetAside != nil {
			t.Errorf("more pairs were promoted than were labelled: %d", index)
		}
	}
	if pairs[1].Score < pairs[2].Score || pairs[2].Score < pairs[3].Score {
		t.Errorf("the remaining pairs lost their similarity order: %v %v %v",
			pairs[1].Score, pairs[2].Score, pairs[3].Score)
	}
}
