//go:build !windows

package util

import (
	"fmt"
	"os"
	"syscall"

	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
)

func DuplicateDevice(src, dest string) error {
	stat := unix.Stat_t{}
	if err := unix.Stat(src, &stat); err != nil {
		return fmt.Errorf("cannot duplicate device because cannot find %s: %v", src, err)
	}
	major := int(stat.Rdev / 256)
	minor := int(stat.Rdev % 256)
	if err := mknod(dest, major, minor); err != nil {
		return fmt.Errorf("cannot duplicate device %s to %s", src, dest)
	}
	if err := os.Chmod(dest, 0660); err != nil {
		return fmt.Errorf("couldn't change permission of the device %s: %s", dest, err)
	}
	return nil
}

func mknod(device string, major, minor int) error {
	var fileMode os.FileMode = 0660
	fileMode |= unix.S_IFBLK
	dev := int((major << 8) | (minor & 0xff) | ((minor & 0xfff00) << 12))
	logrus.Infof("Creating device %s %d:%d", device, major, minor)
	return unix.Mknod(device, uint32(fileMode), dev)
}

func GetFileActualSize(file string) int64 {
	var stat syscall.Stat_t
	if err := syscall.Stat(file, &stat); err != nil {
		logrus.WithError(err).Errorf("Failed to get size of file %v", file)
		return -1
	}
	return stat.Blocks * BlockSizeLinux
}

func GetHeadFileModifyTimeAndSize(file string) (int64, int64, error) {
	var stat syscall.Stat_t
	if err := syscall.Stat(file, &stat); err != nil {
		return 0, 0, err
	}
	return stat.Mtim.Nano(), stat.Blocks * BlockSizeLinux, nil
}
