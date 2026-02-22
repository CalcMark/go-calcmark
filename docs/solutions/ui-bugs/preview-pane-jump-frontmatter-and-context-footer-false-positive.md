---
title: "Fix Preview Pane Jump: Frontmatter Globals Shift and Context Footer │ Test False Positive"
date: 2026-02-22
category: ui-bugs
tags:
  - tui
  - editor
  - preview-pane
  - rendering
  - frontmatter
  - test-methodology
  - false-positive
severity: medium
component:
  - cmd/calcmark/tui/editor/view_panes.go
  - cmd/calcmark/tui/editor/selection_highlighting_test.go
  - cmd/calcmark/tui/editor/preview_pane_jump_test.go
  - cmd/calcmark/tui/editor/frontmatter_preview_jump_test.go
symptoms:
  - "Globals panel content shifts vertically when cursor moves between frontmatter lines"
  - "Test reports preview pane 'jump' when cursor moves to calc line referencing multiple variables"
  - "'Savings Target' heading appears to move by one line in test output"
root_cause:
  - "renderPreviewPaneAligned checked editBuf cursor-line path before isFrontmatter, bypassing globalsPanelIdx advancement"
  - "Tests parsed View() output by splitting on │, but context footer also uses │ as value separator"
files_changed:
  - cmd/calcmark/tui/editor/view_panes.go
  - cmd/calcmark/tui/editor/selection_highlighting_test.go
  - cmd/calcmark/tui/editor/frontmatter_preview_jump_test.go
  - cmd/calcmark/tui/editor/preview_pane_jump_test.go
related_docs:
  - docs/solutions/ui-bugs/tui-editor-rendering-divider-status-bar-error-line.md
  - docs/solutions/ui-bugs/lipgloss-background-bleed-through.md
  - docs/solutions/code-organization/split-view-go-into-cohesive-modules.md
---

# Preview Pane Jump: Frontmatter Globals Shift and Context Footer Test False Positive

## Problem

Two issues were reported as the same symptom — "preview pane content jumps when moving the cursor":

1. **Real bug:** Moving cursor between YAML frontmatter lines caused the globals panel content to shift vertically by one line.
2. **Test false positive:** A pre-existing test (`TestPreviewPaneJump_LastCalcBeforeEmptyLine`) reported that moving the cursor to `fixed_total = rent + utilities + insurance` caused an extra blank line before the "Savings Target" heading. Investigation proved the **rendering was correct** — the test methodology was flawed.

## Root Cause

### Bug 1: Frontmatter Preview Jump (Real)

In `renderPreviewPaneAligned` (view_panes.go), the code checked the `editBuf` cursor-line path BEFORE the `isFrontmatter` check:

```go
// BUGGY ORDER — editBuf fires even for frontmatter lines
if m.editBuf != "" && pl.sourceLineNum == m.cursorLine {
    // editBuf handling — bypasses globalsPanelIdx advancement!
}
if pl.isFrontmatter {
    // globals panel rendering — never reached when editBuf is active
}
```

When the cursor was on a frontmatter line and `editBuf` was non-empty (always true during navigation), the editBuf path would fire first. This bypassed the globals panel rendering logic that advances `globalsPanelIdx`, causing globals content to appear at wrong vertical positions.

### Bug 2: Context Footer │ False Positive (Test Bug)

The test used `extractPreviewPane(View())` which splits `View()` output on `│` to separate source and preview panes:

```go
func extractPreviewPane(view string) []string {
    for _, line := range strings.Split(view, "\n") {
        parts := strings.SplitN(line, "│", 2)
        if len(parts) >= 2 {
            previewLines = append(previewLines, parts[1])
        }
    }
}
```

The `│` character (Unicode box-drawing) serves **two different purposes** in the rendered output:

1. **Pane divider** — rendered by `SideBySide` between Source and Results panes
2. **Value separator** — rendered in the context footer below the panes

When the cursor moves to `fixed_total = rent + utilities + insurance` (which references 3 variables), the context footer renders:

```
rent = $1500.00 │ utilities = $200.00 │ insurance = $150.00
```

The `│` characters in this footer line cause `extractPreviewPane` to count it as an additional pane row, making it look like the preview pane grew by one line.

Direct testing with `renderPreviewPaneAligned` confirmed the actual pane output is identical regardless of cursor position — the "Savings Target" heading stays at the same vertical position.

## Solution

### Bug 1 Fix

Reordered conditional checks in `renderPreviewPaneAligned` to place `isFrontmatter` ABOVE `editBuf`:

```go
// FIXED ORDER — frontmatter always takes precedence
if pl.isFrontmatter {
    // Always render globals panel for frontmatter lines
    if globalsPanelIdx < len(globalsPanelLines) {
        completeLine = ensureFullWidth(globalsPanelLines[globalsPanelIdx], width, pvBg)
        globalsPanelIdx++
    }
    continue
}

if m.editBuf != "" && pl.sourceLineNum == m.cursorLine {
    // editBuf handling — only for non-frontmatter lines
}
```

Frontmatter is a **structural property** of the document that must be rendered consistently regardless of cursor position. The editBuf cursor-line path is **transient UI state** that should only apply to calc blocks and markdown.

### Bug 2 Fix

Rewrote `TestPreviewPaneJump_LastCalcBeforeEmptyLine` to call `renderPreviewPaneAligned` directly instead of parsing `View()` output:

```go
// BEFORE (fragile — parses composed output)
view := m.View()
previewLines := extractPreviewPane(view) // splits on │

// AFTER (robust — tests the component directly)
aligned := m.computeAlignedPanes(leftContentWidth, rightWidth)
preview := m.renderPreviewPaneAligned(rightWidth, paneHeight, aligned)
previewLines := strings.Split(preview, "\n")
```

Removed dead `extractPreviewPane` and `extractSourcePane` helper functions.

Added comprehensive new test files:
- `frontmatter_preview_jump_test.go` — validates globals panel position stability across frontmatter cursor movement
- `preview_pane_jump_test.go` — validates source/preview line count stability at multiple widths (60, 70, 80, 100, 120), editLineCount invariants, and adjacent line movement

## Prevention

### Code Ordering Discipline

When `renderPreviewPaneAligned` handles multiple line types, check them in priority order:

1. **Structural properties first** (frontmatter, block boundaries) — these don't change with cursor position
2. **Transient UI state second** (editBuf cursor-line, selection highlighting) — these depend on cursor position
3. **Default rendering last** (regular content)

### Test Methodology: Never Parse Composed View() Output on │

The `View()` method composes multiple UI layers: pane headers, source pane, divider, preview pane, context footer, separator, and status bar. Several of these layers use `│` for different purposes.

| Layer | Uses │ for |
|-------|-----------|
| SideBySide | Pane divider (structural) |
| Context footer | Value separator (content) |

**Correct approach:** Test each rendering layer at its own API boundary:

```go
// Good: test pane rendering directly
sourcePane := m.renderSourcePaneAligned(width, height, aligned)
previewPane := m.renderPreviewPaneAligned(width, height, aligned)

// Bad: parse composed output
view := m.View()
parts := strings.SplitN(line, "│", 2) // ambiguous!
```

### Recommended Test Patterns

- **Direct render calls** — call `renderSourcePaneAligned`/`renderPreviewPaneAligned` in isolation
- **Anchor position tracking** — find heading positions in preview output, assert they don't move across cursor positions
- **Line count stability** — verify source and preview line counts remain equal and constant as cursor moves
- **Multi-width testing** — test at widths 60, 70, 80, 100, 120 to catch wrapping-dependent bugs

### Anti-Patterns to Avoid

- Splitting `View()` on `│` to extract pane content
- Counting lines containing `│` as "pane rows"
- Helper functions that reverse-engineer layer structure from composed output
