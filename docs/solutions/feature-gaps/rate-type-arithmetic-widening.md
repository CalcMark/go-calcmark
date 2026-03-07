---
title: "Rate arithmetic widening: drop time denominator when rate is on RIGHT of * or /"
category: feature-gaps
tags:
  - rates
  - arithmetic
  - type-widening
  - binary-operators
  - type-system
module: impl/interpreter/operators.go
symptom: >
  Arithmetic between non-rate and rate types failed or produced unexpected results:
  Number / Rate returned "cannot divide number and rate" error;
  Number * Rate returned a Rate (preserving time denominator) instead of a Quantity.
root_cause: >
  The binary operator type dispatch in operators.go had no handling for
  mixed Number/Quantity operands with Rate operands on the right side.
  Rate was treated as an opaque type that could only be scaled
  (Rate * Number) but not decomposed. Missing asymmetric widening rule:
  when a Rate appears on the RIGHT side of * or /, its time denominator
  should be dropped and only its amount used.
date: 2026-03-07
---

# Rate Arithmetic Widening

## Problem Statement

CalcMark rates (e.g., `2 posts/week`, `100 MB/s`) had limited arithmetic interoperability. When a user wrote `3 * (2 posts/week)`, the result was `6 posts/week` (a Rate) instead of the expected `6 posts` (a Quantity). Additionally, `Number / Rate` was an unsupported operation that produced an error. Users had to use workarounds like `over 1 day` to extract a rate's amount before doing downstream math.

**Symptoms:**

- `daily_reads / daily_posts` failed with "cannot divide number and rate" when `daily_posts` was a Rate
- `2 * (2 posts/week)` returned `4 posts/week` (Rate) instead of `4 posts` (Quantity)
- Users needed `weekly_posts/week over 1 day` workarounds to extract amounts from rates

## Root Cause Analysis

The `evalBinaryOperation()` function in `impl/interpreter/operators.go` used hardcoded nested type assertions for dispatch. `Number * Rate` and `Rate * Number` were treated symmetrically — both always returned a Rate. There was no handler for `Number / Rate` at all. The system lacked a concept of "widening" — extracting a rate's amount by dropping the time denominator based on operand position.

## Solution

Added an asymmetric rate arithmetic widening normalization block at the top of `evalBinaryOperation()`. The rule depends on which side the Rate appears:

- **Rate on RIGHT of `*` or `/`**: Extract the rate's Amount (a Quantity), drop the time denominator, and recurse with the extracted amount. Unitless rates further normalize to Number via existing unitless quantity normalization.
- **Rate on LEFT**: Behavior is unchanged — `Rate * Number → Rate`, `Rate / Number → Rate`, `Rate * Quantity → Quantity`, `Rate / Rate → Number`.

This asymmetry is intentional: the left operand is the "subject" of the expression. `read_rate * 3` scales a rate. `daily_users * posts_per_week` multiplies a count by a rate's amount.

### Core Code Change

`impl/interpreter/operators.go`, near the top of `evalBinaryOperation()`:

```go
// Rate arithmetic widening: when a Rate appears on the RIGHT side of
// * or /, extract the rate's Amount (a Quantity) and drop the time
// denominator. This makes "2 * (2 posts/week)" yield "4 posts".
if operator == "*" || operator == "/" {
    if rightRate, ok := right.(*types.Rate); ok {
        if _, leftIsRate := left.(*types.Rate); !leftIsRate {
            return evalBinaryOperation(left, rightRate.Amount, operator)
        }
    }
}
```

### Type Dispatch Table

| Expression | Left | Right | Result | Behavior |
|---|---|---|---|---|
| `rate * 3` | Rate | Number | **Rate** | Rate on left → scaling |
| `3 * rate` | Number | Rate | **Quantity** | Rate on right → widened |
| `rate / 2` | Rate | Number | **Rate** | Rate on left → scaling |
| `100 / rate` | Number | Rate | **Number** | Rate on right → widened |
| `rate * qty` | Rate | Quantity | **Quantity** | Cross-type extraction |
| `qty * rate` | Quantity | Rate | **Quantity** | Rate on right → widened |
| `rate / rate` | Rate | Rate | **Number** | Same-unit ratio (no widening) |

### CalcMark Examples

```text
posts_rate = 2 posts/week
scaled = posts_rate * 3           -> 6 posts/week  (Rate — rate on left)
total  = 3 * posts_rate           -> 6 posts       (Quantity — rate on right)
half   = posts_rate / 2           -> 1 posts/week  (Rate — rate on left)
```

## Files Modified

- `impl/interpreter/operators.go` — widening normalization block + dead code cleanup
- `impl/interpreter/rate_widening_test.go` — new test file for all widening combinations
- `impl/interpreter/compound_units_test.go` — updated `TestNumberTimesRate` expectations
- `cmd/calcmark/tui/editor/testdata/error_wrong_line_type_mismatch` — updated golden file
- `cmd/calcmark/tui/editor/testdata/error_shows_valid_values` — updated golden file
- `testdata/eval/success/features/rates.cm` — updated comments
- `spec/features/registry.go` — added "rate widening" feature entry
- `site/content/docs/language-reference.md` — full documentation with type dispatch table
- `site/content/docs/user-guide.md` — practical explanation with examples

## Prevention Strategies

### 1. Test the Type Dispatch Matrix

The `rate_widening_test.go` file tests all key combinations. Critical regression guards:

| Test | What it prevents |
|------|-----------------|
| `TestRateWidening_NumberTimesRate` | Core widening — `Number * Rate` extracts amount |
| `TestRateWidening_RateTimesNumber_StaysRate` | Asymmetry preservation — left-Rate NOT widened |
| `TestRateWidening_RateDivRate_Unchanged` | Rate/Rate ratio excluded from widening |
| `TestRateWidening_NumberDivRateWithUnit_Error` | Dimensionally invalid ops still fail |
| `TestRateWidening_EndToEnd` | Full pipeline with variable references |

### 2. Maintain Normalization Order

The widening block recurses into `evalBinaryOperation`, which then hits the unitless quantity normalization (lines 73-78). This two-step recursion is intentional:
1. Rate → Amount (Quantity)
2. Unitless Quantity → Number

If adding new normalizations, keep them ordered and guard against unbounded recursion.

### 3. Gotchas to Watch

- **Commutativity is intentionally broken.** `3 * rate` ≠ `rate * 3` in result type. Documentation must be explicit. The parser must never reorder operands.
- **Rate + Rate is NOT widened.** The guard `operator == "*" || operator == "/"` is intentional. Do not widen for `+` or `-`.
- **Rate / Rate with different PerUnits** falls through to the error handler silently. A dedicated error message for mismatched rate denominators would improve UX.
- **Currency rates** (`$10/hour`) widen to Quantity, not Currency, because `Rate.Amount` is always a `*types.Quantity`. This may surprise users expecting currency output.

## Cross-References

- **Language Reference**: `site/content/docs/language-reference.md` § Rate Arithmetic Widening
- **User Guide**: `site/content/docs/user-guide.md` § Rate Arithmetic Widening
- **Feature Registry**: `spec/features/registry.go` — "rate widening" keyword entry
- **Type definitions**: `spec/types/rate.go`, `spec/types/quantity.go`, `spec/types/number.go`
- **Related plan**: `docs/plans/2026-03-06-feat-nl-type-system-improvements-plan.md`
- **Related solution**: `docs/solutions/logic-errors/go-closure-capturing-stale-value-type.md`
