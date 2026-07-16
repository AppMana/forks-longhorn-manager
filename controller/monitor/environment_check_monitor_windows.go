package monitor

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/jinzhu/copier"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"

	lhexec "github.com/longhorn/go-common-libs/exec"
	lhtypes "github.com/longhorn/go-common-libs/types"
	"github.com/longhorn/longhorn-manager/datastore"
	longhorn "github.com/longhorn/longhorn-manager/k8s/pkg/apis/longhorn/v1beta2"
	"github.com/longhorn/longhorn-manager/types"
)

const environmentCheckMonitorSyncPeriod = 30 * time.Minute

type EnvironmentCheckMonitor struct {
	*baseMonitor
	nodeName          string
	collectedDataLock sync.RWMutex
	collectedData     *CollectedEnvironmentCheckInfo
	syncCallback      func(string)
}

type CollectedEnvironmentCheckInfo struct{ conditions []longhorn.Condition }

func NewEnvironmentCheckMonitor(logger logrus.FieldLogger, ds *datastore.DataStore, nodeName string, syncCallback func(string)) (*EnvironmentCheckMonitor, error) {
	ctx, quit := context.WithCancel(context.Background())
	m := &EnvironmentCheckMonitor{
		baseMonitor:   newBaseMonitor(ctx, quit, logger, ds, environmentCheckMonitorSyncPeriod),
		nodeName:      nodeName,
		collectedData: &CollectedEnvironmentCheckInfo{},
		syncCallback:  syncCallback,
	}
	go m.Start()
	return m, nil
}

func (m *EnvironmentCheckMonitor) Start() {
	if err := wait.PollUntilContextCancel(m.ctx, m.syncPeriod, true, func(context.Context) (bool, error) {
		if err := m.run(struct{}{}); err != nil {
			m.logger.WithError(err).Error("Stopped monitoring Windows environment")
		}
		return false, nil
	}); err != nil && !errors.Is(err, context.Canceled) {
		m.logger.WithError(err).Error("Failed to start Windows environment monitor")
	}
}

func (m *EnvironmentCheckMonitor) Stop()                                            { m.quit() }
func (m *EnvironmentCheckMonitor) RunOnce() error                                   { return m.run(struct{}{}) }
func (m *EnvironmentCheckMonitor) UpdateConfiguration(map[string]interface{}) error { return nil }

func (m *EnvironmentCheckMonitor) GetCollectedData() (interface{}, error) {
	m.collectedDataLock.RLock()
	defer m.collectedDataLock.RUnlock()
	data := []longhorn.Condition{}
	if err := copier.CopyWithOption(&data, &m.collectedData.conditions, copier.Option{IgnoreEmpty: true, DeepCopy: true}); err != nil {
		return data, errors.Wrap(err, "failed to copy Windows environment data")
	}
	return data, nil
}

func (m *EnvironmentCheckMonitor) run(interface{}) error {
	node, err := m.ds.GetNode(m.nodeName)
	if err != nil {
		return errors.Wrapf(err, "failed to get Longhorn node %v", m.nodeName)
	}
	collected := m.collectEnvironmentCheckData()
	if reflect.DeepEqual(m.collectedData, collected) {
		return nil
	}
	m.collectedDataLock.Lock()
	m.collectedData = collected
	m.collectedDataLock.Unlock()
	m.syncCallback(node.Namespace + "/" + m.nodeName)
	return nil
}

func (m *EnvironmentCheckMonitor) collectEnvironmentCheckData() *CollectedEnvironmentCheckInfo {
	result := &CollectedEnvironmentCheckInfo{conditions: []longhorn.Condition{}}
	command := `$os=(Get-CimInstance Win32_OperatingSystem).ProductType; $svc=(Get-Service MSiSCSI).Status; Write-Output ("$os|$svc")`
	output, err := lhexec.NewExecutor().Execute(nil, "powershell.exe", []string{"-NoProfile", "-NonInteractive", "-Command", command}, lhtypes.ExecuteDefaultTimeout)
	if err != nil {
		result.conditions = types.SetCondition(result.conditions, longhorn.NodeConditionTypeRequiredPackages, longhorn.ConditionStatusFalse,
			string(longhorn.NodeConditionReasonPackagesNotInstalled), fmt.Sprintf("Windows Server MSiSCSI check failed: %v", err))
		return result
	}
	parts := strings.Split(strings.TrimSpace(output), "|")
	if len(parts) != 2 || (parts[0] != "2" && parts[0] != "3") || !strings.EqualFold(parts[1], "Running") {
		result.conditions = types.SetCondition(result.conditions, longhorn.NodeConditionTypeRequiredPackages, longhorn.ConditionStatusFalse,
			string(longhorn.NodeConditionReasonPackagesNotInstalled), fmt.Sprintf("Windows Server with running MSiSCSI is required; detected %q", strings.TrimSpace(output)))
		return result
	}
	result.conditions = types.SetCondition(result.conditions, longhorn.NodeConditionTypeRequiredPackages, longhorn.ConditionStatusTrue, "", "Windows Server MSiSCSI is running")
	return result
}
