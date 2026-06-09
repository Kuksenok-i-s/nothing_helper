package bt

import (
	"strings"
)

// ParseBluetoothctlOutput parses bluetoothctl "info" / "show" key: value lines.
func ParseBluetoothctlOutput(text string) map[string]string {
	fields := make(map[string]string)
	for _, line := range strings.Split(text, "\n") {
		key, value, ok := strings.Cut(strings.TrimSpace(line), ":")
		if !ok {
			continue
		}
		fields[strings.ToLower(strings.TrimSpace(key))] = strings.TrimSpace(value)
	}
	return fields
}

func deviceConnectedFromInfo(info string) bool {
	fields := ParseBluetoothctlOutput(info)
	return strings.EqualFold(fields["connected"], "yes")
}

func applyBluetoothInfo(dev *Device, info string) {
	fields := ParseBluetoothctlOutput(info)
	if name := fields["name"]; name != "" && (dev.Name == "" || strings.EqualFold(dev.Name, dev.MAC)) {
		dev.Name = name
	}
	if alias := fields["alias"]; alias != "" {
		dev.Alias = alias
	}
}
