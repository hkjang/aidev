package store

import (
	"errors"
	"strings"
	"testing"
)

func TestParseKindAcceptsTheVocabulary(t *testing.T) {
	for _, kind := range Kinds() {
		got, err := ParseKind(string(kind))
		if err != nil || got != kind {
			t.Errorf("%q is in Kinds() but ParseKind returned %q, %v", kind, got, err)
		}
	}
}

// Most notes are not marked, so an unset kind is an ordinary thought.
func TestParseKindDefaultsWhenUnset(t *testing.T) {
	for _, value := range []string{"", "  ", "\t"} {
		got, err := ParseKind(value)
		if err != nil || got != KindThought {
			t.Errorf("ParseKind(%q) = %q, %v", value, got, err)
		}
	}
}

// The hole this closes: kind reached the database straight from the request
// body, so anything at all could be stored in it — a five-thousand-character
// string was accepted and kept.
func TestParseKindRejectsRatherThanRewrites(t *testing.T) {
	for _, value := range []string{"totally-made-up", "decision", strings.Repeat("K", 5000), "생각"} {
		if _, err := ParseKind(value); !errors.Is(err, ErrUnknownKind) {
			t.Errorf("ParseKind(%.20q) should be rejected, got %v", value, err)
		}
	}
}

func TestParseKindIsCaseInsensitive(t *testing.T) {
	if got, err := ParseKind("  Question "); err != nil || got != KindQuestion {
		t.Errorf("ParseKind(\"  Question \") = %q, %v", got, err)
	}
}

// answers is what closes a question. supports and refines sit near it, and if
// one of them were treated as an answer a question would look settled by a note
// that only argued about it.
func TestAnswersIsItsOwnRelation(t *testing.T) {
	got, err := ParseRelation("answers")
	if err != nil || got != RelationAnswers {
		t.Fatalf("ParseRelation(\"answers\") = %q, %v", got, err)
	}
	for _, near := range []Relation{RelationSupports, RelationRefines, RelationRelated} {
		if near == RelationAnswers {
			t.Errorf("%q collapsed into the answer relation", near)
		}
	}
	found := false
	for _, relation := range Relations() {
		if relation == RelationAnswers {
			found = true
		}
	}
	if !found {
		t.Error("answers is accepted but not advertised, so nothing can offer it")
	}
}
