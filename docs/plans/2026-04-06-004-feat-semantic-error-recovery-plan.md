---
title: "feat: Semantic error recovery — interpret statements that pass semantic checking"
type: feat
status: active
date: 2026-04-06
origin: docs/brainstorms/2026-04-06-semantic-error-recovery-requirements.md
---

# Semantic Error Recovery

## Overview

When a CalcMark block has semantic errors, the evaluator currently aborts before interpreting any statements. This change makes the evaluator skip only the errored statements and interpret the rest, reusing the statement-level error recovery infrastructure from #112.

## Problem Frame

After #112, the evaluator recovers from runtime errors (division by zero, cascading references) statement-by-statement. But semantic errors (variable redefinition, undefined variables) still abort the entire block at `impl/document/evaluator.go:727-734`. Valid statements — including those that would produce runtime errors like `1 / 0` — never run, making their results invisible.

(see origin: `docs/brainstorms/2026-04-06-semantic-error-recovery-requirements.md`)

## Requirements Trace

- R1. Interpret statements that passed semantic checking; skip errored ones
- R2. Variables from skipped statements marked as errored → CascadingError propagation
- R3. Runtime errors on valid statements are captured and displayed
- R4. Both evaluation paths: `evaluateCalcBlockWithDoc` and `evaluateCalcBlockSelective`
- R5. Semantic and runtime diagnostics coexist on the same block
- R6. Reuse #112's statement-level recovery, don't duplicate

## Scope Boundaries

- Not changing the semantic checker — it already collects all diagnostics correctly
- Not changing the diagnostic data model or display format
- Not recovering from parse errors (different mechanism)

## Context & Research

### Relevant Code and Patterns

- `impl/document/evaluator.go:697-735` — Semantic error abort in `evaluateCalcBlockWithDoc`
- `impl/document/evaluator.go:480-520` — Parallel abort in `evaluateCalcBlockSelective`
- `impl/document/evaluator.go:753-801` — Existing statement-level recovery loop (the target merge point)
- `spec/semantic/checker.go:153` — `Check()` returns `[]Diagnostic` with `Range` (line numbers)
- `spec/semantic/diagnostics.go:9-18` — Severity levels: Error, Warning, Hint
- `impl/interpreter/errors.go` — `CascadingError` type
- `impl/interpreter/environment.go:85-112` — `SetError`/`GetError` for per-variable error tracking
- `spec/parser/ast/ast.go` — `Node.GetRange()` provides line numbers for mapping diagnostics to nodes

### Institutional Learnings

- #112 established the pattern: evaluate statements one-by-one, `nil` placeholder for failures, `continue` to next. This plan extends that pattern to semantic errors.
- CascadingError with severity "hint" distinguishes root causes from downstream effects — same distinction applies here.

## Key Technical Decisions

- **Map semantic errors to AST nodes by line number:** Semantic diagnostics have `Range.Start.Line`. AST nodes have `GetRange().Start.Line`. Build a set of errored line numbers from semantic diagnostics, then check each node against it before interpretation. This is O(n) and uses data already available.

- **No block-level `SetError` on semantic abort:** Currently `block.SetError(firstSemanticErr)` is called, which sets a block-level error that some formatters display as a banner. After this change, individual semantic diagnostics (already added via `block.AddDiagnostic`) are sufficient. Remove the `block.SetError` call for semantic errors — runtime errors still set it if needed.

- **Errored nodes get same treatment as runtime failures:** Skip interpretation, append `nil` result, mark variable as errored via `env.SetError`. This produces identical downstream behavior (CascadingError) regardless of whether the error was semantic or runtime.

## Open Questions

### Resolved During Planning

- **How to identify which statements have semantic errors?** By line number. `Diagnostic.Range.Start.Line` maps to `ast.Node.GetRange().Start.Line`. Build `map[int]bool` of errored lines from semantic diagnostics.
- **What about multi-line statements?** AST nodes report their start line. A semantic error on a variable redefinition points to the assignment's start line. One-to-one mapping is sufficient.
- **Should `block.SetError` still be called?** Only if there are runtime errors (existing behavior). Semantic errors are already surfaced via per-line diagnostics. Removing the block-level error for semantic-only cases avoids the redundant "Error:" banner when per-line diagnostics already show the issues.

### Deferred to Implementation

- Whether `evaluateCalcBlockSelective` needs any special handling beyond the same pattern — likely identical, but the environment setup differs slightly.

## Implementation Units

- [ ] **Unit 1: Build semantic-error-to-node mapping and skip errored statements**

**Goal:** Replace the early-return abort with a per-statement skip using line-number mapping, then fall through to the existing interpretation loop.

**Requirements:** R1, R2, R3, R4, R5, R6

**Dependencies:** None

**Files:**
- Modify: `impl/document/evaluator.go`
- Test: `impl/document/evaluator_test.go`

**Approach:**
- In both `evaluateCalcBlockWithDoc` (lines 697-735) and `evaluateCalcBlockSelective` (lines 480-520):
  1. Keep the diagnostic collection loop as-is (it already adds per-line diagnostics to the block).
  2. Instead of returning early when `firstSemanticErr != nil`, build a `semanticErrorLines map[int]bool` from the collected diagnostics.
  3. Remove `block.SetError(firstSemanticErr)` for the semantic-only case.
  4. Mark variables from errored statements as errored: iterate nodes, check if the node's start line is in `semanticErrorLines`, and if so, call `env.SetError(assign.Name, ...)`.
  5. Fall through to the interpretation loop. In the loop, before calling `interp.Eval`, check if the node's start line is in `semanticErrorLines`. If so, skip it (append `nil`, `continue`) — same as the existing runtime error handling.
- This reuses the entire existing statement-level recovery loop (R6). The only new code is the line-number check before `interp.Eval`.

**Patterns to follow:**
- Existing statement-level recovery at `evaluator.go:753-794`
- `env.SetError(name, err)` pattern at `evaluator.go:790-792`

**Test scenarios:**
- Happy path: Block with `a = 1 / 0`, `a = 2` (redefine), `c = 3`, `c = 5` (redefine) → `a` shows division-by-zero, `a = 2` shows redefinition error, `c = 3` shows `= 3`, `c = 5` shows redefinition error
- Happy path: Block with one semantic error and one valid statement → valid statement produces result
- Happy path: Block with only semantic errors → all diagnostics shown, no crash
- Edge case: Statement that references a semantically-errored variable → CascadingError diagnostic with "hint" severity
- Edge case: Block with no semantic errors → unchanged behavior (interpretation runs as before)
- Edge case: Block where all statements have semantic errors → diagnostics shown, results are all nil, no block-level error banner
- Integration: Multi-block document where block 2 references a variable from block 1 that had a semantic error → CascadingError propagation across blocks
- Integration: Selective evaluation path (`evaluateCalcBlockSelective`) produces same behavior as full evaluation path

**Verification:**
- `task test` passes with no regressions
- The example from issue #113 produces the expected output
- Existing #112 error recovery tests remain green

## System-Wide Impact

- **Interaction graph:** The evaluator is called from document evaluation, TUI updates, LSP analysis, and CLI eval. All paths benefit from this change via the same code path.
- **Error propagation:** Semantic errors stay as per-line diagnostics. Runtime errors on valid statements now surface alongside them. CascadingError propagation is unchanged.
- **API surface parity:** No API changes. The diagnostic model is unchanged. Formatters (HTML, JSON, text, TUI) already handle per-line diagnostics from #112.
- **Unchanged invariants:** Semantic checker behavior, diagnostic data model, CascadingError type, block-level error for runtime-only cases.

## Risks & Dependencies

| Risk | Mitigation |
|------|------------|
| Removing `block.SetError` for semantic cases changes formatter output | Semantic diagnostics are already per-line; block-level error was redundant. Verify all format tests pass. |
| Line-number mapping misses edge cases (blank lines, comments) | Semantic diagnostics only fire on real statements. AST nodes only exist for real statements. Blank lines have no nodes. |
| Selective eval path has different environment setup | Same pattern applies — the skip logic is identical. Test both paths explicitly. |

## Sources & References

- **Origin document:** [docs/brainstorms/2026-04-06-semantic-error-recovery-requirements.md](docs/brainstorms/2026-04-06-semantic-error-recovery-requirements.md)
- Related issue: #113
- Related PR: #112 (statement-level error recovery)
- Key file: `impl/document/evaluator.go`
