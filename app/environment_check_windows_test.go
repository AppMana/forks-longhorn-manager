package app

import (
	"strings"
	"testing"
)

func TestDiscoverWindowsKubeletKubeconfigPathsPrefersOverride(t *testing.T) {
	const configured = `D:/kubelet/state/kubelet.conf`
	t.Setenv("KUBELET_KUBECONFIG", configured)

	paths := discoverWindowsKubeletKubeconfigPaths()
	if len(paths) == 0 || paths[0] != configured {
		t.Fatalf("paths = %v, want explicit override first", paths)
	}
	seen := map[string]struct{}{}
	for _, path := range paths {
		key := strings.ToLower(path)
		if _, exists := seen[key]; exists {
			t.Fatalf("duplicate path %q in %v", path, paths)
		}
		seen[key] = struct{}{}
	}
}
