//go:build !windows

package updates

import (
	"os"

	"golang.org/x/sys/unix"
)

func lockUpdateStoreFile(file *os.File) error {
	return unix.Flock(int(file.Fd()), unix.LOCK_EX)
}

func unlockUpdateStoreFile(file *os.File) error {
	return unix.Flock(int(file.Fd()), unix.LOCK_UN)
}
