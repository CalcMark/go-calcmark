# Fix Clipboard and Selection Support in TUI Editor

## Overview

The TUI editor's clipboard functionality (select/select all/copy/paste) is not working as expected. Investigation reveals that while basic clipboard operations exist, the primary issue is the lack of Shift+Arrow key selection support due to Bubble Tea framework limitations.

## Problem Statement

Users cannot perform standard text selection operations in the TUI editor:
1. **No Shift+Arrow Selection**: Cannot select text using Shift+Arrow keys (most critical)
2. **No Visual Feedback**: Selected text is not visually highlighted
3. **Inconsistent Key Bindings**: Platform differences not properly handled (Cmd vs Ctrl)
4. **Limited Error Handling**: Clipboard operations fail silently

## Current State Analysis

### Existing Implementation
- `clipboard.go`: Has cut/copy/paste functions using github.com/atotto/clipboard
- `selection.go`: Has selection state management (anchors, ranges)
- Key bindings: Ctrl-A (select all), Ctrl-C (copy), Ctrl-V (paste), Ctrl-X (cut)

### Root Cause
Bubble Tea's `tea.KeyMsg` does not include Shift key state, making Shift+Arrow selection impossible to detect directly.

## Proposed Solution

### Phase 1: Alternative Selection Mechanism
Since Shift+Arrow is blocked by framework limitations, implement alternative selection methods:

1. **Visual Selection Mode** (Vim-style)
   - `Ctrl-Space` or `v`: Enter selection mode
   - Arrow keys extend selection while in mode
   - Any other key exits selection mode
   - Provides clear visual feedback

2. **Click-and-Drag Selection** (if mouse support enabled)
   - Leverage Bubble Tea's mouse events
   - Track mouse down/up for selection range
   - Visual highlight during drag

### Phase 2: Visual Feedback Enhancement
1. **Selection Highlighting**
   - Implement proper background color for selected text
   - Use theme colors: `SelectionBg`, `SelectionFg`
   - Update `renderLine` to apply selection styles

2. **Status Bar Indicators**
   - Show "Selection: X chars" in status line
   - Visual mode indicator: "-- VISUAL --"

### Phase 3: Cross-Platform Consistency
1. **Platform Detection**
   - Detect OS at startup
   - Map appropriate modifier keys:
     - macOS: Cmd for primary, Option for word nav
     - Linux/Windows: Ctrl for primary, Ctrl for word nav

2. **Key Binding Normalization**
   - Create platform-specific key maps
   - Consistent behavior across platforms

### Phase 4: Error Handling
1. **Clipboard Failures**
   - Catch clipboard errors
   - Show user-friendly messages
   - Provide fallback options

2. **Selection Validation**
   - Check selection bounds
   - Handle edge cases (empty selection, out of bounds)

## Technical Approach

### File Changes Required

1. **selection.go**
   - Add `visualMode` field to track selection mode
   - Implement `StartVisualMode()`, `ExtendSelection()`, `ExitVisualMode()`
   - Update selection rendering logic

2. **key_dispatch.go**
   - Add visual mode key handling
   - Route selection extension keys when in visual mode
   - Handle mode transitions

3. **clipboard.go**
   - Add error handling and user feedback
   - Implement clipboard validation
   - Add retry logic for transient failures

4. **render.go**
   - Update line rendering to show selection highlights
   - Apply theme colors for selected text
   - Handle partial line selections

5. **model.go**
   - Add `visualMode` to Model struct
   - Initialize selection state properly

### Implementation Details

```go
// selection.go additions
type SelectionMode int

const (
    SelectionNone SelectionMode = iota
    SelectionVisual
    SelectionLine
    SelectionBlock
)

func (m *Model) StartVisualMode() {
    m.selectionMode = SelectionVisual
    m.SetSelectionAnchor(m.cursorLine, m.cursorCol)
}

func (m *Model) ExtendSelection(line, col int) {
    // Update selection end point
    // Calculate selected range
}
```

```go
// key_dispatch.go visual mode handling
case tea.KeyCtrlSpace:
    if m.selectionMode == SelectionNone {
        m.StartVisualMode()
    } else {
        m.ExitVisualMode()
    }

case tea.KeyLeft, tea.KeyRight, tea.KeyUp, tea.KeyDown:
    if m.selectionMode == SelectionVisual {
        // Extend selection
        m.ExtendSelection(newLine, newCol)
    } else {
        // Normal cursor movement
    }
```

## Testing Strategy

### Catwalk Tests
Create comprehensive tests in `testdata/`:

1. **visual_mode_selection**
   - Test entering/exiting visual mode
   - Test selection extension in all directions
   - Test copy/paste of visual selection

2. **selection_edge_cases**
   - Empty selection
   - Full document selection
   - Multi-line selection
   - Selection at document boundaries

3. **clipboard_operations**
   - Cut/copy/paste in visual mode
   - Select all and copy
   - Paste over selection

### Unit Tests
1. Test selection range calculations
2. Test clipboard error handling
3. Test platform-specific key mappings

## Acceptance Criteria

1. **Shift+Arrow Selection** (implemented via Bubble Tea v2 ModShift support)
   - [x] Shift+Arrow keys select text (up/down/left/right)
   - [x] Shift+Home/End select to line boundaries
   - [x] Shift+Ctrl+Home/End select to document boundaries
   - [x] Shift+Ctrl+Left/Right select by word
   - [x] Selected text is visually highlighted
   - [x] Copy/cut work on selection

2. **Visual Feedback**
   - [x] Selected text has distinct background color (pre-existing)
   - [x] Status bar shows selection character count
   - [x] Plain arrow keys clear selection

3. **Selection-Aware Editing**
   - [x] Typing replaces selected text
   - [x] Backspace deletes selected text
   - [x] Delete key deletes selected text
   - [x] Enter replaces selected text with newline

4. **Clipboard Operations**
   - [x] Ctrl-C copies selected text (pre-existing)
   - [x] Ctrl-X cuts selected text (pre-existing)
   - [x] Ctrl-V pastes at cursor (pre-existing)
   - [x] Ctrl-A selects all text (pre-existing)

5. **Cross-Platform** (deferred)
   - [ ] Works on macOS with Cmd keys
   - [ ] Works on Linux/Windows with Ctrl keys
   - [ ] Word navigation uses appropriate modifiers

6. **Error Handling** (deferred)
   - [ ] Clipboard failures show error message
   - [ ] Operations validate selection bounds
   - [ ] Graceful fallback for unsupported operations

## Migration Notes

- Existing clipboard shortcuts remain unchanged
- Visual mode is additive, doesn't break existing functionality
- Document new selection method in user guide

## References

- Bubble Tea key handling: github.com/charmbracelet/bubbletea
- Clipboard library: github.com/atotto/clipboard
- Similar TUI selection implementations: micro editor, vim