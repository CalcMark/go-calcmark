---
status: complete
phase: 12-undo-redo
source: [12-01-SUMMARY.md, 12-02-SUMMARY.md, 12-03-SUMMARY.md]
started: 2026-02-08T18:10:00Z
updated: 2026-02-08T19:55:00Z
---

## Current Test

number: complete
name: All tests passed
expected: N/A
awaiting: N/A

## Tests

### 1. Basic Undo
expected: Type text, press Ctrl+Z, text is removed and cursor returns to original position
result: pass
notes: Fixed value/pointer receiver bug (783d9de), OpInsertLine for Enter (7adb42e), UTF-8 compliance (d490391)

### 2. Basic Redo
expected: After undoing, press Ctrl+Y, the undone text reappears at the same position
result: pass
notes: Also verified repeated Ctrl+Y doesn't panic (bounds checking added in 7b55e35)

### 3. Rapid Typing Groups Together
expected: Type "hello" quickly (without pausing), press Ctrl+Z once, all 5 characters are removed at once (not one at a time)
result: pass
notes: Timer-based grouping (1s) works predictably - slow typing creates smaller groups

### 4. Enter Creates Boundary
expected: Type "line1", press Enter, type "line2", press Ctrl+Z - only "line2" is removed, "line1" and the new line remain
result: pass

### 5. Navigation Creates Boundary
expected: Type "abc", press arrow key to move, type "xyz", press Ctrl+Z - only "xyz" is removed, "abc" remains
result: pass

### 6. Cursor Position Restored
expected: Navigate to middle of document, type text, press Ctrl+Z - cursor returns to exact position before typing
result: pass
notes: Cursor restored to pre-edit position. Text removal is expected (standard undo model).

### 7. Redo Cleared on New Edit
expected: Type "a", Ctrl+Z to undo, type "b", Ctrl+Y to redo - nothing happens (redo stack cleared by new edit)
result: pass
notes: Linear undo model confirmed. Typing after undo clears redo stack (standard behavior).

### 8. Multiple Undos
expected: Type text, pause 1+ second, type more text, Ctrl+Z undoes second batch, Ctrl+Z again undoes first batch
result: pass
notes: Timer-based grouping (1s) creates proper boundaries between typing sessions.

## Summary

total: 8
passed: 8
issues: 0
pending: 0
skipped: 0

## Gaps

- truth: "Ctrl+Z undoes the last edit and removes typed text"
  status: fix_applied
  reason: "User reported: Nothing happened when I pressed Ctrl-z. Then 'hellohello' duplication bug."
  severity: major
  test: 1
  root_cause: |
    1. Value receiver bug (commit 783d9de) - fixed
    2. Missing line operation types - Enter created new line but undo only handled single-line ops,
       causing content duplication when undoing (commit 7adb42e)
  fix: |
    1. Inlined performUndo/performRedo logic directly into handleUndo/handleRedo
    2. Added OpInsertLine/OpDeleteLine operation types for proper Enter undo
  artifacts: [cmd/calcmark/tui/editor/model.go, cmd/calcmark/tui/editor/undo.go, cmd/calcmark/tui/editor/model_test.go, cmd/calcmark/tui/editor/undo_test.go]
  missing: []
  debug_session: ""

- truth: "Option+arrow keys work after undo/redo operations"
  status: fix_applied
  reason: "User reported app became unresponsive when using Option+arrow keys to navigate"
  severity: major
  test: 6
  root_cause: |
    handleCtrlLeftKey and handleCtrlRightKey accessed runes[col-1] without bounds checking.
    After undo/redo, cursorCol could exceed line length, causing panic.
  fix: |
    Added bounds clamping in handleCtrlLeftKey and handleCtrlRightKey (commit a525c37)
  artifacts: [cmd/calcmark/tui/editor/model.go]
  missing: []
  debug_session: ""
