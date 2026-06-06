package dualpolicy

import (
	"fmt"
	"strings"

	"tws_manager/internal/security"
	"tws_manager/internal/spp"
)

// Mode controls PC-primary dual connection prompting.
type Mode string

const (
	ModeAsk Mode = "ask"
	ModeOff Mode = "off"
)

// ParseMode validates a --pc-primary flag value.
func ParseMode(raw string) (Mode, error) {
	switch Mode(strings.ToLower(strings.TrimSpace(raw))) {
	case "", ModeAsk:
		return ModeAsk, nil
	case ModeOff:
		return ModeOff, nil
	default:
		return "", fmt.Errorf("unknown pc-primary mode %q (want ask|off)", raw)
	}
}

// HostOwnsDual reports whether the local Bluetooth adapter is connected and
// marked owner in the dual device list.
func HostOwnsDual(devices []spp.DualDevice, hostMAC string) bool {
	host, err := security.NormalizeMAC(hostMAC)
	if err != nil || host == "" {
		return false
	}
	for _, dev := range devices {
		if !dev.Connected || !dev.Owner || dev.MAC == "" {
			continue
		}
		mac, err := security.NormalizeMAC(dev.MAC)
		if err != nil {
			continue
		}
		if mac == host {
			return true
		}
	}
	return false
}

// PhoneOwner returns the dual-list peer that currently owns the connection when
// it is not the local Bluetooth adapter (typically a phone).
func PhoneOwner(devices []spp.DualDevice, hostMAC string) (spp.DualDevice, bool) {
	host, err := security.NormalizeMAC(hostMAC)
	if err != nil || host == "" {
		return spp.DualDevice{}, false
	}
	for _, dev := range devices {
		if !dev.Connected || !dev.Owner || dev.MAC == "" {
			continue
		}
		mac, err := security.NormalizeMAC(dev.MAC)
		if err != nil {
			continue
		}
		if mac != host {
			return dev, true
		}
	}
	return spp.DualDevice{}, false
}

// ShouldPrompt reports whether the UI should offer switching dual to the PC.
// Opening the mobile app does not by itself flip owner to the phone; the prompt
// appears only when a non-host peer is Connected+Owner in GET_DUAL_DEVICE_LIST.
func ShouldPrompt(mode Mode, model spp.ModelInfo, devices []spp.DualDevice, hostMAC string) (spp.DualDevice, bool) {
	if mode != ModeAsk || !spp.ModelSupportsFeature(model, "dual") {
		return spp.DualDevice{}, false
	}
	if HostOwnsDual(devices, hostMAC) {
		return spp.DualDevice{}, false
	}
	return PhoneOwner(devices, hostMAC)
}

// HostOwnerStatus returns a short status line when the PC already owns dual.
func HostOwnerStatus(devices []spp.DualDevice, hostMAC string) string {
	if !HostOwnsDual(devices, hostMAC) {
		return ""
	}
	for _, dev := range devices {
		mac, err := security.NormalizeMAC(dev.MAC)
		host, herr := security.NormalizeMAC(hostMAC)
		if err != nil || herr != nil || mac != host {
			continue
		}
		name := dev.Name
		if name == "" {
			name = "PC"
		}
		return fmt.Sprintf("dual: %s is already the active device (owner)", name)
	}
	return "dual: this PC is already the active device (owner)"
}

// PromptText builds a user-facing question for the active phone owner.
func PromptText(phone spp.DualDevice) string {
	name := phone.Name
	if name == "" {
		name = phone.MAC
	}
	return fmt.Sprintf("%s is using dual connection. Switch audio/control to this PC?", name)
}
