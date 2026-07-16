//go:build !windows

package util

import (
	"os/exec"
	"syscall"
)

func ConfigureChildProcess(command *exec.Cmd) {
	command.SysProcAttr = &syscall.SysProcAttr{Pdeathsig: syscall.SIGKILL}
}
