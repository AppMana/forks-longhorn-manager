package backupstore

import (
	"os"
	"unsafe"

	"golang.org/x/sys/windows"
)

const fsctlSetZeroData = 0x000980c8

type fileZeroDataInformation struct {
	FileOffset      int64
	BeyondFinalZero int64
}

func fillZeros(volume *os.File, offset, length int64) error {
	zeroRange := fileZeroDataInformation{
		FileOffset:      offset,
		BeyondFinalZero: offset + length,
	}
	var returned uint32
	return windows.DeviceIoControl(
		windows.Handle(volume.Fd()), fsctlSetZeroData,
		(*byte)(unsafe.Pointer(&zeroRange)), uint32(unsafe.Sizeof(zeroRange)),
		nil, 0, &returned, nil,
	)
}
