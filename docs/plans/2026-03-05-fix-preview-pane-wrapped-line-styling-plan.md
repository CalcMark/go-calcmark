---
title: "Fix: Preview Pane Wrapped Line Styling"
type: fix
status: completed
date: 2026-03-05
issue: https://github.com/CalcMark/go-calcmark/issues/26
---

# Fix: Preview Pane Wrapped Line Styling

## Overview

When calc result lines in the preview pane wrap (due to narrow pane width or long variable names/values), continuation lines lose their ANSI color codes and appear as unstyled plain text. The root cause is `wrapStyledLine()` in `view_util.go` which strips all ANSI codes before wrapping, then returns plain text lines without re-applying styling.

## Problem Statement

**Current behavior**: Wrapped preview calc lines show as plain text (no foreground color, no background).

**Expected behavior**: All wrapped continuation lines maintain their themed styling (variable name color, arrow color, value color, preview pane background).

**Screenshot from issue**: Long variable names/values in preview pane lose color after the first line wraps.

## Root Cause

In `view_util.go:68-92`, `wrapStyledLine()`:

1. Checks if `visualWidth <= maxWidth` — if so, returns the styled string unchanged (correct)
2. When wrapping is needed, calls `stripANSI(line)` to get plain text
3. Wraps plain text via `geometry.WrapText(plainText, maxWidth)`
4. Returns the plain text wrapped lines — **all styling is discarded**

The unstyled lines flow through the alignment engine and into `renderPreviewPaneAligned()` which only pads them with `ensureFullWidth()` — never re-styling.

## Proposed Solution

Make `wrapStyledLine()` ANSI-state-aware using the same pattern proven in `overlayStringAt()` (view_util.go:153-242):

1. Strip ANSI to determine wrap points (as now)
2. Walk the **original styled string** rune-by-rune, tracking active ANSI state
3. Split the styled string at wrap boundaries
4. Prepend accumulated ANSI state to each continuation line so styles carry across

This is a single-function fix with no changes needed to the alignment engine or rendering pipeline.

### Algorithm

```
wrapStyledLine(styledLine, maxWidth):
  1. Early return if fits in maxWidth (existing behavior, unchanged)
  2. plainText = stripANSI(styledLine)
  3. wrappedPlain = geometry.WrapText(plainText, maxWidth)
  4. Walk styledLine rune-by-rune:
     - Track ANSI escape sequences (non-reset codes accumulate, resets clear)
     - Count visual characters to find wrap boundaries from wrappedPlain
     - At each wrap boundary, start a new line prepended with accumulated ANSI state
  5. Return styled wrapped lines
```

### Key Design Decisions

**ANSI state replay**: Each continuation line starts with the accumulated non-reset ANSI codes from prior segments. This matches `overlayStringAt()` semantics and ensures lipgloss foreground/background codes carry across wrap boundaries correctly.

**Trailing reset handling**: Styled content from `renderCalcLine()` ends with `\x1b[0m` resets from lipgloss. These resets are preserved within each wrapped line. `ensureFullWidth()` appends `StyledPadding()` after the content, which applies its own background — this is safe because lipgloss `Width()` correctly ignores ANSI codes when measuring.

**Error lines unaffected**: `renderCalcLine()` pre-truncates error text to `width - 4` characters, so error lines will not wrap in practice. The fix handles them correctly if they do.

**Non-calc lines in CalcBlock**: The `!isActuallyCalc` branch returns `mdRenderer.RenderLine(source)[0]` — already wrapped by glamour at the correct width. If it reaches `wrapStyledLine()`, the ANSI-aware wrap handles it correctly.

**Markdown headings unaffected**: Headings go through `renderMarkdown()` in `resolvePreviewLines()`, completely bypassing `wrapStyledLine()`. No risk of regression.

## Acceptance Criteria

- [x] Wrapped calc result lines in preview pane maintain styling (foreground colors for variable name, arrow, value)
- [x] Wrapped lines maintain preview pane background color
- [x] Non-wrapping lines produce byte-for-byte identical output (no regression)
- [x] PreviewFull mode: `varName → value` wraps with correct multi-segment styling
- [x] PreviewMinimal mode: `→ value` wraps with correct styling
- [x] Changed values (`* varName → value`) wrap with correct bold/yellow styling
- [x] `task test` passes with zero failures
- [x] `task quality` passes

## MVP

### 1. Unit Tests — `cmd/calcmark/tui/editor/view_util_test.go`

Create table-driven tests for `wrapStyledLine()`:

```go
// Test cases:
// - Plain string (no ANSI) that wraps — baseline behavior preserved
// - Styled string that fits — returns original unchanged
// - Multi-segment styled string wrapping at variable name boundary
// - Multi-segment styled string wrapping at value boundary
// - Single-segment styled string (PreviewMinimal mode)
// - WasChanged bold+colored marker wrapping
// - Empty string and zero-width edge cases
```

### 2. Fix `wrapStyledLine()` — `cmd/calcmark/tui/editor/view_util.go`

Replace lines 68-92 with ANSI-state-aware wrapping:

```go
func wrapStyledLine(line string, maxWidth int) []string {
    // Early returns unchanged (existing)
    // ...

    // Strip ANSI for wrap-point computation
    plainText := stripANSI(line)
    wrappedPlain := geometry.WrapText(plainText, maxWidth)
    if len(wrappedPlain) <= 1 {
        return []string{line}
    }

    // Walk styled string, split at wrap boundaries,
    // replay accumulated ANSI state on each continuation line
    // (uses overlayStringAt pattern for ANSI state tracking)
    // ...
}
```

### 3. Regenerate Catwalk Expectations — `cmd/calcmark/tui/editor/testdata/wrapping_calc_lines`

After fix:
```bash
go test ./cmd/calcmark/tui/editor/... -run TestEditorCatwalkWrapping -args -rewrite
```

Review regenerated expectations: continuation lines in `PRV` column should now contain ANSI escape codes instead of plain text.

### 4. Validation

```bash
task test      # Full test suite
task quality   # Linting + quality gates
```

## References

- GitHub issue: https://github.com/CalcMark/go-calcmark/issues/26
- ANSI state tracking pattern: `cmd/calcmark/tui/editor/view_util.go:153-242` (`overlayStringAt`)
- Background bleed-through solution: `docs/solutions/ui-bugs/lipgloss-background-bleed-through.md`
- ANSI state compositing solution: `docs/solutions/ui-bugs/overlay-compositing-ansi-state-bleed-through.md`
- Calc line rendering: `cmd/calcmark/tui/editor/view_panes.go:334-414` (`renderCalcLine`)
- Wrapping entry point: `cmd/calcmark/tui/editor/aligned.go:299` (`resolvePreviewLines`)
