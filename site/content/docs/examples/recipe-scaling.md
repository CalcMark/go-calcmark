---
title: "Recipe Scaling"
summary: "Scale English scones from 9 to 36, convert metric to US customary, and find cost per scone."
weight: 30
calcmark_build: progressive
---

You have a scone recipe in metric units. You need 36 for a tea party. This walkthrough scales every ingredient, converts between metric and US customary, handles temperature, and prices out the batch -- all in one CalcMark document.

The complete CalcMark file is available at {{< repo-file path="testdata/examples/recipe-scaling.cm" >}}.

---

## Base Recipe

Capture the base recipe for 9 scones. Each ingredient carries its unit -- grams for dry, milliliters for liquid, teaspoons for leavening.

```calcmark
flour = 280 grams
sugar = 50 grams
butter = 85 grams
milk = 160 ml
baking_powder = 4 teaspoons
salt = 0.5 teaspoons
eggs = 1

oven = 220 celsius
```

Every value except `eggs` is a quantity with a unit. CalcMark tracks units through arithmetic, so `280 grams * 2` produces `560 g` -- not a bare number. Notice that `baking_powder` displays as `1.33 tbsp` because CalcMark auto-scales quantities to readable units.

**CalcMark features:** Quantities with units (`grams`, `ml`, `teaspoons`, `celsius`); automatic unit display scaling; plain numeric assignment.

---

## Scale Up

You need 36 scones instead of 9. Define a scale factor and let CalcMark carry it forward.

```calcmark
makes = 9
target = 36
scale = target / makes
```

`scale` is `4`. Changing `target` in one place rescales the entire document.

**CalcMark features:** Derived calculations; descriptive variable names.

---

## Scaled Batch

Multiply every ingredient by `scale`. Units propagate automatically.

```calcmark
batch_flour = flour * scale
batch_sugar = sugar * scale
batch_butter = butter * scale
batch_milk = milk * scale
batch_bp = baking_powder * scale
batch_salt = salt * scale
batch_eggs = eggs * scale
```

`batch_flour` displays as `1.12 kg` -- CalcMark auto-scales `1120 grams` to kilograms for readability. You also get `200 g` sugar, `640 ml` milk, `5.33 tbsp` baking powder, and `4` eggs. Each result inherits and scales the unit from the original ingredient.

**CalcMark features:** Quantity arithmetic (quantity times scalar); automatic display scaling (`grams` to `kg`, `teaspoons` to `tbsp`); cross-section variable references.

---

## Convert to US Customary

The `in` keyword converts between compatible units. Mass converts to mass, volume to volume, temperature to temperature.

```calcmark
Dry ingredients by weight:

flour_oz = batch_flour in ounces
sugar_oz = batch_sugar in ounces
butter_lb = batch_butter in pounds

Liquid by volume:

milk_cups = batch_milk in cups

Leavening by volume:

bp_tbsp = batch_bp in tablespoons
salt_tbsp = batch_salt in tablespoons

Oven temperature:

oven_f = oven in fahrenheit
```

`batch_flour in ounces` yields `39.5 ounces`, `batch_butter in pounds` gives `0.7496 pounds`, and `640 ml in cups` produces `2.67 cups`. The oven converts from `220 celsius` to `428 fahrenheit` -- no need to remember conversion formulas.

**CalcMark features:** `in` unit conversion; mass conversion (grams to ounces, pounds); volume conversion (ml to cups, teaspoons to tablespoons); temperature conversion (celsius to fahrenheit); markdown prose between calculations.

---

## Cost per Scone

Price out the batch with currency literals and divide by the target yield.

```calcmark
Grocery cost for the full batch:

cost_flour = $3.50
cost_butter = $7.50
cost_sugar = $1.00
cost_milk = $2.00
cost_eggs = $1.50

groceries = cost_flour + cost_butter + cost_sugar + cost_milk + cost_eggs
per_scone = groceries / target
```

Total grocery cost is `$15.50` for 36 scones, making each one `$0.43`. The `$` sign propagates through addition and division.

**CalcMark features:** Currency literals (`$`); currency arithmetic; cross-section variable references (`target`).

---

## Features Demonstrated

This example showcases the following CalcMark features:

- **Quantities with units** -- `280 grams`, `160 ml`, `4 teaspoons`, `220 celsius`
- **Automatic display scaling** -- `1120 grams` displays as `1.12 kg`, `16 teaspoons` as `5.33 tbsp`
- **Unit conversion with `in`** -- `batch_flour in ounces`, `oven in fahrenheit`
- **Mass conversions** -- grams to ounces, grams to pounds
- **Volume conversions** -- milliliters to cups, teaspoons to tablespoons
- **Temperature conversion** -- celsius to fahrenheit
- **Currency arithmetic** -- `$` preserved through addition and division
- **Scaling via multiplication** -- quantity times scalar preserves units
- **Cross-section references** -- variables propagate across sections
- **Markdown prose** -- headings and paragraphs between calculations

## Try It

{{< repo-file path="testdata/examples/recipe-scaling.cm" >}}

```bash
cm testdata/examples/recipe-scaling.cm

# Or open directly from GitHub — no clone required:
cm remote --http {{< repo-raw-url path="testdata/examples/recipe-scaling.cm" >}}
```
