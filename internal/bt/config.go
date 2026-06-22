package bt

import (
	"encoding/json"
	"os"
	"path/filepath"
)

func ConfigPath() string {
	if dir, err := os.UserConfigDir(); err == nil {
		return filepath.Join(dir, "tws_manager", "devices.json")
	}
	return filepath.Join(".", "devices.json")
}

func LoadConfig(path string) (Config, error) {
	return loadConfigCached(path)
}

func loadConfigFromDisk(path string) (Config, error) {
	cfg := Config{Devices: map[string]string{}, Channels: map[string]int{}}
	data, err := os.ReadFile(path)
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
