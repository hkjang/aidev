package server

import (
	"context"
	"errors"
	"testing"
	"time"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/filemode"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport"
	gitServer "github.com/go-git/go-git/v5/plumbing/transport/server"
)

func TestGitSyncTaskLocalBareFetch(t *testing.T) {
	dir := t.TempDir()
	repo, e := git.PlainInit(dir, true)
	if e != nil {
		t.Fatal(e)
	}
	blob := repo.Storer.NewEncodedObject()
	blob.SetType(plumbing.BlobObject)
	data := []byte("# 로컬 bare 검증\n")
	blob.SetSize(int64(len(data)))
	w, _ := blob.Writer()
	w.Write(data)
	w.Close()
	blobHash, e := repo.Storer.SetEncodedObject(blob)
	if e != nil {
		t.Fatal(e)
	}
	tree := &object.Tree{Entries: []object.TreeEntry{{Name: "note.md", Mode: filemode.Regular, Hash: blobHash}}}
	encoded := repo.Storer.NewEncodedObject()
	if e = tree.Encode(encoded); e != nil {
		t.Fatal(e)
	}
	treeHash, e := repo.Storer.SetEncodedObject(encoded)
	if e != nil {
		t.Fatal(e)
	}
	commit := &object.Commit{Author: object.Signature{Name: "madi test", Email: "test@example.invalid", When: time.Unix(1700000000, 0)}, Committer: object.Signature{Name: "madi test", Email: "test@example.invalid", When: time.Unix(1700000000, 0)}, TreeHash: treeHash, Message: "isolated fixture"}
	encoded = repo.Storer.NewEncodedObject()
	if e = commit.Encode(encoded); e != nil {
		t.Fatal(e)
	}
	head, e := repo.Storer.SetEncodedObject(encoded)
	if e != nil {
		t.Fatal(e)
	}
	if e = repo.Storer.SetReference(plumbing.NewHashReference(plumbing.NewBranchReferenceName("main"), head)); e != nil {
		t.Fatal(e)
	}
	endpoint, e := transport.NewEndpoint(dir)
	if e != nil {
		t.Fatal(e)
	}
	// file endpoints are accepted only by this in-process test fixture. Runtime
	// configuration rejects all filesystem/git-daemon/custom URL protocols.
	remote := &gitSyncRemote{Endpoint: endpoint, Transport: gitServer.NewClient(gitServer.DefaultLoader), Close: func() {}}
	got, e := gitSyncFetch(context.Background(), remote, "main")
	if e != nil {
		t.Fatal(e)
	}
	if got.Head != head {
		t.Fatal("fetch returned wrong commit")
	}
	if _, e = got.Store.EncodedObject(plumbing.BlobObject, blobHash); e != nil {
		t.Fatal("selected branch blob absent", e)
	}
	child := func(message string) plumbing.Hash {
		commit := &object.Commit{Author: object.Signature{Name: "madi test", Email: "test@example.invalid", When: time.Unix(1700000001, 0)}, Committer: object.Signature{Name: "madi test", Email: "test@example.invalid", When: time.Unix(1700000001, 0)}, TreeHash: treeHash, ParentHashes: []plumbing.Hash{head}, Message: message}
		encoded := got.Store.NewEncodedObject()
		if e := commit.Encode(encoded); e != nil {
			t.Fatal(e)
		}
		hash, e := got.Store.SetEncodedObject(encoded)
		if e != nil {
			t.Fatal(e)
		}
		return hash
	}
	candidate := child("explicit preview and confirmation")
	launches := 0
	if uncertain, e := gitSyncPush(context.Background(), remote, "main", got, candidate, func() error { launches++; return nil }); e != nil || uncertain {
		t.Fatal("bounded bare push failed", uncertain, e)
	}
	actual, e := repo.Storer.Reference(plumbing.NewBranchReferenceName("main"))
	if e != nil || actual.Hash() != candidate {
		t.Fatal("remote ref was not atomically updated", e)
	}
	if uncertain, e := gitSyncPush(context.Background(), remote, "main", got, candidate, func() error { launches++; return nil }); e != nil || uncertain || launches != 1 {
		t.Fatal("same commit replay was not idempotent", uncertain, e, launches)
	}
	_, e = gitSyncPush(context.Background(), remote, "main", got, child("stale competitor"), nil)
	var conflict gitSyncConflict
	if !errors.As(e, &conflict) {
		t.Fatal("stale remote unexpectedly overwritten", e)
	}
	actual, e = repo.Storer.Reference(plumbing.NewBranchReferenceName("main"))
	if e != nil || actual.Hash() != candidate {
		t.Fatal("conflict overwrote remote", e)
	}
}
