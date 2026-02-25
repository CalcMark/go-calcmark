package config

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_DefaultsOnly(t *testing.T) {
	// Reset state and load fresh
	cfg, err := Reload()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	// Theme config defaults are empty strings (palette provides actual defaults)
	if cfg.TUI.Theme.Primary != "" {
		t.Errorf("expected default primary empty (palette default), got %s", cfg.TUI.Theme.Primary)
	}
	if cfg.TUI.Theme.Error != "" {
		t.Errorf("expected default error empty (palette default), got %s", cfg.TUI.Theme.Error)
	}
	if cfg.Formatter.DefaultFormat != "text" {
		t.Errorf("expected default format text, got %s", cfg.Formatter.DefaultFormat)
	}
	if cfg.TUI.DarkMode {
		t.Error("expected dark_mode false by default (deprecated, removed from defaults)")
	}
	if cfg.TUI.ColorMode != "dark" {
		t.Errorf("expected default color_mode dark, got %s", cfg.TUI.ColorMode)
	}
}

func TestLoad_UserConfigMerge(t *testing.T) {
	// Create temp home directory
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	// Create XDG config directory
	configDir := filepath.Join(tmpHome, ".config", "calcmark")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}

	// Write partial user config
	userConfig := `[tui.theme]
primary = "#ABCDEF"
`
	configPath := filepath.Join(configDir, "config.toml")
	if err := os.WriteFile(configPath, []byte(userConfig), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	// Reload and verify merge
	cfg, err := Reload()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	// User override should be applied
	if cfg.TUI.Theme.Primary != "#ABCDEF" {
		t.Errorf("expected user override #ABCDEF, got %s", cfg.TUI.Theme.Primary)
	}

	// Other defaults should remain empty (palette defaults)
	if cfg.TUI.Theme.Error != "" {
		t.Errorf("expected default error empty (palette default), got %s", cfg.TUI.Theme.Error)
	}
	if cfg.TUI.Theme.Accent != "" {
		t.Errorf("expected default accent empty (palette default), got %s", cfg.TUI.Theme.Accent)
	}
}

func TestLoad_FallbackConfig(t *testing.T) {
	// Create temp home directory
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	// Create fallback config (no XDG directory)
	fallbackConfig := `[tui.theme]
warning = "#00FF00"
`
	fallbackPath := filepath.Join(tmpHome, ".calcmarkrc.toml")
	if err := os.WriteFile(fallbackPath, []byte(fallbackConfig), 0644); err != nil {
		t.Fatalf("failed to write fallback config: %v", err)
	}

	cfg, err := Reload()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	// Fallback should be applied
	if cfg.TUI.Theme.Warning != "#00FF00" {
		t.Errorf("expected fallback override #00FF00, got %s", cfg.TUI.Theme.Warning)
	}
}

func TestLoad_XDGPriorityOverFallback(t *testing.T) {
	// Create temp home directory
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	// Create fallback with red
	fallbackConfig := `[tui.theme]
primary = "#FF0000"
`
	fallbackPath := filepath.Join(tmpHome, ".calcmarkrc.toml")
	if err := os.WriteFile(fallbackPath, []byte(fallbackConfig), 0644); err != nil {
		t.Fatalf("failed to write fallback: %v", err)
	}

	// Create XDG with green (should win)
	configDir := filepath.Join(tmpHome, ".config", "calcmark")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}
	xdgConfig := `[tui.theme]
primary = "#00FF00"
`
	xdgPath := filepath.Join(configDir, "config.toml")
	if err := os.WriteFile(xdgPath, []byte(xdgConfig), 0644); err != nil {
		t.Fatalf("failed to write XDG config: %v", err)
	}

	cfg, err := Reload()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	// XDG should win
	if cfg.TUI.Theme.Primary != "#00FF00" {
		t.Errorf("expected XDG priority #00FF00, got %s", cfg.TUI.Theme.Primary)
	}
}

func TestLoad_DeprecatedKeysWarning(t *testing.T) {
	// Create temp home directory
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	// Create config with deprecated keys
	configDir := filepath.Join(tmpHome, ".config", "calcmark")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}

	userConfig := `[tui.theme]
primary = "#ABCDEF"
edit_line_bg = "#2E2E2E"
cursor_bg = "#7D56F4"
`
	configPath := filepath.Join(configDir, "config.toml")
	if err := os.WriteFile(configPath, []byte(userConfig), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	// Load should succeed (deprecated keys don't cause errors)
	_, err := Reload()
	if err != nil {
		t.Fatalf("Load() should not error on deprecated keys: %v", err)
	}
	// The warning is printed to stderr, which is expected behavior
}

func TestBuildStyles(t *testing.T) {
	theme := ThemeConfig{
		Primary:   "#111111",
		Accent:    "#222222",
		Error:     "#333333",
		Warning:   "#444444",
		Muted:     "#555555",
		Dimmed:    "#666666",
		Output:    "#777777",
		Bright:    "#888888",
		Separator: "#999999",
	}

	styles := theme.BuildStyles()

	// Verify styles render without panic
	result := styles.Title.Render("test")
	if result == "" {
		t.Error("expected non-empty rendered output")
	}

	// Test all style fields are populated
	_ = styles.Error.Render("error")
	_ = styles.Prompt.Render("prompt")
	_ = styles.Output.Render("output")
	_ = styles.Changed.Render("changed")
	_ = styles.Var.Render("var")
	_ = styles.Hint.Render("hint")
}

func TestBuildStyles_EmptyOverrides(t *testing.T) {
	// Empty ThemeConfig should use palette defaults
	theme := ThemeConfig{}
	styles := theme.BuildStyles()

	// Verify styles render without panic even with empty config
	result := styles.Title.Render("test")
	if result == "" {
		t.Error("expected non-empty rendered output with palette defaults")
	}

	_ = styles.SourcePane.Render("source")
	_ = styles.PreviewPane.Render("preview")
	_ = styles.SourceFrontmatter.Render("frontmatter")
	_ = styles.SourceCalc.Render("calc")
}

// TestThemeOverride_EndToEnd verifies the full path from TOML config
// through Viper → ThemeConfig → BuildStyles → lipgloss style color.
// This catches any disconnect between the config file format and the
// actual rendered styles.
func TestThemeOverride_EndToEnd(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	configDir := filepath.Join(tmpHome, ".config", "calcmark")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}

	// Write a TOML config that overrides primary color
	userConfig := `[tui.theme]
primary = "#FF00FF"
`
	configPath := filepath.Join(configDir, "config.toml")
	if err := os.WriteFile(configPath, []byte(userConfig), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	// Load config (TOML → Viper → ThemeConfig)
	cfg, err := Reload()
	if err != nil {
		t.Fatalf("Reload() error: %v", err)
	}

	// Verify ThemeConfig has the override
	if cfg.TUI.Theme.Primary != "#FF00FF" {
		t.Fatalf("ThemeConfig.Primary = %q, want #FF00FF", cfg.TUI.Theme.Primary)
	}

	// Build styles (ThemeConfig → BuildStyles → lipgloss)
	styles := cfg.TUI.Theme.BuildStyles()

	// Verify the Title style uses the overridden color.
	// lipgloss.Style.GetForeground() returns the TerminalColor set on the style.
	fg := styles.Title.GetForeground()
	if fg == nil {
		t.Fatal("Title style has no foreground color set")
	}

	// overrideColor sets both Light and Dark slots to the user's hex.
	// In lipgloss v2, lipgloss.Color is a color.Color so AdaptiveColor
	// stringifies as "{{R G B A} {R G B A}}" instead of "{#hex #hex}".
	fgStr := fmt.Sprintf("%v", fg)
	if fgStr != "{{255 0 255 255} {255 0 255 255}}" {
		t.Errorf("Title foreground = %v, want {{255 0 255 255} {255 0 255 255}}", fgStr)
	}
}

// TestThemeOverride_PaletteDefault verifies that when no user override is
// provided, styles use the palette default (not empty/zero).
func TestThemeOverride_PaletteDefault(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	// No user config — palette defaults should be used
	cfg, err := Reload()
	if err != nil {
		t.Fatalf("Reload() error: %v", err)
	}

	if cfg.TUI.Theme.Primary != "" {
		t.Fatalf("expected empty primary (palette default), got %q", cfg.TUI.Theme.Primary)
	}

	styles := cfg.TUI.Theme.BuildStyles()

	// Title style should still have a foreground (from palette)
	fg := styles.Title.GetForeground()
	if fg == nil {
		t.Fatal("Title style has no foreground color — palette default not applied")
	}
}

func TestGetStyles_AfterLoad(t *testing.T) {
	_, err := Reload()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	styles := GetStyles()

	// Styles should be usable
	result := styles.Title.Render("CalcMark")
	if result == "" {
		t.Error("expected non-empty styled output")
	}
}
