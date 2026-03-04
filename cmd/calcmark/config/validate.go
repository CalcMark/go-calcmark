package config

import (
	"fmt"
	"os"
	"slices"
	"strings"

	toml "github.com/pelletier/go-toml/v2"
	"github.com/spf13/viper"
)

// ConfigFileResult describes the discovery status of a single config file path.
type ConfigFileResult struct {
	Path  string
	Found bool
	Err   error // TOML syntax error, if any
}

// ValidationResult collects all config file results and validation errors.
type ValidationResult struct {
	Files  []ConfigFileResult
	Errors []string
}

// OK returns true when no validation errors were found.
func (r *ValidationResult) OK() bool {
	return len(r.Errors) == 0
}

// knownKeys is the set of top-level and dotted config keys that the Config
// struct understands (excluding tui.theme.* which has its own known set).
var knownKeys = map[string]bool{
	"locale":                   true,
	"tui.color_mode":           true,
	"tui.dark_mode":            true,
	"formatter.verbose":        true,
	"formatter.include_errors": true,
	"formatter.default_format": true,
}

// ValidateHexColor reports whether s is a valid hex color string.
// Empty string is accepted (means "use palette default").
// Valid formats: #RGB, #RRGGBB (case-insensitive).
func ValidateHexColor(s string) bool {
	if s == "" {
		return true
	}
	if s[0] != '#' {
		return false
	}
	hexPart := s[1:]
	if len(hexPart) != 3 && len(hexPart) != 6 {
		return false
	}
	for i := range len(hexPart) {
		if !isHexDigit(hexPart[i]) {
			return false
		}
	}
	return true
}

func isHexDigit(b byte) bool {
	return (b >= '0' && b <= '9') || (b >= 'a' && b <= 'f') || (b >= 'A' && b <= 'F')
}

// ValidateColorMode reports whether s is a valid color_mode value.
// Empty string is accepted (means "use default").
func ValidateColorMode(s string) bool {
	switch strings.ToLower(s) {
	case "", "light", "dark", "auto":
		return true
	}
	return false
}

// ValidateFormat reports whether s is a valid output format name.
// Empty string is accepted (means "use default").
// validNames is the list of registered format names (passed in to avoid import cycles).
func ValidateFormat(s string, validNames []string) bool {
	if s == "" {
		return true
	}
	return slices.Contains(validNames, s)
}

// Validate discovers config files, checks TOML syntax, loads effective config,
// and validates all field values. validFormatNames is the list of registered
// format names (from format.FormatNames()).
func Validate(validFormatNames []string) *ValidationResult {
	result := &ValidationResult{}

	// 1. Discover config files and probe TOML syntax
	type configFile struct {
		path    string
		pathErr error
	}
	candidates := make([]configFile, 0, 2)

	fallbackPath, fallbackErr := fallbackConfigPath()
	candidates = append(candidates, configFile{fallbackPath, fallbackErr})

	xdgPath, xdgErr := XDGConfigPath()
	candidates = append(candidates, configFile{xdgPath, xdgErr})

	for _, c := range candidates {
		if c.pathErr != nil {
			continue
		}
		fr := ConfigFileResult{Path: c.path}
		data, err := os.ReadFile(c.path)
		if err != nil {
			// File doesn't exist — not an error
			result.Files = append(result.Files, fr)
			continue
		}
		fr.Found = true

		// Probe TOML syntax with go-toml/v2
		var probe map[string]any
		if err := toml.Unmarshal(data, &probe); err != nil {
			fr.Err = err
			result.Errors = append(result.Errors, fmt.Sprintf("%s: TOML syntax error: %v", c.path, err))
		}
		result.Files = append(result.Files, fr)
	}

	// If any file had TOML syntax errors, skip field validation
	// (viper may have loaded partial/corrupt data)
	for _, f := range result.Files {
		if f.Err != nil {
			return result
		}
	}

	// 2. Load effective config via a fresh local viper instance
	c, v, err := loadForValidation()
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("load config: %v", err))
		return result
	}

	// 3. Validate fields
	// Hex colors in theme
	themeColors := map[string]string{
		"primary":         c.TUI.Theme.Primary,
		"accent":          c.TUI.Theme.Accent,
		"error":           c.TUI.Theme.Error,
		"warning":         c.TUI.Theme.Warning,
		"muted":           c.TUI.Theme.Muted,
		"dimmed":          c.TUI.Theme.Dimmed,
		"output":          c.TUI.Theme.Output,
		"bright":          c.TUI.Theme.Bright,
		"separator":       c.TUI.Theme.Separator,
		"source_pane_bg":  c.TUI.Theme.SourcePaneBg,
		"preview_pane_bg": c.TUI.Theme.PreviewPaneBg,
		"status_bar_bg":   c.TUI.Theme.StatusBarBg,
	}
	for name, val := range themeColors {
		if !ValidateHexColor(val) {
			result.Errors = append(result.Errors, fmt.Sprintf("[tui.theme].%s: invalid hex color %q (expected #RGB or #RRGGBB)", name, val))
		}
	}

	// Color mode
	if !ValidateColorMode(c.TUI.ColorMode) {
		result.Errors = append(result.Errors, fmt.Sprintf("[tui].color_mode: invalid value %q (expected light, dark, or auto)", c.TUI.ColorMode))
	}

	// Default format
	if !ValidateFormat(c.Formatter.DefaultFormat, validFormatNames) {
		result.Errors = append(result.Errors, fmt.Sprintf("[formatter].default_format: invalid value %q (expected one of: %s)", c.Formatter.DefaultFormat, strings.Join(validFormatNames, ", ")))
	}

	// 4. Check for unknown keys
	themeKnown := ThemeConfigKnownKeys()
	for _, key := range v.AllKeys() {
		if fieldName, ok := strings.CutPrefix(key, "tui.theme."); ok {
			if !themeKnown[fieldName] {
				result.Errors = append(result.Errors, fmt.Sprintf("unknown config key: tui.theme.%s", fieldName))
			}
			continue
		}
		if !knownKeys[key] {
			result.Errors = append(result.Errors, fmt.Sprintf("unknown config key: %s", key))
		}
	}

	slices.Sort(result.Errors)
	return result
}

// loadForValidation creates a fresh viper instance and loads config files,
// returning the parsed Config and the viper instance (for key inspection).
func loadForValidation() (*Config, *viper.Viper, error) {
	v := viper.New()
	v.SetConfigType("toml")

	if err := v.ReadConfig(strings.NewReader(defaultsToml)); err != nil {
		return nil, nil, fmt.Errorf("read embedded defaults: %w", err)
	}

	if fallbackPath, err := fallbackConfigPath(); err == nil {
		if _, statErr := os.Stat(fallbackPath); statErr == nil {
			v.SetConfigFile(fallbackPath)
			if mergeErr := v.MergeInConfig(); mergeErr != nil {
				return nil, nil, fmt.Errorf("merge %s: %w", fallbackPath, mergeErr)
			}
		}
	}
	if xdgPath, err := XDGConfigPath(); err == nil {
		if _, statErr := os.Stat(xdgPath); statErr == nil {
			v.SetConfigFile(xdgPath)
			if mergeErr := v.MergeInConfig(); mergeErr != nil {
				return nil, nil, fmt.Errorf("merge %s: %w", xdgPath, mergeErr)
			}
		}
	}

	var c Config
	if err := v.Unmarshal(&c); err != nil {
		return nil, nil, fmt.Errorf("unmarshal: %w", err)
	}
	return &c, v, nil
}
