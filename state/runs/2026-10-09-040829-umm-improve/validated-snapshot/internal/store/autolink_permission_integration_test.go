package store

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
)

// Suggesting connections writes them. Every edge it keeps is inserted into the
// space, so it is a change to someone's workspace and not a read.
//
// Editing a note checks that the person may write to the space. This path did
// not: it read the notes, which a view-only member is allowed to do, and then
// inserted edges without asking whether they could. So a member shared in to
// read could put connections into a space that is not theirs to change.
func TestSuggestLinksRefusesAMemberWhoMayOnlyReadIntegration(t *testing.T) {
	db, ownerID, spaceID := autoLinkSpace(t)
	ctx := context.Background()
	stopGateway := useSemanticStub(t, db, ownerID)
	defer stopGateway()
	assertNotesEmbeddedByTheStub(t, db, ownerID, spaceID)

	readerID := uuid.New()
	reader := "autolink_reader_" + strings.ReplaceAll(readerID.String(), "-", "")
	if _, err := db.Pool.Exec(ctx, `INSERT INTO users(id,username,display_name) VALUES($1,$2::citext,$2::text)`, readerID, reader); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Pool.Exec(context.Background(), `DELETE FROM users WHERE id=$1`, readerID) })
	if _, err := db.Pool.Exec(ctx, `INSERT INTO space_members(space_id,user_id,permission) VALUES($1,$2,'view')`, spaceID, readerID); err != nil {
		t.Fatal(err)
	}

	before := edgeCount(t, db, spaceID)

	result, err := db.SuggestLinks(ctx, readerID, spaceID)
	if err != nil {
		t.Fatalf("suggest links as a read-only member: %v", err)
	}
	if len(result.Edges) != 0 {
		t.Errorf("a read-only member was handed %d new edges", len(result.Edges))
	}
	if after := edgeCount(t, db, spaceID); after != before {
		t.Errorf("a read-only member added %d edges to a space they may only read", after-before)
	}

	// The owner is unaffected: refusing the reader must not disable the feature.
	ownerResult, err := db.SuggestLinks(ctx, ownerID, spaceID)
	if err != nil {
		t.Fatalf("suggest links as the owner: %v", err)
	}
	if ownerResult.Outcome != OutcomeSuggested || len(ownerResult.Edges) == 0 {
		t.Fatalf("the owner got outcome=%q with %d edges", ownerResult.Outcome, len(ownerResult.Edges))
	}
}

// A member who may write is still allowed to.
func TestSuggestLinksAllowsAMemberWhoMayWriteIntegration(t *testing.T) {
	db, ownerID, spaceID := autoLinkSpace(t)
	ctx := context.Background()
	stopGateway := useSemanticStub(t, db, ownerID)
	defer stopGateway()
	assertNotesEmbeddedByTheStub(t, db, ownerID, spaceID)

	editorID := uuid.New()
	editor := "autolink_editor_" + strings.ReplaceAll(editorID.String(), "-", "")
	if _, err := db.Pool.Exec(ctx, `INSERT INTO users(id,username,display_name) VALUES($1,$2::citext,$2::text)`, editorID, editor); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Pool.Exec(context.Background(), `DELETE FROM users WHERE id=$1`, editorID) })
	if _, err := db.Pool.Exec(ctx, `INSERT INTO space_members(space_id,user_id,permission) VALUES($1,$2,'edit')`, spaceID, editorID); err != nil {
		t.Fatal(err)
	}

	result, err := db.SuggestLinks(ctx, editorID, spaceID)
	if err != nil {
		t.Fatalf("suggest links as an editing member: %v", err)
	}
	if result.Outcome != OutcomeSuggested || len(result.Edges) == 0 {
		t.Fatalf("a member who may write got outcome=%q with %d edges", result.Outcome, len(result.Edges))
	}
}

func edgeCount(t *testing.T, db *Store, spaceID uuid.UUID) int {
	t.Helper()
	var n int
	if err := db.Pool.QueryRow(context.Background(), `SELECT count(*) FROM note_edges WHERE space_id=$1`, spaceID).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}
