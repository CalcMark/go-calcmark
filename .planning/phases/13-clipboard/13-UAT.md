---
status: testing
phase: 13-clipboard
source: [13-01-SUMMARY.md, 13-02-SUMMARY.md, 13-03-SUMMARY.md]
started: 2026-02-09T04:20:00Z
updated: 2026-02-09T04:20:00Z
---

## Current Test

number: 2
name: Selection Highlighting Visible
expected: |
  1. Open cm, type some text
  2. Press Ctrl+A to select all
  3. Selected text should appear with gray background and white text
  4. Unselected areas should have normal appearance
awaiting: user response

## Tests

### 1. Ctrl+A Select All
expected: Press Ctrl+A and entire document text should be highlighted with gray background. Cursor moves to end of last line.
result: pass
note: Selection works. Minor issue - cursor visual position on wrapped lines shows at logical end, not visual end of wrapped continuation.

### 2. Selection Highlighting Visible
expected: Selected text appears with gray background and white text, distinguishing it from unselected text.
result: issue
reported: "Three issues: (1) Preview pane jump bug is back when cursor moves to last calc before empty line. (2) Selection highlighting loses dim/bright distinction - all text looks same when selected. (3) Wrapped line continuation is NOT highlighted even though selected."
severity: major

### 3. Arrow Keys Clear Selection
expected: After selecting text with Ctrl+A, pressing any arrow key (Up, Down, Left, Right) clears the selection (gray highlighting disappears).
result: [pending]

### 4. Home/End Keys Clear Selection
expected: After selecting text with Ctrl+A, pressing Home or End clears the selection.
result: [pending]

### 5. Typing Clears Selection
expected: After selecting text with Ctrl+A, typing any character clears the selection (highlighting disappears) and the character is inserted.
result: [pending]

### 6. Ctrl+C Copy with Selection
expected: Select text with Ctrl+A, then press Ctrl+C. Text should be copied to system clipboard (verify by pasting into another app like Terminal or Notes). Status bar should show "Copied to clipboard".
result: [pending]

### 7. Ctrl+C Quit without Selection
expected: Without any text selected (just cursor visible, no highlighting), press Ctrl+C. App should quit immediately (standard Unix interrupt behavior).
result: [pending]

### 8. Ctrl+X Cut
expected: Select text with Ctrl+A, then press Ctrl+X. Text should be removed from the document and copied to system clipboard. Status bar should show "Cut to clipboard".
result: [pending]

### 9. Ctrl+V Paste
expected: Copy text from another application to clipboard, then press Ctrl+V in cm. Text should be pasted at cursor position. Status bar should show "Pasted from clipboard".
result: [pending]

### 10. Cut Can Be Undone
expected: After cutting text with Ctrl+X, press Ctrl+Z. The cut text should be restored to the document.
result: [pending]

### 11. Paste Can Be Undone
expected: After pasting text with Ctrl+V, press Ctrl+Z. The pasted text should be removed.
result: [pending]

### 12. Multi-line Paste
expected: Copy multiple lines from another app, paste with Ctrl+V. All lines should appear correctly, splitting the current line if pasting in the middle.
result: [pending]

## Summary

total: 12
passed: 1
issues: 1
pending: 10
skipped: 0

## Gaps

- truth: "Ctrl+A selects entire document without crashing"
  status: fixed
  reason: "User reported: Caught panic: runtime error: slice bounds out of range [:48] with length 41"
  severity: blocker
  test: 1
  root_cause: "renderLineWithCursor used byte indexing with rune-based cursor position"
  artifacts:
    - path: "cmd/calcmark/tui/editor/view.go"
      issue: "content[col] used byte index, col is rune position"
  missing: []
  fix_commit: "7dbe80f"

- truth: "Preview pane content does not jump when cursor moves to last calc before empty line"
  status: failed
  reason: "User reported: Preview pane jump bug is back"
  severity: major
  test: 2
  root_cause: ""
  artifacts: []
  missing: []

- truth: "Selection highlighting preserves dim/bright line distinction"
  status: investigating
  reason: "User reported: Selection highlighting loses dim/bright distinction. Automated test shows cursor line has different styling (231 bright white on 232 dark) vs selected lines (255 on 240)."
  severity: major
  test: 2
  root_cause: "Test confirms distinct ANSI codes for cursor vs selected lines. Visual difference may be subtle or terminal-dependent."
  artifacts:
    - path: "cmd/calcmark/tui/editor/selection_highlighting_test.go"
      issue: "TestSelectionHighlighting_PreservesDimBrightDistinction shows different codes"
  missing: []

- truth: "Wrapped line continuations are fully highlighted when selected"
  status: investigating
  reason: "User reported: Wrapped line continuation is NOT highlighted. Automated test passes with ANSI256 colors showing 38;5;255;48;5;240 codes on wrapped content."
  severity: major
  test: 2
  root_cause: "Automated tests confirm highlighting code works correctly. User visual discrepancy may be terminal-specific."
  artifacts:
    - path: "cmd/calcmark/tui/editor/selection_highlighting_test.go"
      issue: "TestSelectionHighlighting_WrappedLineFullyHighlighted passes with ANSI256 profile"
  missing: []

- truth: "Paste preserves line numbers and styling"
  status: investigating
  reason: "User reported: Missing virtual line numbers and different text color after paste"
  severity: major
  test: 9
  root_cause: "Automated paste tests pass. User's screenshot shows severe view corruption - may be specific to actual clipboard paste or terminal."
  artifacts:
    - path: "cmd/calcmark/tui/editor/selection_highlighting_test.go"
      issue: "TestPaste_LargeDocumentWithScrolling passes - programmatic paste works correctly"
  missing: []
