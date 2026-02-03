# Phase 04 Plan 01: Fix Shared Document Mutation Summary

**Completed:** 2026-02-03
**Duration:** 3 minutes

## One-liner

Fixed 4 failing catwalk tests by creating dedicated test functions with fresh documents, eliminating shared state pollution.

## What Was Done

### Task 1: Create dedicated test functions for failing tests (Commit: 7bcb5ac)
- Added `TestEditorCatwalkInsertAtEnd` - isolated test for insert_at_end
- Added `TestEditorCatwalkInsertLine` - isolated test for insert_line
- Added `TestEditorCatwalkScrollNavigation` - isolated test for scroll_navigation
- Added `TestEditorCatwalkDeleteEmptyLine` - isolated test for delete_empty_line with view observer
- Updated skipTests array in TestEditorCatwalk to exclude all 4 test names
- Each function creates fresh `*document.Document` with "# Header, x=10, y=20, z=30" content

### Task 2: Regenerate test expectations with fresh documents (Commit: 8a93efd)
- Regenerated testdata/insert_at_end with clean document state
- Regenerated testdata/insert_line with clean document state
- Regenerated testdata/scroll_navigation with clean document state
- Regenerated testdata/delete_empty_line with clean document state
- Verified no "jjjotestline" pollution in regenerated expectations

### Task 3: Run full test suite and verify zero failures (Verification only)
- All 4 previously failing tests now pass
- 15 catwalk test functions pass
- 1 pre-existing failure (TestEditorCatwalkCompression) is out of scope for this plan

## Files Modified

| File | Change Type | Purpose |
|------|-------------|---------|
| cmd/calcmark/tui/editor/catwalk_test.go | Modified | Added 4 dedicated test functions, updated skipTests |
| cmd/calcmark/tui/editor/testdata/insert_at_end | Modified | Regenerated with fresh document |
| cmd/calcmark/tui/editor/testdata/insert_line | Modified | Regenerated with fresh document |
| cmd/calcmark/tui/editor/testdata/scroll_navigation | Modified | Regenerated with fresh document |
| cmd/calcmark/tui/editor/testdata/delete_empty_line | Modified | Regenerated with fresh document |

## Decisions Made

| Decision | Rationale |
|----------|-----------|
| Follow established pattern from TestEditorCatwalkViewportScrolling | Consistent with existing codebase, proven to work |
| Use same "# Header, x=10, y=20, z=30" document for all 4 tests | Simple, predictable initial state for edit operations |
| Add view observer to TestEditorCatwalkDeleteEmptyLine | Needed for verifying visual DELETE key behavior |

## Deviations from Plan

None - plan executed exactly as written.

## Verification Results

| Criterion | Result |
|-----------|--------|
| `go test ./cmd/calcmark/tui/editor/... -run Catwalk -v` shows tests passing | PASS (15/16 pass, 1 out-of-scope pre-existing failure) |
| No test output shows polluted content like "jjjotestline" | PASS (grep finds no occurrences) |
| 4 new dedicated test functions exist in catwalk_test.go | PASS |
| skipTests array includes all 4 test file names | PASS |
| `task test` passes (no regressions) | PARTIAL (pre-existing TestEditorCatwalkCompression failure) |

## Next Phase Readiness

**Blockers:** None

**Concerns:**
- TestEditorCatwalkCompression has pre-existing failures (compression/insert_line and compression/type_new_line) - not introduced by this work, out of scope for 04-01

**Notes:**
- The remaining failing test (TestEditorCatwalkCompression) uses the compression subdirectory with different document content and should be investigated separately
- The dedicated test function pattern is now applied to 16 test scenarios total (12 existing + 4 new)

## Commits

| Hash | Message |
|------|---------|
| 7bcb5ac | test(04-01): add dedicated test functions for insert_at_end, insert_line, scroll_navigation, delete_empty_line |
| 8a93efd | test(04-01): regenerate test expectations with fresh documents |
