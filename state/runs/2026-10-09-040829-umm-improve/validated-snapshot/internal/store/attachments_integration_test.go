package store

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
)

// The rule that matters here is that nothing the upload says about itself is
// believed. Everything else follows the access rules a note already has.

func attachmentSpace(t *testing.T) (*Store, uuid.UUID, uuid.UUID, Note) {
	t.Helper()
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		t.Skip("POSTGRES_DSN is not configured")
	}
	db := isolatedStore(t, dsn)
	ctx := context.Background()
	userID, spaceID := uuid.New(), uuid.New()
	name := "attach_" + strings.ReplaceAll(userID.String(), "-", "")
	if _, err := db.Pool.Exec(ctx, `INSERT INTO users(id,username,display_name) VALUES($1,$2::citext,$2::text)`, userID, name); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Pool.Exec(ctx, `INSERT INTO spaces(id,owner_id,name) VALUES($1,$2,'첨부 공간')`, spaceID, userID); err != nil {
		t.Fatal(err)
	}
	note, err := db.CreateNote(ctx, userID, Note{SpaceID: spaceID, AuthorID: userID, Content: "화이트보드 사진을 붙일 생각"})
	if err != nil {
		t.Fatal(err)
	}
	return db, userID, spaceID, note
}

// realPNG is an actual encoded image, not bytes that merely start like one:
// the point is that the sniffing accepts real files.
func realPNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	img.Set(1, 1, color.RGBA{R: 200, G: 40, B: 40, A: 255})
	var out bytes.Buffer
	if err := png.Encode(&out, img); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}

func TestAttachmentRoundTripIntegration(t *testing.T) {
	db, userID, spaceID, note := attachmentSpace(t)
	ctx := context.Background()
	data := realPNG(t)

	saved, err := db.AttachToNote(ctx, userID, note.ID, "화이트보드.png", data)
	if err != nil {
		t.Fatal(err)
	}
	if saved.ContentType != "image/png" || saved.ByteSize != len(data) {
		t.Fatalf("stored as %+v", saved)
	}

	listed, err := db.NoteAttachments(ctx, userID, spaceID)
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 1 || listed[0].ID != saved.ID {
		t.Fatalf("the picture is not on the space: %+v", listed)
	}

	_, read, err := db.ReadAttachment(ctx, userID, saved.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(read, data) {
		t.Fatal("the bytes came back changed")
	}
}

// The whole point of sniffing. A file that claims to be a PNG and is not must
// be refused on what it is, not on what it said.
func TestAttachmentTypeComesFromTheBytesNotTheNameIntegration(t *testing.T) {
	db, userID, _, note := attachmentSpace(t)
	ctx := context.Background()

	// HTML with a script in it, named and offered as a picture. Served from
	// the app's own origin this is stored cross-site scripting.
	html := []byte(`<!doctype html><html><script>alert(document.cookie)</script></html>`)
	if _, err := db.AttachToNote(ctx, userID, note.ID, "innocent.png", html); !errors.Is(err, ErrAttachmentNotAnImage) {
		t.Fatalf("HTML was accepted as a picture: %v", err)
	}

	// SVG is the case a type allowlist usually forgets: it is a real image
	// format and it can carry script.
	svg := []byte(`<?xml version="1.0"?><svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`)
	if _, err := db.AttachToNote(ctx, userID, note.ID, "diagram.svg", svg); !errors.Is(err, ErrAttachmentNotAnImage) {
		t.Fatalf("SVG was accepted: %v", err)
	}

	// And a real picture with a misleading name is fine, because the name is
	// only ever a label.
	if _, err := db.AttachToNote(ctx, userID, note.ID, "notes.txt", realPNG(t)); err != nil {
		t.Fatalf("a real PNG named .txt was refused: %v", err)
	}
}

// The other three formats really are accepted, so the check above is a type
// test rather than "only PNG works".
func TestAttachmentAcceptsTheFormatsItClaimsToIntegration(t *testing.T) {
	db, userID, _, note := attachmentSpace(t)
	ctx := context.Background()
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))

	var jpegBytes, gifBytes bytes.Buffer
	if err := jpeg.Encode(&jpegBytes, img, nil); err != nil {
		t.Fatal(err)
	}
	if err := gif.Encode(&gifBytes, img, nil); err != nil {
		t.Fatal(err)
	}
	for name, data := range map[string][]byte{"image/jpeg": jpegBytes.Bytes(), "image/gif": gifBytes.Bytes()} {
		saved, err := db.AttachToNote(ctx, userID, note.ID, "사진", data)
		if err != nil {
			t.Fatalf("%s was refused: %v", name, err)
		}
		if saved.ContentType != name {
			t.Errorf("%s was stored as %q", name, saved.ContentType)
		}
	}
}

// Refused rather than truncated: half a picture is not a picture.
func TestAttachmentTooLargeIsRefusedIntegration(t *testing.T) {
	db, userID, _, note := attachmentSpace(t)
	oversized := make([]byte, MaxAttachmentBytes+1)
	copy(oversized, realPNG(t))
	if _, err := db.AttachToNote(context.Background(), userID, note.ID, "큰.png", oversized); !errors.Is(err, ErrAttachmentTooLarge) {
		t.Fatalf("err=%v", err)
	}
}

// A long name is cut, and the picture still arrives. The cut used to be a byte
// slice, which in Korean ends inside a character, and PostgreSQL refuses text
// that is not valid UTF-8 — so a photo that broke no rule came back as "그림을
// 저장하지 못했습니다" because of what it was called.
func TestAttachmentAcceptsALongNonASCIIFilenameIntegration(t *testing.T) {
	db, userID, _, note := attachmentSpace(t)
	name := "2026 " + strings.Repeat("화이트보드", 9) + ".png"
	saved, err := db.AttachToNote(context.Background(), userID, note.ID, name, realPNG(t))
	if err != nil {
		t.Fatalf("a picture was refused over its label: %v", err)
	}
	if saved.Filename == "" || !strings.HasPrefix(name, saved.Filename) {
		t.Fatalf("stored label %q is not the start of %q", saved.Filename, name)
	}
}

// Some clients hand over the whole path the picture was read from. What is
// stored — and what the download is named later — has to be the piece the
// person would say, not the directories glued in front of it.
func TestAttachmentStoresOnlyTheLastPieceOfAPathIntegration(t *testing.T) {
	db, userID, _, note := attachmentSpace(t)
	saved, err := db.AttachToNote(context.Background(), userID, note.ID, "C:\\사진\\회의.png", realPNG(t))
	if err != nil {
		t.Fatalf("a picture was refused over its label: %v", err)
	}
	if saved.Filename != "회의.png" {
		t.Fatalf("stored label %q, want %q", saved.Filename, "회의.png")
	}
}

// Nothing is not a picture, and saying "too large" about it sends someone to
// look for a limit they are nowhere near.
func TestAttachmentEmptyUploadIsNotAnImageIntegration(t *testing.T) {
	db, userID, _, note := attachmentSpace(t)
	if _, err := db.AttachToNote(context.Background(), userID, note.ID, "빈.png", nil); !errors.Is(err, ErrAttachmentNotAnImage) {
		t.Fatalf("err=%v", err)
	}
}

func TestAttachmentPerThoughtLimitIntegration(t *testing.T) {
	db, userID, _, note := attachmentSpace(t)
	ctx := context.Background()
	for i := 0; i < MaxAttachmentsPerNote; i++ {
		if _, err := db.AttachToNote(ctx, userID, note.ID, "사진", realPNG(t)); err != nil {
			t.Fatalf("picture %d: %v", i, err)
		}
	}
	if _, err := db.AttachToNote(ctx, userID, note.ID, "하나 더", realPNG(t)); !errors.Is(err, ErrTooManyAttachments) {
		t.Fatalf("err=%v", err)
	}
}

// A picture is part of what a thought says, so it follows the thought's rules.
//
// The stranger is a real account with no access, not an id nobody has. With a
// made-up id the upload is refused by the foreign key on uploaded_by, and this
// test passes whether or not the permission check exists — which is how it read
// the first time, and what the mutation run found.
func TestAttachmentFollowsTheThoughtsPermissionsIntegration(t *testing.T) {
	db, userID, _, note := attachmentSpace(t)
	ctx := context.Background()
	stranger := uuid.New()
	strangerName := "outsider_" + strings.ReplaceAll(stranger.String(), "-", "")
	if _, err := db.Pool.Exec(ctx, `INSERT INTO users(id,username,display_name) VALUES($1,$2::citext,$2::text)`, stranger, strangerName); err != nil {
		t.Fatal(err)
	}

	if _, err := db.AttachToNote(ctx, stranger, note.ID, "남의 사진", realPNG(t)); err == nil {
		t.Fatal("a stranger attached a picture to somebody else's thought")
	}
	saved, err := db.AttachToNote(ctx, userID, note.ID, "내 사진", realPNG(t))
	if err != nil {
		t.Fatalf("the owner was refused: %v", err)
	}
	if _, _, err := db.ReadAttachment(ctx, stranger, saved.ID); err == nil {
		t.Fatal("a stranger read a picture from a space they cannot see")
	}
	if err := db.DeleteAttachment(ctx, stranger, saved.ID); !errors.Is(err, ErrAttachmentNotFound) {
		t.Fatalf("a stranger deleting: %v", err)
	}
	if err := db.DeleteAttachment(ctx, userID, saved.ID); err != nil {
		t.Fatalf("the owner could not delete: %v", err)
	}
}

// Deleting a thought takes its pictures with it. A picture that outlived the
// thought it belonged to would be a copy nobody could find to remove.
func TestDeletingTheThoughtTakesItsPicturesIntegration(t *testing.T) {
	db, userID, spaceID, note := attachmentSpace(t)
	ctx := context.Background()
	saved, err := db.AttachToNote(ctx, userID, note.ID, "사진", realPNG(t))
	if err != nil {
		t.Fatal(err)
	}
	if err := db.DeleteNote(ctx, userID, note.ID); err != nil {
		t.Fatal(err)
	}
	// A soft-deleted thought hides its pictures from every read.
	listed, err := db.NoteAttachments(ctx, userID, spaceID)
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 0 {
		t.Fatalf("a deleted thought still lists pictures: %+v", listed)
	}
	if _, _, err := db.ReadAttachment(ctx, userID, saved.ID); err == nil {
		t.Fatal("a picture on a deleted thought is still readable")
	}
}
