package types

import (
	"fmt"
	"os"
	"unsafe"

	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
)

func getBlockDeviceSizePlatform(devicePath string) (uint64, error) {
	file, err := os.Open(devicePath)
	if err != nil {
		return 0, fmt.Errorf("failed to open block device at %s: %w", devicePath, err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			logrus.WithError(closeErr).Warnf("Failed to close block device %s", devicePath)
		}
	}()
	var size uint64
	_, _, errno := unix.Syscall(unix.SYS_IOCTL, file.Fd(), 0x80081272, uintptr(unsafe.Pointer(&size)))
	if errno != 0 {
		return 0, fmt.Errorf("failed to get block device size for %s: errno=%v", devicePath, errno)
	}
	return size, nil
}
