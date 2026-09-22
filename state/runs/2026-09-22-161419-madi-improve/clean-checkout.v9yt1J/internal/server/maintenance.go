package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

type maintenanceResult struct {
	Documents       int64    `json:"documents_deleted"`
	Sessions        int64    `json:"sessions_deleted"`
	OIDCAttempts    int64    `json:"oidc_attempts_deleted"`
	SAMLAttempts    int64    `json:"saml_attempts_deleted"`
	SAMLReplays     int64    `json:"saml_replays_deleted"`
	ParentsDetached int64    `json:"parents_detached"`
	ParentsRetained int64    `json:"parents_retained"`
	Files           int      `json:"files_deleted"`
	FileErrors      []string `json:"file_errors"`
	Skipped         bool     `json:"skipped"`
	Committed       bool     `json:"committed"`
}

// StartMaintenance runs once at startup and then hourly. The caller owns ctx and
// cancels it during graceful shutdown; no extra deployment setting is required.
func (s *Server) StartMaintenance(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()
		for {
			if ctx.Err() != nil {
				return
			}
			batchCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
			result, err := s.runMaintenance(batchCtx)
			cancel()
			if err != nil {
				slog.Error("maintenance failed", "error", err)
			} else if result.Documents > 0 || result.Sessions > 0 || result.OIDCAttempts > 0 || len(result.FileErrors) > 0 {
				slog.Info("maintenance completed", "documents", result.Documents, "sessions", result.Sessions, "oidc_attempts", result.OIDCAttempts, "files", result.Files, "file_errors", len(result.FileErrors))
			}
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
}

func (s *Server) recordMaintenance(ctx context.Context, result maintenanceResult, err error) {
	if err == nil && (result.Skipped || (result.Documents == 0 && result.Sessions == 0 && result.OIDCAttempts == 0)) {
		return
	}
	// A canceled HTTP/server context must not prevent the failure record itself.
	auditCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	r, _ := http.NewRequestWithContext(auditCtx, http.MethodPost, "http://madi.internal/maintenance", nil)
	details := map[string]any{"result": result}
	action := "MAINTENANCE"
	if err != nil {
		action = "MAINTENANCE_ERROR"
		details["error"] = err.Error()
	}
	if len(result.FileErrors) > 0 {
		action = "MAINTENANCE_ERROR"
	}
	s.audit(r, action, "retention", details)
}

func (s *Server) runMaintenance(ctx context.Context) (result maintenanceResult, err error) {
	result.FileErrors = []string{}
	defer func() { s.recordMaintenance(ctx, result, err) }()
	if err = s.expireSupportSessions(ctx); err != nil {
		return result, err
	}
	if err = s.expireSearchHistory(ctx); err != nil {
		return result, err
	}
	if err = s.expireGraphAI(ctx); err != nil {
		return result, err
	}
	if err = s.purgeEvidence(ctx); err != nil {
		return result, err
	}
	if _, err = s.DB.Exec(ctx, `DELETE FROM knowledge_structured_drafts WHERE id IN (SELECT x.id FROM knowledge_structured_drafts x WHERE x.state<>'committed' AND x.expires_at<=now() AND NOT EXISTS(SELECT 1 FROM knowledge_document_meta m WHERE m.document_id=x.document_id AND (m.legal_hold OR m.retain_until>now())) ORDER BY x.expires_at LIMIT 500)`); err != nil {
		return result, err
	}
	if _, err = s.expireSystemStatusReports(ctx); err != nil {
		return result, err
	}
	if _, err = s.DB.Exec(ctx, `DELETE FROM knowledge_packages WHERE id IN (SELECT p.id FROM knowledge_packages p WHERE expires_at<=now() AND NOT EXISTS(SELECT 1 FROM jsonb_to_recordset(p.source_refs) AS ref(id uuid) JOIN knowledge_document_meta m ON m.document_id=ref.id WHERE m.legal_hold OR m.retain_until>now()) ORDER BY expires_at LIMIT 500)`); err != nil {
		return result, err
	}
	settings, err := s.settings(ctx)
	if err != nil {
		return result, err
	}
	retention := settingInt(settings, "trash_retention_days", 30)
	if retention < 1 || retention > 3650 {
		return result, errors.New("휴지통 보존 기간이 유효하지 않아 유지관리를 중단했습니다")
	}
	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return result, err
	}
	defer tx.Rollback(context.WithoutCancel(ctx))
	// Shared with startup migrations and full restore; another replica simply skips this hour.
	var locked bool
	if err = tx.QueryRow(ctx, "SELECT pg_try_advisory_xact_lock(726234801)").Scan(&locked); err != nil {
		return result, err
	}
	if !locked {
		result.Skipped = true
		return result, nil
	}
	if err = tx.QueryRow(ctx, `SELECT count(*) FROM documents d WHERE deleted_at<now()-make_interval(days=>$1) AND EXISTS(SELECT 1 FROM documents child WHERE child.parent_id=d.id)`, retention).Scan(&result.ParentsRetained); err != nil {
		return result, err
	}
	// Keep ancestors with children in trash: detaching a child would silently
	// discard inherited ACL restrictions, even if the child became "private".
	rows, err := tx.Query(ctx, `SELECT id::text FROM documents d WHERE deleted_at < now()-make_interval(days => $1) AND NOT EXISTS(SELECT 1 FROM documents child WHERE child.parent_id=d.id) AND NOT EXISTS(SELECT 1 FROM knowledge_document_meta m WHERE m.document_id=d.id AND (m.legal_hold OR m.retain_until>now())) ORDER BY deleted_at,id LIMIT 500 FOR UPDATE SKIP LOCKED`, retention)
	if err != nil {
		return result, err
	}
	ids := []string{}
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			break
		}
		ids = append(ids, id)
	}
	if err == nil {
		err = rows.Err()
	}
	rows.Close()
	if err != nil {
		return result, err
	}
	type attachmentFile struct {
		id     string
		object storedObject
	}
	files := []attachmentFile{}
	if len(ids) > 0 {
		rows, err = tx.Query(ctx, `SELECT id::text,path,coalesce(storage_provider_id::text,''),object_key,checksum_sha256,size FROM attachments WHERE document_id=ANY($1::uuid[]) FOR UPDATE`, ids)
		if err != nil {
			return result, err
		}
		for rows.Next() {
			var file attachmentFile
			if err = rows.Scan(&file.id, &file.object.Path, &file.object.ProviderID, &file.object.Key, &file.object.Checksum, &file.object.Size); err != nil {
				break
			}
			files = append(files, file)
		}
		if err == nil {
			err = rows.Err()
		}
		rows.Close()
		if err != nil {
			return result, err
		}
		for _, table := range []string{"notifications", "favorites", "comments", "document_shares", "document_versions", "attachments"} {
			if _, err = tx.Exec(ctx, "DELETE FROM "+table+" WHERE document_id=ANY($1::uuid[])", ids); err != nil {
				return result, err
			}
		}
		tag, e := tx.Exec(ctx, `DELETE FROM documents WHERE id=ANY($1::uuid[])`, ids)
		if e != nil {
			return result, e
		}
		result.Documents = tag.RowsAffected()
	}
	tag, err := tx.Exec(ctx, `DELETE FROM sessions WHERE expires_at<=now()`)
	if err != nil {
		return result, err
	}
	result.Sessions = tag.RowsAffected()
	tag, err = tx.Exec(ctx, `DELETE FROM oidc_attempts WHERE expires_at<=now()`)
	if err != nil {
		return result, err
	}
	result.OIDCAttempts = tag.RowsAffected()
	tag, err = tx.Exec(ctx, `DELETE FROM saml_authn_requests WHERE expires_at<=now()`)
	if err != nil {
		return result, err
	}
	result.SAMLAttempts = tag.RowsAffected()
	tag, err = tx.Exec(ctx, `DELETE FROM saml_assertion_replays WHERE expires_at<=now()`)
	if err != nil {
		return result, err
	}
	result.SAMLReplays = tag.RowsAffected()
	if _, err = tx.Exec(ctx, `DELETE FROM public_share_access WHERE expires_at<=now(); DELETE FROM public_share_rate_limits WHERE expires_at<=now()`); err != nil {
		return result, err
	}
	if err = tx.Commit(ctx); err != nil {
		return result, err
	}
	result.Committed = true
	// Filesystem effects only begin after the relational transaction has committed.
	for _, file := range files {
		if ctx.Err() != nil {
			result.FileErrors = append(result.FileErrors, "종료 중이므로 남은 첨부파일 정리를 보류했습니다")
			break
		}
		if !validID(file.id) {
			result.FileErrors = append(result.FileErrors, "안전하지 않은 저장 경로: "+file.id)
			continue
		}
		var referenced bool
		if e := s.DB.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM attachments WHERE (storage_provider_id=NULLIF($1,'')::uuid AND object_key=$2) OR ($1='' AND path=$3))`, file.object.ProviderID, file.object.Key, file.object.Path).Scan(&referenced); e != nil {
			result.FileErrors = append(result.FileErrors, "첨부파일 참조 확인 실패: "+file.id)
			continue
		}
		if referenced {
			continue
		}
		if e := s.deleteStoredObject(ctx, file.object); e != nil {
			result.FileErrors = append(result.FileErrors, fmt.Sprintf("첨부파일 삭제 실패 %s: %v", file.id, e))
		} else {
			result.Files++
		}
	}
	return result, nil
}
