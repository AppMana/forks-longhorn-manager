package util

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInClusterConfigUsesContainerSandboxMountPoint(t *testing.T) {
	sandbox := t.TempDir()
	serviceAccountDir := filepath.Join(sandbox, filepath.FromSlash("var/run/secrets/kubernetes.io/serviceaccount"))
	require.NoError(t, os.MkdirAll(serviceAccountDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(serviceAccountDir, "token"), []byte("sandbox-token"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(serviceAccountDir, "ca.crt"), []byte("test-ca"), 0o600))

	t.Setenv(containerSandboxMountPointEnv, sandbox)
	t.Setenv("KUBERNETES_SERVICE_HOST", "10.43.0.1")
	t.Setenv("KUBERNETES_SERVICE_PORT", "443")

	config, err := InClusterConfig()
	require.NoError(t, err)
	require.Equal(t, "https://10.43.0.1:443", config.Host)
	require.Equal(t, "sandbox-token", config.BearerToken)
	require.Equal(t, filepath.Join(serviceAccountDir, "token"), config.BearerTokenFile)
	require.Equal(t, filepath.Join(serviceAccountDir, "ca.crt"), config.CAFile)
}

func TestBuildConfigUsesExplicitKubeconfig(t *testing.T) {
	kubeconfig := filepath.Join(t.TempDir(), "config")
	require.NoError(t, os.WriteFile(kubeconfig, []byte(`apiVersion: v1
kind: Config
clusters:
- name: test
  cluster:
    server: https://192.0.2.10:6443
contexts:
- name: test
  context:
    cluster: test
    user: test
current-context: test
users:
- name: test
  user:
    token: explicit-token
`), 0o600))

	t.Setenv(containerSandboxMountPointEnv, `C:\hpc\ignored`)
	config, err := BuildConfig(kubeconfig)
	require.NoError(t, err)
	require.Equal(t, "https://192.0.2.10:6443", config.Host)
	require.Equal(t, "explicit-token", config.BearerToken)
}
