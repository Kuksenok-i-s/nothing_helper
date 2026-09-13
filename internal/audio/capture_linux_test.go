//go:build linux

package audio

import (
	"encoding/json"
	"testing"
)

func TestActiveCaptureRequiresPhysicalRunningHFPForPeer(t *testing.T) {
	for _, tc := range []struct {
		name, state, class, profile, address string
		want                                 bool
	}{
		{"HFP recording", "running", "Audio/Source", "headset-head-unit", "AA:BB:CC:DD:EE:FF", true},
		{"virtual A2DP source", "running", "Audio/Source", "", "AA:BB:CC:DD:EE:FF", false},
		{"suspended HFP", "suspended", "Audio/Source", "headset-head-unit", "AA:BB:CC:DD:EE:FF", false},
		{"playback only", "running", "Audio/Sink", "headset-head-unit", "AA:BB:CC:DD:EE:FF", false},
		{"other headset", "running", "Audio/Source", "headset-head-unit", "00:11:22:33:44:55", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw, _ := json.Marshal([]any{map[string]any{"info": map[string]any{"state": tc.state, "props": map[string]any{"media.class": tc.class, "api.bluez5.profile": tc.profile, "api.bluez5.address": tc.address, "object.id": 17}}}})
			got, err := activeCaptureFromDump(raw, "AA:BB:CC:DD:EE:FF")
			if err != nil || got != tc.want {
				t.Fatalf("got %v, %v", got, err)
			}
		})
	}
	if _, err := activeCaptureFromDump([]byte("invalid"), "AA:BB:CC:DD:EE:FF"); err == nil {
		t.Fatal("malformed output accepted")
	}
}
