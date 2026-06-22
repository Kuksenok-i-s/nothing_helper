package bt

import "strings"

const (
	NothingSPPUUID  = "AEAC4A03-DFF5-498F-843A-34487CF133EB"
	privateDirPerm  = 0o700
	privateFilePerm = 0o600
)

type Device struct {
	MAC       string `json:"mac"`
	Name      string `json:"name"`
	Alias     string `json:"alias,omitempty"`
	Info      string `json:"info,omitempty"`
	Connected bool   `json:"connected"`
	Paired    bool   `json:"paired"`
	SPP       bool   `json:"spp"`
	Channel   int    `json:"channel"`
}

type Config struct {
	Devices  map[string]string `json:"devices"`
	Channels map[string]int    `json:"channels,omitempty"`
}

// RFCOMMProgress reports recovery steps (release, bind, reopen).
type RFCOMMProgress func(step string)

func isCandidate(dev Device) bool {
	name := strings.ToLower(dev.Name)
	alias := strings.ToLower(dev.Alias)
	for _, token := range []string{"nothing", "cmf", "ear", "headphone", "neckband"} {
		if strings.Contains(name, token) || strings.Contains(alias, token) {
			return true
		}
	}
	return false
}

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
