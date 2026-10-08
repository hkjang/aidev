package store

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func mergeSpace(t *testing.T) (*Store, uuid.UUID, uuid.UUID) {
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
	userID, spaceID := uuid.New(), uuid.New()
	name := "merge_" + strings.ReplaceAll(userID.String(), "-", "")
	if _, err = db.Pool.Exec(ctx, `INSERT INTO users(id,username,display_name) VALUES($1,$2::citext,$2::text)`, userID, name); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Pool.Exec(context.Background(), `DELETE FROM users WHERE id=$1`, userID) })
	if _, err = db.Pool.Exec(ctx, `INSERT INTO user_preferences(user_id) VALUES($1)`, userID); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Pool.Exec(ctx, `INSERT INTO spaces(id,owner_id,name) VALUES($1,$2,'merge')`, spaceID, userID); err != nil {
		t.Fatal(err)
	}
	return db, userID, spaceID
}

func note(t *testing.T, db *Store, userID, spaceID uuid.UUID, content string) Note {
	t.Helper()
	n, err := db.CreateNote(context.Background(), userID, Note{SpaceID: spaceID, AuthorID: userID, Content: content})
	if err != nil {
		t.Fatal(err)
	}
	return n
}

// Connections belong to the thought, not to the row that happened to hold it.
// Losing them on merge would quietly destroy the graph umm spent releases
// learning to record.
func TestMergeCarriesConnectionsToTheSurvivorIntegration(t *testing.T) {
	db, userID, spaceID := mergeSpace(t)
	ctx := context.Background()

	keep := note(t, db, userID, spaceID, "인증 토큰 만료를 24시간으로")
	drop := note(t, db, userID, spaceID, "인증 토큰 만료 시간 24시간")
	other := note(t, db, userID, spaceID, "세션 쿠키 설정")
	third := note(t, db, userID, spaceID, "로그인 잠금")

	// Outgoing from the losing note, and incoming to it.
	if _, err := db.CreateEdge(ctx, userID, Edge{SpaceID: spaceID, SourceID: drop.ID, TargetID: other.ID, Relation: RelationSupports}); err != nil {
		t.Fatal(err)
	}
	if _, err := db.CreateEdge(ctx, userID, Edge{SpaceID: spaceID, SourceID: third.ID, TargetID: drop.ID, Relation: RelationRefines}); err != nil {
		t.Fatal(err)
	}

	result, err := db.MergeNotes(ctx, userID, keep.ID, drop.ID, "인증 토큰 만료를 24시간으로")
	if err != nil {
		t.Fatalf("merge: %v", err)
	}
	if result.MovedEdges != 2 {
		t.Errorf("movedEdges=%d, want 2", result.MovedEdges)
	}

	var attached int
	if err = db.Pool.QueryRow(ctx,
		`SELECT count(*) FROM note_edges WHERE source_note_id=$1 OR target_note_id=$1`, keep.ID).Scan(&attached); err != nil {
		t.Fatal(err)
	}
	if attached != 2 {
		t.Errorf("the survivor has %d connections, want 2", attached)
	}
	var orphaned int
	if err = db.Pool.QueryRow(ctx,
		`SELECT count(*) FROM note_edges WHERE source_note_id=$1 OR target_note_id=$1`, drop.ID).Scan(&orphaned); err != nil {
		t.Fatal(err)
	}
	if orphaned != 0 {
		t.Errorf("%d connections still point at the merged-away note", orphaned)
	}
}

// A connection between the two notes being merged would become a note connected
// to itself, and one the survivor already has would become a duplicate. Both are
// dropped, and the count says so rather than the difference going unexplained.
func TestMergeDropsSelfAndDuplicateConnectionsIntegration(t *testing.T) {
	db, userID, spaceID := mergeSpace(t)
	ctx := context.Background()

	keep := note(t, db, userID, spaceID, "생각 A")
	drop := note(t, db, userID, spaceID, "생각 A 사본")
	shared := note(t, db, userID, spaceID, "둘 다 가리키는 생각")

	for _, edge := range []Edge{
		{SpaceID: spaceID, SourceID: keep.ID, TargetID: drop.ID, Relation: RelationRelated},
		{SpaceID: spaceID, SourceID: keep.ID, TargetID: shared.ID, Relation: RelationSupports},
		{SpaceID: spaceID, SourceID: drop.ID, TargetID: shared.ID, Relation: RelationSupports},
	} {
		if _, err := db.CreateEdge(ctx, userID, edge); err != nil {
			t.Fatal(err)
		}
	}

	result, err := db.MergeNotes(ctx, userID, keep.ID, drop.ID, "생각 A")
	if err != nil {
		t.Fatalf("merge: %v", err)
	}
	if result.DroppedEdges != 2 {
		t.Errorf("droppedEdges=%d, want 2 (the pair's own link and the duplicate)", result.DroppedEdges)
	}
	var selfLinked int
	if err = db.Pool.QueryRow(ctx,
		`SELECT count(*) FROM note_edges WHERE source_note_id=target_note_id`).Scan(&selfLinked); err != nil {
		t.Fatal(err)
	}
	if selfLinked != 0 {
		t.Errorf("%d notes ended up connected to themselves", selfLinked)
	}
	var toShared int
	if err = db.Pool.QueryRow(ctx,
		`SELECT count(*) FROM note_edges WHERE source_note_id=$1 AND target_note_id=$2`, keep.ID, shared.ID).Scan(&toShared); err != nil {
		t.Fatal(err)
	}
	if toShared != 1 {
		t.Errorf("the survivor has %d connections to the shared note, want exactly 1", toShared)
	}
}

// The discussion is about the thought. Leaving it on the row that disappears
// would delete it with the note.
func TestMergeCarriesTheDiscussionIntegration(t *testing.T) {
	db, userID, spaceID := mergeSpace(t)
	ctx := context.Background()
	keep := note(t, db, userID, spaceID, "남는 생각")
	drop := note(t, db, userID, spaceID, "합쳐질 생각")

	if _, err := db.Pool.Exec(ctx,
		`INSERT INTO note_comments(note_id,author_id,body) VALUES($1,$2,'여기 중요한 맥락')`, drop.ID, userID); err != nil {
		t.Fatal(err)
	}
	result, err := db.MergeNotes(ctx, userID, keep.ID, drop.ID, "남는 생각")
	if err != nil {
		t.Fatalf("merge: %v", err)
	}
	if result.MovedComments != 1 {
		t.Errorf("movedComments=%d, want 1", result.MovedComments)
	}
	var body string
	if err = db.Pool.QueryRow(ctx, `SELECT body FROM note_comments WHERE note_id=$1`, keep.ID).Scan(&body); err != nil {
		t.Fatalf("the comment did not survive: %v", err)
	}
	if body != "여기 중요한 맥락" {
		t.Errorf("comment=%q", body)
	}
}

// The stored vector describes text that no longer exists, so leaving it would
// have every later comparison judge the survivor by its old wording.
func TestMergeInvalidatesTheStoredVectorIntegration(t *testing.T) {
	db, userID, spaceID := mergeSpace(t)
	ctx := context.Background()
	keep := note(t, db, userID, spaceID, "원래 문장")
	drop := note(t, db, userID, spaceID, "비슷한 문장")
	// Listing is what produces vectors.
	if _, _, err := db.ListNotes(ctx, userID, spaceID, ""); err != nil {
		t.Fatal(err)
	}
	var before int
	if err := db.Pool.QueryRow(ctx, `SELECT count(*) FROM note_embeddings WHERE note_id=$1`, keep.ID).Scan(&before); err != nil {
		t.Fatal(err)
	}
	if before == 0 {
		t.Skip("no vector was stored, so there is nothing to invalidate")
	}
	if _, err := db.MergeNotes(ctx, userID, keep.ID, drop.ID, "합쳐진 새 문장"); err != nil {
		t.Fatalf("merge: %v", err)
	}
	var after int
	if err := db.Pool.QueryRow(ctx, `SELECT count(*) FROM note_embeddings WHERE note_id=$1`, keep.ID).Scan(&after); err != nil {
		t.Fatal(err)
	}
	if after != 0 {
		t.Error("the survivor kept a vector describing the text it no longer has")
	}
}

// Merging is destructive, so the ways it must refuse matter as much as the ways
// it works.
func TestMergeRefusesWhatItCannotDoSafelyIntegration(t *testing.T) {
	db, userID, spaceID := mergeSpace(t)
	ctx := context.Background()
	keep := note(t, db, userID, spaceID, "생각")

	if _, err := db.MergeNotes(ctx, userID, keep.ID, keep.ID, "생각"); !errors.Is(err, ErrMergeSameNote) {
		t.Errorf("merging a note into itself returned %v", err)
	}
	if _, err := db.MergeNotes(ctx, userID, keep.ID, uuid.New(), "생각"); err == nil {
		t.Error("merging with a note that does not exist succeeded")
	}
	// Empty content would leave the surviving thought saying nothing.
	other := note(t, db, userID, spaceID, "다른 생각")
	if _, err := db.MergeNotes(ctx, userID, keep.ID, other.ID, "   "); err == nil {
		t.Error("a merge was allowed to blank the surviving note")
	}

	// Across spaces a connection cannot keep both endpoints, so the merge is
	// refused rather than silently moving a thought or stranding its links.
	elsewhere := uuid.New()
	if _, err := db.Pool.Exec(ctx, `INSERT INTO spaces(id,owner_id,name) VALUES($1,$2,'다른 공간')`, elsewhere, userID); err != nil {
		t.Fatal(err)
	}
	far := note(t, db, userID, elsewhere, "먼 생각")
	if _, err := db.MergeNotes(ctx, userID, keep.ID, far.ID, "생각"); err == nil {
		t.Error("notes in different spaces were merged")
	}

	// And nothing was destroyed along the way.
	var alive int
	if err := db.Pool.QueryRow(ctx,
		`SELECT count(*) FROM notes WHERE id IN ($1,$2,$3) AND deleted_at IS NULL`, keep.ID, other.ID, far.ID).Scan(&alive); err != nil {
		t.Fatal(err)
	}
	if alive != 3 {
		t.Errorf("%d of 3 notes survived the refused merges", alive)
	}
}

// A merge must not become a way to delete someone else's thought.
func TestMergeRefusesWithoutWriteAccessIntegration(t *testing.T) {
	db, userID, spaceID := mergeSpace(t)
	_, strangerID, strangerSpace := mergeSpace(t)
	ctx := context.Background()

	mine := note(t, db, userID, spaceID, "내 생각")
	theirs := note(t, db, strangerID, strangerSpace, "남의 생각")

	if _, err := db.MergeNotes(ctx, userID, mine.ID, theirs.ID, "내 생각"); err == nil {
		t.Fatal("a note in another person's space was merged away")
	}
	var alive bool
	if err := db.Pool.QueryRow(ctx,
		`SELECT deleted_at IS NULL FROM notes WHERE id=$1`, theirs.ID).Scan(&alive); err != nil {
		t.Fatal(err)
	}
	if !alive {
		t.Fatal("the other person's note was deleted")
	}
}
