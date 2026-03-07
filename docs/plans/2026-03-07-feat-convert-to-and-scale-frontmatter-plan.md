---
title: "feat: Add convert_to and scale frontmatter directives"
type: feat
status: active
date: 2026-03-07
---

# Add `convert_to` and `scale` Frontmatter Directives

## Overview

Two new frontmatter directives that apply document-level transforms to output values:

- **`convert_to`** — convert displayed quantities between measurement systems (`si` / `imperial`)
- **`scale`** — multiply displayed quantities by a float factor

Both are post-evaluation display transforms. The underlying evaluation environment is unchanged — only the rendered output is affected. Each directive supports a scalar shorthand and a map form with `unit_categories` filtering.

## Problem Statement

A CalcMark recipe written in metric requires manual `in` conversions on every line to display in imperial. Scaling a recipe from 1x to 4x requires explicit `* scale` on every ingredient, with no way to distinguish scalable quantities (flour, butter) from fixed parameters (oven temperature, bake time). These are document-level concerns that should be declarative, not line-by-line.

## Proposed Solution

### YAML Syntax

```yaml
# Simple forms — apply to all eligible quantities
---
scale: 4
convert_to: imperial
---

# Map forms — apply only to selected unit categories
---
scale:
  factor: 4
  unit_categories: [Mass, Volume]
convert_to:
  system: imperial
  unit_categories: [Mass, Volume, Temperature]
---
```

### System Mapping

| `convert_to` value | Matches canonical.go System values |
|---|---|
| `imperial` | `US_Customary`, `Imperial` |
| `si` | `SI` |
| _(unaffected)_ | `CGS`, `International`, `Nautical` |

Units whose System is not matched by the target are left unchanged. No error, no warning — they pass through silently.

### Application Order

1. Evaluate document normally
2. Apply `scale` (multiply quantities in matching categories)
3. Apply `convert_to` (convert units in matching categories to target system)
4. Format for display

### Type Behavior Matrix

| Type | `scale` | `convert_to` |
|---|---|---|
| Quantity (canonical unit) | Yes, if category matches | Yes, if category matches and target unit exists |
| Quantity (arbitrary unit) | Yes, by default | No (no system mapping) |
| Rate | No (immune) | Yes (convert Amount's unit, not time denominator) |
| Currency | No | No |
| Duration | No | No |
| Number (unitless) | No | No |
| Boolean | No | No |
| Date/Time | No | No |

### Default Category Exclusions for `scale`

When no `unit_categories` is specified, `scale` applies to all Quantity types **except Temperature**. Scaling temperature is physically meaningless (`220°C * 4 = 880°C` is nonsensical). Users can explicitly include Temperature via the map form.

### Interaction Rules

- **Explicit `in` overrides `convert_to`**: If `IsExplicit = true` (set by the `in` keyword), `convert_to` is skipped. The user's explicit unit choice wins.
- **`scale` still applies to explicit `in` results**: Scale represents "I want 4x of everything," which is orthogonal to unit choice. `280 grams in ounces` with `scale: 4` displays `1120 grams in ounces` = `~39.5 ounces`.
- **`as napkin` / `as precise`**: Applied after both transforms. Order: eval → scale → convert → napkin/precise rounding.
- **`exchange_rates`**: Unaffected. Currency conversion is a separate mechanism.
- **Units with no target**: If `convert_to: imperial` is active but a unit has no imperial equivalent (e.g., joules → no BTU in canonical.go), the unit is left unchanged.

### Validation Rules

| Field | Rule |
|---|---|
| `convert_to` (scalar) | Must be `"si"` or `"imperial"` |
| `convert_to.system` | Must be `"si"` or `"imperial"` |
| `scale` (scalar) | Must be a positive finite number (> 0) |
| `scale.factor` | Must be a positive finite number (> 0) |
| `unit_categories` | Case-insensitive. Each must match a canonical category: Mass, Volume, Temperature, Length, Area, Speed, Energy, Power. Invalid category → validation error. |

### JSON Output

Post-transform values appear in the `value` display string. Pre-transform values are preserved in `numeric_value` and `unit` for backward compatibility:

```json
{
  "source": "flour = 280 grams",
  "value": "39.5 ounces",
  "type": "quantity",
  "numeric_value": 280,
  "unit": "grams",
  "variable": "flour"
}
```

The display `value` reflects scale + convert_to. The `numeric_value` and `unit` reflect the raw evaluation result. Programmatic consumers that assert on `numeric_value` are unaffected.

## Technical Approach

### Architecture

The key architectural change is promoting unit conversion factors from `impl/interpreter/unit_library.go` to a shared location in `spec/units/conversion.go`. This allows both the interpreter (for `in` keyword) and the new transform layer (for `convert_to`) to use the same conversion math without violating the dependency rule (spec never depends on impl).

```
spec/units/canonical.go     — unit definitions, system/category metadata
spec/units/conversion.go    — NEW: conversion factors and convert() function
spec/document/frontmatter.go — parse scale/convert_to from YAML

impl/interpreter/            — uses spec/units for `in` keyword (refactored)

format/transform/            — NEW: post-eval transform using spec/units
format/*_formatter.go        — apply transform before display
```

### Implementation Phases

#### Phase 1: Spec Layer — Frontmatter Parsing

Add `scale` and `convert_to` to the frontmatter parser with full validation.

**Files:**
- `spec/document/frontmatter.go` — add to `reservedKeys`, add `ScaleConfig`/`ConvertToConfig` structs, add fields to `Frontmatter` and `frontmatterYAML`, parse both scalar and map forms, validate all fields, serialize support, rawSource invalidation on programmatic modification
- `spec/document/frontmatter_test.go` — table-driven tests for parsing, validation, serialization, round-trips

**Config structs:**

```go
type ScaleConfig struct {
    Factor         decimal.Decimal
    UnitCategories []string // empty = all (except Temperature)
}

type ConvertToConfig struct {
    System         string   // "si" or "imperial"
    UnitCategories []string // empty = all
}
```

**Acceptance criteria:**
- [ ] `scale: 4` parses to `ScaleConfig{Factor: 4, UnitCategories: nil}`
- [ ] `scale: {factor: 4, unit_categories: [Mass, Volume]}` parses correctly
- [ ] `convert_to: imperial` parses to `ConvertToConfig{System: "imperial", UnitCategories: nil}`
- [ ] `convert_to: {system: si, unit_categories: [Mass]}` parses correctly
- [ ] `unit_categories` matching is case-insensitive
- [ ] Invalid system value → error with helpful message
- [ ] Invalid category → error listing valid categories
- [ ] `scale: 0` → error ("must be positive")
- [ ] `scale: -1` → error
- [ ] `Serialize()` reconstructs both directives
- [ ] `rawSource` cleared on programmatic modification
- [ ] Both directives coexist with `exchange:` and `globals:`

#### Phase 2: Spec Layer — Shared Conversion Factors

Extract unit conversion factors into `spec/units/` so both the interpreter and the transform layer can use them.

**Files:**
- `spec/units/conversion.go` — NEW: `Convert(value decimal.Decimal, fromUnit, toUnit string) (decimal.Decimal, error)`, `GetSystemForUnit(unit string) string`, `GetDefaultTargetUnit(unit string, targetSystem string) string`, `CategoryForUnit(unit string) string`
- `spec/units/conversion_test.go` — NEW: test all cross-system conversions
- `impl/interpreter/unit_library.go` — refactor to delegate to `spec/units/conversion.go`
- `impl/interpreter/unit_conversion.go` — update `convertQuantity()` to use shared conversion

**Acceptance criteria:**
- [ ] `Convert(280, "grams", "ounces")` returns `~9.88`
- [ ] `Convert(220, "celsius", "fahrenheit")` returns `428` (affine, not linear)
- [ ] `GetSystemForUnit("grams")` returns `"SI"`
- [ ] `GetSystemForUnit("ounces")` returns `"US_Customary"`
- [ ] `GetDefaultTargetUnit("grams", "imperial")` returns a unit in the imperial mass family
- [ ] `CategoryForUnit("grams")` returns `"Mass"`
- [ ] `CategoryForUnit("eggs")` returns `""` (arbitrary unit)
- [ ] Existing `in` keyword behavior is unchanged (no regressions)
- [ ] All existing unit conversion tests pass

#### Phase 3: Format Layer — Transform Pipeline

New transform package that applies `scale` and `convert_to` to evaluation results.

**Files:**
- `format/transform/transform.go` — NEW: `Apply(result types.Type, scale *ScaleConfig, convertTo *ConvertToConfig) types.Type`
- `format/transform/transform_test.go` — NEW: unit tests for all type × directive combinations
- `format/align.go` or `format/formatter.go` — integrate transform into the format pipeline, applied after `AlignResults` but before `display.Formatter`

**Transform logic:**

```go
func Apply(result types.Type, scale *ScaleConfig, convertTo *ConvertToConfig) types.Type {
    switch v := result.(type) {
    case *types.Quantity:
        v = applyScale(v, scale)
        v = applyConvertTo(v, convertTo)
        return v
    case *types.Rate:
        // immune to scale
        v = applyConvertToRate(v, convertTo) // convert Amount's unit
        return v
    default:
        return result // Currency, Number, Duration, Boolean, Date — unchanged
    }
}
```

**Acceptance criteria:**
- [ ] `Apply(280g, scale:4, nil)` → `1120g` (displayed as `1.12 kg`)
- [ ] `Apply(280g, nil, imperial)` → `~9.88 ounces`
- [ ] `Apply(280g, scale:4, imperial)` → `~39.5 ounces` (scale then convert)
- [ ] `Apply(220°C, scale:4, nil)` → `220°C` (temperature excluded from scale by default)
- [ ] `Apply(220°C, nil, imperial)` → `428°F`
- [ ] `Apply($100, scale:4, imperial)` → `$100` (currency immune)
- [ ] `Apply(3 days, scale:4, nil)` → `3 days` (duration immune)
- [ ] `Apply(5 eggs, scale:4, nil)` → `20 eggs` (arbitrary units scale)
- [ ] `Apply(100 km/h, nil, imperial)` → `~62.14 mph` (rate: convert amount)
- [ ] `Apply(100 km/h, scale:4, nil)` → `100 km/h` (rate: immune to scale)
- [ ] Explicit quantities (`IsExplicit = true`) skip `convert_to`
- [ ] `scale` with `unit_categories: [Mass]` only scales Mass quantities
- [ ] `convert_to` with `unit_categories: [Temperature]` only converts Temperature

#### Phase 4: Formatter Integration

Wire the transform into all output formatters.

**Files:**
- `format/text_formatter.go` — apply transform, show directives in verbose mode
- `format/json_formatter.go` — apply transform to `value` display string; preserve pre-transform `numeric_value`/`unit`; add `scale`/`convert_to` to `JSONFrontmatter`
- `format/html_formatter.go` — apply transform
- `format/md_formatter.go` — apply transform
- `format/cm_formatter.go` — apply transform

**Acceptance criteria:**
- [ ] All formatters produce consistent transformed output
- [ ] JSON `numeric_value` and `unit` remain pre-transform
- [ ] JSON `value` string reflects post-transform
- [ ] JSON frontmatter section includes `scale` and `convert_to` when present
- [ ] Text verbose mode displays active directives
- [ ] Backward compatible: documents without these directives produce identical output

#### Phase 5: Golden Tests and Examples

End-to-end validation with golden test files.

**Files:**
- `testdata/eval/success/features/scale_basic.cm`
- `testdata/eval/success/features/scale_categories.cm`
- `testdata/eval/success/features/convert_to_imperial.cm`
- `testdata/eval/success/features/convert_to_si.cm`
- `testdata/eval/success/features/scale_convert_combined.cm`
- `testdata/eval/errors/features/scale_invalid.cm`
- `testdata/eval/errors/features/convert_to_invalid.cm`
- `testdata/examples/recipe-scaling.cm` — add a variant or update the example to demonstrate the directives
- `spec/features/registry.go` — register `scale` and `convert_to` as features

**Acceptance criteria:**
- [ ] All golden test files parse and evaluate without errors
- [ ] Error test files produce expected diagnostics
- [ ] `task test` passes
- [ ] `task quality` passes

#### Phase 6: TUI Support

Ensure the TUI editor handles these directives correctly.

**Files:**
- TUI frontmatter rendering — ensure `scale` and `convert_to` lines render in the Globals panel
- Catwalk test data — new test files for frontmatter editing with these directives

**Acceptance criteria:**
- [ ] Typing `scale: 4` in frontmatter live-updates all results
- [ ] Typing `convert_to: imperial` live-updates all results
- [ ] Invalid intermediate states (e.g., `scale: `) show error without crash
- [ ] Catwalk tests verify key sequences for editing these directives

## Dependencies & Risks

**Risks:**
- **Conversion factor extraction (Phase 2)** is the riskiest phase — refactoring `unit_library.go` could introduce regressions in the existing `in` keyword. Mitigated by running full test suite after extraction.
- **Display pipeline threading** — ensuring all 5 formatters apply the transform consistently. Mitigated by shared transform function called from a common code path.
- **Missing target units** — some categories have no imperial/SI counterpart in `canonical.go` (e.g., Energy has no BTU). These silently pass through, which is correct but could surprise users. Document this limitation.

**Not in scope (v1):**
- Per-category target unit override (`convert_to: {Mass: pounds}`)
- Visual indicator in TUI for active transforms (e.g., `[x4]` badge)
- `scale` as a visible variable in the namespace
- BTU or other missing units in `canonical.go`
- Inline `@scale` / `@convert_to` assignment syntax

## References

### Internal
- Brainstorm: `docs/brainstorms/2026-03-07-convert-to-and-scale-frontmatter-brainstorm.md`
- Frontmatter parsing: `spec/document/frontmatter.go:30-355`
- Unit definitions: `spec/units/canonical.go:23-477`
- Unit conversion: `impl/interpreter/unit_conversion.go:38-71`
- Unit library: `impl/interpreter/unit_library.go:12-23`
- Display auto-scaling: `format/display/normalize.go:273-336`
- Display formatter: `format/display/formatter.go:72-96`
- IsExplicit pattern: `docs/solutions/ui-bugs/display-formatter-overrides-explicit-unit-conversion.md`
- Frontmatter ordering: `docs/solutions/logic-errors/go-maps-non-deterministic-ordering-frontmatter.md`
- Frontmatter validation: `docs/solutions/logic-errors/exchange-rate-frontmatter-validation.md`
- Frontmatter round-trips: `docs/solutions/ui-bugs/frontmatter-editing-keyboard-dispatch-fixes.md`
