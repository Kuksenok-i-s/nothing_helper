package dualpolicy

import (
	"fmt"
	"os/exec"
	"strings"

	"tws_manager/internal/security"
)

// HostAdapterMAC returns the local default Bluetooth controller address.
func HostAdapterMAC() (string, error) {
	out, err := exec.Command("bluetoothctl", "show").CombinedOutput()
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
