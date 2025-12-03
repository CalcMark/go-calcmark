# Testing the CalcMark TUI Editor

## Architecture Philosophy

**IMPORTANT: This editor does NOT use modal editing (vim-style modes).**

The user is ALWAYS editing the document directly. There is no "normal mode" vs "insert mode". Every keypress can either:
1. Navigate the cursor (arrow keys, j/k for vi-style, Page Up/Down, Home/End)
2. Insert/delete text at the cursor
3. Execute commands (Ctrl+Q to quit, Ctrl+S to save, etc.)

The `InputState` type determines the UI context:
- Which component receives input (document vs dialog)
- What auxiliary UI should display (preview updates, errors, help)
- What auxiliary UI should hide (irrelevant errors/help)

`StateDefault` means normal editing with live preview and error display.

## Testing Framework

We use **catwalk** for data-driven TUI testing. Catwalk allows us to:
- Simulate keyboard input sequences
- Assert on model state (Debug() output)
- Verify visual output (View() rendering)
- Test real interaction flows without mocking

## Why Not Modal Testing?

Previous attempts to test the editor treated navigation keys ('j', 'k', etc.) as requiring a specific "mode". This is WRONG for this editor:

- **Navigation keys work ANY time the cursor is not in a text field**
- **Typing keys work ANY time the user is editing a line**
- The transition is **implicit and natural**, not explicit mode changes

For testing, this means:
- Tests should focus on **cursor position** and **editBuf state**, not "modes"
- Navigation commands work when `editBuf == ""` (not actively editing a line)
- Text input works when a line is loaded into `editBuf` (via arrow keys, clicking, etc.)

## Writing Catwalk Tests

### Test Structure

Tests live in `testdata/` as data-driven files. Example:

```
# Test description
run observe=debug
key j
key j
----
-- debug:
mode=0 cursorLine=2 cursorCol=0 ... editBuf="" ...

run observe=results
----
-- results:
Line 0 (a = 3): value=3, error=""
Line 2 (b = 6): value=6, error=""
```

### Available Observers

- **`debug`**: Model.Debug() - cursor position, state flags, buffer contents
- **`lines`**: Model.DebugLines() - visual line structure, wrapping, alignment
- **`results`**: Model.GetLineResults() - evaluation results, errors per line
- **`view`** (default): Model.View() - full rendered output

### Key Commands

- `key <name>`: Press a named key (j, k, up, down, enter, esc, backspace, etc.)
- `type <text>`: Type text characters
- `run observe=<observer>`: Execute commands and assert on output

### Navigation vs Text Input

**Navigation scenario** (editBuf empty):
```
run observe=debug
key j
----
-- debug:
cursorLine=1 editBuf=""
```

**Text input scenario** (editing a line):
```
# Load line into editBuf first
run observe=debug
key i
type hello
----
-- debug:
cursorLine=0 editBuf="hello"
```

## Common Pitfalls

1. **Don't assume modal behavior**: 'j' is ALWAYS "down" unless a line is actively being edited
2. **Check editBuf state**: If `editBuf != ""`, the user is editing that line
3. **UserIsTyping is for debounce**: It's not a mode indicator, it's for debouncing evaluation
4. **Empty lines matter**: The editor preserves empty lines for block separation

## Running Tests

```bash
# Run all catwalk tests
go test ./cmd/calcmark/tui/editor -run Catwalk -v

# Regenerate expected output (after fixing bugs or changing behavior)
go test ./cmd/calcmark/tui/editor -run Catwalk -v -args -rewrite

# Run specific test file
go test ./cmd/calcmark/tui/editor -run "TestEditorCatwalk/edit_variable" -v
```

## Test Coverage Goals

Every user-facing bug should have a catwalk test that:
1. Reproduces the exact key sequence that triggered the bug
2. Asserts on the buggy behavior (to verify the bug exists)
3. After fixing, the test asserts on the correct behavior
4. Prevents regression

## References

- [catwalk on GitHub](https://github.com/knz/catwalk)
- [datadriven testing library](https://github.com/cockroachdb/datadriven)
- [Bubble Tea documentation](https://github.com/charmbracelet/bubbletea)
