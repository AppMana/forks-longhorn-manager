package util

import (
	"fmt"
	"os"
	"unsafe"

	"github.com/sirupsen/logrus"
	"golang.org/x/sys/windows"
)

type fileStandardInfo struct {
	AllocationSize int64
	EndOfFile      int64
	NumberOfLinks  uint32
	DeletePending  byte
	Directory      byte
	Padding        [2]byte
}

func DuplicateDevice(src, dest string) error {
	return fmt.Errorf("block device duplication is not supported on Windows: %s to %s", src, dest)
}

func getFileAllocationSize(path string) (int64, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer file.Close()
	info := fileStandardInfo{}
	if err := windows.GetFileInformationByHandleEx(
		windows.Handle(file.Fd()), windows.FileStandardInfo,
		(*byte)(unsafe.Pointer(&info)), uint32(unsafe.Sizeof(info)),
	); err != nil {
		return 0, err
	}
	return info.AllocationSize, nil
}

func GetFileActualSize(path string) int64 {
	size, err := getFileAllocationSize(path)
	if err != nil {
		logrus.WithError(err).Errorf("Failed to get size of file %v", path)
		return -1
	}
	return size
}

func GetHeadFileModifyTimeAndSize(path string) (int64, int64, error) {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return 0, 0, err
	}
	size, err := getFileAllocationSize(path)
	if err != nil {
		return 0, 0, err
	}
	return int64(fileInfo.ModTime().Nanosecond()), size, nil
}
