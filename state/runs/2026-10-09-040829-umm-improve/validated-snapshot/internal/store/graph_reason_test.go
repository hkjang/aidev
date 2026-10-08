package store

import (
	"errors"
	"strings"
	"testing"
)

// The bound is on what someone may write, and Korean costs three bytes a
// character. Counting bytes would silently cut the allowance to a third of
// what the screen offers.
func TestParseEdgeReasonCountsCharactersNotBytes(t *testing.T) {
	full := strings.Repeat("가", MaxEdgeReason)
	got, err := ParseEdgeReason(full)
	if err != nil {
		t.Fatalf("%d Korean characters were refused: %v", MaxEdgeReason, err)
	}
	if got != full {
		t.Fatalf("the reason came back changed")
	}
	if _, err := ParseEdgeReason(full + "가"); !errors.Is(err, ErrEdgeReasonTooLong) {
		t.Fatalf("one character past the limit was accepted: %v", err)
	}
}

// Refused rather than truncated: storing half of somebody's sentence is worse
// than telling them it did not fit.
func TestParseEdgeReasonRefusesRatherThanTruncates(t *testing.T) {
	got, err := ParseEdgeReason(strings.Repeat("a", MaxEdgeReason+50))
	if !errors.Is(err, ErrEdgeReasonTooLong) {
		t.Fatalf("err=%v", err)
	}
	if got != "" {
		t.Fatalf("a refused reason came back as %q — something could store it", got)
	}
}

// Empty means the author did not feel the need, which is the normal case. A
// blank line is not a reason somebody wrote, so it collapses to the same thing
// rather than being stored as a reason made of spaces.
func TestParseEdgeReasonTreatsBlankAsNone(t *testing.T) {
	for _, blank := range []string{"", "   ", "\n", " \t\n "} {
		got, err := ParseEdgeReason(blank)
		if err != nil || got != "" {
			t.Fatalf("ParseEdgeReason(%q) = %q, %v", blank, got, err)
		}
	}
	// And a real sentence keeps its own shape, trimmed at the ends only.
	if got, _ := ParseEdgeReason("  지난 분기 수치가 이걸 뒷받침한다  "); got != "지난 분기 수치가 이걸 뒷받침한다" {
		t.Fatalf("got %q", got)
	}
}
