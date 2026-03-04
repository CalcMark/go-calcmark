---
title: "feat: Add compound, grow, and depreciate growth functions"
type: feat
status: active
date: 2026-03-04
deepened: 2026-03-04
brainstorm: docs/brainstorms/2026-03-04-growth-functions-brainstorm.md
---

# feat: Add compound, grow, and depreciate growth functions

## Enhancement Summary

**Deepened on:** 2026-03-04
**Agents used:** Architecture Strategist, Performance Oracle, Security Sentinel, Code Simplicity Reviewer, Pattern Recognition Specialist, Decimal Pow Researcher, Learnings Researcher, NL Parser Explorer

### Key Improvements

1. **BY and COMPOUNDED should NOT be global tokens** — use contextual identifier matching (like `"using"` and `"across"`) instead of adding reserved keywords. Unanimous across architecture, simplicity, and pattern recognition reviews. Eliminates backward-compatibility risk.
2. **Use `decimal.PowInt32()` for integer exponents** — O(log n) binary exponentiation, exact arithmetic, avoids ln/exp code path. Fall back to `Pow()` only for fractional exponents.
3. **Reduce MaxPeriods from 100,000 to 10,000** — still far beyond any real-world scenario, reduces worst-case computation time by 10x.
4. **Extend existing `NormalizeTimeUnit()`** at `spec/types/rate.go:126` instead of creating a new `NormalizePeriodName()` function.
5. **Pre-existing security vulnerability** in `^` operator at `impl/interpreter/operators.go:256` — unbounded `decimal.Pow()` with no exponent guard. Should be addressed as prerequisite or parallel work.

### New Considerations Discovered

- Salvage validation gaps: reject negative salvage, reject salvage > principal
- Integer overflow risk in period calculation (`periodsPerYear * totalYears`) — use `decimal.Decimal` throughout
- Post-computation precision cap: apply `.Round(20)` to prevent unbounded precision growth
- NL argument parsing must use `parseExponent()` (not `parseExpression()`) per established pattern
- Validate at every entry point: both NL and functional paths must validate identically (learning from `docs/solutions/logic-errors/exchange-rate-frontmatter-validation.md`)
- Compound Mode 1 (default annual) is internally just Mode 2 with period=year — simplifies implementation

## Overview

Add three growth calculation functions to CalcMark with both functional and natural language syntax:

- **`compound`** -- Exponential compound growth: A = P(1+r)^n with three rate modes
- **`grow`** -- Linear growth: A = P + (increment_rate x duration)
- **`depreciate`** -- Declining balance depreciation with optional salvage value

These form a new **"Growth"** feature category. The design follows standard financial formulas and was validated against industry practice (see brainstorm).

## Problem Statement

CalcMark users already write manual compound growth formulas (e.g., `projected = current_requests * (1 + growth_rate)` in `engineering.cm`). This is error-prone, especially with time period conversions and compounding frequency. Built-in functions make these calculations cleaner, type-safe, and unit-aware.

## Technical Approach

### Architecture

All changes follow the established CalcMark function registration pattern documented in `docs/plans/2026-02-22-design-architecture-functions.md` and `docs/plans/2026-02-22-feat-plain-language-functions-plan.md`.

The NL syntax uses the **prefix keyword** pattern (like `compress 1 GB using gzip`), where the prefix keyword triggers a dedicated parser function that greedily consumes all subsequent tokens for that expression. This avoids conflicts with the existing `over` keyword used for rate accumulation.

**Dependency flow (spec -> impl):**
```
spec/lexer       -- NO new tokens (by/compounded are contextual identifiers, not reserved keywords)
spec/parser      -- NL parsing + functional parsing for growth functions
spec/semantic    -- FunctionSpec entries for type checking
spec/features    -- Registry entries for autocomplete/help
spec/classifier  -- Function detection for line classification
impl/interpreter -- Evaluation logic (growth_functions.go)
```

### Research Insights: Architecture

**Contextual identifiers, not global tokens (HIGH PRIORITY):**
The `"by"` and `"compounded"` keywords must NOT be added as global lexer tokens or reserved keywords. Instead, handle them as contextual identifiers (IDENTIFIER tokens with value checks), following the exact pattern used by `"using"` in `parseNLCompressFunction()` and `"across"` in `parseNLTransferFunction()` at `spec/parser/nl_functions.go`. This was independently identified by architecture, simplicity, and pattern recognition reviews. Adding global tokens would break backward compatibility if any CalcMark document uses `by` or `compounded` as variable or unit names.

**Classifier detection gap:**
The classifier at `spec/classifier/classifier.go:74` only checks token types (`FUNC_AVG`, `FUNC_SQRT`, etc.) in `containsFunctions()`. Since `compound`, `grow`, `depreciate` are IDENTIFIER tokens, the classifier won't detect them. Two options: (a) add a string-value check in `containsFunctions()` for known function identifiers, or (b) extend `containsSpecialKeywords()` at line 91 to include growth keywords. Option (a) is more consistent with the function-detection intent.

**Validation at every entry point:**
Per documented learning from `docs/solutions/logic-errors/exchange-rate-frontmatter-validation.md`: both NL and functional parsing paths must produce identical validation. Factor validation into shared helper functions that both paths call, rather than duplicating logic.

### Implementation Phases

#### Phase 1: Spec Layer Foundation (Types, Registry)

Add the infrastructure that other layers depend on.

**1.1 Lexer: NO new tokens needed**

- [ ] Do NOT add `BY` or `COMPOUNDED` as global token types. Handle both as contextual identifiers in the NL parser only (IDENTIFIER with value check), following the pattern of `"using"` and `"across"` at `spec/parser/nl_functions.go`.
- [ ] Do NOT add `"to"` as a global token either. Same contextual approach.
- [ ] `OVER` already exists -- no change needed (consumed by prefix parser context).
- [ ] **No changes to `spec/lexer/token.go` or `spec/lexer/lexer.go`.**

> **Research insight:** Making `by` and `compounded` reserved keywords would break backward compatibility if any CalcMark document uses these as variable or unit names. The contextual identifier pattern is well-established and risk-free.

**1.2 Period name normalization** (`spec/types/rate.go`)

- [ ] **Extend existing `NormalizeTimeUnit()`** at `spec/types/rate.go:126` to include adjectival forms:
  - `"yearly"` -> `"year"`, `"monthly"` -> `"month"`, `"weekly"` -> `"week"`, `"daily"` -> `"day"`, `"quarterly"` -> `"quarter"`
- [ ] Add `PeriodToPeriodsPerYear()` that returns how many periods per year:
  - `"year"` -> 1, `"quarter"` -> 4, `"month"` -> 12, `"week"` -> 52, `"day"` -> 365
- [ ] Unit tests for both additions

> **Research insight:** Extending the existing `NormalizeTimeUnit()` instead of creating a new `NormalizePeriodName()` avoids code duplication and keeps all time-unit normalization in one place. The adjectival forms are just more entries in the same conceptual mapping.

**1.3 Feature category and registry** (`spec/features/registry.go`)

- [ ] Add `CategoryGrowth Category = "growth"` constant
- [ ] Add `getGrowthFeatures()` returning Feature entries for `compound`, `grow`, `depreciate`
- [ ] Register in `NewRegistry()` with `r.features = append(r.features, getGrowthFeatures()...)`
- [ ] Include NL aliases with `Parseable: true` and `Example` fields for autosuggest

**1.4 Semantic function specs** (`spec/semantic/function_types.go`)

- [ ] Add `FunctionSpec` for `"compound"`:
  ```
  Params: principal(ArgTypeAny), rate(ArgTypePercentage), duration(ArgTypeDuration), period(ArgTypeString, Optional)
  ```
- [ ] Add `FunctionSpec` for `"grow"`:
  ```
  Params: starting_amount(ArgTypeAny), increment_rate(ArgTypeRate), duration(ArgTypeDuration)
  ```
- [ ] Add `FunctionSpec` for `"depreciate"` (3 required params — period and salvage handled at eval time):
  ```
  Params: principal(ArgTypeAny), rate(ArgTypePercentage), duration(ArgTypeDuration), period_or_salvage(ArgTypeAny, Optional), salvage(ArgTypeAny, Optional)
  ```

**Phase 1 success criteria:** All spec-layer types compile. Registry returns growth features. Period normalization passes unit tests. NO lexer changes.

---

#### Phase 2: Parser -- Functional Syntax + Classifier

Implement standard parenthesized function calls first (simpler, establishes AST structure).

**2.1 Parser tests first** (TDD)

- [ ] Write parser tests in `spec/parser/parser_test.go` (or dedicated `growth_parser_test.go`) verifying:
  - `compound($1000, 5%, 10 years)` -> `FunctionCall{Name:"compound", Args:[Currency, Percentage, Duration]}`
  - `compound($1000, 5%, 10 years, monthly)` -> `FunctionCall` with 4 args
  - `compound($1000, 5%, 10 years, compounded monthly)` -> `FunctionCall` with 4 args where 4th is a compound modifier node
  - `grow(10 GB, 2 TB/week, 18 months)` -> `FunctionCall{Name:"grow", Args:[Quantity, Rate, Duration]}`
  - `depreciate($50000, 15%, 5 years)` -> `FunctionCall{Name:"depreciate", Args:[Currency, Percentage, Duration]}`
  - `depreciate($50000, 15%, 5 years, $5000)` -> 4 args (salvage as positional)
  - `depreciate($50000, 15%, 5 years, monthly)` -> 4 args (period)
  - `depreciate($50000, 15%, 5 years, monthly, $5000)` -> 5 args (period + salvage)
  - All tests initially fail (red phase)

**2.2 Functional parsing implementation**

- [ ] The existing `parseFunctionCall()` at `spec/parser/rdparser.go:1057` already handles arbitrary function names with comma-separated args. For Mode 1 and Mode 2, no parser changes are needed -- `compound(...)` parses naturally.
- [ ] For Mode 3 (`compounded monthly` as a single arg): Since `"compounded"` is now a contextual identifier (not a token type), add handling in argument parsing to recognize an IDENTIFIER with value `"compounded"` followed by another IDENTIFIER as a compound modifier expression. Create a new AST node or use a string-pair representation.
- [ ] Verify all parser tests pass (green phase)

**2.3 Classifier update** (`spec/classifier/classifier.go`)

- [ ] The classifier's `containsFunctions()` at line 74 only checks token types (FUNC_AVG, FUNC_SQRT, etc.). Growth functions are IDENTIFIER tokens, so they won't be detected. Add an IDENTIFIER-value check:
  ```go
  // In containsFunctions() — check for growth function identifiers
  if tok.Type == lexer.IDENTIFIER {
      switch tok.Value {
      case "compound", "grow", "depreciate":
          return true
      }
  }
  ```
- [ ] Also add NL detection: IDENTIFIER `"compound"`/`"grow"`/`"depreciate"` NOT followed by LPAREN (NL form) should also classify as Calculation.
- [ ] Test that lines starting with `compound(`, `grow(`, `depreciate(` and NL forms like `compound $1000 by 5% over 10 years` are classified as Calculation, not Markdown.

> **Research insight:** The classifier gap is a common source of regression — NL forms won't be evaluated if the classifier misidentifies the line as Markdown. Test both functional and NL detection explicitly.

**Phase 2 success criteria:** All functional syntax golden tests parse correctly. Classifier identifies both functional and NL growth function lines.

---

#### Phase 3: Parser -- NL Syntax

Implement natural language prefix keyword parsing.

**3.1 NL parser tests first** (TDD)

- [ ] Write parser tests verifying NL syntax produces the SAME AST as functional syntax:
  - `compound $1000 by 5% over 10 years` == `compound($1000, 5%, 10 years)`
  - `compound 500 customers by 20% per month over 12 months` == `compound(500 customers, 20%, 12 months, monthly)`
  - `compound $1000 by 5% compounded monthly over 10 years` == `compound($1000, 5%, 10 years, compounded monthly)`
  - `grow 10 GB by 2 TB/week over 18 months` == `grow(10 GB, 2 TB/week, 18 months)`
  - `depreciate $50000 by 15% over 5 years` == `depreciate($50000, 15%, 5 years)`
  - `depreciate $50000 by 15% over 5 years to $5000` == `depreciate($50000, 15%, 5 years, $5000)`
  - `depreciate $50000 by 15% compounded monthly over 5 years to $5000` == full 5-arg form
  - All initially fail

**3.2 Natural syntax keyword registration** (`spec/parser/rdparser.go`)

- [ ] Add `"compound"`, `"grow"`, `"depreciate"`, `"by"`, `"compounded"` to `isNaturalSyntaxKeyword()` (prevents consumption as unit names)
- [ ] Do NOT add `"to"` globally -- only check contextually in the depreciate parser

**3.3 NL parser functions** (new file `spec/parser/nl_growth_functions.go`)

Following the pattern at `rdparser.go:1009-1028` and `nl_functions.go`:

- [ ] Add lookahead detection in `parsePrimary()`:
  ```go
  // NL function lookahead: "compound <expr> by ..."
  if identName == "compound" && !p.check(lexer.LPAREN) {
      return p.parseNLCompoundFunction()
  }
  // NL function lookahead: "grow <expr> by ..."
  if identName == "grow" && !p.check(lexer.LPAREN) {
      return p.parseNLGrowFunction()
  }
  // NL function lookahead: "depreciate <expr> by ..."
  if identName == "depreciate" && !p.check(lexer.LPAREN) {
      return p.parseNLDepreciateFunction()
  }
  ```
  The `!p.check(lexer.LPAREN)` ensures `compound(...)` still routes to `parseFunctionCall()`.

- [ ] Implement `parseNLCompoundFunction()`:
  1. Parse principal via `p.parseExponent()` (NOT `parseExpression()` — per established NL pattern)
  2. Check next IDENTIFIER value is `"by"` (error if missing: `compound: expected 'by' after principal amount`)
  3. Parse rate via `p.parseExponent()`
  4. Check for optional period modifiers (IDENTIFIER value checks):
     - If `PER` token or IDENTIFIER `"per"`: consume period identifier, set mode = per-period literal
     - If IDENTIFIER `"compounded"`: consume period identifier, set mode = financial
  5. Consume `OVER` token (error if missing: `compound: expected 'over' after rate`)
  6. Parse duration via `p.parseExponent()`
  7. Return `ast.FunctionCall{Name: "compound", Arguments: [principal, rate, duration, mode_info]}`

- [ ] Implement `parseNLGrowFunction()`:
  1. Parse starting amount via `p.parseExponent()`
  2. Check IDENTIFIER value `"by"` (error if missing)
  3. Parse increment rate via `p.parseExponent()` (must be Rate type at eval time)
  4. Consume `OVER` token
  5. Parse duration via `p.parseExponent()`
  6. Return `ast.FunctionCall{Name: "grow", Arguments: [starting_amount, rate, duration]}`

- [ ] Implement `parseNLDepreciateFunction()`:
  1. Same structure as compound but also checks for `"to"` after duration for salvage value
  2. If next token is IDENTIFIER with value `"to"`, consume it and parse salvage via `p.parseExponent()`
  3. Return `ast.FunctionCall{Name: "depreciate", Arguments: [principal, rate, duration, mode_info?, salvage?]}`

> **Research insight (NL parser patterns):** All existing NL parsers (`parseNLReadFunction`, `parseNLCompressFunction`, `parseNLTransferFunction`) use `p.parseExponent()` for argument parsing, NOT `p.parseExpression()`. This prevents NL keyword tokens from being consumed by inner expression parsing. The parsers return `ast.FunctionCall` nodes, ensuring the existing evaluator dispatch works unchanged.

> **Research insight (contextual keywords):** The `"by"` check should use IDENTIFIER value comparison: `p.check(lexer.IDENTIFIER) && p.peekValue() == "by"`. The existing `"using"` and `"across"` patterns at `nl_functions.go:72,115` demonstrate this exact approach.

**3.4 Expression termination**

The NL parsers greedily consume known growth keywords (`by`, `per`, `compounded`, `over`, `to`). When an unknown token is encountered after the duration (or salvage), the parser returns. This ensures `compound $1000 by 5% over 10 years + $500` parses as `(compound ...) + $500`.

**3.5 Verify all NL parser tests pass**

**Phase 3 success criteria:** All NL syntax tests pass. NL and functional forms produce identical ASTs. No regressions in existing `over`/`per` parsing. `"by"` and `"compounded"` handled as contextual identifiers only.

---

#### Phase 4: Interpreter -- Evaluation Logic

Implement the actual computation.

**4.1 Computation bounds (security)**

Per SECURITY.md, add DoS protection:

- [ ] Define max compounding periods constant in `spec/parser/limits.go`: `MaxCompoundPeriods = 10_000`
- [ ] Validate before exponentiation. Error: `"compound: too many periods (%d exceeds limit of %d). Use a larger period or shorter duration."`
- [ ] Use `decimal.Decimal` for all period arithmetic to prevent integer overflow in `periodsPerYear * totalYears`
- [ ] Apply post-computation precision cap: `.Round(20)` to prevent unbounded precision growth

> **Research insight (security):** The original 100,000 cap is unnecessarily high. 10,000 periods covers daily compounding for 27 years — far beyond any real-world scenario. Reducing to 10,000 cuts worst-case computation time by 10x. Define the constant in `spec/parser/limits.go` alongside `MaxNestingDepth` and `MaxTokenCount`.

> **Research insight (pre-existing vulnerability):** The existing `^` operator at `impl/interpreter/operators.go:256` uses unbounded `decimal.Pow()` with no exponent guard. This is a pre-existing DoS vector (`2 ^ 999999999`). Consider adding an exponent guard there as prerequisite or parallel work item.

**4.2 Core computation functions** (`impl/interpreter/growth_functions.go`)

- [ ] `compoundGrowth(principal, rate, periods decimal.Decimal) decimal.Decimal`
  - Formula: `principal * (1 + rate)^periods`
  - **Use `decimal.PowInt32()`** for integer exponents — O(log n) binary exponentiation, exact arithmetic, avoids ln/exp code path
  - Detect integer exponent at runtime: if `periods.Equal(periods.Truncate(0))`, use `(one.Add(rate)).Pow(periods)` via `PowInt32` path
  - Fall back to general `Pow()` only for fractional exponents (rare in practice)
  - Validates rate bounds: -1 < rate <= 1 (i.e., -100% < rate <= 100%)
  - Apply `.Round(20)` to result

- [ ] `compoundGrowthFinancial(principal, nominalRate decimal.Decimal, periodsPerYear, totalPeriods decimal.Decimal) decimal.Decimal`
  - Formula: `principal * (1 + nominalRate/periodsPerYear)^(totalPeriods)`
  - **All period arithmetic in `decimal.Decimal`** — avoid int overflow
  - Use `DivRound` for `nominalRate/periodsPerYear` to control precision at division point
  - Validates total periods against `MaxCompoundPeriods`

- [ ] `linearGrow(startingAmount, incrementPerPeriod, numPeriods decimal.Decimal) decimal.Decimal`
  - Formula: `startingAmount + (incrementPerPeriod * numPeriods)`

> **Research insight (decimal.Pow):** `shopspring/decimal` v1.4.0 fixed a critical bug where `Pow()` silently truncated fractional exponents. The library's `PowInt32` uses binary exponentiation with O(log n) multiplications — well-suited for compound interest where periods are typically integers. `DivisionPrecision` defaults to 16 digits, which is adequate for financial calculations. The `PowWithPrecision` variant is available if explicit precision control is needed.

> **Research insight (Mode simplification):** Mode 1 (default annual, `compound $1000 by 5% over 10 years`) is internally equivalent to Mode 2 with period=year. Implementation can treat Mode 1 as syntactic sugar: when no period is specified, default to `per year`. This unifies the computation path and reduces branching in the evaluator.

**4.3 Eval function wrappers**

- [x] `evalCompoundFunc(interp *Interpreter, f *ast.FunctionCall) (types.Type, error)`
  - Dispatch based on argument count and types:
    - 3 args: Mode 1 (default annual)
    - 4 args: Check 4th arg type -- identifier = Mode 2 period; compound modifier = Mode 3
  - Extract principal value (handle Number, Currency, Quantity types)
  - Extract rate as decimal (from Percentage)
  - Extract duration (from Duration type)
  - Call `compoundGrowth()` or `compoundGrowthFinancial()`
  - Wrap result in same type as principal (preserve currency/unit)

- [x] `evalGrowFunc(interp *Interpreter, f *ast.FunctionCall) (types.Type, error)`
  - Extract starting amount and increment rate
  - Validate unit compatibility between starting amount and rate's quantity unit
  - Convert duration to rate's time unit (approximate OK)
  - Call `linearGrow()`
  - Wrap result in rate's quantity unit

- [x] `evalDepreciateFunc(interp *Interpreter, f *ast.FunctionCall) (types.Type, error)`
  - Validate rate is positive (error if negative)
  - Validate salvage value if provided: reject negative salvage, reject salvage > principal
  - Negate rate and delegate to compound logic
  - If salvage value provided, apply `max(result, salvage)` floor
  - Dispatch 4th arg: identifier -> period, value -> salvage

> **Research insight (validation):** Factor rate validation, period validation, and type extraction into shared helper functions called by all three eval functions. Both NL and functional paths should produce identical validation errors — avoid duplicating validation logic between parser and interpreter layers. Cross-reference: `docs/solutions/logic-errors/exchange-rate-frontmatter-validation.md`.

**4.4 Function registration** (`impl/interpreter/functions.go`)

- [x] Add to `BuiltinFunctions` slice:
  ```go
  {Name: "compound", Synonyms: []string{}, Description: "Compound growth: A = P(1+r)^n", Signature: "compound(principal, rate, duration, period?)", Category: "Growth"},
  {Name: "grow", Synonyms: []string{}, Description: "Linear growth: A = P + (rate x time)", Signature: "grow(starting_amount, rate, duration)", Category: "Growth"},
  {Name: "depreciate", Synonyms: []string{}, Description: "Declining balance depreciation", Signature: "depreciate(principal, rate, duration, period?, salvage?)", Category: "Growth"},
  ```
- [x] Add to `functionEvalMap`:
  ```go
  "compound":   evalCompoundFunc,
  "grow":       evalGrowFunc,
  "depreciate": evalDepreciateFunc,
  ```

**4.5 Interpreter tests** (`impl/interpreter/growth_functions_test.go`)

- [ ] Test compound Mode 1: `compound($1000, 5%, 10 years)` -> `$1628.89`
- [ ] Test compound Mode 2: `compound(500 customers, 20%, 12 months, monthly)` -> `~4456 customers`
- [ ] Test compound Mode 3: `compound($1000, 5%, 10 years, compounded monthly)` -> `$1647.01`
- [ ] Test grow: `grow(10 GB, 2 TB/week, 18 months)` -> `~156.5 TB`
- [ ] Test grow with currency: `grow($500, $100/month, 3 years)` -> `$4100`
- [ ] Test depreciate basic: `depreciate($50000, 15%, 5 years)` -> `$22185.07`
- [ ] Test depreciate with salvage: `depreciate($50000, 15%, 10 years, $5000)` -> floor at $5000
- [ ] Test error: rate > 100% -> error message
- [ ] Test error: rate <= -100% -> error message
- [ ] Test error: negative rate for depreciate -> error message
- [ ] Test error: incompatible units for grow -> error message
- [ ] Test error: period mismatch for compound Mode 2 -> error message
- [ ] Test error: too many periods (DoS) -> error message
- [ ] Test edge: 0% rate -> returns principal unchanged
- [ ] Test edge: 100% compound rate -> doubles each period
- [ ] Test edge: 100% depreciate rate -> goes to 0 (or salvage)
- [ ] Test edge: negative salvage value -> error message
- [ ] Test edge: salvage > principal -> error message
- [ ] Test edge: very large periods near MaxCompoundPeriods boundary
- [ ] Test NL equivalence: NL and functional produce identical results
- [ ] Test precision: verify `.Round(20)` doesn't corrupt financial calculations at standard precision
- [ ] Test PowInt32 path: integer periods use exact arithmetic (no ln/exp)

> **Research insight (fuzz testing):** Add fuzz targets for growth function inputs. Fuzz the rate, period count, and principal to catch unexpected panics or precision issues. Place in `impl/interpreter/growth_functions_fuzz_test.go`.

**Phase 4 success criteria:** All computation tests pass. Error messages are actionable. DoS protection is active. PowInt32 used for integer exponents.

---

#### Phase 5: Golden Tests and Integration

**5.1 Parser golden tests** (`testdata/spec/valid/features/growth_functions.cm`)

- [x] Create `.cm` file with all valid syntax forms
- [ ] Generate `.expected` golden output
- [x] Cover: all 3 modes of compound, grow variants, depreciate with/without salvage, NL and functional forms, variable references

**5.2 Eval golden tests** (`testdata/eval/success/features/growth_functions.cm`)

- [x] Create `.cm` file with expressions and expected results
- [ ] Generate `.expected` golden output
- [x] Cover: financial calculations with known correct values, unit preservation, variable usage

**5.3 Error golden tests** (`testdata/eval/errors/`)

- [ ] Rate out of bounds
- [ ] Period mismatch
- [ ] Incompatible units for grow
- [ ] Negative rate for depreciate
- [ ] Too many periods

**5.4 Integration test**

- [ ] Growth functions used with variables and multi-line documents
- [ ] Growth function results used in arithmetic (`total - principal`)
- [ ] Growth functions work with `as napkin`

**5.5 Full suite validation**

- [ ] Run `task test` -- all tests pass
- [ ] Run `task quality` -- no regressions

**Phase 5 success criteria:** All golden tests pass. Full `task test` and `task quality` green.

---

## Error Messages

| Error condition | Message |
|---|---|
| Rate > 100% for compound | `compound: rate must be between -100% and 100%, got {rate}%` |
| Rate <= -100% for compound | `compound: rate must be greater than -100%, got {rate}%` |
| Negative rate for depreciate | `depreciate: rate must be positive (got {rate}%). Use compound() with a negative rate for growth` |
| Rate > 100% for depreciate | `depreciate: rate must be between 0% and 100%, got {rate}%` |
| Period mismatch (Mode 2) | `compound: duration unit '{dur_unit}' must match rate period '{period}'. Try: compound {principal} by {rate}% per {dur_unit} over {n} {dur_unit}s` |
| Incompatible units (grow) | `grow: starting amount unit '{start_unit}' is incompatible with rate unit '{rate_unit}'` |
| Too many periods | `compound: {n} periods exceeds maximum of 10000. Use a larger period or shorter duration` |
| Negative salvage value | `depreciate: salvage value must be non-negative, got {value}` |
| Salvage exceeds principal | `depreciate: salvage value {salvage} exceeds principal {principal}` |
| Missing BY keyword (NL) | `compound: expected 'by' after principal amount` |
| Missing OVER keyword (NL) | `compound: expected 'over' after rate` |

> **Research insight (error format):** Existing parser errors include a syntax template hint (e.g., `expected 'by' after principal amount`). Existing interpreter errors use a `function: message` format. Maintain this pattern — parser errors help with syntax, interpreter errors help with semantics.

## Design Decisions from SpecFlow Analysis

Several gaps were identified by SpecFlow. Decisions for each:

| Gap | Decision | Rationale |
|---|---|---|
| `TO` token doesn't exist | Handle `"to"` contextually (identifier check in depreciate parser), NOT as global token | Backwards compatibility; `"to"` is too common a word |
| `BY`/`COMPOUNDED` tokens | Handle contextually as IDENTIFIER value checks, NOT global tokens | Same rationale as `"to"` — unanimous across architecture, simplicity, and pattern reviews |
| `over` keyword conflict | NL prefix parsers consume entire expression including `over`; general `over` handler never fires | Same pattern as existing NL functions |
| `per` keyword conflict | NL parsers consume `per` within their context; rate construction doesn't fire inside growth parse | Prefix parser takes priority |
| Mode 1 duration must be years | Convert any duration to years automatically (12 months -> 1 year) | UX: users think in months |
| Mode 2 period mismatch | Convert duration to match period (2 years -> 24 months when per month) | UX: strict matching is too restrictive |
| `depreciate` 4th arg disambiguation | Type-based: identifier = period, value expression = salvage | Unambiguous at eval time |
| `compounded monthly` in functional syntax | Single compound arg (parser recognizes COMPOUNDED + IDENTIFIER as modifier) | Consistent with NL parsing |
| Period normalization | Extend existing `NormalizeTimeUnit()` at `spec/types/rate.go:126` with adjectival forms | Avoids duplication; single source of truth for time unit mapping |
| NL expression termination | Greedy: consume known keywords, stop at unknown tokens | `compound ... over 10 years + $500` -> `(compound ...) + $500` |
| Max compounding periods | Hard cap at 10,000 (reduced from 100,000) | Security per SECURITY.md; 10K covers daily compounding for 27 years |
| Exponentiation method | `PowInt32` for integer exponents, `Pow` for fractional | Exact arithmetic, O(log n), avoids ln/exp code path |
| Mode 1 simplification | Mode 1 (default annual) is internally Mode 2 with period=year | Unifies computation path, reduces branching |
| 0% rate | Allowed, returns principal unchanged | Mathematically correct |
| Quarterly compounding | Supported from the start | Common financial use case |

## Acceptance Criteria

### Functional Requirements

- [ ] `compound` works with 3 modes: default annual, per-period literal, financial nominal rate
- [ ] `grow` works with any compatible starting amount + rate + duration
- [ ] `depreciate` works with auto-negated rate and optional salvage value
- [ ] All functions support both functional `name(...)` and NL prefix syntax
- [ ] Results preserve principal type (currency -> currency, quantity -> quantity)
- [ ] Variable references work as arguments in both syntax forms

### Non-Functional Requirements

- [ ] Rate bounds enforced (-100% < rate <= 100% for compound; 0% < rate <= 100% for depreciate)
- [ ] Max period cap prevents DoS (10,000 period limit)
- [ ] Salvage value validated (non-negative, not exceeding principal)
- [ ] `PowInt32` used for integer exponents; precision capped at 20 decimal places
- [ ] Error messages are actionable with suggestions
- [ ] No backward-incompatibility with existing CalcMark documents
- [ ] `by`, `compounded`, `to` ALL handled as contextual identifiers only (no global tokens)

### Quality Gates

- [ ] `task test` passes (full suite, not just growth tests)
- [ ] `task quality` passes
- [ ] Golden tests cover all syntax forms and error cases
- [ ] Unit tests for period normalization, computation functions, and eval wrappers

## Files Changed

### New Files

| File | Purpose |
|---|---|
| `spec/parser/nl_growth_functions.go` | NL parsing for compound, grow, depreciate |
| `impl/interpreter/growth_functions.go` | Evaluation logic |
| `impl/interpreter/growth_functions_test.go` | Interpreter unit tests |
| `impl/interpreter/growth_functions_fuzz_test.go` | Fuzz testing for growth inputs |
| `testdata/spec/valid/features/growth_functions.cm` | Parser golden test |
| `testdata/eval/success/features/growth_functions.cm` | Eval golden test |

### Modified Files

| File | Change |
|---|---|
| `spec/parser/rdparser.go` | Add NL lookahead for compound/grow/depreciate in `parsePrimary()`; add keywords to `isNaturalSyntaxKeyword()` |
| `spec/semantic/function_types.go` | Add `FunctionSpec` entries |
| `spec/features/registry.go` | Add `CategoryGrowth`, `getGrowthFeatures()` |
| `spec/classifier/classifier.go` | Add growth function IDENTIFIER value detection |
| `spec/types/rate.go` | Extend `NormalizeTimeUnit()` with adjectival forms; add `PeriodToPeriodsPerYear()` |
| `spec/parser/limits.go` | Add `MaxCompoundPeriods = 10_000` |
| `impl/interpreter/functions.go` | Register in `BuiltinFunctions` and `functionEvalMap` |

### NOT Modified (key clarification)

| File | Reason |
|---|---|
| `spec/lexer/token.go` | NO new tokens — `by`, `compounded` are contextual identifiers |
| `spec/lexer/lexer.go` | NO new reserved keywords |

## References

### Internal

- Architecture: `docs/plans/2026-02-22-design-architecture-functions.md`
- NL patterns: `docs/plans/2026-02-22-feat-plain-language-functions-plan.md`
- Brainstorm: `docs/brainstorms/2026-03-04-growth-functions-brainstorm.md`
- NL keyword examples: `spec/parser/rdparser.go:1009-1028` (read/compress/transfer lookahead)
- Rate functions pattern: `impl/interpreter/rate_functions.go`
- Unit system: `spec/units/canonical.go`
- Feature registry: `spec/features/registry.go:135-196`

### Financial Formulas

- Compound interest: A = P(1+r)^n (per-period), A = P(1+r/n)^(nt) (nominal annual)
- Linear growth: A = P + (increment x periods)
- Declining balance: A = P(1-r)^n with optional max(result, salvage)

### Research Sources (from /deepen-plan)

- `shopspring/decimal` Pow documentation: `PowInt32` (binary exponentiation, O(log n)), `PowWithPrecision`, `DivisionPrecision` default 16
- Existing NL parser patterns: `spec/parser/nl_functions.go` (parseNLReadFunction, parseNLCompressFunction, parseNLTransferFunction)
- Contextual identifier pattern: `spec/parser/nl_functions.go:72,115` ("using", "across")
- Classifier function detection: `spec/classifier/classifier.go:74` (containsFunctions)
- Period normalization: `spec/types/rate.go:126` (NormalizeTimeUnit)
- Documented learning: `docs/solutions/logic-errors/exchange-rate-frontmatter-validation.md` (validate at every entry point)
