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
	old := Config{Devices: map[string]string{"/dev/rfcomm0": "AA:BB:CC:DD:EE:FF"}}
	if err := SaveConfig(legacy, old); err != nil {
		t.Fatal(err)
	}
	cfg, err := loadConfigFromDisk(current)
	if err != nil || cfg.Devices["/dev/rfcomm0"] != old.Devices["/dev/rfcomm0"] {
		t.Fatalf("legacy config not loaded: %+v, %v", cfg, err)
	}
	cfg.Devices["/dev/rfcomm0"] = "11:22:33:44:55:66"
	if err := SaveConfig(current, cfg); err != nil {
		t.Fatal(err)
	}
	got, err := loadConfigFromDisk(current)
	if err != nil || got.Devices["/dev/rfcomm0"] != cfg.Devices["/dev/rfcomm0"] {
		t.Fatalf("new config not preferred: %+v, %v", got, err)
	}
	if err := os.WriteFile(current, []byte("invalid"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := loadConfigFromDisk(current); err == nil {
		t.Fatal("corrupt new config must not silently fall back to old config")
	}
}
