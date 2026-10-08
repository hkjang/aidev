package store

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/hkjang/umm/internal/intelligence"
)

// What the shipped cluster band does to an ordinary space.
//
// Writing the preview turned up a corpus where the default band grouped
// nothing at all, and the first reading of that was alarming: spaces do divide
// into distinct subjects, and a band that groups nothing empties the summarised
// canvas. Measuring said otherwise — across two to eight subjects, with and
// without shared vocabulary, the default groups nearly everything. What
// actually breaks it is a space of near-identical notes, where the pair scores
// are two spikes rather than two humps and mean + 1.1 stddev lands above even
// the within-subject scores.
//
// Both halves are pinned here. The first is the regression that matters: a
// change to the default, to the scale, or to the embedding that quietly emptied
// the canvas would fail it. The second is what gives the first its teeth — it
// proves this measurement can see a collapse when there is one, so passing the
// first means something.

// vocabularies share no characters between subjects, so how much the subjects
// overlap is controlled by the borrowing below rather than by accident.
var vocabularies = [][]string{
	{"고양이", "사료", "급여", "간격", "보관", "습식", "건식", "체중"},
	{"쿠버네티스", "배포", "롤백", "헬름", "차트", "파드", "네임스페이스", "인그레스"},
	{"환율", "금리", "채권", "물가", "수출", "재정", "적자", "성장률"},
	{"등산", "능선", "배낭", "산장", "야영", "코스", "고도", "일출"},
}

// syntheticSpace builds a space of subjects, each note drawing `borrowed` of
// its words from a neighbouring subject. Real notes share vocabulary across
// topics; borrowed=0 is the idealised case where they do not.
func syntheticSpace(rng *rand.Rand, subjects, perSubject, words, borrowed int) [][]float32 {
	vectors := make([][]float32, 0, subjects*perSubject)
	for subject := 0; subject < subjects; subject++ {
		for i := 0; i < perSubject; i++ {
			text := ""
			for w := 0; w < words; w++ {
				pool := vocabularies[subject%len(vocabularies)]
				if w < borrowed {
					pool = vocabularies[(subject+1)%len(vocabularies)]
				}
				text += pool[rng.Intn(len(pool))] + " "
			}
			vectors = append(vectors, intelligence.Embed(text))
		}
	}
	return vectors
}

func groupedPercent(vectors [][]float32, band float64) (BandOutcome, int) {
	outcome := BandOutcome{}
	addClustering(&outcome, intelligence.Prepare(vectors), band)
	if len(vectors) == 0 {
		return outcome, 0
	}
	return outcome, 100 * outcome.Grouped / len(vectors)
}

func TestShippedClusterBandGroupsAnOrdinarySpace(t *testing.T) {
	band := DefaultIntelligenceSettings().ClusterBand
	for _, subjects := range []int{2, 3, 4} {
		for _, borrowed := range []int{0, 2, 4} {
			name := fmt.Sprintf("%d subjects, %d of 12 words borrowed", subjects, borrowed)
			rng := rand.New(rand.NewSource(int64(subjects*10 + borrowed)))
			vectors := syntheticSpace(rng, subjects, 30, 12, borrowed)
			outcome, percent := groupedPercent(vectors, band)
			// Well below what was measured (91-100%) so ordinary variation does
			// not fail it, and well above the collapse the companion shows.
			if percent < 70 {
				t.Fatalf("%s: the shipped band %.2f grouped %d%% of %d notes into %d groups — "+
					"the summarised canvas would draw most of this space one note at a time",
					name, band, percent, len(vectors), outcome.Clusters)
			}
			if outcome.Grouped+outcome.Ungrouped != len(vectors) {
				t.Fatalf("%s: %d grouped + %d alone != %d notes", name, outcome.Grouped, outcome.Ungrouped, len(vectors))
			}
			if outcome.LargestCluster > outcome.Grouped {
				t.Fatalf("%s: largest group %d exceeds the %d grouped", name, outcome.LargestCluster, outcome.Grouped)
			}
		}
	}
}

// The companion. Without it the test above would pass just as happily if
// addClustering grouped everything unconditionally.
func TestNearIdenticalNotesAreWhatCollapsesTheClusterBand(t *testing.T) {
	band := DefaultIntelligenceSettings().ClusterBand
	sentences := []string{
		"고양이 사료 급여 간격과 사료 보관 방법",
		"쿠버네티스 배포 롤백 절차와 헬름 차트 버전",
	}
	// Every note in a subject is the same sentence with an index on the end.
	identical := make([][]float32, 0, 60)
	for i := 0; i < 60; i++ {
		identical = append(identical, intelligence.Embed(fmt.Sprintf("%s %d번 메모", sentences[i%2], i)))
	}
	_, percent := groupedPercent(identical, band)
	if percent > 50 {
		t.Fatalf("near-identical notes grouped %d%%, so this measurement cannot see a collapse "+
			"and the ordinary-space test above proves nothing", percent)
	}

	// And the point of the whole exercise: one word of variation per note is
	// enough to bring it back. That is why the collapse is a property of
	// duplicate text rather than of distinct subjects.
	rng := rand.New(rand.NewSource(7))
	varied := make([][]float32, 0, 60)
	for i := 0; i < 60; i++ {
		pool := vocabularies[i%2]
		varied = append(varied, intelligence.Embed(
			fmt.Sprintf("%s %d번 메모 %s", sentences[i%2], i, pool[rng.Intn(len(pool))])))
	}
	if _, recovered := groupedPercent(varied, band); recovered < 70 {
		t.Fatalf("one varying word per note recovered only %d%%, so the collapse is not about duplicate text after all", recovered)
	}
}
