---
title: "Worked Examples & Tips"
weight: 6
---

Complete calculation scenarios, practical tips, and solutions to common issues.

## Worked Examples {#worked-examples}

### Basic Calculations

```calcmark
# Simple Math

5 + 3
10 - 2
4 * 5
20 / 4
2 ^ 3
10 % 3
```

### Variables

```calcmark
# Budget

salary = $5000
bonus = $500
expenses = $3000

net = salary + bonus - expenses
```

### Comparisons

```calcmark
# Checks

salary = $50000
threshold = $60000

is_high_earner = salary > threshold
needs_raise = salary < $40000
meets_target = salary >= $50000
```

### Complex Expressions

```calcmark
# Mortgage

principal = $200000
rate = 0.04
years = 30
months = years * 12

monthly_rate = rate / 12
payment = principal * (monthly_rate * (1 + monthly_rate) ^ months) / ((1 + monthly_rate) ^ months - 1)
```

### System Sizing

```calcmark
# Server Capacity Planning

peak_load = 50000 req/s
server_capacity = 2000 req/s
servers = peak_load at server_capacity per server with 25% buffer

# Storage
daily_data = 100 MB/s over 1 day
monthly_storage = daily_data * 30
disks = monthly_storage at 2 TB per disk
```

### Mixed Markdown

```calcmark
# My Monthly Budget

I earn a salary and get bonuses.

## Income

monthly_salary = $5000
annual_bonus = $3000
monthly_bonus = annual_bonus / 12

Total monthly income:
total_income = monthly_salary + monthly_bonus

## Expenses

- Rent: $1500
- Food: $600
- Utilities: $200

rent = $1500
food = $600
utilities = $200
total_expenses = rent + food + utilities

## Summary

Monthly surplus:
surplus = total_income - total_expenses

Can I save 20%?
savings_goal = total_income * 0.20
can_save = surplus >= savings_goal
```

---

## Tips {#tips}

### Organize with Markdown {#tip-markdown}

Use headers and prose to structure your thinking:

```calcmark
# Q1 Budget

## Revenue Assumptions
monthly_revenue = $50000
q1_months = 3
total_revenue = monthly_revenue * q1_months

## Cost Breakdown
fixed_costs = $20000
variable_pct = 30%
variable_costs = total_revenue * variable_pct
```

### Use the Preview Pane {#tip-preview}

Press **Ctrl+P** in the editor to toggle the preview pane, which shows evaluated results alongside your source.

### Get Help on Functions {#tip-help}

Run `cm help functions` to see all available functions with descriptions and usage patterns. Run `cm help constants` for unit constants, or `cm help frontmatter` for frontmatter directives.

## Troubleshooting {#troubleshooting}

### "Undefined variable" {#error-undefined}

Variables must be defined before use. Check that:
1. The variable is spelled correctly
2. It's defined on an earlier line
3. No typos in the name

### "Incompatible units" {#error-incompatible}

You can't add meters to kilograms. Check that operations make physical sense.

### "Parse error" {#error-parse}

The line isn't valid CalcMark syntax. Common issues:
- Missing operator between values
- Unclosed parentheses
- Invalid characters
