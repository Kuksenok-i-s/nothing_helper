//go:build linux

package bt

import "strings"

// mergeBluetoothLists combines paired and connected device lists by MAC.
func mergeBluetoothLists(paired, connected []Device) map[string]Device {
	devices := map[string]Device{}
	for _, dev := range paired {
		dev.Paired = true
		devices[dev.MAC] = dev
	}
	for _, dev := range connected {
		cur := devices[dev.MAC]
		if cur.MAC == "" {
			cur = dev
		}
		cur.Connected = true
		if cur.Name == "" {
			cur.Name = dev.Name
		}
		devices[dev.MAC] = cur
	}
	return devices
}

// enrichDiscoveredDevice applies bluetoothctl info to a device entry.
func enrichDiscoveredDevice(dev Device, info string) Device {
	dev.Info = info
	applyBluetoothInfo(&dev, info)
	dev.SPP = strings.Contains(strings.ToUpper(info), NothingSPPUUID)
	dev.Channel = ResolveDeviceChannel(dev.MAC, DefaultRFCOMMChannel)
	return dev
}

// isDiscoveredCandidate reports whether a device should appear in Discover results.
func isDiscoveredCandidate(dev Device) bool {
	return isCandidate(dev) || dev.SPP
}
