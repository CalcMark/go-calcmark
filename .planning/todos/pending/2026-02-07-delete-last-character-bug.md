---
created: 2026-02-07T09:07
title: Fix delete last character on line bug
area: tui
files:
  - cmd/calcmark/tui/editor/update.go
  - cmd/calcmark/tui/editor/model.go
---

## Problem

There's a persistent bug around deleting the last character on a line or the last character under the cursor in the TUI editor.

Symptoms may include:
- Deleting the last character on a line behaves unexpectedly
- Cursor positioning after delete at end of line is wrong
- Backspace or Delete key at line boundaries causes issues

This is likely in the key handling logic for Backspace/Delete in the editor's Update() function.

## Solution

TBD - Needs investigation:
1. Reproduce the exact scenario (last char on line, cursor at end, etc.)
2. Check handleBackspace() and handleDelete() in update.go
3. Verify cursor bounds checking after character deletion
4. Add catwalk test to prevent regression
