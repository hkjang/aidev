package intelligence

import (
	"math"
	"sort"
)

// SimilarityScale turns raw cosine scores into decisions that survive a change
// of embedding backend.
//
// Absolute cutoffs cannot do that. The offline character n-gram algorithm puts
// unrelated notes near 0.18 and the strongest matches near 0.34, so a "related"
// bar at 0.22 is reasonable there. A modern sentence embedding model compresses
// everything into a much narrower, much higher band — unrelated pairs sit near
// 0.36 and paraphrases near 0.70 — where that same 0.22 bar admits every note in
// the workspace and a 0.34 cluster bar collapses them all into one topic.
//
// The fix is to stop asking "is this score above a constant" and start asking
// "is this score unusually high for this set of candidates". The bands below are
// expressed in standard deviations above the mean, which carries the same
// meaning whichever backend produced the numbers.
type SimilarityScale struct {
	Mean    float64
	StdDev  float64
	Samples int
}

// Band names the strength of a relationship, independent of the raw scale the
// active algorithm happens to use.
type Band float64

const (
	// BandRelated is loose enough to suggest a connection worth showing.
	BandRelated Band = 0.6
	// BandCluster demands a clearly above-average match before two notes are
	// presented as one topic.
	BandCluster Band = 1.1
	// BandStrong marks a match close enough to call out explicitly, such as
	// labelling a search hit as semantically similar.
	BandStrong Band = 0.9
)

// minScaleSamples is the point below which a mean and deviation say more about
// sampling noise than about the corpus. Under it the scale reports that it is
// not usable and callers fall back to their legacy constant.
//
// It is deliberately low. Falling back means falling back to an absolute cutoff,
// which is the very thing that breaks when the embedding backend changes, so the
// relative rule should give way only when there is genuinely nothing to measure.
// Related thoughts, in particular, see one score per other note in the space: a
// six-note space offers five samples, and requiring more would send every small
// workspace down the broken path.
const minScaleSamples = 4

// NewSimilarityScale summarises the observed distribution of a candidate set.
// Callers pass every score they are about to judge, not a curated subset — the
// mean is only meaningful if it covers the same population as the decision.
func NewSimilarityScale(scores []float64) SimilarityScale {
	scale := SimilarityScale{Samples: len(scores)}
	if len(scores) == 0 {
		return scale
	}
	for _, score := range scores {
		scale.Mean += score
	}
	scale.Mean /= float64(len(scores))
	if len(scores) < 2 {
		return scale
	}
	variance := 0.0
	for _, score := range scores {
		delta := score - scale.Mean
		variance += delta * delta
	}
	// Sample variance: the scores are a sample of the corpus, not the whole of it.
	scale.StdDev = math.Sqrt(variance / float64(len(scores)-1))
	return scale
}

// Usable reports whether the distribution has enough spread and enough samples
// to place a meaningful threshold. A corpus where every pair scores identically
// carries no signal to threshold on.
func (s SimilarityScale) Usable() bool {
	return s.Samples >= minScaleSamples && s.StdDev > 1e-6
}

// Threshold places a cutoff at the requested number of standard deviations above
// the mean. Callers must check Usable first; on an unusable scale this returns
// the mean, which admits roughly half the candidates rather than none or all.
func (s SimilarityScale) Threshold(band Band) float64 {
	return s.Mean + float64(band)*s.StdDev
}

// ThresholdOr returns the distribution-relative cutoff when the sample supports
// one and the supplied constant otherwise. This is the form call sites use, so
// that a workspace with three notes behaves exactly as it did before.
func (s SimilarityScale) ThresholdOr(band Band, fallback float64) float64 {
	if !s.Usable() {
		return fallback
	}
	return s.Threshold(band)
}

// Typical reports the median of the observed scores, which is the centre a
// relationship should be measured against when the goal is "moderately related"
// rather than "as similar as possible". Dream uses it to aim at a genuine bridge
// instead of a fixed cosine value that only suits one backend.
func Typical(scores []float64) float64 {
	if len(scores) == 0 {
		return 0
	}
	ordered := append([]float64(nil), scores...)
	sort.Float64s(ordered)
	middle := len(ordered) / 2
	if len(ordered)%2 == 1 {
		return ordered[middle]
	}
	return (ordered[middle-1] + ordered[middle]) / 2
}

// BridgeScore rates how well a pair works as a bridge between two ideas: close
// enough to share context, far enough that connecting them says something new.
//
// The peak sits at the midpoint between the typical score and the strongest
// score observed, so the shape follows the backend's own distribution instead of
// assuming cosine 0.35 is the interesting band. With the offline algorithm that
// lands near the old constant; with a sentence embedding model, where nearly
// every pair scores above 0.5, it keeps rating pairs instead of collapsing to
// zero for everything genuinely related.
func BridgeScore(similarity, typical, highest float64) float64 {
	peak := typical + (highest-typical)/2
	spread := math.Max(highest-typical, 1e-6)
	return math.Max(0, 1-math.Abs(similarity-peak)/spread)
}
