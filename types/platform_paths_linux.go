package types

const EngineBinaryName = "longhorn"

func engineBinaryDirectoryOnHost() string { return EngineBinaryDirectoryOnHost }

func engineBinaryDirectoryInContainer() string { return EngineBinaryDirectoryInContainer }

func engineBinaryDirectoryForReplicaManager() string {
	return ReplicaHostPrefix + EngineBinaryDirectoryOnHost
}

func ResolveDefaultDataPath(configured string) string { return configured }
