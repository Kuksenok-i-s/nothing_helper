package connect

import (
	"fmt"
	"strconv"
	"strings"

	"tws_manager/internal/bt"
)

// ResolvePreflightAddress returns an explicit address or a saved MAC for devicePath.
func ResolvePreflightAddress(address, devicePath string) string {
	if address != "" {
		return address
	}
	if mac, ok := bt.LookupDeviceMAC(devicePath); ok {
		return mac
	}
	return ""
}

// ParseDeviceSelection parses a 1-based device index from interactive input.
// An empty line means skip (index 0, no error).
func ParseDeviceSelection(line string, deviceCount int) (int, error) {
	line = strings.TrimSpace(line)
	if line == "" {
		return 0, nil
	}
	index, err := strconv.Atoi(line)
	if err != nil || index < 1 || index > deviceCount {
		return 0, fmt.Errorf("invalid device selection %q", line)
	}
	return index, nil
}

// DeviceAtSelection returns the selected device, applying channel when unset.
func DeviceAtSelection(index int, devices []bt.Device, channel int) (bt.Device, error) {
	if index < 1 || index > len(devices) {
		return bt.Device{}, fmt.Errorf("invalid device selection index %d", index)
	}
	device := devices[index-1]
	if device.Channel == 0 {
		device.Channel = channel
	}
	return device, nil
}

// DeviceFromPreflightAddress builds a device when --addr is provided.
func DeviceFromPreflightAddress(address string, channel int) (bt.Device, bool, error) {
	if address == "" {
		return bt.Device{}, false, nil
	}
	dev, err := DeviceFromAddress(address, channel)
	if err != nil {
		return bt.Device{}, false, err
	}
	return dev, true, nil
}
