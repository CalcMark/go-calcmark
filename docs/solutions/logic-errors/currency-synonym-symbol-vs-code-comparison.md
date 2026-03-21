---
title: Currency synonym failure — Symbol vs Code identity comparison
category: logic-errors
tags:
  - currency
  - type-identity
  - interpreter
  - symbol-normalization
module: impl/interpreter (operators, percentage_of_eval, functions)
symptom: "$32 as a % of 100 USD" fails with "both values must be the same type" even though $ and USD are synonyms; "$50 + 100 USD" fails with "cannot add different currencies"
root_cause: Three interpreter code paths compared Currency.Symbol (display field) instead of Currency.Code (normalized ISO code) for currency identity, causing symbol-form ($) and code-form (USD) to appear as different currencies
date: 2026-03-21
severity: medium
---

## Problem

When a user writes a currency using the symbol form (`$100`) and another using the ISO code form (`100 USD`), the interpreter treats them as different currencies in:

1. `as a % of` operations — `extractDecimalForRatio` built type descriptors using `.Symbol`
2. `+` and `-` arithmetic — `evalBinaryOperation` compared `.Symbol` fields
3. Aggregate functions (`sum`, `avg`) — `uniformCurrency` compared `.Symbol` fields

The `Currency` type already had separate `Symbol` (display) and `Code` (normalized ISO) fields, plus an `IsSameCurrency()` method that correctly compared by `.Code`. These were simply unused at the three comparison sites.

## Root Cause

`NewCurrency("$")` produces `{Symbol: "$", Code: "USD"}` while `NewCurrency("USD")` produces `{Symbol: "USD", Code: "USD"}`. The three broken sites compared `.Symbol` directly, yielding `"$" != "USD"`.

## Fix

1. `percentage_of_eval.go:107` — `v.Symbol` → `v.Code` in type descriptor string
2. `operators.go:236` — `leftCur.Symbol != rightCur.Symbol` → `!leftCur.IsSameCurrency(rightCur)`
3. `functions.go:649` — `c.Symbol != first.Symbol` → `!c.IsSameCurrency(first)`

Error messages updated to use `.Code` for clarity (user sees "USD" and "EUR" instead of "$" and "EUR").

## Key Learning

**Currency identity must always be compared at the canonical code level (`.Code` or `IsSameCurrency()`), never at the display symbol level (`.Symbol`).** The `.Symbol` field preserves user input for display fidelity only.

The existing `evalComparison` function already followed this pattern correctly — the three fixed sites were inconsistent outliers.

## Prevention

When adding new code paths that compare currencies, use `IsSameCurrency()` or compare `.Code`. Grep for `.Symbol.*!=` or `!= .*\.Symbol` in the interpreter to catch regressions.

## References

- Issue: https://github.com/CalcMark/go-calcmark/issues/95
- PR: https://github.com/CalcMark/go-calcmark/pull/98
