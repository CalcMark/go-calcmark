---
created: 2026-02-07T10:20
title: Preview pane shows markdown content (quotes, links) when it should only show results
area: tui
files:
  - cmd/calcmark/tui/editor/results.go
  - cmd/calcmark/tui/editor/view.go
---

## Problem

The Preview Pane is displaying markdown content that should not appear:
- Blockquotes (`> what` renders as `| what`)
- Links (`[link](http://example.com)` renders with underlined URL)

This violates PREVIEW-01: "Preview pane shows ONLY calculation results (not markdown text)"

### Expected behavior:
- Only calculation results displayed (variable assignments, anonymous calculations)
- Headings allowed for context (user preference)
- All other markdown (quotes, links, lists, bold, italic, etc.) should show as blank

### Actual behavior:
- Blockquotes rendered with `|` prefix
- Links rendered with URL underlined

## Solution

TBD - Needs investigation:
1. Check `ComputeResults()` in results.go - what determines if a line is shown
2. Check `renderResultsPane()` in view.go - what gets rendered
3. Ensure only `IsCalc=true` lines OR headings are rendered
4. All other line types should render as blank (preserving vertical spacing)
5. Add catwalk test with mixed markdown content
