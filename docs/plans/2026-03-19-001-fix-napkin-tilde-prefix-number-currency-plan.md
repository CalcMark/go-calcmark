---
title: "fix: `as napkin` missing ~ prefix on number and currency results"
type: fix
status: completed
date: 2026-03-19
---

# fix: `as napkin` missing ~ prefix on number and currency results

## Overview

`as napkin` produces the `~` approximate prefix for quantities and fractions but not for plain numbers or currencies. The `~` signals approximation, which applies to all napkin results regardless of type.

| Input | Current Output | Expected Output |
|-------|---------------|-----------------|
| `1234567 as napkin` | `1.2M` | `~1.2M` |
| `$1234567 as napkin` | `$1.2M` | `~$1.2M` |
| `1234567 GB as napkin` | `~1.14 PB` | `~1.14 PB` (correct) |

Related: [#79](https://github.com/CalcMark/go-calcmark/issues/79)

## Root Cause

Only `types.Quantity` and `types.Fraction` have an `IsNapkin bool` field. `types.Number` and `types.Currency` lack it entirely. The interpreter sets `IsNapkin = true` for Quantity/Fraction in `evalNapkinConversion`, and the display formatter checks the flag — but Number and Currency have no flag to set or check.

## Proposed Solution

Follow the established `IsNapkin` bool pattern (per institutional learning from `display-formatter-overrides-explicit-unit-conversion.md`):

1. Add `IsNapkin bool` to `types.Number` and `types.Currency`
2. Set `IsNapkin = true` in `evalNapkinConversion` for both types
3. Add `~` prefix in display formatter for both types
4. Set `IsApproximate = true` in JSON formatter for both types

## Acceptance Criteria

- [ ] `1234567 as napkin` displays `~1.2M`
- [ ] `$1234567 as napkin` displays `~$1.2M`
- [ ] `-1234567 as napkin` displays `~-1.2M`
- [ ] `-$1234567 as napkin` displays `~-$1.2M`
- [ ] `47 as napkin` displays `~47` (small numbers still get prefix)
- [ ] `$42.50 as napkin` displays `~$43`
- [ ] JSON output: `is_approximate: true` for napkin numbers and currencies
- [ ] Existing quantity/fraction napkin behavior unchanged
- [ ] `task test` passes
- [ ] `task quality` passes

## MVP

### Layer 1: Types — `spec/types/number.go` and `spec/types/currency.go`

Add `IsNapkin bool` field to both structs. No constructor changes needed — the field defaults to `false` and is only set by the interpreter.

```go
// number.go
type Number struct {
	Value    decimal.Decimal
	IsNapkin bool
}

// currency.go
type Currency struct {
	Value    decimal.Decimal
	Symbol   string
	Code     string
	IsNapkin bool
}
```

### Layer 2: Interpreter — `impl/interpreter/napkin_eval.go`

Set `IsNapkin = true` on Number and Currency results:

```go
case *types.Number:
	rounded := roundToNapkinPrecision(v.Value)
	return &types.Number{Value: rounded, IsNapkin: true}, nil

case *types.Currency:
	rounded := roundToNapkinPrecision(v.Value)
	return &types.Currency{Value: rounded, Symbol: v.Symbol, Code: v.Code, IsNapkin: true}, nil
```

### Layer 3: Display — `format/display/formatter.go`

Add `~` prefix in the `Format()` dispatch for Number, and in `FormatCurrency` for Currency (since it already receives the struct):

```go
// In Format():
case *types.Number:
	s := f.FormatNumber(v.Value)
	if v.IsNapkin {
		return "~" + s
	}
	return s

// In FormatCurrency():
func (f Formatter) FormatCurrency(c *types.Currency) string {
	// ... existing formatting logic ...
	result := symbol + sep + numStr  // or "-" + symbol + sep + numStr
	if c.IsNapkin {
		return "~" + result
	}
	return result
}
```

### Layer 4: JSON — `format/json_formatter.go`

Add `IsApproximate` for Number and Currency in `populateResult`:

```go
case *types.Number:
	jr.Type = "number"
	f := v.Value.InexactFloat64()
	jr.NumericValue = &f
	if v.IsNapkin {
		jr.IsApproximate = true
	}
case *types.Currency:
	jr.Type = "currency"
	f := v.Value.InexactFloat64()
	jr.NumericValue = &f
	jr.Unit = v.Code
	if v.IsNapkin {
		jr.IsApproximate = true
	}
```

### Layer 5: Tests

**`impl/interpreter/napkin_eval_test.go`** — Add/update tests:
- Number napkin sets `IsNapkin = true`
- Currency napkin sets `IsNapkin = true`
- Display output includes `~` prefix for both

**`format/display/display_test.go`** — Add tests:
- `FormatNumber` via `Format(*types.Number{IsNapkin: true})` produces `~` prefix
- `FormatCurrency` with `IsNapkin: true` produces `~` prefix

**`format/json_formatter_test.go`** — Add tests:
- Number napkin → `is_approximate: true`
- Currency napkin → `is_approximate: true`

**`testdata/eval/success/features/napkin.cm`** — Update golden file with `~` prefix on number/currency results.

**TUI catwalk test** — If existing catwalk tests cover napkin numbers, update expected output.

## Sources

- GitHub issue: [#79](https://github.com/CalcMark/go-calcmark/issues/79)
- Institutional learning: `docs/solutions/ui-bugs/display-formatter-overrides-explicit-unit-conversion.md` — documents the `IsNapkin` bool pattern
- Institutional learning: `docs/solutions/ui-bugs/currency-code-output-spacing.md` — currency display spacing rules to preserve
