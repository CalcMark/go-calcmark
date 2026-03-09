---
title: "Percentage Widening on Addition and Subtraction"
type: feat
date: 2026-03-09
---

# Percentage Widening on Addition and Subtraction

## What We're Building

A new `Percentage` type that preserves percentage identity through variables and applies natural widening on `+` and `-`:

```
senior_base_salary = $145000        → $145K
senior_burden_rate = 32%            → 32%
senior_fully_loaded = senior_base_salary + senior_burden_rate → $191.4K
```

When a Percentage appears on the right side of `+` or `-`, it widens:
- `value + pct` = `value * (1 + pct.decimal)` — "increase by"
- `value - pct` = `value * (1 - pct.decimal)` — "decrease by"

This follows the same pattern as rate widening on `*` and `/`.

## Why This Approach

**New Percentage type** (not just a flag on Number):
- Percentages are semantically different from bare decimals — `32%` means "a proportion" while `0.32` means "a quantity"
- Variables must preserve percentage identity: `rate = 32%` then `salary + rate` must widen
- Display should show `32%`, not `0.32`
- Follows the precedent of Rate being a distinct type that widens on specific operators

**Breaking change accepted:**
- `100 + 20%` currently gives `100.2` — changes to `120`
- `100 - 20%` currently gives `99.8` — changes to `80`
- The old behavior is unintuitive; nobody writes `+ 20%` meaning `+ 0.2`

## Key Decisions

1. **Percentage is a new type** — not a flag on Number. It has its own display format (`32%`) and widening rules on `+`/`-`.

2. **Widening applies to both `+` and `-`** — symmetric behavior. `value + pct` increases, `value - pct` decreases.

3. **Right-side only** — `salary + 32%` widens, `32% + salary` is an error. Matches rate widening precedent (rate on right of `*` widens, rate on left preserves rate semantics).

4. **Variables preserve percentage identity** — `rate = 32%` stores a Percentage, not a Number. `salary + rate` widens.

5. **Display shows percentage** — `32%` renders as `32%` in all output formats, not `0.32`.

6. **Breaking change** — existing `number + percent_literal` behavior changes. Accepted trade-off.

7. **Multiplication uses decimal value** — `100 * 20%` = `20` (uses `0.2`). `Percentage * Number` and `Number * Percentage` both use the decimal value. Widening is `+`/`-` only.

## Type Interaction Matrix

### Addition / Subtraction (widening)

| Left | Right (Percentage) | `+` Result | `-` Result |
|------|-------------------|------------|------------|
| Number | Percentage | `left * (1 + pct)` → Number | `left * (1 - pct)` → Number |
| Currency | Percentage | `left * (1 + pct)` → Currency | `left * (1 - pct)` → Currency |
| Quantity | Percentage | `left * (1 + pct)` → Quantity | `left * (1 - pct)` → Quantity |
| Percentage | Percentage | `left + right` (decimal add) → Percentage | `left - right` → Percentage |
| Percentage | Number/Currency/Quantity | Error | Error |

### Multiplication / Division (no widening, use decimal value)

| Left | Right | `*` Result | `/` Result |
|------|-------|------------|------------|
| Number | Percentage | Number (`left * pct.decimal`) | Number |
| Percentage | Number | Number (`pct.decimal * right`) | Number |
| Currency | Percentage | Currency (`left * pct.decimal`) | Currency |
| Percentage | Percentage | Percentage (`left.decimal * right.decimal`) | Number |

## Implementation Sketch (Spec Layer)

- New `spec/types/percentage.go` — wraps `decimal.Decimal`, implements `Type` interface
- Lexer already tokenizes `NUMBER_PERCENT` — parser creates Percentage instead of dividing by 100 and storing as Number
- `literal_eval.go` returns `types.Percentage` instead of `types.Number` for `%` suffix

## Implementation Sketch (Impl Layer)

- `operators.go` — add percentage widening block before type dispatch (mirror rate widening pattern):
  ```
  if operator == "+" || operator == "-" {
      if rightPct, ok := right.(*types.Percentage); ok {
          // Widen: value +/- pct → value * (1 +/- pct.Decimal())
      }
  }
  ```
- Formatters — add Percentage case to all output formats (`32%`)

## Analogous Pattern: Rate Widening

The existing rate widening in `operators.go:53-68` is the exact precedent:
- Rate on right side of `*` or `/` → extract Amount, drop time denominator
- Percentage on right side of `+` or `-` → multiply by `(1 +/- decimal)`, preserve left type

## Codebase Audit: Files Requiring Updates

Thorough review of all tests, golden files, and documentation. No golden test expectations are silently wrong under current semantics, but several files have inconsistencies or will need updates.

### Tests with breaking expectations

| File | Line | Current | After widening |
|------|------|---------|----------------|
| `impl/interpreter/percentage_test.go` | 26 | `100 + 20%` → `100.2` | → `120` |
| `impl/interpreter/percentage_test.go` | 27 | `100 - 20%` → `99.8` | → `80` |
| `impl/interpreter/percentage_test.go` | 36 | `1k + 10%` → `1000.1` | → `1100` |
| `impl/interpreter/percentage_test.go` | 18-21 | `20%` → `0.2` (display) | → `20%` |

The "For now" comment at line 24-25 acknowledges the behavior is temporary.

### Examples using raw decimals instead of `%` literals

These files use `0.18`, `0.32`, etc. for values that are conceptually percentages. Once the Percentage type exists with proper display, these should be rewritten to use `%` literals for clarity:

| File | Lines | Current | Natural rewrite |
|------|-------|---------|-----------------|
| `testdata/examples/household-budget.cm` | 16-18 | `federal_rate = 0.18` | `federal_rate = 18%` |
| `testdata/examples/household-budget.cm` | 17 | `state_rate = 0.05` | `state_rate = 5%` |
| `testdata/examples/household-budget.cm` | 18 | `fica_rate = 0.0765` | `fica_rate = 7.65%` |
| `testdata/examples/job-offer.cm` | 12 | `annual_bonus_pct_a = 0.15` | `annual_bonus_pct_a = 15%` |
| `testdata/examples/job-offer.cm` | 31 | `annual_bonus_pct_b = 0.10` | `annual_bonus_pct_b = 10%` |
| `testdata/examples/job-offer.cm` | 37 | `expected_appreciation_b = 0.50` | `expected_appreciation_b = 50%` |
| `testdata/examples/job-offer.cm` | 50-52 | `federal_rate = 0.32` | `federal_rate = 32%` |
| `testdata/examples/job-offer.cm` | 94 | `startup_risk_discount = 0.40` | `startup_risk_discount = 40%` |

**Verified safe:** `total_tax_rate = federal_rate + state_rate + fica_rate` — Percentage + Percentage does decimal add → Percentage. Then `annual_comp * (1 - total_tax_rate)` widens correctly.

### Examples using `* (1 + pct)` workaround (can simplify post-feature)

| File | Line | Current | Simplified |
|------|------|---------|------------|
| `testdata/examples/markdown_financial.cm` | 9 | `license_revenue * (1 + license_growth)` | `license_revenue + license_growth` |
| `testdata/examples/markdown_financial.cm` | 15 | `consulting_revenue * (1 + consulting_growth)` | `consulting_revenue + consulting_growth` |
| `testdata/examples/markdown_financial.cm` | 21 | `maintenance_revenue * (1 + maintenance_growth)` | `maintenance_revenue + maintenance_growth` |
| `testdata/examples/job-offer.cm` | 58 | `annual_comp_a * (1 - total_tax_rate)` | `annual_comp_a - total_tax_rate` |
| `testdata/examples/household-budget.cm` | 21 | `total_gross * total_tax_rate` | stays the same (multiplication, not widening) |

### Documentation claiming "natural" but showing workarounds

| File | Line | Issue |
|------|------|-------|
| `site/content/docs/user-guide.md` | 925 | Says "Percentages work naturally" but shows `price * (1 - discount)` |
| `site/content/docs/language-reference.md` | 420 | Shows `50% → 0.5` — display will change to `50%` |

### No issues found (confirmed safe)

- All `X% of Y` expressions — unaffected (separate AST node, not arithmetic)
- All `compound()`/`depreciate()` calls — percentage passed as function argument, not in `+`/`-`
- All `with X% buffer` capacity expressions — handled by dedicated function
- All `value * pct` multiplication — uses decimal value (no widening on `*`)
- Golden test files in `testdata/eval/` — none use `value + pct` pattern

## Open Questions

None — all key decisions resolved during brainstorming.

## References

- Rate widening: `impl/interpreter/operators.go:53-68`
- Percentage parsing: `spec/document/literal_eval.go:222-228`
- Percentage tests: `impl/interpreter/percentage_test.go`
- Type definitions: `spec/types/`
