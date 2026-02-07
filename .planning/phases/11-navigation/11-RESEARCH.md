# Phase 11: Navigation - Research

**Researched:** 2026-02-07
**Domain:** TUI Editor Navigation Keys
**Confidence:** HIGH

## Summary

Phase 11 implements word, line, and document movement keyboard shortcuts for the CalcMark TUI editor. Research reveals that most navigation functionality is already implemented, with clear gaps for document-level navigation (Ctrl+Home, Ctrl+End) and keybinding conflicts that need resolution.

The editor already handles:
- Word navigation via Ctrl+Arrow and Alt+B/F (NAV-01, NAV-02 - DONE)
- Home/End keys for line start/end (NAV-03, NAV-04 - PARTIALLY DONE)

Remaining work:
- Add Ctrl+Home/Ctrl+End for document navigation (NAV-05, NAV-06 - NOT IMPLEMENTED)
- Resolve Ctrl+A/Ctrl+E keybinding conflicts (currently Ctrl+E = Export)
- Add comprehensive catwalk tests for all navigation requirements

**Primary recommendation:** Add document navigation handlers and resolve keybinding conflicts, keeping export accessible via alternative binding.

## Current Implementation Status

### Already Implemented

| Requirement | Key Binding | Handler | Status |
|-------------|------------|---------|--------|
| NAV-01 | Ctrl+Left, Alt+B | `handleCtrlLeftKey()` | COMPLETE |
| NAV-02 | Ctrl+Right, Alt+F | `handleCtrlRightKey()` | COMPLETE |
| NAV-03 | Home | `handleHomeKey()` | PARTIAL (Ctrl+A missing) |
| NAV-04 | End | `handleEndKey()` | PARTIAL (Ctrl+E conflicts with Export) |
| NAV-05 | Ctrl+Home | - | NOT IMPLEMENTED |
| NAV-06 | Ctrl+End | - | NOT IMPLEMENTED |

### Code Locations

**Key handling:** `/cmd/calcmark/tui/editor/model.go`
- Line 443-513: `handleDefaultKey()` - main key dispatch
- Line 540-601: Navigation handlers (Up/Down/Left/Right/Home/End)
- Line 604-680: Word navigation handlers (`handleCtrlLeftKey()`, `handleCtrlRightKey()`)

**Key definitions:** `/cmd/calcmark/tui/shared/keys.go`
- Line 87-94: LineStart binding (ctrl+a, home)
- Line 103-110: Home/End bindings for document navigation (ctrl+home, ctrl+end)

**Existing tests:**
- `testdata/cursor_navigation` - tests Home/End, arrow wrapping
- `testdata/word_movement` - tests Ctrl+Left/Right word boundaries
- `arrow_navigation_test.go` - unit tests for arrow key navigation

## Architecture Patterns

### Key Handler Pattern

All navigation keys follow this pattern in `handleDefaultKey()`:

```go
func (m Model) handleDefaultKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
    switch msg.Type {
    case tea.KeyHome:
        return m.handleHomeKey()
    case tea.KeyEnd:
        return m.handleEndKey()
    // ... more cases
    }
}
```

Navigation handlers follow a consistent structure:

```go
func (m Model) handleHomeKey() (tea.Model, tea.Cmd) {
    m.loadCurrentLineIntoEditBuffer()  // Ensure editBuf is populated
    m.cursorCol = 0                     // Perform navigation
    return m, nil                       // Return updated model
}
```

### Word Boundary Algorithm

The existing word navigation uses unicode-aware boundary detection:

```go
// Skip whitespace backwards, then skip word characters
for col > 0 && unicode.IsSpace(runes[col-1]) {
    col--
}
for col > 0 && !unicode.IsSpace(runes[col-1]) && !unicode.IsPunct(runes[col-1]) {
    col--
}
```

### Document Navigation Pattern (To Implement)

Document start/end navigation should:
1. Save current line (if editing)
2. Move cursor to first/last line
3. Set column to 0 (for document start) or end-of-line (for document end)
4. Adjust scroll to keep cursor visible

```go
func (m Model) handleCtrlHomeKey() (tea.Model, tea.Cmd) {
    m.loadCurrentLineIntoEditBuffer()
    m.saveCurrentLineAndMoveTo(0)
    m.cursorCol = 0
    return m, nil
}

func (m Model) handleCtrlEndKey() (tea.Model, tea.Cmd) {
    m.loadCurrentLineIntoEditBuffer()
    lastLine := m.TotalLines() - 1
    if lastLine < 0 {
        lastLine = 0
    }
    m.saveCurrentLineAndMoveTo(lastLine)
    m.cursorCol = len(m.editBuf)
    return m, nil
}
```

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Word boundary detection | Custom regex | `unicode.IsSpace/IsPunct` | Already implemented correctly |
| Scroll adjustment | Manual calculation | `adjustScrollForCursor()` | Handles margin correctly |
| Line wrapping navigation | Per-key wrapping | `saveCurrentLineAndMoveTo()` | Maintains editBuf state |

**Key insight:** All cursor movement must go through state transition helpers to maintain editBuf consistency.

## Common Pitfalls

### Pitfall 1: Keybinding Conflicts

**What goes wrong:** Ctrl+E is currently used for Export (line 395-398 in model.go), but NAV-04 requires it for line-end navigation.

**Why it happens:** Ctrl+E is a common Emacs binding for both "end of line" and "export" depending on context.

**How to avoid:**
- Option A: Remove Ctrl+E from export, use only menu/command
- Option B: Keep Ctrl+E for export, document that End key is the only line-end option
- Option C: Make Ctrl+E context-sensitive (export in command mode, line-end in edit mode)

**Recommendation:** Option A - Remove Ctrl+E from export. Users can use Ctrl+Shift+E or a command. Standard readline bindings (Ctrl+A/E) are more intuitive for editing.

**Warning signs:** Tests failing because key does wrong action.

### Pitfall 2: EditBuf Not Loaded

**What goes wrong:** Navigation changes cursor position but editBuf is empty, causing incorrect cursor behavior.

**Why it happens:** Some navigation paths skip `loadCurrentLineIntoEditBuffer()`.

**How to avoid:** ALWAYS call `loadCurrentLineIntoEditBuffer()` at start of navigation handler.

**Warning signs:** Cursor moves but typing doesn't work, or cursor jumps to wrong position.

### Pitfall 3: Scroll Not Adjusted

**What goes wrong:** Document-level navigation moves cursor off-screen.

**Why it happens:** Large jumps (Ctrl+Home/End) may exceed visible viewport.

**How to avoid:** Call `adjustScrollForCursor()` or use `saveCurrentLineAndMoveTo()` which includes it.

**Warning signs:** Cursor invisible after navigation, content doesn't scroll.

### Pitfall 4: macOS Terminal Key Capture

**What goes wrong:** Ctrl+Arrow keys don't work on macOS because Terminal.app captures them.

**Why it happens:** macOS Terminal uses Ctrl+Arrow for desktop switching.

**How to avoid:** Already handled via Alt+B/F alternatives (STATE.md decision [v1.0]).

**Warning signs:** Navigation works in tests but not on real macOS Terminal.

## Code Examples

### Document Start Navigation (To Implement)

```go
// Source: New implementation based on existing handleHomeKey pattern
case tea.KeyCtrlHome:
    return m.handleCtrlHomeKey()

func (m Model) handleCtrlHomeKey() (tea.Model, tea.Cmd) {
    m.loadCurrentLineIntoEditBuffer()
    // Save current line, move to first line
    m.saveCurrentLineAndMoveTo(0)
    m.cursorCol = 0
    return m, nil
}
```

### Document End Navigation (To Implement)

```go
// Source: New implementation based on existing handleEndKey pattern
case tea.KeyCtrlEnd:
    return m.handleCtrlEndKey()

func (m Model) handleCtrlEndKey() (tea.Model, tea.Cmd) {
    m.loadCurrentLineIntoEditBuffer()
    lastLine := m.TotalLines() - 1
    if lastLine < 0 {
        lastLine = 0
    }
    m.saveCurrentLineAndMoveTo(lastLine)
    m.cursorCol = len(m.editBuf)
    return m, nil
}
```

### Ctrl+A for Line Start (To Implement)

```go
// Add to handleDefaultKey switch statement
case tea.KeyCtrlA:
    return m.handleHomeKey()  // Reuse existing handler
```

### Catwalk Test for Document Navigation

```
# Test: Document navigation with Ctrl+Home and Ctrl+End
# Document has 20+ lines to verify scroll behavior

run observe=debug
----
-- debug:
mode=StateDefault cursorLine=0 ...

# Move to middle of document
run observe=debug
key down
key down
key down
key down
key down
----
-- debug:
mode=StateDefault cursorLine=5 ...

# Ctrl+End should jump to last line
run observe=debug
key ctrl+end
----
-- debug:
mode=StateDefault cursorLine=19 cursorCol=... scrollOffset=...

# Ctrl+Home should jump back to first line
run observe=debug
key ctrl+home
----
-- debug:
mode=StateDefault cursorLine=0 cursorCol=0 scrollOffset=0 ...
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Vim-style modal editing | Always-editing mode | v1.0 | Users can type immediately without mode switch |
| Custom key handling | Bubble Tea KeyMsg types | v1.0 | Consistent key handling with ecosystem |
| Single navigation binding | Multiple bindings per action | v1.0 | Better cross-platform compatibility |

**Deprecated/outdated:**
- Vim-style 'j', 'k', 'h', 'l' navigation (removed - user is always in edit mode)
- Modal state tracking for navigation (simplified to InputState for UI overlays only)

## Open Questions

### 1. Ctrl+E Conflict Resolution

**What we know:** Ctrl+E is currently Export (model.go:395-398), but NAV-04 requires it for line-end.

**What's unclear:** User preference for export accessibility vs standard readline bindings.

**Recommendation:** Remove Ctrl+E from export. Export is less frequent than line navigation. Use Ctrl+Shift+E or menu command for export.

### 2. Ctrl+A Availability

**What we know:** Ctrl+A is not currently used in the editor. NAV-03 requires it for line-start.

**What's unclear:** Whether Ctrl+A might be expected for "select all" in future phases.

**Recommendation:** Implement Ctrl+A for line-start now. "Select all" can use Ctrl+Shift+A when selection is implemented.

## Implementation Checklist

1. [ ] Add `tea.KeyCtrlHome` handler in `handleDefaultKey()` switch
2. [ ] Add `tea.KeyCtrlEnd` handler in `handleDefaultKey()` switch
3. [ ] Implement `handleCtrlHomeKey()` - move to line 0, col 0
4. [ ] Implement `handleCtrlEndKey()` - move to last line, end of line
5. [ ] Add `tea.KeyCtrlA` case that calls `handleHomeKey()` (reuse existing)
6. [ ] Resolve Ctrl+E conflict:
   - Remove `tea.KeyCtrlE` from export (line 395-398)
   - Add `tea.KeyCtrlE` case that calls `handleEndKey()`
   - Provide alternative for export (Ctrl+Shift+E or command only)
7. [ ] Create catwalk test file `testdata/document_navigation`
8. [ ] Create catwalk test file `testdata/line_nav_ctrlae` (for Ctrl+A/E)
9. [ ] Add test function in `catwalk_test.go` for new tests
10. [ ] Update help text/keybinding documentation

## Sources

### Primary (HIGH confidence)

- `/cmd/calcmark/tui/editor/model.go` - Current key handling implementation
- `/cmd/calcmark/tui/shared/keys.go` - KeyMap definitions
- `/cmd/calcmark/tui/editor/TESTING.md` - Catwalk testing patterns
- Bubble Tea v1.3.10 key.go - Available KeyType constants

### Secondary (MEDIUM confidence)

- `/cmd/calcmark/tui/editor/testdata/word_movement` - Existing word nav test pattern
- `/cmd/calcmark/tui/editor/testdata/cursor_navigation` - Existing cursor nav test pattern

### Tertiary (LOW confidence)

- macOS Terminal key capture behavior (documented in STATE.md)

## Metadata

**Confidence breakdown:**
- Current implementation: HIGH - direct code inspection
- Key handler pattern: HIGH - verified in codebase
- Conflict resolution: MEDIUM - user preference unclear
- macOS compatibility: MEDIUM - documented but not tested

**Research date:** 2026-02-07
**Valid until:** 2026-03-07 (stable domain, 30 days)
