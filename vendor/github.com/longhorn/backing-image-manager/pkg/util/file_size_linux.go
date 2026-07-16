package util

import (
	"syscall"

	"github.com/longhorn/backing-image-manager/pkg/types"
)

func getFileRealSize(filePath string) (int64, error) {
	var stat syscall.Stat_t
	if err := syscall.Stat(filePath, &stat); err != nil {
		return 0, err
	}
	return stat.Blocks * types.DefaultLinuxBlcokSize, nil
}
