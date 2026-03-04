---
title: "Go Value-Type Closure Capture Bug in TUI Autocomplete"
date: 2026-03-04
category: logic-errors
tags: [go, closures, value-types, tui, autocomplete, state-management, bubble-tea]
severity: medium
component: cmd/calcmark/tui/editor
symptoms:
  - "Position-aware variable suggestions ignored cursor line updates"
  - "Autocomplete showed variables defined below current cursor position"
  - "Closure captured stale cursor position despite Model.cursorLine being updated"
root_cause: >
  Model is a value type returned from New(). Closures created inside New()
  capture the local Model variable. After return, the caller receives a copy
  while the closure still references the original (now-stale) local variable.
  Mutations to the caller's copy never propagate to the closure.
resolution_pattern: >
  Replaced closure-based state capture with pointer indirection: stored
  mutable state on a heap-allocated struct field (*VariableSuggestionSource)
  that both the closure and the returned Model share via pointer.
related_files:
  - cmd/calcmark/tui/editor/autocomplete.go
  - cmd/calcmark/tui/editor/model.go
  - cmd/calcmark/tui/components/suggest.go
---

# Go Value-Type Closure Capture Bug in TUI Autocomplete

## Problem

During implementation of position-aware variable suggestions in the CalcMark TUI editor (Bubble Tea v2), a closure in the `New()` constructor captured the local `Model` value. Since `New()` returns `Model` by value, the closure's `m` and the caller's `m` became separate copies after return. Setting `m.cursorLine` on the caller's copy had no effect on the closure's copy, causing variable suggestions to always filter from line 0 regardless of actual cursor position.

The bug was **silent** — no compilation errors, no panics, just wrong autocomplete results.

## Investigation

1. **Initial approach**: Filter variables by position inside the `getVariables` closure using `m.cursorLine` — this seemed natural since the closure already captured `m`
2. **Symptom**: Test set `m.cursorLine = 2` on the returned model, but the closure still observed `cursorLine = 0`
3. **Discovery**: The closure and the caller's model referenced different memory locations despite appearing to be the same variable

## Root Cause

Go closures capture variables by reference (to the variable itself, not a copy of its value). However, when `New()` returns `Model` by value:

1. The local `m` inside `New()` lives on the stack
2. The closure captures a reference to this stack-local `m`
3. `return m` creates a **copy** — the caller receives a distinct `Model`
4. The closure still references the original stack-local `m`, which is never updated again
5. Any mutations to the caller's `m.cursorLine` are invisible to the closure

```go
// BROKEN: closure captures value-type Model
func New(...) Model {
    m := Model{...}
    varSource := NewVariableSuggestionSource(func() map[string]string {
        // Captures m — but this is the LOCAL copy inside New()
        // After return, caller gets a DIFFERENT m
        // m.cursorLine here is always 0
        return filterByLine(m.cursorLine)  // stale!
    })
    return m  // Returns a COPY
}
```

## Solution

Decoupled mutable cursor state from the closure by using pointer indirection:

1. **Added `CursorLine` field** to `VariableSuggestionSource` struct — set externally before each `GetSuggestions` call
2. **Added `getDefinedLines` callback** — returns `varName -> definition line` mapping (behavior injection, not state capture)
3. **Stored `*VariableSuggestionSource` pointer** as `varSource` field on `Model` — the pointer survives the value copy
4. **Synchronized state** in `updateAutocompleteState`: set `m.varSource.CursorLine = m.cursorLine` before fetching suggestions

```go
// FIXED: mutable state on heap-allocated struct shared via pointer
type VariableSuggestionSource struct {
    getVariables    func() map[string]string
    getDefinedLines func() map[string]int
    CursorLine      int  // Set externally before each GetSuggestions call
}

type Model struct {
    varSource *VariableSuggestionSource  // Pointer survives value copy
    // ...
}

// In updateAutocompleteState — synchronize before use:
if m.varSource != nil {
    m.varSource.CursorLine = m.cursorLine  // Reads from live model
}
```

**Key insight**: When a value-type struct is returned from a constructor, closures created inside the constructor that reference `self` will hold stale references. The fix is pointer indirection — store mutable state on a heap-allocated struct that both the closure and the returned value share.

## Prevention

### Code Review Checklist

- [ ] **Closure capture audit**: For every closure in a `New()` function, verify what it captures. Flag any captures of the local `Model` variable
- [ ] **Mutable state location**: Confirm mutable state lives in pointer-to-struct fields, not directly on the value-type `Model`
- [ ] **Return value form**: When `New()` returns by value, document why closures don't depend on mutable `Model` state

### Anti-Pattern to Watch For

```go
func New() Model {
    m := Model{...}
    m.callback = func() {
        m.field = newValue  // DANGEROUS: closure sees stale m
    }
    return m  // Copy is made here
}
```

### Safe Patterns

1. **Pointer-wrapped mutable state**: Store shared state in `*SomeStruct` field — pointer is copied but both copies point to same heap object
2. **External state synchronization**: Set mutable fields on the shared struct before each use (the pattern used in this fix)
3. **Immutable closures**: If closures only read state that never changes after construction, capture is safe — document the invariant

### Testing Strategy

Write tests that mutate the returned model's state, then verify closures see the updated values:

```go
func TestClosureMutationVisibility(t *testing.T) {
    m := New()
    m.cursorLine = 2  // Mutate caller's copy
    suggestions := m.GetSuggestions("x")
    // Verify suggestions are filtered by line 2, not line 0
}
```

## Related Documentation

- [Locale formatting bypass in TUI](../ui-bugs/locale-formatting-bypass-in-tui.md) — identical closure/value-type pattern in the same codebase
- [Bubble Tea v2 migration fixes](../ui-bugs/bubble-tea-v2-migration-selection-undo-clipboard-fixes.md) — pointer vs value receiver analysis for safe closure capture
- [Ctrl-O stale state](../ui-bugs/ctrl-o-stale-state-and-unsaved-changes-detection.md) — stale state leaking across operation boundaries
- [TUI mode transitions](../ui-bugs/tui-mode-transitions-formatter-indexing-and-bracketed-paste-fixes.md) — centralized state reset patterns

### Feature Context

- Plan: `docs/plans/2026-03-04-feat-nl-autosuggest-examples-plan.md` (Phase 5)
- Brainstorm: `docs/brainstorms/2026-03-04-nl-autosuggest-examples-brainstorm.md`
