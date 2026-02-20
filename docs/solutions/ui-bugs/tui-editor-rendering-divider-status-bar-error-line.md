---
title: "Fix TUI Editor Display Bugs: Divider Rendering, Status Bar Wrapping, Error Line Mapping"
date: 2026-02-20
category: ui-bugs
tags:
  - tui
  - editor
  - rendering
  - diagnostics
  - bubble-tea
  - lipgloss
severity: medium
component:
  - cmd/calcmark/tui/editor/sidebyside.go
  - cmd/calcmark/tui/components/statusbar.go
  - impl/document/evaluator.go
symptoms:
  - "Missing vertical divider between Source and Results panes"
  - "Lines rendering at 79 characters instead of 80"
  - "Status bar text wrapping to second line"
  - "Error diagnostic displayed on wrong source line"
root_cause:
  - "SideBySide.Render() never rendered the divider character reserved by view.go"
  - "RenderStatusBar content width ignored Bar style horizontal padding"
  - "evaluator.go used *ast.Assignment type assertion instead of Node.GetRange() interface method"
files_changed:
  - cmd/calcmark/tui/editor/sidebyside.go
  - cmd/calcmark/tui/components/statusbar.go
  - impl/document/evaluator.go
related_phases:
  - "Phase 10: Preview Pane"
  - "Phase 11.2: UX Redesign"
  - "Phase 13: Clipboard"
commits:
  - "7c3aed8 fix(tui): add pane divider, fix status bar wrapping, fix error line mapping"
  - "0540be1 style: apply go modernize linter suggestions"
---

# TUI Editor Display Bugs: Divider, Status Bar, Error Line

Three independent bugs in the TUI editor that together caused ~12 test failures.

## Bug 1: Missing Vertical Divider Between Panes

### Problem

No `│` character between the Source and Results panes. Rendered lines were 79 characters instead of the expected 80. Affected ~10 tests including SideBySide unit tests, visual layout tests, and catwalk integration tests.

### Root Cause

`SideBySide.Render()` in `sidebyside.go` concatenated left and right panes without inserting a divider:

```go
// BROKEN: left + right with no divider
leftPadded := s.padLine(leftLines[i], s.leftWidth, s.leftBg)
result.WriteString(leftPadded)
rightPadded := s.padLine(rightLines[i], s.rightWidth, s.rightBg)
result.WriteString(rightPadded)
```

Meanwhile `view.go` correctly reserved 1 character for the divider:

```go
const dividerWidth = 1
leftContentWidth := leftWidth - dividerWidth
```

The two components had mismatched expectations: `view.go` assumed `SideBySide` would render the divider, but `SideBySide` never did.

### Fix

Added divider rendering between panes in `SideBySide.Render()` and updated `TotalWidth()`:

```go
dividerStyle := lipgloss.NewStyle().
    Foreground(lipgloss.Color("240")).
    Background(s.leftBg)

leftPadded := s.padLine(leftLines[i], s.leftWidth, s.leftBg)
result.WriteString(leftPadded)
result.WriteString(dividerStyle.Render("│"))  // NEW
rightPadded := s.padLine(rightLines[i], s.rightWidth, s.rightBg)
result.WriteString(rightPadded)

// TotalWidth updated: leftWidth + 1 + rightWidth
```

---

## Bug 2: Status Bar Wrapping to Second Line

### Problem

Status bar "Ctrl+H help" wrapped to a second line. Content rendered at 82 visible characters in an 80-char terminal.

### Root Cause

`RenderStatusBar` computed content to fill the full `width` (80 chars), but `style.Bar` has `Padding(0, 1)` which adds 1 char left + 1 char right = 2 extra characters. Total output: 80 content + 2 padding = 82 chars, which wrapped at 80.

```go
// BROKEN: content sized for full width, ignoring Bar's padding
if totalContent < width-4 {
    padding := (width - totalContent) / 2
```

### Fix

Account for the Bar style's horizontal padding when computing content width:

```go
barHPad := style.Bar.GetHorizontalPadding()
contentWidth := width - barHPad

if totalContent < contentWidth-4 {
    padding := (contentWidth - totalContent) / 2
```

---

## Bug 3: Error Displayed on Wrong Source Line

### Problem

An error from `accumulate(5mb, 1 hour)` on line 9 was displayed on `salary = $5000` (line 3). Affected 2 tests.

### Root Cause

`evaluator.go:453` only extracted `Range` position from `*ast.Assignment` nodes:

```go
// BROKEN: only works for assignments
if assignment, ok := node.(*ast.Assignment); ok && assignment.Range != nil {
    diag.Line = assignment.Range.Start.Line
```

When the failing node was a bare expression (function call), the type assertion failed and the diagnostic got `Line: 0, Column: 0`. The fallback heuristic in `results.go` then placed the error on the first non-empty line of the block.

### Fix

Use `GetRange()` from the `Node` interface, which all AST nodes implement:

```go
// FIXED: works for ALL AST node types
if r := node.GetRange(); r != nil {
    diag.Line = r.Start.Line
    diag.Column = r.Start.Column
}
```

---

## Prevention Strategies

### 1. Component Contract Mismatch (Bug 1)

**Pattern to watch for**: One component reserves space expecting another to fill it. Comments say "reserved for X" but no code renders X.

**Prevention**:
- When a component reserves space, the same component (or its explicitly documented partner) must render it
- Test that `TotalWidth()` matches actual rendered width via property-based tests
- Paired integration tests verify both sides of a contract

### 2. Hidden Style Properties (Bug 2)

**Pattern to watch for**: Width calculations that don't call `style.GetHorizontalPadding()` or `style.GetVerticalPadding()` before sizing content.

**Prevention**:
- Always query style padding before computing content width: `contentWidth := width - style.GetHorizontalPadding()`
- Document style assumptions in comments
- Test at exact boundary widths to catch off-by-one overflows

### 3. Type Assertion Instead of Interface Method (Bug 3)

**Pattern to watch for**: `if x, ok := node.(*SpecificType); ok` when an interface method like `node.GetRange()` exists and covers all types.

**Prevention**:
- Prefer interface methods over type assertions when the interface provides the needed data
- When reviewing code that operates on interface types, ask: "Does this handle all implementations?"
- Test with every concrete type that implements the interface, not just one

---

## Related References

- Phase 10 (Preview Pane): Established SideBySide rendering and 60/40 pane layout
- Phase 11.2 (UX Redesign): Established status bar format with `Padding(0, 1)`
- Phase 9.1 (Separate Validation/Execution): Migrated evaluator diagnostics to `impl/document`
- `cmd/calcmark/tui/editor/TESTING.md`: Catwalk testing documentation
- `spec/ast/nodes.go:11`: `Node` interface with `GetRange() *Range`
