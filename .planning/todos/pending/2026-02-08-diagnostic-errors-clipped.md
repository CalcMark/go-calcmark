---
created: 2026-02-08T18:00
title: Diagnostic errors clipped in preview pane
area: tui
files:
  - cmd/calcmark/tui/editor/view.go
  - cmd/calcmark/tui/components/statusbar.go
---

## Problem

Most diagnostic errors displayed in the preview pane are clipped after a few characters, which is unhelpful. The full message is displayed in the footer status area, but this is not immediately visible while editing.

User proposes: Error messages should be mostly shifted to the status area that can expand/shrink to provide details of an error message at the current Source line.

## Solution

Consider:
1. Preview pane shows abbreviated error indicator (e.g., just "Error" or "⚠")
2. Status bar expands to show full error for the current line
3. Status bar could auto-expand when cursor is on an error line
4. Or use a tooltip/popup pattern for error details

This would improve the error visibility without cluttering the preview pane.
