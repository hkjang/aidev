package intelligence

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"
	"time"
)

// The dataset and the scoring live in quality.go, because the administrator
// screen runs the same measurement against whatever backend is configured. This
// file is where the numbers become assertions: what must not get worse, what a
// candidate model has to beat, and what the default algorithm currently cannot do.

// discriminationFloor is the recorded behaviour of the default offline
// algorithm. It is deliberately a floor and not a target: the value is negative
// because character n-gram hashing scores shared substrings above shared
// meaning. Raising it is the point of any embedding work; this constant exists
// so the number can never quietly get worse.
// Measured at -0.250 on this dataset; the floor sits just below so a
// regression trips it while normal variation does not.
const discriminationFloor = -0.30

// pairAccuracyFloor is the fraction of (paraphrase, lexical-decoy) comparisons
// the algorithm must get the right way round. Measured at 4.2%: the offline
// default answers this question correctly in one comparison out of twenty-four,
// which is what makes every "semantic" feature built on top of it lexical.
const pairAccuracyFloor = 0.02

func measure(t *testing.T, provider Provider) QualityReport {
	t.Helper()
	report, err := MeasureQuality(context.Background(), provider, DefaultQualityBars())
	if err != nil {
		t.Fatalf("measure quality: %v", err)
	}
	return report
}

func reportTable(t *testing.T, name string, report QualityReport) {
	t.Helper()
	var builder strings.Builder
	fmt.Fprintf(&builder, "\n%s\n", name)
	fmt.Fprintf(&builder, "  %-14s %6s %6s %6s %5s\n", "class", "mean", "min", "max", "n")
	for _, class := range report.Classes {
		fmt.Fprintf(&builder, "  %-14s %6.3f %6.3f %6.3f %5d\n",
			class.Class, class.Mean, class.Min, class.Max, class.Count)
	}
	fmt.Fprintf(&builder, "  discrimination (paraphrase - lexical-decoy): %+.3f\n", report.Discrimination)
	fmt.Fprintf(&builder, "  pairwise accuracy (paraphrase > decoy):      %6.1f%%\n", report.PairwiseAccuracy*100)
	fmt.Fprintf(&builder, "  topic separation (same - different):        %+.3f\n", report.TopicSeparation)
	fmt.Fprintf(&builder, "  nearest neighbour shares its topic:         %6.1f%% (%d sentences)\n",
		report.NeighbourPurity*100, report.Sentences)
	fmt.Fprintf(&builder, "  usable as a semantic layer:                 %v\n", report.Semantic)
	t.Log(builder.String())
}

func classMeanIn(report QualityReport, class PairClass) float64 {
	for _, score := range report.Classes {
		if score.Class == class {
			return score.Mean
		}
	}
	return 0
}

// TestEmbeddingQualityIsMeasured records what the active algorithm actually
// captures. It fails only on regression, so it documents the ceiling rather
// than demanding one.
func TestEmbeddingQualityIsMeasured(t *testing.T) {
	report := measure(t, Provider{})
	reportTable(t, "offline default (character n-gram hashing)", report)

	if report.Discrimination < discriminationFloor {
		t.Errorf("discrimination fell to %+.3f, below the recorded floor of %+.3f",
			report.Discrimination, discriminationFloor)
	}
	if report.PairwiseAccuracy < pairAccuracyFloor {
		t.Errorf("pairwise accuracy fell to %.1f%%, below the recorded floor of %.1f%%",
			report.PairwiseAccuracy*100, pairAccuracyFloor*100)
	}
	if report.Algorithm != LocalAlgorithm {
		t.Errorf("the zero-value provider must use the offline algorithm, reported %q", report.Algorithm)
	}
}

// TestOfflineDefaultIsNotASemanticBackend is the assertion the administrator
// screen depends on: the default must report itself as lexical, so the warning
// that tells an operator to configure a model actually appears.
func TestOfflineDefaultIsNotASemanticBackend(t *testing.T) {
	if report := measure(t, Provider{}); report.Semantic {
		t.Fatal("the offline default reported itself as a semantic backend; " +
			"the administrator warning would never be shown")
	}
}

// TestOfflineDefaultRanksVocabularyAboveMeaning states the deficiency as an
// assertion rather than a comment. When an embedding backend finally inverts
// this, the test fails and must be rewritten — which is exactly the moment the
// features built on top stop being lexical.
func TestOfflineDefaultRanksVocabularyAboveMeaning(t *testing.T) {
	report := measure(t, Provider{})
	paraphrase := classMeanIn(report, ClassParaphrase)
	decoy := classMeanIn(report, ClassLexicalDecoy)
	if paraphrase > decoy {
		t.Fatalf("the offline default now scores meaning (%.3f) above vocabulary (%.3f): "+
			"update discriminationFloor and pairAccuracyFloor to lock in the improvement", paraphrase, decoy)
	}
	t.Logf("offline default scores vocabulary %.3f above meaning %.3f (gap %.3f)", decoy, paraphrase, decoy-paraphrase)
}

// TestRelatedThoughtThresholdMissesParaphrases connects the vector measurement
// to a behaviour a person would notice. The store treats 0.22 as "related" and
// 0.34 as "same cluster"; this records how many genuine paraphrases clear those
// bars today.
func TestRelatedThoughtThresholdMissesParaphrases(t *testing.T) {
	const (
		relatedThreshold = .22
		clusterThreshold = .34
	)
	vectors, _ := Provider{}.Embed(context.Background(), pairTexts())
	related, clustered, paraphrases := 0, 0, 0
	decoysClustered, decoys := 0, 0
	for index, pair := range QualityPairs {
		score := Cosine(vectors[index*2], vectors[index*2+1])
		switch pair.Class {
		case ClassParaphrase:
			paraphrases++
			if score >= relatedThreshold {
				related++
			}
			if score >= clusterThreshold {
				clustered++
			}
		case ClassLexicalDecoy:
			decoys++
			if score >= clusterThreshold {
				decoysClustered++
			}
		}
	}
	t.Logf("of %d paraphrases: %d reach the related bar (%.2f), %d reach the cluster bar (%.2f)",
		paraphrases, related, relatedThreshold, clustered, clusterThreshold)
	t.Logf("of %d lexical decoys: %d reach the cluster bar and would be grouped as one topic",
		decoys, decoysClustered)
}

func pairTexts() []string {
	texts := make([]string, 0, len(QualityPairs)*2)
	for _, pair := range QualityPairs {
		texts = append(texts, pair.A, pair.B)
	}
	return texts
}

// TestGatewayEmbeddingQuality runs the same measurement against a configured
// gateway so a candidate model can be compared on identical ground. It is
// skipped unless UMM_EMBEDDING_TEST_URL and UMM_EMBEDDING_TEST_MODEL are set:
//
//	UMM_EMBEDDING_TEST_URL=http://127.0.0.1:11434 \
//	UMM_EMBEDDING_TEST_MODEL=bge-m3 \
//	go test ./internal/intelligence -run Gateway -v
func TestGatewayEmbeddingQuality(t *testing.T) {
	baseURL := strings.TrimSpace(os.Getenv("UMM_EMBEDDING_TEST_URL"))
	model := strings.TrimSpace(os.Getenv("UMM_EMBEDDING_TEST_MODEL"))
	if baseURL == "" || model == "" {
		t.Skip("set UMM_EMBEDDING_TEST_URL and UMM_EMBEDDING_TEST_MODEL to compare a candidate model")
	}
	provider := Provider{Remote: &RemoteConfig{
		BaseURL: baseURL,
		APIKey:  os.Getenv("UMM_EMBEDDING_TEST_KEY"),
		Model:   model,
		Timeout: 60 * time.Second,
	}}
	if _, err := provider.EmbedStrict(context.Background(), []string{"probe"}); err != nil {
		t.Fatalf("gateway unreachable: %v", err)
	}
	report := measure(t, provider)
	reportTable(t, "gateway: "+model, report)

	// The measurement is worthless if the vectors quietly fell back to the local
	// algorithm, so check what actually produced them before judging the model.
	if report.Algorithm == LocalAlgorithm {
		t.Fatal("vectors came from the offline algorithm, not the gateway; the candidate was never measured")
	}
	// A candidate worth adopting has to get the basic question right. These are
	// the numbers to beat, not a claim that the candidate will.
	if report.Discrimination <= 0 {
		t.Errorf("candidate scores vocabulary above meaning (discrimination %+.3f); it is not an improvement",
			report.Discrimination)
	}
	if report.PairwiseAccuracy < 0.8 {
		t.Errorf("candidate ranks a paraphrase above a lexical decoy only %.1f%% of the time",
			report.PairwiseAccuracy*100)
	}
	// Topic grouping is what related thoughts and clustering actually do, and a
	// backend can score well on isolated pairs while mixing subjects.
	if report.NeighbourPurity < defaultSemanticPurityBar {
		t.Errorf("only %.1f%% of sentences had a same-topic nearest match; clusters would mix subjects",
			report.NeighbourPurity*100)
	}
	if report.TopicSeparation <= 0 {
		t.Errorf("topic separation %+.3f: different subjects score as close as the same subject",
			report.TopicSeparation)
	}
	if !report.Semantic {
		t.Errorf("candidate did not clear the semantic bar; umm would still warn the operator about it")
	}
}

// TestSemanticPairsAreWellFormed guards the dataset itself. A duplicated or
// empty pair would quietly skew every number above.
func TestSemanticPairsAreWellFormed(t *testing.T) {
	seen := map[string]bool{}
	counts := map[PairClass]int{}
	for index, pair := range QualityPairs {
		if strings.TrimSpace(pair.A) == "" || strings.TrimSpace(pair.B) == "" {
			t.Fatalf("pair %d has an empty side", index)
		}
		if pair.A == pair.B {
			t.Fatalf("pair %d compares a sentence with itself", index)
		}
		key := pair.A + "\x00" + pair.B
		if seen[key] {
			t.Fatalf("pair %d is a duplicate", index)
		}
		seen[key] = true
		counts[pair.Class]++
	}
	for _, class := range []PairClass{ClassParaphrase, ClassRelated, ClassLexicalDecoy, ClassUnrelated} {
		if counts[class] < 4 {
			t.Errorf("class %s has only %d pairs; too few to average", class, counts[class])
		}
	}
	classes := make([]string, 0, len(counts))
	for class, count := range counts {
		classes = append(classes, fmt.Sprintf("%s=%d", class, count))
	}
	sort.Strings(classes)
	t.Logf("dataset: %d pairs (%s)", len(QualityPairs), strings.Join(classes, ", "))
}

// TestQualityTopicsAreWellFormed guards the topic dataset. A sentence repeated
// across two groups, or a group too small to average, would quietly corrupt both
// topic numbers — and those numbers gate whether umm calls a backend semantic.
func TestQualityTopicsAreWellFormed(t *testing.T) {
	seen := map[string]int{}
	for index, group := range QualityTopics {
		if strings.TrimSpace(group.Topic) == "" {
			t.Fatalf("topic %d has no name", index)
		}
		if len(group.Sentences) < 3 {
			t.Errorf("topic %q has only %d sentences; too few to average", group.Topic, len(group.Sentences))
		}
		for _, sentence := range group.Sentences {
			if strings.TrimSpace(sentence) == "" {
				t.Fatalf("topic %q has an empty sentence", group.Topic)
			}
			if previous, ok := seen[sentence]; ok {
				t.Fatalf("sentence %q appears in topics %d and %d", sentence, previous, index)
			}
			seen[sentence] = index
		}
	}
	if len(QualityTopics) < 3 {
		t.Fatalf("only %d topics; cross-topic scores need more than a single contrast", len(QualityTopics))
	}
	t.Logf("dataset: %d topics, %d sentences", len(QualityTopics), len(seen))
}

// The offline algorithm must fail the topic measurement too. If it ever passes,
// the purity bar is no longer separating a lexical backend from a semantic one
// and needs to be re-derived from fresh measurements.
func TestOfflineDefaultCannotGroupTopics(t *testing.T) {
	report := measure(t, Provider{})
	if report.NeighbourPurity >= defaultSemanticPurityBar {
		t.Fatalf("the offline algorithm reached %.1f%% same-topic neighbours, at or above the %.1f%% bar: "+
			"re-derive defaultSemanticPurityBar from current measurements", report.NeighbourPurity*100, defaultSemanticPurityBar*100)
	}
	t.Logf("offline default: %.1f%% same-topic nearest matches, topic separation %+.3f",
		report.NeighbourPurity*100, report.TopicSeparation)
}

// The bug this pins: the quality report is cached, and the verdict depends on
// thresholds an administrator can change. Returning the stored verdict meant a
// raised bar had no effect until the cache expired — measured live, auto-link
// kept running on a backend the new bar excluded.
//
// The measurements do not depend on the bars, so re-judging is exact and free.
func TestVerdictFollowsTheBarsWithoutRemeasuring(t *testing.T) {
	report := measure(t, Provider{})
	if report.Semantic {
		t.Fatal("the offline default should not clear the shipped bars")
	}

	// Lowering the bars to nothing must still not pass this backend: positive
	// discrimination is a floor, not a setting. An administrator can decide how
	// strict umm is, but not that a backend ranking vocabulary above meaning
	// counts as semantic — that is the failure the whole gate exists to prevent.
	if floored := report.WithBars(QualityBars{Accuracy: 0.01, Purity: 0.01}); floored.Semantic {
		t.Errorf("bars of 0.01 admitted a backend with discrimination %+.3f; "+
			"meaning outranking vocabulary must not be configurable away", report.Discrimination)
	}

	// A backend that does separate meaning follows the bars in both directions.
	working := QualityReport{Discrimination: 0.08, PairwiseAccuracy: 0.83, NeighbourPurity: 0.75}
	if !working.WithBars(QualityBars{Accuracy: 0.65, Purity: 0.6}).Semantic {
		t.Error("a backend measured above the shipped bars should pass")
	}
	strict := working.WithBars(QualityBars{Accuracy: 0.9, Purity: 0.6})
	if strict.Semantic {
		t.Error("raising the accuracy bar above the measurement should fail the verdict")
	}

	// Re-judging must not disturb the measurements themselves.
	if strict.PairwiseAccuracy != working.PairwiseAccuracy || strict.NeighbourPurity != working.NeighbourPurity {
		t.Error("re-judging changed the measurements it was supposed to re-read")
	}
	// The report has to say which bars produced the verdict, or a reader cannot
	// tell a strict deployment from a failing backend.
	if strict.AccuracyBar != 0.9 {
		t.Errorf("the report did not record the bar it was judged against: %.3f", strict.AccuracyBar)
	}
}

// An out-of-range bar is not a stricter rule, it is one that can never be met or
// never fails, so it falls back to the measured default.
func TestQualityBarsNormalize(t *testing.T) {
	d := DefaultQualityBars()
	for _, broken := range []QualityBars{
		{Accuracy: 0, Purity: 0.6},
		{Accuracy: 1.4, Purity: 0.6},
		{Accuracy: 0.65, Purity: -1},
		{Accuracy: 0.65, Purity: 2},
	} {
		got := broken.normalized()
		if got.Accuracy <= 0 || got.Accuracy > 1 || got.Purity <= 0 || got.Purity > 1 {
			t.Errorf("normalizing %+v left an unusable bar: %+v", broken, got)
		}
	}
	if d.normalized() != d {
		t.Error("the defaults must survive normalization unchanged")
	}
}
