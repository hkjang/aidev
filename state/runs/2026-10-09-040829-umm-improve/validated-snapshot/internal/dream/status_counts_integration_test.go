package dream

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hkjang/umm/internal/store"
)

// The tabs on the Dreams page label states, and the list under them arrives
// thirty at a time. Counting what arrived counts the page — measured with
// thirty-seven waiting, the tab read 검토함 30 and changed to 37 when the reader
// pressed a button about older history. These are the numbers that must not
// come from the page.

func countsFixture(t *testing.T) (*store.Store, uuid.UUID, uuid.UUID, func()) {
	t.Helper()
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		t.Skip("POSTGRES_DSN is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	db, err := store.Open(ctx, dsn)
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	if err = db.Migrate(ctx); err != nil {
		cancel()
		t.Fatal(err)
	}
	userID, spaceID := uuid.New(), uuid.New()
	name := "dream_counts_" + userID.String()
	if _, err = db.Pool.Exec(ctx, `INSERT INTO users(id,username,display_name) VALUES($1,$2::citext,$2::text)`, userID, name); err != nil {
		cancel()
		t.Fatal(err)
	}
	if _, err = db.Pool.Exec(ctx, `INSERT INTO spaces(id,owner_id,name) VALUES($1,$2,'Dream 개수')`, spaceID, userID); err != nil {
		cancel()
		t.Fatal(err)
	}
	cleanup := func() {
		db.Pool.Exec(context.Background(), `DELETE FROM users WHERE id=$1`, userID)
		db.Pool.Close()
		cancel()
	}
	return db, userID, spaceID, cleanup
}

func seedDreams(t *testing.T, db *store.Store, userID, spaceID uuid.UUID, status string, n int) {
	t.Helper()
	for i := 0; i < n; i++ {
		if _, err := db.Pool.Exec(context.Background(), `
			INSERT INTO dream_notes(user_id,space_id,dream_type,content,status)
			VALUES($1,$2,'connection','관점',$3)`, userID, spaceID, status); err != nil {
			t.Fatal(err)
		}
	}
}

// The count is of the queue, not of a page of it.
func TestStatusCountsSeePastThePageIntegration(t *testing.T) {
	db, userID, spaceID, cleanup := countsFixture(t)
	defer cleanup()
	ctx := context.Background()
	service := &Service{Store: db}

	// More than one page of the default thirty, deliberately: that is the exact
	// shape the page was wrong about.
	seedDreams(t, db, userID, spaceID, "created", 35)
	seedDreams(t, db, userID, spaceID, "exposed", 2)
	seedDreams(t, db, userID, spaceID, "kept", 3)
	seedDreams(t, db, userID, spaceID, "deleted", 4)

	counts, err := service.StatusCounts(ctx, userID)
	if err != nil {
		t.Fatal(err)
	}
	// created and exposed are both waiting to be looked at; the page puts them
	// in one tab and so must the count.
	for label, want := range map[string]int{"inbox": 37, "kept": 3, "hidden": 4, "all": 44} {
		if counts[label] != want {
			t.Errorf("%s counted %d, want %d", label, counts[label], want)
		}
	}

	// The first page is what the reader actually receives, and the count must
	// not be it.
	page, hasMore, err := service.HistoryPage(ctx, userID, 30, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(page) != 30 || !hasMore {
		t.Fatalf("expected a full first page with more behind it, got %d and hasMore=%v", len(page), hasMore)
	}
	if counts["all"] == len(page) {
		t.Fatal("the count equals the page size — it is counting the page, which is the defect")
	}
}

// A dream in someone else's space is not this person's to count, exactly as it
// is not theirs to list.
func TestStatusCountsAreScopedLikeTheListingIntegration(t *testing.T) {
	db, userID, spaceID, cleanup := countsFixture(t)
	defer cleanup()
	ctx := context.Background()
	service := &Service{Store: db}
	seedDreams(t, db, userID, spaceID, "created", 2)

	strangerID, strangerSpace := uuid.New(), uuid.New()
	strangerName := "dream_counts_stranger_" + strangerID.String()
	if _, err := db.Pool.Exec(ctx, `INSERT INTO users(id,username,display_name) VALUES($1,$2::citext,$2::text)`, strangerID, strangerName); err != nil {
		t.Fatal(err)
	}
	defer db.Pool.Exec(context.Background(), `DELETE FROM users WHERE id=$1`, strangerID)
	if _, err := db.Pool.Exec(ctx, `INSERT INTO spaces(id,owner_id,name) VALUES($1,$2,'남의 공간')`, strangerSpace, strangerID); err != nil {
		t.Fatal(err)
	}
	seedDreams(t, db, strangerID, strangerSpace, "created", 5)

	counts, err := service.StatusCounts(ctx, userID)
	if err != nil {
		t.Fatal(err)
	}
	if counts["inbox"] != 2 || counts["all"] != 2 {
		t.Fatalf("counted someone else's dreams: inbox=%d all=%d, want 2 and 2", counts["inbox"], counts["all"])
	}

	// And the count agrees with what the listing would actually return.
	page, _, err := service.HistoryPage(ctx, userID, 100, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(page) != counts["all"] {
		t.Fatalf("the count says %d and the listing returns %d — they must describe the same set", counts["all"], len(page))
	}
}

// An account with nothing reports zeros rather than an absent map, so a tab can
// tell "none" from "not counted yet".
func TestStatusCountsReportZeroRatherThanNothingIntegration(t *testing.T) {
	db, userID, _, cleanup := countsFixture(t)
	defer cleanup()
	counts, err := (&Service{Store: db}).StatusCounts(context.Background(), userID)
	if err != nil {
		t.Fatal(err)
	}
	for _, label := range []string{"inbox", "kept", "hidden", "all"} {
		if got, ok := counts[label]; !ok || got != 0 {
			t.Errorf("%s: got %d present=%v, want 0 and present", label, got, ok)
		}
	}
}
