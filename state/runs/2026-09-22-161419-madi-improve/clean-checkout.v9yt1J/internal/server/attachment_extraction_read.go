package server

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/hkjang/madi/internal/extract"
	"github.com/jackc/pgx/v5"
)

type attachmentFragment struct {
	ID           string           `json:"id"`
	ExtractionID string           `json:"extraction_id"`
	Ordinal      int              `json:"ordinal"`
	Text         string           `json:"text"`
	ContentHash  string           `json:"content_hash"`
	Position     extract.Position `json:"position"`
}

func scanAttachmentFragment(row pgx.Row) (attachmentFragment, error) {
	var f attachmentFragment
	var raw []byte
	e := row.Scan(&f.ID, &f.ExtractionID, &f.Ordinal, &f.Text, &f.ContentHash, &raw)
	if e == nil {
		e = json.Unmarshal(raw, &f.Position)
	}
	return f, e
}

func (s *Server) activeExtraction(ctx context.Context, p *Principal, extractionID string) (extractionRun, int, error) {
	v, e := s.extractionRead(ctx, p, extractionID)
	if e != nil {
		return v, 0, e
	}
	var version int
	e = s.DB.QueryRow(ctx, `SELECT d.version FROM attachment_extraction_heads h JOIN attachment_extractions e ON e.id=h.extraction_id JOIN attachments a ON a.id=h.attachment_id JOIN documents d ON d.id=a.document_id CROSS JOIN attachment_extraction_settings policy WHERE e.id=$1 AND a.document_id=e.document_id AND a.checksum_sha256=e.checksum AND d.deleted_at IS NULL AND madi_document_allowed($2,d.id,false) AND e.status='ready' AND policy.id=1 AND (policy.data->>'enabled')::boolean AND policy.revision=e.policy_revision`, v.ID, p.ID).Scan(&version)
	return v, version, e
}

func (s *Server) getAttachmentExtraction(w http.ResponseWriter, r *http.Request) {
	v, e := s.extractionRead(r.Context(), current(r), r.PathValue("id"))
	if e != nil {
		apiError(w, 404, "첨부 추출에 접근할 수 없습니다")
		return
	}
	_, version, activeErr := s.activeExtraction(r.Context(), current(r), v.ID)
	jsonResponse(w, 200, map[string]any{"extraction": v, "active": activeErr == nil, "current_document_version": version})
}

func (s *Server) attachmentExtractionContext(w http.ResponseWriter, r *http.Request) {
	p := current(r)
	if !validID(r.PathValue("id")) || !hasIntegrationScope(p, "document:read") {
		apiError(w, 404, "첨부파일에 접근할 수 없습니다")
		return
	}
	var id, doc, wid, name, title, checksum string
	var size int64
	var version int
	e := s.DB.QueryRow(r.Context(), `SELECT a.id::text,d.id::text,d.workspace_id::text,a.name,d.title,a.checksum_sha256,a.size,d.version FROM attachments a JOIN documents d ON d.id=a.document_id WHERE a.id=$1 AND d.deleted_at IS NULL AND madi_document_allowed($2,d.id,false)`, r.PathValue("id"), p.ID).Scan(&id, &doc, &wid, &name, &title, &checksum, &size, &version)
	if e != nil {
		apiError(w, 404, "첨부파일에 접근할 수 없습니다")
		return
	}
	policy, e := extractionPolicyQuery(r.Context(), s.DB, false)
	if e != nil {
		respond(w, nil, e)
		return
	}
	rows, e := s.DB.Query(r.Context(), "SELECT "+extractionRunSelect+" FROM attachment_extractions WHERE attachment_id=$1 ORDER BY created_at DESC LIMIT 50", id)
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer rows.Close()
	runs := []extractionRun{}
	for rows.Next() {
		v, err := scanExtractionRun(rows)
		if err != nil {
			respond(w, nil, err)
			return
		}
		runs = append(runs, v)
	}
	if e = rows.Err(); e != nil {
		respond(w, nil, e)
		return
	}
	if !s.canDocument(r.Context(), p, doc, false) {
		apiError(w, 404, "첨부파일에 접근할 수 없습니다")
		return
	}
	jsonResponse(w, 200, map[string]any{"id": id, "document_id": doc, "workspace_id": wid, "name": name, "document_title": title, "checksum": checksum, "size": size, "document_version": version, "format": extractionFormat(name), "can_write": s.canDocument(r.Context(), p, doc, true) && hasIntegrationScope(p, "document:write"), "runs": runs, "policy": policy})
}

func (s *Server) listAttachmentFragments(w http.ResponseWriter, r *http.Request) {
	v, version, e := s.activeExtraction(r.Context(), current(r), r.PathValue("id"))
	if e != nil {
		apiError(w, 410, "현재 첨부 본문이 없거나 원본·정책이 변경되었습니다. 첨부를 다시 확인하세요")
		return
	}
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if offset < 0 || offset > 20000 {
		apiError(w, 400, "본문 위치를 확인하세요")
		return
	}
	rows, e := s.DB.Query(r.Context(), `SELECT id::text,extraction_id::text,ordinal,text,content_hash,position FROM attachment_extraction_fragments WHERE extraction_id=$1 ORDER BY ordinal LIMIT 101 OFFSET $2`, v.ID, offset)
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer rows.Close()
	result := []attachmentFragment{}
	for rows.Next() {
		f, e := scanAttachmentFragment(rows)
		if e != nil {
			respond(w, nil, e)
			return
		}
		result = append(result, f)
	}
	if e = rows.Err(); e != nil {
		respond(w, nil, e)
		return
	}
	more := len(result) > 100
	if more {
		result = result[:100]
	}
	if _, _, e = s.activeExtraction(r.Context(), current(r), v.ID); e != nil {
		apiError(w, 410, "응답 준비 중 첨부 원본·권한·정책이 변경되었습니다")
		return
	}
	jsonResponse(w, 200, map[string]any{"extraction": v, "document_version": version, "fragments": result, "has_more": more, "next_offset": offset + len(result)})
}

func (s *Server) getAttachmentFragment(w http.ResponseWriter, r *http.Request) {
	v, version, e := s.activeExtraction(r.Context(), current(r), r.PathValue("id"))
	if e != nil || !validID(r.PathValue("fragment")) {
		apiError(w, 410, "현재 첨부 본문을 다시 확인하세요")
		return
	}
	f, e := scanAttachmentFragment(s.DB.QueryRow(r.Context(), `SELECT id::text,extraction_id::text,ordinal,text,content_hash,position FROM attachment_extraction_fragments WHERE extraction_id=$1 AND id=$2`, v.ID, r.PathValue("fragment")))
	if e != nil {
		apiError(w, 404, "본문 위치를 찾을 수 없습니다")
		return
	}
	if _, _, e = s.activeExtraction(r.Context(), current(r), v.ID); e != nil {
		apiError(w, 410, "본문 조회 중 현재 첨부 원본·권한이 변경되었습니다")
		return
	}
	jsonResponse(w, 200, map[string]any{"extraction": v, "document_version": version, "fragment": f})
}
