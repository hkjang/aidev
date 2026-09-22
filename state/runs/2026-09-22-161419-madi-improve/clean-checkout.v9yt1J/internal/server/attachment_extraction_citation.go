package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
)

type attachmentCitationCurrent struct {
	Text    string `json:"text"`
	Title   string `json:"current_title"`
	Version int    `json:"current_version"`
	Fresh   bool   `json:"fresh"`
}

// attachmentCitationTx validates a current generation or checks freshness of an
// archived quote. StartByte/EndByte always address UTF-8 bytes of FragmentID,
// not the parent Markdown or the PDF file. Archived text lives only in the
// caller's authenticated encrypted evidence; this function never manufactures
// an old source from a newer extraction.
func (s *Server) attachmentCitationTx(ctx context.Context, tx pgx.Tx, p *Principal, src aiSource, strict bool) (attachmentCitationCurrent, error) {
	var out attachmentCitationCurrent
	if !hasIntegrationScope(p, "document:read") || !validID(src.AttachmentID) || !validID(src.ExtractionID) || !validID(src.FragmentID) || !validID(src.ID) || src.StartByte < 0 || src.EndByte <= src.StartByte || src.EndByte-src.StartByte > 8192 || len(src.ContentHash) != 64 || len(src.AttachmentChecksum) != 64 || len(src.FragmentHash) != 64 || src.ExtractionRevision < 1 {
		return out, errors.New("첨부 인용 범위를 확인하세요")
	}
	var checksum string
	if e := tx.QueryRow(ctx, `SELECT a.name,d.version,a.checksum_sha256 FROM attachments a JOIN documents d ON d.id=a.document_id WHERE a.id=$1 AND d.id=$2 AND d.deleted_at IS NULL AND madi_document_allowed($3,d.id,false) AND ($4='' OR d.workspace_id=NULLIF($4,'')::uuid) FOR SHARE OF a,d`, src.AttachmentID, src.ID, p.ID, p.WorkspaceID).Scan(&out.Title, &out.Version, &checksum); e != nil {
		return out, e
	}
	if checksum != src.AttachmentChecksum {
		if strict {
			return out, errExtractionChanged
		}
		return out, nil
	}
	var text, hash string
	var position []byte
	e := tx.QueryRow(ctx, `SELECT f.text,f.content_hash,f.position FROM attachment_extraction_fragments f JOIN attachment_extractions e ON e.id=f.extraction_id JOIN attachment_extraction_heads h ON h.extraction_id=e.id JOIN attachment_extraction_settings p ON p.id=1 AND (p.data->>'enabled')::boolean AND p.revision=e.policy_revision WHERE f.id=$1 AND e.id=$2 AND e.attachment_id=$3 AND e.document_id=$4 AND e.status='ready' AND e.revision=$5 AND e.checksum=$6 FOR SHARE OF f,e,h,p`, src.FragmentID, src.ExtractionID, src.AttachmentID, src.ID, src.ExtractionRevision, src.AttachmentChecksum).Scan(&text, &hash, &position)
	if e != nil {
		if strict {
			return out, errExtractionChanged
		}
		return out, nil
	}
	var actualPosition map[string]any
	if json.Unmarshal(position, &actualPosition) != nil || string(jsonValue(actualPosition)) != string(jsonValue(src.AttachmentPosition)) || hash != src.FragmentHash || digest(text) != hash || src.EndByte > len(text) || !utf8.ValidString(text[src.StartByte:src.EndByte]) {
		if strict {
			return out, errExtractionChanged
		}
		return out, nil
	}
	out.Text = text[src.StartByte:src.EndByte]
	if digest(out.Text) != src.ContentHash {
		out.Text = ""
		if strict {
			return out, errExtractionChanged
		}
		return out, nil
	}
	out.Fresh = out.Version == src.Version
	if strict && !out.Fresh {
		return out, errExtractionChanged
	}
	return out, nil
}

func attachmentAISource(v extractionRun, version int, f attachmentFragment, start, end int) (aiSource, error) {
	if start < 0 || end <= start || end > len(f.Text) || end-start > 8192 || !utf8.ValidString(f.Text[start:end]) {
		return aiSource{}, errors.New("첨부 인용은 유효한 UTF-8 구간 8KiB 이하여야 합니다")
	}
	var position map[string]any
	if json.Unmarshal(jsonValue(f.Position), &position) != nil {
		return aiSource{}, errExtractionChanged
	}
	src := aiSource{ID: v.DocumentID, Version: version, Markdown: f.Text[start:end], StartByte: start, EndByte: end, ContentHash: digest(f.Text[start:end]), AttachmentID: v.AttachmentID, AttachmentChecksum: v.Checksum, ExtractionID: v.ID, ExtractionRevision: v.Revision, FragmentID: f.ID, FragmentHash: f.ContentHash, AttachmentPosition: position}
	src.CitationID = digest(string(jsonValue(src)))
	src.URL = fmt.Sprintf("/app/attachments/%s?extraction=%s&fragment=%s", v.AttachmentID, v.ID, f.ID)
	src.CitationURL = fmt.Sprintf("/attachment-extractions/%s/fragments/%s/citation?version=%d&revision=%d&start=%d&end=%d&hash=%s&checksum=%s&fragment_hash=%s", v.ID, f.ID, version, v.Revision, start, end, src.ContentHash, src.AttachmentChecksum, src.FragmentHash)
	return src, nil
}

func (s *Server) validateAttachmentAISources(ctx context.Context, p *Principal, sources []aiSource) error {
	var has bool
	for _, src := range sources {
		if src.AttachmentID != "" {
			has = true
			break
		}
	}
	if !has {
		return nil
	}
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(ctx)
	for _, src := range sources {
		if src.AttachmentID != "" {
			if _, e = s.attachmentCitationTx(ctx, tx, p, src, true); e != nil {
				return e
			}
		}
	}
	return tx.Commit(ctx)
}

func (s *Server) getAttachmentCitation(w http.ResponseWriter, r *http.Request) {
	v, version, e := s.activeExtraction(r.Context(), current(r), r.PathValue("id"))
	if e != nil {
		apiError(w, 410, "첨부 원본·추출 정책을 다시 확인하세요")
		return
	}
	f, e := scanAttachmentFragment(s.DB.QueryRow(r.Context(), `SELECT id::text,extraction_id::text,ordinal,text,content_hash,position FROM attachment_extraction_fragments WHERE extraction_id=$1 AND id=$2`, v.ID, r.PathValue("fragment")))
	if e != nil {
		apiError(w, 404, "첨부 본문 구간을 찾을 수 없습니다")
		return
	}
	q := r.URL.Query()
	start, _ := strconv.Atoi(q.Get("start"))
	end, _ := strconv.Atoi(q.Get("end"))
	expectedVersion, _ := strconv.Atoi(q.Get("version"))
	revision, _ := strconv.ParseInt(q.Get("revision"), 10, 64)
	src, e := attachmentAISource(v, version, f, start, end)
	if e != nil || expectedVersion != version || revision != v.Revision || src.ContentHash != q.Get("hash") || src.AttachmentChecksum != q.Get("checksum") || src.FragmentHash != q.Get("fragment_hash") {
		apiError(w, 409, "첨부 인용 원본 또는 위치가 변경되었습니다")
		return
	}
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	value, e := s.attachmentCitationTx(r.Context(), tx, current(r), src, true)
	if e != nil {
		apiError(w, 409, "첨부 인용 원본 또는 권한이 변경되었습니다")
		return
	}
	src.Title = value.Title
	jsonResponse(w, 200, map[string]any{"source": src, "text": value.Text, "markdown": value.Text, "position": src.AttachmentPosition, "current": true})
}
