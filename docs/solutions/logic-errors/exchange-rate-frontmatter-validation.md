---
title: Exchange rate frontmatter validation and currency conversion robustness
category: logic-errors
tags:
  - currency-conversion
  - frontmatter-validation
  - semantic-analysis
  - exchange-rates
  - format-validation
module: spec/document (frontmatter parsing) and impl/interpreter (currency conversion)
symptom: Invalid exchange rate keys accepted (e.g., USDEUR, EUR/GBP, 123_EUR); inline @exchange assignments bypassed rate validation; missing exchange rates showed unhelpful error messages
root_cause: Currency code validation was absent in frontmatter parser; inline @exchange assignments lacked rate positivity checks; error messages didn't suggest correct FROM_TO format
date: 2026-02-23
severity: medium
---

# Exchange Rate Frontmatter Validation

## Problem

CalcMark's exchange rate frontmatter (`exchange:` block) and the inline `@exchange` assignment syntax had several validation gaps that silently accepted invalid inputs.

**Invalid inputs that were accepted before the fix:**

1. **Non-standard currency codes** were not checked for format. Keys like `US_EUR` (2-letter), `USDD_EUR` (4-letter), `123_EUR` (digits), or `U1D_EUR` (mixed alphanumeric) were accepted with no error.

2. **Inline `@exchange` assignments bypassed rate value validation.** The frontmatter YAML parser rejected zero and negative rates, but the interpreter path for `@exchange.USD_EUR = 0` had no such check.

3. **Error messages for missing exchange rates did not guide users.** When `100 USD in EUR` failed, the error didn't mention the expected `USD_EUR` underscore key format.

## Root Cause

**In `spec/document/frontmatter.go`:** `ParseExchangeRateKey` only checked that the key contained exactly one underscore and non-empty parts. No validation that each part was a 3-letter alphabetic code:

```go
// Before: only split check, no code format validation
parts := strings.Split(key, "_")
if len(parts) != 2 { ... }
from, to := parts[0], parts[1]
if from == "" || to == "" { ... }
// "12_EUR", "USDD_EUR", "U1D_EUR" all passed
```

**In `impl/interpreter/variables.go`:** `evalFrontmatterAssignment` for the `"exchange"` namespace had no check for zero or negative rates, and the local `parseExchangeKey` mirror lacked the same code format validation.

**Architectural constraint:** `impl/interpreter` cannot import `spec/document` due to an import cycle through `spec/document/eval_flow_debug_test.go` (internal test package imports `impl/interpreter`). This forced validation logic duplication.

## Solution

### 1. `isValidCurrencyCode` added to both layers

In `spec/document/frontmatter.go` and `impl/interpreter/variables.go`:

```go
func isValidCurrencyCode(code string) bool {
    if len(code) != 3 {
        return false
    }
    for _, r := range code {
        if r < 'A' || r > 'Z' {
            return false
        }
    }
    return true
}
```

### 2. `ParseExchangeRateKey` validates both codes

After splitting on `_`, each part is checked with `isValidCurrencyCode`. Error messages include the expected format: `"must be exactly 3 letters (e.g., USD, EUR)"`.

### 3. Inline `@exchange` path validates rate value

`evalFrontmatterAssignment` now rejects zero and negative rates:

```go
if rate.IsZero() || rate.IsNegative() {
    return nil, fmt.Errorf("@exchange.%s: exchange rate must be positive, got %s",
        f.Property, rate.String())
}
```

### 4. Error messages suggest correct format

Missing exchange rate errors now include the expected `FROM_TO` key format (e.g., `USD_EUR`).

## Verification

Full test suite passes (`task test` + `task quality`). Test coverage spans:

- **`eval_test.go` / `TestFrontmatterErrors`** (13 cases): unclosed frontmatter, no-underscore key, slash/dot separators, too many underscores, NaN/Inf/negative/zero rates, short/long/numeric/mixed currency codes.
- **`eval_test.go` / `TestCurrencyConversion`** (14 cases): valid conversions, undefined pair errors, same-currency no-op, inline `@exchange`, lowercase normalization, error message format.
- **`spec/document/frontmatter_test.go` / `TestParseExchangeRateKey`** (13 cases): direct validation of the spec-layer function.
- **`impl/interpreter/interpreter_test.go` / `TestParseExchangeKey`** (13 cases) and `TestFrontmatterExchangeRateInvalidFormat` (4 cases): interpreter-layer validation.

## Prevention Strategies

### Validate at Every Entry Point

Exchange rates can be set via frontmatter YAML or inline `@exchange` assignments. Both are user-facing entry points and must enforce identical validation. When designing features with multiple configuration paths:

- Document every entry point exhaustively
- Implement validation at each point
- Test both paths with the same valid and invalid inputs

### Managing Duplicated Validation

The import cycle forces duplication of `isValidCurrencyCode` and `parseExchangeKey` across packages. To prevent divergence:

- Each copy has a comment referencing its counterpart
- Both packages have independent tests using identical test cases
- Code reviewers should check for matching changes when either copy is modified

### Error Message Quality

Validation error messages should:
- State what was received
- State what was expected
- Give a concrete example of the correct format

### Testing Checklist for Frontmatter Features

When adding new frontmatter features:

- [ ] Length boundary tests (too short, exact, too long)
- [ ] Character boundary tests (letters, digits, special chars, mixed)
- [ ] Value boundary tests (zero, negative, NaN, Inf, valid)
- [ ] Both frontmatter and inline assignment paths tested equivalently
- [ ] Error messages checked for helpfulness
- [ ] Full test suite run (`task test`) before declaring stability

## Key Lessons

1. **Validate at every entry point, not just the primary one.** The inline `@exchange` path was added later and missed the rate checks that the frontmatter parser already had.

2. **When import cycles force duplication, document the relationship.** Add comments linking copies so future maintainers know to update both.

3. **Error messages should guide users to correctness.** Including the expected format (`FROM_TO`) and examples (`USD_EUR`) in errors dramatically improves developer experience.

4. **Boundary testing is essential for validation logic.** Off-by-one in length checks and character validation are easy to miss without explicit edge case tests.

## Related Documentation

- [docs/plans/2026-02-23-feat-exchange-rate-robustness-and-tui-insertion-plan.md](../../plans/2026-02-23-feat-exchange-rate-robustness-and-tui-insertion-plan.md) - Feature plan that identified these issues
- [testdata/eval/success/features/currency_conversion.cm](../../../testdata/eval/success/features/currency_conversion.cm) - Golden example documenting canonical exchange rate format
- [SECURITY.md](../../../SECURITY.md) - Project security policy (exchange rate validation follows similar defensive patterns)
