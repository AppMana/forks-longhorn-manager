//go:build !windows

package controller

import "github.com/longhorn/longhorn-manager/types"

func managerServiceLabels() map[string]string {
	return types.MergeStringMaps(types.GetAdmissionWebhookLabel(), types.GetRecoveryBackendLabel())
}
