package store

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

// Looking backwards is only worth anything if what comes back is what was
// actually there. Every case here is a way the reconstruction could quietly
// return today instead of then.

func rewindSpace(t *testing.T) (*Store, uuid.UUID, uuid.UUID) {
	t.Helper()
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		t.Skip("POSTGRES_DSN is not configured")
	}
	db := isolatedStore(t, dsn)
	ctx := context.Background()
	userID, spaceID := uuid.New(), uuid.New()
	name := "rewind_" + strings.ReplaceAll(userID.String(), "-", "")
	if _, err := db.Pool.Exec(ctx, `INSERT INTO users(id,username,display_name) VALUES($1,$2::citext,$2::text)`, userID, name); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Pool.Exec(ctx, `INSERT INTO spaces(id,owner_id,name) VALUES($1,$2,'되감기 공간')`, spaceID, userID); err != nil {
		t.Fatal(err)
	}
	return db, userID, spaceID
}

// backdate moves a row's timestamps so a test can have a past without waiting
// for one.
func backdate(t *testing.T, db *Store, table, column string, id uuid.UUID, ago time.Duration) {
	t.Helper()
	if _, err := db.Pool.Exec(context.Background(),
		"UPDATE "+table+" SET "+column+"=now()-$2::interval WHERE id=$1", id, ago.String()); err != nil {
		t.Fatal(err)
	}
}

func TestSpaceAtGivesTheWordsAndThePlaceItHadThenIntegration(t *testing.T) {
	db, userID, spaceID := rewindSpace(t)
	ctx := context.Background()

	note, err := db.CreateNote(ctx, userID, Note{SpaceID: spaceID, AuthorID: userID, Content: "처음 쓴 문장", X: 10, Y: 20})
	if err != nil {
		t.Fatal(err)
	}
	backdate(t, db, "notes", "created_at", note.ID, 48*time.Hour)

	note.Content = "고쳐 쓴 문장"
	note.X, note.Y = 500, 600
	if _, err := db.UpdateNote(ctx, userID, note, nil); err != nil {
		t.Fatal(err)
	}

	// A day ago is after the note was written and before it was changed.
	snapshot, err := db.SpaceAt(ctx, userID, spaceID, time.Now().Add(-24*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Notes) != 1 {
		t.Fatalf("%d notes in the snapshot", len(snapshot.Notes))
	}
	got := snapshot.Notes[0]
	if got.Content != "처음 쓴 문장" {
		t.Errorf("content came back as %q — that is today, not then", got.Content)
	}
	if got.X != 10 || got.Y != 20 {
		t.Errorf("the note is at (%v,%v); it sat at (10,20) then", got.X, got.Y)
	}

	// And now really is different, so the assertion above is the rewind working
	// rather than the note never having changed.
	current, err := db.SpaceAt(ctx, userID, spaceID, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if current.Notes[0].Content != "고쳐 쓴 문장" || current.Notes[0].X != 500 {
		t.Fatalf("the note did not actually change: %+v", current.Notes[0])
	}
}

// A thought written after the moment being looked at was not on the canvas.
func TestSpaceAtLeavesOutWhatWasNotWrittenYetIntegration(t *testing.T) {
	db, userID, spaceID := rewindSpace(t)
	ctx := context.Background()

	old, err := db.CreateNote(ctx, userID, Note{SpaceID: spaceID, AuthorID: userID, Content: "먼저 쓴 것"})
	if err != nil {
		t.Fatal(err)
	}
	backdate(t, db, "notes", "created_at", old.ID, 48*time.Hour)
	if _, err := db.CreateNote(ctx, userID, Note{SpaceID: spaceID, AuthorID: userID, Content: "나중에 쓴 것"}); err != nil {
		t.Fatal(err)
	}

	snapshot, err := db.SpaceAt(ctx, userID, spaceID, time.Now().Add(-24*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Notes) != 1 || snapshot.Notes[0].Content != "먼저 쓴 것" {
		t.Fatalf("a thought from the future is in the past: %+v", snapshot.Notes)
	}
}

// A thought deleted since was on the canvas then, and has to come back.
func TestSpaceAtBringsBackWhatWasLaterDeletedIntegration(t *testing.T) {
	db, userID, spaceID := rewindSpace(t)
	ctx := context.Background()

	note, err := db.CreateNote(ctx, userID, Note{SpaceID: spaceID, AuthorID: userID, Content: "나중에 지운 생각"})
	if err != nil {
		t.Fatal(err)
	}
	backdate(t, db, "notes", "created_at", note.ID, 48*time.Hour)
	if _, err := db.Pool.Exec(ctx, `UPDATE notes SET deleted_at=now() WHERE id=$1`, note.ID); err != nil {
		t.Fatal(err)
	}

	past, err := db.SpaceAt(ctx, userID, spaceID, time.Now().Add(-24*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if len(past.Notes) != 1 {
		t.Fatalf("the deleted thought did not come back: %+v", past.Notes)
	}
	// And it is gone from today, so the line above is the rewind rather than a
	// delete that never happened.
	now, err := db.SpaceAt(ctx, userID, spaceID, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if len(now.Notes) != 0 {
		t.Fatalf("the thought is still on today's canvas: %+v", now.Notes)
	}
}

// A connection cannot hang off a thought that was not there yet.
func TestSpaceAtDrawsNoConnectionToAThoughtThatDidNotExistIntegration(t *testing.T) {
	db, userID, spaceID := rewindSpace(t)
	ctx := context.Background()

	old, err := db.CreateNote(ctx, userID, Note{SpaceID: spaceID, AuthorID: userID, Content: "오래된 생각"})
	if err != nil {
		t.Fatal(err)
	}
	backdate(t, db, "notes", "created_at", old.ID, 48*time.Hour)
	recent, err := db.CreateNote(ctx, userID, Note{SpaceID: spaceID, AuthorID: userID, Content: "최근 생각"})
	if err != nil {
		t.Fatal(err)
	}
	edge, err := db.CreateEdge(ctx, userID, Edge{
		SpaceID: spaceID, SourceID: recent.ID, TargetID: old.ID, Relation: RelationSupports,
	})
	if err != nil {
		t.Fatal(err)
	}
	// The connection is backdated but one of its ends is not, which is the case
	// that would draw a line into nothing.
	backdate(t, db, "note_edges", "created_at", edge.ID, 48*time.Hour)

	snapshot, err := db.SpaceAt(ctx, userID, spaceID, time.Now().Add(-24*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Edges) != 0 {
		t.Fatalf("a connection was drawn to a thought that did not exist yet: %+v", snapshot.Edges)
	}
}

// A removed connection cannot be drawn — the deletion event has only ever
// carried the edge's id. Saying how many is the honest half.
func TestSpaceAtCountsTheConnectionsItCannotReplayIntegration(t *testing.T) {
	db, userID, spaceID := rewindSpace(t)
	ctx := context.Background()

	first, err := db.CreateNote(ctx, userID, Note{SpaceID: spaceID, AuthorID: userID, Content: "가"})
	if err != nil {
		t.Fatal(err)
	}
	second, err := db.CreateNote(ctx, userID, Note{SpaceID: spaceID, AuthorID: userID, Content: "나"})
	if err != nil {
		t.Fatal(err)
	}
	backdate(t, db, "notes", "created_at", first.ID, 48*time.Hour)
	backdate(t, db, "notes", "created_at", second.ID, 48*time.Hour)
	edge, err := db.CreateEdge(ctx, userID, Edge{
		SpaceID: spaceID, SourceID: second.ID, TargetID: first.ID, Relation: RelationSupports,
	})
	if err != nil {
		t.Fatal(err)
	}
	backdate(t, db, "note_edges", "created_at", edge.ID, 48*time.Hour)
	if err := db.DeleteEdge(ctx, userID, edge.ID); err != nil {
		t.Fatal(err)
	}

	snapshot, err := db.SpaceAt(ctx, userID, spaceID, time.Now().Add(-24*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Edges) != 0 {
		t.Fatalf("a deleted connection was drawn from somewhere: %+v", snapshot.Edges)
	}
	if snapshot.RemovedEdges != 1 {
		t.Fatalf("RemovedEdges=%d; the snapshot does not say a connection is missing", snapshot.RemovedEdges)
	}
}

// "You cannot see this" and "this was empty then" are different answers.
func TestSpaceAtRefusesSomeoneWhoCannotSeeTheSpaceIntegration(t *testing.T) {
	db, userID, spaceID := rewindSpace(t)
	ctx := context.Background()
	if _, err := db.CreateNote(ctx, userID, Note{SpaceID: spaceID, AuthorID: userID, Content: "남의 생각"}); err != nil {
		t.Fatal(err)
	}

	if _, err := db.SpaceAt(ctx, uuid.New(), spaceID, time.Now()); !errors.Is(err, ErrSpaceNotVisible) {
		t.Fatalf("a stranger got a snapshot: %v", err)
	}
	// And the owner still gets one, so the refusal is about access.
	if _, err := db.SpaceAt(ctx, userID, spaceID, time.Now()); err != nil {
		t.Fatalf("the owner was refused: %v", err)
	}
}

// Two edits, and a moment between the first and the second.
//
// A revision holds the state before the change that created it, so the state
// at an instant is the *earliest* revision recorded after it. Every other test
// here edits once, and with one edit "earliest after" and "latest after" pick
// the same row — so none of them can tell the rule from its opposite. This one
// can: at a moment before both edits, the answer is the first text, and taking
// the latest revision instead would return the second.
func TestSpaceAtPicksTheVersionThatWasLiveThenIntegration(t *testing.T) {
	db, userID, spaceID := rewindSpace(t)
	ctx := context.Background()

	note, err := db.CreateNote(ctx, userID, Note{SpaceID: spaceID, AuthorID: userID, Content: "첫 번째 문장"})
	if err != nil {
		t.Fatal(err)
	}
	note.Content = "두 번째 문장"
	if note, err = db.UpdateNote(ctx, userID, note, nil); err != nil {
		t.Fatal(err)
	}
	note.Content = "세 번째 문장"
	if _, err = db.UpdateNote(ctx, userID, note, nil); err != nil {
		t.Fatal(err)
	}

	// The note was written three days ago; the two edits landed two days and
	// one day ago.
	backdate(t, db, "notes", "created_at", note.ID, 72*time.Hour)
	for version, ago := range map[int]time.Duration{1: 48 * time.Hour, 2: 24 * time.Hour} {
		if _, err := db.Pool.Exec(ctx,
			`UPDATE note_revisions SET created_at=now()-$3::interval WHERE note_id=$1 AND version=$2`,
			note.ID, version, ago.String()); err != nil {
			t.Fatal(err)
		}
	}

	for _, want := range []struct {
		ago  time.Duration
		text string
	}{
		{60 * time.Hour, "첫 번째 문장"}, // before either edit
		{36 * time.Hour, "두 번째 문장"}, // after the first, before the second
		{1 * time.Hour, "세 번째 문장"},  // after both
	} {
		snapshot, err := db.SpaceAt(ctx, userID, spaceID, time.Now().Add(-want.ago))
		if err != nil {
			t.Fatal(err)
		}
		if len(snapshot.Notes) != 1 {
			t.Fatalf("%v ago: %d notes", want.ago, len(snapshot.Notes))
		}
		if got := snapshot.Notes[0].Content; got != want.text {
			t.Errorf("%v ago the note read %q, want %q", want.ago, got, want.text)
		}
	}
}

// A picture is dated like everything else. A canvas showing two days ago must
// not show the photograph pasted this morning — it is the one part of a thought
// whose age nobody can tell by reading it, so if the snapshot does not date it
// nothing else will.
func TestSpaceAtLeavesOutAPictureAddedSinceIntegration(t *testing.T) {
	db, userID, spaceID := rewindSpace(t)
	ctx := context.Background()

	note, err := db.CreateNote(ctx, userID, Note{SpaceID: spaceID, AuthorID: userID, Content: "그림이 붙는 생각"})
	if err != nil {
		t.Fatal(err)
	}
	backdate(t, db, "notes", "created_at", note.ID, 72*time.Hour)

	then, err := db.AttachToNote(ctx, userID, note.ID, "그때 붙인 화이트보드.png", realPNG(t))
	if err != nil {
		t.Fatal(err)
	}
	backdate(t, db, "note_attachments", "created_at", then.ID, 48*time.Hour)
	later, err := db.AttachToNote(ctx, userID, note.ID, "오늘 붙인 캡처.png", realPNG(t))
	if err != nil {
		t.Fatal(err)
	}

	past, err := db.SpaceAt(ctx, userID, spaceID, time.Now().Add(-24*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if len(past.Attachments) != 1 || past.Attachments[0].ID != then.ID {
		t.Fatalf("the past canvas is holding a picture from the future: %+v", past.Attachments)
	}
	if past.RemovedAttachments != 0 {
		t.Errorf("RemovedAttachments=%d; nothing was removed", past.RemovedAttachments)
	}

	// Both are on today's canvas, so the line above is the dating rather than
	// the second picture never having been stored.
	now, err := db.SpaceAt(ctx, userID, spaceID, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if len(now.Attachments) != 2 {
		t.Fatalf("today's canvas holds %d pictures, want both: %+v", len(now.Attachments), now.Attachments)
	}
	if now.Attachments[1].ID != later.ID || now.Attachments[1].ContentType != "image/png" {
		t.Errorf("the second picture came back as %+v", now.Attachments[1])
	}
}

// A thought deleted since comes back, and it had a picture. The bytes are
// served against a thought that is not deleted, so the picture cannot be drawn
// — saying how many is the honest half, the same as for a removed connection.
func TestSpaceAtCountsThePicturesItCannotDrawIntegration(t *testing.T) {
	db, userID, spaceID := rewindSpace(t)
	ctx := context.Background()

	note, err := db.CreateNote(ctx, userID, Note{SpaceID: spaceID, AuthorID: userID, Content: "나중에 지운 생각"})
	if err != nil {
		t.Fatal(err)
	}
	backdate(t, db, "notes", "created_at", note.ID, 72*time.Hour)
	picture, err := db.AttachToNote(ctx, userID, note.ID, "지운 생각의 사진.png", realPNG(t))
	if err != nil {
		t.Fatal(err)
	}
	backdate(t, db, "note_attachments", "created_at", picture.ID, 48*time.Hour)
	if _, err := db.Pool.Exec(ctx, `UPDATE notes SET deleted_at=now() WHERE id=$1`, note.ID); err != nil {
		t.Fatal(err)
	}

	past, err := db.SpaceAt(ctx, userID, spaceID, time.Now().Add(-24*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if len(past.Notes) != 1 {
		t.Fatalf("the deleted thought did not come back: %+v", past.Notes)
	}
	if len(past.Attachments) != 0 {
		t.Errorf("a picture nobody can fetch was handed to the canvas: %+v", past.Attachments)
	}
	if past.RemovedAttachments != 1 {
		t.Fatalf("RemovedAttachments=%d; the thought comes back looking as though it never had a picture", past.RemovedAttachments)
	}
}
