---
title: "fix: Inconsistent diagnostics for `as` keyword with invalid conversion targets"
type: fix
status: completed
date: 2026-03-08
deepened: 2026-03-08
---

# fix: Inconsistent diagnostics for `as` keyword with invalid conversion targets

## Enhancement Summary

**Deepened on:** 2026-03-08
**Sections enhanced:** 3
**Research sources:** Source code analysis, spec flow analysis, architecture review

### Key Improvements
1. `c` is a valid unit alias (celsius in `spec/units/canonical.go:233`), so `a as c` takes the UnitConversion path — same class of bug as `a as cm`, not a separate case
2. Parser must also accept `lexer.CURRENCY_CODE` tokens after `as` (not just `IDENTIFIER`) — the `in` keyword already does this at `rdparser.go:631`
3. `a as EUR` currently FAILS to parse because `EUR` is tokenized as `CURRENCY_CODE`, not `IDENTIFIER` — the `as` path only checks `p.check(lexer.IDENTIFIER)`

### New Considerations Discovered
- The `in` keyword at `rdparser.go:631` already accepts both `IDENTIFIER` and `CURRENCY_CODE` — the `as` keyword should mirror this pattern
- `c` = celsius (valid unit), so only `xyz` is truly the "unknown identifier" case. `c` and `cm` are both currency-to-unit mismatch cases.

## Overview

The `as` keyword for unit/currency conversion produces inconsistent behavior depending on the identifier that follows it. Three cases illustrate the problem:

1. `a = $2; b = a as c` — `c` is actually a valid unit (celsius alias), so parser creates UnitConversion, but evaluator tries currency conversion producing confusing "no exchange rate for USD → c" error
2. `a = $2; b = a as cm` — `cm` is a valid unit (centimeters), so the parser accepts it as a UnitConversion. But at runtime, the evaluator sees a Currency input and tries currency conversion with target `cm`, producing a confusing "no exchange rate for USD → cm" error
3. `a = $2; b = a as xyz` — `xyz` is not a valid unit, parser errors at line 407, classifier marks the line as TEXT (not a calculation), so no diagnostic appears at all

## Problem Statement

**Root cause chain:**

1. **Parser is too strict about what follows `as`**: At `spec/parser/rdparser.go:376-407`, the parser only accepts identifiers after `as` if they are valid duration units or quantity units. Any other identifier causes a parse error.

2. **Parse errors cause classifier fallthrough**: At `spec/classifier/classifier.go:254-266`, when `containsSpecialKeywords` detects `as` and tries to parse, a parse error means the line falls through to TEXT classification. No diagnostic is ever shown.

3. **`isCurrency()` accepts any 3 uppercase letters**: At `spec/parser/helpers.go:20-32`, any 3-letter uppercase string is treated as a currency code. No validation against ISO 4217 or any known currency list.

4. **Evaluator has no unit/currency disambiguation**: At `impl/interpreter/unit_conversion_eval.go:21-23`, when a UnitConversion node's input is a Currency value, the evaluator unconditionally routes to `evalCurrencyConversion()`. If the target is `cm` (a valid unit but not a currency), this produces a confusing error.

## Proposed Solution

### 1. Parser: accept any identifier after `as`, create UnitConversion node

Change `spec/parser/rdparser.go:376-407` so the parser accepts **any** identifier after `as`, not just known units/durations. This ensures lines with `as <anything>` always parse successfully and stay classified as calculations.

```go
if p.check(lexer.IDENTIFIER) || p.check(lexer.CURRENCY_CODE) {
    targetUnit := string(p.peek().Value)
    p.advance() // always consume the target identifier/currency code

    // Resolve known time/unit abbreviations as before
    normalizedTimeUnit := types.NormalizeTimeUnit(targetUnit)
    _, isQuantityUnit := units.NormalizeUnitName(targetUnit)
    isDuration := types.IsValidDurationUnit(targetUnit) || types.IsValidDurationUnit(normalizedTimeUnit)

    resolvedUnit := targetUnit
    if !types.IsValidDurationUnit(targetUnit) && types.IsValidDurationUnit(normalizedTimeUnit) {
        resolvedUnit = normalizedTimeUnit
    }

    node := &ast.UnitConversion{
        Quantity:   left,
        TargetUnit: resolvedUnit,
        Range:      left.GetRange(),
    }

    // Allow chaining: "as precise" / "as napkin"
    if p.match(lexer.AS) {
        if p.match(lexer.PRECISE) {
            return &ast.PreciseConversion{Expression: node, Range: left.GetRange()}, nil
        }
        if p.match(lexer.NAPKIN) {
            return &ast.NapkinConversion{Expression: node, Range: left.GetRange()}, nil
        }
        return nil, p.error("expected 'napkin' or 'precise' after unit conversion 'as'")
    }
    return node, nil
}
```

This way, `a as cm`, `a as xyz`, `a as c`, `a as EUR` all produce a UnitConversion node. Validation moves to the evaluator where we have runtime type information.

### 2. Evaluator: validate currency codes and disambiguate unit vs currency

Change `impl/interpreter/unit_conversion_eval.go:21-23` to validate the target before routing:

```go
if currency, ok := result.(*types.Currency); ok {
    // Check if target is a known quantity unit — if so, error with helpful message
    _, isQuantityUnit := units.NormalizeUnitName(u.TargetUnit)
    if isQuantityUnit {
        return nil, fmt.Errorf("cannot convert currency (%s) to unit '%s'; use a currency code like USD, EUR, GBP",
            currency.Code, u.TargetUnit)
    }
    // Only attempt currency conversion if target looks like a currency code
    return interp.evalCurrencyConversion(currency, u.TargetUnit)
}
```

### 3. Evaluator: validate currency target codes in `evalCurrencyConversion()`

In `evalCurrencyConversion()`, validate the target is a plausible currency code (3 uppercase letters) before looking up the exchange rate. If not, produce a clear diagnostic:

```go
func (interp *Interpreter) evalCurrencyConversion(currency *types.Currency, targetCode string) (types.Type, error) {
    normalizedTarget := types.NormalizeCurrencyCode(targetCode)

    // Validate target looks like a currency code (3 uppercase letters)
    if !isValidCurrencyTarget(normalizedTarget) {
        return nil, fmt.Errorf("'%s' is not a valid currency code; use ISO 4217 codes like USD, EUR, GBP", targetCode)
    }

    // ... rest unchanged
}
```

### 4. Parser `isCurrency()`: keep as-is for now

The `isCurrency()` function in `spec/parser/helpers.go` is used by the lexer for parsing currency literals like `$100` or `100 USD`. Changing it to validate against ISO 4217 would break the lexer's ability to parse unknown currencies in frontmatter exchange rates. The validation belongs in the evaluator where we have full context.

### 5. Evaluator: handle unknown units for non-currency values

When the input is NOT a currency and the target is not a known unit or duration, produce a helpful error:

```go
// After all existing type checks, at the bottom of evalUnitConversion()
return nil, fmt.Errorf("cannot convert %s to '%s'; '%s' is not a recognized unit",
    result.TypeName(), u.TargetUnit, u.TargetUnit)
```

## Acceptance Criteria

- [x] `a = $2; b = a as cm` → clear error: "cannot convert currency (USD) to unit 'cm'; use a currency code"
- [x] `a = $2; b = a as xyz` → clear error: "'xyz' is not a valid currency code; use ISO 4217 codes like USD, EUR, GBP"
- [x] `a = $2; b = a as c` → clear error: "cannot convert currency (USD) to unit 'c'; use a currency code" (c = celsius)
- [x] `a = $2; b = a as EUR` → existing behavior: "no exchange rate defined for USD → EUR" (correct)
- [x] `10 meters as feet` → existing behavior: unit conversion works
- [x] `1 day as seconds` → existing behavior: duration conversion works
- [x] Lines with `as <identifier>` always classified as Calculation (never TEXT)
- [x] Error diagnostics appear on the correct line (not wrong line)
- [x] Existing parser and evaluator tests pass
- [x] New tests for each diagnostic scenario
- [x] `task test` and `task quality` pass

## MVP

### Test: golden test files in `testdata/`

Add golden test cases for the `as` keyword diagnostic scenarios covering currency-to-unit mismatch, invalid currency codes, and unknown conversion targets.

### Fix: 2 files

1. **`spec/parser/rdparser.go:376-407`** — Accept any identifier after `as`, always create UnitConversion node
2. **`impl/interpreter/unit_conversion_eval.go:13-70`** — Add currency/unit disambiguation and target validation

### Verify

Run `task test` and `task quality`.

## Technical Considerations

- The parser change is safe because it's strictly more permissive — all previously valid `as` expressions still parse identically. The only difference is that previously-rejected identifiers now produce UnitConversion nodes instead of parse errors.
- The evaluator already handles the happy path for each type (Currency, Duration, Quantity, Rate). We're adding validation for the unhappy paths that currently either silently fail or produce confusing errors.
- `isCurrency()` in `spec/parser/helpers.go` is NOT changed — it's used by the lexer for tokenizing currency literals, which is a separate concern from conversion target validation.
- The `as` chaining (`as precise`, `as napkin`) continues to work because we preserve the existing chaining logic after creating the UnitConversion node.

## Dependencies & Risks

- **Low risk**: Parser change is additive — more inputs accepted, same outputs for existing valid inputs
- **Evaluator changes are localized**: Only affect the error paths, not the conversion logic itself
- **Institutional learning**: `docs/solutions/logic-errors/exchange-rate-frontmatter-validation.md` documents a similar pattern — currency code validation at the frontmatter level. We extend the same principle to conversion targets.

## References

- Prior art: `docs/solutions/logic-errors/exchange-rate-frontmatter-validation.md`
- Key files: `spec/parser/rdparser.go:362-411`, `spec/parser/helpers.go:20-32`, `impl/interpreter/unit_conversion_eval.go:13-98`, `spec/classifier/classifier.go:254-266`
- Currency types: `spec/types/currency.go`
- Unit definitions: `spec/units/canonical.go`
