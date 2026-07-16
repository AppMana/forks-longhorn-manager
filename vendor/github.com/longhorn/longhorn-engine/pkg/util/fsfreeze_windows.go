package util

import (
	"fmt"
	"path/filepath"

	lhexec "github.com/longhorn/go-common-libs/exec"
	"k8s.io/mount-utils"
)

const DevicePathPrefix = ""

func GetFreezePointFromDevicePath(devicePath string) string {
	if devicePath == "" {
		return ""
	}
	return GetFreezePointFromVolumeName(filepath.Base(devicePath))
}

func GetFreezePointFromVolumeName(volumeName string) string {
	if volumeName == "" {
		return ""
	}
	return filepath.Join(`C:\var\lib\longhorn\freeze`, volumeName)
}

func GetDevicePathFromVolumeName(string) string {
	return ""
}

func FreezeFilesystem(string, lhexec.ExecuteInterface) error {
	return fmt.Errorf("filesystem freeze is not supported on Windows")
}

func UnfreezeFilesystem(string, lhexec.ExecuteInterface) (bool, error) {
	return false, nil
}

func UnfreezeAndUnmountFilesystem(string, lhexec.ExecuteInterface, mount.Interface) (bool, error) {
	return false, nil
}

func UnfreezeFilesystemForDevice(string) error {
	return nil
}
