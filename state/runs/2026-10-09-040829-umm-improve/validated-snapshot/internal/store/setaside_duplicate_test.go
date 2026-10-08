package store

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

// Recording a decision only helps if it comes back at the moment the decision is
// being repeated. A near-duplicate of a thought in a line that was decided
// against is exactly that moment.
func TestDuplicateInASetAsideLineIsLabelledIntegration(t *testing.T) {
	db, userID, spaceID := retrievalSpace(t)
	ctx := context.Background()

	rejected, written, unrelated := uuid.New(), uuid.New(), uuid.New()
	if _, err := db.Pool.Exec(ctx, `
		INSERT INTO notes(id,space_id,author_id,content) VALUES
		($1,$4,$5,'배포를 젠킨스로 옮긴다'),
		($2,$4,$5,'배포를 젠킨스로 옮긴다'),
		($3,$4,$5,'전혀 관계없는 생각')`,
		rejected, written, unrelated, spaceID, userID); err != nil {
		t.Fatal(err)
	}
	branch, err := db.CreateBranch(ctx, userID, spaceID, "젠킨스로 이전", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err = db.SetNoteBranch(ctx, userID, rejected, &branch.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = db.ResolveBranch(ctx, userID, branch.ID, BranchAbandoned, "플러그인 호환 부담이 컸습니다"); err != nil {
		t.Fatal(err)
	}

	first := Note{ID: rejected, SpaceID: spaceID}
	second := Note{ID: written, SpaceID: spaceID}
	pairs := []DuplicatePair{{SpaceID: spaceID, First: first, Second: second, Score: 0.97}}
	if err = db.markSetAsideDuplicates(ctx, pairs); err != nil {
		t.Fatal(err)
	}
	if pairs[0].SetAside == nil {
		t.Fatal("a thought repeating a line that was decided against was not labelled")
	}
	if pairs[0].SetAside.Resolution != "플러그인 호환 부담이 컸습니다" {
		t.Errorf("the reason did not come back with it: %q", pairs[0].SetAside.Resolution)
	}
	if pairs[0].SetAsideNoteID == nil || *pairs[0].SetAsideNoteID != rejected {
		t.Errorf("the label points at the wrong side: %v", pairs[0].SetAsideNoteID)
	}

	// Both sides inside the same set-aside line is just two duplicates. Nothing
	// is being repeated against a decision, because the decision covers both.
	if err = db.SetNoteBranch(ctx, userID, written, &branch.ID); err != nil {
		t.Fatal(err)
	}
	bothIn := []DuplicatePair{{SpaceID: spaceID, First: first, Second: second, Score: 0.97}}
	if err = db.markSetAsideDuplicates(ctx, bothIn); err != nil {
		t.Fatal(err)
	}
	if bothIn[0].SetAside != nil {
		t.Error("two duplicates inside the same set-aside line were reported as repeating a decision")
	}

	// A line that is still open, or one that was taken, is the ordinary state of
	// a thought and says nothing about repeating a decision.
	if _, err = db.ReopenBranch(ctx, userID, branch.ID); err != nil {
		t.Fatal(err)
	}
	if err = db.SetNoteBranch(ctx, userID, written, nil); err != nil {
		t.Fatal(err)
	}
	stillOpen := []DuplicatePair{{SpaceID: spaceID, First: first, Second: second, Score: 0.97}}
	if err = db.markSetAsideDuplicates(ctx, stillOpen); err != nil {
		t.Fatal(err)
	}
	if stillOpen[0].SetAside != nil {
		t.Error("an open line was reported as a decision being repeated")
	}
}
