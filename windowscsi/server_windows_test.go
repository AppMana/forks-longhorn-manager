//go:build windows

package windowscsi

import (
	"os"
	"testing"
)

func TestWindowsPathUsesDiscoveredSystemDrive(t *testing.T) {
	t.Setenv("SystemDrive", "D:")
	if got, want := windowsPath("/var/lib/kubelet/plugins/driver/globalmount"), `D:\var\lib\kubelet\plugins\driver\globalmount`; got != want {
		t.Fatalf("windowsPath() = %q, want %q", got, want)
	}
}

func TestWindowsPathPreservesExplicitDrive(t *testing.T) {
	t.Setenv("SystemDrive", "D:")
	if got, want := windowsPath("C:/var/lib/kubelet"), `C:\var\lib\kubelet`; got != want {
		t.Fatalf("windowsPath() = %q, want %q", got, want)
	}
}

func TestWindowsPathFallsBackToSystemRoot(t *testing.T) {
	if err := os.Unsetenv("SystemDrive"); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SystemRoot", `E:\Windows`)
	if got, want := windowsPath(`/var/lib/kubelet`), `E:\var\lib\kubelet`; got != want {
		t.Fatalf("windowsPath() = %q, want %q", got, want)
	}
}

func TestTargetListContainsIsCaseInsensitive(t *testing.T) {
	targets := []string{"iqn.2026-07.io.longhorn:one", "IQN.2026-07.IO.LONGHORN:TWO"}
	if !targetListContains(targets, "iqn.2026-07.io.longhorn:two") {
		t.Fatal("targetListContains did not find an existing target case-insensitively")
	}
	if targetListContains(targets, "iqn.2026-07.io.longhorn:missing") {
		t.Fatal("targetListContains found a target that was not present")
	}
}
