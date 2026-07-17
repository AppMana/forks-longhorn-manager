package types

import (
	"path/filepath"
	"strings"
)

const EngineBinaryName = "longhorn"

func engineBinaryDirectoryOnHost() string { return EngineBinaryDirectoryOnHost }

func engineBinaryDirectoryInContainer() string { return EngineBinaryDirectoryInContainer }

func engineBinaryDirectoryForReplicaManager() string {
	return ReplicaHostPrefix + EngineBinaryDirectoryOnHost
}

func ResolveDefaultDataPath(configured string) string { return configured }

func getReplicaMountedDataPath(dataPath string) string {
	if !strings.HasPrefix(dataPath, ReplicaHostPrefix) {
		return filepath.Join(ReplicaHostPrefix, dataPath)
	}
	return dataPath
}
