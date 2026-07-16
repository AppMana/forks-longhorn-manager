package util

import (
	"os"

	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
)

func mknod(device string, major, minor int) error {
	fileMode := os.FileMode(0660) | unix.S_IFBLK
	dev := int(unix.Mkdev(uint32(major), uint32(minor)))
	logrus.Infof("Creating device %s %d:%d", device, major, minor)
	return unix.Mknod(device, uint32(fileMode), dev)
}
