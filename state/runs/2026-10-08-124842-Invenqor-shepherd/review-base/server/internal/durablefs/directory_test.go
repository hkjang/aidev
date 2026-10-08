package durablefs

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSyncDirectoryValidatesAndClosesDirectory(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "durable")
	if err := os.Mkdir(directory, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := SyncDirectory(directory); err != nil {
		t.Fatalf("SyncDirectory() error = %v", err)
	}
	if err := os.Remove(directory); err != nil {
		t.Fatalf("Remove() after SyncDirectory() error = %v", err)
	}
}

func TestSyncDirectoryRejectsMissingPath(t *testing.T) {
	err := SyncDirectory(filepath.Join(t.TempDir(), "missing"))
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("SyncDirectory() error = %v, want os.ErrNotExist", err)
	}
}

func TestSyncDirectoryRejectsRegularFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(path, []byte("content"), 0o600); err != nil {
		t.Fatal(err)
	}
	err := SyncDirectory(path)
	if err == nil || !strings.Contains(err.Error(), "is not a directory") {
		t.Fatalf("SyncDirectory() error = %v, want not-a-directory error", err)
	}
}
