// Package config provides configuration management for the CalcMark TUI/CLI.
// Configuration is loaded from TOML files with embedded defaults.
package config

// Config is the root configuration structure.
type Config struct {
	TUI       TUIConfig       `mapstructure:"tui"`
	Formatter FormatterConfig `mapstructure:"formatter"`
}

// TUIConfig holds TUI-specific settings.
type TUIConfig struct {
	Theme    ThemeConfig `mapstructure:"theme"`
	DarkMode bool        `mapstructure:"dark_mode"`
}

// ThemeConfig defines all TUI colors as hex strings.
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

	// Markdown preview colors
	MdText    string `mapstructure:"md_text"`    // Markdown body text
	MdH1Bg    string `mapstructure:"md_h1_bg"`   // H1 background
	MdH2Bg    string `mapstructure:"md_h2_bg"`   // H2 background
	MdHeading string `mapstructure:"md_heading"` // H3+ and heading text
	MdLink    string `mapstructure:"md_link"`    // Links
	MdQuote   string `mapstructure:"md_quote"`   // Block quote indicator
	MdCode    string `mapstructure:"md_code"`    // Code text
	MdCodeBg  string `mapstructure:"md_code_bg"` // Code background
}

// FormatterConfig holds output formatter settings.
type FormatterConfig struct {
	Verbose       bool   `mapstructure:"verbose"`
	IncludeErrors bool   `mapstructure:"include_errors"`
	DefaultFormat string `mapstructure:"default_format"`
}
