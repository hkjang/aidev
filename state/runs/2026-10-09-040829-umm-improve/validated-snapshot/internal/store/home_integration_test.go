package store

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

// The home screen is the page a person meets every morning, so what it chooses
// to keep showing them matters more than on a page they visit deliberately.
// Both tests here are about that choice: a guide that will not end, and a
// caveat about a check that never had anything to examine.

func homeUser(t *testing.T, ctx context.Context, db *Store) uuid.UUID {
	t.Helper()
	id := uuid.New()
	name := "home_" + strings.ReplaceAll(id.String(), "-", "")
	if _, err := db.Pool.Exec(ctx, `INSERT INTO users(id,username,display_name) VALUES($1,$2::citext,$2::text)`, id, name); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Pool.Exec(ctx, `INSERT INTO user_preferences(user_id) VALUES($1) ON CONFLICT DO NOTHING`, id); err != nil {
		t.Fatal(err)
	}
	return id
}

// The guide asks for four things. Doing all four is finishing it — nobody
// should have to also find the button that says so.
func TestOnboardingClosesItselfOnceEveryStepIsDoneIntegration(t *testing.T) {
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		t.Skip("POSTGRES_DSN is not configured")
	}
	ctx := context.Background()
	db := isolatedStore(t, dsn)
	userID := homeUser(t, ctx, db)

	spaceID, firstNote, secondNote := uuid.New(), uuid.New(), uuid.New()
	// Each step is added on its own, and the guide has to stay open until the
	// last one lands. A rule that closed early would take the guide away from
	// someone still working through it.
	steps := []struct {
		name string
		sql  string
		args []any
	}{
		{"space", `INSERT INTO spaces(id,owner_id,name) VALUES($1,$2,'home guide')`, []any{spaceID, userID}},
		{"note", `INSERT INTO notes(id,space_id,author_id,content) VALUES($1,$2,$3,'첫 생각'),($4,$2,$3,'두 번째 생각')`, []any{firstNote, spaceID, userID, secondNote}},
		{"connect", `INSERT INTO note_edges(space_id,source_note_id,target_note_id,created_by) VALUES($1,$2,$3,$4)`, []any{spaceID, firstNote, secondNote, userID}},
		{"collaborate", `INSERT INTO note_comments(note_id,author_id,body) VALUES($1,$2,'같이 봅시다')`, []any{firstNote, userID}},
	}
	for i, step := range steps {
		if _, err := db.Pool.Exec(ctx, step.sql, step.args...); err != nil {
			t.Fatalf("%s: %v", step.name, err)
		}
		review, err := db.TodayReview(ctx, userID)
		if err != nil {
			t.Fatalf("%s: %v", step.name, err)
		}
		last := i == len(steps)-1
		if got := review.Onboarding.CompletedAt != nil; got != last {
			t.Fatalf("after %s (%d/%d done, %d%%): completed=%v, want %v",
				step.name, i+1, len(steps), review.Onboarding.Percent, got, last)
		}
	}

	// Finishing is a moment, not a state that keeps being recomputed: a second
	// load must not move the date.
	first, err := db.TodayReview(ctx, userID)
	if err != nil {
		t.Fatal(err)
	}
	second, err := db.TodayReview(ctx, userID)
	if err != nil {
		t.Fatal(err)
	}
	if !first.Onboarding.CompletedAt.Equal(*second.Onboarding.CompletedAt) {
		t.Fatalf("the completion date moved on a reload: %v then %v",
			first.Onboarding.CompletedAt, second.Onboarding.CompletedAt)
	}
	if first.Onboarding.Percent != 100 {
		t.Fatalf("every step was done but the guide reported %d%%", first.Onboarding.Percent)
	}
}

// A skip exists so an empty list is not mistaken for an all-clear. That is a
// service to someone whose thoughts umm could not compare, and noise to someone
// who has not written two yet.
func TestTheBriefOnlyReportsASkippedCheckThatHadSomethingToExamineIntegration(t *testing.T) {
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		t.Skip("POSTGRES_DSN is not configured")
	}
	ctx := context.Background()
	db := isolatedStore(t, dsn)
	userID := homeUser(t, ctx, db)

	// Nothing written at all. The duplicate check could not have found anything
	// either way, so there is no gap to report and the brief has nothing to say.
	brief, err := db.MorningBrief(ctx, userID, time.Now().Add(-24*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if len(brief.Skipped) != 0 {
		t.Fatalf("an account with no thoughts was told a check was skipped: %+v", brief.Skipped)
	}
	if !brief.Quiet {
		t.Fatal("an account with nothing in it produced a brief that was not quiet")
	}

	spaceID := uuid.New()
	if _, err := db.Pool.Exec(ctx, `INSERT INTO spaces(id,owner_id,name) VALUES($1,$2,'중복 검사')`, spaceID, userID); err != nil {
		t.Fatal(err)
	}
	// One thought still cannot be a duplicate of anything: a pair needs two.
	if _, err := db.Pool.Exec(ctx, `INSERT INTO notes(space_id,author_id,content) VALUES($1,$2,'혼자 있는 생각')`, spaceID, userID); err != nil {
		t.Fatal(err)
	}
	brief, err = db.MorningBrief(ctx, userID, time.Now().Add(-24*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if len(brief.Skipped) != 0 {
		t.Fatalf("one thought cannot pair with anything, but a skip was reported: %+v", brief.Skipped)
	}

	// Two comparable thoughts is where the check becomes meaningful — and where
	// staying silent about not running it would read as an all-clear.
	if _, err := db.Pool.Exec(ctx, `INSERT INTO notes(space_id,author_id,content) VALUES($1,$2,'비교할 수 있는 두 번째 생각')`, spaceID, userID); err != nil {
		t.Fatal(err)
	}
	brief, err = db.MorningBrief(ctx, userID, time.Now().Add(-24*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if len(brief.Skipped) != 1 || brief.Skipped[0].Kind != "duplicates" {
		t.Fatalf("two comparable thoughts went unchecked without saying so: %+v", brief.Skipped)
	}
	if brief.Quiet {
		t.Fatal("a brief that skipped a check called itself quiet")
	}

	// An archived thought is out of the canvas and out of the check, so it must
	// not be what makes the pair.
	if _, err := db.Pool.Exec(ctx, `UPDATE notes SET archived=true WHERE space_id=$1 AND content='비교할 수 있는 두 번째 생각'`, spaceID); err != nil {
		t.Fatal(err)
	}
	brief, err = db.MorningBrief(ctx, userID, time.Now().Add(-24*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if len(brief.Skipped) != 0 {
		t.Fatalf("an archived thought was counted as something to compare: %+v", brief.Skipped)
	}
}
