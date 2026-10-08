package durablefs

import (
	"fmt"
	"os"
)

// SyncDirectory makes a directory entry durable on platforms that support
// directory fsync. Windows does not support syncing an opened directory with
// os.File.Sync, but the directory is still opened, validated, and closed so
// callers receive consistent path and handle validation on every platform.
func SyncDirectory(path string) (err error) {
	directory, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open directory %q: %w", path, err)
	}
	defer func() {
		if closeErr := directory.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("close directory %q: %w", path, closeErr)
		}
	}()

	info, err := directory.Stat()
	if err != nil {
		return fmt.Errorf("inspect directory %q: %w", path, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("path %q is not a directory", path)
	}
	if err := syncDirectory(directory); err != nil {
		return fmt.Errorf("sync directory %q: %w", path, err)
	}
	return nil
}
