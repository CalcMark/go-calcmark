# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-02-02)

**Core value:** Fast, offline, verifiable calculations in markdown documents with a simple editor
**Current focus:** Phase 4 - TUI Test Coverage

## Current Position

Phase: 4 of 8 (TUI Test Coverage)
Plan: 2 of 3 in current phase
Status: In progress
Last activity: 2026-02-03 - Completed 04-02-PLAN.md (Add missing catwalk tests)

Progress: [███████████░░░░░░░░░] 55% (11/20 plans)

## Performance Metrics

**Velocity:**
- Total plans completed: 11
- Average duration: 8min
- Total execution time: 1.4 hours

**By Phase:**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| 1. Foundation | 2/2 | 13min | 6.5min |
| 2. TUI Geometry & Layout | 2/2 | 8min | 4min |
| 3. TUI Editor Integration | 5/5 | 55min | 11min |
| 4. TUI Test Coverage | 2/3 | 9min | 4.5min |

**Recent Trend:**
- Last 5 plans: 03-03 (8min), 03-04 (7min), 03-05 (20min), 04-01 (3min), 04-02 (6min)
- Trend: Test coverage plans are fast - patterns established, just applying them

*Updated after each plan completion*

## Accumulated Context

### Decisions

Decisions are logged in PROJECT.md Key Decisions table.
Recent decisions affecting current work:

- Roadmap: TUI geometry is Phase 2 (after foundation), editor integration is Phase 3 -- geometry must be solid before cursor/scrolling work
- Roadmap: Autocomplete grouped with YAML front matter in Phase 6 (both are differentiators requiring stable editor)
- Roadmap: Documentation last (Phase 8) -- docs written after features are stable to avoid churn
- 01-01: Used go-version-file: go.mod instead of hardcoded Go version to prevent CI/go.mod drift
- 01-01: Stayed on Go 1.24.x (1.24.12) rather than jumping to 1.25.x for release stability
- 01-01: Fixed viewport height off-by-one by correcting overhead calculation from 5 to 6
- 01-02: Used runewidth.RuneWidth(rune) instead of runewidth.StringWidth(string(rune)) for per-character width in geometry wrapping loop
- 01-02: Kept ComputeAlignedModel in editor package (not geometry) -- depends on editor-specific types
- 01-02: Dependency direction: geometry <- editor (geometry has zero TUI framework imports)
- 02-01: Used dedicated TestEditorCatwalkLayoutAlignment instead of TestEditorCatwalk -- avoids shared document mutation between catwalk test files
- 02-01: Used ComputeAlignedModel with mock renderers for narrow-width SC tests -- tests alignment algorithm in isolation
- 02-02: No code changes needed -- all five SC tests pass on existing pipeline
- 02-02: Visual polish (column headers, divider padding/centering) deferred as out of Phase 2 scope
- 03-01: Use unicode.IsSpace/IsPunct for word boundaries -- standard word boundaries, not CalcMark expression-aware
- 03-01: Dedicated test functions for navigation tests avoid shared document pollution between catwalk test walks
- 03-02: scrollMargin = 3 lines provides good context without excessive scrolling
- 03-02: adjustScrollForCursor() centralizes all scroll logic for consistency
- 03-02: Arrow key navigation calls adjustScrollForCursor() via saveCurrentLineAndMoveTo()
- 03-03: evalDebounceDelay = 100ms (conservative default, can tune lower)
- 03-03: Dedicated test functions with fresh documents for evaluation tests avoid shared state pollution
- 03-04: Keep Model (model.go), delete ModelV2 (model_v2.go) -- Model is complete (2003 lines), ModelV2 was incomplete (665 lines)
- 03-04: app.go already used Model, no changes needed -- ModelV2 was experimental code never integrated
- 03-04: Delete redundant tests that only tested ModelV2 -- visual_layout_test.go already tests same properties for Model
- 03-05: Backspace at col 0 joins with previous line; delete at end joins with next line -- standard text editor behavior
- 03-05: Anonymous calculations show only value, not "varName → value" -- only assignments show variable names
- 03-05: Diagnostic messages use em-dash pattern with specific values and suggestions -- "extremely useful for users"
- 03-05: Variable redefinition shows original definition line number -- helps users find first assignment
- 04-01: Dedicated test functions with fresh documents for insert_at_end, insert_line, scroll_navigation, delete_empty_line -- pattern now applied to 16 test scenarios
- 04-02: Created typing_text, text_wrapping_40col, long_document_scroll tests -- all SC1 interaction types now covered
- 04-02: VHS tapes archived to branch archive/vhs-tapes and removed from filesystem -- not in CI, reducing clutter

### Visual Polish Backlog (from Phase 2 visual checkpoint)

These items were identified during 02-02 visual verification and deferred:
1. Column headers ("Source" / "Preview") not displayed at top of columns
2. Vertical pipe divider has no padding from source content (cramped appearance)
3. Divider not visually centered between panes

### Pending Todos

None.

### Blockers/Concerns

- ~~CI release workflow uses Go 1.21 but go.mod requires 1.24.x~~ RESOLVED in 01-01
- ~~Two editor implementations in flight (Model vs ModelV2)~~ RESOLVED in 03-04: ModelV2 deleted, Model is the single implementation
- ~~TestEditorCatwalk test failures (insert_at_end, insert_line, scroll_navigation, delete_empty_line)~~ RESOLVED in 04-01: dedicated test functions with fresh documents
- WASM binary size unknown -- must measure early in Phase 7
- Pre-existing `task quality` modernize warnings (~39 across codebase) -- not blocking but should be addressed eventually
- TestEditorCatwalkCompression has pre-existing failures (compression/insert_line, compression/type_new_line) -- different issue, not related to document mutation

## Session Continuity

Last session: 2026-02-03
Stopped at: Completed 04-02-PLAN.md - Added 3 missing catwalk tests
Resume file: None

### Phase 4 Progress Notes
- 04-01: Fixed 4 failing tests (insert_at_end, insert_line, scroll_navigation, delete_empty_line) by applying dedicated test function pattern
- 04-02: Added 3 new catwalk tests (typing_text, text_wrapping_40col, long_document_scroll), archived VHS tapes
- Remaining work: 04-03 (binary-reproducible test harness - may be optional)
