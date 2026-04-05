package hooks

import (
	"fmt"
	"os"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

type cachedConfig struct {
	cfg   HooksConfig
	mtime time.Time
	path  string
}

var (
	configCache   cachedConfig
	configCacheMu sync.Mutex
)

// LoadConfig loads hooks.yaml from path with mtime-based caching.
// On cache hit (same path + same mtime), no disk I/O occurs — Claude fires
// hooks on every tool use so avoiding redundant reads is critical.
// Returns (config, failOpen, error).
func LoadConfig(path string) (HooksConfig, bool, error) {
	configCacheMu.Lock()
	defer configCacheMu.Unlock()

	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return HooksConfig{}, true, fmt.Errorf("hooks.yaml not found at %s", path)
	}
	if err != nil {
		return HooksConfig{}, true, fmt.Errorf("stating %s: %w", path, err)
	}

	if configCache.path == path && !info.ModTime().After(configCache.mtime) {
		return configCache.cfg, resolveFailOpen(configCache.cfg), nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return HooksConfig{}, true, fmt.Errorf("reading %s: %w", path, err)
	}

	var cfg HooksConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return HooksConfig{}, true, fmt.Errorf("parsing %s: %w", path, err)
	}

	configCache = cachedConfig{cfg: cfg, mtime: info.ModTime(), path: path}
	return cfg, resolveFailOpen(cfg), nil
}

// resetCache clears the in-process config cache. Exposed for test isolation.
func resetCache() {
	configCacheMu.Lock()
	defer configCacheMu.Unlock()
	configCache = cachedConfig{}
}

func resolveFailOpen(cfg HooksConfig) bool {
	if cfg.Settings != nil && cfg.Settings.FailOpen != nil {
		return *cfg.Settings.FailOpen
	}
	return true
}
