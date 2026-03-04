package config

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestValidateHexColor(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"", true},
		{"#FFF", true},
		{"#fff", true},
		{"#FFFFFF", true},
		{"#ffffff", true},
		{"#AbCdEf", true},
		{"#123", true},
		{"#1A2B3C", true},
		// Invalid
		{"red", false},
		{"FFF", false},
		{"#FF", false},
		{"#FFFF", false},
		{"#FFFFF", false},
		{"#FFFFFFF", false},
		{"#GGG", false},
		{"#12345Z", false},
		{"#", false},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := ValidateHexColor(tt.input); got != tt.want {
				t.Errorf("ValidateHexColor(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestValidateColorMode(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"", true},
		{"light", true},
		{"dark", true},
		{"auto", true},
		{"Light", true},
		{"DARK", true},
		// Invalid
		{"system", false},
		{"night", false},
		{"on", false},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := ValidateColorMode(tt.input); got != tt.want {
				t.Errorf("ValidateColorMode(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestValidateFormat(t *testing.T) {
	validNames := []string{"cm", "html", "json", "md", "text"}
	tests := []struct {
		input string
		want  bool
	}{
		{"", true},
		{"text", true},
		{"json", true},
		{"html", true},
		{"md", true},
		{"cm", true},
		// Invalid
		{"xml", false},
		{"yaml", false},
		{"TEXT", false},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := ValidateFormat(tt.input, validNames); got != tt.want {
				t.Errorf("ValidateFormat(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestValidate_ValidConfig(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("XDG_CONFIG_HOME", "")

	configDir := filepath.Join(tmpHome, ".config", "calcmark")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.toml"), []byte(`
locale = "de-DE"

[tui]
color_mode = "light"

[tui.theme]
primary = "#5B21B6"

[formatter]
default_format = "json"
`), 0644); err != nil {
		t.Fatal(err)
	}

	result := Validate([]string{"cm", "html", "json", "md", "text"})
	if !result.OK() {
		t.Errorf("expected no errors, got: %v", result.Errors)
	}
}

func TestValidate_TOMLSyntaxError(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("XDG_CONFIG_HOME", "")

	configDir := filepath.Join(tmpHome, ".config", "calcmark")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.toml"), []byte("primary = [\n"), 0644); err != nil {
		t.Fatal(err)
	}

	result := Validate([]string{"text"})
	if result.OK() {
		t.Fatal("expected errors for TOML syntax error")
	}
	found := false
	for _, e := range result.Errors {
		if strings.Contains(e, "TOML syntax error") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected TOML syntax error, got: %v", result.Errors)
	}
}

func TestValidate_InvalidHexColor(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("XDG_CONFIG_HOME", "")

	configDir := filepath.Join(tmpHome, ".config", "calcmark")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.toml"), []byte(`
[tui.theme]
primary = "red"
`), 0644); err != nil {
		t.Fatal(err)
	}

	result := Validate([]string{"text"})
	if result.OK() {
		t.Fatal("expected errors for invalid hex color")
	}
	found := false
	for _, e := range result.Errors {
		if strings.Contains(e, "primary") && strings.Contains(e, "invalid hex color") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected invalid hex color error for primary, got: %v", result.Errors)
	}
}

func TestValidate_InvalidColorMode(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("XDG_CONFIG_HOME", "")

	configDir := filepath.Join(tmpHome, ".config", "calcmark")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.toml"), []byte(`
[tui]
color_mode = "system"
`), 0644); err != nil {
		t.Fatal(err)
	}

	result := Validate([]string{"text"})
	if result.OK() {
		t.Fatal("expected errors for invalid color_mode")
	}
	found := false
	for _, e := range result.Errors {
		if strings.Contains(e, "color_mode") && strings.Contains(e, "system") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected invalid color_mode error, got: %v", result.Errors)
	}
}

func TestValidate_InvalidDefaultFormat(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("XDG_CONFIG_HOME", "")

	configDir := filepath.Join(tmpHome, ".config", "calcmark")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.toml"), []byte(`
[formatter]
default_format = "xml"
`), 0644); err != nil {
		t.Fatal(err)
	}

	result := Validate([]string{"cm", "html", "json", "md", "text"})
	if result.OK() {
		t.Fatal("expected errors for invalid default_format")
	}
	found := false
	for _, e := range result.Errors {
		if strings.Contains(e, "default_format") && strings.Contains(e, "xml") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected invalid default_format error, got: %v", result.Errors)
	}
}

func TestValidate_UnknownKey(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("XDG_CONFIG_HOME", "")

	configDir := filepath.Join(tmpHome, ".config", "calcmark")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.toml"), []byte(`
magic = true
`), 0644); err != nil {
		t.Fatal(err)
	}

	result := Validate([]string{"text"})
	if result.OK() {
		t.Fatal("expected errors for unknown key")
	}
	found := false
	for _, e := range result.Errors {
		if strings.Contains(e, "unknown config key") && strings.Contains(e, "magic") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected unknown key error, got: %v", result.Errors)
	}
}

func TestValidate_NoConfigFiles(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("XDG_CONFIG_HOME", "")

	result := Validate([]string{"text"})
	if !result.OK() {
		t.Errorf("expected no errors when no config files exist, got: %v", result.Errors)
	}
	// Should still report file discovery
	if len(result.Files) == 0 {
		t.Error("expected file discovery results even with no config files")
	}
}

func TestValidate_MultipleErrors(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("XDG_CONFIG_HOME", "")

	configDir := filepath.Join(tmpHome, ".config", "calcmark")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.toml"), []byte(`
[tui]
color_mode = "system"

[tui.theme]
primary = "red"

[formatter]
default_format = "xml"
`), 0644); err != nil {
		t.Fatal(err)
	}

	result := Validate([]string{"cm", "html", "json", "md", "text"})
	if result.OK() {
		t.Fatal("expected multiple errors")
	}
	if len(result.Errors) < 3 {
		t.Errorf("expected at least 3 errors (color_mode, primary, default_format), got %d: %v",
			len(result.Errors), result.Errors)
	}
}

// TestKnownKeysCompleteness ensures knownKeys stays in sync with the Config
// struct's mapstructure tags.
func TestKnownKeysCompleteness(t *testing.T) {
	themeKnown := ThemeConfigKnownKeys()

	// Walk the Config struct and collect all mapstructure tag paths
	expected := collectMapstructureKeys(reflect.TypeFor[Config](), "")

	for _, key := range expected {
		if fieldName, ok := strings.CutPrefix(key, "tui.theme."); ok {
			if !themeKnown[fieldName] {
				t.Errorf("Config struct has mapstructure key %q but ThemeConfigKnownKeys() is missing %q", key, fieldName)
			}
			continue
		}
		if !knownKeys[key] {
			t.Errorf("Config struct has mapstructure key %q but knownKeys is missing it", key)
		}
	}
}

// collectMapstructureKeys recursively collects dotted key paths from mapstructure tags.
func collectMapstructureKeys(t reflect.Type, prefix string) []string {
	var keys []string
	for i := range t.NumField() {
		field := t.Field(i)
		tag := field.Tag.Get("mapstructure")
		if tag == "" || tag == "-" {
			continue
		}
		fullKey := tag
		if prefix != "" {
			fullKey = prefix + "." + tag
		}
		if field.Type.Kind() == reflect.Struct {
			keys = append(keys, collectMapstructureKeys(field.Type, fullKey)...)
		} else {
			keys = append(keys, fullKey)
		}
	}
	return keys
}
