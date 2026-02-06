# Feature Research: v1.1 CalcMark Language

**Domain:** CLI/TUI calculation notepad with interpreter correctness focus
**Researched:** 2026-02-06
**Confidence:** HIGH for undo/redo and file ops (established patterns), MEDIUM for unit conversion (domain-specific)

## Research Focus

This research targets three feature areas for v1.1:
1. Undo/redo in text editors
2. Save/Save-As/Quit-with-prompt file operations
3. Unit conversion correctness requirements

---

## 1. Undo/Redo Feature Requirements

### Table Stakes (Must Have)

| Requirement | User Expectation | CalcMark Status | Priority |
|-------------|------------------|-----------------|----------|
| **Ctrl+Z to undo** | Universal keyboard shortcut, works in every text editor | Partially implemented (100-state stack) | P1 |
| **Ctrl+Y or Ctrl+Shift+Z to redo** | Standard shortcuts; inconsistency is "extremely annoying" per research | Implemented but untested | P1 |
| **Unlimited undo within session** | Users expect to undo "all actions taken in the session" | Limited to 100 states | P1 |
| **Character batching** | Typing "hello" should undo as one action, not 5 keystrokes | Not implemented (each change is separate) | P1 |
| **Cursor position restoration** | Undo must restore cursor to previous position | Implemented | P1 |
| **Redo stack cleared on new edit** | Standard behavior: undo, then type clears redo stack | Implemented | P2 |
| **Session-scoped** | Undo stack lost when editor closes (standard expectation) | Implemented | N/A |

### Research Sources (HIGH confidence)

- [You Don't Know Undo/Redo](https://dev.to/isaachagoel/you-dont-know-undoredo-4hol) - Detailed analysis of user expectations
- [Undo/Redo Implementations in Text Editors](https://www.mattduck.com/undo-redo-text-editors) - Implementation patterns
- [super_editor Undo Redo Design](https://github.com/superlistapp/super_editor/wiki/Design-Thoughts:-Undo-Redo) - Modern editor design

### Character Batching Rules

Users expect batching based on:

| Pattern | Expected Behavior |
|---------|-------------------|
| Continuous typing | Batch until pause (~1s) or word boundary |
| Typing then delete | Separate undo actions |
| Paste operation | Single undo action |
| Replace via autocomplete | Single undo action |
| Cut operation | Single undo action |

**Implementation note:** Current CalcMark saves state on every `pushUndoState()` call. Need debouncing/batching logic.

### Correctness Test Cases

```
# Undo/redo test matrix
1. Type "hello" -> undo should remove "hello" (not one char)
2. Type, pause 2s, type more -> two undo actions
3. Undo 3x, redo 2x -> correct state
4. Undo, type something -> redo stack cleared
5. Undo to initial state -> cannot undo further (no error)
6. Redo when stack empty -> no error, no change
7. Cursor at end of doc, undo -> cursor moves to where edit was
8. Select + delete -> undo restores selection
```

### Anti-Features (Do Not Build)

| Feature | Why Avoid | Alternative |
|---------|-----------|-------------|
| **Persistent undo across sessions** | Users expect undo lost on close; breaks mental model | None needed |
| **Undo tree (vim style)** | Complex UX, niche use case | Linear undo is standard |
| **Per-line undo** | Confusing scope, not expected | Document-level undo |

---

## 2. File Operations Feature Requirements

### Table Stakes (Must Have)

| Requirement | User Expectation | CalcMark Status | Priority |
|-------------|------------------|-----------------|----------|
| **Save (Ctrl+S)** | Save to existing file, no prompt if named | Implemented | P1 |
| **Modified indicator** | Show unsaved state clearly (dot in title, asterisk) | Exists (`modified` flag) | P1 |
| **Quit with unsaved changes prompt** | "You have unsaved changes" dialog before exit | Not implemented | P1 |
| **Save As (Ctrl+Shift+S or :saveas)** | Save to new filename | Not implemented | P1 |
| **New file prompt on quit** | If file is new (never saved), prompt for filename | Needs verification | P1 |
| **Disable save button when clean** | No action if nothing to save | Should verify | P2 |

### Research Sources (HIGH confidence)

- [VSCode Issue #104690](https://github.com/microsoft/vscode/issues/104690) - Trust issues from inconsistent save prompts
- [EditorWindow.hasUnsavedChanges](https://docs.unity3d.com/ScriptReference/EditorWindow-hasUnsavedChanges.html) - Unity documentation on patterns
- [Writing for the User: Unsaved Changes](https://bootcamp.uxdesign.cc/writing-for-the-user-unsaved-changes-d6e33d884c26) - UX copywriting

### Quit Dialog Design

Best practice dialog structure:

```
+------------------------------------------+
| You have unsaved changes.                |
|                                          |
| What would you like to do with them?     |
|                                          |
| [Save and Quit]  [Discard]  [Cancel]     |
+------------------------------------------+
```

**Key UX principles:**
1. Primary button = "Save and Quit" (safest action)
2. "Discard" should require confirmation or be secondary
3. "Cancel" returns to editor (Esc key)
4. Red traffic light / dot indicator for macOS style

### Save As Implementation

Two approaches:

| Approach | Pros | Cons | Recommendation |
|----------|------|------|----------------|
| Command mode (`:saveas file.cm`) | Consistent with editor commands | Users must know command | Implement first |
| Modal dialog | Familiar GUI pattern | Complex in TUI | Defer post-v1.1 |

### Correctness Test Cases

```
# File operation test matrix
1. Open file, no changes, quit -> no prompt
2. Open file, make change, quit -> prompt appears
3. Open file, make change, save, quit -> no prompt
4. Open file, make change, Ctrl+S -> file saved, modified=false
5. New file (no path), Ctrl+S -> prompt for filename (Save As behavior)
6. Save As to same filename -> works (overwrites)
7. Save As to new filename -> new file created, original unchanged
8. Quit prompt: press Cancel -> returns to editor
9. Quit prompt: press Discard -> exits without save
10. Quit prompt: press Save -> saves then exits
```

### Anti-Features (Do Not Build)

| Feature | Why Avoid | Alternative |
|---------|-----------|-------------|
| **Auto-save on timer** | Breaks user mental model; CalcMark is file-based | Manual save only |
| **Backup/recovery system** | Scope creep for v1.1 | Standard OS file recovery |
| **Multiple open files** | Already ruled out in v1.0 scope | Single file per session |

---

## 3. Unit Conversion Correctness Requirements

### The Known Bug

**Issue:** `accumulate(5mb/s, 1 day) as napkin` returns "430K" instead of "~400 GB"

**Root cause analysis:**
1. `accumulate(5 MB/s, 1 day)` correctly returns `*types.Quantity{Value: 432000000000, Unit: "MB"}`
   - 5 MB/s * 86400 seconds = 432,000,000,000 bytes = 432 GB
2. `as napkin` in `evalNapkinConversion()` extracts only the numeric value (line 29)
3. Result is `*types.Number{Value: 430000000000}` - the unit "MB" is lost
4. Display formatter sees a bare number, formats as "430B" (430 billion)
5. But wait - the bug report says "430K", suggesting the magnitude calculation is also wrong

**The fix must:**
1. Preserve unit through napkin conversion, OR
2. Convert to base units before napkin display, OR
3. Return a display-only type that carries unit context

### Table Stakes (Correctness Requirements)

| Requirement | User Expectation | Priority |
|-------------|------------------|----------|
| **Dimensional consistency** | Units cannot disappear through operations | P1 |
| **Magnitude preservation** | 5 MB/s * 86400s = 432 GB, not 432K | P1 |
| **Unit family awareness** | Display in appropriate scale (GB not bytes) | P1 |
| **Rate accumulation** | rate * time = quantity with correct units | P1 |
| **Napkin + units** | `X as napkin` should show "~400 GB" not "~400" | P1 |

### Research Sources (MEDIUM confidence)

- [Dimensional Analysis - Chemistry LibreTexts](https://chem.libretexts.org/Courses/Solano_Community_College/Chem_160/Chapter_01:_Chemical_Foundations/1.6:_Dimensional_Analysis_(Unit_Conversions)) - Correctness criteria
- [OpenStax Chemistry 2e](https://openstax.org/books/chemistry-2e/pages/1-6-mathematical-treatment-of-measurement-results) - Unit conversion rules

### Dimensional Analysis Rules

From physics/chemistry education:

| Rule | Meaning | CalcMark Implication |
|------|---------|---------------------|
| **Dimensional homogeneity** | Both sides of equation must have same dimensions | Additions/subtractions must check unit compatibility |
| **Unit cancellation** | rate (MB/s) * duration (s) = quantity (MB) | Implemented in `accumulateRate()` |
| **No unit loss** | Operations cannot discard dimensional information | `as napkin` violates this currently |
| **Conversion factor precision** | Standardized factors are exact (infinite sig figs) | Use exact decimal math (shopspring/decimal) |

### Unit Conversion Edge Cases to Test

```
# Rate accumulation
accumulate(5 MB/s, 1 day) -> 432 GB (not 432000000 MB, not 432000000000 bytes)
accumulate(100 req/s, 1 hour) -> 360K requests
accumulate($0.10/hour, 30 days) -> $72

# Rate conversion
100 MB/s per day -> 8.64 TB/day
1000 req/s per hour -> 3.6M req/hour

# Quantity normalization for display
1000 m -> 1 km
23400000 GB -> 22.89 PB (via NormalizeForDisplay)
100000 bytes -> 97.66 KB

# Napkin with units (THE BUG)
1234567 bytes as napkin -> ~1.2 MB (normalized + rounded)
432000000000 bytes as napkin -> ~400 GB
5 TB + 10 TB as napkin -> ~15 TB

# Edge: mixed systems (should error or handle gracefully)
5 miles + 10 km -> ? (needs unit conversion first)
```

### Unit Conversion Implementation Audit Checklist

Based on `spec/units/canonical.go` and `format/display/normalize.go`:

| Unit Family | Registered? | Display Normalization? | Notes |
|-------------|-------------|------------------------|-------|
| SI Length (mm, cm, m, km) | Yes | Yes | Good |
| US Length (in, ft, yd, mi) | Yes | Yes | Good |
| SI Mass (mg, g, kg, t) | Yes | Yes | Good |
| US Mass (oz, lb) | Yes | Yes | Good |
| SI Volume (ml, l) | Yes | Yes | Good |
| US Volume (tsp, tbsp, cup, pt, qt, gal) | Yes | Yes | Good |
| Data Storage (bytes, KB, MB, GB, TB, PB) | Yes | Yes | Bug in napkin conversion |
| Power (W, kW, MW) | Yes | Yes | Good |
| Energy SI (J, kJ) | Yes | Yes | Good |
| Energy food (cal, kcal) | Yes | Yes | Good |
| Area SI (sq cm, sq m, ha, sq km) | Yes | Yes | Good |
| Area US (sq in, sq ft, sq yd, ac, sq mi) | Yes | Yes | Good |
| Temperature (C, F, K) | In canonical.go | Not in normalize.go | Gap: non-linear conversion |
| Speed (m/s, km/h, mph, knot) | In canonical.go | Not in normalize.go | Gap: compound units |

### Correctness Test Matrix

```
# Basic quantity display
5000 m -> "5 km"
1024 KB -> "1 MB" (or "1024 KB" depending on display mode)
2.5 hours -> "2.5 hours"

# Rate display
100 MB/s -> "100 MB/s"
1000000 bytes/s -> "976.56 KB/s"

# Accumulation results
5 MB/s * 1 day -> displays as "432 GB" (not raw bytes)
100 req/s * 1 hour -> displays as "360K requests"

# Napkin conversion (THE FIXES NEEDED)
(5 MB/s * 1 day) as napkin -> "~400 GB" (CURRENTLY BROKEN: shows "430K")
1234567 users as napkin -> "~1.2M users" (should preserve unit)
$1234567 as napkin -> "~$1.2M" (currency should work)

# Temperature (special case - non-linear)
32 F as C -> 0 C (not a linear multiply)
100 C as F -> 212 F

# Compound rate display
5 miles/gallon -> "5 miles/gallon" (fuel efficiency)
60 km/h -> "60 km/h"
```

### Anti-Patterns to Avoid

| Anti-Pattern | Why Bad | Correct Approach |
|--------------|---------|------------------|
| **Stripping units for napkin** | Loses dimensional information | Keep units, normalize to human scale |
| **Displaying raw base units** | "432000000000 bytes" is unreadable | Use NormalizeForDisplay |
| **Silent unit mismatch** | Adding km + miles silently fails | Error or auto-convert with warning |
| **Precision loss in conversion** | 1 inch != 2.54 cm due to rounding | Use exact decimal arithmetic |

---

## Feature Priority Matrix

| Feature | User Value | Fix Complexity | Priority |
|---------|------------|----------------|----------|
| Napkin preserves units | HIGH (correctness) | MEDIUM | P1 |
| Character batching in undo | HIGH (UX) | MEDIUM | P1 |
| Quit with unsaved prompt | HIGH (data safety) | LOW | P1 |
| Unlimited undo (vs 100) | MEDIUM | LOW | P1 |
| Save As command | MEDIUM | LOW | P1 |
| Temperature conversion | LOW (edge case) | HIGH | P2 |
| Speed unit normalization | LOW (edge case) | MEDIUM | P2 |

---

## Correctness Test Coverage Recommendations

### Undo/Redo Tests

```go
// cmd/calcmark/tui/editor/undo_test.go
func TestUndoBatching_ContinuousTyping(t *testing.T)
func TestUndoBatching_PauseCreatesSeparateAction(t *testing.T)
func TestUndo_CursorPositionRestored(t *testing.T)
func TestRedo_ClearedOnNewEdit(t *testing.T)
func TestUndo_CannotUndoPastInitialState(t *testing.T)
```

### File Operation Tests

```go
// cmd/calcmark/tui/editor/file_ops_test.go
func TestQuit_NoChangesNoPrompt(t *testing.T)
func TestQuit_UnsavedChangesPromptShown(t *testing.T)
func TestSave_ClearsModifiedFlag(t *testing.T)
func TestSaveAs_NewFileCreated(t *testing.T)
func TestSaveAs_OriginalUnchanged(t *testing.T)
```

### Unit Conversion Tests

```go
// impl/interpreter/napkin_test.go
func TestNapkinConversion_PreservesUnits(t *testing.T)
func TestNapkinConversion_AccumulatedRate(t *testing.T)
func TestNapkinConversion_Currency(t *testing.T)

// format/display/display_test.go
func TestFormatQuantity_LargeValues(t *testing.T)
func TestFormatRate_WithNormalization(t *testing.T)
```

---

## Sources

### Undo/Redo (HIGH confidence)
- [You Don't Know Undo/Redo - DEV Community](https://dev.to/isaachagoel/you-dont-know-undoredo-4hol)
- [Undo/Redo Implementations in Text Editors](https://www.mattduck.com/undo-redo-text-editors)
- [super_editor Wiki: Undo Redo Design](https://github.com/superlistapp/super_editor/wiki/Design-Thoughts:-Undo-Redo)
- [Implementing Undo in a Text Editor](https://routley.io/posts/text-editor-undo)
- [Text Editor Data Structures: Rethinking Undo](https://cdacamar.github.io/data%20structures/algorithms/benchmarking/text%20editors/c++/rethinking-undo/)

### File Operations (HIGH confidence)
- [VSCode Issue #104690: Missing prompt to save changes](https://github.com/microsoft/vscode/issues/104690)
- [Unity EditorWindow.hasUnsavedChanges](https://docs.unity3d.com/ScriptReference/EditorWindow-hasUnsavedChanges.html)
- [Writing for the User: Unsaved Changes](https://bootcamp.uxdesign.cc/writing-for-the-user-unsaved-changes-d6e33d884c26)
- [Bootstrap Studio Forum: Unsaved Changes Dialog](https://forum.bootstrapstudio.io/t/ux-change-close-with-unsaved-changes-dialog/11976)

### Unit Conversion (MEDIUM confidence)
- [Dimensional Analysis - Chemistry LibreTexts](https://chem.libretexts.org/Courses/Solano_Community_College/Chem_160/Chapter_01:_Chemical_Foundations/1.6:_Dimensional_Analysis_(Unit_Conversions))
- [OpenStax Chemistry 2e: Measurement Results](https://openstax.org/books/chemistry-2e/pages/1-6-mathematical-treatment-of-measurement-results)
- [Omnicalculator: Dimensional Analysis](https://www.omnicalculator.com/conversion/dimensional-analysis)

### CalcMark Codebase (Authoritative)
- `/Users/bitsbyme/projects/go-calcmark/impl/interpreter/napkin_eval.go` - Current napkin implementation
- `/Users/bitsbyme/projects/go-calcmark/format/display/normalize.go` - Unit normalization
- `/Users/bitsbyme/projects/go-calcmark/spec/units/canonical.go` - Canonical units
- `/Users/bitsbyme/projects/go-calcmark/cmd/calcmark/tui/editor/model.go` - Current undo implementation

---

*Feature research for: CalcMark v1.1 interpreter correctness and editor completion*
*Researched: 2026-02-06*
