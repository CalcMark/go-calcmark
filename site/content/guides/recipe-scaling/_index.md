---
title: "Recipe Scaling & Cooking"
summary: "Scale recipes with fractions, measurement conventions, and the scale directive."
weight: 30
---

# Recipe Scaling & Cooking

A recipe is a CalcMark document waiting to happen — quantities with units, scaling by servings, cost per portion. CalcMark handles fractions (`2/3 cup`), measurement conventions (US vs Imperial), and document-wide scaling with a single frontmatter directive.

**Try it now:** Open the [Recipe Scaling example in Lark](https://lark.calcmark.org) or run `cm remote recipe-scaling` in the editor.

---

## Key Features for This Domain

| Feature | What It Does | Reference |
|---------|-------------|-----------|
| Fractions (`2/3 cup`, `1/4 tsp`) | Exact rational numbers with units | [Language Reference](/docs/language-reference/#type-system) |
| `scale` directive | Multiply all quantities by a factor: `scale: 3` triples everything | [Frontmatter](/docs/user-guide/frontmatter/#scale-convert) |
| `measurement:` conventions | `volume: imperial` makes `1 cup` = 284 mL (UK) instead of 240 mL (US) | [Units](/docs/user-guide/units/#measurement-conventions) |
| Inline qualifiers (`imp pt`, `us cup`) | Override the document convention for a single line | [Units](/docs/user-guide/units/#measurement-conventions) |
| `convert_to: si` | Convert all quantities to metric in output | [Frontmatter](/docs/user-guide/frontmatter/#scale-convert) |
| `@scale` reference | Use the scale factor in expressions: `per_loaf = total / @scale` | [Frontmatter](/docs/user-guide/frontmatter/#directive-references) |

---

## Walkthrough: Scaling a UK Scone Recipe

### Step 1: Base Recipe with Imperial Measurements

Declare the document uses Imperial volume (UK cups, pints, fluid ounces):

```calcmark
---
measurement:
  volume: imperial
scale:
  factor: 2
  unit_categories: [Mass, Volume]
---

# Scones (makes 8)

flour = 8 oz
butter = 2 oz
sugar = 1 oz
milk = 5 fl oz
```

With `volume: imperial`, the `5 fl oz` of milk is 142 mL (Imperial fluid ounce), not 148 mL (US). The `scale: 2` doubles all Mass and Volume quantities — `flour` displays as `16 oz`, `milk` as `10 fl oz`.

### Step 2: Cost Per Batch

Currency is not in `unit_categories`, so prices stay at single-batch values:

```calcmark
flour_cost = $2.50
butter_cost = $1.80
sugar_cost = $0.40
milk_cost = $0.60
total_cost = sum of flour_cost, butter_cost, sugar_cost, milk_cost
per_scone = total_cost / (8 * @scale)
```

`per_scone` divides by `8 * @scale` (16 scones for a doubled batch). The `@scale` reference reads the factor from frontmatter.

### Step 3: Metric Conversion

Add `convert_to: si` to see everything in metric:

```yaml
---
measurement:
  volume: imperial
scale:
  factor: 2
  unit_categories: [Mass, Volume]
convert_to: si
---
```

Now `flour` displays as `454 g` (16 oz → SI), `milk` as `284 ml` (10 imp fl oz → SI). The three directives compose: **measurement** resolves unit identity → **scale** multiplies → **convert_to** changes the output system.

---

## US vs Imperial: What Changes?

The `measurement:` directive affects how bare unit names are interpreted:

| Unit | US (default) | Imperial |
|------|-------------|----------|
| `1 cup` | 240 mL | 284 mL |
| `1 pint` | 473 mL | 568 mL |
| `1 fl oz` | 29.6 mL | 28.4 mL |
| `1 gallon` | 3.785 L | 4.546 L |

Use inline qualifiers when a recipe mixes conventions:

```text
us_butter = 1 us cup       → always 240 mL
uk_cream = 1 imp pt        → always 568 mL
```

---

## What to Read Next

- **Complete example:** [Recipe Scaling](/docs/examples/recipe-scaling/) — full walkthrough with `@scale`, fractions, and cost analysis
- **Measurement conventions:** [Units & Measurement](/docs/user-guide/units/) — all three axes (volume, mass, ton)
- **Language spec:** [Measurement Conventions](/docs/language-reference/#measurement-conventions), [Scale](/docs/language-reference/#scale)
