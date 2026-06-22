//go:build linux

package bt

import (
	"tws_manager/internal/security"
)

func sanitizeConfig(cfg Config) Config {
	out := Config{Devices: map[string]string{}, Channels: map[string]int{}}
	if cfg.Devices != nil {
		for path, mac := range cfg.Devices {
			devPath, err := security.ValidateRFCOMMDevice(path)
			if err != nil {
				continue
			}
			normMAC, err := security.NormalizeMAC(mac)
			if err != nil {
				continue
			}
			out.Devices[devPath] = normMAC
		}
	}
	if cfg.Channels != nil {
		for mac, channel := range cfg.Channels {
			normMAC, err := security.NormalizeMAC(mac)
			if err != nil {
				continue
			}
			if err := security.ValidateChannel(channel); err != nil {
				continue
			}
			out.Channels[normMAC] = channel
		}
	}
	return out
}

// LookupDeviceMAC returns a saved MAC for an RFCOMM device path.
func LookupDeviceMAC(devicePath string) (string, bool) {
	devPath, err := security.ValidateRFCOMMDevice(devicePath)
	if err != nil {
		return "", false
	}
	cfg, err := LoadConfig(ConfigPath())
	if err != nil {
		return "", false
	}
	mac, ok := cfg.Devices[devPath]
	return mac, ok && mac != ""
}

// RememberDeviceMAC persists devicePath -> MAC for later revive without --addr.
func RememberDeviceMAC(devicePath, mac string) error {
	devPath, err := security.ValidateRFCOMMDevice(devicePath)
	if err != nil {
		return err
	}
	normMAC, err := security.NormalizeMAC(mac)
	if err != nil {
		return err
	}
	path := ConfigPath()
	cfg, err := LoadConfig(path)
	if err != nil {
		return err
	}
	cfg.Devices[devPath] = normMAC
	return SaveConfig(path, cfg)
}
