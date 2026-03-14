# Percentage Display and number() Hygiene

**Date:** 2026-03-14
**Status:** Draft

## Four Related Ideas

### 1. `as percent` / `as percentage` — type conversion (Number → Percentage)

A **type conversion**, not display sugar. Takes a Number (typically a ratio like `0.62`) and returns a `Percentage` type (`62%`). The `* 100` math is baked into the conversion — downstream calculations see `62`, not `0.62`.

```
ratio = number(gross_profit) / number(total_rev)
gross_margin = ratio as percentage
```

`gross_margin` is `Percentage(62%)`. If you later write `tax = gross_margin * something`, the value is `62`, not `0.62`.

- `as percent` and `as percentage` are synonyms
- `percent(0.62)` is the function form, `0.62 as percentage` is the NL form
- Returns `Percentage` type (already exists in the type system)
- Pipeline stage: during eval (stage 3) — same as other `as` conversions
- The `* 100` multiplication happens at conversion time, not display time

### 2. Relationship to existing `as a % of`

`X as a % of Y` is a **binary operation** that computes the ratio AND converts in one step:

```
gross_margin = gross_profit as a % of total_revenue
```

This already exists and produces `Percentage(62%)`. The two features are complementary:

| Syntax | Operands | What it does |
|--------|----------|-------------|
| `X as a % of Y` | binary | Compute ratio and convert: `(X / Y) * 100` → Percentage |
| `X as percentage` | unary | Convert existing ratio: `X * 100` → Percentage |
| `percent(X)` | unary function | Same as above, function syntax |

The unary form fills the gap when you already have a ratio from `number(X) / number(Y)` and want to convert it to a Percentage for downstream math.

### 3. INFO diagnostic for redundant `number()` wrapping

The semantic analyzer detects `number(x)` where `x` is already a plain `Number` type and emits an info diagnostic:

```
capacity = number(hc) * net_hrs        // capacity is already a Number
consumed = est / number(capacity)       // ← INFO: number() unnecessary, capacity is already a plain number
```

Requires:
- New diagnostic severity: `INFO` (alongside `ERROR`, `WARNING`)
- Type-tracking in the semantic analyzer to know what type a variable holds
- Only emit when the inner expression is provably a Number (don't guess across blocks)

### 4. INFO diagnostic for `* 100` percentage pattern

When the analyzer sees `ratio * 100` where `ratio` is a division result, suggest `as percentage`:

```
margin = number(profit) / number(rev) * 100    // ← INFO: consider `as percentage` instead of * 100
```

This is harder to detect reliably — the `* 100` could be intentional (e.g., converting a rate to basis points). Only hint when:
- The value being multiplied is the result of a division
- The multiplier is exactly `100`
- The result is assigned to a variable with "pct", "percent", "ratio", or "margin" in the name (heuristic)

## Dependencies

- `as percentage` / `percent()` (item 1) must ship before the `* 100` INFO diagnostic (item 4) can suggest it
- The INFO severity level is shared infrastructure for both diagnostics (items 3 and 4)
- Type-tracking in the semantic analyzer is prerequisite for the `number()` hygiene diagnostic (item 3)
- `as a % of` already exists — item 1 is a companion, not a replacement
