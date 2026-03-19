---
title: "Build vs. Buy: Datacenter Cost Analysis"
date: 2026-03-18
author: Larky
description: >
  Should you build a small dedicated datacenter or rent space in
  someone else's? This document models the full lifecycle cost using
  CalcMark's embedded mode.
---

# Build vs. Buy: Datacenter Cost Analysis

Should you build a small dedicated datacenter or rent space in someone
else's? This analysis models the full lifecycle cost — capital
expenditure, operating expenses, depreciation, and a head-to-head
comparison with colocation.

All costs are in USD. The facility is **1,000 square feet**, a typical
size for a small private datacenter.

---

## Facility Construction

Construction costs for dedicated datacenters typically range from $625
to $1,135 per gross square foot. We use a midpoint estimate:

```calcmark
sqft = 1000
cost_sqft = $875
capex = sqft * cost_sqft
```

The budget is heavily weighted toward electrical and mechanical systems,
not the building shell:

```calcmark
capex = $875000

elec = 42.5% of capex
hvac = 17.5% of capex
fitout = 22.5% of capex
shell = 17.5% of capex
```

## Power Infrastructure

Datacenter construction is often quoted per megawatt (MW) of
commissioned IT load. Standard facilities run $7M–$12M per MW.
Our facility has a modest 200 kW load:

```cm
it_load = 200 kilowatts
load_frac = 1/5
mw_cost = $9.5M
power_capex = load_frac * mw_cost
```

## International Equipment Pricing

Specialized cooling equipment from a European vendor, quoted in euros:

```cm
---
exchange:
  EUR_USD: 1.09
---
eu_cooling = €45000
cooling_usd = eu_cooling in USD
```

## Operating Expenses

Ongoing costs for a small datacenter typically fall between $50,000 and
$100,000 annually, dominated by maintenance (40%), electricity, and
labor:

```calcmark
opex = $75000
maintenance = 40% of opex
```

Operating costs don't stay flat. Assuming 4% annual inflation, here's
where opex lands after 5 and 10 years:

```calcmark
opex_5yr = compound $75000 by 4% over 5 years
opex_10yr = compound $75000 by 4% over 10 years
```

## Equipment Depreciation

Servers lose value quickly. Declining-balance depreciation at 20%/year
over a 5-year refresh cycle:

```cm
servers = depreciate $200000 by 20% over 5 years
```

Cooling systems last longer — a $45,000 unit at 15%/year with a $5,000
salvage floor:

```cm
cooling = depreciate $45000 by 15% over 10 years to $5000
```

## Colocation Alternative

Colocation means renting space in a third-party facility. A typical
10-cabinet setup runs about £2,200/month:

```cm
---
exchange:
  GBP_USD: 1.27
---
colo_monthly_gbp = £2200
colo_monthly_usd = colo_monthly_gbp in USD
colo_3yr = $2794/month over 3 years
```

## Build vs. Buy Comparison

Compare 3-year total cost of ownership:

```calcmark
capex = $875000
opex = $75000

build_tco = capex + (opex * 3)
colo_3yr = $2794/month over 3 years
savings = colo_3yr - build_tco
build_is_cheaper = build_tco < colo_3yr
```

## Modular Alternative

Modular (prefab) datacenters offer a pay-as-you-grow model. Starting
with one module and adding one per year:

```cm
grow $150000 by $150000 over 4
```

Modular builds typically save 20–30% on upfront capital:

```cm
capex = $875000
modular_savings = 25% of capex
```

## At Scale

What if you're planning a fleet of 1,000 identical facilities?

```cm
---
scale:
  factor: 1000
  unit_categories: [Currency]
---
capex = $875000
opex = $75000
build_tco = capex + (opex * 3)
```

---

## Methodology

- **CapEx benchmarks** sourced from Uptime Institute and Turner &
  Townsend datacenter cost surveys (2024–2025).
- **OpEx ranges** based on industry averages for Tier II facilities
  under 5,000 sq ft[^1].
- **Depreciation** uses declining-balance method per IRS Publication
  946, Modified Accelerated Cost Recovery System (MACRS).
- **Colo pricing** reflects UK market rates for managed colocation
  with 2N power redundancy[^2].

[^1]: Small-facility opex varies significantly by region and local
      utility rates. The $50K–$100K range assumes a US metro area.
[^2]: Colo pricing converted from GBP at the exchange rate defined
      in each block's frontmatter.
