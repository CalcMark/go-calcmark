---
title: "Ctrl+O file open appends to in-memory document instead of replacing it"
date: 2026-02-22
category: ui-bugs
tags:
  - tui-editor
  - bubble-tea
  - file-open
  - state-reset
  - debounce
  - change-tracking
  - unsaved-changes
  - elm-architecture
components:
  - cmd/calcmark/tui/editor
severity: medium
symptoms:
  - Opening a file with Ctrl+O appends the opened document to whatever text was already in the editor buffer
  - Typed text persists in memory across file open operations
  - Unsaved changes are not detected after the debounce timer fires, allowing data loss without warning
  - The unsaved-changes prompt does not appear before opening a new file
  - Stale mutable state (pendingKey, yankBuffer, searchTerm) leaks across file open boundaries
root_causes:
  - openFile() did not re-initialize the editor pane; it loaded content into existing mutable state instead of resetting it
  - transitionToProcessing() committed the edit buffer to the document via updateCurrentLine() but never set m.modified to true, breaking the change-tracking fast-path
  - hasUnsavedChanges() relied on m.modified as a fast-path, but the debounce handler bypassed this flag
  - No unsaved-changes guard existed on the Ctrl+O path (only Ctrl+Q had one)
  - Unsaved-changes guard logic was duplicated across Ctrl+Q, Ctrl+O, and the command menu
---

## Problem Statement

The TUI editor's Ctrl+O (open file) command failed to reset the editor to a clean state before loading a new document. Instead of replacing the buffer contents, it appended the opened file's content to whatever text was already in memory, producing a corrupted combined document. The open-file path also lacked the unsaved-changes guard that Ctrl+Q (quit) had, meaning users received no warning about losing edits.

A deeper issue compounded the problem: the debounce timer's `transitionToProcessing()` function committed the in-flight edit buffer (`editBuf`) to the document model via `updateCurrentLine()`, but never set `m.modified = true`. Because `hasUnsavedChanges()` used `m.modified` as a fast-path exit, changes committed by the debounce timer were invisible to the change-tracking system.

## Investigation Steps

1. **Symptom identification**: Opening a file with Ctrl+O left previously typed text visible and appended the new document to the old one. This pointed to incomplete state reset during the file-open operation.

2. **State audit of openFile()**: The function only partially reset editor state. Fields like `editBuf`, `pendingKey`, `yankBuffer`, `searchTerm`, `searchMatches`, `searchIdx`, `autocompleteState`, and `pendingSaveAction` carried values from the previous document.

3. **Unsaved changes flow analysis**: The Ctrl+O path had no unsaved-changes guard. Users could open a new file and silently lose all edits.

4. **hasUnsavedChanges() logic audit**: After adding a `m.modified` fast-path, the function short-circuited to false even when the document had been changed through the debounce/processing pipeline.

5. **Bubble Tea Cmd processing discovery**: The debounce timer fires `transitionToProcessing()`, which calls `updateCurrentLine(m.editBuf)` to commit text to the document. This never set `m.modified = true`, so the fast-path returned false.

6. **Test methodology gap**: Unit tests calling `Update()` directly never process the returned `Cmd`s, so the debounce timer `Cmd` was silently dropped. Only catwalk data-driven tests (which process Cmds between run blocks) exposed the timing-dependent bug.

## Root Cause Analysis

### Cause 1: Incomplete state reset in openFile()

When Ctrl+O loaded a new file, mutable fields (`editBuf`, selection state, undo history, search state, autocomplete state, changed-block tracking) all persisted from the previous document session.

### Cause 2: Missing m.modified = true in transitionToProcessing()

In Bubble Tea's Elm architecture, keystrokes do not immediately commit text to the document model. A debounce timer `Cmd` fires later, and `transitionToProcessing()` commits the `editBuf`. This function called `updateCurrentLine(m.editBuf)` but never set `m.modified = true`. Because `hasUnsavedChanges()` short-circuits on `!m.modified`, the editor believed the document was unmodified.

## Solution

### 1. Full state reset in openFile() (`file_operations.go`)

Every mutable field is explicitly zeroed when opening a new file:

```go
// Editing state
m.editBuf = ""
m.userIsTyping = false
m.frontmatterErr = nil
m.changedBlockIDs = make(map[string]bool)
m.selectionAnchorLine = -1
m.selectionAnchorCol = -1
m.pendingKey = 0
m.yankBuffer = ""

// Search state
m.searchTerm = ""
m.searchMatches = nil
m.searchIdx = 0

// Overlay / prompt state
m.autocompleteState = components.AutosuggestState{}
m.pendingSaveAction = PendingNone

// Undo history
m.undoManager.Clear()
```

### 2. Extracted promptSaveIfNeeded() helper (`save_prompt_handler.go`)

A reusable helper centralizes the unsaved-changes guard:

```go
func (m *Model) promptSaveIfNeeded(action PendingAction, promptMsg string) bool {
    if !m.hasUnsavedChanges() {
        return false
    }
    m.pendingSaveAction = action
    m.mode = StateSavePrompt
    m.statusMsg = promptMsg
    return true
}
```

Used by Ctrl+Q, Ctrl+O, and command menu equivalents.

### 3. Improved hasUnsavedChanges() (`file_operations.go`)

Checks for in-flight typing that has not yet been committed:

```go
func (m *Model) hasUnsavedChanges() bool {
    if m.userIsTyping {
        lines := m.GetLines()
        if m.cursorLine < len(lines) && m.editBuf != lines[m.cursorLine] {
            return true
        }
    }
    if !m.modified {
        return false
    }
    currentContent := m.getDocumentContent()
    return currentContent != m.savedContent
}
```

### 4. transitionToProcessing() sets m.modified (`state.go`)

The single most critical one-line fix:

```go
func (m *Model) transitionToProcessing() {
    m.userIsTyping = false
    m.state = StateProcessing
    m.updateCurrentLine(m.editBuf)
    m.modified = true  // Mark document as modified after committing editBuf
    m.redetectBlockTypes()
    m.reEvaluate()
    m.transitionToReady()
}
```

### 5. Unified command paths

Ctrl+Q, Ctrl+O, and command menu Open/Quit all use `promptSaveIfNeeded()`.

### 6. Cleanup

Removed vestigial `m.quitting = false` from file picker ESC. Moved save prompt logic to dedicated `save_prompt_handler.go`.

## Key Insight: Bubble Tea Cmd Processing in Tests

Standard Go unit tests that call `Update()` directly never execute the returned `Cmd`s. The debounce timer `Cmd` is silently discarded, so `transitionToProcessing()` is never exercised in unit tests. The missing `m.modified = true` was invisible.

Catwalk data-driven tests faithfully process `Cmd`s between run blocks, matching the real Bubble Tea runtime. This is why the bug only surfaced in catwalk tests.

**General principle**: Any state transition that depends on `Cmd` execution (timers, I/O callbacks, debounced actions) cannot be reliably tested by calling `Update()` alone. A test harness that processes the full `Cmd` pipeline is essential.

## Prevention Strategies

### Ensuring new Model fields are reset in openFile()

- **Compile-time exhaustiveness check**: Write a test using `reflect.VisibleFields(reflect.TypeOf(Model{}))` that asserts each field appears in an explicit allow-list (either "reset in openFile" or "preserved across openFile"). New fields that aren't listed fail the test.
- **Consider fresh-Model construction**: Instead of resetting fields individually, construct a fresh Model and carry forward only configuration fields. This inverts the problem from "remember to zero every new field" to "remember to carry forward config fields."

### Ensuring m.modified is set consistently

- **Funnel all document mutations through a single method**: A `commitToDocument()` method that always sets `m.modified = true` would eliminate the class of "forgot to set modified" bugs.
- **Consider derived state**: Compute `hasUnsavedChanges()` by comparing current effective source (document + uncommitted editBuf) against a save-point snapshot, eliminating the manual flag entirely.

### Catching Cmd-processing differences

- **Unit tests**: Test pure logic with no Cmd side effects.
- **Catwalk tests**: Test all interaction flows involving Cmd processing, debouncing, or multi-step state transitions.
- **Document this boundary** in TESTING.md so contributors know which tool to use.

### Checklist for adding new Model fields

1. Declare the field with a clear comment
2. Classify as MUTABLE (per-file) or CONFIG (lifetime)
3. If MUTABLE: add reset logic to openFile() and field coverage test
4. If it affects document content: ensure mutations go through a centralized method
5. If it interacts with Cmds: write the test as a catwalk test, not a unit test
6. Update documentation if the field introduces a new concept

## Files Changed

- `cmd/calcmark/tui/editor/file_operations.go` -- openFile resets, hasUnsavedChanges improvements
- `cmd/calcmark/tui/editor/save_prompt_handler.go` -- NEW: save prompt logic
- `cmd/calcmark/tui/editor/globals_handler.go` -- stripped to globals panel only
- `cmd/calcmark/tui/editor/key_dispatch.go` -- uses promptSaveIfNeeded()
- `cmd/calcmark/tui/editor/command_menu.go` -- uses promptSaveIfNeeded()
- `cmd/calcmark/tui/editor/file_picker_handler.go` -- removed vestigial m.quitting
- `cmd/calcmark/tui/editor/state.go` -- transitionToProcessing sets m.modified
- `cmd/calcmark/tui/editor/model.go` -- PendingAction type
- `cmd/calcmark/tui/editor/model_test.go` -- expanded tests
- `cmd/calcmark/tui/editor/testdata/open_unsaved_prompt` -- NEW: catwalk test

## Related Documentation

- [Frontmatter editing and keyboard dispatch fixes](frontmatter-editing-keyboard-dispatch-fixes.md)
- [TUI editor rendering, divider, status bar fixes](tui-editor-rendering-divider-status-bar-error-line.md)
- [TUI Editor TESTING.md](../../../cmd/calcmark/tui/editor/TESTING.md)
