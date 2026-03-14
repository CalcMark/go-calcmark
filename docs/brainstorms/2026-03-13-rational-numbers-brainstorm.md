# Brainstorm: Rational Numbers (Fractions) in CalcMark

**Date:** 2026-03-13
**Status:** Brainstorm complete

## What We're Building

Native rational number support in CalcMark so that `1/3` is stored and displayed as the exact fraction `1/3`, not the decimal approximation `0.333333333333333`.

### Requirements

- **Fraction literals**: `1/3`, `2/3`, `1/4`, `7/8` — integer/integer with no spaces
- **Fraction + unit**: `1/3 cup`, `3/4 inch`, `5/8 in` — fraction followed by a unit (space required before unit)
- **Mixed number input**: `11 3/8 inch` — integer followed by fraction implies addition (common for measurements, bolt sizes)
- **Display as fractions**: results render as `1/3`, not `0.333...`
- **Mixed numbers**: improper fractions display as mixed numbers (e.g. `7/3` → `2 1/3`)
- **Simplify to lowest terms**: `2/4` → `1/2`, `6/9` → `2/3`
- **Threshold-based fallback**: fractions with large denominators (e.g. >1000) fall back to decimal display
- **Whitespace disambiguation**: `1/3` (no spaces) = fraction, `1 / 3` (spaces) = division

### Non-goals

- Variable-based fractions (`a/b` is always division)
- Fraction display format flags (future work)
- Unicode fraction characters (e.g. ½, ⅓)
- `'` and `"` as foot/inch unit aliases (lexer ambiguity; word aliases `ft`/`in` already work)

## Why This Approach

### Approach chosen: Lexer-level fraction token

The lexer recognizes `integer/integer` (no whitespace) as a single `FRACTION` token. This mirrors how `QUANTITY` tokens already work (number+unit scanned contiguously). A new `FractionLiteral` AST node and `types.Fraction` type wrap Go's `math/big.Rat` for exact arithmetic.

**Why not parser-level rewrite?** Tokens don't carry whitespace metadata today. Adding it would be invasive and the parser shouldn't need to reason about spacing.

**Why not runtime promotion?** Breaks the "literal integers only" rule — `a / b` would unexpectedly produce fractions. Violates user expectations about division.

### Why `math/big.Rat`?

Go's standard library provides exact rational arithmetic. It integrates cleanly:
- Convert to `decimal.Decimal` when mixing with non-fraction types
- Convert from `decimal.Decimal` when the result is a clean fraction
- No external dependency needed

## Key Decisions

1. **Lexer token**: `FRACTION` token scanned as `integer/integer` (no spaces between)
2. **AST node**: `FractionLiteral` with numerator and denominator fields
3. **Runtime type**: `types.Fraction` wrapping `math/big.Rat`
4. **Display**: Always simplified to lowest terms. Improper fractions shown as mixed numbers (e.g. `7/3` → `2 1/3`)
5. **Threshold**: Denominators > 1000 fall back to decimal display
6. **Disambiguation**: Whitespace is the signal — `1/3` is fraction, `1 / 3` is division
7. **Mixed arithmetic**: Fraction op Fraction → exact rational. Fraction op Number → convert Number to rational, compute, stay fraction when result has denominator ≤ threshold; otherwise fall back to decimal. Fraction is the "sticky" type. Once a variable holds a Fraction (e.g. `a = 1/3`), all subsequent arithmetic with `a` follows fraction rules — fractions are a _literal_ syntax but a _runtime_ type.
8. **Units**: `1/3 cup` — space required between fraction and unit. `FRACTION` token followed by unit lookahead (same pattern as `NUMBER` → `QUANTITY`).
9. **Mixed number input**: `11 3/8` — parser sees `NUMBER` immediately followed by `FRACTION` and creates a `MixedNumberLiteral` (whole + fraction). Works with units: `11 3/8 inch`. This is syntactic sugar for `11 + 3/8`.
10. **Denominator zero**: `1/0` passes the lexer as a valid `FRACTION` token, caught at semantic analysis as division-by-zero (consistent with existing `DiagDivisionByZero`).
11. **Negative fractions**: `-1/3` is `MINUS` + `FRACTION` — unary negation applied at parser level, no special lexer handling.
12. **Rate interaction**: `100/3` followed by `/s` — lexer produces `FRACTION(100,3)`, then rate parsing in the parser handles the `/s` separately. Edge case but not a conflict.
13. **`as napkin` on fractions**: Round to the nearest common fraction (halves, thirds, quarters, fifths, sixths, eighths, tenths, twelfths, sixteenths). If no common fraction is close, round to nearest integer. Prefixed with `~`.
14. **`as precise` on fractions**: No-op — fractions are already exact.
15. **`number()` function**: Explicit escape hatch from fraction to decimal. `number(1/3)` → `0.333333333333333`.

## Architecture Impact

### Layers affected (spec → impl, dependency flows downward)

| Layer | Change |
|-------|--------|
| `spec/lexer` | New `FRACTION` token type, scanning logic in `readNumber()` |
| `spec/parser` | Handle `FRACTION` token → `FractionLiteral` node |
| `spec/ast` | New `FractionLiteral` node type |
| `spec/types` | New `Fraction` type wrapping `math/big.Rat` |
| `spec/semantic` | Type checking for fraction operations |
| `impl/interpreter` | Evaluate `FractionLiteral`, fraction arithmetic, mixed-type operations, napkin rounding for fractions |
| `impl/interpreter` | `number()` function: fraction → decimal conversion |
| `format/display` | Fraction display: simplified, mixed numbers, threshold fallback, `~` prefix for napkin |
| `format/json` | Fraction JSON output: numerator, denominator, IsApproximate flag |

### Complexity management

The lexer and parser are already complex. New fraction scanning and mixed-number parsing must not make this worse.

- **Extract pure functions**: Fraction scanning logic (`isFractionLiteral(input) → numerator, denominator, ok`) should be a standalone, heavily-tested pure function — not inline in the lexer's main loop.
- **Test desired outcomes, not implementation**: Golden tests should validate that `1/3` produces the right result and `1 / 3` produces a different right result — not test internal token sequences.
- **Regression coverage**: Before any implementation, write tests for every existing feature that could break (rates, division, quantities with `/` in them) to lock in current behavior.

### Interaction with existing features

- **Rates**: `100 MB/s` already handled before fraction scanning — no conflict (rates use `QUANTITY/unit` pattern, not `integer/integer`)
- **Division**: `1 / 3` (with spaces) remains division → `Number(0.333...)`. Only `1/3` (no spaces) becomes a fraction.
- **Percentages**: `1/3` as `33.333...%` — conversion from Fraction to Percentage should work via `big.Rat` → `decimal.Decimal`
- **Unit conversion**: `1/3 cup to tablespoons` — convert fraction value through unit system
- **`as napkin`**: Rounds fractions to the nearest common fraction (halves, thirds, quarters, etc.). E.g. `11/12 cup as napkin` → `~1 cup`. Uses the same `~` prefix. Purely presentational — the interpreter produces the approximated fraction, the formatter renders it.
- **`as precise`**: No-op on fractions (already exact). Use `number(1/3)` to explicitly convert to decimal `0.333...`.
- **`number()` function**: Converts a fraction to its decimal equivalent. `number(1/3)` → `0.333333333333333`.

## Resolved Questions

1. **Fraction + unit spacing**: Space required before unit. `1/3 cup` = valid. `1/3cup` = not valid. Keeps lexer simple.
2. **Negative fractions**: `-1/3` is `MINUS` + `FRACTION`. Unary negation applied at parser level, no special lexer token.
3. **Denominator of zero**: `1/0` passes lexer, caught at semantic analysis (consistent with existing `DiagDivisionByZero`).
4. **`'`/`"` unit aliases**: Deferred. Word aliases `ft`/`in`/`inch`/`foot` already cover the use case without lexer ambiguity.

## Open Questions

1. **Threshold value**: 1000 was suggested as the display threshold for falling back to decimal. Is this the right number? Should it be configurable?

## Use Cases (Motivating Examples)

```calcmark
# The killer demo: exact arithmetic
a = 1/3 + 1/3 + 1/3            // → 1 (not 0.999...)

# Cooking
flour = 2/3 cup
sugar = 1/4 cup
total = flour + sugar           // → 11/12 cup

# Hardware / Bolt sizes
bolt = 5/8 inch
pipe = 11 3/8 inch              // Mixed number input

# Arithmetic
b = 1/6
c = a + b                      // → 1/2 (exact, simplified)  [a is still 1/3 from above]
d = c * 3                      // → 1 1/2 (fraction propagates through variables)

# Division still works
x = 10 / 3                     // → 3.333... (spaces = division)
y = 1/3                        // → 1/3 (no spaces = fraction)

# Napkin and precision
total as napkin                 // → ~1 1/2 (rounds to nearest common fraction)
flour as napkin                 // → ~2/3 cup (already a common fraction)
11/12 cup as napkin             // → ~1 cup
number(1/3)                    // → 0.333333333333333
1/3 as precise                 // → 1/3 (no-op, already exact)
```
