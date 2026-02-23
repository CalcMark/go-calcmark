---
title: "Overlay compositing ANSI state bleed-through"
date: 2026-02-22
category: ui-bugs
tags: [tui, ansi, lipgloss, overlay, compositing, background-styling, bubbletea, autocomplete, context-footer, terminal-rendering]
severity: medium
components:
  - cmd/calcmark/tui/editor/view_util.go
  - cmd/calcmark/tui/components/contextfooter.go
  - cmd/calcmark/tui/components/statusbar.go
  - cmd/calcmark/tui/editor/sidebyside.go
  - cmd/calcmark/tui/editor/overlay_test.go
symptoms:
  - "Terminal default background bleeds through to the RIGHT of the autocomplete popup overlay"
  - "Terminal default background visible in context footer between styled text segments"
  - "Unstyled gaps around parentheses, spaces, and indentation in context footer"
root_cause: "overlayStringAt() did not track/replay ANSI escape sequences when compositing onto single-envelope base lines, and raw string literals between lipgloss.Render() calls had no background styling"
resolution_type: code-fix
time_to_resolve: ~4 hours
related_docs:
  - docs/solutions/ui-bugs/lipgloss-background-bleed-through.md
  - docs/solutions/code-organization/split-view-go-into-cohesive-modules.md
  - docs/THEMING.md
---

# Overlay Compositing ANSI State Bleed-Through

This is the **second round** of bleed-through fixes. The [first round](lipgloss-background-bleed-through.md) addressed missing `Background()` calls on lipgloss styles and adaptive color palette adoption. This round fixes ANSI escape sequence state management during overlay compositing and unstyled gaps in the context footer.

## Problem

Three distinct areas of terminal background color bleed-through:

1. **Autocomplete popup right-side bleed**: Characters to the RIGHT of the popup overlay lost their background color, showing the terminal default (white on light terminals).

2. **Context footer inter-segment gaps**: Raw string literals (`" "`, `"("`, `")"`, `"  "`) between `lipgloss.Render()` calls had no background styling, creating tiny visible strips of terminal default background.

3. **Status bar area**: Gaps where inner ANSI resets from sub-components cleared the outer background mid-line.

## Root Cause

### Mechanism A: Single-Envelope ANSI Wrapping

`SideBySide.padLine()` in `sidebyside.go` calls `stripResetCodes()` to remove all internal `\x1b[0m` resets, then wraps the entire line in a single background envelope:

```
\x1b[48;2;R;G;Bm  1 some text          \x1b[0m
```

One opening background code at the start, one reset at the end -- zero ANSI codes in between.

### Mechanism B: Lost State in overlayStringAt()

The old `overlayStringAt()` composited popup overlays by:
1. Copying the base line up to the overlay column
2. Appending the overlay content
3. Appending `\x1b[0m` to reset overlay styles
4. Skipping the overlaid region in the base line
5. Appending the rest of the base line

After step 3's reset, the function needed to re-establish the base line's background. It relied on finding ANSI codes within the skipped region to replay -- but with single-envelope lines, the skip loop found **zero** internal codes. Everything after the overlay rendered unstyled.

### Mechanism C: Unstyled Inter-Segment Strings

In `contextfooter.go`, concatenation like:

```go
line1 := funcStyle.Render(name) + "(" + paramStyle.Render(param) + ")"
```

The `"("` and `")"` literals had no background color, creating 1-character gaps.

## Solution

### Fix 1: ANSI State Tracking in overlayStringAt()

**File**: `cmd/calcmark/tui/editor/view_util.go` (lines 152-241)

Added a `baseANSIState` accumulator that collects every non-reset ANSI escape sequence from both the pre-overlay scan and the skip-over-overlay scan, then replays them after the overlay's reset:

```go
var baseANSIState []rune

// Pre-overlay scan: collect ANSI state
if r == '\x1b' {
    var esc []rune
    // ... collect escape sequence ...
    if escStr != "\x1b[0m" && escStr != "\x1b[m" {
        baseANSIState = append(baseANSIState, esc...)
    }
}

// Skip-over-overlay scan: resets clear state, non-resets add to it
if escStr == "\x1b[0m" || escStr == "\x1b[m" {
    baseANSIState = baseANSIState[:0]
} else {
    baseANSIState = append(baseANSIState, esc...)
}

// After overlay: replay accumulated state
result = append(result, baseANSIState...)
result = append(result, baseRunes[baseIdx:]...)
```

This handles both cases:
- **Single-envelope**: Opening `\x1b[48;2;R;G;Bm` is captured in the first loop and replayed
- **Multi-segment**: All non-reset codes from both loops are accumulated

### Fix 2: bgText() Helper in contextfooter.go

**File**: `cmd/calcmark/tui/components/contextfooter.go` (lines 88-91)

```go
bgText := func(s string) string {
    return lipgloss.NewStyle().Background(bg).Render(s)
}
```

All raw inter-segment strings now use this helper:

```go
// Before (bleed-through):
line1 += " " + syntaxStyle.Render(state.AutocompleteSyntax)
line1 := funcStyle.Render(name) + "(" + paramStyle.Render(param) + ")"

// After (no bleed):
line1 += bgText(" ") + syntaxStyle.Render(state.AutocompleteSyntax)
line1 := funcStyle.Render(name) + bgText("(") + paramStyle.Render(param) + bgText(")")
```

### Fix 3: Explicit Per-Segment Backgrounds in statusbar.go

**File**: `cmd/calcmark/tui/components/statusbar.go`

Each line is built from individually-styled segments with `StyledPadding()` fills:

```go
buildLine := func(content string, contentVisualWidth int) string {
    line := StyledPadding(1, barBg) + content
    remaining := width - 1 - contentVisualWidth
    if remaining > 0 {
        line += StyledPadding(remaining, barBg)
    }
    return line
}
```

## Verification

### Regression Tests

**File**: `cmd/calcmark/tui/editor/overlay_test.go`

Two tests specifically target bleed-through. Both use **raw ANSI escape codes** rather than `lipgloss.Render()` because lipgloss strips ANSI in test environments (no terminal detected):

- **`TestOverlayStringAt_MultiSegmentBase`**: Verifies trailing characters after overlay retain ANSI styling with multi-segment base lines
- **`TestOverlayStringAt_SingleEnvelopeBase`**: Simulates real `SideBySide.padLine()` output, overlays a popup, and asserts that characters between the overlay reset and the divider have ANSI styling. Fails with "BLEED" message if unstyled characters detected.

### Full Test Suite

All tests pass via `task test` and `task quality` with zero regressions.

## Prevention

### Design Principles

1. **No raw string literals in styled pipelines.** If a rune appears on screen, it must pass through `lipgloss.Style.Render()`. This includes spaces, separators, padding, and punctuation.

2. **Compositing must preserve escape state.** When overlaying one styled string on another, track ANSI state at the splice point and replay it after the overlay ends. Treat ANSI escape sequences as invisible tokens, not as characters to skip.

3. **Pad with style, not with spaces.** Use `StyledPadding(width, bg)` instead of `strings.Repeat(" ", width)`.

4. **Never rely on outer Render() for background coverage.** Inner `\x1b[0m` resets will punch holes. Style each segment explicitly.

### Code Review Checklist

- [ ] String concatenation between `Render()` calls: every joining character is styled
- [ ] Width padding uses `StyledPadding()` or equivalent, not bare spaces
- [ ] Overlay/compositing functions track and replay ANSI state
- [ ] Width calculations use `lipgloss.Width()`, not `len()`
- [ ] Full-width components (status bar, footer) have no unstyled gaps edge-to-edge

### Testing

- Use raw ANSI codes in tests (not `lipgloss.Render()`) when testing ANSI state behavior
- Assert presence of ANSI escape codes in regions that should be styled
- Test with forced `termenv.TrueColor` profile when lipgloss output is needed

### Common Pitfalls

| Pitfall | Example | Fix |
|---------|---------|-----|
| Raw separator | `Render(a) + " \| " + Render(b)` | `Render(a) + sepStyle.Render(" \| ") + Render(b)` |
| Bare padding | `line + strings.Repeat(" ", n)` | `line + StyledPadding(n, bg)` |
| Outer Render reliance | `container.Render(inner1 + inner2)` | Style each segment with explicit bg |
| Overlay without ANSI tracking | Splice runes, append rest | Track `baseANSIState`, replay after reset |
| lipgloss in tests | `style.Render()` produces no ANSI | Use raw `\x1b[48;2;R;G;Bm` codes |

## Related

- [First round bleed-through fix](lipgloss-background-bleed-through.md) -- Missing `Background()` calls on styles
- [View module split](../code-organization/split-view-go-into-cohesive-modules.md) -- Created the file structure where fixes were applied
- [Theming spec](../../THEMING.md) -- Color system architecture
- [bubbletea#1004](https://github.com/charmbracelet/bubbletea/issues/1004) -- Related terminal rendering issue
- Phase research: `.planning/phases/11.2-ux-redesign/11.2-RESEARCH.md` (lines 175-178: "Pitfall 5: Background color bleeding through popup")

## Commit

`6583511 fix: eliminate terminal background bleed-through in TUI overlays and footer`
Released in v1.1.2.
