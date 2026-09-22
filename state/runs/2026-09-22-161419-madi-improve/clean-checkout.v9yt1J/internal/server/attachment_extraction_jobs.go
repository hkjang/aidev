package server

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/hkjang/madi/internal/extract"
	"github.com/jackc/pgx/v5"
)

func (s *Server) registerAttachmentExtraction() {
	s.admin("GET /api/v1/admin/attachment-extraction/settings", s.extractionSettings)
	s.admin("PUT /api/v1/admin/attachment-extraction/settings", s.extractionSettings)
	s.handle("POST /api/v1/attachments/{id}/extractions", s.queueAttachmentExtraction)
	s.handle("GET /api/v1/attachments/{id}/extraction-context", s.attachmentExtractionContext)
	s.handle("GET /api/v1/documents/{id}/extractions", s.listDocumentExtractions)
	s.handle("DELETE /api/v1/attachment-extractions/{id}", s.cancelAttachmentExtraction)
	s.handle("GET /api/v1/attachment-extractions/{id}", s.getAttachmentExtraction)
	s.handle("GET /api/v1/attachment-extractions/{id}/fragments", s.listAttachmentFragments)
	s.handle("GET /api/v1/attachment-extractions/{id}/fragments/{fragment}", s.getAttachmentFragment)
	s.handle("GET /api/v1/attachment-extractions/{id}/fragments/{fragment}/citation", s.getAttachmentCitation)
	s.handle("POST /api/v1/attachment-extractions/{id}/ai", s.runAttachmentAI)
	s.RegisterJobHandler("attachment.extract", s.executeAttachmentExtraction)
}

func (s *Server) currentExtractionJob(ctx context.Context, j Job) (extractionRun, *Principal, error) {
	v, e := scanExtractionRun(s.DB.QueryRow(ctx, "SELECT "+extractionRunSelect+" FROM attachment_extractions WHERE id=$1", str(j.Payload, "extraction_id")))
	if e != nil {
		return v, nil, errExtractionChanged
	}
	p, e := s.workerPrincipal(ctx, j.ActorID, j.TokenID, j.WorkspaceID)
	if e != nil {
		return v, nil, errExtractionChanged
	}
	if v.ActorID != p.ID || v.JobID != j.ID || v.WorkspaceID != j.WorkspaceID || !oneOf(v.Status, "queued", "running", "ready") || !hasIntegrationScope(p, "document:write") || !hasIntegrationScope(p, "document:read") {
		return v, p, errExtractionChanged
	}
	var valid bool
	e = s.DB.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM attachments a JOIN documents d ON d.id=a.document_id CROSS JOIN attachment_extraction_settings policy WHERE a.id=$1 AND d.id=$2 AND d.workspace_id=$3 AND d.version=$4 AND d.deleted_at IS NULL AND madi_document_allowed($5,d.id,true) AND a.checksum_sha256=$6 AND policy.id=1 AND policy.revision=$7 AND (policy.data->>'enabled')::boolean AND ($8 OR (policy.data->>'ocr_enabled')::boolean) AND ($12 OR ($9<>'' AND EXISTS(SELECT 1 FROM sessions WHERE user_id=$5 AND token_hash=$9 AND expires_at>clock_timestamp())) OR ($10<>'' AND EXISTS(SELECT 1 FROM api_keys WHERE id=NULLIF($11,'')::uuid AND user_id=$5 AND token_hash=$10 AND revoked_at IS NULL AND expires_at>clock_timestamp()))))`, v.AttachmentID, v.DocumentID, v.WorkspaceID, v.DocumentVersion, p.ID, v.Checksum, v.PolicyRevision, len(v.OCRPages) == 0, v.SessionHash, v.TokenHash, p.TokenID, v.Status == "ready").Scan(&valid)
	if e != nil || !valid {
		return v, p, errExtractionChanged
	}
	return v, p, nil
}

func (s *Server) executeAttachmentExtraction(ctx context.Context, j Job) (result map[string]any, err error) {
	v, p, e := s.currentExtractionJob(ctx, j)
	if e != nil {
		_, _ = s.DB.Exec(ctx, `UPDATE attachment_extractions SET status='obsolete',session_hash='',token_hash='',updated_at=now() WHERE id=$1 AND job_id=$2 AND status IN ('queued','running')`, str(j.Payload, "extraction_id"), j.ID)
		return nil, jobPermanent(e.Error())
	}
	if v.Status == "ready" {
		return v.Result, nil
	}
	defer func() {
		if err != nil {
			cleanup, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			_, _ = s.DB.Exec(cleanup, `UPDATE attachment_extractions SET status='failed',error=$2,session_hash='',token_hash='',updated_at=now(),completed_at=now() WHERE id=$1 AND status IN ('queued','running')`, v.ID, "첨부 추출에 실패했거나 현재 원본·권한·정책이 바뀌었습니다. 원본을 확인하고 다시 요청하세요")
		}
	}()
	policy, e := extractionPolicyQuery(ctx, s.DB, false)
	if e != nil {
		return nil, e
	}
	if _, e = s.DB.Exec(ctx, `UPDATE attachment_extractions SET status='running',updated_at=now() WHERE id=$1 AND status IN ('queued','running')`, v.ID); e != nil {
		return nil, e
	}
	runCtx, cancel := context.WithTimeout(ctx, time.Duration(number(policy.Data, "timeout_seconds", 300))*time.Second)
	defer cancel()
	monitorDone := make(chan struct{})
	defer close(monitorDone)
	go func() {
		tick := time.NewTicker(time.Second)
		defer tick.Stop()
		for {
			select {
			case <-monitorDone:
				return
			case <-runCtx.Done():
				return
			case <-tick.C:
				if _, _, e := s.currentExtractionJob(runCtx, j); e != nil {
					cancel()
					return
				}
			}
		}
	}()
	var previousTemp string
	if e = s.DB.QueryRow(runCtx, "SELECT temp_path FROM attachment_extractions WHERE id=$1", v.ID).Scan(&previousTemp); e != nil {
		return nil, e
	}
	if previousTemp != "" {
		if e = cleanExtractionTemp(previousTemp); e != nil {
			return nil, jobPermanent("이전 추출 임시 폴더를 안전하게 정리하지 못했습니다")
		}
	}
	dir, e := os.MkdirTemp("", "madi-extract-")
	if e != nil {
		return nil, e
	}
	defer func() {
		_ = cleanExtractionTemp(dir)
		cleanup, stop := context.WithTimeout(context.Background(), time.Second)
		defer stop()
		_, _ = s.DB.Exec(cleanup, "UPDATE attachment_extractions SET temp_path='' WHERE id=$1 AND temp_path=$2", v.ID, dir)
	}()
	if _, e = s.DB.Exec(runCtx, "UPDATE attachment_extractions SET temp_path=$2 WHERE id=$1", v.ID, dir); e != nil {
		return nil, e
	}
	input, e := s.materializeObject(runCtx, v.Object, extractionLimits(policy).InputBytes)
	if e != nil {
		return nil, jobPermanent("첨부 원본 무결성을 확인할 수 없습니다")
	}
	e = extract.CopySource(dir, input, extractionLimits(policy).InputBytes)
	input.Close()
	os.Remove(input.Name())
	if e != nil {
		return nil, e
	}
	var projection extract.Result
	if v.Format == "pdf" {
		projection, e = extract.PDF(runCtx, dir, v.OCRPages, extractionLimits(policy))
	} else {
		f, openErr := os.Open(filepath.Join(dir, "source"))
		if openErr != nil {
			return nil, openErr
		}
		defer f.Close()
		if v.Format == "text" {
			var raw []byte
			raw, e = io.ReadAll(io.LimitReader(f, int64(extractionLimits(policy).TextBytes)+1))
			if e == nil {
				projection, e = extractionPlainText(raw, extractionLimits(policy))
			}
		} else {
			projection, e = extract.Office(runCtx, f, v.Object.Size, v.Format, extractionLimits(policy))
		}
	}
	if e != nil {
		return nil, jobPermanent("첨부 본문을 추출하지 못했습니다. 지원 형식·암호화·원본 크기·격리 설정을 확인하세요")
	}
	if _, p, e = s.currentExtractionJob(runCtx, j); e != nil {
		return nil, jobPermanent(e.Error())
	}
	tx, e := s.DB.Begin(runCtx)
	if e != nil {
		return nil, e
	}
	defer tx.Rollback(context.Background())
	var state string
	if tx.QueryRow(runCtx, "SELECT status FROM attachment_extractions WHERE id=$1 FOR UPDATE", v.ID).Scan(&state) != nil || state != "running" {
		return nil, jobPermanent("첨부 추출이 취소되었습니다")
	}
	if e = s.extractionGuardTx(runCtx, tx, p, v); e != nil {
		return nil, jobPermanent(e.Error())
	}
	masked, e := s.protectExtractionFragments(runCtx, tx, p, v, &projection)
	if e != nil {
		return nil, jobPermanent("민감정보 보호 정책에 따라 추출 본문을 저장하지 않았습니다")
	}
	rows := make([][]any, 0, len(projection.Fragments))
	for i, f := range projection.Fragments {
		rows = append(rows, []any{newID(), v.ID, i, f.Text, digest(f.Text), jsonValue(f.Position)})
	}
	if _, e = tx.CopyFrom(runCtx, pgx.Identifier{"attachment_extraction_fragments"}, []string{"id", "extraction_id", "ordinal", "text", "content_hash", "position"}, pgx.CopyFromRows(rows)); e != nil {
		return nil, e
	}
	_, e = tx.Exec(runCtx, `UPDATE attachment_extractions SET status='obsolete',revision=revision+1 WHERE attachment_id=$1 AND status='ready' AND id<>$2`, v.AttachmentID, v.ID)
	if e == nil {
		_, e = tx.Exec(runCtx, `INSERT INTO attachment_extraction_heads(attachment_id,extraction_id) VALUES($1,$2) ON CONFLICT(attachment_id) DO UPDATE SET extraction_id=EXCLUDED.extraction_id,updated_at=now()`, v.AttachmentID, v.ID)
	}
	result = map[string]any{"pages": projection.Pages, "empty_pages": projection.EmptyPages, "warnings": projection.Warnings, "fragments": len(projection.Fragments), "masked": masked, "format": v.Format, "source_checksum": v.Checksum, "extractor_revision": 1, "local_only": true}
	if e == nil {
		_, e = tx.Exec(runCtx, `UPDATE attachment_extractions SET status='ready',result=$2,completed_at=now(),updated_at=now(),session_hash='',token_hash='' WHERE id=$1`, v.ID, jsonValue(result))
	}
	// Final expiry and lease/cancellation guard: no visible fragment exists until
	// all original actor/source checks and the durable job receipt commit.
	if e == nil {
		e = s.extractionActorTx(runCtx, tx, p, v)
	}
	if e == nil {
		effect := context.WithValue(runCtx, automationEffectKey{}, automationEffectContext{JobID: j.ID, Index: 0})
		e = s.recordAutomationEffect(effect, tx, result)
	}
	if e == nil {
		e = tx.Commit(runCtx)
	}
	return result, e
}

func extractionPlainText(data []byte, limits extract.Limits) (extract.Result, error) {
	r := extract.Result{Fragments: []extract.Fragment{}, Warnings: []string{}, EmptyPages: []int{}}
	if len(data) > limits.TextBytes || !utf8.Valid(data) {
		return r, extract.ErrLimit
	}
	for i, line := range strings.Split(string(data), "\n") {
		for len(line) > 0 {
			n := len(line)
			if n > 16<<10 {
				n = 16 << 10
				for !utf8.RuneStart(line[n]) {
					n--
				}
			}
			if strings.TrimSpace(line[:n]) != "" {
				r.Fragments = append(r.Fragments, extract.Fragment{Text: line[:n], Position: extract.Position{Paragraph: i + 1}})
			}
			line = line[n:]
			if len(r.Fragments) > limits.Fragments {
				return r, extract.ErrLimit
			}
		}
	}
	return r, nil
}

func (s *Server) protectExtractionFragments(ctx context.Context, tx pgx.Tx, p *Principal, v extractionRun, result *extract.Result) (bool, error) {
	masked := false
	for start := 0; start < len(result.Fragments); {
		end, size := start, 0
		values := []any{}
		for end < len(result.Fragments) && end-start < 1000 && size+len(result.Fragments[end].Text) <= 512<<10 {
			values = append(values, result.Fragments[end])
			size += len(result.Fragments[end].Text)
			end++
		}
		clean, e := s.ProtectDocumentMetadataTx(ctx, tx, p, v.DocumentID, v.WorkspaceID, values)
		if e != nil {
			return masked, e
		}
		if len(clean.Findings) > 0 {
			result.Warnings = append(result.Warnings, "추출 본문·위치 이름에서 민감정보 후보를 감지했습니다. 보호 정책을 적용했으며 원본 첨부 자체는 바꾸지 않습니다")
		}
		if clean.Changed {
			masked = true
			raw, e := json.Marshal(clean.Value)
			if e != nil {
				return masked, e
			}
			var fragments []extract.Fragment
			if json.Unmarshal(raw, &fragments) != nil || len(fragments) != end-start {
				return masked, errors.New("정제 본문 형식을 확인하세요")
			}
			for i, fragment := range fragments {
				if len(fragment.Text) > 16384 {
					return masked, extract.ErrLimit
				}
				result.Fragments[start+i] = fragment
			}
		}
		start = end
	}
	return masked, nil
}

func cleanExtractionTemp(dir string) error {
	if dir == "" {
		return nil
	}
	if filepath.Dir(dir) != os.TempDir() || !strings.HasPrefix(filepath.Base(dir), "madi-extract-") {
		return errors.New("임시 추출 경로를 확인하세요")
	}
	info, e := os.Lstat(dir)
	if os.IsNotExist(e) {
		return nil
	}
	if e != nil || !info.IsDir() || info.Mode().Perm() != 0700 {
		return errors.New("임시 추출 경로의 소유 범위를 확인하세요")
	}
	// This exact directory was generated by MkdirTemp and persisted before any
	// input bytes. Landlock denies native symlink creation and sibling access.
	return os.RemoveAll(dir)
}

func (s *Server) StartAttachmentExtraction(ctx context.Context) {
	go func() {
		cleanup := func() {
			// A common job can be rejected before this handler starts (for example a
			// revoked API key). Reconcile that terminal state without replaying it.
			_, _ = s.DB.Exec(ctx, `UPDATE attachment_extractions e SET status='cancelled',session_hash='',token_hash='',completed_at=now(),updated_at=now() WHERE e.id IN(SELECT v.id FROM attachment_extractions v JOIN automation_jobs j ON j.id=v.job_id WHERE v.status IN ('queued','running') AND j.status IN ('failed','cancelled') ORDER BY v.updated_at LIMIT 1000)`)
			// Only the active projection is needed for current search. Keep small
			// result metadata but discard superseded derived bodies after 24 hours.
			_, _ = s.DB.Exec(ctx, `DELETE FROM attachment_extraction_fragments WHERE id IN(SELECT f.id FROM attachment_extraction_fragments f JOIN attachment_extractions e ON f.extraction_id=e.id WHERE e.status IN ('obsolete','failed','cancelled') AND e.updated_at<now()-interval '24 hours' AND NOT EXISTS(SELECT 1 FROM attachment_extraction_heads h WHERE h.extraction_id=e.id) ORDER BY e.updated_at,f.id LIMIT 2000)`)
			rows, e := s.DB.Query(ctx, `SELECT id::text,temp_path FROM attachment_extractions e WHERE temp_path<>'' AND NOT EXISTS(SELECT 1 FROM automation_jobs j WHERE j.id=e.job_id AND j.status='running' AND j.lease_until>clock_timestamp()) LIMIT 100`)
			if e != nil {
				return
			}
			type entry struct{ id, path string }
			var values []entry
			for rows.Next() {
				var v entry
				if rows.Scan(&v.id, &v.path) == nil {
					values = append(values, v)
				}
			}
			rows.Close()
			for _, v := range values {
				if cleanExtractionTemp(v.path) == nil {
					_, _ = s.DB.Exec(ctx, "UPDATE attachment_extractions SET temp_path='' WHERE id=$1 AND temp_path=$2", v.id, v.path)
				}
			}
		}
		cleanup()
		tick := time.NewTicker(30 * time.Second)
		defer tick.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-tick.C:
				cleanup()
			}
		}
	}()
}
