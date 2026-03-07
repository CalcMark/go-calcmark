---
title: "Examples"
summary: "Worked examples showing CalcMark in action."
weight: 50
---

Explore complete CalcMark files that demonstrate real-world use cases. Run any example with:

```bash
cm testdata/examples/<filename>.cm

# Or open directly from GitHub — no clone required:
cm remote --http https://raw.githubusercontent.com/CalcMark/go-calcmark/refs/heads/main/testdata/examples/<filename>.cm
```

## Real-World Use Cases

- [Household Budget](household-budget/) -- Monthly budget with rates, conversions, and napkin math
- [Job Offer Comparison](job-offer/) -- Comparing compensation packages
- [Project Workback](project-workback/) -- Sprint planning with dates and durations
- [Recipe Scaling](recipe-scaling/) -- Unit conversions and proportional scaling
- [System Sizing](system-sizing/) -- Infrastructure capacity planning
- [Datacenter Build Cost](datacenter-cost/) -- Full lifecycle cost analysis with growth, depreciation, and exchange rates
- [Investment & Growth](investment-growth/) -- Compound growth, depreciation, and linear growth for financial modeling
- [Home Renovation](home-renovation/) -- Averages, square roots, constants, booleans, and rate conversions

## Feature Reference

These examples are drawn from the project's {{< repo-file path="testdata/eval/success/features" type="tree" text="golden test suite" >}} and demonstrate every language feature:

- [Functions & Natural Language](functions-and-nl/) -- All function call styles
- [Network & Storage](network-and-storage/) -- Latency, throughput, read/seek, compression
- [Rates & Capacity Planning](rates-and-capacity/) -- Rate expressions, `over`, `at...per`
- [Napkin Math](napkin-math/) -- Quick estimation with `as napkin`
- [Dates & Durations](dates-and-durations/) -- Date keywords, literals, arithmetic
