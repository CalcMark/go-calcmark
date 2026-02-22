---
title: "Fix Terminal Background Bleed-Through Across All TUI Views"
date: 2026-02-22
category: ui-bugs
tags:
  - tui
  - editor
  - repl
  - lipgloss
  - theming
  - background-colors
  - adaptive-color
  - overlays
severity: medium
component:
  - cmd/calcmark/tui/editor/overlay_style.go
  - cmd/calcmark/tui/editor/view.go
  - cmd/calcmark/tui/editor/view_panes.go
  - cmd/calcmark/tui/editor/view_overlays.go
  - cmd/calcmark/tui/repl/view.go
  - cmd/calcmark/config/theme.go
  - cmd/calcmark/config/theme/palette.go
symptoms:
  - "Dark-backgrounded popups (help, export, command menu, file picker) on light terminals"
  - "Terminal default background bleeding through text elements without explicit Background() set"
  - "Frontmatter lines not visually distinct from body text in source pane"
  - "REPL mode showing terminal default background instead of themed background"
  - "Gaps between styled text segments (e.g., calc result arrow/value) showing terminal default"
root_cause:
  - "lipgloss styles with Foreground() but no Background() inherit terminal default background"
  - "lipgloss.Place() for overlays only had WithWhitespaceForeground, missing WithWhitespaceBackground"
  - "Overlay border style had no Background() set"
  - "REPL view elements had no Background() or Width() for full-width coverage"
  - "Frontmatter color too subtle (#6B7280) against dark background"
files_changed:
  - cmd/calcmark/tui/editor/overlay_style.go
  - cmd/calcmark/tui/editor/view.go
  - cmd/calcmark/tui/editor/view_panes.go
  - cmd/calcmark/tui/editor/view_overlays.go
  - cmd/calcmark/tui/repl/view.go
  - cmd/calcmark/config/theme.go
  - cmd/calcmark/config/theme/palette.go
related_docs:
  - docs/THEMING.md
  - docs/plans/2026-02-22-refactor-tui-theme-consistency-plan.md
  - docs/brainstorms/2026-02-22-tui-theme-consistency-brainstorm.md
  - docs/solutions/ui-bugs/tui-editor-rendering-divider-status-bar-error-line.md
commits:
  - "c5ba7a0 fix(theme): eliminate terminal background bleed-through across all TUI views"
---

# Fix Terminal Background Bleed-Through Across All TUI Views

## Problem

After migrating the TUI theme system from hardcoded ANSI-256 colors to a centralized `lipgloss.AdaptiveColor` palette with light/dark support, the terminal's default background was bleeding through in multiple areas of the UI. This was most visible when running a dark-themed TUI on a light terminal (or vice versa), where unstyled gaps and elements would show the terminal's background color instead of the theme's background color.

### Symptoms Observed

1. **Popup/modal overlays** (Ctrl-Q help, Ctrl-E export, command menu, file picker) appeared with dark backgrounds on light terminals because the overlay backdrop and border had no explicit background color.

2. **Text elements throughout the editor** (line numbers, source text, calc results, globals panel entries) showed terminal default background between styled segments.

3. **Frontmatter lines** in the source pane were not visually distinct from markdown body text — the tinting color was too similar to see.

4. **REPL mode** had no themed backgrounds at all — every element used inline `lipgloss.NewStyle()` with only foreground colors.

## Root Cause

The core issue is a fundamental property of lipgloss: **a style that sets `Foreground()` without `Background()` inherits the terminal's default background color**, not the theme's background. This creates visual "holes" wherever text is rendered without an explicit background.

Seven specific root causes were identified:

### 1. Overlay Backdrop (view.go)

`lipgloss.Place()` calls for centering overlays on screen only had `WithWhitespaceForeground()` but not `WithWhitespaceBackground()`. The whitespace fill characters around the overlay used the terminal default background.

### 2. Overlay Border Style (overlay_style.go)

The shared `OverlayStyle` border style (`bs`) had `Foreground(theme.OverlayBorder)` but no `Background()`, causing border characters to show terminal background between the overlay content and the border.

### 3. Editor Styles Missing Backgrounds (theme.go)

Eight pre-built styles in `config.Styles` were constructed without explicit backgrounds:
- `LineNumber`, `SourceText` (should use source pane bg)
- `CalcVarName`, `CalcArrow`, `CalcValue` (should use preview pane bg)
- `SourceFrontmatter`, `SourceMarkdown`, `SourceCalc` (should use source pane bg)

### 4. Block Tint Function Missing Background (view_panes.go)

`applyBlockTint()` applied foreground color tints for frontmatter/calc/markdown lines but the markdown default case had no background, allowing terminal default to show through on prose lines.

### 5. Calc Result Gaps (view_panes.go)

Spaces between calc result segments (variable name, arrow, value) were plain `" "` strings without any styling, creating unstyled gaps. Error and blocked-error styles also lacked backgrounds.

### 6. Globals Panel (view_overlays.go)

All text elements in the globals panel (header, hint, error text, empty state message, variable entries) used `lipgloss.NewStyle()` with only foreground colors, missing `Background(paneBg)`.

### 7. REPL View (repl/view.go)

Every single UI element in the REPL (mode indicator, welcome message, history entries, hints, errors, separator, help footer) was styled with inline `lipgloss.NewStyle()` with only foreground colors and no backgrounds or width constraints.

## Solution

### Fix 1: Overlay Backdrop — Add WithWhitespaceBackground

**File:** `cmd/calcmark/tui/editor/view.go`

Added `lipgloss.WithWhitespaceBackground(theme.OverlayWhitespaceFg)` to all 4 `lipgloss.Place()` calls (help, command menu, file picker, export overlays):

```go
return lipgloss.Place(m.width, m.height,
    lipgloss.Center, lipgloss.Center,
    helpView,
    lipgloss.WithWhitespaceChars(" "),
    lipgloss.WithWhitespaceForeground(theme.OverlayWhitespaceFg),
    lipgloss.WithWhitespaceBackground(theme.OverlayWhitespaceFg),
)
```

### Fix 2: Overlay Border Style — Add Background

**File:** `cmd/calcmark/tui/editor/overlay_style.go`

```go
bs := lipgloss.NewStyle().
    Foreground(theme.OverlayBorder).
    Background(theme.OverlayBg)  // Added
```

### Fix 3: Editor Styles — Add Pane-Specific Backgrounds

**File:** `cmd/calcmark/config/theme.go`

Added `.Background(sourcePaneBg)` or `.Background(previewPaneBg)` to all 8 style definitions that were missing backgrounds.

### Fix 4: Block Tint Function — Accept and Apply Background

**File:** `cmd/calcmark/tui/editor/view_panes.go`

Changed `applyBlockTint` signature to accept a `bg lipgloss.TerminalColor` parameter and apply it to ALL cases including the markdown default:

```go
func applyBlockTint(content string, sourceLineIdx, fmCount int, isCalc bool, bg lipgloss.TerminalColor) string {
    // ...
    default:
        return lipgloss.NewStyle().
            Foreground(theme.SourceMarkdown).
            Background(bg).  // Now applies background
            Render(content)
    }
}
```

### Fix 5: Calc Result Styled Spaces

**File:** `cmd/calcmark/tui/editor/view_panes.go`

Replaced bare `" "` between calc result segments with styled spaces:

```go
sp := lipgloss.NewStyle().Background(pvBg).Render(" ")
// Used in: changedMarker + m.styles.CalcVarName.Render(r.VarName) + sp + m.styles.CalcArrow.Render("->") + sp + valueStyle.Render(r.Value)
```

### Fix 6: Globals Panel Backgrounds

**File:** `cmd/calcmark/tui/editor/view_overlays.go`

Added `Background(paneBg)` to all globals panel text elements: header, hint, error, empty state, and variable entry prefix/name/value styles.

### Fix 7: REPL Full-Width Backgrounds

**File:** `cmd/calcmark/tui/repl/view.go`

Added `Background(theme.SourcePaneBg)` and `.Width(m.width)` to all REPL elements. For history entries, added manual padding to ensure full-width coverage.

### Fix 8: Frontmatter Color Visibility

**File:** `cmd/calcmark/config/theme/palette.go`

Brightened `SourceFrontmatter` dark color from `#6B7280` to `#8B95A3` for more visible contrast against the dark background.

## Key Principle

**Every lipgloss style that renders visible text MUST have an explicit `Background()` set.** The background should come from the appropriate pane context:

| Context | Background Source |
|---------|------------------|
| Source pane content | `m.sourcePaneBg()` or `theme.SourcePaneBg` |
| Preview pane content | `m.previewPaneBg()` or `theme.PreviewPaneBg` |
| Overlay content | `theme.OverlayBg` |
| Status bar | `theme.StatusBg` |
| Context footer | `theme.ContextFooterBg` |
| REPL elements | `theme.SourcePaneBg` |

## Prevention Strategies

### 1. Code Review Checklist

When reviewing TUI rendering code, check:
- Does every `lipgloss.NewStyle()` chain include `.Background()`?
- Does every `lipgloss.Place()` include both `WithWhitespaceForeground` AND `WithWhitespaceBackground`?
- Are string concatenations between styled segments using styled spaces (`lipgloss.NewStyle().Background(bg).Render(" ")`) instead of bare `" "`?

### 2. Grep Pattern for Detection

Run this to find potentially problematic styles:

```bash
# Find lipgloss styles that set Foreground but not Background on the same chain
grep -n "Foreground(" cmd/calcmark/tui/**/*.go | grep -v "Background("
```

Note: This produces false positives for styles used in contexts where background is inherited. Manual review is needed.

### 3. Visual Testing

Always test theme changes on BOTH dark and light terminals. The bleed-through is only visible when the terminal's background differs from the theme's expected background. Use `color_mode = "light"` in config to force light palette on a dark terminal (or vice versa) for rapid testing.

### 4. Architectural Pattern

When adding new UI components, follow this pattern:

```go
// Get the appropriate background for this context
bg := m.sourcePaneBg() // or m.previewPaneBg(), theme.OverlayBg, etc.

// Every style gets explicit background
textStyle := lipgloss.NewStyle().
    Foreground(theme.SomeColor).
    Background(bg)

// Styled spaces between elements
sp := lipgloss.NewStyle().Background(bg).Render(" ")

// Full-width lines
line := ensureFullWidth(styledContent, width, bg)
```

## Related Issues

- Previous rendering fix: `docs/solutions/ui-bugs/tui-editor-rendering-divider-status-bar-error-line.md` — fixed divider, status bar, and error line mapping
- Frontmatter editing fix: `docs/solutions/ui-bugs/frontmatter-editing-keyboard-dispatch-fixes.md` — fixed keyboard dispatch during frontmatter editing
- Theme system documentation: `docs/THEMING.md`
