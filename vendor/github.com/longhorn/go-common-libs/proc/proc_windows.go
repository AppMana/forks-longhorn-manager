package proc

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/mitchellh/go-ps"
)

// ProcessStatus mirrors the small subset of Linux process status used by the
// namespace helpers. Windows has no process namespaces, so ancestor traversal
// is intentionally unavailable.
type ProcessStatus struct {
	Name string
	Pid  uint64
	PPid uint64
}

type ProcessFinder struct{ procDirectory string }

func NewProcFinder(procDir string) *ProcessFinder { return &ProcessFinder{procDirectory: procDir} }

func (p *ProcessFinder) GetProcessStatus(pid string) (*ProcessStatus, error) {
	wanted, err := strconv.Atoi(pid)
	if err != nil {
		return nil, err
	}
	processes, err := ps.Processes()
	if err != nil {
		return nil, err
	}
	for _, process := range processes {
		if process.Pid() == wanted {
			return &ProcessStatus{Name: process.Executable(), Pid: uint64(process.Pid()), PPid: uint64(process.PPid())}, nil
		}
	}
	return nil, fmt.Errorf("process %d is not found", wanted)
}

func (p *ProcessFinder) FindAncestorByName(ancestorProcess, pid string) (*ProcessStatus, error) {
	return nil, fmt.Errorf("process namespace ancestry is not supported on Windows")
}

func GetProcessPIDs(processName, _ string) ([]uint64, error) {
	processes, err := ps.Processes()
	if err != nil {
		return nil, err
	}
	var result []uint64
	for _, process := range processes {
		name := strings.TrimSuffix(process.Executable(), ".exe")
		if strings.EqualFold(name, strings.TrimSuffix(processName, ".exe")) {
			result = append(result, uint64(process.Pid()))
		}
	}
	if len(result) == 0 {
		return []uint64{1}, nil
	}
	return result, nil
}

func GetHostNamespacePID(string) uint64 { return 1 }

func GetNamespaceDirectory(procDir, pid string) string { return filepath.Join(procDir, pid, "ns") }

func GetHostNamespaceDirectory(procDir string) string { return GetNamespaceDirectory(procDir, "1") }

func GetProcessAncestorNamespaceDirectory(process, procDir string) (string, error) {
	return "", fmt.Errorf("process namespaces are not supported on Windows")
}

func GetProcessNamespaceDirectory(processName, procDir string) (string, error) {
	return "", fmt.Errorf("process namespaces are not supported on Windows")
}

func FindProcessByName(name string) (*os.Process, error) {
	processes, err := ps.Processes()
	if err != nil {
		return nil, err
	}
	for _, process := range processes {
		if strings.EqualFold(process.Executable(), name) || strings.EqualFold(process.Executable(), name+".exe") {
			return os.FindProcess(process.Pid())
		}
	}
	return nil, fmt.Errorf("process %s is not found", name)
}

func FindProcessByCmdline(cmdline string) ([]*os.Process, error) {
	return nil, fmt.Errorf("process command-line discovery for %q is not supported on Windows", cmdline)
}
