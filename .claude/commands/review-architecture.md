---
name: review-architecture
description: Review go-calcmark packages for code smells, architectural smells, and state management issues. Produces prioritized P0/P1/P2 findings.
allowed-tools:
  - Read
  - Grep
  - Glob
  - Bash
  - Task
---

Perform an architectural and code quality review of the go-calcmark codebase. Produce a prioritized report of findings classified as P0 (must fix — correctness, dependency violations, maintainability blockers), P1 (should fix — code smells, state management issues, readability problems), and P2 (nice to have — style, modernization, minor cleanup).

This review is reusable. Run it after any significant feature work, before a release, or when onboarding to audit health.

## Principles

These principles define what "clean" means for this project. Evaluate all findings against them:

1. **Dependency direction is sacred.** `spec/` never imports `impl/`. `impl/` never imports `cmd/`. `format/` never imports `cmd/`. No UI libraries (`bubbletea`, `lipgloss`, `charmbracelet`, `bubbles`) in `spec/` or `impl/`. Violations are always P0.

2. **Elm architecture in the TUI.** The editor uses Bubble Tea's Model→Update→View cycle. State should flow through the return chain, not through pointer-receiver side effects hidden inside value-receiver methods. Mutations in value-receiver methods that call pointer-receiver helpers are a smell — the returned Model may silently lose changes. Flag cases where `(m Model)` methods call `(m *Model)` methods that modify state.

3. **Pure functions where possible.** Functions that read state should not modify it. `hasUnsavedChanges()` should never flush buffers. `renderFoo()` should never mutate scroll offsets. If a render function needs to adjust scroll state, that belongs in the Update phase, not the View phase. Flag View-phase mutations as P0.

4. **Explicit, minimal state.** Every field on Model should earn its place. Derived state that can be computed from other fields should not be stored. Redundant state that must be kept in sync is a bug waiting to happen. Flag redundant or derived state fields.

5. **File size and function count.** Files over 1000 LOC are a smell. Files over 2000 LOC are a P1. Files over 3000 LOC are a P0 — they need to be split. Functions over 50 LOC are a smell. Functions over 100 LOC need extraction.

6. **Command dispatch consistency.** If the same action can be triggered from multiple places (keybinding, command menu, help overlay), all paths must go through the same dispatch function. Duplicate logic is a P1.

7. **Dead code.** Unused exported functions, unreachable branches, commented-out code, and TODO/FIXME markers older than 2 months are P2 individually but P1 in aggregate if there are more than 5.

8. **Test coverage gaps.** Untested public functions in core packages (`spec/`, `impl/`) are P1. Untested TUI handlers are P2 unless they modify files or state destructively (then P1).

9. **Error handling.** Silently swallowed errors (empty `if err != nil {}` blocks or `_ = someFunc()`) are P1. User-visible operations that don't report errors are P1.

10. **String-typed dispatch.** Switch statements on string literals for command dispatch (e.g., `case "Save":`) are fragile. If there's no compile-time guarantee that command names match between definition and dispatch, flag as P2. If a typo could cause silent failure, flag as P1.

## Review Process

### Phase 1: Dependency Audit

Verify the sacred dependency direction:

```bash
# spec/ must not import impl/ or cmd/ (excluding test files)
grep -rn "impl/" spec/ --include="*.go" | grep -v "_test.go" | grep -v "vendor/"

# impl/ must not import cmd/ or tui
grep -rn "cmd/" impl/ --include="*.go" | grep -v "_test.go" | grep -v "vendor/"

# Check for UI library leaks into core
grep -rn "bubbletea\|lipgloss\|charmbracelet\|bubbles" spec/ impl/ --include="*.go"

# format/ must not import cmd/
grep -rn "cmd/" format/ --include="*.go" | grep -v "_test.go"
```

Any matches (excluding known exceptions documented in CLAUDE.md) are P0.

### Phase 2: State Management Audit

In `cmd/calcmark/tui/editor/`:

1. **Value vs pointer receiver consistency.** List all methods on Model. Flag cases where a value-receiver `(m Model)` method calls a pointer-receiver `(m *Model)` method — the mutations from the pointer method are lost unless the value-receiver explicitly returns the modified model.

```bash
# Find all pointer-receiver methods on Model
grep -n "func (m \*Model)" cmd/calcmark/tui/editor/*.go

# Find all value-receiver methods on Model
grep -n "func (m Model)" cmd/calcmark/tui/editor/*.go
```

2. **View-phase mutations.** Read all `render*` functions in `view.go` and `*_overlay.go` files. Flag any that modify `m` fields (scroll offsets, state flags, etc.). These belong in Update, not View.

3. **Redundant state fields.** Read the Model struct definition. For each field, ask: can this be derived from other fields? Flag candidates.

4. **State reset completeness.** When entering overlay states (StateHelp, StateCommandMenu, StateFilePicker, StateSavePrompt), verify all relevant state is reset. Stale state from a previous overlay session is a bug.

### Phase 3: Code Scale Audit

```bash
# Files over 1000 LOC (smell)
find cmd/calcmark/tui/editor -name "*.go" ! -name "*_test.go" -exec wc -l {} + | sort -rn | head -20

# Functions over 50 lines
# For each large file, count function bodies
```

For files over 2000 LOC, propose concrete split boundaries based on logical cohesion.

### Phase 4: Dispatch Consistency Audit

Verify that all triggerable actions use the shared dispatch:

1. Read `executeCommandByName()` in `command_menu.go`
2. Compare with direct key handlers in `model.go` (Ctrl+S, Ctrl+O, Ctrl+N, Ctrl+E, Ctrl+Q, Ctrl+Z, Ctrl+Y)
3. Flag any case where the keybinding handler has logic that differs from the command dispatch for the same action

### Phase 5: Dead Code and Hygiene

```bash
# Unused functions (gopls reports these)
go vet ./cmd/calcmark/tui/editor/... 2>&1

# TODO/FIXME markers
grep -rn "TODO\|FIXME\|HACK\|XXX" --include="*.go" | grep -v vendor/

# Commented-out code blocks (3+ consecutive comment lines that look like code)
grep -n "^[[:space:]]*//" cmd/calcmark/tui/editor/*.go | head -50
```

### Phase 6: Error Handling

```bash
# Swallowed errors
grep -n "_ =" cmd/calcmark/tui/editor/*.go --include="*.go"

# Empty error blocks
grep -A1 "if err != nil" cmd/calcmark/tui/editor/*.go | grep -B1 "^[[:space:]]*}"
```

### Phase 7: String-Typed Dispatch Safety

Check if command names used in `executeCommandByName()` switch cases match the names defined in `EditorCommands` and `HelpCategories()`. A typo in any of these three locations causes a silent no-op.

```bash
# Extract all command names from dispatch
grep 'case "' cmd/calcmark/tui/editor/command_menu.go

# Extract all names from EditorCommands
grep 'Name:' cmd/calcmark/tui/editor/command_menu.go

# Extract all CommandName values from HelpCategories
grep 'CommandName:' cmd/calcmark/tui/editor/help_overlay.go
```

Cross-reference these three lists. Any mismatch is P1.

## Output Format

Present findings in this exact format:

```markdown
# Architecture Review: go-calcmark

**Date:** YYYY-MM-DD
**Scope:** [packages reviewed]
**Commit:** [short SHA]

## Summary

- P0 findings: N
- P1 findings: N
- P2 findings: N

## P0 — Must Fix

### [FINDING-ID]: [Title]
- **Location:** `file:line`
- **Principle violated:** [which principle from above]
- **Description:** [what's wrong]
- **Impact:** [why this matters]
- **Fix:** [concrete action]

## P1 — Should Fix

### [FINDING-ID]: [Title]
[same structure]

## P2 — Nice to Have

### [FINDING-ID]: [Title]
[same structure]

## Architecture Health Score

Rate each dimension 1-5 (5 = excellent):

| Dimension | Score | Notes |
|-----------|-------|-------|
| Dependency Direction | X/5 | |
| State Management | X/5 | |
| Code Organization | X/5 | |
| Test Coverage | X/5 | |
| Error Handling | X/5 | |
| Dead Code / Hygiene | X/5 | |

**Overall:** X/30
```

Use finding IDs like `P0-01`, `P1-03`, `P2-07` for easy reference in follow-up conversations.

## After the Review

After producing the report, ask the user which findings they want to address in this session. Do not start fixing anything automatically — the review is diagnostic, not prescriptive action.
