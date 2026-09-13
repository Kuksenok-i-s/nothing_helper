//go:build linux

package dualpolicy

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"tws_manager/internal/security"
)

var execCombinedOutput = func(ctx context.Context, name string, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, name, args...).CombinedOutput()
}

// HostAdapterMAC returns the local default Bluetooth controller address.
func HostAdapterMAC() (string, error) {
	if hostAdapterMACHook != nil {
		return hostAdapterMACHook()
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	out, err := execCombinedOutput(ctx, "bluetoothctl", "show")
	if err != nil {
		return "", fmt.Errorf("bluetoothctl show: %w: %s", err, strings.TrimSpace(string(out)))
	}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "Controller ") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		return security.NormalizeMAC(fields[1])
	}
	return "", fmt.Errorf("bluetoothctl show: controller MAC not found")
}
