package store

import (
	"context"
	"sync"
	"time"

	"github.com/hkjang/umm/internal/intelligence"
)

// An operator can configure an embedding model, see no error, and still be
// running on the offline character n-gram algorithm — because the model name was
// wrong, because the gateway refused the request, or because it was never
// configured at all. Nothing in the product told them, and every feature that
// says "related" or "similar" quietly meant "shares vocabulary".
//
// This measures the configured backend on the labelled pairs in
// internal/intelligence/quality.go and hands the numbers to the administrator
// screen, so the person who can fix it is the person who can see it.

type embeddingQualityCache struct {
	mu       sync.Mutex
	report   intelligence.QualityReport
	provider intelligence.Provider
	measured time.Time
}

// MeasureEmbeddingQuality scores the configured embedding backend.
//
// It resolves the provider from settings directly rather than through
// EmbeddingProvider, which substitutes the local algorithm while the remote
// circuit breaker is open. An operator asking "what is my gateway doing" during
// an outage needs the attempt and its error, not a silent local answer.
//
// Results are cached per provider identity: the administrator screen loads this
// on open, and re-embedding 44 sentences on every visit would be a needless
// round trip to someone's inference server. Pass refresh to force a new run.
func (s *Store) MeasureEmbeddingQuality(ctx context.Context, refresh bool) (intelligence.QualityReport, error) {
	provider := s.configuredEmbeddingProvider(ctx)
	settings := s.IntelligenceSettings(ctx)
	ttl := time.Duration(settings.QualityCacheMinutes) * time.Minute

	s.embeddingQuality.mu.Lock()
	cached, cachedProvider, measured := s.embeddingQuality.report, s.embeddingQuality.provider, s.embeddingQuality.measured
	s.embeddingQuality.mu.Unlock()

	if !refresh && !measured.IsZero() &&
		time.Since(measured) < ttl &&
		sameEmbeddingProvider(cachedProvider, provider) {
		// Re-judge rather than return the stored verdict. The measurements do not
		// change with the thresholds, but the answer does, and an administrator
		// who just raised the bar must not be told the old one still holds.
		return cached.WithBars(settings.QualityBars()), nil
	}

	report, err := intelligence.MeasureQuality(ctx, provider, settings.QualityBars())
	if err != nil {
		return intelligence.QualityReport{}, err
	}
	// Report the backend the operator configured, even when the vectors fell back
	// to local, so the screen can say "configured but unreachable" rather than
	// presenting the fallback as if it were the chosen model.
	if provider.Remote != nil {
		report.Model = provider.Remote.Model
	}

	s.embeddingQuality.mu.Lock()
	s.embeddingQuality.report, s.embeddingQuality.provider, s.embeddingQuality.measured = report, provider, time.Now()
	s.embeddingQuality.mu.Unlock()
	return report, nil
}

// configuredEmbeddingProvider reads the gateway settings without the circuit
// breaker or the settings TTL in the way.
func (s *Store) configuredEmbeddingProvider(ctx context.Context) intelligence.Provider {
	var settings embeddingSettings
	if s.GetSetting(ctx, "ai_gateway", &settings) != nil {
		return intelligence.Provider{}
	}
	return s.embeddingProviderFromSettings(settings)
}
