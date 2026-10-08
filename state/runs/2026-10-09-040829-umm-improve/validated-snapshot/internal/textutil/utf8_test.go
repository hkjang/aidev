package textutil

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestLimitUTF8Bytes(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		maxBytes int
		want     string
	}{
		{name: "unchanged", value: "한글", maxBytes: 6, want: "한글"},
		{name: "rune ends at limit", value: strings.Repeat("a", 2) + "한tail", maxBytes: 5, want: strings.Repeat("a", 2) + "한"},
		{name: "rune crosses limit", value: strings.Repeat("a", 4) + "한tail", maxBytes: 5, want: strings.Repeat("a", 4)},
		{name: "invalid bytes", value: "a" + string([]byte{0xff, 0xfe}) + "한", maxBytes: 4, want: "a한"},
		{name: "non-positive limit", value: "value", maxBytes: 0, want: ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := LimitUTF8Bytes(test.value, test.maxBytes)
			if got != test.want {
				t.Fatalf("LimitUTF8Bytes() = %q, want %q", got, test.want)
			}
			if (test.maxBytes >= 0 && len(got) > test.maxBytes) || !utf8.ValidString(got) {
				t.Fatalf("result must be valid UTF-8 within %d bytes: len=%d valid=%v", test.maxBytes, len(got), utf8.ValidString(got))
			}
		})
	}
}
