---
title: "Fix frontmatter editing persistence, structural edits, and word navigation help"
date: 2026-02-22
category: ui-bugs
tags: [frontmatter, editing, navigation, yaml, keyboard, macos, rawSource, catwalk]
severity: medium
component: cmd/calcmark/tui/editor
symptoms:
  - Frontmatter variable edits revert when navigating with arrow keys after Ctrl+F insert
  - Cannot press Enter to add new lines inside YAML frontmatter block
  - Ctrl+Left/Right word navigation appears broken on macOS (intercepted by Mission Control)
root_causes:
  - "updateCurrentLine() returns early for frontmatter lines (targetLine < 0), never persisting edits"
  - "Frontmatter.Serialize() reconstructs YAML from parsed maps, destroying structural edits"
  - "Help overlay shows Ctrl+Arrow but macOS intercepts these system-wide for Mission Control"
resolution_type: code-fix
files_changed:
  - spec/document/frontmatter.go
  - spec/document/frontmatter_test.go
  - cmd/calcmark/tui/editor/editing.go
  - cmd/calcmark/tui/editor/key_dispatch.go
  - cmd/calcmark/tui/editor/help_overlay.go
  - cmd/calcmark/tui/editor/command_menu.go
  - cmd/calcmark/tui/editor/model.go
  - cmd/calcmark/tui/editor/frontmatter_test.go
  - cmd/calcmark/tui/editor/word_nav_dispatch_test.go
  - cmd/calcmark/tui/editor/testdata/frontmatter_editing
  - cmd/calcmark/tui/editor/testdata/frontmatter_insert
related:
  - docs/solutions/ui-bugs/tui-editor-rendering-divider-status-bar-error-line.md
  - cmd/calcmark/tui/editor/TESTING.md
---

# Frontmatter Editing and Keyboard Dispatch Fixes

## Problem

Three related bugs in the TUI editor's frontmatter and keyboard handling:

1. **Edits revert on navigation**: After `Ctrl+F` to insert frontmatter, changing a variable name (e.g., `my_var` to `growth_rate`), the edit reverts as soon as the user navigates with arrow keys.

2. **Enter key doesn't work in frontmatter**: Pressing Enter on a frontmatter line does nothing — the new line is immediately lost because the YAML is reconstructed from parsed maps.

3. **Word navigation appears broken**: `Ctrl+Left/Right` for word-by-word navigation doesn't work on macOS. The help overlay shows `Ctrl+Arrow` as the binding, but macOS Mission Control intercepts these keys system-wide.

## Investigation

### Bug 1: Edit Revert

Traced the flow: user edits `editBuf` on a frontmatter line, then presses Down arrow. `saveCurrentLineAndMoveTo()` calls `updateCurrentLine(editBuf)`. Inside `updateCurrentLine()`:

```go
// editing.go — BEFORE fix
fmCount := m.frontmatterLineCount()
targetLine := m.cursorLine - fmCount

if targetLine < 0 {
    return  // <-- No-op! Frontmatter edits silently dropped
}
```

Frontmatter lines have `cursorLine < fmCount`, so `targetLine` is negative. The function returned without saving.

### Bug 2: Structural Edits Lost

Regular document blocks store raw `[]string` source lines preserved exactly through edits. Frontmatter follows a different path:

```
User edits line → updateCurrentLine() rebuilds doc → NewDocument() parses YAML
→ GetLines() calls fm.Serialize() → Serialize() reconstructs YAML from maps
→ Original formatting (empty lines, whitespace, comments) is gone
```

The `Serialize()` method iterated `Globals` and `Exchange` maps to produce fresh YAML, destroying any structural edits the user made.

### Bug 3: Word Navigation

Wrote a dispatch test sending all 8 key variants through `Update()`:

| Key Message | Dispatches? |
|---|---|
| `tea.KeyCtrlRight` | Yes |
| `tea.KeyCtrlLeft` | Yes |
| `tea.KeyRight` + `Alt: true` | Yes |
| `tea.KeyLeft` + `Alt: true` | Yes |
| `Rune('f')` + `Alt: true` | Yes |
| `Rune('b')` + `Alt: true` | Yes |
| `Rune('F')` + `Alt: true` | Yes |
| `Rune('B')` + `Alt: true` | Yes |

All 8 dispatch correctly. The code is correct — macOS simply intercepts `Ctrl+Arrow` before it reaches the terminal. The help overlay was misleading users to press keys that can't work.

## Solution

### Fix 1: Persist Frontmatter Edits (`editing.go`)

When `targetLine < 0`, rebuild the entire document with the modified line:

```go
if targetLine < 0 {
    lines := m.GetLines()
    if m.cursorLine >= len(lines) {
        return
    }
    lines[m.cursorLine] = newContent
    content := strings.Join(lines, "\n") + "\n"
    newDoc, err := document.NewDocument(content)
    if err != nil {
        m.frontmatterErr = err
        return
    }
    m.frontmatterErr = nil
    m.doc = newDoc
    m.eval = implDoc.NewEvaluator()
    _ = m.eval.Evaluate(m.doc)
    m.autoPinVariables()
    return
}
```

### Fix 2: rawSource Preservation (`spec/document/frontmatter.go`)

Added `rawSource string` field to `Frontmatter` struct that captures the original YAML text:

```go
type Frontmatter struct {
    Exchange  map[string]decimal.Decimal
    Globals   map[string]string
    rawSource string // preserves exact text including --- delimiters
}
```

Three coordinated changes:

- **`ParseFrontmatter()`**: Captures raw text during parsing:
  ```go
  fm.rawSource = strings.Join(lines[0:closeIdx+1], "\n") + "\n"
  ```

- **`Serialize()`**: Returns raw source when available:
  ```go
  if f.rawSource != "" {
      return f.rawSource + "\n"  // Add CommonMark blank line
  }
  // Fall back to reconstruction...
  ```

- **`SetGlobal()` / `SetExchangeRate()`**: Clear rawSource on programmatic modification:
  ```go
  f.rawSource = ""  // Invalidate — Serialize() will reconstruct
  ```

### Fix 3: Help Accelerators (`help_overlay.go`, `command_menu.go`)

Updated displayed accelerators from `Ctrl+Arrow` to `Opt+Arrow / Opt+B/F`:

```go
{Name: "Word Left", Accelerator: "Opt+<- / Opt+B", Kind: HelpAdvisory},
{Name: "Word Right", Accelerator: "Opt+-> / Opt+F", Kind: HelpAdvisory},
```

## Tests

### Catwalk Tests (data-driven, full interaction flow)

- `testdata/frontmatter_editing`: Variable rename (my_var to growth_rate), Enter key inserting new line, typing new variable, verifying persistence after navigation
- `testdata/frontmatter_insert`: Ctrl+F insertion flow

### Unit Tests

- `TestUpdateCurrentLineFrontmatterPersists`: Direct test of updateCurrentLine for frontmatter
- `TestFrontmatterEditSurvivesNavigation`: Full Ctrl+F, edit, navigate, verify flow
- `TestFrontmatterEnterKeyInsertsLine`: Enter on frontmatter line persists
- `TestFrontmatterEnterAndTypeNewVariable`: Enter + type new variable, both persist
- `TestFrontmatter_RawSourcePreservation`: Parse/serialize round-trip preserves exact text
- `TestFrontmatter_RawSourceClearedOnModification`: SetGlobal clears rawSource
- `TestFrontmatter_RawSourceRoundTrip`: Double round-trip produces identical output
- `TestWordNavigationDispatch`: All 8 key variants dispatch correctly through Update()

## Prevention Strategies

### 1. Frontmatter as First-Class Citizen

Frontmatter lines exist outside the block model. Any function that iterates blocks must also handle frontmatter lines, or explicitly document that it doesn't. When `targetLine` arithmetic produces a negative value, that's a signal to handle frontmatter — not to bail out silently.

**Principle**: Never silently discard user edits. A no-op return from a save function is a bug.

### 2. Lossless Round-Trips for Raw Text

When users edit text directly, the parse-serialize cycle must be lossless for structural content. The `rawSource` pattern works: preserve the original text alongside parsed data, and only reconstruct when the data is modified programmatically.

**Principle**: Parse for understanding, serialize from source. Keep the original text as ground truth.

### 3. Platform-Aware Help Text

On macOS, `Ctrl+Arrow` is a system shortcut. Help text must show bindings that actually work on the user's platform. Ideally, generate help from the keybinding configuration rather than hardcoding strings.

**Principle**: Help text is a specification — verify it against reality with tests.

### 4. TDD with Catwalk for UI Bugs

Every user-facing TUI bug should have a catwalk test that:
1. Reproduces the exact key sequence that triggered the bug
2. Fails before the fix (proving the bug exists)
3. Passes after the fix (validating the solution)
4. Prevents regression permanently

The `editBuf == ""` sentinel for "not loaded" is a known gotcha — tests that clear a line to empty will see it reload from the document. Edit in-place (backspace + retype) instead of clearing.
