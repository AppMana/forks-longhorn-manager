//go:build !windows

package io

import (
	"fmt"
	"syscall"

	"github.com/shirou/gopsutil/v3/disk"
	"golang.org/x/sys/unix"

	"github.com/longhorn/go-common-libs/types"
)

func getDiskStatForPath(path string) (types.DiskStat, error) {
	var statfs unix.Statfs_t
	if err := unix.Statfs(path, &statfs); err != nil {
		return types.DiskStat{}, err
	}
	usage, err := disk.Usage(path)
	if err != nil {
		return types.DiskStat{}, err
	}

	var fsidValue uint64
	for _, component := range statfs.Fsid.Val {
		fsidValue = (fsidValue << 32) | uint64(uint32(component))
	}
	return types.DiskStat{
		DiskID:           fmt.Sprintf("%012x", fsidValue),
		Path:             path,
		Type:             usage.Fstype,
		Driver:           types.DiskDriverNone,
		FreeBlocks:       int64(statfs.Bfree),
		TotalBlocks:      int64(statfs.Blocks),
		BlockSize:        int64(statfs.Bsize),
		StorageMaximum:   int64(statfs.Blocks) * int64(statfs.Bsize),
		StorageAvailable: int64(statfs.Bfree) * int64(statfs.Bsize),
	}, nil
}

func getFileAllocatedSize(path string) (uint64, error) {
	var stat syscall.Stat_t
	if err := syscall.Stat(path, &stat); err != nil {
		return 0, err
	}
	return uint64(stat.Blocks) * uint64(stat.Blksize), nil
}
