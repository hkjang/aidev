package store

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

// This is somebody's record of what happened to their own writing, so the two
// things that matter are that it is theirs and that it is complete for the
// window it claims. Both need a real database: the scoping lives in the query.

func aiUsageUser(t *testing.T) (*Store, uuid.UUID) {
	t.Helper()
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		t.Skip("POSTGRES_DSN is not configured")
	}
	db := isolatedStore(t, dsn)
	ctx := context.Background()
	userID := uuid.New()
	name := "ai_usage_" + strings.ReplaceAll(userID.String(), "-", "")
	if _, err := db.Pool.Exec(ctx, `INSERT INTO users(id,username,display_name) VALUES($1,$2::citext,$2::text)`, userID, name); err != nil {
		t.Fatal(err)
	}
	return db, userID
}

func recordCall(t *testing.T, db *Store, userID uuid.UUID, purpose Purpose, status string, ago time.Duration) {
	t.Helper()
	if _, err := db.Pool.Exec(context.Background(), `
		INSERT INTO ai_calls(user_id,model,status,input_tokens,output_tokens,purpose,created_at)
		VALUES($1,'a-model',$2,10,20,$3,now()-$4::interval)`,
		userID, status, string(purpose), ago.String()); err != nil {
		t.Fatal(err)
	}
}

func TestPersonalAIUsageIsOnlyYourOwnIntegration(t *testing.T) {
	db, userID := aiUsageUser(t)
	ctx := context.Background()

	stranger := uuid.New()
	name := "ai_other_" + strings.ReplaceAll(stranger.String(), "-", "")
	if _, err := db.Pool.Exec(ctx, `INSERT INTO users(id,username,display_name) VALUES($1,$2::citext,$2::text)`, stranger, name); err != nil {
		t.Fatal(err)
	}
	recordCall(t, db, userID, PurposeAssist, "success", time.Hour)
	recordCall(t, db, stranger, PurposeDream, "success", time.Hour)
	recordCall(t, db, stranger, PurposeAsk, "success", time.Hour)

	usage, err := db.PersonalAIUsage(ctx, userID, time.Now().AddDate(0, 0, -30))
	if err != nil {
		t.Fatal(err)
	}
	if usage.Total != 1 || len(usage.Entries) != 1 {
		t.Fatalf("someone else's calls reached this record: total=%d entries=%d", usage.Total, len(usage.Entries))
	}
	if usage.Entries[0].Purpose != string(PurposeAssist) {
		t.Fatalf("purpose=%q", usage.Entries[0].Purpose)
	}
	if usage.Counts[string(PurposeDream)] != 0 || usage.Counts[string(PurposeAsk)] != 0 {
		t.Fatalf("the counts include somebody else's calls: %v", usage.Counts)
	}
}

// A failed call still sent the prompt. Leaving it out would mean the words went
// to a gateway and the record said they had not.
func TestPersonalAIUsageListsFailedCallsIntegration(t *testing.T) {
	db, userID := aiUsageUser(t)
	recordCall(t, db, userID, PurposeAsk, "failed", time.Hour)

	usage, err := db.PersonalAIUsage(context.Background(), userID, time.Now().AddDate(0, 0, -30))
	if err != nil {
		t.Fatal(err)
	}
	if usage.Total != 1 || usage.Entries[0].Status != "failed" {
		t.Fatalf("a failed call is missing from the record: %+v", usage)
	}
}

// The counts describe the window. The list is a page of it, and reading the
// totals off the page would understate a busy month.
func TestPersonalAIUsageCountsTheWindowNotThePageIntegration(t *testing.T) {
	db, userID := aiUsageUser(t)
	for i := 0; i < maxAIUsageEntries+5; i++ {
		recordCall(t, db, userID, PurposeDream, "success", time.Hour)
	}

	usage, err := db.PersonalAIUsage(context.Background(), userID, time.Now().AddDate(0, 0, -30))
	if err != nil {
		t.Fatal(err)
	}
	if len(usage.Entries) != maxAIUsageEntries {
		t.Fatalf("%d entries, want a page of %d", len(usage.Entries), maxAIUsageEntries)
	}
	if !usage.Truncated {
		t.Fatal("the answer does not say there is more than it showed")
	}
	if usage.Total != maxAIUsageEntries+5 || usage.Counts[string(PurposeDream)] != maxAIUsageEntries+5 {
		t.Fatalf("the totals were read off the page: total=%d counts=%v", usage.Total, usage.Counts)
	}
}

// Asking about thirty days must not answer with a year.
func TestPersonalAIUsageRespectsTheWindowIntegration(t *testing.T) {
	db, userID := aiUsageUser(t)
	recordCall(t, db, userID, PurposeAssist, "success", 2*time.Hour)
	recordCall(t, db, userID, PurposeDream, "success", 40*24*time.Hour)

	usage, err := db.PersonalAIUsage(context.Background(), userID, time.Now().AddDate(0, 0, -30))
	if err != nil {
		t.Fatal(err)
	}
	if usage.Total != 1 || usage.Counts[string(PurposeDream)] != 0 {
		t.Fatalf("a call from outside the window was counted: %+v", usage)
	}

	// And widening the window finds it, so the exclusion above is the window
	// working rather than the call never having been written.
	wider, err := db.PersonalAIUsage(context.Background(), userID, time.Now().AddDate(0, 0, -60))
	if err != nil {
		t.Fatal(err)
	}
	if wider.Total != 2 {
		t.Fatalf("widening the window did not find the older call: %+v", wider)
	}
}

// Newest first: someone opening this is looking for what just happened.
func TestPersonalAIUsageIsNewestFirstIntegration(t *testing.T) {
	db, userID := aiUsageUser(t)
	recordCall(t, db, userID, PurposeDream, "success", 3*time.Hour)
	recordCall(t, db, userID, PurposeAsk, "success", time.Hour)

	usage, err := db.PersonalAIUsage(context.Background(), userID, time.Now().AddDate(0, 0, -30))
	if err != nil {
		t.Fatal(err)
	}
	if len(usage.Entries) != 2 || usage.Entries[0].Purpose != string(PurposeAsk) {
		t.Fatalf("not newest first: %+v", usage.Entries)
	}
}

// Rows written before v0.67.0 carry no purpose. Empty has to survive to the
// screen as empty, so it can say "not recorded" rather than inventing a label.
func TestPersonalAIUsageKeepsAnUnrecordedPurposeEmptyIntegration(t *testing.T) {
	db, userID := aiUsageUser(t)
	recordCall(t, db, userID, "", "success", time.Hour)

	usage, err := db.PersonalAIUsage(context.Background(), userID, time.Now().AddDate(0, 0, -30))
	if err != nil {
		t.Fatal(err)
	}
	if len(usage.Entries) != 1 || usage.Entries[0].Purpose != "" {
		t.Fatalf("an unrecorded purpose was given a label: %+v", usage.Entries)
	}
}

// The vocabulary in Go and the CHECK constraint in SQL have to be the same set,
// or one of them is decoration.
func TestEveryPurposeIsAcceptedByTheDatabaseIntegration(t *testing.T) {
	db, userID := aiUsageUser(t)
	for _, purpose := range Purposes() {
		recordCall(t, db, userID, purpose, "success", time.Hour)
	}
	usage, err := db.PersonalAIUsage(context.Background(), userID, time.Now().AddDate(0, 0, -30))
	if err != nil {
		t.Fatal(err)
	}
	if usage.Total != len(Purposes()) {
		t.Fatalf("%d calls recorded for %d purposes", usage.Total, len(Purposes()))
	}
	// And something outside the vocabulary is refused, so the constraint is
	// doing work rather than accepting anything.
	_, err = db.Pool.Exec(context.Background(),
		`INSERT INTO ai_calls(user_id,model,status,purpose) VALUES($1,'m','success','whatever-it-felt-like')`, userID)
	if err == nil {
		t.Fatal("the database accepted a purpose outside the vocabulary")
	}
}
