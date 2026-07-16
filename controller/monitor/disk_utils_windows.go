package monitor

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/sirupsen/logrus"

	lhns "github.com/longhorn/go-common-libs/ns"
	lhtypes "github.com/longhorn/go-common-libs/types"

	"github.com/longhorn/longhorn-manager/datastore"
	"github.com/longhorn/longhorn-manager/engineapi"
	longhorn "github.com/longhorn/longhorn-manager/k8s/pkg/apis/longhorn/v1beta2"
	"github.com/longhorn/longhorn-manager/util"
)

const uuidGenerationRetries = 20

func getDiskStat(diskType longhorn.DiskType, diskName, diskPath string, diskDriver longhorn.DiskDriver, client *DiskServiceClient) (*lhtypes.DiskStat, error) {
	if diskType != longhorn.DiskTypeFilesystem {
		return nil, fmt.Errorf("block-type disks require the V2 engine, which is not supported on Windows")
	}
	return lhns.GetDiskStat(diskPath)
}

func getDiskHealth(_ longhorn.DiskType, _ string, _ string, _ longhorn.DiskDriver, lastCollectedAt time.Time, _ *DiskServiceClient, _ logrus.FieldLogger) (map[string]longhorn.HealthData, time.Time, error) {
	return nil, lastCollectedAt, nil
}

func getDiskConfig(diskType longhorn.DiskType, _ string, diskPath string, _ longhorn.DiskDriver, _ *DiskServiceClient) (*util.DiskConfig, error) {
	if diskType != longhorn.DiskTypeFilesystem {
		return nil, fmt.Errorf("block-type disks require the V2 engine, which is not supported on Windows")
	}
	diskCfgFilePath := filepath.Join(diskPath, util.DiskConfigFile)
	output, err := lhns.ReadFileContent(diskCfgFilePath)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read host disk config file %v", util.DiskConfigFile)
	}
	cfg := &util.DiskConfig{}
	if err := json.Unmarshal([]byte(output), cfg); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal host %v content: %v", diskCfgFilePath, output)
	}
	cfg.State = diskStateReady
	return cfg, nil
}

func generateDiskConfig(diskType longhorn.DiskType, diskName, _ string, diskPath, _ string, _ *DiskServiceClient, ds *datastore.DataStore) (*util.DiskConfig, error) {
	if diskType != longhorn.DiskTypeFilesystem {
		return nil, fmt.Errorf("block-type disks require the V2 engine, which is not supported on Windows")
	}
	allPrefixes, err := ds.GetAllDiskUUIDFirstFourChar()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get all disk UUID prefixes")
	}
	diskUUID, err := generateUniqueFirstFourCharUUID(allPrefixes)
	if err != nil {
		return nil, err
	}
	cfg := &util.DiskConfig{DiskName: diskName, DiskUUID: diskUUID, State: diskStateReady}
	encoded, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}
	if _, err := lhns.CreateDirectory(diskPath, time.Now()); err != nil {
		return nil, errors.Wrap(err, "failed to create Windows disk directory")
	}
	diskCfgFilePath := filepath.Join(diskPath, util.DiskConfigFile)
	if _, err := lhns.GetFileInfo(diskCfgFilePath); err == nil {
		return nil, fmt.Errorf("disk config on %v exists, cannot override", diskCfgFilePath)
	}
	if err := lhns.WriteFile(diskCfgFilePath, string(encoded)); err != nil {
		return nil, err
	}
	if _, err := lhns.CreateDirectory(filepath.Join(diskPath, util.ReplicaDirectory), time.Now()); err != nil {
		return nil, errors.Wrap(err, "failed to create replica subdirectory")
	}
	if err := lhns.SyncFile(diskCfgFilePath); err != nil {
		return nil, err
	}
	return cfg, nil
}

func DeleteDisk(diskType longhorn.DiskType, diskName, diskUUID, diskPath, diskDriver string, client *engineapi.DiskService) error {
	if diskType == longhorn.DiskTypeFilesystem {
		return util.DeleteDiskPathReplicaSubdirectoryAndDiskCfgFile(diskPath)
	}
	return fmt.Errorf("block-type disks require the V2 engine, which is not supported on Windows")
}

func getSpdkReplicaInstanceNames(_ *DiskServiceClient, _ string, _ string, _ string) (map[string]string, error) {
	return nil, fmt.Errorf("SPDK replica discovery is not supported on Windows")
}

func generateUniqueFirstFourCharUUID(existing map[string]bool) (string, error) {
	for i := 0; i < uuidGenerationRetries; i++ {
		candidate := util.UUID()
		if !existing[candidate[:4]] {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("failed to generate a unique disk UUID")
}
