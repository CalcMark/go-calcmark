---
title: "fix: Keyboard navigation and selection state inconsistencies"
type: fix
status: active
date: 2026-03-08
issue: 38
---

# fix: Keyboard navigation and selection state inconsistencies

## Overview

Multiple keyboard navigation bugs in the TUI editor, primarily affecting Cmd+Arrow key combos in Ghostty. Cmd+Right opens Export dialog, Cmd+Left selects/duplicates text, Cmd+Down/Up don't navigate to document boundaries. Alt+Up/Down are unhandled (needed for Terminal.app fallback). Selection state inconsistencies during navigation compound these issues.

## Problem Statement

| Bug | Key combo | Expected | Actual |
|-----|-----------|----------|--------|
| 1 | Cmd+Right | End of line | Opens Export dialog |
| 2 | Cmd+Left | Start of line | Selects and duplicates text |
| 3 | Cmd+Down | End of document | Unknown/broken |
| 4 | Cmd+Up | Start of document | Unknown/broken |
| 5 | Alt+Up | Start of document | Unhandled (no-op) |
| 6 | Alt+Down | End of document | Unhandled (no-op) |

**Key discovery**: Cmd+A/C/V work fine (ModSuper detected for letter keys), but Cmd+Arrow is broken. This means Ghostty encodes Cmd+Arrow differently than Cmd+letter.

## Proposed Solution

### Phase 1: Debug infrastructure (--debug-keys flag)

Add a `--debug-keys` runtime flag that logs raw key event data to stderr. This reveals exactly what Ghostty sends for Cmd+Arrow combos.

#### `cmd/calcmark/main.go` or CLI flag setup

- [x] Add `--debug-keys` boolean flag
- [x] Pass flag value to editor Model via config
- [x] When enabled, log every `tea.KeyPressMsg` to stderr before dispatch

#### `cmd/calcmark/tui/editor/key_dispatch.go`

- [x] At top of `handleKey()`, if debug mode enabled, log: `Code`, `Mod`, `Text`, `String()` to stderr
- [x] Format: `[KEY] code=%d mod=%v text=%q str=%q`

### Phase 2: Diagnose and fix Cmd+Arrow dispatch

Once we see the raw key events, fix the dispatch. Likely scenarios:

**Scenario A**: Ghostty sends Cmd+Arrow with both `ModSuper` and `ModCtrl` set. The global Ctrl handler (line 31) checks `!ModSuper` which should exclude it, but if the modifier bits differ...

**Scenario B**: Ghostty sends Cmd+Arrow as a different key code (not `tea.KeyLeft`/`tea.KeyRight`) when combined with Super.

**Scenario C**: The key arrives as `msg.String()` match (e.g., `"ctrl+e"`) rather than the `msg.Code`/`msg.Mod` path, bypassing the Super block entirely.

#### `cmd/calcmark/tui/editor/key_dispatch.go`

- [ ] Based on debug output, add the correct dispatch path for Cmd+Arrow in Ghostty
- [ ] Ensure the global Ctrl+E handler (line 47) does NOT intercept Cmd+Right
- [ ] Ensure Cmd+Left maps to `handleHomeKey()`
- [ ] Ensure Cmd+Right maps to `handleEndKey()`
- [ ] Ensure Cmd+Up maps to `handleCtrlHomeKey()`
- [ ] Ensure Cmd+Down maps to `handleCtrlEndKey()`
- [ ] Ensure Shift+Cmd+Arrow variants extend selection correctly

### Phase 3: Add Alt+Up/Down handlers

Alt+Left/Right already handled (line 212-230). Add Alt+Up/Down for document boundary navigation.

#### `cmd/calcmark/tui/editor/key_dispatch.go`

- [x] In the Alt block (line 212-230), add:
  - `alt+up` → `handleCtrlHomeKey()` (document start)
  - `alt+down` → `handleCtrlEndKey()` (document end)
- [x] Also handle Shift+Alt+Up/Down for selection extension:
  - In Shift block (line 172-210), add `hasAlt` cases for Up/Down → `handleShiftCtrlHomeKey()` / `handleShiftCtrlEndKey()`

### Phase 4: Catwalk tests

Write data-driven tests for every bug. Catwalk v2 driver supports `cmd+left`, `cmd+right`, `cmd+up`, `cmd+down`, `alt+up`, `alt+down`, and `shift+` variants via `parseKeyV2()`.

#### Catwalk test files to populate

Empty placeholder directories already exist. Populate with test data:

- [ ] `testdata/cmd_arrow_navigation/` — Cmd+Arrow moves cursor to line/doc boundaries
- [ ] `testdata/cmd_arrow_bug/` — Cmd+Right does NOT open Export, Cmd+Left does NOT select/duplicate
- [ ] `testdata/cmd_shortcuts/` — Cmd+A/C/V/Z work correctly (regression guard)
- [ ] `testdata/shift_selection/` — Shift+Cmd+Arrow extends selection, Shift+Alt+Up/Down extends to doc boundaries

#### Test scenarios per file

**cmd_arrow_navigation:**
```
# Multi-line document setup, cursor at middle
# cmd+right → end of current line
# cmd+left → start of current line
# cmd+down → end of document
# cmd+up → start of document
```

**cmd_arrow_bug (regression guard):**
```
# Cmd+Right must NOT trigger Export
# After cmd+right, mode should still be StateDefault (mode=0)
# Cmd+Left must NOT alter text content
```

**shift_selection:**
```
# shift+cmd+right → select to end of line
# shift+cmd+left → select to start of line
# shift+cmd+down → select to end of document
# shift+cmd+up → select to start of document
# shift+alt+up → select to start of document (Terminal.app fallback)
# shift+alt+down → select to end of document
```

### Phase 5: Cross-terminal verification

- [ ] Test in Ghostty (primary target)
- [ ] Test in Terminal.app (Alt key fallbacks)
- [ ] Test in iTerm2 (hybrid behavior)
- [ ] Document any terminal-specific notes

## Acceptance Criteria

- [ ] `--debug-keys` flag logs raw key events to stderr
- [ ] Cmd+Right → end of line (not Export) in Ghostty
- [ ] Cmd+Left → start of line (no text duplication) in Ghostty
- [ ] Cmd+Down → end of document in Ghostty
- [ ] Cmd+Up → start of document in Ghostty
- [ ] Shift+Cmd+Arrow extends selection in all four directions
- [ ] Alt+Up → start of document (Terminal.app fallback)
- [ ] Alt+Down → end of document (Terminal.app fallback)
- [ ] Shift+Alt+Up/Down extends selection to document boundaries
- [ ] All previously empty catwalk test directories populated with tests
- [ ] `task test` passes
- [ ] `task quality` passes
- [ ] Tested in Ghostty, Terminal.app, and iTerm2

## Technical Considerations

- The global Ctrl handler (line 31-53) runs BEFORE the Super block (line 98). If a Cmd+Arrow key somehow matches `case 'e':` in the global handler, it triggers Export. The debug flag will reveal this.
- `prepareNavigation(extendSelection bool)` must be called correctly for all new Alt+Up/Down paths.
- Empty documents and single-line documents are edge cases — Cmd+Down on a single-line doc should go to end of that line.
- The `msg.String()` fallback (line 232) may catch keys that the modifier blocks miss — this could be part of the problem.

## Dependencies & Risks

- **Low risk**: Debug flag is additive, no behavior change
- **Medium risk**: Changing key dispatch could break other terminals. Mitigated by catwalk tests and cross-terminal verification.
- **Institutional learning**: `docs/solutions/ui-bugs/bubble-tea-v2-migration-selection-undo-clipboard-fixes.md` documents the exact modifier patterns to use. `docs/solutions/ui-bugs/frontmatter-editing-keyboard-dispatch-fixes.md` documents the 5-layer dispatch architecture.

## References

- Brainstorm: `docs/brainstorms/2026-03-08-fix-keyboard-nav-selection-bugs-brainstorm.md`
- Issue: #38
- Key dispatch: `cmd/calcmark/tui/editor/key_dispatch.go:15-289`
- Navigation handlers: `cmd/calcmark/tui/editor/navigation.go`
- Selection state: `cmd/calcmark/tui/editor/selection.go`
- Catwalk v2 driver: `cmd/calcmark/tui/editor/catwalkv2_test.go:463-532`
- Testing guide: `cmd/calcmark/tui/editor/TESTING.md`
- Prior art: `docs/solutions/ui-bugs/bubble-tea-v2-migration-selection-undo-clipboard-fixes.md`
- Prior art: `docs/solutions/ui-bugs/frontmatter-editing-keyboard-dispatch-fixes.md`
- Prior art: `docs/solutions/ui-bugs/missing-ctrl-n-keyboard-shortcut.md`
