//go:build windows

package types

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolveContainerMountPathForHostProcess(t *testing.T) {
	sandbox := `C:\ProgramData\containerd\root\hpc\sandbox`
	t.Setenv(containerSandboxMountPointEnv, sandbox)

	require.Equal(t, filepath.Join(sandbox, `tls-files`), ResolveContainerMountPath(`C:\tls-files`))
	require.Equal(t, filepath.Join(sandbox, `var\lib\longhorn\tls`), ResolveContainerMountPath(`C:\var\lib\longhorn\tls`))
	require.Equal(t, filepath.Join(sandbox, `tls-files`), ResolveContainerMountPath(filepath.Join(sandbox, `tls-files`)))
}
