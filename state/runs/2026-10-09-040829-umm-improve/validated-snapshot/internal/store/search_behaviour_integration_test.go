package store

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
)

// What searching actually does, against a database.
//
// The pieces have tests: the patterns escape wildcards, the predicate ANDs one
// probe per term, an empty term list matches nothing. What none of them can
// show is the promise those pieces exist for — that typing a percent sign finds
// the thought containing a percent sign, and not every thought there is.
//
// That is the failure worth guarding. Escaping breaking does not look broken:
// search keeps working for ordinary words and quietly returns everything for
// anyone whose notes contain % or _, which is most people writing about
// progress or snake_case.
func TestSearchingFindsWhatWasWrittenIntegration(t *testing.T) {
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		t.Skip("POSTGRES_DSN is not configured")
	}
	ctx := context.Background()
	db := isolatedStore(t, dsn)

	ownerID, spaceID := uuid.New(), uuid.New()
	username := "search_" + strings.ReplaceAll(ownerID.String(), "-", "")
	if _, err := db.Pool.Exec(ctx, `INSERT INTO users(id,username,display_name) VALUES($1,$2::citext,$2::text)`, ownerID, username); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Pool.Exec(ctx, `INSERT INTO spaces(id,owner_id,name) VALUES($1,$2,'검색')`, spaceID, ownerID); err != nil {
		t.Fatal(err)
	}
	written := []string{
		"인증 토큰 만료를 15분으로 줄이자",
		"배포는 50% 완료되었다",
		"snake_case 이름 규칙",
		"AUTH 토큰 대문자",
	}
	for _, content := range written {
		if _, err := db.Pool.Exec(ctx, `INSERT INTO notes(space_id,author_id,content) VALUES($1,$2,$3)`, spaceID, ownerID, content); err != nil {
			t.Fatal(err)
		}
	}

	found := func(query string) []string {
		t.Helper()
		notes, _, err := db.ListNotes(ctx, ownerID, spaceID, query)
		if err != nil {
			t.Fatal(err)
		}
		out := make([]string, 0, len(notes))
		for _, note := range notes {
			out = append(out, note.Content)
		}
		return out
	}

	for _, tc := range []struct {
		query string
		want  int
		note  string
	}{
		{"토큰", 2, "a Korean word inside longer words"},
		{"토큰 만료", 1, "every term has to appear"},
		{"만료 토큰", 1, "the order of the terms does not matter"},
		{"auth", 1, "case is ignored"},
		// The ones the escaping exists for. A percent that matched anything
		// would return all four, and nothing else in the test would notice.
		{"50%", 1, "a percent sign is a character, not a wildcard"},
		{"%", 1, "a bare percent finds only the thought containing one"},
		{"_", 1, "an underscore is a character, not any-single-character"},
		{"snake_case", 1, "an underscore inside a word"},
		{"없는말", 0, "nothing matches, and it says so rather than guessing"},
		// A blank query is not a search; it is the canvas asking for everything.
		{"   ", len(written), "a blank query is not a filter"},
	} {
		if got := found(tc.query); len(got) != tc.want {
			t.Errorf("%s: %q found %d (%v), want %d", tc.note, tc.query, len(got), got, tc.want)
		}
	}
}
