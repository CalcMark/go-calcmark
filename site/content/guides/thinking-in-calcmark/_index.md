---
title: "Thinking in CalcMark"
summary: "Transition from calculator thinking to idiomatic CalcMark — sum of, percentage operators, rates, and growth functions."
weight: 24
---

You're moving from Austin to Denver. New job, new apartment, new cost of living. Time to crunch some numbers.

You could open a spreadsheet. Or you could write it down the way you think about it.

## Key Idioms Covered

| Calculator Style | CalcMark Idiom | Reference |
|-----------------|----------------|-----------|
| `a + b + c + d` | `sum of a, b, c, d` | [Functions](/docs/user-guide/functions/) |
| `x * 1.08` | `x + 8%` | [Language Reference](/docs/language-reference/#percentage-arithmetic) |
| `2.75 * 2 * 22` | `$5.50/day over 1 month` | [Language Reference](/docs/language-reference/#rates) |
| `P*(1+r/n)^(nt)` | `compound P by r% monthly over t years` | [Functions](/docs/user-guide/functions/) |
| `sum(a,b,c)` | `sum of a, b, c` | [Functions](/docs/user-guide/functions/) |

## Simple Arithmetic Already Works

Start with the basics. How much more is rent in Denver?

{{< tabs group="output" >}}
{{< tab name="CalcMark" >}}
```text
rent_austin = $1450
rent_denver = $1750
difference = rent_denver - rent_austin
```
{{< /tab >}}
{{< tab name="Editor" >}}
```text
rent_austin = $1450                  $1,450.00
rent_denver = $1750                  $1,750.00
difference = rent_denver - ...       $300.00
```
{{< /tab >}}
{{< tab name="JSON" >}}
```json
{
  "source": "difference = rent_denver - rent_austin",
  "value": "$300.00",
  "type": "currency",
  "numeric_value": 300,
  "unit": "USD",
  "variable": "difference"
}
```
{{< /tab >}}
{{< /tabs >}}

No special syntax needed. CalcMark understands currency, does the arithmetic, and formats the result. If you can type it into a calculator, it works here too.

The difference is what comes next.

## Adding Things Up

Your instinct from a calculator:

```text
total = rent + groceries + transport + utilities
```

That works! But when the list grows to eight items, it gets unwieldy. CalcMark has a better way:

{{< tabs group="output" >}}
{{< tab name="CalcMark" >}}
```text
rent = $1750
groceries = $400
transport = $165
utilities = $120
total = sum of rent, groceries, transport, utilities
```
{{< /tab >}}
{{< tab name="Editor" >}}
```text
rent = $1750                         $1,750.00
groceries = $400                     $400.00
transport = $165                     $165.00
utilities = $120                     $120.00
total = sum of rent, ...             $2,435.00
```
{{< /tab >}}
{{< tab name="JSON" >}}
```json
{
  "source": "total = sum of rent, groceries, transport, utilities",
  "value": "$2,435.00",
  "type": "currency",
  "numeric_value": 2435,
  "unit": "USD",
  "variable": "total"
}
```
{{< /tab >}}
{{< /tabs >}}

`sum of` reads like English and scales to any number of items. You could also write `sum(rent, groceries, transport, utilities)` — both produce identical results. But the NL form is what makes CalcMark feel like writing, not programming.

{{< callout "tip" >}}
Run `cm help sum` to see both syntaxes. Try `cm help functions` for the full list.
{{< /callout >}}

## Percentages Without the Math

The Denver job pays 8% more. Your calculator instinct:

```text
adjusted = salary * 1.08
```

Or worse:

```text
adjusted = salary + salary * 0.08
```

In CalcMark, percentages are operators:

{{< tabs group="output" >}}
{{< tab name="CalcMark" >}}
```text
salary = $85000
adjusted = salary + 8%
after_tax = adjusted - 24%
```
{{< /tab >}}
{{< tab name="Editor" >}}
```text
salary = $85000                      $85K
adjusted = salary + 8%               $91.8K
after_tax = adjusted - 24%           $69.77K
```
{{< /tab >}}
{{< tab name="JSON" >}}
```json
{
  "source": "adjusted = salary + 8%",
  "value": "$91.8K",
  "type": "currency",
  "numeric_value": 91800,
  "unit": "USD",
  "variable": "adjusted"
}
```
{{< /tab >}}
{{< /tabs >}}

`salary + 8%` means "salary plus 8% of salary." `adjusted - 24%` means "adjusted minus 24% of adjusted." No manual conversion to decimals. No remembering whether to multiply by 1.08 or 0.08.

## Rates and Accumulation

Denver has great public transit. How much will your commute cost per month?

Your calculator instinct: count the rides, multiply everything out:

```text
cost_per_ride = $2.75
rides_per_day = 2
days_per_month = 22
monthly_commute = 2.75 * 2 * 22
```

That works, but you're doing the work the computer should do. CalcMark understands rates:

{{< tabs group="output" >}}
{{< tab name="CalcMark" >}}
```text
fare = $5.50/day
monthly_commute = fare over 1 month
```
{{< /tab >}}
{{< tab name="Editor" >}}
```text
fare = $5.50/day                     5.5 $/day
monthly_commute = fare over 1 ...    $165.00
```
{{< /tab >}}
{{< tab name="JSON" >}}
```json
[
  {
    "source": "fare = $5.50/day",
    "value": "5.5 $/day",
    "type": "rate",
    "numeric_value": 5.5,
    "unit": "$/day",
    "variable": "fare"
  },
  {
    "source": "monthly_commute = fare over 1 month",
    "value": "$165.00",
    "type": "currency",
    "numeric_value": 165,
    "unit": "USD",
    "variable": "monthly_commute"
  }
]
```
{{< /tab >}}
{{< /tabs >}}

`$5.50/day` creates a **Rate** — a first-class type in CalcMark. `over` accumulates it across a time period. You write what you mean ("$5.50 per day, over one month") and CalcMark handles the multiplication.

{{< callout "tip" >}}
Run `cm help keywords` to see `per`, `over`, `at`, and other contextual keywords.
{{< /callout >}}

## Putting It Together

Here's your full Denver budget. First, the way you'd write it on a calculator:

{{< tabs group="version" >}}
{{< tab name="Calculator Style" >}}
```text
salary = $85000
take_home = salary + salary * 0.08
take_home = take_home - take_home * 0.24
monthly_income = take_home / 12

rent = $1750
groceries = $400
commute = 2.75 * 2 * 22
utilities = $120
total_expenses = rent + groceries + commute + utilities

remaining = monthly_income - total_expenses
```
{{< /tab >}}
{{< tab name="CalcMark Style" >}}
```text
salary = $85000
take_home = salary + 8% - 24%
monthly_income = take_home / 12

rent = $1750
groceries = $400
commute = $5.50/day over 1 month
utilities = $120
total_expenses = sum of rent, groceries, commute, utilities

remaining = monthly_income - total_expenses
```
{{< /tab >}}
{{< /tabs >}}

Same result. But the CalcMark version is shorter, reads more naturally, and doesn't require you to remember percentage formulas or count multiplication operands.

## Saving for a House

You're settled in Denver. Time to start saving for a down payment.

Your calculator instinct — the compound interest formula:

```text
future = 500 * (1 + 0.045 / 12) ^ (12 * 5)
```

Quick — what does that compute? You'd have to stare at it to decode it. In CalcMark:

{{< tabs group="output" >}}
{{< tab name="CalcMark" >}}
```text
compound $500 by 4.5% monthly over 5 years
```
{{< /tab >}}
{{< tab name="Editor" >}}
```text
compound $500 by 4.5% monthly ...   $625.90
```
{{< /tab >}}
{{< tab name="JSON" >}}
```json
{
  "source": "compound $500 by 4.5% monthly over 5 years",
  "value": "$625.90",
  "type": "currency",
  "numeric_value": 625.9,
  "unit": "USD"
}
```
{{< /tab >}}
{{< /tabs >}}

`compound $500 by 4.5% monthly over 5 years` reads like a sentence. No formula memorization. No getting the parentheses wrong.

The same pattern works for things losing value. Your car:

{{< tabs group="output" >}}
{{< tab name="CalcMark" >}}
```text
depreciate $35000 by 15% over 5 years
```
{{< /tab >}}
{{< tab name="Editor" >}}
```text
depreciate $35000 by 15% over ...    $15.53K
```
{{< /tab >}}
{{< tab name="JSON" >}}
```json
{
  "source": "depreciate $35000 by 15% over 5 years",
  "value": "$15.53K",
  "type": "currency",
  "numeric_value": 15529.69,
  "unit": "USD"
}
```
{{< /tab >}}
{{< /tabs >}}

{{< callout "tip" >}}
Run `cm help compound` to see all growth function variants — including `grow` for linear growth.
{{< /callout >}}

## Beyond the Calculator

CalcMark also has domain-specific functions for engineering and infrastructure:

```text
read 100 MB from ssd
```

That computes how long it takes to read 100 MB from an SSD (0.18 seconds). It reads like English and produces real results. See the [System Sizing guide](/guides/system-sizing/) for the full story on capacity planning, transfer times, and compression estimates.

## NL Syntax with Variables

NL functions work with both literal values and variable references. Define values once and refer to them by name:

```text
yearly_storage = 796 GB
compress yearly_storage using gzip    -- variable reference
compress 796 GB using gzip            -- literal value (equivalent)
```

Both forms produce identical results. Use whichever reads most naturally for your document.

## Quick Reference

| Calculator Style | Idiomatic CalcMark | Why |
|------------------|--------------------|-----|
| `a + b + c + d` | `sum of a, b, c, d` | Reads like English, scales to any length |
| `x * 1.08` | `x + 8%` | Percentage as operator, not decimal math |
| `x - x * 0.24` | `x - 24%` | Same pattern for deductions |
| `2.75 * 2 * 22` | `$5.50/day over 1 month` | Rate accumulation handles the multiplication |
| `P*(1+r/n)^(nt)` | `compound P by r% monthly over t years` | Reads like a sentence |
| `sum(a,b,c)` | `sum of a, b, c` | NL form preferred |
| `avg(a,b,c)` | `average of a, b, c` | Same pattern |
| `sqrt(16)` | `square root of 16` | Same pattern |

## What to Read Next

- **Units and conversions:** [Understanding Measurements](/guides/understanding-measurements/) — quantities, fractions, napkin math, and JSON output semantics
- **Engineering functions:** [System Sizing & Capacity Planning](/guides/system-sizing/) — rates, capacity planning, and `at...per` syntax
- **All functions:** [Functions & NL Syntax](/docs/user-guide/functions/) — complete reference with both NL and functional forms
- **Full language spec:** [Language Reference](/docs/language-reference/) — percentages, rates, growth functions, and everything else
