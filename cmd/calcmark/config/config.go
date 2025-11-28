package config

import (
	_ "embed"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/spf13/viper"
)

//go:embed defaults.toml
var defaultsToml string

var (
	cfg     *Config
	styles  Styles
	once    sync.Once
	loadErr error
)

// Load initializes configuration from embedded defaults and user config files.
// Safe to call multiple times; only loads once.
// Returns the config and any error from loading.
func Load() (*Config, error) {
	once.Do(func() {
		cfg, loadErr = load()
		if cfg != nil {
			styles = cfg.TUI.Theme.BuildStyles()
		}
	})
	return cfg, loadErr
}

// Get returns the loaded configuration.
// Panics if Load() hasn't been called or failed.
func Get() *Config {
	if cfg == nil {
		panic("config.Load() must be called before config.Get()")
	}
	return cfg
}

// GetStyles returns pre-built lipgloss styles from the loaded theme.
// Panics if Load() hasn't been called or failed.
func GetStyles() Styles {
	if cfg == nil {
		panic("config.Load() must be called before config.GetStyles()")
	}
	return styles
}

// load performs the actual configuration loading.
func load() (*Config, error) {
	v := viper.New()
	v.SetConfigType("toml")

	// 1. Load embedded defaults (always succeeds or panics at build time)
	if err := v.ReadConfig(strings.NewReader(defaultsToml)); err != nil {
		// Invalid embedded defaults is a build-time error
		panic("invalid embedded defaults.toml: " + err.Error())
	}

	// 2. Merge user config files (order matters: later overrides earlier)
	home, err := os.UserHomeDir()
	if err == nil && home != "" {
		// Fallback: ~/.calcmarkrc.toml (lower priority)
		fallbackPath := filepath.Join(home, ".calcmarkrc.toml")
		if _, statErr := os.Stat(fallbackPath); statErr == nil {
			v.SetConfigFile(fallbackPath)
			_ = v.MergeInConfig() // Ignore errors - malformed config uses defaults
		}

		// Primary: ~/.config/calcmark/config.toml (XDG standard, higher priority)
		xdgPath := filepath.Join(home, ".config", "calcmark", "config.toml")
		if _, statErr := os.Stat(xdgPath); statErr == nil {
			v.SetConfigFile(xdgPath)
			_ = v.MergeInConfig()
		}
	}

	// 3. Unmarshal into struct
	var c Config
	if err := v.Unmarshal(&c); err != nil {
		return nil, err
	}

	return &c, nil
}

// Reload forces a fresh config load. Use for testing only.
func Reload() (*Config, error) {
	once = sync.Once{}
	cfg = nil
	styles = Styles{}
	loadErr = nil
	return Load()
}

// Error returns any error from the last load attempt.
func Error() error {
	return loadErr
}
