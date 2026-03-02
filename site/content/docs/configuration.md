---
title: "Configuration"
summary: "Customize CalcMark's locale, appearance, and formatter defaults."
weight: 40
---

CalcMark supports user configuration via TOML files. Configuration controls display locale, color mode, theme colors, pane backgrounds, and formatter defaults.

## Configuration File Locations

CalcMark checks for configuration files in this order (later files override earlier ones):

1. **Embedded defaults** (compiled into the binary)
2. `~/.calcmarkrc.toml` (dotfile fallback)
3. `~/.config/calcmark/config.toml` (XDG standard, recommended)

You only need to specify values you want to override -- unspecified values use the built-in adaptive palette.

## Quick Start

Create a config file:

```bash
mkdir -p ~/.config/calcmark
touch ~/.config/calcmark/config.toml
```

Add your customizations:

```toml
[tui]
color_mode = "light"  # Use "light" if you have a light terminal background
```

## Display Locale {#locale}

The `locale` setting controls how numbers are formatted in output -- specifically the decimal separator and thousands grouping separator.

```toml
# ~/.config/calcmark/config.toml
locale = "de-DE"
```

### Supported Locales

| Locale | Decimal | Thousands | Example: `$1500` | Example: `3.14` |
|--------|---------|-----------|------------------|-----------------|
| `en-US` (default) | `.` | `,` | `$1,500.00` | `3.14` |
| `de-DE` | `,` | `.` | `$1.500,00` | `3,14` |
| `fr-FR` | `,` | (non-breaking space) | `$1 500,00` | `3,14` |

Locale affects **output formatting only** -- input syntax always uses `.` for decimals and `,` or `_` for thousands grouping, regardless of locale.

### What Locale Changes

- **Decimal separator** in numbers, quantities, and currency values
- **Thousands grouping separator** in currency mid-range values (e.g., `$1,500.00`)
- **Decimal within K/M/B suffixes** (e.g., `1.5M` in en-US becomes `1,5M` in de-DE)

### What Locale Does Not Change

- **K/M/B/T suffix letters** are always English, regardless of locale
- **Currency symbol position** stays the same (always prefix, e.g., `$`)
- **Input syntax** -- CalcMark source code always uses `.` for decimals
- **CalcMark format** (`cm convert --to=cm`) stays locale-independent for portability
- **JSON `raw_value`** is always machine-readable ASCII (see [JSON Output](#json-raw-value))

### Precedence

The locale is resolved in this order (first match wins):

1. `--locale` CLI flag
2. `locale` in config file (`config.toml`)
3. `en-US` (default)

### CLI Flag Override

Override the locale for a single invocation:

```bash
cm eval --locale=de-DE budget.cm
cm convert doc.cm --to=json --locale=fr-FR
```

If the locale string is invalid, CalcMark prints a warning to stderr and falls back to `en-US`.

## Color Mode CLI Override

The `--color-mode` flag overrides the config file setting for a single invocation:

```bash
cm --color-mode=light budget.cm
cm --color-mode=dark budget.cm
```

## Full Configuration Reference

All supported configuration keys are shown below. Leave color values empty (`""`) to use the built-in adaptive palette, which automatically provides appropriate colors for your color mode.

```toml
# CalcMark Configuration

# Display locale for number formatting (decimal/thousand separators).
# Supported: "en-US", "de-DE", "fr-FR"
locale = "en-US"

[tui]
# Color mode: "light" or "dark"
# Set to "light" if you use a light terminal background.
color_mode = "dark"

[tui.theme]
# Color overrides (hex strings: #RGB or #RRGGBB).
# Leave empty to use the built-in adaptive palette defaults.

# Primary brand color - titles, prompts, variable names
primary = ""

# Accent color - borders, panel highlights
accent = ""

# Error messages
error = ""

# Changed/modified indicator
warning = ""

# Help text, secondary info
muted = ""

# Hints, suggestions, preview text
dimmed = ""

# Calculation results/output
output = ""

# Syntax emphasis in help text
bright = ""

# Divider lines
separator = ""

# Pane backgrounds (leave empty for palette defaults)
source_pane_bg = ""
preview_pane_bg = ""
status_bar_bg = ""

[formatter]
# Default verbosity for output
verbose = false

# Include error details in exports
include_errors = true

# Default output format: "text", "json", "html", "md", "cm"
default_format = "text"
```

## Theme Examples

### Light Terminal Theme

If you use a light terminal background:

```toml
[tui]
color_mode = "light"

[tui.theme]
primary = "#5B21B6"
accent = "#7C3AED"
error = "#DC2626"
warning = "#D97706"
muted = "#6B7280"
dimmed = "#9CA3AF"
output = "#374151"
bright = "#111827"
separator = "#D1D5DB"
```

### High Contrast Theme

For better visibility:

```toml
[tui.theme]
primary = "#FFFF00"
accent = "#00FFFF"
error = "#FF0000"
warning = "#FFA500"
muted = "#FFFFFF"
dimmed = "#CCCCCC"
output = "#FFFFFF"
bright = "#FFFFFF"
separator = "#888888"
```

### Monochrome Theme

Minimal color palette:

```toml
[tui.theme]
primary = "#FFFFFF"
accent = "#AAAAAA"
error = "#FF6666"
warning = "#FFFFFF"
muted = "#888888"
dimmed = "#666666"
output = "#CCCCCC"
bright = "#FFFFFF"
separator = "#444444"
```

### Custom Pane Backgrounds

Customize the editor pane backgrounds:

```toml
[tui.theme]
source_pane_bg = "#1A1A2E"
preview_pane_bg = "#16213E"
status_bar_bg = "#0F3460"
```

## JSON Output and Locale {#json-raw-value}

When using `cm convert --to=json`, each result includes two value fields:

| Field | Description | Example (de-DE) |
|-------|-------------|-----------------|
| `value` | Locale-formatted display string | `"1.500,00"` |
| `raw_value` | Machine-readable ASCII (always en-US style) | `"1500"` |

The `raw_value` field is guaranteed to contain only ASCII characters, making it safe for programmatic consumption regardless of locale. Use `value` for display and `raw_value` for computation.

```json
{
  "source": "price = $1500",
  "value": "$1.500,00",
  "raw_value": "$1500",
  "variable": "price"
}
```

## What Can't Be Configured

The following are part of the internal adaptive palette and are not user-configurable:

- Markdown preview colors (headings, links, code blocks)
- Popup and dialog borders
- Separator shade variations
- Autocomplete highlight colors

These colors derive from the palette and adapt automatically to your color mode.

## Troubleshooting

### Config not loading?

1. Check file permissions: `ls -la ~/.config/calcmark/config.toml`
2. Validate TOML syntax (e.g., with `tomlv` or an online validator)
3. Ensure valid hex colors (must start with `#`)

### Colors look wrong?

- Ensure your terminal supports TrueColor (24-bit color)
- Some terminals need TrueColor enabled explicitly (e.g., `export COLORTERM=truecolor`)
- Try a simpler theme to isolate the issue
- Use `--color-mode=light` if your terminal has a light background

### Deprecation warnings?

If you see `dark_mode is deprecated`, replace it in your config:

```toml
# Before (deprecated)
dark_mode = true

# After
color_mode = "dark"
```

### Reset to defaults

Delete or rename your config file:

```bash
rm ~/.config/calcmark/config.toml
```
