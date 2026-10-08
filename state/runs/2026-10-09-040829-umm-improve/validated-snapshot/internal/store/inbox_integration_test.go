package store

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func inboxUser(t *testing.T) (*Store, uuid.UUID) {
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
	name := "inbox_" + strings.ReplaceAll(userID.String(), "-", "")
	if _, err = db.Pool.Exec(ctx, `INSERT INTO users(id,username,display_name) VALUES($1,$2::citext,$2::text)`, userID, name); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Pool.Exec(context.Background(), `DELETE FROM users WHERE id=$1`, userID) })
	if _, err = db.Pool.Exec(ctx, `INSERT INTO user_preferences(user_id) VALUES($1)`, userID); err != nil {
		t.Fatal(err)
	}
	return db, userID
}

// Capture has to work the first time, before any space exists, or it is not
// capture — it is the same "choose a space first" it replaces.
func TestCaptureCreatesTheInboxOnFirstUseIntegration(t *testing.T) {
	db, userID := inboxUser(t)
	ctx := context.Background()

	note, err := db.CaptureThought(ctx, userID, "회고 때 물어볼 것 하나")
	if err != nil {
		t.Fatalf("capture: %v", err)
	}
	inbox, err := db.InboxSpace(ctx, userID)
	if err != nil {
		t.Fatalf("inbox space: %v", err)
	}
	if note.SpaceID != inbox.ID {
		t.Errorf("the thought landed in %v, not the inbox %v", note.SpaceID, inbox.ID)
	}
	if !inbox.IsInbox {
		t.Error("the inbox space does not report itself as one; the interface cannot mark it")
	}

	// Capturing again must reuse the same inbox rather than make another.
	if _, err = db.CaptureThought(ctx, userID, "두 번째 생각"); err != nil {
		t.Fatalf("second capture: %v", err)
	}
	var inboxes int
	if err = db.Pool.QueryRow(ctx, `SELECT count(*) FROM spaces WHERE owner_id=$1 AND is_inbox`, userID).Scan(&inboxes); err != nil {
		t.Fatal(err)
	}
	if inboxes != 1 {
		t.Fatalf("%d inbox spaces exist; captures would be scattered across them", inboxes)
	}
}

// Deleting the inbox would leave captures with nowhere to land, and the next one
// would silently make a new one — losing whatever was in the old.
func TestInboxCannotBeDeletedIntegration(t *testing.T) {
	db, userID := inboxUser(t)
	ctx := context.Background()
	inbox, err := db.InboxSpace(ctx, userID)
	if err != nil {
		t.Fatal(err)
	}
	if err = db.DeleteSpace(ctx, userID, inbox.ID); !errors.Is(err, ErrInboxSpace) {
		t.Fatalf("deleting the inbox returned %v; it must be refused with a reason", err)
	}
	var alive bool
	if err = db.Pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM spaces WHERE id=$1)`, inbox.ID).Scan(&alive); err != nil {
		t.Fatal(err)
	}
	if !alive {
		t.Fatal("the inbox was deleted anyway")
	}
	// An ordinary space is still deletable; the guard must not be a blanket one.
	other, err := db.CreateSpace(ctx, userID, "보통 공간")
	if err != nil {
		t.Fatal(err)
	}
	if err = db.DeleteSpace(ctx, userID, other.ID); err != nil {
		t.Errorf("an ordinary space could not be deleted: %v", err)
	}
}

// Filing is the other half of capture. Without it the inbox is a place thoughts
// go to be forgotten.
func TestFilingAThoughtMovesItIntegration(t *testing.T) {
	db, userID := inboxUser(t)
	ctx := context.Background()

	note, err := db.CaptureThought(ctx, userID, "인증 토큰 만료를 24시간으로")
	if err != nil {
		t.Fatal(err)
	}
	target, err := db.CreateSpace(ctx, userID, "보안")
	if err != nil {
		t.Fatal(err)
	}
	moved, removed, err := db.MoveNote(ctx, userID, note.ID, target.ID)
	if err != nil {
		t.Fatalf("move: %v", err)
	}
	if moved.SpaceID != target.ID {
		t.Errorf("the thought is in %v, not the destination %v", moved.SpaceID, target.ID)
	}
	if removed != 0 {
		t.Errorf("a freshly captured thought had %d connections to remove", removed)
	}
	if moved.Version <= note.Version {
		t.Error("the version did not advance; a concurrent editor would not notice the move")
	}
}

// A connection is scoped to a space and needs both endpoints in it. A note that
// leaves cannot keep them, and the caller has to be told rather than find out
// afterwards.
func TestMovingAConnectedThoughtReportsWhatItCostsIntegration(t *testing.T) {
	db, userID := inboxUser(t)
	ctx := context.Background()

	origin, err := db.CreateSpace(ctx, userID, "출발")
	if err != nil {
		t.Fatal(err)
	}
	first, err := db.CreateNote(ctx, userID, Note{SpaceID: origin.ID, AuthorID: userID, Content: "생각 하나"})
	if err != nil {
		t.Fatal(err)
	}
	second, err := db.CreateNote(ctx, userID, Note{SpaceID: origin.ID, AuthorID: userID, Content: "생각 둘"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.CreateEdge(ctx, userID, Edge{
		SpaceID: origin.ID, SourceID: first.ID, TargetID: second.ID, Relation: RelationSupports,
	}); err != nil {
		t.Fatal(err)
	}
	target, err := db.CreateSpace(ctx, userID, "도착")
	if err != nil {
		t.Fatal(err)
	}
	_, removed, err := db.MoveNote(ctx, userID, first.ID, target.ID)
	if err != nil {
		t.Fatalf("move: %v", err)
	}
	if removed != 1 {
		t.Errorf("removedEdges=%d; the caller cannot warn about what the move costs", removed)
	}
	var left int
	if err = db.Pool.QueryRow(ctx, `SELECT count(*) FROM note_edges WHERE source_note_id=$1 OR target_note_id=$1`, first.ID).Scan(&left); err != nil {
		t.Fatal(err)
	}
	if left != 0 {
		t.Errorf("%d connections survived the move and now span two spaces", left)
	}
}

// A move must not become a way into a space the person cannot write to.
func TestMoveRefusesADestinationWithoutWriteAccessIntegration(t *testing.T) {
	db, userID := inboxUser(t)
	otherDB, strangerID := inboxUser(t)
	ctx := context.Background()

	note, err := db.CaptureThought(ctx, userID, "옮길 생각")
	if err != nil {
		t.Fatal(err)
	}
	strangersSpace, err := otherDB.CreateSpace(ctx, strangerID, "남의 공간")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err = db.MoveNote(ctx, userID, note.ID, strangersSpace.ID); err == nil {
		t.Fatal("a thought was moved into a space the person has no write access to")
	}
	var stayed uuid.UUID
	if err = db.Pool.QueryRow(ctx, `SELECT space_id FROM notes WHERE id=$1`, note.ID).Scan(&stayed); err != nil {
		t.Fatal(err)
	}
	if stayed == strangersSpace.ID {
		t.Fatal("the note reached the other person's space despite the refusal")
	}
}

// Where a thought belongs is a judgement, and umm should only claim to make it
// when it can. On a lexical backend the honest answer is recency, not vocabulary
// overlap dressed up as understanding.
func TestSpaceSuggestionsSayWhatTheyAreBasedOnIntegration(t *testing.T) {
	db, userID := inboxUser(t)
	ctx := context.Background()

	note, err := db.CaptureThought(ctx, userID, "세션 쿠키 설정을 정리하자")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"보안", "자전거"} {
		space, createErr := db.CreateSpace(ctx, userID, name)
		if createErr != nil {
			t.Fatal(createErr)
		}
		if _, createErr = db.CreateNote(ctx, userID, Note{SpaceID: space.ID, AuthorID: userID, Content: name + " 메모"}); createErr != nil {
			t.Fatal(createErr)
		}
	}
	suggestions, err := db.SuggestSpaces(ctx, userID, note.ID, 3)
	if err != nil {
		t.Fatalf("suggest spaces: %v", err)
	}
	if len(suggestions) == 0 {
		t.Fatal("no destination was offered at all")
	}
	for _, suggestion := range suggestions {
		if suggestion.Basis != BasisRecent && suggestion.Basis != BasisMeaning {
			t.Errorf("basis %q does not say what the ranking rests on", suggestion.Basis)
		}
		if suggestion.Space.IsInbox {
			t.Error("the inbox was offered as a destination for a thought already in it")
		}
	}
	// The default backend is lexical, so umm must not claim to be ranking by
	// meaning. This is the same line auto-link draws.
	if suggestions[0].Basis != BasisRecent {
		t.Errorf("basis=%q on the offline embedding; umm claimed an understanding it was measured not to have",
			suggestions[0].Basis)
	}
}

// The bug this pins: loadEmbeddings only reads stored vectors, and a captured
// thought has none until its space is listed. Comparing against a missing vector
// scores every candidate at zero — a ranking that looks ordered but is not, and
// which the interface would present as a judgement.
func TestSpaceSuggestionsProduceRealScoresIntegration(t *testing.T) {
	db, userID := inboxUser(t)
	ctx := context.Background()
	stopGateway := useSemanticStub(t, db, userID)
	defer stopGateway()

	note, err := db.CaptureThought(ctx, userID, autoLinkNotes[0])
	if err != nil {
		t.Fatal(err)
	}
	security, err := db.CreateSpace(ctx, userID, "보안")
	if err != nil {
		t.Fatal(err)
	}
	cycling, err := db.CreateSpace(ctx, userID, "자전거")
	if err != nil {
		t.Fatal(err)
	}
	for _, content := range autoLinkNotes[1:3] {
		if _, err = db.CreateNote(ctx, userID, Note{SpaceID: security.ID, AuthorID: userID, Content: content}); err != nil {
			t.Fatal(err)
		}
	}
	for _, content := range autoLinkNotes[3:] {
		if _, err = db.CreateNote(ctx, userID, Note{SpaceID: cycling.ID, AuthorID: userID, Content: content}); err != nil {
			t.Fatal(err)
		}
	}

	suggestions, err := db.SuggestSpaces(ctx, userID, note.ID, 3)
	if err != nil {
		t.Fatalf("suggest spaces: %v", err)
	}
	if len(suggestions) < 2 {
		t.Fatalf("expected both spaces to be ranked, got %d", len(suggestions))
	}
	if suggestions[0].Basis != BasisMeaning {
		t.Fatalf("basis=%q with a semantic backend", suggestions[0].Basis)
	}
	if suggestions[0].Score <= 0 {
		t.Fatal("every candidate scored zero; the order is not a ranking")
	}
	// The thought is one of the security notes, so that space has to win — and
	// win by a margin, or the score is not carrying information.
	if suggestions[0].Space.ID != security.ID {
		t.Errorf("a security thought was filed toward %q", suggestions[0].Space.Name)
	}
	if suggestions[0].Score <= suggestions[1].Score {
		t.Errorf("the two spaces scored %.3f and %.3f; the ranking does not separate them",
			suggestions[0].Score, suggestions[1].Score)
	}
}
