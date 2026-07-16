package util

import (
	"fmt"
	"path/filepath"

	"golang.org/x/sys/windows"
)

func fstypeToKind(fstype int64) (string, error) {
	return "", fmt.Errorf("numeric filesystem type %v is not available on Windows", fstype)
}

func getMountKind(mountPoint string) (string, error) {
	absPath, err := filepath.Abs(mountPoint)
	if err != nil {
		return "", err
	}
	root := filepath.VolumeName(absPath) + `\`
	rootPath, err := windows.UTF16PtrFromString(root)
	if err != nil {
		return "", err
	}
	if windows.GetDriveType(rootPath) == windows.DRIVE_REMOTE {
		return "cifs", nil
	}

	filesystemName := make([]uint16, windows.MAX_PATH+1)
	if err := windows.GetVolumeInformation(
		rootPath, nil, 0, nil, nil, nil, &filesystemName[0], uint32(len(filesystemName)),
	); err != nil {
		return "", err
	}
	return windows.UTF16ToString(filesystemName), nil
}
