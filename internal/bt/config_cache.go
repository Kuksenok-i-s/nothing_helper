package bt

import "sync"

var (
	configCacheMu sync.RWMutex
	configCache   = map[string]Config{}
	configLoadErr = map[string]error{}
)

func loadConfigCached(path string) (Config, error) {
	configCacheMu.RLock()
	if cfg, ok := configCache[path]; ok {
		err := configLoadErr[path]
		configCacheMu.RUnlock()
		return cfg, err
	}
	configCacheMu.RUnlock()

	configCacheMu.Lock()
	defer configCacheMu.Unlock()
	if cfg, ok := configCache[path]; ok {
		return cfg, configLoadErr[path]
	}
	cfg, err := loadConfigFromDisk(path)
	configCache[path] = cfg
	configLoadErr[path] = err
	return cfg, err
}

func invalidateConfigCache(path string) {
	configCacheMu.Lock()
	defer configCacheMu.Unlock()
	delete(configCache, path)
	delete(configLoadErr, path)
}

func invalidateAllConfigCache() {
	configCacheMu.Lock()
	defer configCacheMu.Unlock()
	configCache = map[string]Config{}
	configLoadErr = map[string]error{}
}
