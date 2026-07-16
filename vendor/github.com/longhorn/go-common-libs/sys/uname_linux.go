package sys

import (
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
)

func getArch() (string, error) {
	utsname := &unix.Utsname{}
	if err := unix.Uname(utsname); err != nil {
		logrus.WithError(err).Warn("Failed to get system architecture")
		return "", err
	}
	return utsField(utsname.Machine[:]), nil
}

func getKernelRelease() (string, error) {
	utsname := &unix.Utsname{}
	if err := unix.Uname(utsname); err != nil {
		logrus.WithError(err).Warn("Failed to get kernel release")
		return "", err
	}
	return utsField(utsname.Release[:]), nil
}

func utsField(field []byte) string {
	result := make([]byte, 0, len(field))
	for _, b := range field {
		if b == 0 {
			break
		}
		result = append(result, b)
	}
	return string(result)
}
