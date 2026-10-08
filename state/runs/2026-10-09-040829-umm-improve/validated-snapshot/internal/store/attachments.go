package store

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hkjang/umm/internal/textutil"
)

/*
Pictures on a thought.

The part of a decision that is not a sentence — a whiteboard photo, a
screenshot of the thing being argued about — used to live outside the space
that held everything else about it.

Two rules do most of the work here, and both are about not believing the
upload. The type is decided by reading the bytes, never by the header or the
filename: a file that says it is a PNG and is not is the entire attack. And the
vocabulary is four raster formats, which excludes SVG deliberately — SVG is XML
and can carry script, and this is served from the same origin as the app.
*/

// MaxAttachmentBytes is the ceiling on one picture. Large enough for a phone
// photo of a whiteboard, small enough that a canvas cannot quietly become a
// file server. Enforced here and again by a CHECK constraint.
const MaxAttachmentBytes = 5 << 20

// MaxAttachmentsPerNote bounds one thought. A thought with a dozen pictures is
// not a thought any more.
const MaxAttachmentsPerNote = 8

var (
	// ErrAttachmentTooLarge is returned rather than truncating. Half a picture
	// is not a picture.
	ErrAttachmentTooLarge = errors.New("attachment too large")
	// ErrAttachmentNotAnImage is returned when the bytes are not one of the
	// four formats, whatever the upload claimed.
	ErrAttachmentNotAnImage = errors.New("attachment is not a supported image")
	// ErrTooManyAttachments is returned at the per-thought limit.
	ErrTooManyAttachments = errors.New("too many attachments on this thought")
	// ErrAttachmentNotFound covers both "no such picture" and "not yours",
	// because telling those apart tells a stranger what exists.
	ErrAttachmentNotFound = errors.New("attachment not found")
)

// Attachment is a picture, without its bytes.
type Attachment struct {
	ID          uuid.UUID  `json:"id"`
	NoteID      uuid.UUID  `json:"noteId"`
	ContentType string     `json:"contentType"`
	ByteSize    int        `json:"byteSize"`
	Filename    string     `json:"filename,omitempty"`
	UploadedBy  *uuid.UUID `json:"uploadedBy,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
}

// supportedImage decides what the bytes are.
//
// http.DetectContentType reads the leading bytes and recognises the formats
// browsers do. Anything it does not name as one of these four is refused,
// which is how SVG stays out without a special case: it sniffs as XML or
// plain text, never as an image.
func supportedImage(data []byte) (string, bool) {
	detected := http.DetectContentType(data)
	// DetectContentType can append parameters; the type alone is what matters.
	if semicolon := strings.Index(detected, ";"); semicolon >= 0 {
		detected = strings.TrimSpace(detected[:semicolon])
	}
	switch detected {
	case "image/png", "image/jpeg", "image/gif", "image/webp":
		return detected, true
	}
	return "", false
}

// AttachToNote stores a picture against a thought.
//
// Edit permission, the same as writing the thought: a picture is part of what
// the thought says.
func (s *Store) AttachToNote(ctx context.Context, userID, noteID uuid.UUID, filename string, data []byte) (Attachment, error) {
	if len(data) > MaxAttachmentBytes {
		return Attachment{}, ErrAttachmentTooLarge
	}
	// An empty upload is not a picture that is too large; it falls to the
	// sniffing below, which names nothing and says so.
	contentType, ok := supportedImage(data)
	if !ok {
		return Attachment{}, ErrAttachmentNotAnImage
	}

	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return Attachment{}, err
	}
	defer tx.Rollback(ctx)

	// The note, the space and the permission in one statement, so a picture
	// cannot be attached against access that was revoked while it uploaded.
	var spaceID uuid.UUID
	err = tx.QueryRow(ctx, `
		SELECT n.space_id FROM notes n
		JOIN spaces sp ON sp.id=n.space_id
		LEFT JOIN space_members m ON m.space_id=sp.id AND m.user_id=$2
		WHERE n.id=$1 AND n.deleted_at IS NULL
		  AND (sp.owner_id=$2 OR m.permission IN ('edit','manage'))
		FOR UPDATE OF n`, noteID, userID).Scan(&spaceID)
	if err != nil {
		return Attachment{}, err
	}

	var existing int
	if err = tx.QueryRow(ctx, `SELECT count(*) FROM note_attachments WHERE note_id=$1`, noteID).Scan(&existing); err != nil {
		return Attachment{}, err
	}
	if existing >= MaxAttachmentsPerNote {
		return Attachment{}, ErrTooManyAttachments
	}

	var attachment Attachment
	err = tx.QueryRow(ctx, `
		INSERT INTO note_attachments(note_id,space_id,uploaded_by,content_type,byte_size,filename,bytes)
		VALUES($1,$2,$3,$4,$5,$6,$7)
		RETURNING id,note_id,content_type,byte_size,filename,uploaded_by,created_at`,
		noteID, spaceID, userID, contentType, len(data), safeFilename(filename), data).
		Scan(&attachment.ID, &attachment.NoteID, &attachment.ContentType, &attachment.ByteSize,
			&attachment.Filename, &attachment.UploadedBy, &attachment.CreatedAt)
	if err != nil {
		return Attachment{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return Attachment{}, err
	}
	return attachment, nil
}

// safeFilename keeps a name only as a label. Directory separators and control
// characters are removed because this string reaches a Content-Disposition
// header and, if anybody ever writes it to disk, a path.
//
// The length is cut between characters, not at a byte. A name long enough to
// need cutting is usually a sentence, and in Korean a sentence is three bytes
// per character, so a plain byte slice ends mid-character — and PostgreSQL
// refuses text that is not valid UTF-8. That refusal loses the picture, which
// was never the problem, over the label, which is decoration.
// Some clients send the whole path a picture was read from, and the name a
// person would say is only its last piece — removing the separators alone would
// glue "C:\사진\회의.png" into "C:사진회의.png". Both separators are split on
// here rather than by filepath.Base, which on Linux does not know that "\" ever
// separated anything. An empty last piece is walked back rather than accepted:
// a name ending in a separator would otherwise leave no label at all, and the
// piece before it is still something the person typed.
func safeFilename(name string) string {
	pieces := strings.FieldsFunc(strings.TrimSpace(name), func(r rune) bool {
		return r == '/' || r == '\\'
	})
	for i := len(pieces) - 1; i >= 0; i-- {
		if cleaned := cleanFilenamePiece(pieces[i]); cleaned != "" {
			return cleaned
		}
	}
	return ""
}

// cleanFilenamePiece removes what must not reach a header or a text column. The
// separators are dropped again here — they cannot survive the split above, and
// that is the point: nothing downstream has to trust that they were.
func cleanFilenamePiece(piece string) string {
	cleaned := strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f || r == '/' || r == '\\' || r == '"' {
			return -1
		}
		return r
	}, strings.TrimSpace(piece))
	return textutil.LimitUTF8Bytes(cleaned, 120)
}

// NoteAttachments lists the pictures on the thoughts of a space, for anyone who
// may read it. Without bytes: a canvas draws them by fetching each one.
func (s *Store) NoteAttachments(ctx context.Context, userID, spaceID uuid.UUID) ([]Attachment, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT a.id,a.note_id,a.content_type,a.byte_size,a.filename,a.uploaded_by,a.created_at
		FROM note_attachments a
		JOIN notes n ON n.id=a.note_id AND n.deleted_at IS NULL
		JOIN spaces sp ON sp.id=a.space_id
		LEFT JOIN space_members m ON m.space_id=sp.id AND m.user_id=$2
		WHERE a.space_id=$1 AND (sp.owner_id=$2 OR m.user_id=$2)
		ORDER BY a.created_at`, spaceID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Attachment{}
	for rows.Next() {
		var a Attachment
		if err := rows.Scan(&a.ID, &a.NoteID, &a.ContentType, &a.ByteSize, &a.Filename,
			&a.UploadedBy, &a.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// ReadAttachment returns one picture's bytes to somebody who may read its
// space. The permission is in the same statement as the read.
func (s *Store) ReadAttachment(ctx context.Context, userID, attachmentID uuid.UUID) (Attachment, []byte, error) {
	var a Attachment
	var data []byte
	err := s.Pool.QueryRow(ctx, `
		SELECT a.id,a.note_id,a.content_type,a.byte_size,a.filename,a.uploaded_by,a.created_at,a.bytes
		FROM note_attachments a
		JOIN notes n ON n.id=a.note_id AND n.deleted_at IS NULL
		JOIN spaces sp ON sp.id=a.space_id
		LEFT JOIN space_members m ON m.space_id=sp.id AND m.user_id=$2
		WHERE a.id=$1 AND (sp.owner_id=$2 OR m.user_id=$2)`, attachmentID, userID).
		Scan(&a.ID, &a.NoteID, &a.ContentType, &a.ByteSize, &a.Filename, &a.UploadedBy, &a.CreatedAt, &data)
	if err != nil {
		return Attachment{}, nil, err
	}
	return a, data, nil
}

// DeleteAttachment removes a picture. Edit permission, like putting one on.
func (s *Store) DeleteAttachment(ctx context.Context, userID, attachmentID uuid.UUID) error {
	command, err := s.Pool.Exec(ctx, `
		DELETE FROM note_attachments a
		WHERE a.id=$1
		  AND EXISTS(
		    SELECT 1 FROM spaces sp
		    LEFT JOIN space_members m ON m.space_id=sp.id AND m.user_id=$2
		    WHERE sp.id=a.space_id AND (sp.owner_id=$2 OR m.permission IN ('edit','manage')))`,
		attachmentID, userID)
	if err != nil {
		return err
	}
	if command.RowsAffected() == 0 {
		return ErrAttachmentNotFound
	}
	return nil
}
