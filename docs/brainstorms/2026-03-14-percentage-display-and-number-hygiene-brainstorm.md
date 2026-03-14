# Percentage Display and number() Hygiene

**Date:** 2026-03-14
**Status:** Draft

## Three Related Ideas

### 1. `as percent` / `as percentage` display modifier

Display sugar like `as napkin`. Not a function, not a new type — just a display hint.

```
gross_margin = number(gross_profit) / number(total_rev) as percent
```

Displays `62%` instead of `0.62`. The value stays `0.62` internally.

- `as percent` and `as percentage` are synonyms
- Sets `IsPercent: true` flag on the result (same pattern as `IsNapkin`, `IsPrecise`)
- Formatter renders `value * 100` + `%` suffix
- Pipeline stage: during eval (stage 3), same as `as napkin`
- No new type needed — this is purely porcelain

### 2. INFO diagnostic for redundant `number()` wrapping

The semantic analyzer detects `number(x)` where `x` is already a plain `Number` type and emits a hint:

```
capacity = number(hc) * net_hrs        // capacity is already a Number
consumed = est / number(capacity)       // ← INFO: number() unnecessary, capacity is already a plain number
```

Requires:
- New diagnostic severity: `INFO` (alongside `ERROR`, `WARNING`)
- Type-tracking in the semantic analyzer to know what type a variable holds
- Only emit when the inner expression is provably a Number (don't guess across blocks)

### 3. INFO diagnostic for `* 100` percentage pattern

When the analyzer sees `ratio * 100` where `ratio` is a division result between 0 and 1, suggest `as percent`:

```
margin = number(profit) / number(rev) * 100    // ← INFO: consider using `as percent` instead of * 100
```

This is harder to detect reliably — the `* 100` could be intentional (e.g., converting a rate to basis points). Only hint when:
- The value being multiplied is the result of a division
- The multiplier is exactly `100`
- The result is assigned to a variable with "pct", "percent", "ratio", or "margin" in the name (heuristic)

## Dependencies

- `as percent` must ship before the INFO diagnostics can suggest it
- The INFO severity level is shared infrastructure for both diagnostics
- Type-tracking in the semantic analyzer is prerequisite for the `number()` hygiene hint
