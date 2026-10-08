package store

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/hkjang/umm/internal/intelligence"
)

func cosineOf(vectors map[uuid.UUID][]float32, a, b uuid.UUID) float64 {
	return float64(intelligence.Cosine(vectors[a], vectors[b]))
}

func similarityCutoffForTest(db *Store, ctx context.Context, scores []float64) float64 {
	return intelligence.NewSimilarityScale(scores).ThresholdOr(
		intelligence.Band(db.IntelligenceSettings(ctx).RelatedBand), legacyRelatedCutoff)
}

// The count a canvas shows on each thought must be the same whether it is
// computed from a fresh pass over the square or read back from the scores
// already taken. Similarity is symmetric, so a pair stored once can be read
// from either side — this pins that the index arithmetic that does so keeps
// picking the right row.
//
// The line is drawn per thought, from that thought's own scores, because that
// is the line the list it opens is drawn from. It used to be one line across
// every pair in the space, and the number and the list disagreed.
func TestRelatedCountMatchesTheFullSquareIntegration(t *testing.T) {
	db, userID, spaceID := retrievalSpace(t)
	ctx := context.Background()

	contents := []string{
		"인증 토큰 만료 시간을 24시간으로 정했다",
		"세션 쿠키는 HttpOnly와 SameSite를 함께 설정한다",
		"로그인 실패가 반복되면 계정을 일시적으로 잠근다",
		"주말에 자전거를 타고 한강을 따라 달렸다",
		"자전거 체인에 기름을 새로 발랐다",
		"라이딩이 끝나면 스트레칭을 꼭 한다",
		"배포 승인은 당번을 정해 돌아가며 맡는다",
	}
	for _, content := range contents {
		if _, err := db.Pool.Exec(ctx, `INSERT INTO notes(id,space_id,author_id,content) VALUES($1,$2,$3,$4)`,
			uuid.New(), spaceID, userID, content); err != nil {
			t.Fatal(err)
		}
	}

	notes, _, err := db.ListNotes(ctx, userID, spaceID, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(notes) != len(contents) {
		t.Fatalf("listed %d notes, want %d", len(notes), len(contents))
	}

	// The same counts, computed the obvious way: for each thought, score it
	// against every other, draw the line from those scores, count what clears.
	vectors := db.loadEmbeddings(ctx, notes)
	want := make([]int, len(notes))
	for i := range notes {
		row := make([]float64, 0, len(notes)-1)
		for j := range notes {
			if i != j {
				row = append(row, cosineOf(vectors, notes[i].ID, notes[j].ID))
			}
		}
		cutoff := similarityCutoffForTest(db, ctx, row)
		for _, score := range row {
			if score >= cutoff {
				want[i]++
			}
		}
	}

	for i, note := range notes {
		if note.RelatedCount != want[i] {
			t.Errorf("note %d: relatedCount=%d, the full square says %d (%q)",
				i, note.RelatedCount, want[i], note.Content)
		}
	}
}
