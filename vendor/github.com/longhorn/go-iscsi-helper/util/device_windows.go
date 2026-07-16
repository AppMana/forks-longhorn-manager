package util

import "fmt"

func mknod(device string, major, minor int) error {
	return fmt.Errorf("duplicating Linux block device %q is not supported on Windows", device)
}
