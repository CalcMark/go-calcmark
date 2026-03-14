---
title: "fix: Example quality pass and currency/quantity division bug"
type: fix
status: active
date: 2026-03-14
---

# Example Quality Pass and Currency/Quantity Division Bug

## Overview

A thorough audit of `testdata/examples/` revealed two categories of issues: an interpreter bug where same-type division produces the wrong result type (`$500 / $1000` → `$0.50` instead of `0.5`), and widespread example quality issues where files don't use CalcMark's best idioms (fractions, `sum of`, NL functions, `% of`).

## Problem Statement

### Bug: Currency / Currency returns Currency instead of Number

`$500 / $1000` should yield `0.5` (dimensionless ratio), not `$0.50`. When dividing the same type by itself, the units cancel out. This is basic dimensional analysis:

- `$500 / $1000` → `0.5` (not `$0.50`)
- `10 meters / 5 meters` → `2` (not `2 meters`)
- `100 kg / 25 kg` → `4` (not `4 kg`)

Currently:
- Currency/Currency `/` returns Currency (`$500 / $1000` → `$0.50`)
- Currency/Currency `*` returns Currency (`$100 * $50` → `$5,000`) — dollars-squared is nonsensical
- Quantity/Quantity `/` errors with "unsupported quantity operation: /"
- Quantity/Quantity `*` errors with "unsupported quantity operation: *"
- Fraction/Fraction `/` with same unit preserves unit (`2/3 cup / 1/3 cup` → `2 cup` instead of `2`)

This causes downstream display bugs in examples: `profit_margin = operating_income / total_revenue * 100` shows `$41.01` instead of a number.

**Note on `* 100` percentage pattern:** Even after fixing division, `profit / revenue * 100` produces `Number(41.01)`, not `41.01%`. The examples should be rewritten to avoid `* 100` entirely — the idiomatic CalcMark way is to leave the ratio as-is (0.41) or use `number()` to strip currency before dividing. A future `as percentage` conversion is out of scope for this plan.

### Example quality: not using CalcMark's best features

Many examples were written before fractions, `sum of`, and NL syntax existed. They use patterns like:
- `total = a + b + c + d + e + f + g` instead of `sum of a, b, c, d, e, f, g`
- `savings = income * 0.20` instead of `20% of income`
- `rate = $300/month per day` producing `$6.666667/day` (ugly)
- `load_frac = 200 / 1000` instead of `1/5` (fraction)
- Manual `* (1 + rate)` instead of compound/growth NL functions

## Proposed Solution

### Phase 1: Fix same-type division

**Design principle:** Addition, subtraction, and multiplication between same types work and are kept. **Division** between same types should error — dividing dollars by dollars or kg by kg doesn't produce a meaningful result in the real world.

- `$100 + $50` → `$150` (works, keep — adding prices is natural)
- `$100 - $50` → `$50` (works, keep — subtracting prices is natural)
- `$100 * $50` → **error** (dollars × dollars = nonsensical)
- `$100 / $50` → **error** (dollars ÷ dollars = nonsensical)
- `$100 * 5` → `$500` (works, keep — scaling by a number is natural)
- `$100 / 5` → `$20` (works, keep — splitting by a number is natural)
- `10 kg * 5 kg` → error (already errors — correct)
- `10 kg / 5 kg` → error (already errors — needs better message with `number()` hint)
- `2/3 cup / 1/3 cup` → **error** (currently returns `2 cup` — wrong)
- `2/3 cup * 3` → `2 cup` (works, keep — scaling by a number)

**Simple rule:** Only a bare number (Number or dimensionless Fraction) can multiply or divide a typed value (Currency, Quantity, Duration, Fraction-with-unit). Same-type × and ÷ always errors. Same-type + and - works.

Verified: no internal functions (`compound`, `grow`, `depreciate`) depend on Currency × Currency — they all extract decimal values before arithmetic.

When users want a ratio, they should use `number()`: `number($500) / number($1000)` → `0.5`.

**Interpreter change:** Same-type division and multiplication should error with a helpful message.

| Expression | Current | Expected |
|-----------|---------|----------|
| `$500 / $1000` | `$0.50` (wrong) | error with `number()` hint |
| `$100 * $50` | `$5,000` (wrong) | error with `number()` hint |
| `10 dogs / 5 dogs` | error (correct) | error with better `number()` hint |
| `10 kg * 5 kg` | error (correct) | error with better `number()` hint |
| `2/3 cup / 1/3 cup` | `2 cup` (wrong) | error with `number()` hint |
| `2/3 cup * 1/3 cup` | untested | error with `number()` hint |
| `$100 / 5` | `$20.00` | `$20.00` (unchanged — Currency / Number) |
| `$100 * 5` | `$500.00` | `$500.00` (unchanged — Currency * Number) |
| `$100 + $50` | `$150.00` | `$150.00` (unchanged — same-type +/-) |
| `10 kg + 5 kg` | `15 kg` | `15 kg` (unchanged — same-type +/-) |
| `10 kg * 3` | `30 kg` | `30 kg` (unchanged — Quantity * Number) |

**Files:**

| File | Change |
|------|--------|
| `impl/interpreter/operators.go` | Currency/Currency `/` and `*`: return error with `number()` hint instead of computing |
| `impl/interpreter/fraction_ops.go` | `evalFractionOperation`: error when both operands have units (same or different) for `/` and `*` |
| `impl/interpreter/operators_test.go` | Tests for all error cases |
| `impl/interpreter/fraction_ops_test.go` | Tests for fraction same-unit division error |

**Risk:** This is a **breaking change** for anyone relying on `$x / $y` returning Currency. The current behavior is silently wrong — an error with a helpful message is better than a misleading result. The `number()` workaround is well-documented and straightforward.

### Phase 2: Example quality pass

Update each example file to use modern CalcMark idioms. Grouped by issue type:

#### Use `sum of` for 3+ items

| File | Line | Current | Updated |
|------|------|---------|---------|
| `budget.cm` | ~12 | `fixed_total = rent + utilities + insurance` | `fixed_total = sum of rent, utilities, insurance` |
| `household-budget.cm` | ~36 | `total_fixed = rent + car_payment + ...` (7 items) | `total_fixed = sum of rent, car_payment, ...` |
| `household-budget.cm` | ~50 | `total_variable = groceries + gas + ...` (7 items) | `total_variable = sum of groceries, gas, ...` |
| `markdown_financial.cm` | ~43 | `total_expenses = salaries + infrastructure + ...` | `total_expenses = sum of salaries, infrastructure, ...` |
| `recipe-scaling.cm` | ~29 | `total_cost = cost_flour + cost_sugar + ...` (6 items) | `total_cost = sum of cost_flour, cost_sugar, ...` |

#### Use `% of` instead of `* 0.xx`

| File | Line | Current | Updated |
|------|------|---------|---------|
| `budget.cm` | ~15-16 | `savings_rate = 0.20` + `savings = total_income * savings_rate` | `savings = 20% of total_income` |

#### Use fractions where natural

| File | Line | Current | Updated |
|------|------|---------|---------|
| `recipe-scaling.cm` | throughout | decimal quantities | Use `2/3 cup`, `1/4 tsp`, etc. |
| `datacenter-cost.cm` | ~62 | `load_frac = 200 / 1000` | `load_frac = 1/5` |

#### Fix ratio/percentage expressions (rewrite to use `number()`)

After Phase 1, `$x / $y` will error. Examples that compute ratios must be rewritten:

| File | Line | Current (broken) | Rewritten |
|------|------|-----------------|-----------|
| `markdown_financial.cm` | ~48 | `profit_margin = operating_income / total_revenue * 100` | `profit_margin = number(operating_income) / number(total_revenue) * 100` |
| `household-budget.cm` | ~74-76 | `pct = category / total * 100` | `pct = number(category) / number(total) * 100` |
| `household-budget.cm` | ~104 | `months_runway = current_emergency / monthly_expenses` | `months_runway = number(current_emergency) / number(monthly_expenses)` |
| `job-offer.cm` | ~114-115 | `equity_pct = annual_stock / annual_comp * 100` | `equity_pct = number(annual_stock) / number(annual_comp) * 100` |

#### Fix broken calculations

| File | Issue |
|------|-------|
| `markdown_engineering.cm` | Safety factor shows `0.000403` instead of `~1.5` — units lost during cross-multiplication causing dimensional mismatch. Needs full rewrite of structural calc section. |
| `markdown_scientific.cm` | Efficiency shows `0.33%` but summary claims `5%` — calculation vs prose mismatch. |
| `datacenter-cost.cm` | Annual electricity missing 200 kW multiplier ($876 instead of ~$175K). |

#### Fix rate display issues

| File | Line | Issue |
|------|------|-------|
| `household-budget.cm` | ~118-120 | `$300/month per day` → `$6.666667/day` (ugly precision). Rewrite as `dining_out / 30 days` or round. |

### Phase 3: Diagnostic messages and helpful errors

Enhance error messages to explain *why* an operation isn't supported and *how* to fix it. These are teaching moments — the user tried something reasonable, and we should explain the CalcMark mental model.

**Diagnostic message format:** `what went wrong` + `why` + `how to fix it`

| Expression | Error message |
|-----------|--------------|
| `$100 / $50` | `cannot divide currency by currency — dividing $100 by $50 doesn't produce a dollar amount. Use number() to get a ratio: number($100) / number($50)` |
| `$100 * $50` | `cannot multiply currency by currency — the result would be "square dollars" which isn't a real unit. To scale a price, multiply by a plain number: $100 * 50` |
| `10 kg / 5 kg` | `cannot divide quantity by quantity — dividing kg by kg doesn't produce kg. Use number() to get a ratio: number(10 kg) / number(5 kg)` |
| `10 kg * 5 kg` | `cannot multiply quantity by quantity — the result would be "square kg" which isn't a real unit. To scale, multiply by a plain number: 10 kg * 5` |

### Phase 4: Document intentional limitations on the site

Add a "What's Not Supported (and Why)" section to the language reference or a dedicated page. Use the diagnostic errors as source material — every good error message is also a good doc entry.

**File:** `site/content/docs/language-reference.md` (new section) or `site/content/docs/common-mistakes.md` (new page)

**Content outline:**

#### Type arithmetic rules
- Currency / Currency → error (explain: dollars divided by dollars isn't dollars)
- Quantity / Quantity → error (explain: kg divided by kg isn't kg)
- Currency * Currency → error (explain: square dollars isn't a thing)
- **Use `number()` to extract the raw value** when you want a ratio
- Show before/after: `$500 / $1000` (error) → `number($500) / number($1000)` (0.5)

#### Why fractions require no spaces
- `1/3` = fraction, `1 / 3` = division
- Explain: CalcMark uses whitespace to disambiguate, like Swift
- This means `a/b` where `a` and `b` are variables is always division (only literal integers produce fractions)

#### Why some units don't work with fractions
- `1/3 cup` works, `1/3 hour` doesn't
- Explain: fractions are natural for cooking/hardware, not for time (use `20 minutes` instead)

These are examples of things users will try, get an error, and want to understand why. The docs should preempt the confusion.

### Phase 5: Verify markdown/HTML output

After Phases 1-4, re-run all examples through `--format html` and `--format markdown` to verify output quality. Check for:
- Results that don't match surrounding prose
- Ugly numeric precision in formatted output
- Missing or wrong units
- Error messages that are clear and actionable

## Acceptance Criteria

### Bug fix (Phase 1)

- [ ] `$500 / $1000` → error with `number()` hint
- [ ] `$100 * $50` → error with `number()` hint
- [ ] `10 dogs / 5 dogs` → error with `number()` hint (improve existing message)
- [ ] `10 kg * 5 kg` → error with `number()` hint (improve existing message)
- [ ] `2/3 cup / 1/3 cup` → error with `number()` hint
- [ ] `2/3 cup * 1/3 cup` → error with `number()` hint
- [ ] `$100 / €50` → error (different currencies, already correct)
- [ ] `$100 / 5` → `$20.00` (Currency / Number — unchanged)
- [ ] `$100 * 5` → `$500.00` (Currency * Number — unchanged)
- [ ] `1/3 * $200` → `$66.67` (dimensionless Fraction * Currency — unchanged)
- [ ] `10 kg * 3` → `30 kg` (Quantity * Number — unchanged)
- [ ] `number($500) / number($1000)` → `0.5` (workaround works)
- [ ] Error messages suggest `number()` as the fix
- [ ] All existing tests pass (update tests that relied on wrong behavior)

### Example quality (Phase 2)

- [ ] `recipe-scaling.cm` uses fractions (`2/3 cup`, `1/4 tsp`)
- [ ] All files with 3+ item sums use `sum of`
- [ ] Percentage calculations display correctly after Phase 1 fix
- [ ] No `* 0.xx` patterns where `% of` is clearer
- [ ] All examples produce clean output via `--format display -v`

### Diagnostic quality (Phase 3)

- [ ] Every same-type arithmetic error explains *why* it doesn't work
- [ ] Every error suggests the `number()` fix with a concrete example using the user's actual values
- [ ] Error messages are conversational, not technical jargon

### Documentation (Phase 4)

- [ ] "What's Not Supported" section in site docs with real-world explanations
- [ ] Type arithmetic rules documented with before/after examples
- [ ] Fraction whitespace rule documented
- [ ] Fraction unit restrictions documented

### Output quality (Phase 5)

- [ ] `task test` passes
- [ ] `task quality` passes
- [ ] All 18 examples produce reasonable `--format html` output
- [ ] No currency values where dimensionless ratios are expected

## Dependencies & Risks

**Phase 1 is a breaking change** for Currency/Currency division. Any document relying on `$x / $y` returning Currency will now get Number. This is intentionally breaking because the current behavior is mathematically incorrect.

**Phase 2 depends on Phase 1** for the percentage display fixes — those files can't show correct output until currency division returns Number.

## Sources & References

- Audit findings from conversation on 2026-03-14
- Cross-layer checklist: `docs/solutions/logic-errors/adding-new-type-fraction-cross-layer-checklist.md`
- Operator dispatch: `impl/interpreter/operators.go:160-305`
