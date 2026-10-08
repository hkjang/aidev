package httpapi

import (
	"strings"
	"testing"
)

// The section arrives from the settings screen as loose JSON. Every mistake an
// administrator can make in it has to be named, because a target that looks
// saved and sends nowhere is found by the person trying to send.
func TestValidateHandoffSettings(t *testing.T) {
	target := func(name, origin string, formats ...any) map[string]any {
		return map[string]any{"name": name, "origin": origin, "formats": formats}
	}
	cases := []struct {
		name    string
		section map[string]any
		wantErr string
	}{
		{"empty list is the default", map[string]any{"targets": []any{}}, ""},
		{"one service", map[string]any{"targets": []any{target("Ptium", "https://ptium.intra", "markdown", "docx")}}, ""},
		{"missing list", map[string]any{}, "targets"},
		{"list is not a list", map[string]any{"targets": "https://ptium.intra"}, "배열"},
		{"no name", map[string]any{"targets": []any{target("  ", "https://ptium.intra", "markdown")}}, "이름"},
		{"origin with a path", map[string]any{"targets": []any{target("Ptium", "https://ptium.intra/handoff", "markdown")}}, "오리진"},
		{"origin without a scheme", map[string]any{"targets": []any{target("Ptium", "ptium.intra", "markdown")}}, "오리진"},
		{"origin with credentials", map[string]any{"targets": []any{target("Ptium", "https://a:b@ptium.intra", "markdown")}}, "오리진"},
		{"same origin twice", map[string]any{"targets": []any{target("Ptium", "https://ptium.intra", "markdown"), target("Ptium 2", "HTTPS://PTIUM.INTRA", "markdown")}}, "두 번"},
		{"no formats", map[string]any{"targets": []any{target("Ptium", "https://ptium.intra")}}, "형식을 하나 이상"},
		{"a word outside the standard", map[string]any{"targets": []any{target("Ptium", "https://ptium.intra", "pdf")}}, "표준에 없는"},
		{"a format that is not a string", map[string]any{"targets": []any{target("Ptium", "https://ptium.intra", 3)}}, "표준에 없는"},
	}
	for _, tc := range cases {
		err := validateHandoffSettings(tc.section)
		switch {
		case tc.wantErr == "" && err != nil:
			t.Errorf("%s: unexpected error %v", tc.name, err)
		case tc.wantErr != "" && err == nil:
			t.Errorf("%s: accepted, want an error mentioning %q", tc.name, tc.wantErr)
		case tc.wantErr != "" && !strings.Contains(err.Error(), tc.wantErr):
			t.Errorf("%s: error %q does not mention %q", tc.name, err, tc.wantErr)
		}
	}
	tooMany := make([]any, maxHandoffTargets+1)
	for index := range tooMany {
		tooMany[index] = target("svc", "https://svc"+strings.Repeat("x", index)+".intra", "markdown")
	}
	if err := validateHandoffSettings(map[string]any{"targets": tooMany}); err == nil {
		t.Error("a list beyond the bound was accepted")
	}
}

// A claim is a credential for five minutes; the access log lives longer.
func TestLoggedPathHidesTheClaim(t *testing.T) {
	cases := map[string]string{
		"/api/v1/handoff/claims/Qm9vazEyMzQ1Njc4OTA":       "/api/v1/handoff/claims/{claim}",
		"/api/v1/handoff/claims":                           "/api/v1/handoff/claims",
		"/api/v1/handoff/claims/":                          "/api/v1/handoff/claims/",
		"/api/v1/handoff/targets":                          "/api/v1/handoff/targets",
		"/api/v1/spaces/1/export/markdown":                 "/api/v1/spaces/1/export/markdown",
		"/handoff/claims/Qm9vazEyMzQ1Njc4OTA":              "/handoff/claims/Qm9vazEyMzQ1Njc4OTA",
		"/api/v1/handoff/claims/Qm9vazEyMzQ1Njc4OTA/extra": "/api/v1/handoff/claims/{claim}",
	}
	for given, want := range cases {
		if got := loggedPath(given); got != want {
			t.Errorf("loggedPath(%q) = %q, want %q", given, got, want)
		}
	}
}

// The name the document travels under is the space's, made safe the way every
// download name umm gives is made safe, and never empty.
func TestHandoffFilename(t *testing.T) {
	cases := map[string]string{
		"2026년 3분기 개편안":         "2026년 3분기 개편안.md",
		"  우리 팀\n회고  ":          "우리 팀 회고.md",
		`C:\a\b"c/d`:            "C:abcd.md",
		"":                      "umm.md",
		"\"\"":                  "umm.md",
		strings.Repeat("가", 50): strings.Repeat("가", 40) + ".md",
	}
	for given, want := range cases {
		if got := handoffFilename(given); got != want {
			t.Errorf("handoffFilename(%q) = %q, want %q", given, got, want)
		}
	}
}
