---
date: 2026-04-06
topic: semantic-error-recovery
issue: 113
---

# Semantic Error Recovery: Interpret Valid Statements

## Problem Frame

When a CalcMark block has semantic errors (variable redefinition, undefined variables), the evaluator aborts before interpreting ANY statements. Statements that passed semantic checking never run, so successful computations and runtime errors are both invisible.

Example:
```
a = 1 / 0    ← no diagnostic shown (interpretation never ran)
a = 2        ← ✗ cannot reassign 'a'
c = 3        ← no value shown
c = 5        ← ✗ cannot reassign 'c'
```

Expected: `a = 1 / 0` shows `✗ division by zero`, `c = 3` shows `= 3`.

PR #112 added statement-level error recovery for runtime errors (division by zero, cascading errors). But semantic errors still abort the block entirely at `impl/document/evaluator.go:735` before the interpretation loop runs.

## Requirements

- R1. After semantic checking, the evaluator runs interpretation for statements that passed semantic checking. Statements with semantic errors are skipped.
- R2. Variables from skipped (semantically errored) statements are marked as errored in the environment, so dependent statements produce `CascadingError` diagnostics — the same propagation mechanism already used for runtime errors.
- R3. Runtime errors (e.g., `1 / 0`) on semantically valid statements are captured and displayed, not hidden by the semantic abort.
- R4. Both evaluation paths handle this: `evaluateCalcBlockWithDoc` (full evaluation) and `evaluateCalcBlockSelective` (reactive/selective evaluation).
- R5. The block's diagnostic output includes both semantic diagnostics and any runtime diagnostics from interpreted statements — they coexist on the same block.
- R6. The existing statement-level error recovery from #112 is reused, not duplicated.

## Non-goals

- Changing the semantic checker itself. It already collects all diagnostics correctly (#112).
- Recovering from parse errors (different mechanism).
- Changing the diagnostic data model or display format.

## Success Criteria

- Given the example above: `a = 1 / 0` shows `✗ division by zero`, `a = 2` shows `✗ cannot reassign 'a'`, `c = 3` shows `= 3`, `c = 5` shows `✗ cannot reassign 'c'`.
- Statements with no semantic or runtime errors produce their computed values.
- Statements that reference errored variables produce cascading error diagnostics.
- `task test` passes. Existing error recovery tests from #112 remain green.
