package util

import "os/exec"

// Windows job-object ownership is supplied by the instance manager. There is
// no parent-death signal field in CreateProcess.
func ConfigureChildProcess(*exec.Cmd) {}
