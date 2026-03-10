---
title: "Recipe Scaling"
summary: "Double a banana bread recipe and convert imperial to metric — with two lines of frontmatter."
weight: 30
calcmark_build: progressive
---

You have an American banana bread recipe written in cups, ounces, and fahrenheit. You need to double the batch for a party, and your kitchen has metric measuring tools. Two lines of YAML frontmatter handle both transformations — every ingredient is scaled and converted automatically.

The complete CalcMark file is available at {{< repo-file path="testdata/examples/recipe-scaling.cm" >}}.

---

## Frontmatter: Scale and Convert

The `scale` directive multiplies quantities by a factor. Adding `Currency` to `unit_categories` makes costs scale too. `Custom` scales units like `bananas` and `eggs` that aren't in the standard unit library. The `convert_to` directive converts results to a target measurement system (`si` or `imperial`). Together, they transform the entire document.

```yaml
---
scale:
  factor: 2
  unit_categories: [Mass, Volume, Currency, Custom]
convert_to: si
---
```

`scale: {factor: 2, unit_categories: [Mass, Volume, Currency, Custom]}` doubles all quantities, currency values, and custom units. `convert_to: si` converts imperial units (cups, ounces, fahrenheit) to metric (ml, grams, celsius). You write the recipe once in its original units — the frontmatter does the rest.

**CalcMark features:** `scale` frontmatter directive with `unit_categories`; `convert_to` frontmatter directive; `Currency` and `Custom` opt-in for scaling.

---

## Ingredients

Write each ingredient as a quantity with its original unit. The displayed results are already doubled and in metric.

```calcmark
flour = 1.5 cups
sugar = 0.75 cups
butter = 4 ounces
milk = 0.25 cups
bananas = 3 bananas
eggs = 2 eggs
baking_soda = 1 teaspoons
salt = 0.25 teaspoons
vanilla = 1 teaspoons
```

`flour` is written as `1.5 cups` but displays as `720 ml` — that's `1.5 × 2 = 3 cups`, converted to milliliters. `butter` goes from `4 ounces` to `227 g`. Custom units like `bananas` and `eggs` are scaled (3 → 6, 2 → 4) because `Custom` is in `unit_categories`, but they aren't converted since they have no metric equivalent.

**CalcMark features:** Quantities with units (`cups`, `ounces`, `teaspoons`); custom units (`bananas`, `eggs`); automatic scaling; automatic unit conversion.

---

## Oven Temperature

Temperature is not listed in `unit_categories`, so it converts but does not scale. Doubling a recipe does not mean doubling the oven temperature — you control this by choosing which categories to include.

```calcmark
oven = 350 fahrenheit
```

`350 fahrenheit` converts to `176.666667 celsius`. The scale factor only applies to categories listed in `unit_categories`, so temperature is unaffected.

**CalcMark features:** Temperature conversion (fahrenheit to celsius); category-based scaling control.

---

## Cost per Loaf

Because `Currency` is listed in `unit_categories`, costs scale with the batch. The `@scale` directive references the scale factor directly in expressions, so `per_loaf` stays correct when you change the batch size.

```calcmark
cost_flour = $0.50
cost_sugar = $0.30
cost_butter = $1.50
cost_milk = $0.25
cost_bananas = $0.75
cost_eggs = $0.60

total_cost = cost_flour + cost_sugar + cost_butter + cost_milk + cost_bananas + cost_eggs
per_loaf = total_cost / @scale
```

Each ingredient cost is doubled by the scale factor. `total_cost` sums to `$7.80` (the doubled batch cost), and `per_loaf = total_cost / @scale` gives `$3.90` — the cost for one loaf. The `@scale` reference reads the factor from frontmatter, so changing `factor: 2` to `factor: 5` automatically updates both the scaled costs and the per-loaf calculation.

**CalcMark features:** Currency literals (`$`); currency opt-in scaling via `unit_categories: [Currency]`; `@scale` directive reference.

---

## What the Frontmatter Does

| What | Scale | Convert |
|------|-------|---------|
| Quantities (`cups`, `ounces`) | Scaled when category is in `unit_categories` | Converted to target system |
| Custom units (`bananas`, `eggs`) | Scaled when `Custom` is in `unit_categories` | No conversion (no system mapping) |
| Currency (`$`) | Scaled when `Currency` is in `unit_categories` | Unaffected |
| Temperature (`fahrenheit`) | Scaled when `Temperature` is in `unit_categories` | Converted to celsius |
| Numbers (bare) | Scaled when `Number` is in `unit_categories` | Unaffected |

Every category is opt-in. To change the batch size, edit `scale`. To switch between metric and imperial, change `convert_to` to `si` or `imperial`. The rest of the document adjusts automatically.

---

## Features Demonstrated

This example showcases the following CalcMark features:

- **`scale` frontmatter** — doubles every quantity with one line of YAML
- **`unit_categories` filtering** — opt-in scaling for `Mass`, `Volume`, `Currency`, `Custom`
- **`Custom` category** — scales custom units like `bananas` and `eggs`
- **`@scale` directive** — reference the scale factor in expressions
- **`convert_to` frontmatter** — converts imperial to metric across the document
- **Category-based control** — temperature converts but does not scale (not in `unit_categories`)
- **Custom units** — `bananas` and `eggs` scale without conversion
- **Quantities with units** — `cups`, `ounces`, `teaspoons`, `fahrenheit`
- **Currency arithmetic** — `$` preserved through addition and division

## Try It

{{< repo-file path="testdata/examples/recipe-scaling.cm" >}}

```bash
cm testdata/examples/recipe-scaling.cm
```
