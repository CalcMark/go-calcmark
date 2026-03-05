---
title: "Explicit unit conversion results overridden by display auto-scaling"
date: 2026-03-05
module: format/display
severity: medium
tags:
  - unit-conversion
  - display-formatting
  - user-intent
  - IsExplicit
  - NormalizeForDisplay
symptoms:
  - "`200 kilowatts in megawatts` displays `200 kW` instead of `0.2 MW`"
  - "Explicit `in`/`as` conversion results are auto-scaled back to a different unit"
  - "User-requested target unit is silently replaced by NormalizeForDisplay"
root_cause: "FormatQuantity unconditionally called NormalizeForDisplay on all Quantity values, including those produced by explicit `in`/`as` unit conversion, overriding the user's chosen target unit"
---

# Explicit unit conversion results overridden by display auto-scaling

## Problem

When a user writes `200 kilowatts in megawatts`, the interpreter correctly computes `Quantity{0.2, "megawatts"}`. But the display formatter's `NormalizeForDisplay()` re-scales it back to `200 kW` — the unit the user was converting *from*.

The core issue: there was no way to distinguish "the interpreter picked this unit" from "the user explicitly asked for this unit via `in`/`as` conversion." Every `*types.Quantity` looked the same to the formatter.

## Root Cause

`NormalizeForDisplay()` in `format/display/normalize.go:273` auto-scales every quantity to the largest unit in the same family where the value is >= 1. For `0.2 megawatts`, kilowatts gives `200` (>= 1), so it "wins" over megawatts. This is desirable for computed results but silently overrides explicit user intent.

## Solution

Added `IsExplicit bool` field to the `Quantity` struct, following the existing `IsNapkin` pattern.

### Step 1 — Add field to Quantity (`spec/types/quantity.go`)

```go
type Quantity struct {
    Value      decimal.Decimal
    Unit       string
    IsNapkin   bool // True if this is a napkin estimate
    IsExplicit bool // True if unit was explicitly chosen via `in`/`as` conversion
}
```

### Step 2 — Set the flag in the interpreter (`impl/interpreter/unit_conversion_eval.go:62-63`)

```go
// Mark as explicit so formatters skip auto-scaling
converted.IsExplicit = true
```

Set only for the `*types.Quantity` path in `evalUnitConversion` — not currency, duration, or rate conversions.

### Step 3 — Check in the formatter (`format/display/formatter.go:79-81`)

```go
if q.IsExplicit {
    return f.formatExplicitQuantity(q)
}
```

`formatExplicitQuantity` skips `NormalizeForDisplay` and formats the value directly with the user's chosen unit. It includes:
- An `Inf`/`NaN` guard falling back to `decimal.String()` for values beyond float64 range
- Scientific notation (`%g`) for extreme values (< 0.001 or >= 1e12)
- Standard `formatWithSuffix` for normal values

### Step 4 — Expose in JSON output (`format/json_formatter.go`)

Added `IsExplicit bool` field to `JSONResult` so downstream consumers can detect explicit conversions.

## Key Behaviors

- **Flag drops on arithmetic**: `(200 kW in MW) * 2` produces `400 kW` (auto-scaled). Arithmetic creates new `*Quantity` without copying `IsExplicit`.
- **Napkin re-scales normally**: `200 kW in MW as napkin` → `~200 kW`. The napkin evaluator creates a fresh `Quantity` with `IsNapkin: true`, dropping `IsExplicit`.
- **Chained conversions work**: `(1 meter in feet) in meters` → `1 meters`. The second `evalUnitConversion` sets `IsExplicit` on the new result.

## Design Decision: Bool Field vs Wrapper Type

Two approaches were considered:

| Approach | Pros | Cons |
|----------|------|------|
| `ExplicitQuantity` wrapper type | Compile-time distinction | Required unwrapping at 15+ type-switch sites; every future type switch must handle both types |
| `IsExplicit bool` field | 3-file change; follows `IsNapkin` precedent; zero changes to operators/functions | Flag could theoretically be lost if a code path constructs `Quantity` without copying it |

The wrapper approach was implemented first but rejected — it touched operators, growth functions, capacity functions, percentage-of, napkin evaluator, rate evaluator, availability functions, and multiple formatter sites. The bool field approach reduced the change to 3 production files.

The "flag getting lost" downside is actually the desired behavior: arithmetic *should* drop the flag, and `IsNapkin` proves the pattern works.

## Prevention

### When to use bool flags vs wrapper types on Quantity

Use a **bool field** when:
- The distinction is binary and relevant only at specific pipeline stages (display, serialization)
- Most consumers (operators, functions) don't care about the distinction
- Many type-switch sites exist that would need updating for a wrapper type

Use a **wrapper type** when:
- The distinction changes valid operations (e.g., immutability)
- You want the compiler to force every consumer to handle both variants

### Warning signs in future changes

- Any change to `NormalizeForDisplay` that doesn't check `IsExplicit` is suspect
- New `Quantity{Value: ..., Unit: ...}` literals that don't propagate metadata fields
- Formatter code that branches on unit magnitude without an explicit-flag gate
- New convenience helpers that wrap Quantity creation without exposing `IsExplicit`

### Tests that guard against regression

- `format/display/explicit_quantity_test.go` — explicit formatting, scientific notation, flag precedence over `IsNapkin`
- `impl/interpreter/explicit_unit_test.go` — end-to-end: explicit display, arithmetic flag drop, chained conversion, napkin on explicit

## Related Documentation

- [Brainstorm](../brainstorms/../../../docs/brainstorms/2026-03-05-explicit-unit-display-brainstorm.md)
- [Plan](../plans/../../../docs/plans/2026-03-05-feat-explicit-unit-display-plan.md)
- [Locale formatting bypass in TUI](locale-formatting-bypass-in-tui.md) — all display paths must use the locale-aware formatter
- [Currency code output spacing](currency-code-output-spacing.md) — display formatting heuristics
- [JSON output cleanup plan](../../../docs/plans/2026-03-03-feat-json-output-cleanup-plan.md) — `populateResult()` type switch, `IsNapkin` → `is_approximate` mapping
