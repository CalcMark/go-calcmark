---
phase: 03
plan: 03
subsystem: tui-editor
tags: [evaluation, debounce, live-preview, catwalk-tests]
dependency-graph:
  requires:
    - 03-01 (cursor navigation)
    - 03-02 (viewport scrolling)
  provides:
    - 100ms evaluation debounce delay
    - Catwalk tests for evaluation pipeline
    - Catwalk tests for dependent variable updates
  affects:
    - 03-04 (error display in preview)
tech-stack:
  added: []
  patterns:
    - Debounced re-evaluation after typing pauses
    - tea.Tick with snapshot for stale edit detection
    - Catwalk data-driven tests for evaluation behavior
key-files:
  created:
    - cmd/calcmark/tui/editor/testdata/evaluation_debounce
    - cmd/calcmark/tui/editor/testdata/dependent_results
  modified:
    - cmd/calcmark/tui/editor/model.go
    - cmd/calcmark/tui/editor/catwalk_test.go
decisions:
  - id: 03-03-debounce
    choice: "100ms debounce delay (configurable constant)"
    context: "Balance between responsiveness and performance"
    rationale: "Conservative default, can tune lower if interpreter allows"
  - id: 03-03-dedicated-tests
    choice: "Dedicated test functions with fresh documents"
    context: "TestEditorCatwalk shares document causing mutation"
    rationale: "Avoids shared state pollution between catwalk test files"
metrics:
  duration: ~8min
  completed: 2026-02-03
---

# Phase 3 Plan 3: Evaluation Debounce Configuration Summary

100ms evaluation debounce with catwalk tests proving dependent variable updates work correctly.

## One-liner

100ms evaluation debounce with tests verifying dependent variable propagation through tax/price/total chain.

## What Was Done

### Task 1: Update debounce delay to 100ms (57b91e1)
- Changed `evalDebounceDelay` from 50ms to 100ms
- Added documentation comment explaining purpose and tuning guidance
- Per CONTEXT.md decision: "Results update on a ~100ms debounce after typing pauses"

### Task 2: Create catwalk tests for evaluation debounce (ea52cf0)
- Created `testdata/evaluation_debounce` with tests for:
  - Initial evaluation: calculations show results (rate, principal, interest)
  - Editing triggers re-evaluation: changing principal updates dependent interest
  - Non-calculation lines show blank in results
- Added `TestEditorCatwalkEvaluationDebounce` test function with fresh document

### Task 3: Create catwalk tests for dependent variable updates (eb6057e)
- Created `testdata/dependent_results` with tests for:
  - Initial dependent calculation: total = price * (1 + tax) = 110
  - Changing source variable: tax 10% -> 20%, total 110 -> 120
  - Multiple dependent updates: price 100 -> 200, total 120 -> 240
- Added `TestEditorCatwalkDependentResults` test function with fresh document
- Verifies evaluation pipeline propagates changes to dependencies

## Key Implementation Details

### Evaluation Pipeline Flow
1. User types -> `handleRuneInput()` -> `transitionToEditing()`
2. `debounceUpdate()` schedules tea.Tick with editBuf snapshot
3. After 100ms, `evalDebounceMsg` received
4. If editBuf matches snapshot -> `transitionToProcessing()`
5. `transitionToProcessing()` calls `reEvaluate()` -> dependent results update
6. Results visible via `GetLineResults()` for preview pane rendering

### Test Documents
- **evaluation_debounce**: rate=10%, principal=1000, interest=principal*rate
- **dependent_results**: tax=10%, price=100, total=price*(1+tax)

Both documents test the dependency chain: changing a source variable propagates to dependents.

## Verification Results

| Check | Status |
|-------|--------|
| `go build ./cmd/calcmark/...` | PASS |
| evalDebounceDelay is 100ms | PASS |
| evaluation_debounce tests pass | PASS |
| dependent_results tests pass | PASS |
| Pre-existing TestEditorCatwalk failures | KNOWN (shared document mutation) |

## Deviations from Plan

None - plan executed exactly as written.

## Success Criteria Met

- [x] Debounce delay is 100ms (configurable constant)
- [x] Typing triggers re-evaluation after debounce
- [x] Results update correctly in preview pane
- [x] Dependent variables update when source changes
- [x] Non-calculation lines show blank in results
- [x] Previous results remain visible during evaluation (no flicker - verified by editBuf state in tests)

## Next Phase Readiness

**Ready for 03-04**: Error display in preview pane

The evaluation pipeline is confirmed working. The next plan (03-04) will add error display showing:
- Error indicator (X) in source pane
- Full diagnostic message in results/preview pane

The `results` observer already shows error strings, so the foundation is in place.

## Notes

The pre-existing `TestEditorCatwalk` and `TestEditorCatwalkCompression` failures are caused by shared document mutation (documented in STATE.md). These failures predate this plan and are not caused by the changes here. The dedicated test functions (`TestEditorCatwalkEvaluationDebounce`, `TestEditorCatwalkDependentResults`) use fresh documents to avoid this issue.
