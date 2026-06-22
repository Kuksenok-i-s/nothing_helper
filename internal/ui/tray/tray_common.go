package tray

import (
	"fmt"
	"strings"

	"tws_manager/internal/session"
	"tws_manager/internal/spp"
)

func deviceDisplayName(snap session.Snapshot) string {
	name := snap.Model.Product
	if name == "" {
		name = snap.Model.Codename
	}
	if name == "" {
		name = snap.Device.Name
	}
	return name
}

func statusTitle(snap session.Snapshot) string {
	name := deviceDisplayName(snap)
	if snap.Connected {
		if name == "" {
			name = "device"
		}
		return "Connected: " + name
	}
	return "Disconnected"
}

func tooltipForSnapshot(snap session.Snapshot) string {
	name := snap.Model.Codename
	if name == "" {
		name = snap.Device.Name
	}
	if name == "" {
		name = "tws_manager"
	}
	return strings.TrimSpace(fmt.Sprintf("%s · %s", name, formatBatteries(snap.Batteries)))
}

func formatBatteries(data map[string]spp.Battery) string {
	if len(data) == 0 {
		return "n/a"
	}
	parts := make([]string, 0, 3)
	labels := map[string]string{"left": "L", "right": "R", "case": "C", "stereo": "S"}
	for _, key := range []string{"left", "right", "case", "stereo"} {
		if b, ok := data[key]; ok {
			tag := fmt.Sprintf("%s%d%%", labels[key], b.Percent)
			if b.Charging {
				tag += "+"
			}
			parts = append(parts, tag)
		}
	}
	return strings.Join(parts, " ")
}
