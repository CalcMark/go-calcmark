---
title: "Formatting & Display"
weight: 5
---

CalcMark offers several formatting and display features for rates, dates, napkin-math estimates, precise values, capacity planning, multiplier suffixes, and percentages.

### Rates and the `over` keyword {#rates}

#### Rate Literals

Create rates using the slash syntax:

```calcmark
net_bandwidth = 100 MB/s
salary = $120000/year
load = 1000 req/s
```

#### Rate Accumulation with `over`

Use `over` to calculate the total from a rate over time:

```calcmark
link_speed = 100 MB/s
daily_transfer = link_speed over 1 day

hourly_rate = $75/hour
daily_earnings = hourly_rate over 8 hours
```

#### Rate Conversion

Convert rates to different time units using `convert_rate()`:

```calcmark
convert_rate(1000 req/s, minute)    -> 60000 req/min
convert_rate($120000/year, month)   -> $10000/month
```

#### Rate Arithmetic Widening

When you multiply or divide **by** a rate (rate on the right), CalcMark drops the time denominator and uses the rate's amount. This is called **widening** -- the rate widens into a plain quantity.

When the rate is on the **left** side, it stays a rate. This lets you scale rates naturally with `rate * number`.

```text
posts_rate = 2 posts/week
scaled = posts_rate * 3           -> 6 posts/week  (rate * number → rate)
total  = 3 * posts_rate           -> 6 posts       (number * rate → quantity)
```

The rule is simple: **left operand wins**. If you start with a rate, you get a rate. If you start with a number or quantity and multiply by a rate, you get a number or quantity.

See the [Language Reference: Rate Arithmetic Widening](/docs/language-reference/#rate-arithmetic-widening) for the full type dispatch table.

### Date Arithmetic {#date-arithmetic}

#### Date Literals

```calcmark
project_start = Jan 15 2025
christmas = Dec 25 2025
now = today
```

CalcMark recognizes `today`, `tomorrow`, and `yesterday` as date keywords.

#### Duration Arithmetic

```calcmark
project_start = Jan 15 2025
duration = 12 weeks
project_end = project_start + duration

deadline = Jun 1 2025
launch = deadline - 2 weeks
```

#### The `from` Keyword

```calcmark
7 days from Jan 1 2025   -> Wed, Jan 8, 2025
2 weeks from today       -> (today + 14 days)
```

### Napkin Math {#napkin-math}

The `as napkin` modifier rounds results to 2 significant figures and normalizes units. It adds a `~` prefix to signal the result is an approximation.

**Syntax:** `expression as napkin`

**Works with:** Number, Quantity, Currency, Duration, Rate

```calcmark
432000 MB as napkin                 -> ~420 GB
100 MB/s over 30 days as napkin    -> ~248 TB
$1234567 as napkin                  -> ~$1.2M
```

This is useful for quick back-of-the-envelope calculations where exact precision is not needed.

### Precise Display {#precise-display}

The `as precise` modifier is the opposite of `as napkin`. It shows full float precision, skipping all display rounding. This is useful when you need exact values from unit conversions.

**Syntax:** `expression as precise`

```calcmark
10 meters as feet                  -> 32.8 feet
10 meters as feet as precise       -> 32.808399 feet
```

Explicit unit conversions are rounded by default for readability. Use `as precise` when you need the exact value.

### Capacity Planning {#capacity-planning}

Use the `at...per` syntax to calculate how many units you need:

```calcmark
disks = 10 TB at 2 TB per disk
servers = 10000 req/s at 450 req/s per server
servers_buffered = 10000 req/s at 450 req/s per server with 20% buffer
```

### Multiplier Suffixes {#multiplier-suffixes}

Use K, M, B for large numbers:

```calcmark
users = 10M
revenue = $5B
requests = 100K
```

### Percentages {#percentages}

Percentages work naturally in calculations:

```calcmark
price = $100
discount = 20%
sale_price = price * (1 - discount)

tax_rate = 8.25%
tax = price * tax_rate
```
