package store

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func briefUser(t *testing.T) (*Store, uuid.UUID) {
	t.Helper()
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		t.Skip("POSTGRES_DSN is not configured")
	}
	ctx := context.Background()
	db, err := Open(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Pool.Close() })
	userID := uuid.New()
	name := "brief_" + strings.ReplaceAll(userID.String(), "-", "")
	if _, err = db.Pool.Exec(ctx, `INSERT INTO users(id,username,display_name) VALUES($1,$2::citext,$2::text)`, userID, name); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Pool.Exec(context.Background(), `DELETE FROM users WHERE id=$1`, userID) })
	if _, err = db.Pool.Exec(ctx, `INSERT INTO user_preferences(user_id) VALUES($1)`, userID); err != nil {
		t.Fatal(err)
	}
	return db, userID
}

func briefFor(t *testing.T, db *Store, userID uuid.UUID) MorningBrief {
	t.Helper()
	brief, err := db.MorningBrief(context.Background(), userID, time.Now().Add(-24*time.Hour))
	if err != nil {
		t.Fatalf("morning brief: %v", err)
	}
	return brief
}

// The property the brief exists to protect. An empty duplicates list means two
// different things depending on whether umm was fit to look, and a reader who
// cannot tell them apart will read "nothing found" as "nothing there".
func TestBriefSaysWhatItCouldNotCheckIntegration(t *testing.T) {
	db, userID := briefUser(t)
	restore := useOfflineEmbedding(t, db)
	defer restore()

	// Two thoughts, because that is where the property starts to hold. A
	// duplicate is a pair, so on an account with fewer than two there is no
	// unexamined pair to mistake for an all-clear — and saying a check was
	// skipped would describe a gap that does not exist. That case is covered by
	// TestTheBriefOnlyReportsASkippedCheckThatHadSomethingToExamineIntegration.
	ctx := context.Background()
	space, err := db.CreateSpace(ctx, userID, "확인하지 못한 것")
	if err != nil {
		t.Fatal(err)
	}
	for _, content := range []string{"겹칠 수도 있는 생각 하나", "겹칠 수도 있는 생각 둘"} {
		if _, err = db.CreateNote(ctx, userID, Note{SpaceID: space.ID, AuthorID: userID, Content: content}); err != nil {
			t.Fatal(err)
		}
	}

	brief := briefFor(t, db, userID)

	if len(brief.Duplicates) != 0 {
		t.Fatalf("the offline algorithm reported %d duplicates; its ranges overlap with unrelated text", len(brief.Duplicates))
	}
	found := false
	for _, skip := range brief.Skipped {
		if skip.Kind == "duplicates" && skip.Reason == SkipBackendNotSemantic {
			found = true
		}
	}
	if !found {
		t.Fatalf("the brief did not say duplicates went unchecked: %+v", brief.Skipped)
	}
	// Something was skipped, so this is not a quiet morning — it is an unexamined
	// one, and those must not look the same.
	if brief.Quiet {
		t.Error("a brief that skipped a check reported itself as quiet")
	}
}

// With a backend fit to judge, the same account finds the pair — and nothing is
// marked unchecked.
func TestBriefFindsDuplicateThoughtsIntegration(t *testing.T) {
	db, userID := briefUser(t)
	ctx := context.Background()
	stopGateway := useSemanticStub(t, db, userID)
	defer stopGateway()

	space, err := db.CreateSpace(ctx, userID, "중복")
	if err != nil {
		t.Fatal(err)
	}
	// The stub returns the same vector for identical text, so an exact repeat is
	// a duplicate at 1.0 — enough to exercise the threshold and the reporting.
	for _, content := range []string{autoLinkNotes[0], autoLinkNotes[0], autoLinkNotes[3]} {
		if _, err = db.CreateNote(ctx, userID, Note{SpaceID: space.ID, AuthorID: userID, Content: content}); err != nil {
			t.Fatal(err)
		}
	}
	// Reading a space no longer indexes it. The brief catches up on its own, so
	// this is here to make the precondition explicit rather than incidental.
	db.SweepEmbeddingsOnce(ctx, &space.ID)

	brief := briefFor(t, db, userID)
	if len(brief.Skipped) != 0 {
		t.Fatalf("nothing should be unchecked with a semantic backend: %+v", brief.Skipped)
	}
	if len(brief.Duplicates) != 1 {
		t.Fatalf("expected exactly the repeated thought, got %d pairs", len(brief.Duplicates))
	}
	pair := brief.Duplicates[0]
	if pair.Score < db.IntelligenceSettings(ctx).DuplicateSimilarity {
		t.Errorf("score %.3f is below the threshold it was selected by", pair.Score)
	}
	if pair.First.Content != pair.Second.Content {
		t.Errorf("the pair is not the repeated thought: %q vs %q", pair.First.Content, pair.Second.Content)
	}
	if pair.Space != "중복" {
		t.Errorf("the pair does not name its space: %q", pair.Space)
	}
	// The unrelated third note must not be dragged in.
	if brief.Duplicates[0].First.Content == autoLinkNotes[3] || brief.Duplicates[0].Second.Content == autoLinkNotes[3] {
		t.Error("an unrelated thought was reported as a duplicate")
	}
}

// Captured thoughts that were never filed are the other thing waiting, and the
// brief is where a person would notice them piling up.
func TestBriefCountsUnfiledCapturesIntegration(t *testing.T) {
	db, userID := briefUser(t)
	ctx := context.Background()

	for _, content := range []string{"첫 생각", "둘째 생각", "셋째 생각"} {
		if _, err := db.CaptureThought(ctx, userID, content); err != nil {
			t.Fatal(err)
		}
	}
	if brief := briefFor(t, db, userID); brief.Unfiled != 3 {
		t.Errorf("unfiled=%d, want 3", brief.Unfiled)
	}

	// Filing one has to bring the count down, or the number is decoration.
	notes, _, err := db.ListNotes(ctx, userID, mustInbox(t, db, userID), "")
	if err != nil || len(notes) == 0 {
		t.Fatalf("list inbox: %v", err)
	}
	target, err := db.CreateSpace(ctx, userID, "정리함")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err = db.MoveNote(ctx, userID, notes[0].ID, target.ID); err != nil {
		t.Fatal(err)
	}
	if brief := briefFor(t, db, userID); brief.Unfiled != 2 {
		t.Errorf("after filing one, unfiled=%d, want 2", brief.Unfiled)
	}
}

// A count that includes another person's workspace would leak how much they are
// working, so the brief is scoped to what this person can reach.
func TestBriefDoesNotCountAnotherPersonsWorkIntegration(t *testing.T) {
	db, userID := briefUser(t)
	otherDB, strangerID := briefUser(t)
	ctx := context.Background()

	strangersSpace, err := otherDB.CreateSpace(ctx, strangerID, "남의 공간")
	if err != nil {
		t.Fatal(err)
	}
	first, err := otherDB.CreateNote(ctx, strangerID, Note{SpaceID: strangersSpace.ID, AuthorID: strangerID, Content: "가"})
	if err != nil {
		t.Fatal(err)
	}
	second, err := otherDB.CreateNote(ctx, strangerID, Note{SpaceID: strangersSpace.ID, AuthorID: strangerID, Content: "나"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = otherDB.Pool.Exec(ctx,
		`INSERT INTO note_edges(space_id,source_note_id,target_note_id,relation,origin,confidence,created_by)
		 VALUES($1,$2,$3,'related','auto',0.8,$4)`, strangersSpace.ID, first.ID, second.ID, strangerID); err != nil {
		t.Fatal(err)
	}

	if brief := briefFor(t, db, userID); brief.Suggestions != 0 {
		t.Errorf("suggestions=%d; the brief counted a workspace this person cannot see", brief.Suggestions)
	}
}

func mustInbox(t *testing.T, db *Store, userID uuid.UUID) uuid.UUID {
	t.Helper()
	inbox, err := db.InboxSpace(context.Background(), userID)
	if err != nil {
		t.Fatal(err)
	}
	return inbox.ID
}

// useOfflineEmbedding pins the store to the built-in algorithm.
//
// app_settings holds one global ai_gateway row, so a test that simply expects
// the default is at the mercy of whatever ran — or whoever clicked — before it.
// This makes the backend an input rather than an assumption, and fails loudly if
// the offline path is not what it got.
func useOfflineEmbedding(t *testing.T, db *Store) func() {
	t.Helper()
	ctx := context.Background()
	// Snapshot the row verbatim, including updated_by. Restoring through
	// PutSetting would need an actor, and a nil one violates the foreign key —
	// the write fails, the row stays deleted, and every later test and smoke run
	// on this database sees no gateway configured at all.
	var value []byte
	var updatedBy *uuid.UUID
	hadPrevious := db.Pool.QueryRow(ctx,
		`SELECT value,updated_by FROM app_settings WHERE key='ai_gateway'`).Scan(&value, &updatedBy) == nil
	if _, err := db.Pool.Exec(ctx, `DELETE FROM app_settings WHERE key='ai_gateway'`); err != nil {
		t.Fatal(err)
	}
	report, err := db.MeasureEmbeddingQuality(ctx, true)
	if err != nil {
		t.Fatal(err)
	}
	if report.Semantic {
		t.Fatalf("the offline algorithm reported itself as semantic; this test would prove nothing")
	}
	return func() {
		if !hadPrevious {
			return
		}
		if _, err := db.Pool.Exec(context.Background(),
			`INSERT INTO app_settings(key,value,updated_by,updated_at) VALUES('ai_gateway',$1,$2,now())
			 ON CONFLICT(key) DO UPDATE SET value=EXCLUDED.value,updated_by=EXCLUDED.updated_by`,
			value, updatedBy); err != nil {
			// Leaving the row missing breaks every later test and smoke run
			// against this database, so this must not pass quietly.
			t.Errorf("failed to restore the ai_gateway setting: %v", err)
		}
	}
}
