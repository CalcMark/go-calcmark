package config

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"charm.land/lipgloss/v2/compat"
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
	if fallbackPath, err := FallbackConfigPath(); err == nil {
		if _, statErr := os.Stat(fallbackPath); statErr == nil {
			v.SetConfigFile(fallbackPath)
			_ = v.MergeInConfig() // Ignore errors - malformed config uses defaults
		}
	}
	if xdgPath, err := XDGConfigPath(); err == nil {
		if _, statErr := os.Stat(xdgPath); statErr == nil {
			v.SetConfigFile(xdgPath)
			_ = v.MergeInConfig()
		}
	}

	// 3. Check for deprecated theme keys before unmarshal
	warnDeprecatedThemeKeys(v)

	// 4. Unmarshal into struct
	var c Config
	if err := v.Unmarshal(&c); err != nil {
		return nil, err
	}

	// 5. Warn about deprecated color_mode and dark_mode values
	warnDeprecatedColorMode(c.TUI.ColorMode, v.IsSet("tui.dark_mode"))

	// 6. Apply color mode to lipgloss
	// NOTE: This is called during cobra PersistentPreRunE, which happens
	// BEFORE alternate screen is entered. We must not trigger any terminal
	// queries here or they will cause visible artifacts.
	applyColorMode(c.TUI.ColorMode, c.TUI.DarkMode)

	return &c, nil
}

// warnDeprecatedThemeKeys checks for unknown/deprecated keys under [tui.theme]
// and logs warnings to stderr. This helps users discover that their old
// customizations for removed fields need updating.
func warnDeprecatedThemeKeys(v *viper.Viper) {
	known := ThemeConfigKnownKeys()
	prefix := "tui.theme."

	for _, key := range v.AllKeys() {
		if !strings.HasPrefix(key, prefix) {
			continue
		}
		fieldName := strings.TrimPrefix(key, prefix)
		if !known[fieldName] {
			// Only warn if the user actually set a non-default value
			val := v.GetString(key)
			if val != "" {
				fmt.Fprintf(os.Stderr, "calcmark: deprecated config key [tui.theme].%s — this key has been removed. See docs for the simplified theme config.\n", fieldName)
			}
		}
	}
}

// warnDeprecatedColorMode logs warnings for deprecated color_mode and dark_mode usage.
func warnDeprecatedColorMode(colorMode string, darkModeExplicit bool) {
	if strings.EqualFold(colorMode, "auto") {
		fmt.Fprintf(os.Stderr, "calcmark: color_mode=\"auto\" is deprecated and now treated as \"dark\". Set color_mode=\"light\" or color_mode=\"dark\" explicitly.\n")
	}
	if darkModeExplicit {
		fmt.Fprintf(os.Stderr, "calcmark: dark_mode is deprecated. Use color_mode=\"dark\" or color_mode=\"light\" instead.\n")
	}
}

// applyColorMode sets lipgloss's background detection based on config.
// Priority: color_mode > dark_mode (for backward compatibility).
//
// IMPORTANT: We do NOT trigger terminal queries here because that would
// query the terminal BEFORE alternate screen mode is entered, causing
// terminal artifacts to appear on screen.
func applyColorMode(colorMode string, darkMode bool) {
	switch strings.ToLower(colorMode) {
	case "light":
		compat.HasDarkBackground = false
	case "dark":
		compat.HasDarkBackground = true
	case "auto":
		// Deprecated: "auto" resolves to dark
		compat.HasDarkBackground = true
	case "":
		// Legacy fallback: use dark_mode setting
		compat.HasDarkBackground = darkMode
	default:
		// Invalid color_mode, fall back to dark
		compat.HasDarkBackground = true
	}
}

// IsDarkMode returns whether we're rendering for dark background.
func IsDarkMode() bool {
	return compat.HasDarkBackground
}

// ApplyColorModeOverride applies a color mode override (from CLI flag or env var).
// This is used to override the config file setting.
func ApplyColorModeOverride(colorMode string) {
	// Use empty darkMode (false) since we're overriding
	applyColorMode(colorMode, false)
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

// DefaultsTOML returns the raw embedded defaults.toml content.
func DefaultsTOML() string {
	return defaultsToml
}

// XDGConfigPath returns the XDG config file path for the current user.
// Respects $XDG_CONFIG_HOME per the XDG Base Directory Specification.
// Falls back to ~/.config/calcmark/config.toml if not set.
func XDGConfigPath() (string, error) {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "calcmark", "config.toml"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	return filepath.Join(home, ".config", "calcmark", "config.toml"), nil
}

// FallbackConfigPath returns the legacy dotfile config path (~/.calcmarkrc.toml).
func FallbackConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	return filepath.Join(home, ".calcmarkrc.toml"), nil
}
