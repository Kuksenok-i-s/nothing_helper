package session

import (
	"strings"

	"tws_manager/internal/bt"
	"tws_manager/internal/security"
)

func validateConnectInputs(transportRef string, channel int) error {
	if transportRef != "" {
		if _, err := security.ValidateTransportRef(transportRef); err != nil {
			return err
		}
	}
	return security.ValidateChannel(channel)
}

func enrichConnectDevice(device bt.Device, transportRef string) bt.Device {
	device = fillConnectDeviceMAC(device, transportRef)
	return enrichConnectDeviceInfo(device)
}

func fillConnectDeviceMAC(device bt.Device, transportRef string) bt.Device {
	if device.MAC == "" {
		if mac, ok := bt.LookupDeviceMAC(transportRef); ok {
			device.MAC = mac
			if device.Name == "" || device.Name == transportRef {
				device.Name = mac
			}
		}
	}
	return device
}

func enrichConnectDeviceInfo(device bt.Device) bt.Device {
	if device.MAC != "" && (device.Info == "" || device.Name == "" || strings.EqualFold(device.Name, device.MAC)) {
		device = bt.EnrichDeviceInfo(device)
	}
	return device
}

func connectProgressLabel(device bt.Device, transportRef string) string {
	if transportRef != "" {
		return transportRef
	}
	if device.MAC != "" {
		return device.MAC
	}
	return ""
}

func prepareConnectDevice(device bt.Device, transportRef string, channel int) (bt.Device, int, string, error) {
	if err := validateConnectInputs(transportRef, channel); err != nil {
		return device, channel, "", err
	}
	device = enrichConnectDevice(device, transportRef)
	channel = bt.ResolveDeviceChannel(device.MAC, channel)
	return device, channel, connectProgressLabel(device, transportRef), nil
}
