---
created: 2026-02-07T10:15
title: Preview pane inserts extra blank line when cursor on last calc before empty line
area: tui
files:
  - cmd/calcmark/tui/editor/view.go
  - cmd/calcmark/tui/editor/results.go
---

## Problem

When the cursor is on the final calculation line before an empty line followed by markdown (non-calculation content), the Preview Pane incorrectly inserts an extra blank line.

### Reproduction steps:
1. Have a document with:
   - A calculation (e.g., `savings = total_income * savings_rate`)
   - Followed by an empty line
   - Followed by markdown (e.g., `## Discretionary`)
2. Position cursor on the line BEFORE the calculation (e.g., `savings_rate = 0.20`) - preview is correct
3. Arrow down to the calculation line (`savings = ...`) - extra blank line appears in preview

### Expected behavior:
Preview pane maintains consistent vertical alignment regardless of cursor position.

### Actual behavior:
An extra blank line appears in the preview pane between the calculation result and the next section header when cursor is on the last calculation before an empty line.

## Solution

TBD - Needs investigation:
1. Check how cursor line affects preview line generation
2. Look at `renderResultsPane()` in view.go
3. Check `ComputeResults()` in results.go for cursor-dependent behavior
4. Add catwalk test to reproduce and validate fix
