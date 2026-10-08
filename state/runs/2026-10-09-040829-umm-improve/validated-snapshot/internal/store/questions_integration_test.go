package store

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func questionSpace(t *testing.T) (*Store, uuid.UUID, uuid.UUID) {
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
	name := "question_" + strings.ReplaceAll(userID.String(), "-", "")
	if _, err = db.Pool.Exec(ctx, `INSERT INTO users(id,username,display_name) VALUES($1,$2::citext,$2::text)`, userID, name); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Pool.Exec(context.Background(), `DELETE FROM users WHERE id=$1`, userID) })
	if _, err = db.Pool.Exec(ctx, `INSERT INTO user_preferences(user_id) VALUES($1)`, userID); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Pool.Exec(ctx, `INSERT INTO spaces(id,owner_id,name) VALUES($1,$2,'질문')`, spaceID, userID); err != nil {
		t.Fatal(err)
	}
	return db, userID, spaceID
}

func ask(t *testing.T, db *Store, userID, spaceID uuid.UUID, content string) Note {
	t.Helper()
	n, err := db.CreateNote(context.Background(), userID,
		Note{SpaceID: spaceID, AuthorID: userID, Content: content, Kind: string(KindQuestion)})
	if err != nil {
		t.Fatal(err)
	}
	return n
}

// The whole point: a question with an answer recorded against it is closed, and
// one without is still open.
func TestAnsweringClosesAQuestionIntegration(t *testing.T) {
	db, userID, spaceID := questionSpace(t)
	ctx := context.Background()

	open := ask(t, db, userID, spaceID, "재시도 상한을 몇으로 둘까")
	settled := ask(t, db, userID, spaceID, "토큰 만료를 몇 시간으로 둘까")
	answer, err := db.CreateNote(ctx, userID, Note{SpaceID: spaceID, AuthorID: userID, Content: "24시간으로 정했다"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.CreateEdge(ctx, userID, Edge{
		SpaceID: spaceID, SourceID: answer.ID, TargetID: settled.ID, Relation: RelationAnswers,
	}); err != nil {
		t.Fatal(err)
	}

	questions, err := db.OpenQuestions(ctx, userID, &spaceID)
	if err != nil {
		t.Fatalf("open questions: %v", err)
	}
	if len(questions) != 1 {
		t.Fatalf("expected only the unanswered question, got %d", len(questions))
	}
	if questions[0].Note.ID != open.ID {
		t.Errorf("the answered question was reported as open")
	}
	if questions[0].Space != "질문" {
		t.Errorf("space=%q", questions[0].Space)
	}
}

// A thought that argues with a question does not settle it. If supports or
// refines counted as answers, a question would look resolved by a note that only
// disagreed with it.
func TestNearbyRelationsDoNotCloseAQuestionIntegration(t *testing.T) {
	db, userID, spaceID := questionSpace(t)
	ctx := context.Background()

	question := ask(t, db, userID, spaceID, "캐시를 늘려야 할까")
	for _, relation := range []Relation{RelationSupports, RelationRefines, RelationContradicts, RelationRelated} {
		other, err := db.CreateNote(ctx, userID, Note{SpaceID: spaceID, AuthorID: userID, Content: "의견 " + string(relation)})
		if err != nil {
			t.Fatal(err)
		}
		if _, err = db.CreateEdge(ctx, userID, Edge{
			SpaceID: spaceID, SourceID: other.ID, TargetID: question.ID, Relation: relation,
		}); err != nil {
			t.Fatal(err)
		}
	}

	questions, err := db.OpenQuestions(ctx, userID, &spaceID)
	if err != nil {
		t.Fatalf("open questions: %v", err)
	}
	if len(questions) != 1 {
		t.Fatalf("a question circled by four opinions is still open; got %d", len(questions))
	}
	// The count is what separates an untouched question from an argued-over one.
	if questions[0].Attempts != 4 {
		t.Errorf("attempts=%d, want 4", questions[0].Attempts)
	}
}

// Ordinary thoughts are not questions, however they are worded.
func TestOnlyMarkedQuestionsAreReportedIntegration(t *testing.T) {
	db, userID, spaceID := questionSpace(t)
	ctx := context.Background()
	// Written as a question but never marked as one. umm does not read notes and
	// decide they are asking something.
	if _, err := db.CreateNote(ctx, userID, Note{SpaceID: spaceID, AuthorID: userID, Content: "이거 정말 맞나?"}); err != nil {
		t.Fatal(err)
	}
	questions, err := db.OpenQuestions(ctx, userID, &spaceID)
	if err != nil {
		t.Fatalf("open questions: %v", err)
	}
	if len(questions) != 0 {
		t.Fatalf("umm inferred a question nobody marked: %d", len(questions))
	}
}

// An answer that was deleted no longer settles anything, so the question opens
// back up rather than staying closed by a note that is gone.
func TestDeletingTheAnswerReopensTheQuestionIntegration(t *testing.T) {
	db, userID, spaceID := questionSpace(t)
	ctx := context.Background()

	question := ask(t, db, userID, spaceID, "배포 주기를 어떻게 할까")
	answer, err := db.CreateNote(ctx, userID, Note{SpaceID: spaceID, AuthorID: userID, Content: "주 1회로 한다"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.CreateEdge(ctx, userID, Edge{
		SpaceID: spaceID, SourceID: answer.ID, TargetID: question.ID, Relation: RelationAnswers,
	}); err != nil {
		t.Fatal(err)
	}
	if questions, qErr := db.OpenQuestions(ctx, userID, &spaceID); qErr != nil || len(questions) != 0 {
		t.Fatalf("the question should be closed first: %d (%v)", len(questions), qErr)
	}

	if err = db.DeleteNote(ctx, userID, answer.ID); err != nil {
		t.Fatal(err)
	}
	questions, err := db.OpenQuestions(ctx, userID, &spaceID)
	if err != nil {
		t.Fatalf("open questions: %v", err)
	}
	if len(questions) != 1 {
		t.Fatal("the question stayed closed by an answer that no longer exists")
	}
}

// The store is where the unvalidated kind was reaching the database, so the
// refusal has to hold there and not only in the parser.
func TestStoreRefusesAnUnknownKindOnBothPathsIntegration(t *testing.T) {
	db, userID, spaceID := questionSpace(t)
	ctx := context.Background()

	if _, err := db.CreateNote(ctx, userID, Note{
		SpaceID: spaceID, AuthorID: userID, Content: "임의 종류", Kind: strings.Repeat("K", 5000),
	}); !errors.Is(err, ErrUnknownKind) {
		t.Errorf("create accepted an arbitrary kind: %v", err)
	}

	good, err := db.CreateNote(ctx, userID, Note{SpaceID: spaceID, AuthorID: userID, Content: "보통 생각"})
	if err != nil {
		t.Fatal(err)
	}
	good.Kind = "totally-made-up"
	if _, err = db.UpdateNote(ctx, userID, good, nil); !errors.Is(err, ErrUnknownKind) {
		t.Errorf("update was a way around the check: %v", err)
	}
	var stored string
	if err = db.Pool.QueryRow(ctx, `SELECT kind FROM notes WHERE id=$1`, good.ID).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if stored != string(KindThought) {
		t.Errorf("the note's kind became %q despite the refusal", stored)
	}
}

// Another person's questions are not this person's to read.
func TestOpenQuestionsStayWithinReachableSpacesIntegration(t *testing.T) {
	db, userID, _ := questionSpace(t)
	otherDB, strangerID, strangerSpace := questionSpace(t)
	ctx := context.Background()

	ask(t, otherDB, strangerID, strangerSpace, "남의 질문")
	questions, err := db.OpenQuestions(ctx, userID, nil)
	if err != nil {
		t.Fatalf("open questions: %v", err)
	}
	for _, item := range questions {
		if item.SpaceID == strangerSpace {
			t.Fatal("another person's question was returned")
		}
	}
}
