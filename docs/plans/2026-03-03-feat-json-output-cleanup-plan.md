---
title: "feat: Clean up JSON output structure"
type: feat
status: completed
date: 2026-03-03
issue: https://github.com/CalcMark/go-calcmark/issues/11
brainstorm: docs/brainstorms/2026-03-03-json-output-cleanup-brainstorm.md
---

# feat: Clean up JSON output structure

## Overview

Restructure the JSON output from `cm convert --to=json` to remove redundant block-level fields and replace the opaque `raw_value` string with structured, type-aware decomposition fields. This gives consumers (syntax highlighters, external tools) rich type information and machine-readable numeric values alongside the existing locale-formatted display string.

## Problem Statement

The current JSON output has three issues:

1. **Redundancy:** `JSONBlock.Output` (last value only) and `JSONBlock.Variables` (list of names) duplicate data already in `results[]`.
2. **Inconsistency:** `Output` uses raw `String()` while `Value` uses locale-formatted `Format()` — consumers see different representations for the same value.
3. **Opacity:** `raw_value` is a flat string (`"$7000.00"`) that consumers must parse to extract the numeric value and unit. No type discrimination is available.

## Proposed Solution

### Structural changes

**Remove from `JSONBlock`:**
- `Output` field (redundant with `results[].value`)
- `Variables` field (redundant with `results[].variable`)

**Replace on `JSONResult`:**
- Remove `raw_value` (opaque string)
- Add `type` (string, always present) — the CalcMark type name
- Add `numeric_value` (`*float64`, omitempty) — machine-readable numeric value, nil for non-numeric types
- Add `unit` (string, omitempty) — unit identifier, empty for unitless types
- Add `is_approximate` (bool, omitempty) — true for napkin estimates (Quantity-only in v1)
- Add `error` (string, omitempty) — per-result error message for failed statements
- Add `date_value` (string, omitempty) — ISO 8601 date string, Date type only

**Keep unchanged:**
- `JSONBlock.Error` — block-level errors (parse failures, semantic errors)
- `JSONBlock.Diagnostics` — structured diagnostic details
- `JSONBlock.Type`, `Source`, `HTML`
- `JSONResult.Source`, `Value`, `Variable`

### New JSONResult struct

```go
type JSONResult struct {
    Source        string   `json:"source"`
    Value         string   `json:"value"`                      // locale-formatted display string
    Type          string   `json:"type"`                       // "number", "currency", "quantity", "rate", "duration", "date", "time", "boolean"
    NumericValue  *float64 `json:"numeric_value,omitempty"`    // nil for non-numeric types (Date, Time, Boolean)
    Unit          string   `json:"unit,omitempty"`             // ISO 4217 for currency, compound for rate, canonical for duration
    DateValue     string   `json:"date_value,omitempty"`       // ISO 8601 date, Date type only
    IsApproximate bool     `json:"is_approximate,omitempty"`   // true for napkin estimates (Quantity only in v1)
    Error         string   `json:"error,omitempty"`            // per-statement error message
    Variable      string   `json:"variable,omitempty"`
}
```

### New JSONBlock struct

```go
type JSONBlock struct {
    Type        string           `json:"type"`           // "calculation" or "text"
    Source      []string         `json:"source"`
    Results     []JSONResult     `json:"results,omitempty"`
    Error       string           `json:"error,omitempty"`
    Diagnostics []JSONDiagnostic `json:"diagnostics,omitempty"`
    HTML        string           `json:"html,omitempty"`
}
```

## Technical Approach

### Type decomposition mapping

The JSON formatter needs a type switch on `types.Type` to populate the new fields. This parallels the existing type switch in `format/display/formatter.go:44`.

| CalcMark Type | `type` | `numeric_value` | `unit` | `date_value` | Notes |
|---|---|---|---|---|---|
| `*types.Number` | `"number"` | `v.Value.InexactFloat64()` | — | — | |
| `*types.Currency` | `"currency"` | `v.Value.InexactFloat64()` | `v.Code` (ISO 4217) | — | Use `Code` not `Symbol` |
| `*types.Quantity` | `"quantity"` | `v.Value.InexactFloat64()` | `v.Unit` | — | Check `v.IsNapkin` |
| `*types.Rate` | `"rate"` | `v.Amount.Value.InexactFloat64()` | `v.Amount.Unit + "/" + abbreviateTimeUnit(v.PerUnit)` | — | Compound unit string |
| `*types.Duration` | `"duration"` | `v.Value.InexactFloat64()` | `v.Unit` | — | As-stored (plural/singular) |
| `*types.Date` | `"date"` | — | — | `v.Time.Format("2006-01-02")` | `value` = locale-formatted long form |
| `*types.Time` | `"time"` | — | — | — | `value` already machine-readable |
| `*types.Boolean` | `"boolean"` | — | — | — | |
| `nil` (error) | — | — | — | — | Only `source` + `error` populated |

### Error handling

The evaluator stops at the first error in a block. For a block like:
```
x = 10
y = unknown + 1
z = x + 5
```

Results: `x` succeeds, `y` fails, `z` is never evaluated.

**JSON output:**
- `x` → full result with `type`, `numeric_value`, etc.
- `y` → `{ "source": "y = unknown + 1", "error": "undefined variable: unknown_var" }`
- `z` → omitted from `results[]` (never evaluated)

Block-level `error` remains for block-wide failures. Per-result `error` is for statement-level failures within a partially-evaluated block.

### `numeric_value` precision

Uses `*float64` pointer type:
- `nil` = field absent from JSON (non-numeric types)
- `0.0` = explicitly present (e.g., `x = 0`)

CalcMark uses `shopspring/decimal` for exact arithmetic. `InexactFloat64()` converts to float64 for JSON serialization. Precision is limited to ~15 significant digits. This is acceptable for typical calculations; edge cases with extremely large integers will lose trailing precision. Documented in user-facing docs.

### `is_approximate` scope

Only `types.Quantity` has `IsNapkin` today. `Currency` and `Rate` do not. In v1, `is_approximate` only appears on Quantity results. Extending to Currency/Rate is a future enhancement (requires adding `IsNapkin` to those structs in `spec/types/`).

## Implementation Phases

### Phase 1: Update structs and type-switch logic

**Files:**
- `format/json_formatter.go:21-61` — Update `JSONBlock` (remove `Output`, `Variables`), update `JSONResult` (remove `RawValue`, add new fields)
- `format/json_formatter.go:64-145` — Add type-switch in the result-building loop to populate `Type`, `NumericValue`, `Unit`, `DateValue`, `IsApproximate`

**Implementation:**
1. Update `JSONBlock` struct: remove `Output string` and `Variables []string` fields
2. Update `JSONResult` struct per the new definition above
3. In the `Format()` method, replace the current `jr.RawValue = stmt.Result.String()` line (around line 111) with a type switch:

```go
func populateResult(jr *JSONResult, result types.Type) {
    switch v := result.(type) {
    case *types.Number:
        jr.Type = "number"
        f := v.Value.InexactFloat64()
        jr.NumericValue = &f
    case *types.Currency:
        jr.Type = "currency"
        f := v.Value.InexactFloat64()
        jr.NumericValue = &f
        jr.Unit = v.Code
    case *types.Quantity:
        jr.Type = "quantity"
        f := v.Value.InexactFloat64()
        jr.NumericValue = &f
        jr.Unit = v.Unit
        if v.IsNapkin {
            jr.IsApproximate = true
        }
    case *types.Rate:
        jr.Type = "rate"
        f := v.Amount.Value.InexactFloat64()
        jr.NumericValue = &f
        jr.Unit = v.CompoundUnit()
    case *types.Duration:
        jr.Type = "duration"
        f := v.Value.InexactFloat64()
        jr.NumericValue = &f
        jr.Unit = v.Unit
    case *types.Date:
        jr.Type = "date"
        jr.DateValue = v.Time.Format("2006-01-02")
    case *types.Time:
        jr.Type = "time"
    case *types.Boolean:
        jr.Type = "boolean"
    }
}
```

4. Remove the block-level `Output` and `Variables` population (around lines 129-133)
5. Add `CompoundUnit() string` method to `*Rate` in `spec/types/rate.go` — returns `Amount.Unit + "/" + abbreviateTimeUnit(PerUnit)`
6. Handle per-result errors: when `stmt.Result == nil` and the statement is non-blank/non-result-line, this means the evaluator failed on or before this statement. Populate `jr.Error` from `block.Error().Error()`. Since the evaluator stops at the first error, at most one result will have an `error` field — the last entry in `results[]` before the evaluator stopped. Do not call `populateResult` for error entries (no `type` field).

**Rate compound unit access:** `abbreviateTimeUnit` is unexported in `spec/types/rate.go:165`. Add a public `CompoundUnit() string` method on `*Rate` that returns `Amount.Unit + "/" + abbreviateTimeUnit(PerUnit)`. This keeps unit logic in the type package and gives the JSON formatter a clean API to call.

### Phase 2: Update tests

**File:** `format/json_formatter_test.go`

Tests to update:
- [x] `TestJSONFormatterSimple` — verify new fields present, old fields absent
- [x] `TestJSONFormatterStructure` — verify `type` and `unit` fields for Currency
- [x] `TestJSONFormatterPerStatementResults` — verify each result has `type` field
- [x] `TestJSONFormatterVariableResultMapping` — verify `numeric_value` and `unit` on Currency results
- [x] `TestJSONFormatterRawValue` → **rename** to `TestJSONFormatterNumericValue` — verify `numeric_value` is a number, `unit` is ISO 4217 code
- [x] `TestJSONFormatterCurrencyResults` — verify `unit: "USD"` not `unit: "$"`

New tests to add:
- [x] `TestJSONFormatterNumberType` — plain number: `type: "number"`, `numeric_value` present, no `unit`
- [x] `TestJSONFormatterQuantityType` — quantity with unit: `type: "quantity"`, `unit: "kg"`
- [x] `TestJSONFormatterRateType` — rate: `type: "rate"`, compound `unit: "MB/s"`
- [x] `TestJSONFormatterDurationType` — duration: `type: "duration"`, `unit: "hours"`
- [x] `TestJSONFormatterDateType` — date: `type: "date"`, `date_value` ISO 8601, no `numeric_value` (Note: CalcMark date syntax incompatible with test harness; covered by type-switch unit tests)
- [x] `TestJSONFormatterTimeType` — time: `type: "time"`, no `numeric_value` (Note: CalcMark time syntax incompatible with test harness; covered by type-switch unit tests)
- [x] `TestJSONFormatterBooleanType` — boolean: `type: "boolean"`, no `numeric_value`
- [x] `TestJSONFormatterNapkinEstimate` — napkin quantity: `is_approximate: true`
- [x] `TestJSONFormatterZeroNumericValue` — `x = 0`: `numeric_value` is explicitly `0`, not absent
- [x] `TestJSONFormatterPerResultError` — partial block failure: first result ok, second has `error`
- [x] `TestJSONFormatterBlockFieldsRemoved` — verify `output` and `variables` keys absent from JSON
- [x] `TestJSONFormatterLocaleNumericValue` — verify `numeric_value` is locale-independent even with de-DE locale

**Gotchas from institutional learnings:**
- Use separate `resultIdx` counter when iterating results (blank line alignment bug)
- Test with locale-specific formatter to verify `numeric_value`/`unit` are locale-independent
- `JSONDocument` struct should be updated in tests to match new field set

### Phase 3: Update documentation

**Files:**
- `site/content/docs/user-guide.md:119-136` — Update JSON output section with new field examples, replace `raw_value` references
- `site/content/docs/configuration.md:230-248` — Update "JSON Output and Locale" section, replace `raw_value` table with new field descriptions
- `site/content/docs/cli-reference.md:105,151-153` — Update JSON format description, remove `RawResult` reference, add `type`/`numeric_value`/`unit` descriptions

**Documentation changes:**
- Replace all `raw_value` references with `type`, `numeric_value`, `unit`
- Add type mapping table showing all 8 types
- Add `date_value` field documentation
- Note that `numeric_value` uses IEEE 754 float64 (~15 significant digits)
- Update the "Use `raw_value` for programmatic consumption" guidance to "Use `type` for dispatch, `numeric_value` + `unit` for computation, `value` for display"

## Acceptance Criteria

- [x] `JSONBlock` no longer emits `output` or `variables` fields
- [x] `JSONResult` no longer emits `raw_value` field
- [x] Every successful result has a `type` field matching one of: `number`, `currency`, `quantity`, `rate`, `duration`, `date`, `time`, `boolean`
- [x] Numeric types have `numeric_value` as a JSON number (including zero)
- [x] Currency results have `unit` set to ISO 4217 code (e.g., `"USD"`, not `"$"`)
- [x] Rate results have compound `unit` (e.g., `"MB/s"`)
- [x] Date results have `date_value` in ISO 8601 format
- [x] Napkin Quantity results have `is_approximate: true`
- [x] Failed results have `source` + `error`, no type/numeric_value
- [x] `value` field remains locale-formatted for all types
- [x] `numeric_value`, `unit`, `date_value`, `is_approximate` are all locale-independent
- [x] All existing tests updated, all new type-specific tests pass
- [x] `task test` passes (full suite)
- [x] `task quality` passes
- [x] User-guide, configuration, and CLI reference docs updated

## Dependencies & Risks

**Evaluator stops at first error:** Per-result `error` only works for the *first* failing line. Subsequent lines are never evaluated and won't appear in `results[]`. This is a known limitation — the evaluator would need to be modified for full per-line error reporting, which is out of scope.

**Float64 precision:** `InexactFloat64()` loses precision beyond ~15 significant digits. Acceptable for typical use; documented as a known limitation.

## References

### Internal
- `format/json_formatter.go` — JSON formatter implementation
- `format/json_formatter_test.go` — existing tests
- `format/display/formatter.go:44` — canonical type switch pattern
- `format/align.go` — `AlignResults()` shared alignment logic
- `spec/types/` — all type definitions
- `docs/solutions/ui-bugs/locale-formatting-bypass-in-tui.md` — locale wiring gotcha
- `docs/solutions/logic-errors/go-maps-non-deterministic-ordering-frontmatter.md` — map ordering gotcha

### External
- [Issue #11](https://github.com/CalcMark/go-calcmark/issues/11) — Feature request
- [Brainstorm](docs/brainstorms/2026-03-03-json-output-cleanup-brainstorm.md) — Design decisions
