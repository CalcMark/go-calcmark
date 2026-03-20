---
title: "feat: Support rate conversions using `per` keyword on variables"
type: feat
status: completed
date: 2026-03-20
---

# feat: Support rate conversions using `per` keyword on variables

## Overview

Enable `d per year` as a natural-language synonym for `convert_rate(d, year)` when `d` is a variable holding a Rate value. Currently, `per` only desugars to `convert_rate` when the left operand is an `ast.RateLiteral` node. When the left operand is an `ast.Identifier` (variable), the parser incorrectly takes the rate-creation path, treating the variable as the amount of a *new* rate rather than an existing rate to convert.

## Problem Statement

```calcmark
# Works — literal rate + per = convert_rate
5 million/day per second          # → ~57.87/second

# Works — functional syntax
d = 10 dogs/day
y = convert_rate(d, year)         # → 3,650 dogs/year

# FAILS — variable + per creates a broken rate instead of converting
y = d per year                    # Creates RateLiteral(Identifier("d"), "year") — wrong!
```

The parser at `spec/parser/rdparser.go:540` checks `if _, isRate := left.(*ast.RateLiteral); !isRate` — this is a parse-time AST type check that cannot know the runtime type of a variable. The fix must handle the case where `left` is an `ast.Identifier` (or any non-RateLiteral expression) followed by `per <time-unit>`.

## Proposed Solution

**Parser-level change** in `parseMultiplicative()` at `spec/parser/rdparser.go`:

The current logic has two `per` branches:
1. **Line 540**: `!isRate` → create `RateLiteral` (rate creation: `5 GB per day`)
2. **Line 583**: `isRate` → desugar to `convert_rate` (rate conversion: `100 MB/day per second`)

The fix: when `left` is an `ast.Identifier` and `per <time-unit>` follows, desugar to `convert_rate(identifier, time_unit)` instead of creating a `RateLiteral`. This is safe because:
- At parse time, we cannot distinguish `d per year` (convert rate in variable) from `amount per year` (create new rate with variable amount)
- The semantic distinction is: **if left is an identifier, treat `per` as conversion** because creating a rate from a bare variable name is not useful (you'd write `d / year` or use a literal)
- The interpreter's `evalConvertRateFunc` already validates that arg[0] resolves to a `*types.Rate` at runtime

**Key change**: In the `!isRate` branch (line 540), add a sub-check: if `left` is `*ast.Identifier`, desugar to `convert_rate` instead of creating a `RateLiteral`.

## Technical Considerations

### Parser Change (spec/parser/rdparser.go)

Modify the `!isRate` branch at line 540:

```go
// Check for rate with "per" keyword: "5 GB per day"
// But skip if left is already a RateLiteral (from slash syntax)
if _, isRate := left.(*ast.RateLiteral); !isRate {
    if p.match(lexer.PER) {
        if !p.match(lexer.IDENTIFIER) {
            return nil, p.error("expected time unit after 'per'")
        }
        timeUnit := string(p.previous().Value)

        if !isTimeUnit(timeUnit) {
            return nil, p.error(fmt.Sprintf("'%s' is not a valid time unit", timeUnit))
        }

        // NEW: If left is an identifier, desugar to convert_rate
        // Variables holding rates should be converted, not used as rate amounts
        if _, isIdent := left.(*ast.Identifier); isIdent {
            targetNode := &ast.Identifier{
                Name:  timeUnit,
                Range: left.GetRange(),
            }
            return &ast.FunctionCall{
                Name:      "convert_rate",
                Arguments: []ast.Node{left, targetNode},
                Range:     left.GetRange(),
            }, nil
        }

        // Otherwise create a new rate literal (e.g., "5 GB per day")
        left = &ast.RateLiteral{
            Amount:     left,
            PerUnit:    timeUnit,
            SourceText: "",
            Range:      left.GetRange(),
        }
    }
}
```

### Semantic Checker (spec/semantic/checker.go)

No changes needed — `convert_rate` is already handled at line 344. The desugared `convert_rate(identifier, time_unit)` will be checked correctly (first arg type-checked, second arg skipped as identifier).

### Interpreter (impl/interpreter/rate_functions.go)

No changes needed — `evalConvertRateFunc` already evaluates arg[0] at runtime and checks `if rate, ok := rateVal.(*types.Rate)`. If the variable doesn't hold a Rate, a clear runtime error is produced.

### Considerations from Institutional Learnings

1. **`isNaturalSyntaxKeyword`**: `per` is already registered — no change needed
2. **Rate arithmetic widening**: No new type interactions — we're producing a `convert_rate` FunctionCall, same as the existing path
3. **`IsExplicit` flag**: The existing `convert_rate` interpreter code handles this — no additional work
4. **NL/functional parity testing**: Must test both `convert_rate(d, year)` and `d per year` produce identical results
5. **Display formatter**: No changes — results flow through existing `convert_rate` display path

### Edge Cases

- `d per year` where `d` is a Number (not a Rate) → runtime error from `evalConvertRateFunc` ("expected Rate, got Number")
- `d per year` where `d` is a Quantity → runtime error (same path)
- `(a + b) per year` where result is a Rate → this is NOT an Identifier, so it takes the rate-creation path. This is acceptable — parenthesized expressions creating rates is valid syntax
- `d per invalid_unit` → parser error "not a valid time unit" (existing validation)

## Acceptance Criteria

- [ ] `d = 10 dogs/day; y = d per year` evaluates to `3,650 dogs/year`
- [ ] `d per year` produces identical results to `convert_rate(d, year)` for all time units
- [ ] Literal rate creation still works: `5 GB per day` creates a Rate (not a convert_rate call)
- [ ] Rate literal conversion still works: `5 million/day per second` converts correctly
- [ ] Runtime error when variable doesn't hold a Rate: `n = 5; x = n per year` → clear error
- [ ] Parser golden tests pass for new syntax in `testdata/spec/valid/features/rate_functions.cm`
- [ ] Eval golden tests pass for new syntax in `testdata/eval/success/features/rate_functions.cm`
- [ ] Parser unit test in `spec/parser/rate_test.go` verifies AST structure (FunctionCall, not RateLiteral)
- [ ] `task test` passes with all existing tests unchanged
- [ ] `task quality` passes

## Success Metrics

- `d per year` works as documented in issue #87
- No regressions in existing rate creation or conversion syntax
- Cross-syntax parity: `convert_rate(x, unit)` === `x per unit` for variable `x`

## Dependencies & Risks

- **Low risk**: Change is scoped to a single branch in `parseMultiplicative()` — no new tokens, no new AST nodes, no interpreter changes
- **Backwards compatibility**: The only breaking change is `identifier per time_unit` — previously this created a RateLiteral with the identifier as amount. This was not useful behavior (creating a rate from a bare variable name) and is exactly what issue #87 asks to change
- **No dependency changes**: All existing infrastructure (convert_rate, semantic checker, interpreter) already supports this

## Sources & References

- GitHub issue: [#87](https://github.com/CalcMark/go-calcmark/issues/87)
- Parser per-handling: `spec/parser/rdparser.go:538-608`
- convert_rate interpreter: `impl/interpreter/rate_functions.go:57-105`
- Semantic checker: `spec/semantic/checker.go:344`
- Institutional learnings:
  - `docs/solutions/logic-errors/compound-bare-frequency-modifier-silently-ignored.md` — keyword consumption anti-pattern
  - `docs/solutions/integration-issues/nl-functional-syntax-parity-and-doc-staleness.md` — NL/functional parity testing
  - `docs/solutions/feature-gaps/rate-type-arithmetic-widening.md` — Rate operator dispatch
