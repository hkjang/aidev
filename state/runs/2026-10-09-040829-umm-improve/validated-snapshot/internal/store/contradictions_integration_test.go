package store

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func contradictionSpace(t *testing.T) (*Store, uuid.UUID, uuid.UUID) {
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
	name := "contra_" + strings.ReplaceAll(userID.String(), "-", "")
	if _, err = db.Pool.Exec(ctx, `INSERT INTO users(id,username,display_name) VALUES($1,$2::citext,$2::text)`, userID, name); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Pool.Exec(context.Background(), `DELETE FROM users WHERE id=$1`, userID) })
	if _, err = db.Pool.Exec(ctx, `INSERT INTO user_preferences(user_id) VALUES($1)`, userID); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Pool.Exec(ctx, `INSERT INTO spaces(id,owner_id,name) VALUES($1,$2,'상충')`, spaceID, userID); err != nil {
		t.Fatal(err)
	}
	return db, userID, spaceID
}

// The disagreement has a direction — one thought contradicts another — and the
// two sides must not be swapped on the way out.
func TestContradictionsReportBothSidesInOrderIntegration(t *testing.T) {
	db, userID, spaceID := contradictionSpace(t)
	ctx := context.Background()

	claim, err := db.CreateNote(ctx, userID, Note{SpaceID: spaceID, AuthorID: userID, Content: "캐시를 늘리면 지연이 준다"})
	if err != nil {
		t.Fatal(err)
	}
	counter, err := db.CreateNote(ctx, userID, Note{SpaceID: spaceID, AuthorID: userID, Content: "캐시를 늘려도 지연은 그대로였다"})
	if err != nil {
		t.Fatal(err)
	}
	// A connection that is not a disagreement, to prove the query is selective.
	unrelated, err := db.CreateNote(ctx, userID, Note{SpaceID: spaceID, AuthorID: userID, Content: "측정 방법을 적어 두자"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.CreateEdge(ctx, userID, Edge{
		SpaceID: spaceID, SourceID: counter.ID, TargetID: claim.ID, Relation: RelationContradicts,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err = db.CreateEdge(ctx, userID, Edge{
		SpaceID: spaceID, SourceID: unrelated.ID, TargetID: claim.ID, Relation: RelationSupports,
	}); err != nil {
		t.Fatal(err)
	}

	found, err := db.Contradictions(ctx, userID, &spaceID)
	if err != nil {
		t.Fatalf("contradictions: %v", err)
	}
	if len(found) != 1 {
		t.Fatalf("expected only the disagreement, got %d", len(found))
	}
	if found[0].Claim.ID != counter.ID || found[0].Counter.ID != claim.ID {
		t.Error("the two sides were swapped; the connection reads source contradicts target")
	}
	if found[0].Origin != OriginManual {
		t.Errorf("origin=%q; a reader cannot tell who recorded the disagreement", found[0].Origin)
	}
	if found[0].Space != "상충" {
		t.Errorf("space=%q", found[0].Space)
	}
}

// A workspace where nobody has marked anything must come back empty rather than
// look examined. umm does not detect contradictions, and the caller has to be
// able to tell "none recorded" from "none exist".
func TestContradictionsAreEmptyWhenNobodyRecordedAnyIntegration(t *testing.T) {
	db, userID, spaceID := contradictionSpace(t)
	ctx := context.Background()
	for _, content := range []string{"생각 하나", "정반대 생각"} {
		if _, err := db.CreateNote(ctx, userID, Note{SpaceID: spaceID, AuthorID: userID, Content: content}); err != nil {
			t.Fatal(err)
		}
	}
	found, err := db.Contradictions(ctx, userID, &spaceID)
	if err != nil {
		t.Fatalf("contradictions: %v", err)
	}
	if len(found) != 0 {
		t.Fatalf("umm reported %d disagreements it was never told about", len(found))
	}
}

// A deleted thought has no disagreement left to report.
func TestContradictionsIgnoreDeletedThoughtsIntegration(t *testing.T) {
	db, userID, spaceID := contradictionSpace(t)
	ctx := context.Background()
	first, err := db.CreateNote(ctx, userID, Note{SpaceID: spaceID, AuthorID: userID, Content: "주장"})
	if err != nil {
		t.Fatal(err)
	}
	second, err := db.CreateNote(ctx, userID, Note{SpaceID: spaceID, AuthorID: userID, Content: "반론"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.CreateEdge(ctx, userID, Edge{
		SpaceID: spaceID, SourceID: second.ID, TargetID: first.ID, Relation: RelationContradicts,
	}); err != nil {
		t.Fatal(err)
	}
	if err = db.DeleteNote(ctx, userID, second.ID); err != nil {
		t.Fatal(err)
	}
	found, err := db.Contradictions(ctx, userID, &spaceID)
	if err != nil {
		t.Fatalf("contradictions: %v", err)
	}
	if len(found) != 0 {
		t.Errorf("a disagreement with a deleted side was still reported")
	}
}

// Another person's disagreements are not this person's to read.
func TestContradictionsStayWithinReachableSpacesIntegration(t *testing.T) {
	db, userID, _ := contradictionSpace(t)
	otherDB, strangerID, strangerSpace := contradictionSpace(t)
	ctx := context.Background()

	a, err := otherDB.CreateNote(ctx, strangerID, Note{SpaceID: strangerSpace, AuthorID: strangerID, Content: "남의 주장"})
	if err != nil {
		t.Fatal(err)
	}
	b, err := otherDB.CreateNote(ctx, strangerID, Note{SpaceID: strangerSpace, AuthorID: strangerID, Content: "남의 반론"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = otherDB.CreateEdge(ctx, strangerID, Edge{
		SpaceID: strangerSpace, SourceID: b.ID, TargetID: a.ID, Relation: RelationContradicts,
	}); err != nil {
		t.Fatal(err)
	}

	found, err := db.Contradictions(ctx, userID, nil)
	if err != nil {
		t.Fatalf("contradictions: %v", err)
	}
	for _, item := range found {
		if item.SpaceID == strangerSpace {
			t.Fatal("another person's disagreement was returned")
		}
	}
}
