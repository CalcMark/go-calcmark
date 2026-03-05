---
title: Smart Rounding for Explicit Conversions + `as precise` Modifier
type: feat
status: completed
date: 2026-03-05
---

# Smart Rounding for Explicit Conversions + `as precise` Modifier

## Overview

Explicit `as`/`in` conversions currently bypass `roundForDisplay()`, producing unnecessarily precise results (e.g., `0.000277777777777778 hour`). This feature applies the existing magnitude-based rounding thresholds to explicit conversions and adds `as precise` as an escape hatch for full precision.

## Problem Statement

`formatExplicitQuantity()` in `format/display/formatter.go:100` skips `NormalizeForDisplay()` (and thus `roundForDisplay()`), going directly to `formatWithSuffix()`. This is correct for preserving the user's chosen unit, but wrong for precision — raw float output is not human-readable:

- `1 second as hour` -> `0.000277777777777778 hour` (should be `0.0003 hour`)
- `(1 hour / 23) as millisecond` -> `156521.7391304472 millisecond` (should be `156,522 millisecond`)

The auto-normalized path already has sensible thresholds in `roundForDisplay()` (`normalize.go:431-451`):

| Value Range | Decimals | Example |
|-------------|----------|---------|
| >= 100      | 0        | 156,522 |
| 10-99       | 1        | 22.3    |
| 1-9         | 2        | 3.14    |
| < 1         | 4        | 0.0003  |

## Proposed Solution

Two changes:

### 1. Apply `roundForDisplay()` in `formatExplicitQuantity()`

Call `roundForDisplay()` on the value before passing to `formatWithSuffix()`, unless `IsPrecise` is set. This is a ~3-line change in `format/display/formatter.go:122-123`.

### 2. Add `as precise` modifier

Follows the exact `as napkin` pattern through the stack:

| Layer | `as napkin` (existing) | `as precise` (new) |
|-------|----------------------|-------------------|
| Lexer | `NAPKIN` token (`token.go:72`) | `PRECISE` token |
| Reserved words | `"napkin": NAPKIN` (`lexer.go:38`) | `"precise": PRECISE` |
| AST node | `NapkinConversion` (`nodes.go:82`) | `PreciseConversion` |
| Parser | `parseAdditive()` `:358` + `parseUnary()` `:727` | Same locations, add `PRECISE` branch |
| Interpreter | `evalNapkinConversion()` (`napkin_eval.go:22`) | `evalPreciseConversion()` — evaluate, set `IsPrecise=true` |
| Quantity flag | `IsNapkin bool` (`quantity.go:15`) | `IsPrecise bool` |
| Formatter | `if q.IsNapkin` prefix `~` (`formatter.go:92`) | `if q.IsPrecise` skip `roundForDisplay()` |
| Feature registry | `"as napkin"` entry (`registry.go:557`) | `"as precise"` entry |

## Technical Considerations

- **`roundForDisplay()` is package-internal** (lowercase) — already callable from `formatExplicitQuantity()` since both are in `format/display/`. No export needed.
- **Scientific notation path untouched** — Values < 0.001 or >= 1e12 already use `%g` formatting (`formatter.go:115`). `roundForDisplay()` only applies to the normal-range path at line 123.
- **`as precise` on non-explicit quantities** — Works on any expression, just like `as napkin`. Sets `IsPrecise=true`, formatter skips rounding. Simple and consistent.
- **`IsNapkin` + `IsPrecise` conflict** — `as napkin` takes precedence (earlier check in `FormatQuantity()`). Edge case not worth guarding against.
- **No currency/duration impact** — Currency formatting uses `FormatCurrency()`, durations use `FormatDuration()`. Neither calls `formatExplicitQuantity()`.

## Acceptance Criteria

### Smart rounding for explicit conversions

- [x] `1 second as hour` — duration path unchanged (separate concern)
- [x] `10 meters as feet` displays `32.8 feet` (10-99 range, 1dp)
- [x] `500 grams as pound` displays `1.1 pound` (1-9 range, 2dp, trailing zero trimmed)
- [x] Scientific notation still used for extreme values (e.g., `8.88e-16 PB`)
- [x] Auto-normalized quantities unchanged (no regression)
- [x] Napkin formatting unchanged (no regression)
- [x] Explicit conversions show comma-separated numbers, not K/M/B/T suffixes

### `as precise` modifier

- [x] `10 meters as feet as precise` shows full precision `32.808399 feet`
- [x] `as precise` works on plain expressions: `3.14159265358979 as precise` shows full decimal
- [x] `as precise` parses correctly at both expression levels (like `as napkin`)
- [x] `PRECISE` is a reserved keyword in lexer
- [x] `PreciseConversion` AST node exists with `Expression` field
- [x] `IsPrecise` flag on `Quantity` struct
- [x] Feature registered in `spec/features/registry.go`
- [x] Golden test files in `testdata/` cover both features

## Success Metrics

- All existing golden tests pass (no regression)
- New golden tests demonstrate rounded explicit conversion output
- `as precise` golden tests show full-precision escape hatch

## Dependencies & Risks

- **Low risk**: `roundForDisplay()` is battle-tested on the auto-normalized path
- **No breaking changes**: Output changes are cosmetic improvements — previously raw floats become rounded
- **Reserved word addition**: `precise` becomes a reserved keyword. Unlikely to conflict since it's not a unit name or common variable

## References & Research

### Internal References

- `format/display/formatter.go:100-124` — `formatExplicitQuantity()`, target for rounding
- `format/display/normalize.go:431-451` — `roundForDisplay()` thresholds
- `spec/types/quantity.go:12-17` — `Quantity` struct, needs `IsPrecise` field
- `spec/lexer/token.go:72` — `NAPKIN` token definition (pattern to follow)
- `spec/lexer/lexer.go:38` — `NAPKIN` reserved word (pattern to follow)
- `spec/ast/nodes.go:82-93` — `NapkinConversion` AST node (pattern to follow)
- `spec/parser/rdparser.go:358,727` — `as napkin` parsing (two locations)
- `impl/interpreter/napkin_eval.go:22` — `evalNapkinConversion()` (pattern to follow)
- `impl/interpreter/unit_conversion_eval.go:63` — sets `IsExplicit = true`
- `spec/features/registry.go:557-563` — `as napkin` feature entry (pattern to follow)
- `docs/brainstorms/2026-03-05-display-precision-brainstorm.md` — design decisions
