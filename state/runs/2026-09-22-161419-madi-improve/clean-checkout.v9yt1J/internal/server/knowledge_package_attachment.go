package server

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

type packageAttachmentInput struct {
	Source    aiSource `json:"source"`
	Mandatory bool     `json:"mandatory"`
	Reason    string   `json:"reason"`
}

func packageAttachmentOpenAPISchema() map[string]any {
	uuid := map[string]any{"type": "string", "format": "uuid"}
	hash := map[string]any{"type": "string", "pattern": "^[a-f0-9]{64}$"}
	positive := map[string]any{"type": "integer", "minimum": 1}
	coordinate := map[string]any{"type": "integer", "minimum": 0, "maximum": 16384}
	source := map[string]any{"type": "object", "properties": map[string]any{
		"id": uuid, "version": map[string]any{"type": "integer", "minimum": 1, "maximum": 2147483647}, "attachment_id": uuid, "attachment_checksum": hash,
		"extraction_id": uuid, "extraction_revision": positive, "fragment_id": uuid, "fragment_hash": hash, "attachment_position": map[string]any{"type": "object"},
		"start_byte": coordinate, "end_byte": coordinate, "content_hash": hash,
	}, "required": []string{"id", "version", "attachment_id", "attachment_checksum", "extraction_id", "extraction_revision", "fragment_id", "fragment_hash", "attachment_position", "start_byte", "end_byte", "content_hash"}}
	return map[string]any{"type": "object", "properties": map[string]any{"source": source, "mandatory": map[string]any{"type": "boolean"}, "reason": map[string]any{"type": "string", "description": "UTF-8 최대1000바이트"}}, "required": []string{"source"}, "description": "현재 첨부 추출 조각 기준 UTF-8 최대8KiB. mandatory는 선택한 구간만 보존하며 첨부 전체를 뜻하지 않습니다."}
}

// A caller selects a bounded, versioned extraction span. Titles, URLs and
// citation identifiers come from the server, never from supplied source text.
func (s *Server) packageAttachmentTx(ctx context.Context, tx pgx.Tx, p *Principal, in packageAttachmentInput) (knowledgePackageQuote, error) {
	x := in.Source
	src := aiSource{ID: x.ID, Version: x.Version, AttachmentID: x.AttachmentID, AttachmentChecksum: x.AttachmentChecksum, ExtractionID: x.ExtractionID, ExtractionRevision: x.ExtractionRevision, FragmentID: x.FragmentID, FragmentHash: x.FragmentHash, AttachmentPosition: x.AttachmentPosition, StartByte: x.StartByte, EndByte: x.EndByte, ContentHash: x.ContentHash}
	value, e := s.attachmentCitationTx(ctx, tx, p, src, true)
	if e != nil {
		return knowledgePackageQuote{}, e
	}
	src.Title = value.Title
	src.Markdown = value.Text
	src.CitationID = digest(string(jsonValue(src)))
	src.URL = fmt.Sprintf("/app/attachments/%s?extraction=%s&fragment=%s", src.AttachmentID, src.ExtractionID, src.FragmentID)
	src.CitationURL = fmt.Sprintf("/attachment-extractions/%s/fragments/%s/citation?version=%d&revision=%d&start=%d&end=%d&hash=%s&checksum=%s&fragment_hash=%s", src.ExtractionID, src.FragmentID, src.Version, src.ExtractionRevision, src.StartByte, src.EndByte, src.ContentHash, src.AttachmentChecksum, src.FragmentHash)
	reason := strings.TrimSpace(in.Reason)
	if reason == "" {
		reason = "사용자가 원본 위치를 확인하고 이 업무의 참고 자료로 선택한 첨부 추출 구간"
	}
	return knowledgePackageQuote{evidenceQuote: evidenceQuote{src, value.Text}, Mandatory: in.Mandatory, Reason: reason, Representation: "extracted_text"}, nil
}
