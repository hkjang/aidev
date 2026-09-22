package server

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/filemode"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// Only explicitly selected regular files are changed. Everything else,
// including unrelated root entries, is retained by object hash. No checkout,
// Git hooks, filters, credential helpers or executable commands are involved.
func gitSyncCandidate(ctx context.Context, repo gitSyncRepository, files []gitSyncFile, runID string, when time.Time) (plumbing.Hash, error) {
	if len(files) == 0 || len(files) > 5000 || !validID(runID) {
		return plumbing.ZeroHash, errors.New("Git 전송 파일 또는 실행 ID를 확인하세요")
	}
	type node struct {
		children map[string]*node
		file     *gitSyncFile
	}
	changes := &node{children: map[string]*node{}}
	seen := map[string]bool{}
	for i := range files {
		f := &files[i]
		if !gitSyncSafeFile(f.Path) || seen[strings.ToLower(f.Path)] || int64(len(f.Data)) > gitSyncMaxObjectBytes || gitSyncContentHash(f.Data) != f.Hash {
			return plumbing.ZeroHash, errors.New("Git 파일 경로·해시가 올바르지 않습니다")
		}
		seen[strings.ToLower(f.Path)] = true
		n := changes
		parts := strings.Split(f.Path, "/")
		for _, part := range parts {
			if n.file != nil {
				return plumbing.ZeroHash, errors.New("Git 파일과 폴더 경로가 충돌합니다")
			}
			if n.children[part] == nil {
				n.children[part] = &node{children: map[string]*node{}}
			}
			n = n.children[part]
		}
		if len(n.children) > 0 {
			return plumbing.ZeroHash, errors.New("Git 파일과 폴더 경로가 충돌합니다")
		}
		n.file = f
	}
	root := plumbing.ZeroHash
	if !repo.Head.IsZero() {
		c, e := object.GetCommit(repo.Store, repo.Head)
		if e != nil {
			return root, e
		}
		root = c.TreeHash
	}
	var write func(plumbing.Hash, *node) (plumbing.Hash, error)
	write = func(old plumbing.Hash, n *node) (plumbing.Hash, error) {
		if e := ctx.Err(); e != nil {
			return plumbing.ZeroHash, e
		}
		entries := map[string]object.TreeEntry{}
		if !old.IsZero() {
			t, e := object.GetTree(repo.Store, old)
			if e != nil {
				return plumbing.ZeroHash, e
			}
			for _, entry := range t.Entries {
				entries[entry.Name] = entry
			}
		}
		for name, child := range n.children {
			previous, exists := entries[name]
			for existing := range entries {
				if existing != name && strings.EqualFold(existing, name) {
					return plumbing.ZeroHash, errors.New("Git 대소문자 경로가 충돌합니다")
				}
			}
			var hash plumbing.Hash
			var e error
			mode := filemode.Regular
			if child.file != nil {
				if exists && previous.Mode != filemode.Regular {
					return plumbing.ZeroHash, errors.New("Git 비정규 파일 또는 폴더를 덮어쓸 수 없습니다")
				}
				encoded := repo.Store.NewEncodedObject()
				encoded.SetType(plumbing.BlobObject)
				encoded.SetSize(int64(len(child.file.Data)))
				w, err := encoded.Writer()
				if err != nil {
					return hash, err
				}
				_, err = w.Write(child.file.Data)
				closeErr := w.Close()
				if err != nil {
					return hash, err
				}
				if closeErr != nil {
					return hash, closeErr
				}
				hash, e = repo.Store.SetEncodedObject(encoded)
			} else {
				if exists && previous.Mode != filemode.Dir {
					return plumbing.ZeroHash, errors.New("Git 파일을 폴더로 바꿀 수 없습니다")
				}
				base := plumbing.ZeroHash
				if exists {
					base = previous.Hash
				}
				hash, e = write(base, child)
				mode = filemode.Dir
			}
			if e != nil {
				return plumbing.ZeroHash, e
			}
			entries[name] = object.TreeEntry{Name: name, Mode: mode, Hash: hash}
		}
		tree := &object.Tree{}
		for _, entry := range entries {
			tree.Entries = append(tree.Entries, entry)
		}
		sort.Sort(object.TreeEntrySorter(tree.Entries))
		encoded := repo.Store.NewEncodedObject()
		if e := tree.Encode(encoded); e != nil {
			return plumbing.ZeroHash, e
		}
		return repo.Store.SetEncodedObject(encoded)
	}
	tree, e := write(root, changes)
	if e != nil {
		return plumbing.ZeroHash, e
	}
	signature := object.Signature{Name: "madi", Email: "madi@localhost", When: when.UTC().Truncate(time.Second)}
	commit := &object.Commit{Author: signature, Committer: signature, TreeHash: tree, Message: "madi knowledge sync\n\nRun: " + runID + "\n"}
	if !repo.Head.IsZero() {
		commit.ParentHashes = []plumbing.Hash{repo.Head}
	}
	encoded := repo.Store.NewEncodedObject()
	if e = commit.Encode(encoded); e != nil {
		return plumbing.ZeroHash, e
	}
	return repo.Store.SetEncodedObject(encoded)
}

type gitSyncMapping struct{ Path, DocumentID, AttachmentID, LocalHash, RemoteHash string }
type gitSyncChange struct {
	Path         string `json:"path"`
	Action       string `json:"action"`
	Size         int    `json:"size"`
	DocumentID   string `json:"document_id,omitempty"`
	AttachmentID string `json:"attachment_id,omitempty"`
	Reason       string `json:"reason,omitempty"`
}

func gitSyncPushChanges(files []gitSyncFile, remote map[string]gitSyncFile, mappings map[string]gitSyncMapping) ([]gitSyncChange, bool) {
	changes := []gitSyncChange{}
	conflict := false
	for _, file := range files {
		previous, exists := remote[file.Path]
		mapping, mapped := mappings[file.Path]
		change := gitSyncChange{Path: file.Path, Size: len(file.Data), DocumentID: file.DocumentID, AttachmentID: file.AttachmentID, Action: "create"}
		switch {
		case exists && previous.Hash == file.Hash:
			change.Action = "unchanged"
		case exists && (!mapped || mapping.RemoteHash != previous.Hash):
			change.Action = "conflict"
			change.Reason = "원격 파일이 새로 생겼거나 마지막 동기화 이후 변경되었습니다"
			conflict = true
		case exists:
			change.Action = "update"
		case mapped && mapping.RemoteHash != "":
			change.Action = "conflict"
			change.Reason = "원격에서 파일이 삭제되었습니다. 자동 복구하지 않습니다"
			conflict = true
		}
		if mapped && mapping.DocumentID != "" && mapping.DocumentID != file.DocumentID {
			change.Action = "conflict"
			change.Reason = "다른 문서가 사용하는 경로입니다"
			conflict = true
		}
		changes = append(changes, change)
	}
	return changes, conflict
}
