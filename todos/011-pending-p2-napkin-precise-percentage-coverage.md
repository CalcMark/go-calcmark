---
status: pending
priority: p2
issue_id: "011"
tags: [code-review, pattern-consistency, type-system]
dependencies: []
---

# Missing Percentage in napkin_eval.go and precise_eval.go

## Problem Statement

`evalNapkinConversion` and `evalPreciseConversion` handle all numeric types (Number, Quantity, Currency, Duration, Rate) but not Percentage. A user writing `20% as napkin` or `20% as precise` would get an unhelpful error.

## Findings

- Pattern-recognition-specialist identified the gap
- `napkin_eval.go:30-74` — missing `*types.Percentage` case
- `precise_eval.go:19-43` — missing `*types.Percentage` case
- Both should pass through unchanged (rounding a percentage value is semantically odd)

## Technical Details

**Affected files:**
- `impl/interpreter/napkin_eval.go`
- `impl/interpreter/precise_eval.go`

**Fix:** Add `*types.Percentage` case that passes through unchanged (return the percentage as-is), or add a clear error message explaining why napkin/precise don't apply to percentages.

## Acceptance Criteria

- [ ] `20% as napkin` either works or produces a clear error
- [ ] `20% as precise` either works or produces a clear error
- [ ] Pattern-consistent with other type handling
