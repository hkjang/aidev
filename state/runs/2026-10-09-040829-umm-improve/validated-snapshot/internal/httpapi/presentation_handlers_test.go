package httpapi

import "testing"

// A length cap arrives as text on a query string, and a preview is the
// cheapest thing in this feature. Refusing to draw one over a typo in an
// optional parameter would cost more than ignoring it.
func TestMaxSlidesParam(t *testing.T) {
	cases := []struct {
		raw  string
		want int
	}{
		{"", 0},
		{"20", 20},
		{" 20 ", 20},
		{"0", 0},
		{"-5", 0},     // negative is no cap, not a backwards one
		{"열두 장", 0},   // not a number: no cap rather than an error
		{"9999", 500}, // past any real talk, held at the ceiling
		{"500", 500},
	}
	for _, c := range cases {
		if got := maxSlidesParam(c.raw); got != c.want {
			t.Errorf("maxSlidesParam(%q) = %d, want %d", c.raw, got, c.want)
		}
	}
}
