package io

import (
	"fmt"
	"os"
	"path/filepath"
	"unsafe"

	"github.com/shirou/gopsutil/v3/disk"
	"golang.org/x/sys/windows"

	"github.com/longhorn/go-common-libs/types"
)

type fileStandardInfo struct {
	AllocationSize int64
	EndOfFile      int64
	NumberOfLinks  uint32
	DeletePending  byte
	Directory      byte
	Padding        [2]byte
}

func getDiskStatForPath(path string) (types.DiskStat, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return types.DiskStat{}, err
	}
	volume := filepath.VolumeName(absPath) + `\`
	volumePath, err := windows.UTF16PtrFromString(volume)
	if err != nil {
		return types.DiskStat{}, err
	}
	var serial, maxComponentLength, filesystemFlags uint32
	filesystemName := make([]uint16, windows.MAX_PATH+1)
	if err := windows.GetVolumeInformation(
		volumePath, nil, 0, &serial, &maxComponentLength, &filesystemFlags,
		&filesystemName[0], uint32(len(filesystemName)),
	); err != nil {
		return types.DiskStat{}, err
	}
	usage, err := disk.Usage(volume)
	if err != nil {
		return types.DiskStat{}, err
	}
	const blockSize = uint64(4096)
	return types.DiskStat{
		DiskID:           fmt.Sprintf("%08x", serial),
		Path:             path,
		Type:             windows.UTF16ToString(filesystemName),
		Driver:           types.DiskDriverNone,
		FreeBlocks:       int64(usage.Free / blockSize),
		TotalBlocks:      int64(usage.Total / blockSize),
		BlockSize:        int64(blockSize),
		StorageMaximum:   int64(usage.Total),
		StorageAvailable: int64(usage.Free),
	}, nil
}

func getFileAllocatedSize(path string) (uint64, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	info := fileStandardInfo{}
	if err := windows.GetFileInformationByHandleEx(
		windows.Handle(file.Fd()),
		windows.FileStandardInfo,
		(*byte)(unsafe.Pointer(&info)),
		uint32(unsafe.Sizeof(info)),
	); err != nil {
		return 0, err
	}
	return uint64(info.AllocationSize), nil
}
