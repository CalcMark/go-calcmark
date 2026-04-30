// Package config provides configuration management for the CalcMark TUI/CLI.
// Configuration is loaded from TOML files with embedded defaults.
package config

// Config is the root configuration structure.
type Config struct {
	// Locale is the BCP 47 locale tag for display formatting (e.g., "en-US", "de-DE").
	// Affects decimal and thousand separators in output. Top-level because locale
	// is application-wide and will eventually affect input parsing too.
	// Precedence: --locale flag > config.toml > "en-US" default.
	Locale      string            `mapstructure:"locale" toml:"locale"`
	TUI         TUIConfig         `mapstructure:"tui" toml:"tui"`
	Formatter   FormatterConfig   `mapstructure:"formatter" toml:"formatter"`
	Measurement MeasurementConfig `mapstructure:"measurement" toml:"measurement"`
}

// TUIConfig holds TUI-specific settings.
type TUIConfig struct {
	Theme            ThemeConfig `mapstructure:"theme" toml:"theme"`
	DarkMode         bool        `mapstructure:"dark_mode" toml:"-"` // Deprecated: use ColorMode instead
	ColorMode        string      `mapstructure:"color_mode" toml:"color_mode"`
	UnicodeFractions *bool       `mapstructure:"unicode_fractions" toml:"unicode_fractions"` // nil = default (true)
}

// ThemeConfig defines user-facing color overrides as hex strings.
// These override the corresponding AdaptiveColor palette slot (light or dark)
// based on the configured color_mode. Internal structural colors (popup borders,
// separator shades, etc.) are not user-configurable and derive from the palette.
type ThemeConfig struct {
	Primary   string `mapstructure:"primary" toml:"primary"`     // Titles, prompts, variable names
	Accent    string `mapstructure:"accent" toml:"accent"`       // Borders, highlights
	Error     string `mapstructure:"error" toml:"error"`         // Error messages
	Warning   string `mapstructure:"warning" toml:"warning"`     // Changed indicators
	Muted     string `mapstructure:"muted" toml:"muted"`         // Help text
	Dimmed    string `mapstructure:"dimmed" toml:"dimmed"`       // Hints, suggestions
	Output    string `mapstructure:"output" toml:"output"`       // Calculation results
	Bright    string `mapstructure:"bright" toml:"bright"`       // Syntax emphasis
	Separator string `mapstructure:"separator" toml:"separator"` // Divider lines

	// Pane backgrounds
	SourcePaneBg  string `mapstructure:"source_pane_bg" toml:"source_pane_bg"`   // Background color for source pane
	PreviewPaneBg string `mapstructure:"preview_pane_bg" toml:"preview_pane_bg"` // Background color for preview pane
	StatusBarBg   string `mapstructure:"status_bar_bg" toml:"status_bar_bg"`     // Background color for status bar
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
	Verbose       bool   `mapstructure:"verbose" toml:"verbose"`
	IncludeErrors bool   `mapstructure:"include_errors" toml:"include_errors"`
	DefaultFormat string `mapstructure:"default_format" toml:"default_format"`

	// DateFormat is a user DSL string overriding the locale default
	// for Date display (e.g., "MON dd, YYYY", "YYYY-MM-dd").
	// See format/display/date_format_dsl.go for supported tokens.
	// Empty (default) uses the locale's date layout.
	DateFormat string `mapstructure:"date_format" toml:"date_format"`

	// PeriodDateFormat is the user DSL for date endpoints inside
	// Period output. Empty falls back to DateFormat, then to the
	// built-in compact "dd-MON-YYYY". Set this independently when
	// the verbose DateFormat would make twin-date output unreadable.
	PeriodDateFormat string `mapstructure:"period_date_format" toml:"period_date_format"`
}

// MeasurementConfig holds default measurement conventions.
// These are global defaults that can be overridden per-document via frontmatter.
// Precedence: frontmatter > config.toml > built-in defaults (US Customary).
type MeasurementConfig struct {
	// Volume: "us" (default) or "imperial".
	// Controls how bare volume names (gallon, pint, cup, fl oz) are interpreted.
	Volume string `mapstructure:"volume" toml:"volume"`

	// Mass: "standard" (default) or "troy".
	// "standard" = avoirdupois (everyday weight: 1 oz = 28.35g).
	// "troy" = precious metals (1 troy oz = 31.10g).
	Mass string `mapstructure:"mass" toml:"mass"`

	// Ton: "short" (default), "long", or "metric".
	// "short" = US (2000 lb), "long" = Imperial (2240 lb), "metric" = 1000 kg.
	Ton string `mapstructure:"ton" toml:"ton"`

	// Strict: annotate bare ambiguous units in output (default true).
	// When true, "oz" displays as "us oz" or "troy oz" depending on convention.
	Strict bool `mapstructure:"strict" toml:"strict"`
}
