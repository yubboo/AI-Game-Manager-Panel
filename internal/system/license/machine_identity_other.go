//go:build !windows

package license

import (
	"os"
	"runtime"
	"strings"
)

func machineIdentity() string {
	for _, p := range []string{"/etc/machine-id", "/var/lib/dbus/machine-id"} {
		if b, err := os.ReadFile(p); err == nil && strings.TrimSpace(string(b)) != "" {
			return strings.TrimSpace(string(b))
		}
	}
	host, _ := os.Hostname()
	return runtime.GOOS + "|" + runtime.GOARCH + "|" + host
}
