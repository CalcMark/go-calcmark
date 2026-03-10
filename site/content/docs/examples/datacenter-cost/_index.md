---
title: "Datacenter Build Cost"
summary: "Full lifecycle cost analysis: CapEx, OpEx, depreciation, growth, and build-vs-colo comparison."
weight: 55
calcmark_build: progressive
calcmark_source: testdata/examples/datacenter-cost.cm
---

Should you build a small dedicated datacenter or rent space in someone else's? This walkthrough models the full lifecycle cost using CalcMark, showcasing exchange rates, growth functions, depreciation, and napkin math along the way.

The complete CalcMark file is available at {{< repo-file path="testdata/examples/datacenter-cost.cm" >}}.

---

## Front Matter

Every CalcMark document can start with YAML front matter. Here we define currency exchange rates and a global variable for facility size that's available throughout the document.

```calcmark
---
exchange:
  USD_EUR: 0.92
  EUR_USD: 1.09
  USD_GBP: 0.79
  GBP_USD: 1.27
globals:
  sqft: 1000
---
```

**CalcMark features:** `exchange` front matter for currency conversion rates (`FROM_TO: rate` format); `globals` front matter for document-wide variables.

---

## Facility Sizing & Baseline CapEx

Construction costs for dedicated datacenters typically range from $625 to $1,135 per gross square foot. We use a midpoint estimate for a 1,000 sqft facility:

```calcmark
cost_sqft = $875
capex = @globals.sqft * cost_sqft
```

`@globals.sqft` references the `sqft` variable from the front matter `globals` block. The `@globals.` prefix makes it clear the value comes from frontmatter, not from a local definition. Multiplying a number by a currency preserves the currency unit: `1000 * $875 = $875,000`.

**CalcMark features:** `@globals.name` directive references; currency arithmetic.

---

## CapEx Breakdown by Component

The budget is heavily weighted toward electrical and mechanical systems, not the building shell. The `X% of value` syntax calculates a percentage directly:

```calcmark
elec = 42.5% of capex
hvac = 17.5% of capex
fitout = 22.5% of capex
shell = 17.5% of capex

Sanity check -- components should sum to total:

components = elec + hvac + fitout + shell
```

The percentages (42.5 + 17.5 + 22.5 + 17.5 = 100%) mirror industry benchmarks: electrical systems dominate at 40-45%, followed by building fit-out, HVAC/cooling, and the land/shell.

**CalcMark features:** `X% of value` natural language syntax; currency addition.

---

## Tier Comparison

Uptime Institute tiers define redundancy levels. Tier II (partial redundancy) and Tier III (concurrently maintainable) differ dramatically in cost:

```calcmark
tier_ii = $2300 * @globals.sqft
tier_iii = $7700 * @globals.sqft
tier_mult = 7700 / 2300
```

Moving to Tier III more than triples the cost per square foot. For AI-optimized facilities requiring specialized liquid cooling, costs can reach $20M+ per MW.

**CalcMark features:** Currency multiplication; plain division for ratios.

---

## Power Infrastructure

Datacenter construction is often quoted per megawatt (MW) of commissioned IT load. A small 1,000 sqft facility might support 200 kW of IT load:

```calcmark
it_load = 200 kilowatts

A standard 1 MW facility costs $7M-$12M. Our load as a fraction of 1 MW:

load_frac = 200 / 1000
mw_cost = $9.5M
power = load_frac * mw_cost

For reference, 1000 kilowatts converts cleanly:

1000 kilowatts in megawatts
```

CalcMark knows SI power units and can convert between them using the `in` keyword. The `$9.5M` syntax uses a multiplier suffix -- CalcMark supports `k` (thousand), `M` (million), and `B` (billion).

**CalcMark features:** Quantities with units (`kilowatts`); `in` unit conversion; multiplier suffixes (`$9.5M`); markdown prose between calculations.

---

## International Equipment Pricing

Specialized cooling equipment from a European vendor is quoted in euros. The exchange rates defined in the front matter make conversion seamless:

```calcmark
eu_cooling = €45000
cooling = eu_cooling in USD
```

The `in USD` expression looks up the `EUR_USD` rate from the front matter (1.09) and multiplies: `€45,000 * 1.09 = $49,050`.

**CalcMark features:** `€` currency literal; `in USD` exchange rate conversion; front matter `exchange` rates.

---

## Operating Expenses

Ongoing costs for a small datacenter typically fall between $50,000 and $100,000 annually:

```calcmark
opex = $75000
maint = 40% of opex

Electricity is often quoted per kilowatt-hour. At $0.10/hour for our
200 kW load, the annual electricity cost accumulates quickly:

annual_elec = $0.10/hour over 1 year

What does that rate look like per day?

daily_elec = $0.10/hour per day
```

The `$0.10/hour` is a rate literal. The `over 1 year` syntax accumulates that rate across the time period (0.10 * 8,760 hours = $876). The `per day` syntax converts the rate to a different time unit without accumulating.

**CalcMark features:** Rate literals (`$0.10/hour`); `rate over duration` for accumulation; `rate per unit` for rate conversion; `X% of value` for percentages.

---

## OpEx Growth Projection

Operating costs don't stay flat. The `compound` function models exponential growth -- here, 4% annual inflation over 5 and 10 years:

```calcmark
opex_5yr = compound $75000 by 4% over 5 years
opex_10yr = compound $75000 by 4% over 10 years
```

This is the natural language form of `compound(principal, rate, periods)`. After 10 years at 4% inflation, $75K in annual opex grows to over $111K.

**CalcMark features:** `compound` function with natural language syntax (`compound X by Y% over Z years`).

---

## Equipment Depreciation

The `depreciate` function models declining-balance depreciation. Servers lose value quickly at 20% per year:

```calcmark
servers = depreciate $200000 by 20% over 5 years

cooling_depr = depreciate $45000 by 15% over 10 years to $5000
```

The `to $5000` clause sets a salvage floor -- the asset won't depreciate below that value regardless of the rate and time. After 5 years, $200K in servers is worth only ~$65.5K. The cooling unit drops from $45K toward the $5K floor over 10 years.

**CalcMark features:** `depreciate` function with natural language syntax; salvage floor with `to` keyword.

---

## Build vs. Colocation

Colocation means renting space in a third-party facility. A typical 10-cabinet setup in the UK runs about £2,200/month:

```calcmark
colo_mo = £2200
colo_usd = colo_mo in USD

Using the rate-over-time syntax to accumulate monthly costs:

colo_3yr = $2794/month over 3 years

Compare with building your own over the same 3 years:

build_tco = capex + (opex * 3)
savings = colo_3yr - build_tco
build_cheaper = build_tco < colo_3yr
```

The `$2794/month over 3 years` accumulates monthly colocation fees into a total. Because the rate uses a currency (`$`), the result is a proper currency value that can be subtracted from other currencies. Building costs ~$1.1M over 3 years vs ~$102K for colocation. For a small facility, colo wins decisively.

**CalcMark features:** `£` currency literal; `in USD` exchange rate conversion; `rate over duration` for cost accumulation; comparison operators (`<`) producing booleans.

---

## Modular Alternative

Modular (prefab) datacenters offer a pay-as-you-grow model. The `grow` function models linear growth -- adding one $150K module per year:

```calcmark
grow $150000 by $150000 over 4

modular_savings = 25% of capex
```

Unlike `compound` (exponential), `grow` adds a fixed increment each period. Four modules at $150K each totals $750K, and modular builds typically save 20-30% vs traditional construction.

**CalcMark features:** `grow` function with natural language syntax (`grow X by Y over Z`); linear vs exponential growth.

---

## Executive Summary

The `as napkin` modifier rounds any value to 2 significant figures with a human-readable suffix (K, M, B):

```calcmark
capex as napkin
opex as napkin
build_tco as napkin
colo_3yr as napkin
```

Quick-reference numbers for stakeholder conversations: ~$880K to build, ~$75K/year to run, ~$1.1M total over 3 years vs ~$100K for colocation.

**CalcMark features:** `as napkin` for human-readable rounding.

---

## Features Demonstrated

This example showcases the following CalcMark features:

- **Front matter** -- `exchange` rates and `globals` variables
- **Currency literals** -- `$`, `€`, `£` symbols
- **Exchange rate conversion** -- `value in USD`, `value in EUR`
- **Multiplier suffixes** -- `$9.5M`, `$2300`
- **Percentage calculations** -- `42.5% of capex`
- **Unit conversion** -- `1000 kilowatts in megawatts`
- **Rate literals** -- `$0.10/hour`
- **Rate accumulation** -- `$0.10/hour over 1 year`, `$2794/month over 3 years`
- **Rate conversion** -- `$0.10/hour per day`
- **`compound()`** -- exponential growth for inflation projection
- **`grow()`** -- linear growth for modular capacity
- **`depreciate()`** -- declining-balance depreciation with salvage floor
- **`as napkin`** -- human-readable rounding for executive summaries
- **Markdown prose** -- headings, paragraphs, and inline comments between calculations

## Try It

{{< repo-file path="testdata/examples/datacenter-cost.cm" >}}

```bash
cm testdata/examples/datacenter-cost.cm
```
