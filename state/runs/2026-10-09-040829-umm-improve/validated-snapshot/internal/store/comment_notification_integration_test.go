package store

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
)

// Who a comment notifies, and who it does not.
//
// The rule is three conditions on one line:
//
//	if noteAuthor != userID && noteAuthorCanView && !mentioned[noteAuthor]
//
// Only the middle one had a test. The other two are the difference between a
// notification worth reading and one that trains people to ignore the bell:
// being told about your own comment, and being told twice about a comment that
// already named you.
func TestWhoACommentNotifiesIntegration(t *testing.T) {
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		t.Skip("POSTGRES_DSN is not configured")
	}
	ctx := context.Background()
	db := isolatedStore(t, dsn)

	authorID, otherID := uuid.New(), uuid.New()
	name := func(id uuid.UUID) string { return "notify_" + strings.ReplaceAll(id.String(), "-", "") }
	if _, err := db.Pool.Exec(ctx, `INSERT INTO users(id,username,display_name) VALUES($1,$2::citext,$2::text),($3,$4::citext,$4::text)`,
		authorID, name(authorID), otherID, name(otherID)); err != nil {
		t.Fatal(err)
	}
	spaceID, noteID := uuid.New(), uuid.New()
	if _, err := db.Pool.Exec(ctx, `INSERT INTO spaces(id,owner_id,name) VALUES($1,$2,'댓글 알림')`, spaceID, authorID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Pool.Exec(ctx, `INSERT INTO space_members(space_id,user_id,permission) VALUES($1,$2,'edit')`, spaceID, otherID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Pool.Exec(ctx, `INSERT INTO notes(id,space_id,author_id,content) VALUES($1,$2,$3,'생각')`, noteID, spaceID, authorID); err != nil {
		t.Fatal(err)
	}

	kinds := func(userID uuid.UUID) []string {
		t.Helper()
		rows, err := db.Pool.Query(ctx, `SELECT kind FROM notifications WHERE user_id=$1 ORDER BY created_at`, userID)
		if err != nil {
			t.Fatal(err)
		}
		defer rows.Close()
		out := []string{}
		for rows.Next() {
			var kind string
			if err := rows.Scan(&kind); err != nil {
				t.Fatal(err)
			}
			out = append(out, kind)
		}
		return out
	}
	joined := func(userID uuid.UUID) string { return strings.Join(kinds(userID), ",") }

	// Commenting on your own thought tells you nothing you did not just do.
	if _, _, err := db.CreateComment(ctx, authorID, noteID, nil, "내가 내 생각에 다는 댓글", nil); err != nil {
		t.Fatal(err)
	}
	if got := joined(authorID); got != "" {
		t.Errorf("commenting on your own thought notified you: %q", got)
	}

	// Someone else's comment does.
	if _, _, err := db.CreateComment(ctx, otherID, noteID, nil, "다른 사람의 댓글", nil); err != nil {
		t.Fatal(err)
	}
	if got := joined(authorID); got != "comment" {
		t.Errorf("a comment from someone else gave the author %q, want one comment", got)
	}
	if got := joined(otherID); got != "" {
		t.Errorf("the commenter was notified about their own comment: %q", got)
	}

	// A comment that names the author is one notification, not two. Being told
	// twice about the same sentence is how a bell stops being read.
	if _, _, err := db.CreateComment(ctx, otherID, noteID, nil, "@"+name(authorID)+" 확인 부탁드립니다", []string{name(authorID)}); err != nil {
		t.Fatal(err)
	}
	if got := joined(authorID); got != "comment,mention" {
		t.Errorf("a comment naming the author gave %q, want the earlier comment plus one mention", got)
	}
}
