package app

import (
	"fmt"

	"github.com/urfave/cli"
)

// The Windows CSI node service is a separate HostProcess binary. The manager
// image intentionally retains command names so scripts fail clearly instead of
// silently invoking the Linux implementation.
func DeployDriverCmd() cli.Command {
	return unsupportedWindowsCommand("deploy-driver")
}

func CSICommand() cli.Command {
	return unsupportedWindowsCommand("csi")
}

func unsupportedWindowsCommand(name string) cli.Command {
	return cli.Command{
		Name:   name,
		Hidden: true,
		Action: func(*cli.Context) error {
			return fmt.Errorf("%s is provided by the longhorn-windows-csi HostProcess service", name)
		},
	}
}
