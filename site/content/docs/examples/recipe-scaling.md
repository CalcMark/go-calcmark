---
title: "Recipe Scaling"
summary: "Scale a bread recipe from 1 to 4 loaves, convert metric to US customary, and estimate cost per loaf."
weight: 30
calcmark_build: progressive
---

You have a European bread recipe in metric units. You need 4 loaves for a dinner party. This walkthrough scales the recipe, converts units, handles temperature, estimates timing, and calculates cost per loaf -- all in one CalcMark document.

The complete CalcMark file is available at {{< repo-file path="testdata/examples/recipe-scaling.cm" >}}.

---

## Original Recipe

Start by capturing the base recipe for a single loaf in grams. Then compute the hydration ratio -- a key baker's percentage that tells you how wet the dough is.

```calcmark
original_flour_g = 500
original_water_g = 350
original_salt_g = 10
original_yeast_g = 7

Hydration ratio (baker's percentage):

hydration_pct = original_water_g / original_flour_g * 100
```

`hydration_pct` evaluates to `70` -- a 70% hydration dough, which produces a nice open crumb. All values here are plain numbers since we're doing manual unit bookkeeping with variable names.

**CalcMark features:** Variable assignment; arithmetic expressions; markdown prose between calculations.

---

## Scaling Factor

You want 4 loaves instead of 1. Define a scale factor and let CalcMark carry it through.

```calcmark
original_yield = 1
target_yield = 4
scale = target_yield / original_yield
```

`scale` is `4`. Simple division, but naming the value makes the intent clear and lets you tweak the yield in one place.

**CalcMark features:** Derived calculations; descriptive variable names for self-documenting math.

---

## Scaled Quantities

Multiply every ingredient by the scale factor. The recipe stays in metric for now.

```calcmark
scaled_flour_g = original_flour_g * scale
scaled_water_g = original_water_g * scale
scaled_salt_g = original_salt_g * scale
scaled_yeast_g = original_yeast_g * scale
```

You get `2000` g flour, `1400` g water, `40` g salt, and `28` g yeast. Each line references both the original amount and the scale factor, so changing either propagates automatically.

**CalcMark features:** Variable references across sections; multiplication for batch scaling.

---

## Convert to US Customary

Not everyone owns a kitchen scale. Define conversion factors and divide to get cups and teaspoons.

```calcmark
grams_per_cup_flour = 120
grams_per_cup_water = 237
grams_per_tsp_salt = 6
grams_per_tsp_yeast = 3

flour_cups = scaled_flour_g / grams_per_cup_flour
water_cups = scaled_water_g / grams_per_cup_water
salt_tsp = scaled_salt_g / grams_per_tsp_salt
yeast_tsp = scaled_yeast_g / grams_per_tsp_yeast
```

Results: roughly `16.67` cups of flour, `5.91` cups of water, `6.67` tsp salt, and `9.33` tsp yeast. These are manual conversions using plain arithmetic -- useful when your target units aren't in CalcMark's built-in unit system.

**CalcMark features:** Division for unit conversion; named conversion factors for clarity.

---

## Temperature Conversion

Here's where CalcMark's built-in unit conversion shines. Assign a value with a temperature unit, then convert it with `in`.

```calcmark
proof_temp_c = 24 celsius
proof_temp_f = proof_temp_c in fahrenheit

oven_temp_c = 230 celsius
oven_temp_f = oven_temp_c in fahrenheit
```

`24 celsius in fahrenheit` yields `75.2` and `230 celsius in fahrenheit` yields `446`. The `in` keyword handles the non-linear Celsius-to-Fahrenheit formula for you -- no need to remember `(C * 9/5) + 32`.

**CalcMark features:** Temperature units (`celsius`); `in` unit conversion (`in fahrenheit`); quantities with units.

---

## Timing Adjustments

Larger batches need longer rise times. Apply a 15% adjustment factor and sum everything up.

```calcmark
base_rise_minutes = 90
rise_adjustment = 1.15
adjusted_rise = base_rise_minutes * rise_adjustment

base_proof_minutes = 45
adjusted_proof = base_proof_minutes * rise_adjustment

bake_time = 45
total_time_minutes = adjusted_rise + adjusted_proof + bake_time
total_time_hours = total_time_minutes / 60
```

`adjusted_rise` is `103.5` minutes, `adjusted_proof` is `51.75` minutes, and `total_time_minutes` sums to `200.25` -- about `3.34` hours from start to finish.

**CalcMark features:** Multiplication for scaling; multi-step derived calculations; division for unit conversion.

---

## Cost Analysis

Price out the scaled ingredients and find the cost per loaf.

```calcmark
flour_price_per_kg = 3.50
salt_price_per_kg = 1.50
yeast_price_per_100g = 5.00

flour_cost = scaled_flour_g / 1000 * flour_price_per_kg
salt_cost = scaled_salt_g / 1000 * salt_price_per_kg
yeast_cost = scaled_yeast_g / 100 * yeast_price_per_100g

total_ingredient_cost = flour_cost + salt_cost + yeast_cost
cost_per_loaf = total_ingredient_cost / target_yield
```

Total ingredient cost is `8.46` for 4 loaves, making `cost_per_loaf` just `2.115`. Homemade bread is cheap.

**CalcMark features:** Chained arithmetic; cross-section variable references (`scaled_flour_g`, `target_yield`).

---

## Shopping List Summary

Round up to practical quantities you can actually buy at the store.

```calcmark
flour_to_buy_kg = 2
salt_to_buy_g = 50
yeast_packets = 1
```

Plain assignments work as a quick reference table. CalcMark doesn't require every line to be a formula -- simple constants are valid too.

**CalcMark features:** Plain numeric assignments; markdown headings as section organizers.

---

## Nutritional Estimate

Estimate calories per loaf and per slice using flour as the dominant calorie source.

```calcmark
calories_per_gram_flour = 3.64
total_calories = scaled_flour_g * calories_per_gram_flour
calories_per_loaf = total_calories / target_yield

Assuming 12 slices per loaf:

slices_per_loaf = 12
calories_per_slice = calories_per_loaf / slices_per_loaf
```

Each loaf has roughly `1820` calories, or about `151.67` calories per slice. The inline prose ("Assuming 12 slices per loaf") is rendered as markdown and keeps the reasoning visible alongside the math.

**CalcMark features:** Markdown prose between calculations; multi-step derived values.

---

## Features Demonstrated

This example showcases the following CalcMark features:

- **Variable assignment and references** -- named values that propagate across sections
- **Arithmetic operators** -- `+`, `-`, `*`, `/` for scaling, conversion, and cost analysis
- **Temperature units** -- `celsius`, `fahrenheit` as first-class units
- **Unit conversion** -- `in fahrenheit` to convert between temperature scales
- **Derived calculations** -- multi-step chains like `scaled_flour_g / 1000 * flour_price_per_kg`
- **Markdown prose** -- headings, paragraphs, and inline comments between calculations
- **Plain constants** -- simple assignments for shopping lists and reference values

## Try It

{{< repo-file path="testdata/examples/recipe-scaling.cm" >}}

```bash
cm testdata/examples/recipe-scaling.cm
```
