package store

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/hkjang/umm/internal/intelligence"
)

func TestNoteSearchPatternsEscapesWildcards(t *testing.T) {
	patterns := noteSearchPatterns(`50% a_b c\d`)
	want := []string{`%50\%%`, `%a\_b%`, `%c\\d%`}
	if len(patterns) != len(want) {
		t.Fatalf("expected %d patterns, got %v", len(want), patterns)
	}
	for i, pattern := range patterns {
		if pattern != want[i] {
			t.Fatalf("pattern %d: want %q, got %q", i, want[i], pattern)
		}
	}
}

func TestNoteSearchPatternsBoundsTermCount(t *testing.T) {
	terms := strings.Repeat("term ", maxSearchTerms+5)
	if got := len(noteSearchPatterns(terms)); got != maxSearchTerms {
		t.Fatalf("expected the term count to be capped at %d, got %d", maxSearchTerms, got)
	}
	if got := len(noteSearchPatterns("   ")); got != 0 {
		t.Fatalf("a blank query has no patterns, got %d", got)
	}
}

func TestAllMatchIsAConjunctionOfIndexableProbes(t *testing.T) {
	b := &queryBuilder{}
	b.bind("already-bound")
	predicate := allMatch(noteTextExpression, []string{"%a%", "%b%"}, b)

	// Each term must be its own ILIKE against the indexed expression: that is
	// what lets PostgreSQL intersect trigram bitmaps instead of scanning.
	if got := strings.Count(predicate, "ILIKE"); got != 2 {
		t.Fatalf("expected one ILIKE per term, got %d in %q", got, predicate)
	}
	if !strings.Contains(predicate, " AND ") {
		t.Fatalf("terms must be ANDed together, got %q", predicate)
	}
	if !strings.Contains(predicate, "$2") || !strings.Contains(predicate, "$3") {
		t.Fatalf("placeholders must continue after already-bound arguments, got %q", predicate)
	}
	if len(b.args) != 3 {
		t.Fatalf("expected 3 bound arguments, got %d", len(b.args))
	}
}

func TestAllMatchWithNoTermsMatchesNothing(t *testing.T) {
	b := &queryBuilder{}
	if got := allMatch(noteTextExpression, nil, b); got != "false" {
		t.Fatalf("an empty term list must not match every row, got %q", got)
	}
	if len(b.args) != 0 {
		t.Fatalf("no placeholders should be bound, got %d", len(b.args))
	}
}

// The trigram index in migration 007 is built on this exact expression. If the
// two drift apart the planner silently falls back to a sequential scan.
func TestNoteTextExpressionMatchesTheIndexedExpression(t *testing.T) {
	if noteTextExpression != `(n.title || ' ' || n.content)` {
		t.Fatalf("noteTextExpression changed to %q without a matching migration", noteTextExpression)
	}
}

func TestEmbedQueryReportsTheFallbackAlgorithm(t *testing.T) {
	gateway := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "gateway unavailable", http.StatusBadGateway)
	}))
	defer gateway.Close()

	db := &Store{}
	db.embeddings.provider = intelligence.Provider{Remote: &intelligence.RemoteConfig{
		BaseURL: gateway.URL,
		Model:   "remote-model",
		Timeout: time.Second,
	}}
	db.embeddings.loadedAt = time.Now()

	vector, algorithm := db.EmbedQuery(context.Background(), "fallback query")
	if algorithm != intelligence.LocalAlgorithm {
		t.Fatalf("a failed remote query must select local stored vectors, got %q", algorithm)
	}
	if len(vector) != intelligence.Dimensions {
		t.Fatalf("expected a %d-dimensional local fallback, got %d", intelligence.Dimensions, len(vector))
	}
}
