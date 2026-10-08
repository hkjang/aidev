package store

import (
	"math"
	"testing"
)

// A stored value outside the range where it means anything must fall back to the
// shipped default rather than be used. A band of 40 standard deviations admits
// nothing and a negative one admits everything; neither is a threshold.
func TestIntelligenceSettingsNormalizeOutOfRangeValues(t *testing.T) {
	d := DefaultIntelligenceSettings()
	broken := IntelligenceSettings{
		RelatedBand:         40,
		ClusterBand:         -3,
		StrongBand:          math.NaN(),
		AutoLinkBand:        99,
		SemanticAccuracyBar: 1.5,
		SemanticPurityBar:   -0.2,
		AutoLinkMaxRun:      0,
		AutoLinkMinNote:     1,
		QualityCacheMinutes: 100000,
		DuplicateSimilarity: 0.3,
	}.normalized()

	for _, check := range []struct {
		name      string
		got, want float64
	}{
		{"related band", broken.RelatedBand, d.RelatedBand},
		{"cluster band", broken.ClusterBand, d.ClusterBand},
		{"strong band", broken.StrongBand, d.StrongBand},
		{"auto-link band", broken.AutoLinkBand, d.AutoLinkBand},
		{"accuracy bar", broken.SemanticAccuracyBar, d.SemanticAccuracyBar},
		{"purity bar", broken.SemanticPurityBar, d.SemanticPurityBar},
		{"duplicate similarity", broken.DuplicateSimilarity, d.DuplicateSimilarity},
	} {
		if check.got != check.want {
			t.Errorf("%s: got %v, want the default %v", check.name, check.got, check.want)
		}
	}
	if broken.AutoLinkMaxRun != d.AutoLinkMaxRun || broken.AutoLinkMinNote != d.AutoLinkMinNote {
		t.Errorf("auto-link counts were not normalized: %+v", broken)
	}
	if broken.QualityCacheMinutes != d.QualityCacheMinutes {
		t.Errorf("cache window was not normalized: %d", broken.QualityCacheMinutes)
	}
}

// The defaults are the values umm was measured against. If one drifts, every
// threshold in the product moves with it and the release notes stop describing
// the behaviour, so they are pinned here rather than only in a migration.
func TestDefaultsMatchTheMeasuredValues(t *testing.T) {
	d := DefaultIntelligenceSettings()
	for _, check := range []struct {
		name      string
		got, want float64
	}{
		{"related band", d.RelatedBand, 0.6},
		{"cluster band", d.ClusterBand, 1.1},
		{"strong band", d.StrongBand, 0.9},
		{"auto-link band", d.AutoLinkBand, 1.1},
		{"accuracy bar", d.SemanticAccuracyBar, 0.65},
		{"purity bar", d.SemanticPurityBar, 0.6},
		// Measured: near-duplicates land at 0.943+ on bge-m3 and 0.954+ on
		// paraphrase-multilingual, while the next class down tops out at 0.681.
		{"duplicate similarity", d.DuplicateSimilarity, 0.92},
	} {
		if check.got != check.want {
			t.Errorf("%s default is %v, but umm was measured and documented at %v", check.name, check.got, check.want)
		}
	}
	if !d.AutoLinkEnabled {
		t.Error("auto-link is off by default; the gate is what makes it safe, not silence")
	}
}

// A value already inside the range must be kept exactly. Normalisation exists to
// catch nonsense, not to quietly overrule an administrator.
func TestIntelligenceSettingsKeepValidValues(t *testing.T) {
	chosen := IntelligenceSettings{
		RelatedBand: 0.9, ClusterBand: 1.6, StrongBand: 1.2, AutoLinkBand: 2.0,
		SemanticAccuracyBar: 0.8, SemanticPurityBar: 0.75, DuplicateSimilarity: 0.95,
		AutoLinkMaxRun: 5, AutoLinkMinNote: 20, QualityCacheMinutes: 60,
		AutoLinkEnabled: false,
	}
	got := chosen.normalized()
	if got != chosen {
		t.Errorf("a valid configuration was altered\n  got:  %+v\n  want: %+v", got, chosen)
	}
}
