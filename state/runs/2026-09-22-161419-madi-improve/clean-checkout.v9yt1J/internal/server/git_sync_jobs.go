package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

type gitSyncRemoteFactory func(context.Context, map[string]any, []string, bool) (*gitSyncRemote, error)

func (s *Server) executeGitSyncJob(ctx context.Context, j Job) (map[string]any, error) {
	return s.runGitSyncJob(ctx, j, newGitSyncRemote)
}
func (s *Server) runGitSyncJob(ctx context.Context, j Job, makeRemote gitSyncRemoteFactory) (result map[string]any, err error) {
	run, e := gitSyncScanRun(s.DB.QueryRow(ctx, "SELECT "+gitSyncRunSelect+" FROM git_sync_runs WHERE id=$1", str(j.Payload, "run_id")))
	if e != nil {
		return nil, jobPermanent("Git 작업을 찾을 수 없습니다")
	}
	if run.JobID != j.ID || run.OwnerID != j.ActorID || run.WorkspaceID != j.WorkspaceID {
		return nil, jobPermanent("Git 작업 실행 정보가 일치하지 않습니다")
	}
	if run.Status == "succeeded" {
		return map[string]any{"run_id": run.ID, "status": "succeeded"}, nil
	}
	if oneOf(run.Status, "cancelled", "conflict", "failed", "unknown") {
		return nil, jobPermanent("Git 작업이 중단되었습니다. 새 미리보기 또는 원격 결과 확인이 필요합니다")
	}
	defer func() {
		if err != nil {
			finish, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			target := "failed"
			var conflict gitSyncConflict
			if errors.As(err, &conflict) {
				target = "conflict"
			}
			_, _ = s.DB.Exec(finish, `UPDATE git_sync_runs SET status=CASE WHEN status='running' AND direction='push' THEN 'unknown' ELSE $2 END,report=report||$3::jsonb,revision=revision+1,updated_at=now() WHERE id=$1 AND status IN ('preparing','queued','running')`, run.ID, target, jsonValue(map[string]any{"error": err.Error()}))
		}
	}()
	c, e := s.loadGitSyncConnection(ctx, run.ConnectionID)
	if e != nil || c.OwnerID != j.OwnerID {
		return nil, jobPermanent("Git 연결 실행 계정이 변경되었습니다")
	}
	actor, e := s.workerPrincipal(ctx, j.ActorID, j.TokenID, j.WorkspaceID)
	if e != nil {
		return nil, e
	}
	owner, e := s.workerPrincipal(ctx, j.OwnerID, "", j.WorkspaceID)
	if e != nil {
		return nil, e
	}
	if actor.Kind != "user" || actor.TokenID != "" || actor.ScopeRestricted || !s.automationManager(ctx, actor, c.WorkspaceID) || !s.canSpace(ctx, actor, c.WorkspaceID, c.SpaceID, true) || !s.canSpace(ctx, owner, c.WorkspaceID, c.SpaceID, true) {
		return nil, jobPermanent("Git 실행자의 현재 워크스페이스·공간 권한이 없습니다")
	}
	policy, revision, e := s.gitSyncSettings(ctx)
	if e != nil {
		return nil, e
	}
	if !c.Enabled || !boolean(policy, "enabled") || gitSyncFingerprint(c, revision) != run.Fingerprint {
		return nil, jobPermanent("Git 연결 정책이 변경되었습니다. 다시 동의하세요")
	}
	if !run.ExpiresAt.After(time.Now()) {
		return nil, jobPermanent("Git 미리보기가 만료되었습니다")
	}
	config, e := s.gitSyncDecryptedConfig(c)
	if e != nil {
		return nil, e
	}
	remote, e := makeRemote(ctx, config, listStrings(policy["allowed_hosts"]), boolean(policy, "allow_private_networks"))
	if e != nil {
		return nil, e
	}
	defer remote.Close()
	repo, e := gitSyncFetch(ctx, remote, str(config, "branch"))
	if e != nil {
		return nil, e
	}
	limit := int64(number(policy, "max_snapshot_mb", 100)) << 20
	if j.Kind == "git-sync.preview" {
		if run.Status != "preparing" {
			return nil, jobPermanent("이미 처리된 Git 미리보기입니다")
		}
		files, e := gitSyncRemoteFiles(ctx, repo, str(config, "prefix"), limit)
		if e != nil {
			return nil, e
		}
		mappings, e := s.gitSyncMappings(ctx, c.ID)
		if e != nil {
			return nil, e
		}
		var snapshot gitSyncSnapshot
		var report map[string]any
		if run.Direction == "push" {
			snapshot, e = s.gitSyncBuildExport(ctx, actor, c, run.DocumentIDs, limit)
			if e == nil {
				changes, conflict := gitSyncPushChanges(snapshot.Files, files, mappings)
				report = map[string]any{"files": changes, "has_conflicts": conflict, "remote_visibility_warning": "Git 저장소의 접근 권한은 madi와 독립적입니다. 선택한 개인·선택 공유 문서와 첨부파일도 Git 열람자에게 공개됩니다. 삭제 파일은 자동 반영하지 않습니다"}
			}
		} else {
			snapshot, report, e = s.gitSyncBuildPull(ctx, actor, c, files, mappings, listStrings(run.Report["requested_paths"]), limit)
		}
		if e != nil {
			return nil, e
		}
		snapshot.RemoteCommit = repo.Head.String()
		snapshot.ConfigFingerprint = run.Fingerprint
		run.SourceVersions = snapshot.SourceVersions
		raw := jsonValue(snapshot)
		if len(raw) > 145<<20 {
			return nil, errGitSyncLimit
		}
		cipher, e := s.encrypt(string(raw))
		if e != nil {
			return nil, e
		}
		tx, e := s.DB.Begin(ctx)
		if e != nil {
			return nil, e
		}
		defer tx.Rollback(ctx)
		if e = s.gitSyncGuardTx(ctx, tx, run, c, actor, run.Direction == "pull"); e != nil {
			return nil, e
		}
		tag, e := tx.Exec(ctx, `UPDATE git_sync_runs SET status='preview',snapshot_cipher=$2,snapshot_size=$3,source_versions=$4,remote_commit=$5,report=$6,revision=revision+1,updated_at=now() WHERE id=$1 AND status='preparing' AND job_id=$7`, run.ID, []byte(cipher), len(raw), jsonValue(snapshot.SourceVersions), snapshot.RemoteCommit, jsonValue(report), j.ID)
		if e != nil {
			return nil, e
		}
		if tag.RowsAffected() != 1 {
			return nil, jobPermanent("Git 미리보기가 취소되거나 변경되었습니다")
		}
		if e = tx.Commit(ctx); e != nil {
			return nil, e
		}
		return map[string]any{"run_id": run.ID, "status": "preview"}, nil
	}
	if j.Kind != "git-sync.execute" || !oneOf(run.Status, "queued", "running") {
		return nil, jobPermanent("실행 가능한 Git 작업이 아닙니다")
	}
	if run.Status == "running" {
		// A process may have died after the remote accepted the pack. Never launch
		// a second write: inspect only the exact persisted candidate.
		if run.Direction == "push" && run.CandidateCommit != "" && repo.Head.String() == run.CandidateCommit {
			snapshot, e := s.gitSyncSnapshot(ctx, run.ID)
			if e != nil {
				return nil, e
			}
			return s.gitSyncFinishPush(ctx, run, c, snapshot, run.CandidateCommit)
		}
		return nil, jobPermanent("이전 실행의 결과가 불명확합니다. 원격 결과 확인 후 새 미리보기가 필요합니다")
	}
	if repo.Head.String() != run.RemoteCommit {
		return nil, gitSyncConflict{}
	}
	snapshot, e := s.gitSyncSnapshot(ctx, run.ID)
	if e != nil {
		return nil, e
	}
	if snapshot.ConfigFingerprint != run.Fingerprint || snapshot.RemoteCommit != run.RemoteCommit {
		return nil, jobPermanent("저장된 Git 미리보기 지문이 일치하지 않습니다")
	}
	if run.Direction == "pull" {
		return s.gitSyncApplyPull(ctx, j, run, c, actor, snapshot)
	}
	// Rebuild from current sources, including attachments and paths; an equal
	// version alone does not prove files or metadata stayed unchanged.
	current, e := s.gitSyncBuildExport(ctx, actor, c, snapshot.SelectedDocumentIDs, limit)
	if e != nil {
		return nil, e
	}
	if !gitSyncSameFiles(current.Files, snapshot.Files) {
		return nil, jobPermanent("선택 원본·첨부파일이 변경되었습니다. 새 미리보기가 필요합니다")
	}
	candidate, e := gitSyncCandidate(ctx, repo, snapshot.Files, run.ID, run.CreatedAt)
	if e != nil {
		return nil, e
	}
	remote.Close()
	transferCtx, stopTransfer := context.WithCancel(ctx)
	defer stopTransfer()
	writeRemote, e := makeRemote(transferCtx, config, listStrings(policy["allowed_hosts"]), boolean(policy, "allow_private_networks"))
	if e != nil {
		return nil, e
	}
	defer writeRemote.Close()
	transferStarted := make(chan struct{})
	monitorDone := make(chan struct{})
	defer close(monitorDone)
	go func() {
		select {
		case <-transferStarted:
		case <-monitorDone:
			return
		case <-transferCtx.Done():
			return
		}
		ticker := time.NewTicker(250 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-monitorDone:
				return
			case <-transferCtx.Done():
				return
			case <-ticker.C:
				tx, e := s.DB.Begin(transferCtx)
				if e == nil {
					e = s.gitSyncGuardTx(transferCtx, tx, run, c, actor, false)
					_ = tx.Rollback(transferCtx)
				}
				if e != nil {
					stopTransfer()
					return
				}
			}
		}
	}()
	launched := false
	uncertain, e := gitSyncPush(transferCtx, writeRemote, str(config, "branch"), repo, candidate, func() error {
		tx, e := s.DB.Begin(ctx)
		if e != nil {
			return e
		}
		defer tx.Rollback(ctx)
		if e = s.gitSyncGuardTx(ctx, tx, run, c, actor, false); e != nil {
			return e
		}
		var active bool
		if e = tx.QueryRow(ctx, `SELECT status='running' AND NOT cancel_requested AND lease_id=$2::uuid AND lease_until>now() FROM automation_jobs WHERE id=$1 FOR SHARE`, j.ID, j.LeaseID).Scan(&active); e != nil || !active {
			return jobPermanent("Git 실행 임대가 만료되었거나 취소되었습니다")
		}
		tag, e := tx.Exec(ctx, `UPDATE git_sync_runs SET status='running',candidate_commit=$2,revision=revision+1,updated_at=now() WHERE id=$1 AND status='queued' AND job_id=$3`, run.ID, candidate.String(), j.ID)
		if e != nil {
			return e
		}
		if tag.RowsAffected() != 1 {
			return jobPermanent("Git 실행이 취소되거나 변경되었습니다")
		}
		if e = tx.Commit(ctx); e != nil {
			return e
		}
		launched = true
		close(transferStarted)
		return nil
	})
	if e != nil {
		if launched && !uncertain {
			_, _ = s.DB.Exec(ctx, "UPDATE git_sync_runs SET status='failed',report=report||$2::jsonb,updated_at=now() WHERE id=$1 AND status='running'", run.ID, jsonValue(map[string]any{"error": e.Error()}))
		}
		return nil, e
	}
	return s.gitSyncFinishPush(ctx, run, c, snapshot, candidate.String())
}
func gitSyncSameFiles(a, b []gitSyncFile) bool {
	if len(a) != len(b) {
		return false
	}
	byPath := map[string]gitSyncFile{}
	for _, f := range a {
		byPath[f.Path] = f
	}
	for _, f := range b {
		v, ok := byPath[f.Path]
		if !ok || v.Hash != f.Hash || v.DocumentID != f.DocumentID || v.AttachmentID != f.AttachmentID {
			return false
		}
	}
	return true
}
func (s *Server) gitSyncSnapshot(ctx context.Context, id string) (gitSyncSnapshot, error) {
	var result gitSyncSnapshot
	var cipher []byte
	var size int64
	e := s.DB.QueryRow(ctx, "SELECT snapshot_cipher,snapshot_size FROM git_sync_runs WHERE id=$1", id).Scan(&cipher, &size)
	if e != nil {
		return result, e
	}
	if size < 1 || size > 145<<20 || len(cipher) > 200<<20 {
		return result, errGitSyncLimit
	}
	raw, e := s.decrypt(string(cipher))
	if e != nil || int64(len(raw)) != size {
		return result, errors.New("암호화 Git 미리보기를 읽을 수 없습니다")
	}
	if e = json.Unmarshal([]byte(raw), &result); e != nil {
		return result, e
	}
	return result, nil
}
func (s *Server) gitSyncFinishPush(ctx context.Context, run gitSyncRun, c gitSyncConnection, snapshot gitSyncSnapshot, commit string) (map[string]any, error) {
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return nil, e
	}
	defer tx.Rollback(ctx)
	var status string
	if e = tx.QueryRow(ctx, "SELECT status FROM git_sync_runs WHERE id=$1 FOR UPDATE", run.ID).Scan(&status); e != nil {
		return nil, e
	}
	if !oneOf(status, "running", "unknown", "queued", "succeeded") {
		return nil, errors.New("Git 실행 상태가 변경되었습니다")
	}
	for _, f := range snapshot.Files {
		localHash := f.Hash
		if f.DocumentID != "" && f.AttachmentID == "" {
			localHash = fmt.Sprintf("v:%d", snapshot.SourceVersions[f.DocumentID])
		}
		_, e = tx.Exec(ctx, `INSERT INTO git_sync_mappings(connection_id,path,document_id,attachment_id,local_hash,remote_hash) VALUES($1,$2,NULLIF($3,'')::uuid,NULLIF($4,'')::uuid,$5,$6) ON CONFLICT(connection_id,path) DO UPDATE SET document_id=EXCLUDED.document_id,attachment_id=EXCLUDED.attachment_id,local_hash=EXCLUDED.local_hash,remote_hash=EXCLUDED.remote_hash,updated_at=now()`, c.ID, f.Path, f.DocumentID, f.AttachmentID, localHash, f.Hash)
		if e != nil {
			return nil, e
		}
	}
	_, e = tx.Exec(ctx, "UPDATE git_sync_runs SET status='succeeded',candidate_commit=$2,revision=revision+1,updated_at=now() WHERE id=$1", run.ID, commit)
	if e == nil {
		_, e = tx.Exec(ctx, "UPDATE git_sync_connections SET last_remote_commit=$2,updated_at=now() WHERE id=$1", c.ID, commit)
	}
	if e == nil {
		e = tx.Commit(ctx)
	}
	return map[string]any{"run_id": run.ID, "status": "succeeded", "commit": commit}, e
}

// Retain metadata/history, not indefinite decrypted-document snapshots. A
// restored/past preview must never authorize a fresh transfer by itself.
func (s *Server) StartGitSync(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for {
			_, _ = s.DB.Exec(ctx, `UPDATE git_sync_runs r SET status=CASE WHEN r.status='running' AND r.direction='push' THEN 'unknown' WHEN j.status='cancelled' THEN 'cancelled' ELSE 'failed' END,report=r.report||jsonb_build_object('error','공통 작업이 중단되었습니다. 실행 이력을 확인하고 새 미리보기를 만드세요'),revision=r.revision+1,updated_at=now() FROM automation_jobs j WHERE r.job_id=j.id AND r.status IN ('preparing','queued','running') AND (j.status IN ('failed','cancelled') OR (j.status='running' AND j.lease_until<now() AND j.attempts>=j.max_attempts))`)
			_, _ = s.DB.Exec(ctx, `UPDATE git_sync_runs SET snapshot_cipher=NULL,snapshot_size=0,status=CASE WHEN status IN ('preparing','preview','queued') THEN 'cancelled' ELSE status END,revision=revision+1 WHERE expires_at<now() AND (snapshot_cipher IS NOT NULL OR status IN ('preparing','preview','queued')) AND status NOT IN ('running','unknown')`)
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
}
