package sys

import (
	"fmt"
	"runtime"

	"golang.org/x/sys/windows"
)

func getArch() (string, error) {
	switch runtime.GOARCH {
	case "amd64":
		return "x86_64", nil
	case "arm64":
		return "aarch64", nil
	default:
		return runtime.GOARCH, nil
	}
}

func getKernelRelease() (string, error) {
	version := windows.RtlGetVersion()
	return fmt.Sprintf("%d.%d.%d", version.MajorVersion, version.MinorVersion, version.BuildNumber), nil
}
