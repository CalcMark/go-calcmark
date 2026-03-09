---
title: "Percentage Type with Widening on Addition and Subtraction"
type: feat
status: active
date: 2026-03-09
---

# feat: Percentage Type with Widening on Addition and Subtraction

## Overview

Add a `Percentage` type to CalcMark that preserves percentage identity through variables and applies natural widening on `+` and `-`. This enables `salary + 32%` to mean "increase salary by 32%" instead of the current unintuitive `salary + 0.32`.

**Breaking change:** `100 + 20%` changes from `100.2` to `120`. Accepted — the old behavior is unintuitive and no existing golden tests or example files use the `value + pct` pattern.

## Proposed Solution

Follow the rate widening pattern (`operators.go:52-68`): detect `*types.Percentage` on the right side of `+`/`-` and apply `left * (1 ± pct.Decimal())`, preserving the left operand's type.

### Type Interaction Matrix

#### Addition / Subtraction (widening)

| Left | Right (Percentage) | `+` Result | `-` Result |
|------|-------------------|------------|------------|
| Number | Percentage | `left * (1 + pct)` → Number | `left * (1 - pct)` → Number |
| Currency | Percentage | `left * (1 + pct)` → Currency | `left * (1 - pct)` → Currency |
| Quantity | Percentage | `left * (1 + pct)` → Quantity | `left * (1 - pct)` → Quantity |
| Duration | Percentage | `left * (1 + pct)` → Duration | `left * (1 - pct)` → Duration |
| Rate | Percentage | `left * (1 + pct)` → Rate | `left * (1 - pct)` → Rate |
| Percentage | Percentage | decimal add → Percentage | decimal sub → Percentage |
| Percentage | Non-Percentage | Error | Error |

#### Multiplication / Division (no widening)

| Left | Right | `*` Result | `/` Result |
|------|-------|------------|------------|
| Number | Percentage | Number | Number |
| Currency | Percentage | Currency | Currency |
| Percentage | Number | Number | Number |
| Percentage | Percentage | Percentage | Number |

#### Comparison

Percentages compare by underlying decimal value. `Percentage` vs `Percentage` compares directly. `Percentage` vs `Number` compares decimal values.

#### Chained operations

Left-to-right evaluation, each widening independently:
- `$100 + 10% - 20%` = `$100 * 1.10 * 0.80` = `$88`

This is consistent with standard expression evaluation order.

### Display

`Percentage(0.325)` displays as `32.5%` — multiply by 100, strip trailing zeros, append `%`.

### Left-side error message

`32% + $145000` → Error: `cannot add percentage to currency; percentage must appear on the right (e.g., $145000 + 32%)`

## Files to Modify

### Phase 1: Type Definition (spec layer)

| File | Change |
|------|--------|
| `spec/types/percentage.go` | **NEW** — `Percentage` struct with `Value decimal.Decimal`, `String()` → `"32%"` |
| `spec/types/number.go` | Add `*Percentage` case to `typeName()` and `ToDecimal()` |

### Phase 2: Parsing (spec + impl layers)

| File | Change |
|------|--------|
| `spec/document/literal_eval.go:222-228` | Return `*types.Percentage` instead of dividing by 100 and returning decimal |
| `impl/interpreter/helpers.go` | Update `expandNumberLiteral()` to return `*types.Percentage` for `%` suffix |
| `impl/interpreter/percentage_of_eval.go` | Handle `*types.Percentage` in type switch (was `*types.Number`) |

### Phase 3: Operators (impl layer)

| File | Change |
|------|--------|
| `impl/interpreter/operators.go` | Add percentage widening block before type dispatch (mirror rate widening at lines 52-68) |
| `impl/interpreter/operators.go` | Add `*Percentage` cases in `evalBinaryOperation` for `*`, `/` |
| `impl/interpreter/operators.go` | Add `*Percentage` case in `evalComparison` |
| `impl/interpreter/operators.go` | Add `*Percentage` case in `formatTypeForError()` |
| `impl/interpreter/operators.go` | Update unary negation to handle `*Percentage` |

### Phase 4: Function integration

| File | Change |
|------|--------|
| `impl/interpreter/capacity_functions.go` | Add `*Percentage` case to `extractDecimalValue()` |
| Any growth function helpers | Add `*Percentage` case where `extractDecimalValue()` is used |

### Phase 5: Formatters

| File | Change |
|------|--------|
| `format/display/formatter.go:39-63` | Add `*types.Percentage` case in `Format()` type switch |
| `format/json_formatter.go:84-124` | Add `*types.Percentage` case in `populateResult()` — set `Type: "percentage"` |

### Phase 6: Semantic analysis

| File | Change |
|------|--------|
| `spec/semantic/types.go` | Add `TypePercentage` kind, update `CheckTypeCompatibility()`, `kindToString()` |

### Phase 7: Feature registry and docs

| File | Change |
|------|--------|
| `spec/features/registry.go:639-648` | Update `%` operator example from `50% → 0.5` to `50% → 50%` |
| `site/content/docs/user-guide.md:923-934` | Update percentages section — show `price - discount` instead of `price * (1 - discount)` |
| `site/content/docs/language-reference.md` | Update percentage literal description and examples |

### Phase 8: Tests

| File | Change |
|------|--------|
| `impl/interpreter/percentage_test.go` | Update breaking expectations: `100 + 20%` → `120`, `100 - 20%` → `80`, `1k + 10%` → `1100`, `20%` display → `20%` |
| `impl/interpreter/percentage_units_test.go` | Verify `1 lb * 20%` still → `0.2 lb` (no widening on `*`) |
| `testdata/eval/success/features/percentage_widening.cm` | **NEW** — golden test with Currency, Quantity, Duration, Rate widening |
| `testdata/eval/error/percentage_left_side.cm` | **NEW** — golden test for left-side error |

### Phase 9: Example updates (optional, can be separate PR)

| File | Change |
|------|--------|
| `testdata/examples/job-offer.cm` | Convert `0.32` → `32%`, simplify `* (1 - total_tax_rate)` → `- total_tax_rate` |
| `testdata/examples/household-budget.cm` | Convert `0.18` → `18%`, etc. |
| `testdata/examples/markdown_financial.cm` | Convert `* (1 + license_growth)` → `+ license_growth` |

## Technical Considerations

- **Learnings: Rate widening pattern** (`docs/solutions/feature-gaps/rate-type-arithmetic-widening.md`) — widening is intentionally asymmetric and must be documented. The widening block recurses into `evalBinaryOperation()`, guard against unbounded recursion.
- **Learnings: Display formatter** (`docs/solutions/ui-bugs/display-formatter-overrides-explicit-unit-conversion.md`) — use type metadata flags to preserve user intent in formatting.
- **Learnings: NaN/Inf** (`docs/solutions/security-issues/nan-inf-panic-yaml-frontmatter-scale.md`) — guard float parsing for percentage frontmatter values.
- **Two code paths for `%` parsing**: `expandNumber()` in `literal_eval.go` (globals path) AND `expandNumberLiteral()` in `helpers.go` (expression path). Both must return `Percentage`.
- **`extractDecimalValue()`** is used by `compound`, `grow`, `capacityAt`, and other functions. Must handle `*Percentage` or those functions break.
- **Chained evaluation is left-to-right**: `$100 + 10% - 20%` = `$88`, not `$90`. This is natural but should be documented.

## Acceptance Criteria

- [ ] `Percentage` type exists in `spec/types/percentage.go`
- [ ] `32%` literal creates a Percentage, displays as `32%`
- [ ] `salary + 32%` widens to `salary * 1.32`, preserving salary's type (Number, Currency, Quantity, Duration, Rate)
- [ ] `price - 20%` widens to `price * 0.80`
- [ ] `32% + salary` produces helpful error
- [ ] `rate1 + rate2` (both Percentage) does decimal addition → Percentage
- [ ] `value * 20%` uses decimal value (no widening), returns value's type
- [ ] `20% * 5` returns Number (not Percentage)
- [ ] `10% of 200` still works (returns `20`)
- [ ] `compound $1000 by 5% over 10` still works
- [ ] `capacity with 20% buffer` still works
- [ ] `globals: tax_rate: "8%"` creates Percentage
- [ ] `0%`, `100%`, `200%`, `-10%` edge cases handled
- [ ] Chained: `$100 + 10% - 20%` = `$88`
- [ ] All formatters handle Percentage (display, JSON)
- [ ] Semantic analyzer handles TypePercentage
- [ ] `task quality` passes

## References

- Brainstorm: `docs/brainstorms/2026-03-09-percentage-widening-brainstorm.md`
- GitHub issue: #46
- Rate widening precedent: `impl/interpreter/operators.go:52-68`
- Current percentage parsing: `spec/document/literal_eval.go:222-228`
- Rate widening learnings: `docs/solutions/feature-gaps/rate-type-arithmetic-widening.md`
- Display formatter learnings: `docs/solutions/ui-bugs/display-formatter-overrides-explicit-unit-conversion.md`
- NaN/Inf security learnings: `docs/solutions/security-issues/nan-inf-panic-yaml-frontmatter-scale.md`
