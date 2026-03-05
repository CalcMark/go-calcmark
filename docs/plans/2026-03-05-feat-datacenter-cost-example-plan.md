---
title: "feat: Add datacenter build cost example"
type: feat
status: active
date: 2026-03-05
brainstorm: docs/brainstorms/2026-03-05-datacenter-cost-example-brainstorm.md
---

# feat: Add datacenter build cost example

## Overview

Create a comprehensive datacenter build cost analysis example that showcases CalcMark's financial modeling features (compound, grow, depreciate, exchange rates, napkin, accumulate) through a real-world scenario. Produces two files: a standalone `.cm` document and an interleaved article page for the site.

## Proposed Solution

### Deliverable 1: `testdata/examples/datacenter-cost.cm`

A standalone, runnable CalcMark document with embedded markdown prose. Follows the pattern of `system-sizing.cm` — headings, short prose paragraphs between calculations, readable on its own.

**Front matter:**

```cm
---
exchange:
  USD_EUR: 0.92
  USD_GBP: 0.79
globals:
  facility_sqft: 1000
---
```

**Sections (in document order):**

1. **Title & intro prose** — what we're modeling
2. **Facility Sizing & Baseline CapEx** — `cost_per_sqft`, total build, multiplier suffixes (`$2.3M`)
3. **CapEx Breakdown by Component** — electrical (40-45%), HVAC (15-20%), fit-out (20-25%), land/shell (15-20%) using `X% of total` NL syntax
4. **Tier Comparison (II vs III)** — Tier II at ~$2300/sqft vs Tier III at ~$7700/sqft, cost multiplier
5. **Power Infrastructure** — cost per MW ($7M-$12M), kW→MW unit conversion using `in megawatts`
6. **International Equipment Pricing** — European cooling vendor in EUR (`€`), convert `in USD` using exchange rates; UK colo pricing in GBP, convert `in USD`
7. **Operating Expenses (OpEx)** — annual costs, electricity rate as `$/kW` rate, maintenance percentage
8. **OpEx Growth Over Time** — `compound` annual OpEx by inflation rate over 10 years (NL form)
9. **Equipment Depreciation** — `depreciate` servers and cooling equipment with salvage floor (NL form with `to`)
10. **Build vs. Colocation** — 3-year TCO: `rate over duration` for monthly colo costs, comparison with build + OpEx
11. **Modular/Prefab Alternative** — `grow` capacity in increments (NL form), 20-30% savings
12. **Executive Summary** — `as napkin` for all key figures

**CalcMark features demonstrated:**

| Feature | Where Used |
|---------|-----------|
| Front matter `exchange` | Sections 6, 10 (USD/EUR, USD/GBP rates) |
| Front matter `globals` | `facility_sqft` referenced throughout |
| `compound()` NL form | Section 8 (OpEx inflation projection) |
| `grow()` NL form | Section 11 (modular capacity additions) |
| `depreciate()` NL form | Section 9 (server/cooling depreciation with salvage) |
| `rate over duration` | Section 10 (colo costs over 3 years) |
| `in EUR` / `in USD` | Sections 6, 10 (international vendor/colo conversion) |
| `as napkin` | Section 12 (summary rounding) |
| `X% of value` NL syntax | Section 3 (CapEx component breakdown) |
| `value in unit` conversion | Section 5 (kW to MW power conversion) |
| Currency literals (`$`, `€`, `£`) | Throughout |
| Multiplier suffixes (`M`) | Sections 2, 5 |
| Rates (`$X/month`) | Sections 7, 10 |
| Comparison operators | Section 10 (build vs colo decision) |
| Markdown prose | Throughout |

**Concrete `.cm` sketch (key calculations):**

```cm
---
exchange:
  USD_EUR: 0.92
  USD_GBP: 0.79
globals:
  sqft: 1000
---

# Datacenter Build Cost Analysis

...prose defines abbreviations: sqft (square feet), capex (capital expenditure),
opex (operating expenditure), colo (colocation), tco (total cost of ownership)...

## Facility Sizing & Baseline CapEx

cost_sqft = $875
capex = sqft * cost_sqft

## CapEx Breakdown

elec = 42.5% of capex
hvac = 17.5% of capex
fitout = 22.5% of capex
shell = 17.5% of capex

## Tier Comparison

tier_ii = $2300 * sqft
tier_iii = $7700 * sqft
tier_mult = tier_iii / tier_ii

## Power Infrastructure

it_load = 200 kilowatts
load_mw = it_load in megawatts
mw_cost = $9.5M
power = load_mw * mw_cost

## International Equipment

eu_cooling = €45000
cooling = eu_cooling in USD

## Operating Expenses

opex = $75000
elec_rate = $0.12
maint = 40% of opex

## OpEx Growth (5 years at 4% inflation)

compound $75000 by 4% over 5 years

## Equipment Depreciation

depreciate $200000 by 20% over 5 years
depreciate $45000 by 15% over 10 years to $5000

## Build vs Colocation

colo_mo = £2200
colo_usd = colo_mo in USD
colo_3yr = colo_usd * 36
build_tco = capex + (opex * 3)
cheaper = build_tco < colo_3yr

## Modular Alternative

grow $150000 by $150000 over 4

## Summary

capex as napkin
opex as napkin
```

> **Note:** This is a sketch. Exact variable names, prose, and values will be refined during implementation to ensure all calculations parse and evaluate correctly.

### Deliverable 2: `site/content/docs/examples/datacenter-cost.md`

An **interleaved article** that breaks the `.cm` file into annotated snippets. Unlike other example pages that embed the full file in one block, this page presents snippets with explanations.

**Hugo frontmatter:**

```yaml
---
title: "Datacenter Build Cost"
summary: "Full lifecycle cost analysis: CapEx, OpEx, depreciation, growth, and build-vs-colo comparison."
weight: 55
---
```

**Article structure:**

Each section follows this pattern:

```markdown
## Section Title

[1-3 sentences of domain context about datacenter costs]

```cm
[CalcMark snippet for this section]
`` `

**CalcMark features used:** brief callout of what's demonstrated

---
```

**Sections mirror the .cm file** (12 sections as listed above), each with:
- Domain explanation (what the numbers mean in datacenter context)
- The CalcMark snippet
- Feature callout (which CalcMark syntax is showcased)

**Footer sections:**

```markdown
## Features Demonstrated

[Bulleted list of all CalcMark features with links to language reference]

## Try It

```bash
cm testdata/examples/datacenter-cost.cm
`` `
```

### Deliverable 3: Update `site/content/docs/examples/_index.md`

Add the new example to the index page under "Real-World Use Cases":

```markdown
- [Datacenter Build Cost](datacenter-cost/) -- Full lifecycle cost analysis with growth, depreciation, and exchange rates
```

## Acceptance Criteria

- [ ] `testdata/examples/datacenter-cost.cm` runs successfully with `cm` (no parse or eval errors)
- [ ] `.cm` file uses all target features: compound, grow, depreciate, accumulate, exchange rates, napkin, in/as conversions, globals, percentages, rates, multiplier suffixes
- [ ] `.cm` file is standalone readable with markdown prose between calculations
- [ ] `site/content/docs/examples/datacenter-cost.md` presents interleaved snippets with explanations
- [ ] Every snippet in the article page is a verbatim excerpt from the `.cm` file
- [ ] Examples index (`_index.md`) links to the new page
- [ ] `task test` passes (no regressions)
- [ ] Dollar amounts and percentages in the example are realistic per the source material

## Implementation Steps

1. **Write `testdata/examples/datacenter-cost.cm`** — the standalone CalcMark document
2. **Validate with `cm`** — run `cm testdata/examples/datacenter-cost.cm` to confirm it parses and evaluates
3. **Write `site/content/docs/examples/datacenter-cost.md`** — the interleaved article
4. **Update `site/content/docs/examples/_index.md`** — add link to index
5. **Run `task test`** — verify no regressions
6. **Run `task quality`** — verify quality gates pass

## References

- Brainstorm: `docs/brainstorms/2026-03-05-datacenter-cost-example-brainstorm.md`
- Pattern: `testdata/examples/system-sizing.cm` (standalone .cm with prose)
- Pattern: `site/content/docs/examples/system-sizing.md` (article page)
- Exchange rate syntax: `testdata/spec/valid/features/exchange_rates.cm`
- Growth functions: `testdata/eval/success/features/growth_functions.cm`
- Learning: `docs/solutions/logic-errors/exchange-rate-frontmatter-validation.md` (exchange rate format)
