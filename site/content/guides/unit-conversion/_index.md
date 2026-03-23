---
title: "Unit Conversion & Measurement"
summary: "Convert between unit systems, handle ambiguous units, and use measurement conventions."
weight: 40
---

CalcMark knows about physical units — length, mass, volume, temperature, speed, energy, power, area, data sizes, force, impulse, pressure, acceleration, and frequency. You can convert between them with the `in` keyword, declare document-wide measurement conventions, and use inline qualifiers to be explicit about which definition you mean.

**Try it now:** Try it in the {{< lark "" "CalcMark Lark playground" >}} or run `cm remote unit-conversion` in the editor.

---

## Key Features for This Domain

| Feature | What It Does | Reference |
|---------|-------------|-----------|
| `in` keyword | `10 meters in feet` → `32.81 feet` | [Units](/docs/user-guide/units/#unit-conversion) |
| `measurement:` frontmatter | Declare which gallon, ounce, or ton you mean | [Units](/docs/user-guide/units/#measurement-conventions) |
| Inline qualifiers | `10 troy oz`, `1 imp gal`, `5 short ton` | [Units](/docs/user-guide/units/#measurement-conventions) |
| Strict annotation | Output shows `2 us oz` so readers know which definition is active | [Language Reference](/docs/language-reference/#measurement-conventions) |
| `convert_to: si` / `imperial` | Convert all quantities in the document to a target system | [Frontmatter](/docs/user-guide/frontmatter/#scale-convert) |
| `number()` | Strip units for dimensionless ratios: `number(10 kg) / number(5 kg)` → `2` | [Language Reference](/docs/language-reference/#functions) |

---

## The Ambiguity Problem

Some unit names have different definitions depending on country or domain:

| Unit | US Customary | Imperial (UK) | Difference |
|------|-------------|---------------|------------|
| gallon | 3.785 L | 4.546 L | ~20% |
| pint | 473 mL | 568 mL | ~20% |
| fluid ounce | 29.6 mL | 28.4 mL | ~4% |
| cup | 240 mL | 284 mL | ~18% |

| Unit | Standard (avoirdupois) | Troy (precious metals) | Difference |
|------|----------------------|----------------------|------------|
| ounce | 28.35 g | 31.10 g | ~10% |
| pound | 453.6 g | 373.2 g | ~18% |

| Unit | Short (US) | Long (Imperial) | Metric |
|------|-----------|----------------|--------|
| ton | 907 kg | 1,016 kg | 1,000 kg |

By default, CalcMark uses US Customary. A 4% error on fluid ounces compounds across a recipe. A 20% error on gallons is the difference between a full tank and running out of fuel.

---

## Declaring Your Convention

Use `measurement:` in frontmatter to declare what you mean:

```yaml
---
measurement:
  volume: imperial    # gallon, pint, fl oz → UK definitions
  mass: troy          # ounce, pound → precious metals
  ton: long           # ton → 2240 lb (Imperial)
---
```

Only specify axes that differ from US defaults. Each axis is independent — a UK jeweler might use `volume: imperial` with `mass: troy`.

**What is "standard" mass?** Standard means avoirdupois — the everyday weight system (1 oz = 28.35g, 1 lb = 16 oz). This is what your grocery store and bathroom scale use. Troy weight is only for precious metals and gemstones (1 troy oz = 31.10g, 1 troy lb = 12 troy oz).

---

## Inline Qualifiers

Override the document convention for a single expression:

```calcmark
gold = 10 troy oz in grams
milk = 2 imp pt in ml
cargo = 5 short ton in kg
us_milk = 1 us gal in liters
```

Prefixes: `us`, `imp`/`imperial`, `troy`, `short`, `long`, `metric`. These work regardless of frontmatter — always available.

---

## How Directives Compose

All three directives compose cleanly. Here's how `1 gallon` flows through each combination:

| Directives | Result | What happened |
|-----------|--------|---------------|
| (none) | `1 gal` | US gallon, default display |
| `measurement: { volume: imperial }` | `1 imperial gallon` | Resolved as imperial |
| `convert_to: si` | `3.79 l` | US gallon → liters |
| `measurement: { volume: imperial }` + `convert_to: si` | `4.55 l` | Imperial gallon → liters |
| `measurement: { volume: imperial }` + `scale: 3` + `convert_to: si` | `13.6 l` | Imperial × 3 → liters |
| `1 us gal` (inline) + `measurement: { volume: imperial }` | `3.79 l` | Inline overrides convention |

Pipeline order: **measurement** (resolve names) → **evaluate** → **scale** (multiply) → **convert_to** (change system) → **annotate** (strict mode display).

---

## What to Read Next

- **Complete example:** [Measurement Conventions](/docs/examples/) — inline qualifiers, imperial volume, troy mass
- **Formal spec:** [Measurement Conventions](/docs/language-reference/#measurement-conventions) — full axis table, precedence rules, strict annotation
- **User guide:** [Units & Measurement](/docs/user-guide/units/) — quick reference for unit conversion and measurement
- **Measurement fundamentals:** [Understanding Measurements](/guides/understanding-measurements/) — progressive walkthrough of quantities, fractions, napkin math, and JSON output semantics
- **Recipe guide:** [Recipe Scaling](/guides/recipe-scaling/) — measurement conventions in a cooking context
