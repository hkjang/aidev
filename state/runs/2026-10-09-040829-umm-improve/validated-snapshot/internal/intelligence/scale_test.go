package intelligence

import (
	"math"
	"testing"
)

// These two distributions are the ones measured by quality_test.go: the offline
// character n-gram algorithm spreads scores low and wide, a sentence embedding
// model packs them high and narrow. A threshold rule is only correct if it
// separates the same pairs under both.
var (
	charGramScores = []float64{0.19, 0.16, 0.44, 0.18, 0.21, 0.13, 0.39, 0.17, 0.22, 0.15, 0.31, 0.12}
	sentenceScores = []float64{0.70, 0.57, 0.62, 0.36, 0.68, 0.55, 0.64, 0.38, 0.72, 0.53, 0.60, 0.35}
)

func TestScaleSeparatesTheSamePairsUnderBothBackends(t *testing.T) {
	// In each distribution the last quarter are the unrelated pairs and the
	// high end are the genuine matches; a good cutoff admits the second group
	// and rejects the first, whatever the raw numbers look like.
	for _, testCase := range []struct {
		name          string
		scores        []float64
		strongMatch   float64
		unrelatedPair float64
	}{
		{"character n-gram", charGramScores, 0.44, 0.12},
		{"sentence embedding", sentenceScores, 0.72, 0.35},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			scale := NewSimilarityScale(testCase.scores)
			if !scale.Usable() {
				t.Fatalf("scale over %d samples should be usable", len(testCase.scores))
			}
			cutoff := scale.Threshold(BandCluster)
			if testCase.strongMatch < cutoff {
				t.Errorf("strong match %.2f fell below the cluster cutoff %.3f", testCase.strongMatch, cutoff)
			}
			if testCase.unrelatedPair >= cutoff {
				t.Errorf("unrelated pair %.2f cleared the cluster cutoff %.3f", testCase.unrelatedPair, cutoff)
			}
		})
	}
}

// The failure this whole file exists to prevent: a constant tuned for one
// backend admits everything under another.
func TestFixedCutoffAdmitsEverythingUnderASentenceModel(t *testing.T) {
	const legacyRelatedCutoff = .22
	admitted := 0
	for _, score := range sentenceScores {
		if score >= legacyRelatedCutoff {
			admitted++
		}
	}
	if admitted != len(sentenceScores) {
		t.Fatalf("expected the legacy cutoff to admit every sentence-model score, admitted %d of %d",
			admitted, len(sentenceScores))
	}
	scale := NewSimilarityScale(sentenceScores)
	relativeAdmitted := 0
	for _, score := range sentenceScores {
		if score >= scale.Threshold(BandRelated) {
			relativeAdmitted++
		}
	}
	if relativeAdmitted == len(sentenceScores) || relativeAdmitted == 0 {
		t.Fatalf("relative cutoff admitted %d of %d; it must discriminate", relativeAdmitted, len(sentenceScores))
	}
	t.Logf("legacy cutoff admitted %d/%d, relative cutoff admitted %d/%d",
		admitted, len(sentenceScores), relativeAdmitted, len(sentenceScores))
}

// A space with a handful of notes still has to work. Falling back means falling
// back to an absolute cutoff, so the relative rule has to cover the small case
// rather than hand it to the constant that breaks under a different backend.
func TestScaleHandlesASmallSpace(t *testing.T) {
	// Five other notes, the shape RelatedNotes sees in a six-note space, scored
	// by a sentence embedding model.
	scores := []float64{0.72, 0.68, 0.55, 0.38, 0.36}
	scale := NewSimilarityScale(scores)
	if !scale.Usable() {
		t.Fatalf("a five-sample space must not fall back to an absolute cutoff")
	}
	cutoff := scale.Threshold(BandRelated)
	admitted := 0
	for _, score := range scores {
		if score >= cutoff {
			admitted++
		}
	}
	if admitted == 0 || admitted == len(scores) {
		t.Fatalf("cutoff %.3f admitted %d of %d; it must discriminate", cutoff, admitted, len(scores))
	}
	t.Logf("five-sample cutoff %.3f admitted %d of %d", cutoff, admitted, len(scores))
}

func TestScaleFallsBackOnThinOrFlatSamples(t *testing.T) {
	const fallback = .22
	if got := NewSimilarityScale(nil).ThresholdOr(BandRelated, fallback); got != fallback {
		t.Errorf("empty sample should use the fallback, got %.3f", got)
	}
	if got := NewSimilarityScale([]float64{.4, .5, .6}).ThresholdOr(BandRelated, fallback); got != fallback {
		t.Errorf("three samples describe no distribution, got %.3f", got)
	}
	flat := make([]float64, 12)
	for i := range flat {
		flat[i] = .5
	}
	if got := NewSimilarityScale(flat).ThresholdOr(BandRelated, fallback); got != fallback {
		t.Errorf("a distribution with no spread carries no signal to threshold on, got %.3f", got)
	}
}

func TestScaleStatistics(t *testing.T) {
	scale := NewSimilarityScale([]float64{2, 4, 4, 4, 5, 5, 7, 9})
	if math.Abs(scale.Mean-5) > 1e-9 {
		t.Errorf("mean: want 5, got %v", scale.Mean)
	}
	// Sample standard deviation of that set is exactly sqrt(32/7).
	wantStdDev := math.Sqrt(32.0 / 7.0)
	if math.Abs(scale.StdDev-wantStdDev) > 1e-9 {
		t.Errorf("stddev: want %v, got %v", wantStdDev, scale.StdDev)
	}
	if got := scale.Threshold(1); math.Abs(got-(5+wantStdDev)) > 1e-9 {
		t.Errorf("threshold at one deviation: want %v, got %v", 5+wantStdDev, got)
	}
}

func TestTypical(t *testing.T) {
	if got := Typical([]float64{.1, .5, .9}); math.Abs(got-.5) > 1e-9 {
		t.Errorf("odd count median: want .5, got %v", got)
	}
	if got := Typical([]float64{.2, .4, .6, .8}); math.Abs(got-.5) > 1e-9 {
		t.Errorf("even count median: want .5, got %v", got)
	}
	if got := Typical(nil); got != 0 {
		t.Errorf("empty median should be zero, got %v", got)
	}
	// Typical must not be perturbed by one extreme score the way a mean is.
	if got := Typical([]float64{.4, .5, .6, 40}); math.Abs(got-.55) > 1e-9 {
		t.Errorf("median should resist an outlier, got %v", got)
	}
}

// BridgeScore has to keep rating pairs under a backend where everything scores
// high. The old fixed peak at 0.35 returns exactly zero for any pair above 0.70,
// which is most genuinely related pairs once a sentence model is configured.
func TestBridgeScoreFollowsTheBackendDistribution(t *testing.T) {
	const legacyPeak = .35
	legacyBridge := func(similarity float64) float64 {
		return math.Max(0, 1-math.Abs(similarity-legacyPeak)/legacyPeak)
	}
	typical, highest := Typical(sentenceScores), 0.72
	deadUnderLegacy := 0
	ratedUnderRelative := 0
	for _, score := range sentenceScores {
		if legacyBridge(score) == 0 {
			deadUnderLegacy++
		}
		if BridgeScore(score, typical, highest) > 0 {
			ratedUnderRelative++
		}
	}
	if deadUnderLegacy == 0 {
		t.Fatal("expected the legacy peak to zero out sentence-model pairs")
	}
	if ratedUnderRelative <= deadUnderLegacy {
		t.Fatalf("relative bridge rated %d pairs, legacy left %d dead; the fix must rate more",
			ratedUnderRelative, deadUnderLegacy)
	}
	t.Logf("legacy peak zeroed %d/%d sentence-model pairs; relative bridge rates %d/%d",
		deadUnderLegacy, len(sentenceScores), ratedUnderRelative, len(sentenceScores))

	// The peak must still sit between the typical and the strongest pair, so a
	// near-duplicate is not treated as the most interesting bridge.
	peak := typical + (highest-typical)/2
	if BridgeScore(peak, typical, highest) <= BridgeScore(highest, typical, highest) {
		t.Error("a mid-band pair should outrank a near-duplicate as a bridge")
	}
	if BridgeScore(peak, typical, highest) <= BridgeScore(typical, typical, highest) {
		t.Error("a mid-band pair should outrank a merely typical pair as a bridge")
	}
}

func TestBridgeScoreHandlesADegenerateSpread(t *testing.T) {
	// Every pair identical: no bridge is better than another, and nothing may
	// divide by zero.
	if got := BridgeScore(.5, .5, .5); math.IsNaN(got) || math.IsInf(got, 0) {
		t.Fatalf("degenerate spread produced %v", got)
	}
}
