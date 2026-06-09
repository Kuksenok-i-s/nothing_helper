package presenter

import (
	"fmt"
	"strings"

	"tws_manager/internal/spp"
)

// Command describes a UI-selectable protocol action.
//
// Every command exposed through the catalog is safe by construction: BuildCommands
// only emits whitelisted GET/SET presets. Unsafe operations (raw scan, arbitrary
// SETs) are enforced at the session layer and are not part of this catalog,
// except the raw-scan entry which is added only when --unsafe is active.
type Command struct {
	Title    string
	Desc     string
	Fields   []string
	Cmd      uint16
	Advanced bool
}

// IsScanCommand reports whether the command is the advanced raw scan entry.
func IsScanCommand(c Command) bool {
	return c.Advanced && strings.Contains(strings.ToLower(c.Title), "scan")
}

// IsDualAction reports whether the command connects/disconnects a dual peer.
// Those actions are surfaced via a dedicated device menu, not the GET/SET lists.
func IsDualAction(c Command) bool {
	return len(c.Fields) >= 2 && c.Fields[0] == "dual" &&
		(c.Fields[1] == "connect" || c.Fields[1] == "disconnect")
}

// IsSetCommand reports whether the command writes state to the device.
func IsSetCommand(c Command) bool {
	if c.Advanced || IsDualAction(c) {
		return false
	}
	if len(c.Fields) >= 2 && strings.EqualFold(c.Fields[1], "set") {
		return true
	}
	return c.Cmd >= 0xF000
}

// IsGetCommand reports whether the command only reads state from the device.
func IsGetCommand(c Command) bool {
	if c.Advanced || IsDualAction(c) || IsSetCommand(c) {
		return false
	}
	return true
}

// ToggleFeature is a simple on/off SET feature that the UI renders as a switch
// which doubles as a live indicator of the current device state.
type ToggleFeature struct {
	Feature   string
	Label     string
	OnFields  []string
	OffFields []string
}

// toggleLabels lists the on/off features (in display order) and their labels.
var toggleLabels = []struct{ feature, label string }{
	{"lag", "Low latency"},
	{"spatial", "Spatial audio"},
	{"dual", "Dual connection"},
}

// IsToggleSetCommand reports whether c is one half of an on/off toggle feature,
// so the GET/SET list can omit it in favour of a switch.
func IsToggleSetCommand(c Command) bool {
	if len(c.Fields) != 3 || !strings.EqualFold(c.Fields[1], "set") {
		return false
	}
	if !strings.EqualFold(c.Fields[2], "on") && !strings.EqualFold(c.Fields[2], "off") {
		return false
	}
	for _, t := range toggleLabels {
		if c.Fields[0] == t.feature {
			return true
		}
	}
	return false
}

// ToggleFeatures extracts on/off switch features from a command catalog. A
// feature is included only when both its "on" and "off" SET commands exist.
func ToggleFeatures(commands []Command) []ToggleFeature {
	type pair struct{ on, off []string }
	found := map[string]*pair{}
	for _, c := range commands {
		if !IsToggleSetCommand(c) {
			continue
		}
		p := found[c.Fields[0]]
		if p == nil {
			p = &pair{}
			found[c.Fields[0]] = p
		}
		if strings.EqualFold(c.Fields[2], "on") {
			p.on = c.Fields
		} else {
			p.off = c.Fields
		}
	}
	out := make([]ToggleFeature, 0, len(toggleLabels))
	for _, t := range toggleLabels {
		if p := found[t.feature]; p != nil && p.on != nil && p.off != nil {
			out = append(out, ToggleFeature{Feature: t.feature, Label: t.label, OnFields: p.on, OffFields: p.off})
		}
	}
	return out
}

// ToggleStateOn interprets a decoded config value (e.g. "spatial=on …",
// "low_latency=on …", "dual=on …") as a boolean on/off state.
func ToggleStateOn(feature, configValue string) bool {
	v := strings.ToLower(configValue)
	switch feature {
	case "lag":
		return strings.Contains(v, "low_latency=on")
	case "spatial":
		return strings.Contains(v, "spatial=on")
	case "dual":
		return strings.Contains(v, "dual=on")
	}
	return strings.Contains(v, "=on")
}

// BuildCommands returns the command catalog for the given model and dual device list.
func BuildCommands(model spp.ModelInfo, dualDevices []spp.DualDevice, allowUnsafe bool) []Command {
	base := []Command{
		{Title: "Info: battery", Desc: "GET battery levels", Cmd: spp.CmdGetBattery},
		{Title: "Info: identity", Desc: "GET identity", Cmd: spp.CmdGetIdentity},
		{Title: "Info: status", Desc: "GET earphone status", Cmd: spp.CmdGetStatus},
		{Title: "Info: firmware", Desc: "GET firmware version", Cmd: spp.CmdGetFirmwareVersion},
		{Title: "Info: config", Desc: "GET remote config", Cmd: spp.CmdGetRemoteConfig},
	}
	features := []Command{
		{Title: "Audio: anc get", Desc: "Current ANC mode", Fields: []string{"anc", "get"}},
		{Title: "Audio: eq get", Desc: "Current EQ mode", Fields: []string{"eq", "get"}},
		{Title: "Audio: spatial get", Desc: "Spatial audio status", Fields: []string{"spatial", "get"}},
		{Title: "Audio: low latency get", Desc: "Low-latency mode status", Fields: []string{"lag", "get"}},
		{Title: "Audio: dual get", Desc: "Dual connection status", Fields: []string{"dual", "get"}},
	}
	items := make([]Command, 0, len(base)+len(features)+16)
	for _, item := range base {
		items = append(items, item)
	}
	for _, item := range features {
		// Feature key comes from the command fields ("lag", "anc", ...), not the
		// display title: "Audio: low latency get" would otherwise resolve to "low".
		feature := item.Fields[0]
		if spp.ModelSupportsFeature(model, feature) {
			items = append(items, item)
		}
	}
	for _, item := range []Command{
		{Title: "SET: anc off", Desc: "Disable ANC", Fields: []string{"anc", "set", "off"}},
		{Title: "SET: anc strong", Desc: "ANC high", Fields: []string{"anc", "set", "strong"}},
		{Title: "SET: anc medium", Desc: "ANC mid", Fields: []string{"anc", "set", "medium"}},
		{Title: "SET: anc weak", Desc: "ANC low", Fields: []string{"anc", "set", "weak"}},
		{Title: "SET: anc adaptive", Desc: "Adaptive ANC", Fields: []string{"anc", "set", "adaptive"}},
		{Title: "SET: anc transparency", Desc: "Transparency mode", Fields: []string{"anc", "set", "transparency"}},
		{Title: "SET: eq balanced", Desc: "EQ preset 0 balanced", Fields: []string{"eq", "set", "0"}},
		{Title: "SET: eq more bass", Desc: "EQ preset 3 more bass", Fields: []string{"eq", "set", "3"}},
		{Title: "SET: spatial on", Desc: "Enable spatial audio", Fields: []string{"spatial", "set", "on"}},
		{Title: "SET: spatial off", Desc: "Disable spatial audio", Fields: []string{"spatial", "set", "off"}},
		{Title: "SET: low latency on", Desc: "Enable low-latency mode", Fields: []string{"lag", "set", "on"}},
		{Title: "SET: low latency off", Desc: "Disable low-latency mode", Fields: []string{"lag", "set", "off"}},
		{Title: "SET: dual on", Desc: "Enable dual connection", Fields: []string{"dual", "set", "on"}},
		{Title: "SET: dual off", Desc: "Disable dual connection", Fields: []string{"dual", "set", "off"}},
	} {
		feature := item.Fields[0]
		if spp.ModelSupportsFeature(model, feature) {
			items = append(items, item)
		}
	}
	for _, dev := range dualDevices {
		action := "connect"
		if dev.Connected {
			action = "disconnect"
		}
		name := dev.Name
		if name == "" {
			name = dev.MAC
		}
		items = append(items, Command{
			Title:  fmt.Sprintf("Dual: %s %s", action, name),
			Desc:   fmt.Sprintf("%s %s via SET_CONNECT_DEVICE", action, dev.MAC),
			Fields: []string{"dual", action, dev.MAC},
		})
	}
	if allowUnsafe {
		items = append(items, Command{
			Title:    "Advanced: raw scan",
			Desc:     "Comment: scan c001 c020 500ms; GET 0xC0xx only",
			Advanced: true,
		})
	}
	return items
}
