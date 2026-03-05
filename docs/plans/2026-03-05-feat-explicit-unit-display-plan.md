---
title: "feat: Respect explicit unit conversions in display"
type: feat
status: completed
date: 2026-03-05
brainstorm: docs/brainstorms/2026-03-05-explicit-unit-display-brainstorm.md
---

# feat: Respect explicit unit conversions in display

## Overview

When a user writes `200 kilowatts in megawatts`, the result should display as `0.2 MW`, not `200 kW`. Currently the interpreter correctly computes `Quantity{0.2, "megawatts"}`, but `NormalizeForDisplay()` in the formatter re-scales it to the "best" unit (value >= 1), overriding user intent.

## Problem Statement

The formatter's auto-scaling (`format/display/normalize.go:273`) treats all quantities the same. It doesn't know whether a unit was explicitly chosen by the user (`in megawatts`) or computed by arithmetic. Three explicit conversion surfaces are affected:

- `in` keyword: `200 kilowatts in megawatts`
- `as` keyword: `200 kilowatts as megawatts`
- `per` (convert_rate NL): `$0.10/hour per day`

## Proposed Solution

### New type: `ExplicitQuantity`

Add a thin wrapper type in `spec/types/` that embeds `*Quantity` and implements `types.Type`. This signals to all consumers (formatters, JSON output, TUI preview) that the unit was deliberately chosen.

**File: `spec/types/explicit.go`**

```go
// ExplicitQuantity wraps a Quantity whose unit was explicitly chosen
// by the user via `in`, `as`, or `per` conversion. Formatters should
// display the value in this unit without auto-scaling.
type ExplicitQuantity struct {
    *Quantity
}

func (e *ExplicitQuantity) String() string {
    return e.Quantity.String()
}

// Inner returns the underlying Quantity, stripping the explicit flag.
// Used by operators to unwrap before arithmetic.
func (e *ExplicitQuantity) Inner() *Quantity {
    return e.Quantity
}
```

### Unwrapping strategy

Go type switches won't match `*ExplicitQuantity` against `case *types.Quantity`. There are **15 sites** in `impl/interpreter/` and **2 in `format/`** that switch on `*types.Quantity`. Rather than adding `case *types.ExplicitQuantity` to all 15 interpreter sites, use a **centralized unwrap at the operator/function entry points**:

**Add to `impl/interpreter/` a helper:**

```go
// unwrapExplicit strips the ExplicitQuantity wrapper if present.
// This ensures arithmetic on explicit values produces plain Quantities,
// naturally dropping the explicit flag per the design decision.
func unwrapExplicit(t types.Type) types.Type {
    if eq, ok := t.(*types.ExplicitQuantity); ok {
        return eq.Inner()
    }
    return t
}
```

**Call `unwrapExplicit()` in two places:**
1. `evalBinaryOp()` — unwrap both left and right operands before the type switch
2. `evalNode()` return values flowing into `evalBinaryOp` don't need additional unwrapping since `evalBinaryOp` handles it

This is a 2-line change in `operators.go` rather than modifying 15 case sites.

### Conversion sites that produce `ExplicitQuantity`

Three sites in the interpreter need to wrap their results:

| Site | File | Current return | New return |
|------|------|---------------|------------|
| `evalUnitConversion` (qty path) | `unit_conversion_eval.go:57` | `*Quantity` | `*ExplicitQuantity` |
| `evalUnitConversion` (duration path) | `unit_conversion_eval.go:44` | `*Duration` | `*Duration` (unchanged) |
| `convertRateTimeUnit` | `rate_functions.go:90` | `*Rate` | `*Rate` (unchanged, rate display works differently) |

Only the quantity unit conversion at line 57 needs wrapping. Duration and rate conversions display correctly already.

For `per` (convert_rate), the result is a `*Rate` which has its own formatting path and doesn't go through `NormalizeForDisplay`, so no change needed there either.

### Formatter changes

**File: `format/display/formatter.go`**

Add a case before the `*types.Quantity` case in `Format()`:

```go
case *types.ExplicitQuantity:
    return f.FormatExplicitQuantity(v)
```

**New method `FormatExplicitQuantity`:** Format the value and unit directly without calling `NormalizeForDisplay()`. Use `formatWithSuffix()` for the number part and append the unit symbol.

For extreme values (e.g., `1 year in nanoseconds`), use scientific notation when the value exceeds a threshold (e.g., > 10 digits).

### Napkin interaction

`as napkin` should always re-scale. The napkin evaluator in `impl/interpreter/napkin_eval.go` produces a `*Quantity` with `IsNapkin: true`. If it receives an `ExplicitQuantity`, it should unwrap and produce a plain `Quantity` with `IsNapkin: true`, letting `NormalizeForDisplay` choose the best unit.

### Scientific notation for extreme explicit values

When `FormatExplicitQuantity` encounters a value with more than ~10 significant digits (or absolute value < 0.001 or > 1e12), format with scientific notation:

```go
// 1 year in nanoseconds → 3.156e+16 ns (not 31556952000000000 ns)
// 1 nanosecond in years → 3.17e-17 years (not 0.0000000000000000317 years)
```

## Acceptance Criteria

- [ ] `200 kilowatts in megawatts` displays as `0.2 MW`
- [ ] `500 kilowatts in megawatts` displays as `0.5 MW`
- [ ] `1000 kilowatts in megawatts` displays as `1 MW` (same as today)
- [ ] `1 meter in millimeters` displays as `1000 mm` (not `1 m`)
- [ ] `1 GB in MB` displays as `1000 MB` (not `1 GB`)
- [ ] Arithmetic on explicit quantities drops the flag: `(200 kW in MW) * 2` displays normally (auto-scaled)
- [ ] `as napkin` on explicit quantities re-scales normally
- [ ] Extreme conversions use scientific notation: `1 year in seconds` → reasonable display
- [ ] TUI preview pane, `cm eval --format text`, and `cm eval --format json` all respect explicit units
- [ ] No regressions: `task test` and `task quality` pass
- [ ] Existing auto-scaling for computed quantities is unchanged

## Implementation Steps

### Phase 1: Type and unwrap infrastructure

1. Create `spec/types/explicit.go` with `ExplicitQuantity` type
2. Add `unwrapExplicit()` helper in `impl/interpreter/`
3. Add unwrap calls in `evalBinaryOp()` for both operands
4. Write unit tests for `ExplicitQuantity` creation, `String()`, `Inner()`
5. Write unit tests for `unwrapExplicit()` (explicit → unwrapped, non-explicit → unchanged)

### Phase 2: Conversion sites

1. Modify `evalUnitConversion()` in `unit_conversion_eval.go:57` to wrap result in `ExplicitQuantity`
2. Update napkin evaluator to unwrap `ExplicitQuantity` before napkin processing
3. Write integration tests: `200 kilowatts in megawatts` produces `*ExplicitQuantity`
4. Write integration tests: arithmetic on explicit values produces plain `*Quantity`

### Phase 3: Formatter

1. Add `case *types.ExplicitQuantity` to `Format()` in `format/display/formatter.go`
2. Implement `FormatExplicitQuantity()` — format without `NormalizeForDisplay()`
3. Add scientific notation for extreme values
4. Add corresponding case to `format/json_formatter.go`
5. Write display tests: explicit quantities render with user's chosen unit
6. Write display tests: extreme values use scientific notation

### Phase 4: Golden tests and validation

1. Add golden test files for explicit unit conversion display
2. Run `task test` — fix any regressions
3. Run `task quality` — verify linting passes
4. Test TUI preview pane manually with `cm testdata/examples/datacenter-cost.cm`

## Implementation Note

The plan above proposed an `ExplicitQuantity` wrapper type, but implementation used a simpler approach: an `IsExplicit bool` field on `Quantity`, following the existing `IsNapkin` pattern. This reduced the change from ~15 type switch modifications to 3 files:

1. `spec/types/quantity.go` — added `IsExplicit bool` field
2. `impl/interpreter/unit_conversion_eval.go` — sets `converted.IsExplicit = true`
3. `format/display/formatter.go` — checks `IsExplicit` and skips `NormalizeForDisplay`

The flag naturally drops through arithmetic (new `*Quantity` values don't copy it), which is the desired behavior. No unwrapping, no new type, no changes to operators or NL functions.

## References

- Brainstorm: `docs/brainstorms/2026-03-05-explicit-unit-display-brainstorm.md`
- Auto-scaling logic: `format/display/normalize.go:273-336`
- Formatter dispatch: `format/display/formatter.go:39-60`
- Unit conversion eval: `impl/interpreter/unit_conversion_eval.go:13-63`
- Quantity type: `spec/types/quantity.go`
