---
title: "Frontmatter Directives"
weight: 3
---

CalcMark uses YAML frontmatter to configure document-wide behavior: global variables, directive references, template interpolation, and scaling/conversion transforms.

Editors with CalcMark LSP support (including the TUI and the CalcMark web editor) show hover documentation, completion, and document-outline entries for CalcMark-specific frontmatter keys. Keys outside the CalcMark set (e.g., Jekyll-style `title`, `date`, `author`) pass through untouched with no editor hints. The authoritative list of CalcMark-grammar keys lives in [`spec/document/frontmatter_registry.go`](https://github.com/CalcMark/go-calcmark/blob/main/spec/document/frontmatter_registry.go).

### Global Variables {#global-variables}

Define reusable values in the frontmatter that can be referenced throughout your document:

```yaml
---
globals:
  base_date: Jan 15 2025
  tax_rate: 0.32
  base_price: $100
  sprint_length: 2 weeks
  bandwidth: 100 MB/s
---
```

Reference globals in expressions with the `@globals.` prefix:

```text
net_price = @globals.base_price * (1 - @globals.tax_rate)
project_end = @globals.base_date + @globals.sprint_length * 6
monthly_transfer = @globals.bandwidth over 30 days
```

Globals support all CalcMark literal types:

- **Numbers**: `42`, `3.14`, `1.5K`, `25%`
- **Quantities**: `10 meters`, `5 kg`, `100 MB`
- **Currencies**: `$100`, `50 EUR`
- **Dates**: `Jan 15 2025`
- **Durations**: `5 days`, `2 weeks`
- **Rates**: `100 MB/s`, `$50/hour`
- **Booleans**: `true`, `false`

Globals must be literal values. Expressions like `1 + 1` are not allowed.

### @Directive References {#directive-references}

Use `@scale` and `@globals.name` to reference frontmatter values in expressions:

```text
per_unit = total_cost / @scale
tax = income * @globals.tax_rate
```

`@scale` resolves to the numeric scale factor from frontmatter. `@globals.name` resolves to the typed value of a named global. These are the only supported directives -- `@exchange`, `@convert_to`, and other names produce errors.

`@globals` without a field name (e.g., bare `@globals`) is a parser error. Use `@globals.name` to reference a specific global.

### Template Variables {#template-variables}

Use `{{variable_name}}` in prose to embed calculated values. Tags are resolved after all calculations run, so forward references work -- put a summary at the top and calculations below:

```text
## Summary

Total cost: {{total_cost}}
Per person: {{per_person}}


headcount = 12
rate = $150000
total_cost = headcount * rate
per_person = total_cost / headcount
```

The summary renders with **$1.8M** and **$150K** substituted in. If a variable is not defined, the `{{tag}}` is left as-is in the output.

#### Inline Patterns

Interpolated values render **bold** in Markdown output and are wrapped in `<span class="cm-interpolated">` in HTML. You can combine them with other Markdown formatting:

```text
The grand total is {{total_cost}}.

_Headcount: {{team_size}}_

## Budget: {{annual_budget}}

| Metric | Value |
|--------|-------|
| Revenue | {{revenue}} |
| Margin | {{margin}} |
```

Backticks around `{{var}}` are automatically stripped -- `` `{{cost}}` `` renders as **$1.8M**, not `$1.8M`. This prevents interpolated values from appearing as inline code in HTML.

#### Tips

- **Forward references are the killer feature.** Write your executive summary first, add calculations below, and the summary stays current when assumptions change.
- **Whitespace is tolerated.** `{{ total_cost }}` works the same as `{{total_cost}}`.
- **Undefined variables are safe.** `{{unknown}}` passes through unchanged -- you won't get errors for tags that don't match a variable.
- **Scale and convert_to apply.** Interpolated values match what CalcBlock results display, including any frontmatter transforms.

See the [Language Reference: Template Interpolation](/docs/language-reference/#template-interpolation) for the complete specification, the [Services P&L](/docs/examples/services-pl/) for a production dashboard with interpolated summary tables, and the [Household Budget](/docs/examples/household-budget/) for inline results in prose.

### Scale and Convert {#scale-convert}

Use `scale` and `convert_to` in the frontmatter to transform results across an entire document. This is useful for recipes, engineering estimates, and any document where you want to change units or multiply quantities globally.

#### Scaling

Scaling is explicit -- you specify which categories to scale in `unit_categories`. A bare `scale: 2` sets the factor (accessible via `@scale`) but scales nothing on its own.

```yaml
---
scale:
  factor: 2
  unit_categories: [Mass, Volume]
---
```

```text
flour = 1.5 cups       → 3 cups (doubled — Volume)
eggs = 3 eggs           → 3 eggs (unchanged — custom unit, not in categories)
oven = 350 fahrenheit   → 350 fahrenheit (unchanged — not in categories)
price = $5.00           → $5.00 (unchanged — not in categories)
```

`flour` displays as `3 cups` because Volume is in the category list. `eggs` is unchanged because `eggs` is a custom unit -- add `Custom` to `unit_categories` to scale it. Temperature and currency are unaffected unless you add `Temperature` or `Currency` to `unit_categories`.

In the editor, scaled values show a `×` (multiply) suffix in the results pane. Converted values show a `•` (bullet) suffix. When both scaling and conversion apply, the result shows `×•`. The context footer shows `@scale = N` when you're on a scaled line.

#### Converting

Convert all quantities to a target measurement system:

```yaml
---
convert_to: si
---
```

```text
distance = 5 miles       -> 8.05 km
weight = 10 pounds       -> 4.54 kg
temp = 72 fahrenheit     -> 22.2222 celsius
```

`distance` displays as `8.05 km`, `weight` as `4.54 kg`, and `temp` as `22.2222 celsius`. Converted values show a `•` suffix in the editor results pane. If you use the `in` keyword explicitly (e.g., `distance in feet`), that conversion takes priority over `convert_to`.

#### Both Together

When both are present, scale applies first, then convert:

```yaml
---
scale:
  factor: 2
  unit_categories: [Mass]
convert_to: si
---
```

```text
butter = 4 ounces       -> 227 g (doubled, then converted)
```

`butter` is first doubled to `8 ounces` (Mass is in `unit_categories`), then converted to `227 g`. In the editor, this result shows `×•` -- both the scale and convert indicators.

#### Filtering by Category

Limit scale or convert to specific unit categories:

```yaml
---
scale:
  factor: 4
  unit_categories: [Mass, Volume]
convert_to:
  system: si
  unit_categories: [Length]
---
```

Valid categories: `Acceleration`, `Area`, `Currency`, `Custom`, `DataSize`, `Energy`, `Force`, `Frequency`, `Impulse`, `Length`, `Mass`, `Number`, `Power`, `Pressure`, `Speed`, `Temperature`, `Volume`. `Custom` matches units not in the standard library (e.g., `bananas`, `eggs`). These are derived from the unit definitions -- run `cm help frontmatter` for the current list. Use `unit_categories: [All]` to scale every category.

See the [Recipe Scaling](/docs/examples/recipe-scaling/) example for a complete walkthrough, and the [Language Reference — Frontmatter](/docs/language-reference/#frontmatter) for the full specification.

#### Transform Indicators {#transform-indicators}

The editor results pane shows symbols next to values that have been transformed:

| Indicator | Meaning | Example |
|-----------|---------|---------|
| `×` | Scaled | `4 buns×` -- factor applied |
| `•` | Converted | `4.54 kg•` -- unit system changed |
| `×•` | Both | `227 g×•` -- scaled then converted |
| _(none)_ | Untransformed | `10 m` -- no transforms applied |

For details on measurement conventions that interact with scaling and conversion, see [Units & Measurement: Measurement Conventions](../units/#measurement-conventions).

### Fiscal Calendar {#fiscal-calendar}

Two keys configure how `FQ1`–`FQ4`, `FYxxxx`, `this fiscal quarter`, and `this fiscal year` resolve:

```yaml
---
fiscal_year_starts: february 2     # required for any fiscal expression
calendar_year_offset: before       # optional — `before` (default) or `after`
---
```

`fiscal_year_starts` accepts a month name (`july`, `oct`) or a month + day (`July 15`, `April 1`).

`calendar_year_offset` controls how an FY label maps to a calendar year when the fiscal year does not start in January:

- `before` (default) — FY label = year FY ends in, so the FY *starts* in the calendar year **before** its label. Matches the Australian government year, the US tax year for non-calendar fiscal periods, and most publicly traded companies. With `fiscal_year_starts: february 2`, `FY2026` = Feb 2 2025 → Feb 1 2026.
- `after` — FY label = year FY starts in, so the FY *ends* in the calendar year **after** its label. With the same start, `FY2026` = Feb 2 2026 → Feb 1 2027.

The setting only affects labeling; the duration and quarter shape of the fiscal year are unchanged. It has no effect when `fiscal_year_starts: january`.

See [Dates & Time → Fiscal Periods](../dates/#fiscal) for fiscal expressions and worked examples.
