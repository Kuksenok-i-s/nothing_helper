package bt

import (
	"encoding/json"
	"os"
	"path/filepath"
)

var configPathOverride func() string
var bluetoothInfoFn = BluetoothInfo

func ConfigPath() string {
	if configPathOverride != nil {
		return configPathOverride()
	}
	if dir, err := os.UserConfigDir(); err == nil {
		return filepath.Join(dir, "nothing_helper", "devices.json")
	}
	return filepath.Join(".", "devices.json")
}

func LoadConfig(path string) (Config, error) {
	return loadConfigCached(path)
}

func loadConfigFromDisk(path string) (Config, error) {
	cfg := Config{Devices: map[string]string{}, Channels: map[string]int{}}
	data, err := os.ReadFile(path)
	// Read the previous app's settings only when the new default file is absent.
	// SaveConfig always writes to the new path, so subsequent edits migrate it.
	if os.IsNotExist(err) {
		if dir, configErr := os.UserConfigDir(); configErr == nil && path == filepath.Join(dir, "nothing_helper", "devices.json") {
			data, err = os.ReadFile(filepath.Join(dir, "tws_manager", "devices.json"))
		}
	}
	if os.IsNotExist(err) {
		return cfg, nil
	}
	if err != nil {
		return cfg, err
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}
	return sanitizeConfig(cfg), nil
}

func SaveConfig(path string, cfg Config) error {
	cfg = sanitizeConfig(cfg)
	if err := os.MkdirAll(filepath.Dir(path), privateDirPerm); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, append(data, '\n'), privateFilePerm); err != nil {
		return err
	}
	invalidateConfigCache(path)
	return nil
}

// SetConfigPathHook overrides ConfigPath in tests. Pass nil to restore default.
func SetConfigPathHook(fn func() string) {
	configPathOverride = fn
}

// SetBluetoothInfoHook overrides Bluetooth lookups in tests. Pass nil to restore default.
func SetBluetoothInfoHook(fn func(string) (string, error)) {
	if fn == nil {
		bluetoothInfoFn = BluetoothInfo
		return
	}
	bluetoothInfoFn = fn
}
