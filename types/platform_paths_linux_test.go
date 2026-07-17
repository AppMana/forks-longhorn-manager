//go:build !windows

package types

import "testing"

func TestGetReplicaMountedDataPathUsesLinuxHostMount(t *testing.T) {
	got := GetReplicaMountedDataPath("/var/lib/longhorn/replicas/volume")
	want := "/host/var/lib/longhorn/replicas/volume"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
