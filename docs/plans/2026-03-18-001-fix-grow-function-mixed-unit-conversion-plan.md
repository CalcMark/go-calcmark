---
title: "fix: grow() ignores unit conversion for mixed compatible quantities"
type: fix
status: completed
date: 2026-03-18
---

# fix: grow() ignores unit conversion for mixed compatible quantities

## Overview

`grow(50GB, 20TB, 6 months)` returns `170 GB` instead of the expected `122,930 GB`. The function strips units via `extractDecimalValue()`, treating `20 TB` as bare `20` rather than converting it to the amount's unit (GB) first. The same stripping means `6 months` is correctly treated as `6` periods, but this behavior is undocumented.

## Problem Statement

Two issues in `evalGrowFunc` (`impl/interpreter/growth_functions.go:231-268`):

### Root Cause 1: `extractDecimalValue` discards units without conversion

Both `amountVal` and `incrementVal` go through `extractDecimalValue()` which extracts `.Value` and discards the unit. So `50 GB` → `50`, `20 TB` → `20`. The math becomes `50 + (20 × 6) = 170`, wrapped with the first argument's unit → `170 GB`.

**Expected**: Convert `20 TB` to `20,480 GB` (first-unit-wins), then `50 + (20480 × 6) = 122,930 GB`.

The regular arithmetic path (`50 GB + 20 TB`) correctly handles this via `evalQuantityOperation` → `convertQuantity()`. The `grow()` function bypasses this entirely.

### Root Cause 2: Period unit silently ignored

`6 months` is a `*types.Duration{Value: 6, Unit: "months"}`. `extractPeriodsFromDuration` returns `6` and discards `"months"`. This is actually correct semantics for `grow()` (6 periods of linear addition), but the unit is silently dropped with no validation or documentation.

## Proposed Solution

Modify `evalGrowFunc` to convert the increment to the amount's unit when both are `*types.Quantity` with compatible but different units, using the existing `convertQuantity()` infrastructure.

### Implementation

In `impl/interpreter/growth_functions.go`, after evaluating both `amountVal` and `incrementVal`:

1. If both are `*types.Quantity` and units differ → call `convertQuantity(incrementQty, amountQty.Unit)` before extracting the decimal
2. If both are `*types.Currency` and symbols differ → return an error (mixing `$` and `€` is not meaningful)
3. If types are mismatched (e.g., Quantity + Currency) → current behavior is fine (extract raw values)

The `convertQuantity` function already handles compatibility checking and returns a clear error for incompatible units.

### Files to Modify

1. **`impl/interpreter/growth_functions.go`** — `evalGrowFunc`: Add unit conversion between amount and increment when both are `*types.Quantity`
2. **`impl/interpreter/growth_functions_test.go`** — Add test cases for mixed-unit `grow()` calls
3. **`testdata/eval/success/features/growth_functions.cm`** — Add golden test cases
4. **`testdata/spec/valid/features/growth_functions.cm`** — Add valid spec examples

### Also affected: `evalDepreciateFunc`

`depreciate()` uses the same `extractDecimalValue` pattern for its value parameter. If the salvage value has different units than the initial value (e.g., `depreciate(1TB, 10%, 5, 100GB)`), the same bug applies. Should be fixed in the same pass.

## Acceptance Criteria

- [ ] `grow(50GB, 20TB, 6)` returns `122,930 GB` (converts TB→GB before multiplication)
- [ ] `grow(50GB, 20TB, 6 months)` returns `122,930 GB` (duration period count extracted correctly)
- [ ] `grow(50GB, 500GB, 3)` returns `1,550 GB` (same-unit still works)
- [ ] `grow(100, 20, 5)` returns `200` (plain numbers unchanged)
- [ ] `grow($500, $100, 36)` returns `$4,100.00` (currency unchanged)
- [ ] `grow(50GB, 20 incompatible_unit, 6)` returns a clear error about incompatible units
- [ ] NL syntax `grow 50GB by 20TB over 6 months` produces the same result as functional form
- [ ] `depreciate()` also handles mixed compatible units correctly
- [ ] All existing tests continue to pass

## Technical Considerations

- **First-unit-wins rule**: Matches existing arithmetic behavior (`50 GB + 20 TB = 20,530 GB`)
- **`convertQuantity` reuse**: Already handles unit compatibility checking and conversion via `units.Convert`
- **Binary data units**: In this codebase, `1 TB = 1024 GB` (binary/IEC convention per `spec/units/conversion.go`)
- **NL/functional parity**: Per learnings doc, "every fix is two fixes" — must verify both parser paths produce identical results
- **`IsExplicit`/`IsNapkin` flags**: Per learnings, when constructing Quantity values, propagate metadata flags from the original. The existing `wrapResult` already handles this correctly by copying the unit from the original amount.

## Sources

- Implementation: `impl/interpreter/growth_functions.go:231-268`
- Unit conversion: `impl/interpreter/unit_conversion.go:13-44` (`evalQuantityOperation`, `convertQuantity`)
- Tests: `impl/interpreter/growth_functions_test.go:263-289`
- Spec: `spec/features/registry.go:668-678`
- Param types: `spec/types/param_types.go:151-158`
- Learning: `docs/solutions/logic-errors/compound-bare-frequency-modifier-silently-ignored.md`
- Learning: `docs/solutions/integration-issues/nl-functional-syntax-parity-and-doc-staleness.md`
