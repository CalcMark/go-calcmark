---
title: "Split view.go (1,663 LOC) into 6 cohesive rendering modules"
date: 2026-02-22
category: code-organization
tags: [tui, editor, refactoring, bubble-tea, view-rendering, architecture-review]
severity: P0
component: cmd/calcmark/tui/editor
symptoms:
  - view.go exceeded 1,663 LOC
  - Single file handling multiple rendering concerns (panes, lines, overlays, footer, utilities)
  - Difficult to navigate, review, and maintain individual rendering features
root_cause: All View-phase rendering logic accumulated in one file with no structural forcing function to distribute it
resolution: Split into 6 files by logical cohesion — view.go (329), view_panes.go (361), view_lines.go (255), view_overlays.go (370), view_footer.go (184), view_util.go (208)
related:
  - docs/solutions/ui-bugs/tui-editor-rendering-divider-status-bar-error-line.md
  - cmd/calcmark/tui/editor/TESTING.md
  - .planning/codebase/CONCERNS.md
---

# Split view.go into 6 Cohesive Rendering Modules

## Problem

`view.go` had grown to 1,663 lines of code containing the entirety of the TUI editor's rendering logic. Every rendering concern lived in a single file: the top-level `View()` orchestrator, source and preview pane layout, individual line rendering (cursor, selection, wrapped lines, edit buffer), overlay and popup rendering (globals panel, autocomplete, command menu, file picker), footer and context rendering, and low-level ANSI/string utilities.

The file was flagged P0 in an architecture review (project rule: files over 1000 LOC are a smell, over 2000 are P1, over 3000 are P0).

## Root Cause

The growth was natural and incremental. Go packages share a namespace, so any method on `Model` with a `render` prefix gravitates to `view.go` because it is "view code." Each new UI feature — autocomplete popup, file picker overlay, globals panel, context footer, selection highlighting, wrapped line editing — added its rendering logic to the only existing rendering file. There was no structural forcing function to distribute rendering into separate files early.

## Solution

Split into 6 files using **logical cohesion by rendering concern** as the guiding principle. Each file is responsible for one distinct rendering concern. The `view.go` file was reduced to its irreducible core: the `View()` method, the `alignedPanes` struct, and its two supporting computation functions.

The principle applied consistently: if a function answers "how do I render X?", it belongs in the file named after X. If it answers "what string/ANSI manipulation is needed?", it belongs in `view_util.go`.

### Split Boundaries

**`view.go` (329 LOC) — Orchestrator and alignment core**
- `View()` — the `tea.Model` interface method; top-level render dispatcher
- `alignedPanes` struct — shared data contract between pane renderers
- `computeAlignedPanes()` — converts cached `AlignedModel` to `alignedPanes` format
- `computeAlignedModelFresh()` — computes fresh `AlignedModel` from model state
- `sourceLine` and `previewLine` struct definitions

**`view_panes.go` (361 LOC) — Source and preview pane rendering**
- `renderSourcePaneAligned()` — scrollable source pane with gutter, tilde fill, scroll offset conversion
- `renderPreviewPaneAligned()` — scrollable preview pane with globals panel integration
- `renderCalcLine()` — single calculation result line (error/blocked/full/minimal modes)
- `sourcePaneBg()`, `previewPaneBg()` — background color helpers

**`view_lines.go` (255 LOC) — Individual line rendering**
- `renderLineWithSelection()` — selection highlighting (rune-safe before/selected/after segments)
- `renderLineWithCursor()` — block cursor at column, edit-style and non-edit backgrounds
- `renderEditLine()` — thin wrapper for edit buffer rendering
- `renderEditLineWrapped()` — multi-line wrapped edit buffer with cursor tracking

**`view_overlays.go` (370 LOC) — Modal overlays and popups**
- `renderGlobalsPanel()` — collapsible globals panel (collapsed/expanded/no-globals states)
- `renderAutocompletePopup()` — autocomplete popup via `components.RenderPopupBox`
- `calculatePopupScreenPosition()` — screen coordinates for popup placement
- `renderCommandMenuPopup()` — command menu with selection highlighting
- `renderFilePickerOverlay()` — file picker with purpose-aware title/hints

Note: `renderHelpOverlay()` and `renderExportOverlay()` live in their own pre-existing files (`help_overlay.go`, `export_overlay.go`) — they were already separated before this refactoring.

**`view_footer.go` (184 LOC) — Footer and context rendering**
- `renderContextFooter()` — builds context footer state from cursor line results, diagnostics, references
- `getLineReferences()` — queries evaluator environment for variable references
- `extractErrorHint()` — extracts brief error hint for inline display
- `formatFunctionParamHint()` — formats function parameter hints

**`view_util.go` (208 LOC) — String and ANSI utilities**
- `padToWidth()` — pads string to N visual columns using `components.StyledPadding`
- `ensureFullWidth()`, `ensureLinesAreFullWidth()` — full-width padding for assembled lines
- `overlayPadLine()` — strips ANSI resets then pads/backgrounds for overlay use
- `wrapStyledLine()` — wraps ANSI-styled line by extracting plain text
- `stripANSI()` — removes ANSI escape sequences for width calculation
- `overlayPopupOnLines()` — pure compositor for popup overlay
- `overlayStringAt()` — overlays string at visual column, skipping ANSI in width accounting

## Verification

1. **All tests pass.** Full test suite via `go test ./...` — no modifications needed since all files share `package editor`.
2. **`go vet` clean.** No static analysis errors. No import cycles possible within same package.
3. **`go build` clean.** All cross-file function calls resolve correctly within the single compilation unit.

## Prevention

**Split early.** Establish a 400 LOC review checkpoint for view files. When a file hits this threshold, reviewers should ask "is this still cohesive?" rather than waiting for 1,600+ LOC.

**One rectangular region = one file.** In a split-pane TUI, think spatially: left pane rendering, right pane rendering, modal overlays, footer — each gets its own file.

**Shared data structures define ownership.** The struct lives in the file that computes it. `alignedPanes` lives in `view.go` because `computeAlignedPanes()` lives there. Consumer files (`view_panes.go`) use but don't define it.

**Utility gravity.** Utilities orbit the component that spawned them. If a utility is only used by one file, it belongs in that file, not in `view_util.go`. Reserve `view_util.go` for functions used by 2+ other view files.

**PR review checklist:**
- No view file exceeds 450 LOC (`wc -l cmd/calcmark/tui/editor/view*.go`)
- New rendering code is in the correct file by concern
- New utility functions are used by 2+ files (else inline them)
- No View-phase mutations (renders must not modify model state)
