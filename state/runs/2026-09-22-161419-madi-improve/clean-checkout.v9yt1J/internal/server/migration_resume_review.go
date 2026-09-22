package server

import (
	"context"
	"errors"
	"net/http"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
)

func (s *Server) migrationSessionItemRequest(w http.ResponseWriter, r *http.Request) (migrationSession, migrationSessionItem, bool) {
	v, ok := s.migrationSessionRequest(w, r)
	if !ok {
		return v, migrationSessionItem{}, false
	}
	item, e := scanMigrationItem(s.DB.QueryRow(r.Context(), "SELECT "+migrationItemSelect+" FROM migration_session_items WHERE id=$1 AND session_id=$2", r.PathValue("item"), v.ID))
	if e != nil {
		apiError(w, 404, "이관 항목을 찾을 수 없습니다")
		return v, item, false
	}
	return v, item, true
}

func migrationTextPreview(v string) (string, bool) {
	if len(v) <= 256<<10 {
		return v, false
	}
	v = v[:256<<10]
	for !utf8.ValidString(v) {
		v = v[:len(v)-1]
	}
	return v, true
}

func (s *Server) getMigrationSessionItem(w http.ResponseWriter, r *http.Request) {
	v, item, ok := s.migrationSessionItemRequest(w, r)
	if !ok {
		return
	}
	if !v.ExpiresAt.After(time.Now()) {
		apiError(w, 410, "이관 원문 비교 보관 기간이 만료되었습니다")
		return
	}
	if item.Status != "prepared" || len(item.PreparedData) == 0 {
		jsonResponse(w, 200, map[string]any{"item": migrationPublicItem(item)})
		return
	}
	prepared, e := s.migrationPreparedValue(item)
	if e != nil {
		respond(w, nil, e)
		return
	}
	raw, e := s.migrationItemRaw(r.Context(), item)
	if e != nil {
		apiError(w, 410, "원본 비교 데이터가 만료되었거나 손상되었습니다")
		return
	}
	out := map[string]any{"item": migrationPublicItem(item), "title": prepared.Title, "canonical_sha256": item.Compatibility["canonical_sha256"], "parent_id": prepared.ParentID, "attachments": prepared.Attachments}
	if item.Kind == "csv" && prepared.CSV != nil {
		columns := inferMigrationCSV(*prepared.CSV)
		out["inferred_columns"] = columns
		out["row_count"] = len(prepared.CSV.Rows)
		limit := min(20, len(prepared.CSV.Rows))
		out["sample_rows"] = prepared.CSV.Rows[:limit]
	} else if item.Kind != "attachment" {
		source, sourceTruncated := migrationTextPreview(string(raw))
		canonical, canonicalTruncated := migrationTextPreview(prepared.Markdown)
		out["source"] = source
		out["canonical"] = canonical
		out["source_truncated"] = sourceTruncated
		out["canonical_truncated"] = canonicalTruncated
		out["diff"] = boundedDocumentDiff(string(raw), prepared.Markdown)
		if oneOf(item.Disposition, "changed", "conflict") && s.canDocument(r.Context(), current(r), item.TargetID, false) {
			var currentMarkdown string
			if s.DB.QueryRow(r.Context(), "SELECT markdown FROM documents WHERE id=$1 AND deleted_at IS NULL", item.TargetID).Scan(&currentMarkdown) == nil {
				out["current_diff"] = boundedDocumentDiff(currentMarkdown, prepared.Markdown)
			}
		}
	}
	if !s.migrationSessionAllowed(r.Context(), current(r), v) {
		apiError(w, 403, "이관 권한이 변경되었습니다")
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	jsonResponse(w, 200, out)
}

func (s *Server) downloadMigrationSessionItem(w http.ResponseWriter, r *http.Request) {
	v, item, ok := s.migrationSessionItemRequest(w, r)
	if !ok {
		return
	}
	if !v.ExpiresAt.After(time.Now()) {
		apiError(w, 410, "이관 원본 보관 기간이 만료되었습니다")
		return
	}
	raw, e := s.migrationItemRaw(r.Context(), item)
	if e != nil {
		apiError(w, 410, "이관 원본을 사용할 수 없습니다")
		return
	}
	if r.URL.Query().Get("view") == "canonical" {
		prepared, e := s.migrationPreparedValue(item)
		if e != nil || oneOf(item.Kind, "attachment", "csv") {
			apiError(w, 400, "정본 Markdown이 있는 문서를 선택하세요")
			return
		}
		raw = []byte(prepared.Markdown)
	}
	if !s.migrationSessionAllowed(r.Context(), current(r), v) {
		apiError(w, 403, "이관 권한이 변경되었습니다")
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", `attachment; filename="madi-migration-item.bin"`)
	_, _ = w.Write(raw)
}

func migrationLockReview(ctx context.Context, tx pgx.Tx, id string, revision int64, hash string) (migrationSession, error) {
	v, e := scanMigrationSession(tx.QueryRow(ctx, "SELECT "+migrationSessionSelect+" FROM migration_sessions WHERE id=$1 AND expires_at>now() FOR UPDATE", id))
	if e != nil {
		return v, e
	}
	if v.Status != "ready" || v.Revision != revision || v.PlanHash != hash {
		return v, errors.New("이관 검토 내용이 바뀌었습니다. 최신 미리보기를 다시 확인하세요")
	}
	return v, nil
}

func (s *Server) reviewMigrationSessionItem(w http.ResponseWriter, r *http.Request) {
	v, item, ok := s.migrationSessionItemRequest(w, r)
	if !ok {
		return
	}
	var in struct {
		Revision     int64                `json:"revision"`
		PlanHash     string               `json:"plan_hash"`
		Decision     string               `json:"decision"`
		Types        []migrationCSVColumn `json:"types"`
		ConfirmTypes bool                 `json:"confirm_types"`
	}
	if decode(r, &in) != nil || !oneOf(in.Decision, "apply", "skip") {
		apiError(w, 400, "검토할 항목·처리 방법을 확인하세요")
		return
	}
	if item.Disposition == "conflict" && in.Decision != "skip" {
		apiError(w, 409, "변경된 기존 문서는 덮어쓸 수 없습니다. 제외하거나 별도 원본 식별자로 새 이관을 준비하세요")
		return
	}
	if item.Kind == "folder" && in.Decision == "skip" {
		apiError(w, 400, "자동 폴더는 하위 항목을 보호하기 위해 개별 제외할 수 없습니다")
		return
	}
	if item.Kind == "csv" && in.Decision == "apply" {
		if !in.ConfirmTypes || !hasIntegrationScope(current(r), "database:write") {
			apiError(w, 400, "CSV 자료형은 데이터베이스 쓰기 권한으로 명시 확인해야 합니다")
			return
		}
		prepared, e := s.migrationPreparedValue(item)
		if e != nil || prepared.CSV == nil {
			apiError(w, 400, "CSV 준비 데이터를 다시 확인하세요")
			return
		}
		if e = validateMigrationCSVColumns(*prepared.CSV, in.Types); e != nil {
			apiError(w, 400, e.Error())
			return
		}
		for i := range in.Types {
			in.Types[i].Samples = nil
			in.Types[i].Reason = "사용자 확인"
		}
	}
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	if _, e = migrationLockReview(r.Context(), tx, v.ID, in.Revision, in.PlanHash); e != nil {
		apiError(w, 409, e.Error())
		return
	}
	_, e = tx.Exec(r.Context(), "UPDATE migration_session_items SET metadata=jsonb_set(metadata,'{decision}',to_jsonb($2::text)),csv_types=CASE WHEN kind='csv' AND $2='apply' THEN $3::jsonb ELSE csv_types END,types_confirmed=CASE WHEN kind='csv' AND $2='apply' THEN true ELSE types_confirmed END WHERE id=$1", item.ID, in.Decision, jsonValue(in.Types))
	if e == nil && item.Kind == "csv" && in.Decision == "apply" && item.Disposition == "unchanged" {
		// Equal source bytes do not imply an equal explicitly chosen schema.
		_, e = tx.Exec(r.Context(), `UPDATE migration_session_items i SET disposition='changed' WHERE i.id=$1 AND (
 (SELECT jsonb_agg(jsonb_build_object('name',p->>'name','type',p->>'type','options',coalesce(nullif(p->'options','null'),'[]')) ORDER BY n) FROM databases d CROSS JOIN LATERAL jsonb_array_elements(d.properties) WITH ORDINALITY x(p,n) WHERE d.id=i.target_id)
 IS DISTINCT FROM (SELECT jsonb_agg(jsonb_build_object('name',p->>'name','type',p->>'type','options',coalesce(nullif(p->'options','null'),'[]')) ORDER BY n) FROM jsonb_array_elements($2::jsonb) WITH ORDINALITY x(p,n)))`, item.ID, jsonValue(in.Types))
	}
	var report map[string]any
	var hash string
	if e == nil {
		report, hash, e = migrationSessionPlanTx(r.Context(), tx, v.ID)
	}
	if e == nil {
		_, e = tx.Exec(r.Context(), "UPDATE migration_sessions SET report=$2,plan_hash=$3,revision=revision+1,updated_at=now() WHERE id=$1", v.ID, jsonValue(report), hash)
	}
	if e == nil {
		e = tx.Commit(r.Context())
	}
	var reviewError permanentJobError
	if errors.As(e, &reviewError) {
		apiError(w, 409, reviewError.Error())
		return
	}
	respond(w, map[string]any{"revision": v.Revision + 1, "plan_hash": hash, "report": report}, e)
}

func (s *Server) cancelMigrationSession(w http.ResponseWriter, r *http.Request) {
	v, ok := s.migrationSessionRequest(w, r)
	if !ok {
		return
	}
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	var status string
	e = tx.QueryRow(r.Context(), "SELECT status FROM migration_sessions WHERE id=$1 FOR UPDATE", v.ID).Scan(&status)
	if e != nil {
		respond(w, nil, e)
		return
	}
	if status == "completed" {
		apiError(w, 409, "완료된 이관은 취소로 문서를 삭제하지 않습니다")
		return
	}
	_, e = tx.Exec(r.Context(), "UPDATE automation_jobs SET cancel_requested=true WHERE id=(SELECT job_id FROM migration_sessions WHERE id=$1) AND status IN ('pending','running')", v.ID)
	if e == nil {
		_, e = tx.Exec(r.Context(), "DELETE FROM migration_session_chunks WHERE item_id IN (SELECT id FROM migration_session_items WHERE session_id=$1)", v.ID)
	}
	if e == nil {
		_, e = tx.Exec(r.Context(), "UPDATE migration_session_items SET prepared_data=NULL WHERE session_id=$1", v.ID)
	}
	if e == nil {
		_, e = tx.Exec(r.Context(), "UPDATE migration_sessions SET status='cancelled',purged_at=now(),revision=revision+1,updated_at=now() WHERE id=$1", v.ID)
	}
	if e == nil {
		e = tx.Commit(r.Context())
	}
	respond(w, map[string]bool{"cancelled": e == nil}, e)
}

func (s *Server) expireMigrationSessions(ctx context.Context) {
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return
	}
	defer tx.Rollback(ctx)
	rows, e := tx.Query(ctx, "SELECT id::text FROM migration_sessions WHERE expires_at<=now() AND purged_at IS NULL ORDER BY expires_at LIMIT 20 FOR UPDATE SKIP LOCKED")
	if e != nil {
		return
	}
	ids := []string{}
	for rows.Next() {
		var id string
		if rows.Scan(&id) != nil {
			rows.Close()
			return
		}
		ids = append(ids, id)
	}
	e = rows.Err()
	rows.Close()
	if e != nil || len(ids) == 0 {
		return
	}
	_, e = tx.Exec(ctx, "UPDATE automation_jobs SET cancel_requested=true WHERE id IN (SELECT job_id FROM migration_sessions WHERE id=ANY($1::uuid[])) AND status IN ('pending','running')", ids)
	if e == nil {
		_, e = tx.Exec(ctx, "DELETE FROM migration_session_chunks WHERE item_id IN (SELECT id FROM migration_session_items WHERE session_id=ANY($1::uuid[]))", ids)
	}
	if e == nil {
		_, e = tx.Exec(ctx, "UPDATE migration_session_items SET prepared_data=NULL WHERE session_id=ANY($1::uuid[])", ids)
	}
	if e == nil {
		_, e = tx.Exec(ctx, "UPDATE migration_sessions SET status=CASE WHEN status='completed' THEN status ELSE 'cancelled' END,purged_at=now(),error=CASE WHEN status='completed' THEN '원본 비교 데이터의 보관 기간이 만료되었습니다' ELSE '이관 보관 기간이 만료되었습니다' END,revision=revision+1 WHERE id=ANY($1::uuid[])", ids)
	}
	if e == nil {
		_ = tx.Commit(ctx)
	}
}
