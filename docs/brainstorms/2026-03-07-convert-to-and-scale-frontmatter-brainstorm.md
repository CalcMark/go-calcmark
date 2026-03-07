# Document-Level Unit Conversion and Scaling via Frontmatter

**Date:** 2026-03-07
**Status:** Brainstorm

## Problem

A CalcMark recipe or engineering document written in metric should be viewable in imperial (and vice versa) without rewriting every line. Similarly, scaling a recipe from 9 to 36 servings currently requires explicit `* scale` on every ingredient — a manual process that doesn't distinguish between quantities you want to scale (flour, butter) and parameters you don't (oven temperature, bake time).

## Proposed Frontmatter Directives

Two directives, each with a scalar shorthand and a map form.

### `convert_to` — Display Unit System

Convert output quantities to a target measurement system. The underlying values don't change — this is a display-layer transform. `280 grams` is still `280 grams` internally, but renders as `9.88 ounces` when converting to imperial.

**Simple form** — convert everything:

```yaml
---
convert_to: imperial
---
```

**Map form** — convert only selected categories:

```yaml
---
convert_to:
  system: imperial
  unit_categories: [Mass, Volume, Temperature]
---
```

Values for `system`: `si` or `imperial`.

### `scale` — Global Quantity Multiplier

Multiply output quantities by a float, applied after evaluation but before display.

**Simple form** — scale everything:

```yaml
---
scale: 4
---
```

**Map form** — scale only selected categories:

```yaml
---
scale:
  factor: 4
  unit_categories: [Mass, Volume]
---
```

### Syntax Pattern

Scalar value means "apply to all quantities." Map value means "here are the options." This is a common YAML pattern — consistent with how `exchange_rates` already works as a nested structure. The `unit_categories` key is shared between both directives.

Categories match CalcMark's existing unit taxonomy from `spec/units/canonical.go`: Mass, Volume, Temperature, Length, Area, Speed, Energy, Power.

Absence of `unit_categories` means "apply to all non-rate quantities."

## How Rates Behave

**Rates are immune to `scale`** — they are ratios, not absolute quantities. `$3.50/kg` is still `$3.50/kg` regardless of batch size. The scaling happens naturally through arithmetic when a rate is multiplied by a (now-scaled) quantity.

**Rates participate in `convert_to`** — `$3.50/kg` should display as `$1.59/lb` when converting to imperial. The denominator unit converts while the numerator (currency) stays unchanged.

## Currency Is Untouched

Neither `convert_to` nor `scale` affects currency values. Currency conversion already has its own mechanism via the `exchange_rates` frontmatter directive. Mixing measurement-system conversion with currency conversion would muddy the semantics.

## Application Order

When both directives are present:

1. Evaluate the document normally
2. Apply `scale` (multiply quantities in matching categories)
3. Apply `convert_to` (convert units in matching categories)

Scale before convert — you want to scale `280 grams * 4 = 1120 grams`, then convert `1120 grams` to `39.5 ounces`. Not convert first then scale.

## Full Example

A scone recipe written in metric, scaled to 4x and displayed in imperial:

```yaml
---
scale:
  factor: 4
  unit_categories: [Mass, Volume]
convert_to:
  system: imperial
  unit_categories: [Mass, Volume, Temperature]
---
```

```
flour = 280 grams        # displays as 39.5 ounces (scaled 4x, then converted)
milk = 160 ml             # displays as 2.67 cups (scaled 4x, then converted)
oven = 220 celsius        # displays as 428 fahrenheit (converted only, not scaled)
bake_time = 15 minutes    # displays as 15 minutes (neither scaled nor converted)
```

And the simple forms for a quick "show me everything in metric, doubled":

```yaml
---
scale: 2
convert_to: si
---
```

## Design Decisions

1. **`imperial`/`si` map to unit families, not specific units.** `convert_to: imperial` converts to the imperial family (Mass → ounces/pounds, Volume → cups/gallons, etc.) and lets CalcMark's existing auto-scaling display logic pick the readable unit within that family. No explicit per-category target mapping is needed.

2. **Explicit `in` overrides `convert_to`.** If a user writes `flour in ounces`, the explicit conversion wins over any document-level directive. Explicit `in` is a stronger signal of intent.

3. **`scale` is purely implicit — no visible variable.** It's a display transform, not a variable in the namespace. If the user wants to reference the scale factor in formulas, they define their own variable (`scale = 4`). This avoids shadowing ambiguity between frontmatter directives and user-defined variables.

4. **Compound units are atomic categories.** Speed is `Length/Time`, but `unit_categories: [Length]` does NOT decompose Speed and scale its numerator. Compound units (Speed, Energy, Power) are their own categories. Include them explicitly if you want them affected.

5. **Arbitrary units participate in `scale` by default.** `5 eggs` becomes `20 eggs` with `scale: 4`. Arbitrary units have no canonical category, so they're included unless the user provides a `unit_categories` filter that restricts to specific canonical categories. Plain numbers (no unit) also participate in scaling by default.
