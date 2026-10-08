package softwarecatalog

import (
	"regexp"
	"strings"
	"testing"
)

// The catalogue has its own wildcard matcher, separate from the one the
// classifier uses. Two implementations of one operation is how a pair drifts
// into disagreeing, and here they would disagree about which product a host is
// running - so this holds it to the same reference the classifier's is held to.
//
// It has to stay linear for the same reason: it runs against every observation
// on every ingest.
func TestCatalogueGlobAgreesWithAReference(t *testing.T) {
	reference := func(pattern, value string) bool {
		parts := strings.Split(pattern, "*")
		for index, part := range parts {
			parts[index] = regexp.QuoteMeta(part)
		}
		return regexp.MustCompile("^" + strings.Join(parts, ".*") + "$").MatchString(value)
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
			got := glob(pattern, value)
			want := reference(pattern, value)
			checked++
			if got != want {
				t.Errorf(
					"glob(%q, %q) = %v, a regular expression built from the same "+
						"pattern says %v",
					pattern, value, got, want,
				)
			}
		}
	}
	if checked == 0 {
		t.Fatal("no combinations were checked, so this proves nothing")
	}
}

func TestCatalogueGlobStaysLinearOnAPathologicalPattern(t *testing.T) {
	if glob(strings.Repeat("*a", 20)+"*b", strings.Repeat("a", 4096)) {
		t.Fatal("a value with no 'b' must not match a pattern ending in 'b'")
	}
}
