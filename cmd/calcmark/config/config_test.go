package config

import (
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

	// Verify defaults match expected values
	if cfg.TUI.Theme.Primary != "#7D56F4" {
		t.Errorf("expected default primary #7D56F4, got %s", cfg.TUI.Theme.Primary)
	}
	if cfg.TUI.Theme.Error != "#FF5555" {
		t.Errorf("expected default error #FF5555, got %s", cfg.TUI.Theme.Error)
	}
	if cfg.Formatter.DefaultFormat != "text" {
		t.Errorf("expected default format text, got %s", cfg.Formatter.DefaultFormat)
	}
	if !cfg.TUI.DarkMode {
		t.Error("expected dark_mode true by default")
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

	// Other defaults should be preserved
	if cfg.TUI.Theme.Error != "#FF5555" {
		t.Errorf("expected default error preserved, got %s", cfg.TUI.Theme.Error)
	}
	if cfg.TUI.Theme.Accent != "#874BFD" {
		t.Errorf("expected default accent preserved, got %s", cfg.TUI.Theme.Accent)
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
