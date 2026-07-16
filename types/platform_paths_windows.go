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
