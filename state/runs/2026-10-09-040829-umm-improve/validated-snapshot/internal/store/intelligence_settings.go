package store

import (
	"context"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/hkjang/umm/internal/intelligence"
)

// Everything umm decides about similarity used to be a constant in the source:
// where the bar for "related" sits, when a workspace is grouped into topics, how
// good an embedding has to be before umm will guess, how many connections it may
// propose at once.
//
// Those numbers were chosen by measurement, and they are good defaults. But they
// were measured on one dataset in two languages, and an operator running a
// narrow corpus — one domain, one language, mostly short notes — may genuinely
// need a different line. Leaving them in the binary meant the only way to find
// out was to rebuild it.
//
// They are settings now. The defaults are exactly the measured values, so a
// deployment that changes nothing behaves exactly as before.

// IntelligenceSettings holds the thresholds behind related thoughts, clustering,
// search labelling and auto-link.
type IntelligenceSettings struct {
	// RelatedBand and ClusterBand are standard deviations above the mean of the
	// scores being judged, not raw cosine values — that is what makes them mean
	// the same thing whichever embedding backend produced the numbers.
	RelatedBand float64 `json:"related_band"`
	ClusterBand float64 `json:"cluster_band"`
	StrongBand  float64 `json:"strong_band"`

	// AutoLinkEnabled turns off proposals entirely, for a deployment that wants
	// the graph to contain only what people put in it.
	AutoLinkEnabled bool    `json:"autolink_enabled"`
	AutoLinkBand    float64 `json:"autolink_band"`
	AutoLinkMaxRun  int     `json:"autolink_max_per_run"`
	AutoLinkMinNote int     `json:"autolink_min_notes"`

	// SemanticAccuracyBar and SemanticPurityBar decide whether umm considers the
	// active embedding fit to judge meaning at all. Lowering them admits a
	// backend that lands just under the shipped line — an operator running one
	// language and one narrow domain may legitimately need that.
	//
	// They cannot admit a backend that ranks shared vocabulary above shared
	// meaning. That is gated on discrimination being positive, which is not a
	// setting: the shipped local algorithm measures -0.25 and no value here
	// changes it. The administrator screen says so next to the fields, and it is
	// repeated because the fields print how the local algorithm scores, which
	// otherwise reads as an invitation to set the bars just below it.
	SemanticAccuracyBar float64 `json:"semantic_accuracy_bar"`
	SemanticPurityBar   float64 `json:"semantic_purity_bar"`

	// DuplicateSimilarity is the one absolute cosine threshold umm keeps, and it
	// earns the exception by measurement. Every other bar is relative because
	// backends disagree wildly about what "close" means — but near-identical text
	// lands at the top of any sane embedding space, and two models agree on where:
	// bge-m3 scores near-duplicates 0.943-0.983 and paraphrase-multilingual
	// 0.954-0.986, while the next class down tops out at 0.681 and 0.581. The gap
	// is wide enough that one number works for both.
	DuplicateSimilarity float64 `json:"duplicate_similarity"`

	// QualityCacheMinutes is how long a measurement of the backend is reused.
	// Measuring costs one embedding request of sixty sentences.
	QualityCacheMinutes int `json:"quality_cache_minutes"`
}

// DefaultIntelligenceSettings are the measured values umm ships with.
func DefaultIntelligenceSettings() IntelligenceSettings {
	return IntelligenceSettings{
		RelatedBand:         float64(intelligence.BandRelated),
		ClusterBand:         float64(intelligence.BandCluster),
		StrongBand:          float64(intelligence.BandStrong),
		AutoLinkEnabled:     true,
		AutoLinkBand:        float64(intelligence.BandCluster),
		AutoLinkMaxRun:      12,
		AutoLinkMinNote:     6,
		SemanticAccuracyBar: intelligence.DefaultQualityBars().Accuracy,
		SemanticPurityBar:   intelligence.DefaultQualityBars().Purity,
		DuplicateSimilarity: 0.92,
		QualityCacheMinutes: 10,
	}
}

// clampFloat keeps a stored value inside the range where it still describes
// something. A band of 40 standard deviations admits nothing and a negative one
// admits everything; neither is a threshold, so both fall back to the default.
func clampFloat(value, low, high, fallback float64) float64 {
	if math.IsNaN(value) || value < low || value > high {
		return fallback
	}
	return value
}

func clampInt(value, low, high, fallback int) int {
	if value < low || value > high {
		return fallback
	}
	return value
}

// normalized replaces anything out of range with the shipped default, so a bad
// saved value degrades to known behaviour rather than to nothing working.
func (s IntelligenceSettings) normalized() IntelligenceSettings {
	d := DefaultIntelligenceSettings()
	// Bands are in standard deviations. Below zero admits more than half the
	// candidates, above four admits almost nothing in a normal distribution.
	s.RelatedBand = clampFloat(s.RelatedBand, 0, 4, d.RelatedBand)
	s.ClusterBand = clampFloat(s.ClusterBand, 0, 4, d.ClusterBand)
	s.StrongBand = clampFloat(s.StrongBand, 0, 4, d.StrongBand)
	s.AutoLinkBand = clampFloat(s.AutoLinkBand, 0, 4, d.AutoLinkBand)
	s.SemanticAccuracyBar = clampFloat(s.SemanticAccuracyBar, 0, 1, d.SemanticAccuracyBar)
	s.SemanticPurityBar = clampFloat(s.SemanticPurityBar, 0, 1, d.SemanticPurityBar)
	// Below 0.7 this stops meaning "the same thought twice" — the offline
	// algorithm alone reaches 0.889 on genuinely different sentences.
	s.DuplicateSimilarity = clampFloat(s.DuplicateSimilarity, 0.7, 1, d.DuplicateSimilarity)
	s.AutoLinkMaxRun = clampInt(s.AutoLinkMaxRun, 1, 100, d.AutoLinkMaxRun)
	s.AutoLinkMinNote = clampInt(s.AutoLinkMinNote, 3, 1000, d.AutoLinkMinNote)
	s.QualityCacheMinutes = clampInt(s.QualityCacheMinutes, 1, 1440, d.QualityCacheMinutes)
	return s
}

// QualityBars is what the measurement should judge against on this deployment.
func (s IntelligenceSettings) QualityBars() intelligence.QualityBars {
	return intelligence.QualityBars{Accuracy: s.SemanticAccuracyBar, Purity: s.SemanticPurityBar}
}

const intelligenceSettingsTTL = 30 * time.Second

type intelligenceCache struct {
	mu       sync.Mutex
	settings IntelligenceSettings
	loadedAt time.Time
}

// IntelligenceSettings resolves the active thresholds.
//
// Cached briefly: these are read on every canvas load and every search, and a
// database round trip per read would put the settings table on the hot path. The
// window is short enough that an administrator saving a change sees it take
// effect while they are still looking at the screen.
func (s *Store) IntelligenceSettings(ctx context.Context) IntelligenceSettings {
	s.intelligence.mu.Lock()
	defer s.intelligence.mu.Unlock()
	if !s.intelligence.loadedAt.IsZero() && time.Since(s.intelligence.loadedAt) < intelligenceSettingsTTL {
		return s.intelligence.settings
	}
	settings := DefaultIntelligenceSettings()
	var stored IntelligenceSettings
	if s.GetSetting(ctx, "intelligence", &stored) == nil {
		settings = stored.normalized()
	}
	s.intelligence.settings, s.intelligence.loadedAt = settings, time.Now()
	return settings
}

// InvalidateIntelligenceSettings drops the cache so a save takes effect at once.
func (s *Store) InvalidateIntelligenceSettings() {
	s.intelligence.mu.Lock()
	defer s.intelligence.mu.Unlock()
	s.intelligence.loadedAt = time.Time{}
}

// DecryptSetting turns a stored secret back into its plaintext, or returns an
// empty string when it cannot. Callers use it to reuse a saved credential
// without asking the administrator to retype it; an unreadable value yields
// nothing rather than the ciphertext, so a stale key is never sent to a gateway
// as if it were the real one.
func (s *Store) DecryptSetting(value string) string {
	if !strings.HasPrefix(value, "enc:") {
		return value
	}
	if s.Cipher == nil {
		return ""
	}
	plain, err := s.Cipher.Decrypt(strings.TrimPrefix(value, "enc:"))
	if err != nil {
		return ""
	}
	return plain
}
