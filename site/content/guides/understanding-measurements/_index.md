---
title: "Understanding Measurements"
summary: "Learn how CalcMark handles quantities, unit conversions, fractions, and napkin math — and what each means for your data."
weight: 25
---

In 1999, NASA lost the $327 million Mars Climate Orbiter because one engineering team used pound-force seconds while another used newton-seconds. A unit mismatch burned up a spacecraft.

Larky isn't sending anything to Mars — but their cookie recipe is *critical*. When their friend in London asks for it in metric, getting the butter wrong isn't an option. This guide follows Larky's recipe from simple quantities through unit conversions, fractions, napkin estimates, and measurement conventions — building one concept at a time.

Each example has three views. **CalcMark** shows what you write. **Editor** shows what the CalcMark editor displays. **JSON** shows the machine-readable output from `cm --format json`.


## Key Features Covered

| Feature | What It Does | Reference |
|---------|-------------|-----------|
| Quantities (`2 cups`) | Numbers with physical units | [Units](/docs/user-guide/units/) |
| `in` keyword | Inline unit conversion: `8 oz in grams` | [Units](/docs/user-guide/units/#unit-conversion) |
| Fractions (`1/2 tsp`) | Exact rational numbers | [Language Reference](/docs/language-reference/#type-system) |
| `convert_to: si` | Convert all quantities to metric | [Frontmatter](/docs/user-guide/frontmatter/#scale-convert) |
| `as napkin` | Round to human-friendly estimates | [Formatting](/docs/user-guide/formatting/#napkin-math) |
| Custom units (`24 cookies`) | Domain-specific units unaffected by conversion directives | [Language Reference](/docs/language-reference/#type-system) |
| `measurement:` | Declare US vs Imperial conventions | [Units](/docs/user-guide/units/#measurement-conventions) |
| Force & Impulse units | Physical units like `newton`, `pound-force`, `newton-second` | [Units](/docs/user-guide/units/#units) |


## Quantities: Numbers with Units

Larky starts with a simple recipe:

{{< tabs group="output" >}}
{{< tab name="CalcMark" >}}
```text
flour = 2 cups
butter = 8 oz
sugar = 1 cup
```
{{< /tab >}}
{{< tab name="Editor" >}}
```text
flour = 2 cups                      1 pt
butter = 8 oz                       8 oz
sugar = 1 cup                       1 cup
```
{{< /tab >}}
{{< tab name="JSON" >}}
```json
{
  "source": "flour = 2 cups",
  "value": "1 pt",
  "type": "quantity",
  "numeric_value": 2,
  "unit": "cups"
}
```
{{< /tab >}}
{{< /tabs >}}

Each line creates a **Quantity** — a number paired with a unit. Notice something in the JSON: the `value` field shows "1 pt" (CalcMark normalizes the display), but `numeric_value` stays `2` and `unit` stays `cups`. The display is human-friendly; the data preserves what you wrote.


## Inline Conversions

Larky's friend in London needs metric. The `in` keyword converts units on a single line:

{{< tabs group="output" >}}
{{< tab name="CalcMark" >}}
```text
flour = 2 cups in ml
butter = 8 oz in grams
```
{{< /tab >}}
{{< tab name="Editor" >}}
```text
flour = 2 cups in ml                480 ml
butter = 8 oz in grams              227 grams
```
{{< /tab >}}
{{< tab name="JSON" >}}
```json
{
  "source": "flour = 2 cups in ml",
  "value": "480 ml",
  "type": "quantity",
  "numeric_value": 480,
  "unit": "ml",
  "is_explicit": true
}
```
{{< /tab >}}
{{< /tabs >}}

With `in`, both the display and the data change. The JSON shows `numeric_value: 480` and `unit: "ml"` — the conversion is real, not just cosmetic. The `is_explicit` flag records that this conversion was requested explicitly.

Conversions only work between compatible units. `cups in ml` works (both are volume). `cups in grams` would be an error — volume and mass are different physical dimensions.


## Fractions

Recipes use fractions naturally. CalcMark preserves them as exact rational numbers:

{{< tabs group="output" >}}
{{< tab name="CalcMark" >}}
```text
vanilla = 1/2 tsp
baking_powder = 3/4 tsp
double_vanilla = vanilla * 2
```
{{< /tab >}}
{{< tab name="Editor" >}}
```text
vanilla = 1/2 tsp                   0.5 tsp
baking_powder = 3/4 tsp             0.75 tsp
double_vanilla = vanilla * 2         1 tsp
```
{{< /tab >}}
{{< tab name="JSON" >}}
```json
{
  "source": "vanilla = 1/2 tsp",
  "value": "0.5 tsp",
  "type": "fraction",
  "numeric_value": 0.5,
  "numerator": 1,
  "denominator": 2,
  "unit": "teaspoon"
}
```
{{< /tab >}}
{{< /tabs >}}

Fractions get their own type in JSON — `"type": "fraction"` — with `numerator` and `denominator` fields alongside the decimal `numeric_value`. When you multiply `1/2 * 2`, CalcMark simplifies to `1/1`, which displays as `1 tsp`.

The distinction between fractions and division is whitespace: `1/2` (no spaces) is a fraction; `1 / 2` (with spaces) is division. Both equal 0.5, but a fraction remembers its exact rational form.


## Document-Wide Conversion

Converting every line with `in` gets tedious. The `convert_to` frontmatter directive converts all quantities in the document at once:

{{< tabs group="output" >}}
{{< tab name="CalcMark" >}}
```yaml
---
convert_to: si
---

flour = 2 cups
butter = 8 oz
sugar = 1 cup
oven = 350 fahrenheit in celsius
```
{{< /tab >}}
{{< tab name="Editor" >}}
```text
flour = 2 cups                      480 ml
butter = 8 oz                       227 g
sugar = 1 cup                       240 ml
oven = 350 fahrenheit in celsius     177 celsius
```
{{< /tab >}}
{{< tab name="JSON" >}}
```json
{
  "source": "flour = 2 cups",
  "value": "480 ml",
  "type": "quantity",
  "numeric_value": 0.48,
  "unit": "liter"
}
```
{{< /tab >}}
{{< /tabs >}}

With `convert_to: si`, every quantity converts to metric. The JSON `numeric_value` and `unit` both change — the flour is now `0.48` liters (displayed as "480 ml"). This is a real data transformation, not just a display change.

The oven temperature uses an explicit `in celsius` conversion. When a line already has an inline conversion, `convert_to` respects it — no double-converting.

You can also use `convert_to: imperial` to go the other direction.


## Napkin Math

Larky wants to buy flour in bulk for 10 batches. He doesn't need exact numbers — just a rough sense of how much:

{{< tabs group="output" >}}
{{< tab name="CalcMark" >}}
```text
flour_per_batch = 2 cups
batches = 10
bulk_flour = flour_per_batch * batches as napkin
```
{{< /tab >}}
{{< tab name="Editor" >}}
```text
flour_per_batch = 2 cups             1 pt
batches = 10                         10
bulk_flour = ... as napkin           ~1.25 gal
```
{{< /tab >}}
{{< tab name="JSON" >}}
```json
{
  "source": "bulk_flour = flour_per_batch * batches as napkin",
  "value": "~1.25 gal",
  "type": "quantity",
  "numeric_value": 1.25,
  "unit": "gal",
  "is_approximate": true
}
```
{{< /tab >}}
{{< /tabs >}}

`as napkin` does two things: it **normalizes** to a human-friendly unit (20 cups becomes gallons) and **rounds** to 2 significant figures. The `~` prefix signals "this is a rough estimate."

Look at the JSON carefully. Unlike the bare quantity (where `numeric_value` stays in the original unit), napkin math transforms everything — `numeric_value` is `1.25` and `unit` is `"gal"`. But it also sets `is_approximate: true`, telling downstream tools that this value is intentionally rounded.

Compare with the same quantity without napkin:

{{< tabs group="output" >}}
{{< tab name="CalcMark" >}}
```text
exact_flour = 20 cups
```
{{< /tab >}}
{{< tab name="Editor" >}}
```text
exact_flour = 20 cups                1.25 gal
```
{{< /tab >}}
{{< tab name="JSON" >}}
```json
{
  "source": "exact_flour = 20 cups",
  "value": "1.25 gal",
  "type": "quantity",
  "numeric_value": 20,
  "unit": "cups"
}
```
{{< /tab >}}
{{< /tabs >}}

Without napkin, the display normalizes to "1.25 gal" but `numeric_value` stays at `20` and `unit` stays `"cups"`. The data is untouched.


## Measurement Conventions

Larky's friend isn't just in London — she's British. In the UK, a cup is 284 mL, not the US 240 mL. That's an 18% difference.

The `measurement:` frontmatter tells CalcMark which regional definitions to use:

{{< tabs group="output" >}}
{{< tab name="CalcMark" >}}
```yaml
---
measurement:
  volume: imperial
---

milk = 1 cup
```
{{< /tab >}}
{{< tab name="Editor" >}}
```text
milk = 1 cup                        1 imperial cup
```
{{< /tab >}}
{{< tab name="JSON" >}}
```json
{
  "source": "milk = 1 cup",
  "value": "1 imperial cup",
  "type": "quantity",
  "numeric_value": 1,
  "unit": "imperial cup"
}
```
{{< /tab >}}
{{< /tabs >}}

With `volume: imperial`, `1 cup` means 284 mL (UK) instead of 240 mL (US default). CalcMark annotates the display as "imperial cup" so there's no ambiguity.

When a recipe mixes conventions, use inline qualifiers to override per-line:

```text
us_milk = 1 us cup in ml
uk_milk = 1 imp cup in ml
```

`us_milk` is 240 ml. `uk_milk` is 284 ml. Qualifiers (`us`, `imp`, `imperial`, `troy`, `short`, `long`, `metric`) always work regardless of the document's `measurement:` setting.

The `measurement:` directive has three independent axes:

| Axis | Options | Default | Affects |
|------|---------|---------|---------|
| `volume` | `us` (default), `imperial` | US Customary | cup, pint, fl oz, gallon |
| `mass` | `standard` (default), `troy` | Avoirdupois | ounce, pound |
| `ton` | `short` (default), `long`, `metric` | Short ton (US) | ton |

Only specify axes that differ from the defaults.


## Custom Units

Not everything in a recipe has a standard physical unit. CalcMark lets you use any word as a unit:

{{< tabs group="output" >}}
{{< tab name="CalcMark" >}}
```text
salt = 1 pinch
yield = 24 cookies
```
{{< /tab >}}
{{< tab name="Editor" >}}
```text
salt = 1 pinch                      1 pinch
yield = 24 cookies                   24 cookies
```
{{< /tab >}}
{{< tab name="JSON" >}}
```json
{
  "source": "yield = 24 cookies",
  "value": "24 cookies",
  "type": "quantity",
  "numeric_value": 24,
  "unit": "cookies"
}
```
{{< /tab >}}
{{< /tabs >}}

Custom units like `pinch` and `cookies` behave like any other quantity — you can do arithmetic with them and they'll carry through your calculations. The key difference: `convert_to: si` and `measurement:` ignore them entirely. There's no SI equivalent of a "pinch," so CalcMark leaves it alone.

This makes custom units perfect for recipe yields, serving counts, or any domain-specific measure that shouldn't be touched by unit conversion directives.


## What's Really in Your Data?

This is the question that trips people up: *when I pipe my CalcMark document through `--format json`, what does the data actually look like?*

Here's the same `flour = 2 cups` processed four different ways:

| How | `value` | `numeric_value` | `unit` | `is_approximate` |
|-----|---------|-----------------|--------|-------------------|
| `flour = 2 cups` | `1 pt` | `2` | `cups` | — |
| `flour = 2 cups in ml` | `480 ml` | `480` | `ml` | — |
| `flour = 2 cups` + `convert_to: si` | `480 ml` | `0.48` | `liter` | — |
| `flour = 2 cups as napkin` | `~1 pt` | `1` | `pt` | `true` |

The key distinctions:

- **`value`** always shows a human-friendly, normalized display string. It's what you'd see in the editor.
- **`numeric_value` + `unit`** are the machine-readable data. Without any conversion, they preserve exactly what you wrote. With `in` or `convert_to`, they reflect the converted values precisely. With `as napkin`, they're rounded and normalized.
- **`is_approximate`** only appears with napkin math. It tells any tool consuming the JSON: "this value was intentionally rounded — don't treat it as exact."

When you pipe CalcMark to another tool via `--format json`, `convert_to` and `in` give you precisely converted data — safe for further computation. Napkin math gives you a rounded estimate flagged with `is_approximate: true`, so your downstream tool knows to treat it accordingly.


## The Mars Climate Orbiter: A Worked Example

Remember the spacecraft from the opening? Let's use CalcMark to see exactly what went wrong.

The Mars Climate Orbiter's thrusters reported impulse in pound-force-seconds. The navigation software expected newton-seconds. Here's the mismatch:

{{< tabs group="output" >}}
{{< tab name="CalcMark" >}}
```text
thruster_impulse = 110 pound-force-seconds
expected_impulse = thruster_impulse in newton-seconds
```
{{< /tab >}}
{{< tab name="Editor" >}}
```text
thruster_impulse = 110 pound-force-seconds    110 pound-force-seconds
expected_impulse = ... in newton-seconds       489 newton-seconds
```
{{< /tab >}}
{{< tab name="JSON" >}}
```json
{
  "source": "expected_impulse = thruster_impulse in newton-seconds",
  "value": "489 newton-seconds",
  "type": "quantity",
  "numeric_value": 489.30442,
  "unit": "newton-seconds",
  "is_explicit": true
}
```
{{< /tab >}}
{{< /tabs >}}

The navigation software received `110` and treated it as newton-seconds — but the actual value was `489` newton-seconds. The software underestimated the thruster force by a factor of 4.45, sending the orbiter too close to Mars.

One conversion. $327 million. That's why units matter.


## What to Read Next

- **Scale a recipe:** [Recipe Scaling & Cooking](/guides/recipe-scaling/) — use the `scale` directive to multiply quantities, with `@scale` for per-batch calculations
- **Handle ambiguous units:** [Unit Conversion & Measurement](/guides/unit-conversion/) — US vs Imperial gallons, troy vs standard ounces, and the directive composition pipeline
- **Full reference:** [Language Reference](/docs/language-reference/) — complete specification of quantities, fractions, measurement conventions, and display directives
- **JSON output fields:** [Configuration](/docs/configuration/) — detailed breakdown of every JSON field, type mappings, and locale behavior
