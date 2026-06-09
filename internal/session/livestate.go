package session

import (
	"fmt"
	"strings"

	"tws_manager/internal/spp"
)

// clearLiveStateLocked resets per-connection data. Caller holds s.mu.
// A manually selected model (--model / UI pick) survives reconnects;
// auto-detected models are re-detected for the new device.
func (s *Session) clearLiveStateLocked() {
	s.batteries = map[string]spp.Battery{}
	s.config = map[string]string{}
	s.dualList = nil
	s.pending = map[byte]pendingTX{}
	if !s.manualModel {
		s.model = spp.DefaultModel()
	}
}

func mergeBatteries(current, update map[string]spp.Battery) map[string]spp.Battery {
	out := cloneBatteries(current)
	if out == nil {
		out = map[string]spp.Battery{}
	}
	for part, battery := range update {
		out[part] = battery
	}
	if _, ok := update["case"]; ok {
		delete(out, "stereo")
	}
	return out
}

func cloneBatteries(src map[string]spp.Battery) map[string]spp.Battery {
	if len(src) == 0 {
		return nil
	}
	out := make(map[string]spp.Battery, len(src))
	for part, battery := range src {
		out[part] = battery
	}
	return out
}

// recordConfig stores the latest decoded device configuration (ANC, low
// latency, dual, EQ, spatial) so the UI can render it after auto-discovery.
func (s *Session) recordConfig(parsed spp.ParsedPacket) {
	var key string
	switch parsed.Kind {
	case "anc_response", "anc_changed":
		key = "anc"
	case "lag_response", "lag_changed":
		key = "lag"
	case "eq_response":
		key = "eq"
	case "spatial_response":
		key = "spatial"
	case "dual_response", "dual_switch_changed":
		key = "dual"
	default:
		return
	}
	value := parsed.Summary
	if _, rest, ok := strings.Cut(parsed.Summary, ": "); ok {
		value = rest
	}
	if value == "" {
		return
	}
	s.mu.Lock()
	s.config[key] = value
	s.mu.Unlock()
}

// matchRequest pairs an incoming response with the originating request by the
// echoed FSN. Falls back to deriving the request command from the response
// command (0x40xx -> 0xC0xx).
func (s *Session) matchRequest(pkt spp.Packet) string {
	s.mu.Lock()
	p, ok := s.pending[pkt.FSN]
	if ok {
		delete(s.pending, pkt.FSN)
	}
	s.mu.Unlock()
	if ok {
		return p.command
	}
	if pkt.Cmd&0xF000 == 0x4000 {
		return fmt.Sprintf("%04x", pkt.Cmd|0x8000)
	}
	return ""
}
