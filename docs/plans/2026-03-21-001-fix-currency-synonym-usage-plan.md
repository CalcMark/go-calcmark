---
title: "fix: Currency synonym usage — $ and USD treated as different currencies"
type: fix
status: active
date: 2026-03-21
issue: https://github.com/CalcMark/go-calcmark/issues/95
---

# fix: Currency synonym usage — $ and USD treated as different currencies

## Overview

When a user defines a currency with the three-letter ISO code (`100 USD`) and then uses the symbol form (`$32`) in a percentage or arithmetic operation, CalcMark incorrectly reports a type mismatch. The `Currency` type already has separate `Symbol` and `Code` fields and an `IsSameCurrency()` method that correctly normalizes — but two critical code paths compare `.Symbol` instead of `.Code`.

## Problem Statement

```
price = 100 USD
$32 as a % of price
```

Expected: `32%` — `$` and `USD` are synonyms for the same currency.
Actual: Error — "cannot compute currency:$ as % of currency:USD — both values must be the same type."

The same class of bug affects currency arithmetic (`$50 + 100 USD` fails).

## Root Cause

Two locations compare currency identity using the display `Symbol` field rather than the normalized `Code` field:

1. **`impl/interpreter/percentage_of_eval.go:107`** — `extractDecimalForRatio` builds type descriptor as `"currency:" + v.Symbol` instead of `"currency:" + v.Code`.
2. **`impl/interpreter/operators.go:236`** — Currency arithmetic compares `leftCur.Symbol != rightCur.Symbol` instead of using `.Code` or `IsSameCurrency()`.

The `Currency` type at `spec/types/currency.go` already has:
- `Code` field: always the normalized ISO code (e.g., `"USD"`)
- `Symbol` field: the original user input (e.g., `"$"` or `"USD"`)
- `IsSameCurrency()` method: correctly compares via `.Code`

## Proposed Solution

**Strategy:** Compare by `.Code` at all currency comparison sites. Preserve `.Symbol` for display fidelity (users who write `$` continue to see `$` in output).

### Fix 1: `percentage_of_eval.go` line 107

```go
// Before
return v.Value, "currency:" + v.Symbol, nil

// After
return v.Value, "currency:" + v.Code, nil
```

### Fix 2: `operators.go` line 236

```go
// Before
if leftCur.Symbol != rightCur.Symbol {
    return nil, fmt.Errorf("cannot %s different currencies: %s and %s",
        operator, leftCur.Symbol, rightCur.Symbol)
}

// After
if !leftCur.IsSameCurrency(rightCur) {
    return nil, fmt.Errorf("cannot %s different currencies: %s and %s",
        operator, leftCur.Code, rightCur.Code)
}
```

Also update error messages in this path to use `.Code` for clarity.

## Acceptance Criteria

- [ ] `$32 as a % of (100 USD)` returns `32%` (the reported bug)
- [ ] `32 USD as a % of $100` returns `32%` (reverse direction)
- [ ] `$50 + 100 USD` returns `$150.00` (arithmetic synonym)
- [ ] `$200 - 100 USD` returns `$100.00` (subtraction synonym)
- [ ] Cross-currency `$50 as % of 100 EUR` still errors correctly
- [ ] Cross-currency `$50 + 100 EUR` still errors correctly
- [ ] Exchange rate scenario: define `USD_EUR` rate, convert, then `as % of` works
- [ ] All existing tests pass (`task test`)
- [ ] Quality gates pass (`task quality`)

## TDD Implementation Plan

### Phase 1: RED — Write failing tests

#### 1a. Unit test in `impl/interpreter/operators_test.go`

Add test cases to `TestAsPercentOf`:
- `$32 as a % of (100 USD)` → expect `32%`
- `32 USD as a % of $100` → expect `32%`
- `$50 as a % of 100 EUR` → expect error (different currencies)

Add test cases for currency arithmetic:
- `$50 + 100 USD` → expect `$150`
- `$200 - 100 USD` → expect `$100`

#### 1b. Golden test file: `testdata/eval/success/features/currency_synonym.cm`

Test mixed `$`/`USD` usage in:
- Basic arithmetic (`+`, `-`)
- `as a % of` calculations
- Variable references across styles

#### 1c. Golden test file: `testdata/eval/success/features/currency_synonym_exchange.cm`

Test with exchange rate frontmatter:
- Define `USD_EUR` rate
- Convert `$100 in EUR`
- Percentage calculations with converted values

### Phase 2: GREEN — Minimal fix

1. Change `percentage_of_eval.go:107`: `v.Symbol` → `v.Code`
2. Change `operators.go:236`: `leftCur.Symbol != rightCur.Symbol` → `!leftCur.IsSameCurrency(rightCur)`
3. Update error messages in `operators.go` to use `.Code`

### Phase 3: REFACTOR

1. Audit for any other `.Symbol`-based comparisons in the interpreter
2. Ensure error messages consistently use ISO codes for currency identity
3. Verify display formatting still uses `.Symbol` for user-facing output

## Dependencies & Risks

- **Low risk**: The fix is two one-line changes in well-understood code paths.
- **`IsSameCurrency()` is already tested**: `spec/types/types_test.go` lines 70-83 prove it handles `$` vs `USD` correctly.
- **No spec changes needed**: The `spec/` directory is not modified; only `impl/interpreter/` is touched.
- **Display preservation**: Using `.Code` only for comparison (not display) means existing output formatting is unchanged.

## Sources & References

- Issue: [#95 Failed currency synonym usage](https://github.com/CalcMark/go-calcmark/issues/95)
- Root cause: `impl/interpreter/percentage_of_eval.go:107` and `impl/interpreter/operators.go:236`
- Currency type: `spec/types/currency.go` (lines 11-16: struct, lines 30-44: NewCurrency, lines 71-73: IsSameCurrency)
- Existing synonym test: `spec/types/types_test.go:70-83`
- Related learning: `docs/solutions/ui-bugs/currency-code-output-spacing.md`
- Related learning: `docs/solutions/logic-errors/exchange-rate-frontmatter-validation.md`
