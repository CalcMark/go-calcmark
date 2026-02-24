# Bubble Tea v2 Upgrade & Shift Selection Implementation Plan

## Overview

Upgrade the TUI editor from Bubble Tea v1.3.10 to v2.0.0 to enable native Shift+Arrow text selection, providing standard non-modal text editing behavior that users expect.

**Core Philosophy Change**: v2 shifts from imperative commands to declarative View fields. All terminal behavior configuration now lives in the `View()` method rather than scattered across initialization and runtime commands.

## Goals

1. **Upgrade to Bubble Tea v2** - Migrate all code to the new API
2. **Implement Shift+Arrow Selection** - Standard text selection without modal editing
3. **Remove Visual Mode** - Eliminate the Vim-style visual mode completely
4. **Maintain All Functionality** - Ensure no regressions in existing features
5. **Cross-Platform Support** - Ensure selection works on macOS, Linux, and Windows

## Phase 1: Preparation & Analysis

### 1.1 Backup Current State
- Create a new branch: `feature/bubble-tea-v2-upgrade`
- Document current key bindings and behavior
- Save current test baselines

### 1.2 Dependency Analysis
```bash
# Current dependencies to update:
- github.com/charmbracelet/bubbletea v1.3.10 → charm.land/bubbletea/v2 v2.0.0
- github.com/charmbracelet/bubbles → charm.land/bubbles/v2 (if used)
- github.com/charmbracelet/lipgloss → charm.land/lipgloss/v2 (required update)
```

**Note**: Both Bubble Tea and Lip Gloss moved to vanity domains at `charm.land`

### 1.3 Impact Assessment
Files that will need changes:
- **All editor files** (~50+ files in `cmd/calcmark/tui/editor/`)
- **Key handling**: `key_dispatch.go`, `navigation.go`
- **Model structure**: `model.go`
- **View rendering**: `view.go`, `view_*.go`
- **All test files**: `*_test.go`
- **App initialization**: `app.go`

## Phase 2: Core Migration

### 2.1 Update Import Paths
```go
// Before
import tea "github.com/charmbracelet/bubbletea"
import "github.com/charmbracelet/lipgloss"

// After
import tea "charm.land/bubbletea/v2"
import "charm.land/lipgloss/v2"
```

### 2.2 Update View Method Signature
```go
// Before
func (m Model) View() string {
    return m.renderView()
}

// After
func (m Model) View() tea.View {
    content := m.renderView()
    v := tea.NewView(content)

    // Set declarative view fields (replaces old program options)
    v.AltScreen = false  // We don't use alt screen
    v.MouseMode = tea.MouseModeCellMotion  // For mouse selection
    v.ReportFocus = false  // Focus tracking if needed
    v.DisableBracketedPasteMode = false  // Keep paste detection

    // Optional: Set window title
    // v.WindowTitle = "CalcMark Editor"

    return v
}
```

**Important**: All terminal behavior configuration now lives in View(), not in program initialization.

### 2.3 Update Key Message Handling

**CRITICAL**: `tea.KeyMsg` is now an interface. Use `tea.KeyPressMsg` for key presses.

```go
// Before (v1)
case tea.KeyMsg:
    switch msg.Type {
    case tea.KeyUp:
        return m.handleUpKey()
    case tea.KeyCtrlC:
        return m.handleCopy()
    case tea.KeyRunes:
        switch string(msg.Runes) {
        case " ":
            return m.handleSpace()
        case "a":
            return m.handleChar('a')
        }
    }
    if msg.Alt && len(msg.Runes) == 1 {
        // Handle Alt+key
    }

// After (v2)
case tea.KeyPressMsg:
    switch msg.String() {
    case "up":
        return m.handleUpKey()
    case "ctrl+c":
        return m.handleCopy()
    case "space":  // Note: space is now "space", not " "
        return m.handleSpace()
    case "a":
        return m.handleChar('a')
    }
    if msg.Mod.Contains(tea.ModAlt) {
        // Handle Alt+key using msg.Mod
    }
```

### 2.4 Field Migration Reference
| v1 Field | v2 Field | Notes |
|----------|----------|-------|
| `msg.Type` | `msg.Code` | Returns a `rune` value |
| `msg.Runes` | `msg.Text` | Now `string`, not `[]rune` |
| `msg.Alt` | `msg.Mod.Contains(tea.ModAlt)` | Use `Mod` field with Contains() |
| N/A | `msg.BaseCode` | US PC-101 layout key (for international keyboards) |
| N/A | `msg.ShiftedCode` | Shifted character code (e.g., 'B' for shift+b) |

### 2.5 Handle Paste Events
```go
// Before (v1)
case tea.KeyMsg:
    if msg.Paste {
        // Handle paste
    }

// After (v2)
case tea.PasteMsg:
    content := string(msg)  // The pasted content
    return m.handlePaste(content)

case tea.PasteStartMsg:
    // Optional: Mark paste operation start

case tea.PasteEndMsg:
    // Optional: Mark paste operation end
```

## Phase 3: Implement Shift+Arrow Selection

### 3.1 Remove Visual Mode Code
- Delete all `visualMode` related code from `model.go`
- Remove `IsVisualMode()`, `StartVisualMode()`, `ExitVisualMode()`
- Remove Ctrl+G key binding
- Clean up status bar visual mode indicators

### 3.2 Add Selection State Management
```go
// In Model struct
type Model struct {
    // ... existing fields ...

    // Selection state (non-modal)
    selecting           bool  // True when Shift is held during navigation
    selectionAnchorLine int   // Line where selection started
    selectionAnchorCol  int   // Column where selection started
}
```

### 3.3 Implement Shift+Arrow Detection
```go
func (m Model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
    switch msg.String() {
    // Shift+Arrow starts/extends selection
    case "shift+up":
        if !m.selecting {
            m.StartSelection()
        }
        m.cursorLine--
        m.ExtendSelection()
        return m, nil

    case "shift+down":
        if !m.selecting {
            m.StartSelection()
        }
        m.cursorLine++
        m.ExtendSelection()
        return m, nil

    case "shift+left":
        if !m.selecting {
            m.StartSelection()
        }
        m.cursorCol--
        m.ExtendSelection()
        return m, nil

    case "shift+right":
        if !m.selecting {
            m.StartSelection()
        }
        m.cursorCol++
        m.ExtendSelection()
        return m, nil

    // Regular arrow keys clear selection
    case "up", "down", "left", "right":
        m.ClearSelection()
        // ... normal navigation ...

    // Shift+Ctrl+Arrow for word selection
    case "shift+ctrl+left":
        if !m.selecting {
            m.StartSelection()
        }
        m.moveWordLeft()
        m.ExtendSelection()
        return m, nil

    case "shift+ctrl+right":
        if !m.selecting {
            m.StartSelection()
        }
        m.moveWordRight()
        m.ExtendSelection()
        return m, nil
    }
}
```

### 3.4 Update Selection Methods
```go
func (m *Model) StartSelection() {
    m.selecting = true
    m.selectionAnchorLine = m.cursorLine
    m.selectionAnchorCol = m.cursorCol
}

func (m *Model) ExtendSelection() {
    // Selection automatically extends from anchor to current cursor
    // No need for explicit action, just maintain selecting = true
}

func (m *Model) ClearSelection() {
    m.selecting = false
    m.selectionAnchorLine = -1
    m.selectionAnchorCol = -1
}

func (m *Model) HasSelection() bool {
    return m.selecting &&
           (m.selectionAnchorLine != m.cursorLine ||
            m.selectionAnchorCol != m.cursorCol)
}
```

## Phase 4: Mouse Selection Support

### 4.1 Update Mouse Message Handling

**CRITICAL**: `tea.MouseMsg` is now an interface. Different event types have specific message types.

```go
// Before (v1)
case tea.MouseMsg:
    switch msg.Type {
    case tea.MouseLeft:
        x, y := msg.X, msg.Y
        m.handleClick(x, y)
    case tea.MouseWheelUp:
        m.scrollUp()
    }

// After (v2)
case tea.MouseClickMsg:
    mouse := msg.Mouse()  // Get coordinates via Mouse() method
    if mouse.Button == tea.MouseLeft {  // Note: tea.MouseLeft, not tea.MouseButtonLeft
        m.handleClick(mouse.X, mouse.Y)
    }

case tea.MouseReleaseMsg:
    mouse := msg.Mouse()
    if mouse.Button == tea.MouseLeft {
        m.handleMouseRelease(mouse.X, mouse.Y)
    }

case tea.MouseWheelMsg:
    mouse := msg.Mouse()
    if mouse.Button == tea.MouseWheelUp {
        m.scrollUp()
    } else if mouse.Button == tea.MouseWheelDown {
        m.scrollDown()
    }

case tea.MouseMotionMsg:
    mouse := msg.Mouse()
    if m.mouseSelecting {
        m.handleMouseDrag(mouse.X, mouse.Y)
    }
```

**Button Constants Changed**:
- `tea.MouseButtonLeft` → `tea.MouseLeft`
- `tea.MouseButtonRight` → `tea.MouseRight`
- `tea.MouseButtonMiddle` → `tea.MouseMiddle`
- `tea.MouseButtonWheelUp` → `tea.MouseWheelUp`
- `tea.MouseButtonWheelDown` → `tea.MouseWheelDown`

### 4.2 Implement Click and Drag Selection
```go
type Model struct {
    // ... existing fields ...
    mouseSelecting bool  // True during mouse drag selection
}

func (m *Model) handleMouseClick(pos tea.MousePos) {
    m.ClearSelection()
    m.setCursorFromMouse(pos.X, pos.Y)
    m.mouseSelecting = true
    m.StartSelection()
}

func (m *Model) handleMouseDrag(pos tea.MousePos) {
    m.setCursorFromMouse(pos.X, pos.Y)
    m.ExtendSelection()
}

func (m *Model) handleMouseRelease(pos tea.MousePos) {
    m.mouseSelecting = false
}
```

## Phase 5: Program Initialization Updates

### 5.1 Update Program Creation

```go
// Before (v1)
p := tea.NewProgram(
    model,
    tea.WithAltScreen(),
    tea.WithMouseCellMotion(),
    tea.WithInputTTY(),
)
err := p.Start()  // or p.StartReturningModel()

// After (v2)
p := tea.NewProgram(model)
// All options moved to View() method
// Only these program options remain:
// - tea.WithColorProfile(profile) for testing
// - tea.WithWindowSize(w, h) for initial size

finalModel, err := p.Run()  // Note: Run() replaces Start()
```

### 5.2 Removed Program Options Reference

| v1 Program Option | v2 View Field |
|-------------------|---------------|
| `tea.WithAltScreen()` | `view.AltScreen = true` |
| `tea.WithMouseCellMotion()` | `view.MouseMode = tea.MouseModeCellMotion` |
| `tea.WithMouseAllMotion()` | `view.MouseMode = tea.MouseModeAllMotion` |
| `tea.WithReportFocus()` | `view.ReportFocus = true` |
| `tea.WithoutBracketedPaste()` | `view.DisableBracketedPasteMode = true` |
| `tea.WithInputTTY()` | Removed (automatic) |
| `tea.WithANSICompressor()` | Removed (automatic) |

### 5.3 Update Commands

```go
// Before (v1) - Commands returned from Update()
return m, tea.Batch(
    tea.EnterAltScreen,
    tea.EnableMouseCellMotion,
    tea.HideCursor,
)

// After (v2) - Set in View() method
func (m Model) View() tea.View {
    v := tea.NewView(content)
    v.AltScreen = true
    v.MouseMode = tea.MouseModeCellMotion
    v.Cursor.Hidden = true
    return v
}
```

### 5.4 API Renames

```go
// Sequentially → Sequence
cmd := tea.Sequence(cmd1, cmd2, cmd3)  // was tea.Sequentially

// WindowSize command change
cmd := tea.RequestWindowSize  // Returns Msg directly, not Cmd
```

## Phase 6: Testing & Validation

### 6.1 Update Existing Tests

Key changes for all test files:
- Change `tea.KeyMsg` to `tea.KeyPressMsg` for key press handling
- Update `View()` calls to handle `tea.View` return type
- Fix mouse event handling to use new message types
- Update key field access (`msg.Type` → `msg.Code`, etc.)

### 6.2 Create New Selection Tests
```go
// shift_selection_test.go
func TestShiftArrowSelection(t *testing.T) {
    tests := []struct {
        name     string
        keys     []string
        expected string
    }{
        {
            name: "shift+right selects text",
            keys: []string{"shift+right", "shift+right", "shift+right"},
            expected: "selected: 'Hel'",
        },
        {
            name: "shift+down selects lines",
            keys: []string{"shift+down"},
            expected: "selected: 'Hello world\n'",
        },
        {
            name: "regular arrow clears selection",
            keys: []string{"shift+right", "shift+right", "left"},
            expected: "selected: ''",
        },
    }
    // ... test implementation ...
}
```

### 6.3 Catwalk Tests
Create catwalk tests in `testdata/`:
- `shift_arrow_selection/` - Basic Shift+Arrow selection
- `shift_word_selection/` - Shift+Ctrl+Arrow word selection
- `mouse_selection/` - Click and drag selection
- `selection_operations/` - Copy/cut/paste with selection

### 6.4 Terminal Compatibility Testing
Test on different terminals:
- **Full support**: Ghostty, Kitty, Alacritty, iTerm2, WezTerm
- **Degraded support**: Terminal.app, older terminals
- Document any limitations

## Phase 7: Cleanup

### 7.1 Remove Dead Code
- Remove all visual mode related code
- Remove Ctrl+G handler
- Clean up unused selection methods

### 7.2 Update Documentation
- Update README with new selection behavior
- Document Shift+Arrow selection in help text
- Update command menu with selection shortcuts

### 7.3 Performance Optimization
- Profile selection rendering performance
- Optimize selection range calculations
- Ensure smooth selection with large documents

## Common Mistakes to Avoid

Based on the official upgrade guide, watch out for these pitfalls:

1. **Forgetting View() Return Type**: The most common error is forgetting to change `View() string` to `View() tea.View`

2. **Using Wrong Message Type**: Using `tea.KeyMsg` (interface) instead of `tea.KeyPressMsg` (concrete type) in switches

3. **Direct Mouse Coordinate Access**: Trying to access `msg.X` directly instead of calling `msg.Mouse()` first

4. **Space Key Matching**: Still using `case " ":` instead of `case "space":` for space detection

5. **Program Options**: Trying to use removed options like `tea.WithAltScreen()` instead of View fields

6. **Commands in Update**: Returning removed commands like `tea.EnterAltScreen` from Update()

7. **Wrong Button Constants**: Using old `tea.MouseButtonLeft` instead of `tea.MouseLeft`

8. **Paste Detection**: Still checking `msg.Paste` flag instead of handling `tea.PasteMsg`

## Risk Mitigation

### Potential Issues & Solutions

1. **Terminal Compatibility**
   - Risk: Some terminals don't support Shift detection
   - Mitigation: Provide fallback mouse selection, document limitations

2. **Breaking Changes in Tests**
   - Risk: All tests will need updates
   - Mitigation: Systematic update using search/replace, run tests frequently

3. **Performance Regression**
   - Risk: New rendering system might be slower
   - Mitigation: Profile before/after, use v2's performance improvements

4. **User Muscle Memory**
   - Risk: Users used to Ctrl+G visual mode
   - Mitigation: Clear documentation, maybe temporary help message

## Success Criteria

✅ All tests pass with Bubble Tea v2
✅ Shift+Arrow selection works as expected
✅ Shift+Ctrl+Arrow selects words
✅ Mouse click-and-drag selection works
✅ Copy/Cut/Paste work with selections
✅ No visual mode artifacts remain
✅ Performance is equal or better than v1
✅ Works on major terminals (with graceful degradation)

## Migration Checklist

Based on the official UPGRADE_GUIDE_V2.md, here's a systematic checklist:

### Import Updates
- [ ] Replace all `github.com/charmbracelet/bubbletea` → `charm.land/bubbletea/v2`
- [ ] Replace all `github.com/charmbracelet/lipgloss` → `charm.land/lipgloss/v2`
- [ ] Replace all `github.com/charmbracelet/bubbles` → `charm.land/bubbles/v2`

### View Method Updates
- [ ] Change all `View() string` to `View() tea.View`
- [ ] Wrap return strings with `tea.NewView()`
- [ ] Move all terminal configuration to View fields

### Key Handling Updates
- [ ] Replace `tea.KeyMsg` with `tea.KeyPressMsg` in type switches
- [ ] Update `msg.Type` to `msg.Code`
- [ ] Update `msg.Runes` to `msg.Text`
- [ ] Update `msg.Alt` to `msg.Mod.Contains(tea.ModAlt)`
- [ ] Change space key detection from `" "` to `"space"`
- [ ] Add handling for `tea.KeyReleaseMsg` if needed

### Mouse Handling Updates
- [ ] Update `tea.MouseMsg` type switches to specific types
- [ ] Add `msg.Mouse()` calls to access coordinates
- [ ] Update button constants (remove "Button" prefix)
- [ ] Handle `tea.MouseClickMsg`, `tea.MouseReleaseMsg`, etc.

### Paste Event Updates
- [ ] Replace `msg.Paste` checks with `tea.PasteMsg` handling
- [ ] Add `tea.PasteStartMsg` and `tea.PasteEndMsg` if needed

### Program Updates
- [ ] Remove all `tea.With*` options from `NewProgram()`
- [ ] Move options to View() method as fields
- [ ] Replace `p.Start()` with `p.Run()`
- [ ] Remove any calls to removed commands

### Command Updates
- [ ] Replace `tea.Sequentially()` with `tea.Sequence()`
- [ ] Update `tea.WindowSize()` to `tea.RequestWindowSize`
- [ ] Remove toggle commands (EnterAltScreen, etc.)

### Testing Updates
- [ ] Update all test key message types
- [ ] Fix View() return type handling in tests
- [ ] Update mouse event assertions
- [ ] Verify catwalk tests still pass

## Timeline Estimate

This is a significant refactor affecting ~50+ files:

1. **Preparation**: Document current state, create branch
2. **Core Migration**: Update imports, View method, key handling
3. **Selection Implementation**: Remove visual mode, add Shift selection
4. **Testing**: Update all tests, create new selection tests
5. **Validation**: Terminal compatibility, performance testing
6. **Cleanup**: Remove dead code, update docs

## Notes

- Keep the Export overlay fix (already implemented)
- Preserve all existing clipboard operations
- Maintain backward compatibility for file operations
- Consider keeping a temporary "legacy mode" flag if needed