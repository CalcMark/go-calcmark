// Package config provides configuration management for the CalcMark TUI/CLI.
// Configuration is loaded from TOML files with embedded defaults.
package config

// Config is the root configuration structure.
type Config struct {
	// Locale is the BCP 47 locale tag for display formatting (e.g., "en-US", "de-DE").
	// Affects decimal and thousand separators in output. Top-level because locale
	// is application-wide and will eventually affect input parsing too.
	// Precedence: --locale flag > config.toml > "en-US" default.
	Locale    string          `mapstructure:"locale"`
	TUI       TUIConfig       `mapstructure:"tui"`
	Formatter FormatterConfig `mapstructure:"formatter"`
}

// TUIConfig holds TUI-specific settings.
type TUIConfig struct {
	Theme     ThemeConfig `mapstructure:"theme"`
	DarkMode  bool        `mapstructure:"dark_mode"`  // Deprecated: use ColorMode instead
	ColorMode string      `mapstructure:"color_mode"` // "light" or "dark"
}

// ThemeConfig defines user-facing color overrides as hex strings.
// These override the corresponding AdaptiveColor palette slot (light or dark)
// based on the configured color_mode. Internal structural colors (popup borders,
// separator shades, etc.) are not user-configurable and derive from the palette.
type ThemeConfig struct {
	Primary   string `mapstructure:"primary"`   // Titles, prompts, variable names
	Accent    string `mapstructure:"accent"`    // Borders, highlights
	Error     string `mapstructure:"error"`     // Error messages
	Warning   string `mapstructure:"warning"`   // Changed indicators
	Muted     string `mapstructure:"muted"`     // Help text
	Dimmed    string `mapstructure:"dimmed"`    // Hints, suggestions
	Output    string `mapstructure:"output"`    // Calculation results
	Bright    string `mapstructure:"bright"`    // Syntax emphasis
	Separator string `mapstructure:"separator"` // Divider lines

	// Pane backgrounds
	SourcePaneBg  string `mapstructure:"source_pane_bg"`  // Background color for source pane
	PreviewPaneBg string `mapstructure:"preview_pane_bg"` // Background color for preview pane
	StatusBarBg   string `mapstructure:"status_bar_bg"`   // Background color for status bar
}

// ThemeConfigKnownKeys returns the set of known keys under [tui.theme].
// Used to detect deprecated/unknown keys in user config and log warnings.
func ThemeConfigKnownKeys() map[string]bool {
	return map[string]bool{
		"primary":         true,
		"accent":          true,
		"error":           true,
		"warning":         true,
		"muted":           true,
		"dimmed":          true,
		"output":          true,
		"bright":          true,
		"separator":       true,
		"source_pane_bg":  true,
		"preview_pane_bg": true,
		"status_bar_bg":   true,
	}
}

// FormatterConfig holds output formatter settings.
type FormatterConfig struct {
	Verbose       bool   `mapstructure:"verbose"`
	IncludeErrors bool   `mapstructure:"include_errors"`
	DefaultFormat string `mapstructure:"default_format"`
}
