package types

import (
	"fmt"
	"sort"
	"strings"

	longhorn "github.com/longhorn/longhorn-manager/k8s/pkg/apis/longhorn/v1beta2"
)

// Engine image capabilities are stable wire identifiers. New capabilities
// must be additive; a node that does not advertise a required capability is
// ineligible for that role.
const (
	EngineCapabilityV1               = "engine:v1"
	EngineCapabilityRWO              = "access-mode:rwo"
	EngineCapabilityRWOP             = "access-mode:rwop"
	EngineCapabilityRWX              = "access-mode:rwx"
	EngineCapabilityBestEffort       = "data-locality:best-effort"
	EngineCapabilityStrictLocal      = "data-locality:strict-local"
	EngineCapabilityEncryption       = "volume:encryption"
	EngineCapabilityBackingImage     = "volume:backing-image"
	EngineCapabilityFilesystemFreeze = "snapshot:filesystem-freeze"

	FrontendCapabilityISCSI       = "frontend:iscsi"
	FrontendCapabilityLiveUpgrade = "frontend:live-upgrade"
	FrontendCapabilityExt4        = "workload-fs:ext4"
	FrontendCapabilityXFS         = "workload-fs:xfs"
	FrontendCapabilityNTFS        = "workload-fs:ntfs"
	FrontendCapabilityReFS        = "workload-fs:refs"

	DiskCapabilitySparse = "replica-store:sparse"
	DiskCapabilityNTFS   = "replica-store:ntfs"
	DiskCapabilityReFS   = "replica-store:refs"
)

type VolumeRequirements struct {
	Controller []string
	Replica    []string
	Frontend   []string
	Disk       []string
}

// ResolveVolumeRequirements translates a VolumeSpec into role-specific hard
// requirements. The scheduler consumes the Replica set; attachment and disk
// reconciliation consume the other sets.
func ResolveVolumeRequirements(volume *longhorn.Volume) VolumeRequirements {
	requirements := VolumeRequirements{}
	if volume == nil {
		return requirements
	}

	if !IsDataEngineV2(volume.Spec.DataEngine) {
		requirements.Controller = append(requirements.Controller, EngineCapabilityV1)
		requirements.Replica = append(requirements.Replica, EngineCapabilityV1)
	}

	switch volume.Spec.AccessMode {
	case longhorn.AccessModeReadWriteMany:
		requirements.Controller = append(requirements.Controller, EngineCapabilityRWX)
		requirements.Replica = append(requirements.Replica, EngineCapabilityRWX)
	case longhorn.AccessModeReadWriteOncePod:
		requirements.Controller = append(requirements.Controller, EngineCapabilityRWOP)
		requirements.Replica = append(requirements.Replica, EngineCapabilityRWOP)
	default:
		requirements.Controller = append(requirements.Controller, EngineCapabilityRWO)
		requirements.Replica = append(requirements.Replica, EngineCapabilityRWO)
	}

	switch volume.Spec.DataLocality {
	case longhorn.DataLocalityBestEffort:
		requirements.Controller = append(requirements.Controller, EngineCapabilityBestEffort)
		requirements.Replica = append(requirements.Replica, EngineCapabilityBestEffort)
	case longhorn.DataLocalityStrictLocal:
		requirements.Controller = append(requirements.Controller, EngineCapabilityStrictLocal)
		requirements.Replica = append(requirements.Replica, EngineCapabilityStrictLocal)
	}

	if volume.Spec.Encrypted {
		requirements.Controller = append(requirements.Controller, EngineCapabilityEncryption)
		requirements.Replica = append(requirements.Replica, EngineCapabilityEncryption)
	}
	if volume.Spec.BackingImage != "" {
		requirements.Controller = append(requirements.Controller, EngineCapabilityBackingImage)
		requirements.Replica = append(requirements.Replica, EngineCapabilityBackingImage)
	}
	if volume.Spec.FreezeFilesystemForSnapshot == longhorn.FreezeFilesystemForSnapshotEnabled {
		requirements.Controller = append(requirements.Controller, EngineCapabilityFilesystemFreeze)
	}

	if volume.Spec.Frontend == longhorn.VolumeFrontendISCSI {
		requirements.Frontend = append(requirements.Frontend, FrontendCapabilityISCSI)
	}
	switch strings.ToLower(volume.Spec.WorkloadFileSystem) {
	case "ext4":
		requirements.Frontend = append(requirements.Frontend, FrontendCapabilityExt4)
	case "xfs":
		requirements.Frontend = append(requirements.Frontend, FrontendCapabilityXFS)
	case "ntfs":
		requirements.Frontend = append(requirements.Frontend, FrontendCapabilityNTFS)
	case "refs":
		requirements.Frontend = append(requirements.Frontend, FrontendCapabilityReFS)
	}

	requirements.Disk = append(requirements.Disk, DiskCapabilitySparse)
	return requirements
}

func MissingCapabilities(advertised, required []string) []string {
	available := map[string]struct{}{}
	for _, capability := range advertised {
		available[capability] = struct{}{}
	}
	missing := []string{}
	for _, capability := range required {
		if _, ok := available[capability]; !ok {
			missing = append(missing, capability)
		}
	}
	sort.Strings(missing)
	return missing
}

func FormatMissingCapabilities(role string, missing []string) string {
	return fmt.Sprintf("%s capabilities unavailable: %s", role, strings.Join(missing, ", "))
}

// LegacyLinuxNodeCapabilities preserves the existing Linux behavior while
// capabilities roll out. Windows nodes never use this fallback.
func LegacyLinuxNodeCapabilities() longhorn.EngineImageNodeCapabilities {
	common := []string{
		EngineCapabilityV1,
		EngineCapabilityRWO,
		EngineCapabilityRWOP,
		EngineCapabilityRWX,
		EngineCapabilityBestEffort,
		EngineCapabilityStrictLocal,
		EngineCapabilityEncryption,
		EngineCapabilityBackingImage,
		EngineCapabilityFilesystemFreeze,
	}
	return longhorn.EngineImageNodeCapabilities{
		Controller: append([]string{}, common...),
		Replica:    append([]string{}, common...),
		Frontend: []string{
			FrontendCapabilityISCSI,
			FrontendCapabilityLiveUpgrade,
			FrontendCapabilityExt4,
			FrontendCapabilityXFS,
		},
		Disk: []string{DiskCapabilitySparse},
	}
}

// WindowsV1NodeCapabilities is deliberately narrower than Linux. In
// particular it omits RWX, strict-local, encryption, backing images, and
// filesystem freeze. The scheduler must never infer those features merely
// because the engine image pod is ready on the node.
func WindowsV1NodeCapabilities() longhorn.EngineImageNodeCapabilities {
	common := []string{
		EngineCapabilityV1,
		EngineCapabilityRWO,
		EngineCapabilityRWOP,
		EngineCapabilityBestEffort,
	}
	return longhorn.EngineImageNodeCapabilities{
		Controller: append([]string{}, common...),
		Replica:    append([]string{}, common...),
		Frontend: []string{
			FrontendCapabilityISCSI,
			FrontendCapabilityLiveUpgrade,
			FrontendCapabilityNTFS,
			FrontendCapabilityReFS,
		},
		Disk: []string{
			DiskCapabilitySparse,
			DiskCapabilityNTFS,
			DiskCapabilityReFS,
		},
	}
}
