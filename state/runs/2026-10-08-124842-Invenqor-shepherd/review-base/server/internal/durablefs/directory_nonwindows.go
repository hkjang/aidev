//go:build !windows

package durablefs

import "os"

func syncDirectory(directory *os.File) error {
	return directory.Sync()
}
