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

| Syntax | Form | What it does |
|--------|------|-------------|
| `X as a % of Y` | NL binary | Compute ratio and convert: `(X / Y) * 100` → Percentage |
| `percent(X, Y)` | function binary | Same — functional equivalent of `X as a % of Y` |
| `X as percentage` | NL unary | Convert existing ratio: `X * 100` → Percentage |
| `percent(X)` | function unary | Same — functional equivalent of `X as percentage` |

`percent()` is overloaded by arity — one arg converts, two args computes the ratio. This keeps the pattern where every NL syntax has a function equivalent:

```
gross_margin = percent(gross_profit, total_revenue)    // binary: compute + convert
gross_margin = gross_profit as a % of total_revenue    // NL equivalent

ratio = number(profit) / number(rev)
margin = percent(ratio)           // unary: convert 0.62 → 62%
margin = ratio as percentage      // NL equivalent
```

The binary form also solves the `number()` wrapping problem — `percent($500, $1000)` handles the type stripping internally, so users don't need to write `number($500) / number($1000) as percentage`.

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
