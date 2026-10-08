package httpapi

import (
	"testing"

	"github.com/hkjang/umm/internal/store"
)

// What a saved picture is called: the person's label, with the ending umm read
// off the bytes rather than the one the label claimed.
func TestPictureName(t *testing.T) {
	tests := []struct {
		name          string
		filename      string
		contentType   string
		wantStem      string
		wantExtension string
	}{
		{
			name:          "the label already ends the right way",
			filename:      "화이트보드.png",
			contentType:   "image/png",
			wantStem:      "화이트보드",
			wantExtension: ".png",
		},
		{
			// Kept as written: correcting the case would be renaming a file
			// nobody asked to rename.
			name:          "an ending spelled another way is still that ending",
			filename:      "photo.JPEG",
			contentType:   "image/jpeg",
			wantStem:      "photo",
			wantExtension: ".JPEG",
		},
		{
			// The whole reason the format is read off the bytes. The words are
			// the person's and are kept; the claim about the format is not.
			name:          "a label that claims another format keeps its words",
			filename:      "diagram.svg",
			contentType:   "image/png",
			wantStem:      "diagram.svg",
			wantExtension: ".png",
		},
		{
			// A dotted name is not an ending. Cutting at the last dot would
			// turn a date into "2026.09".
			name:          "a date in the name is not an ending",
			filename:      "2026.09.09 회의",
			contentType:   "image/jpeg",
			wantStem:      "2026.09.09 회의",
			wantExtension: ".jpg",
		},
		{
			// The label is optional; the ending is what makes the saved file
			// open at all.
			name:          "no label at all",
			filename:      "",
			contentType:   "image/webp",
			wantStem:      "",
			wantExtension: ".webp",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			stem, extension := pictureName(store.Attachment{Filename: test.filename, ContentType: test.contentType})
			if stem != test.wantStem || extension != test.wantExtension {
				t.Errorf("pictureName = %q + %q, want %q + %q", stem, extension, test.wantStem, test.wantExtension)
			}
		})
	}
}
