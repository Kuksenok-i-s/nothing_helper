package session

import (
	"testing"

	"nothing_helper/internal/spp"
)

func TestRecordConfig(t *testing.T) {
	tests := []struct {
		name    string
		parsed  spp.ParsedPacket
		wantKey string
		wantVal string
		wantOK  bool
	}{
		{name: "anc response", parsed: spp.ParsedPacket{Kind: "anc_response", Summary: "anc: transparency"}, wantKey: "anc", wantVal: "transparency", wantOK: true},
		{name: "anc changed", parsed: spp.ParsedPacket{Kind: "anc_changed", Summary: "anc_changed: off"}, wantKey: "anc", wantVal: "off", wantOK: true},
		{name: "lag", parsed: spp.ParsedPacket{Kind: "lag_response", Summary: "lag: low"}, wantKey: "lag", wantVal: "low", wantOK: true},
		{name: "lag changed", parsed: spp.ParsedPacket{Kind: "lag_changed", Summary: "lag_changed: high"}, wantKey: "lag", wantVal: "high", wantOK: true},
		{name: "eq", parsed: spp.ParsedPacket{Kind: "eq_response", Summary: "eq: balanced"}, wantKey: "eq", wantVal: "balanced", wantOK: true},
		{name: "spatial", parsed: spp.ParsedPacket{Kind: "spatial_response", Summary: "spatial: on"}, wantKey: "spatial", wantVal: "on", wantOK: true},
		{name: "dual", parsed: spp.ParsedPacket{Kind: "dual_response", Summary: "dual: enabled"}, wantKey: "dual", wantVal: "enabled", wantOK: true},
		{name: "dual switch", parsed: spp.ParsedPacket{Kind: "dual_switch_changed", Summary: "dual_switch_changed: 1"}, wantKey: "dual", wantVal: "1", wantOK: true},
		{name: "no colon keeps summary", parsed: spp.ParsedPacket{Kind: "eq_response", Summary: "balanced"}, wantKey: "eq", wantVal: "balanced", wantOK: true},
		{name: "empty value skipped", parsed: spp.ParsedPacket{Kind: "eq_response", Summary: "eq: "}, wantOK: false},
		{name: "unknown kind", parsed: spp.ParsedPacket{Kind: "battery", Summary: "battery: 80"}, wantOK: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := New(nil, false, false)
			s.recordConfig(tt.parsed)
			snap := s.Snapshot()
			if !tt.wantOK {
				if len(snap.Config) != 0 {
					t.Fatalf("config=%v, want empty", snap.Config)
				}
				return
			}
			if got := snap.Config[tt.wantKey]; got != tt.wantVal {
				t.Fatalf("config[%q]=%q, want %q (full=%v)", tt.wantKey, got, tt.wantVal, snap.Config)
			}
		})
	}
}

func TestConfigKeyForParsedKind(t *testing.T) {
	if key, ok := configKeyForParsedKind("anc_response"); !ok || key != "anc" {
		t.Fatalf("key=%q ok=%v", key, ok)
	}
	if _, ok := configKeyForParsedKind("battery"); ok {
		t.Fatal("expected unknown kind")
	}
}

func TestConfigValueFromSummary(t *testing.T) {
	if got := configValueFromSummary("anc: off"); got != "off" {
		t.Fatalf("value=%q", got)
	}
	if got := configValueFromSummary("plain"); got != "plain" {
		t.Fatalf("value=%q", got)
	}
}
