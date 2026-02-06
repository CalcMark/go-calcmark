# Architecture Research

**Domain:** TUI Editor + CLI for an Interpreted Language (CalcMark)
**Researched:** 2026-02-06 (updated for v1.1 milestone)
**Confidence:** HIGH

## v1.1 Milestone Architecture Additions

This section documents how v1.1 features integrate with the existing architecture.

### Overview: v1.1 Feature Integration Points

| Feature | Location | Integration Complexity |
|---------|----------|------------------------|
| Napkin unit bug fix | `impl/interpreter/napkin_eval.go` | LOW - localized fix |
| Unlimited undo/redo | `cmd/calcmark/tui/editor/model.go` | LOW - remove limit |
| Ctrl+Z/Ctrl+Y bindings | `cmd/calcmark/tui/editor/model.go` | LOW - add key handlers |
| Save/Quit prompts | Already implemented | DONE - verify behavior |

---

## Undo/Redo State Architecture

### Current Implementation

Location: `cmd/calcmark/tui/editor/model.go`

```go
type Model struct {
    // ...

    // Undo/redo
    undoStack []string // Document content snapshots
    redoStack []string

    // ...
}
```

**Current behavior:**
- `undoStack` stores document content as strings
- Maximum 100 entries (hardcoded in `pushUndoState()`)
- `redoStack` cleared on new changes
- Snapshots pushed via `pushUndoState()` after document modifications

**Key methods:**
- `pushUndoState()` - Called after edit operations (lines 266-277)
- `undo()` - Pops from undoStack, pushes to redoStack, rebuilds document (lines 1313-1335)
- `redo()` - Pops from redoStack, pushes to undoStack, rebuilds document (lines 1338-1357)

### v1.1 Enhancement: Unlimited Undo

**Current limit (line 273-275):**
```go
// Limit undo stack size
if len(m.undoStack) > 100 {
    m.undoStack = m.undoStack[1:]
}
```

**Recommendation:** Remove this limit for v1.1. CalcMark documents are typically small (<10KB). Even 1000 undo states = ~10MB memory, which is acceptable for a desktop CLI tool.

**Alternative (if memory becomes an issue):** Implement delta-based undo using content diffs, but defer this complexity until users report actual issues.

### v1.1 Enhancement: Keyboard Bindings

**Current access:** Undo/redo only accessible via `/undo` and `/redo` slash commands.

**Needed:** Add standard keyboard shortcuts in `handleDefaultKey()`:
- Ctrl+Z for undo
- Ctrl+Y or Ctrl+Shift+Z for redo

**Implementation location:** `model.go` lines 466-512 (handleDefaultKey switch statement)

### Integration Points

Undo state is pushed in these locations:
- `handleEnterKey()` line 727 - After line insertion
- `handleBackspaceKey()` line 762 - After line join
- `transitionToProcessing()` indirectly via `redetectBlockTypes()`
- `saveCurrentLineAndMoveTo()` line 1110 - After navigation with save
- `insertLine()` line 1264 - After new line insertion
- `deleteLine()` line 2039 - After line deletion

---

## Unit System Architecture

### Three-Layer Unit System

```
Layer 1: spec/units/canonical.go
    │
    │   Defines: Unit names, symbols, aliases, categories
    │   Purpose: Language specification (what units exist)
    │   Used by: Parser, autocomplete, validation
    │   Key types: UnitMapping, StandardUnits map
    │
Layer 2: impl/interpreter/unit_library.go
    │
    │   Defines: Conversion functions (ToBaseUnit, FromBaseUnit)
    │   Purpose: Mathematical conversion at evaluation time
    │   Used by: Interpreter during calculation
    │   Key types: UnitInfo, QuantityCategory
    │
Layer 3: format/display/normalize.go
    │
    │   Defines: Display normalization (1000m → 1km)
    │   Purpose: Human-readable output formatting
    │   Used by: TUI preview pane, formatters
    │   Key function: NormalizeForDisplay()
```

### Unit Conversion Flow

```
Input: "100 feet in meters"
        │
        ▼
Parser (spec/parser) → AST: InConversion{Quantity{100, "feet"}, "meters"}
        │
        ▼
Interpreter (impl/interpreter/unit_conversion_eval.go)
        │
        ├── GetUnitInfo("feet") → UnitInfo{Category: Length, ToBaseUnit: feet→meters}
        ├── GetUnitInfo("meters") → UnitInfo{Category: Length, FromBaseUnit: meters→meters}
        ├── Verify same category
        └── Convert: feet → meters (base) → meters (target)
        │
        ▼
Result: Quantity{30.48, "meters"}
        │
        ▼
Display (format/display/normalize.go)
        │
        └── NormalizeForDisplay(30.48, "meters") → (30.48, "m")
        │
        ▼
Output: "30.48 m"
```

---

## Napkin Formatter Bug Analysis

### The Bug

**Symptom:** `accumulate(5mb/s, 1 day) as napkin` returns "430K" instead of ~400GB

**Root Cause Location:** `impl/interpreter/napkin_eval.go` lines 22-36

```go
func (interp *Interpreter) evalNapkinConversion(n *ast.NapkinConversion) (types.Type, error) {
    value, err := interp.evalNode(n.Expression)
    // ...

    switch v := value.(type) {
    case *types.Number:
        numValue = v.Value
    case *types.Quantity:
        // BUG: Just uses numeric value, DISCARDS the unit
        numValue = v.Value  // ← Unit lost here!
    case *types.Rate:
        // BUG: Also loses unit context
        numValue = v.Amount.Value
    // ...
    }

    // Returns Number only, never preserves unit
    return types.NewNumber(decimal.NewFromFloat(roundedFloat)), nil
}
```

### Fix Required

When input is a Quantity, return a Quantity with the same unit:

```go
func (interp *Interpreter) evalNapkinConversion(n *ast.NapkinConversion) (types.Type, error) {
    value, err := interp.evalNode(n.Expression)
    if err != nil {
        return nil, err
    }

    var numValue decimal.Decimal
    var originalUnit string  // NEW: preserve unit

    switch v := value.(type) {
    case *types.Number:
        numValue = v.Value
    case *types.Quantity:
        numValue = v.Value
        originalUnit = v.Unit  // NEW: capture unit
    case *types.Rate:
        numValue = v.Amount.Value
        originalUnit = v.Amount.Unit  // NEW: capture unit from rate amount
    // ... rest of cases
    }

    // ... rounding logic ...

    // NEW: Return appropriate type based on input
    roundedValue := decimal.NewFromFloat(roundedFloat)
    if originalUnit != "" {
        return &types.Quantity{Value: roundedValue, Unit: originalUnit}, nil
    }
    return types.NewNumber(roundedValue), nil
}
```

### Files to Modify for Fix

1. `impl/interpreter/napkin_eval.go` - Fix unit preservation
2. Add test in `impl/interpreter/napkin_unary_test.go` - Test for quantity with unit
3. Verify `impl/interpreter/napkin.go` - `formatNapkin()` may need updates for display

---

## File Operations Architecture

### Existing Infrastructure (v1.0)

**Save implementation** (model.go lines 1443-1484):
```go
func (m *Model) saveFile(filename string) {
    // Uses filepath or provided filename
    // Ensures .cm extension
    // Gets absolute path
    // Writes via os.WriteFile
    // Updates: m.filepath, m.savedContent, m.modified, m.statusMsg
}
```

**Unsaved changes detection** (model.go lines 1556-1561):
```go
func (m *Model) hasUnsavedChanges() bool {
    currentContent := m.getDocumentContent()
    return currentContent != m.savedContent
}
```

**Quit handling** (model.go lines 376-383):
```go
case tea.KeyCtrlQ:
    if m.hasUnsavedChanges() {
        m.mode = StateSavePrompt
        m.statusMsg = "Unsaved changes! Save before quit? (y/n/c)"
        return m, nil
    }
    m.quitting = true
    return m, tea.Quit
```

### v1.1 Status

All file operations appear to be implemented:
- **Ctrl+S**: Save (lines 386-395)
- **Ctrl+Q**: Quit with unsaved changes prompt (lines 376-383)
- **StateSaveAsPath**: Save-as dialog (lines 993-1025)
- **hasUnsavedChanges()**: Dirty state detection (lines 1556-1561)

**Verification needed:** Test edge cases:
- Save to readonly directory
- Save with disk full
- Quit during save operation
- Save-as with existing filename (overwrite prompt?)

---

## Editor State Machine

Location: `cmd/calcmark/tui/editor/state.go`

### Core States (EditorState)

```
┌─────────────┐
│  StateReady │ ←──────────────────────────────┐
│             │                                │
│  Invariants:│                                │
│  - doc != nil                                │
│  - eval != nil                               │
│  - userIsTyping = false                      │
└──────┬──────┘                                │
       │ User starts typing                    │
       ▼                                       │
┌─────────────┐                                │
│StateEditing │                                │
│             │                                │
│  Invariants:│                                │
│  - editBuf populated                         │
│  - userIsTyping = true                       │
└──────┬──────┘                                │
       │ Debounce fires / ENTER / navigation   │
       ▼                                       │
┌─────────────────┐                            │
│StateProcessing  │────────────────────────────┘
│                 │
│  Actions:
│  - userIsTyping = false
│  - Save editBuf to document
│  - Re-detect block types
│  - Re-evaluate
│  - Auto-transition to StateReady
└─────────────────┘
```

### UI States (InputState)

Separate from core editing states, these control which UI component receives input:

| State | Purpose | Exit Condition |
|-------|---------|----------------|
| StateDefault | Normal editing | Modal triggers |
| StateAutocomplete | Autocomplete popup visible | ESC, Tab, selection |
| StateGlobals | Globals panel focused | ESC, Enter |
| StateHelp | Help overlay visible | F1, ESC |
| StateExportFormat | Export format selection | Number key, ESC |
| StateExportPath | Export path input | Enter, ESC |
| StateSavePrompt | Unsaved changes dialog | y/n/c key |
| StateSaveAsPath | Save-as filename input | Enter, ESC |

---

## Suggested Build Order for v1.1

### Phase 1: Interpreter Correctness (Priority)

1. **Add failing test** for `as napkin` with Quantity input
   - File: `impl/interpreter/napkin_unary_test.go` (or new file)
   - Test case: `accumulate(5 mb/s, 1 day) as napkin` should preserve "GB" or "mb" unit

2. **Fix `evalNapkinConversion()`** to preserve units
   - File: `impl/interpreter/napkin_eval.go`
   - Change: Track `originalUnit`, return Quantity when unit exists

3. **Verify fix** with `task test`

4. **Add edge case tests**:
   - Rate with napkin: `100 GB/day as napkin`
   - Currency with napkin: `$1234567 as napkin`
   - Duration with napkin: `86400 seconds as napkin`

### Phase 2: Undo/Redo Enhancement

1. **Remove 100-state limit**
   - File: `cmd/calcmark/tui/editor/model.go`
   - Line: 273-275 (delete the if block)

2. **Add Ctrl+Z keybinding for undo**
   - File: `cmd/calcmark/tui/editor/model.go`
   - Location: `handleKey()` or `handleDefaultKey()` switch statement
   - Implementation: `case tea.KeyCtrlZ: m.undo(); return m, nil`

3. **Add Ctrl+Y keybinding for redo**
   - Same location as above
   - Note: Ctrl+Shift+Z may require special handling for terminals

4. **Optional: Add undo/redo state to status bar**
   - Show count: "3 undos | 1 redo"

### Phase 3: File Operation Verification

1. **Test existing save/quit/save-as functionality**
2. **Add catwalk tests** for file operation flows
3. **Handle edge cases** (readonly, disk full, etc.)

---

## Key Files Reference

| Feature | Primary File | Key Functions/Locations |
|---------|--------------|-------------------------|
| Undo/Redo State | model.go lines 137-138 | `undoStack`, `redoStack` fields |
| Undo/Redo Logic | model.go lines 266-277, 1313-1357 | `pushUndoState()`, `undo()`, `redo()` |
| State Machine | state.go | `transitionToReady()`, `transitionToEditing()`, `transitionToProcessing()` |
| Napkin Evaluation | napkin_eval.go | `evalNapkinConversion()` |
| Napkin Display | napkin.go | `formatNapkin()` |
| Unit Conversion | unit_conversion.go | `evalQuantityOperation()`, `convertQuantity()` |
| Unit Registry | unit_library.go | `GetUnitInfo()`, category definitions |
| Unit Canonical | spec/units/canonical.go | `StandardUnits`, `NormalizeUnitName()` |
| Display Formatting | format/display/display.go | `Format()`, `FormatQuantity()` |
| Display Normalization | format/display/normalize.go | `NormalizeForDisplay()` |
| Save/File Ops | model.go lines 1443-1620 | `saveFile()`, `hasUnsavedChanges()`, `openFile()` |
| Key Handling | model.go lines 359-513 | `handleKey()`, `handleDefaultKey()` |

---

## Standard Architecture

### System Overview

```
+-------------------------------------------------------------------+
|                       CLI Layer (cobra)                            |
|   root.go  eval.go  convert.go  edit.go  tui.go  version.go      |
+-----+-------------------+-------------------+--------------------+
      |                   |                   |
      v                   v                   v
+------------+   +-----------------+   +---------------+
|  eval cmd  |   |    TUI App      |   |  convert cmd  |
| (headless) |   | (App struct)    |   |  (headless)   |
+-----+------+   +--------+--------+   +-------+-------+
      |                   |                     |
      |          +--------+--------+            |
      |          |                 |            |
      |   +------v------+  +------v------+     |
      |   | REPL Model  |  |Editor Model |     |
      |   | (simple)    |  | (two-pane)  |     |
      |   +------+------+  +------+------+     |
      |          |                 |            |
      +----------+---------+------+            |
                           |                   |
                           v                   v
+-------------------------------------------------------------------+
|                   Shared Component Layer                           |
|  StatusBar  ContextFooter  Suggest  Globals  Pinned  SideBySide   |
+-------------------------------------------------------------------+
                           |
                           v
+-------------------------------------------------------------------+
|                     Pure Computation Layer                         |
|  AlignedModel  LineModel  Results  Geometry  WrapText  Markdown   |
+-------------------------------------------------------------------+
                           |
              +------------+------------+
              |                         |
              v                         v
+------------------------+   +------------------------+
|     spec/ (Language)   |   |  impl/ (Runtime)       |
| lexer, parser, ast,    |   | interpreter, evaluator |
| semantic, document,    |   | document, types, wasm  |
| types, units           |   |                        |
+------------------------+   +------------------------+
```

### Component Responsibilities

| Component | Responsibility | Typical Implementation |
|-----------|----------------|------------------------|
| **CLI Layer** (cobra) | Parse CLI args, dispatch to subcommands, load config | `cmd/calcmark/cmd/` -- thin routing layer |
| **App** | Top-level Bubble Tea model, mode switching (REPL/Editor) | `cmd/calcmark/tui/app.go` -- orchestrator |
| **REPL Model** | Single-line input, scrolling history, slash commands | `cmd/calcmark/tui/repl/` -- standalone model |
| **Editor Model** | Two-pane editor: source (left), results (right) | `cmd/calcmark/tui/editor/` -- the big one |
| **Shared Components** | Reusable pure-rendering components for any TUI mode | `cmd/calcmark/tui/components/` |
| **Pure Computation** | Geometry, alignment, wrapping -- no side effects | `aligned.go`, `linemodel.go`, `sidebyside.go` |
| **spec/** | Language grammar, AST, types, document model | Independent of runtime/UI |
| **impl/** | Interpreter, evaluator, environment, WASM bindings | Depends on spec, never on UI |
| **format/** | Output formatters (text, JSON, HTML, MD, CM) | Depends on spec, never on UI |

### Dependency Flow (Critical Invariant)

```
spec/  <--  impl/  <--  format/
  ^           ^            ^
  |           |            |
  +-----+----+-----+------+
        |          |
      cmd/calcmark/tui/     (depends on all three)
        |
      cmd/calcmark/cmd/     (depends on tui/ and config/)
```

**Rule:** `spec/` NEVER imports from `impl/`, `format/`, or `cmd/`. This is enforced by convention and tested by successful compilation. Violating this breaks WASM builds and language spec independence.

---

## Architectural Patterns

### Pattern 1: Pure Computation Core (Functional Core, Imperative Shell)

**What:** Separate pure computation from side-effectful Bubble Tea model logic. All geometry, alignment, and wrapping are pure functions. The Bubble Tea model is a thin shell that calls pure functions and wires up state.

**Example:**
```go
// Pure function -- fully testable, no dependencies on bubbletea/lipgloss
func ComputeAlignedModel(input AlignedModelInput, ...) AlignedModel { ... }

// Imperative shell -- Bubble Tea model calls pure functions
func (m *Model) GetAlignedModel(sourceWidth, previewWidth int) *AlignedModel {
    // ... compute cache key, call pure function, cache result ...
}
```

### Pattern 2: State Machine for Editor Modes

**What:** Explicit state transitions with invariant checking. The editor has states (Ready, Editing, Processing) with documented invariants and transition functions.

**Example from codebase:**
```go
// state.go -- explicit transitions with invariant enforcement
func (m *Model) transitionToReady() {
    // INVARIANT: Document must exist with at least 1 block
    if m.doc == nil || len(m.doc.GetBlocks()) == 0 {
        m.doc, _ = document.NewDocument("\n")
    }
    // ...
    m.state = StateReady
}
```

### Pattern 3: Document Rebuild for Content Changes

When document content changes:
1. Get content as string
2. Create new document via `document.NewDocument(content)`
3. Create new evaluator
4. Re-evaluate
5. Restore cursor position

**This is why undo/redo is simple:** Just store content string, rebuild on restore.

---

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Undo/Redo Integration | HIGH | Current implementation clearly documented, enhancement path clear |
| Napkin Bug Location | HIGH | Root cause identified in napkin_eval.go line 28 |
| Unit System Architecture | HIGH | Three-layer design well-documented in source |
| File Operations | HIGH | Infrastructure exists, just needs verification |
| State Machine | HIGH | Explicit, well-documented in state.go |

---

## Sources

- `/Users/bitsbyme/projects/go-calcmark/cmd/calcmark/tui/editor/model.go` - TUI editor state
- `/Users/bitsbyme/projects/go-calcmark/cmd/calcmark/tui/editor/state.go` - State machine
- `/Users/bitsbyme/projects/go-calcmark/impl/interpreter/napkin_eval.go` - Bug location
- `/Users/bitsbyme/projects/go-calcmark/impl/interpreter/unit_library.go` - Unit registry
- `/Users/bitsbyme/projects/go-calcmark/format/display/normalize.go` - Display normalization
- `/Users/bitsbyme/projects/go-calcmark/.planning/PROJECT.md` - v1.1 requirements
- CalcMark codebase analysis (HIGH confidence -- direct code reading)

---
*Architecture research for: CalcMark v1.1 Milestone*
*Updated: 2026-02-06*
