---
title: "Panic on NaN/Inf float64 values from YAML frontmatter parsing"
category: security-issues
tags:
  - security
  - yaml
  - nan
  - inf
  - panic
  - decimal
  - frontmatter
  - scale
  - convert_to
module:
  - spec/document
  - spec/units
symptom: "Process crashes with unrecoverable panic when YAML frontmatter contains .nan, .inf, or -.inf values that reach decimal.NewFromFloat()"
root_cause: "YAML 1.1 permits .nan, .inf, and -.inf as valid float64 values. These parse successfully into Go float64 but cause decimal.NewFromFloat() to panic. No guards existed in parseScaleConfig or Convert to reject non-finite floats."
date_solved: 2026-03-07
severity: critical
---

# Panic on NaN/Inf float64 values from YAML frontmatter parsing

## Problem Statement

CalcMark's YAML frontmatter parser accepts `scale` and `convert_to` directives. When a user supplies `scale: .nan` or `scale: .inf`, Go's YAML library (`gopkg.in/yaml.v3`, which follows YAML 1.1) parses these as `float64` NaN/Inf values. The `shopspring/decimal` library's `decimal.NewFromFloat()` panics on non-finite inputs, crashing the entire CalcMark process with an unrecoverable panic.

This is a denial-of-service vector: any untrusted CalcMark document can crash the interpreter.

## Root Cause Analysis

YAML 1.1 (section 10.2.2) recognizes `.nan`, `.inf`, and `-.inf` as special float literals. Go's YAML decoder maps these to `math.NaN()`, `math.Inf(1)`, and `math.Inf(-1)`. The `parseScaleConfig` function in `spec/document/frontmatter.go` type-asserts the parsed value as `float64` and passes it directly to `decimal.NewFromFloat()` without checking for non-finite values.

Two attack surfaces were identified:

1. **Frontmatter parsing** (`spec/document/frontmatter.go`): Both scalar form (`scale: .nan`) and map form (`scale: {factor: .inf}`) flow through `parseScaleConfig` to `decimal.NewFromFloat()`.
2. **Unit conversion float64 round-trip** (`spec/units/conversion.go`): The `Convert` function converts `decimal.Decimal` to `float64` for unit math, then converts back via `decimal.NewFromFloat()`. Extremely large scaled values could overflow to `Inf` during this round-trip.

A pre-existing guard in `validateExchangeRate` already protected exchange rates from the same class of issue, but the pattern had not been applied to `scale` or `Convert`.

## Working Solution

Add `math.IsNaN`/`math.IsInf` guards before every `decimal.NewFromFloat()` call on the untrusted-input path. Return descriptive errors instead of panicking.

### Code Changes

**`spec/document/frontmatter.go` — `parseScaleConfig`**

Scalar float64 case:
```go
case float64:
    if math.IsNaN(v) || math.IsInf(v, 0) {
        return nil, fmt.Errorf("scale factor must be a finite number")
    }
    return validateScaleConfig(decimal.NewFromFloat(v), nil)
```

Map-form float64 factor:
```go
case float64:
    if math.IsNaN(f) || math.IsInf(f, 0) {
        return nil, fmt.Errorf("scale.factor must be a finite number")
    }
    factor = decimal.NewFromFloat(f)
```

Unknown sub-key rejection (both `scale` and `convert_to`):
```go
for key := range v {
    if key != "factor" && key != "unit_categories" {
        return nil, fmt.Errorf("unknown key %q in scale; valid keys: factor, unit_categories", key)
    }
}
```

**`spec/units/conversion.go` — `Convert`**

Guards before and after the float64 round-trip:
```go
v, _ := value.Float64()
if math.IsNaN(v) || math.IsInf(v, 0) {
    return decimal.Zero, fmt.Errorf("cannot convert %s to %s (non-finite value)", fromUnit, toUnit)
}
baseValue := sourceInfo.ToBaseUnit(v)
targetValue := targetInfo.FromBaseUnit(baseValue)
if math.IsNaN(targetValue) || math.IsInf(targetValue, 0) {
    return decimal.Zero, fmt.Errorf("cannot convert %s to %s (conversion produced non-finite result)", fromUnit, toUnit)
}
return decimal.NewFromFloat(targetValue), nil
```

### Tests Added

Five new test cases in `spec/document/frontmatter_test.go`:

| Test Name | Input | Expected Error |
|-----------|-------|----------------|
| NaN scalar | `scale: .nan` | `"scale factor must be a finite number"` |
| Inf scalar | `scale: .inf` | `"scale factor must be a finite number"` |
| Map factor NaN | `scale: {factor: .nan}` | `"scale.factor must be a finite number"` |
| Unknown sub-key (scale) | `scale: {factor: 4, bogus: true}` | `"unknown key"` |
| Unknown sub-key (convert_to) | `convert_to: {system: imperial, bogus: true}` | `"unknown key"` |

## Prevention Strategies

1. **Guard pattern**: Any `float64` from untrusted input (YAML, JSON, user input) must be checked with `math.IsNaN`/`math.IsInf` before passing to `decimal.NewFromFloat()`.
2. **Fuzz testing**: The existing fuzz test in `spec/document/fuzz_test.go` asserts "NewDocument must never panic on any input." Add NaN/Inf YAML seeds to the corpus for faster discovery.
3. **Future improvement**: Consider a `SafeDecimalFromFloat` wrapper function to eliminate the class of bug entirely. A `forbidigo` lint rule could enforce its use.
4. **Code review**: Any new `decimal.NewFromFloat` call should be scrutinized for non-finite input paths.

## Related Documentation

- `docs/solutions/logic-errors/exchange-rate-frontmatter-validation.md` — Same NaN/Inf guard pattern applied to exchange rates
- `docs/plans/2026-02-23-feat-exchange-rate-robustness-and-tui-insertion-plan.md` — Original plan that identified the vulnerability class
- `SECURITY.md` — Project security policy (does not yet have a specific NaN/Inf entry)
- `spec/document/fuzz_test.go` — Fuzz test ensuring no panics on arbitrary input
