package durablefs

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestEnsurePrivateDirectoryAcceptsFsGroupWritablePVCWhenChmodIsDenied(
	t *testing.T,
) {
	path := filepath.Join(t.TempDir(), "pvc-root")
	if err := os.Mkdir(path, 0o770); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o770); err != nil {
		t.Fatal(err)
	}
	err := ensurePrivateDirectory(path, func(string, os.FileMode) error {
		return os.ErrPermission
	})
	if err != nil {
		t.Fatalf("ensurePrivateDirectory(fsGroup PVC) error = %v", err)
	}
	entries, err := os.ReadDir(path)
	if err != nil || len(entries) != 0 {
		t.Fatalf("write probe was not cleaned up: %#v, %v", entries, err)
	}
}

func TestEnsurePrivateDirectoryRejectsOtherAccessiblePVC(t *testing.T) {
	path := filepath.Join(t.TempDir(), "unsafe-pvc-root")
	if err := os.Mkdir(path, 0o777); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o777); err != nil {
		t.Fatal(err)
	}
	err := ensurePrivateDirectory(path, func(string, os.FileMode) error {
		return os.ErrPermission
	})
	if err == nil || !errors.Is(err, os.ErrPermission) {
		t.Fatalf("unsafe PVC error = %v, want permission failure", err)
	}
}
