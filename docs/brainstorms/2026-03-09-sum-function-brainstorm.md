# Brainstorm: sum() Function

**Date:** 2026-03-09
**Status:** Ready for planning

## What We're Building

A variadic `sum()` function that adds 2+ values of compatible types, with automatic unit conversion within the same dimension and auto-scaling of the result.

### Supported types
- **Number**: `sum(1, 2, 3)` → `6`
- **Currency**: `sum($574.2K, $650K, $368.64K, $475.2K)` → `$2.07M`
- **Quantity** (same dimension): `sum(1 kg, 500 g)` → `1.5 kg`
- **Duration**: `sum(1 hour, 30 minutes)` → `1.5 hours`

### Syntax
- **Traditional**: `sum(a, b, c)`
- **NL**: `sum of a, b, c` (mirrors `average of` pattern)

### Aliases
- `total` is a **search-only** alias (discoverable in help/autocomplete, not parseable)
- Only `sum` and `sum of` are parseable

## Why This Approach

The motivating use case is financial rollups like:

```
total_senior_cost = senior_consultants * senior_fully_loaded → $574.2K
total_mid_cost = mid_consultants * mid_fully_loaded → $650K
total_junior_cost = junior_consultants * junior_fully_loaded → $368.64K
total_mgmt_cost = mgmt_hc * mgmt_avg_fully_loaded → $475.2K

total_labor_cost = sum of total_senior_cost, total_mid_cost, total_junior_cost, total_mgmt_cost → $2.07M
```

This is cleaner than chaining `a + b + c + d` when you have many named line items.

## Key Decisions

1. **Minimum 2 arguments** — `sum(a)` is an error (summing one thing is meaningless).
2. **All compatible types from day one** — Number, Currency, Quantity, Duration. Not starting narrow.
3. **Auto-conversion within dimension** — `sum(1 kg, 500 g)` works. Uses existing `convertQuantity()` infrastructure for cross-system conversion (e.g., g + lbs) and `NormalizeForDisplay()` for auto-scaling.
4. **First argument determines output dimension** — `sum(1 g, 10 lbs)` converts lbs→grams, sums, then auto-scales (→ kg if large enough).
5. **NL syntax: `sum of`** — Lexer fuses `sum` + `of` into `FUNC_SUM_OF` token, same pattern as `average of`.
6. **`total` is not parseable** — Search-only alias for discovery.
7. **Category: Math** — Same category as `avg`.

## Architecture Notes

All infrastructure exists:
- **`convertQuantity()`** in `impl/interpreter/unit_conversion.go` handles cross-system unit conversion (g↔lbs, m↔ft, etc.)
- **`NormalizeForDisplay()`** in `format/display/normalize.go` handles auto-scaling to human-readable units
- **`avg` function** in `impl/interpreter/functions.go` is the structural template — `sum` is identical minus the division
- **Lexer fusion** for `sum of` follows the `average of` → `FUNC_AVERAGE_OF` pattern exactly

### Four layers to touch (enforced by consistency tests):
1. `spec/semantic/function_types.go` — `FunctionSpecs["sum"]` with `Variadic: true`
2. `impl/interpreter/functions.go` — `BuiltinFunctions` entry + `evalSumFunc`
3. `spec/features/registry.go` — Feature entry with NL example
4. Lexer/parser — `FUNC_SUM_OF` token, fusion rule, NL parse dispatch

### Key difference from avg:
`avg` only accepts `Number` and `Currency` (via `extractNumbers()`). `sum` needs a broader type-dispatch that handles `Quantity` and `Duration` too, using `convertQuantity()` for unit harmonization.

## Open Questions

None — all questions resolved during brainstorming.
