---
title: "Rates & Capacity Planning"
summary: "Rate expressions, accumulation with 'over', rate conversion, and capacity planning with 'at...per'."
weight: 30
calcmark_build: progressive
---

From {{< repo-file path="testdata/eval/success/features/rates.cm" >}},
{{< repo-file path="testdata/eval/success/features/rate_functions.cm" >}},
and {{< repo-file path="testdata/eval/success/features/capacity_at.cm" >}}.

## Rate Expressions

CalcMark supports rates with any time unit. Both `/` and `per` syntaxes work.

```calcmark
# Bandwidth rates
r1 = 100 MB/s
r2 = 1 GB/sec
r3 = 10 TB per second

# Cost rates
r4 = $0.10/h
r5 = $100 per hour
r6 = $50/mo
r7 = $10000 per year

# Request rates
r8 = 1000 req/min
r9 = 50000 requests per minute
r10 = 1500000 req/s

# Arbitrary units
r11 = 20 apples/sec
r12 = 100 widgets per minute
r13 = 1000 cars/day
```

## Accumulate: `rate over time`

Calculate totals from a rate over a time period.

```calcmark
100 MB/s over 1 day
5 GB/day over 1 year
$0.10/hour over 30 days
1000 req/s over 1 hour
50 KB/s over 1 hour
100 widgets/hour over 1 week
$5/day over 365 days
```

## Convert Rate: `rate per target_unit`

Convert a rate to a different time unit.

```calcmark
5 million/day per second
10 TB/month per second
1000 req/s per minute
100k/hour per second
```

## Rate Unit Conversion with `in`

Convert both quantity and time units.

```calcmark
speed1 = 10 m/s in inch/s
speed2 = 100 km/h in mile/h
rate1 = 60 m/s in m/min
speed3 = 1 km/h in m/s
data_rate = 10 MB/day in seconds
```

## Capacity Planning: `X at Y per unit`

Calculate how many units are needed.

```calcmark
# Basic capacity planning
storage_disks = 10 TB at 2 TB per disk
web_servers = 10000 req/s at 450 req/s per server
network_connections = 100 MB/s at 10 MB/s per connection
fruit_crates = 100 apples at 30 per crate
production_batches = 100 at 25 per batch
```

## Capacity with Buffer Percentages

```calcmark
buffered_disks = 10 TB at 2 TB per disk with 10% buffer
buffered_servers = 10000 req/s at 450 req/s per server with 20% buffer
large_buffer = 100 at 50 per unit with 100% buffer
```

## Capacity with Slash Syntax

```calcmark
slash_disks = 10 TB at 2 TB/disk
slash_batches = 100 at 25/batch
slash_with_buffer = 10 GB/day at 2 GB/disk with 30% buffer
```

## Edge Cases

```calcmark
# Demand less than capacity (minimum 1 unit)
minimum_units = 5 at 10 per unit

# Exact division
exact_division = 100 at 50 per container
```

### What This Demonstrates

- Rate expressions with `/` and `per` syntax
- All time units: seconds, minutes, hours, days, weeks, months, years
- `over` keyword for accumulation
- `per` keyword for rate conversion
- `at...per` syntax for capacity planning
- Buffer percentages with `with X% buffer`
- Slash syntax alternative: `2 TB/disk` instead of `2 TB per disk`
- Ceiling division (always rounds up)
