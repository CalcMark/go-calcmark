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

The `scale` directive multiplies every quantity by a factor. The `convert_to` directive converts results to a target measurement system (`si` or `imperial`). Together, they transform the entire document.

```yaml
---
scale: 2
convert_to: si
---
```

`scale: 2` doubles all quantities. `convert_to: si` converts imperial units (cups, ounces, fahrenheit) to metric (ml, grams, celsius). You write the recipe once in its original units — the frontmatter does the rest.

**CalcMark features:** `scale` frontmatter directive; `convert_to` frontmatter directive.

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

`flour` is written as `1.5 cups` but displays as `720 ml` — that's `1.5 × 2 = 3 cups`, converted to milliliters. `butter` goes from `4 ounces` to `227 g`. Arbitrary units like `bananas` and `eggs` are scaled (3 → 6, 2 → 4) but not converted, since they have no metric equivalent.

**CalcMark features:** Quantities with units (`cups`, `ounces`, `teaspoons`); arbitrary units (`bananas`, `eggs`); automatic scaling; automatic unit conversion.

---

## Oven Temperature

Temperature is a special case — `convert_to` converts it, but `scale` does not. Doubling a recipe does not mean doubling the oven temperature.

```calcmark
oven = 350 fahrenheit
```

`350 fahrenheit` converts to `176.666667 celsius`. The scale factor is ignored for temperature by default, which is exactly what you want.

**CalcMark features:** Temperature conversion (fahrenheit to celsius); temperature excluded from scale by default.

---

## Cost per Loaf

Currency values are immune to both `scale` and `convert_to`. Price out the ingredients at their original cost, then divide by the number of loaves.

```calcmark
cost_flour = $0.50
cost_sugar = $0.30
cost_butter = $1.50
cost_milk = $0.25
cost_bananas = $0.75
cost_eggs = $0.60

total_cost = cost_flour + cost_sugar + cost_butter + cost_milk + cost_bananas + cost_eggs
per_loaf = total_cost / 2
```

Total grocery cost is `$3.90` for two loaves, or `$1.95` per loaf. The `$` symbol propagates through addition and division. Currency is unaffected by frontmatter transforms — costs reflect what you actually pay, not a scaled quantity.

**CalcMark features:** Currency literals (`$`); currency arithmetic; currency immune to `scale` and `convert_to`.

---

## What the Frontmatter Does

| What | Scale | Convert |
|------|-------|---------|
| Quantities (`cups`, `ounces`) | Multiplied by factor | Converted to target system |
| Temperature (`fahrenheit`) | Excluded by default | Converted to celsius |
| Arbitrary units (`bananas`, `eggs`) | Multiplied by factor | No conversion (no system mapping) |
| Currency (`$`) | Unaffected | Unaffected |
| Numbers (bare) | Unaffected | Unaffected |

To change the batch size, edit `scale`. To switch between metric and imperial, change `convert_to` to `si` or `imperial`. The rest of the document adjusts automatically.

---

## Features Demonstrated

This example showcases the following CalcMark features:

- **`scale` frontmatter** — doubles every quantity with one line of YAML
- **`convert_to` frontmatter** — converts imperial to metric across the document
- **Temperature exclusion** — oven temperature converts but does not scale
- **Arbitrary units** — `bananas` and `eggs` scale without conversion
- **Currency immunity** — costs are unaffected by transforms
- **Quantities with units** — `cups`, `ounces`, `teaspoons`, `fahrenheit`
- **Currency arithmetic** — `$` preserved through addition and division

## Try It

{{< repo-file path="testdata/examples/recipe-scaling.cm" >}}

```bash
cm testdata/examples/recipe-scaling.cm
```
