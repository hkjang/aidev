package classify

import (
	"regexp"
	"strings"
	"testing"
)

// matchPattern is greedy and never backtracks, which is what keeps it linear:
// a rule is operator-authored and runs against every asset on every ingest, so
// a matcher that can blow up on "*a*a*a*a*b" would be a denial of service with
// no attacker needed.
//
// Linear is only worth having if it is also right. Taking the earliest
// occurrence of each segment is the standard argument - an earlier match leaves
// strictly more room for what follows - but the argument is not the code. This
// compares it against a regular expression built from the same pattern, over
// every combination of a small alphabet, which covers the overlapping cases the
// argument is easy to get wrong on.
func TestWildcardMatchingAgreesWithAReference(t *testing.T) {
	reference := func(pattern, value string) bool {
		parts := strings.Split(pattern, "*")
		for index, part := range parts {
			parts[index] = regexp.QuoteMeta(part)
		}
		compiled := regexp.MustCompile("^" + strings.Join(parts, ".*") + "$")
		return compiled.MatchString(value)
	}

	alphabet := []string{"", "a", "b", "ab", "ba", "aa", "abb", "bab", "abab"}
	patterns := []string{
		"a", "ab", "*", "**", "*a", "a*", "*a*", "a*b", "*a*b", "a*b*",
		"*a*b*", "*ab*ab", "*ab*b*c", "*abc*abc", "*a*a*a", "ab*ba",
		"*aa*", "aa*aa", "*b*a*b*",
	}

	checked := 0
	for _, pattern := range patterns {
		for _, value := range alphabet {
			// matchPattern is documented to reject an empty pattern outright,
			// and a regular expression would accept "" against "".
			if pattern == "" {
				continue
			}
			got := matchPattern(pattern, value)
			want := reference(pattern, value)
			checked++
			if got != want {
				t.Errorf(
					"matchPattern(%q, %q) = %v, a regular expression built from "+
						"the same pattern says %v",
					pattern, value, got, want,
				)
			}
		}
	}
	if checked == 0 {
		t.Fatal("no combinations were checked, so this proves nothing")
	}
}

// A rule that takes exponential time would stop ingest for every host, so the
// shape that does that to a backtracking matcher is worth pinning.
func TestPathologicalPatternStaysLinear(t *testing.T) {
	pattern := "*a*a*a*a*a*a*a*a*a*a*a*a*a*a*a*a*a*a*a*a*b"
	value := strings.Repeat("a", 4096)
	// A backtracking matcher does not return from this in any useful time.
	if matchPattern(pattern, value) {
		t.Fatal("a value with no 'b' must not match a pattern ending in 'b'")
	}
}
