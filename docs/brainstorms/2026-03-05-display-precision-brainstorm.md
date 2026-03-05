# Display Precision for Explicit Conversions

**Date:** 2026-03-05
**Status:** Brainstorm complete

## What We're Building

Two changes to how Calcmark displays numbers:

1. **Smart rounding for explicit conversions** — Apply the existing `roundForDisplay()` magnitude-based thresholds to explicit `as`/`in` conversions, which currently show raw float precision.

2. **`as precise` modifier** — A new display modifier (opposite of `as napkin`) that shows full float precision for any expression. Works on any expression, not just conversions.

## Why This Approach

Explicit conversions (`as hour`, `in millisecond`) currently bypass `NormalizeForDisplay()` and its `roundForDisplay()` rounding. This produces unnecessarily precise results:

- `1 second as hour` → `0.000277777777777778 hour` (should be `0.0003 hour`)
- `(1 hour / 23) as millisecond` → `156521.7391304472 millisecond` (should be `156,522 millisecond`)

The auto-normalized path already has sensible magnitude-based thresholds:

| Value Range | Decimals | Example |
|-------------|----------|---------|
| >= 100      | 0        | 156,522 |
| 10-99       | 1        | 22.3    |
| 1-9         | 2        | 3.14    |
| < 1         | 4        | 0.0003  |

These thresholds work well and should apply to explicit conversions too. For the rare case where full precision is needed, `as precise` provides an escape hatch — symmetrical with the existing `as napkin`.

## Key Decisions

- **Reuse `roundForDisplay()` thresholds** for explicit conversions rather than inventing a new rounding scheme
- **`as precise` works on any expression**, not limited to conversions — mirrors how `as napkin` works
- **`as precise` sets `IsPrecise = true`** on the result, telling the formatter to skip all display rounding
- **Percentage display** (`20%` showing as `0.2`) tracked separately — different concern

## Scope

- Applies only to explicit conversions (`IsExplicit = true`) — auto-normalized quantities already round correctly
- Does not change currency, napkin, or duration formatting
- Does not introduce configurable precision (YAGNI) — magnitude thresholds + `as precise` escape hatch is sufficient

## Related

- `format/display/normalize.go` — `roundForDisplay()` and `NormalizeForDisplay()`
- `format/display/formatter.go` — `formatExplicitQuantity()` is where rounding needs to be applied
- `spec/types/quantity.go` — `Quantity` struct needs `IsPrecise` flag
- `impl/interpreter/napkin_eval.go` — pattern to follow for `as precise` implementation
- Percentage display issue tracked separately
