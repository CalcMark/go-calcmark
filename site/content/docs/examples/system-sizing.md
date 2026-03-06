---
title: "System Sizing"
summary: "Back-of-napkin infrastructure estimation for a 10M user app with storage, bandwidth, and capacity planning."
weight: 50
calcmark_build: progressive
---

How many servers do you need for a social media app with 10 million users? This walkthrough sizes the infrastructure from scratch -- user activity, storage, database capacity, network latency, and availability -- using CalcMark's built-in system functions.

The complete CalcMark file is available at {{< repo-file path="testdata/examples/system-sizing.cm" >}}.

---

## User Activity Assumptions

You start with 10M monthly active users. Assume 40% are active daily. Each user posts about twice a week, so you divide by 7 to get a daily rate.

```calcmark
monthly_users = 10M
daily_active_pct = 0.40
daily_users = monthly_users * daily_active_pct

posts_per_user_per_day = 2 / 7
daily_posts = daily_users * posts_per_user_per_day
daily_posts_napkin = daily_posts as napkin
```

`10M` is a multiplier suffix -- CalcMark expands it to 10,000,000. That gives you 4M daily active users generating roughly 1.14M posts per day. The `as napkin` modifier rounds to two significant figures: `~1.1M`.

**CalcMark features:** Multiplier suffixes (`10M`); `as napkin` for human-readable rounding.

---

## Read vs Write Ratio

Social media is read-heavy. Users scroll far more than they post. You assume 100 reads per user per day.

```calcmark
reads_per_user_per_day = 100
daily_reads = daily_users * reads_per_user_per_day
read_write_ratio = daily_reads / daily_posts
```

That gives 400M daily reads versus ~1.14M daily writes -- a read-to-write ratio of 350:1. This ratio drives your caching and replication strategy.

**CalcMark features:** Plain division for ratios; variable references across sections.

---

## Traffic Rates

You need per-second rates for capacity planning. CalcMark's rate conversion syntax turns daily totals into rates. Peak traffic is typically 3x average.

```calcmark
read_rate = (daily_reads)/day per second
write_rate = (daily_posts)/day per second

peak_multiplier = 3
peak_read_rate = read_rate * peak_multiplier
```

The `(value)/day per second` syntax divides the daily total by 86,400 seconds. You get ~4.63K reads/s average, peaking at ~13.89K reads/s. Write rate is a modest ~13.2/s.

**CalcMark features:** Rate conversion (`(value)/day per second`); arithmetic on rate values.

---

## Storage Requirements

Each post is about 2 KB of text and metadata. About 30% of posts include an image averaging 500 KB. CalcMark's `compress()` function estimates gzip compression on the text portion.

```calcmark
avg_post_size = 2 KB
daily_post_storage = daily_posts * avg_post_size
yearly_post_storage = daily_post_storage * 365

posts_with_media_pct = 0.30
avg_image_size = 500 KB
daily_media_storage = daily_posts * posts_with_media_pct * avg_image_size
yearly_media_storage = daily_media_storage * 365

compressed_posts = compress(yearly_post_storage, gzip)
total_yearly_storage = compressed_posts + yearly_media_storage
```

Text storage is ~796 GB/year before compression. `compress(yearly_post_storage, gzip)` applies a typical 3:1 gzip ratio, bringing it down to ~265 GB. Media dominates at ~58.3 TB/year. Total yearly storage lands at ~58.5 TB.

**CalcMark features:** Arbitrary units (`KB`, `GB`, `TB`); `compress()` function with compression algorithm argument.

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

A typical database query reads ~5 MB of data. CalcMark's `seek()` and `read()` functions use well-known device characteristics to estimate I/O time across HDD, SSD, and NVMe.

```calcmark
query_data = 5 MB
hdd_query_time = seek(hdd) + read(query_data, hdd)
ssd_query_time = seek(ssd) + read(query_data, ssd)
nvme_query_time = seek(nvme) + read(query_data, nvme)
```

HDD: ~43 ms. SSD: ~9.2 ms. NVMe: ~1.4 ms. NVMe is 30x faster than HDD for this workload. The right choice for hot data is clear.

**CalcMark features:** `seek()` function for device seek latency; `read()` function for data read time by device type.

---

## Network Latency Budget

You build a latency budget for an API response: network round-trip, database query, and application processing time.

```calcmark
network_rtt = rtt(regional)
db_query = nvme_query_time
app_processing = 10 ms
total_latency = network_rtt + db_query + app_processing
```

`rtt(regional)` returns 10 ms for a regional network hop. Adding the NVMe query (~1.4 ms) and 10 ms of app processing, your total latency budget is ~21.4 ms. Well under the 100 ms threshold users notice.

**CalcMark features:** `rtt()` function with network distance argument; millisecond units; time addition.

---

## Bandwidth Requirements

Each response averages 10 KB. You multiply by peak read rate to get bandwidth, then compare against standard network link capacities.

```calcmark
avg_response_kb = 10
peak_bandwidth_kbs = peak_read_rate * avg_response_kb
peak_bandwidth_mbs = peak_bandwidth_kbs / 1000

gigabit_capacity = throughput(gigabit)
ten_gig_capacity = throughput(ten_gig)
```

Peak bandwidth is ~139 MB/s. A single gigabit link handles 125 MB/s -- not enough. A 10-gigabit link at ~1.22 GB/s gives you plenty of headroom.

**CalcMark features:** `throughput()` function for standard network link capacities.

---

## CDN and Caching

With a 95% cache hit target, only 5% of reads hit the origin. You calculate origin traffic and the time to transfer a media file from origin to CDN edge.

```calcmark
cache_hit_target = 0.95
cache_miss_rate = 1 - cache_hit_target
origin_read_rate = read_rate * cache_miss_rate

media_transfer = transfer_time(avg_image_size, continental, ten_gig)
```

Origin sees ~231 req/s instead of ~4,630 req/s -- a 20x reduction. `transfer_time()` estimates ~50 ms to push a 500 KB image across a continental distance over a 10-gig link.

**CalcMark features:** `transfer_time()` function with data size, distance, and link speed arguments.

---

## Availability and Downtime

The `downtime()` function converts an availability target into concrete allowed downtime. You compare three nines (99.9%) with four nines (99.99%).

```calcmark
monthly_downtime = downtime(0.999, month)
yearly_downtime = downtime(0.999, year)

strict_monthly_downtime = downtime(0.9999, month)
```

Three nines gives you 43.2 minutes/month or 8.76 hours/year of allowed downtime. Four nines shrinks that to just 4.32 minutes/month. The jump from three to four nines is a 10x reduction in error budget.

**CalcMark features:** `downtime()` function with availability target and time period.

---

## Summary

The `as napkin` modifier rounds everything into quick-reference numbers for stakeholder conversations.

```calcmark
storage_napkin = total_yearly_storage as napkin
traffic_napkin = daily_reads as napkin
servers_napkin = total_db_servers as napkin
```

Bottom line: ~58.7 TB of storage per year, 400M daily reads, 5 database servers. That is the back-of-napkin infrastructure for a 10M-user social media app.

**CalcMark features:** `as napkin` for executive-summary rounding.

---

## Features Demonstrated

This example showcases the following CalcMark features:

- **Multiplier suffixes** -- `10M` expands to 10,000,000
- **`as napkin`** -- human-readable rounding to 2 significant figures
- **Rate conversion** -- `(value)/day per second` for daily-to-per-second rates
- **Capacity planning** -- `at ... per server with N% buffer`
- **`compress()`** -- storage compression estimation with algorithm argument
- **`seek()` and `read()`** -- storage I/O latency by device type (HDD, SSD, NVMe)
- **`rtt()`** -- network round-trip time by distance
- **`throughput()`** -- standard network link capacities
- **`transfer_time()`** -- data transfer time across distance and link speed
- **`downtime()`** -- availability target to allowed downtime conversion
- **Arbitrary units** -- `KB`, `MB`, `GB`, `TB`, `ms`, `req/s`, `server`
- **Markdown prose** -- headings, paragraphs, and inline comments between calculations

## Try It

{{< repo-file path="testdata/examples/system-sizing.cm" >}}

```bash
cm testdata/examples/system-sizing.cm
```
