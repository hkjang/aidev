package server

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
)

//go:embed export_jobs.sql
var exportJobsSchema string

type exportRun struct {
	ID, WorkspaceID, OwnerID, TokenID, SessionHash, IP, Format, DatabaseID, Fingerprint, Status, JobID string
	TokenBound                                                                                         bool
	Constraints                                                                                        actorConstraints
	IDs                                                                                                []string
	Revision                                                                                           int64
	Expires                                                                                            time.Time
}

func (s *Server) migrateExports(ctx context.Context) error {
	_, e := s.DB.Exec(ctx, exportJobsSchema)
	return e
}
func (s *Server) registerExports() {
	s.handle("GET /api/v1/exports", s.listExports)
	s.handle("POST /api/v1/exports", s.createExport)
	s.handle("GET /api/v1/exports/{id}", s.getExport)
	s.handle("DELETE /api/v1/exports/{id}", s.cancelExport)
	s.handle("GET /api/v1/exports/{id}/download", s.downloadExport)
	s.admin("GET /api/v1/admin/exports/settings", s.getExportSettings)
	s.admin("PUT /api/v1/admin/exports/settings", s.putExportSettings)
	s.RegisterJobHandler("export.build", s.executeExportJob)
}
func (s *Server) StartExports(ctx context.Context) {
	go func() {
		t := time.NewTicker(time.Minute)
		defer t.Stop()
		for {
			if ctx.Err() != nil {
				return
			}
			s.expireExports(ctx)
			s.expireDistributions(ctx)
			select {
			case <-ctx.Done():
				return
			case <-t.C:
			}
		}
	}()
}
func (s *Server) expireExports(ctx context.Context) {
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return
	}
	defer tx.Rollback(ctx)
	_, e = tx.Exec(ctx, `UPDATE export_runs SET status=CASE WHEN status IN ('queued','running') THEN 'cancelled' ELSE 'expired' END,updated_at=now() WHERE expires_at<=now() AND status IN ('queued','running','ready')`)
	if e == nil {
		_, e = tx.Exec(ctx, `UPDATE automation_jobs SET cancel_requested=true WHERE id IN (SELECT job_id FROM export_runs WHERE status IN ('cancelled','expired')) AND status IN ('pending','running')`)
	}
	if e == nil {
		_, e = tx.Exec(ctx, `DELETE FROM export_artifact_blobs WHERE run_id IN (SELECT id FROM export_runs WHERE status IN ('cancelled','expired','failed'))`)
	}
	if e == nil {
		_, e = tx.Exec(ctx, `UPDATE export_runs r SET status=CASE WHEN j.status='cancelled' THEN 'cancelled' ELSE 'failed' END,report='{"error":"작업이 완료되지 않았습니다. 현재 권한과 원본을 확인하고 다시 생성하세요"}',updated_at=now() FROM automation_jobs j WHERE r.job_id=j.id AND r.status IN ('queued','running') AND j.status IN ('failed','cancelled')`)
	}
	if e == nil {
		_ = tx.Commit(ctx)
	}
}
func (s *Server) readExport(ctx context.Context, id string) (exportRun, error) {
	return scanExportRun(s.DB.QueryRow(ctx, "SELECT "+exportRunSelect+" FROM export_runs WHERE id=$1", id))
}

const exportRunSelect = `id::text,workspace_id::text,owner_id::text,coalesce(token_id::text,''),token_bound,actor_constraints,session_hash,request_ip,format,document_ids,coalesce(database_id::text,''),source_fingerprint,policy_revision,status,coalesce(job_id::text,''),expires_at`

func scanExportRun(row pgx.Row) (exportRun, error) {
	var r exportRun
	var c []byte
	e := row.Scan(&r.ID, &r.WorkspaceID, &r.OwnerID, &r.TokenID, &r.TokenBound, &c, &r.SessionHash, &r.IP, &r.Format, &r.IDs, &r.DatabaseID, &r.Fingerprint, &r.Revision, &r.Status, &r.JobID, &r.Expires)
	if e == nil {
		e = json.Unmarshal(c, &r.Constraints)
	}
	return r, e
}
func (s *Server) exportPrincipal(ctx context.Context, r exportRun) (*Principal, error) {
	if r.TokenBound && r.TokenID == "" {
		return nil, errors.New("원래 API 키가 삭제되었습니다")
	}
	ctx = context.WithValue(ctx, jobContextKey{}, jobContext{ActorID: r.OwnerID, TokenID: r.TokenID, Constraints: r.Constraints})
	p, e := s.workerPrincipal(ctx, r.OwnerID, r.TokenID, r.WorkspaceID)
	if e != nil {
		return nil, e
	}
	if r.SessionHash != "" {
		var active bool
		if s.DB.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM sessions WHERE user_id=$1 AND token_hash=$2 AND expires_at>now())`, r.OwnerID, r.SessionHash).Scan(&active) != nil || !active {
			return nil, errors.New("내보내기를 요청한 로그인 세션이 만료되었습니다")
		}
	}
	if r.TokenID != "" {
		var allowed []string
		if s.DB.QueryRow(ctx, `SELECT ip_allowlist FROM api_keys WHERE id=$1`, r.TokenID).Scan(&allowed) != nil || !integrationIPAllowed(r.IP, allowed) {
			return nil, errors.New("원래 API 키의 IP 정책이 변경되었습니다")
		}
	}
	return p, nil
}
func (s *Server) exportFingerprint(ctx context.Context, p *Principal, r exportRun) (string, error) {
	if !s.canWorkspace(ctx, p, r.WorkspaceID, false) {
		return "", errors.New("워크스페이스 접근 권한이 없습니다")
	}
	if r.Format == "csv" {
		req := (&http.Request{}).WithContext(context.WithValue(ctx, principalKey, p))
		if !hasIntegrationScope(p, "database:read") || !s.canDatabase(req, r.DatabaseID, false) {
			return "", errors.New("데이터베이스 내보내기 권한이 없습니다")
		}
		var fp string
		var count int
		e := s.DB.QueryRow(ctx, `SELECT count(*) FROM (SELECT id FROM database_rows WHERE database_id=$1 LIMIT 10001) x`, r.DatabaseID).Scan(&count)
		if e != nil {
			return "", e
		}
		if count > 10000 {
			return "", errors.New("CSV 내보내기는 최대 10,000행입니다")
		}
		var size int64
		if e = s.DB.QueryRow(ctx, `SELECT coalesce(sum(octet_length(values::text)),0) FROM database_rows WHERE database_id=$1`, r.DatabaseID).Scan(&size); e != nil {
			return "", e
		}
		if size > 100<<20 {
			return "", errors.New("CSV 원본 데이터가 100MB를 초과합니다")
		}
		e = s.DB.QueryRow(ctx, `SELECT md5(to_jsonb(d)::text||coalesce((SELECT string_agg(md5(to_jsonb(x)::text),'' ORDER BY x.id) FROM database_rows x WHERE x.database_id=d.id),'')) FROM databases d WHERE id=$1 AND workspace_id=$2`, r.DatabaseID, r.WorkspaceID).Scan(&fp)
		return fp, e
	}
	if !hasIntegrationScope(p, "document:read") || len(r.IDs) == 0 || len(r.IDs) > 1000 {
		return "", errors.New("문서 내보내기 범위가 올바르지 않습니다")
	}
	var fp string
	var count int
	if e := s.DB.QueryRow(ctx, `SELECT count(*) FROM (SELECT id FROM attachments WHERE document_id=ANY($1::uuid[]) LIMIT 5001) bounded`, r.IDs).Scan(&count); e != nil {
		return "", e
	}
	if count+len(r.IDs)+1 > 5000 {
		return "", errors.New("첨부를 포함하여 최대 5,000개 파일까지 내보낼 수 있습니다")
	}
	e := s.DB.QueryRow(ctx, `SELECT coalesce(md5(string_agg(md5(jsonb_build_object('id',d.id,'version',d.version,'title',d.title,'tags',d.tags,'aliases',d.aliases,'parent_id',d.parent_id,'space_id',d.space_id,'visibility',d.visibility,'status',d.status,'icon',d.icon,'attachments',(SELECT jsonb_agg(to_jsonb(a)||jsonb_build_object('provider',(SELECT to_jsonb(sp) FROM storage_providers sp WHERE sp.id=a.storage_provider_id)) ORDER BY a.id) FROM attachments a WHERE a.document_id=d.id))::text),'' ORDER BY d.id)),''),count(*) FROM documents d WHERE d.id=ANY($1::uuid[]) AND d.workspace_id=$2 AND d.deleted_at IS NULL AND madi_document_allowed($3,d.id,false)`, r.IDs, r.WorkspaceID, p.ID).Scan(&fp, &count)
	if count != len(r.IDs) {
		return "", errors.New("선택 문서의 현재 권한을 확인할 수 없습니다")
	}
	return fp, e
}
func (s *Server) validateExport(ctx context.Context, r exportRun) (*Principal, int64, error) {
	if !r.Expires.After(time.Now()) || oneOf(r.Status, "cancelled", "expired", "failed") {
		return nil, 0, errors.New("내보내기 작업이 취소되었거나 만료되었습니다")
	}
	var enabled bool
	var revision, max int64
	e := s.DB.QueryRow(ctx, `SELECT enabled,revision,max_archive_mb FROM export_settings WHERE id`).Scan(&enabled, &revision, &max)
	if e != nil || !enabled || revision != r.Revision {
		return nil, 0, errors.New("관리자 내보내기 정책이 변경되었습니다")
	}
	p, e := s.exportPrincipal(ctx, r)
	if e != nil {
		return nil, 0, e
	}
	fp, e := s.exportFingerprint(ctx, p, r)
	if e != nil || fp != r.Fingerprint {
		return nil, 0, errors.New("원본 문서·첨부·권한이 변경되었습니다. 내보내기를 다시 생성하세요")
	}
	return p, max << 20, nil
}

const exportPublic = `to_jsonb(e)-ARRAY['token_id','token_bound','actor_constraints','session_hash','request_ip','source_fingerprint']||jsonb_build_object('job_status',(SELECT status FROM automation_jobs WHERE id=e.job_id))`

func (s *Server) listExports(w http.ResponseWriter, r *http.Request) {
	wid := r.URL.Query().Get("workspace_id")
	if !s.canWorkspace(r.Context(), current(r), wid, false) {
		apiError(w, 403, "내보내기 조회 권한이 없습니다")
		return
	}
	v, e := s.rows(r.Context(), `SELECT `+exportPublic+` FROM export_runs e WHERE workspace_id=$1 AND owner_id=$2 ORDER BY created_at DESC LIMIT 100`, wid, current(r).ID)
	respond(w, v, e)
}
func (s *Server) getExport(w http.ResponseWriter, r *http.Request) {
	v, e := s.one(r.Context(), `SELECT `+exportPublic+` FROM export_runs e WHERE id=$1 AND owner_id=$2`, r.PathValue("id"), current(r).ID)
	if e != nil || !s.canWorkspace(r.Context(), current(r), str(v, "workspace_id"), false) {
		apiError(w, 404, "내보내기 작업이 없습니다")
		return
	}
	respond(w, v, nil)
}
func (s *Server) createExport(w http.ResponseWriter, r *http.Request) {
	var in struct {
		WorkspaceID string   `json:"workspace_id"`
		Format      string   `json:"format"`
		DocumentIDs []string `json:"document_ids"`
		DatabaseID  string   `json:"database_id"`
	}
	if decode(r, &in) != nil || !oneOf(in.Format, "markdown", "portable", "html", "json", "csv") {
		apiError(w, 400, "내보내기 형식을 선택하세요")
		return
	}
	run := exportRun{ID: newID(), WorkspaceID: in.WorkspaceID, OwnerID: current(r).ID, TokenID: current(r).TokenID, TokenBound: current(r).TokenID != "", Constraints: constraintsFor(current(r)), IP: integrationClientIP(r), Format: in.Format, IDs: in.DocumentIDs, DatabaseID: in.DatabaseID, Expires: time.Now().Add(24 * time.Hour), Status: "queued"}
	if cookie, e := r.Cookie("madi_session"); e == nil && run.TokenID == "" {
		run.SessionHash = digest(cookie.Value)
	}
	seen := map[string]bool{}
	for _, id := range run.IDs {
		if !validID(id) || seen[id] {
			apiError(w, 400, "문서 ID 선택이 중복되거나 올바르지 않습니다")
			return
		}
		seen[id] = true
	}
	if run.Format == "csv" {
		run.IDs = []string{}
		if !validID(run.DatabaseID) {
			apiError(w, 400, "데이터베이스를 선택하세요")
			return
		}
	} else {
		run.DatabaseID = ""
	}
	var enabled bool
	var max int64
	e := s.DB.QueryRow(r.Context(), `SELECT enabled,revision,max_archive_mb FROM export_settings WHERE id`).Scan(&enabled, &run.Revision, &max)
	if e != nil || !enabled {
		apiError(w, 403, "관리자가 내보내기를 사용 중지했습니다")
		return
	}
	run.Fingerprint, e = s.exportFingerprint(r.Context(), current(r), run)
	if e != nil {
		apiError(w, 403, e.Error())
		return
	}
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	_, e = tx.Exec(r.Context(), `SELECT pg_advisory_xact_lock(hashtextextended($1,4457))`, run.OwnerID)
	var active int
	var used int64
	if e == nil {
		e = tx.QueryRow(r.Context(), `SELECT count(*) FILTER(WHERE status IN ('queued','running')),coalesce(sum(artifact_bytes) FILTER(WHERE status='ready' AND expires_at>now()),0) FROM export_runs WHERE owner_id=$1`, run.OwnerID).Scan(&active, &used)
	}
	if e != nil {
		respond(w, nil, e)
		return
	}
	if active >= 5 || used >= 200<<20 {
		apiError(w, 409, "대기 작업 5개 또는 임시 결과 200MB 한도입니다. 기존 작업을 취소하세요")
		return
	}
	_, e = tx.Exec(r.Context(), `INSERT INTO export_runs(id,workspace_id,owner_id,token_id,token_bound,actor_constraints,session_hash,request_ip,format,document_ids,database_id,source_fingerprint,policy_revision) VALUES($1,$2,$3,NULLIF($4,'')::uuid,$5,$6,$7,$8,$9,$10,NULLIF($11,'')::uuid,$12,$13)`, run.ID, run.WorkspaceID, run.OwnerID, run.TokenID, run.TokenBound, jsonValue(run.Constraints), run.SessionHash, run.IP, run.Format, run.IDs, run.DatabaseID, run.Fingerprint, run.Revision)
	if e == nil {
		run.JobID, e = s.EnqueueJob(r.Context(), tx, "export.build", run.OwnerID, run.WorkspaceID, map[string]any{"export_id": run.ID})
	}
	if e == nil {
		_, e = tx.Exec(r.Context(), `UPDATE export_runs SET job_id=$2 WHERE id=$1`, run.ID, run.JobID)
	}
	if e == nil {
		_, e = tx.Exec(r.Context(), `UPDATE automation_jobs SET timeout_seconds=3600,max_attempts=3 WHERE id=$1`, run.JobID)
	}
	if e == nil {
		e = tx.Commit(r.Context())
	}
	respond(w, map[string]any{"id": run.ID, "job_id": run.JobID, "status": "queued"}, e)
}
func (s *Server) cancelExport(w http.ResponseWriter, r *http.Request) {
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	tag, e := tx.Exec(r.Context(), `UPDATE export_runs SET status='cancelled',updated_at=now() WHERE id=$1 AND owner_id=$2`, r.PathValue("id"), current(r).ID)
	if e != nil || tag.RowsAffected() != 1 {
		apiError(w, 404, "내보내기 작업이 없습니다")
		return
	}
	_, e = tx.Exec(r.Context(), `UPDATE automation_jobs SET cancel_requested=true WHERE id=(SELECT job_id FROM export_runs WHERE id=$1) AND status IN ('pending','running')`, r.PathValue("id"))
	if e == nil {
		_, e = tx.Exec(r.Context(), `DELETE FROM export_artifact_blobs WHERE run_id=$1`, r.PathValue("id"))
	}
	if e == nil {
		e = tx.Commit(r.Context())
	}
	respond(w, map[string]any{"cancelled": true}, e)
}
func (s *Server) getExportSettings(w http.ResponseWriter, r *http.Request) {
	v, e := s.one(r.Context(), `SELECT to_jsonb(e) FROM export_settings e WHERE id`)
	respond(w, v, e)
}
func (s *Server) putExportSettings(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Enabled  bool  `json:"enabled"`
		Max      int   `json:"max_archive_mb"`
		Revision int64 `json:"revision"`
	}
	if decode(r, &in) != nil || in.Max < 1 || in.Max > 100 {
		apiError(w, 400, "압축 결과 한도는 1~100MB입니다")
		return
	}
	tag, e := s.DB.Exec(r.Context(), `UPDATE export_settings SET enabled=$1,max_archive_mb=$2,revision=revision+1,updated_at=now() WHERE id AND revision=$3`, in.Enabled, in.Max, in.Revision)
	if e == nil && tag.RowsAffected() != 1 {
		apiError(w, 409, "정책이 변경되었습니다. 다시 조회하세요")
		return
	}
	s.audit(r, "EXPORT_POLICY_UPDATE", "exports", map[string]any{"enabled": in.Enabled, "max_archive_mb": in.Max})
	respond(w, map[string]any{"saved": true}, e)
}
func (s *Server) executeExportJob(ctx context.Context, j Job) (result map[string]any, err error) {
	run, e := s.readExport(ctx, str(j.Payload, "export_id"))
	if e != nil {
		return nil, jobPermanent("내보내기 작업을 찾지 못했습니다")
	}
	if run.Status == "ready" {
		return map[string]any{"id": run.ID, "ready": true}, nil
	}
	defer func() {
		if err != nil {
			_, _ = s.DB.Exec(context.WithoutCancel(ctx), `UPDATE export_runs SET status='failed',report=$2,updated_at=now() WHERE id=$1 AND status IN ('queued','running')`, run.ID, jsonValue(map[string]any{"error": "원본·권한·크기·저장소 상태를 확인하고 다시 생성하세요"}))
		}
	}()
	p, max, e := s.validateExport(ctx, run)
	if e != nil {
		return nil, jobPermanent(e.Error())
	}
	_, e = s.DB.Exec(ctx, `UPDATE export_runs SET status='running',updated_at=now() WHERE id=$1 AND status='queued'`, run.ID)
	if e != nil {
		return nil, e
	}
	data, name, kind, e := s.buildExportArtifact(ctx, p, run, max)
	if e != nil {
		return nil, jobPermanent(e.Error())
	}
	latest, e := s.readExport(ctx, run.ID)
	if e != nil {
		return nil, e
	}
	if _, _, e = s.validateExport(ctx, latest); e != nil {
		return nil, jobPermanent(e.Error())
	}
	cipher, e := s.encrypt(string(data))
	if e != nil {
		return nil, e
	}
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return nil, e
	}
	defer tx.Rollback(ctx)
	// Hold canonical resources until the ready transition commits. FK locks also
	// serialize attachment/row insertion with this short completion checkpoint.
	_, e = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,4457))`, run.OwnerID)
	if e == nil {
		_, e = tx.Exec(ctx, `SELECT id FROM export_settings WHERE id FOR SHARE`)
	}
	if e == nil {
		_, e = tx.Exec(ctx, `SELECT id FROM users WHERE id=$1 FOR SHARE`, run.OwnerID)
	}
	if e == nil && run.Format == "csv" {
		_, e = tx.Exec(ctx, `SELECT id FROM databases WHERE id=$1 FOR UPDATE`, run.DatabaseID)
	}
	if e == nil && run.Format != "csv" {
		_, e = tx.Exec(ctx, `SELECT id FROM documents WHERE id=ANY($1::uuid[]) ORDER BY id FOR UPDATE`, run.IDs)
	}
	if e != nil {
		return nil, e
	}
	if _, _, e = s.validateExport(ctx, latest); e != nil {
		return nil, jobPermanent(e.Error())
	}
	var used int64
	if e = tx.QueryRow(ctx, `SELECT coalesce(sum(artifact_bytes),0) FROM export_runs WHERE owner_id=$1 AND status='ready' AND expires_at>now() AND id<>$2`, run.OwnerID, run.ID).Scan(&used); e != nil {
		return nil, e
	}
	if used+int64(len(data)) > 200<<20 {
		return nil, jobPermanent("개인별 임시 결과 200MB 한도입니다. 기존 결과를 폐기하고 다시 생성하세요")
	}
	var status string
	e = tx.QueryRow(ctx, `SELECT status FROM export_runs WHERE id=$1 FOR UPDATE`, run.ID).Scan(&status)
	if e != nil {
		return nil, e
	}
	if status != "running" {
		return nil, jobPermanent("내보내기가 취소되었습니다")
	}
	_, e = tx.Exec(ctx, `INSERT INTO export_artifact_blobs(run_id,ciphertext) VALUES($1,$2) ON CONFLICT(run_id) DO UPDATE SET ciphertext=excluded.ciphertext`, run.ID, []byte(cipher))
	if e == nil {
		_, e = tx.Exec(ctx, `UPDATE export_runs SET status='ready',filename=$2,content_type=$3,artifact_bytes=$4,report=$5,updated_at=now() WHERE id=$1`, run.ID, name, kind, len(data), jsonValue(map[string]any{"documents": len(run.IDs), "format": run.Format, "retention_hours": 24, "regenerable": true}))
	}
	if e == nil {
		e = tx.Commit(ctx)
	}
	return map[string]any{"id": run.ID, "bytes": len(data)}, e
}
func (s *Server) downloadExport(w http.ResponseWriter, r *http.Request) {
	run, e := s.readExport(r.Context(), r.PathValue("id"))
	if e != nil || run.OwnerID != current(r).ID || run.Status != "ready" || run.TokenID != current(r).TokenID {
		apiError(w, 404, "다운로드할 결과가 없습니다")
		return
	}
	if _, _, e = s.validateExport(r.Context(), run); e != nil {
		apiError(w, 409, e.Error())
		return
	}
	scope := "document:read"
	if run.Format == "csv" {
		scope = "database:read"
	}
	if !hasIntegrationScope(current(r), scope) {
		apiError(w, 403, "현재 키의 다운로드 권한이 없습니다")
		return
	}
	var cipher []byte
	var name, kind string
	e = s.DB.QueryRow(r.Context(), `SELECT b.ciphertext,e.filename,e.content_type FROM export_runs e JOIN export_artifact_blobs b ON b.run_id=e.id WHERE e.id=$1`, run.ID).Scan(&cipher, &name, &kind)
	if e != nil {
		apiError(w, 410, "임시 결과가 만료되었습니다. 다시 생성하세요")
		return
	}
	data, e := s.decrypt(string(cipher))
	if e != nil {
		apiError(w, 500, "임시 결과를 복호화할 수 없습니다")
		return
	}
	w.Header().Set("Content-Type", kind)
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, name))
	w.Header().Set("Content-Length", fmt.Sprint(len(data)))
	checked := time.Time{}
	for offset := 0; offset < len(data); {
		if time.Since(checked) > time.Second {
			latest, err := s.readExport(r.Context(), run.ID)
			if err != nil {
				return
			}
			if _, _, err = s.validateExport(r.Context(), latest); err != nil {
				return
			}
			checked = time.Now()
		}
		end := min(offset+64<<10, len(data))
		if _, e = w.Write([]byte(data[offset:end])); e != nil {
			return
		}
		offset = end
	}
	s.audit(r, "EXPORT_DOWNLOAD", run.ID, map[string]any{"format": run.Format, "bytes": len(data)})
}
