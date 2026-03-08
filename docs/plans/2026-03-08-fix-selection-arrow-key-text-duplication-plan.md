---
title: "fix: Selection plus arrow key duplicates text"
type: fix
status: completed
date: 2026-03-08
issue: 35
deepened: 2026-03-08
---

# fix: Selection plus arrow key duplicates text

## Enhancement Summary

**Deepened on:** 2026-03-08
**Sections enhanced:** 6
**Research sources:** Source code analysis, existing catwalk tests, institutional learnings (6 solution docs)

### Key Improvements
1. Verified existing catwalk tests contain wrong expected values — confirms the bug is baked into test expectations
2. Discovered wider scope: Home/End/PgUp/PgDown after Ctrl+A also destroy document content (`totalSource=1`)
3. Precise `collapseSelectionTo` implementation spec with same-line vs cross-line optimization
4. Identified `saveCurrentLine(true)` as the correct save path (triggers `redetectBlockTypes` + `reEvaluate`)

### New Considerations Discovered
- Existing `testdata/selection` expectations for Home/End/PgUp/PgDown show `totalSource=1` after Ctrl+A — document destroyed, not just duplicated
- `SelectAll()` sets cursor to end of last line but does NOT set `editBufLoaded=true` or load `editBuf` — so editBuf is stale/empty after Ctrl+A
- The `saveCurrentLineAndMoveTo` function unconditionally writes `m.editBuf` to the document (line 445), even when editBuf is empty/stale

## Overview

When text is selected in the TUI editor and the user presses an unmodified arrow key, three things go wrong: the selection remains visible, the cursor doesn't move to the expected position, and text from one line gets duplicated onto another. This violates the editor's core invariant: **never adjust text the user didn't explicitly modify**.

## Problem Statement

**Repro** (from issue #35):
1. Open a new document, type `a = 1`, press Enter (cursor on empty line 2).
2. Press Cmd+Left to select `a = 1`.
3. Press Left Arrow.

**Expected**: Selection clears, cursor collapses to the start of the selection.
**Observed**: Selection persists, cursor doesn't move, line 1 text is duplicated on line 2.

## Root Cause

In `cmd/calcmark/tui/editor/navigation.go`, all four arrow-key handlers call `prepareNavigation(false)` which:

1. **Clears selection first** via `ClearSelection()` — destroying the selection boundaries before the handler can read them.
2. **Calls `loadCurrentLineIntoEditBuffer()`** — which may reload stale content if `editBufLoaded` is false.
3. Then the handler moves the cursor normally (one position) instead of collapsing to the selection boundary.

When `saveCurrentLineAndMoveTo()` is subsequently called (e.g., wrapping to previous line on Left at col 0), it writes the stale `editBuf` to the document via `updateCurrentLine()`, duplicating text.

### Research Insights: Exact Destruction Mechanism

Tracing through the Ctrl+A → Left sequence step by step:

1. `SelectAll()` sets `selectionAnchorLine=0, selectionAnchorCol=0, cursorLine=3, cursorCol=6`. It does **not** touch `editBuf` or `editBufLoaded`.
2. `handleLeftKey()` calls `prepareNavigation(false)`.
3. `prepareNavigation` calls `ClearSelection()` — selection range is now lost.
4. `prepareNavigation` calls `loadCurrentLineIntoEditBuffer()`. Since `editBufLoaded` may be false (Ctrl+A didn't set it), this loads `lines[3]` = `"z = 30"` into `editBuf`.
5. `handleLeftKey` checks `cursorCol > 0` (6 > 0 = true), decrements to 5. Returns.
6. The cursor is now at (3, 5) instead of (0, 0). **Wrong position.**

For the text duplication path (issue repro with 2 lines):
1. User types `a = 1`, presses Enter. Cursor at line 1, col 0. `editBuf=""`, `editBufLoaded=true` (empty line was loaded).
2. Cmd+Left creates selection. Cursor moves to line 0.
3. Left Arrow: `prepareNavigation` calls `loadCurrentLineIntoEditBuffer()`. If `editBufLoaded=true` with stale content from line 1, it's a no-op. Then `saveCurrentLineAndMoveTo(cursorLine-1)` writes the stale empty `editBuf` to line 0, **destroying "a = 1"**.

### Key Code Paths

- `handleLeftKey()` — `navigation.go:44-53`
- `handleRightKey()` — `navigation.go:55-64`
- `handleUpKey()` — `navigation.go:28-33`
- `handleDownKey()` — `navigation.go:36-41`
- `prepareNavigation()` — `navigation.go:18-26`
- `ClearSelection()` — `selection.go:38-41`
- `GetSelectionRange()` — `selection.go:46-64`
- `loadCurrentLineIntoEditBuffer()` — `editing.go:402-411`
- `saveCurrentLineAndMoveTo()` — `editing.go:443-475` — unconditionally writes `editBuf` to document at line 445
- `saveCurrentLine(bool)` — `editing.go:415-439` — conditional save with `redetectBlockTypes` + `reEvaluate`

## Proposed Solution

Add a selection-collapse check at the top of each arrow handler, **before** calling `prepareNavigation`. Standard code-editor behavior (VS Code, Sublime):

| Key | With Selection | Cursor Collapses To |
|-----|---------------|---------------------|
| Left | Yes | Selection **start** (min position) |
| Right | Yes | Selection **end** (max position) |
| Up | Yes | Selection **start** (min position) |
| Down | Yes | Selection **end** (max position) |

When no selection exists, behavior is unchanged.

**Design decision:** Up/Down collapse only (no additional movement after collapse). This matches VS Code/Sublime behavior. Some editors (macOS TextEdit) move an additional line — we explicitly choose not to.

### Implementation Pattern

For each of the four handlers, insert this early-return before `prepareNavigation`:

```go
func (m Model) handleLeftKey() (tea.Model, tea.Cmd) {
    if m.HasSelection() {
        startLine, startCol, _, _ := m.GetSelectionRange()
        m.collapseSelectionTo(startLine, startCol)
        return m, nil
    }
    m.prepareNavigation(false)
    // ... existing movement logic unchanged
}
```

### New helper: `collapseSelectionTo(line, col int)` in `selection.go`

Centralizes the collapse logic to avoid duplication across four handlers. Place in `selection.go` since it's selection lifecycle management:

```go
// collapseSelectionTo clears the active selection and moves the cursor to the
// given position. Used by unmodified arrow keys to collapse selection to a
// boundary without mutating document content.
func (m *Model) collapseSelectionTo(line, col int) {
    m.undoManager.ForceBoundary()

    // Flush dirty edit buffer for the CURRENT line before moving.
    // Use saveCurrentLine which triggers redetect + reEvaluate.
    if m.editBufLoaded {
        m.saveCurrentLine(true)
    }

    m.cursorLine = line
    m.cursorCol = col
    m.ClearSelection()
    m.editBufLoaded = false // Lazy reload on next interaction
    m.adjustScrollForCursor()
}
```

### Research Insights: Why `saveCurrentLine(true)` Not `updateCurrentLine`

From `docs/solutions/ui-bugs/tui-mode-transitions-formatter-indexing-and-bracketed-paste-fixes.md`: edit buffer flushing must trigger `redetectBlockTypes()` and `reEvaluate()` to keep the preview pane in sync. Using `saveCurrentLine(true)` (editing.go:415-439) handles this correctly. Using bare `updateCurrentLine` would skip re-evaluation, causing stale preview content.

However, `saveCurrentLine(true)` also sets `m.modified = true`. Since collapse is a navigation-only action (no text mutation), this is slightly misleading but harmless — the content is unchanged so the flag is a no-op if already set. If `editBufLoaded` is true but `editBuf` matches the document line, the `updateCurrentLine` inside `saveCurrentLine` is effectively a no-op.

### Same-Line Optimization

When the collapse target is on the **same line** as the cursor:
- No need to save/reload — `editBuf` already has the correct line content
- Just update `cursorCol`, clear selection, and return

```go
func (m *Model) collapseSelectionTo(line, col int) {
    m.undoManager.ForceBoundary()

    if line != m.cursorLine {
        // Cross-line: flush dirty buffer, lazy reload
        if m.editBufLoaded {
            m.saveCurrentLine(true)
        }
        m.editBufLoaded = false
    }

    m.cursorLine = line
    m.cursorCol = col
    m.ClearSelection()
    m.adjustScrollForCursor()
}
```

## Acceptance Criteria

- [ ] Left Arrow with active selection collapses cursor to selection start, clears selection, no text mutation
- [ ] Right Arrow with active selection collapses cursor to selection end, clears selection, no text mutation
- [ ] Up Arrow with active selection collapses cursor to selection start, clears selection, no text mutation
- [ ] Down Arrow with active selection collapses cursor to selection end, clears selection, no text mutation
- [ ] Works with forward selections (anchor before cursor)
- [ ] Works with reverse selections (anchor after cursor)
- [ ] Works with single-line selections
- [ ] Works with multi-line selections
- [ ] Works with Ctrl+A (select all) then arrow key
- [ ] Works at document boundaries (first line, last line, col 0, end of line)
- [ ] Document content is never mutated by collapse (no duplication) — verified via `lines` observer showing all 4 original lines intact
- [ ] Undo boundary is forced before collapse
- [ ] Scroll adjusts if collapsed position is off-screen
- [ ] Degenerate selection (Shift+Right then Shift+Left = no selection) falls through to normal nav
- [ ] All existing catwalk tests pass (updated expectations where needed)
- [ ] `task test` and `task quality` pass

## Technical Considerations

### editBuf Lifecycle During Collapse

When collapse moves the cursor to a **different line**:
- Save current line via `saveCurrentLine(true)` if `editBufLoaded` is true (flush dirty buffer with redetect/reeval)
- Set `editBufLoaded = false` (next operation loads fresh from document)
- Do NOT call `saveCurrentLineAndMoveTo()` — that unconditionally writes editBuf to the document (line 445), which is the root cause of text duplication

When collapse stays on the **same line**:
- `editBufLoaded` can remain true, `editBuf` is already correct for this line
- Just update `cursorCol`

### Scope: Plain Arrow Keys + Follow-Up for Others

Home/End/PageUp/PageDown/Ctrl+Arrow also call `prepareNavigation(false)` and suffer the same stale-editBuf bug. **Evidence from existing tests**: after Ctrl+A, Home/End/PgUp/PgDown all show `totalSource=1` (document destroyed from 4 lines to 1). This is the same root cause but manifests differently.

Standard editor behavior for those keys: clear selection and navigate normally (NOT collapse to boundary). The fix for those keys is different — they need `prepareNavigation` to handle selection clearing without corrupting editBuf. File as a follow-up issue after this fix lands.

### Undo Boundary

The collapse path bypasses `prepareNavigation`, so it must explicitly call `m.undoManager.ForceBoundary()` to separate undo groups. This matches the pattern in `prepareNavigation` (line 24).

### Institutional Learning: Selection Rendering

From `docs/solutions/ui-bugs/bubble-tea-v2-migration-selection-undo-clipboard-fixes.md`: selection rendering was previously buggy because it operated on pre-styled ANSI text, corrupting column arithmetic. The current rendering pipeline works on raw text with explicit tint colors. The collapse fix does not touch rendering — it only changes cursor position and selection state — so no rendering regressions are expected.

### Institutional Learning: Edit Buffer Ordering

From `docs/solutions/ui-bugs/frontmatter-editing-keyboard-dispatch-fixes.md`: `updateCurrentLine()` must be called before navigation to persist edits. Our `collapseSelectionTo` respects this by calling `saveCurrentLine(true)` before changing `cursorLine`. The ordering is: save → move → clear → adjust scroll.

## MVP

### Test: `cmd/calcmark/tui/editor/testdata/selection_collapse`

New catwalk test file reproducing the exact issue #35 scenario and all selection-collapse permutations. The initial document is `# Header\nx = 10\ny = 20\nz = 30` (4 lines, indices 0-3).

Key test cases:
1. **Ctrl+A → Left**: cursor should be at (0, 0), all 4 lines intact (`totalSource=4`)
2. **Ctrl+A → Right**: cursor should be at (3, 6), all 4 lines intact
3. **Ctrl+A → Up**: cursor should be at (0, 0), all 4 lines intact
4. **Ctrl+A → Down**: cursor should be at (3, 6), all 4 lines intact
5. **Shift+Down×2 → Left**: forward multi-line selection, collapse to start
6. **Shift+Up×2 → Right**: reverse multi-line selection, collapse to end
7. **Shift+Right×3 → Left**: single-line forward selection, collapse to start
8. **Shift+Left×3 → Right**: single-line reverse selection, collapse to end
9. **Shift+Right → Shift+Left → Left**: degenerate selection (anchor=cursor), normal nav
10. **Content integrity**: `lines` observer after every collapse shows unmodified document

### Fix: `cmd/calcmark/tui/editor/navigation.go` + `selection.go`

1. Add `collapseSelectionTo(line, col int)` to `selection.go`
2. Add early-return `HasSelection()` checks in `handleLeftKey`, `handleRightKey`, `handleUpKey`, `handleDownKey` in `navigation.go`

### Update: `cmd/calcmark/tui/editor/testdata/selection`

Existing catwalk expectations for arrow-key-after-Ctrl+A have wrong expected values baked in. Current wrong expectations vs corrected:

| Test | Current Expected | Corrected Expected |
|------|-----------------|-------------------|
| Ctrl+A → Left | `cursorLine=3 cursorCol=5 editBuf="z = 30"` | `cursorLine=0 cursorCol=0 totalSource=4` |
| Ctrl+A → Right | `cursorLine=3 cursorCol=6 editBuf="z = 30"` | `cursorLine=3 cursorCol=6 totalSource=4` |
| Ctrl+A → Up | `cursorLine=2 cursorCol=6 editBuf="y = 20"` | `cursorLine=0 cursorCol=0 totalSource=4` |
| Ctrl+A → Down | `cursorLine=3 cursorCol=6 editBuf="y = 20"` | `cursorLine=3 cursorCol=6 totalSource=4` |

Regenerate with:

```bash
go test ./cmd/calcmark/tui/editor -run Catwalk -v -args -rewrite
```

**Warning**: The Home/End/PgUp/PgDown tests will also be corrected by `-rewrite` but their behavior depends on fixing `prepareNavigation` for those handlers too. If those are not fixed in this PR, their expectations will still be wrong. Consider running `-rewrite` only for the `selection_collapse` test file initially.

## Dependencies & Risks

- **Low risk**: The fix adds an early-return path that only activates when `HasSelection()` is true. All no-selection behavior is unchanged.
- **Dependency**: `GetSelectionRange()` must return correct start/end regardless of selection direction (forward vs reverse). Verified: it normalizes via comparison at `selection.go:58` (`anchorLine < curLine || (anchorLine == curLine && anchorCol <= curCol)`).
- **Risk**: Updating existing catwalk expectations may reveal other latent bugs (especially Home/End/PgUp/PgDown document destruction). Treat each as a separate investigation.
- **Follow-up**: File a new issue for Home/End/PgUp/PgDown/Ctrl+Arrow document destruction after Ctrl+A.

## References

- Issue: [#35](https://github.com/CalcMark/go-calcmark/issues/35)
- Related solution: `docs/solutions/ui-bugs/bubble-tea-v2-migration-selection-undo-clipboard-fixes.md`
- Related solution: `docs/solutions/ui-bugs/tui-mode-transitions-formatter-indexing-and-bracketed-paste-fixes.md`
- Related solution: `docs/solutions/ui-bugs/frontmatter-editing-keyboard-dispatch-fixes.md`
- Testing guide: `cmd/calcmark/tui/editor/TESTING.md`
- Key files: `navigation.go`, `selection.go`, `editing.go`, `state.go`
