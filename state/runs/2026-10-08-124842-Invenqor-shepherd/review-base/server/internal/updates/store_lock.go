package updates

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// storeFileLock is an OS-backed advisory lock shared by every Server process
// mounting the update store. The lock file is intentionally persistent: the OS
// releases the lock when a process crashes, so there is no stale PID/lease to
// guess at or delete while another Pod may own it.
type storeFileLock struct {
	file *os.File
}

func (s *Store) withFileLock(operation func() error) (err error) {
	lock, err := s.acquireFileLock()
	if err != nil {
		return err
	}
	defer func() {
		err = errors.Join(err, lock.release())
	}()
	return operation()
}

func (s *Store) acquireFileLock() (*storeFileLock, error) {
	path := filepath.Join(s.root, ".store.lock")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open update store lock: %w", err)
	}
	if err := lockUpdateStoreFile(file); err != nil {
		file.Close()
		return nil, fmt.Errorf("lock update store: %w", err)
	}
	return &storeFileLock{file: file}, nil
}

func (lock *storeFileLock) release() error {
	if lock == nil || lock.file == nil {
		return nil
	}
	err := unlockUpdateStoreFile(lock.file)
	closeErr := lock.file.Close()
	lock.file = nil
	if err != nil {
		return fmt.Errorf("unlock update store: %w", err)
	}
	if closeErr != nil {
		return fmt.Errorf("close update store lock: %w", closeErr)
	}
	return nil
}
