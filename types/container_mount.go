package types

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const containerSandboxMountPointEnv = "CONTAINER_SANDBOX_MOUNT_POINT"

// ResolveContainerMountPath returns the path at which a process can consume a
// Kubernetes volume mount. Windows HostProcess containers on containerd 1.6
// expose mounts beneath CONTAINER_SANDBOX_MOUNT_POINT only. Newer containerd
// releases additionally create the requested direct bind, but preserve the
// sandbox-relative layout for compatibility.
func ResolveContainerMountPath(path string) string {
	if runtime.GOOS != "windows" {
		return path
	}

	sandbox := os.Getenv(containerSandboxMountPointEnv)
	if sandbox == "" {
		return path
	}

	cleanPath := filepath.Clean(path)
	if relative, err := filepath.Rel(sandbox, cleanPath); err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return cleanPath
	}

	cleanPath = strings.TrimPrefix(cleanPath, filepath.VolumeName(cleanPath))
	cleanPath = strings.TrimLeft(cleanPath, `/\`)
	return filepath.Join(sandbox, cleanPath)
}
