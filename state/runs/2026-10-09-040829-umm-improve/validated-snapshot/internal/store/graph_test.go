package store

import (
	"errors"
	"testing"
)

func TestParseRelationAcceptsTheVocabulary(t *testing.T) {
	for _, relation := range Relations() {
		got, err := ParseRelation(string(relation))
		if err != nil {
			t.Errorf("%q is in Relations() but ParseRelation rejected it: %v", relation, err)
		}
		if got != relation {
			t.Errorf("ParseRelation(%q) = %q", relation, got)
		}
	}
}

// A line dragged between two thoughts says "these belong together" and nothing
// more, so an unset relation is the generic one rather than an error.
func TestParseRelationDefaultsWhenUnset(t *testing.T) {
	for _, value := range []string{"", "   ", "\t\n"} {
		got, err := ParseRelation(value)
		if err != nil {
			t.Errorf("ParseRelation(%q) should default, got %v", value, err)
		}
		if got != RelationRelated {
			t.Errorf("ParseRelation(%q) = %q, want %q", value, got, RelationRelated)
		}
	}
}

// The values that used to encode provenance must no longer be accepted as
// meanings. If either comes back, a client can once again claim that Dream made
// a connection it drew itself.
func TestParseRelationRejectsProvenanceClaims(t *testing.T) {
	for _, value := range []string{"dreamed", "expanded", "auto", "manual", "agent"} {
		if _, err := ParseRelation(value); !errors.Is(err, ErrUnknownRelation) {
			t.Errorf("ParseRelation(%q) should be rejected, got %v", value, err)
		}
	}
}

// Silently rewriting an unrecognised relation to "related" would record a
// connection the caller did not describe and hide the mistake.
func TestParseRelationRejectsRatherThanRewrites(t *testing.T) {
	for _, value := range []string{"supprots", "관련", "'; DROP TABLE note_edges; --", string(make([]byte, 5000))} {
		if _, err := ParseRelation(value); !errors.Is(err, ErrUnknownRelation) {
			t.Errorf("ParseRelation(%q...) should be rejected, got %v", value[:min(20, len(value))], err)
		}
	}
}

func TestParseRelationIsCaseInsensitive(t *testing.T) {
	got, err := ParseRelation("  Supports ")
	if err != nil || got != RelationSupports {
		t.Errorf("ParseRelation(\"  Supports \") = %q, %v", got, err)
	}
}

func TestOnlyAutoOriginIsInferred(t *testing.T) {
	for _, origin := range []Origin{OriginManual, OriginAgent, OriginDream, OriginDevelopment, OriginImport} {
		if origin.Inferred() {
			t.Errorf("%q is an assertion by someone, not an inference", origin)
		}
	}
	if !OriginAuto.Inferred() {
		t.Error("an edge umm decided on its own must be marked as inferred")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
