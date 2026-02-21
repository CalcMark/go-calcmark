---
created: 2026-02-07T23:50
title: Preview pane errors not shown on initial file load
area: tui
files:
  - cmd/calcmark/tui/editor/model.go
  - cmd/calcmark/tui/editor/state.go
  - cmd/calcmark/cmd/tui.go
---

## Problem

When loading a file with an error (e.g., `testdata/examples/engineering.cm` with undefined `growth_rate`), the error only appears in the preview pane AFTER the cursor moves for the first time.

The error should be visible immediately on file load.

Reported during UAT for Phase 11.1.

Steps to reproduce:
1. Run `./cm testdata/examples/engineering.cm`
2. Notice preview pane doesn't show error
3. Press any arrow key
4. Error now shows in preview pane

## Solution

TBD - Need to investigate:
- Why the initial render doesn't show the error
- Possible causes: cache key mismatch, evaluation timing, view not computing aligned panes on first render
