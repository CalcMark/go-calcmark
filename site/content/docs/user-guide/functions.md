---
title: "Functions & NL Syntax"
weight: 4
---

CalcMark provides built-in functions for common calculations, with natural language (NL) alternatives for many of them. This page covers all functions, their NL forms, and specialized domain functions for networking, storage, and growth modeling.

### Built-in Functions {#built-in-functions}

{{< feature-table category="function" >}}

See the [Function Reference](#function-reference) below for detailed examples of every function, including natural language forms.

### Function Reference {#function-reference}

Detailed examples for every built-in function, showing both function-call and natural language syntax.

#### `avg` / `average of` {#avg-examples}

```calcmark
avg(10, 20, 30)                    -> 20
avg(1, 2, 3, 4, 5)                -> 3
average of 100, 200, 300           -> 200  (NL form)
avg($100, $200, $300)              -> $200.00  (preserves currency)
```

#### `sum` / `sum of` {#sum-examples}

```calcmark
sum($100, $200, $300)              -> $600.00
sum(10, 20, 30)                    -> 60
sum of total_a, total_b, total_c   -> (NL form, sums variables)
```

#### `number` {#number-examples}

```calcmark
number(10 kg)                      -> 10
number($250)                       -> 250
number(3.5 hours)                  -> 3.5
```

Strips the unit or currency from any value, returning a plain number. Useful when you need the raw numeric value for a calculation that doesn't preserve units.

#### `sqrt` / `square root of` {#sqrt-examples}

```calcmark
sqrt(16)                           -> 4
sqrt(2)                            -> 1.4142...
square root of 144                 -> 12  (NL form)
sqrt($100)                         -> $10.00  (preserves currency)
```

#### `accumulate` / `over` {#accumulate-examples}

```calcmark
accumulate(100 MB/s, 1 hour)       -> 360000 MB
accumulate($75/hour, 8 hours)      -> $600
100 MB/s over 1 day                -> ~8.64 TB  (keyword form)
$120000/year over 1 month          -> $10000
```

#### `convert_rate` {#convert-rate-examples}

```calcmark
convert_rate(1000 req/s, minute)   -> 60000 req/min
convert_rate($120000/year, month)  -> $10000/month
```

#### `capacity` / `at...per` {#capacity-examples}

```calcmark
capacity(10 TB, 2 TB, disk)              -> 5 disks
capacity(10000 req/s, 500 req/s, server) -> 20 servers
10 TB at 2 TB per disk                   -> 5 disks  (NL form)
10000 req/s at 450 req/s per server with 20% buffer -> 27 servers
```

#### `downtime` {#downtime-examples}

```calcmark
downtime(0.999, year)              -> 8.76 hour
downtime(0.999, month)             -> 43.2 minute
downtime(0.9999, month)            -> 4.32 minute
```

#### `rtt` {#rtt-examples}

```calcmark
rtt(local)                         -> 0.5 ms
rtt(regional)                      -> 10 ms
rtt(continental)                   -> 50 ms
rtt(global)                        -> 150 ms
```

#### `throughput` {#throughput-examples}

```calcmark
throughput(gigabit)                 -> 125 MB/s
throughput(ten_gig)                -> 1.22 GB/s
throughput(wifi)                   -> 12.5 MB/s
throughput(five_g)                 -> 50 MB/s
```

#### `transfer_time` / `transfer...across` {#transfer-time-examples}

```calcmark
transfer_time(1 GB, regional, gigabit)   -> ~8 seconds
transfer_time(500 MB, continental, gigabit)
transfer 1 GB across regional gigabit    -> (NL form)
transfer 100 MB across local ten_gig

data = 500 KB
transfer data across continental ten_gig -> (variable reference)
```

#### `read` / `read...from` {#read-examples}

```calcmark
read(100 MB, ssd)                  -> ~0.18 seconds
read(1 GB, nvme)                   -> ~0.29 seconds
read 100 MB from ssd               -> (NL form)
read 10 GB from pcie_ssd

data = 5 MB
read data from nvme                -> (variable reference)
```

#### `seek` {#seek-examples}

```calcmark
seek(hdd)                          -> 10 ms
seek(ssd)                          -> 0.1 ms
seek(nvme)                         -> 0.01 ms
db_query_hdd = seek(hdd) + read(5 MB, hdd)
```

#### `compress` / `compress...using` {#compress-examples}

```calcmark
compress(1 GB, gzip)               -> ~333 MB
compress(500 MB, lz4)              -> ~250 MB
compress(2 GB, zstd)               -> 585 MB
compress 1 GB using gzip            -> (NL form)
compress 500 MB using lz4

data = 1 GB
compress data using zstd            -> (variable reference)
```

#### `compound` / `compound...by...over` {#compound-examples}

```calcmark
compound($1000, 5%, 10)                              -> $1628.89
compound(500 customers, 20%, 12)                     -> 4458.05 customers
compound($1000, 5%, 10 years, monthly)               -> $1647.01
compound($1000, 5%, 10 years, quarterly)             -> $1643.62
compound $1000 by 5% over 10 years                   -> $1628.89  (NL form)
compound $1000 by 5% monthly over 10 years
compound $1000 by 5% per month over 12 months
```

#### `grow` / `grow...by...over` {#grow-examples}

```calcmark
grow(100, 20, 5)                   -> 200
grow($500, $100, 36)               -> $4100.00
grow 100 by 20 over 5 months       -> 200  (NL form)
```

#### `depreciate` / `depreciate...by...over` {#depreciate-examples}

```calcmark
depreciate($50000, 15%, 5)                -> $22185.27
depreciate($50000, 15%, 20, $5000)        -> $5000.00  (salvage floor)
depreciate $50000 by 15% over 5 years     -> $22185.27  (NL form)
depreciate $50000 by 15% over 5 years to $5000
```

#### Unit Handling in Functions

**Same units are preserved:**

```calcmark
avg($100, $200, $300) -> $200.00
sqrt($100) -> $10.00
```

**Mixed units are dropped:**

```calcmark
avg($100, €200) -> 150  (no units)
average of $50, €100, £150 -> 100  (no units)
```

### Natural Language Syntax {#natural-language-syntax}

CalcMark supports natural language forms for many functions. These are equivalent to the function-call syntax. Arguments can be literal values (`100 MB`) or variable references (`data`).

#### Function Aliases

| Natural Language | Equivalent | Example |
|-----------------|------------|---------|
| `average of X, Y, Z` | `avg(X, Y, Z)` | `average of 10, 20, 30` |
| `sum of X, Y, Z` | `sum(X, Y, Z)` | `sum of $100, $200, $300` |
| `square root of X` | `sqrt(X)` | `square root of 144` |
| `read X from Y` | `read(X, Y)` | `read 100 MB from ssd` |
| `compress X using Y` | `compress(X, Y)` | `compress 1 GB using gzip` |
| `transfer X across Y Z` | `transfer_time(X, Y, Z)` | `transfer 1 GB across regional gigabit` |
| `compound X by Y% over Z` | `compound(X, Y%, Z)` | `compound $1000 by 5% over 10 years` |
| `compound X by Y% monthly over Z` | `compound(X, Y%, Z, monthly)` | `compound $1000 by 5% monthly over 10 years` |
| `compound X by Y% per P over Z` | `compound(X, Y%, Z, P)` | `compound $1000 by 5% per month over 12 months` |
| `compound X by Y% compounded F over Z` | `compound(X, Y%, Z, compounded F)` | `compound $1000 by 12% compounded monthly over 10 years` |
| `grow X by Y over Z` | `grow(X, Y, Z)` | `grow 100 by 20 over 5 months` |
| `depreciate X by Y% over Z` | `depreciate(X, Y%, Z)` | `depreciate $50000 by 15% over 5 years` |
| `depreciate X by Y% over Z to W` | `depreciate(X, Y%, Z, W)` | `depreciate $50000 by 15% over 5 years to $5000` |

#### Capacity Planning Syntax {#capacity-syntax}

The `at...per` syntax is a natural language form for the `capacity()` function:

```text
demand at capacity per unit
demand at capacity per unit with N% buffer
```

Examples:

```calcmark
10 TB at 2 TB per disk                         -> 5 disks
10000 req/s at 450 req/s per server             -> 23 servers
10000 req/s at 450 req/s per server with 20%    -> 27 servers
100 apples at 30 per crate                      -> 4 crates
```

The slash syntax also works: `10 TB at 2 TB/disk`.

#### Rate Accumulation with `over` {#over}

The `over` keyword accumulates a rate over a time duration:

```text
rate over duration
```

This is equivalent to `accumulate(rate, duration)`:

```calcmark
100 MB/s over 1 day         -> total data transferred
$75/hour over 8 hours       -> daily earnings
1000 req/s over 1 hour      -> total requests
```

#### Rate Conversion with `per`

The `per` keyword in a rate context creates a rate literal:

```calcmark
1000 requests per second    -> 1000 req/s
$50 per hour                -> $50/hour
```

### Network Functions {#network-functions}

#### Round-Trip Time

```calcmark
rtt(local)          -> 0.5 ms   (same datacenter)
rtt(regional)       -> 10 ms    (same region)
rtt(continental)    -> 50 ms    (cross-continent)
rtt(global)         -> 150 ms   (worldwide)
```

#### Throughput

```calcmark
throughput(gigabit)      -> 125 MB/s
throughput(ten_gig)      -> 1.22 GB/s
throughput(hundred_gig)  -> 12500 MB/s
throughput(wifi)         -> 12.5 MB/s
throughput(four_g)       -> 2.5 MB/s
throughput(five_g)       -> 50 MB/s
```

#### Transfer Time

Calculate data transfer time across a network:

```calcmark
transfer_time(1 GB, regional, gigabit)
transfer 1 GB across regional gigabit       (NL form)
```

#### Downtime from Availability

```calcmark
downtime(99.9%, year)     -> 8.76 hour
downtime(99.99%, month)   -> 4.32 minute
```

### Storage Functions {#storage-functions}

#### Read Time

```calcmark
read(1 GB, ssd)
read(1 GB, nvme)
read(1 GB, hdd)
read 100 MB from ssd
```

Storage types and approximate read speeds: SSD ~500 MB/s, NVMe ~3 GB/s, HDD ~100 MB/s.

#### Seek Latency

```calcmark
seek(ssd)       -> 0.1 ms
seek(nvme)      -> 0.01 ms
seek(pcie_ssd)  -> 0.01 ms
seek(hdd)       -> 10 ms
```

#### Compression

```calcmark
compress(1 GB, gzip)     -> 333 MB   (3:1 ratio)
compress(1 GB, lz4)      -> 512 MB   (2:1 ratio)
compress(1 GB, zstd)     -> 293 MB   (3.5:1 ratio)
compress(1 GB, bzip2)    -> 250 MB   (4:1 ratio)
compress(1 GB, snappy)   -> 400 MB   (2.5:1 ratio)
compress(1 GB, none)     -> 1 GB     (1:1, no compression)

compress 1 GB using gzip (NL form)
```

### Growth Functions {#growth-functions}

#### Compound Growth

The 4th argument controls which formula `compound()` uses. This matters -- `monthly` and `month` give different answers:

**Without a frequency modifier** -- simple compound growth, once per year:

```text
A = P × (1 + r)^t
```

```calcmark
compound($1000, 5%, 10)                              -> $1628.89
compound $1000 by 5% over 10 years                   -> $1628.89
```

5% annual rate applied once per year for 10 years. A plain number like `10` means 10 years.

**With a frequency adverb** (`monthly`, `quarterly`, `weekly`, `daily`, `yearly`) -- financial compounding, multiple times per year:

```text
A = P × (1 + r/n)^(n × t)
```

where `r` is the annual rate, `n` is compounding frequency per year, and `t` is years.

```calcmark
compound($1000, 5%, 10, monthly)                     -> $1647.01
compound($1000, 5%, 10, quarterly)                    -> $1643.62
compound($1000, 5%, 24 months, monthly)              -> $1103.43
compound $1000 by 5% monthly over 10 years            -> $1647.01
```

The 5% annual rate is split into 12 monthly applications (5%/12 per month), compounded 120 times over 10 years. More frequent compounding yields higher returns: monthly ($1,647) > quarterly ($1,643) > yearly ($1,629).

**Arguments:**

| # | Name | Description |
|---|------|-------------|
| 1 | principal | Starting amount (number, currency, or quantity) |
| 2 | rate | Annual growth rate as percentage |
| 3 | duration | Time in years. A plain number like `10` means 10 years. Durations like `24 months` are converted to years |
| 4 | modifier | Optional: `monthly`, `quarterly`, `daily`, `weekly`, `yearly`. Compounds multiple times per year instead of once |

{{< callout type="info" >}}
**`month` vs `monthly`**: The noun `month` is a period label -- `compound($1000, 5%, 12, month)` means "5% per month, applied 12 times" (the rate is per-month, not annual). The adverb `monthly` means "5% annual rate, compounded 12 times per year" -- a different formula with a different result.
{{< /callout >}}

#### Linear Growth

Calculate linear (additive) growth over time:

```calcmark
grow($500, $100, 36)               -> $4100.00
grow(100, 20, 5)                   -> 200

grow 100 by 20 over 5 months       (NL form)
```

**Arguments:**

| # | Name | Description |
|---|------|-------------|
| 1 | initial | Starting amount |
| 2 | increment | Amount added each period |
| 3 | periods | Number of periods (number or duration) |

#### Depreciation

Calculate declining-balance depreciation with optional salvage floor:

```calcmark
depreciate($50000, 15%, 5)                -> $22185.27
depreciate($50000, 15%, 20, $5000)        -> $5000.00  (salvage floor)

depreciate $50000 by 15% over 5 years     (NL form)
depreciate $50000 by 15% over 5 years to $5000
```

**Arguments:**

| # | Name | Description |
|---|------|-------------|
| 1 | value | Starting value |
| 2 | rate | Depreciation rate as percentage |
| 3 | periods | Number of periods (number or duration) |
| 4 | salvage | Optional: minimum floor value |

---

**Related guides:** [System Sizing](/guides/system-sizing/) (network/storage functions) | [Business Planning](/guides/business-planning/) (growth functions)
