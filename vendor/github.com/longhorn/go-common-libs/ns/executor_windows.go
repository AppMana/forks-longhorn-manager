package ns

import (
	"time"

	commonexec "github.com/longhorn/go-common-libs/exec"
	"github.com/longhorn/go-common-libs/types"
)

// Executor preserves the namespace-executor API while executing directly in a
// HostProcess container, which already shares the Windows host.
type Executor struct {
	namespaces  []types.Namespace
	nsDirectory string
	executor    commonexec.ExecuteInterface
}

func NewNamespaceExecutor(string, string, []types.Namespace) (*Executor, error) {
	return &Executor{executor: commonexec.NewExecutor()}, nil
}

func (nsexec *Executor) Execute(envs []string, binary string, args []string, timeout time.Duration) (string, error) {
	return nsexec.executor.Execute(envs, binary, args, timeout)
}

func (nsexec *Executor) ExecuteWithStdin(_ []string, binary string, args []string, stdinString string, timeout time.Duration) (string, error) {
	return nsexec.executor.ExecuteWithStdin(binary, args, stdinString, timeout)
}

func (nsexec *Executor) ExecuteWithStdinPipe(_ []string, binary string, args []string, stdinString string, timeout time.Duration) (string, error) {
	return nsexec.executor.ExecuteWithStdinPipe(binary, args, stdinString, timeout)
}
