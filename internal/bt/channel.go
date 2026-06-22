package bt

import (
	"tws_manager/internal/security"
)

const DefaultRFCOMMChannel = 15

// ResolveDeviceChannel returns a saved RFCOMM channel for mac when present,
// otherwise hint, otherwise DefaultRFCOMMChannel.
func ResolveDeviceChannel(mac string, hint int) int {
	if mac != "" {
		if ch, ok := LookupDeviceChannel(mac); ok {
			return ch
		}
	}
	if hint > 0 {
		return hint
	}
	return DefaultRFCOMMChannel
}

// LookupDeviceChannel returns a persisted RFCOMM channel for the given MAC.
func LookupDeviceChannel(mac string) (int, bool) {
	normMAC, err := security.NormalizeMAC(mac)
	if err != nil {
		return 0, false
	}
	cfg, err := LoadConfig(ConfigPath())
	if err != nil {
		return 0, false
	}
	ch, ok := cfg.Channels[normMAC]
	if !ok || ch <= 0 {
		return 0, false
	}
	return ch, true
}

// RememberDeviceChannel persists the working RFCOMM channel for a MAC address.
func RememberDeviceChannel(mac string, channel int) error {
	normMAC, err := security.NormalizeMAC(mac)
	if err != nil {
		return err
	}
	if err := security.ValidateChannel(channel); err != nil {
		return err
	}
	path := ConfigPath()
	cfg, err := LoadConfig(path)
	if err != nil {
		return err
	}
	if cfg.Channels == nil {
		cfg.Channels = map[string]int{}
	}
	cfg.Channels[normMAC] = channel
	return SaveConfig(path, cfg)
}

