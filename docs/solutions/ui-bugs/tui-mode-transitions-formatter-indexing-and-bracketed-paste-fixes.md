---
title: "TUI mode-transition centralization, formatter result alignment, and bracketed paste handling"
date: 2026-02-26
category: ui-bugs
tags:
  - tui-editor
  - output-formatters
  - bracketed-paste
  - refactoring
components:
  - cmd/calcmark/tui/editor/
  - format/
severity: medium
symptoms:
  - "Ctrl+K wrong line, no undo; Ctrl+E non-functional"
  - "Stale overlay state after dismiss; stale undo history on file open"
  - "Bracketed paste (tea.PasteMsg) silently dropped"
  - "Preview pane wrapping diverged from source pane"
  - "Formatter results misaligned after blank lines in calc blocks"
root_causes:
  - "Mode assignments scattered across 10+ files with inconsistent field resets"
  - "deleteLine() called without undo capture or edit buffer flush"
  - "No tea.PasteMsg handler; alignment used committed text, not live editBuf"
  - "Source-line loop index used to access results array (blank lines don't produce results)"
---

## Overview

Three distinct problem areas fixed across commits `f71cd1c`, `42409f8`, and `67c3cb0`, building on the earlier Bubble Tea v2 migration work. The common theme is user-facing correctness in the TUI editor and output formatters.

## Problem 1: Scattered Mode Transitions Caused Stale State

### Symptoms

- Ctrl+K deleted the wrong line (next instead of current), had no undo support, and didn't trigger re-evaluation
- Ctrl+E was completely non-functional (disabled due to a legacy macOS terminal workaround never restored)
- Dismissing overlays (help, command menu, export, file picker, save prompt) sometimes left residual state causing ghost interactions on subsequent opens
- Opening a new file while editing could inherit stale undo history and cursor state

### Root Cause

Mode transitions (`m.mode = StateXxx`) were scattered across 10 files with 35+ inline assignments. Each call site had to independently remember which fields to reset. Some forgot `pendingSaveAction`, others forgot `newFileName`. The `openFile` method failed to call `transitionToReady()`. The Ctrl+K handler called low-level `deleteLine()` directly without capturing undo state or flushing the edit buffer.

### Solution

Created `mode_transitions.go` to centralize all mode changes with consistent field resets:

```go
// exitOverlay returns to StateDefault, resetting overlay-specific fields.
func (m *Model) exitOverlay() {
    m.mode = StateDefault
    m.pendingSaveAction = PendingNone
    m.newFileName = ""
}

// resetForNewDocument resets all mutable editor state for a fresh document.
func (m *Model) resetForNewDocument(doc *document.Document, eval *implDoc.Evaluator, absPath, content string) {
    m.doc = doc
    m.eval = eval
    m.filepath = absPath
    // ... all fields reset ...
    m.transitionToReady()
    m.InvalidateAlignedCache()
}
```

Replaced all 35+ scattered `m.mode = ...` assignments with calls to centralized methods. Changed `deleteLine()` to call `fullReEvaluate()` (instead of `reEvaluate()`) so dependent expressions update after deletion. Wrapped `deleteLine()` in `handleDeleteLine()` that follows the layered mutation pattern:

```go
func (m Model) handleDeleteLine() (tea.Model, tea.Cmd) {
    m.undoManager.ForceBoundary()
    m.saveCurrentLine(true) // flush edit buffer first
    m.editBufLoaded = false

    lines := m.GetLines()
    if m.cursorLine >= len(lines) {
        return m, nil
    }
    deletedContent := lines[m.cursorLine]
    m.deleteLine()

    op := EditOperation{Type: OpDeleteLine, Line: m.cursorLine, OldText: deletedContent}
    undoCmd := m.recordEdit(op)
    m.undoManager.ForceBoundary()
    return m, undoCmd
}
```

Restored Ctrl+E as Export shortcut.

## Problem 2: Blank Lines Caused Result Drift in All Formatters

### Symptoms

When a calc block contained blank lines (e.g., `a = 1\n\nb = 2\n\nc = a + b`), inline result annotations (`--> value`) were appended to the wrong source lines. After blank lines, results shifted by the count of blanks above. The last statement's result could be lost entirely when the index overflowed.

### Root Cause

All formatters used the loop variable `i` from `for i, line := range sourceLines` to index into `results[i]`. But results are indexed per-AST-statement (one per non-blank calc line) while `sourceLines` includes blanks. With source `["a = 1", "", "b = 2"]` and results `[1, 2]`, at index 2 the code accessed `results[2]` (out of bounds) while `results[1]` was never displayed.

### Solution

Introduced a separate `resultIdx` counter that only advances for non-blank source lines. Applied identically to all four formatters:

```go
// Before: loop index i used for both source lines and results
for i, line := range sourceLines {
    if resultIdx < len(results) && results[i] != nil { // BUG: 'i' includes blank lines, results[] does not
        // ...
    }
}

// After: separate counter for results
resultIdx := 0
for _, line := range sourceLines {
    if line == "" { continue }
    if resultIdx < len(results) && results[resultIdx] != nil {
        fmt.Fprintf(w, " --> %s", display.Format(results[resultIdx]))
    }
    resultIdx++
}
```

**Enhancement (JSON formatter):** While fixing the indexing bug, the JSON formatter also gained structured per-statement results (`source`, `value`, `variable` fields), structured diagnostics with line/column positions, and rendered HTML for text blocks.

## Problem 3: Bracketed Paste Events Silently Dropped

### Symptoms

On terminals with bracketed paste (macOS Terminal.app, iTerm2), Cmd+V paste did nothing. The clipboard content was silently discarded. Additionally, the preview pane showed misaligned line counts when typing long lines that wrapped.

### Root Cause

1. **Bracketed paste:** Terminals intercept Cmd+V and send content as `tea.PasteMsg` instead of a key event. The editor's `Update` had no `case tea.PasteMsg:` handler.
2. **Edit buffer alignment:** `computeAlignedModelFresh` used committed document text, not the live `editBuf`. Long typed lines wrapped differently than committed text, causing source/preview pane disagreement.

### Solution

Added `tea.PasteMsg` handler that shares insertion logic with the existing Ctrl+V path:

```go
// In model.go Update():
case tea.PasteMsg:
    return m.handleBracketedPaste(msg.Content)

// In clipboard.go - shared insertion logic:
func (m Model) insertPastedText(text string) (tea.Model, tea.Cmd) {
    if len(text) > maxPasteSize {
        m.statusMsg = "Paste too large (>1MB)"
        return m, nil
    }
    // sanitize and insert...
}
```

For alignment, fed the live edit buffer into alignment computation and removed duplicate wrapping logic from `view_panes.go`:

```go
// Feed live editBuf into alignment model
if m.editBufLoaded {
    input.EditBuf = m.editBuf
    input.EditBufLine = m.cursorLine
}
```

## Prevention Strategies

### Code Review Checklist

These items encode the architectural patterns that prevent recurrence:

- [ ] New `m.mode = State*` outside `mode_transitions.go`? Must be moved to a transition method.
- [ ] New overlay or editor mode? Must have dedicated `enter*()` / `exit*()` methods in `mode_transitions.go`.
- [ ] New field on `Model`? Must be included in `resetForNewDocument()`.
- [ ] New destructive edit operation? Must follow `handleX()` / `x()` layered pattern with undo support.
- [ ] Reading `GetLines()` inside a mutation? Ensure `saveCurrentLine()` is called first to flush `editBuf`.
- [ ] New formatter or formatter change? Must have blank-line regression tests.
- [ ] Source lines and results iterated together? Must use separate `resultIdx` (or future `ForEachStatement()` helper).
- [ ] New `tea.Msg` handler? Check if the same action can arrive via other message types (key press, paste, mouse).
- [ ] View wrapping or line count? Must use `ComputeAlignedModel()`, not recompute locally.

### Test Coverage Added

| Area | Test | File |
|------|------|------|
| Mode transitions | Ctrl+E triggers export overlay | `file_operations_test.go` |
| Mode transitions | Ctrl+K deletes current line with undo | `delete_line_test.go` |
| Mode transitions | openFile resets state from StateEditing | `file_operations_test.go` |
| Mode transitions | Cmd+key shortcuts | `testdata/cmd_shortcuts` |
| Formatter indexing | Blank lines in calc block (markdown) | `markdown_formatter_test.go` |
| Formatter indexing | Per-statement results (JSON) | `json_formatter_test.go` |
| Alignment | EditBuf asymmetric widths | `aligned_test.go` |
| Alignment | EditBuf wraps in source pane | `aligned_test.go` |
| Alignment | Cursor wrap alignment (catwalk) | `testdata/preview_pane/cursor_wrap_alignment` |

### Remaining Test Gaps

- CI grep check for raw `m.mode =` assignments outside `mode_transitions.go`
- Round-trip enter/exit tests for all overlay modes
- Blank-line regression tests for HTML and text formatters
- Cross-formatter consistency assertions (all formatters agree on source/result pairs)
- Bracketed paste (`PasteMsg`) unit test (dispatch exists in `f71cd1c`, handler in `67c3cb0`)
- `sourcePreviewMatch` hard assertion in all catwalk debug output
- Extract shared `ForEachStatement()` helper for source/result iteration across formatters

## Related Documentation

- [bubble-tea-v2-migration-selection-undo-clipboard-fixes.md](bubble-tea-v2-migration-selection-undo-clipboard-fixes.md) - Predecessor: the v2 migration that surfaced mode transition and paste handling gaps
- [frontmatter-editing-keyboard-dispatch-fixes.md](frontmatter-editing-keyboard-dispatch-fixes.md) - Earlier keyboard dispatch routing fixes following the same pattern
- [ctrl-o-stale-state-and-unsaved-changes-detection.md](ctrl-o-stale-state-and-unsaved-changes-detection.md) - Same class of stale state bug in mode transitions
- [tui-editor-rendering-divider-status-bar-error-line.md](tui-editor-rendering-divider-status-bar-error-line.md) - Earlier source-to-output alignment correction
- [split-view-go-into-cohesive-modules.md](../code-organization/split-view-go-into-cohesive-modules.md) - Created the file structure that the alignment fix operates within
- [Design: Output Formatters](../../plans/2026-02-22-design-output-formatters.md) - Architecture that the formatter indexing fix aligns to
- [Plan: Bubble Tea v2 Upgrade](../../plans/2026-02-24-bubble-tea-v2-upgrade-plan.md) - Original plan motivating the v2 migration
