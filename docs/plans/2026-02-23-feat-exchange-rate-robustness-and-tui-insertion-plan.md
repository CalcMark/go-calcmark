---
title: "feat: Exchange Rate Robustness and TUI Insertion"
type: feat
status: completed
date: 2026-02-23
---

# Exchange Rate Robustness and TUI Insertion

## Enhancement Summary

**Deepened on:** 2026-02-23
**Sections enhanced:** All phases
**Research agents used:** pattern-recognition-specialist, performance-oracle, security-sentinel, code-simplicity-reviewer, architecture-strategist, Go decimal/currency best practices researcher

### Key Improvements from Research

1. **Bug found**: Error message at `unit_conversion_eval.go:71` uses `/` separator (`USD/EUR`) but the correct frontmatter format is `_` (`USD_EUR`). Must fix.
2. **Security finding**: `decimal.NewFromFloat` does not reject NaN/Inf from YAML `.nan`/`.inf` values. Must guard.
3. **Security finding**: No bounds-checking on exchange rate values (zero, negative, extreme). Must guard.
4. **Duplication found**: `parseExchangeKey` in `impl/interpreter/variables.go` duplicates `ParseExchangeRateKey` in `spec/document/frontmatter.go`. Should consolidate.
5. **Architecture insight**: Golden error test harness (`TestEvalErrorFilesShouldFailEval`) does NOT read `.cm` files - it uses hard-coded inline expressions. Error golden files are orphaned documentation, not automated tests.
6. **Simplicity insight**: Phase 2 (ISO validation) adds a new integration point for warning-only diagnostics with no concrete user complaint driving it. The runtime error already tells users what's wrong. Descope to a bug fix instead.
7. **Performance insight**: All exchange rate operations are well within the 100ms TUI debounce budget. No hot-path concerns.

### Revised Scope

Based on research, the plan is restructured into:
- **Phase 1**: Golden testdata + targeted test expansion (keep, refined)
- **Phase 1b**: Bug fixes discovered during research (new)
- **Phase 2**: ISO 4217 validation (descoped - defer unless user requests)
- **Phase 3**: Ctrl+F template change (keep - user explicitly requested it)

## Problem Statement / Motivation

1. **Test coverage gaps**: No golden `.cm` testdata files exercise exchange rates. Failure cases exist in Go unit tests but are incomplete. Users hitting errors in the wild won't get consistent diagnostics.

2. **Bug: Wrong format in error message**: The runtime error at `unit_conversion_eval.go:71` tells users `exchange: { USD/EUR: <rate> }` but the actual format is `USD_EUR` (underscore). Users following the hint will get a parse error.

3. **Security: NaN/Inf/negative rates silently accepted**: YAML `.nan` and `.inf` values pass through `decimal.NewFromFloat` without validation. Zero and negative rates produce silently wrong results.

4. **Ctrl+F only inserts globals**: The TUI's `insertFrontmatter()` hardcodes a globals-only template. The user explicitly wants exchange rate examples in the default template.

5. **Code duplication**: `parseExchangeKey` in `impl/interpreter/variables.go` is a near-verbatim copy of `ParseExchangeRateKey` in `spec/document/frontmatter.go`.

## Proposed Solution

### Phase 1: Comprehensive Exchange Rate Testing

Add golden testdata files and expand unit test tables for exchange rate scenarios.

#### 1a. Golden testdata success file

Create `testdata/eval/success/features/currency_conversion.cm`:

```calcmark
---
exchange:
  USD_EUR: 0.92
  EUR_GBP: 0.86
  GBP_JPY: 191.50
---

# Currency Conversion

## Basic conversion
price = 100 USD
price_in_euros = price in EUR

## Dollar symbol conversion
total = $200 in EUR

## Variable with conversion
budget = 5000 EUR
budget_gbp = budget in GBP

## @exchange inline assignment
@exchange.USD_GBP = 0.79
salary = 3000 USD
salary_gbp = salary in GBP
```

This file is automatically picked up by `TestEvalFilesEvaluate` in `golden_eval_test.go`. No harness changes needed.

> **Research insight**: No golden error `.cm` file. The error test harness (`TestEvalErrorFilesShouldFailEval`) does NOT read files from disk - it uses hard-coded inline expressions. Adding an error `.cm` file provides documentation value only, not automated coverage. Error behavior is better tested via `eval_test.go` table-driven tests.

#### 1b. Expand `eval_test.go` table-driven tests

Add to `TestCurrencyConversion` (targeting genuinely uncovered scenarios):

| Test case | Description | Expected |
|-----------|-------------|----------|
| `"no frontmatter at all"` | `100 USD in EUR` without any `---` | Error containing `"no exchange rate defined for USD"` |
| `"wrong pair defined"` | `EUR_JPY: 130.50` defined, `100 USD in EUR` used | Error naming the missing `USD → EUR` pair |
| `"reverse direction not auto-computed"` | `USD_EUR: 0.92` defined, `100 EUR in USD` | Error (explicit design decision, not a bug) |
| `"@exchange inline then convert"` | `@exchange.USD_EUR = 0.92` then `100 USD in EUR` | Success: `€92.00` |
| `"multiple sequential conversions"` | USD→EUR then EUR→GBP in same document | Both succeed independently |
| `"error message has correct format hint"` | Assert error contains `USD_EUR` (underscore), not `USD/EUR` (slash) | Validates the bug fix from Phase 1b |

> **Research insight (simplicity)**: The existing 7 cases already cover: basic conversion, symbol conversion, missing rate, same-currency no-op (2 variants), and variable conversion. Zero/negative rate tests are deferred since those are addressed by the security guard in Phase 1b (which rejects them at parse time).

Add to `TestFrontmatterErrors`:

| Test case | Description | Expected |
|-----------|-------------|----------|
| `"exchange rate not a number"` | `USD_EUR: "not_a_number"` | YAML type error |
| `"too many underscores"` | `USD_EUR_GBP: 0.5` | Format error: `expected format 'FROM_TO'` |
| `"NaN exchange rate"` | `USD_EUR: .nan` | Error: `not a finite number` |
| `"Inf exchange rate"` | `USD_EUR: .inf` | Error: `not a finite number` |
| `"negative exchange rate"` | `USD_EUR: -0.5` | Error: `must be positive` |
| `"zero exchange rate"` | `USD_EUR: 0` | Error: `must be positive` |

### Phase 1b: Bug Fixes Discovered During Research

These are concrete bugs found by the research agents that should be fixed alongside the tests.

#### Fix 1: Error message format hint (bug)

**File**: `impl/interpreter/unit_conversion_eval.go:71`

**Current** (wrong):
```go
return nil, fmt.Errorf("no exchange rate defined for %s → %s; add to frontmatter: exchange: { %s/%s: <rate> }",
    currency.Code, normalizedTarget, currency.Code, normalizedTarget)
```

**Fixed**:
```go
return nil, fmt.Errorf("no exchange rate defined for %s → %s; add to frontmatter: exchange:\n  %s_%s: <rate>",
    currency.Code, normalizedTarget, currency.Code, normalizedTarget)
```

Changes `/` to `_` and uses YAML-style multiline hint instead of inline `{ }` which is not the standard frontmatter format.

#### Fix 2: NaN/Inf guard on exchange rate parsing (security)

**File**: `spec/document/frontmatter.go`, in the exchange rate processing loop (~line 250-261)

Add after YAML deserialization, before `decimal.NewFromFloat`:

```go
import "math"

// Guard against YAML .nan and .inf values
if math.IsNaN(rate) || math.IsInf(rate, 0) {
    return nil, "", fmt.Errorf("exchange rate for '%s' is not a finite number", key)
}
```

> **Research insight (security)**: `yaml.v3` maps YAML `.inf` to Go `+Inf` and `.nan` to Go `NaN`. `decimal.NewFromFloat(math.NaN())` produces an undefined decimal that corrupts subsequent multiplication results silently.

#### Fix 3: Positive rate validation (security)

**File**: `spec/document/frontmatter.go`, after `decimal.NewFromFloat` conversion

```go
d := decimal.NewFromFloat(rate)
if !d.IsPositive() {
    return nil, "", fmt.Errorf("exchange rate for '%s' must be positive, got %s", key, d.String())
}
```

> **Research insight (security)**: Zero exchange rates produce `€0.00` from `$100 in EUR` with no warning — silent data corruption. Negative rates flip the sign of currency values. Neither has a valid financial meaning. Reject at parse time.

#### Fix 4: Eliminate duplicate `parseExchangeKey` (code smell)

**File**: `impl/interpreter/variables.go` (~line 58-70)

Replace the duplicated function body with a call to the spec-layer function:

```go
import "github.com/CalcMark/go-calcmark/spec/document"

func parseExchangeKey(property string) (string, string, error) {
    return document.ParseExchangeRateKey(property)
}
```

Or inline the call at the call site and remove the function entirely.

> **Research insight (patterns)**: This duplication means exchange key validation logic exists in two places with identical error messages. Any future changes to the format (e.g., supporting 3-currency keys) would need updating in both.

### Phase 2: ISO 4217 Validation (DEFERRED)

> **Research insight (simplicity + architecture)**: The simplicity reviewer correctly identifies this as YAGNI for now. The runtime error at `unit_conversion_eval.go:71` is already specific and actionable — users see `"no exchange rate defined for USD → XYZ"` the moment they try to use an undefined rate. A WARNING that fires on the frontmatter key itself would fire whether or not the rate is ever used (false positive for preparatory frontmatter). Additionally, the semantic checker currently only processes AST nodes, not frontmatter maps — wiring frontmatter diagnostics requires a new integration point.
>
> If this is desired later, the architecture strategist confirms Option C (standalone function in `spec/semantic/currency.go` called from `impl/document/evaluator.go` before `ApplyFrontmatter`) is the correct approach.

### Phase 3: Improve Ctrl+F Frontmatter Insertion

Change `insertFrontmatter()` in `cmd/calcmark/tui/editor/editing.go:750` to insert **both** an exchange rate example and a globals example. The user explicitly requested this.

**New template**:

```yaml
---
exchange:
  USD_EUR: 0.92
globals:
  my_var: 42
---
```

**Implementation changes**:

```go
// editing.go:757 - Change from globals-only to exchange + globals
fmBlock := "---\nexchange:\n  USD_EUR: 0.92\nglobals:\n  my_var: 42\n---\n"
```

> **Research insight (patterns)**: The pattern specialist noted that constructing frontmatter via `Frontmatter.Serialize()` instead of a magic string literal would be more robust. However, `Serialize()` adds `\n\n` after the closing `---` for CommonMark compatibility, and the current `insertFrontmatter` manually prepends `fmBlock + content`. Using the string literal is consistent with the existing pattern and simpler. The template is tested by `TestInsertFrontmatter`.

Cursor placement: Position on line 3 (`  USD_EUR: 0.92`) instead of line 2. Update `m.cursorLine = 3` (was `2`).

> **Research insight (learnings)**: Per the frontmatter editing solution (`docs/solutions/ui-bugs/frontmatter-editing-keyboard-dispatch-fixes.md`), cursor position after frontmatter insertion is critical because `updateCurrentLine()` uses it to determine which line to persist. The offset change from 2 to 3 is a required mechanical update, not overengineering.

**Tests to update**:
- `TestInsertFrontmatter` - verify new template content (both exchange and globals present), cursor on line 3
- `TestInsertFrontmatterAlreadyExists` - unchanged (still no-op)
- Catwalk `frontmatter_insert` test - regenerate expectations with `--rewrite`
- Add `TestInsertFrontmatterHasExchangeRate` - verify `doc.GetFrontmatter().Exchange["USD_EUR"]` exists

**Command menu update** in `command_menu.go:36`:

```go
{Name: "Insert Frontmatter", Accelerator: "Ctrl+F", Description: "Add exchange rates and globals", Category: "edit"},
```

## Acceptance Criteria

### Functional Requirements

- [x] Golden `.cm` file exists for successful currency conversions (`testdata/eval/success/features/currency_conversion.cm`)
- [x] `TestCurrencyConversion` has at least 12 test cases covering: no frontmatter, wrong pair, reverse direction, inline assignment, sequential, error format
- [x] `TestFrontmatterErrors` has at least 7 error cases covering: non-numeric rate, malformed keys, NaN, Inf, negative, zero
- [x] Error message at `unit_conversion_eval.go:71` uses `_` separator (not `/`)
- [x] NaN and Inf exchange rates rejected at parse time with clear error
- [x] Zero and negative exchange rates rejected at parse time with clear error
- [x] Duplicate `parseExchangeKey` eliminated from `impl/interpreter/variables.go` (kept as mirror due to import cycle, with comment)
- [x] Ctrl+F inserts both exchange rate and globals examples
- [x] Cursor lands on the exchange rate value line (line 2, 0-indexed) after Ctrl+F
- [x] Command menu description updated to mention exchange rates

### Non-Functional Requirements

- [x] All existing tests pass (`task test`)
- [x] Quality gates pass (`task quality`) (pre-existing staticcheck warnings in untouched files)
- [x] No backwards compatibility breaks (existing valid frontmatter documents parse identically; only NaN/Inf/negative/zero rates are newly rejected)
- [x] WASM build inherits all parse-time guards automatically (they live in `spec/`)

### Quality Gates

- [x] `task test` passes with 0 failures
- [x] `task quality` passes (pre-existing staticcheck warnings only)
- [x] Catwalk tests regenerated and passing
- [x] No new lint warnings

## Success Metrics

- Error message for "wrong currency pair" includes the exact missing pair AND correct format hint with `_` separator
- NaN/Inf/zero/negative rates produce clear parse-time errors instead of silent corruption
- Ctrl+F template gives immediate working example for currency conversion
- No duplicate exchange key parsing logic across spec/impl

## Dependencies & Prerequisites

- `spec/document/frontmatter.go` parsing and serialization are stable (covered by solutions learnings)
- Catwalk test infrastructure is in place for TUI editor
- Golden test harness in `impl/interpreter/golden_eval_test.go` automatically discovers success `.cm` files
- `math` package for NaN/Inf detection (stdlib, no new dependency)

## Risk Analysis & Mitigation

| Risk | Impact | Mitigation |
|------|--------|------------|
| Rejecting zero/negative rates breaks existing documents | Low | No valid financial use case for these values; users will get a clear error message explaining why |
| Rejecting NaN/Inf breaks YAML documents using `.nan`/`.inf` | Very Low | These are not meaningful exchange rates; the current behavior (silent corruption) is worse |
| Ctrl+F template change breaks catwalk tests | Low | Regenerate expectations with `--rewrite` |
| Frontmatter ordering sensitivity (map iteration) | High | Already solved per docs/solutions learning; use insertion-order slices |
| Error message format change breaks tests | Low | Tests asserting on the old `/` format will fail, catching the change; update assertions |

## Implementation Order

1. **Phase 1 first** - Write the golden `.cm` success file and new `eval_test.go` / `TestFrontmatterErrors` test cases. Verify they expose the format bug and NaN/Inf issues.
2. **Phase 1b second** - Fix the 4 bugs. Re-run tests to verify fixes.
3. **Phase 3 third** - TUI template change. Update tests. Regenerate catwalk expectations.
4. **Final** - `task test` and `task quality` full suite.

## References & Research

### Internal References

- Exchange rate runtime conversion: `impl/interpreter/unit_conversion_eval.go:59-82`
- Frontmatter parsing: `spec/document/frontmatter.go:193-307`
- Currency code validation: `spec/semantic/currency.go:42-73`
- TUI frontmatter insertion: `cmd/calcmark/tui/editor/editing.go:750-784`
- Command menu: `cmd/calcmark/tui/editor/command_menu.go:36`
- Environment exchange rate storage: `impl/interpreter/environment.go:80-93`
- Golden test harness: `impl/interpreter/golden_eval_test.go`
- Error golden test harness: `impl/interpreter/golden_eval_test.go:57-92` (NOTE: does not read `.cm` files)
- Existing exchange rate tests: `eval_test.go:187-283`
- Frontmatter TUI tests: `cmd/calcmark/tui/editor/frontmatter_test.go`
- Duplicate parseExchangeKey: `impl/interpreter/variables.go:58-70`

### Institutional Learnings

- **Map ordering**: Go maps are non-deterministic; frontmatter uses `exchangeKeys []string` for insertion order (docs/solutions/logic-errors/go-maps-non-deterministic-ordering-frontmatter.md)
- **Frontmatter editing**: `updateCurrentLine()` must persist edits; `Serialize()` must preserve YAML structure (docs/solutions/ui-bugs/frontmatter-editing-keyboard-dispatch-fixes.md)
- **Diagnostic line mapping**: Use `Node.GetRange()` interface, never type assertions (docs/solutions/ui-bugs/tui-editor-rendering-divider-status-bar-error-line.md)
- **State reset**: Enumerate ALL mutable fields for TUI state changes (docs/solutions/ui-bugs/ctrl-o-stale-state-and-unsaved-changes-detection.md)
- **Catwalk testing**: Required for TUI bugs; unit tests miss timing-dependent issues

### External Research

- `shopspring/decimal`: `NewFromFloat` has ~15 significant digit precision; use `NewFromString` where possible. NaN/Inf inputs produce undefined results — must guard. `IsPositive()` correctly rejects zero. Division precision defaults to 16 via global `decimal.DivisionPrecision`.
- `golang.org/x/text/currency.ParseISO`: Static table lookup, O(1), zero allocation. Safe for hot paths. Pre-filtered to 3-char uppercase ASCII in go-calcmark.
- YAML v3: Maps `.inf` → Go `+Inf`, `.nan` → Go `NaN` as valid `float64` values. Must explicitly guard.
