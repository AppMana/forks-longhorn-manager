//go:build !windows

package backupstore

import (
	"os"
	"syscall"
)

func fillZeros(volume *os.File, offset, length int64) error {
	return syscall.Fallocate(int(volume.Fd()), 0, offset, length)
}
