package sys

import "fmt"

// FindBlockDeviceForMount is a Linux block-device operation. Windows disks are
// discovered through the Windows storage APIs by the Longhorn disk monitor.
func FindBlockDeviceForMount(mountPath string) (string, error) {
	return "", fmt.Errorf("block device discovery for mount %q is not supported on Windows", mountPath)
}

// ResolveBlockDeviceToPhysicalDevice is a Linux sysfs operation.
func ResolveBlockDeviceToPhysicalDevice(blockDevice string) (string, error) {
	return "", fmt.Errorf("physical block device discovery for %q is not supported on Windows", blockDevice)
}
