---
phase: 05-help-system
plan: 02
subsystem: tui/editor
tags: [help, keyboard, statusbar, bubbles]
dependency-graph:
  requires: [05-01]
  provides: [help-overlay, status-bar-enhancements, f1-toggle]
  affects: [06-autocomplete]
tech-stack:
  added: []
  patterns: [bubbles-help-package, input-state-overlay]
key-files:
  created:
    - cmd/calcmark/tui/editor/help_overlay.go
    - cmd/calcmark/tui/editor/testdata/help_toggle
  modified:
    - cmd/calcmark/tui/shared/keys.go
    - cmd/calcmark/tui/editor/model.go
    - cmd/calcmark/tui/editor/view.go
    - cmd/calcmark/tui/components/statusbar.go
    - cmd/calcmark/tui/editor/catwalk_test.go
    - cmd/calcmark/tui/components/components_test.go
decisions:
  - id: 05-02-f1-key
    desc: Use F1 for help instead of ? to avoid conflict with calc expressions
  - id: 05-02-line-col-format
    desc: Status bar shows L{line}:{col} format (e.g., L5:12)
  - id: 05-02-eval-indicator
    desc: Show EVAL... during typing debounce, calc count when idle
metrics:
  duration: ~15min
  completed: 2026-02-03
---

# Phase 05 Plan 02: TUI Help Overlay and Status Bar Summary

**One-liner:** F1 toggles centered help overlay listing keybindings; status bar shows line:col and EVAL... during typing.

## What Was Built

### Help Overlay (F1 Toggle)
- Created `help_overlay.go` with `renderHelpOverlay()` using bubbles/help package
- Changed Help key binding from `?` to `f1` (avoids calc expression conflict)
- Added `keys` field to Model struct initialized with `DefaultKeyMap()`
- Added help toggle handling in `handleKey()` using `key.Matches(msg, m.keys.Help)`
- Renders centered overlay when `mode == StateHelp` in View()
- Overlay shows keybindings organized by category with title and footer

### Enhanced KeyMap
- Added editor-specific bindings: WordLeft, WordRight, LineStart, LineEnd, NewLine, Backspace, DeleteWord
- Organized FullHelp() into categories: Navigation, Word Navigation, Editing, File, Other
- Cleaner help text with capitalized labels (F1, Ctrl+S, etc.)

### Status Bar Enhancements
- Added `Column` and `EvalInProgress` fields to `StatusBarState`
- Status bar shows `L{line}:{col}` format (e.g., L5:12) instead of L5/100
- Shows "EVAL..." during debounce period (when `userIsTyping == true`)
- Shows calc count when not evaluating

### InputState String Method
- Added `String()` method to `InputState` for readable debug output
- Debug() output now shows `mode=StateHelp` instead of `mode=2`

## Commits

| Hash | Type | Description |
|------|------|-------------|
| f9fd4e0 | feat | help overlay, keys.go updates (in 05-01 commit) |
| c8904c6 | feat | status bar with column position and eval indicator |
| 3facf2e | test | catwalk test for help toggle behavior |

## Decisions Made

1. **F1 for Help (not ?)**: The `?` key conflicts with typing in calc expressions (e.g., "price?"). F1 is universally recognized as help.

2. **Line:Col Format**: Changed from L5/100 to L5:12 format to show cursor position within line, not just line number vs total.

3. **EVAL... Indicator**: Uses existing `userIsTyping` field which is already set during debounce period - no new state needed.

## Deviations from Plan

None - plan executed exactly as written.

## Verification

- [x] F1 toggles help overlay on/off
- [x] Help overlay shows keybindings organized by category
- [x] Escape also dismisses help overlay
- [x] Status bar shows cursor position as L{line}:{col}
- [x] Status bar shows EVAL... during debounce period
- [x] Normal editing continues after dismissing help
- [x] All tests pass (`task test`)
- [x] Catwalk test validates help toggle behavior

## Next Phase Readiness

**Phase 05 Complete:** Both 05-01 (CLI help commands) and 05-02 (TUI help overlay) are done.

**Ready for Phase 06 (Autocomplete):**
- KeyMap infrastructure is in place for new bindings
- Help overlay pattern established for other UI overlays
- Status bar enhanced - ready for autocomplete indicators

## Technical Notes

### Help Overlay Architecture
```
F1 pressed
    -> key.Matches(msg, m.keys.Help) == true
    -> toggle m.mode between StateDefault and StateHelp
    -> View() checks if m.mode == StateHelp
    -> renderHelpOverlay() uses bubbles/help to render keybindings
    -> lipgloss.Place() centers overlay on screen
```

### Status Bar Format
```
Before: "L5/100 | 10 calcs"
After:  "L5:12 | 10 calcs"  or  "L5:12 | EVAL..."
```
