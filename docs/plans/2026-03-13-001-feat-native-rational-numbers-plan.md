---
title: "feat: Native Rational Numbers (Fractions)"
type: feat
status: active
date: 2026-03-13
origin: docs/brainstorms/2026-03-13-rational-numbers-brainstorm.md
---

# Native Rational Numbers (Fractions)

## Enhancement Summary

**Deepened on:** 2026-03-13
**Research agents used:** 7 (math/big.Rat best practices, lexer disambiguation patterns, performance oracle, security sentinel, code simplicity reviewer, new-calcmark-feature skill gap analysis, golden test breakage audit)

### Key Improvements
1. **Zero breaking changes confirmed** — golden test audit found no existing `INT/INT` (no spaces) patterns in test data; this feature is purely additive
2. **Security hardening** — exponentiation cap, int64 overflow protection, denominator pre-check before arithmetic, `safeNewFromFloat()` helper
3. **Performance optimizations** — avoid string-based Rat↔Decimal conversion (use `NewFromBigRat`/`.Rat()` directly), pre-compute napkin candidates, denominator BitLen pre-check
4. **Simplifications applied** — eliminate `MixedNumberLiteral` AST node (desugar at parse time), store int64 on `FractionLiteral`, inline small helpers
5. **Missing layers discovered** — classifier and detector need FRACTION token support
6. **Fraction as scalar multiplier** — `1/3 * $200` works (→ `$66.67`), fraction quantities restricted to Length/Mass/Volume/Custom

### New Considerations Discovered
- Fraction exponentiation is a DoS vector: `(1/3) ^ 10000` computes `3^10000`
- `Evaluate()` API bypasses semantic analysis — interpreter must independently validate `1/0`
- `MaxNumberLength` is defined but unenforced in the lexer (pre-existing gap)
- Swift provides decade-long precedent for whitespace-sensitive operator disambiguation

## Overview

Add native rational number support to CalcMark so that `1/3` is stored and displayed as the exact fraction `1/3`, not the decimal approximation `0.333333333333333`. This touches all layers of the architecture: lexer, parser, AST, type system, interpreter, and formatters.

The whitespace rule disambiguates fractions from division: `1/3` (no spaces) is a fraction literal, `1 / 3` (spaces) is division. Golden test audit confirms this is **purely additive** — the existing codebase consistently uses spaces around `/` for division. No existing expressions change meaning.

## Problem Statement / Motivation

CalcMark targets real-world calculations: cooking recipes, hardware measurements, capacity planning. These domains use fractions naturally: `2/3 cup flour`, `5/8 inch bolt`, `11 3/8 inch pipe`. Currently, `1/3 + 1/3 + 1/3` evaluates to `0.999999999999999` instead of exactly `1`. Rational number support eliminates this class of precision errors and produces output that matches how humans think about these quantities.

## Proposed Solution

Implement fractions as a first-class type through all layers:

1. **Lexer**: New `FRACTION` token for `integer/integer` (no spaces)
2. **Parser**: `FractionLiteral` AST node; mixed numbers desugared as `BinaryOp("+", ...)`
3. **Type system**: `types.Fraction` wrapping Go's `math/big.Rat`
4. **Interpreter**: Exact rational arithmetic with fraction-sticky semantics
5. **Formatters**: Simplified fractions, mixed numbers, napkin approximation

### Key design rules (see brainstorm: docs/brainstorms/2026-03-13-rational-numbers-brainstorm.md)

- **Whitespace disambiguation**: `1/3` = fraction, `1 / 3` = division (Swift precedent validates this approach)
- **Literal integers only**: Only `INT/INT` syntax produces fractions; `a/b` is always division
- **Fraction is sticky**: Once a value is a fraction, arithmetic preserves it (e.g., `a = 1/3; b = a * 2` → `2/3`)
- **Simplify to lowest terms**: GCD reduction after every operation (Go's `big.Rat.norm()` does this automatically)
- **Mixed number display**: Improper fractions shown as mixed numbers (e.g., `7/3` → `2 1/3`)
- **Threshold fallback**: Denominator > 1000 after GCD reduction → fall back to decimal display
- **Fraction quantities**: Only Length, Mass, Volume, Custom unit categories. NOT Duration, Currency, Boolean, or Rate
- **Fraction as scalar multiplier**: `1/3 * $200` → `$66.67` (fraction converts to decimal, result is Currency). Works with ALL types — the restriction above is only for fraction *quantities* like `1/3 cup`
- **`as napkin`**: Rounds to nearest common fraction (halves, thirds, quarters, eighths, etc.)
- **`as precise`**: No-op on fractions (already exact). Use `number(1/3)` for decimal conversion
- **Space required before unit**: `1/3 cup` valid, `1/3cup` invalid

## Technical Approach

### Architecture

All changes follow the spec → impl dependency direction. The spec layer defines the token, AST nodes, and type. The impl layer handles evaluation, formatting, and display.

**Internal representation**: `math/big.Rat` from Go's standard library. Research confirms this is the right choice — no better Go alternative exists. `big.Rat` is ~1.3x faster and allocates ~45% less memory than `shopspring/decimal` for equivalent arithmetic.

**Conversion bridges** (critical — never use `NewFromFloat()`):
- **Rat → Decimal**: `decimal.NewFromBigRat(rat, 15)` — shopspring's native API, avoids float64 intermediary
- **Decimal → Rat**: `d.Rat()` — shopspring's native method, exact conversion
- **Never** use `decimal.NewFromFloat(rat.Float64())` — introduces IEEE 754 noise and panics on Inf

**Security constraints**:
- GCD reduction after every arithmetic operation (automatic via `big.Rat.norm()`)
- **Denominator pre-check before arithmetic**: if `left.Denom().BitLen() + right.Denom().BitLen() > 30`, convert to decimal before operating (avoids computing huge intermediate `big.Int` products only to discard them)
- Maximum computation denominator: if denominator exceeds `10^9` after GCD reduction, permanently convert to decimal
- **Exponentiation cap**: if either operand of `^` is a Fraction and exponent > 100, convert to decimal before computing (prevents DoS via `(1/3) ^ 10000` computing `3^10000`)
- **Numerator magnitude limit**: if `|numerator| > 10^18` after GCD, convert to decimal (prevents int64 overflow in JSON serialization)
- `1/0` caught at both semantic analysis AND interpreter (defense-in-depth, since `Evaluate()` API bypasses semantic checker)
- Create `safeNewFromFloat(f float64) (decimal.Decimal, error)` helper — guards against NaN/Inf panics. Apply retroactively to all existing `decimal.NewFromFloat()` call sites

### Implementation Phases

#### Phase 1: Type System + Lexer Foundation

Add the Fraction type, AST node, and FRACTION token together — they're interdependent and small enough to ship as one phase.

**Deliverables:**
- `spec/types/fraction.go` — new `Fraction` type wrapping `math/big.Rat`
- `spec/ast/nodes.go` — add `FractionLiteral` node
- `spec/lexer/token.go` + `spec/lexer/fraction.go` — FRACTION token and pure scanning function
- Unit tests for all three

**Files:**

| File | Change |
|------|--------|
| `spec/types/fraction.go` | **New.** `Fraction` struct: `Value *big.Rat`, `IsNapkin bool`, `Unit string` (empty if dimensionless). Methods: `String()`, `Num() *big.Int`, `Denom() *big.Int`, `ToDecimal() decimal.Decimal`, `IsProper() bool`. `NewFraction(num, denom int64) (*Fraction, error)` returns error if `denom == 0`. |
| `spec/types/fraction_test.go` | **New.** Table-driven tests: construction, simplification (`2/4` → `1/2`), String() (mixed numbers, threshold fallback), zero denominator rejection, ToDecimal(). |
| `spec/types/number.go` | Add `*types.Fraction` case to `ToDecimal()` (line ~34) and `typeName()` (line ~50). |
| `spec/ast/nodes.go` | Add `FractionLiteral{Numerator int64, Denominator int64, SourceText string, Range *Range}`. Add case in `ContainsScaleRef()` (line ~386). |
| `spec/lexer/token.go` | Add `FRACTION` to TokenType const block and `String()` method. |
| `spec/lexer/fraction.go` | **New.** Pure function `tryParseFraction(chars []rune, pos int) (num, denom int64, consumed int, ok bool)`. Only matches `digits/digits` where the integer part has no decimal point, no multiplier suffix, and no preceding space before `/`. Enforces `MaxNumberLength` on each component independently. |
| `spec/lexer/fraction_test.go` | **New.** Exhaustive tests: `1/3` ✓, `1 / 3` ✗, `1.5/3` ✗, `1/3cup` ✗, `100/3` ✓, `1/0` ✓, `0/3` ✓, `1e3/4` ✗, `1,000/3` ✓. |
| `spec/lexer/lexer.go` | In `readNumber()`, after reading integer part (line ~203) and BEFORE decimal point check (line ~208): call `tryParseFraction()`. If match, produce FRACTION token and return immediately. |
| `spec/lexer/lexer_test.go` | Regression tests: `1 / 3` → NUMBER DIVIDE NUMBER, `100 MB/s` → rate tokens unchanged. |

**Research insight — lexer placement**: The fraction check must come after the integer digits but before the decimal point check. If the integer has a decimal point, it cannot be a fraction numerator. This avoids any backtracking. The `tryParseFraction()` pure function takes `chars` and `pos` — no lexer state dependency, fully testable in isolation (Rob Pike pattern).

**Research insight — FractionLiteral stores int64, not strings**: The parser already has integer values (splits token on `/`). Storing as strings then re-parsing in the interpreter is unnecessary. `strconv.ParseInt` happens once at parse time. The interpreter creates `big.NewRat(num, denom)` directly.

**Success criteria:**
- `types.NewFraction(1, 3).String()` → `"1/3"`
- `types.NewFraction(7, 3).String()` → `"2 1/3"` (mixed number)
- `types.NewFraction(2, 4).String()` → `"1/2"` (simplified)
- `types.NewFraction(1, 0)` → error
- Lexer: `1/3` → `FRACTION`, `1 / 3` → `NUMBER DIVIDE NUMBER`
- All existing tests pass (no regressions)

#### Phase 2: Parser + Classifier + Detector

Parse FRACTION tokens into AST nodes. Handle mixed number syntax. Update classifier and detector.

**Deliverables:**
- `FRACTION` token handling in `parsePrimary()`
- Mixed number detection: `NUMBER` followed by `FRACTION` → desugar as `BinaryOp("+", NumberLiteral, FractionLiteral)`
- Unit attachment: `FRACTION` followed by unit identifier
- Classifier and detector recognize FRACTION tokens

**Files:**

| File | Change |
|------|--------|
| `spec/parser/rdparser.go` | In `parsePrimary()` (line ~845): add `FRACTION` token case — split value on `/`, create `FractionLiteral` with int64 fields. After creating a `NumberLiteral`, check if NUMBER is integer and next token is `FRACTION` → desugar to `BinaryOp("+", NumberLiteral, FractionLiteral)`. For both bare FRACTION and mixed number results, check if followed by unit identifier → wrap appropriately. |
| `spec/classifier/classifier.go` | Add `FRACTION` to token classification so fraction-containing lines are recognized as calculations. |
| `spec/document/detector.go` | Add `FRACTION` to document detection so documents with fraction expressions are detected as CalcMark. |
| `spec/parser/parser_test.go` | Tests: `1/3` → FractionLiteral, `11 3/8` → BinaryOp("+", NumberLiteral("11"), FractionLiteral(3,8)), `1/3 cup` → FractionLiteral with unit, `11 3/8 inch` → mixed number with unit, `-1/3` → UnaryOp(-, FractionLiteral). |
| `spec/parser/golden_test.go` | Ensure new golden test files are picked up. |
| `testdata/spec/valid/features/fractions.cm` | **New.** Golden parse tests. |

**Research insight — eliminate MixedNumberLiteral**: `11 3/8` is syntactic sugar for `11 + 3/8`. The parser desugars it to `BinaryOp("+", NumberLiteral(11), FractionLiteral(3,8))`. The interpreter already handles `Number + Fraction` via the normalization block. The formatter reconstructs mixed number display from improper fractions (since `7/3` already displays as `2 1/3`). This eliminates ~50-70 LOC across every layer.

**Research insight — classifier + detector gaps**: The `new-calcmark-feature` skill identified that `spec/classifier/classifier.go` and `spec/document/detector.go` must recognize `FRACTION` tokens. Without this, fraction-only lines may not be classified as calculations, and documents with only fraction expressions may not be detected as CalcMark.

**Range tracking (institutional learning):**
Every new node must have `Range` set. Use `tokenRange(fractionToken)` for FractionLiteral. For desugared mixed numbers, use `ast.SpanNodes(numberNode, fractionNode)`.

**Success criteria:**
- `1/3` parses as `FractionLiteral{Numerator: 1, Denominator: 3}`
- `11 3/8 inch` parses as BinaryOp with unit `"inch"`
- `1 / 3` still parses as `BinaryOp{"/", NumberLiteral("1"), NumberLiteral("3")}`
- Classifier recognizes lines containing FRACTION tokens
- All existing parser tests pass

#### Phase 3: Semantic Analysis

Add fraction-specific checks to the semantic analyzer.

**Files:**

| File | Change |
|------|--------|
| `spec/semantic/checker.go` | Add case for `FractionLiteral` in the AST walker. Check: denominator is not zero → `DiagDivisionByZero` warning. Validate unit category is allowed (Length, Mass, Volume, Custom — NOT Duration, Currency, Boolean, Rate) for fraction *quantities* only. |
| `spec/semantic/checker_test.go` | Tests: `1/0` → warning, `1/3 hour` → error (Duration not allowed with fraction quantities), `1/3 cup` → valid, `$1/3` → not a fraction (currency parsing takes priority). |
| `testdata/spec/invalid/features/fractions.cm` | **New.** Invalid fraction tests. |

**Note:** Semantic analysis is advisory — the interpreter must independently validate `1/0` since the `Evaluate()` API can bypass semantic analysis (see Phase 4 defense-in-depth).

**Success criteria:**
- `1/0` produces `DiagDivisionByZero` warning
- `1/3 hour` produces diagnostic error
- `1/3 cup` passes semantic analysis
- All existing semantic tests pass

#### Phase 4: Interpreter — Evaluation + Arithmetic

Evaluate fraction literals and implement fraction arithmetic.

**Deliverables:**
- Evaluate `FractionLiteral`
- Fraction arithmetic: `+`, `-`, `*`, `/`, `^` for Fraction × Fraction and Fraction × Number
- Exponentiation: integer exponents stay fraction, fractional exponents convert to decimal
- `sqrt()`: exact fraction result when both numerator and denominator are perfect squares, decimal fallback otherwise
- Fraction as scalar multiplier for Currency, Duration, Rate, Quantity (all types)
- `number()` function support
- `as napkin` and `as precise` for fractions

**Files:**

| File | Change |
|------|--------|
| `impl/interpreter/interpreter.go` | Add `*ast.FractionLiteral` case in `evalNode()` (line ~65). Inline: validate denominator ≠ 0 (defense-in-depth), create `types.NewFraction(num, denom)`. |
| `impl/interpreter/operators.go` | Add Fraction normalization block at TOP of `evalBinaryOperation()`, before existing dispatch. See strategy below. |
| `impl/interpreter/operators_test.go` | Tests: `1/3 + 1/3 + 1/3` → `1`, `1/3 + 1/4` → `7/12`, `1/3 * 3` → `1`, `2/3 cup + 1/4 cup` → `11/12 cup`, `1/3 * 2.5` → `5/6`, `1/3 * $200` → `$66.67`, `(1/3) ^ 200` → decimal fallback, `(2/3) ^ 3` → `8/27`, `(1/3) ^ (1/2)` → decimal. |
| `impl/interpreter/napkin_eval.go` | Add `*types.Fraction` case. Round to nearest common fraction. Pre-compute candidates at package init. |
| `impl/interpreter/napkin_eval_test.go` | Tests: `11/12 as napkin` → `~1`, `5/12 as napkin` → `~1/2`, `2/3 as napkin` → `~2/3`. |
| `impl/interpreter/precise_eval.go` | Add `*types.Fraction` case → no-op. |
| `impl/interpreter/functions.go` | Add `*types.Fraction` case in `ExtractNumber()`: `decimal.NewFromBigRat(f.Value, 15)`. Add Fraction-aware `sqrt()`: check if `num` and `denom` are perfect squares → return `Fraction(√num, √denom)`, else convert to decimal and use existing Newton's method. |
| `impl/interpreter/number_function_test.go` | Test: `number(1/3)` → `0.333333333333333`. |
| `impl/interpreter/functions_test.go` | Tests: `sqrt(1/4)` → `1/2` (exact), `sqrt(4/9)` → `2/3` (exact), `sqrt(1/3)` → `0.577...` (decimal fallback). |
| `impl/interpreter/comparison.go` | Add Fraction cases in `evalComparison()`: convert both to `big.Rat` for exact comparison. |
| `impl/interpreter/helpers.go` | Add `safeNewFromFloat(f float64) (decimal.Decimal, error)` — guards NaN/Inf. Add `numberToFraction(n *types.Number) *types.Fraction` and `fractionToNumber(f *types.Fraction) *types.Number` conversion helpers. |
| `testdata/eval/success/features/fractions.cm` | **New.** Golden eval tests. |

**Operator dispatch strategy:**

```go
// Fraction normalization — at TOP of evalBinaryOperation(), before existing dispatch

// Fraction ^ anything: three cases
if leftFrac, ok := left.(*types.Fraction); ok && operator == "^" {
    if exp, ok := right.(*types.Number); ok {
        // Case 1: exponent > 100 → convert to decimal (DoS protection)
        if exp.Value.Abs().GreaterThan(decimal.NewFromInt(100)) {
            left = fractionToNumber(leftFrac)
            // fall through to Number ^ Number
        } else if exp.Value.Equal(exp.Value.Truncate(0)) {
            // Case 2: integer exponent → exact fraction result
            // (2/3) ^ 3 → 8/27
            n := exp.Value.IntPart()
            return fractionPow(leftFrac, n) // exact rational exponentiation
        } else {
            // Case 3: fractional exponent → convert to decimal
            // (1/3) ^ (1/2) is irrational, cannot be a fraction
            left = fractionToNumber(leftFrac)
            // fall through to Number ^ Number
        }
    }
}

// Denominator pre-check: if product would overflow, convert to decimal
if leftFrac, ok := left.(*types.Fraction); ok {
    if rightFrac, ok := right.(*types.Fraction); ok {
        if leftFrac.Denom().BitLen() + rightFrac.Denom().BitLen() > 30 {
            return evalBinaryOperation(fractionToNumber(leftFrac), fractionToNumber(rightFrac), operator)
        }
        return evalFractionOperation(leftFrac, rightFrac, operator)
    }
    if rightNum, ok := right.(*types.Number); ok {
        rightFrac := numberToFraction(rightNum)
        return evalFractionOperation(leftFrac, rightFrac, operator)
    }
    // Fraction × Currency/Duration/Rate/Quantity → convert fraction to decimal, use existing dispatch
    left = fractionToNumber(leftFrac)
    // fall through to existing dispatch
}
// Mirror for right-side fraction
if rightFrac, ok := right.(*types.Fraction); ok {
    if leftNum, ok := left.(*types.Number); ok {
        leftFrac := numberToFraction(leftNum)
        return evalFractionOperation(leftFrac, rightFrac, operator)
    }
    right = fractionToNumber(rightFrac)
    // fall through to existing dispatch
}
```

**Key insight — Fraction as scalar multiplier**: When a Fraction meets a non-numeric type (Currency, Duration, Rate, Quantity), convert the fraction to decimal and fall through to existing dispatch. `1/3 * $200` → `Number(0.333...) * Currency($200)` → `Currency($66.67)`. This avoids adding Fraction cases for every type combination.

**Napkin algorithm**: Brute-force over common denominators {2, 3, 4, 6, 8}. For each denominator d, find nearest `p/d` to target via `p = round(target * d)`. Compare distances, pick closest. Pre-compute candidate `big.Rat` values at package init via `sync.Once`. O(1) with 5 iterations.

**Exponentiation and sqrt rules:**
- **Integer exponent**: `(2/3) ^ 3` → `8/27` (exact). Compute `num^n / denom^n` via `big.Int.Exp()`. Cap at exponent 100 to prevent DoS.
- **Fractional exponent**: `(1/3) ^ (1/2)` → convert to decimal, use existing `Pow()`. Result is irrational, cannot stay fraction.
- **`sqrt()` on fractions**: Check if `isqrt(num)^2 == num && isqrt(denom)^2 == denom` (both perfect squares). If yes → `Fraction(isqrt(num), isqrt(denom))`. If no → convert to decimal, use existing Newton's method. Helper: `isqrt(n)` via `big.Int.Sqrt()`.
- **Fraction exponent on Fraction**: `(1/4) ^ (1/2)` — the base `1/4` has perfect-square components, but the exponent `1/2` is fractional → convert to decimal. Not worth special-casing rational roots.

**Research insight — napkin for mixed numbers**: For `2.333...`, extract integer part first, then napkin-round the fractional remainder. `2 + 1/3` → napkin rounds `1/3` (already common) → `~2 1/3`.

**Success criteria:**
- `1/3 + 1/3 + 1/3` → exactly `1`
- `2/3 cup + 1/4 cup` → `11/12 cup`
- `1/3 * $200` → `$66.67`
- `(2/3) ^ 3` → `8/27` (exact fraction)
- `(1/3) ^ (1/2)` → decimal (irrational)
- `(1/3) ^ 200` → decimal (DoS cap)
- `sqrt(1/4)` → `1/2` (exact, perfect squares)
- `sqrt(1/3)` → `0.577...` (decimal fallback)
- `11/12 as napkin` → `~1`
- `number(1/3)` → `0.333333333333333`
- `1/0` → runtime error (even without semantic checker)
- All existing interpreter tests pass

#### Phase 5: Formatters + Integration + Polish

Render fractions in display and JSON formats. Feature registry, fuzzer, golden test audit.

**Files:**

| File | Change |
|------|--------|
| `format/display/formatter.go` | Add `FormatFraction(f *types.Fraction) string` and case in `Format()`. Display: simplified fraction or mixed number via `DivMod` algorithm. Prefix with `~` if `IsNapkin`. Append unit if present. Bypass `roundForDisplay` (fractions are exact, should not be decimal-rounded). |
| `format/display/formatter_test.go` | Tests: `Fraction(1,3)` → `"1/3"`, `Fraction(7,3)` → `"2 1/3"`, `Fraction(2,4)` → `"1/2"`, napkin → `"~2/3"`. |
| `format/json_formatter.go` | Add `*types.Fraction` case in `populateResult()`. Set `Type: "fraction"`, `NumericValue` as decimal approximation via `NewFromBigRat`. Add `Numerator` and `Denominator` fields — check `big.Int.IsInt64()` before converting; omit fields if overflow. |
| `format/json_formatter_test.go` | Test JSON output with normal fractions and overflow-size numerators. |
| `spec/features/registry.go` | Add fraction features: `"fraction literal"` (syntax: `1/3`), `"mixed number"` (syntax: `11 3/8`), `"fraction arithmetic"`. |
| `spec/features/registry_test.go` | Verify new features registered. |
| `spec/lexer/fuzz_test.go` | Add fraction patterns: `1/3`, `0/0`, `1/0`, `999/1000`, `1/99999`, `11 3/8`, `1e3/4`, `$1/3`, `1/3/4`. |
| `spec/parser/fuzz_test.go` | Add fraction parse patterns. |

**Display format rules (via `DivMod` algorithm):**
```go
// 1. denominator == 1 → display as integer
// 2. denominator > 1000 → display as decimal
// 3. |numerator| > denominator → mixed number ("2 1/3")
// 4. else → simple fraction ("2/3")
// 5. IsNapkin → prefix "~"
// 6. Unit → append " cup" etc.
```

**JSON schema — overflow-safe:**
```go
type JSONResult struct {
    // ... existing fields ...
    Numerator   *int64 `json:"numerator,omitempty"`
    Denominator *int64 `json:"denominator,omitempty"`
}
// In populateResult(): check f.Num().IsInt64() before setting.
// If overflow, omit numerator/denominator, only emit numericValue.
```

**Regression audit** (golden test audit confirmed zero existing `INT/INT` patterns break):
- `testdata/spec/valid/expressions/arithmetic.cm` — all division uses spaces ✓
- `testdata/eval/success/features/rates.cm` — rates use `unit/unit` pattern ✓
- `testdata/spec/valid/features/arbitrary_units.cm` — `20 cats / 4` has spaces ✓
- `testdata/seed.cm` — `d = c / 2` has spaces ✓

**Success criteria:**
- `echo "a = 1/3" | ./cm --format json` → `{"type":"fraction","numerator":1,"denominator":3,"numericValue":0.333...}`
- `echo "a = 7/3" | ./cm --format display` → `2 1/3`
- `task test` passes with zero failures
- `task quality` passes
- Fuzzer runs 30s without crashes
- Feature registry includes fraction features

## System-Wide Impact

### Interaction Graph

`1/3 cup` input → lexer `FRACTION` token → classifier recognizes as calculation → parser `FractionLiteral` + unit → semantic check (unit category allowed?) → interpreter creates `types.Fraction` with unit → `as napkin` rounds to common fraction → display formatter renders `"1/3 cup"` or `"~1/3 cup"`. Assignment to variable stores `types.Fraction`, subsequent arithmetic dispatches through fraction normalization block in `evalBinaryOperation()`.

`1/3 * $200` input → fraction normalization block converts `1/3` to `Number(0.333...)` → falls through to existing `Number * Currency` dispatch → `Currency($66.67)`.

### Error Propagation

- `1/0` → lexer passes FRACTION token, semantic analysis emits `DiagDivisionByZero` warning, interpreter independently returns runtime error (defense-in-depth for `Evaluate()` API callers)
- `1/3 hour` → semantic analysis emits error (fraction quantity not supported for Duration)
- `1/3 * 1 hour` → valid (fraction as scalar multiplier, result is Duration)
- Fraction arithmetic overflow (denominator > 10^9 after GCD) → interpreter silently converts to `types.Number` (decimal), no error
- Exponentiation overflow (`(1/3) ^ 200`) → pre-converted to decimal, no DoS
- `number()` on fraction → `decimal.NewFromBigRat(f.Value, 15)`, clean conversion

### State Lifecycle Risks

- **Type sticky conversion**: Once a value is Fraction, it stays Fraction through Fraction×Fraction and Fraction×Number arithmetic. When meeting Currency/Duration/Rate, fraction converts to decimal — result is the other type.
- **One-way decimal fallback**: If denominator exceeds threshold, converts to Number permanently — correct and intentional.
- **No orphaned state**: Fraction is a value type with no external resources.
- **Serialization**: JSON formatter always includes `NumericValue` for backwards compatibility. `Numerator`/`Denominator` omitted if they overflow int64.

### API Surface Parity

- Display formatter: needs `FormatFraction()`
- JSON formatter: needs `Fraction` case + new fields (overflow-safe)
- `number()` function: needs `Fraction` case in `ExtractNumber()`
- `as napkin`: needs `Fraction` case in `evalNapkinConversion()`
- `as precise`: needs `Fraction` case in `evalPreciseConversion()`
- Comparison operators: needs `Fraction` case
- Unary operators: needs `Fraction` case for negation
- Classifier: needs `FRACTION` token recognition
- Detector: needs `FRACTION` token recognition

## Acceptance Criteria

### Functional Requirements

- [ ] `1/3` (no spaces) produces exact fraction `1/3`
- [ ] `1 / 3` (spaces) produces decimal `0.333...` (division, unchanged)
- [ ] `1/3 + 1/3 + 1/3` equals exactly `1`
- [ ] `2/3 cup + 1/4 cup` equals `11/12 cup`
- [ ] `11 3/8 inch` parsed as mixed number, displays as `11 3/8 inch`
- [ ] `5/8 in` works with unit aliases
- [ ] `7/3` displays as `2 1/3` (mixed number)
- [ ] `2/4` simplifies to `1/2`
- [ ] `11/12 cup as napkin` rounds to `~1 cup`
- [ ] `1/3 as precise` is a no-op
- [ ] `number(1/3)` returns `0.333333333333333`
- [ ] `-1/3` works (unary negation)
- [ ] `1/0` produces semantic warning AND runtime error
- [ ] `1/3 hour` produces error (Duration not allowed with fraction quantities)
- [ ] `1/3 * 1 hour` works (fraction as scalar multiplier → Duration result)
- [ ] `1/3 * $200` works → `$66.67` (fraction as scalar multiplier → Currency result)
- [ ] `$1/3` is NOT a fraction (currency parsing takes priority)
- [ ] Variable propagation: `a = 1/3; b = a + a` → `2/3`
- [ ] JSON output includes `numerator`, `denominator`, `numericValue` fields
- [ ] `100 MB/s` rate parsing is unaffected
- [ ] `(2/3) ^ 3` equals `8/27` (exact fraction with integer exponent)
- [ ] `(1/3) ^ (1/2)` produces decimal (fractional exponent → irrational)
- [ ] `(1/3) ^ 200` does not hang (exponentiation cap → decimal fallback)
- [ ] `sqrt(1/4)` equals `1/2` (exact fraction, perfect squares)
- [ ] `sqrt(4/9)` equals `2/3` (exact fraction, perfect squares)
- [ ] `sqrt(1/3)` produces decimal (not perfect squares → fallback)

### Non-Functional Requirements

- [ ] GCD reduction after every fraction operation (automatic via `big.Rat.norm()`)
- [ ] Denominator pre-check via `BitLen()` before arithmetic
- [ ] Computation denominator limit: 10^9 after GCD → convert to decimal
- [ ] Exponentiation cap: exponent > 100 → convert fraction to decimal first
- [ ] Numerator magnitude: `|n| > 10^18` → convert to decimal (int64 safety)
- [ ] Display denominator threshold: > 1000 → decimal fallback
- [ ] Fraction scanning extracted as pure function (testable in isolation)
- [ ] `safeNewFromFloat()` helper created and applied
- [ ] No regressions in existing test suite
- [ ] Fuzzer corpus includes fraction patterns (critical: `0/0`, `1/0`, large numbers, `1/3/4`)
- [ ] Classifier and detector recognize FRACTION tokens

### Quality Gates

- [ ] `task test` passes
- [ ] `task quality` passes
- [ ] All golden test expectations updated (audit confirms zero breaking changes)
- [ ] Feature registry entries added
- [ ] Fuzzer runs 30s without crashes

## Dependencies & Risks

**Dependencies:**
- `math/big` (Go stdlib) — no external dependency needed
- Existing `shopspring/decimal` — bridging via `NewFromBigRat()`/`.Rat()` (native API, not `NewFromFloat`)

**Risks:**
1. **Breaking change**: Golden test audit confirms ZERO existing `INT/INT` (no spaces) patterns. This is purely additive. Risk is limited to user documents outside the test suite.
2. **Lexer complexity**: `readNumber()` is already ~300 lines. Mitigation: `tryParseFraction()` is a pure function in `fraction.go` — adds ~10 lines to `readNumber()` itself.
3. **Operator dispatch explosion**: Adding Fraction to the type matrix. Mitigation: fraction normalization block handles Fraction×Fraction and Fraction×Number; all other combinations convert fraction to decimal and fall through to existing dispatch.
4. **Rate disambiguation**: `100/3 MB/s` — lexer produces FRACTION, parser handles rate `/s` separately. Tested explicitly.
5. **Exponentiation DoS**: `(1/3) ^ 10000`. Mitigation: exponent cap at 100, convert to decimal above.
6. **Denominator explosion**: Chained multiplication of coprime fractions. Mitigation: BitLen pre-check + 10^9 post-check.
7. **int64 overflow in JSON**: Large numerators. Mitigation: check `IsInt64()` before serializing, omit fields if overflow.

## Sources & References

### Origin

- **Brainstorm document:** [docs/brainstorms/2026-03-13-rational-numbers-brainstorm.md](docs/brainstorms/2026-03-13-rational-numbers-brainstorm.md) — Key decisions: whitespace disambiguation, lexer-level FRACTION token, `math/big.Rat` internal representation, mixed number I/O, napkin rounds to common fractions, `as precise` is no-op.

### Internal References

- Lexer number scanning: `spec/lexer/lexer.go:156-468`
- Parser multiplicative: `spec/parser/rdparser.go:432-560`
- Rate detection: `spec/parser/rate_helpers.go:84-103`
- AST nodes: `spec/ast/nodes.go`
- Type system: `spec/types/number.go:34-73`
- Operator dispatch: `impl/interpreter/operators.go:52-305`
- Napkin evaluation: `impl/interpreter/napkin_eval.go`
- `number()` function: `impl/interpreter/functions.go:236-272`
- Display formatter: `format/display/formatter.go:39-66`
- JSON formatter: `format/json_formatter.go:85-129`
- Canonical units: `spec/units/canonical.go`
- Classifier: `spec/classifier/classifier.go`
- Detector: `spec/document/detector.go`

### External References

- [Go math/big.Rat documentation](https://pkg.go.dev/math/big#Rat)
- [shopspring/decimal NewFromBigRat](https://pkg.go.dev/github.com/shopspring/decimal#NewFromBigRat)
- [Julia Rational Numbers](https://docs.julialang.org/en/v1/manual/complex-and-rational-numbers/) — `//` operator precedent
- [Swift operator whitespace disambiguation](https://developer.apple.com/documentation/swift/operator-declarations) — decade-long precedent for whitespace-sensitive operators
- [The Great Rational Explosion](https://schneide.blog/2017/03/09/the-great-rational-explosion/) — denominator growth analysis
- [Lehmer's GCD algorithm](https://en.wikipedia.org/wiki/Lehmer's_GCD_algorithm) — O(n²/log n), used by Go's big.Rat

### Institutional Learnings Applied

- `shopspring/decimal` panics on NaN/Inf — never use `NewFromFloat()` for bridging; use `NewFromBigRat()`/`.Rat()`
- Operator dispatch requires asymmetric rules — fraction normalization block with decimal fallthrough handles this
- AST Range tracking is mandatory — all new nodes set Range
- Test behavior not implementation — golden tests assert rendered output
- Avoid catch-all dispatch — every type combination explicit or errors
- `Evaluate()` API bypasses semantic checker — interpreter must validate independently
- Classifier and detector must recognize new token types — skipping causes silent bugs
