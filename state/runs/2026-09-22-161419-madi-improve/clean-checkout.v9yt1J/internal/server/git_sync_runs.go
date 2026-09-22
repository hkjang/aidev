package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
)

type gitSyncRun struct {
	ID              string           `json:"id"`
	ConnectionID    string           `json:"connection_id"`
	WorkspaceID     string           `json:"workspace_id"`
	OwnerID         string           `json:"owner_id"`
	Direction       string           `json:"direction"`
	Status          string           `json:"status"`
	Fingerprint     string           `json:"config_fingerprint"`
	DocumentIDs     []string         `json:"document_ids"`
	SourceVersions  map[string]int64 `json:"source_versions"`
	Report          map[string]any   `json:"report"`
	RemoteCommit    string           `json:"remote_commit"`
	CandidateCommit string           `json:"candidate_commit"`
	JobID           string           `json:"job_id"`
	Revision        int64            `json:"revision"`
	ExpiresAt       time.Time        `json:"expires_at"`
	CreatedAt       time.Time        `json:"created_at"`
}

const gitSyncRunSelect = `id::text,connection_id::text,workspace_id::text,owner_id::text,direction,status,config_fingerprint,document_ids,source_versions,report,remote_commit,candidate_commit,coalesce(job_id::text,''),revision,expires_at,created_at`

func gitSyncScanRun(row pgx.Row) (gitSyncRun, error) {
	var run gitSyncRun
	var ids, versions, report []byte
	e := row.Scan(&run.ID, &run.ConnectionID, &run.WorkspaceID, &run.OwnerID, &run.Direction, &run.Status, &run.Fingerprint, &ids, &versions, &report, &run.RemoteCommit, &run.CandidateCommit, &run.JobID, &run.Revision, &run.ExpiresAt, &run.CreatedAt)
	if e == nil {
		e = json.Unmarshal(ids, &run.DocumentIDs)
	}
	if e == nil {
		e = json.Unmarshal(versions, &run.SourceVersions)
	}
	if e == nil {
		e = json.Unmarshal(report, &run.Report)
	}
	return run, e
}
func gitSyncFingerprint(c gitSyncConnection, revision int64) string {
	return gitSyncContentHash(jsonValue(map[string]any{"connection_id": c.ID, "revision": c.Revision, "policy_revision": revision, "owner": c.OwnerID, "workspace": c.WorkspaceID, "space": c.SpaceID}))
}
func (s *Server) gitSyncMappings(ctx context.Context, id string) (map[string]gitSyncMapping, error) {
	rows, e := s.DB.Query(ctx, `SELECT path,coalesce(document_id::text,''),coalesce(attachment_id::text,''),local_hash,remote_hash FROM git_sync_mappings WHERE connection_id=$1`, id)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := map[string]gitSyncMapping{}
	for rows.Next() {
		var m gitSyncMapping
		if e = rows.Scan(&m.Path, &m.DocumentID, &m.AttachmentID, &m.LocalHash, &m.RemoteHash); e != nil {
			return nil, e
		}
		out[m.Path] = m
	}
	return out, rows.Err()
}
func (s *Server) gitSyncVisibleRun(r *http.Request) (gitSyncRun, gitSyncConnection, error) {
	run, e := gitSyncScanRun(s.DB.QueryRow(r.Context(), "SELECT "+gitSyncRunSelect+" FROM git_sync_runs WHERE id=$1", r.PathValue("run")))
	if e != nil {
		return run, gitSyncConnection{}, e
	}
	c, e := s.loadGitSyncConnection(r.Context(), run.ConnectionID)
	if e != nil || run.OwnerID != current(r).ID || !s.gitSyncManager(r, c) {
		return run, c, errors.New("Git 작업을 볼 권한이 없습니다")
	}
	for id := range run.SourceVersions {
		if !s.canDocument(r.Context(), current(r), id, false) {
			return run, c, errors.New("원본 문서의 현재 권한이 없습니다")
		}
	}
	return run, c, nil
}
func (s *Server) listGitSyncRuns(w http.ResponseWriter, r *http.Request) {
	c, e := s.loadGitSyncConnection(r.Context(), r.PathValue("id"))
	if e != nil || !s.gitSyncManager(r, c) {
		apiError(w, 404, "Git 연결을 찾을 수 없습니다")
		return
	}
	rows, e := s.DB.Query(r.Context(), "SELECT "+gitSyncRunSelect+" FROM git_sync_runs WHERE connection_id=$1 AND owner_id=$2 ORDER BY created_at DESC LIMIT 100", c.ID, current(r).ID)
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer rows.Close()
	out := []gitSyncRun{}
	for rows.Next() {
		run, e := gitSyncScanRun(rows)
		if e != nil {
			respond(w, nil, e)
			return
		}
		out = append(out, run)
	}
	// Row metadata only. The detailed file report is returned by the ACL-checked
	// single-run endpoint, never as a stale collection after document revocation.
	for i := range out {
		out[i].Report = map[string]any{}
		out[i].SourceVersions = map[string]int64{}
		out[i].DocumentIDs = []string{}
	}
	respond(w, out, rows.Err())
}
func (s *Server) getGitSyncRun(w http.ResponseWriter, r *http.Request) {
	run, _, e := s.gitSyncVisibleRun(r)
	if e != nil {
		apiError(w, 404, "Git 작업 또는 원본 문서 권한을 확인하세요")
		return
	}
	jsonResponse(w, 200, run)
}
func (s *Server) queueGitSyncPreview(w http.ResponseWriter, r *http.Request) {
	c, e := s.loadGitSyncConnection(r.Context(), r.PathValue("id"))
	if e != nil || !s.gitSyncManager(r, c) {
		apiError(w, 404, "Git 연결을 찾을 수 없습니다")
		return
	}
	var in struct {
		Direction   string   `json:"direction"`
		DocumentIDs []string `json:"document_ids"`
		Paths       []string `json:"paths"`
		ReadConsent bool     `json:"read_consent"`
	}
	if decode(r, &in) != nil || !oneOf(in.Direction, "push", "pull") || !in.ReadConsent {
		apiError(w, 400, "방향과 원격 저장소 읽기 동의를 확인하세요")
		return
	}
	if in.Direction == "push" && (len(in.DocumentIDs) == 0 || len(in.DocumentIDs) > 1000) {
		apiError(w, 400, "전송할 문서를 1~1000개 선택하세요")
		return
	}
	if in.Direction == "pull" && len(in.DocumentIDs) > 0 {
		apiError(w, 400, "원격 수신은 문서 ID 대신 원격 경로를 선택하세요")
		return
	}
	if len(in.Paths) > 1000 {
		apiError(w, 400, "원격 경로는 최대 1000개입니다")
		return
	}
	for _, p := range in.Paths {
		if !gitSyncSafeFile(p) {
			apiError(w, 400, "원격 경로를 확인하세요")
			return
		}
	}
	seen := map[string]bool{}
	for _, id := range in.DocumentIDs {
		if !validID(id) || seen[id] || !s.canDocument(r.Context(), current(r), id, false) {
			apiError(w, 403, "선택 문서 접근 권한을 확인하세요")
			return
		}
		seen[id] = true
	}
	policy, policyRevision, e := s.gitSyncSettings(r.Context())
	if e != nil || !c.Enabled || !boolean(policy, "enabled") {
		apiError(w, 400, "관리자 Git 정책과 연결을 활성화하세요")
		return
	}
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	var revision int64
	if e = tx.QueryRow(r.Context(), "SELECT revision FROM git_sync_connections WHERE id=$1 FOR UPDATE", c.ID).Scan(&revision); e != nil || revision != c.Revision {
		apiError(w, 409, "Git 연결이 변경되었습니다")
		return
	}
	var busy bool
	e = tx.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM git_sync_runs WHERE connection_id=$1 AND status IN ('preparing','queued','running'))`, c.ID).Scan(&busy)
	if e != nil {
		respond(w, nil, e)
		return
	}
	if busy {
		apiError(w, 409, "이 연결에서 진행 중인 작업이 있습니다")
		return
	}
	id := newID()
	fingerprint := gitSyncFingerprint(c, policyRevision)
	if in.DocumentIDs == nil {
		in.DocumentIDs = []string{}
	}
	_, e = tx.Exec(r.Context(), `INSERT INTO git_sync_runs(id,connection_id,workspace_id,owner_id,actor_constraints,direction,config_fingerprint,document_ids,report) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`, id, c.ID, c.WorkspaceID, current(r).ID, jsonValue(constraintsFor(current(r))), in.Direction, fingerprint, jsonValue(in.DocumentIDs), jsonValue(map[string]any{"requested_paths": in.Paths}))
	jobID := ""
	if e == nil {
		jobID, e = s.EnqueueJob(r.Context(), tx, "git-sync.preview", c.OwnerID, c.WorkspaceID, map[string]any{"run_id": id})
	}
	if e == nil {
		_, e = tx.Exec(r.Context(), "UPDATE git_sync_runs SET job_id=$2 WHERE id=$1", id, jobID)
	}
	if e == nil {
		_, e = tx.Exec(r.Context(), "UPDATE automation_jobs SET max_attempts=3,timeout_seconds=300 WHERE id=$1", jobID)
	}
	if e == nil {
		e = tx.Commit(r.Context())
	}
	if e != nil {
		respond(w, nil, e)
		return
	}
	s.audit(r, "GIT_SYNC_PREVIEW", id, map[string]any{"direction": in.Direction, "document_count": len(in.DocumentIDs)})
	jsonResponse(w, 202, map[string]any{"id": id, "job_id": jobID})
}
func (s *Server) confirmGitSync(w http.ResponseWriter, r *http.Request) {
	run, c, e := s.gitSyncVisibleRun(r)
	if e != nil {
		apiError(w, 404, "Git 작업을 찾을 수 없습니다")
		return
	}
	var in struct {
		Revision        int64  `json:"revision"`
		Fingerprint     string `json:"config_fingerprint"`
		RemoteCommit    string `json:"remote_commit"`
		ConfirmExternal bool   `json:"confirm_external_visibility"`
	}
	if decode(r, &in) != nil || !in.ConfirmExternal || in.Revision != run.Revision || in.Fingerprint != run.Fingerprint || in.RemoteCommit != run.RemoteCommit {
		apiError(w, 400, "전송 목록·Git 독립 권한·현재 미리보기 확인이 필요합니다")
		return
	}
	if run.Status != "preview" || !run.ExpiresAt.After(time.Now()) || boolean(run.Report, "has_conflicts") || boolean(run.Report, "selection_required") {
		apiError(w, 409, "새 미리보기에서 충돌과 파일 선택을 확인하세요")
		return
	}
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	if e = s.gitSyncGuardTx(r.Context(), tx, run, c, current(r), run.Direction == "pull"); e != nil {
		apiError(w, 409, e.Error())
		return
	}
	var busy bool
	e = tx.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM git_sync_runs WHERE connection_id=$1 AND id!=$2 AND status IN ('preparing','queued','running'))`, c.ID, run.ID).Scan(&busy)
	if e != nil || busy {
		apiError(w, 409, "다른 Git 작업이 진행 중입니다")
		return
	}
	jobID, e := s.EnqueueJob(r.Context(), tx, "git-sync.execute", c.OwnerID, c.WorkspaceID, map[string]any{"run_id": run.ID})
	if e == nil {
		_, e = tx.Exec(r.Context(), "UPDATE automation_jobs SET max_attempts=3,timeout_seconds=300 WHERE id=$1", jobID)
	}
	if e == nil {
		tag, err := tx.Exec(r.Context(), `UPDATE git_sync_runs SET status='queued',job_id=$2,confirmed_at=now(),revision=revision+1,updated_at=now() WHERE id=$1 AND revision=$3 AND status='preview'`, run.ID, jobID, in.Revision)
		e = err
		if e == nil && tag.RowsAffected() != 1 {
			e = errors.New("미리보기가 변경되었습니다")
		}
	}
	if e == nil {
		e = tx.Commit(r.Context())
	}
	if e != nil {
		apiError(w, 409, e.Error())
		return
	}
	s.audit(r, "GIT_SYNC_CONFIRM", run.ID, map[string]any{"direction": run.Direction})
	jsonResponse(w, 202, map[string]any{"id": run.ID, "job_id": jobID})
}
func (s *Server) cancelGitSync(w http.ResponseWriter, r *http.Request) {
	run, _, e := s.gitSyncVisibleRun(r)
	if e != nil {
		apiError(w, 404, "Git 작업을 찾을 수 없습니다")
		return
	}
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	var status string
	e = tx.QueryRow(r.Context(), "SELECT status FROM git_sync_runs WHERE id=$1 FOR UPDATE", run.ID).Scan(&status)
	if e != nil || !oneOf(status, "preparing", "preview", "queued", "running", "unknown") {
		apiError(w, 409, "현재 상태에서는 취소할 수 없습니다")
		return
	}
	target := "cancelled"
	if run.Direction == "push" && oneOf(status, "running", "unknown") {
		target = "unknown"
	}
	_, e = tx.Exec(r.Context(), "UPDATE git_sync_runs SET status=$2,revision=revision+1,updated_at=now() WHERE id=$1", run.ID, target)
	if e == nil && run.JobID != "" {
		_, e = tx.Exec(r.Context(), "UPDATE automation_jobs SET cancel_requested=true,updated_at=now() WHERE id=$1 AND status IN ('pending','running')", run.JobID)
	}
	if e == nil {
		e = tx.Commit(r.Context())
	}
	respond(w, map[string]any{"status": target, "message": "이미 시작된 원격 쓰기는 취소만으로 되돌리지 않습니다. 결과 불명은 원격 확인이 필요합니다"}, e)
}

// Called in the final mutation transaction, not only during preview. These
// locks serialize selected versions, policy and run state with local changes.
func (s *Server) gitSyncGuardTx(ctx context.Context, tx pgx.Tx, run gitSyncRun, c gitSyncConnection, p *Principal, writing bool) error {
	var revision, policyRevision int64
	var enabled, policyEnabled bool
	if e := tx.QueryRow(ctx, "SELECT revision,enabled FROM git_sync_connections WHERE id=$1 FOR UPDATE", c.ID).Scan(&revision, &enabled); e != nil {
		return e
	}
	if e := tx.QueryRow(ctx, "SELECT revision,coalesce((data->>'enabled')::boolean,false) FROM git_sync_settings WHERE id=1 FOR SHARE").Scan(&policyRevision, &policyEnabled); e != nil {
		return e
	}
	if !enabled || !policyEnabled || revision != c.Revision || gitSyncFingerprint(c, policyRevision) != run.Fingerprint {
		return errors.New("Git 연결·정책이 변경되어 새 동의가 필요합니다")
	}
	for _, id := range []string{p.ID, c.OwnerID} {
		var role, kind string
		var disabled bool
		if e := tx.QueryRow(ctx, "SELECT role,kind,disabled FROM users WHERE id=$1 FOR SHARE", id).Scan(&role, &kind, &disabled); e != nil || disabled || role == "viewer" || (id == p.ID && kind != "user") {
			return errors.New("Git 실행 계정의 현재 권한을 확인하세요")
		}
		var allowed bool
		if e := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM workspace_members WHERE workspace_id=$1 AND user_id=$2 AND role IN ('owner','admin','editor')) AND madi_space_allowed($2,NULLIF($3,'')::uuid,true)`, c.WorkspaceID, id, c.SpaceID).Scan(&allowed); e != nil || !allowed {
			return errors.New("Git 실행 계정의 워크스페이스·공간 권한이 변경되었습니다")
		}
	}
	var manager bool
	if e := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM workspace_members WHERE workspace_id=$1 AND user_id=$2 AND role IN ('owner','admin'))`, c.WorkspaceID, p.ID).Scan(&manager); e != nil || !manager || p.TokenID != "" || p.ScopeRestricted {
		return errors.New("현재 사용자의 Git 관리 권한이 없습니다")
	}
	if len(run.SourceVersions) > 0 {
		var allowed bool
		e := tx.QueryRow(ctx, `WITH expected AS (SELECT key::uuid id,value::bigint expected_version FROM jsonb_each_text($1::jsonb)), locked AS MATERIALIZED (SELECT d.id,d.version,e.expected_version FROM documents d JOIN expected e ON e.id=d.id WHERE d.workspace_id=$4 AND d.deleted_at IS NULL ORDER BY d.id FOR SHARE OF d) SELECT count(*)=$6 AND coalesce(bool_and(version=expected_version AND madi_document_allowed($2,id,$5) AND madi_document_allowed($3,id,$5)),false) FROM locked`, jsonValue(run.SourceVersions), p.ID, c.OwnerID, c.WorkspaceID, writing, len(run.SourceVersions)).Scan(&allowed)
		if e != nil || !allowed {
			return errors.New("원본 문서 버전·공유 권한이 변경되었습니다. 다시 미리보기 하세요")
		}
	}
	return nil
}
