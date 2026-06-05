package bt

import (
	"os"
	"strings"
	"testing"
)

func TestChannelCandidates(t *testing.T) {
	got := channelCandidates(7)
	if len(got) < 3 || got[0] != 7 {
		t.Fatalf("candidates = %v, want 7 first", got)
	}
	if got[1] != DefaultRFCOMMChannel {
		t.Fatalf("candidates[1] = %d, want %d", got[1], DefaultRFCOMMChannel)
	}
	seen := map[int]struct{}{}
	for _, ch := range got {
		if _, ok := seen[ch]; ok {
			t.Fatalf("duplicate channel %d in %v", ch, got)
		}
		seen[ch] = struct{}{}
	}
	if _, ok := seen[DefaultRFCOMMChannel]; !ok {
		t.Fatalf("missing default channel in %v", got)
	}
}

func TestResolveDeviceChannel(t *testing.T) {
	if got := ResolveDeviceChannel("", 0); got != DefaultRFCOMMChannel {
		t.Fatalf("empty = %d", got)
	}
	if got := ResolveDeviceChannel("", 20); got != 20 {
		t.Fatalf("hint = %d", got)
	}
}

func TestSanitizeConfigChannels(t *testing.T) {
	cfg := sanitizeConfig(Config{
		Devices: map[string]string{"/dev/rfcomm0": "aa:bb:cc:dd:ee:ff"},
		Channels: map[string]int{
			"aa:bb:cc:dd:ee:ff": 9,
			"bad-mac":           10,
			"11:22:33:44:55:66": 99,
		},
	})
	if cfg.Channels["AA:BB:CC:DD:EE:FF"] != 9 {
		t.Fatalf("channels = %+v", cfg.Channels)
	}
	if len(cfg.Channels) != 1 {
		t.Fatalf("channels = %+v, want only valid entry", cfg.Channels)
	}
}

func TestSaveConfigRoundTripChannels(t *testing.T) {
	path := t.TempDir() + "/cfg.json"
	cfg := sanitizeConfig(Config{
		Devices:  map[string]string{"/dev/rfcomm0": "AA:BB:CC:DD:EE:FF"},
		Channels: map[string]int{"AA:BB:CC:DD:EE:FF": 7},
	})
	if err := SaveConfig(path, cfg); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Channels["AA:BB:CC:DD:EE:FF"] != 7 {
		t.Fatalf("loaded channels = %+v", loaded.Channels)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"channels"`) {
		t.Fatalf("config missing channels key: %s", data)
	}
}
