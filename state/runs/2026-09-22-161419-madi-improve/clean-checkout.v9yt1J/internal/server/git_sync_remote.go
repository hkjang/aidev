package server

import (
	"context"
	"errors"
	"io"
	"os"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/format/packfile"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/protocol/packp"
	"github.com/go-git/go-git/v5/plumbing/protocol/packp/capability"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/storage/memory"
)

type gitSyncRepository struct {
	Head  plumbing.Hash
	Store *memory.Storage
}

type gitSyncConflict struct{}

func (gitSyncConflict) Error() string {
	return "원격 Git 브랜치가 미리보기 이후 변경되었습니다. 덮어쓰지 않고 다시 비교해야 합니다"
}

// Returns uncertain=true after a receive-pack attempt whose final outcome was
// not confirmed. The caller must persist the candidate ID before calling and
// must not automatically replay an uncertain write; inspect that exact commit.
func gitSyncPush(ctx context.Context, remote *gitSyncRemote, branch string, repo gitSyncRepository, candidate plumbing.Hash, beforeLaunch func() error) (uncertain bool, err error) {
	commit, e := object.GetCommit(repo.Store, candidate)
	if e != nil {
		return false, errors.New("전송할 Git commit이 없습니다")
	}
	if repo.Head.IsZero() {
		if len(commit.ParentHashes) != 0 {
			return false, errors.New("새 Git 브랜치의 부모 commit이 올바르지 않습니다")
		}
	} else if len(commit.ParentHashes) != 1 || commit.ParentHashes[0] != repo.Head {
		return false, errors.New("Git 전송은 미리보기 원격 commit의 직접 후속 commit만 허용합니다")
	}
	session, e := remote.Transport.NewReceivePackSession(remote.Endpoint, remote.Auth)
	if e != nil {
		return false, errors.New("Git 쓰기 연결을 만들지 못했습니다")
	}
	defer session.Close()
	refs, e := session.AdvertisedReferencesContext(ctx)
	if e != nil && !errors.Is(e, transport.ErrEmptyRemoteRepository) {
		return false, errors.New("Git 쓰기 대상 브랜치를 확인하지 못했습니다")
	}
	if refs == nil {
		refs = packp.NewAdvRefs()
	}
	current := refs.References[plumbing.NewBranchReferenceName(branch).String()]
	if current == candidate {
		return false, nil
	}
	if current != repo.Head {
		return false, gitSyncConflict{}
	}
	if !refs.Capabilities.Supports(capability.ReportStatus) {
		return false, errors.New("Git 서버가 쓰기 결과 확인 프로토콜을 지원하지 않습니다")
	}
	iterator, e := repo.Store.IterEncodedObjects(plumbing.AnyObject)
	if e != nil {
		return false, e
	}
	defer iterator.Close()
	hashes := []plumbing.Hash{}
	var size int64
	if e = iterator.ForEach(func(obj plumbing.EncodedObject) error {
		hashes = append(hashes, obj.Hash())
		size += obj.Size()
		if len(hashes) > int(gitSyncMaxObjects) || size > gitSyncMaxInflatedBytes || obj.Size() > gitSyncMaxObjectBytes {
			return errGitSyncLimit
		}
		return nil
	}); e != nil {
		return false, e
	}
	f, e := os.CreateTemp("", "madi-git-push-*.pack")
	if e != nil {
		return false, e
	}
	defer f.Close()
	defer os.Remove(f.Name())
	if _, e = packfile.NewEncoder(&gitLimitWriter{ctx: ctx, out: f, remaining: gitSyncMaxPackBytes}, repo.Store, false).Encode(hashes, 0); e != nil {
		return false, errors.New("Git 전송 pack을 만들지 못했거나 크기 한도를 초과했습니다")
	}
	if _, e = f.Seek(0, io.SeekStart); e != nil {
		return false, e
	}
	if beforeLaunch != nil {
		if e = beforeLaunch(); e != nil {
			return false, e
		}
	}
	if e = ctx.Err(); e != nil {
		return false, e
	}
	request := packp.NewReferenceUpdateRequestFromCapabilities(refs.Capabilities)
	request.Commands = []*packp.Command{{Name: plumbing.NewBranchReferenceName(branch), Old: repo.Head, New: candidate}}
	request.Packfile = io.NopCloser(contextVaultReader{ctx, f})
	if remote.AllowPack != nil {
		remote.AllowPack()
	}
	status, e := session.ReceivePack(ctx, request)
	if e != nil || status == nil {
		return true, errors.New("Git 전송 결과가 불확실합니다. 같은 commit이 원격에 있는지 확인한 뒤 새 작업을 진행하세요")
	}
	if e = status.Error(); e != nil {
		return false, errors.New("Git 서버가 전송을 거부했습니다. 브랜치 보호·권한·동시 변경을 확인하세요")
	}
	return false, nil
}

// Fetch only the configured branch tip, never tags/submodules, and never check
// out a worktree. Thin packs and sideband output are intentionally not requested
// so that raw pack bytes can be bounded and validated before object parsing.
func gitSyncFetch(ctx context.Context, remote *gitSyncRemote, branch string) (gitSyncRepository, error) {
	empty := gitSyncRepository{Store: memory.NewStorage()}
	session, e := remote.Transport.NewUploadPackSession(remote.Endpoint, remote.Auth)
	if e != nil {
		return empty, errors.New("Git 읽기 연결에 실패했습니다. 주소·인증·TLS/SSH 키를 확인하세요")
	}
	defer session.Close()
	refs, e := session.AdvertisedReferencesContext(ctx)
	if errors.Is(e, transport.ErrEmptyRemoteRepository) {
		return empty, nil
	}
	if e != nil {
		return empty, errors.New("Git 브랜치 목록을 읽지 못했습니다")
	}
	if len(refs.References) > 10000 {
		return empty, errGitSyncLimit
	}
	head := refs.References[plumbing.NewBranchReferenceName(branch).String()]
	if head.IsZero() {
		return empty, nil
	}
	request := packp.NewUploadPackRequest()
	request.Wants = []plumbing.Hash{head}
	if refs.Capabilities.Supports(capability.Shallow) {
		request.Capabilities.Set(capability.Shallow)
		request.Depth = packp.DepthCommits(1)
	}
	if refs.Capabilities.Supports(capability.OFSDelta) {
		request.Capabilities.Set(capability.OFSDelta)
	}
	if refs.Capabilities.Supports(capability.NoProgress) {
		request.Capabilities.Set(capability.NoProgress)
	}
	if remote.AllowPack != nil {
		remote.AllowPack()
	}
	response, e := session.UploadPack(ctx, request)
	if e != nil {
		return empty, errors.New("Git 브랜치 내용을 가져오지 못했습니다")
	}
	defer response.Close()
	f, e := os.CreateTemp("", "madi-git-pack-*.pack")
	if e != nil {
		return empty, e
	}
	defer f.Close()
	defer os.Remove(f.Name())
	writer := &gitLimitWriter{ctx: ctx, out: f, remaining: gitSyncMaxPackBytes}
	if _, e = io.Copy(writer, contextVaultReader{ctx, response}); e != nil {
		return empty, errors.New("Git pack 수신이 실패했거나 크기 한도를 초과했습니다")
	}
	store, e := gitSyncDecodePack(ctx, f)
	if e != nil {
		return empty, e
	}
	if _, e = object.GetCommit(store, head); e != nil {
		return empty, errors.New("Git 브랜치가 유효한 commit을 가리키지 않습니다")
	}
	return gitSyncRepository{Head: head, Store: store}, nil
}
