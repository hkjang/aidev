package store

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func retrievalSpace(t *testing.T) (*Store, uuid.UUID, uuid.UUID) {
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
	name := "retrieve_" + strings.ReplaceAll(userID.String(), "-", "")
	if _, err = db.Pool.Exec(ctx, `INSERT INTO users(id,username,display_name) VALUES($1,$2::citext,$2::text)`, userID, name); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Pool.Exec(context.Background(), `DELETE FROM users WHERE id=$1`, userID) })
	if _, err = db.Pool.Exec(ctx, `INSERT INTO user_preferences(user_id) VALUES($1)`, userID); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Pool.Exec(ctx, `INSERT INTO spaces(id,owner_id,name) VALUES($1,$2,'검색')`, spaceID, userID); err != nil {
		t.Fatal(err)
	}
	return db, userID, spaceID
}

// The rule that matters most. A note marked as excluded from AI must never be
// returned for a question, because everything downstream sends what it gets to a
// gateway — umm shipped exactly that defect once, in a path that captured content
// before checking the flag.
func TestRetrievalNeverReturnsExcludedThoughtsIntegration(t *testing.T) {
	db, userID, spaceID := retrievalSpace(t)
	ctx := context.Background()

	open, err := db.CreateNote(ctx, userID, Note{SpaceID: spaceID, AuthorID: userID, Content: "배포 파이프라인은 이미지를 만든다"})
	if err != nil {
		t.Fatal(err)
	}
	secret, err := db.CreateNote(ctx, userID, Note{SpaceID: spaceID, AuthorID: userID, Content: "배포 파이프라인 비밀 키 회전 절차"})
	if err != nil {
		t.Fatal(err)
	}
	excluded := true
	secret.AIExcluded = true
	if _, err = db.UpdateNote(ctx, userID, secret, &excluded); err != nil {
		t.Fatal(err)
	}

	result, err := db.RetrieveForQuestion(ctx, userID, "배포 파이프라인", 10)
	if err != nil {
		t.Fatalf("retrieve: %v", err)
	}
	for _, item := range result.Thoughts {
		if item.Note.ID == secret.ID {
			t.Fatal("an AI-excluded thought was retrieved and would have been sent to a gateway")
		}
	}
	if result.Excluded < 1 {
		t.Error("the exclusion was silent; an answer built from less than expected must say so")
	}
	found := false
	for _, item := range result.Thoughts {
		if item.Note.ID == open.ID {
			found = true
		}
	}
	if !found {
		t.Error("the ordinary thought was dropped along with the excluded one")
	}
}

// A space marked as excluded covers everything in it, and checking only the note
// would leak every thought in that space.
func TestRetrievalHonoursSpaceLevelExclusionIntegration(t *testing.T) {
	db, userID, spaceID := retrievalSpace(t)
	ctx := context.Background()
	if _, err := db.CreateNote(ctx, userID, Note{SpaceID: spaceID, AuthorID: userID, Content: "임베딩 게이트웨이 설정"}); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Pool.Exec(ctx, `UPDATE spaces SET ai_excluded=true WHERE id=$1`, spaceID); err != nil {
		t.Fatal(err)
	}
	result, err := db.RetrieveForQuestion(ctx, userID, "임베딩 게이트웨이", 10)
	if err != nil {
		t.Fatalf("retrieve: %v", err)
	}
	if len(result.Thoughts) != 0 {
		t.Fatalf("%d thoughts came back from a space excluded from AI", len(result.Thoughts))
	}
	if result.Excluded < 1 {
		t.Error("the whole space was dropped without saying so")
	}
}

// Following the graph one step is what makes this different from search: the
// answer to a question is often recorded next to the thought that raises it.
func TestRetrievalFollowsConnectionsOneStepIntegration(t *testing.T) {
	db, userID, spaceID := retrievalSpace(t)
	ctx := context.Background()

	question, err := db.CreateNote(ctx, userID, Note{SpaceID: spaceID, AuthorID: userID, Content: "토큰 만료를 몇 시간으로 둘까"})
	if err != nil {
		t.Fatal(err)
	}
	// Deliberately shares no words with the question, so only the connection can
	// reach it.
	answer, err := db.CreateNote(ctx, userID, Note{SpaceID: spaceID, AuthorID: userID, Content: "하루면 충분하다고 결론"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.CreateEdge(ctx, userID, Edge{
		SpaceID: spaceID, SourceID: answer.ID, TargetID: question.ID, Relation: RelationAnswers,
	}); err != nil {
		t.Fatal(err)
	}

	result, err := db.RetrieveForQuestion(ctx, userID, "토큰 만료", 10)
	if err != nil {
		t.Fatalf("retrieve: %v", err)
	}
	var reached *Retrieved
	for i := range result.Thoughts {
		if result.Thoughts[i].Note.ID == answer.ID {
			reached = &result.Thoughts[i]
		}
	}
	if reached == nil {
		t.Fatal("the connected answer was not reached; retrieval is only doing search")
	}
	if reached.Via != ViaConnection {
		t.Errorf("via=%q, want %q", reached.Via, ViaConnection)
	}
	if reached.Through == nil || *reached.Through != question.ID {
		t.Error("the reached thought does not say what it was reached through")
	}
}

// A thought excluded from AI must not be reachable through the graph either.
// Blocking it in search and letting the neighbour walk pull it in would be the
// same leak by a longer route.
func TestRetrievalDoesNotReachExcludedThoughtsThroughConnectionsIntegration(t *testing.T) {
	db, userID, spaceID := retrievalSpace(t)
	ctx := context.Background()

	anchor, err := db.CreateNote(ctx, userID, Note{SpaceID: spaceID, AuthorID: userID, Content: "인증 토큰 정책"})
	if err != nil {
		t.Fatal(err)
	}
	secret, err := db.CreateNote(ctx, userID, Note{SpaceID: spaceID, AuthorID: userID, Content: "실제 서명 키 값과 회전 절차"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.CreateEdge(ctx, userID, Edge{
		SpaceID: spaceID, SourceID: secret.ID, TargetID: anchor.ID, Relation: RelationRefines,
	}); err != nil {
		t.Fatal(err)
	}
	excluded := true
	secret.AIExcluded = true
	if _, err = db.UpdateNote(ctx, userID, secret, &excluded); err != nil {
		t.Fatal(err)
	}

	result, err := db.RetrieveForQuestion(ctx, userID, "인증 토큰 정책", 10)
	if err != nil {
		t.Fatalf("retrieve: %v", err)
	}
	for _, item := range result.Thoughts {
		if item.Note.ID == secret.ID {
			t.Fatal("an excluded thought was reached through a connection")
		}
	}
}

// Retrieval must not become a way to read another person's memory.
func TestRetrievalStaysWithinReachableSpacesIntegration(t *testing.T) {
	db, userID, _ := retrievalSpace(t)
	otherDB, strangerID, strangerSpace := retrievalSpace(t)
	ctx := context.Background()

	if _, err := otherDB.CreateNote(ctx, strangerID, Note{
		SpaceID: strangerSpace, AuthorID: strangerID, Content: "남의 배포 파이프라인 메모",
	}); err != nil {
		t.Fatal(err)
	}
	result, err := db.RetrieveForQuestion(ctx, userID, "배포 파이프라인", 10)
	if err != nil {
		t.Fatalf("retrieve: %v", err)
	}
	for _, item := range result.Thoughts {
		if item.Note.SpaceID == strangerSpace {
			t.Fatal("another person's thought was retrieved")
		}
	}
}
