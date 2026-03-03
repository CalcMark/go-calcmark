---
title: "fix: Autocomplete acceptance not recorded on undo stack"
type: fix
status: completed
date: 2026-03-03
issue: https://github.com/CalcMark/go-calcmark/issues/15
---

# fix: Autocomplete acceptance not recorded on undo stack

## Overview

When a user accepts an autosuggest via TAB, the text replacement (prefix -> completion) is not recorded on the undo stack. Pressing Cmd+Z/Ctrl+Z after accepting an autocomplete corrupts the buffer instead of reverting the completion.

## Problem Statement

`acceptAutocomplete()` in `cmd/calcmark/tui/editor/model.go:1004-1038` directly mutates `editBuf` via string slicing without:
1. Capturing before-state for undo
2. Calling `ForceBoundary()` to isolate the operation
3. Recording an `EditOperation` via `AddOperation()`

Every other editing operation (typing, paste, backspace, delete, enter) properly records operations. Autocomplete acceptance is the only mutation path that bypasses the undo system.

**Repro** (from issue #15):
1. Open empty cm editor
2. Type `2 cm` — observe autosuggest for centimeters
3. Press TAB to accept — text updates to `2 centimeter`
4. Press Cmd+Z to undo
5. **Expected**: `centimeter` disappears, second Cmd+Z restores `cm`
6. **Observed**: Buffer shows `2 cntimeter` — the `e` (where `m` was) is deleted instead

## Proposed Solution

Follow the paste handler pattern from `clipboard.go:100-141`:

1. Capture before-state (`cursorLine`, `cursorCol`, `scrollOffset`)
2. Call `m.undoManager.ForceBoundary()` before the edit (commits any pending typing batch)
3. Perform the text replacement (unchanged from current code)
4. Record `OpReplace` with the exact field mapping below
5. Set `m.modified = true`
6. Call `m.undoManager.ForceBoundary()` after the edit (commits the autocomplete batch)

### EditOperation Field Mapping

```
Op.Type         = OpReplace
Op.Line         = m.cursorLine        // line where edit occurs
Op.Col          = prefixStart          // rune position where replacement STARTS
Op.OldText      = prefix               // typed prefix being replaced (e.g., "cm")
Op.NewText      = insertText           // completion text incl. "(" for functions
Op.CursorLine   = m.cursorLine        // for undo cursor restore
Op.CursorCol    = m.cursorCol         // cursor pos BEFORE edit (end of prefix)
Op.ScrollOffset = m.scrollOffset      // for undo scroll restore
```

**Critical distinction**: `Op.Col` = `prefixStart` (where the replacement begins), NOT `m.cursorCol` (cursor at end of prefix). The undo system uses `Op.Col` to locate the text for deletion/insertion, while `CursorCol` is only used for cursor restoration.

### Implementation Choices

| Decision | Choice | Rationale |
|---|---|---|
| Use `AddOperation` vs `recordEdit` | `AddOperation` directly | `ForceBoundary()` already commits the batch; `recordEdit`'s grouping timer is unnecessary |
| Set `m.modified` | Explicitly `= true` | Matches paste handler; ensures save indicator updates before debounce fires |
| Call `reEvaluate()` | No — rely on debounce | Current behavior; delay is ~150ms and acceptable |

## Technical Considerations

### Undo/Redo Correctness

The `OpReplace` undo/redo paths in `undo_operations.go` handle this correctly:

- **Undo** (line 242): Deletes `runeLen(NewText)` chars at `Col`, inserts `OldText` at `Col`
- **Redo** (line 318): Deletes `runeLen(OldText)` chars at `Col`, inserts `NewText` at `Col`
- **Redo cursor** (line 129): Places cursor at `Col + runeLen(NewText)` — correct end-of-completion position

### Undo Grouping Interaction

When the user types "cm" then TABs to accept "centimeter":

1. Typing "c" and "m" create `OpInsert` operations in the current undo batch
2. First `ForceBoundary()` commits batch 1: `[OpInsert("c"), OpInsert("m")]`
3. `OpReplace(OldText="cm", NewText="centimeter")` is recorded
4. Second `ForceBoundary()` commits batch 2: `[OpReplace]`

Result: Ctrl+Z undoes the autocomplete (restores "cm"), second Ctrl+Z undoes the typing (removes "cm"). This matches user expectations.

### Institutional Learnings Applied

From `docs/solutions/ui-bugs/bubble-tea-v2-migration-selection-undo-clipboard-fixes.md`:
- Value receiver calling `m.undoManager.ForceBoundary()` is safe — `undoManager` is a pointer field, mutations persist through the returned model copy
- The paste handler already proves this pattern works with the Bubble Tea v2 architecture

## Acceptance Criteria

### Core Requirements
- [x]`acceptAutocomplete()` records `OpReplace` on the undo stack (`model.go`)
- [x]Ctrl+Z after autocomplete accept restores the typed prefix
- [x]Ctrl+Y after undo re-applies the completion
- [x]Function completions (with appended "(") undo completely including the paren
- [x]Prior typing commits as a separate undo batch before the autocomplete

### Edge Cases
- [x]Prefix at start of line (col 0) — `prefixStart` = 0
- [x]Multiple consecutive autocomplete acceptances undo in reverse order
- [x]Redo cursor positions at end of completion text

### Testing (TDD per CLAUDE.md)
- [x]New catwalk test `testdata/autocomplete_undo` — reproduces bug first (fails), then validates fix (passes)
- [x]Test: type prefix -> TAB accept -> verify text -> Ctrl+Z -> verify prefix restored
- [x]Test: accept -> Ctrl+Z -> Ctrl+Y -> verify redo
- [x]Test: accept function (with "(") -> Ctrl+Z -> verify prefix without paren
- [x]Test: type -> accept -> type more -> Ctrl+Z sequence (grouped correctly)
- [x]`task test` passes with zero regressions
- [x]`task quality` passes

## Files to Modify

| File | Change |
|---|---|
| `cmd/calcmark/tui/editor/model.go` | Add undo recording to `acceptAutocomplete()` (~15 lines) |
| `cmd/calcmark/tui/editor/testdata/autocomplete_undo` | New catwalk test file |
| `cmd/calcmark/tui/editor/catwalk_test.go` | Register dedicated test function for `autocomplete_undo` |

## References

- Issue: [#15](https://github.com/CalcMark/go-calcmark/issues/15)
- Paste handler pattern: `cmd/calcmark/tui/editor/clipboard.go:100-141`
- Undo system: `cmd/calcmark/tui/editor/undo.go`, `undo_operations.go`
- Existing autocomplete tests: `cmd/calcmark/tui/editor/testdata/autocomplete`
- Catwalk testing docs: `cmd/calcmark/tui/editor/TESTING.md`
- Learnings: `docs/solutions/ui-bugs/bubble-tea-v2-migration-selection-undo-clipboard-fixes.md`
