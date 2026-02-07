---
created: 2026-02-07T20:34
title: Add visual file picker for Save/Save-As dialogs
area: tui
files:
  - cmd/calcmark/tui/editor/model.go
  - cmd/calcmark/tui/editor/view.go
---

## Problem

Current save-as flow is basic text input in the status bar. Users expect a more friendly experience like:
- Visual file browser to navigate directories
- See existing files to avoid overwriting
- Tab completion for paths
- Create new directories if needed

User typed `~/test.cm` and expected it to work (fixed with tilde expansion), but the overall UX is still minimal.

## Solution

Integrate charmbracelet/bubbles filepicker component:
- https://pkg.go.dev/github.com/charmbracelet/bubbles/filepicker
- Already have bubbles v0.21.0 in go.mod

Implementation approach:
1. Add new InputState: `StateFilePicker`
2. Embed `filepicker.Model` in editor Model
3. On Ctrl+S (new file) or Ctrl+Shift+S (save-as), show filepicker
4. Allow both directory navigation AND typing a new filename
5. Show current directory, allow creating new files in selected directory
6. Style to match editor theme (dark background, etc.)

Consider also:
- File type filtering (show only .cm files, or all files)
- Quick access to recent directories
- Current working directory indicator
