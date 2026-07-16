package util

import (
	"github.com/sirupsen/logrus"

	lhns "github.com/longhorn/go-common-libs/ns"
	lhtypes "github.com/longhorn/go-common-libs/types"
	lhspdkutil "github.com/longhorn/go-spdk-helper/pkg/util"
)

func removeDMDevice(devicePath string) error {
	namespaces := []lhtypes.Namespace{lhtypes.NamespaceMnt, lhtypes.NamespaceIpc}
	nsexec, err := lhns.NewNamespaceExecutor(lhtypes.ProcessNone, lhtypes.HostProcDirectory, namespaces)
	if err != nil {
		return err
	}
	if err := lhspdkutil.DmsetupRemove(devicePath, true, true, nsexec); err != nil {
		if isIgnorableDMRemoveError(err) {
			logrus.WithError(err).Debugf("Dm device %s is already removed.", devicePath)
			return nil
		}
		return err
	}
	return nil
}
