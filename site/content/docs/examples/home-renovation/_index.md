---
title: "Home Renovation"
summary: "Kitchen remodel and patio budget using averages, square roots, constants, booleans, and rate conversions."
weight: 57
calcmark_build: progressive
calcmark_source: testdata/examples/home-renovation.cm
---

Planning a kitchen remodel and backyard patio is a great way to see CalcMark's math functions, constants, boolean logic, and rate conversions working together in one document.

The complete CalcMark file is available at {{< repo-file path="testdata/examples/home-renovation.cm" >}}.

---

## Averaging Contractor Quotes

You've collected three quotes for the kitchen remodel. The `avg` function takes any number of values and returns their mean:

```calcmark
quote_a = $8500
quote_b = $9200
quote_c = $7800

average_quote = average of quote_a, quote_b, quote_c
```

The `average of` syntax is the natural language form — you can also write `avg(quote_a, quote_b, quote_c)` using function-call syntax.

**CalcMark features:** `average of` (NL form of `avg()`); variable references.

---

## Room Measurements with `sqrt`

You need the kitchen diagonal for countertop layout planning. CalcMark's `sqrt` function handles the Pythagorean theorem:

```calcmark
kitchen_length = 15
kitchen_width = 12
kitchen_sqft = kitchen_length * kitchen_width
diagonal = sqrt(kitchen_length ^ 2 + kitchen_width ^ 2)
```

The `^` operator handles exponentiation, and `sqrt` returns the square root.

**CalcMark features:** `sqrt()` function; exponentiation (`^`); derived calculations.

---

## Circular Patio with `PI`

The backyard patio is a 12-foot diameter circle. CalcMark has `PI` as a built-in constant:

```calcmark
patio_radius = 6
patio_area = PI * patio_radius ^ 2
```

`PI` is read-only — you can't reassign it. `E` (Euler's number) is also available.

**CalcMark features:** `PI` constant; exponentiation with constants.

---

## Precise Display

Concrete is sold per square foot, so you need the exact area for your order. The `as precise` modifier shows full precision, skipping CalcMark's default display rounding:

```calcmark
patio_area as precise
```

This shows the full decimal expansion instead of a rounded value. Useful when you need exact quantities for materials ordering. It's the opposite of `as napkin`.

**CalcMark features:** `as precise` modifier.

---

## Material Costs

Cabinets, countertops, patio concrete, and tile flooring. Multiplying a number by a currency preserves the `$` unit:

```calcmark
cabinets = $6200
countertop_sqft = 45
countertop_rate = $75
countertops = countertop_sqft * countertop_rate

Patio concrete at $8/sqft (rounded up to nearest whole number):

patio_concrete = patio_area * $8

Tile flooring for the kitchen:

tile_per_sqft = $12
flooring = kitchen_sqft * tile_per_sqft
```

CalcMark preserves currency types through arithmetic — multiplying a count by a currency produces a currency, not a plain number.

**CalcMark features:** Currency literals (`$`); currency arithmetic; markdown prose between calculations.

---

## Rate Conversion with `convert_rate`

Your contractor charges $800/day. The `convert_rate` function converts a rate to a different time unit without accumulating:

```calcmark
daily_rate = $800/day
weekly_rate = convert_rate(daily_rate, week)
```

This is different from `$800/day over 1 week`, which would *accumulate* the rate into a total. `convert_rate` keeps it as a rate.

**CalcMark features:** `convert_rate()` function; rate literals (`$800/day`).

---

## Accumulation with `accumulate`

Lumber arrives at 2 pallets per day over the 3-week build. The `accumulate` function totals a rate over a duration:

```calcmark
total_deliveries = 2 pallets/day over 3 weeks
```

CalcMark handles the unit conversion (days to weeks) automatically. The `over` keyword is the natural language form — you can also write `accumulate(2 pallets/day, 3 weeks)` using function-call syntax.

**CalcMark features:** rate accumulation with `over`; rate with custom units (`pallets/day`); automatic time unit conversion.

---

## Project Budget

The `X% of value` syntax calculates a percentage directly:

```calcmark
subtotal = cabinets + countertops + patio_concrete + flooring
contingency = 15% of subtotal
total = subtotal + contingency
```

**CalcMark features:** `X% of value` syntax; currency addition across sections.

---

## Boolean Logic

CalcMark supports `and`, `or`, and `not` operators that produce boolean results. Here you check two conditions at once:

```calcmark
budget_ok = total < $50000 and contingency > $1500
under_budget = not (total > $50000)
```

The `and` operator requires both comparisons to be true. The `not` operator inverts a boolean. You can also use `or` to check if *either* condition is true.

**CalcMark features:** Boolean operators (`and`, `not`); comparison operators (`<`, `>`); parentheses for grouping.

---

## Napkin Math

The `as napkin` modifier rounds to 2 significant figures with a human-readable suffix:

```calcmark
total as napkin
```

The total rounds to a quick number for conversation — useful for ballpark estimates.

**CalcMark features:** `as napkin` modifier.

---

## Features Demonstrated

This example showcases the following CalcMark features:

- **`avg()`** — averaging multiple values
- **`sqrt()`** — square root for geometric calculations
- **`PI` constant** — built-in mathematical constant
- **`as precise`** — full-precision display for exact values
- **`convert_rate()`** — converting rates to different time units
- **`accumulate()`** — totaling a rate over a duration
- **`X% of value`** — percentage calculations
- **Boolean operators** — `and`, `not` with comparison operators
- **`as napkin`** — quick rounding for estimates
- **Currency arithmetic** — `$` preserved through calculations
- **Rate literals** — `$800/day`, `2 pallets/day`
- **Custom units** — `pallets` as a custom unit

## Try It

{{< repo-file path="testdata/examples/home-renovation.cm" >}}

```bash
cm testdata/examples/home-renovation.cm
```
