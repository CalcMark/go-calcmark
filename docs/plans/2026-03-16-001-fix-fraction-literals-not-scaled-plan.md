---
title: "fix: Fraction literals not scaled by scale transform"
type: fix
status: completed
date: 2026-03-16
---

# fix: Fraction literals not scaled by scale transform

## Enhancement Summary

**Deepened on:** 2026-03-16
**Sections enhanced:** 5
**Research agents used:** performance-oracle, code-simplicity-reviewer, pattern-recognition-specialist, security-sentinel, spec-flow-analyzer, convert_to explorer

### Key Improvements
1. Added `ExceedsComputationLimit()` safety check after scaling (security-critical)
2. Added `convert_to` support for fractions with known units (same class of bug)
3. Fixed `decimal.Decimal.Rat()` API usage (no-arg method, not `Rat(nil)`)
4. Changed `applyScaleFraction` return type to `types.Type` for safe fallback

### New Considerations Discovered
- Fractions with known units (e.g., `1/3 cup`) also need `convert_to` handling
- Non-integer scale factors can produce fractions exceeding computation limits
- `WouldConvert` also needs a `*types.Fraction` case for TUI indicator

## Overview

Fraction literals like `1/2 tomato` are silently ignored by the scale transform, while their decimal equivalents (`.5 tomato`) scale correctly. This is because `*types.Fraction` is missing from the type switch in `Apply()` and `WouldScale()`.

## Problem Statement

Given this CalcMark document:

```
---
scale:
  factor: 4
  unit_categories: [Mass, Volume, Custom]
convert_to: si
---
avocados = 3 avocados
1/2 tomato
.5 tomato
```

- `.5 tomato` → evaluates to `*types.Quantity{Value: 0.5, Unit: "tomato"}` → **scales to 2 tomato** (correct)
- `1/2 tomato` → evaluates to `*types.Fraction{Value: 1/2, Unit: "tomato"}` → **stays 1/2 tomato** (bug)

Both expressions represent the same value (0.5 tomato) and should scale identically.

## Root Cause

In `spec/transform/transform.go`:

1. **`Apply()` (line 24-57)**: The type switch handles `*types.Quantity`, `*types.Rate`, `*types.Currency`, and `*types.Number`. `*types.Fraction` falls through to the `default` case and is returned unchanged.

2. **`WouldScale()` (line 62-77)**: Same missing case — the TUI scale indicator never shows for fraction values.

3. **`impl/document/evaluator.go` `unitOf()` (line 730-739)**: Missing `*types.Fraction` case means `convert_to` detection also fails for fractions.

## Proposed Solution

Add `*types.Fraction` handling to all affected functions. Fractions should scale by multiplying their `big.Rat` value by the scale factor, **preserving exact rational representation** when possible, with safe fallback to decimal when computation limits are exceeded.

### Design Decision: Preserve Fraction Type After Scaling

When `1/3 cup` is scaled by 2, the result should be `2/3 cup` (still a Fraction), not `0.666... cup` (a Quantity). This preserves the user's intent to work with exact fractions and maintains display consistency.

The scale factor (`decimal.Decimal`) can be converted to `*big.Rat` via `scale.Factor.Rat()` (no arguments — returns a new `*big.Rat`).

### Design Decision: convert_to Degrades Fractions to Quantity

When `1/3 cup` has `convert_to: si`, the fraction must convert to milliliters. Unit conversion factors (e.g., cup → ml = 236.588) are inherently non-rational, so the result must become a `*types.Quantity`. This is the same loss-of-exactness that happens when fractions exceed computation limits — it's acceptable because the conversion itself is approximate.

### Research Insights

**Performance:**
- `big.Rat.Mul` is O(1) for bounded integer sizes (guaranteed by computation limits)
- 2 heap allocations per scaled fraction (~128 bytes) — negligible at document scale
- No need to cache `factorRat` in `ApplyToResults` unless profiling indicates otherwise

**Security:**
- Must check `ExceedsComputationLimit()` after scaling — every arithmetic code path in the interpreter does this (`fraction_ops.go:79-85`)
- Extreme scale factors (e.g., `1e308`) could produce unbounded `big.Int` values without this check
- The frontmatter parser rejects NaN/Inf but has no upper bound on scale factor magnitude

**Pattern Consistency:**
- Use the `out` variable idiom from the Quantity case for extensibility
- `applyScaleFraction` must return `types.Type` (not `*types.Fraction`) to allow fallback to Quantity/Number when limits are exceeded

## Acceptance Criteria

- [x] `1/2 tomato` scales to `2 tomato` with `scale: {factor: 4, unit_categories: [Custom]}`
- [x] `1/3 cup` scales to `4/3 cup` (or `1 1/3 cup`) with `scale: {factor: 4, unit_categories: [Volume]}`
- [x] Dimensionless fractions (no unit) only scale when `"Number"` is in `unit_categories`
- [x] `WouldScale()` returns true for fractions that would be scaled
- [x] `unitOf()` returns the fraction's unit for `convert_to` detection
- [x] Scaled fractions remain `*types.Fraction` when within computation limits
- [x] Scaled fractions degrade to `*types.Quantity` or `*types.Number` when `ExceedsComputationLimit()` triggers
- [x] `1/3 cup` with `convert_to: si` converts to liter (as `*types.Quantity`)
- [x] `WouldConvert()` returns true for fractions with convertible known units
- [x] Original fraction values are not mutated (clone before scaling)
- [x] Napkin fractions (`~1/3 cup`) scale correctly with `IsNapkin` preserved
- [x] All existing tests continue to pass

## Technical Approach

### Files to Modify

#### 1. `spec/transform/transform.go` — `Apply()`

Add a `*types.Fraction` case between `*types.Number` and `default`, following the Quantity pattern with the `out` variable idiom:

```go
case *types.Fraction:
    // Fractions support scaling. convert_to requires decimal precision,
    // so convert to Quantity for unit conversion when needed.
    out := applyScaleFraction(cloneFraction(v), scale)
    if convertTo != nil {
        if frac, ok := out.(*types.Fraction); ok && frac.Unit != "" {
            qty := &types.Quantity{Value: frac.ToDecimal(), Unit: frac.Unit, IsNapkin: frac.IsNapkin}
            return applyConvertToQuantity(qty, convertTo)
        }
    }
    return out
```

#### 2. `spec/transform/transform.go` — New `applyScaleFraction()` helper

Returns `types.Type` to allow safe fallback:

```go
func applyScaleFraction(f *types.Fraction, scale *document.ScaleConfig) types.Type {
    if scale == nil || len(scale.UnitCategories) == 0 {
        return f
    }
    var category string
    if f.Unit != "" {
        category = units.CategoryForUnit(f.Unit)
    } else {
        category = "Number"
    }
    if !categoryMatches(category, scale.UnitCategories) {
        return f
    }
    factorRat := scale.Factor.Rat()
    f.Value = new(big.Rat).Mul(f.Value, factorRat)
    // Safety: if scaling exceeded computation limits, fall back to decimal.
    if f.ExceedsComputationLimit() {
        d := f.ToDecimal()
        if f.Unit != "" {
            return &types.Quantity{Value: d, Unit: f.Unit, IsNapkin: f.IsNapkin}
        }
        n := types.NewNumber(d)
        n.IsNapkin = f.IsNapkin
        return n
    }
    return f
}
```

#### 3. `spec/transform/transform.go` — New `cloneFraction()` helper

```go
func cloneFraction(f *types.Fraction) *types.Fraction {
    return &types.Fraction{
        Value:    new(big.Rat).Set(f.Value),
        IsNapkin: f.IsNapkin,
        Unit:     f.Unit,
    }
}
```

#### 4. `spec/transform/transform.go` — `WouldScale()`

Add case before `default`:

```go
case *types.Fraction:
    if v.Unit != "" {
        category := units.CategoryForUnit(v.Unit)
        return categoryMatches(category, scale.UnitCategories)
    }
    return categoryMatches("Number", scale.UnitCategories)
```

#### 5. `spec/transform/transform.go` — `WouldConvert()`

Add case before `default`:

```go
case *types.Fraction:
    if v.Unit == "" {
        return false
    }
    qty := &types.Quantity{Value: v.ToDecimal(), Unit: v.Unit}
    return wouldConvertQuantity(qty, convertTo)
```

#### 6. `impl/document/evaluator.go` — `unitOf()`

Add case:

```go
case *types.Fraction:
    return t.Unit
```

### Tests — `spec/transform/transform_test.go`

New test cases (use table-driven where similar):

- `TestApply_ScaleFractionWithCustomUnit` — `1/2 tomato` × 4 = `2 tomato`
- `TestApply_ScaleFractionWithKnownUnit` — `1/3 cup` × 4 = `1 1/3 cup`
- `TestApply_FractionNotScaledWithoutCategory` — `1/2 cup` unscaled when Volume not in categories
- `TestApply_DimensionlessFractionScaledWithNumber` — `1/4` × 4 = `1` when Number in categories
- `TestApply_DimensionlessFractionImmuneByDefault` — `1/4` unscaled without Number category
- `TestApply_FractionDoesNotMutateOriginal` — clone safety
- `TestApply_FractionExceedsComputationLimit` — extreme scale factor degrades to Quantity
- `TestApply_FractionConvertTo` — `1/3 cup` with `convert_to: si` converts to ml as Quantity
- `TestWouldScale_FractionWithCategory` — returns true for matching category
- `TestWouldScale_DimensionlessFraction` — follows Number rules
- `TestWouldConvert_Fraction` — returns true for convertible known units
- Extend `TestApply_AllCategoryScalesEverything` — add a Fraction value

### Golden Test — `testdata/eval/success/features/scale_fractions.cm`

Integration test combining fractions with scale directive to verify end-to-end behavior including scale + convert_to.

## Edge Cases

| Case | Input | Scale | Result | Type |
|------|-------|-------|--------|------|
| Custom unit | `1/2 tomato` | factor: 4, Custom | `2 tomato` | Fraction (denom=1) |
| Known unit | `1/3 cup` | factor: 4, Volume | `1 1/3 cup` | Fraction |
| Dimensionless | `1/4` | factor: 4, Number | `1` | Fraction (denom=1) |
| Category mismatch | `1/2 tomato` | factor: 4, Mass | `1/2 tomato` | Fraction (unchanged) |
| Non-integer factor | `1/3 cup` | factor: 1.5, Volume | `1/2 cup` | Fraction |
| Exceeds limit | `1/999999937` | factor: 1.123456789, Number | decimal fallback | Number |
| Napkin | `~1/3 cup` | factor: 2, Volume | `~2/3 cup` | Fraction (IsNapkin) |
| Mixed number | `2 1/3 cups` | factor: 3, Volume | `7 cups` | Fraction (denom=1) |
| convert_to | `1/3 cup` | convert_to: si | `~78.86 ml` | Quantity |
| Scale + convert_to | `1/2 cup` | factor: 2, Volume + si | `~236.59 ml` | Quantity |

## Institutional Learnings Applied

From `docs/solutions/logic-errors/adding-new-type-fraction-cross-layer-checklist.md`:
- **Display normalization**: Scaled fractions must flow through the normal display pipeline, not bypass it.
- **Clone before mutate**: Always clone values before transform to avoid aliasing bugs (existing pattern with `cloneQuantity`).

From `docs/solutions/ui-bugs/display-formatter-overrides-explicit-unit-conversion.md`:
- **IsExplicit flag**: Not applicable to fractions currently (fractions don't support `in`/`as` conversion yet), but worth noting for future work.

From `docs/solutions/security-issues/nan-inf-panic-yaml-frontmatter-scale.md`:
- Scale factor validation already rejects NaN/Inf, but has no upper bound. The `ExceedsComputationLimit()` check provides defense-in-depth for fractions.

## Sources

- Bug location: `spec/transform/transform.go:24-57` (Apply), `spec/transform/transform.go:62-77` (WouldScale)
- Secondary fix: `impl/document/evaluator.go:730-739` (unitOf)
- Fraction type: `spec/types/fraction.go:12-16`
- Existing scale tests: `spec/transform/transform_test.go`
- Computation limit pattern: `impl/interpreter/fraction_ops.go:79-85`
- Cross-layer checklist: `docs/solutions/logic-errors/adding-new-type-fraction-cross-layer-checklist.md`
- Security precedent: `docs/solutions/security-issues/nan-inf-panic-yaml-frontmatter-scale.md`
