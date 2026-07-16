package types

import "fmt"

func getBlockDeviceSizePlatform(devicePath string) (uint64, error) {
	return 0, fmt.Errorf("block-type Longhorn disks are not supported on Windows: %s", devicePath)
}
