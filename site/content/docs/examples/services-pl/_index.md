---
title: "Services P&L"
summary: "Post-sales consulting P&L model: headcount, utilization, engagement mix, scenarios, and board narratives."
weight: 58
calcmark_build: progressive
calcmark_source: testdata/examples/services-pl.cm
---

How does a professional services business actually make money? This walkthrough models the full P&L (Profit & Loss statement) of a post-sales consulting arm attached to a SaaS company — from headcount and cost structure, through revenue capacity and engagement mix, to gross margin and board-ready scenario analysis.

A services P&L is fundamentally a people business wrapped in a software context. You sell time, expertise, and outcomes. Unlike the SaaS product, services revenue is not recurring by default, it does not scale without headcount, and gross margins are structurally lower. Four levers govern every line:

1. **Capacity** — how many billable people, and how many hours can they sell?
2. **Rates** — what do you charge per hour or per engagement?
3. **Utilization** — what fraction of available capacity actually generates revenue?
4. **Mix** — which engagement types fill the book, and do they carry different margins?

The complete CalcMark file is available at {{< repo-file path="testdata/examples/services-pl.cm" >}}.

---

## At a Glance

The document opens with a summary table that uses template interpolation — `{{total_rev}}`, `{{gross_margin}}`, etc. — to embed key metrics inline. These values are computed hundreds of lines below, but CalcMark resolves forward references automatically after all calculations run. When any assumption changes, the summary updates with it.

```text
| Metric | Baseline | Challenged |
|--------|----------|------------|
| Revenue | {{total_rev}} | {{sb_total_rev}} |
| Gross Margin | {{gross_margin}} | {{sb_gross_margin}} |
| Team Size | {{team_hc}} | {{team_hc}} |
| Utilization | {{target_util}} | {{sb_util}} |
```

**CalcMark features:** Template interpolation `{{var}}`; forward references.

---

## Globals and Assumptions

Two values appear throughout the model — working days per year (250 = 52 weeks × 5 days − 10 holidays) and hours per day (standard US full-time). Rather than repeat them, the document declares them once in frontmatter as globals.

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

The team is tiered by seniority. The tier mix affects both cost structure and the types of work the team can credibly execute. This model uses a 12-person billable team — small enough to run as a single practice, large enough to staff a diverse engagement portfolio.

Headcount is tracked in `people` — a custom unit that makes the staffing model self-documenting and flows naturally into cost calculations.

```calcmark
seniors = 3 people
mids = 5 people
juniors = 4 people
billable_hc = sum of seniors, mids, juniors
```

Support roles (practice management, ops, enablement) are typically 15–20% of billable headcount in mid-market SaaS PS organizations (SPI Research, 2023 PS Maturity Benchmark). We use 18% — one practice lead plus a fractional ops role.

```calcmark
mgmt_rate = 18%
mgmt_hc = billable_hc * mgmt_rate
team_hc = billable_hc + mgmt_hc
```

`billable_hc` evaluates to `12 people`, `mgmt_hc` to `2.16 people`, and `team_hc` to `14.16 people`.

**CalcMark features:** `sum of` NL function; custom `people` unit; percentage normalization on `*`.

---

## Fully-Loaded Cost

"Fully loaded" means base salary plus the full burden of employing someone: payroll taxes, health insurance, retirement contributions, equity grants, and recruiting amortization. A common rule of thumb in US SaaS companies is a 1.25–1.35x multiplier on base salary for individual contributors, rising to 1.3–1.4x for roles with richer equity packages (Carta Total Compensation Report, 2023).

Percentage widening makes burden rates natural: `$145,000 + 32%` means "increase by 32%". Under the hood, CalcMark evaluates this as `$145,000 * (1 + 0.32)`.

Senior consultants (8–12 years experience, can lead enterprise engagements): base salary benchmarked to the 75th percentile for "Solutions Architect" in US metro markets (Levels.fyi, Glassdoor mid-2023 range: $135K–$160K).

```calcmark
sr_base = $145000
sr_burden = 32%
sr_loaded = sr_base + sr_burden
```

Mid-level consultants (4–7 years, can run standard implementations independently): benchmarked to median "Implementation Consultant" (Glassdoor range: $90K–$115K).

```calcmark
mid_base = $100000
mid_burden = 30%
mid_loaded = mid_base + mid_burden
```

Junior consultants (1–3 years, execution-focused, paired with seniors on complex work): benchmarked to entry-level "Technical Consultant" (Glassdoor range: $65K–$80K).

```calcmark
jr_base = $72000
jr_burden = 28%
jr_loaded = jr_base + jr_burden
```

Management fully-loaded cost includes higher equity grants and assumes a Director-level role — blended across a Practice Director ($180K base) and fractional VP allocation.

```calcmark
mgmt_loaded = $220000
```

`sr_loaded` = `$191.4K`, `mid_loaded` = `$130K`, `jr_loaded` = `$92.16K`.

**CalcMark features:** Percentage widening on `+` (value + pct = value increased by pct); currency arithmetic.

---

## Total Labor Cost

Multiplying a `people` quantity by a currency uses Quantity × Currency coercion: `3 people * $191,400` yields `$574,200`. The unit is stripped and the dollar value propagates.

```calcmark
sr_cost = seniors * sr_loaded
mid_cost = mids * mid_loaded
jr_cost = juniors * jr_loaded
mgmt_cost = mgmt_hc * mgmt_loaded
```

```calcmark
labor_cost = sum of sr_cost, mid_cost, jr_cost, mgmt_cost
```

`labor_cost` evaluates to `$2.07M`.

**CalcMark features:** Quantity × Currency coercion; `sum of` NL function.

---

## Non-Labor Overhead

Beyond labor, services organizations carry travel, tooling, training, and allocated infrastructure costs. Each scales with headcount, not some fixed number.

### Travel & Expenses

Travel & Expenses (T&E) — flights, hotels, meals, and ground transport for on-site client engagements — are the largest variable non-labor cost. The average US business trip runs $1,293–$1,771 (GBTA, 2023). Implementation consultants doing on-site delivery average 8–10 trips/year; managers travel less frequently.

```calcmark
billable_trips = 8
billable_trip_cost = $1500
billable_te = billable_trips * billable_trip_cost

mgmt_trips = 2.5
mgmt_trip_cost = $1400
mgmt_te = mgmt_trips * mgmt_trip_cost
```

Not all T&E is a pure loss. Enterprise contracts typically allow the firm to bill a portion of travel directly to the customer as a reimbursable line item on the invoice. The recovery rate is the fraction billed back — 40% is typical when the engagement book is a mix of time-and-materials work (where travel is reimbursed) and fixed-price projects (where travel costs are absorbed by the firm). Percentage widening makes the deduction natural: `te_gross - te_recovery` means "reduce the gross amount by 40%".

```calcmark
te_recovery = 40%
te_gross = (billable_hc * billable_te) + (mgmt_hc * mgmt_te)
te_net = te_gross - te_recovery
```

Gross T&E is `$151.56K`; after billing 40% back to customers, the net cost to the company is `$90.94K`.

**CalcMark features:** Percentage widening on `-` (value - pct = value reduced by pct); Quantity × Currency coercion.

### Tooling & Software

A Professional Services Automation (PSA) tool handles resource management, time tracking, project billing, and utilization reporting — the operational backbone of any PS org. A mid-market PSA (Kantata/Mavenlink, Certinia) plus collaboration stack (Slack, Zoom, Google Workspace) runs $2,500–$4,000 per person per year (G2 pricing data, 2023).

```calcmark
tooling_pp = $3200
tooling = team_hc * tooling_pp
```

### Training & Enablement

Consultants require continuous certification and product enablement as the platform evolves. The Association for Talent Development (ATD) reports a cross-industry average of $1,220 per employee; PS teams with active product certification requirements typically spend $1,500–$2,000 per person.

```calcmark
training_pp = $1800
training = team_hc * training_pp
```

### Allocated Overhead

This captures the PS team's proportional share of company-wide General & Administrative (G&A) expenses — the corporate functions that support every team: HR, legal, finance, IT support, and office/occupancy costs. Rather than track these line by line, companies typically allocate a per-head charge proportional to headcount. For a SaaS company where PS operates as a cost center with its own P&L, $3,000–$5,000 per person is typical (based on SaaS company 10-K G&A disclosures scaled to headcount).

```calcmark
overhead_pp = $3800
overhead = team_hc * overhead_pp
```

### Non-Labor Rollup

```calcmark
non_labor = sum of te_net, tooling, training, overhead
org_cost = labor_cost + non_labor
```

---

## Revenue Capacity

### Available Billable Hours

The starting point for revenue modeling is theoretical capacity: how many hours could the team bill if everything went perfectly? Available hours use the globals declared in frontmatter.

```calcmark
gross_hrs = @globals.working_days * @globals.hours_per_day
```

Not all of those hours are billable. Non-billable time includes internal meetings, product training, pre-sales support, PTO, sick days, and company events. SPI Research (2023) reports 10–15% non-billable time for well-run PS organizations. We use 12%.

```calcmark
non_billable = 12%
net_hrs = gross_hrs - non_billable
```

`gross_hrs` = `2K`, `net_hrs` = `1.76K` (2,000 reduced by 12%).

Where headcount carries the `people` unit, `number()` extracts the raw count for calculations where the unit shouldn't propagate — like computing total hours.

```calcmark
total_capacity = number(billable_hc) * net_hrs
```

**CalcMark features:** `number()` function; `@globals.*` references; percentage widening on `-`.

### Utilization

Utilization is the fraction of net available hours that are actually billed to clients. It is the single most important operational metric in a services business. Industry benchmarks (SPI Research, TSIA) vary by segment:

- **Enterprise SaaS implementation**: 65–75% blended target
- **Advisory/managed services**: 70–80% (more recurring work, smoother loading)
- **Project-based boutique**: 55–70% (longer sales cycles, more bench time)

A business running below 60% has a structural problem — either a demand shortfall, a utilization tracking problem, or a staffing mismatch. Above 80% sustained signals understaffing and burnout risk. 72% is a realistic plan target for a growth-stage SaaS PS team — above median but not heroic.

```calcmark
target_util = 72%
billed_hours = total_capacity * target_util
```

---

## Billing Rates & T&M Revenue

T&M (Time and Materials) is the simplest pricing model: the client pays an hourly rate for each consultant's time. Rates reflect market positioning — enterprise SaaS PS arms typically price at a premium to pure-play systems integrators because they own the product knowledge.

Rates are benchmarked to US mid-market SaaS PS (TSIA rate card survey, 2023): seniors $200–$275/hr, mids $140–$190/hr, juniors $100–$135/hr.

```calcmark
sr_rate = $225
mid_rate = $165
jr_rate = $115
```

Tier-specific utilization varies: seniors carry more pre-sales and methodology work (reducing their billable fraction), while juniors are more purely execution-focused. These reflect observed ranges from TSIA benchmarks.

```calcmark
sr_util = 68%
mid_util = 74%
jr_util = 78%
```

Billable hours per tier multiply headcount (stripped to a raw number with `number()`) by net available hours and the tier's utilization rate.

```calcmark
sr_hours = number(seniors) * net_hrs * sr_util
mid_hours = number(mids) * net_hrs * mid_util
jr_hours = number(juniors) * net_hrs * jr_util
```

Revenue per tier is hours × rate.

```calcmark
sr_tm_rev = sr_hours * sr_rate
mid_tm_rev = mid_hours * mid_rate
jr_tm_rev = jr_hours * jr_rate

total_tm_rev = sum of sr_tm_rev, mid_tm_rev, jr_tm_rev
```

**CalcMark features:** `number()` to extract headcount for hour calculations.

---

## Engagement Packaging

### Why Mix Matters

Not all revenue is created equal. A pure T&M model is transparent and flexible, but it creates lumpy revenue, scope creep disputes, and margin uncertainty. Fixed-price packages trade some upside for predictability. The strategic goal is a portfolio that balances recurring (retainer) revenue for a stable base, fixed-price packages that standardize delivery and protect margin, and T&M for complex work where scope is genuinely unknown.

### Implementation Packages (Fixed-Price)

Fixed-price packages should be sized so that an average delivery runs at roughly 80–85% of the price — leaving 15–20% margin before overhead allocation. Delivery costs below reflect internal labor plus direct costs based on typical hours-to-complete from historical project data.

**Quick Start** (2–3 weeks, junior + mid pair): standard onboarding and configuration. Priced to compete with self-service but guarantee a successful go-live.

```calcmark
qs_price = $18000
qs_cost = $13500
qs_margin = qs_price - qs_cost
```

**Standard Implementation** (6–8 weeks, mid-led with senior oversight): full integration, data migration, and user training. The workhorse package.

```calcmark
std_price = $55000
std_cost = $42000
std_margin = std_price - std_cost
```

**Enterprise Implementation** (12–16 weeks, senior-led cross-functional team): complex multi-system integration, custom workflows, executive sponsorship.

```calcmark
ent_price = $130000
ent_cost = $95000
ent_margin = ent_price - ent_cost
```

### Advisory & Strategic Engagements

Advisory day rate: senior consultant at $225/hr × 8 hours, rounded up for preparation and travel time. Strategy workshops are 2-day intensive sessions with pre-work and deliverables.

```calcmark
advisory_day_rate = $3200
workshop_price = $22000
workshop_cost = $14000
```

### Success Retainers (Monthly Recurring)

Monthly retainer for ongoing optimization, quarterly business reviews, and priority support. Priced to be accretive after month 3 as delivery effort stabilizes. $6,500/month is competitive with outsourced Customer Success Manager rates. The retainer delivery cost uses `% of` — CalcMark's natural-language percentage syntax.

```calcmark
retainer_monthly = $6500
retainer_annual = retainer_monthly * 12
retainer_delivery = 65% of retainer_annual
retainer_margin = retainer_annual - retainer_delivery
```

`retainer_annual` = `$78K`, delivery cost = `$50.7K`, margin = `$27.3K`.

**CalcMark features:** `% of` natural-language syntax.

### Training (Group, Fixed-Price Per Cohort)

Group training: 1-day instructor-led session for up to 20 users. Delivery cost covers instructor time, materials, and environment setup.

```calcmark
cohort_price = $8500
cohort_cost = $4200
cohort_margin = cohort_price - cohort_cost
```

---

## Annual Volume & Revenue Rollup

These are planned engagement counts for the modeled year. The mix reflects a business with a strong implementation core and a growing retainer base. Volume assumptions are derived from pipeline coverage (3x for implementations, 2x for retainers) and historical close rates.

```calcmark
qs_count = 22
std_count = 14
ent_count = 5
advisory_days = 180
workshops = 8
retainers = 18
cohorts = 24
```

Revenue per engagement type multiplies volume by unit price.

```calcmark
qs_rev = qs_count * qs_price
std_rev = std_count * std_price
ent_rev = ent_count * ent_price
advisory_rev = advisory_days * advisory_day_rate
workshop_rev = workshops * workshop_price
retainer_rev = retainers * retainer_annual
training_rev = cohorts * cohort_price

packaged_rev = sum of qs_rev, std_rev, ent_rev, advisory_rev, workshop_rev, retainer_rev, training_rev
```

Total packaged revenue evaluates to `$4.18M`.

---

## P&L Summary

### Total Revenue

For the P&L, we use the packaged/engagement-based revenue model rather than the T&M capacity model. The T&M figures from the previous section serve as a cross-check on capacity consumption.

```calcmark
total_rev = packaged_rev
```

### Cost of Revenue

Cost of Revenue (COR) — sometimes called Cost of Goods Sold (COGS) — includes the direct costs of delivering engagements: delivery labor and travel. It excludes management overhead and the "practice" costs that sit in operating expenses.

72% of total labor is allocated to delivery, matching the target utilization rate. The remainder covers bench time (consultants between engagements), pre-sales support (scoping calls, demos), and internal projects.

```calcmark
delivery_pct = 72%
delivery_labor = labor_cost * delivery_pct
delivery_te = te_net
total_cor = delivery_labor + delivery_te
```

### Gross Profit & Gross Margin

Gross profit is revenue minus cost of revenue — the money left to cover operating expenses and generate contribution. Gross margin expresses this as a percentage of revenue. Wrap both sides with `number()` to get a dimensionless ratio — `number($X) / number($Y)` gives a plain number, while `$X / number($Y)` would give a currency result.

```calcmark
gross_profit = total_rev - total_cor
gross_margin = number(gross_profit) / number(total_rev)
```

Industry benchmarks for SaaS-attached professional services: below 15% gross margin is struggling, 15–25% is typical, and 25–35% is strong. Pure-play consulting firms run 30–45%, but they carry lower base salaries and less product R&D overhead.

### Operating Expenses

Operating expenses (opex) sit below gross profit on the P&L. These are the costs of running the practice itself — management salaries and the per-person overhead (tooling, training, allocated G&A) calculated earlier.

```calcmark
practice_mgmt = mgmt_cost
practice_overhead = sum of tooling, training, overhead

total_opex = practice_mgmt + practice_overhead
```

### Contribution

Contribution (also called operating income) is what the services business unit delivers to the company after covering all its own costs. Contribution margin is this figure as a percentage of revenue.

```calcmark
contribution = gross_profit - total_opex
contribution_margin = number(contribution) / number(total_rev)
```

Gross profit = `$2.6M` at 62% gross margin. After opex, contribution = `$2M` at 48% contribution margin.

---

## Key Performance Metrics

These are the numbers you should know cold before any board meeting or quarterly business review.

### Efficiency Metrics

Revenue per head and cost per head measure how productively the team is deployed. The blended rate is the effective hourly rate implied by total revenue divided by total billed hours — useful for sanity-checking package pricing. Revenue per labor dollar measures how much top-line revenue each dollar of compensation generates; healthy services businesses target 2.5–4x (TSIA benchmark).

```calcmark
rev_per_hc = total_rev / number(billable_hc)
cost_per_hc = labor_cost / number(billable_hc)
blended_rate = number(total_rev) / (number(billable_hc) * net_hrs * target_util)
rev_per_labor_dollar = number(total_rev) / number(labor_cost)
```

### Capacity Check

This cross-validates that the planned engagement volume is feasible given team size and utilization assumptions. Bench capacity is the fraction of available hours not consumed by planned delivery — a bench below 10% leaves no slack for unexpected demand, attrition, or delivery problems; above 30% signals too much unproductive capacity.

```calcmark
est_delivery_hrs = number(total_rev) / blended_rate
capacity_consumed = est_delivery_hrs / total_capacity
bench = 1 - capacity_consumed
```

### Revenue Mix

Recurring (retainer) revenue as a percentage of total is a leading indicator of business stability and predictability — boards want to see this growing. The per-type breakdowns help spot mix shift when presenting a bridge between periods.

```calcmark
recurring_ratio = number(retainer_rev) / number(total_rev)

impl_rev_pct = number(qs_rev + std_rev + ent_rev) / number(total_rev)
advisory_rev_pct = number(advisory_rev + workshop_rev) / number(total_rev)
training_rev_pct = number(training_rev) / number(total_rev)
```

**CalcMark features:** `number()` for unit-stripping in per-head calculations.

---

## Scenario A — Strong Year

In a strong year, three things go right simultaneously: utilization beats plan (demand exceeds forecast), mix shifts toward higher-margin packages, and the retainer base grows (reducing revenue volatility).

78% utilization is top-quartile performance per SPI benchmarks. 24 retainers represents 6 net-new wins (33% growth over the base of 18). 7 enterprise deals is 2 additional over plan from strong pipeline conversion.

```calcmark
sa_util = 78%
sa_retainers = 24
sa_ent_count = 7

sa_hours = total_capacity * sa_util
sa_retainer_rev = sa_retainers * retainer_annual
sa_ent_rev = sa_ent_count * ent_price
```

The incremental revenue from retainer and enterprise wins above plan.

```calcmark
sa_uplift = (sa_retainers - retainers) * retainer_annual + (sa_ent_count - ent_count) * ent_price
sa_total_rev = total_rev + sa_uplift
```

The cost base does not grow proportionally — fixed and semi-fixed labor costs are already in place. Incremental margin on uplift revenue is modeled at 55% because delivery labor for the additional engagements is already on payroll (they were previously under-utilized).

```calcmark
sa_incr_margin_rate = 55%
sa_incr_margin = sa_uplift * sa_incr_margin_rate
sa_gross_profit = gross_profit + sa_incr_margin
sa_gross_margin = number(sa_gross_profit) / number(sa_total_rev)
```

---

## Scenario B — Challenged Year

In a difficult year, problems compound. The most common failure modes are: utilization miss (sales fell short, pipeline was shallow or deals slipped — you're carrying bench you can't bill), rate pressure (enterprise deals bundled services at deep discounts to close the ARR), mix degradation (smaller/lower-margin engagements instead of enterprise), and delivery overruns (fixed-price engagements blew through estimates).

61% utilization is bottom-quartile, indicating a demand or staffing problem. 18% pricing concession means enterprise deals were heavily discounted to close the software ARR. 14 retainers means 4 churned accounts (22% churn, well above the 10–15% norm). 3 enterprise deals is 2 fewer than plan from pipeline conversion miss.

```calcmark
sb_util = 61%
sb_discount = 18%
sb_retainers = 14
sb_ent_count = 3
```

The discount uses percentage widening — `blended_rate - 18%` reduces the effective rate by 18%.

```calcmark
sb_eff_rate = blended_rate - sb_discount
sb_hours = total_capacity * sb_util
sb_tm_rev = sb_hours * sb_eff_rate
```

```calcmark
sb_retainer_rev = sb_retainers * retainer_annual
sb_ent_rev = sb_ent_count * ent_price
sb_impl_rev = qs_count * qs_price + std_count * std_price
sb_total_rev = sum of sb_retainer_rev, sb_ent_rev, sb_impl_rev, advisory_rev, workshop_rev, training_rev

sb_shortfall = total_rev - sb_total_rev
```

Labor costs don't shrink with utilization — salaries are paid whether people are on the bench or billing clients. This is the core dynamic that makes utilization misses so punishing in a services P&L.

```calcmark
sb_gross_profit = sb_total_rev - total_cor
sb_gross_margin = number(sb_gross_profit) / number(sb_total_rev)
sb_contribution = sb_gross_profit - total_opex
sb_contribution_margin = number(sb_contribution) / number(sb_total_rev)
```

---

## Board Metrics Slate

When presenting to a board, the most useful format is a bridge from prior period to current period that separates the "we sold more" story from the "we charged more" story from the "our mix changed" story. These four metrics should be on every services dashboard:

- **Revenue vs. Plan** — the headline number
- **Gross Margin %** — is the business unit economically sound?
- **Utilization %** — are we running efficiently?
- **Recurring Revenue %** — how much of next year's revenue is already visible?

```calcmark
board_rev_attainment = number(sb_total_rev) / number(total_rev)
board_gross_margin = sb_gross_margin
board_util = sb_util
board_recurring_pct = number(sb_retainer_rev) / number(sb_total_rev)
```

Revenue per head is a proxy for operational efficiency and team leverage. In healthy professional services businesses, revenue per billable head runs 2.5–4x fully-loaded compensation (TSIA benchmark). Below 2x signals rates are too low, utilization is too low, or the cost base is too high.

```calcmark
board_rev_per_head = sb_total_rev / number(billable_hc)
rev_to_comp = number(board_rev_per_head) / number(sr_loaded)
```

---

## CalcMark Features Used

- **Frontmatter globals** — `working_days` and `hours_per_day` declared once, referenced everywhere via `@globals.*`
- **Custom `people` unit** — Headcount tracked as a typed quantity (`3 people`, `12 people`) for self-documenting staffing models
- **Quantity × Currency coercion** — `seniors * sr_loaded` multiplies `3 people` by `$191.4K` to yield `$574.2K`; the unit is stripped and the dollar value propagates
- **`number()` function** — Extracts the raw numeric value from a `people` quantity for calculations where the unit shouldn't carry (hours, per-head ratios)
- **Percentage widening** — `base + rate%` for burden loading, `gross - rate%` for recovery deductions, `rate - discount%` for pricing concessions
- **`% of` syntax** — `65% of retainer_annual` for delivery cost allocation
- **`sum of` NL function** — Readable rollups for headcount, labor cost, revenue, and overhead
- **Currency arithmetic** — Dollar signs propagate through every operation
- **Percentage literals** — `18%`, `72%`, `40%` are self-documenting rate assumptions
- **Template interpolation** — `{{total_rev}}`, `{{gross_margin}}` in the "At a Glance" summary table; forward references resolve after all calculations run, so the summary at the top stays current when assumptions change
