package store

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

// The record must contain only what a person marked. Anything inferred mixed in
// with it is worse than no record: when you need it you cannot tell which half
// is a decision and which half is a guess.
func TestTurningPointsAreOnlyWhatWasMarkedIntegration(t *testing.T) {
	db, userID, spaceID := retrievalSpace(t)
	ctx := context.Background()

	questionID, answerID, claimID, counterID, quietID := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	if _, err := db.Pool.Exec(ctx, `
		INSERT INTO notes(id,space_id,author_id,content,kind) VALUES
		($1,$6,$7,'회고 주기를 어떻게 할까','question'),
		($2,$6,$7,'격주로 두 번 해 보고 정한다','thought'),
		($3,$6,$7,'배포는 매일 해야 한다','thought'),
		($4,$6,$7,'매일 배포는 검토를 건너뛰게 만든다','thought'),
		($5,$6,$7,'아무도 표시하지 않은 조용한 생각','thought')`,
		questionID, answerID, claimID, counterID, quietID, spaceID, userID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Pool.Exec(ctx, `
		INSERT INTO note_edges(space_id,source_note_id,target_note_id,relation,origin,created_by) VALUES
		($1,$2,$3,'answers','manual',$6),
		($1,$4,$5,'contradicts','manual',$6)`,
		spaceID, answerID, questionID, claimID, counterID, userID); err != nil {
		t.Fatal(err)
	}
	branch, err := db.CreateBranch(ctx, userID, spaceID, "매일 배포로 전환", &claimID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.ResolveBranch(ctx, userID, branch.ID, BranchAbandoned,
		"검토를 건너뛰게 되는 비용이 더 컸습니다"); err != nil {
		t.Fatal(err)
	}
	// An open line is not a turning point. Nothing has been decided yet, and
	// listing it would turn a record of decisions into a list of work in progress.
	if _, err = db.CreateBranch(ctx, userID, spaceID, "아직 정하지 않은 갈래", nil); err != nil {
		t.Fatal(err)
	}

	points, _, err := db.TurningPoints(ctx, userID, &spaceID)
	if err != nil {
		t.Fatal(err)
	}

	seen := map[string]TurningPoint{}
	for _, point := range points {
		seen[point.Kind] = point
		if point.Space == "" || point.SpaceID == uuid.Nil {
			t.Errorf("%s point does not say where it happened: %+v", point.Kind, point)
		}
		if point.At.IsZero() {
			t.Errorf("%s point has no time, so it cannot be placed in order", point.Kind)
		}
		if point.Subject == "" {
			t.Errorf("%s point has no subject", point.Kind)
		}
	}
	if len(seen) != 3 {
		t.Fatalf("expected abandoned, answered and contradicted; got %d kinds from %d points", len(seen), len(points))
	}
	if _, ok := seen[TurningAdopted]; ok {
		t.Error("an adopted point appeared with nothing adopted")
	}

	// The reason is the part that goes missing first, so it travels with the
	// decision rather than being looked up separately.
	if got := seen[TurningAbandoned].Detail; got != "검토를 건너뛰게 되는 비용이 더 컸습니다" {
		t.Errorf("the reason did not travel with the decision: %q", got)
	}
	if got := seen[TurningAnswered].Subject; got != "회고 주기를 어떻게 할까" {
		t.Errorf("answered point subject = %q, want the question", got)
	}
	if got := seen[TurningAnswered].Detail; got != "격주로 두 번 해 보고 정한다" {
		t.Errorf("answered point detail = %q, want the answering thought", got)
	}
	for _, point := range points {
		if point.Subject == "아무도 표시하지 않은 조용한 생각" {
			t.Error("a thought nobody marked was reported as a turning point")
		}
		if point.Subject == "아직 정하지 않은 갈래" {
			t.Error("an unresolved line was reported as a decision")
		}
	}
}

// Ordering is the whole point of a record. Merging in SQL rather than taking the
// newest N of each kind is what keeps a rare kind from vanishing behind a common
// one — which would read as "nothing was decided that year".
func TestTurningPointsAreOrderedNewestFirstIntegration(t *testing.T) {
	db, userID, spaceID := retrievalSpace(t)
	ctx := context.Background()

	for index := 0; index < 3; index++ {
		branch, err := db.CreateBranch(ctx, userID, spaceID, "갈래 "+string(rune('가'+index)), nil)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = db.ResolveBranch(ctx, userID, branch.ID, BranchAdopted, "이유 "+string(rune('가'+index))); err != nil {
			t.Fatal(err)
		}
		// Resolved times come from the database clock, so they need separating.
		if _, err = db.Pool.Exec(ctx, `UPDATE branches SET resolved_at=now() - make_interval(days => $2) WHERE id=$1`,
			branch.ID, index); err != nil {
			t.Fatal(err)
		}
	}

	points, _, err := db.TurningPoints(ctx, userID, &spaceID)
	if err != nil {
		t.Fatal(err)
	}
	if len(points) != 3 {
		t.Fatalf("got %d points, want 3", len(points))
	}
	for index := 1; index < len(points); index++ {
		if points[index].At.After(points[index-1].At) {
			t.Fatalf("point %d is newer than the one before it: %v then %v",
				index, points[index-1].At, points[index].At)
		}
	}
	if time.Since(points[0].At) > 24*time.Hour {
		t.Errorf("the newest point is %v old; ordering is reversed", time.Since(points[0].At))
	}
}

// Someone must not learn what another person decided in a space they cannot see.
func TestTurningPointsStayInsideVisibleSpacesIntegration(t *testing.T) {
	db, userID, spaceID := retrievalSpace(t)
	ctx := context.Background()

	otherUser, otherSpace := uuid.New(), uuid.New()
	name := "stranger_" + otherUser.String()
	if _, err := db.Pool.Exec(ctx, `INSERT INTO users(id,username,display_name) VALUES($1,$2::citext,$2::text)`, otherUser, name); err != nil {
		t.Fatal(err)
	}
	defer db.Pool.Exec(context.Background(), `DELETE FROM users WHERE id=$1`, otherUser)
	if _, err := db.Pool.Exec(ctx, `INSERT INTO spaces(id,owner_id,name) VALUES($1,$2,'남의 공간')`, otherSpace, otherUser); err != nil {
		t.Fatal(err)
	}
	branch, err := db.CreateBranch(ctx, otherUser, otherSpace, "남의 결정", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.ResolveBranch(ctx, otherUser, branch.ID, BranchAdopted, "남의 이유"); err != nil {
		t.Fatal(err)
	}

	points, _, err := db.TurningPoints(ctx, userID, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, point := range points {
		if point.SpaceID == otherSpace {
			t.Fatalf("a decision from someone else's space leaked: %+v", point)
		}
	}
	_ = spaceID
}

// A record that stops at the newest hundred and twenty without saying so reads
// as the whole story. Everywhere else umm names what it left out.
func TestTurningPointsSayWhenThereAreMoreIntegration(t *testing.T) {
	db, userID, spaceID := retrievalSpace(t)
	ctx := context.Background()

	for index := 0; index < maxTurningPoints+5; index++ {
		branch, err := db.CreateBranch(ctx, userID, spaceID, "갈래 "+uuid.New().String()[:8], nil)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = db.ResolveBranch(ctx, userID, branch.ID, BranchAdopted, "이유"); err != nil {
			t.Fatal(err)
		}
	}

	points, more, err := db.TurningPoints(ctx, userID, &spaceID)
	if err != nil {
		t.Fatal(err)
	}
	if len(points) != maxTurningPoints {
		t.Fatalf("returned %d points, want the cap of %d", len(points), maxTurningPoints)
	}
	if !more {
		t.Error("the record was cut off and did not say so")
	}

	fewer, alsoMore, err := db.TurningPoints(ctx, userID, nil)
	if err != nil {
		t.Fatal(err)
	}
	_ = fewer
	if !alsoMore {
		t.Error("the cross-space read did not report the cut either")
	}
}
