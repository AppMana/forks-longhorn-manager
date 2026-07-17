//go:build windows

package types

import "testing"

func TestGetReplicaMountedDataPathUsesWindowsHostPath(t *testing.T) {
	got := GetReplicaMountedDataPath(`C:\var\lib\longhorn\replicas\volume`)
	want := `C:\var\lib\longhorn\replicas\volume`
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
