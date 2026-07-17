package app

import (
	"bufio"
	"fmt"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func platformEnvironmentCheck() error {
	// Windows client editions are intentionally excluded. Server editions ship
	// the supported iSCSI initiator service used by the Windows V1 frontend.
	command := `$productType=(Get-CimInstance Win32_OperatingSystem).ProductType; ` +
		`$service=Get-Service MSiSCSI -ErrorAction Stop; ` +
		`if (($productType -notin 2,3) -or ($service.Status -ne 'Running')) { exit 1 }`
	output, err := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-Command", command).CombinedOutput()
	if err != nil {
		return fmt.Errorf("Windows Server MSiSCSI environment check failed: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return configureWindowsKubernetesServiceEndpoint()
}

func platformStartsWebhooks() bool {
	// The Linux control-plane managers own the admission and conversion
	// webhooks. Windows managers run node-local controllers and data services.
	return false
}

func configureWindowsKubernetesServiceEndpoint() error {
	host := os.Getenv("KUBERNETES_SERVICE_HOST")
	port := os.Getenv("KUBERNETES_SERVICE_PORT")
	if port == "" {
		port = "443"
	}
	if host != "" {
		connection, err := net.DialTimeout("tcp", net.JoinHostPort(host, port), 2*time.Second)
		if err == nil {
			_ = connection.Close()
			return nil
		}
	}

	// Some Windows kube-proxy/HNS combinations create Service policies that
	// are unreachable from HostProcess containers. Kubelet already has a local
	// API proxy endpoint; reuse only its address while retaining this pod's
	// service-account token and CA through rest.InClusterConfig.
	paths := []string{
		`C:\var\lib\rancher\rke2\agent\kubelet.kubeconfig`,
		`C:\var\lib\kubelet\kubeconfig`,
		`C:\etc\kubernetes\kubelet.conf`,
	}
	for _, path := range paths {
		endpoint, err := kubeconfigServerEndpoint(path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return err
		}
		if err := os.Setenv("KUBERNETES_SERVICE_HOST", endpoint.Hostname()); err != nil {
			return err
		}
		endpointPort := endpoint.Port()
		if endpointPort == "" {
			endpointPort = "443"
		}
		return os.Setenv("KUBERNETES_SERVICE_PORT", endpointPort)
	}
	return fmt.Errorf("Kubernetes Service VIP %s is unreachable and no kubelet kubeconfig was found", net.JoinHostPort(host, port))
}

func kubeconfigServerEndpoint(path string) (*url.URL, error) {
	file, err := os.Open(filepath.Clean(path))
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "server:") {
			continue
		}
		endpoint, err := url.Parse(strings.TrimSpace(strings.TrimPrefix(line, "server:")))
		if err != nil || endpoint.Hostname() == "" {
			return nil, fmt.Errorf("invalid Kubernetes server endpoint in %s", path)
		}
		return endpoint, nil
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return nil, fmt.Errorf("Kubernetes server endpoint is absent from %s", path)
}
