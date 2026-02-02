---
phase: 01-foundation
plan: 02
subsystem: ui
tags: [geometry, wrapping, runewidth, two-column, refactoring, pure-functions]

# Dependency graph
requires:
  - phase: 01-foundation/01
    provides: "mattn/go-runewidth as direct dependency"
provides:
  - "Pure geometry package at cmd/calcmark/tui/geometry/ with WrapText, CalculateRowGeometry, StringWidth"
  - "Editor package wired to use geometry.WrapText exclusively (no local copy)"
  - "23 table-driven tests for geometry functions covering CJK, emoji, wrapping, edge cases"
affects:
  - "Phase 2 (two-column rendering builds on geometry.CalculateRowGeometry)"
  - "Phase 3 (editor integration uses geometry.WrapText for cursor/scrolling)"
  - "Any future TUI refactoring (geometry is now framework-independent)"

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Pure geometry functions with zero TUI framework dependencies"
    - "runewidth.StringWidth for unicode width measurement in geometry layer"
    - "runewidth.RuneWidth for single-rune measurement (avoids string conversion)"

key-files:
  created:
    - "cmd/calcmark/tui/geometry/doc.go"
    - "cmd/calcmark/tui/geometry/geometry.go"
    - "cmd/calcmark/tui/geometry/geometry_test.go"
  modified:
    - "cmd/calcmark/tui/editor/linemodel.go"
    - "cmd/calcmark/tui/editor/aligned.go"
    - "cmd/calcmark/tui/editor/view.go"
    - "cmd/calcmark/tui/editor/model.go"
    - "cmd/calcmark/tui/editor/model_v2.go"
    - "cmd/calcmark/tui/editor/linemodel_test.go"
    - "cmd/calcmark/tui/editor/aligned_test.go"
    - "cmd/calcmark/tui/editor/aligned_empty_lines_test.go"
    - "cmd/calcmark/tui/editor/preview_alignment_integration_test.go"
    - "cmd/calcmark/tui/editor/view_alignment_test.go"

key-decisions:
  - "Used runewidth.RuneWidth(rune) instead of runewidth.StringWidth(string(rune)) for per-rune measurement -- avoids string allocation in hot loop"
  - "Kept ComputeAlignedModel in editor package (not moved to geometry) -- it depends on editor-specific types"
  - "Applied modernize fix (built-in max) only to new geometry code, not across entire codebase"

patterns-established:
  - "geometry package: pure functions with only mattn/go-runewidth + stdlib"
  - "Dependency direction: geometry <- editor (never reverse)"
  - "WrapText is single-sourced in geometry, all editor code imports from geometry"

# Metrics
duration: 6min
completed: 2026-02-02
---

# Phase 1 Plan 02: Geometry Package Extraction Summary

**Pure geometry package extracted with WrapText and CalculateRowGeometry using runewidth (zero TUI deps), editor wired to import from geometry exclusively**

## Performance

- **Duration:** 6 min
- **Started:** 2026-02-02T22:32:27Z
- **Completed:** 2026-02-02T22:38:27Z
- **Tasks:** 2
- **Files modified:** 14 (3 created, 11 modified)

## Accomplishments
- Created `cmd/calcmark/tui/geometry/` package with `WrapText`, `CalculateRowGeometry`, and `StringWidth` -- all pure functions with zero TUI framework dependencies
- Ported WrapText from editor/linemodel.go replacing `lipgloss.Width` with `runewidth.StringWidth` and `runewidth.RuneWidth` for identical behavior on plain text
- Implemented `CalculateRowGeometry` for two-column row layout computation with asymmetric wrapping support
- Updated all 24 WrapText call sites across 10 editor files (5 source, 5 test) to use `geometry.WrapText`
- Removed lipgloss import from linemodel.go entirely
- 23 comprehensive table-driven tests covering: wrapping, CJK double-width, emoji, empty strings, word boundaries, hard breaks, asymmetric column wrapping, zero-width edge cases

## Task Commits

Each task was committed atomically:

1. **Task 1: Create geometry package with WrapText and CalculateRowGeometry** - `3dff658` (feat)
2. **Task 2: Wire editor package to use geometry.WrapText and remove duplicate** - `061285e` (refactor)

## Files Created/Modified
- `cmd/calcmark/tui/geometry/doc.go` - Package documentation (zero TUI deps guarantee)
- `cmd/calcmark/tui/geometry/geometry.go` - WrapText, CalculateRowGeometry, StringWidth functions
- `cmd/calcmark/tui/geometry/geometry_test.go` - 23 table-driven tests
- `cmd/calcmark/tui/editor/linemodel.go` - Removed WrapText function and lipgloss import, uses geometry.WrapText
- `cmd/calcmark/tui/editor/aligned.go` - Uses geometry.WrapText, added geometry import
- `cmd/calcmark/tui/editor/view.go` - Uses geometry.WrapText, added geometry import
- `cmd/calcmark/tui/editor/model.go` - Uses geometry.WrapText, added geometry import
- `cmd/calcmark/tui/editor/model_v2.go` - Uses geometry.WrapText, added geometry import
- `cmd/calcmark/tui/editor/linemodel_test.go` - Uses geometry.WrapText, added geometry import
- `cmd/calcmark/tui/editor/aligned_test.go` - Uses geometry.WrapText, added geometry import
- `cmd/calcmark/tui/editor/aligned_empty_lines_test.go` - Uses geometry.WrapText, added geometry import
- `cmd/calcmark/tui/editor/preview_alignment_integration_test.go` - Uses geometry.WrapText, added geometry import
- `cmd/calcmark/tui/editor/view_alignment_test.go` - Uses geometry.WrapText, added geometry import

## Decisions Made
- Used `runewidth.RuneWidth(rune)` instead of `runewidth.StringWidth(string(rune))` for per-character width measurement in the wrapping loop -- avoids a string allocation per rune, more efficient
- Kept `ComputeAlignedModel` in the editor package rather than moving to geometry -- it depends on `LineResult`, `PreviewMode`, and render callbacks that are editor-specific types
- Applied `max()` modernize fix only in new geometry.go code; pre-existing modernize warnings across the codebase remain outside this plan's scope (documented in 01-01-SUMMARY.md)

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None - the extraction was purely mechanical. The switch from `lipgloss.Width` to `runewidth.StringWidth`/`runewidth.RuneWidth` produced identical behavior for all plain text inputs, confirming that lipgloss.Width is a thin wrapper around runewidth for non-ANSI strings.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- Geometry package ready for Phase 2 to build upon (CalculateRowGeometry is the foundation for correct two-column rendering)
- All 22 packages pass `task test` with zero failures
- Dependency direction is clean: geometry <- editor (no circular imports possible)
- Pre-existing `task quality` modernize warnings remain in files outside this plan's scope

---
*Phase: 01-foundation*
*Completed: 2026-02-02*
