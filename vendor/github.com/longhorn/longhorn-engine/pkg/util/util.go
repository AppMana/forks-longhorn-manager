package util

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/handlers"
	"github.com/sirupsen/logrus"

	"github.com/longhorn/longhorn-engine/pkg/types"
)

var (
	MaximumVolumeNameSize = 64
	validVolumeName       = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]+$`)

	unixDomainSocketDirectoryInContainer = "/host/var/lib/longhorn/unix-domain-socket/"
)

const (
	BlockSizeLinux = 512

	randomIDLenth = 8
)

func ParseAddresses(name string) (string, string, string, int, error) {
	host, strPort, err := net.SplitHostPort(name)
	if err != nil {
		return "", "", "", 0, fmt.Errorf("invalid address %s : couldn't find host and port", name)
	}

	port, _ := strconv.Atoi(strPort)

	return net.JoinHostPort(host, strconv.Itoa(port)),
		net.JoinHostPort(host, strconv.Itoa(port+1)),
		net.JoinHostPort(host, strconv.Itoa(port+2)),
		port + 2, nil
}

func GetGRPCAddress(address string) string {
	address = strings.TrimPrefix(address, "tcp://")

	address = strings.TrimPrefix(address, "http://")

	address = strings.TrimSuffix(address, "/v1")

	return address
}

func GetPortFromAddress(address string) (int, error) {
	address = strings.TrimSuffix(address, "/v1")

	_, strPort, err := net.SplitHostPort(address)
	if err != nil {
		return 0, fmt.Errorf("invalid address %s, must have a port in it", address)
	}

	port, err := strconv.Atoi(strPort)
	if err != nil {
		return 0, err
	}

	return port, nil
}

func Filter(list []string, check func(string) bool) []string {
	result := make([]string, 0, len(list))
	for _, i := range list {
		if check(i) {
			result = append(result, i)
		}
	}
	return result
}

type filteredLoggingHandler struct {
	filteredPaths  map[string]struct{}
	handler        http.Handler
	loggingHandler http.Handler
}

func FilteredLoggingHandler(filteredPaths map[string]struct{}, writer io.Writer, router http.Handler) http.Handler {
	return filteredLoggingHandler{
		filteredPaths:  filteredPaths,
		handler:        router,
		loggingHandler: handlers.CombinedLoggingHandler(writer, router),
	}
}

func (h filteredLoggingHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case "GET":
		if _, exists := h.filteredPaths[req.URL.Path]; exists {
			h.handler.ServeHTTP(w, req)
			return
		}
	}
	h.loggingHandler.ServeHTTP(w, req)
}

func RemoveDevice(dev string) error {
	if _, err := os.Stat(dev); err == nil {
		if err := remove(dev); err != nil {
			return fmt.Errorf("failed to removing device %s, %v", dev, err)
		}
	}
	return nil
}

func removeAsync(path string, done chan<- error) {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		logrus.WithError(err).Errorf("Unable to remove: %v", path)
		done <- err
	}
	done <- nil
}

func remove(path string) error {
	done := make(chan error)
	go removeAsync(path, done)
	select {
	case err := <-done:
		return err
	case <-time.After(30 * time.Second):
		return fmt.Errorf("timeout trying to delete %s", path)
	}
}

func ValidVolumeName(name string) bool {
	if len(name) > MaximumVolumeNameSize {
		return false
	}
	return validVolumeName.MatchString(name)
}

func Volume2ISCSIName(name string) string {
	return strings.ReplaceAll(name, "_", ":")
}

func Now() string {
	return time.Now().UTC().Format(time.RFC3339)
}

func ParseLabels(labels []string) (map[string]string, error) {
	result := map[string]string{}
	for _, label := range labels {
		kv := strings.SplitN(label, "=", 2)
		if len(kv) != 2 {
			return nil, fmt.Errorf("invalid label not in <key>=<value> format %v", label)
		}
		key := kv[0]
		value := kv[1]
		if errList := IsQualifiedName(key); len(errList) > 0 {
			return nil, fmt.Errorf("invalid key %v for label: %v", key, errList[0])
		}
		// We don't need to validate the Label value since we're allowing for any form of data to be stored, similar
		// to Kubernetes Annotations. Of course, we should make sure it isn't empty.
		if value == "" {
			return nil, fmt.Errorf("invalid empty value for label with key %v", key)
		}
		result[key] = value
	}
	return result, nil
}

func UnescapeURL(url string) string {
	// Deal with escape in url inputted from bash
	result := strings.Replace(url, "\\u0026", "&", 1)
	result = strings.Replace(result, "u0026", "&", 1)
	result = strings.TrimLeft(result, "\"'")
	result = strings.TrimRight(result, "\"'")
	return result
}

func CheckBackupType(backupTarget string) (string, error) {
	u, err := url.Parse(backupTarget)
	if err != nil {
		return "", err
	}

	return u.Scheme, nil
}

func ResolveBackingFilepath(fileOrDirpath string) (string, error) {
	fileOrDir, err := os.Open(fileOrDirpath)
	if err != nil {
		return "", err
	}
	defer func() {
		if errClose := fileOrDir.Close(); errClose != nil {
			logrus.WithError(errClose).Errorf("Failed to close file %v", fileOrDirpath)
		}
	}()

	fileOrDirInfo, err := fileOrDir.Stat()
	if err != nil {
		return "", err
	}

	if fileOrDirInfo.IsDir() {
		files, err := fileOrDir.Readdir(-1)
		if err != nil {
			return "", err
		}
		if len(files) != 1 {
			return "", fmt.Errorf("expected exactly one file, found %d files/subdirectories", len(files))
		}
		if files[0].IsDir() {
			return "", fmt.Errorf("expected exactly one file, found a subdirectory")
		}
		return filepath.Join(fileOrDirpath, files[0].Name()), nil
	}

	return fileOrDirpath, nil
}

func GetAddresses(volumeName, address string, dataServerProtocol types.DataServerProtocol) (string, string, string, int, error) {
	switch dataServerProtocol {
	case types.DataServerProtocolTCP:
		return ParseAddresses(address)
	case types.DataServerProtocolUNIX:
		controlAddress, _, syncAddress, syncPort, err := ParseAddresses(address)
		sockPath := filepath.Join(unixDomainSocketDirectoryInContainer, volumeName+".sock")
		return controlAddress, sockPath, syncAddress, syncPort, err
	default:
		return "", "", "", -1, fmt.Errorf("unsupported protocol: %v", dataServerProtocol)
	}
}

func UUID() string {
	return uuid.New().String()
}

func RandomID() string {
	return UUID()[:randomIDLenth]
}
