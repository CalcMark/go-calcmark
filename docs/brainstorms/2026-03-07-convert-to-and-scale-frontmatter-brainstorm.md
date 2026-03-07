# Document-Level Unit Conversion and Scaling via Frontmatter

**Date:** 2026-03-07
**Status:** Brainstorm

## Problem

A CalcMark recipe or engineering document written in metric should be viewable in imperial (and vice versa) without rewriting every line. Similarly, scaling a recipe from 9 to 36 servings currently requires explicit `* scale` on every ingredient — a manual process that doesn't distinguish between quantities you want to scale (flour, butter) and parameters you don't (oven temperature, bake time).

## Proposed Frontmatter Directives

### `convert_to` — Display Unit System

Convert all output quantities to a target measurement system.

```yaml
---
convert_to: imperial
---
```

Values: `si` or `imperial`.

The underlying values don't change — this is a display-layer transform. `280 grams` is still `280 grams` internally, but renders as `9.88 ounces` when `convert_to: imperial`.

### `convert_categories` — Selective Conversion

Whitelist which unit categories participate in conversion.

```yaml
---
convert_to: imperial
convert_categories: [Mass, Volume, Temperature]
---
```

Absence means "convert everything." Categories match CalcMark's existing unit taxonomy from `spec/units/canonical.go`: Mass, Volume, Temperature, Length, Area, Speed, Energy, Power.

### `scale` — Global Quantity Multiplier

Multiply all output quantities by a float.

```yaml
---
scale: 4
---
```

This is a pure numeric multiplication applied after evaluation but before display. Every quantity in the document gets multiplied by the scale factor.

### `scale_categories` — Selective Scaling

Whitelist which unit categories participate in scaling.

```yaml
---
scale: 4
scale_categories: [Mass, Volume]
---
```

Absence means "scale all non-rate quantities." This is where the real power lies for recipes: scale ingredients (Mass, Volume) but leave Temperature and Time untouched.

## How Rates Behave

**Rates are immune to `scale`** — they are ratios, not absolute quantities. `$3.50/kg` is still `$3.50/kg` regardless of batch size. The scaling happens naturally through arithmetic when a rate is multiplied by a (now-scaled) quantity.

**Rates participate in `convert_to`** — `$3.50/kg` should display as `$1.59/lb` when `convert_to: imperial`. The denominator unit converts while the numerator (currency) stays unchanged.

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
scale: 4
scale_categories: [Mass, Volume]
convert_to: imperial
convert_categories: [Mass, Volume, Temperature]
---
```

```
flour = 280 grams        # displays as 39.5 ounces (scaled 4x, then converted)
milk = 160 ml             # displays as 2.67 cups (scaled 4x, then converted)
oven = 220 celsius        # displays as 428 fahrenheit (converted only, not scaled)
bake_time = 15 minutes    # displays as 15 minutes (neither scaled nor converted)
```

## Open Questions

1. **What units does `imperial` map to for each category?** For Mass, is the target `ounces` or `pounds`? CalcMark's auto-scaling display logic already handles this (small masses → oz, large → lb), so the answer is probably "convert to the imperial family and let auto-scaling pick the readable unit."

2. **Interaction with explicit `in` conversions.** If a user writes `flour in ounces` and `convert_to: si` is set, does the explicit conversion win? Probably yes — explicit `in` should override the document-level directive.

3. **Should `scale` create a visible variable?** Could be useful to reference: `batch_flour = flour * @scale`. Or is it purely implicit?

4. **Compound categories.** Speed is `Length/Time`. If `scale_categories: [Length]` is set, does a speed value get its numerator scaled? Probably not — compound units should be treated as their own category, not decomposed.

5. **What about arbitrary units?** `5 eggs` has no category in the canonical unit system. Should `scale` apply to arbitrary units by default? A recipe user would expect `5 eggs` to become `20 eggs` when `scale: 4`. Probably yes — arbitrary units should participate in scaling unless excluded.
