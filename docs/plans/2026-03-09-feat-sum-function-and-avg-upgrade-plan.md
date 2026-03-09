---
title: "feat: sum() function and avg() quantity/duration upgrade"
type: feat
status: completed
date: 2026-03-09
brainstorm: docs/brainstorms/2026-03-09-sum-function-brainstorm.md
related_issues:
  - https://github.com/CalcMark/go-calcmark/issues/46
---

# feat: sum() function and avg() quantity/duration upgrade

## Overview

Add a variadic `sum()` function that aggregates 2+ values of compatible types, and upgrade the existing `avg()` function to support the same expanded type semantics. Both functions gain support for Quantity and Duration arguments with automatic unit conversion within the same unit family and auto-scaling at display time.

## Problem Statement

CalcMark users writing financial rollups or resource summaries must chain `a + b + c + d` for multi-line totals. A dedicated `sum()` function is cleaner:

```
total_senior_cost = senior_consultants * senior_fully_loaded → $574.2K
total_mid_cost = mid_consultants * mid_fully_loaded → $650K
total_junior_cost = junior_consultants * junior_fully_loaded → $368.64K
total_mgmt_cost = mgmt_hc * mgmt_avg_fully_loaded → $475.2K

total_labor_cost = sum of total_senior_cost, total_mid_cost, total_junior_cost, total_mgmt_cost → $2.07M
```

Additionally, `avg()` today rejects Quantity and Duration arguments — `avg(1 kg, 2 kg)` fails. Both aggregation functions should handle the same types.

## Proposed Solution

### sum() — new function

| Aspect | Decision |
|--------|----------|
| Min args | 2 (sum of 1 value is meaningless) |
| Supported types | Number, Currency, Quantity, Duration, Percentage |
| NL syntax | `sum of a, b, c` (lexer fusion, mirrors `average of`) |
| Traditional syntax | `sum(a, b, c)` |
| Aliases | `total` = search-only (not parseable, avoids variable name conflict) |
| Category | Math |
| Unit conversion | Auto-convert within same unit family via `convertQuantity()` |
| Result unit | First argument's unit; display auto-scales via formatter |
| Cross-system | Supported (e.g., g + lbs) via existing `convertQuantity()` bridge |

*Percentage type (#46) is landing concurrently. `sum(10%, 20%, 30%)` → `60%`. The `aggregateValues()` helper includes a `*types.Percentage` code path from day one. `ArgTypePercentage` semantic hint already exists in `function_types.go`.

### avg() — upgrade

Expand `avg()` to accept Quantity and Duration in addition to Number and Currency. Same auto-conversion and unit handling as sum().

**Backwards compatibility note:** `avg()` currently accepts mixed Number+Currency (e.g., `avg($100, 50)` → `75` as plain Number via `extractNumbers()`). This coercion behavior is preserved for backwards compatibility. The new Quantity/Duration paths are additive.

## Type Compatibility Matrix

The first argument determines the expected type. All subsequent arguments must be the same type family.

| First arg type | Compatible with | Incompatible with | Conversion |
|---------------|----------------|-------------------|------------|
| Number | Number | Everything else | None needed |
| Currency (same code) | Currency (same code) | Different currencies, Number, Quantity, Duration, Percentage | None needed |
| Currency (diff code) | — | — | Error: "use explicit currency conversion" |
| Quantity | Quantity (same dimension) | Different dimensions, Number, Currency, Duration, Percentage | `convertQuantity()` to first arg's unit |
| Duration | Duration | Number, Currency, Quantity, Percentage | `Duration.Convert()` to first arg's unit |
| Percentage | Percentage | Number, Currency, Quantity, Duration | Decimal add/sub → Percentage |

**Strict homogeneity rules (matching `+` operator behavior):**
- `sum($100, 50)` → error (Currency + Number)
- `sum(1 kg, 2)` → error (Quantity + Number)
- `sum(1 hour, 30)` → error (Duration + Number)
- `sum($100, EUR50)` → error (different currency codes; use explicit conversion first)
- `sum(1 kg, 5 meters)` → error (incompatible dimensions)

**Rationale:** These rules match how the `+` operator already works in CalcMark. sum() is semantically `a + b + c`, so its type rules should be identical.

## Technical Approach

### Key design: shared type-dispatch helper

Both `evalSum` and `evalAvg` (upgraded) call a shared `aggregateValues()` function:

```
aggregateValues(args []types.Type) (decimal.Decimal, types.Type, error)
```

Returns: (sum as decimal, template value for result construction, error).

The template value is the first argument (possibly unit-converted) used to construct the correctly-typed result. For avg, divide the sum by `len(args)`.

**Type dispatch logic:**
1. Inspect first arg type → determines expected type family
2. For each subsequent arg:
   - Assert same type family, error if mismatch
   - Convert units if needed (Quantity → `convertQuantity()`, Duration → `Duration.Convert()`)
   - Accumulate decimal value
3. Return sum + first-arg template for result construction

### Display normalization

Auto-scaling happens at **display time, not eval time**. This is consistent with all existing Quantity/Currency arithmetic. The interpreter returns raw values; the formatter applies `NormalizeForDisplay()` for quantities and `formatWithSuffix()` for currencies.

Duration does **not** auto-scale (no Duration unit family in `normalize.go`). `sum(30 minutes, 30 minutes)` → `60 minutes` (not auto-scaled to 1 hour). This matches existing Duration arithmetic behavior.

## Implementation Phases

### Phase 1: Lexer + Parser (spec layer, no impl dependency)

**Files:**

- `spec/lexer/token.go` — Add `FUNC_SUM` and `FUNC_SUM_OF` token constants + `String()` cases
- `spec/lexer/lexer.go` — Add `"sum": FUNC_SUM` to `ReservedKeywords`; add `"sum" + "of"` fusion rule in `combineMultiTokenFunctions()`
- `spec/parser/rdparser.go` — Add `lexer.FUNC_SUM` to `parsePrimary()` function-call match (line ~1060); add `lexer.FUNC_SUM_OF` to NL function match (line ~1065); add min-2-arg validation in `parseFunctionCall()` for sum
- `spec/parser/nl_functions.go` — Add `lexer.FUNC_SUM_OF` case mapping to `funcName = "sum"`, following variadic comma-separated arg path (same as `average of`)

**Tests:**

- `spec/parser/parser_function_test.go` — Add `TestParseSumFunction` (parenthesized, 2+ args), `TestParseSumOfFunction` (NL syntax), test 0-arg and 1-arg error cases
- `spec/lexer/lexer_test.go` — Add tokenization test for `sum of` fusion
- `testdata/spec/valid/features/functions.cm` — Add `sum(1, 2, 3)` parse examples
- `testdata/spec/valid/features/natural_language.cm` — Add `sum of 1, 2, 3` parse examples

**Learnings to apply:**
- Always set `Range` on `ast.FunctionCall` nodes from NL parser (per docs/solutions/logic-errors/nl-function-missing-ast-range.md)
- Test both functional and NL syntax independently — they use separate parser code paths (per docs/solutions/integration-issues/nl-functional-syntax-parity-and-doc-staleness.md)

### Phase 2: Semantic spec + Feature registry (spec layer)

**Files:**

- `spec/semantic/function_types.go` — Add `"sum"` FunctionSpec: `Variadic: true`, `ArgType: ArgTypeAny` (with comment: accepts Number/Currency/Quantity/Duration)
- `spec/features/registry.go` — Add Feature entry:
  ```
  Name: "sum", Category: CategoryFunction
  Syntax: "sum(a, b, c, ...)"
  Description: "Calculate the sum of values"
  Aliases: [{Name: "sum of", Parseable: true, Example: "sum of $100, $200, $300"},
            {Name: "total", Parseable: false}]
  Example: "sum($100, $200, $300) → $600"
  NLExample: "sum of total_a, total_b, total_c"
  ```

**Also update avg():**

- `spec/semantic/function_types.go` — Change `avg` ArgType from `ArgTypeNumber` to `ArgTypeAny` (now accepts Quantity/Duration too)

### Phase 3: Interpreter — shared infrastructure + sum() + avg() upgrade

**Files:**

- `impl/interpreter/functions.go`:
  - Add shared `aggregateValues(args []types.Type) (decimal.Decimal, types.Type, error)` helper
  - Add `evalSumFunc` wrapper in `functionEvalMap`
  - Add `evalSum(args []types.Type) (types.Type, error)` — calls `aggregateValues`, constructs typed result
  - Upgrade `evalAverage` — add Quantity/Duration code paths using `aggregateValues` for those types, keep existing `extractNumbers()` path for Number/Currency (backwards compat)
  - Add `FunctionDef` for sum in `BuiltinFunctions` slice

**aggregateValues design:**

```go
func aggregateValues(args []types.Type) (sum decimal.Decimal, firstArg types.Type, err error) {
    firstArg = args[0]
    switch first := firstArg.(type) {
    case *types.Number:
        // All args must be Number
        for _, arg := range args {
            n, ok := arg.(*types.Number)
            if !ok { return ..., fmt.Errorf("sum(): expected number, got %s", typeDesc(arg)) }
            sum = sum.Add(n.Value)
        }
    case *types.Currency:
        // All args must be Currency with same code
        for _, arg := range args {
            c, ok := arg.(*types.Currency)
            if !ok { return ..., fmt.Errorf("sum(): expected currency, got %s", typeDesc(arg)) }
            if c.Code != first.Code { return ..., fmt.Errorf("sum(): mixed currencies ...") }
            sum = sum.Add(c.Value)
        }
    case *types.Quantity:
        // All args must be Quantity; convert to first's unit
        sum = first.Value
        for _, arg := range args[1:] {
            q, ok := arg.(*types.Quantity)
            if !ok { return ..., fmt.Errorf("sum(): expected quantity, got %s", typeDesc(arg)) }
            converted, err := convertQuantity(q, first.Unit)
            if err != nil { return ..., fmt.Errorf("sum(): %w", err) }
            sum = sum.Add(converted.Value)
        }
    case *types.Duration:
        // All args must be Duration; convert to first's unit
        sum = first.Value
        for _, arg := range args[1:] {
            d, ok := arg.(*types.Duration)
            if !ok { return ..., fmt.Errorf("sum(): expected duration, got %s", typeDesc(arg)) }
            converted, err := d.Convert(first.Unit)
            if err != nil { return ..., fmt.Errorf("sum(): %w", err) }
            sum = sum.Add(converted.Value)
        }
    case *types.Percentage:
        // All args must be Percentage; decimal addition → Percentage
        sum = first.Value
        for _, arg := range args[1:] {
            p, ok := arg.(*types.Percentage)
            if !ok { return ..., fmt.Errorf("sum(): expected percentage, got %s", typeDesc(arg)) }
            sum = sum.Add(p.Value)
        }
    default:
        return ..., fmt.Errorf("sum(): unsupported type %T", firstArg)
    }
    return sum, firstArg, nil
}
```

**evalSum uses aggregateValues + constructs result:**
- Number → `types.NewNumber(sum)`
- Currency → `types.NewCurrency(sum, first.Symbol)`
- Quantity → `&types.Quantity{Value: sum, Unit: first.Unit}`
- Duration → `&types.Duration{Value: sum, Unit: first.Unit}`
- Percentage → `types.NewPercentage(sum)` (or equivalent constructor)

**evalAverage upgrade:**
- Keep existing `extractNumbers()` + `uniformCurrency()` path for Number/Currency args (backwards compat)
- Add new code path: if first arg is `*types.Quantity` or `*types.Duration`, call `aggregateValues`, divide sum by count

**Tests:**

- `impl/interpreter/type_audit_test.go` — Add sum type preservation tests mirroring avg tests (lines 229-270). Add new Quantity/Duration tests for both sum and avg.
- `impl/interpreter/function_consistency_test.go` — No changes (tests are generic), but must pass
- `impl/interpreter/registry_test.go` — Bump expected function count from 15 to 16; add `"sum"` to expected names
- `testdata/eval/success/features/functions.cm` — Add sum evaluation examples
- `testdata/eval/success/features/natural_language.cm` — Add `sum of` NL evaluation examples

**Parity tests (critical — per learnings):**
- Every `sum(a, b, c)` test must have a matching `sum of a, b, c` test asserting identical output
- Every new `avg(1 kg, 2 kg)` test must have a matching `average of 1 kg, 2 kg` test

### Phase 4: Error cases + edge cases

**Golden test files for errors:**

- `testdata/eval/errors/` — Add test cases:
  - `sum(1)` → "sum() requires at least 2 arguments"
  - `sum()` → parse error or "sum() requires at least 2 arguments"
  - `sum($100, 50)` → type mismatch error
  - `sum(1 kg, 5 meters)` → incompatible dimensions error
  - `sum($100, EUR50)` → mixed currency error
  - `sum(1 kg, 2)` → type mismatch error
  - `sum(10 MB/s, 5 MB/s)` → "sum() does not support rate values"

## Acceptance Criteria

- [x] `sum(1, 2, 3)` → `6`
- [x] `sum($100, $200, $300)` → `$600`
- [x] `sum(1 kg, 500 g)` → `1.5 kg`
- [x] `sum(1 g, 10 lbs)` → converts lbs to grams, auto-scales at display
- [x] `sum(1 hour, 30 minutes)` → `1.5 hours`
- [x] `sum of $574.2K, $650K, $368.64K, $475.2K` → `$2.07M`
- [x] NL and traditional syntax produce identical results
- [x] `avg(1 kg, 2 kg)` → `1.5 kg` (upgrade)
- [x] `sum(10%, 20%, 30%)` → `60%`
- [x] `avg(1 hour, 30 minutes)` → `0.75 hours` (upgrade)
- [x] `avg(10%, 20%, 30%)` → `20%` (upgrade)
- [x] `average of 1 kg, 500 g` → same result as `avg(1 kg, 500 g)`
- [x] sum(x) with 1 arg → clear error
- [x] Mixed types → clear error messages naming the function
- [x] `task quality` passes
- [x] All consistency tests pass (function_consistency_test.go)
- [x] AST Range set on all NL-parsed FunctionCall nodes

## Files Changed Summary

| File | Change |
|------|--------|
| `spec/lexer/token.go` | Add `FUNC_SUM`, `FUNC_SUM_OF` + String() |
| `spec/lexer/lexer.go` | Reserved keyword + fusion rule |
| `spec/parser/rdparser.go` | Wire FUNC_SUM and FUNC_SUM_OF into parsePrimary() |
| `spec/parser/nl_functions.go` | Add FUNC_SUM_OF dispatch |
| `spec/semantic/function_types.go` | Add sum spec, update avg ArgType |
| `spec/features/registry.go` | Add sum Feature entry |
| `impl/interpreter/functions.go` | aggregateValues(), evalSum, evalAvg upgrade, BuiltinFunctions entry |
| `impl/interpreter/registry_test.go` | Bump count to 16, add "sum" |
| `impl/interpreter/type_audit_test.go` | Sum + avg type preservation tests |
| `spec/parser/parser_function_test.go` | Parse tests for sum/sum of |
| `testdata/eval/success/features/functions.cm` | Sum golden examples |
| `testdata/eval/success/features/natural_language.cm` | Sum NL golden examples |
| `testdata/eval/errors/` | Error case golden tests |
| `testdata/spec/valid/features/functions.cm` | Sum parse examples |
| `testdata/spec/valid/features/natural_language.cm` | Sum NL parse examples |

## Future Considerations

- **Rate support:** `sum(10 MB/s, 5 MB/s)` is mathematically valid. Deferred — error at launch with clear message.
- **Duration auto-scaling:** Could add a Duration unit family to `format/display/normalize.go` so `sum(30 min, 30 min)` displays as `1 hour` instead of `60 minutes`. Deferred — matches current Duration display behavior.
- **`total` parseable alias:** Permanently excluded due to variable name conflicts (`total = sum(...)`).

## References

- Brainstorm: `docs/brainstorms/2026-03-09-sum-function-brainstorm.md`
- avg() template: `impl/interpreter/functions.go:476-503`
- Unit conversion: `impl/interpreter/unit_conversion.go:39-53`
- Display normalization: `format/display/normalize.go:273-336`
- NL function AST Range bug: `docs/solutions/logic-errors/nl-function-missing-ast-range.md`
- NL/functional parity: `docs/solutions/integration-issues/nl-functional-syntax-parity-and-doc-staleness.md`
- Percentage widening issue: https://github.com/CalcMark/go-calcmark/issues/46
