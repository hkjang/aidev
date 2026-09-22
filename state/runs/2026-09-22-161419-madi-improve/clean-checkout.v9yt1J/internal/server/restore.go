package server

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/jackc/pgx/v5"
)

// Restore requires a deliberate confirmation in the management UI. A transaction
// restores all relational data, while attachment files are staged under a new UUID
// directory. Existing attachment files are retained for operator recovery.
func (s *Server) restoreBackup(w http.ResponseWriter, r *http.Request) {
	// The upload limit includes a small bounded allowance for multipart framing.
	r.Body = http.MaxBytesReader(w, r.Body, backupCompressedLimit+(1<<20))
	if r.ParseMultipartForm(4<<20) != nil {
		apiError(w, 400, "256MB 이하의 madi 백업 ZIP을 선택하세요")
		return
	}
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}
	if r.FormValue("confirmation") != "RESTORE" {
		apiError(w, 400, "복원 확인란에 RESTORE를 입력하세요")
		return
	}
	input, _, e := r.FormFile("file")
	if e != nil {
		apiError(w, 400, "복원할 백업 파일을 선택하세요")
		return
	}
	defer input.Close()
	data, e := io.ReadAll(io.LimitReader(input, (256<<20)+1))
	if e != nil || len(data) > 256<<20 {
		apiError(w, 400, "백업 파일을 읽을 수 없거나 크기를 초과했습니다")
		return
	}
	archive, e := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if e != nil {
		apiError(w, 400, "올바른 ZIP 파일이 아닙니다")
		return
	}
	entries := map[string]*zip.File{}
	var size uint64
	for _, entry := range archive.File {
		if entry.UncompressedSize64 > (1<<30)-size {
			apiError(w, 400, "압축 해제 크기가 1GB를 초과합니다")
			return
		}
		size += entry.UncompressedSize64
		if size > backupExpandedLimit || len(entries) >= backupEntryLimit {
			apiError(w, 400, "웹 복원은 압축 해제 1GB/10,000개 항목까지 지원합니다. 대규모 복원은 PostgreSQL 백업을 사용하세요")
			return
		}
		if !filepath.IsLocal(entry.Name) || strings.Contains(entry.Name, "\\") || entry.Mode()&os.ModeSymlink != 0 {
			apiError(w, 400, "백업에 안전하지 않은 경로가 있습니다")
			return
		}
		if _, exists := entries[entry.Name]; exists {
			apiError(w, 400, "백업에 중복 경로가 있습니다")
			return
		}
		entries[entry.Name] = entry
	}
	manifest, ok := entries["backup.json"]
	if !ok || manifest.UncompressedSize64 > 256<<20 {
		apiError(w, 400, "madi 백업 정보가 없거나 크기를 초과했습니다")
		return
	}
	reader, e := manifest.Open()
	if e != nil {
		apiError(w, 400, "백업 정보를 읽을 수 없습니다")
		return
	}
	raw, e := io.ReadAll(io.LimitReader(reader, (256<<20)+1))
	reader.Close()
	if e != nil || len(raw) > 256<<20 {
		apiError(w, 400, "백업 정보를 읽을 수 없습니다")
		return
	}
	var backup struct {
		Format  string                     `json:"format"`
		Version string                     `json:"version"`
		Tables  map[string]json.RawMessage `json:"tables"`
	}
	if json.Unmarshal(raw, &backup) != nil || backup.Format != "madi-logical-backup" || backup.Version != s.Version {
		apiError(w, 400, "현재 서비스와 같은 버전의 madi 논리 백업이 필요합니다")
		return
	}
	tables := backupTables
	for _, table := range tables {
		if b, exists := backup.Tables[table]; !exists || len(b) == 0 || b[0] != '[' {
			apiError(w, 400, "백업 테이블이 누락되었습니다: "+table)
			return
		}
	}
	// Secrets in workspace settings, history and webhook modules use the same
	// master key. Check them before staging files or changing live data.
	for _, table := range []string{"settings_history", "workspace_settings", "workspace_settings_history", "automation_webhooks", "storage_providers"} {
		var entries []map[string]any
		if json.Unmarshal(backup.Tables[table], &entries) != nil {
			apiError(w, 400, "백업 테이블 형식을 확인하세요: "+table)
			return
		}
		for _, entry := range entries {
			secrets := []string{}
			if table == "storage_providers" {
				config, ok := entry["config"].(map[string]any)
				if !ok {
					apiError(w, 400, "백업 저장소 설정 형식을 확인하세요")
					return
				}
				for _, key := range storageSecrets {
					if value := str(config, key); value != "" {
						secrets = append(secrets, value)
					}
				}
			} else if table == "automation_webhooks" {
				secrets = append(secrets, str(entry, "secret_ciphertext"))
			} else {
				data, _ := entry["data"].(map[string]any)
				for _, key := range secretSettings {
					if value := str(data, key); value != "" {
						secrets = append(secrets, value)
					}
				}
			}
			for _, secret := range secrets {
				if _, err := s.decrypt(secret); err != nil {
					apiError(w, 400, "백업 생성 당시의 ENCRYPTION_KEY가 필요합니다")
					return
				}
			}
		}
	}
	var configs, users, files []map[string]any
	if json.Unmarshal(backup.Tables["settings"], &configs) != nil || len(configs) != 1 || json.Unmarshal(backup.Tables["users"], &users) != nil || json.Unmarshal(backup.Tables["attachments"], &files) != nil {
		apiError(w, 400, "백업 구조를 확인하세요")
		return
	}
	targetConfig, e := s.settings(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	stored, ok := configs[0]["data"].(map[string]any)
	if !ok {
		apiError(w, 400, "백업 설정 형식이 잘못되었습니다")
		return
	}
	validatedConfig := defaultSettings()
	for key, value := range stored {
		validatedConfig[key] = value
	}
	for _, key := range secretSettings {
		if value, exists := stored[key]; exists {
			if _, ok := value.(string); !ok {
				apiError(w, 400, "백업 비밀 설정의 형식이 올바르지 않습니다")
				return
			}
		}
		if value := str(stored, key); value != "" {
			if validatedConfig[key], e = s.decrypt(value); e != nil {
				apiError(w, 400, "백업 생성 당시의 ENCRYPTION_KEY가 필요합니다")
				return
			}
		}
	}
	validatedConfig["storage_path"] = str(targetConfig, "storage_path")
	if e = validateSettings(validatedConfig); e != nil {
		apiError(w, 400, "백업의 운영 설정이 올바르지 않습니다: "+e.Error())
		return
	}
	var adminID string
	var activeAdmins int
	for _, u := range users {
		if str(u, "role") == "admin" && !boolean(u, "disabled") && str(u, "kind") == "user" {
			activeAdmins++
			if str(u, "email") == current(r).Email {
				adminID = str(u, "id")
			}
		}
	}
	if activeAdmins == 0 {
		apiError(w, 400, "백업에 활성 관리자 계정이 없습니다")
		return
	}
	for _, file := range files {
		fid := str(file, "id")
		if !validID(fid) {
			apiError(w, 400, "백업의 첨부파일 ID가 잘못되었습니다")
			return
		}
		entry, exists := entries["attachments/"+fid]
		if !exists || entry.UncompressedSize64 > 50<<20 {
			apiError(w, 400, "백업의 첨부파일이 없거나 크기 제한을 초과했습니다")
			return
		}
	}
	stage := filepath.Join(str(targetConfig, "storage_path"), "restore-"+newID())
	if e = os.MkdirAll(stage, 0700); e != nil {
		apiError(w, 500, "첨부파일 저장 경로의 쓰기 권한을 확인하세요")
		return
	}
	staged := []string{}
	committed := false
	defer func() {
		if !committed {
			for _, file := range staged {
				os.Remove(file)
			}
			os.Remove(stage)
		}
	}()
	for _, file := range files {
		fid := str(file, "id")
		target := filepath.Join(stage, fid)
		src, err := entries["attachments/"+fid].Open()
		if err != nil {
			apiError(w, 400, "첨부파일을 읽을 수 없습니다")
			return
		}
		dst, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err != nil {
			src.Close()
			respond(w, nil, err)
			return
		}
		staged = append(staged, target)
		n, err := io.Copy(dst, io.LimitReader(src, (50<<20)+1))
		closeErr := dst.Close()
		src.Close()
		if err == nil {
			err = closeErr
		}
		if err != nil || n > 50<<20 {
			apiError(w, 400, "첨부파일 복원에 실패했습니다")
			return
		}
		file["path"] = target
		file["size"] = n
		file["storage_provider_id"] = nil
		file["object_key"] = ""
		if hash := str(file, "checksum_sha256"); hash != "" {
			check, err := s.materializeObject(r.Context(), objectMap(file), 50<<20)
			if err != nil {
				apiError(w, 400, "복원 첨부파일 SHA-256 검증에 실패했습니다")
				return
			}
			check.Close()
			os.Remove(check.Name())
		}
	}
	stored["storage_path"] = str(targetConfig, "storage_path")
	backup.Tables["settings"] = jsonValue(configs)
	backup.Tables["attachments"] = jsonValue(files)
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	if _, e = tx.Exec(r.Context(), "SELECT pg_advisory_xact_lock(726234801)"); e != nil {
		respond(w, nil, e)
		return
	}
	// These exact table names are application constants; archive content never forms SQL identifiers.
	truncateNames := []string{}
	for _, table := range append(append([]string{}, backupTables...), ephemeralTables...) {
		truncateNames = append(truncateNames, pgx.Identifier{table}.Sanitize())
	}
	if _, e = tx.Exec(r.Context(), "TRUNCATE "+strings.Join(truncateNames, ",")); e != nil {
		respond(w, nil, e)
		return
	}
	for _, table := range tables {
		rows, err := tx.Query(r.Context(), "SELECT attname FROM pg_attribute WHERE attrelid=$1::regclass AND attnum>0 AND NOT attisdropped AND attgenerated='' ORDER BY attnum", table)
		if err != nil {
			respond(w, nil, err)
			return
		}
		cols := []string{}
		for rows.Next() {
			var name string
			if err = rows.Scan(&name); err != nil {
				break
			}
			cols = append(cols, pgx.Identifier{name}.Sanitize())
		}
		rows.Close()
		if err == nil {
			err = rows.Err()
		}
		if err != nil {
			respond(w, nil, err)
			return
		}
		columns := strings.Join(cols, ",")
		query := fmt.Sprintf("INSERT INTO %s (%s) SELECT %s FROM jsonb_populate_recordset(NULL::%s,$1::jsonb)", pgx.Identifier{table}.Sanitize(), columns, columns, pgx.Identifier{table}.Sanitize())
		if _, err = tx.Exec(r.Context(), query, []byte(backup.Tables[table])); err != nil {
			apiError(w, 400, "백업 데이터 검증에 실패했습니다. 기존 데이터는 유지됩니다. 테이블: "+table)
			return
		}
	}
	// Restoring history must not redeliver webhooks or repeat external actions.
	if _, e = tx.Exec(r.Context(), `UPDATE job_settings SET paused=true; UPDATE automation_rules SET enabled=false; UPDATE automation_webhooks SET enabled=false; UPDATE automation_jobs SET status='cancelled',cancel_requested=true,lease_id=NULL,lease_until=NULL,last_error='백업 복원으로 중단되었습니다. 관리자가 작업을 확인하세요',finished_at=now() WHERE status IN ('pending','running'); SELECT setval(pg_get_serial_sequence('automation_job_attempts','id'),coalesce((SELECT max(id) FROM automation_job_attempts),1),(SELECT count(*)>0 FROM automation_job_attempts))`); e != nil {
		respond(w, nil, e)
		return
	}
	if _, e = tx.Exec(r.Context(), `UPDATE backup_policy SET enabled=false,next_run=NULL; UPDATE storage_providers SET enabled=false; UPDATE storage_settings SET provider_id=NULL; UPDATE storage_assignments SET provider_id=NULL; UPDATE backup_artifacts SET managed=false,last_error='복원 전 보관소의 이력입니다. 자동 보존 정리에서 제외됩니다'; UPDATE workspace_settings SET data=jsonb_set(data,'{lifecycle_enabled}','false'::jsonb,true)`); e != nil {
		respond(w, nil, e)
		return
	}
	if _, e = tx.Exec(r.Context(), `UPDATE migration_imports SET status='cancelled',source_data=NULL WHERE status<>'completed'; UPDATE connector_settings SET enabled=false; UPDATE connector_configs SET enabled=false,next_run=NULL; UPDATE sql_sources SET enabled=false`); e != nil {
		respond(w, nil, e)
		return
	}
	if _, e = tx.Exec(r.Context(), `UPDATE approval_policy_clock SET revision=revision+1 WHERE id=1; UPDATE approval_requests SET status='superseded',version=version+1,completed_at=now(),updated_at=now() WHERE status='pending' OR (resource_kind<>'document' AND status='approved'); UPDATE sql_source_queries SET enabled=false WHERE origin_proposal_id IS NOT NULL; UPDATE sql_source_proposals SET status='cancelled',updated_at=now() WHERE status IN ('proposed','review') OR (approval_id IS NOT NULL AND status='approved')`); e != nil {
		respond(w, nil, e)
		return
	}
	if _, e = tx.Exec(r.Context(), `UPDATE notification_settings SET enabled=false; UPDATE notification_channels SET enabled=false; UPDATE notification_outbox SET processed_at=now() WHERE processed_at IS NULL; UPDATE notification_deliveries SET status='skipped',message='백업 복원으로 전송을 중단했습니다',updated_at=now() WHERE status IN ('pending','sending','failed')`); e != nil {
		respond(w, nil, e)
		return
	}
	if _, e = tx.Exec(r.Context(), `UPDATE inbound_capture_settings SET hooks_enabled=false,imap_enabled=false; UPDATE inbound_capture_channels SET enabled=false,next_run=NULL; UPDATE inbound_capture_messages SET status='cancelled',message='백업 복원으로 수집을 중단했습니다',completed_at=now() WHERE status='pending'`); e != nil {
		respond(w, nil, e)
		return
	}
	if _, e = tx.Exec(r.Context(), `UPDATE runbook_settings SET enabled=false,revision=revision+1; UPDATE runbook_runners SET enabled=false; UPDATE runbook_executions SET status='cancelled',cancel_requested=false,completed_at=now(),updated_at=now(),last_error='백업 복원으로 실행을 중단했습니다' WHERE status IN ('prepared','review','approved','rejected'); UPDATE runbook_executions SET status='unknown',cancel_requested=false,reconcile_after=now(),updated_at=now(),last_error='복원 이전 외부 실행을 운영자가 확인해야 합니다' WHERE status IN ('queued','running'); UPDATE runbook_executions SET cancel_requested=false WHERE status='unknown'; UPDATE runbook_execution_steps SET state='unknown' WHERE state IN ('launching','running'); UPDATE runbook_execution_steps SET state='cancelled',completed_at=now() WHERE state='queued'; SELECT setval(pg_get_serial_sequence('runbook_events','id'),coalesce((SELECT max(id) FROM runbook_events),1),(SELECT count(*)>0 FROM runbook_events))`); e != nil {
		respond(w, nil, e)
		return
	}
	if _, e = tx.Exec(r.Context(), `UPDATE protection_settings SET data=jsonb_set(data,'{public_shares_enabled}','false'::jsonb,true),revision=revision+1; UPDATE public_shares SET revoked_at=coalesce(revoked_at,now()),revision=revision+1`); e != nil {
		respond(w, nil, e)
		return
	}
	if _, e = tx.Exec(r.Context(), `DO $$ BEGIN IF to_regclass('rag_index_grants') IS NOT NULL THEN UPDATE rag_index_grants SET active=false,auto_reindex=false,revision=revision+1,last_job_id=NULL; DELETE FROM rag_vector_chunks; DELETE FROM rag_vector_indexes; DELETE FROM rag_reindex_queue; END IF; END $$;`); e != nil {
		respond(w, nil, e)
		return
	}
	if _, e = tx.Exec(r.Context(), `DO $$ BEGIN IF to_regclass('rag_generations') IS NOT NULL THEN UPDATE rag_generations SET status='disabled',revision=revision+1,index_job_id=NULL; UPDATE rag_generation_state SET active_id=NULL,revision=revision+1,mode='exact'; UPDATE rag_generation_validations SET session_hash='',expires_at=now(),consumed_at=coalesce(consumed_at,now()); END IF; END $$;`); e != nil {
		respond(w, nil, e)
		return
	}
	if _, e = tx.Exec(r.Context(), `DO $$ BEGIN IF to_regclass('git_sync_settings') IS NOT NULL THEN UPDATE git_sync_settings SET data=jsonb_set(data,'{enabled}','false'::jsonb,true),revision=revision+1; UPDATE git_sync_connections SET enabled=false,revision=revision+1; UPDATE git_sync_runs SET status=CASE WHEN status='running' THEN 'unknown' ELSE 'cancelled' END,snapshot_cipher=NULL,snapshot_size=0,revision=revision+1 WHERE status IN ('preparing','preview','queued','running'); END IF; END $$;`); e != nil {
		respond(w, nil, e)
		return
	}
	if _, e = tx.Exec(r.Context(), `UPDATE workspace_agents SET enabled=false,revision=revision+1; UPDATE agent_runs SET status='cancelled',session_hash='',error='백업 복원으로 실행을 취소했습니다',updated_at=now() WHERE status IN ('pending','running','awaiting_confirmation'); UPDATE agent_actions SET status='cancelled' WHERE status IN ('planned','confirmed'); SELECT setval(pg_get_serial_sequence('agent_events','id'),coalesce((SELECT max(id) FROM agent_events),1),(SELECT count(*)>0 FROM agent_events))`); e != nil {
		respond(w, nil, e)
		return
	}
	if _, e = tx.Exec(r.Context(), `UPDATE support_sessions SET status='revoked',ended_at=now(),session_hash='' WHERE status='active'; UPDATE search_history_preferences SET enabled=false,revision=revision+1; UPDATE settings SET data=jsonb_set(jsonb_set(data,'{support_enabled}','false'::jsonb,true),'{otel_enabled}','false'::jsonb,true) WHERE id=1`); e != nil {
		respond(w, nil, e)
		return
	}
	if _, e = tx.Exec(r.Context(), `UPDATE graph_ai_runs SET status='cancelled',finished_at=now(),error='백업 복원으로 분석이 중단되었습니다' WHERE status='running'; UPDATE graph_ai_runs SET session_hash=''; UPDATE graph_ai_actions SET status='cancelled',decided_at=now() WHERE status='proposed'`); e != nil {
		respond(w, nil, e)
		return
	}
	if _, e = tx.Exec(r.Context(), `UPDATE attachment_extraction_settings SET data=data||'{"enabled":false,"ocr_enabled":false}'::jsonb,revision=revision+1; INSERT INTO attachment_extraction_policy_history(revision,data) SELECT revision,data FROM attachment_extraction_settings WHERE id=1;
UPDATE attachment_extractions SET status=CASE WHEN status IN ('queued','running','ready') THEN 'obsolete' ELSE status END,session_hash='',token_hash='',temp_path='',revision=revision+1; DELETE FROM attachment_extraction_heads; DELETE FROM attachment_extraction_fragments;
UPDATE export_settings SET enabled=false,revision=revision+1; UPDATE export_runs SET status=CASE WHEN status='ready' THEN 'expired' ELSE 'cancelled' END,session_hash='',updated_at=now() WHERE status IN ('queued','running','ready'); DELETE FROM export_artifact_blobs;
UPDATE knowledge_distribution_policy SET enabled=false,revision=revision+1; INSERT INTO knowledge_distribution_policy_history(revision,enabled,max_valid_days) SELECT revision,enabled,max_valid_days FROM knowledge_distribution_policy WHERE id=1;
UPDATE knowledge_distribution_keys SET revoked_at=coalesce(revoked_at,now()),revision=revision+1;
UPDATE knowledge_distribution_exports SET status='expired' WHERE status IN ('queued','awaiting_review','ready'); DELETE FROM knowledge_distribution_artifacts;`); e != nil {
		respond(w, nil, e)
		return
	}
	if e = invalidateCollaborationRestoreTx(r.Context(), tx); e != nil {
		respond(w, nil, e)
		return
	}
	if e = s.invalidateSystemStatusRestoreTx(r.Context(), tx); e != nil {
		respond(w, nil, e)
		return
	}
	if e = invalidateImpactExceptionsRestoreTx(r.Context(), tx); e != nil {
		respond(w, nil, e)
		return
	}
	if _, e = tx.Exec(r.Context(), `UPDATE knowledge_questions SET state='proposed',revision=revision+1,confirmed_by=NULL,confirmed_at=NULL,updated_at=now() WHERE state='confirmed'`); e != nil {
		respond(w, nil, e)
		return
	}
	if _, e = tx.Exec(r.Context(), `UPDATE knowledge_structured_drafts SET state='expired',revision=revision+1,expires_at=now() WHERE state='draft'`); e != nil {
		respond(w, nil, e)
		return
	}
	if _, e = tx.Exec(r.Context(), `UPDATE knowledge_path_progress SET state='needs_recheck',revision=revision+1,updated_at=now(),approval_id=NULL WHERE state IN ('confirmed','pending_review','approved')`); e != nil {
		respond(w, nil, e)
		return
	}
	if _, e = tx.Exec(r.Context(), `UPDATE knowledge_evidence_policy SET enabled=false,version=version+1; UPDATE knowledge_package_policy SET enabled=false,version=version+1; INSERT INTO knowledge_evidence_policy_history(version,enabled,retention_days) SELECT version,enabled,retention_days FROM knowledge_evidence_policy WHERE id=1; INSERT INTO knowledge_package_policy_history(version,enabled,retention_hours,token_counter,allow_http) SELECT version,enabled,retention_hours,token_counter,allow_http FROM knowledge_package_policy WHERE id=1`); e != nil {
		respond(w, nil, e)
		return
	}
	if _, e = tx.Exec(r.Context(), `UPDATE migration_sessions SET status=CASE WHEN status IN ('uploading','preparing','ready','committing') THEN 'cancelled' ELSE status END,request_session_hash='',request_token_hash='',revision=revision+1,updated_at=now(),purged_at=now(); DELETE FROM migration_session_chunks; UPDATE migration_session_items SET prepared_data=NULL,staged_object='{}',storage_fingerprint=''; UPDATE migration_session_objects SET status='discarded';`); e != nil {
		respond(w, nil, e)
		return
	}
	if e = tx.Commit(r.Context()); e != nil {
		respond(w, nil, e)
		return
	}
	committed = true
	if adminID != "" {
		if e = s.createSession(w, r, adminID); e != nil {
			respond(w, nil, e)
			return
		}
		p := *current(r)
		p.ID = adminID
		r = r.WithContext(context.WithValue(r.Context(), principalKey, &p))
	} else {
		r = r.WithContext(context.WithValue(r.Context(), principalKey, (*Principal)(nil)))
	}
	s.audit(r, "BACKUP_RESTORE", "system", map[string]any{"version": backup.Version, "attachments": len(files), "previous_attachments_retained": true})
	jsonResponse(w, 200, map[string]any{"ok": true, "relogin_required": adminID == "", "message": "백업을 복원했습니다. 기존 세션은 만료되고 작업 큐·자동화는 중지되었습니다. 관리자가 확인한 뒤 다시 활성화하세요. 이전 첨부파일은 복구 목적으로 보존됩니다."})
}
