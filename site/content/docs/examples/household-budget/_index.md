---
title: "Household Budget"
summary: "Monthly budget with income, taxes, expenses, savings goals, and the 50/30/20 rule."
weight: 10
calcmark_build: progressive
calcmark_source: testdata/examples/household-budget.cm
---

Where does your paycheck actually go? This walkthrough builds a monthly budget for a two-income household, tracking income through taxes, fixed and variable expenses, savings goals, and a 50/30/20 rule health check. Along the way you'll see CalcMark's currency arithmetic, percentage calculations, and rate conversions in action.

The complete CalcMark file is available at {{< repo-file path="testdata/examples/household-budget.cm" >}}.

---

## Income

Start with gross salaries. CalcMark's `$` currency literal tags values as US dollars, and adding two currency values preserves the unit.

```calcmark
gross_salary_1 = $6500
gross_salary_2 = $5200
total_gross = gross_salary_1 + gross_salary_2
```

`total_gross` evaluates to `$11,700`. Currency arithmetic works just like plain numbers -- the dollar sign carries through every operation.

**CalcMark features:** Currency literals (`$`); currency addition.

---

## Tax Withholding

Tax rates are plain decimals. Multiplying a currency value by a decimal produces a currency result, so you can compute taxes without losing the `$` unit.

```calcmark
Estimated combined effective tax rates:

federal_rate = 0.18
state_rate = 0.05
fica_rate = 0.0765

total_tax_rate = federal_rate + state_rate + fica_rate
total_taxes = total_gross * total_tax_rate
net_income = total_gross - total_taxes
```

The combined rate is 0.3065 (about 30.7%), giving `total_taxes = $3,586.05` and `net_income = $8,113.95`.

**CalcMark features:** Decimal arithmetic; currency multiplication and subtraction; markdown prose between calculations.

---

## Fixed Expenses

Fixed expenses stay the same each month. List them individually, then sum them up. CalcMark handles long addition chains cleanly.

```calcmark
These don't change month to month:

rent = $2200
car_payment = $450
car_insurance = $180
health_insurance = $400
phone_plans = $120
internet = $80
streaming = $45

total_fixed = rent + car_payment + car_insurance + health_insurance + phone_plans + internet + streaming
```

`total_fixed` comes to `$3,475`. Each line item is a named variable you can reference later -- for example, `streaming` reappears in the 50/30/20 analysis below.

**CalcMark features:** Currency literals; variable references in expressions; multi-operand addition.

---

## Variable Expenses

Variable expenses fluctuate month to month. Use estimates based on past spending and sum them the same way.

```calcmark
Estimates based on past spending:

groceries = $800
gas = $250
utilities = $150
dining_out = $300
entertainment = $200
personal_care = $100
household_supplies = $150

total_variable = groceries + gas + utilities + dining_out + entertainment + personal_care + household_supplies
```

`total_variable` evaluates to `$1,950`. Keeping variable expenses separate from fixed expenses lets you see where you have room to cut.

**CalcMark features:** Currency addition; named variables for later reuse.

---

## Savings Goals

Savings are not expenses -- they're transfers to future-you. Defining them as their own category makes the budget easier to reason about.

```calcmark
emergency_fund_contribution = $500
retirement_401k = $600
vacation_fund = $200

total_savings = emergency_fund_contribution + retirement_401k + vacation_fund
```

`total_savings` is `$1,300` per month. You'll use this value in the health check and emergency fund sections below.

**CalcMark features:** Currency addition; forward references via named variables.

---

## Summary

Now pull everything together. Subtract total outflow from net income to see what's left over each month.

```calcmark
total_expenses = total_fixed + total_variable
total_outflow = total_expenses + total_savings
remaining = net_income - total_outflow
```

`total_expenses` is `$5,425`, `total_outflow` is `$6,725`, and `remaining` is `$1,388.95`. That leftover amount is your buffer for unexpected costs or additional saving.

**CalcMark features:** Currency arithmetic; derived calculations referencing earlier variables.

---

## Budget Health Check

Express each category as a percentage of net income. Use `number()` to extract the raw value from a currency amount, then divide to get a plain ratio.

```calcmark
Calculate percentages of net income:

savings_rate = number(total_savings) / number(net_income) * 100
fixed_pct = number(total_fixed) / number(net_income) * 100
variable_pct = number(total_variable) / number(net_income) * 100
```

`savings_rate` is about 16.02, `fixed_pct` is about 42.83, and `variable_pct` is about 24.03. These percentages give you a quick snapshot of where your money goes.

**CalcMark features:** `number()` to extract raw values for ratio calculations; percentage math.

---

## The 50/30/20 Rule Check

The 50/30/20 rule is a popular budgeting guideline. You can reclassify your expenses into needs, wants, and savings, then check the split against the targets.

```calcmark
The 50/30/20 rule suggests:
- 50% on needs (fixed + essential variable)
- 30% on wants (discretionary)
- 20% on savings

needs = total_fixed + groceries + gas + utilities
wants = dining_out + entertainment + personal_care + streaming
savings_check = total_savings

needs_pct = number(needs) / number(net_income) * 100
wants_pct = number(wants) / number(net_income) * 100
savings_pct = number(savings_check) / number(net_income) * 100
```

`needs_pct` lands around 47.7%, `wants_pct` around 7.95%, and `savings_pct` around 16.02%. Needs are under the 50% ceiling. Wants are well below 30%, which means there is room to redirect more toward savings to hit the 20% target.

**CalcMark features:** Markdown prose (including bullet lists) between calculations; mixing variables from different sections.

---

## Emergency Fund Status

How many months could your emergency fund cover? Divide the balance by monthly expenses. Then calculate how long it takes to reach the recommended six-month target.

```calcmark
Current emergency fund balance:

current_emergency = $8500
monthly_expenses = total_fixed + total_variable
months_runway = number(current_emergency) / number(monthly_expenses)
target_months = 6
target_fund = monthly_expenses * target_months
shortfall = target_fund - current_emergency
months_to_goal = number(shortfall) / number(emergency_fund_contribution)
```

`months_runway` is about 1.57 -- roughly six weeks of coverage. The six-month `target_fund` is `$32,550`, leaving a `shortfall` of `$24,050`. At `$500/month`, `months_to_goal` is about 48 months (four years).

**CalcMark features:** Currency division producing plain numbers; goal planning with derived variables.

---

## Daily Discretionary

CalcMark's rate conversion syntax lets you express monthly costs as daily amounts. The `per day` keyword converts a monthly rate without accumulating it.

```calcmark
Per-day discretionary spending as a rate:

daily_dining = $300/month per day
daily_entertainment = $200/month per day
daily_discretionary = $500/month per day
```

`$300/month per day` divides by ~30.44 days, giving about `$9.86/day` for dining out. Seeing spending at the daily level makes it easier to evaluate impulse purchases.

**CalcMark features:** Rate literals (`$300/month`); `rate per unit` for rate conversion.

---

## Features Demonstrated

This example showcases the following CalcMark features:

- **Currency literals** -- `$` symbols for US dollar values
- **Currency arithmetic** -- addition, subtraction, multiplication, division
- **Decimal arithmetic** -- plain numbers for tax rates and percentages
- **Named variables** -- reusable values referenced across sections
- **Percentage calculations** -- `number()` to extract values for ratio calculations
- **Rate literals** -- `$300/month`, `$200/month`
- **Rate conversion** -- `$300/month per day` to convert time units
- **Markdown prose** -- headings, paragraphs, and bullet lists between calculations
- **Template interpolation** -- `{{variable}}` to embed computed values in prose

## Try It

{{< repo-file path="testdata/examples/household-budget.cm" >}}

```bash
cm testdata/examples/household-budget.cm
```
