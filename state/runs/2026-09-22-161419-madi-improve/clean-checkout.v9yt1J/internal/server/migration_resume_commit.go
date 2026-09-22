package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sort"

	"github.com/jackc/pgx/v5"
)

func (s *Server) commitMigrationSession(w http.ResponseWriter, r *http.Request) {
	v, ok := s.migrationSessionRequest(w, r)
	if !ok {
		return
	}
	var in struct {
		Revision     int64  `json:"revision"`
		PlanHash     string `json:"plan_hash"`
		Confirmation string `json:"confirmation"`
	}
	if decode(r, &in) != nil || in.Confirmation != "IMPORT" {
		apiError(w, 400, "원문 비교·자료형·변경 목록을 확인한 뒤 IMPORT로 확정하세요")
		return
	}
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	v, e = migrationLockReview(r.Context(), tx, v.ID, in.Revision, in.PlanHash)
	if e != nil {
		apiError(w, 409, e.Error())
		return
	}
	report, hash, e := migrationSessionPlanTx(r.Context(), tx, v.ID)
	if e != nil {
		var changed permanentJobError
		if errors.As(e, &changed) {
			apiError(w, 409, changed.Error())
			return
		}
		respond(w, nil, e)
		return
	}
	if hash != in.PlanHash || number(report, "total", 0) != number(report, "prepared", -1) || number(report, "conflicts", 1) > 0 || number(report, "csv_unconfirmed", 1) > 0 || number(report, "broken_dependencies", 1) > 0 {
		apiError(w, 409, "모든 준비·충돌·참조 제외 처리·CSV 자료형 확인이 완료되어야 합니다")
		return
	}
	sessionHash, tokenHash := "", ""
	if current(r).TokenID != "" {
		e = tx.QueryRow(r.Context(), "SELECT token_hash FROM api_keys WHERE id=$1", current(r).TokenID).Scan(&tokenHash)
	} else if cookie, err := r.Cookie("madi_session"); err == nil {
		sessionHash = digest(cookie.Value)
	}
	jobID := ""
	if e == nil {
		jobID, e = s.EnqueueJob(r.Context(), tx, "migration.session.commit", v.UserID, v.WorkspaceID, map[string]any{"session_id": v.ID, "plan_hash": hash})
	}
	if e == nil {
		_, e = tx.Exec(r.Context(), "UPDATE migration_sessions SET status='committing',job_id=$2,request_session_hash=$3,request_token_hash=$4,request_ip=$5,error='',updated_at=now() WHERE id=$1", v.ID, jobID, sessionHash, tokenHash, integrationClientIP(r))
	}
	if e == nil {
		_, e = tx.Exec(r.Context(), "UPDATE automation_jobs SET timeout_seconds=3600,max_attempts=5 WHERE id=$1", jobID)
	}
	if e == nil {
		e = tx.Commit(r.Context())
	}
	respond(w, map[string]string{"job_id": jobID}, e)
}

func (s *Server) commitMigrationSessionJob(ctx context.Context, j Job) (result map[string]any, err error) {
	defer func() {
		var permanent permanentJobError
		if err != nil && (errors.As(err, &permanent) || j.Attempts >= j.MaxAttempts || errors.Is(err, errMigrationChanged)) {
			message := "이관 확정을 완료하지 못했습니다. 원본·대상·권한·저장소를 검토한 후 다시 준비하세요"
			if errors.As(err, &permanent) || errors.Is(err, errMigrationChanged) {
				message = err.Error()
			}
			_, _ = s.DB.Exec(context.WithoutCancel(ctx), "UPDATE migration_sessions SET status='failed',error=$2,updated_at=now() WHERE id=$1 AND job_id=$3 AND status='committing'", str(j.Payload, "session_id"), message, j.ID)
		}
	}()
	return s.commitMigrationSessionAtomic(ctx, j)
}

func (s *Server) commitMigrationSessionAtomic(ctx context.Context, j Job) (map[string]any, error) {
	v, p, e := s.migrationSessionJob(ctx, j)
	if e != nil {
		return nil, e
	}
	if v.Status == "completed" {
		return v.Report, nil
	}
	if v.Status != "committing" {
		return nil, jobPermanent("확정 대기 상태의 이관이 아닙니다")
	}
	items, e := s.migrationSessionIndex(ctx, v.ID)
	if e != nil {
		return nil, e
	}
	if _, e = s.DB.Exec(ctx, `UPDATE migration_sessions SET report=report||'{"phase":"staging_objects"}'::jsonb WHERE id=$1 AND status='committing'`, v.ID); e != nil {
		return nil, e
	}
	if e = s.stageMigrationObjects(ctx, j, v, items); e != nil {
		return nil, e
	}
	if _, e = s.DB.Exec(ctx, `UPDATE migration_sessions SET report=report||'{"phase":"publication_lock"}'::jsonb WHERE id=$1 AND status='committing'`, v.ID); e != nil {
		return nil, e
	}
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return nil, e
	}
	defer tx.Rollback(ctx)
	if _, e = tx.Exec(ctx, `SET LOCAL lock_timeout='5s'; SET LOCAL statement_timeout='10min'`); e != nil {
		return nil, e
	}
	// No document/row is visible before the final commit, even after a crash.
	_, e = tx.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtextextended($1,9326))", v.WorkspaceID+":"+v.UserID+":"+v.SourceKey)
	if e != nil {
		return nil, e
	}
	v, e = scanMigrationSession(tx.QueryRow(ctx, "SELECT "+migrationSessionSelect+" FROM migration_sessions WHERE id=$1 FOR UPDATE", v.ID))
	if e != nil {
		return nil, e
	}
	if v.Status == "completed" {
		return v.Report, nil
	}
	if v.Status != "committing" || v.PlanHash != str(j.Payload, "plan_hash") {
		return nil, jobPermanent("이관 확정이 취소되거나 변경되었습니다")
	}
	// Existing membership rows are locked by migrationActorTx; these tables also
	// admit ACL-narrowing INSERTs, which row locks alone cannot prevent. Limit
	// this conservative fence to the final atomic publication transaction.
	if _, e = tx.Exec(ctx, "LOCK TABLE spaces,space_members,document_shares IN SHARE MODE"); e != nil {
		return nil, e
	}
	ids, dbIDs := []string{}, []string{}
	needsCSV := false
	for _, i := range items {
		if str(i.Metadata, "decision") == "skip" {
			continue
		}
		if i.Kind == "csv" {
			needsCSV = true
			if i.BindingRevision > 0 {
				dbIDs = append(dbIDs, i.TargetID)
			}
		} else if oneOf(i.Kind, "document", "folder") && i.BindingRevision > 0 {
			ids = append(ids, i.TargetID)
		}
	}
	sort.Strings(ids)
	sort.Strings(dbIDs)
	for _, lock := range []struct {
		table string
		ids   []string
	}{{"documents", ids}, {"databases", dbIDs}} {
		query := "SELECT id FROM " + lock.table + " WHERE id=ANY($1::uuid[]) ORDER BY id FOR UPDATE"
		if lock.table == "documents" {
			query = `WITH RECURSIVE ancestry AS(SELECT id,parent_id,ARRAY[id] seen FROM documents WHERE id=ANY($1::uuid[]) UNION ALL SELECT d.id,d.parent_id,a.seen||d.id FROM documents d JOIN ancestry a ON d.id=a.parent_id WHERE cardinality(a.seen)<20 AND NOT d.id=ANY(a.seen)) SELECT d.id FROM documents d WHERE d.id IN(SELECT id FROM ancestry) ORDER BY d.id FOR UPDATE`
		}
		rows, err := tx.Query(ctx, query, lock.ids)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return nil, err
		}
	}
	if e = s.migrationActorTx(ctx, tx, p, v, needsCSV); e != nil {
		return nil, jobPermanent(e.Error())
	}
	if e = s.validateSignedMigrationTx(ctx, tx, v); e != nil {
		return nil, jobPermanent(e.Error())
	}
	for _, i := range items {
		if str(i.Metadata, "decision") == "skip" {
			continue
		}
		if e = migrationBindingGuardTx(ctx, tx, p, v, i); e != nil {
			return nil, jobPermanent(e.Error())
		}
	}
	// Serialize tree mutations and revalidate the complete placement before the
	// transaction becomes visible. All new documents remain private drafts.
	if _, e = tx.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtextextended($1,22))", "documents/"+v.WorkspaceID); e != nil {
		return nil, e
	}
	counts := map[string]any{"new": 0, "changed": 0, "unchanged": 0, "skipped": 0, "documents": 0, "databases": 0, "attachments": 0, "atomic": true, "session_id": v.ID}
	for _, item := range items {
		if str(item.Metadata, "decision") == "skip" {
			counts["skipped"] = counts["skipped"].(int) + 1
			continue
		}
		counts[item.Disposition] = number(counts, item.Disposition, 0) + 1
		if item.Disposition == "unchanged" {
			continue
		}
		if _, p, e = s.migrationSessionJob(ctx, j); e != nil {
			return nil, e
		}
		item, e = scanMigrationItem(tx.QueryRow(ctx, "SELECT "+migrationItemSelect+" FROM migration_session_items WHERE id=$1", item.ID))
		if e != nil {
			return nil, e
		}
		prepared, err := s.migrationPreparedValue(item)
		if err != nil {
			return nil, err
		}
		switch item.Kind {
		case "document", "folder":
			e = s.commitMigrationDocumentTx(ctx, tx, p, v, item, prepared)
			counts["documents"] = counts["documents"].(int) + 1
		case "csv":
			e = s.commitMigrationCSVTx(ctx, tx, p, v, item, prepared)
			counts["databases"] = counts["databases"].(int) + 1
		}
		if e != nil {
			return nil, e
		}
	}
	// Parent rows now exist. The common placement gate checks both ancestry ACL
	// and combined existing/new subtree depth; uploaded UUIDs never authorize it.
	req := automationRequest(ctx, p, http.MethodPost, nil)
	for _, item := range items {
		if !oneOf(item.Kind, "document", "folder") || item.Disposition == "unchanged" || str(item.Metadata, "decision") == "skip" {
			continue
		}
		var data []byte
		if e = tx.QueryRow(ctx, "SELECT prepared_data FROM migration_session_items WHERE id=$1", item.ID).Scan(&data); e != nil {
			return nil, e
		}
		item.PreparedData = data
		prepared, err := s.migrationPreparedValue(item)
		if err != nil {
			return nil, err
		}
		if e = s.treePlacementTx(req, tx, "documents", v.WorkspaceID, item.TargetID, prepared.ParentID); e != nil {
			return nil, e
		}
		if _, e = tx.Exec(ctx, "UPDATE documents SET parent_id=NULLIF($2,'')::uuid WHERE id=$1", item.TargetID, prepared.ParentID); e != nil {
			return nil, e
		}
	}
	if e = s.commitMigrationAttachmentsTx(ctx, tx, p, v, items); e != nil {
		return nil, e
	}
	for _, item := range items {
		if str(item.Metadata, "decision") == "skip" {
			continue
		}
		version, hash, allowed, err := migrationTargetFingerprint(ctx, tx, p, item.Kind, item.TargetID, v.WorkspaceID)
		if err != nil || !allowed {
			return nil, errMigrationChanged
		}
		_, e = tx.Exec(ctx, `INSERT INTO migration_source_bindings(workspace_id,user_id,source_key,source_id,kind,target_id,source_hash,target_version,target_hash,session_id) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) ON CONFLICT(workspace_id,user_id,source_key,source_id) DO UPDATE SET kind=EXCLUDED.kind,target_id=EXCLUDED.target_id,source_hash=EXCLUDED.source_hash,target_version=EXCLUDED.target_version,target_hash=EXCLUDED.target_hash,session_id=EXCLUDED.session_id,revision=migration_source_bindings.revision+1,updated_at=now()`, v.WorkspaceID, v.UserID, v.SourceKey, item.SourceID, item.Kind, item.TargetID, item.SourceHash, version, hash, v.ID)
		if e != nil {
			return nil, e
		}
		if item.Kind == "attachment" {
			counts["attachments"] = counts["attachments"].(int) + 1
		}
	}
	if _, p, e = s.migrationSessionJob(ctx, j); e != nil {
		return nil, e
	}
	if e = s.migrationActorTx(ctx, tx, p, v, needsCSV); e != nil {
		return nil, e
	}
	if e = s.finishSignedMigrationTx(ctx, tx, v); e != nil {
		return nil, e
	}
	effectCtx := context.WithValue(ctx, automationEffectKey{}, automationEffectContext{JobID: j.ID, Index: 0})
	if e = s.recordAutomationEffect(effectCtx, tx, counts); e != nil {
		return nil, e
	}
	_, e = tx.Exec(ctx, "UPDATE migration_sessions SET status='completed',report=$2,completed_at=now(),updated_at=now(),revision=revision+1,request_session_hash='',request_token_hash='' WHERE id=$1", v.ID, jsonValue(counts))
	if e == nil {
		e = tx.Commit(ctx)
	}
	return counts, e
}

func (s *Server) commitMigrationDocumentTx(ctx context.Context, tx pgx.Tx, p *Principal, v migrationSession, item migrationSessionItem, prepared migrationPrepared) error {
	protected, e := s.protectCanonicalDocumentTx(ctx, tx, p, item.TargetID, v.WorkspaceID, prepared.Title, prepared.Markdown, prepared.Tags, prepared.Aliases, map[string]any{})
	if e != nil {
		return e
	}
	if protected.Changed {
		return errMigrationChanged
	}
	event := "document.created"
	if item.Disposition == "new" {
		_, e = tx.Exec(ctx, `INSERT INTO documents(id,workspace_id,owner_id,space_id,title,markdown,tags,aliases,icon,status,visibility) VALUES($1,$2,$3,NULLIF($4,'')::uuid,$5,$6,$7,$8,$9,'draft','private')`, item.TargetID, v.WorkspaceID, p.ID, v.SpaceID, prepared.Title, prepared.Markdown, jsonValue(prepared.Tags), jsonValue(prepared.Aliases), prepared.Icon)
	} else {
		var raw []byte
		if e = tx.QueryRow(ctx, "SELECT to_jsonb(d) FROM documents d WHERE id=$1", item.TargetID).Scan(&raw); e != nil {
			return e
		}
		before := map[string]any{}
		if e = json.Unmarshal(raw, &before); e != nil {
			return e
		}
		changes := map[string]any{"title": prepared.Title, "markdown": prepared.Markdown, "tags": prepared.Tags, "aliases": prepared.Aliases, "parent_id": prepared.ParentID}
		status, err := approvalSaveStatusTx(ctx, tx, item.TargetID, before, changes, "draft")
		if err != nil {
			return err
		}
		_, e = tx.Exec(ctx, `UPDATE documents SET title=$2,markdown=$3,tags=$4,aliases=$5,icon=$6,status=$7,version=version+1,block_metadata='{}',updated_at=now() WHERE id=$1`, item.TargetID, prepared.Title, prepared.Markdown, jsonValue(prepared.Tags), jsonValue(prepared.Aliases), prepared.Icon, status)
		if e == nil {
			e = collaborationResetTx(ctx, tx, item.TargetID, newID(), prepared.Markdown, int(item.ExpectedVersion+1), "source_replaced")
		}
		event = "document.updated"
	}
	if e != nil {
		return e
	}
	_, e = tx.Exec(ctx, `INSERT INTO document_versions(document_id,version,title,markdown,tags,user_id) SELECT id,version,title,markdown,tags,$2 FROM documents WHERE id=$1`, item.TargetID, p.ID)
	if e == nil {
		e = s.enqueueEvent(ctx, tx, Event{Type: event, WorkspaceID: v.WorkspaceID, ResourceID: item.TargetID, After: map[string]any{"id": item.TargetID, "title": prepared.Title, "tags": prepared.Tags}})
	}
	return e
}

func (s *Server) commitMigrationCSVTx(ctx context.Context, tx pgx.Tx, p *Principal, v migrationSession, item migrationSessionItem, prepared migrationPrepared) error {
	if !item.TypesConfirmed || prepared.CSV == nil || !hasIntegrationScope(p, "database:write") {
		return errMigrationChanged
	}
	if e := validateMigrationCSVColumns(*prepared.CSV, item.CSVTypes); e != nil {
		return e
	}
	props := []map[string]any{}
	for _, col := range item.CSVTypes {
		if col.Options == nil {
			col.Options = []string{}
		}
		props = append(props, map[string]any{"id": newID(), "name": col.Name, "type": col.Type, "options": col.Options})
	}
	var properties any
	if e := json.Unmarshal(jsonValue(props), &properties); e != nil {
		return e
	}
	if e := validateProperties(properties); e != nil {
		return e
	}
	clean, e := s.ProtectDocumentMetadataTx(ctx, tx, p, "", v.WorkspaceID, map[string]any{"name": prepared.Title, "properties": props})
	if e != nil {
		return e
	}
	if clean.Changed {
		return errMigrationChanged
	}
	if item.Disposition == "new" {
		_, e = tx.Exec(ctx, "INSERT INTO databases(id,workspace_id,space_id,name,properties) VALUES($1,$2,NULLIF($3,'')::uuid,$4,$5)", item.TargetID, v.WorkspaceID, v.SpaceID, prepared.Title, jsonValue(props))
	} else {
		// Deleting rows that are relation targets would break independent user
		// data. The database FK/typed relation validators must not be bypassed.
		var refs bool
		if tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM databases d,jsonb_array_elements(d.properties) p WHERE d.id<>$1 AND p->>'type'='relation' AND p->>'target_database_id'=$1::text)`, item.TargetID).Scan(&refs) != nil || refs {
			return errors.New("다른 데이터베이스가 참조하는 CSV 대상은 바꿀 수 없습니다. 별도 원본으로 복사하세요")
		}
		_, e = tx.Exec(ctx, "DELETE FROM database_rows WHERE database_id=$1", item.TargetID)
		if e == nil {
			_, e = tx.Exec(ctx, "UPDATE databases SET name=$2,properties=$3 WHERE id=$1", item.TargetID, prepared.Title, jsonValue(props))
		}
	}
	if e != nil {
		return e
	}
	for _, row := range prepared.CSV.Rows {
		values := map[string]any{}
		for col, value := range row {
			converted, err := convertMigrationCSVCell(value, item.CSVTypes[col])
			if err != nil {
				return err
			}
			values[str(props[col], "id")] = converted
		}
		clean, err := s.ProtectDocumentMetadataTx(ctx, tx, p, "", v.WorkspaceID, values)
		if err != nil {
			return err
		}
		if clean.Changed {
			return errMigrationChanged
		}
		if _, e = tx.Exec(ctx, "INSERT INTO database_rows(id,database_id,values,created_by,updated_by) VALUES($1,$2,$3,$4,$4)", newID(), item.TargetID, jsonValue(values), p.ID); e != nil {
			return e
		}
	}
	event := "database.created"
	if item.Disposition != "new" {
		event = "database.updated"
	}
	return s.enqueueDatabaseEvent(automationRequest(ctx, p, http.MethodPost, nil), tx, event, item.TargetID, "", nil)
}
