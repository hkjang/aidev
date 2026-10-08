package store

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

// Two copies of the same thought sit at the top of any similarity ranking, so
// they were always the first connection suggested. "Related" understates them —
// the honest reading is that one thought was written twice, and the morning
// review already handles that. Suggesting a link as well makes one situation
// cost two chores.
func TestAutoLinkSkipsNearDuplicatesIntegration(t *testing.T) {
	db, userID, spaceID := retrievalSpace(t)
	ctx := context.Background()

	report, err := db.MeasureEmbeddingQuality(ctx, false)
	if err != nil || !report.Semantic {
		t.Skip("auto-link is gated on a semantic backend, and so is this rule")
	}

	// Enough notes for the run to be worth doing at all, plus one exact copy.
	contents := []string{
		"배포 파이프라인을 젠킨스로 직접 운영하면 통제권을 가져올 수 있다",
		"배포 파이프라인을 젠킨스로 직접 운영하면 통제권을 가져올 수 있다",
		"회고 주기를 월간에서 격주로 줄이는 실험을 해 보자",
		"주기를 짧게 하면 한 번에 다루는 주제가 얕아질 수 있다",
		"온보딩 문서를 다시 쓸지 아직 결정하지 못했다",
		"온보딩 문서는 신입이 첫 주에 막히는 지점부터 다시 쓰는 게 낫다",
		"배포 승인은 당번제로 돌리고 주간 회고에서 조정한다",
	}
	ids := make([]uuid.UUID, len(contents))
	for index, content := range contents {
		ids[index] = uuid.New()
		if _, err = db.Pool.Exec(ctx, `INSERT INTO notes(id,space_id,author_id,content) VALUES($1,$2,$3,$4)`,
			ids[index], spaceID, userID, content); err != nil {
			t.Fatal(err)
		}
	}

	result, err := db.SuggestLinks(ctx, userID, spaceID)
	if err != nil {
		t.Fatal(err)
	}
	if result.Outcome != OutcomeSuggested {
		t.Skipf("nothing was suggested in this workspace (%s); the rule cannot be observed", result.Outcome)
	}

	duplicate := pairKey(ids[0], ids[1])
	for _, edge := range result.Edges {
		if pairKey(edge.SourceID, edge.TargetID) == duplicate {
			t.Fatal("auto-link proposed a connection between two copies of the same thought")
		}
	}
	// And the rest of the run is unaffected: skipping the pair must not skip
	// the workspace.
	if len(result.Edges) == 0 {
		t.Error("skipping the duplicate pair left no suggestions at all")
	}
}
