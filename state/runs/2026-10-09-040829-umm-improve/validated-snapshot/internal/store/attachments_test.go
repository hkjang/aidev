package store

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// A filename is a label and decides nothing, but it still crosses into a text
// column, and the one thing text will not accept is bytes that are not
// characters. Cutting a long name at a byte count ends inside a character
// whenever the name is not ASCII, and PostgreSQL refuses the whole row — so the
// picture, which was never the problem, is lost over its label.

func TestSafeFilenameCutsBetweenCharactersNotBytes(t *testing.T) {
	// Long enough to need cutting, and arranged so that the 120th byte lands
	// inside a character rather than between two.
	name := "2026 " + strings.Repeat("화이트보드", 9) + ".png"
	if len(name) <= 120 || utf8.RuneStart(name[120]) {
		t.Fatalf("this name no longer tests the boundary: %d bytes, rune start at 120 = %v", len(name), utf8.RuneStart(name[120]))
	}

	cleaned := safeFilename(name)
	if !utf8.ValidString(cleaned) {
		t.Fatalf("a cut name is not text any more: %q", cleaned)
	}
	if len(cleaned) > 120 {
		t.Fatalf("kept %d bytes, wanted at most 120", len(cleaned))
	}
	if !strings.HasPrefix(name, cleaned) {
		t.Fatalf("the label is no longer what the person called it: %q", cleaned)
	}
}

func TestSafeFilenameKeepsAShortNameWhole(t *testing.T) {
	if got := safeFilename("  화이트보드.png  "); got != "화이트보드.png" {
		t.Fatalf("got %q", got)
	}
}

// Some clients hand over the whole path they read the picture from. The name a
// person would say is the last piece of it; gluing the directories to the front
// makes a label nobody recognises, and this label is what the file is called
// again when it is downloaded.
func TestSafeFilenameKeepsOnlyTheLastPieceOfAPath(t *testing.T) {
	for given, want := range map[string]string{
		"C:\\사진\\회의.png": "회의.png",
		"a/b/c.png":      "c.png",
		"/tmp/화이트보드.png": "화이트보드.png",
		"회의.png":         "회의.png",
		"C:/사진/회의.png":   "회의.png",
	} {
		if got := safeFilename(given); got != want {
			t.Errorf("safeFilename(%q) = %q, want %q", given, got, want)
		}
	}
}

// A name that ends in a separator has an empty last piece. Losing the label
// entirely over that is worse than naming the picture after the piece before
// it, which is still something the person typed.
func TestSafeFilenameWalksBackPastATrailingSeparator(t *testing.T) {
	for given, want := range map[string]string{
		"사진/":      "사진",
		"a/b/":     "b",
		"C:\\사진\\": "사진",
	} {
		if got := safeFilename(given); got != want {
			t.Errorf("safeFilename(%q) = %q, want %q", given, got, want)
		}
	}
}

// The last piece is still cleaned: this string reaches a Content-Disposition
// header and a text column.
func TestSafeFilenameDropsControlsAndQuotesFromTheLastPiece(t *testing.T) {
	got := safeFilename("../etc/pass\"wd\x00.png")
	if got != "passwd.png" {
		t.Fatalf("got %q", got)
	}
	if strings.ContainsAny(got, "/\\") {
		t.Fatalf("a separator survived: %q", got)
	}
}
