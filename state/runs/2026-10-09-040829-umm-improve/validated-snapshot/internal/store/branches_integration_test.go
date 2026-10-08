package store

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
)

// The guarantee the whole feature rests on: a thought in an abandoned line is
// never handed back unlabelled. Both ways in are checked — the thought that
// matched the question, and the one reached through a connection. The second
// matters more, because nobody chose to look at it.
func TestAbandonedThoughtsAreLabelledOnEveryRetrievalPathIntegration(t *testing.T) {
	db, userID, spaceID := retrievalSpace(t)
	ctx := context.Background()

	matchID, neighbourID := uuid.New(), uuid.New()
	if _, err := db.Pool.Exec(ctx, `
		INSERT INTO notes(id,space_id,author_id,content) VALUES
		($1,$3,$4,'배포 파이프라인을 젠킨스로 옮기는 실험'),
		($2,$3,$4,'상시 대기 서버를 돌보는 손이 모자랍니다')`,
		matchID, neighbourID, spaceID, userID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Pool.Exec(ctx, `
		INSERT INTO note_edges(space_id,source_note_id,target_note_id,relation,origin,created_by)
		VALUES($1,$2,$3,'related','manual',$4)`, spaceID, matchID, neighbourID, userID); err != nil {
		t.Fatal(err)
	}

	branch, err := db.CreateBranch(ctx, userID, spaceID, "젠킨스로 이전", &matchID)
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []uuid.UUID{matchID, neighbourID} {
		if err = db.SetNoteBranch(ctx, userID, id, &branch.ID); err != nil {
			t.Fatal(err)
		}
	}

	// While the line is open, nothing is marked: an open line is the ordinary
	// state of a thought and a caveat on it would be noise.
	open, err := db.RetrieveForQuestion(ctx, userID, "배포 파이프라인 젠킨스", 12)
	if err != nil {
		t.Fatal(err)
	}
	for _, thought := range open.Thoughts {
		if thought.Branch == nil {
			t.Fatalf("thought %s lost its branch label", thought.Note.ID)
		}
		if thought.Branch.Status != BranchOpen {
			t.Errorf("open branch reported as %q", thought.Branch.Status)
		}
	}

	if _, err = db.ResolveBranch(ctx, userID, branch.ID, BranchAbandoned,
		"에이전트 관리 비용이 옮겨서 얻는 것보다 컸습니다"); err != nil {
		t.Fatal(err)
	}

	result, err := db.RetrieveForQuestion(ctx, userID, "배포 파이프라인 젠킨스", 12)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Thoughts) < 2 {
		t.Fatalf("expected both thoughts back, got %d", len(result.Thoughts))
	}
	viaSeen := map[string]bool{}
	for _, thought := range result.Thoughts {
		viaSeen[thought.Via] = true
		if thought.Branch == nil {
			t.Fatalf("thought reached via %s came back with no branch; an abandoned line reads as current",
				thought.Via)
		}
		if thought.Branch.Status != BranchAbandoned {
			t.Errorf("branch status = %q, want abandoned", thought.Branch.Status)
		}
		if thought.Branch.Name != "젠킨스로 이전" {
			t.Errorf("branch name = %q", thought.Branch.Name)
		}
	}
	if !viaSeen[ViaMatch] || !viaSeen[ViaConnection] {
		t.Errorf("both retrieval paths were not exercised: %v", viaSeen)
	}
}

// Resolving without a reason is the same forgetting this feature exists to
// prevent, one step later: you know you rejected it and no longer know why.
func TestResolvingABranchRequiresAReasonIntegration(t *testing.T) {
	db, userID, spaceID := retrievalSpace(t)
	ctx := context.Background()

	branch, err := db.CreateBranch(ctx, userID, spaceID, "격주 회고", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.ResolveBranch(ctx, userID, branch.ID, BranchAbandoned, "   "); !errors.Is(err, ErrResolutionRequired) {
		t.Fatalf("a branch was abandoned with no reason: %v", err)
	}
	if _, err = db.ResolveBranch(ctx, userID, branch.ID, "dropped", "이유"); !errors.Is(err, ErrUnknownBranchStatus) {
		t.Fatalf("an unknown status was accepted: %v", err)
	}

	resolved, err := db.ResolveBranch(ctx, userID, branch.ID, BranchAdopted, "두 번 해 보니 논의가 얕아지지 않았습니다")
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Status != BranchAdopted || resolved.ResolvedAt == nil {
		t.Fatalf("resolve did not record the outcome: %+v", resolved)
	}

	// Reopening clears the reason, because it described a decision that no longer
	// stands and would read as current if it stayed.
	reopened, err := db.ReopenBranch(ctx, userID, branch.ID)
	if err != nil {
		t.Fatal(err)
	}
	if reopened.Status != BranchOpen || reopened.Resolution != "" || reopened.ResolvedAt != nil {
		t.Fatalf("reopen left stale resolution behind: %+v", reopened)
	}
}

// Deleting a line must not delete the thinking that happened in it.
func TestDeletingABranchKeepsItsThoughtsIntegration(t *testing.T) {
	db, userID, spaceID := retrievalSpace(t)
	ctx := context.Background()

	noteID := uuid.New()
	if _, err := db.Pool.Exec(ctx, `INSERT INTO notes(id,space_id,author_id,content) VALUES($1,$2,$3,'갈래 안에서 적은 생각')`,
		noteID, spaceID, userID); err != nil {
		t.Fatal(err)
	}
	branch, err := db.CreateBranch(ctx, userID, spaceID, "지울 갈래", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err = db.SetNoteBranch(ctx, userID, noteID, &branch.ID); err != nil {
		t.Fatal(err)
	}
	if err = db.DeleteBranch(ctx, userID, branch.ID); err != nil {
		t.Fatal(err)
	}

	var content string
	var stillFiled *uuid.UUID
	if err = db.Pool.QueryRow(ctx, `SELECT content,branch_id FROM notes WHERE id=$1 AND deleted_at IS NULL`, noteID).
		Scan(&content, &stillFiled); err != nil {
		t.Fatalf("deleting a branch took its thoughts with it: %v", err)
	}
	if stillFiled != nil {
		t.Errorf("note still points at a deleted branch: %v", stillFiled)
	}
}

// A branch belongs to one space, and so does the thought filed under it.
// Otherwise a thought carries a label its readers cannot see the meaning of.
func TestBranchesDoNotCrossSpacesIntegration(t *testing.T) {
	db, userID, spaceID := retrievalSpace(t)
	ctx := context.Background()

	otherSpace, noteID := uuid.New(), uuid.New()
	if _, err := db.Pool.Exec(ctx, `INSERT INTO spaces(id,owner_id,name) VALUES($1,$2,'다른 공간')`, otherSpace, userID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Pool.Exec(ctx, `INSERT INTO notes(id,space_id,author_id,content) VALUES($1,$2,$3,'다른 공간의 생각')`,
		noteID, otherSpace, userID); err != nil {
		t.Fatal(err)
	}
	branch, err := db.CreateBranch(ctx, userID, spaceID, "이 공간의 갈래", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err = db.SetNoteBranch(ctx, userID, noteID, &branch.ID); err == nil {
		t.Fatal("a thought was filed under a branch from another space")
	}

	// And a root note from another space cannot anchor the line either.
	if _, err = db.CreateBranch(ctx, userID, spaceID, "남의 뿌리", &noteID); err == nil {
		t.Fatal("a branch was rooted at a note in another space")
	}
}

// Someone who can only read a space must not be able to reshape how its
// thoughts are labelled.
func TestBranchesRequireEditPermissionIntegration(t *testing.T) {
	db, userID, spaceID := retrievalSpace(t)
	ctx := context.Background()

	viewerID := uuid.New()
	name := "viewer_" + strings.ReplaceAll(viewerID.String(), "-", "")
	if _, err := db.Pool.Exec(ctx, `INSERT INTO users(id,username,display_name) VALUES($1,$2::citext,$2::text)`, viewerID, name); err != nil {
		t.Fatal(err)
	}
	defer db.Pool.Exec(context.Background(), `DELETE FROM users WHERE id=$1`, viewerID)
	if _, err := db.Pool.Exec(ctx, `INSERT INTO space_members(space_id,user_id,permission) VALUES($1,$2,'view')`, spaceID, viewerID); err != nil {
		t.Fatal(err)
	}

	if _, err := db.CreateBranch(ctx, viewerID, spaceID, "읽기 전용 사용자의 갈래", nil); err == nil {
		t.Error("a view-only member created a branch")
	}
	branch, err := db.CreateBranch(ctx, userID, spaceID, "주인의 갈래", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.ResolveBranch(ctx, viewerID, branch.ID, BranchAbandoned, "마음에 들지 않습니다"); err == nil {
		t.Error("a view-only member resolved someone else's branch")
	}
	// Reading it is fine — that is what view means.
	list, err := db.ListBranches(ctx, viewerID, spaceID)
	if err != nil || len(list) != 1 {
		t.Errorf("a view-only member could not list branches: %v %d", err, len(list))
	}
}
