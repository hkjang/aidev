package store

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
)

// The why behind a connection is the half that disappears first from anybody's
// memory, and the half a database is actually good at keeping. What is checked
// here is that it survives the write, comes back on the read, and can be added
// after the fact — because at the moment a line is drawn is exactly when the
// reason is least likely to be known.

func edgeReasonSpace(t *testing.T) (*Store, uuid.UUID, uuid.UUID, Note, Note) {
	t.Helper()
	db, userID, spaceID := contradictionSpace(t)
	ctx := context.Background()
	first, err := db.CreateNote(ctx, userID, Note{SpaceID: spaceID, AuthorID: userID, Content: "회고를 격주로 줄이자"})
	if err != nil {
		t.Fatal(err)
	}
	second, err := db.CreateNote(ctx, userID, Note{SpaceID: spaceID, AuthorID: userID, Content: "주기가 짧으면 논의가 얕아진다"})
	if err != nil {
		t.Fatal(err)
	}
	return db, userID, spaceID, first, second
}

func TestEdgeReasonSurvivesTheRoundTripIntegration(t *testing.T) {
	db, userID, spaceID, first, second := edgeReasonSpace(t)
	ctx := context.Background()

	const why = "지난 두 분기 회고록이 둘 다 같은 안건을 다시 올렸다"
	created, err := db.CreateEdge(ctx, userID, Edge{
		SpaceID: spaceID, SourceID: second.ID, TargetID: first.ID,
		Relation: RelationContradicts, Reason: why,
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.Reason != why {
		t.Fatalf("the reason came back from the write as %q", created.Reason)
	}

	_, edges, err := db.ListNotes(ctx, userID, spaceID, "")
	if err != nil {
		t.Fatal(err)
	}
	var found *Edge
	for i := range edges {
		if edges[i].ID == created.ID {
			found = &edges[i]
		}
	}
	if found == nil {
		t.Fatal("the connection is not in the space")
	}
	if found.Reason != why {
		t.Fatalf("reading the space back gave reason %q, want %q", found.Reason, why)
	}
}

// A reason that can only be given at the moment the line is drawn is one that
// mostly does not get given.
func TestEdgeReasonCanBeAddedAndClearedLaterIntegration(t *testing.T) {
	db, userID, spaceID, first, second := edgeReasonSpace(t)
	ctx := context.Background()

	created, err := db.CreateEdge(ctx, userID, Edge{
		SpaceID: spaceID, SourceID: second.ID, TargetID: first.ID, Relation: RelationSupports,
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.Reason != "" {
		t.Fatalf("a connection nobody explained came back with %q", created.Reason)
	}

	const why = "측정치가 나중에 나와서 그때는 몰랐다"
	updated, err := db.SetEdgeReason(ctx, userID, created.ID, why)
	if err != nil {
		t.Fatalf("set: %v", err)
	}
	if updated.Reason != why {
		t.Fatalf("reason=%q", updated.Reason)
	}
	// Everything else about the line is what the line is, and must not move.
	if updated.SourceID != created.SourceID || updated.TargetID != created.TargetID ||
		updated.Relation != created.Relation || updated.Origin != created.Origin {
		t.Fatalf("setting a reason changed the connection: %+v -> %+v", created, updated)
	}

	cleared, err := db.SetEdgeReason(ctx, userID, created.ID, "   ")
	if err != nil {
		t.Fatalf("clear: %v", err)
	}
	if cleared.Reason != "" {
		t.Fatalf("a blank reason was stored as %q", cleared.Reason)
	}
}

// The SQL bound and the Go bound have to agree, or one of them is decoration.
// This goes through the store rather than ParseEdgeReason so that a constraint
// that disagreed with the parser would show up as a database error here.
func TestEdgeReasonBoundIsEnforcedByTheDatabaseIntegration(t *testing.T) {
	db, userID, spaceID, first, second := edgeReasonSpace(t)
	ctx := context.Background()

	created, err := db.CreateEdge(ctx, userID, Edge{
		SpaceID: spaceID, SourceID: second.ID, TargetID: first.ID, Relation: RelationSupports,
	})
	if err != nil {
		t.Fatal(err)
	}
	// Exactly at the limit, in Korean, so a byte-counting constraint would
	// refuse what the parser allows.
	full := strings.Repeat("가", MaxEdgeReason)
	if _, err := db.SetEdgeReason(ctx, userID, created.ID, full); err != nil {
		t.Fatalf("%d Korean characters were refused by the database: %v", MaxEdgeReason, err)
	}
	// And writing past it directly is refused by the constraint, not only by
	// the parser in front of it.
	_, err = db.Pool.Exec(ctx, `UPDATE note_edges SET reason=$2 WHERE id=$1`, created.ID, full+"가")
	if err == nil {
		t.Fatal("the database accepted a reason past the limit; the CHECK is not doing anything")
	}
}

// Annotating a space is changing it. Someone who may only look must not be
// able to write a sentence into somebody else's graph.
func TestEdgeReasonNeedsEditPermissionIntegration(t *testing.T) {
	db, userID, spaceID, first, second := edgeReasonSpace(t)
	ctx := context.Background()

	created, err := db.CreateEdge(ctx, userID, Edge{
		SpaceID: spaceID, SourceID: second.ID, TargetID: first.ID, Relation: RelationSupports,
	})
	if err != nil {
		t.Fatal(err)
	}
	stranger := uuid.New()
	if _, err := db.SetEdgeReason(ctx, stranger, created.ID, "내가 적었다"); err == nil {
		t.Fatal("someone with no access wrote a reason onto this connection")
	}

	// And the owner still can, so the refusal above is about permission rather
	// than about the call never working.
	if _, err := db.SetEdgeReason(ctx, userID, created.ID, "내가 적었다"); err != nil {
		t.Fatalf("the owner was refused too: %v", err)
	}
}

// Every path that hands an edge to a caller has to carry the reason.
//
// The column was added to the space read and the write, and the panel that
// shows connections reads through Backlinks — which still selected the old
// column list, so the sentence appeared when it was saved and vanished on the
// next load. Adding a field to one read path and not the others is the shape
// this keeps coming back in, so this walks the paths rather than one of them.
func TestEdgeReasonComesBackFromEveryReadIntegration(t *testing.T) {
	db, userID, spaceID, first, second := edgeReasonSpace(t)
	ctx := context.Background()

	const why = "같은 회고록을 두 번 읽고 이었다"
	drawn, err := db.CreateEdge(ctx, userID, Edge{
		SpaceID: spaceID, SourceID: second.ID, TargetID: first.ID,
		Relation: RelationSupports, Reason: why,
	})
	if err != nil {
		t.Fatal(err)
	}

	// The space, which is what the canvas and the deck compiler read.
	_, edges, err := db.ListNotes(ctx, userID, spaceID, "")
	if err != nil {
		t.Fatal(err)
	}
	seen := false
	for _, edge := range edges {
		if edge.ID == drawn.ID {
			seen = true
			if edge.Reason != why {
				t.Errorf("ListNotes gave reason %q", edge.Reason)
			}
		}
	}
	if !seen {
		t.Fatal("ListNotes did not return the connection at all")
	}

	// The connections panel, which is the only place the reason can be written
	// and therefore the one place it must be readable.
	backlinks, err := db.Backlinks(ctx, userID, first.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(backlinks) != 1 {
		t.Fatalf("%d backlinks", len(backlinks))
	}
	if backlinks[0].Edge.Reason != why {
		t.Errorf("Backlinks gave reason %q, want %q", backlinks[0].Edge.Reason, why)
	}

	// Accepting a suggestion returns the edge it just changed. Someone can
	// explain a proposed connection before standing behind it, and standing
	// behind it must not erase what they wrote.
	guessed, err := db.CreateNote(ctx, userID, Note{SpaceID: spaceID, AuthorID: userID, Content: "umm이 찾은 연결"})
	if err != nil {
		t.Fatal(err)
	}
	var suggestionID uuid.UUID
	if err := db.Pool.QueryRow(ctx, `
		INSERT INTO note_edges(space_id,source_note_id,target_note_id,relation,origin,confidence,created_by,reason)
		VALUES($1,$2,$3,'related','auto',0.5,$4,$5) RETURNING id`,
		spaceID, guessed.ID, first.ID, userID, why).Scan(&suggestionID); err != nil {
		t.Fatal(err)
	}
	accepted, err := db.AcceptSuggestion(ctx, userID, suggestionID)
	if err != nil {
		t.Fatal(err)
	}
	if accepted.Reason != why {
		t.Errorf("accepting a suggestion gave reason %q, want %q", accepted.Reason, why)
	}
}
