package auth

import (
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestNewAPIKeyMaterialFailsClosed(t *testing.T) {
	entropyErr := errors.New("entropy unavailable")
	tests := []struct {
		name        string
		generate    func(int) (string, error)
		wantEntropy bool
	}{
		{
			name:        "secret generation",
			wantEntropy: true,
			generate: func(int) (string, error) {
				return "", entropyErr
			},
		},
		{
			name:        "prefix generation",
			wantEntropy: true,
			generate: func(bytes int) (string, error) {
				if bytes == 32 {
					return "full-secret", nil
				}
				return "", entropyErr
			},
		},
		{
			name: "short prefix",
			generate: func(bytes int) (string, error) {
				if bytes == 32 {
					return "full-secret", nil
				}
				return "short", nil
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			prefix, raw, err := newAPIKeyMaterial(test.generate)
			if err == nil || prefix != "" || raw != "" {
				t.Fatalf("failed entropy produced key material: prefix=%q raw=%q err=%v", prefix, raw, err)
			}
			if test.wantEntropy && !errors.Is(err, entropyErr) {
				t.Fatalf("entropy error was not propagated: %v", err)
			}
		})
	}
}

func TestNewAPIKeyMaterialFormatsGeneratedEntropy(t *testing.T) {
	prefix, raw, err := newAPIKeyMaterial(func(bytes int) (string, error) {
		if bytes == 32 {
			return "full-secret", nil
		}
		return "AbCdEfGh", nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if prefix != "abcdefgh" || raw != "umm_key_abcdefgh_full-secret" {
		t.Fatalf("unexpected key material: prefix=%q raw=%q", prefix, raw)
	}
}

func TestOriginOfTruncatesUserAgentOnUTF8Boundary(t *testing.T) {
	tests := []struct {
		name  string
		agent string
		want  string
	}{
		{name: "ascii", agent: strings.Repeat("a", 301), want: strings.Repeat("a", 300)},
		{name: "rune ends at limit", agent: strings.Repeat("a", 297) + "한" + "tail", want: strings.Repeat("a", 297) + "한"},
		{name: "rune crosses limit", agent: strings.Repeat("a", 299) + "한" + "tail", want: strings.Repeat("a", 299)},
		{name: "invalid bytes", agent: strings.Repeat("a", 299) + string([]byte{0xff, 0xfe}) + "b", want: strings.Repeat("a", 299) + "b"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest("GET", "/login", nil)
			request.Header.Set("User-Agent", test.agent)
			got := OriginOf(request).UserAgent
			if got != test.want {
				t.Fatalf("user agent = %q, want %q", got, test.want)
			}
			if len(got) > 300 || !utf8.ValidString(got) {
				t.Fatalf("user agent must be valid UTF-8 within 300 bytes: len=%d valid=%v", len(got), utf8.ValidString(got))
			}
		})
	}
}
