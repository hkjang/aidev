package server

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"path"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

//go:embed migration_schema.sql
var migrationSchema string

type migrationContextKey struct{}

func (s *Server) migrateImports(ctx context.Context) error {
	_, e := s.DB.Exec(ctx, migrationSchema)
	if e == nil {
		_, e = s.DB.Exec(ctx, migrationResumeSchema)
	}
	return e
}
func (s *Server) registerImports() {
	s.handle("GET /api/v1/migrations/sessions", s.listMigrationSessions)
	s.handle("POST /api/v1/migrations/sessions", s.createMigrationSession)
	s.handle("GET /api/v1/migrations/sessions/{id}", s.getMigrationSession)
	s.handle("DELETE /api/v1/migrations/sessions/{id}", s.cancelMigrationSession)
	s.handle("GET /api/v1/migrations/sessions/{id}/items", s.listMigrationSessionItems)
	s.handle("POST /api/v1/migrations/sessions/{id}/items", s.registerMigrationSessionItems)
	s.handle("GET /api/v1/migrations/sessions/{id}/items/{item}", s.getMigrationSessionItem)
	s.handle("GET /api/v1/migrations/sessions/{id}/items/{item}/download", s.downloadMigrationSessionItem)
	s.handle("GET /api/v1/migrations/sessions/{id}/items/{item}/chunks", s.migrationSessionItemChunks)
	s.handle("PUT /api/v1/migrations/sessions/{id}/items/{item}/chunks/{chunk}", s.uploadMigrationSessionChunk)
	s.handle("PUT /api/v1/migrations/sessions/{id}/items/{item}/review", s.reviewMigrationSessionItem)
	s.handle("POST /api/v1/migrations/sessions/{id}/prepare", s.beginMigrationSessionPrepare)
	s.handle("POST /api/v1/migrations/sessions/{id}/commit", s.commitMigrationSession)
	s.admin("GET /api/v1/admin/migration/settings", s.migrationSetting)
	s.admin("PUT /api/v1/admin/migration/settings", s.migrationSetting)
	s.RegisterJobHandler("migration.session.prepare", s.prepareMigrationSessionJob)
	s.RegisterJobHandler("migration.session.commit", s.commitMigrationSessionJob)
	s.handle("GET /api/v1/migrations", s.listMigrations)
	s.handle("POST /api/v1/migrations", s.stageMigration)
	s.handle("GET /api/v1/migrations/{id}", s.getMigration)
	s.handle("POST /api/v1/migrations/{id}/run", s.runMigration)
	s.handle("DELETE /api/v1/migrations/{id}", s.cancelMigration)
	s.RegisterJobHandler("migration.preview", s.previewMigrationJob)
	s.RegisterJobHandler("migration.import", s.executeMigrationJob)
}

func (s *Server) StartImports(ctx context.Context) {
	go func() {
		tick := time.NewTicker(time.Hour)
		defer tick.Stop()
		for {
			if ctx.Err() != nil {
				return
			}
			s.expireMigrationSources(ctx)
			s.expireMigrationSessions(ctx)
			s.cleanupMigrationObjects(ctx)
			select {
			case <-ctx.Done():
				return
			case <-tick.C:
			}
		}
	}()
}
func (s *Server) expireMigrationSources(ctx context.Context) {
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return
	}
	defer tx.Rollback(ctx)
	rows, e := tx.Query(ctx, `WITH expired AS (SELECT id FROM migration_imports WHERE expires_at<=now() AND source_data IS NOT NULL ORDER BY expires_at LIMIT 100 FOR UPDATE SKIP LOCKED) UPDATE migration_imports SET source_data=NULL,status='cancelled',report='{"error":"원본 파일의 7일 보관 기간이 만료되었습니다"}' WHERE id IN (SELECT id FROM expired) RETURNING coalesce(job_id::text,'')`)
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
		if id != "" {
			ids = append(ids, id)
		}
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return
	}
	if _, e = tx.Exec(ctx, "UPDATE automation_jobs SET cancel_requested=true WHERE id=ANY($1::uuid[]) AND status IN ('pending','running')", ids); e == nil {
		_ = tx.Commit(ctx)
	}
}
func (s *Server) migrationAccess(r *http.Request, v map[string]any) bool {
	return str(v, "user_id") == current(r).ID && hasIntegrationScope(current(r), "document:write") && s.canSpace(r.Context(), current(r), str(v, "workspace_id"), str(v, "space_id"), true)
}

const migrationPublic = `to_jsonb(m)-'source_data'||jsonb_build_object('source_bytes',CASE WHEN source_data IS NULL THEN 0 ELSE source_size END,'job_status',(SELECT status FROM automation_jobs WHERE id=m.job_id),'job_error',(SELECT last_error FROM automation_jobs WHERE id=m.job_id))`

func (s *Server) listMigrations(w http.ResponseWriter, r *http.Request) {
	wid := r.URL.Query().Get("workspace_id")
	if !hasIntegrationScope(current(r), "document:write") || !s.canWorkspace(r.Context(), current(r), wid, true) {
		apiError(w, 403, "가져오기 권한이 없습니다")
		return
	}
	rows, e := s.rows(r.Context(), "SELECT "+migrationPublic+" FROM migration_imports m WHERE workspace_id=$1 AND user_id=$2 ORDER BY created_at DESC LIMIT 200", wid, current(r).ID)
	if e == nil {
		filtered := []map[string]any{}
		for _, v := range rows {
			if s.migrationAccess(r, v) {
				filtered = append(filtered, v)
			}
		}
		rows = filtered
	}
	respond(w, rows, e)
}
func (s *Server) getMigration(w http.ResponseWriter, r *http.Request) {
	v, e := s.one(r.Context(), "SELECT "+migrationPublic+" FROM migration_imports m WHERE id=$1", r.PathValue("id"))
	if e != nil || !s.migrationAccess(r, v) {
		apiError(w, 404, "가져오기 이력을 찾을 수 없습니다")
		return
	}
	respond(w, v, nil)
}
func (s *Server) stageMigration(w http.ResponseWriter, r *http.Request) {
	wid := r.URL.Query().Get("workspace_id")
	space := r.URL.Query().Get("space_id")
	format := r.URL.Query().Get("format")
	if !oneOf(format, "markdown", "obsidian", "notion", "html", "csv", "json") {
		apiError(w, 400, "가져오기 형식을 선택하세요")
		return
	}
	if !hasIntegrationScope(current(r), "document:write") || !s.canSpace(r.Context(), current(r), wid, space, true) || (format == "csv" && !hasIntegrationScope(current(r), "database:write")) {
		apiError(w, 403, "대상 공간에 가져오기 권한이 없습니다")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 52<<20)
	if r.ParseMultipartForm(2<<20) != nil {
		apiError(w, 400, "가져오기 파일은 50MB 이하여야 합니다")
		return
	}
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}
	file, header, e := r.FormFile("file")
	if e != nil {
		apiError(w, 400, "파일을 선택하세요")
		return
	}
	defer file.Close()
	raw, e := io.ReadAll(io.LimitReader(file, (50<<20)+1))
	name := path.Base(strings.ReplaceAll(header.Filename, "\\", "/"))
	if e != nil || len(raw) == 0 || len(raw) > 50<<20 || len(name) > 240 || strings.ContainsAny(name, "\x00\n\r") {
		apiError(w, 400, "파일 이름과 50MB 크기 제한을 확인하세요")
		return
	}
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	_, e = tx.Exec(r.Context(), "SELECT pg_advisory_xact_lock(hashtextextended($1,9324))", current(r).ID)
	var bytes int64
	if e == nil {
		e = tx.QueryRow(r.Context(), "SELECT coalesce(sum(octet_length(source_data)),0) FROM migration_imports WHERE user_id=$1", current(r).ID).Scan(&bytes)
	}
	if e != nil {
		respond(w, nil, e)
		return
	}
	if bytes+int64(len(raw)) > 200<<20 {
		apiError(w, 400, "개인별 가져오기 대기 파일은 합계 200MB까지입니다. 기존 대기 파일을 취소하세요")
		return
	}
	id := newID()
	cipher, e := s.encrypt(string(raw))
	if e != nil {
		respond(w, nil, e)
		return
	}
	_, e = tx.Exec(r.Context(), `INSERT INTO migration_imports(id,workspace_id,user_id,space_id,filename,format,source_data,source_encrypted,source_size) VALUES($1,$2,$3,NULLIF($4,'')::uuid,$5,$6,$7,true,$8)`, id, wid, current(r).ID, space, name, format, []byte(cipher), len(raw))
	var jobID string
	if e == nil {
		jobID, e = s.EnqueueJob(r.Context(), tx, "migration.preview", current(r).ID, wid, map[string]any{"migration_id": id})
	}
	if e == nil {
		_, e = tx.Exec(r.Context(), "UPDATE migration_imports SET job_id=$2 WHERE id=$1", id, jobID)
	}
	if e == nil {
		e = tx.Commit(r.Context())
	}
	respond(w, map[string]string{"id": id, "job_id": jobID}, e)
}
func (s *Server) runMigration(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Confirmation string `json:"confirmation"`
	}
	if decode(r, &in) != nil || in.Confirmation != "IMPORT" {
		apiError(w, 400, "미리보기를 확인한 뒤 IMPORT로 확정하세요")
		return
	}
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	var raw []byte
	e = tx.QueryRow(r.Context(), "SELECT "+migrationPublic+" FROM migration_imports m WHERE id=$1 FOR UPDATE", r.PathValue("id")).Scan(&raw)
	var v map[string]any
	if e == nil {
		e = json.Unmarshal(raw, &v)
	}
	if e != nil || !s.migrationAccess(r, v) {
		apiError(w, 404, "가져오기 이력을 찾을 수 없습니다")
		return
	}
	preview, _ := v["preview"].(map[string]any)
	if !(oneOf(str(v, "status"), "ready", "failed") || (str(v, "status") == "running" && str(v, "job_status") == "failed")) || !boolean(preview, "validated") || number(v, "source_bytes", 0) == 0 {
		apiError(w, 409, "유효한 미리보기와 원본 파일이 필요합니다")
		return
	}
	if str(v, "format") == "csv" && !hasIntegrationScope(current(r), "database:write") {
		apiError(w, 403, "데이터베이스 가져오기 권한이 없습니다")
		return
	}
	jobID, e := s.EnqueueJob(r.Context(), tx, "migration.import", current(r).ID, str(v, "workspace_id"), map[string]any{"migration_id": str(v, "id")})
	if e == nil {
		_, e = tx.Exec(r.Context(), "UPDATE migration_imports SET status='running',job_id=$2,report='{}' WHERE id=$1", str(v, "id"), jobID)
	}
	if e == nil {
		_, e = tx.Exec(r.Context(), "UPDATE automation_jobs SET timeout_seconds=3600,max_attempts=3 WHERE id=$1", jobID)
	}
	if e == nil {
		e = tx.Commit(r.Context())
	}
	respond(w, map[string]string{"job_id": jobID}, e)
}
func (s *Server) cancelMigration(w http.ResponseWriter, r *http.Request) {
	v, e := s.one(r.Context(), "SELECT "+migrationPublic+" FROM migration_imports m WHERE id=$1", r.PathValue("id"))
	if e != nil || !s.migrationAccess(r, v) {
		apiError(w, 404, "가져오기 이력을 찾을 수 없습니다")
		return
	}
	if str(v, "status") == "completed" {
		apiError(w, 409, "완료된 가져오기의 문서는 문서 메뉴에서 관리하세요")
		return
	}
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	_, e = tx.Exec(r.Context(), "UPDATE migration_imports SET status='cancelled',source_data=NULL WHERE id=$1 AND status<>'completed'", str(v, "id"))
	if e == nil {
		_, e = tx.Exec(r.Context(), "UPDATE automation_jobs SET cancel_requested=true WHERE id=NULLIF($1,'')::uuid AND status IN ('pending','running')", str(v, "job_id"))
	}
	if e == nil {
		e = tx.Commit(r.Context())
	}
	respond(w, map[string]bool{"ok": true}, e)
}
func (s *Server) migrationJobSource(ctx context.Context, j Job) (map[string]any, []byte, *Principal, error) {
	var metadata, raw []byte
	e := s.DB.QueryRow(ctx, "SELECT "+migrationPublic+",source_data FROM migration_imports m WHERE id=$1 AND expires_at>now()", str(j.Payload, "migration_id")).Scan(&metadata, &raw)
	if e != nil {
		return nil, nil, nil, jobPermanent("가져오기 파일이 없거나 7일 보관 기한이 지났습니다")
	}
	var v map[string]any
	if json.Unmarshal(metadata, &v) != nil {
		return nil, nil, nil, errors.New("가져오기 이력을 읽을 수 없습니다")
	}
	p, e := s.workerPrincipal(ctx, j.ActorID, j.TokenID, j.WorkspaceID)
	if e != nil {
		return nil, nil, nil, e
	}
	r := automationRequest(ctx, p, "POST", nil)
	if j.OwnerID != str(v, "user_id") || !s.migrationAccess(r, v) || str(v, "workspace_id") != j.WorkspaceID || str(v, "job_id") != j.ID || str(v, "status") == "cancelled" {
		return nil, nil, nil, jobPermanent("가져오기 작업의 현재 권한이 없거나 작업이 취소되었습니다")
	}
	if boolean(v, "source_encrypted") {
		plain, e := s.decrypt(string(raw))
		if e != nil {
			return nil, nil, nil, jobPermanent("가져오기 원본을 해독하지 못했습니다")
		}
		raw = []byte(plain)
	}
	return v, raw, p, nil
}
func (s *Server) previewMigrationJob(ctx context.Context, j Job) (map[string]any, error) {
	v, raw, p, e := s.migrationJobSource(ctx, j)
	if e != nil {
		return nil, e
	}
	input, csv, e := prepareMigrationInput(str(v, "filename"), str(v, "format"), raw)
	if e != nil {
		s.DB.Exec(ctx, "UPDATE migration_imports SET status='failed',report=$2 WHERE id=$1 AND job_id=$3", str(v, "id"), jsonValue(map[string]any{"error": e.Error()}), j.ID)
		return nil, jobPermanent(e.Error())
	}
	preview := map[string]any{"validated": true, "documents": 0, "folders": 0, "attachments": 0, "databases": 0, "rows": 0, "warnings": []string{}, "sample": []map[string]any{}}
	if csv != nil {
		preview["databases"] = 1
		preview["rows"] = len(csv.Rows)
		preview["columns"] = csv.Header
	} else {
		docs, byFile, e := prepareVaultDocuments(input)
		if e != nil {
			return nil, jobPermanent(e.Error())
		}
		files := 0
		folders := 0
		samples := []map[string]any{}
		byID := map[string]*vaultDocument{}
		for _, d := range docs {
			byID[d.NewID] = d
		}
		for _, d := range docs {
			seen := map[string]bool{}
			cur := d
			depth := 0
			for cur != nil {
				if seen[cur.NewID] || depth >= 20 {
					return nil, jobPermanent("가져올 문서 트리는 최대 20단계까지 지원합니다")
				}
				seen[cur.NewID] = true
				depth++
				cur = byID[cur.ParentNewID]
			}
			if d.Folder {
				folders++
				continue
			}
			files++
			if len(samples) < 30 {
				var dup int
				_ = s.DB.QueryRow(ctx, "SELECT count(*) FROM documents WHERE workspace_id=$1 AND title=$2 AND deleted_at IS NULL AND madi_document_allowed($3,id,false)", j.WorkspaceID, d.Title, p.ID).Scan(&dup)
				samples = append(samples, map[string]any{"title": d.Title, "bytes": len(d.Markdown), "duplicates": dup})
			}
		}
		preview["documents"] = files
		preview["folders"] = folders
		preview["attachments"] = len(input.files) - len(byFile)
		preview["sample"] = samples
		preview["databases"] = len(input.databases)
		csvRows := 0
		for _, data := range input.databases {
			csvRows += len(data.Rows)
		}
		preview["rows"] = csvRows
		if len(input.databases) > 0 && !hasIntegrationScope(p, "database:write") {
			return nil, jobPermanent("Notion CSV를 가져오려면 데이터베이스 쓰기 권한이 필요합니다")
		}
		preview["warnings"] = []string{"기존 동명 문서를 덮어쓰지 않고 새 문서로 가져옵니다. 첨부와 링크를 반영한 최종 개수는 결과 보고서에서 확인하세요."}
	}
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return nil, e
	}
	defer tx.Rollback(ctx)
	preview, e = s.protectEventMetadataTx(ctx, tx, preview)
	if e != nil {
		return nil, e
	}
	_, e = tx.Exec(ctx, "UPDATE migration_imports SET status='ready',preview=$2 WHERE id=$1 AND job_id=$3 AND status='pending'", str(v, "id"), jsonValue(preview), j.ID)
	if e == nil {
		e = tx.Commit(ctx)
	}
	return preview, e
}
func (s *Server) executeMigrationJob(ctx context.Context, j Job) (map[string]any, error) {
	v, raw, p, e := s.migrationJobSource(ctx, j)
	if e != nil {
		return nil, e
	}
	if str(v, "status") == "completed" {
		report, _ := v["report"].(map[string]any)
		return report, nil
	}
	input, csv, e := prepareMigrationInput(str(v, "filename"), str(v, "format"), raw)
	if e != nil {
		return nil, jobPermanent(e.Error())
	}
	ctx = context.WithValue(ctx, migrationContextKey{}, str(v, "id"))
	ctx = context.WithValue(ctx, automationEffectKey{}, automationEffectContext{JobID: j.ID, Index: 0})
	if csv != nil {
		return s.importMigrationCSV(ctx, j, p, v, *csv)
	}
	r := automationRequest(ctx, p, "POST", nil)
	query := r.URL.Query()
	query.Set("workspace_id", j.WorkspaceID)
	query.Set("space_id", str(v, "space_id"))
	r.URL.RawQuery = query.Encode()
	rec := httptest.NewRecorder()
	s.importVaultInput(rec, r, input)
	result, e := automationResponse(rec)
	if e != nil {
		s.DB.Exec(context.WithoutCancel(ctx), "UPDATE migration_imports SET report=$2 WHERE id=$1 AND status='running'", str(v, "id"), jsonValue(map[string]any{"error": e.Error()}))
	}
	return result, e
}
func (s *Server) recordMigrationEffect(ctx context.Context, tx pgx.Tx, report map[string]any) error {
	id, _ := ctx.Value(migrationContextKey{}).(string)
	if id == "" {
		return nil
	}
	var status string
	if e := tx.QueryRow(ctx, "SELECT status FROM migration_imports WHERE id=$1 FOR UPDATE", id).Scan(&status); e != nil {
		return e
	}
	if status != "running" {
		return jobPermanent("가져오기가 이미 완료되었거나 취소되었습니다")
	}
	if e := s.recordAutomationEffect(ctx, tx, report); e != nil {
		return e
	}
	_, e := tx.Exec(ctx, "UPDATE migration_imports SET status='completed',report=$2,source_data=NULL WHERE id=$1", id, jsonValue(report))
	return e
}
func (s *Server) importMigrationCSV(ctx context.Context, j Job, p *Principal, v map[string]any, data migrationCSV) (map[string]any, error) {
	if !jobScope(p, "database:write") {
		return nil, jobPermanent("데이터베이스 쓰기 권한이 없습니다")
	}
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return nil, e
	}
	defer tx.Rollback(ctx)
	id, e := s.importCSVTx(ctx, tx, p, j.WorkspaceID, str(v, "space_id"), strings.TrimSuffix(str(v, "filename"), path.Ext(str(v, "filename"))), data)
	if e != nil {
		return nil, e
	}
	r := automationRequest(ctx, p, "POST", nil)
	report := map[string]any{"databases": 1, "rows": len(data.Rows), "database_id": id, "errors": []string{}}
	if e = s.recordMigrationEffect(ctx, tx, report); e != nil {
		return nil, e
	}
	if e = tx.Commit(ctx); e != nil {
		return nil, e
	}
	s.audit(r, "MIGRATION_CSV", id, map[string]int{"rows": len(data.Rows)})
	return report, nil
}
