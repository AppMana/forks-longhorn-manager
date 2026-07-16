//go:build windows

package main

import (
	"flag"

	"github.com/longhorn/longhorn-manager/windowscsi"
	"github.com/sirupsen/logrus"
)

func main() {
	driverName := flag.String("driver-name", "driver.longhorn.io", "CSI driver name")
	version := flag.String("version", "dev", "driver version")
	nodeID := flag.String("node-id", "", "Kubernetes node name")
	managerURL := flag.String("manager-url", "http://longhorn-backend:9500/v1", "Longhorn API URL")
	endpoint := flag.String("endpoint", `unix://C:\var\lib\kubelet\plugins\driver.longhorn.io\csi.sock`, "CSI Unix socket or named pipe")
	flag.Parse()
	if *nodeID == "" {
		logrus.Fatal("--node-id is required")
	}
	server, err := windowscsi.New(*driverName, *version, *nodeID, *managerURL)
	if err != nil {
		logrus.WithError(err).Fatal("initialize Windows CSI node server")
	}
	defer server.Close()
	if err := server.Serve(*endpoint); err != nil {
		logrus.WithError(err).Fatal("serve Windows CSI node API")
	}
}
