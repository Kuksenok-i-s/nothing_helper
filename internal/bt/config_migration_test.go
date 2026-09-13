package bt

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRenamedAppConfigMigration(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HOME", root)
	t.Setenv("XDG_CONFIG_HOME", root)
	dir, err := os.UserConfigDir()
	if err != nil {
		t.Fatal(err)
	}
	legacy := filepath.Join(dir, "tws_manager", "devices.json")
	current := filepath.Join(dir, "nothing_helper", "devices.json")
	old := Config{Channels: map[string]int{"AA:BB:CC:DD:EE:FF": 15}}
	if err := SaveConfig(legacy, old); err != nil {
		t.Fatal(err)
	}
	cfg, err := loadConfigFromDisk(current)
	if err != nil || cfg.Channels["AA:BB:CC:DD:EE:FF"] != old.Channels["AA:BB:CC:DD:EE:FF"] {
		t.Fatalf("legacy config not loaded: %+v, %v", cfg, err)
	}
	cfg.Channels["AA:BB:CC:DD:EE:FF"] = 16
	if err := SaveConfig(current, cfg); err != nil {
		t.Fatal(err)
	}
	got, err := loadConfigFromDisk(current)
	if err != nil || got.Channels["AA:BB:CC:DD:EE:FF"] != cfg.Channels["AA:BB:CC:DD:EE:FF"] {
		t.Fatalf("new config not preferred: %+v, %v", got, err)
	}
	if err := os.WriteFile(current, []byte("invalid"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := loadConfigFromDisk(current); err == nil {
		t.Fatal("corrupt new config must not silently fall back to old config")
	}
}
