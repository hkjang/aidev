package server

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport"
)

func (s *Server) inspectGitSync(w http.ResponseWriter, r *http.Request) {
	run, c, e := s.gitSyncVisibleRun(r)
	if e != nil {
		apiError(w, 404, "Git 작업을 찾을 수 없습니다")
		return
	}
	if run.Status != "unknown" || run.Direction != "push" || run.CandidateCommit == "" {
		apiError(w, 409, "확인할 원격 전송 commit이 없습니다")
		return
	}
	policy, _, e := s.gitSyncSettings(r.Context())
	if e != nil || !boolean(policy, "enabled") || !c.Enabled {
		apiError(w, 400, "현재 원격 연결과 Git 정책을 먼저 확인·활성화하세요")
		return
	}
	config, e := s.gitSyncDecryptedConfig(c)
	if e != nil {
		apiError(w, 400, e.Error())
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	remote, e := newGitSyncRemote(ctx, config, listStrings(policy["allowed_hosts"]), boolean(policy, "allow_private_networks"))
	if e != nil {
		apiError(w, 400, e.Error())
		return
	}
	defer remote.Close()
	current, e := gitSyncInspectHead(ctx, remote, str(config, "branch"))
	if e != nil {
		apiError(w, 502, e.Error())
		return
	}
	if current != run.CandidateCommit {
		jsonResponse(w, 200, map[string]any{"status": "unknown", "commit": current, "message": "현재 원격 브랜치가 후보 commit과 다릅니다. 쓰기를 재시도하지 않았습니다. Git 관리자와 원격 이력을 확인하세요"})
		return
	}
	snapshot, e := s.gitSyncSnapshot(ctx, run.ID)
	if e != nil {
		apiError(w, 409, "후보 commit은 확인했으나 복원·만료로 미리보기가 없어 자동 완료 처리하지 않았습니다")
		return
	}
	_, e = s.gitSyncFinishPush(ctx, run, c, snapshot, current)
	if e != nil {
		respond(w, nil, e)
		return
	}
	s.audit(r, "GIT_SYNC_INSPECT", run.ID, map[string]any{"commit": current})
	jsonResponse(w, 200, map[string]any{"status": "succeeded", "commit": current, "message": "저장된 후보 commit과 현재 원격 브랜치가 일치합니다. 추가 전송 없이 완료 처리했습니다"})
}
func gitSyncInspectHead(ctx context.Context, remote *gitSyncRemote, branch string) (string, error) {
	session, e := remote.Transport.NewUploadPackSession(remote.Endpoint, remote.Auth)
	if e != nil {
		return "", errors.New("Git 읽기 연결을 확인할 수 없습니다")
	}
	defer session.Close()
	refs, e := session.AdvertisedReferencesContext(ctx)
	if errors.Is(e, transport.ErrEmptyRemoteRepository) {
		return plumbing.ZeroHash.String(), nil
	}
	if e != nil || refs == nil || len(refs.References) > 10000 {
		return "", errors.New("Git 원격 브랜치 결과를 확인할 수 없습니다")
	}
	return refs.References[plumbing.NewBranchReferenceName(branch).String()].String(), nil
}
