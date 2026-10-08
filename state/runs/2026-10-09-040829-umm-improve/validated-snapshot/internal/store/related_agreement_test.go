package store

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

// The number on a card and the list it opens have to be the same thing.
//
// The card showed a count drawn across every pair in the space at the cluster
// band; pressing it opened a list drawn from that one thought's scores at the
// related band. The bands differ by default, so they disagreed — measured on
// this very fixture, five of seven thoughts. Three of them showed no number at
// all, because a count of zero hides the chip, while the list had one or two
// thoughts waiting behind it that nobody would ever be told about.
func TestTheNumberOnACardMatchesWhatOpeningItShowsIntegration(t *testing.T) {
	db, userID, spaceID := retrievalSpace(t)
	ctx := context.Background()
	texts := []string{
		"인증 토큰 만료를 15분으로 줄이자",
		"토큰 만료가 짧으면 재로그인이 잦아진다",
		"세션 토큰 갱신 주기를 다시 보자",
		"배포 파이프라인을 젠킨스에서 옮기는 문제",
		"젠킨스 플러그인 호환이 발목을 잡는다",
		"회고 주기를 격주로 바꾸는 안",
		"점심 메뉴는 국수",
	}
	ids := make([]uuid.UUID, len(texts))
	for i, text := range texts {
		note, err := db.CreateNote(ctx, userID, Note{SpaceID: spaceID, AuthorID: userID, Content: text})
		if err != nil {
			t.Fatal(err)
		}
		ids[i] = note.ID
	}
	notes, _, err := db.ListNotes(ctx, userID, spaceID, "")
	if err != nil {
		t.Fatal(err)
	}
	byID := make(map[uuid.UUID]Note, len(notes))
	for _, n := range notes {
		byID[n.ID] = n
	}

	// Asked for more than the space holds, so the list is never the limit and
	// any difference is a real disagreement rather than paging.
	for i, id := range ids {
		panel, relatedErr := db.RelatedNotes(ctx, userID, id, 20)
		if relatedErr != nil {
			t.Fatal(relatedErr)
		}
		if got, want := byID[id].RelatedCount, len(panel); got != want {
			t.Errorf("thought %d says %d related but opens %d: %s", i, got, want, texts[i])
		}
	}
}
