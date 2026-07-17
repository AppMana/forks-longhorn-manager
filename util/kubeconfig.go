package util

import (
	"net"
	"os"
	"path/filepath"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	containerSandboxMountPointEnv = "CONTAINER_SANDBOX_MOUNT_POINT"
	serviceAccountTokenPath       = "var/run/secrets/kubernetes.io/serviceaccount/token"
	serviceAccountRootCAPath      = "var/run/secrets/kubernetes.io/serviceaccount/ca.crt"
)

// BuildConfig builds an explicit kubeconfig when one is supplied and otherwise
// uses the service account projected into the current pod.
func BuildConfig(kubeconfigPath string) (*rest.Config, error) {
	if kubeconfigPath != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	}
	return InClusterConfig()
}

// InClusterConfig is rest.InClusterConfig with Windows HostProcess volume
// compatibility. containerd 1.6 exposes projected volumes only beneath the
// container sandbox mount. containerd 1.7 and newer also bind them at their
// requested absolute paths, while retaining the sandbox-relative layout for
// compatibility. Resolving through CONTAINER_SANDBOX_MOUNT_POINT therefore
// gives both runtime generations one stable service-account path.
func InClusterConfig() (*rest.Config, error) {
	sandboxMountPoint := os.Getenv(containerSandboxMountPointEnv)
	if sandboxMountPoint == "" {
		return rest.InClusterConfig()
	}

	host, port := os.Getenv("KUBERNETES_SERVICE_HOST"), os.Getenv("KUBERNETES_SERVICE_PORT")
	if host == "" || port == "" {
		return nil, rest.ErrNotInCluster
	}

	tokenFile := filepath.Join(sandboxMountPoint, filepath.FromSlash(serviceAccountTokenPath))
	rootCAFile := filepath.Join(sandboxMountPoint, filepath.FromSlash(serviceAccountRootCAPath))
	token, err := os.ReadFile(tokenFile)
	if err != nil {
		return nil, err
	}

	return &rest.Config{
		Host: "https://" + net.JoinHostPort(host, port),
		TLSClientConfig: rest.TLSClientConfig{
			CAFile: rootCAFile,
		},
		BearerToken:     string(token),
		BearerTokenFile: tokenFile,
	}, nil
}
