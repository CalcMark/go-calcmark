---
status: pending
priority: p2
issue_id: "009"
tags: [code-review, architecture, semantic-analysis]
dependencies: []
---

# Semantic Analysis: Percentage * non-Number Types Produce False Diagnostics

## Problem Statement

The semantic checker in `spec/semantic/types.go` only allows `Percentage * Number` and `Number * Percentage` for `*`/`/` operators. But the interpreter handles `Percentage * Currency`, `Percentage * Quantity`, etc. via normalization (converting percentage to Number first). This means `$100 * 20%` passes at runtime but would emit a false diagnostic during semantic analysis.

## Findings

- Architecture-strategist agent identified the gap at `spec/semantic/types.go:130-135`
- The runtime normalizes Percentage to Number at `impl/interpreter/operators.go:93-100` before type dispatch
- Semantic checker doesn't know about this normalization, so it falls through to the type-mismatch error

## Technical Details

**Affected files:**
- `spec/semantic/types.go` — `CheckTypeCompatibility` function

**Fix:** Extend the Percentage rules to accept `Percentage * Currency`, `Percentage * Quantity`, `Percentage * Duration`, `Percentage * Rate` for `*`/`/`.

## Acceptance Criteria

- [ ] `$100 * 20%` does not produce a semantic diagnostic
- [ ] `5 kg * 10%` does not produce a semantic diagnostic
- [ ] Semantic analysis tests updated
