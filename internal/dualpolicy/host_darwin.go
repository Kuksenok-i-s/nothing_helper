//go:build darwin

package dualpolicy

import (
	"fmt"
	"os/exec"
	"regexp"

	"tws_manager/internal/bt"
	"tws_manager/internal/security"
)

var systemProfilerMAC = regexp.MustCompile(`(?i)Bluetooth Address:\s*([0-9A-F:]{17})`)

// HostAdapterMAC returns the local Bluetooth controller address.
func HostAdapterMAC() (string, error) {
	if mac, err := bt.HostAdapterMAC(); err == nil && mac != "" {
		return mac, nil
	}
	out, err := exec.Command("system_profiler", "SPBluetoothDataType").CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("system_profiler SPBluetoothDataType: %w", err)
	}
	if m := systemProfilerMAC.FindSubmatch(out); len(m) == 2 {
		return security.NormalizeMAC(string(m[1]))
	}
	return "", fmt.Errorf("host adapter MAC not found in system_profiler output")
}
