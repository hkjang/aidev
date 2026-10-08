package apitime

import (
	"testing"
	"time"
)

// The console was quietly wrong in the SQLite fallback: a naive timestamp has no
// zone, so a browser reads stored UTC as local time and every audit entry moved
// by the viewer's offset.
func TestNormalisesEveryShapeADriverCanReturn(t *testing.T) {
	want := "2026-07-29T19:17:12Z"
	cases := []struct {
		name  string
		value any
		want  any
	}{
		{"postgres time.Time", time.Date(2026, 7, 29, 19, 17, 12, 0, time.UTC), want},
		{
			"postgres non-UTC zone",
			time.Date(2026, 7, 30, 4, 17, 12, 0, time.FixedZone("KST", 9*3600)),
			want,
		},
		{"sqlite CURRENT_TIMESTAMP text", "2026-07-29 19:17:12", want},
		{"sqlite bytes", []byte("2026-07-29 19:17:12"), want},
		{"go String() layout", "2026-07-29 19:17:12 +0000 UTC", want},
		{
			"go String() with a monotonic reading",
			"2026-07-29 19:17:12 +0000 UTC m=+0.014901234",
			want,
		},
		{"already RFC3339", "2026-07-29T19:17:12Z", want},
		{"null stays null", nil, nil},
		{"empty text is null", "", nil},
		{"unknown text is preserved", "sometime yesterday", "sometime yesterday"},
	}
	for _, testCase := range cases {
		if got := Normalise(testCase.value); got != testCase.want {
			t.Errorf("%s: Normalise(%v) = %v, want %v",
				testCase.name, testCase.value, got, testCase.want)
		}
	}
}

// Sub-second precision must survive: a rotation and the audit entry describing
// it can land in the same second, and the ordering has to stay readable.
func TestKeepsSubSecondPrecision(t *testing.T) {
	got := Normalise("2026-07-29 19:17:12.973169037 +0000 UTC")
	if got != "2026-07-29T19:17:12.973169037Z" {
		t.Fatalf("Normalise() = %v", got)
	}
}
