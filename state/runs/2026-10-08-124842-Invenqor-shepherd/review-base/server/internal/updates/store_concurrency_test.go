package updates

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestTwoStoreInstancesShareOneMutationLock(t *testing.T) {
	root := t.TempDir()
	first, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	lock, err := first.acquireFileLock()
	if err != nil {
		t.Fatal(err)
	}
	acquired := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		done <- second.withFileLock(func() error {
			close(acquired)
			return nil
		})
	}()
	select {
	case <-acquired:
		t.Fatal("a second Store acquired the shared mutation lock concurrently")
	case <-time.After(100 * time.Millisecond):
	}
	if err := lock.release(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-acquired:
	case <-time.After(3 * time.Second):
		t.Fatal("the shared mutation lock did not wake its next owner")
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

func TestStoreLockCrashHelper(t *testing.T) {
	root := os.Getenv("INVENQOR_TEST_UPDATE_LOCK_ROOT")
	ready := os.Getenv("INVENQOR_TEST_UPDATE_LOCK_READY")
	if root == "" || ready == "" {
		t.Skip("subprocess helper")
	}
	store, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	lock, err := store.acquireFileLock()
	if err != nil {
		t.Fatal(err)
	}
	defer lock.release()
	if err := os.WriteFile(ready, []byte("locked"), 0o600); err != nil {
		t.Fatal(err)
	}
	select {}
}

func TestStoreLockIsReleasedWhenItsOwnerCrashes(t *testing.T) {
	root := t.TempDir()
	ready := filepath.Join(t.TempDir(), "ready")
	command := exec.Command(os.Args[0], "-test.run=^TestStoreLockCrashHelper$")
	command.Env = append(os.Environ(),
		"INVENQOR_TEST_UPDATE_LOCK_ROOT="+root,
		"INVENQOR_TEST_UPDATE_LOCK_READY="+ready,
	)
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = command.Process.Kill() }()
	deadline := time.Now().Add(5 * time.Second)
	for !fileExists(ready) && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if !fileExists(ready) {
		t.Fatal("lock helper did not acquire the store lock")
	}
	if err := command.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	_ = command.Wait()

	store, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	acquired := make(chan error, 1)
	go func() {
		lock, lockErr := store.acquireFileLock()
		if lockErr == nil {
			lockErr = lock.release()
		}
		acquired <- lockErr
	}()
	select {
	case err := <-acquired:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("a crashed owner left a stale update store lock")
	}
	if !fileExists(filepath.Join(root, ".store.lock")) {
		t.Fatal("the persistent lock file unexpectedly disappeared")
	}
}

func unsignedV2TestManifest(version string) Manifest {
	return Manifest{
		Version: version, Channel: "stable", OS: "linux", Architecture: "x86_64",
		Signature: testSignature, ManifestSignature: testSignature,
		SignatureScheme: SignatureSchemeEd25519, SignatureVersion: SignatureVersionV2,
		Rollout: 100,
	}
}

func TestConcurrentRetireAndRolloutNeverResurrectAManifest(t *testing.T) {
	root := t.TempDir()
	first, _ := Open(root)
	second, _ := Open(root)
	for index := 0; index < 40; index++ {
		version := fmt.Sprintf("1.0.%d", index)
		base := version + "-linux-x86_64"
		if _, err := first.Publish(
			unsignedV2TestManifest(version), bytes.NewReader([]byte("agent-"+version)),
		); err != nil {
			t.Fatal(err)
		}
		start := make(chan struct{})
		var wait sync.WaitGroup
		wait.Add(2)
		go func() {
			defer wait.Done()
			<-start
			_, _ = first.SetRollout(base, 25)
		}()
		var retireErr error
		go func() {
			defer wait.Done()
			<-start
			retireErr = second.Retire(base)
		}()
		close(start)
		wait.Wait()
		if retireErr != nil {
			t.Fatal(retireErr)
		}
		if fileExists(filepath.Join(root, base+".json")) ||
			fileExists(filepath.Join(root, base+".bin")) {
			t.Fatalf("release %s was resurrected or only partially retired", base)
		}
	}
}

func TestStoreRepairsHalfCommittedReleaseInvariant(t *testing.T) {
	root := t.TempDir()
	store, _ := Open(root)
	base := "1.2.3-linux-x86_64"
	if err := os.WriteFile(filepath.Join(root, base+".bin"), []byte("orphan"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Publish(
		unsignedV2TestManifest("1.2.3"), bytes.NewReader([]byte("complete")),
	); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(root, base+".bin")); err != nil {
		t.Fatal(err)
	}
	if _, err := store.SetRollout(base, 50); err == nil {
		t.Fatal("SetRollout accepted a manifest whose artifact was missing")
	}
	if fileExists(filepath.Join(root, base+".json")) {
		t.Fatal("SetRollout preserved an artifact-less manifest")
	}
}
