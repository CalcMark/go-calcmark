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

```yaml
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

"Fully loaded" = base salary + burden (taxes, benefits, equity). Percentage widening makes this natural: `$145,000 + 32%` means "increase by 32%". Under the hood, CalcMark evaluates `$145,000 + 32%` as `$145,000 * (1 + 0.32)`.

```calcmark
senior_base_salary = $145000
senior_burden_rate = 32%
senior_fully_loaded = senior_base_salary + senior_burden_rate

mid_base_salary = $100000
mid_burden_rate = 30%
mid_fully_loaded = mid_base_salary + mid_burden_rate

junior_base_salary = $72000
junior_burden_rate = 28%
junior_fully_loaded = junior_base_salary + junior_burden_rate

mgmt_avg_fully_loaded = $220000
```

`senior_fully_loaded` = `$191.4K`, `mid_fully_loaded` = `$130K`, `junior_fully_loaded` = `$92.16K`.

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

## Non-Labor Overhead

T&E uses percentage widening on `-` to apply a billable recovery rate. Writing `gross - 40%` means "reduce by 40%".

```calcmark
te_billable_recovery_rate = 40%
billable_te_per_person = $12000
mgmt_te_per_person = $3500

travel_and_expenses_gross = (total_billable_hc * billable_te_per_person) + (mgmt_hc * mgmt_te_per_person)
travel_and_expenses = travel_and_expenses_gross - te_billable_recovery_rate
```

The gross T&E is `$151.56K`; after 40% recovery, net T&E is `$90.94K`.

**CalcMark features:** Percentage widening on `-` (value - pct = value reduced by pct).

```calcmark
tooling_per_person = $3200
tooling_and_software = total_team_hc * tooling_per_person

training_per_person = $1800
training_and_enablement = total_team_hc * training_per_person

allocated_overhead_per_person = $3800
allocated_overhead = total_team_hc * allocated_overhead_per_person

total_non_labor_overhead = sum of travel_and_expenses, tooling_and_software, training_and_enablement, allocated_overhead
total_org_cost = total_labor_cost + total_non_labor_overhead
```

---

## Revenue Capacity

Available hours use the globals declared in frontmatter. Percentage widening naturally expresses deductions.

```calcmark
gross_hours_per_person = @globals.working_days * @globals.hours_per_day
non_billable_deduction = 12%
net_available_hours_per_person = gross_hours_per_person - non_billable_deduction
total_billable_capacity = total_billable_hc * net_available_hours_per_person
```

`gross_hours_per_person` = `2K`, `net_available_hours_per_person` = `1.76K` (2,000 reduced by 12%).

```calcmark
target_utilization = 72%
actual_billed_hours = total_billable_capacity * target_utilization
```

---

## Billing Rates & T&M Revenue

Rates and tier-specific utilization drive the T&M capacity cross-check.

```calcmark
senior_bill_rate = $225
mid_bill_rate = $165
junior_bill_rate = $115

senior_utilization = 68%
mid_utilization = 74%
junior_utilization = 78%

senior_billed_hours = senior_consultants * net_available_hours_per_person * senior_utilization
mid_billed_hours = mid_consultants * net_available_hours_per_person * mid_utilization
junior_billed_hours = junior_consultants * net_available_hours_per_person * junior_utilization

senior_tm_revenue = senior_billed_hours * senior_bill_rate
mid_tm_revenue = mid_billed_hours * mid_bill_rate
junior_tm_revenue = junior_billed_hours * junior_bill_rate

total_tm_revenue = sum of senior_tm_revenue, mid_tm_revenue, junior_tm_revenue
```

---

## Engagement Packaging

Fixed-price packages and retainers. The retainer delivery cost uses `% of` — CalcMark's natural-language percentage calculation.

```calcmark
quick_start_price = $18000
quick_start_delivery_cost = $13500
quick_start_margin = quick_start_price - quick_start_delivery_cost

standard_implementation_price = $55000
standard_implementation_delivery_cost = $42000
standard_implementation_margin = standard_implementation_price - standard_implementation_delivery_cost

enterprise_implementation_price = $130000
enterprise_implementation_delivery_cost = $95000
enterprise_implementation_margin = enterprise_implementation_price - enterprise_implementation_delivery_cost

advisory_day_rate = $3200
strategy_workshop_price = $22000
strategy_workshop_delivery_cost = $14000

retainer_monthly_price = $6500
retainer_annual_value = retainer_monthly_price * 12
retainer_annual_delivery_cost = 65% of retainer_annual_value
retainer_annual_margin = retainer_annual_value - retainer_annual_delivery_cost

training_cohort_price = $8500
training_cohort_delivery_cost = $4200
training_cohort_margin = training_cohort_price - training_cohort_delivery_cost
```

`retainer_annual_value` = `$78K`, delivery cost = `$50.7K`, margin = `$27.3K`.

**CalcMark features:** `% of` natural-language syntax.

---

## Annual Volume & Revenue Rollup

```calcmark
quick_start_engagements = 22
standard_engagements = 14
enterprise_engagements = 5
advisory_days_sold = 180
strategy_workshops = 8
retainers_active = 18
training_cohorts = 24

quick_start_revenue = quick_start_engagements * quick_start_price
standard_revenue = standard_engagements * standard_implementation_price
enterprise_revenue = enterprise_engagements * enterprise_implementation_price
advisory_revenue = advisory_days_sold * advisory_day_rate
workshop_revenue = strategy_workshops * strategy_workshop_price
retainer_revenue = retainers_active * retainer_annual_value
training_revenue = training_cohorts * training_cohort_price

total_packaged_revenue = sum of quick_start_revenue, standard_revenue, enterprise_revenue, advisory_revenue, workshop_revenue, retainer_revenue, training_revenue
```

Total packaged revenue evaluates to `$4.18M`.

---

## P&L Summary

```calcmark
total_revenue = total_packaged_revenue

delivery_labor_pct = 72%
delivery_labor_cost = total_labor_cost * delivery_labor_pct
delivery_te_cost = travel_and_expenses
total_cor = delivery_labor_cost + delivery_te_cost

gross_profit = total_revenue - total_cor
gross_margin = gross_profit / total_revenue

practice_management_cost = total_mgmt_cost
non_delivery_overhead = sum of tooling_and_software, training_and_enablement, allocated_overhead

total_opex = practice_management_cost + non_delivery_overhead

contribution = gross_profit - total_opex
contribution_margin = contribution / total_revenue
```

Gross profit = `$2.6M` at 62% gross margin. After opex, contribution = `$2M` at 48% contribution margin.

---

## Key Performance Metrics

```calcmark
revenue_per_billable_hc = total_revenue / total_billable_hc
cost_per_billable_hc = total_labor_cost / total_billable_hc
blended_bill_rate = total_revenue / (total_billable_hc * net_available_hours_per_person * target_utilization)
revenue_per_dollar_of_labor = total_revenue / total_labor_cost

estimated_hours_to_deliver = total_revenue / blended_bill_rate
capacity_consumed = estimated_hours_to_deliver / total_billable_capacity
bench_capacity = 1 - capacity_consumed

recurring_revenue_ratio = retainer_revenue / total_revenue

implementation_rev_pct = (quick_start_revenue + standard_revenue + enterprise_revenue) / total_revenue
advisory_rev_pct = (advisory_revenue + workshop_revenue) / total_revenue
training_rev_pct = training_revenue / total_revenue
```

---

## Scenario A — Strong Year

Utilization beats plan at 78%, retainer base grows to 24, enterprise engagements rise to 7. Incremental revenue drops through at 55% margin because the cost base is already in place.

```calcmark
scenario_a_utilization = 78%
scenario_a_retainers = 24
scenario_a_enterprise_engagements = 7

scenario_a_billable_hours = total_billable_capacity * scenario_a_utilization
scenario_a_retainer_rev = scenario_a_retainers * retainer_annual_value
scenario_a_enterprise_rev = scenario_a_enterprise_engagements * enterprise_implementation_price

scenario_a_revenue_uplift = (scenario_a_retainers - retainers_active) * retainer_annual_value + (scenario_a_enterprise_engagements - enterprise_engagements) * enterprise_implementation_price
scenario_a_total_revenue = total_revenue + scenario_a_revenue_uplift

scenario_a_incremental_margin_rate = 55%
scenario_a_incremental_margin = scenario_a_revenue_uplift * scenario_a_incremental_margin_rate
scenario_a_gross_profit = gross_profit + scenario_a_incremental_margin
scenario_a_gross_margin = scenario_a_gross_profit / scenario_a_total_revenue
```

---

## Scenario B — Challenged Year

Utilization misses at 61%, enterprise deals include 18% pricing concessions, retainer base contracts to 14. The discount uses percentage widening — `blended_bill_rate - 18%` reduces the rate by 18%.

```calcmark
scenario_b_utilization = 61%
scenario_b_discount_rate = 18%
scenario_b_retainers = 14
scenario_b_enterprise_engagements = 3

scenario_b_effective_bill_rate = blended_bill_rate - scenario_b_discount_rate
scenario_b_billed_hours = total_billable_capacity * scenario_b_utilization
scenario_b_tm_revenue = scenario_b_billed_hours * scenario_b_effective_bill_rate

scenario_b_retainer_rev = scenario_b_retainers * retainer_annual_value
scenario_b_enterprise_rev = scenario_b_enterprise_engagements * enterprise_implementation_price
scenario_b_impl_rev = quick_start_engagements * quick_start_price + standard_engagements * standard_implementation_price
scenario_b_total_revenue = sum of scenario_b_retainer_rev, scenario_b_enterprise_rev, scenario_b_impl_rev, advisory_revenue, workshop_revenue, training_revenue

scenario_b_revenue_shortfall = total_revenue - scenario_b_total_revenue

scenario_b_gross_profit = scenario_b_total_revenue - total_cor
scenario_b_gross_margin = scenario_b_gross_profit / scenario_b_total_revenue
scenario_b_contribution = scenario_b_gross_profit - total_opex
scenario_b_contribution_margin = scenario_b_contribution / scenario_b_total_revenue
```

---

## Board Metrics Slate

The four metrics every services dashboard needs, plus revenue per head as a proxy for team leverage.

```calcmark
board_revenue_attainment = scenario_b_total_revenue / total_revenue
board_gross_margin = scenario_b_gross_margin
board_utilization = scenario_b_utilization
board_recurring_pct = scenario_b_retainer_rev / scenario_b_total_revenue
board_rev_per_head = scenario_b_total_revenue / total_billable_hc
rev_to_comp_ratio = board_rev_per_head / senior_fully_loaded
```

Healthy services businesses target a revenue-to-compensation ratio of 2.5–4x.

---

## CalcMark Features Used

- **Frontmatter globals** — `working_days` and `hours_per_day` declared once, referenced everywhere via `@globals.*`
- **Percentage widening** — `base + rate%` for burden loading, `gross - rate%` for recovery deductions, `rate - discount%` for pricing concessions
- **`% of` syntax** — `65% of retainer_annual_value` for delivery cost allocation
- **`sum of` NL function** — Readable rollups for headcount, labor cost, revenue, and overhead
- **Currency arithmetic** — Dollar signs propagate through every operation
- **Percentage literals** — `18%`, `72%`, `40%` are self-documenting rate assumptions
- **Rich prose** — Narrative context, industry benchmarks, and board presentation frameworks interleaved with live calculations
