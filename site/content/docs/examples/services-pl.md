---
title: "Services P&L"
summary: "Post-sales consulting P&L model: headcount, utilization, engagement mix, scenarios, and board narratives."
weight: 58
calcmark_build: progressive
---

How does a professional services business actually make money? This walkthrough models the full P&L of a post-sales consulting arm attached to a SaaS company — from headcount and cost structure, through revenue capacity and engagement mix, to gross margin and board-ready scenario analysis.

The complete CalcMark file is available at {{< repo-file path="testdata/examples/services-pl.cm" >}}.

---

## Globals and Assumptions

Two values appear throughout the model — working days per year and hours per day. Rather than repeat them, the document declares them once in frontmatter as globals.

```calcmark
---
globals:
  working_days: 250
  hours_per_day: 8
---
```

Reference them anywhere with `@globals.working_days` and `@globals.hours_per_day`.

**CalcMark features:** Frontmatter `globals:` directive; `@globals.*` references.

---

## Team Composition

The team is tiered by seniority. CalcMark's `sum of` natural-language function totals headcount without parentheses.

```calcmark
senior_consultants = 3
mid_consultants = 5
junior_consultants = 4
total_billable_hc = sum of senior_consultants, mid_consultants, junior_consultants
```

Management overhead is 18% of billable headcount. Percentage literals make the rate self-documenting.

```calcmark
mgmt_overhead_rate = 18%
mgmt_hc = total_billable_hc * mgmt_overhead_rate
total_team_hc = total_billable_hc + mgmt_hc
```

`total_billable_hc` evaluates to `12`, `mgmt_hc` to `2.16`, and `total_team_hc` to `14.16`.

**CalcMark features:** `sum of` NL function; percentage literals; percentage normalization on `*`.

---

## Fully-Loaded Cost

"Fully loaded" = base salary + burden (taxes, benefits, equity). Percentage widening makes this natural: `$145,000 + 32%` means "increase by 32%".

```calcmark
senior_base_salary = $145000
senior_burden_rate = 32%
senior_fully_loaded = senior_base_salary + senior_burden_rate
```

`senior_fully_loaded` evaluates to `$191.4K`. Under the hood, `$145,000 + 32%` becomes `$145,000 * (1 + 0.32)`. The same pattern works for all tiers.

**CalcMark features:** Percentage widening on `+` (value + pct = value increased by pct); currency arithmetic.

---

## Total Labor Cost

```calcmark
total_senior_cost = senior_consultants * senior_fully_loaded
total_mid_cost = mid_consultants * mid_fully_loaded
total_junior_cost = junior_consultants * junior_fully_loaded
total_mgmt_cost = mgmt_hc * mgmt_avg_fully_loaded

total_labor_cost = sum of total_senior_cost, total_mid_cost, total_junior_cost, total_mgmt_cost
```

`total_labor_cost` evaluates to `$2.07M`. The `sum of` function reads like a natural rollup.

---

## Travel & Expenses

T&E uses percentage widening on `-` to apply a billable recovery rate. Writing `gross - 40%` means "reduce by 40%".

```calcmark
te_billable_recovery_rate = 40%
travel_and_expenses_gross = (total_billable_hc * billable_te_per_person) + (mgmt_hc * mgmt_te_per_person)
travel_and_expenses = travel_and_expenses_gross - te_billable_recovery_rate
```

The gross T&E is `$151.56K`; after 40% recovery, net T&E is `$90.94K`.

**CalcMark features:** Percentage widening on `-` (value - pct = value reduced by pct).

---

## Revenue Capacity

Available hours use the globals declared in frontmatter. Percentage widening naturally expresses deductions.

```calcmark
gross_hours_per_person = @globals.working_days * @globals.hours_per_day
non_billable_deduction = 12%
net_available_hours_per_person = gross_hours_per_person - non_billable_deduction
```

`gross_hours_per_person` = `2K`, `net_available_hours_per_person` = `1.76K` (2,000 reduced by 12%).

---

## Engagement Packaging

The retainer delivery cost uses `% of` — CalcMark's natural-language percentage calculation.

```calcmark
retainer_monthly_price = $6500
retainer_annual_value = retainer_monthly_price * 12
retainer_annual_delivery_cost = 65% of retainer_annual_value
retainer_annual_margin = retainer_annual_value - retainer_annual_delivery_cost
```

`retainer_annual_value` = `$78K`, delivery cost = `$50.7K`, margin = `$27.3K`.

**CalcMark features:** `% of` natural-language syntax.

---

## Revenue Rollup

Total packaged revenue sums seven engagement types using `sum of`:

```calcmark
total_packaged_revenue = sum of quick_start_revenue, standard_revenue, enterprise_revenue, advisory_revenue, workshop_revenue, retainer_revenue, training_revenue
```

Evaluates to `$4.18M`.

---

## P&L Summary

```calcmark
total_revenue = total_packaged_revenue
delivery_labor_pct = 72%
delivery_labor_cost = total_labor_cost * delivery_labor_pct
total_cor = delivery_labor_cost + travel_and_expenses

gross_profit = total_revenue - total_cor
gross_margin = gross_profit / total_revenue
```

Gross profit = `$2.6M`, gross margin = 62%. After operating expenses (practice management + non-delivery overhead), contribution margin is 48%.

---

## Scenario Analysis

The model runs two scenarios against the baseline.

**Scenario A (Strong Year):** Utilization beats plan at 78%, retainer base grows to 24, enterprise engagements rise to 7. Incremental revenue drops through at 55% margin because the cost base is already in place.

**Scenario B (Challenged Year):** Utilization misses at 61%, enterprise deals include 18% pricing concessions, retainer base contracts to 14. The discount uses percentage widening:

```calcmark
scenario_b_effective_bill_rate = blended_bill_rate - scenario_b_discount_rate
```

`blended_bill_rate - 18%` reduces the rate by 18%, modeling the discount naturally.

---

## Board Metrics Slate

The model closes with the four metrics every services dashboard needs:

```calcmark
board_revenue_attainment = scenario_b_total_revenue / total_revenue
board_gross_margin = scenario_b_gross_margin
board_utilization = scenario_b_utilization
board_recurring_pct = scenario_b_retainer_rev / scenario_b_total_revenue
board_rev_per_head = scenario_b_total_revenue / total_billable_hc
rev_to_comp_ratio = board_rev_per_head / senior_fully_loaded
```

The revenue-to-compensation ratio is a proxy for team leverage — healthy services businesses run 2.5–4x.

---

## CalcMark Features Used

This example exercises several CalcMark features working together:

- **Frontmatter globals** — `working_days` and `hours_per_day` declared once, referenced everywhere via `@globals.*`
- **Percentage widening** — `base + rate%` for burden loading, `gross - rate%` for recovery deductions, `rate - discount%` for pricing concessions
- **`% of` syntax** — `65% of retainer_annual_value` for delivery cost allocation
- **`sum of` NL function** — Readable rollups for headcount, labor cost, revenue, and overhead
- **Currency arithmetic** — Dollar signs propagate through every operation
- **Percentage literals** — `18%`, `72%`, `40%` are self-documenting rate assumptions
- **Rich prose** — Narrative context, industry benchmarks, and board presentation frameworks interleaved with live calculations
