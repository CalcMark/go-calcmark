---
created: 2026-02-07T21:45
title: Full help as rendered markdown manpage with go:embed
area: tui
files:
  - cmd/calcmark/tui/editor/view.go
  - cmd/calcmark/tui/editor/help.md
---

## Problem

The F1 full help overlay is currently a hardcoded string in the view code. This makes it:
- Hard to maintain and update
- Not easily editable by non-developers
- Missing proper formatting/structure

User wants it to be like a man page - a single markdown document that covers all CalcMark language and app features comprehensively.

## Solution

1. Create `cmd/calcmark/tui/editor/help.md` with full documentation:
   - CalcMark language reference
   - All editor keybindings
   - Function reference
   - Unit reference
   - Examples

2. Use `go:embed` to include the markdown at compile time:
   ```go
   //go:embed help.md
   var helpContent string
   ```

3. Use BubbleTea's markdown rendering component (glamour or similar) to render it:
   - https://github.com/charmbracelet/glamour

4. Add scrolling/navigation for long content:
   - Page up/down
   - Home/End to jump to start/end
   - Maybe search (Ctrl+F)?

Major concern: Easily navigating a long markdown document within the TUI.

Approach options:
- Simple scroll with page keys
- Table of contents with jump-to-section
- Search functionality
