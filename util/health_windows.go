package util

import (
	"fmt"

	"github.com/sirupsen/logrus"

	longhorn "github.com/longhorn/longhorn-manager/k8s/pkg/apis/longhorn/v1beta2"
)

// Windows disk health is not gathered through Linux SMART tooling. Returning
// an error lets the disk monitor preserve its previous health data while the
// capacity and readiness paths continue normally.
func CollectHealthDataFromMountPath(mountPath, diskName string, logger logrus.FieldLogger) (map[string]longhorn.HealthData, error) {
	return nil, fmt.Errorf("SMART collection for Windows mount %q is not supported", mountPath)
}

func CollectHealthDataForBlockDevice(devicePath, diskName string, logger logrus.FieldLogger) (map[string]longhorn.HealthData, error) {
	return nil, fmt.Errorf("SMART collection for Windows device %q is not supported", devicePath)
}
