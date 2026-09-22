package server

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/jackc/pgx/v5"
)

func (s *Server) queueAttachmentExtraction(w http.ResponseWriter, r *http.Request) {
	p := current(r)
	id := r.PathValue("id")
	if p.PluginID != "" || !hasIntegrationScope(p, "document:read") || !hasIntegrationScope(p, "document:write") || !validID(id) {
		apiError(w, 403, "첨부 본문 추출에는 현재 문서 읽기·쓰기 권한과 직접 요청이 필요합니다")
		return
	}
	var in struct {
		DocumentVersion int    `json:"document_version"`
		Checksum        string `json:"checksum"`
		OCRPages        []int  `json:"ocr_pages"`
		Confirmation    string `json:"confirmation"`
	}
	if decode(r, &in) != nil || in.DocumentVersion < 1 || len(in.OCRPages) > 20 {
		apiError(w, 400, "현재 첨부와 OCR 페이지 선택을 확인하세요")
		return
	}
	if len(in.OCRPages) > 0 && in.Confirmation != "OCR" {
		apiError(w, 400, "빈 페이지에 대한 OCR 실행을 명시적으로 확인하세요")
		return
	}
	seen := map[int]bool{}
	for _, page := range in.OCRPages {
		if page < 1 || page > 500 || seen[page] {
			apiError(w, 400, "OCR 페이지는 중복 없이 1~500 사이입니다")
			return
		}
		seen[page] = true
	}
	policy, e := extractionPolicyQuery(r.Context(), s.DB, false)
	if e != nil {
		respond(w, nil, e)
		return
	}
	if !boolean(policy.Data, "enabled") || len(in.OCRPages) > 0 && !boolean(policy.Data, "ocr_enabled") {
		apiError(w, 403, "관리자가 해당 첨부 추출 기능을 켜지 않았습니다")
		return
	}
	v := extractionRun{ID: newID(), AttachmentID: id, ActorID: p.ID, PolicyRevision: policy.Revision, OCRPages: in.OCRPages, RequestIP: integrationClientIP(r)}
	if v.OCRPages == nil {
		v.OCRPages = []int{}
	}
	var name string
	e = s.DB.QueryRow(r.Context(), `SELECT d.id::text,d.workspace_id::text,d.version,a.name,a.path,coalesce(a.storage_provider_id::text,''),a.object_key,a.checksum_sha256,a.size FROM attachments a JOIN documents d ON d.id=a.document_id WHERE a.id=$1 AND d.deleted_at IS NULL AND madi_document_allowed($2,d.id,true)`, id, p.ID).Scan(&v.DocumentID, &v.WorkspaceID, &v.DocumentVersion, &name, &v.Object.Path, &v.Object.ProviderID, &v.Object.Key, &v.Object.Checksum, &v.Object.Size)
	if e != nil || p.WorkspaceID != "" && p.WorkspaceID != v.WorkspaceID {
		apiError(w, 404, "첨부파일을 찾을 수 없거나 접근할 수 없습니다")
		return
	}
	if in.DocumentVersion != v.DocumentVersion || in.Checksum != v.Object.Checksum {
		apiError(w, 409, "문서 또는 첨부 원본이 바뀌었습니다. 현재 상태를 확인하세요")
		return
	}
	v.Format = extractionFormat(name)
	if v.Format == "" || len(in.OCRPages) > 0 && v.Format != "pdf" {
		apiError(w, 400, "PDF·DOCX·PPTX·XLSX·UTF-8 텍스트를 지원하며 OCR은 PDF에만 적용합니다")
		return
	}
	if v.Object.Size > extractionLimits(policy).InputBytes {
		apiError(w, 413, "관리자가 설정한 추출 원본 크기 한도를 초과했습니다")
		return
	}
	if p.TokenID != "" {
		parts := strings.Fields(r.Header.Get("Authorization"))
		if len(parts) != 2 {
			apiError(w, 403, "원래 API 키를 확인하세요")
			return
		}
		v.TokenHash = integrationHash(parts[1])
	} else {
		cookie, e := r.Cookie("madi_session")
		if e != nil {
			apiError(w, 401, "다시 로그인하세요")
			return
		}
		v.SessionHash = digest(cookie.Value)
	}
	// Legacy attachments can lack a recorded checksum. Backfill from a bounded,
	// verified object read, then CAS the exact original storage metadata below.
	legacy := v.Object.Checksum == ""
	oldObject := v.Object
	if legacy {
		f, e := s.materializeObject(r.Context(), v.Object, extractionLimits(policy).InputBytes)
		if e != nil {
			apiError(w, 409, "첨부 원본 무결성을 확인할 수 없습니다")
			return
		}
		h := sha256.New()
		_, e = io.Copy(h, f)
		f.Close()
		os.Remove(f.Name())
		if e != nil {
			apiError(w, 409, "첨부 원본을 읽을 수 없습니다")
			return
		}
		v.Object.Checksum = hex.EncodeToString(h.Sum(nil))
	}
	v.Checksum = v.Object.Checksum
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	if legacy {
		tag, e := tx.Exec(r.Context(), `UPDATE attachments SET checksum_sha256=$2 WHERE id=$1 AND checksum_sha256='' AND path=$3 AND coalesce(storage_provider_id::text,'')=$4 AND object_key=$5 AND size=$6`, id, v.Checksum, oldObject.Path, oldObject.ProviderID, oldObject.Key, oldObject.Size)
		if e != nil || tag.RowsAffected() != 1 {
			apiError(w, 409, "첨부 원본이 바뀌었습니다. 다시 확인하세요")
			return
		}
	}
	if e = s.extractionGuardTx(r.Context(), tx, p, v); e != nil {
		apiError(w, 409, e.Error())
		return
	}
	if _, e = tx.Exec(r.Context(), "SELECT pg_advisory_xact_lock(hashtextextended($1,9741))", p.ID); e != nil {
		respond(w, nil, e)
		return
	}
	var pending int
	if e = tx.QueryRow(r.Context(), `SELECT count(*) FROM attachment_extractions WHERE actor_id=$1 AND status IN ('queued','running')`, p.ID).Scan(&pending); e != nil {
		respond(w, nil, e)
		return
	}
	if pending >= 5 {
		apiError(w, 429, "개인 첨부 추출 대기 작업은 최대 5개입니다")
		return
	}
	_, e = tx.Exec(r.Context(), `INSERT INTO attachment_extractions(id,attachment_id,document_id,workspace_id,actor_id,policy_revision,document_version,checksum,object_snapshot,format,ocr_pages,session_hash,token_hash,request_ip) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`, v.ID, id, v.DocumentID, v.WorkspaceID, p.ID, v.PolicyRevision, v.DocumentVersion, v.Checksum, jsonValue(v.Object), v.Format, v.OCRPages, v.SessionHash, v.TokenHash, v.RequestIP)
	if e != nil {
		apiError(w, 409, "이 첨부의 추출 작업이 이미 진행 중입니다")
		return
	}
	job, e := s.EnqueueJob(r.Context(), tx, "attachment.extract", p.ID, v.WorkspaceID, map[string]any{"extraction_id": v.ID})
	if e == nil {
		_, e = tx.Exec(r.Context(), `UPDATE attachment_extractions SET job_id=$2 WHERE id=$1`, v.ID, job)
	}
	if e == nil {
		_, e = tx.Exec(r.Context(), `UPDATE automation_jobs SET timeout_seconds=$2,max_attempts=2 WHERE id=$1`, job, number(policy.Data, "timeout_seconds", 300))
	}
	if e == nil {
		e = tx.Commit(r.Context())
	}
	if e != nil {
		respond(w, nil, e)
		return
	}
	s.audit(r, "ATTACHMENT_EXTRACTION_REQUEST", id, map[string]any{"extraction_id": v.ID, "ocr_pages": v.OCRPages})
	jsonResponse(w, 202, map[string]any{"id": v.ID, "job_id": job, "checksum": v.Checksum})
}

func (s *Server) listDocumentExtractions(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !hasIntegrationScope(current(r), "document:read") || !s.canDocument(r.Context(), current(r), id, false) {
		apiError(w, 404, "문서에 접근할 수 없습니다")
		return
	}
	rows, e := s.DB.Query(r.Context(), "SELECT "+extractionRunSelect+" FROM attachment_extractions WHERE document_id=$1 ORDER BY created_at DESC LIMIT 100", id)
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer rows.Close()
	result := []extractionRun{}
	for rows.Next() {
		v, e := scanExtractionRun(rows)
		if e != nil {
			respond(w, nil, e)
			return
		}
		result = append(result, v)
	}
	p, e := extractionPolicyQuery(r.Context(), s.DB, false)
	if e != nil {
		respond(w, nil, e)
		return
	}
	respond(w, map[string]any{"runs": result, "policy": p}, rows.Err())
}

func (s *Server) extractionRead(ctx context.Context, p *Principal, id string) (extractionRun, error) {
	if !validID(id) || !hasIntegrationScope(p, "document:read") {
		return extractionRun{}, pgx.ErrNoRows
	}
	v, e := scanExtractionRun(s.DB.QueryRow(ctx, "SELECT "+extractionRunSelect+" FROM attachment_extractions WHERE id=$1", id))
	if e != nil || !s.canDocument(ctx, p, v.DocumentID, false) {
		return extractionRun{}, pgx.ErrNoRows
	}
	return v, nil
}

func (s *Server) cancelAttachmentExtraction(w http.ResponseWriter, r *http.Request) {
	v, e := s.extractionRead(r.Context(), current(r), r.PathValue("id"))
	if e != nil || v.ActorID != current(r).ID || !s.canDocument(r.Context(), current(r), v.DocumentID, true) {
		apiError(w, 404, "해당 추출 작업에 접근할 수 없습니다")
		return
	}
	if !oneOf(v.Status, "queued", "running") {
		apiError(w, 409, "이미 완료된 추출은 취소할 수 없습니다")
		return
	}
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	updated, e := tx.Exec(r.Context(), `UPDATE attachment_extractions SET status='cancelled',revision=revision+1,session_hash='',token_hash='',updated_at=now() WHERE id=$1 AND status IN ('queued','running')`, v.ID)
	if e == nil && updated.RowsAffected() != 1 {
		apiError(w, 409, "취소 요청 중 추출이 완료되었습니다. 현재 결과를 다시 확인하세요")
		return
	}
	if e == nil {
		_, e = tx.Exec(r.Context(), `UPDATE automation_jobs SET cancel_requested=true,status=CASE WHEN status='pending' THEN 'cancelled' ELSE status END WHERE id=$1 AND status IN ('pending','running')`, v.JobID)
	}
	if e == nil {
		e = tx.Commit(r.Context())
	}
	respond(w, map[string]any{"cancelled": true}, e)
}
