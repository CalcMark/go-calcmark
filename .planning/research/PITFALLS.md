# Pitfalls Research

**Domain:** Editor undo/redo, unit conversion correctness, file operations
**Researched:** 2026-02-06
**Milestone:** v1.1 CalcMark Language (interpreter correctness + editor completion)
**Confidence:** HIGH (codebase analysis + verified ecosystem patterns)

---

## Critical Pitfalls

Mistakes that cause data loss, incorrect results, or require rewrites.

### Pitfall 1: Unit Context Lost During Type Transformations

**What goes wrong:** `accumulate(5mb/s, 1 day) as napkin` returns "430K" instead of ~400GB. The napkin formatter receives a Quantity but loses the unit context when rounding/formatting.

**Why it happens:** Looking at the codebase flow:
1. `evalAccumulate()` correctly returns a `*types.Quantity` with `Value` and `Unit` (see `/impl/interpreter/rate_functions.go:38-41`)
2. `evalNapkinConversion()` in napkin_eval.go extracts `numValue = v.Value` from Quantity (line 29)
3. **BUG:** It then returns `types.NewNumber(decimal.NewFromFloat(roundedFloat))` (line 99) — a plain Number with NO UNIT

The napkin formatter explicitly discards the unit type, returning just the rounded number.

**Consequences:**
- Calculation results are numerically correct but display incorrectly
- Users see "430K" (the MB value without unit) instead of "~400 GB"
- Breaks trust in the calculator for the exact use case it's designed for

**Prevention:** `evalNapkinConversion` must preserve the type hierarchy:
- If input is Quantity, return Quantity (with rounded value, same unit)
- If input is Rate, return Rate (with rounded amount, same denominator)
- If input is Currency, return Currency (with rounded value, same symbol)
- Only Numbers should return as Numbers

**Detection:**
- Test: `accumulate(rate, time) as napkin` should preserve unit
- Test: `(100 MB * 1000) as napkin` should show "~100 GB" not "~100K"
- Golden test: napkin.cm should include quantity preservation tests

**Which phase should address:** Interpreter correctness (FIRST priority)

**Codebase reference:** `/impl/interpreter/napkin_eval.go` lines 24-29, 99

---

### Pitfall 2: Undo/Redo Cursor Position Not Tracked

**What goes wrong:** Undoing a text change restores the content but leaves the cursor in a wrong position—possibly pointing at a non-existent location or a completely different line than where the edit occurred.

**Why it happens:** The current undo system only stores document content snapshots (`undoStack []string` in model.go:137-138), not the cursor position that went with each state. When users press undo, they expect to see the cursor jump back to WHERE the change was made.

**Consequences:**
- Cursor ends up at invalid position causing out-of-bounds errors
- User disorientation: "I undid, but where am I now?"
- Breaks the mental model of "undo = go back to exactly how it was"

**Prevention:** Store cursor position with each undo state:
```go
type UndoState struct {
    Content    string
    CursorLine int
    CursorCol  int
    ScrollOffset int  // Optional but improves UX
}
undoStack []UndoState
```

**Detection:**
- Cursor goes to line 0, col 0 after every undo
- Test: undo after editing line 10 should return cursor to line 10
- User reports: "undo takes me to the wrong place"

**Which phase should address:** Undo/redo implementation (early in milestone)

**Sources:**
- [Undo/Redo Implementations in Text Editors](https://www.mattduck.com/undo-redo-text-editors)
- [Design Thoughts: Undo Redo - super_editor Wiki](https://github.com/superlistapp/super_editor/wiki/Design-Thoughts:-Undo-Redo)

---

### Pitfall 3: File Save Not Atomic (Data Loss Risk)

**What goes wrong:** User presses Ctrl+S, save starts writing, power fails mid-write—file is now truncated or corrupted, original content lost.

**Why it happens:** Current save in model.go:1471 uses `os.WriteFile()` directly, which:
1. Truncates the file to 0 bytes
2. Writes new content
If crash occurs between steps 1 and 2, file is empty.

**Consequences:**
- User loses their document entirely
- No recovery option exists
- Trust in the editor destroyed

**Prevention:** Implement atomic save pattern:
```go
// Write to temp file first
tmpFile := filename + ".tmp"
if err := os.WriteFile(tmpFile, content, 0644); err != nil {
    return err
}
// Atomically rename temp to target (atomic on POSIX)
if err := os.Rename(tmpFile, filename); err != nil {
    os.Remove(tmpFile) // Clean up on failure
    return err
}
```

**Platform note:** `os.Rename` is guaranteed atomic on POSIX (Linux, macOS). On Windows, it may or may not be atomic, but is still safer than direct write.

**Detection:**
- Review: Check for tmp+rename pattern in save code
- User reports: "My file was empty after crash"

**Which phase should address:** File operations (HIGH priority)

**Sources:**
- [File Save Operation Should Be Atomic - fritzing-app Issue](https://github.com/fritzing/fritzing-app/issues/4148)
- [Towards Atomic File Modifications - DEV Community](https://dev.to/martinhaeusler/towards-atomic-file-modifications-2a9n)
- [npm/write-file-atomic](https://github.com/npm/write-file-atomic)

---

### Pitfall 4: Save Doesn't Flush Pending Edits

**What goes wrong:** User is typing on a line, presses Ctrl+S immediately. The current line content in `editBuf` hasn't been committed to the document yet, so save writes the old version.

**Why it happens:** The debounce mechanism means edits aren't committed until 100ms after typing stops. Ctrl+S might trigger during this window.

**Current state:** Looking at model.go `saveFile()`, it calls `getDocumentContent()` which reads from blocks, not from `editBuf`.

**Consequences:**
- User thinks they saved their latest typing
- File on disk is missing the last few characters
- Very confusing when file reopens without recent edits

**Prevention:**
- Before save, flush editBuf to document: `m.updateCurrentLine(m.editBuf)`
- Cancel any pending debounce timer
- Then proceed with save

**Detection:**
- Test: Type characters, immediately Ctrl+S, verify file contains typed chars
- User reports: "My last edits weren't saved"

**Which phase should address:** Save implementation (HIGH priority)

---

### Pitfall 5: Redo Stack Cleared on Any Edit

**What goes wrong:** After undoing 3 times, user makes a small edit, and ALL redo history is lost—even if the edit was unrelated to the undone changes.

**Why it happens:** The current implementation in model.go:276 does `m.redoStack = nil` on any new change. This is the standard "linear history" approach.

**Consequences:**
- User loses ability to redo after making any change
- "Undo tree" workflows (explore alternatives then return) become impossible
- User frustration when they accidentally type and lose redo stack

**Prevention:**
- Document this as expected behavior (if linear history is the design choice)
- OR implement tree-based undo (more complex but preserves all history)
- At minimum: show visual indicator when redo is available/cleared

**Detection:**
- Test: Undo 3x, type 1 char, verify redo stack is empty
- User reports: "I lost my redo history"

**Which phase should address:** Undo/redo implementation (document behavior or enhance)

**Sources:**
- [You Don't Know Undo/Redo - DEV Community](https://dev.to/isaachagoel/you-dont-know-undoredo-4hol)

---

## Moderate Pitfalls

Mistakes that cause delays, confusion, or technical debt.

### Pitfall 6: Undo Granularity Too Fine (Every Keystroke)

**What goes wrong:** Each typing pause creates an undo state. Pressing undo requires 20+ presses to undo a single sentence.

**Why it happens:** `pushUndoState()` is called after every edit via debounce. The debounce delay is 100ms, which means rapid typing creates separate undo states for each pause.

**Prevention:** Group edits into logical undo units:
- Whitespace boundaries (word-level undo)
- Paragraph/line boundaries
- Time-based grouping (1-2 second window)
- Navigation breaks an undo group (arrow key = new group)

**Detection:**
- Test: Type "hello world", count undo operations needed to remove it
- Should be 1-2 undos, not 11

**Which phase should address:** Undo/redo enhancement (after basic undo works)

---

### Pitfall 7: Unit Prefix Confusion (MB vs MiB vs Mb)

**What goes wrong:** User types "5 mb/s" meaning megabytes, but system interprets differently, off by factor of 8 or ~5%.

**Why it happens:** CalcMark's unit system (canonical.go) handles many aliases but:
- `mb` could mean: megabytes (MB), mebibytes (MiB), megabits (Mb)
- Case sensitivity varies by context
- The lexer/parser must make a choice, users may not know which

**Current state:** Looking at canonical.go, data units aren't explicitly listed (focus is on physical units). The rate parsing handles "MB/s" but case sensitivity is unclear.

**Consequences:**
- Off by 8x when confusing bits and bytes
- Off by ~5% when confusing SI (MB) vs binary (MiB)
- User gets wrong answers without realizing

**Prevention:**
- Audit all data unit aliases for consistent handling
- Document the conventions clearly (SI = 1000-based MB, binary = 1024-based MiB)
- Consider warning on ambiguous units
- Add explicit case handling

**Detection:**
- Test: `5 mb/s` should equal `5 MB/s` (if case-insensitive)
- Test: `5 MiB` should be different from `5 MB` if both supported
- Golden test: Add unit conversion edge cases

**Which phase should address:** Unit conversion audit

**Sources:**
- [NIST Metrication Errors](https://www.nist.gov/pml/owm/metrication-errors-and-mishaps)
- [Unit Conversion Mistakes](https://freeunitconvertertool.com/the-most-common-unit-conversion-mistakes-to-avoid/)
- [NASA Mars Climate Orbiter](https://sites.google.com/view/onlineunitconversions/four-tragedies-caused-by-erroneous-unit-conversion)

---

### Pitfall 8: Undo Before editBuf Commit Loses Edit

**What goes wrong:** User types "abc", immediately presses Ctrl+Z (undo). The "abc" was never committed to the document, so undo does nothing visible—but the editBuf is also not cleared, leaving the editor in an inconsistent state.

**Why it happens:** Undo operates on committed document states, but editBuf exists as a pending change buffer. The two systems don't communicate.

**Prevention:**
- Before undo: flush editBuf to document first
- OR: Cancel editBuf changes on undo (discard uncommitted typing)
- Document which behavior is chosen

**Detection:**
- Test: Type characters, immediately undo, verify consistent state
- editBuf and document should agree after undo

**Which phase should address:** Undo/redo implementation

---

## Minor Pitfalls

Mistakes that cause annoyance but are easily fixable.

### Pitfall 9: Save-As Overwrites Without Warning

**What goes wrong:** User chooses Save-As and enters a filename that already exists. File is overwritten without confirmation.

**Why it happens:** `os.WriteFile()` always overwrites. No pre-check for file existence.

**Prevention:**
```go
if _, err := os.Stat(filename); err == nil {
    // File exists, prompt for confirmation
    m.mode = StateOverwritePrompt
    return
}
```

**Detection:**
- Test: Save-As to existing file, verify prompt appears
- User reports: "It overwrote my file without asking"

**Which phase should address:** Save-As implementation

---

### Pitfall 10: Relative Path Handling in Save

**What goes wrong:** User types "myfile.cm" in Save-As. File is saved relative to... what? Working directory at app start? Current file's directory? Unclear.

**Why it happens:** `filepath.Abs()` uses current working directory, but in a TUI app, users may not know what that is.

**Current state:** model.go:1460 uses `filepath.Abs(filename)` which uses cwd.

**Prevention:**
- If current file is open, default to its directory
- Display the full path in status bar after save
- Consider showing full path preview before saving

**Detection:**
- Test: Open `/tmp/a.cm`, Save-As "b.cm", verify it goes to `/tmp/b.cm` not cwd
- User reports: "I can't find where my file was saved"

**Which phase should address:** Save-As implementation

---

### Pitfall 11: Rate Time Unit Normalization Inconsistency

**What goes wrong:** `100 MB/second` and `100 MB/s` should be identical, but might display or compare differently.

**Why it happens:** `NormalizeTimeUnit()` in rate.go handles this, but display uses `abbreviateTimeUnit()` which has its own mapping.

**Current state:** Looking at rate.go, normalization is done in `NewRate()`, which is good. Display abbreviates separately.

**Prevention:**
- Always normalize at construction time (already done, good)
- Ensure display and comparison both use canonical forms
- Test round-trip: parse -> display -> parse should be identical

**Detection:**
- Test: `100 MB/second == 100 MB/s` should be true
- Test: Display of both should be identical

**Which phase should address:** Unit conversion audit (low priority)

---

## Phase-Specific Warnings

| Phase Topic | Likely Pitfall | Mitigation |
|-------------|---------------|------------|
| Napkin formatter fix | Lose other type metadata (Currency symbol, Rate denominator) | Test ALL types through napkin, not just Quantity |
| Napkin formatter fix | Edge cases in rounding (negative, very small, very large) | Test: -1.5M, 0.001, 1e15 |
| Undo/redo | Cursor position not restored | Store cursor with each state from day 1 |
| Undo/redo | Edit buffer not committed before undo | Flush editBuf before manipulating undo stack |
| Undo/redo | Redo cleared unexpectedly | Document linear history behavior |
| File save | Direct write causes corruption on crash | Use temp file + atomic rename pattern |
| File save | editBuf not flushed | Commit pending edits before save |
| Save-As | Overwrite without confirmation | Check existence, prompt user |
| Save-As | Unclear path resolution | Default to current file's directory |
| Unit audit | Case sensitivity assumptions | Audit all unit aliases for consistent casing |
| Unit audit | Binary vs SI prefix confusion | Document conventions, add tests |

---

## Codebase-Specific Implementation Notes

### Current Undo/Redo (model.go lines 137-138, 265-277, 1313-1357)

The foundation is good but needs:
1. `UndoState` struct with cursor position (not just content string)
2. editBuf flush before `pushUndoState()`
3. Clear documentation of redo-clearing behavior
4. Consider grouping edits by word boundaries

### Napkin Bug (napkin_eval.go lines 24-29, 99)

This is a type-erasure problem. The fix must:
1. Check input type in the switch
2. Apply rounding to the numeric value
3. Reconstruct the SAME type with rounded value
4. Never downgrade Quantity -> Number

```go
// Current (wrong):
return types.NewNumber(decimal.NewFromFloat(roundedFloat))

// Fixed:
switch v := value.(type) {
case *types.Quantity:
    return types.NewQuantity(decimal.NewFromFloat(roundedFloat), v.Unit)
case *types.Rate:
    // Preserve rate structure with rounded amount
    ...
}
```

### File Operations (model.go lines 1443-1620)

Needs:
1. Atomic save (temp file + rename)
2. editBuf flush before save
3. Existence check for Save-As
4. Directory resolution for relative paths

---

## Sources

- [Undo/Redo Implementations in Text Editors](https://www.mattduck.com/undo-redo-text-editors) - Comprehensive overview of undo strategies
- [You Don't Know Undo/Redo](https://dev.to/isaachagoel/you-dont-know-undoredo-4hol) - Scope, granularity, and reaction handling
- [Design Thoughts: Undo Redo - super_editor](https://github.com/superlistapp/super_editor/wiki/Design-Thoughts:-Undo-Redo) - Cursor/selection tracking requirements
- [Towards Atomic File Modifications](https://dev.to/martinhaeusler/towards-atomic-file-modifications-2a9n) - Atomic save patterns
- [write-file-atomic (npm)](https://github.com/npm/write-file-atomic) - Reference implementation of atomic writes
- [NIST Metrication Errors](https://www.nist.gov/pml/owm/metrication-errors-and-mishaps) - Real-world unit conversion disasters
- [Unit Conversion Mistakes](https://freeunitconvertertool.com/the-most-common-unit-conversion-mistakes-to-avoid/) - Common patterns
- [NASA Mars Climate Orbiter](https://sites.google.com/view/onlineunitconversions/four-tragedies-caused-by-erroneous-unit-conversion) - Famous unit mixup ($327M loss)
- CalcMark codebase analysis: model.go, napkin_eval.go, rate.go, canonical.go (HIGH confidence)

---
*Pitfalls research for: CalcMark v1.1 milestone*
*Researched: 2026-02-06*
*Focus: Interpreter correctness, undo/redo, file operations*
