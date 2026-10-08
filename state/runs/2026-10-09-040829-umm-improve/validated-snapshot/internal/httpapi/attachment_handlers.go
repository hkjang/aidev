package httpapi

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/hkjang/umm/internal/store"
	"github.com/jackc/pgx/v5"
)

/*
Pictures on a thought, over HTTP.

The delicate part is not storing them, it is handing them back. These bytes
came from a person and are served from the same origin as the application, so
a file that the browser decides to render as a document rather than a picture
is stored cross-site scripting. Three things stop that, and none of them is the
upload's word about itself:

  - the type is decided by reading the bytes when they are stored, and only
    four raster formats are kept at all — SVG is XML and can carry script;
  - the response repeats nosniff and carries a Content-Security-Policy of its
    own that permits nothing, so even if a browser were talked into treating
    the body as a document, that document could do nothing;
  - it is served as an attachment-or-inline image and never as a page.
*/

// uploadAttachment puts a picture on a thought.
func (s *Server) uploadAttachment(w http.ResponseWriter, r *http.Request) {
	// notes:write: a picture is part of what the thought says.
	if !requireScope(w, r, "notes:write") {
		return
	}
	noteID, ok := parseID(w, r, "noteID")
	if !ok {
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, 400, "붙일 그림을 찾지 못했습니다.")
		return
	}
	defer file.Close()

	// Read at most one byte past the limit, so "too large" is known without
	// holding an arbitrary amount of somebody's upload in memory.
	data, err := io.ReadAll(io.LimitReader(file, store.MaxAttachmentBytes+1))
	if err != nil {
		writeError(w, 400, "그림을 읽지 못했습니다.")
		return
	}

	filename := ""
	if header != nil {
		filename = header.Filename
	}
	p := principal(r)
	attachment, err := s.Store.AttachToNote(r.Context(), p.User.ID, noteID, filename, data)
	if err != nil {
		writeAttachmentError(w, r, err)
		return
	}
	writeJSON(w, 201, attachment)
}

// writeAttachmentError says which rule was met, and what the limit is, because
// "그림을 붙이지 못했습니다" leaves someone with a 6MB photo nowhere to go.
func writeAttachmentError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, store.ErrAttachmentTooLarge):
		writeProblem(w, r, http.StatusRequestEntityTooLarge, "attachment-too-large", "그림이 너무 큽니다",
			"한 장에 "+strconv.Itoa(store.MaxAttachmentBytes/(1<<20))+"MB까지 붙일 수 있습니다.",
			map[string]any{"maxBytes": store.MaxAttachmentBytes})
	case errors.Is(err, store.ErrAttachmentNotAnImage):
		writeProblem(w, r, http.StatusUnsupportedMediaType, "attachment-not-an-image", "그림 파일이 아닙니다",
			"PNG · JPEG · GIF · WebP만 붙일 수 있습니다. 파일 이름이 아니라 내용으로 판단합니다.",
			map[string]any{"allowed": []string{"image/png", "image/jpeg", "image/gif", "image/webp"}})
	case errors.Is(err, store.ErrTooManyAttachments):
		writeProblem(w, r, http.StatusConflict, "too-many-attachments", "그림이 너무 많습니다",
			"생각 하나에 "+strconv.Itoa(store.MaxAttachmentsPerNote)+"장까지 붙일 수 있습니다.",
			map[string]any{"maxPerNote": store.MaxAttachmentsPerNote})
	case errors.Is(err, pgx.ErrNoRows):
		writeError(w, 404, "생각을 찾을 수 없습니다.")
	default:
		slog.Warn("attachment upload failed", "error", err)
		writeError(w, 500, "그림을 저장하지 못했습니다.")
	}
}

// serveAttachment hands the bytes back.
func (s *Server) serveAttachment(w http.ResponseWriter, r *http.Request) {
	if !requireScope(w, r, "notes:read") {
		return
	}
	attachmentID, ok := parseID(w, r, "attachmentID")
	if !ok {
		return
	}
	p := principal(r)
	attachment, data, err := s.Store.ReadAttachment(r.Context(), p.User.ID, attachmentID)
	if err != nil {
		// Not found and not yours answer the same way: telling those apart
		// tells a stranger what exists.
		writeError(w, 404, "그림을 찾을 수 없습니다.")
		return
	}

	// The stored type, which was decided by reading the bytes rather than by
	// believing the upload. Never the client's Accept, never the filename.
	w.Header().Set("Content-Type", attachment.ContentType)
	// Repeated here although the global middleware sets it on every response:
	// this is the one route where the body is somebody's file, so the header
	// belongs beside the thing it protects rather than only in a middleware a
	// future route change could sit outside.
	w.Header().Set("X-Content-Type-Options", "nosniff")
	// A policy of its own, permitting nothing. The page policy allows scripts
	// with a nonce; this body must never be treated as a page at all.
	w.Header().Set("Content-Security-Policy", "default-src 'none'; sandbox")
	// Still inline — the canvas draws this with an <img> — but carrying the
	// label the picture arrived with, because the address it is fetched from is
	// a uuid and would otherwise be the only name a saved copy ever gets.
	w.Header().Set("Content-Disposition", inlineDisposition(pictureName(attachment)))
	// Rows are never rewritten — a changed picture is a new row with a new id —
	// so this is safe to keep, and private because it is somebody's space.
	w.Header().Set("Cache-Control", "private, max-age=31536000, immutable")
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

// pictureEndings is what each format umm keeps is called on disk, and the
// endings a label is already allowed to carry for it.
var pictureEndings = map[string][]string{
	"image/png":  {".png"},
	"image/jpeg": {".jpg", ".jpeg"},
	"image/gif":  {".gif"},
	"image/webp": {".webp"},
}

// pictureName splits a stored picture into the name a download should carry.
//
// The label is a person's words and is kept: it is the only readable thing
// about a file whose address is a uuid. The ending is not the label's to
// decide. umm named the format by reading the bytes, exactly as it refuses to
// believe the upload anywhere else here, so a label claiming a different one —
// `diagram.svg` over PNG bytes — keeps its words and gains the true ending
// rather than being trusted or being cut off.
//
// A label that already ends the right way keeps that ending as it wrote it, so
// `사진.JPEG` does not come back as `사진.JPEG.jpg`.
func pictureName(attachment store.Attachment) (stem, extension string) {
	endings := pictureEndings[attachment.ContentType]
	label := strings.TrimSpace(attachment.Filename)
	if len(endings) == 0 {
		return label, ""
	}
	lowered := strings.ToLower(label)
	for _, ending := range endings {
		if strings.HasSuffix(lowered, ending) {
			return label[:len(label)-len(ending)], label[len(label)-len(ending):]
		}
	}
	return label, endings[0]
}

// listSpaceAttachments returns what is on a space's thoughts, without bytes.
func (s *Server) listSpaceAttachments(w http.ResponseWriter, r *http.Request) {
	if !requireScope(w, r, "notes:read") {
		return
	}
	spaceID, ok := parseID(w, r, "spaceID")
	if !ok {
		return
	}
	p := principal(r)
	attachments, err := s.Store.NoteAttachments(r.Context(), p.User.ID, spaceID)
	if err != nil {
		writeError(w, 500, "그림 목록을 불러오지 못했습니다.")
		return
	}
	writeJSON(w, 200, map[string]any{"attachments": attachments})
}

// deleteAttachment removes one.
func (s *Server) deleteAttachment(w http.ResponseWriter, r *http.Request) {
	if !requireScope(w, r, "notes:write") {
		return
	}
	attachmentID, ok := parseID(w, r, "attachmentID")
	if !ok {
		return
	}
	p := principal(r)
	if err := s.Store.DeleteAttachment(r.Context(), p.User.ID, attachmentID); err != nil {
		if errors.Is(err, store.ErrAttachmentNotFound) {
			writeError(w, 404, "그림을 찾을 수 없습니다.")
			return
		}
		slog.Warn("attachment delete failed", "attachment_id", attachmentID, "error", err)
		writeError(w, 500, "그림을 지우지 못했습니다.")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
