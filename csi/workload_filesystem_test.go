package csi

import (
	"testing"

	csipb "github.com/container-storage-interface/spec/lib/go/csi"
)

func mountCapability(fsType string) *csipb.VolumeCapability {
	return &csipb.VolumeCapability{AccessType: &csipb.VolumeCapability_Mount{Mount: &csipb.VolumeCapability_MountVolume{FsType: fsType}}}
}

func TestWorkloadFilesystemFromCapabilities(t *testing.T) {
	tests := []struct {
		name         string
		capabilities []*csipb.VolumeCapability
		want         string
		wantError    bool
	}{
		{name: "default", capabilities: []*csipb.VolumeCapability{mountCapability("")}, want: "ext4"},
		{name: "ntfs normalized", capabilities: []*csipb.VolumeCapability{mountCapability("NTFS")}, want: "ntfs"},
		{name: "refs normalized", capabilities: []*csipb.VolumeCapability{mountCapability("ReFS")}, want: "refs"},
		{name: "block", capabilities: []*csipb.VolumeCapability{{AccessType: &csipb.VolumeCapability_Block{Block: &csipb.VolumeCapability_BlockVolume{}}}}, want: ""},
		{name: "conflict", capabilities: []*csipb.VolumeCapability{mountCapability("ntfs"), mountCapability("refs")}, wantError: true},
		{name: "unsupported", capabilities: []*csipb.VolumeCapability{mountCapability("fat32")}, wantError: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := workloadFilesystemFromCapabilities(test.capabilities)
			if (err != nil) != test.wantError {
				t.Fatalf("error = %v, wantError %v", err, test.wantError)
			}
			if got != test.want {
				t.Fatalf("filesystem = %q, want %q", got, test.want)
			}
		})
	}
}
