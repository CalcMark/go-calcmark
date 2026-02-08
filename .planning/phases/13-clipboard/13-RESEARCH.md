# Phase 13: Clipboard - Research

**Researched:** 2026-02-08
**Domain:** TUI text selection and system clipboard integration
**Confidence:** HIGH

## Summary

This phase adds text selection and clipboard operations (cut/copy/paste) to the CalcMark TUI editor. The research identifies the standard approach for implementing text selection in a custom TUI text editor, using the already-included `github.com/atotto/clipboard` library for system clipboard access.

The implementation requires two main components:
1. **Selection state management** - Tracking anchor and cursor positions to define a selection range
2. **Clipboard operations** - Using atotto/clipboard for cross-platform system clipboard access (already an indirect dependency)

The existing undo/redo infrastructure (Phase 12) provides the foundation for paste operations, which should be recorded as undoable edits. The prior decision `[11.2]: Ctrl+A not used for line-start (reserved for select-all in Phase 13)` clarifies that Ctrl+A is available for select-all.

**Primary recommendation:** Implement selection as anchor/cursor position pair with visual highlighting, integrate atotto/clipboard directly (promote from indirect to direct dependency), and model cut/paste as undo-compatible edit operations.

## Standard Stack

The established libraries/tools for this domain:

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| github.com/atotto/clipboard | v0.1.4 | Cross-platform clipboard access | Already indirect dependency, pure Go on macOS/Windows, simple API |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| github.com/charmbracelet/lipgloss | v1.1.1+ | Selection highlighting style | Already in use for all TUI styling |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| atotto/clipboard | golang.design/x/clipboard | Requires CGO on macOS/Linux, supports images (not needed) |
| Custom selection | charmbracelet/bubbles textarea | Would require refactoring to use textarea component |

**Installation:**
```bash
# Already indirect dep, promote to direct:
go get github.com/atotto/clipboard@v0.1.4
```

## Architecture Patterns

### Selection Model: Anchor + Cursor

The standard pattern for text selection uses two positions:

```
Selection = (AnchorLine, AnchorCol) to (CursorLine, CursorCol)

Examples:
- No selection: Anchor == Cursor (or Anchor == nil/-1,-1)
- Forward selection: Anchor < Cursor (user dragged/shifted right/down)
- Backward selection: Anchor > Cursor (user dragged/shifted left/up)
```

**Key principle:** Selection is always between anchor (fixed point) and cursor (moving point). When user holds Shift and moves cursor, anchor stays fixed.

### Recommended Model Additions

```go
// Add to Model struct in model.go:

// Selection state
selectionAnchorLine int  // Line of selection start, -1 if no selection
selectionAnchorCol  int  // Column of selection start

// Example helper methods:
// HasSelection() bool
// GetSelectionRange() (startLine, startCol, endLine, endCol int)
// ClearSelection()
// SelectAll()
// GetSelectedText() string
// DeleteSelection() string  // Returns deleted text for clipboard
```

### Pattern 1: Selection Range Normalization

**What:** Normalize anchor/cursor to always get start <= end
**When to use:** Before any operation on selection (get text, delete, highlight)
**Example:**
```go
// Source: Standard text editor pattern
func (m *Model) GetSelectionRange() (startLine, startCol, endLine, endCol int) {
    if m.selectionAnchorLine < 0 {
        return -1, -1, -1, -1 // No selection
    }

    // Normalize: ensure start is before end
    if m.selectionAnchorLine < m.cursorLine ||
       (m.selectionAnchorLine == m.cursorLine && m.selectionAnchorCol <= m.cursorCol) {
        return m.selectionAnchorLine, m.selectionAnchorCol, m.cursorLine, m.cursorCol
    }
    return m.cursorLine, m.cursorCol, m.selectionAnchorLine, m.selectionAnchorCol
}
```

### Pattern 2: Selection Highlighting in View

**What:** Apply inverted/highlighted style to selected text during rendering
**When to use:** In renderSourcePaneAligned when rendering each line
**Example:**
```go
// Source: Standard TUI editor pattern
func (m *Model) renderLineWithSelection(lineNum int, lineText string, width int) string {
    if !m.HasSelection() {
        return lineText // No selection, render normally
    }

    startLine, startCol, endLine, endCol := m.GetSelectionRange()

    // Check if this line is in selection range
    if lineNum < startLine || lineNum > endLine {
        return lineText // Line not selected
    }

    // Calculate selection bounds for this line
    selectStart := 0
    selectEnd := runeLen(lineText)

    if lineNum == startLine {
        selectStart = startCol
    }
    if lineNum == endLine {
        selectEnd = endCol
    }

    // Apply selection style to range
    // ... (split string, apply style to middle part)
}
```

### Pattern 3: Clipboard Operations as Edit Operations

**What:** Cut and Paste recorded as undoable edits
**When to use:** All clipboard operations
**Example:**
```go
// Paste creates an Insert operation:
op := EditOperation{
    Type:    OpInsert,
    Line:    m.cursorLine,
    Col:     m.cursorCol,
    OldText: "",
    NewText: pastedText,
    // ... cursor state
}

// Cut creates a Delete operation:
op := EditOperation{
    Type:    OpDelete,
    Line:    startLine,
    Col:     startCol,
    OldText: selectedText,
    NewText: "",
    // ... cursor state
}
```

### Anti-Patterns to Avoid

- **Storing selected text separately:** Don't duplicate the selected text in a field. Compute it from anchor/cursor positions on demand.
- **Complex selection state machine:** Keep it simple - just anchor position. null anchor = no selection.
- **Bypassing undo for clipboard ops:** Cut and paste MUST go through recordEdit() for undo to work.

## Don't Hand-Roll

Problems that look simple but have existing solutions:

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| System clipboard access | Custom pbcopy/xclip shell calls | atotto/clipboard | Cross-platform, handles edge cases |
| Selection style | Custom ANSI codes | lipgloss.NewStyle() | Consistent with existing styles |
| UTF-8 text slicing | byte indexing | runeSlice() already in model.go | UTF-8 safe, already battle-tested |

**Key insight:** Selection is conceptually simple but has edge cases around multi-line text, UTF-8 boundaries, and visual display that are easy to get wrong. Use established patterns and existing helpers.

## Common Pitfalls

### Pitfall 1: Selection Not Cleared on Cursor Movement

**What goes wrong:** User moves cursor without Shift, but selection persists
**Why it happens:** Forgot to clear selection in navigation handlers
**How to avoid:** Add `m.ClearSelection()` to ALL cursor movement functions that don't have Shift held
**Warning signs:** After selecting text with Shift+Arrow, pressing Arrow alone still shows selection

### Pitfall 2: Cut/Paste Not Integrated with Undo

**What goes wrong:** Ctrl+Z after paste doesn't undo the paste
**Why it happens:** Clipboard operations bypass recordEdit()
**How to avoid:** All edits (including from clipboard) must call recordEdit() per decision [12-03]
**Warning signs:** Pasted text can't be undone

### Pitfall 3: Selection Anchor Not Reset After Operation

**What goes wrong:** After cut/paste, selection still visually present
**Why it happens:** Anchor wasn't cleared after operation
**How to avoid:** ClearSelection() after cut, paste, or any editing operation
**Warning signs:** Ghost selection visible after cut

### Pitfall 4: Ctrl+C Conflict with Quit

**What goes wrong:** Ctrl+C is currently "quit" in the editor
**Why it happens:** Existing keybinding from Phase 1
**How to avoid:** Redefine Ctrl+C behavior: copy if selection exists, quit if no selection (or use Ctrl+Q for quit)
**Warning signs:** Can't copy text because it quits the app

**NOTE:** From model.go line 457-460, Ctrl+C is currently "standard Unix interrupt signal - quit immediately". The requirements state CLIP-03: "Ctrl+C copies selection to clipboard (when text selected)". This requires careful handling - copy when selection exists, quit otherwise.

### Pitfall 5: Clipboard Fails Silently on Linux

**What goes wrong:** Copy/paste does nothing on Linux
**Why it happens:** Linux requires xclip or xsel to be installed
**How to avoid:** Check `clipboard.Unsupported` on init, show warning in status bar
**Warning signs:** Works on macOS/Windows but not Linux

## Code Examples

Verified patterns from official sources:

### Reading from Clipboard (atotto/clipboard)
```go
// Source: https://pkg.go.dev/github.com/atotto/clipboard
import "github.com/atotto/clipboard"

func (m Model) handleCtrlV() (tea.Model, tea.Cmd) {
    text, err := clipboard.ReadAll()
    if err != nil {
        m.statusMsg = "Clipboard error"
        m.statusIsErr = true
        return m, nil
    }
    if text == "" {
        return m, nil // Nothing to paste
    }

    // Insert text at cursor (may be multi-line)
    // ... implementation using existing insert logic
}
```

### Writing to Clipboard (atotto/clipboard)
```go
// Source: https://pkg.go.dev/github.com/atotto/clipboard
func (m *Model) copySelectionToClipboard() error {
    if !m.HasSelection() {
        return nil // Nothing to copy
    }

    text := m.GetSelectedText()
    return clipboard.WriteAll(text)
}
```

### Checking Clipboard Availability
```go
// Source: https://pkg.go.dev/github.com/atotto/clipboard
import "github.com/atotto/clipboard"

func checkClipboardAvailable() bool {
    // Unsupported is set during init if clipboard is unavailable
    return !clipboard.Unsupported
}
```

### Selection Highlighting Style
```go
// Consistent with existing styles in config/styles.go
selectionStyle := lipgloss.NewStyle().
    Background(lipgloss.Color("240")).  // Gray background
    Foreground(lipgloss.Color("255"))   // White text
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Shell out to pbcopy/xclip | Use clipboard library | 2020+ | Simpler, more portable |
| Separate copy buffer (yankBuffer) | System clipboard integration | Phase 13 | Better UX, cross-app compatibility |
| Vim-style yank (internal) | Standard Ctrl+C/X/V | Phase 11.2 decisions | More intuitive for non-vim users |

**Deprecated/outdated:**
- `yankBuffer` field in Model: Currently used for vim-style yy/p commands. May be deprecated in favor of system clipboard, or kept for vim compatibility.

## Open Questions

Things that couldn't be fully resolved:

1. **What happens to existing yankBuffer?**
   - What we know: Model has `yankBuffer string` field used for vim-style yy/dd/p operations
   - What's unclear: Should this be removed, replaced with clipboard, or kept alongside?
   - Recommendation: Keep yankBuffer for internal line-level operations (dd/yy/p), use system clipboard for Ctrl+C/X/V. They serve different purposes.

2. **Shift+Arrow keybindings in bubbletea?**
   - What we know: Requirements only specify Ctrl+A for select-all, not Shift+Arrow for extending selection
   - What's unclear: Can bubbletea detect Shift modifier on arrow keys?
   - Recommendation: Start with Ctrl+A (select-all) only per requirements. Add Shift+Arrow if supported and time permits.

3. **Multi-line paste handling?**
   - What we know: Clipboard may contain multiple lines
   - What's unclear: Exact implementation for inserting multi-line content
   - Recommendation: Split pasted text by newlines, insert first part at cursor, insert remaining as new lines. Model as single multi-line insert for undo.

## Sources

### Primary (HIGH confidence)
- https://pkg.go.dev/github.com/atotto/clipboard - Official API documentation
- /Users/bitsbyme/projects/go-calcmark/cmd/calcmark/tui/editor/model.go - Existing cursor/editBuf implementation
- /Users/bitsbyme/projects/go-calcmark/cmd/calcmark/tui/editor/undo.go - EditOperation patterns

### Secondary (MEDIUM confidence)
- https://github.com/charmbracelet/bubbles/pull/825 - Reference implementation of selection in bubbles/textarea
- https://github.com/atotto/clipboard - GitHub README with platform support details

### Tertiary (LOW confidence)
- WebSearch results on TUI text selection patterns - General patterns, not verified for Go

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - atotto/clipboard is already in go.mod, well-documented
- Architecture: HIGH - Anchor/cursor selection is standard text editor pattern
- Pitfalls: HIGH - Based on existing codebase analysis (Ctrl+C conflict, undo integration)

**Research date:** 2026-02-08
**Valid until:** 60 days (stable domain, no rapid changes expected)
