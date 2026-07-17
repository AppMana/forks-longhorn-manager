package app

import (
	"github.com/longhorn/go-iscsi-helper/iscsi"
	iscsiutil "github.com/longhorn/go-iscsi-helper/util"

	lhns "github.com/longhorn/go-common-libs/ns"
	lhtypes "github.com/longhorn/go-common-libs/types"
)

func platformEnvironmentCheck() error {
	// Other tools and settings are checked periodically by the node controller.
	namespaces := []lhtypes.Namespace{lhtypes.NamespaceMnt, lhtypes.NamespaceNet}
	nsexec, err := lhns.NewNamespaceExecutor(iscsiutil.ISCSIdProcess, lhtypes.HostProcDirectory, namespaces)
	if err != nil {
		return err
	}
	return iscsi.CheckForInitiatorExistence(nsexec)
}

func platformStartsWebhooks() bool {
	return true
}
