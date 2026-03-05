# Brainstorm: Respect Explicit Unit Conversions in Display

**Date:** 2026-03-05
**Status:** Draft

## What We're Building

When a user explicitly converts units via `in`, `as`, or `per` (convert_rate), the formatter should display the result in the user's chosen unit instead of auto-scaling to a "better" unit.

Currently `200 kilowatts in megawatts` displays as `200 kW` because `NormalizeForDisplay()` in `format/display/normalize.go` re-scales 0.2 MW back to 200 kW (preferring values >= 1). The conversion itself works correctly — the interpreter returns `Quantity{0.2, "megawatts"}` — but the formatter overrides it.

### Scope

Three explicit conversion surfaces:
- `in` keyword: `200 kilowatts in megawatts`
- `as` keyword: `200 kilowatts as megawatts`
- `per` (convert_rate NL): `$0.10/hour per day`

Does NOT cover:
- Currency conversions (already work correctly, no auto-scaling families)
- User config for auto-scaling behavior (YAGNI for now)

## Why This Approach

**ExplicitQuantity wrapper type** in `spec/types/` — a thin type that embeds `*Quantity` and signals to all formatters that the unit was deliberately chosen by the user. This is preferred over:

- **Flag on Quantity** — mutates a core type, risk of flag getting lost or misinterpreted
- **Formatter heuristics** — can't reliably distinguish computed vs explicit without intent signal

The wrapper approach gives clean separation: the interpreter marks intent, the formatter respects it, and arithmetic naturally strips it (returns plain Quantity).

## Key Decisions

1. **New type `ExplicitQuantity`** in `spec/types/` that embeds `*Quantity` and implements `types.Type`
2. **Three conversion sites** produce `ExplicitQuantity`: `in`, `as`, `per` (convert_rate)
3. **Formatter skips `NormalizeForDisplay()`** for `ExplicitQuantity` — displays value and unit as-is
4. **Arithmetic drops explicit flag** — any operation on an `ExplicitQuantity` produces a plain `Quantity` or other result type
5. **Napkin always re-scales** — `as napkin` ignores explicit intent and picks the most human-readable form
6. **Scientific notation for extreme values** — when an explicit conversion produces very large or very small numbers (e.g., years to nanoseconds), use scientific notation instead of raw digits
7. **Currencies unaffected** — no `ExplicitCurrency` needed; currency formatting already works correctly
8. **All output modes benefit** — TUI preview pane, `cm eval` text/json/html/md/cm all consume `types.Type`, so the fix is universal

## Open Questions

None — all key decisions resolved.
