//go:build !windows

package util

import (
	"fmt"
	"syscall"

	"golang.org/x/sys/unix"
)

func fstypeToKind(fstype int64) (string, error) {
	switch fstype {
	case unix.NFS_SUPER_MAGIC:
		return "nfs", nil
	case unix.CIFS_SUPER_MAGIC, unix.SMB2_SUPER_MAGIC, unix.SMB_SUPER_MAGIC:
		return "cifs", nil
	default:
		return "", fmt.Errorf("unknown fstype %v", fstype)
	}
}

func getMountKind(mountPoint string) (string, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(mountPoint, &stat); err != nil {
		return "", err
	}
	return fstypeToKind(int64(stat.Type))
}
