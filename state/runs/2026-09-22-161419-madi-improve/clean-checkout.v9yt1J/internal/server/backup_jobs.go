package server

import (
	"archive/zip"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const (
	backupCompressedLimit = 256 << 20
	backupExpandedLimit   = 1 << 30
	backupEntryLimit      = 10000
)

var errBackupTooLarge = errors.New("논리 백업은 압축 256MB, 해제 1GB, 10,000개 항목까지 지원합니다. 대규모 운영은 PostgreSQL 백업과 스토리지 백업을 사용하세요")

type backupLimitWriter struct {
	w         io.Writer
	remaining int64
}

func (w *backupLimitWriter) Write(p []byte) (int, error) {
	if int64(len(p)) > w.remaining {
		return 0, errBackupTooLarge
	}
	n, err := w.w.Write(p)
	w.remaining -= int64(n)
	return n, err
}

func (s *Server) createBackupFile(ctx context.Context) (*os.File, error) {
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return nil, e
	}
	defer tx.Rollback(ctx)
	// Shared lifecycle lock prevents attachment deletion during the repeatable
	// snapshot. New uploads create immutable objects and do not invalidate it.
	if _, e = tx.Exec(ctx, "SET TRANSACTION ISOLATION LEVEL REPEATABLE READ READ ONLY"); e != nil {
		return nil, e
	}
	if _, e = tx.Exec(ctx, "SELECT pg_advisory_xact_lock(726234801)"); e != nil {
		return nil, e
	}
	snapshot := map[string]json.RawMessage{}
	var metadataSize int64
	for _, table := range backupTables {
		var raw []byte
		if e = tx.QueryRow(ctx, "SELECT coalesce(jsonb_agg(to_jsonb(t)),'[]'::jsonb) FROM "+table+" t").Scan(&raw); e != nil {
			return nil, e
		}
		snapshot[table] = raw
		metadataSize += int64(len(raw))
		if metadataSize > backupCompressedLimit {
			return nil, errBackupTooLarge
		}
	}
	manifest := map[string]any{"format": "madi-logical-backup", "version": s.Version, "created_at": time.Now().UTC(), "tables": snapshot, "encryption_key_included": false, "restore": "Use the same madi version and ENCRYPTION_KEY; all attachments restore to target local storage. Native PostgreSQL backup remains required for PITR."}
	manifestBytes, e := json.Marshal(manifest)
	if e != nil {
		return nil, e
	}
	if len(manifestBytes) > backupCompressedLimit {
		return nil, errBackupTooLarge
	}
	var attachments []map[string]any
	if e = json.Unmarshal(snapshot["attachments"], &attachments); e != nil {
		return nil, e
	}
	if len(attachments)+1 > backupEntryLimit {
		return nil, errBackupTooLarge
	}
	expanded := int64(len(manifestBytes))
	for _, a := range attachments {
		size := objectMap(a).Size
		if size < 0 || size > 50<<20 || size > backupExpandedLimit-expanded {
			return nil, errBackupTooLarge
		}
		expanded += size
	}
	tmp, e := os.CreateTemp("", "madi-backup-*.zip")
	if e != nil {
		return nil, e
	}
	good := false
	defer func() {
		if !good {
			tmp.Close()
			os.Remove(tmp.Name())
		}
	}()
	zw := zip.NewWriter(&backupLimitWriter{w: tmp, remaining: backupCompressedLimit})
	file, e := zw.Create("backup.json")
	if e == nil {
		_, e = file.Write(manifestBytes)
	}
	for _, a := range attachments {
		if e != nil {
			break
		}
		var src *os.File
		src, e = s.materializeObject(ctx, objectMap(a), 50<<20)
		if e != nil {
			break
		}
		file, e = zw.Create("attachments/" + str(a, "id"))
		if e == nil {
			_, e = io.Copy(file, contextVaultReader{ctx, src})
		}
		src.Close()
		os.Remove(src.Name())
	}
	if closeErr := zw.Close(); e == nil {
		e = closeErr
	}
	if e != nil {
		return nil, e
	}
	if e = tx.Commit(ctx); e != nil {
		return nil, e
	}
	if _, e = tmp.Seek(0, 0); e != nil {
		return nil, e
	}
	good = true
	return tmp, nil
}
func (s *Server) registerBackupJobs() {
	s.RegisterJobHandler("backup.snapshot", s.executeScheduledBackup)
	s.admin("GET /api/v1/admin/backups/policy", func(w http.ResponseWriter, r *http.Request) {
		v, e := s.one(r.Context(), "SELECT to_jsonb(p) FROM backup_policy p WHERE id=1")
		respond(w, v, e)
	})
	s.admin("PUT /api/v1/admin/backups/policy", s.saveBackupPolicy)
	s.admin("POST /api/v1/admin/backups/run", s.runBackupNow)
	s.admin("GET /api/v1/admin/backups", func(w http.ResponseWriter, r *http.Request) {
		v, e := s.rows(r.Context(), "SELECT to_jsonb(a)-'path' FROM backup_artifacts a ORDER BY created_at DESC LIMIT 1000")
		respond(w, v, e)
	})
	s.admin("GET /api/v1/admin/backups/{id}/download", s.downloadBackupArtifact)
}
func (s *Server) saveBackupPolicy(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Enabled         bool   `json:"enabled"`
		ProviderID      string `json:"provider_id"`
		LocalPath       string `json:"local_path"`
		IntervalMinutes int    `json:"interval_minutes"`
		RetentionCount  int    `json:"retention_count"`
	}
	if decode(r, &in) != nil || in.IntervalMinutes < 5 || in.IntervalMinutes > 525600 || in.RetentionCount < 1 || in.RetentionCount > 1000 {
		apiError(w, 400, "예약 간격은 5~525600분, 보존 개수는 1~1000개입니다")
		return
	}
	if in.ProviderID != "" {
		p, e := s.storageProvider(r.Context(), in.ProviderID)
		if e != nil || !p.Enabled || p.WorkspaceID != "" {
			apiError(w, 400, "전체 백업은 서비스 관리자 소유의 공용 활성 저장소만 사용합니다")
			return
		}
	}
	if !filepath.IsAbs(in.LocalPath) || filepath.Clean(in.LocalPath) != in.LocalPath || in.LocalPath == "/" {
		apiError(w, 400, "백업 전용 로컬 절대 경로를 입력하세요")
		return
	}
	var wid string
	if s.DB.QueryRow(r.Context(), "SELECT workspace_id::text FROM workspace_members WHERE user_id=$1 ORDER BY workspace_id LIMIT 1", current(r).ID).Scan(&wid) != nil {
		apiError(w, 400, "백업 작업 소유자의 워크스페이스가 필요합니다")
		return
	}
	_, e := s.DB.Exec(r.Context(), `UPDATE backup_policy SET enabled=$1,owner_id=$2,workspace_id=$3,provider_id=NULLIF($4,'')::uuid,local_path=$5,interval_minutes=$6,retention_count=$7,next_run=CASE WHEN $1 THEN now()+make_interval(mins=>$6) ELSE NULL END,updated_at=now() WHERE id=1`, in.Enabled, current(r).ID, wid, in.ProviderID, in.LocalPath, in.IntervalMinutes, in.RetentionCount)
	if e == nil {
		s.audit(r, "BACKUP_POLICY", "backup", map[string]any{"enabled": in.Enabled, "interval_minutes": in.IntervalMinutes, "retention_count": in.RetentionCount})
	}
	respond(w, map[string]bool{"ok": true}, e)
}
func (s *Server) runBackupNow(w http.ResponseWriter, r *http.Request) {
	var wid string
	if s.DB.QueryRow(r.Context(), "SELECT workspace_id::text FROM workspace_members WHERE user_id=$1 ORDER BY workspace_id LIMIT 1", current(r).ID).Scan(&wid) != nil {
		apiError(w, 400, "작업 소유자의 워크스페이스가 필요합니다")
		return
	}
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	id, e := s.EnqueueJob(r.Context(), tx, "backup.snapshot", current(r).ID, wid, map[string]any{})
	if e == nil {
		_, e = tx.Exec(r.Context(), "UPDATE automation_jobs SET timeout_seconds=3600,max_attempts=3 WHERE id=$1", id)
	}
	if e == nil {
		e = tx.Commit(r.Context())
	}
	respond(w, map[string]string{"job_id": id}, e)
}
func (s *Server) StartBackupSchedule(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for {
			if ctx.Err() != nil {
				return
			}
			s.scheduleBackups(ctx)
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
}
func (s *Server) scheduleBackups(ctx context.Context) {
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return
	}
	defer tx.Rollback(ctx)
	var owner, wid string
	e = tx.QueryRow(ctx, `SELECT owner_id::text,workspace_id::text FROM backup_policy WHERE id=1 AND enabled AND next_run<=now() AND NOT (SELECT paused FROM job_settings WHERE id=1) FOR UPDATE SKIP LOCKED`).Scan(&owner, &wid)
	if e != nil {
		return
	}
	id, e := s.EnqueueJob(ctx, tx, "backup.snapshot", owner, wid, map[string]any{})
	if e == nil {
		_, e = tx.Exec(ctx, "UPDATE automation_jobs SET timeout_seconds=3600,max_attempts=3 WHERE id=$1", id)
	}
	if e == nil {
		_, e = tx.Exec(ctx, "UPDATE backup_policy SET next_run=now()+make_interval(mins=>interval_minutes) WHERE id=1")
	}
	if e == nil {
		_ = tx.Commit(ctx)
	}
}
func (s *Server) executeScheduledBackup(ctx context.Context, j Job) (map[string]any, error) {
	p, e := s.workerPrincipal(ctx, j.OwnerID, "", j.WorkspaceID)
	if e != nil || p.Role != "admin" || p.Kind != "user" {
		return nil, jobPermanent("전체 백업을 실행할 활성 서비스 관리자가 필요합니다")
	}
	var existing string
	if s.DB.QueryRow(ctx, "SELECT id::text FROM backup_artifacts WHERE job_id=$1", j.ID).Scan(&existing) == nil {
		return map[string]any{"backup_id": existing}, nil
	}
	var providerID, root string
	var retention int
	if e = s.DB.QueryRow(ctx, "SELECT coalesce(provider_id::text,''),local_path,retention_count FROM backup_policy WHERE id=1").Scan(&providerID, &root, &retention); e != nil {
		return nil, e
	}
	provider := storageProvider{Kind: "local", Enabled: true, Config: map[string]any{"root": root}}
	if providerID != "" {
		provider, e = s.storageProvider(ctx, providerID)
		if e != nil || !provider.Enabled || provider.WorkspaceID != "" {
			return nil, jobPermanent("백업 대상 저장소를 다시 확인하세요")
		}
	}
	file, e := s.createBackupFile(ctx)
	if e != nil {
		if errors.Is(e, errBackupTooLarge) {
			return nil, jobPermanent(errBackupTooLarge.Error())
		}
		return nil, errors.New("백업 스냅샷 생성 실패: 첨부파일 무결성 및 저장소 상태를 확인하세요")
	}
	defer file.Close()
	defer os.Remove(file.Name())
	id := newID()
	object, e := s.putStoredObject(ctx, provider, "backups/"+id+".zip", file, backupCompressedLimit, "application/zip")
	if e != nil {
		_ = s.cleanupStoredObject(ctx, object)
		return nil, e
	}
	_, e = s.DB.Exec(ctx, "INSERT INTO backup_artifacts(id,job_id,provider_id,object_key,path,checksum_sha256,size) VALUES($1,$2,NULLIF($3,'')::uuid,$4,$5,$6,$7)", id, j.ID, object.ProviderID, object.Key, object.Path, object.Checksum, object.Size)
	if e != nil {
		_ = s.cleanupStoredObject(ctx, object)
		return nil, e
	}
	r := automationRequest(ctx, p, "POST", nil)
	s.audit(r, "BACKUP_SNAPSHOT", id, map[string]any{"size": object.Size, "checksum_sha256": object.Checksum})
	retainedErrors := s.cleanupBackupArtifacts(ctx, retention)
	return map[string]any{"backup_id": id, "size": object.Size, "checksum_sha256": object.Checksum, "retention_errors": retainedErrors}, nil
}
func (s *Server) cleanupBackupArtifacts(ctx context.Context, retention int) int {
	rows, e := s.rows(ctx, "SELECT to_jsonb(a) FROM backup_artifacts a WHERE managed ORDER BY created_at DESC,id OFFSET $1", retention)
	if e != nil {
		return 1
	}
	failures := 0
	for _, row := range rows {
		o := objectMap(row)
		o.ProviderID = str(row, "provider_id")
		if !validID(str(row, "id")) || o.Key != "backups/"+str(row, "id")+".zip" {
			failures++
			continue
		}
		// Only remove a ledger-owned artifact whose bytes still match the original
		// checksum. Foreign/replaced files are retained and surfaced to the operator.
		file, e := s.materializeObject(ctx, o, 1<<30)
		if e == nil {
			file.Close()
			os.Remove(file.Name())
			e = s.deleteStoredObject(ctx, o)
		}
		if e != nil {
			failures++
			_, _ = s.DB.Exec(ctx, "UPDATE backup_artifacts SET last_error='보존 정리 실패: 저장소 권한·객체 체크섬을 확인하세요' WHERE id=$1", row["id"])
			continue
		}
		_, e = s.DB.Exec(ctx, "DELETE FROM backup_artifacts WHERE id=$1", row["id"])
		if e != nil {
			failures++
		}
	}
	return failures
}
func (s *Server) downloadBackupArtifact(w http.ResponseWriter, r *http.Request) {
	row, e := s.one(r.Context(), "SELECT to_jsonb(a) FROM backup_artifacts a WHERE id=$1", r.PathValue("id"))
	if e != nil {
		apiError(w, 404, "백업을 찾을 수 없습니다")
		return
	}
	object := objectMap(row)
	object.ProviderID = str(row, "provider_id")
	file, e := s.materializeObject(r.Context(), object, 1<<30)
	if e != nil {
		apiError(w, 502, "백업 저장소 읽기 또는 무결성 검사에 실패했습니다")
		return
	}
	defer file.Close()
	defer os.Remove(file.Name())
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", `attachment; filename="madi-backup-`+str(row, "id")+`.zip"`)
	s.audit(r, "BACKUP_DOWNLOAD", str(row, "id"), nil)
	http.ServeContent(w, r, "backup.zip", time.Time{}, file)
}
