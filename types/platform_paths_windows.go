package types

import (
	"path/filepath"
	"strings"
)

const (
	EngineBinaryName       = "longhorn.exe"
	WindowsDefaultDataPath = `C:\var\lib\longhorn`
)

func engineBinaryDirectoryOnHost() string { return `C:\var\lib\longhorn\engine-binaries` }

func engineBinaryDirectoryInContainer() string { return engineBinaryDirectoryOnHost() }

func engineBinaryDirectoryForReplicaManager() string { return engineBinaryDirectoryOnHost() }

func ResolveDefaultDataPath(configured string) string {
	cleaned := strings.TrimRight(filepath.ToSlash(configured), "/")
	if cleaned == "/var/lib/longhorn" {
		return WindowsDefaultDataPath
	}
	return configured
}

// HostProcess child processes already see the host filesystem. Unlike the
// Linux instance-manager container, there is no /host bind-mount prefix.
func getReplicaMountedDataPath(dataPath string) string { return filepath.Clean(dataPath) }
