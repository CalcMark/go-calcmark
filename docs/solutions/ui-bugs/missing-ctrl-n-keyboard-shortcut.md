---
title: "CTRL-N keyboard shortcut missing for new document creation in TUI editor"
date: 2026-02-27
category: ui-bugs
tags:
  - tui-editor
  - bubble-tea
  - keyboard-shortcut
  - new-document
  - unsaved-changes
  - save-prompt
  - command-menu
  - state-reset
components:
  - cmd/calcmark/tui/editor
severity: medium
symptoms:
  - CTRL-N keyboard shortcut does not exist in the cm TUI editor
  - No way to create a new empty document without quitting and restarting the editor
  - The command menu has no "New" entry for creating fresh documents
root_causes:
  - PendingNew action type was not defined in the PendingAction enum
  - No Ctrl+N key binding existed in the global shortcut handler
  - No "New" command existed in EditorCommands
  - The newFile() method did not exist in file_operations.go
  - The save_prompt_handler had no PendingNew arm for completing or cancelling the action
---

## Problem Statement

The TUI editor's CTRL-N shortcut was entirely absent. Users expected the standard "New Document" shortcut to reset the editor with an empty document, optionally prompting to save unsaved changes first (following the same pattern as Ctrl+Q for quit and Ctrl+O for open). Without this shortcut, the only way to start a fresh document was to quit and relaunch the editor.

## Investigation Steps

1. **Key binding audit**: Searched the global Ctrl+key handler in `key_dispatch.go` and confirmed no `case 'n'` branch existed. Ctrl+Q, Ctrl+S, Ctrl+O, Ctrl+E, and Ctrl+F were all mapped, but Ctrl+N was absent.

2. **Command menu audit**: Reviewed `EditorCommands` in `command_menu.go` and confirmed no "New" entry existed in the command list.

3. **Pattern analysis**: Studied the existing Ctrl+O and Ctrl+Q implementations to understand the established pattern: key dispatch calls `executeCommandByName()`, which calls `promptSaveIfNeeded()` if there are unsaved changes, otherwise executes the action directly.

4. **State machine audit**: Verified that `PendingAction` enum had `PendingQuit` and `PendingOpen` but no `PendingNew`, and the save prompt handler had no arm for a new-document action.

5. **Reset mechanism discovery**: Found `resetForNewDocument()` already existed as a comprehensive state reset helper (cursor, editBuf, undo stack, selection, search state, etc.), used by `openFile()`.

## Root Cause Analysis

The feature was simply never implemented. All five layers of the key dispatch pipeline lacked the necessary wiring:

1. **model.go**: `PendingAction` enum missing `PendingNew`
2. **command_menu.go**: `EditorCommands` missing "New" entry, `executeCommandByName` missing handler
3. **key_dispatch.go**: Global Ctrl+key handler missing `case 'n'`
4. **file_operations.go**: `newFile()` method did not exist
5. **save_prompt_handler.go**: `completePendingSaveAction` and `actionCancelledMsg` missing `PendingNew` arms

## Solution

### 1. Added `PendingNew` to PendingAction enum (`model.go`)

```go
const (
    PendingNone PendingAction = iota
    PendingQuit
    PendingOpen
    PendingNew  // Save prompt was triggered by Ctrl+N
)
```

### 2. Added "New" command to EditorCommands and handler (`command_menu.go`)

```go
var EditorCommands = []Command{
    {Name: "New", Accelerator: "Ctrl+N", Description: "New empty document", Category: "file"},
    {Name: "Save", Accelerator: "Ctrl+S", Description: "Save document", Category: "file"},
    // ...
}

// In executeCommandByName:
case "New":
    if m.promptSaveIfNeeded(PendingNew, "Unsaved changes! Save before new? (y/n/c)") {
        return m, nil
    }
    m.newFile()
    return m, nil
```

### 3. Added Ctrl+N key binding (`key_dispatch.go`)

```go
case 'n':
    return m.executeCommandByName("New")
```

### 4. Implemented `newFile()` (`file_operations.go`)

```go
func (m *Model) newFile() {
    doc, err := document.NewDocument("\n")
    if err != nil {
        m.statusMsg = fmt.Sprintf("New file failed: %v", err)
        m.statusIsErr = true
        return
    }
    eval := implDoc.NewEvaluator()
    _ = eval.Evaluate(doc)
    m.resetForNewDocument(doc, eval, "", "\n")
    m.statusMsg = "New document"
}
```

### 5. Added PendingNew handling (`save_prompt_handler.go`)

```go
// In completePendingSaveAction:
case PendingNew:
    m.exitOverlay()
    m.newFile()
    return m, nil

// In actionCancelledMsg:
case PendingNew:
    return "New cancelled"
```

## Gotcha: Shifted Command Menu Indices

Adding "New" at index 0 of `EditorCommands` shifted all subsequent command indices by 1, breaking three existing tests:

| Test | Breakage | Fix |
|------|----------|-----|
| `TestEditorCatwalkExportFlow` | Export was at old index 3, now at 4 | Added 1 more `key down` press |
| `TestEditorCatwalkHelpInteractive` | Full Help was at old index 15, now at 16 | Added 1 more `key down` press |
| `TestAllCommandMenuActionsViaUpdate` | Save assumed to be at index 0 | Changed to navigate by name |

## Testing

### Unit tests (7 new tests in `file_operations_test.go`)

| Test | Coverage |
|------|----------|
| `TestNewFile` | Direct `newFile()` call resets filepath, cursor, editBuf, state |
| `TestCtrlNCreatesNewDocument` | Ctrl+N without unsaved changes creates new empty document |
| `TestCtrlNWithUnsavedChanges` | Ctrl+N with unsaved changes enters `StateSavePrompt` with `PendingNew` |
| `TestCtrlNSavePromptNo` | Pressing 'n' at prompt discards changes and creates new document |
| `TestCtrlNSavePromptCancel` | Pressing 'c' at prompt returns to editing |
| `TestCtrlNSavePromptYes` | Pressing 'y' saves file then creates new document |

### Catwalk tests (2 new test files)

**`testdata/new_document_ctrl_n`**: Verifies Ctrl+N on a clean document resets to one empty line (totalSource: 4 -> 1).

**`testdata/new_unsaved_prompt`**: Verifies the full prompt flow: type text -> Ctrl+N shows `StateSavePrompt` -> 'c' cancels -> Ctrl+N again -> 'n' discards and creates new empty document.

## Prevention Strategies

### 1. Navigate command menu by name, not index

Tests should search for commands by `Name` field rather than counting down-arrow presses:

```go
// Resilient pattern:
for m.commandMenuState.Selected < len(EditorCommands)-1 {
    if EditorCommands[m.commandMenuState.Selected].Name == "Export" {
        break
    }
    newModel, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
    m = newModel.(Model)
}
```

### 2. Add new commands at end of category when possible

Adding at the beginning of `EditorCommands` shifts all indices. Adding at the end of a category minimizes test impact.

### 3. Checklist for future command additions

- [ ] Add command entry to `EditorCommands` in correct category
- [ ] Add `case "Name":` in `executeCommandByName()`
- [ ] Add `case 'x':` in global Ctrl+key handler if shortcut exists
- [ ] If command triggers save prompt, add `PendingXxx` to enum and handler
- [ ] Write unit tests for all save prompt paths (y/n/c)
- [ ] Write catwalk test for the key sequence
- [ ] Run `task test` to verify no regressions
- [ ] Run `task quality` for lint/vet

## Related Documentation

- [Ctrl+O stale state and unsaved changes detection](ctrl-o-stale-state-and-unsaved-changes-detection.md) - Established the `promptSaveIfNeeded()` pattern reused here
- [TUI mode transitions, formatter indexing, and bracketed paste fixes](tui-mode-transitions-formatter-indexing-and-bracketed-paste-fixes.md) - Centralized mode transitions via `mode_transitions.go`
- [Frontmatter editing keyboard dispatch fixes](frontmatter-editing-keyboard-dispatch-fixes.md) - Keyboard dispatch architecture context
- [Bubble Tea v2 migration, selection, undo, clipboard fixes](bubble-tea-v2-migration-selection-undo-clipboard-fixes.md) - Framework upgrade context
- [cmd/calcmark/tui/editor/TESTING.md](../../../../cmd/calcmark/tui/editor/TESTING.md) - Catwalk testing guide
