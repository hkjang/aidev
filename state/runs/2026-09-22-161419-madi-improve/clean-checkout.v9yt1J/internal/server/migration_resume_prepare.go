package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"path"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
)

func (s *Server) migrationSessionJob(ctx context.Context, j Job) (migrationSession, *Principal, error) {
	v, e := scanMigrationSession(s.DB.QueryRow(ctx, "SELECT "+migrationSessionSelect+" FROM migration_sessions WHERE id=$1 AND expires_at>now()", str(j.Payload, "session_id")))
	if e != nil {
		return v, nil, jobPermanent("이관 세션이 없거나 보관 기간이 만료되었습니다")
	}
	p, e := s.workerPrincipal(ctx, j.ActorID, j.TokenID, j.WorkspaceID)
	if e != nil {
		return v, nil, e
	}
	if v.UserID != j.OwnerID || v.WorkspaceID != j.WorkspaceID || v.JobID != j.ID || v.Status == "cancelled" || !s.migrationSessionAllowed(ctx, p, v) {
		return v, nil, jobPermanent("이관 작업의 현재 권한이 없거나 취소되었습니다")
	}
	if v.Status == "committing" {
		tx, err := s.DB.Begin(ctx)
		if err != nil {
			return v, nil, err
		}
		err = s.migrationActorTx(ctx, tx, p, v, false)
		_ = tx.Rollback(ctx)
		if err != nil {
			return v, nil, jobPermanent("확정 요청의 세션·키 또는 현재 권한이 만료되었습니다")
		}
	}
	return v, p, nil
}

func (s *Server) migrationSessionIndex(ctx context.Context, id string) ([]migrationSessionItem, error) {
	selectMeta := strings.Replace(migrationItemSelect, "prepared_data", "NULL::bytea", 1)
	rows, e := s.DB.Query(ctx, "SELECT "+selectMeta+" FROM migration_session_items WHERE session_id=$1 ORDER BY file_path,id", id)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	items := []migrationSessionItem{}
	for rows.Next() {
		v, e := scanMigrationItem(rows)
		if e != nil {
			return nil, e
		}
		items = append(items, v)
		if len(items) > 100000 {
			return nil, errors.New("이관 항목 수 한도를 초과했습니다")
		}
	}
	return items, rows.Err()
}

func (s *Server) beginMigrationSessionPrepare(w http.ResponseWriter, r *http.Request) {
	v, ok := s.migrationSessionRequest(w, r)
	if !ok {
		return
	}
	var in struct {
		Revision int64 `json:"revision"`
	}
	if decode(r, &in) != nil {
		apiError(w, 400, "현재 이관 revision을 확인하세요")
		return
	}
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	v, e = scanMigrationSession(tx.QueryRow(r.Context(), "SELECT "+migrationSessionSelect+" FROM migration_sessions WHERE id=$1 AND expires_at>now() FOR UPDATE", v.ID))
	if e != nil || !oneOf(v.Status, "uploading", "failed") || v.Revision != in.Revision {
		apiError(w, 409, "이관 상태가 변경되었습니다. 현재 체크포인트를 다시 확인하세요")
		return
	}
	var pending int
	e = tx.QueryRow(r.Context(), "SELECT count(*) FROM migration_session_items WHERE session_id=$1 AND received_bytes<>source_bytes", v.ID).Scan(&pending)
	if e != nil {
		respond(w, nil, e)
		return
	}
	if v.ItemCount == 0 || pending > 0 {
		apiError(w, 409, "모든 원본 청크를 받은 뒤 준비할 수 있습니다")
		return
	}
	// Automatic lease retry preserves prepared checkpoints. An explicit retry
	// after a terminal validation failure rechecks the complete source binding.
	if v.Status == "failed" {
		var staged bool
		if e = tx.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM migration_session_items WHERE session_id=$1 AND staged_object<>'{}' UNION ALL SELECT 1 FROM migration_session_objects o JOIN migration_session_items i ON i.id=o.item_id WHERE i.session_id=$1 AND o.status IN('planned','ready'))`, v.ID).Scan(&staged); e != nil {
			respond(w, nil, e)
			return
		}
		if staged {
			// Preserve reviewed canonical bytes and immutable object identities after
			// an exhausted transport retry. A changed policy still fails closed on
			// the next final validation; cancel/re-upload to create a different plan.
			report, hash, err := migrationSessionPlanTx(r.Context(), tx, v.ID)
			if err != nil {
				apiError(w, 409, err.Error())
				return
			}
			_, e = tx.Exec(r.Context(), `UPDATE migration_sessions SET status='ready',report=$2,plan_hash=$3,revision=revision+1,error='',request_session_hash='',request_token_hash='',updated_at=now() WHERE id=$1`, v.ID, jsonValue(report), hash)
			if e == nil {
				e = tx.Commit(r.Context())
			}
			respond(w, map[string]string{"status": "ready"}, e)
			return
		}
		_, e = tx.Exec(r.Context(), "UPDATE migration_session_items SET checkpoint=0,status='pending',prepared_data=NULL,error='',types_confirmed=false WHERE session_id=$1", v.ID)
		if e != nil {
			respond(w, nil, e)
			return
		}
		if _, e = tx.Exec(r.Context(), "UPDATE migration_sessions SET prepared_count=0 WHERE id=$1", v.ID); e != nil {
			respond(w, nil, e)
			return
		}
	}
	_, e = tx.Exec(r.Context(), "UPDATE migration_session_items SET status='pending',error='' WHERE session_id=$1 AND status='failed'", v.ID)
	jobID := ""
	if e == nil {
		jobID, e = s.EnqueueJob(r.Context(), tx, "migration.session.prepare", v.UserID, v.WorkspaceID, map[string]any{"session_id": v.ID})
	}
	if e == nil {
		_, e = tx.Exec(r.Context(), "UPDATE migration_sessions SET status='preparing',job_id=$2,error='',updated_at=now() WHERE id=$1", v.ID, jobID)
	}
	if e == nil {
		_, e = tx.Exec(r.Context(), "UPDATE automation_jobs SET timeout_seconds=3600,max_attempts=5 WHERE id=$1", jobID)
	}
	if e == nil {
		e = tx.Commit(r.Context())
	}
	respond(w, map[string]string{"job_id": jobID}, e)
}

func (s *Server) ensureMigrationFolders(ctx context.Context, v migrationSession) error {
	items, e := s.migrationSessionIndex(ctx, v.ID)
	if e != nil {
		return e
	}
	byPath := map[string]migrationSessionItem{}
	folders := map[string]bool{}
	attachments := false
	for _, i := range items {
		byPath[i.FilePath] = i
		if i.Kind == "attachment" {
			attachments = true
		}
		for d := path.Dir(i.FilePath); d != "." && d != ""; d = path.Dir(d) {
			folders[d] = true
		}
	}
	paths := []string{}
	for d := range folders {
		if byPath[d+".md"].ID == "" && byPath[d+".markdown"].ID == "" {
			paths = append(paths, d)
		}
	}
	sort.Strings(paths)
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(ctx)
	var status string
	if e = tx.QueryRow(ctx, "SELECT status FROM migration_sessions WHERE id=$1 FOR UPDATE", v.ID).Scan(&status); e != nil {
		return e
	}
	if status != "preparing" {
		return jobPermanent("이관 준비가 취소되었습니다")
	}
	empty := digest("")
	for _, d := range paths {
		_, e = tx.Exec(ctx, `INSERT INTO migration_session_items(id,session_id,source_id,source_hash,file_path,kind,source_bytes,chunk_count,target_id,status) VALUES($1,$2,$3,$4,$5,'folder',0,0,$6,'pending') ON CONFLICT(session_id,source_id) DO NOTHING`, newID(), v.ID, migrationFolderSource(d), empty, d+"/.madi-folder.md", newID())
		if e != nil {
			return e
		}
	}
	if attachments {
		_, e = tx.Exec(ctx, `INSERT INTO migration_session_items(id,session_id,source_id,source_hash,file_path,kind,source_bytes,chunk_count,target_id,status,metadata) VALUES($1,$2,'@madi-folder:attachment-root',$3,'.madi-attachments-root.md','folder',0,0,$4,'pending','{"attachment_root":true}') ON CONFLICT(session_id,source_id) DO NOTHING`, newID(), v.ID, empty, newID())
		if e != nil {
			return e
		}
	}
	var count, maxItems int
	if e = tx.QueryRow(ctx, "SELECT count(*) FROM migration_session_items WHERE session_id=$1", v.ID).Scan(&count); e != nil {
		return e
	}
	if e = tx.QueryRow(ctx, "SELECT max_items FROM migration_settings WHERE id=1 FOR SHARE").Scan(&maxItems); e != nil {
		return e
	}
	if count > maxItems {
		return jobPermanent("자동 폴더를 포함한 이관 항목 수가 설정 한도를 초과했습니다")
	}
	_, e = tx.Exec(ctx, "UPDATE migration_sessions SET item_count=$2 WHERE id=$1", v.ID, count)
	if e == nil {
		e = tx.Commit(ctx)
	}
	return e
}

func migrationTargetFingerprint(ctx context.Context, tx pgx.Tx, p *Principal, kind, id, wid string) (int64, string, bool, error) {
	var version int64
	var hash string
	var allowed bool
	switch kind {
	case "document", "folder":
		e := tx.QueryRow(ctx, `SELECT version,md5(jsonb_build_array(title,markdown,tags,aliases,icon,status,visibility,parent_id,space_id,block_metadata)::text),owner_id=$2 AND madi_document_allowed($2,id,true) FROM documents WHERE id=$1 AND workspace_id=$3 AND deleted_at IS NULL`, id, p.ID, wid).Scan(&version, &hash, &allowed)
		return version, hash, allowed, e
	case "attachment":
		e := tx.QueryRow(ctx, `SELECT 1,md5(jsonb_build_array(a.name,a.content_type,a.size,a.checksum_sha256,a.object_key,a.storage_provider_id)::text),a.user_id=$2 AND madi_document_allowed($2,a.document_id,true) FROM attachments a JOIN documents d ON d.id=a.document_id WHERE a.id=$1 AND d.workspace_id=$3 AND d.deleted_at IS NULL`, id, p.ID, wid).Scan(&version, &hash, &allowed)
		return version, hash, allowed, e
	case "csv":
		if !hasIntegrationScope(p, "database:write") {
			return 0, "", false, nil
		}
		e := tx.QueryRow(ctx, `SELECT 1,md5(jsonb_build_array(d.name,d.properties,d.space_id)::text||coalesce((SELECT string_agg(md5(jsonb_build_array(r.id,r.values,r.created_by,r.updated_by)::text),'' ORDER BY r.id) FROM database_rows r WHERE r.database_id=d.id),'')),madi_space_allowed($2,d.space_id,true) FROM databases d WHERE d.id=$1 AND d.workspace_id=$3`, id, p.ID, wid).Scan(&version, &hash, &allowed)
		return version, hash, allowed, e
	}
	return 0, "", false, errors.New("지원하지 않는 이관 대상입니다")
}

func (s *Server) classifyMigrationItems(ctx context.Context, p *Principal, v migrationSession) error {
	items, e := s.migrationSessionIndex(ctx, v.ID)
	if e != nil {
		return e
	}
	for _, item := range items {
		if item.Checkpoint > 0 {
			continue
		}
		tx, e := s.DB.Begin(ctx)
		if e != nil {
			return e
		}
		e = func() error {
			defer tx.Rollback(ctx)
			var status string
			if e := tx.QueryRow(ctx, "SELECT status FROM migration_sessions WHERE id=$1 FOR UPDATE", v.ID).Scan(&status); e != nil {
				return e
			}
			if status != "preparing" {
				return jobPermanent("이관 준비가 중단되었습니다")
			}
			var id, hash, targetHash, kind string
			var version, revision int64
			e := tx.QueryRow(ctx, `SELECT target_id::text,source_hash,target_version,target_hash,revision,kind FROM migration_source_bindings WHERE workspace_id=$1 AND user_id=$2 AND source_key=$3 AND source_id=$4`, v.WorkspaceID, v.UserID, v.SourceKey, item.SourceID).Scan(&id, &hash, &version, &targetHash, &revision, &kind)
			if e != nil && e != pgx.ErrNoRows {
				return e
			}
			item.Disposition = "new"
			if e == nil {
				currentVersion, currentHash, allowed, e := migrationTargetFingerprint(ctx, tx, p, kind, id, v.WorkspaceID)
				item.TargetID = id
				item.ExpectedVersion = currentVersion
				item.TargetHash = currentHash
				item.BindingRevision = revision
				item.Disposition = "changed"
				if kind != item.Kind || e != nil || !allowed || version != currentVersion || targetHash != currentHash {
					item.Disposition = "conflict"
				} else if hash == item.SourceHash {
					item.Disposition = "unchanged"
				}
				if item.Kind == "attachment" && item.Disposition == "changed" {
					item.Metadata["previous_target_id"] = id
					item.TargetID = newID()
				}
			}
			_, e = tx.Exec(ctx, `UPDATE migration_session_items SET disposition=$2,target_id=$3,expected_version=$4,binding_revision=$5,target_hash=$6,metadata=$7,checkpoint=1 WHERE id=$1 AND checkpoint=0`, item.ID, item.Disposition, item.TargetID, item.ExpectedVersion, item.BindingRevision, item.TargetHash, jsonValue(item.Metadata))
			if e != nil {
				return e
			}
			return tx.Commit(ctx)
		}()
		if e != nil {
			return e
		}
	}
	return nil
}

func (s *Server) prepareMigrationSessionJob(ctx context.Context, j Job) (result map[string]any, err error) {
	defer func() {
		var permanent permanentJobError
		if err != nil && (errors.As(err, &permanent) || j.Attempts >= j.MaxAttempts) {
			message := "준비를 완료하지 못했습니다. 파일·링크·보호 정책을 확인하고 다시 준비하세요"
			if errors.As(err, &permanent) {
				message = permanent.Error()
			}
			_, _ = s.DB.Exec(context.WithoutCancel(ctx), "UPDATE migration_sessions SET status='failed',error=$2,updated_at=now() WHERE id=$1 AND job_id=$3 AND status='preparing'", str(j.Payload, "session_id"), message, j.ID)
		}
	}()
	return s.prepareMigrationSessionItems(ctx, j)
}

func (s *Server) prepareMigrationSessionItems(ctx context.Context, j Job) (map[string]any, error) {
	v, p, e := s.migrationSessionJob(ctx, j)
	if e != nil {
		return nil, e
	}
	if v.Status == "ready" {
		return v.Report, nil
	}
	if v.Status != "preparing" {
		return nil, jobPermanent("이관 준비 상태가 아닙니다")
	}
	if e = s.ensureMigrationFolders(ctx, v); e != nil {
		return nil, e
	}
	if e = s.classifyMigrationItems(ctx, p, v); e != nil {
		return nil, e
	}
	items, e := s.migrationSessionIndex(ctx, v.ID)
	if e != nil {
		return nil, e
	}
	index := newMigrationLinkIndex(items)
	if e = index.validateParents(items); e != nil {
		return nil, jobPermanent(e.Error())
	}
	for _, item := range items {
		if item.Status == "prepared" {
			continue
		}
		if _, p, e = s.migrationSessionJob(ctx, j); e != nil {
			return nil, e
		}
		if e = s.prepareMigrationSessionItem(ctx, p, v, item, index, items); e != nil {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			errorText := "원본 변환·보호 또는 링크 검증에 실패했습니다: " + e.Error()
			if len(errorText) > 1000 {
				errorText = errorText[:1000]
			}
			_, _ = s.DB.Exec(ctx, "UPDATE migration_session_items SET status='failed',error=$2 WHERE id=$1", item.ID, errorText)
			_, _ = s.DB.Exec(ctx, "UPDATE migration_sessions SET status='failed',error=$2,updated_at=now() WHERE id=$1 AND status='preparing'", v.ID, errorText)
			return nil, jobPermanent(errorText)
		}
	}
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return nil, e
	}
	defer tx.Rollback(ctx)
	var status string
	if e = tx.QueryRow(ctx, "SELECT status FROM migration_sessions WHERE id=$1 FOR UPDATE", v.ID).Scan(&status); e != nil {
		return nil, e
	}
	if status != "preparing" {
		return nil, jobPermanent("이관 준비가 취소되었습니다")
	}
	report, planHash, e := migrationSessionPlanTx(ctx, tx, v.ID)
	if e != nil {
		return nil, e
	}
	_, e = tx.Exec(ctx, "UPDATE migration_sessions SET status='ready',report=$2,plan_hash=$3,revision=revision+1,updated_at=now() WHERE id=$1", v.ID, jsonValue(report), planHash)
	if e == nil {
		e = tx.Commit(ctx)
	}
	return report, e
}

func migrationSessionPlanTx(ctx context.Context, tx pgx.Tx, id string) (map[string]any, string, error) {
	var raw []byte
	var plan string
	e := tx.QueryRow(ctx, `SELECT jsonb_build_object('total',count(*),'prepared',count(*) FILTER(WHERE status='prepared'),'new',count(*) FILTER(WHERE disposition='new'),'changed',count(*) FILTER(WHERE disposition='changed'),'unchanged',count(*) FILTER(WHERE disposition='unchanged'),'conflicts',count(*) FILTER(WHERE disposition='conflict' AND coalesce(metadata->>'decision','apply')<>'skip'),'skipped',count(*) FILTER(WHERE metadata->>'decision'='skip'),'csv_unconfirmed',count(*) FILTER(WHERE kind='csv' AND NOT types_confirmed AND coalesce(metadata->>'decision','apply')<>'skip')),
 encode(sha256(convert_to(coalesce(string_agg(encode(sha256(convert_to(jsonb_build_array(id,source_hash,status,target_id,disposition,expected_version,binding_revision,target_hash,csv_types,types_confirmed,metadata,encode(sha256(prepared_data),'hex'))::text,'UTF8')),'hex'),'' ORDER BY id),''),'UTF8')),'hex') FROM migration_session_items WHERE session_id=$1`, id).Scan(&raw, &plan)
	var out map[string]any
	if e == nil {
		e = json.Unmarshal(raw, &out)
	}
	if e == nil {
		var size, objects int64
		size, objects, e = migrationStoragePlanTx(ctx, tx, id)
		out["storage_bytes"] = size
		out["storage_objects"] = objects
	}
	if e == nil {
		out["broken_dependencies"], e = migrationDependencyIssuesTx(ctx, tx, id)
	}
	return out, plan, e
}
