---
title: "Investment & Growth"
summary: "Compound growth, depreciation, and linear growth for financial and business modeling."
weight: 56
calcmark_build: progressive
calcmark_source: testdata/examples/investment-growth.cm
---

How fast does your retirement fund grow? How quickly does a car lose value? This walkthrough models investment returns, business growth, and asset depreciation using CalcMark's growth functions.

The complete CalcMark file is available at {{< repo-file path="testdata/examples/investment-growth.cm" >}}.

---

## Retirement Savings

You start with $50,000 in savings, a 7% annual return, and 30 years until retirement. Define the inputs as variables, then pass them to `compound` using the natural language form.

```calcmark
initial_savings = $50000
annual_return = 7%
years_to_retire = 30

retirement_fund = compound initial_savings by annual_return over years_to_retire
```

Without a frequency modifier, `compound` compounds once per year: `A = P × (1+r)^t`. The 7% annual rate is applied once per year for 30 years, growing $50K to ~$380.6K. The NL form accepts variable references in any argument position.

**CalcMark features:** `compound` NL form with variable references; currency literals (`$50000`); percentage literals (`7%`).

---

## Monthly Compounding

Real-world investments compound more frequently than once per year. Adding `monthly` switches to the financial compounding formula:

```text
A = P × (1 + r/n)^(n × t)
```

where `r` = annual rate, `n` = 12 (monthly), `t` = years.

```calcmark
retirement_monthly = compound initial_savings by annual_return compounded monthly over 30 years
```

Monthly compounding pushes the result to ~$405.8K -- about $25K more than annual compounding over the same 30 years. The frequency adverb (`monthly`, `quarterly`, `weekly`, `daily`, `yearly`) is what triggers this formula. Without it, you get simple growth.

**CalcMark features:** `compound ... compounded monthly` NL modifier; variable references for principal and rate; `30 years` duration annotation.

---

## Business Growth

Growth functions are not limited to money. You can use custom units like `customers` or `users`. Here you model a startup's customer base growing 15% per month over 24 months.

```calcmark
starting_customers = 500 customers
monthly_growth = compound starting_customers by 15% over 24
```

Starting from 500 customers at 15% monthly growth, you reach ~14.3K customers after two years. The NL form references `starting_customers` by name — CalcMark preserves the `customers` unit through the calculation.

The natural language form reads like a sentence:

```calcmark
compound 1000 users by 10% over 12 months
```

Starting with 1,000 users and growing 10% per month for a year yields ~3.1K users. The NL form (`compound X by Y% over Z`) is equivalent to calling `compound(X, Y%, Z)`.

**CalcMark features:** Custom units (`customers`, `users`); `compound` natural language form (`compound X by Y% over Z`); unit preservation through growth functions.

---

## Linear Growth

Not all growth is exponential. The `grow()` function adds a fixed increment each period. Here you model adding $200 per month to a savings account over 5 years (60 months).

```calcmark
savings_plan = grow $0 by $200 over 60
```

`grow(start, increment, periods)` adds the increment once per period: `$0 + ($200 * 60) = $12K`. No compounding -- just steady accumulation.

The natural language form works with custom units too:

```calcmark
grow 100 subscribers by 50 over 52 weeks
```

Starting with 100 subscribers and adding 50 per week for a year gives you 2,700 subscribers. The NL form (`grow X by Y over Z`) is equivalent to `grow(X, Y, Z)`.

**CalcMark features:** `grow()` function for linear (additive) growth; `grow` natural language form; custom units (`subscribers`).

---

## Asset Depreciation

Assets lose value over time. The `depreciate()` function models declining-balance depreciation -- each period reduces the remaining value by a fixed percentage.

```calcmark
car_value = depreciate $35000 by 20% over 5
```

A $35K car losing 20% per year is worth ~$11.5K after 5 years. Each year applies the 20% rate to the current value, not the original price.

You can set a salvage floor with the `to` keyword. The asset will never depreciate below that value:

```calcmark
depreciate $80000 by 25% over 10 years to $2000
```

This $80K piece of equipment depreciates at 25% per year but cannot drop below $2,000. After 10 years it lands at ~$4,505 -- still above the floor. Without the floor, it would reach the same value here, but the `to` clause protects against longer time horizons.

Finally, office furniture at a gentler rate:

```calcmark
furniture = depreciate $15000 by 15% over 7
```

$15K in furniture at 15% per year is worth ~$4,809 after 7 years.

**CalcMark features:** `depreciate()` function for declining-balance depreciation; `depreciate` natural language form (`depreciate X by Y% over Z`); salvage floor with `to` keyword.

---

## Features Demonstrated

This example showcases the following CalcMark features:

- **`compound()`** -- exponential growth with percentage rates
- **`monthly`** -- compounding frequency modifier (also `quarterly`, `weekly`, `daily`, `yearly`)
- **`compound` NL form** -- `compound X by Y% over Z`
- **`grow()`** -- linear (additive) growth with fixed increments
- **`grow` NL form** -- `grow X by Y over Z`
- **`depreciate()`** -- declining-balance depreciation
- **`depreciate` NL form** -- `depreciate X by Y% over Z`
- **Salvage floor** -- `to $2000` keyword in depreciation
- **Custom units** -- `customers`, `users`, `subscribers`
- **Currency literals** -- `$50000`, `$35000`
- **Percentage literals** -- `7%`, `15%`, `20%`
- **Markdown prose** -- headings, paragraphs, and inline comments between calculations

## Try It

{{< repo-file path="testdata/examples/investment-growth.cm" >}}

```bash
cm testdata/examples/investment-growth.cm
```
