---
created: 2026-02-07T23:50
title: Preview pane shows non-calculation markdown elements
area: tui
files:
  - cmd/calcmark/tui/editor/aligned.go
  - cmd/calcmark/tui/editor/sidebyside_test.go
---

## Problem

User reports that adding:
- `1. a list item` (numbered list)
- `some text` (paragraph)
- `[link](a link)` (links)

...to Source causes them to render in the preview pane (Results pane).

According to the Phase 11.1-03 fix, only headings (# lines) and calculation results should appear in the preview pane. Other markdown should be filtered out.

Reported during UAT for Phase 11.1.

## Solution

TBD - The filtering logic in `ComputeAlignedModel` in aligned.go may have a bug. Needs investigation:
- Check if ordered lists are handled differently
- Verify the filtering logic catches all cases
