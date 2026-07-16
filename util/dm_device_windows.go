package util

import "fmt"

func removeDMDevice(devicePath string) error {
	return fmt.Errorf("device-mapper removal for %q is not supported on Windows", devicePath)
}
