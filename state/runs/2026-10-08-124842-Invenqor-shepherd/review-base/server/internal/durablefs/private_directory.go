package durablefs

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
)

// EnsurePrivateDirectory secures directories created by Invenqor while also
// supporting Kubernetes PVC mount roots. A CSI-mounted root is commonly owned
// by root:fsGroup with mode 0770, so an unprivileged Pod can write it but cannot
// chmod it. That one case is accepted only after verifying there are no other
// permissions and proving the current process can create a 0600 file.
func EnsurePrivateDirectory(path string) error {
	return ensurePrivateDirectory(path, os.Chmod)
}

func ensurePrivateDirectory(
	path string,
	chmod func(string, os.FileMode) error,
) error {
	if err := os.MkdirAll(path, 0o700); err != nil {
		return fmt.Errorf("create private directory: %w", err)
	}
	chmodErr := chmod(path, 0o700)
	if chmodErr == nil {
		return nil
	}
	if !errors.Is(chmodErr, fs.ErrPermission) {
		return fmt.Errorf("secure private directory: %w", chmodErr)
	}
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("inspect group-writable directory: %w", err)
	}
	// 0770 is the maximum accepted permission set. In particular, any access
	// for "other" users remains a hard failure even if the directory is writable.
	if !info.IsDir() || info.Mode().Perm()&^os.FileMode(0o770) != 0 {
		return fmt.Errorf(
			"secure private directory: chmod failed and mode %04o is not private: %w",
			info.Mode().Perm(),
			chmodErr,
		)
	}
	probe, err := os.CreateTemp(path, ".invenqor-write-probe-*")
	if err != nil {
		return fmt.Errorf("verify group-writable directory: %w", err)
	}
	probePath := probe.Name()
	defer os.Remove(probePath)
	if err := probe.Chmod(0o600); err != nil {
		_ = probe.Close()
		return fmt.Errorf("secure directory write probe: %w", err)
	}
	if err := probe.Close(); err != nil {
		return fmt.Errorf("close directory write probe: %w", err)
	}
	return nil
}
