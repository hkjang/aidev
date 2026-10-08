//go:build windows

package durablefs

import "os"

func syncDirectory(_ *os.File) error {
	return nil
}
