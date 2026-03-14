---
title: "Units & Measurement"
weight: 1
---

CalcMark has built-in support for physical units, data sizes, and unit conversion. This page covers supported units, conversion syntax, and measurement conventions for ambiguous units.

### Supported Units {#units}

CalcMark supports a wide range of units across categories:

- **Area**: cm², m², km², ha, in², ft², yd², mi², acre
- **DataSize**: byte, KB, MB, GB, TB, PB (and binary: KiB, MiB, GiB, TiB)
- **Energy**: J, kJ, cal, kcal, kWh
- **Length**: m, cm, mm, km, in, ft, yd, mi, nmi (nautical mile)
- **Mass**: mg, g, kg, metric ton (t), oz, lb
- **Power**: W, kW, MW, hp
- **Speed**: m/s, km/h, mph, knot
- **Temperature**: C, F, K
- **Volume**: mL, L, tsp, tbsp, cup, pt, qt, gal

Time units (`second`, `minute`, `hour`, `day`, `week`, `month`, `year`) are used in durations and rates but are not a conversion category.

Run `cm help constants` for the complete list with aliases and descriptions.

### Unit Conversion {#unit-conversion}

Convert between compatible units using `in` or `as`:

```calcmark
distance = 5 miles
distance_km = distance in km

temp_c = 20 celsius
temp_f = temp_c in fahrenheit

file_size = 1.5 GB
file_size_mb = file_size in MB
```

### Measurement Conventions {#measurement-conventions}

Some units are ambiguous. A "gallon" in the US (3.785 L) is different from a "gallon" in the UK (4.546 L). An "ounce" of gold (troy, 31.10g) differs from an "ounce" of flour (standard, 28.35g).

By default, CalcMark uses US Customary definitions. Add `measurement:` to your frontmatter to declare different conventions:

```calcmark
---
measurement:
  volume: imperial
---

milk = 2 pint
milk_in_ml = milk in ml
```

`milk` resolves as 2 imperial pints, and `milk_in_ml` shows `1,137 ml` (not 946 ml as it would with US pints).

Three independent axes are available:

| Axis | Options | Default | Affected Units |
|------|---------|---------|---------------|
| `volume` | `us`, `imperial` | `us` | gallon, quart, pint, fl oz, cup |
| `mass` | `standard`, `troy` | `standard` | ounce, pound |
| `ton` | `short`, `long`, `metric` | `short` | ton |

"Standard" mass means avoirdupois -- the everyday weight system (1 oz = 28.35g). Troy is for precious metals (1 troy oz = 31.10g).

#### Inline Qualifiers

Override the convention for a single expression with an explicit prefix:

```text
gold = 10 troy oz       → always troy, regardless of frontmatter
beer = 1 imp gal        → always imperial gallon
cargo = 5 short ton     → always US short ton
```

Prefixes: `us`, `imp`/`imperial`, `troy`, `short`, `long`, `metric`.

#### Strict Annotation

When `measurement:` is present, the formatter annotates bare ambiguous units in output so readers know which definition is active. `2 oz` displays as `2 us oz`. Set `strict: false` to suppress this:

```yaml
measurement:
  volume: imperial
  strict: false
```

Measurement conventions compose with `scale` and `convert_to` — see the [Language Reference](/docs/language-reference/#measurement-conventions) for the full composition table and pipeline order.

---

**Related guides:** [Unit Conversion & Measurement](/guides/unit-conversion/) | [Recipe Scaling](/guides/recipe-scaling/)
