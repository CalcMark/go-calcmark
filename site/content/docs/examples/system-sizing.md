---
title: "System Sizing"
summary: "Back-of-napkin infrastructure estimation for a 10M user app with storage, bandwidth, and capacity planning."
weight: 50
calcmark_build: progressive
---

How many servers do you need for a social media app with 10 million users? This walkthrough sizes the infrastructure from scratch -- user activity, storage, database capacity, network latency, and availability -- using CalcMark's built-in system functions.

The complete CalcMark file is available at {{< repo-file path="testdata/examples/system-sizing.cm" >}}.

---

## User Activity

Start with 10M monthly active users. A 40% DAU/MAU ratio is typical for an engaged social platform -- Instagram-class engagement. The `% of` syntax reads like plain English.

```calcmark
monthly_users = 10M
daily_users = 40% of monthly_users
```

Each user posts about twice per week. Name the variable clearly and divide by 7 to get the daily rate. `as napkin` rounds to two significant figures for quick reference.

```calcmark
posts_per_user_per_week = 2
daily_posts = daily_users * posts_per_user_per_week / 7
daily_posts_napkin = daily_posts as napkin
```

That gives 4M daily active users generating ~1.1M posts per day.

**CalcMark features:** Multiplier suffixes (`10M`); `% of` for percentages; `as napkin` for human-readable rounding.

---

## Read vs Write Ratio

Social media is read-heavy -- users scroll far more than they post. Assume 100 reads per user per day across timeline, profiles, and search.

```calcmark
reads_per_user_per_day = 100
daily_reads = daily_users * reads_per_user_per_day
read_write_ratio = daily_reads / daily_posts
```

400M daily reads versus ~1.14M daily writes -- a read-to-write ratio of 350:1. This ratio drives your caching and replication strategy: invest heavily in read replicas and edge caching.

**CalcMark features:** Plain division for ratios; variable references across sections.

---

## Traffic Rates

You need per-second rates for capacity planning. CalcMark's rate conversion syntax turns daily totals into rates. Peak traffic is typically 3x average (lunch hour, evening scrolling, breaking news).

```calcmark
read_rate = (daily_reads)/day per second
write_rate = (daily_posts)/day per second

peak_multiplier = 3
peak_read_rate = read_rate * peak_multiplier
```

~4.63K reads/s average, peaking at ~13.89K reads/s. Write rate is a modest ~13.2/s -- your bottleneck is reads, not writes.

**CalcMark features:** Rate conversion (`(value)/day per second`); arithmetic on rate values.

---

## Storage Requirements

Each post is about 2 KB of text and metadata. 30% of posts include an image averaging 500 KB. CalcMark's `compress()` function estimates gzip compression on the text portion.

```calcmark
avg_post_size = 2 KB
daily_post_storage = daily_posts * avg_post_size
yearly_post_storage = daily_post_storage * 365

daily_media_storage = daily_posts * 30% * 500 KB
yearly_media_storage = daily_media_storage * 365

compressed_posts = compress(yearly_post_storage, gzip)
total_yearly_storage = compressed_posts + yearly_media_storage
```

Text storage is ~796 GB/year before compression. `compress()` applies a typical 3:1 gzip ratio, bringing it down to ~265 GB. Media dominates at ~58.3 TB/year. Total yearly storage lands at ~58.5 TB.

**CalcMark features:** Data size units (`KB`, `GB`, `TB`); inline `30%` for percentages; `compress()` with compression algorithm argument.

---

## Database Sizing

CalcMark's capacity planning syntax reads like a sentence. You specify a request rate, a per-server capacity, and a buffer percentage. It returns the number of servers needed.

```calcmark
db_read_replicas = peak_read_rate at 5000 req/s per server with 20% buffer
db_primaries = write_rate at 2000 req/s per server with 25% buffer
total_db_servers = db_read_replicas + db_primaries
```

At 13.89K peak reads/s with each server handling 5,000 req/s and a 20% headroom buffer, you need 4 read replicas. The write rate is low enough for 1 primary. Total: 5 database servers.

**CalcMark features:** `at ... per server with N% buffer` capacity planning syntax.

---

## Storage I/O Performance

A typical database query reads ~5 MB of data. CalcMark's `seek()` and `read ... from` syntax uses well-known device characteristics to estimate I/O time across HDD, SSD, and NVMe.

```calcmark
hdd_query_time = seek(hdd) + read 5 MB from hdd
ssd_query_time = seek(ssd) + read 5 MB from ssd
nvme_query_time = seek(nvme) + read 5 MB from nvme
```

HDD: ~43 ms. SSD: ~9.2 ms. NVMe: ~1.4 ms. NVMe is 30x faster than HDD for this workload. The right choice for hot data is clear.

**CalcMark features:** `seek()` for device seek latency; `read ... from` natural language syntax for data read time by device type.

---

## Network Latency Budget

Build a latency budget for an API response: network round-trip, database query, and application processing time.

```calcmark
network_rtt = rtt(regional)
db_query = nvme_query_time
app_processing = 10 ms
total_latency = network_rtt + db_query + app_processing
```

`rtt(regional)` returns 10 ms for a regional network hop. Adding the NVMe query (~1.4 ms) and 10 ms of app processing, your total latency budget is ~21.4 ms. Well under the 100 ms threshold users notice.

**CalcMark features:** `rtt()` with network distance argument; millisecond units; time addition.

---

## Bandwidth Requirements

Each response averages 10 KB. The `over` keyword accumulates the peak read rate over one second, then multiplies by response size to get bandwidth.

```calcmark
peak_bandwidth = (peak_read_rate over 1 second) * 10 KB

gigabit_capacity = throughput(gigabit)
ten_gig_capacity = throughput(ten_gig)
```

Peak bandwidth is ~136 MB/s. A single gigabit link handles 125 MB/s -- not enough. A 10-gigabit link at ~1.22 GB/s gives plenty of headroom.

**CalcMark features:** `over` for rate accumulation; `throughput()` for standard network link capacities.

---

## CDN and Caching

With a 95% cache hit target, only 5% of reads hit the origin. The `transfer ... across` syntax estimates time to push a media file from origin to CDN edge.

```calcmark
cache_hit_target = 95%
origin_read_rate = read_rate * (1 - cache_hit_target)

media_transfer = transfer 500 KB across continental ten_gig
```

Origin sees ~231 req/s instead of ~4,630 req/s -- a 20x reduction. Transfer time is ~50 ms to push a 500 KB image across a continental distance over a 10-gig link.

**CalcMark features:** Percentage syntax (`95%`); `transfer ... across` natural language syntax with data size, distance, and link speed.

---

## Availability and Error Budget

The `downtime()` function converts an availability target into concrete allowed downtime. But raw downtime means nothing until you spend it against real operational costs.

```calcmark
monthly_downtime = downtime(99.9%, month)
yearly_downtime = downtime(99.9%, year)
```

Three nines gives you 43.2 minutes/month or 8.76 hours/year. Now subtract deploy costs -- assume each rolling deploy takes 5 minutes (draining, health checks, restart):

```calcmark
deploy_time = 5 minutes
deploys_per_month = 8
total_deploy_downtime = deploy_time * deploys_per_month
remaining_error_budget = monthly_downtime - total_deploy_downtime
```

Eight deploys consume 40 of your 43.2 minutes, leaving only **3.2 minutes** for actual incidents. That is a razor-thin margin.

Now check four nines:

```calcmark
strict_monthly_downtime = downtime(99.99%, month)
strict_deploy_budget = strict_monthly_downtime - total_deploy_downtime
```

The budget goes **negative** (-35.7 minutes). At four nines your entire monthly allowance is 4.32 minutes -- eight 5-minute deploys already exceed it by 8x. This is the calculation that forces a zero-downtime deployment strategy (blue-green, canary).

**CalcMark features:** `downtime()` with percentage availability target and time period; Duration arithmetic (subtraction, scaling) to spend the error budget.

---

## Summary

The `as napkin` modifier rounds everything into quick-reference numbers for stakeholder conversations.

```calcmark
storage_napkin = total_yearly_storage as napkin
traffic_napkin = daily_reads as napkin
servers_napkin = total_db_servers as napkin
error_budget_napkin = remaining_error_budget as napkin
```

Bottom line: ~58.7 TB of storage per year, 400M daily reads, 5 database servers, and ~3.2 minutes of monthly error budget after deploys. That is the back-of-napkin infrastructure for a 10M-user social media app.

**CalcMark features:** `as napkin` for executive-summary rounding.

---

## Features Demonstrated

This example showcases the following CalcMark features:

- **Multiplier suffixes** -- `10M` expands to 10,000,000
- **Percentage syntax** -- `40%`, `30%`, `95%`, `99.9%` for readable proportions
- **`% of`** -- `40% of monthly_users` reads like plain English
- **`as napkin`** -- human-readable rounding to 2 significant figures
- **Rate conversion** -- `(value)/day per second` for daily-to-per-second rates
- **`over`** -- `peak_read_rate over 1 second` accumulates a rate over time
- **Capacity planning** -- `at ... per server with N% buffer`
- **`compress()`** -- storage compression estimation with algorithm argument
- **`seek()` and `read ... from`** -- storage I/O latency by device type (HDD, SSD, NVMe)
- **`rtt()`** -- network round-trip time by distance
- **`throughput()`** -- standard network link capacities
- **`transfer ... across`** -- data transfer time across distance and link speed
- **`downtime()`** -- availability target to allowed downtime conversion; Duration arithmetic to spend the error budget
- **Data size units** -- `KB`, `MB`, `GB`, `TB`, `ms`, `req/s`, `server`
- **Markdown prose** -- headings, paragraphs, and inline commentary between calculations

## Try It

{{< repo-file path="testdata/examples/system-sizing.cm" >}}

```bash
cm testdata/examples/system-sizing.cm

# Or open directly from GitHub — no clone required:
cm remote --http {{< repo-raw-url path="testdata/examples/system-sizing.cm" >}}
```
