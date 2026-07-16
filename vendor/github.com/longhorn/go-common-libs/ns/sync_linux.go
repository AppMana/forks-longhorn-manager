package ns

import "syscall"

func syncFilesystem() error {
	syscall.Sync()
	return nil
}
