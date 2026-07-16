package types

import (
	"testing"

	longhorn "github.com/longhorn/longhorn-manager/k8s/pkg/apis/longhorn/v1beta2"
)

func TestWindowsRequirementsAreRoleSpecific(t *testing.T) {
	volume := &longhorn.Volume{Spec: longhorn.VolumeSpec{
		AccessMode:         longhorn.AccessModeReadWriteOnce,
		DataEngine:         longhorn.DataEngineTypeV1,
		Frontend:           longhorn.VolumeFrontendISCSI,
		WorkloadFileSystem: "refs",
	}}
	requirements := ResolveVolumeRequirements(volume)
	windows := WindowsV1NodeCapabilities()
	for role, check := range map[string][]string{
		"controller": MissingCapabilities(windows.Controller, requirements.Controller),
		"replica":    MissingCapabilities(windows.Replica, requirements.Replica),
		"frontend":   MissingCapabilities(windows.Frontend, requirements.Frontend),
		"disk":       MissingCapabilities(windows.Disk, requirements.Disk),
	} {
		if len(check) != 0 {
			t.Fatalf("Windows RWO ReFS %s requirements are missing %v", role, check)
		}
	}
}

func TestWindowsRejectsRWXAndStrictLocal(t *testing.T) {
	volume := &longhorn.Volume{Spec: longhorn.VolumeSpec{
		AccessMode:   longhorn.AccessModeReadWriteMany,
		DataEngine:   longhorn.DataEngineTypeV1,
		DataLocality: longhorn.DataLocalityStrictLocal,
	}}
	missing := MissingCapabilities(WindowsV1NodeCapabilities().Replica, ResolveVolumeRequirements(volume).Replica)
	if len(missing) != 2 || missing[0] != EngineCapabilityRWX || missing[1] != EngineCapabilityStrictLocal {
		t.Fatalf("expected RWX and strict-local to be rejected, got %v", missing)
	}
}

func TestWorkloadFilesystemConstrainsFrontendPlatform(t *testing.T) {
	volume := &longhorn.Volume{Spec: longhorn.VolumeSpec{
		AccessMode:         longhorn.AccessModeReadWriteOnce,
		DataEngine:         longhorn.DataEngineTypeV1,
		Frontend:           longhorn.VolumeFrontendISCSI,
		WorkloadFileSystem: "ext4",
	}}
	requirements := ResolveVolumeRequirements(volume).Frontend
	if missing := MissingCapabilities(LegacyLinuxNodeCapabilities().Frontend, requirements); len(missing) != 0 {
		t.Fatalf("Linux ext4 frontend unexpectedly misses %v", missing)
	}
	missing := MissingCapabilities(WindowsV1NodeCapabilities().Frontend, requirements)
	if len(missing) != 1 || missing[0] != FrontendCapabilityExt4 {
		t.Fatalf("expected Windows frontend to reject ext4, got %v", missing)
	}
}
