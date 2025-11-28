# CalcMark Configuration

CalcMark supports user configuration via TOML files. Configuration controls TUI appearance (colors, themes) and formatter defaults.

## Configuration File Locations

CalcMark checks for configuration files in this order (first found wins for each setting):

1. `~/.config/calcmark/config.toml` (XDG standard, recommended)
2. `~/.calcmarkrc.toml` (dotfile fallback)

You only need to specify values you want to override - unspecified values use sensible defaults.

## Quick Start

Create a config file:

```bash
mkdir -p ~/.config/calcmark
touch ~/.config/calcmark/config.toml
```

Add your customizations:

```toml
[tui.theme]
primary = "#00FF00"  # Change the primary color to green
```

## Full Configuration Reference

```toml
# CalcMark Configuration

[tui]
dark_mode = true  # Assume dark terminal background

[tui.theme]
# All colors are hex strings (#RGB or #RRGGBB)
# Defaults are high-contrast for readability on dark backgrounds

# Primary brand color - titles, prompts, variable names
primary = "#7D56F4"

# Accent color - borders, panel highlights
accent = "#874BFD"

# Error messages
error = "#FF5555"

# Changed/modified indicator (e.g., updated variables)
warning = "#FFAA00"

# Help text, secondary info
muted = "#888888"

# Hints, suggestions, preview text
dimmed = "#BBBBBB"

# Calculation results/output - bright for readability
output = "#FFFFFF"

# Syntax emphasis in help text
bright = "#FFFFFF"

# Divider lines
separator = "#555555"

# Markdown preview colors (used in /md mode)
md_text = "#FFFFFF"       # Body text
md_h1_bg = "#FF9900"      # H1 heading background (orange)
md_h2_bg = "#33CC33"      # H2 heading background (green)
md_heading = "#FF9900"    # H3+ heading text color
md_link = "#22AA22"       # Link color (darker green for readability)
md_quote = "#888888"      # Block quote indicator (gray │)
md_code = "#33CC33"       # Inline code text
md_code_bg = "#333333"    # Code background

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
dark_mode = false

[tui.theme]
primary = "#5B21B6"    # Darker purple
accent = "#7C3AED"
error = "#DC2626"
warning = "#D97706"
muted = "#6B7280"
dimmed = "#9CA3AF"
output = "#374151"
bright = "#111827"
separator = "#D1D5DB"

# Markdown preview for light backgrounds
md_text = "#1F2937"
md_h1_bg = "#C2410C"
md_h2_bg = "#15803D"
md_heading = "#C2410C"
md_link = "#15803D"
md_code = "#15803D"
md_code_bg = "#F3F4F6"
```

### High Contrast Theme

For better visibility:

```toml
[tui.theme]
primary = "#FFFF00"    # Yellow
accent = "#00FFFF"     # Cyan
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

## Troubleshooting

### Config not loading?

1. Check file permissions: `ls -la ~/.config/calcmark/config.toml`
2. Validate TOML syntax: `cat ~/.config/calcmark/config.toml`
3. Ensure valid hex colors (must start with `#`)

### Colors look wrong?

- Ensure your terminal supports TrueColor (24-bit color)
- Some terminals need TrueColor enabled explicitly
- Try a simpler theme to isolate the issue

### Reset to defaults

Simply delete or rename your config file:

```bash
rm ~/.config/calcmark/config.toml
```
