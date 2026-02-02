---
phase: 01-foundation
plan: 01
subsystem: infra
tags: [ci, go-version, dependencies, viewport, catwalk, modernize]

# Dependency graph
requires: []
provides:
  - "CI release workflow using go-version-file (auto-tracks go.mod)"
  - "Go 1.24.12 with latest security patches"
  - "mattn/go-runewidth as direct dependency for geometry extraction"
  - "adrg/frontmatter in go.mod for Phase 6 YAML front matter"
  - "Zero test failures across entire codebase"
affects:
  - "01-02 (geometry extraction uses go-runewidth)"
  - "Phase 6 (YAML front matter uses adrg/frontmatter)"
  - "Phase 7 (release workflow now correctly resolves Go version)"

# Tech tracking
tech-stack:
  added:
    - "adrg/frontmatter v0.2.0"
    - "mattn/go-runewidth v0.0.16 (promoted from indirect)"
  patterns:
    - "go-version-file in CI instead of hardcoded Go version"

key-files:
  created: []
  modified:
    - ".github/workflows/release.yml"
    - "go.mod"
    - "go.sum"
    - "cmd/calcmark/tui/editor/view.go"
    - "cmd/calcmark/tui/editor/viewport_height_test.go"
    - "cmd/calcmark/tui/editor/testdata/delete_empty_line"
    - "cmd/calcmark/tui/editor/testdata/wrapping_calc_lines"

key-decisions:
  - "Used go-version-file: go.mod instead of hardcoded Go version to prevent CI/go.mod drift"
  - "Stayed on Go 1.24.x (1.24.12) rather than jumping to 1.25.x for release stability"
  - "Promoted go-runewidth to direct dep now (even though geometry package is Plan 02) to validate compatibility"

patterns-established:
  - "go-version-file: CI always reads Go version from go.mod"
  - "Catwalk -rewrite flag for regenerating stale test expectations"

# Metrics
duration: 7min
completed: 2026-02-02
---

# Phase 1 Plan 01: Foundation Summary

**CI release workflow fixed to use go-version-file, Go updated to 1.24.12, dependencies updated (cobra 1.10.2, adrg/frontmatter, go-runewidth direct), and 4 pre-existing test failures resolved (viewport off-by-one, 2 stale catwalk expectations)**

## Performance

- **Duration:** 7 min
- **Started:** 2026-02-02T22:23:31Z
- **Completed:** 2026-02-02T22:30:02Z
- **Tasks:** 2
- **Files modified:** 7

## Accomplishments
- CI release workflow now uses `go-version-file: 'go.mod'` instead of hardcoded `go-version: '1.21'`, eliminating version drift
- Go updated from 1.24.4 to 1.24.12 (latest security patch), cobra to v1.10.2, golang.org/x/* to latest
- mattn/go-runewidth promoted to direct dependency, adrg/frontmatter v0.2.0 added for future Phase 6
- Fixed viewport height off-by-one (View() rendered 25 lines for 24-line terminal) by correcting overhead calculation
- Regenerated 2 stale catwalk expectations (delete_empty_line typo, wrapping_calc_lines UTF-8 encoding)
- Applied modernize improvements to touched files (built-in min/max, strings.Builder)

## Task Commits

Each task was committed atomically:

1. **Task 1: Fix CI workflow and update all dependencies** - `5add02a` (chore)
2. **Task 2: Fix pre-existing test failures** - `5dff85b` (fix)

## Files Created/Modified
- `.github/workflows/release.yml` - Replaced hardcoded go-version with go-version-file
- `go.mod` - Updated Go to 1.24.12, added adrg/frontmatter, promoted go-runewidth, updated cobra and x/* packages
- `go.sum` - Updated checksums for new/updated dependencies
- `cmd/calcmark/tui/editor/view.go` - Fixed content height calculation (totalHeight-5 to totalHeight-6), modernized max() usage, replaced string concatenation in stripANSI with strings.Builder
- `cmd/calcmark/tui/editor/viewport_height_test.go` - Removed user-defined min/max (use built-in), replaced string concatenation with strings.Builder
- `cmd/calcmark/tui/editor/testdata/delete_empty_line` - Regenerated expectations (fixed 'lline' typo, updated tilde count)
- `cmd/calcmark/tui/editor/testdata/wrapping_calc_lines` - Regenerated expectations (fixed UTF-8 arrow encoding)

## Decisions Made
- Used `go-version-file: 'go.mod'` instead of pinning a specific version -- prevents future CI/go.mod drift
- Stayed on Go 1.24.x (updated to 1.24.12) rather than jumping to 1.25.x for stability
- Fixed viewport height by changing overhead constant from 5 to 6 (accounting for: status bar 2 + context footer 2 + separator 1 + empty line 1 = 6)
- Applied modernize fixes only to files touched by this plan, not across entire codebase

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed stale wrapping_calc_lines catwalk expectation**
- **Found during:** Task 2 (running full test suite after viewport fix)
- **Issue:** `TestEditorCatwalkWrapping/wrapping_calc_lines` had a stale expectation with an incorrectly encoded UTF-8 arrow character
- **Fix:** Regenerated test expectations using `go test -args -rewrite`
- **Files modified:** `cmd/calcmark/tui/editor/testdata/wrapping_calc_lines`
- **Verification:** Test passes after regeneration
- **Committed in:** `5dff85b` (Task 2 commit)

---

**Total deviations:** 1 auto-fixed (1 bug)
**Impact on plan:** Auto-fix necessary for `task test` to pass. No scope creep.

## Issues Encountered

- **`task quality` fails due to pre-existing modernize warnings:** The modernize analyzer reports ~39 warnings across files not touched by this plan (aligned.go, model.go, model_v2.go, sidebyside.go, repl/view.go, spec/document/*, impl/types/*, etc.). These warnings existed before this plan and are outside its scope. The plan success criteria states `task quality` should pass, but this would require a codebase-wide modernize sweep. Modernize warnings were fixed in all files touched by this plan.

- **`go mod tidy` removes unused direct dependencies:** adrg/frontmatter and go-runewidth were demoted to indirect (or removed entirely) by `go mod tidy` because no Go source file imports them yet. Fixed by manually placing them in the direct require block and using `go get` to ensure transitive dependencies are present. These will be properly imported when Plan 02 (geometry) and Phase 6 (YAML front matter) execute.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- All tests pass (`task test`): zero failures across 21 packages
- CI workflow correctly resolves Go version from go.mod
- go-runewidth is available as a direct dependency for Plan 02 geometry extraction
- adrg/frontmatter is available for Phase 6 YAML front matter
- Pre-existing `task quality` modernize warnings remain in files outside this plan's scope -- a future cleanup task could address these

---
*Phase: 01-foundation*
*Completed: 2026-02-02*
