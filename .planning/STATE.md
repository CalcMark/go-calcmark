# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-02-02)

**Core value:** Fast, offline, verifiable calculations in markdown documents with a simple editor
**Current focus:** Phase 8 - Documentation (READY)

## Current Position

Phase: 8 of 8 (Documentation)
Plan: 0 of 3 in current phase
Status: Ready to start
Last activity: 2026-02-04 - Completed Phase 7

Progress: [████████████████████] 95% (19/20 plans)

## Performance Metrics

**Velocity:**
- Total plans completed: 18
- Average duration: 8min
- Total execution time: ~2.5 hours

**By Phase:**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| 1. Foundation | 2/2 | 13min | 6.5min |
| 2. TUI Geometry & Layout | 2/2 | 8min | 4min |
| 3. TUI Editor Integration | 5/5 | 55min | 11min |
| 4. TUI Test Coverage | 3/3 | 15min | 5min |
| 5. Help System | 2/2 | 19min | 9.5min |
| 6. Differentiators | 2/2 | 57min | 28.5min |
| 7. Distribution | 2/2 | 13min | 6.5min |

**Recent Trend:**
- Last 5 plans: 05-02 (15min), 06-01 (12min), 06-02 (45min), 07-01 (5min), 07-02 (8min)
- Trend: Infrastructure tasks are fast; feature development takes longer

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
- 03-05: Anonymous calculations show only value, not "varName -> value" -- only assignments show variable names
- 03-05: Diagnostic messages use em-dash pattern with specific values and suggestions -- "extremely useful for users"
- 03-05: Variable redefinition shows original definition line number -- helps users find first assignment
- 04-01: Dedicated test functions with fresh documents for insert_at_end, insert_line, scroll_navigation, delete_empty_line -- pattern now applied to 16 test scenarios
- 04-02: Created typing_text, text_wrapping_40col, long_document_scroll tests -- all SC1 interaction types now covered
- 04-02: VHS tapes archived to branch archive/vhs-tapes and removed from filesystem -- not in CI, reducing clutter
- 04-03: Fixed compression test shared document mutation, same pattern as 04-01 -- all tests now pass 10 consecutive runs
- 05-01: Registry sync test parses functions.go AST to verify registry matches implementation
- 05-01: Categories displayed in logical order (Math, Conversion, Network, Storage, Capacity) not alphabetical
- 05-01: Plain text output (no ANSI codes) ensures compatibility when piping to less/more
- 05-01: Removed DisableDefaultCmd to enable Cobra built-in completion
- 05-02: Use F1 for help (not ?) to avoid conflict with calc expressions
- 05-02: Status bar shows L{line}:{col} format (e.g., L5:12)
- 05-02: Show EVAL... during typing debounce, calc count when idle
- 06-01: FunctionDef struct with metadata + Eval field using init() to avoid initialization cycle
- 06-01: Added isFunctionToken to detector for built-in function recognition
- 06-01: "mean" added as synonym for avg (needed for SC4 autocomplete)
- 07-01: Use GoReleaser v2 with brews section (brews is still correct for CLI formulas; homebrew_casks is for GUI apps)
- 07-01: WASM infrastructure removed -- no longer needed, simplifies build and CI
- 07-02: Correct Homebrew tap syntax is `calcmark/tap/calcmark` (lowercase org name)
- 07-02: 7 active DIST requirements after removing WASM/Scoop/man pages/bundled completions (DIST-06 through DIST-10 removed)

### Visual Polish Backlog (from Phase 2 visual checkpoint)

These items were identified during 02-02 visual verification and deferred:
1. Column headers ("Source" / "Preview") not displayed at top of columns
2. Vertical pipe divider has no padding from source content (cramped appearance)
3. Divider not visually centered between panes

### Pending Todos

- ~~**BUG (2026-02-03)**: `avg(2,4,4)` and other function calls not showing results in Preview pane of TUI editor.~~ RESOLVED in 06-01: Added isFunctionToken to detector
- **IDEA (2026-02-04)**: Preview pane should only include calculation outputs (not full rendered markdown), while preserving strict vertical alignment. Consider for post-v1 UX enhancement.

### Blockers/Concerns

- ~~CI release workflow uses Go 1.21 but go.mod requires 1.24.x~~ RESOLVED in 01-01
- ~~Two editor implementations in flight (Model vs ModelV2)~~ RESOLVED in 03-04: ModelV2 deleted, Model is the single implementation
- ~~TestEditorCatwalk test failures (insert_at_end, insert_line, scroll_navigation, delete_empty_line)~~ RESOLVED in 04-01: dedicated test functions with fresh documents
- ~~TestEditorCatwalkCompression has pre-existing failures (compression/insert_line, compression/type_new_line)~~ RESOLVED in 04-03: dedicated test functions with fresh documents
- ~~Function result display bug~~ RESOLVED in 06-01: isFunctionToken added to detector
- ~~WASM binary size unknown~~ RESOLVED in 07-01: WASM infrastructure removed
- Pre-existing `task quality` modernize warnings (~39 across codebase) -- not blocking but should be addressed eventually

### Phase 6 Architecture Context

**Function metadata refactor (COMPLETED 2026-02-04):**
Implementation: `FunctionDef` struct in `impl/interpreter/functions.go` with:
- Metadata fields: Name, Synonyms, Description, Signature, Category
- Eval field: function pointer populated via init() to avoid Go initialization cycle
- `BuiltinFunctions` slice is single source of truth for all 12 functions
- Registry helpers derive FunctionInfo from BuiltinFunctions

Benefits achieved:
1. Adding a function without metadata causes test failure (TestFunctionInfoComplete)
2. Help docs, autocomplete, and diagnostics all use same source
3. "mean" synonym available for autocomplete (Plan 02)

### Phase 7 Architecture Context

**GoReleaser setup (COMPLETED 2026-02-04):**
- `.goreleaser.yaml` configured for 7 platform builds
- GitHub Actions workflow uses goreleaser/goreleaser-action@v6
- Homebrew formula generation configured for calcmark/homebrew-tap
- WASM infrastructure removed (impl/wasm/, release.sh, Taskfile tasks)
- Local snapshot builds validated for all 7 platforms
- Planning docs updated: ROADMAP.md and REQUIREMENTS.md reflect final scope

**User setup required before first release:**
- Create `calcmark/homebrew-tap` repository on GitHub (public)
- Create PAT with 'repo' scope
- Add `HOMEBREW_TAP_GITHUB_TOKEN` secret to go-calcmark repo

## Session Continuity

Last session: 2026-02-04
Stopped at: Completed 07-02-PLAN.md (Phase 7 complete)
Resume file: None

### Phase 7 Progress (2026-02-04) - COMPLETE
- 07-01: GoReleaser configuration, WASM removal, release workflow update
- 07-02: Local validation, ROADMAP/REQUIREMENTS updates, release readiness verified
