//go:build windows

package windowscsi

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Microsoft/go-winio"
	csipb "github.com/container-storage-interface/spec/lib/go/csi"
	diskapi "github.com/kubernetes-csi/csi-proxy/client/api/disk/v1"
	filesystemapi "github.com/kubernetes-csi/csi-proxy/client/api/filesystem/v1"
	iscsiapi "github.com/kubernetes-csi/csi-proxy/client/api/iscsi/v1alpha2"
	volumeapi "github.com/kubernetes-csi/csi-proxy/client/api/volume/v2alpha1"
	diskclient "github.com/kubernetes-csi/csi-proxy/client/groups/disk/v1"
	filesystemclient "github.com/kubernetes-csi/csi-proxy/client/groups/filesystem/v1"
	iscsiclient "github.com/kubernetes-csi/csi-proxy/client/groups/iscsi/v1alpha2"
	volumeclient "github.com/kubernetes-csi/csi-proxy/client/groups/volume/v2alpha1"
	longhornclient "github.com/longhorn/longhorn-manager/client"
	longhorn "github.com/longhorn/longhorn-manager/k8s/pkg/apis/longhorn/v1beta2"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	corev1 "k8s.io/api/core/v1"
)

const targetPort uint32 = 3260

type Server struct {
	csipb.UnimplementedIdentityServer
	csipb.UnimplementedNodeServer
	driverName, version, nodeID string
	api                         *longhornclient.RancherClient
	disk                        *diskclient.Client
	filesystem                  *filesystemclient.Client
	iscsi                       *iscsiclient.Client
	volume                      *volumeclient.Client
}

func New(driverName, version, nodeID, managerURL string) (*Server, error) {
	api, err := longhornclient.NewRancherClient(&longhornclient.ClientOpts{Url: managerURL})
	if err != nil {
		return nil, err
	}
	disk, err := diskclient.NewClient()
	if err != nil {
		return nil, err
	}
	filesystem, err := filesystemclient.NewClient()
	if err != nil {
		_ = disk.Close()
		return nil, err
	}
	iscsi, err := iscsiclient.NewClient()
	if err != nil {
		_ = filesystem.Close()
		_ = disk.Close()
		return nil, err
	}
	volume, err := volumeclient.NewClient()
	if err != nil {
		_ = iscsi.Close()
		_ = filesystem.Close()
		_ = disk.Close()
		return nil, err
	}
	return &Server{driverName: driverName, version: version, nodeID: nodeID, api: api,
		disk: disk, filesystem: filesystem, iscsi: iscsi, volume: volume}, nil
}

func (s *Server) Close() {
	_ = s.volume.Close()
	_ = s.iscsi.Close()
	_ = s.filesystem.Close()
	_ = s.disk.Close()
}

func (s *Server) Serve(endpoint string) error {
	listener, err := listen(endpoint)
	if err != nil {
		return err
	}
	server := grpc.NewServer()
	csipb.RegisterIdentityServer(server, s)
	csipb.RegisterNodeServer(server, s)
	logrus.Infof("Windows CSI node service listening on %s", endpoint)
	return server.Serve(listener)
}

func listen(endpoint string) (net.Listener, error) {
	if strings.HasPrefix(strings.ToLower(endpoint), "unix://") {
		path := strings.TrimPrefix(endpoint, "unix://")
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return nil, err
		}
		_ = os.Remove(path)
		return net.Listen("unix", path)
	}
	return winio.ListenPipe(normalizePipePath(endpoint), nil)
}

func normalizePipePath(endpoint string) string {
	endpoint = strings.TrimPrefix(strings.TrimPrefix(endpoint, "npipe://"), "npipe:")
	if strings.HasPrefix(endpoint, `\\.\pipe\`) {
		return endpoint
	}
	return `\\.\pipe\` + strings.ReplaceAll(strings.TrimLeft(endpoint, `./\`), "/", `\`)
}

func (s *Server) GetPluginInfo(context.Context, *csipb.GetPluginInfoRequest) (*csipb.GetPluginInfoResponse, error) {
	return &csipb.GetPluginInfoResponse{Name: s.driverName, VendorVersion: s.version}, nil
}
func (*Server) GetPluginCapabilities(context.Context, *csipb.GetPluginCapabilitiesRequest) (*csipb.GetPluginCapabilitiesResponse, error) {
	return &csipb.GetPluginCapabilitiesResponse{}, nil
}
func (*Server) Probe(context.Context, *csipb.ProbeRequest) (*csipb.ProbeResponse, error) {
	return &csipb.ProbeResponse{}, nil
}
func (s *Server) NodeGetInfo(context.Context, *csipb.NodeGetInfoRequest) (*csipb.NodeGetInfoResponse, error) {
	return &csipb.NodeGetInfoResponse{
		NodeId: s.nodeID,
		AccessibleTopology: &csipb.Topology{Segments: map[string]string{
			corev1.LabelHostname: s.nodeID,
			corev1.LabelOSStable: "windows",
		}},
	}, nil
}
func (*Server) NodeGetCapabilities(context.Context, *csipb.NodeGetCapabilitiesRequest) (*csipb.NodeGetCapabilitiesResponse, error) {
	response := &csipb.NodeGetCapabilitiesResponse{}
	for _, capability := range []csipb.NodeServiceCapability_RPC_Type{
		csipb.NodeServiceCapability_RPC_STAGE_UNSTAGE_VOLUME, csipb.NodeServiceCapability_RPC_GET_VOLUME_STATS,
		csipb.NodeServiceCapability_RPC_SINGLE_NODE_MULTI_WRITER,
	} {
		response.Capabilities = append(response.Capabilities, &csipb.NodeServiceCapability{Type: &csipb.NodeServiceCapability_Rpc{
			Rpc: &csipb.NodeServiceCapability_RPC{Type: capability}}})
	}
	return response, nil
}

type attachment struct {
	portal     *iscsiapi.TargetPortal
	iqn        string
	disk       uint32
	volumeID   string
	filesystem string
}

func parseISCSIEndpoint(endpoint string) (*iscsiapi.TargetPortal, string, error) {
	parsed, err := url.Parse(endpoint)
	if err != nil || !strings.EqualFold(parsed.Scheme, "iscsi") || parsed.Hostname() == "" {
		return nil, "", fmt.Errorf("invalid Longhorn iSCSI endpoint %q", endpoint)
	}
	port := uint64(targetPort)
	if parsed.Port() != "" {
		port, err = strconv.ParseUint(parsed.Port(), 10, 32)
		if err != nil {
			return nil, "", fmt.Errorf("invalid Longhorn iSCSI endpoint port %q: %w", endpoint, err)
		}
	}
	parts := strings.Split(strings.Trim(parsed.EscapedPath(), "/"), "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] != "1" {
		return nil, "", fmt.Errorf("invalid Longhorn iSCSI target path %q", parsed.Path)
	}
	iqn, err := url.PathUnescape(parts[0])
	if err != nil {
		return nil, "", err
	}
	return &iscsiapi.TargetPortal{TargetAddress: parsed.Hostname(), TargetPort: uint32(port)}, iqn, nil
}

func (s *Server) attachment(longhornVolumeID string, capability *csipb.VolumeCapability) (*attachment, error) {
	volume, err := s.api.Volume.ById(longhornVolumeID)
	if err != nil {
		return nil, err
	}
	if volume == nil || volume.State != string(longhorn.VolumeStateAttached) || !volume.Ready {
		return nil, status.Errorf(codes.FailedPrecondition, "volume %s is not attached and ready", longhornVolumeID)
	}
	if volume.DataEngine != string(longhorn.DataEngineTypeV1) || volume.Encrypted || volume.AccessMode == string(longhorn.AccessModeReadWriteMany) {
		return nil, status.Errorf(codes.Unimplemented, "volume %s requests an unsupported Windows data path", longhornVolumeID)
	}
	return s.attachmentFromVolume(volume, capability)
}

func (s *Server) attachmentFromVolume(volume *longhornclient.Volume, capability *csipb.VolumeCapability) (*attachment, error) {
	var controller *longhornclient.Controller
	for i := range volume.Controllers {
		if volume.Controllers[i].HostId == s.nodeID {
			controller = &volume.Controllers[i]
			break
		}
	}
	if controller == nil || controller.Endpoint == "" {
		return nil, status.Errorf(codes.FailedPrecondition, "volume %s has no engine frontend on node %s", volume.Name, s.nodeID)
	}
	portal, iqn, err := parseISCSIEndpoint(controller.Endpoint)
	if err != nil {
		return nil, err
	}
	filesystem := ""
	if mount := capability.GetMount(); mount != nil {
		filesystem = strings.ToLower(mount.FsType)
		if filesystem == "" {
			filesystem = "ntfs"
		}
		if filesystem != "ntfs" && filesystem != "refs" {
			return nil, status.Errorf(codes.InvalidArgument, "Windows supports NTFS and ReFS, not %q", filesystem)
		}
	}
	return &attachment{portal: portal, iqn: iqn, filesystem: filesystem}, nil
}

func (s *Server) NodeStageVolume(ctx context.Context, req *csipb.NodeStageVolumeRequest) (*csipb.NodeStageVolumeResponse, error) {
	if req.VolumeId == "" || req.StagingTargetPath == "" || req.VolumeCapability == nil {
		return nil, status.Error(codes.InvalidArgument, "volume, staging path, and capability are required")
	}
	if req.VolumeCapability.GetBlock() != nil {
		return nil, status.Error(codes.Unimplemented, "Windows raw block volumes are not supported")
	}
	attachment, err := s.attachment(req.VolumeId, req.VolumeCapability)
	if err != nil {
		return nil, err
	}
	if err := s.ensurePortal(ctx, attachment.portal, attachment.iqn); err != nil {
		return nil, err
	}
	if _, err := s.iscsi.ConnectTarget(ctx, &iscsiapi.ConnectTargetRequest{TargetPortal: attachment.portal, Iqn: attachment.iqn}); err != nil {
		if disks, getErr := s.targetDisks(ctx, attachment); getErr != nil || len(disks) == 0 {
			return nil, err
		}
	}
	disks, err := s.targetDisks(ctx, attachment)
	if err != nil || len(disks) != 1 {
		return nil, status.Errorf(codes.Internal, "target %s exposes %d disks: %v", attachment.iqn, len(disks), err)
	}
	diskNumber, err := strconv.ParseUint(disks[0], 10, 32)
	if err != nil {
		return nil, err
	}
	attachment.disk = uint32(diskNumber)
	if _, err := s.disk.SetDiskState(ctx, &diskapi.SetDiskStateRequest{DiskNumber: attachment.disk, IsOnline: true}); err != nil {
		return nil, err
	}
	if _, err := s.disk.PartitionDisk(ctx, &diskapi.PartitionDiskRequest{DiskNumber: attachment.disk}); err != nil {
		return nil, err
	}
	volumes, err := s.volume.ListVolumesOnDisk(ctx, &volumeapi.ListVolumesOnDiskRequest{DiskNumber: attachment.disk})
	if err != nil || len(volumes.GetVolumeIds()) != 1 {
		return nil, status.Errorf(codes.Internal, "disk %d exposes %d volumes: %v", attachment.disk, len(volumes.GetVolumeIds()), err)
	}
	attachment.volumeID = volumes.VolumeIds[0]
	formatted, err := s.volume.IsVolumeFormatted(ctx, &volumeapi.IsVolumeFormattedRequest{VolumeId: attachment.volumeID})
	if err != nil {
		return nil, err
	}
	if !formatted.Formatted {
		if _, err := s.volume.FormatVolume(ctx, &volumeapi.FormatVolumeRequest{VolumeId: attachment.volumeID, Filesystem: attachment.filesystem}); err != nil {
			return nil, err
		}
	}
	if err := s.ensureDirectory(ctx, req.StagingTargetPath); err != nil {
		return nil, err
	}
	if _, err := s.volume.MountVolume(ctx, &volumeapi.MountVolumeRequest{VolumeId: attachment.volumeID, TargetPath: windowsPath(req.StagingTargetPath)}); err != nil {
		return nil, err
	}
	return &csipb.NodeStageVolumeResponse{}, nil
}

func (s *Server) NodePublishVolume(ctx context.Context, req *csipb.NodePublishVolumeRequest) (*csipb.NodePublishVolumeResponse, error) {
	if req.TargetPath == "" || req.StagingTargetPath == "" || req.VolumeCapability == nil {
		return nil, status.Error(codes.InvalidArgument, "target, staging path, and capability are required")
	}
	if req.Readonly {
		return nil, status.Error(codes.Unimplemented, "read-only Windows mounts are not supported")
	}
	target := windowsPath(req.TargetPath)
	exists, err := s.filesystem.PathExists(ctx, &filesystemapi.PathExistsRequest{Path: target})
	if err != nil {
		return nil, err
	}
	if exists.Exists {
		isLink, linkErr := s.filesystem.IsSymlink(ctx, &filesystemapi.IsSymlinkRequest{Path: target})
		if linkErr == nil && isLink.IsSymlink {
			return &csipb.NodePublishVolumeResponse{}, nil
		}
		// Kubelet pre-creates an empty target directory on Windows. Remove only
		// that directory; Force=false deliberately refuses recursive deletion.
		if _, err := s.filesystem.Rmdir(ctx, &filesystemapi.RmdirRequest{Path: target, Force: false}); err != nil {
			return nil, err
		}
	}
	// Rmdir removes only the kubelet-created leaf. Keep publication robust to a
	// retry racing kubelet cleanup by making sure its parent still exists.
	if err := s.ensureDirectory(ctx, filepath.Dir(target)); err != nil {
		return nil, err
	}
	_, err = s.filesystem.CreateSymlink(ctx, &filesystemapi.CreateSymlinkRequest{
		SourcePath: windowsPath(req.StagingTargetPath), TargetPath: target})
	if err != nil {
		isLink, checkErr := s.filesystem.IsSymlink(ctx, &filesystemapi.IsSymlinkRequest{Path: target})
		if checkErr != nil || !isLink.IsSymlink {
			return nil, err
		}
	}
	return &csipb.NodePublishVolumeResponse{}, nil
}

func (s *Server) NodeUnpublishVolume(ctx context.Context, req *csipb.NodeUnpublishVolumeRequest) (*csipb.NodeUnpublishVolumeResponse, error) {
	if req.TargetPath == "" {
		return nil, status.Error(codes.InvalidArgument, "target path is required")
	}
	_, err := s.filesystem.Rmdir(ctx, &filesystemapi.RmdirRequest{Path: windowsPath(req.TargetPath), Force: true})
	if err != nil {
		exists, checkErr := s.filesystem.PathExists(ctx, &filesystemapi.PathExistsRequest{Path: windowsPath(req.TargetPath)})
		if checkErr != nil || exists.Exists {
			return nil, err
		}
	}
	return &csipb.NodeUnpublishVolumeResponse{}, nil
}

func (s *Server) NodeUnstageVolume(ctx context.Context, req *csipb.NodeUnstageVolumeRequest) (*csipb.NodeUnstageVolumeResponse, error) {
	if req.VolumeId == "" || req.StagingTargetPath == "" {
		return nil, status.Error(codes.InvalidArgument, "volume and staging path are required")
	}
	volumeID, err := s.volume.GetVolumeIDFromTargetPath(ctx, &volumeapi.GetVolumeIDFromTargetPathRequest{TargetPath: windowsPath(req.StagingTargetPath)})
	if err == nil && volumeID.VolumeId != "" {
		_, _ = s.volume.UnmountVolume(ctx, &volumeapi.UnmountVolumeRequest{VolumeId: volumeID.VolumeId, TargetPath: windowsPath(req.StagingTargetPath)})
		disk, diskErr := s.volume.GetDiskNumberFromVolumeID(ctx, &volumeapi.GetDiskNumberFromVolumeIDRequest{VolumeId: volumeID.VolumeId})
		if diskErr == nil {
			_, _ = s.disk.SetDiskState(ctx, &diskapi.SetDiskStateRequest{DiskNumber: disk.DiskNumber, IsOnline: false})
		}
	}
	// Unstage races controller detach and PVC deletion. The normal attachment
	// lookup deliberately rejects a volume once it is no longer ready, but the
	// controller endpoint is still sufficient to log out the host session. Use
	// it directly so deletion cannot strand an iSCSI connection and finalizer.
	if volume, getErr := s.api.Volume.ById(req.VolumeId); getErr == nil && volume != nil {
		if attachment, attachErr := s.attachmentFromVolume(volume, mountCapability("ntfs")); attachErr == nil {
			_, _ = s.iscsi.DisconnectTarget(ctx, &iscsiapi.DisconnectTargetRequest{TargetPortal: attachment.portal, Iqn: attachment.iqn})
		}
	}
	return &csipb.NodeUnstageVolumeResponse{}, nil
}

func (s *Server) NodeGetVolumeStats(ctx context.Context, req *csipb.NodeGetVolumeStatsRequest) (*csipb.NodeGetVolumeStatsResponse, error) {
	if req.VolumeId == "" || req.VolumePath == "" {
		return nil, status.Error(codes.InvalidArgument, "volume and volume path are required")
	}
	volumeID, err := s.volume.GetVolumeIDFromTargetPath(ctx, &volumeapi.GetVolumeIDFromTargetPathRequest{TargetPath: windowsPath(req.VolumePath)})
	if err != nil {
		return nil, err
	}
	stats, err := s.volume.GetVolumeStats(ctx, &volumeapi.GetVolumeStatsRequest{VolumeId: volumeID.VolumeId})
	if err != nil {
		return nil, err
	}
	return &csipb.NodeGetVolumeStatsResponse{Usage: []*csipb.VolumeUsage{{Unit: csipb.VolumeUsage_BYTES,
		Total: stats.TotalBytes, Used: stats.UsedBytes, Available: stats.TotalBytes - stats.UsedBytes}}}, nil
}

func (s *Server) NodeExpandVolume(ctx context.Context, req *csipb.NodeExpandVolumeRequest) (*csipb.NodeExpandVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "Windows online expansion requires persistent iSCSI LUN resize support")
}

func (s *Server) ensurePortal(ctx context.Context, portal *iscsiapi.TargetPortal, iqn string) error {
	// A previous NodeStageVolume attempt may have refreshed this shared portal
	// before its kubelet deadline expired. Check the cached discovery state
	// first so a retry does not repeat the expensive SendTargets refresh.
	if targets, err := s.iscsi.DiscoverTargetPortal(ctx, &iscsiapi.DiscoverTargetPortalRequest{TargetPortal: portal}); err == nil &&
		targetListContains(targets.Iqns, iqn) {
		return nil
	}

	// AddTargetPortal is an idempotent upsert in the Windows csi-proxy fork:
	// it creates a missing portal and refreshes SendTargets on an existing one.
	// Refreshing is essential for Longhorn's shared portal because its IQN set
	// changes as volume engines start and stop.
	if _, err := s.iscsi.AddTargetPortal(ctx, &iscsiapi.AddTargetPortalRequest{TargetPortal: portal}); err != nil {
		return err
	}
	targets, err := s.iscsi.DiscoverTargetPortal(ctx, &iscsiapi.DiscoverTargetPortalRequest{TargetPortal: portal})
	if err != nil {
		return err
	}
	if targetListContains(targets.Iqns, iqn) {
		return nil
	}
	return status.Errorf(codes.NotFound, "iSCSI target %s was not discovered at %s:%d", iqn, portal.TargetAddress, portal.TargetPort)
}

func targetListContains(targets []string, iqn string) bool {
	for _, target := range targets {
		if strings.EqualFold(target, iqn) {
			return true
		}
	}
	return false
}

func (s *Server) targetDisks(ctx context.Context, attachment *attachment) ([]string, error) {
	response, err := s.iscsi.GetTargetDisks(ctx, &iscsiapi.GetTargetDisksRequest{TargetPortal: attachment.portal, Iqn: attachment.iqn})
	if err != nil {
		return nil, err
	}
	return response.DiskIDs, nil
}

func (s *Server) ensureDirectory(ctx context.Context, path string) error {
	path = windowsPath(path)
	exists, err := s.filesystem.PathExists(ctx, &filesystemapi.PathExistsRequest{Path: path})
	if err != nil {
		return err
	}
	if exists.Exists {
		return nil
	}
	_, err = s.filesystem.Mkdir(ctx, &filesystemapi.MkdirRequest{Path: path})
	return err
}

func windowsPath(path string) string {
	path = filepath.Clean(strings.ReplaceAll(path, "/", `\`))
	if filepath.VolumeName(path) != "" || !strings.HasPrefix(path, `\`) {
		return path
	}
	drive := strings.TrimSpace(os.Getenv("SystemDrive"))
	if drive == "" {
		drive = filepath.VolumeName(os.Getenv("SystemRoot"))
	}
	if drive == "" {
		drive = "C:"
	}
	return drive + path
}
func mountCapability(filesystem string) *csipb.VolumeCapability {
	return &csipb.VolumeCapability{AccessType: &csipb.VolumeCapability_Mount{Mount: &csipb.VolumeCapability_MountVolume{FsType: filesystem}}}
}
