---
title: "design: TUI color theming specification with adaptive light/dark palette"
type: design
status: completed
date: 2026-02-22
---

# CalcMark TUI Color Theming Specification

**Version:** 1.0.0  
**Status:** Design Complete — Ready for Implementation  
**Target:** `cm` CLI tool (REPL and Editor modes)

---

## Overview

This document specifies how CalcMark's terminal UI handles color theming to ensure readability across light and dark terminal backgrounds.

### Design Goals

1. **Works out of the box** — Auto-detect terminal background and adapt
2. **User override** — Escape hatch when detection fails
3. **Semantic colors** — Named by purpose, not by hue
4. **Minimal palette** — Constraint breeds consistency

### Non-Goals

- Multiple named themes (e.g., "Solarized", "Dracula")
- Per-element color customization
- True color requirements (must work on 256-color terminals)

---

## Approach
```
┌─────────────────────────────────────────────────────────────────────┐
│                        Color Resolution Flow                        │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│   1. Check config/flag    ──►  User specified "light" or "dark"?   │
│          │                              │                           │
│          │ no                          yes                          │
│          ▼                              │                           │
│   2. Query terminal       ──►  OSC 11 response received?           │
│          │                              │                           │
│          │ no                          yes                          │
│          ▼                              ▼                           │
│   3. Assume dark          ◄──  Use detected/specified mode         │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

**Why assume dark?** ~80% of developer terminals use dark themes. When detection fails, this is the safer bet.

---

## Semantic Color Palette

Colors are named by their **purpose**, not their appearance. Each semantic color has light and dark variants.

### Core Palette

| Semantic Name | Purpose                              | Light Mode | Dark Mode |
|---------------|--------------------------------------|------------|-----------|
| `Text`        | Primary text, prose                  | `#1a1a1a`  | `#e5e5e5` |
| `TextMuted`   | Secondary text, operators            | `#666666`  | `#888888` |
| `Result`      | Computed values in preview           | `#0969da`  | `#58a6ff` |
| `ResultMuted` | Variable names in preview            | `#57606a`  | `#7d8590` |
| `Error`       | Warnings, errors                     | `#b35900`  | `#f0a020` |
| `Header`      | Markdown headers                     | `#1a1a1a`  | `#ffffff` |
| `Cursor`      | Cursor line highlight                | `#f6f8fa`  | `#262626` |
| `Selection`   | Selected text background             | `#ddf4ff`  | `#1f3a5f` |

### Accent Colors (Sparingly Used)

| Semantic Name | Purpose                       | Light Mode | Dark Mode |
|---------------|-------------------------------|------------|-----------|
| `Hint`        | Autosuggestions, help text    | `#6e7781`  | `#6e7781` |
| `Command`     | REPL commands                 | `#8250df`  | `#a371f7` |
| `Success`     | Confirmations (save complete) | `#1a7f37`  | `#3fb950` |

### Border & Background

| Semantic Name | Purpose                       | Light Mode | Dark Mode |
|---------------|-------------------------------|------------|-----------|
| `Border`      | Pane borders                  | `#d1d5db`  | `#3d3d3d` |
| `PaneBg`      | Pane background (if distinct) | `#ffffff`  | `#1a1a1a` |
| `StatusBg`    | Status bar background         | `#f3f4f6`  | `#262626` |
| `StatusFg`    | Status bar text               | `#374151`  | `#d1d5d9` |

---

## Configuration

### Hierarchy (Highest to Lowest Priority)
```
1. CLI flag:        --color-mode=light
2. Env variable:    CM_COLOR_MODE=light
3. Config file:     color_mode: light
4. Auto-detection:  lipgloss.HasDarkBackground()
5. Default:         dark
```

### Viper Integration

Config file locations (standard Viper precedence):
- `$XDG_CONFIG_HOME/calcmark/config.yaml`
- `~/.config/calcmark/config.yaml`
- `~/.calcmark.yaml`
- `./calcmark.yaml` (for project-local override)
```yaml
# ~/.config/calcmark/config.yaml

# Color mode: "auto", "light", or "dark"
# Default: "auto" (detect from terminal)
color_mode: auto
```

### CLI Flag
```bash
cm --color-mode=light          # Force light mode
cm --color-mode=dark           # Force dark mode  
cm --color-mode=auto           # Explicit auto-detect (default)
cm edit budget.cm --color-mode=light
```

### Environment Variable
```bash
export CM_COLOR_MODE=light
cm edit budget.cm
```

---

## Implementation

### File Structure
```
cmd/cm/
├── config/
│   └── config.go          # Viper setup, color mode resolution
├── theme/
│   ├── theme.go           # Semantic color definitions
│   ├── palette.go         # AdaptiveColor values
│   └── styles.go          # Lipgloss styles using palette
└── main.go                # Flag registration
```

### config/config.go
```go
package config

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/viper"
)

type ColorMode string

const (
	ColorModeAuto  ColorMode = "auto"
	ColorModeLight ColorMode = "light"
	ColorModeDark  ColorMode = "dark"
)

// Config holds all application configuration
type Config struct {
	ColorMode ColorMode `mapstructure:"color_mode"`
}

var cfg Config

func Init() error {
	// Set defaults
	viper.SetDefault("color_mode", "auto")

	// Config file search paths
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("$XDG_CONFIG_HOME/calcmark")
	viper.AddConfigPath("$HOME/.config/calcmark")
	viper.AddConfigPath("$HOME")
	viper.AddConfigPath(".")

	// Environment variables: CM_COLOR_MODE, etc.
	viper.SetEnvPrefix("CM")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()

	// Read config file (ignore "not found" errors)
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return err
		}
	}

	// Unmarshal into struct
	if err := viper.Unmarshal(&cfg); err != nil {
		return err
	}

	// Apply color mode to lipgloss
	applyColorMode(cfg.ColorMode)

	return nil
}

func applyColorMode(mode ColorMode) {
	switch mode {
	case ColorModeLight:
		lipgloss.SetHasDarkBackground(false)
	case ColorModeDark:
		lipgloss.SetHasDarkBackground(true)
	case ColorModeAuto:
		// lipgloss auto-detects; we just don't override
		// Detection happens lazily on first AdaptiveColor use
	}
}

// Get returns the current configuration
func Get() Config {
	return cfg
}

// IsDarkMode returns whether we're rendering for dark background
func IsDarkMode() bool {
	return lipgloss.HasDarkBackground()
}
```

### theme/palette.go
```go
package theme

import "github.com/charmbracelet/lipgloss"

// Semantic color palette with light/dark variants
var (
	// Core text colors
	Text = lipgloss.AdaptiveColor{
		Light: "#1a1a1a",
		Dark:  "#e5e5e5",
	}

	TextMuted = lipgloss.AdaptiveColor{
		Light: "#666666",
		Dark:  "#888888",
	}

	// Results and values
	Result = lipgloss.AdaptiveColor{
		Light: "#0969da",
		Dark:  "#58a6ff",
	}

	ResultMuted = lipgloss.AdaptiveColor{
		Light: "#57606a",
		Dark:  "#7d8590",
	}

	// Errors and warnings (amber, not red)
	Error = lipgloss.AdaptiveColor{
		Light: "#b35900",
		Dark:  "#f0a020",
	}

	// Headers
	Header = lipgloss.AdaptiveColor{
		Light: "#1a1a1a",
		Dark:  "#ffffff",
	}

	// UI elements
	Cursor = lipgloss.AdaptiveColor{
		Light: "#f6f8fa",
		Dark:  "#262626",
	}

	Selection = lipgloss.AdaptiveColor{
		Light: "#ddf4ff",
		Dark:  "#1f3a5f",
	}

	// Hints and help
	Hint = lipgloss.AdaptiveColor{
		Light: "#6e7781",
		Dark:  "#6e7781",
	}

	// Commands
	Command = lipgloss.AdaptiveColor{
		Light: "#8250df",
		Dark:  "#a371f7",
	}

	// Success states
	Success = lipgloss.AdaptiveColor{
		Light: "#1a7f37",
		Dark:  "#3fb950",
	}

	// Borders and backgrounds
	Border = lipgloss.AdaptiveColor{
		Light: "#d1d5db",
		Dark:  "#3d3d3d",
	}

	PaneBg = lipgloss.AdaptiveColor{
		Light: "#ffffff",
		Dark:  "#1a1a1a",
	}

	StatusBg = lipgloss.AdaptiveColor{
		Light: "#f3f4f6",
		Dark:  "#262626",
	}

	StatusFg = lipgloss.AdaptiveColor{
		Light: "#374151",
		Dark:  "#d1d5d9",
	}
)
```

### theme/styles.go
```go
package theme

import "github.com/charmbracelet/lipgloss"

// Pre-built styles using the semantic palette
var (
	// Text styles
	TextStyle = lipgloss.NewStyle().
		Foreground(Text)

	MutedStyle = lipgloss.NewStyle().
		Foreground(TextMuted)

	HeaderStyle = lipgloss.NewStyle().
		Foreground(Header).
		Bold(true)

	// Result styles (preview pane)
	ResultStyle = lipgloss.NewStyle().
		Foreground(Result)

	ResultNameStyle = lipgloss.NewStyle().
		Foreground(ResultMuted)

	// Error style
	ErrorStyle = lipgloss.NewStyle().
		Foreground(Error)

	// Hint style (autosuggestions)
	HintStyle = lipgloss.NewStyle().
		Foreground(Hint)

	// Command style (REPL commands)
	CommandStyle = lipgloss.NewStyle().
		Foreground(Command)

	// Success style
	SuccessStyle = lipgloss.NewStyle().
		Foreground(Success)

	// Cursor line (subtle highlight)
	CursorLineStyle = lipgloss.NewStyle().
		Background(Cursor)

	// Selection
	SelectionStyle = lipgloss.NewStyle().
		Background(Selection)

	// Borders
	BorderStyle = lipgloss.NewStyle().
		BorderForeground(Border)

	// Status bar
	StatusBarStyle = lipgloss.NewStyle().
		Background(StatusBg).
		Foreground(StatusFg).
		Padding(0, 1)
)

// PaneBorder returns a border style for panes
func PaneBorder() lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(Border)
}
```

### main.go Integration
```go
package main

import (
	"fmt"
	"os"

	"github.com/CalcMark/go-calcmark/cmd/cm/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
	Use:   "cm",
	Short: "CalcMark calculator and document editor",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return config.Init()
	},
}

func init() {
	// Persistent flag available to all subcommands
	rootCmd.PersistentFlags().String("color-mode", "auto",
		"Color mode: 'auto', 'light', or 'dark'")

	// Bind flag to viper
	viper.BindPFlag("color_mode", rootCmd.PersistentFlags().Lookup("color-mode"))
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
```

---

## Usage Examples

### Using Styles in Components
```go
func (p PreviewPane) renderResult(r LineResult) string {
	if r.Error != nil {
		return theme.ErrorStyle.Render("⚠ " + r.Error.Error())
	}

	formatted := formatValue(r.Value)

	if r.VarName != "" {
		return fmt.Sprintf("%s  %s",
			theme.ResultNameStyle.Render(padLeft(r.VarName, 20)),
			theme.ResultStyle.Render(formatted))
	}

	return theme.ResultStyle.Render(formatted)
}
```

---

## Testing Checklist

1. **Dark terminal** (default) — All text readable
2. **Light terminal** — All text readable, no washed-out colors
3. **Force light mode** — `cm --color-mode=light` on dark terminal
4. **Force dark mode** — `cm --color-mode=dark` on light terminal
5. **Config file** — `color_mode: light` respected
6. **Env override** — `CM_COLOR_MODE=dark` overrides config
7. **Flag override** — `--color-mode` overrides env and config

---

## Lipgloss Caveats

- `HasDarkBackground()` queries the terminal once and caches
- Some terminals (older xterm, screen) don't respond to OSC 11
- SSH sessions through jump hosts may not forward the query properly

---

**End of Specification**
