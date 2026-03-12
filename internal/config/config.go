// Package config handles d9s configuration.
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// Config holds the d9s application configuration.
type Config struct {
	DefaultContext  string        `json:"default_context"`
	StatsInterval   time.Duration `json:"stats_interval"`
	LogTailLines    int           `json:"log_tail_lines"`
	Theme           string        `json:"theme"` // "dark" or "light"
	RefreshInterval time.Duration `json:"refresh_interval"`
}

// Default returns a Config with sensible defaults.
func Default() *Config {
	return &Config{
		DefaultContext:  "",
		StatsInterval:   2 * time.Second,
		LogTailLines:    200,
		Theme:           "dark",
		RefreshInterval: 5 * time.Second,
	}
}

// configFilePath returns the path to the config file.
func configFilePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".d9s.json"
	}
	return filepath.Join(home, ".config", "d9s", "config.json")
}

// Load reads the config file, falling back to defaults if missing.
func Load() (*Config, error) {
	cfg := Default()
	path := configFilePath()

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return cfg, nil
	}
	if err != nil {
		return cfg, err
	}

	if err := json.Unmarshal(data, cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

// Save writes the config to disk.
func Save(cfg *Config) error {
	path := configFilePath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
