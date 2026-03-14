---
title: "Business Planning & Financial Modeling"
summary: "Build P&L statements, budgets, and financial projections with CalcMark."
weight: 20
---

# Business Planning & Financial Modeling

Revenue forecasts, cost breakdowns, margin analysis, budget health checks — financial models are just calculations embedded in narrative. CalcMark makes the narrative and the math live in one document.

**Try it now:** Open the [Household Budget in Lark](https://lark.calcmark.org) or run `cm remote household-budget` in the editor.

---

## Key Features for This Domain

| Feature | What It Does | Reference |
|---------|-------------|-----------|
| Currency arithmetic (`$100 + $50`) | Add, subtract, multiply currencies naturally | [Currency](/docs/user-guide/currency/) |
| `sum of a, b, c` | Sum multiple line items without long `+` chains | [Functions](/docs/user-guide/functions/#function-reference) |
| `number()` for ratios | Strip currency for dimensionless ratios: `number($500) / number($1000)` → `0.5` | [Formatting](/docs/user-guide/formatting/) |
| `% of` syntax | `20% of income` → percentage calculation | [Percentages](/docs/user-guide/formatting/#percentages) |
| `{{variable}}` interpolation | Embed results in prose: "Total: {{revenue}}" | [Templates](/docs/user-guide/frontmatter/#template-variables) |
| Exchange rates | Convert between currencies with frontmatter rates | [Currency](/docs/user-guide/currency/) |
| Growth functions | `compound($1000, 7%, 10 years)` for projections | [Growth Functions](/docs/user-guide/functions/#growth-functions) |

---

## Walkthrough: Budget Health Check

### Step 1: Income

Start with gross income and deductions:

```calcmark
salary_1 = $6500
salary_2 = $5200
total_gross = salary_1 + salary_2

tax_rate = 0.24
net_income = total_gross * (1 - tax_rate)
```

Currency arithmetic preserves the `$` through every operation.

### Step 2: Expenses with `sum of`

Group expenses and total them without long addition chains:

```calcmark
rent = $1800
utilities = $250
insurance = $150
groceries = $600
dining_out = $300

fixed_costs = sum of rent, utilities, insurance
variable_costs = sum of groceries, dining_out
total_expenses = fixed_costs + variable_costs
```

### Step 3: Ratios with `number()`

Dividing currency by currency is an error — dollars divided by dollars doesn't produce a dollar amount. Wrap both sides with `number()` to get a dimensionless ratio:

```calcmark
savings = net_income - total_expenses
savings_rate = number(savings) / number(net_income) * 100
```

`savings_rate` is a plain number (e.g., `41.5`) representing a percentage. Which side you wrap matters — `$100 / number($50)` gives `$2.00` (currency), while `number($100) / number($50)` gives `2` (plain number). See [number() reference](/docs/language-reference/#functions) for the full type rules.

### Step 4: Inline Results with Templates

Put a summary at the top of your document that updates automatically:

```text
## Monthly Summary

Net income: {{net_income}}
Total expenses: {{total_expenses}}
Savings: {{savings}} ({{savings_rate}}%)
```

The `{{variable}}` tags are replaced with formatted values after all calculations run. Forward references work — your summary can appear before the calculations.

---

## What to Read Next

- **Complete examples:** [Household Budget](/docs/examples/household-budget/), [Services P&L](/docs/examples/services-pl/), [Job Offer Comparison](/docs/examples/job-offer/)
- **Investment modeling:** [Investment Growth](/docs/examples/investment-growth/) — compound growth, depreciation
- **Functions reference:** [Growth Functions](/docs/user-guide/functions/#growth-functions)
- **Language spec:** [Type Arithmetic Rules](/docs/language-reference/#type-arithmetic-rules), [Template Interpolation](/docs/language-reference/#template-interpolation)
