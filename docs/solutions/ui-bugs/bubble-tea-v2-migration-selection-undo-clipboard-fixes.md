---
title: "Bubble Tea v1→v2 Migration with Selection, Undo, and Clipboard Fixes"
date: "2026-02-26"
problem_type: "integration-issue"
severity: "high"
components: ["TUI editor", "Bubble Tea framework", "lipgloss", "bubbles", "catwalk test driver", "text selection", "clipboard", "undo/redo system", "rendering pipeline"]
technologies: ["Go", "Bubble Tea v1→v2", "lipgloss v1→v2", "bubbles v0.21→v2", "catwalk", "ANSI rendering", "Kitty keyboard protocol"]
symptoms:
  - "Shift+Arrow selection impossible under Bubble Tea v1 (no modifier key state)"
  - "Selection highlighting invisible despite model tracking selection correctly"
  - "ANSI rendering corruption: rune column positions mapped to escape code bytes"
  - "Frontmatter undo corruption: editBuf held stale content after document rebuild"
  - "Clipboard paste injecting ANSI escape codes into document"
  - "Cmd+Right triggering export mode via stale Ctrl+E accelerator"
  - "Catwalk test infrastructure incompatible with v2 API"
root_causes:
  - "Bubble Tea v1 tea.KeyMsg had no Shift modifier state"
  - "renderLineWithSelection operated on pre-styled ANSI text, corrupting column arithmetic"
  - "editBuf loaded before redetectBlockTypes() which changes line numbering"
  - "Clipboard paste had no ANSI stripping or size limits"
  - "74 files needed API migration for new import paths, message types, and View signature"
resolution_time: "5 commits over ~3 days"
tags: ["framework-migration", "breaking-changes", "TDD", "TUI-editor", "regression-testing", "ANSI-rendering", "state-management"]
---

# Bubble Tea v1→v2 Migration with Selection, Undo, and Clipboard Fixes

## Problem Statement

The TUI editor needed Shift+Arrow text selection, which was impossible under Bubble Tea v1 because `tea.KeyMsg` did not expose modifier key state. The migration to Bubble Tea v2 (`charm.land/bubbletea/v2`) was a major refactor touching 125 files (+7,643/−1,893 lines) that also uncovered latent bugs in the rendering pipeline, undo system, and clipboard operations.

Three intertwined challenges:
1. **API migration**: Every file using Bubble Tea/Lip Gloss needed updating (new import paths, message types, declarative `View()`)
2. **Shift+Arrow selection**: The primary motivation — v2's `tea.KeyPressMsg` exposes `msg.Mod` for modifier detection
3. **Regression cascade**: Migration exposed rendering corruption, undo state corruption, and clipboard safety issues

## Investigation Steps

The work progressed across 5 commits (oldest to newest):

1. **`a0ee819`** — Core framework migration: updated all import paths, changed `View() string` to `View() tea.View`, replaced `tea.KeyMsg` with `tea.KeyPressMsg`, migrated mouse handling and program options.

2. **`3b4a3bb`** — Test migration: rewrote catwalk test driver for v2 compatibility, updated all test files from v1 API patterns, fixed Alt+b/f word navigation dispatch (check `Code` not `Text`), regenerated all catwalk expectations.

3. **`c06c59d`** — Shift+Arrow selection: added `ensureSelectionAnchor()`, `prepareNavigation(extendSelection bool)`, 12 new `handleShift*` handlers, macOS Cmd+Arrow support via `tea.ModSuper`.

4. **`48c2e6a`** — Visual highlighting fix: selection state was tracked but invisible on screen. Rewrote rendering pipeline in `view_lines.go` to work on raw text with explicit tint colors instead of pre-styled ANSI text.

5. **`cedfb92`** — Clipboard/undo hardening: fixed `handleCut` re-evaluation ordering, fixed undo/redo editBuf reload ordering, added ANSI stripping and 1MB paste limit, added frontmatter-aware `DeleteSelection`.

## Root Cause Analysis

### Selection was invisible despite being tracked in model state

`renderLineWithSelection()` applied selection styling to the cursor line first, then `renderLineWithCursor()` re-rendered the entire line from plain text, overwriting the selection highlighting. The function accepted pre-tinted ANSI text, so rune-based column arithmetic mapped to escape code bytes instead of visible characters.

**Fix**: Rewrote `renderLineWithCursor` to handle selection highlighting inline. Added `editLineSelectionRange()` to compute selection bounds. The cursor line renderer partitions content into segments (normal, selection, cursor) and batch-renders each with the correct style. Non-cursor lines receive raw text plus tint colors (not pre-styled text).

### Undo corrupted documents after frontmatter operations

Two interacting bugs:
1. `editBuf` was loaded from the document *before* `redetectBlockTypes()` rebuilt it. The rebuild via `NewDocument()` could change line numbering, so editBuf held content from pre-rebuild line numbering.
2. Clearing frontmatter `rawSource` before applying undo operations forced `Serialize()` to reconstruct from maps, which could change line count, causing operations to apply to wrong lines.

**Fix**: Reordered the undo handler: (1) apply operations, (2) rebuild document via `redetectBlockTypes()`, (3) **then** load editBuf. Set `m.editBufLoaded = false` before rebuild to prevent stale reads.

```go
// CRITICAL FIX: rebuild BEFORE loading editBuf
m.editBufLoaded = false
m.redetectBlockTypes()        // rebuilds document, may change line count
m.reEvaluate()
m.loadCurrentLineIntoEditBuffer()  // NOW safe to load
```

### Clipboard paste accepted ANSI escape codes

Pasted content from the system clipboard could contain ANSI escape sequences, corrupting the rune-based column arithmetic used throughout the rendering pipeline.

**Fix**: Added `stripANSI()` call and `maxPasteSize` (1MB) limit in `handlePaste`.

### Shift key detection was impossible in v1

Bubble Tea v1's `tea.KeyMsg` had no modifier information beyond `Alt`. `Shift+Up` was indistinguishable from plain `Up` at the application level.

**Fix**: Bubble Tea v2's `tea.KeyPressMsg` exposes `msg.Mod.Contains(tea.ModShift)`.

## Working Solution

### Key Dispatch Migration (`key_dispatch.go`)

```go
// v1 — type-based dispatch, no modifier detection
func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
    switch msg.Type {
    case tea.KeyUp:
        ...
    case tea.KeyRunes:
        return m.handleRuneInput(msg.Runes)
    }
}

// v2 — modifier checks + string-based dispatch
func (m Model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
    if msg.Mod.Contains(tea.ModShift) {
        switch msg.Code {
        case tea.KeyLeft:
            return m.handleShiftLeftKey()
        // ...
        }
    }
    switch msg.String() {
    case "up":
        ...
    default:
        if msg.Text != "" {
            return m.handleRuneInput([]rune(msg.Text))
        }
    }
}
```

### Unified Navigation Preamble (`navigation.go`)

The `prepareNavigation` pattern eliminated 30+ lines of duplicated preamble across all navigation handlers:

```go
func (m *Model) prepareNavigation(extendSelection bool) {
    if extendSelection {
        m.ensureSelectionAnchor()
    } else {
        m.ClearSelection()
    }
    m.undoManager.ForceBoundary()
    m.loadCurrentLineIntoEditBuffer()
}
```

Plain navigation: `m.prepareNavigation(false)`. Shift navigation: `m.prepareNavigation(true)`.

### Rendering Pipeline Fix (`view_lines.go`)

```go
// Before: accepted pre-styled ANSI text, column positions were wrong
func (m Model) renderLineWithSelection(lineNum int, lineText string) string

// After: accepts raw text + tint colors, applies styles last
func (m Model) renderLineWithSelection(lineNum int, rawText string, tintFg, tintBg color.Color) string
```

For the cursor line, `editLineSelectionRange()` computes selection bounds and the renderer partitions text into styled segments without mixing ANSI codes with column arithmetic.

### Clipboard Hardening (`clipboard.go`)

```go
func (m Model) handleCut() (tea.Model, tea.Cmd) {
    deletedText, cmd := m.DeleteSelection()
    m.modified = true
    m.reEvaluate()  // FIX: re-evaluate immediately, not after clipboard write
    clipboard.WriteAll(deletedText)
    ...
}

func (m Model) handlePaste() (tea.Model, tea.Cmd) {
    text := clipboard.ReadAll()
    if len(text) > maxPasteSize { return ... }  // 1MB limit
    text = stripANSI(text)                       // prevent ANSI injection
    ...
}
```

## Key Technical Insights

1. **V2's `KeyPressMsg.String()` uses different names**: Space is `"space"` not `" "`. Modifiers prefix the key: `"shift+left"`. Shift detection must happen *before* the main `switch msg.String()` block.

2. **Modifier detection must exclude Super on macOS**: The Kitty keyboard protocol sends `ModSuper` for Cmd. Guard Ctrl shortcuts with `!msg.Mod.Contains(tea.ModSuper)` to prevent Cmd+Arrow triggering Ctrl+Arrow behavior.

3. **ANSI codes and rune arithmetic don't mix**: ANSI escape codes are multi-byte sequences that don't correspond to visible characters. All column arithmetic must happen on plain text; styling is applied only at the final render step.

4. **`editBufLoaded` is a critical state flag**: It distinguishes "empty because the line is empty" from "hasn't been loaded yet". Set it to `false` before any document rebuild to prevent stale reads.

5. **Selection anchor uses -1 sentinel, not a boolean**: `selectionAnchorLine == -1` means "no selection", eliminating state synchronization bugs between a separate flag and coordinates.

6. **Frontmatter-aware `DeleteSelection`**: Selection deletion touching frontmatter requires an atomic `document.NewDocument()` rebuild because the spec layer manages frontmatter as a single unit.

7. **Lip Gloss v2 changed color types**: Colors moved from `lipgloss.Color` (string-based) to `image/color.Color` (Go standard library), which is why rendering functions accept `color.Color` parameters.

## Prevention Strategies

### Terminal Rendering: Three-Stage Pipeline

```
Raw Text → Position/Select (rune arithmetic) → Style (lipgloss) → ANSI Output
```

Never apply ANSI codes before position-dependent operations. Never pass styled text to functions that do column arithmetic.

### State Management: Reload After Mutations

Any operation that modifies `m.doc` (especially `redetectBlockTypes()`) must:
1. Set `m.editBufLoaded = false`
2. Perform the mutation
3. Then reload derived state (`loadCurrentLineIntoEditBuffer()`)

Use `OpDocReplace` for whole-document changes (frontmatter insertion/deletion) to avoid line-number mismatches in undo history.

### Framework Migration: Tests First

1. Rewrite test infrastructure before migrating application code
2. Validate each subsystem independently (key dispatch, rendering, state, selection)
3. Run the full test suite (`task test`) after every wave of changes
4. Never skip tests for "quick" framework changes

### Testing: Reproduce Before Fixing

Every user-facing TUI bug must have a catwalk test that:
1. Reproduces the exact key sequence
2. Proves the bug exists (test fails)
3. Validates the fix (test passes)

## Related Documentation

- [Bubble Tea v2 Upgrade Plan](../../plans/2026-02-24-bubble-tea-v2-upgrade-plan.md)
- [Clipboard/Selection Support Plan](../../plans/2026-02-24-fix-clipboard-selection-support-plan.md)
- [Frontmatter Stability Plan](../../plans/2026-02-24-frontmatter-stability-preview-alignment-plan.md)
- [Flat Line Buffer Brainstorm](../../brainstorms/2026-02-26-flat-line-buffer-brainstorm.md)
- [Overlay Compositing ANSI Fix](overlay-compositing-ansi-state-bleed-through.md)
- [Frontmatter Keyboard Dispatch Fixes](frontmatter-editing-keyboard-dispatch-fixes.md)
- [Ctrl+O Stale State Detection](ctrl-o-stale-state-and-unsaved-changes-detection.md)
- [Lipgloss Background Bleed-Through](lipgloss-background-bleed-through.md)
- [Catwalk Testing Guide](../../../../cmd/calcmark/tui/editor/TESTING.md)
