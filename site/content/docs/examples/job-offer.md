---
title: "Job Offer Comparison"
summary: "Compare two job offers with salary, equity, bonuses, and risk-adjusted analysis."
weight: 20
calcmark_build: progressive
---

You have two job offers. One is Big Tech with a higher base and signing bonus. The other is a growth startup with larger equity upside. How do you compare them fairly? This walkthrough builds a side-by-side model in CalcMark, converting everything to effective monthly take-home pay.

The complete CalcMark file is available at {{< repo-file path="testdata/examples/job-offer.cm" >}}.

---

## Offer A: Big Tech Company

Start with the guaranteed cash. Offer A has a $180K base, a 15% annual bonus, and a $30K signing bonus. RSUs vest over four years with a one-year cliff.

```cm
base_salary_a = 180000
signing_bonus_a = 30000
annual_bonus_pct_a = 0.15
annual_bonus_a = base_salary_a * annual_bonus_pct_a

stock_grant_a = 200000
vest_years_a = 4
annual_stock_a = stock_grant_a / vest_years_a

annual_comp_a = base_salary_a + annual_bonus_a + annual_stock_a
```

The annual bonus comes out to 27,000 and the stock vests at 50,000 per year. Total annual comp: 277,000.

**CalcMark features:** Variable assignment; arithmetic operators (`*`, `/`, `+`).

---

## Offer B: Growth Startup

Offer B trades base salary for equity upside. The base is $150K with a 10% bonus and no signing bonus. Stock options are worth $400K on paper, but you only expect 50% appreciation.

```cm
base_salary_b = 150000
signing_bonus_b = 0
annual_bonus_pct_b = 0.10
annual_bonus_b = base_salary_b * annual_bonus_pct_b

option_value_b = 400000
expected_appreciation_b = 0.50
effective_stock_value_b = option_value_b * expected_appreciation_b
vest_years_b = 4
annual_stock_b = effective_stock_value_b / vest_years_b

annual_comp_b = base_salary_b + annual_bonus_b + annual_stock_b
```

Even with a $400K option grant, the expected annual stock value is only 50,000 after applying the appreciation discount. Total annual comp: 215,000.

**CalcMark features:** Multi-step derived values; variable reuse across expressions.

---

## Tax Estimation

You need after-tax numbers to make a real comparison. This uses simplified marginal rates for federal, state, and FICA. Stock comp has different tax treatment in practice, but a flat blended rate works for a first pass.

```cm
federal_rate = 0.32
state_rate = 0.093
fica_rate = 0.0765

total_tax_rate = federal_rate + state_rate + fica_rate

after_tax_a = annual_comp_a * (1 - total_tax_rate)
after_tax_b = annual_comp_b * (1 - total_tax_rate)
```

The combined tax rate is about 49%. After tax, Offer A yields roughly 141,000 and Offer B roughly 110,000.

**CalcMark features:** Parenthesized sub-expressions; chaining derived variables.

---

## Monthly Take-Home

Convert annual after-tax pay to monthly so you can compare against your budget.

```cm
monthly_a = after_tax_a / 12
monthly_b = after_tax_b / 12
monthly_difference = monthly_a - monthly_b
```

Offer A puts about $2,600 more in your pocket each month.

**CalcMark features:** Division for unit conversion; subtraction for deltas.

---

## First Year Analysis

Year one is different because Offer A includes a $30K signing bonus. You want to see how that changes the monthly picture.

```cm
year1_gross_a = annual_comp_a + signing_bonus_a
year1_gross_b = annual_comp_b + signing_bonus_b

year1_net_a = year1_gross_a * (1 - total_tax_rate)
year1_net_b = year1_gross_b * (1 - total_tax_rate)

year1_monthly_a = year1_net_a / 12
year1_monthly_b = year1_net_b / 12
```

The signing bonus pushes Offer A's first-year monthly take-home even higher. The gap between the two offers is widest in year one.

**CalcMark features:** Reusing previously defined variables (`signing_bonus_a`, `total_tax_rate`); building scenario-specific totals.

---

## Four Year Total

Equity vests over four years, so you should compare the full vesting period. This is the total gross compensation including the signing bonus.

```cm
four_year_a = annual_comp_a * 4 + signing_bonus_a
four_year_b = annual_comp_b * 4 + signing_bonus_b

four_year_net_a = four_year_a * (1 - total_tax_rate)
four_year_net_b = four_year_b * (1 - total_tax_rate)
```

Over four years, Offer A delivers roughly $580K after tax vs $440K for Offer B. That is a $140K difference.

**CalcMark features:** Multiplication for multi-year projection; consistent tax application.

---

## Risk-Adjusted Value

Startup equity is inherently riskier than Big Tech RSUs. Apply a 40% discount to the startup stock value to reflect the chance the equity ends up worthless.

```cm
startup_risk_discount = 0.40
risk_adjusted_stock_b = annual_stock_b * (1 - startup_risk_discount)
risk_adjusted_annual_b = base_salary_b + annual_bonus_b + risk_adjusted_stock_b
risk_adjusted_monthly_b = risk_adjusted_annual_b * (1 - total_tax_rate) / 12
```

After risk-adjusting the equity, Offer B's monthly take-home drops significantly. This gives you a more conservative baseline for comparison.

**CalcMark features:** Multi-operator expressions; chaining arithmetic in a single assignment.

---

## Benefits Comparison

Benefits have real dollar value. Big Tech typically offers richer packages with better healthcare, 401(k) matching, and perks.

```cm
benefits_a = 15000
benefits_b = 8000

total_value_a = annual_comp_a + benefits_a
total_value_b = annual_comp_b + benefits_b
```

Adding $15K in benefits for Offer A and $8K for Offer B widens the gap further.

**CalcMark features:** Simple addition to combine compensation and benefits.

---

## Summary Metrics

Break the offers into cash comp vs equity to see how much of each package depends on stock performance.

```cm
cash_comp_a = base_salary_a + annual_bonus_a
cash_comp_b = base_salary_b + annual_bonus_b

equity_pct_a = annual_stock_a / annual_comp_a * 100
equity_pct_b = annual_stock_b / annual_comp_b * 100
```

Equity makes up about 18% of Offer A and 23% of Offer B. The startup bet is more concentrated in stock.

**CalcMark features:** Percentage calculation via `/ total * 100`; separating cash from equity.

---

## Decision Factors

Finally, calculate the monthly advantage and figure out how much the startup stock would need to appreciate for Offer B to match Offer A.

```cm
monthly_advantage_a = monthly_a - monthly_b
annual_advantage_a = monthly_advantage_a * 12

comp_gap = annual_comp_a - (base_salary_b + annual_bonus_b)
required_annual_stock = comp_gap
required_total_stock = required_annual_stock * vest_years_b
required_appreciation = required_total_stock / option_value_b
```

The break-even appreciation tells you exactly how much the startup stock needs to grow for the offers to be equivalent. If you believe the startup can exceed that threshold, Offer B wins on expected value.

**CalcMark features:** Parenthesized sub-expressions; break-even analysis with derived variables.

---

## Features Demonstrated

This example showcases the following CalcMark features:

- **Variable assignment** -- named values for every compensation component
- **Arithmetic operators** -- `+`, `-`, `*`, `/` for compensation math
- **Parenthesized expressions** -- `(1 - total_tax_rate)` for tax calculations
- **Derived variables** -- multi-step calculations that build on earlier results
- **Markdown prose** -- explanatory text between calculations
- **Scenario modeling** -- year one, four year, and risk-adjusted views of the same data
- **Break-even analysis** -- solving for the stock appreciation threshold

## Try It

{{< repo-file path="testdata/examples/job-offer.cm" >}}

```bash
cm testdata/examples/job-offer.cm
```
