//go:build darwin

package bt

import (
	"fmt"

	"tws_manager/internal/security"
)

func sanitizeConfig(cfg Config) Config {
	out := Config{Devices: map[string]string{}, Channels: map[string]int{}}
	if cfg.Devices != nil {
		for key, mac := range cfg.Devices {
			normMAC, err := security.NormalizeMAC(mac)
			if err != nil {
				continue
			}
			if key != "" {
				if _, err := security.ValidateTransportRef(key); err != nil {
					if _, err2 := security.NormalizeMAC(key); err2 != nil {
						continue
					}
				}
			}
			out.Devices[key] = normMAC
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

// LookupDeviceMAC returns a saved MAC for a transport ref or MAC key.
func LookupDeviceMAC(ref string) (string, bool) {
	if ref == "" {
		return "", false
	}
	if mac, err := security.NormalizeMAC(ref); err == nil {
		return mac, true
	}
	ref, err := security.ValidateTransportRef(ref)
	if err != nil {
		return "", false
	}
	if mac, err := security.NormalizeMAC(ref); err == nil {
		return mac, true
	}
	cfg, err := LoadConfig(ConfigPath())
	if err != nil {
		return "", false
	}
	mac, ok := cfg.Devices[ref]
	return mac, ok && mac != ""
}

// RememberDeviceMAC persists transportRef -> MAC on Darwin (ref may be MAC or rfcomm:MAC:CH).
func RememberDeviceMAC(transportRef, mac string) error {
	normMAC, err := security.NormalizeMAC(mac)
	if err != nil {
		return err
	}
	key := transportRef
	if key == "" {
		key = normMAC
	} else if _, err := security.ValidateTransportRef(key); err != nil {
		if normalized, normErr := security.NormalizeMAC(key); normErr == nil {
			key = normalized
		} else {
			return fmt.Errorf("invalid transport ref %q: %w", transportRef, err)
		}
	}
	path := ConfigPath()
	cfg, err := LoadConfig(path)
	if err != nil {
		return err
	}
	cfg.Devices[key] = normMAC
	return SaveConfig(path, cfg)
}

// TransportLabel builds the canonical Darwin transport label.
func TransportLabel(mac string, channel int) string {
	return fmt.Sprintf("rfcomm:%s:%d", mac, channel)
}
