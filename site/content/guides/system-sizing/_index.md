---
title: "System Sizing & Capacity Planning"
summary: "Model server fleets, bandwidth, storage, and SLA budgets with CalcMark."
weight: 10
calcmark_build: progressive
---

How many servers do you need? How much bandwidth? What's the storage budget for the next quarter? These are napkin-math problems — you need rough answers fast, not exact answers slow.

CalcMark is built for this. Write your assumptions in plain text, let the math flow, and change any number to see the cascade.

**Try it now:** Open the {{< lark "system-sizing" "System Sizing example in Lark" >}} or run `cm remote system-sizing` in the editor.

---

## Key Features for This Domain

| Feature | What It Does | Reference |
|---------|-------------|-----------|
| Rates (`100 MB/s`, `$50/hour`) | Model throughput, cost rates, request rates | [Rates](/docs/user-guide/formatting/#rates) |
| `at...per` capacity syntax | "10 TB at 2 TB per disk" → 5 disks | [Capacity Planning](/docs/user-guide/formatting/#capacity-planning) |
| `as napkin` | Round to 2 sig figs for back-of-envelope estimates | [Napkin Math](/docs/user-guide/formatting/#napkin-math) |
| Multiplier suffixes (`10K`, `1.5M`) | Compact notation for large numbers | [Multipliers](/docs/user-guide/formatting/#multiplier-suffixes) |
| `over` keyword | Accumulate a rate over time: `100 MB/s over 1 day` | [Rates](/docs/user-guide/formatting/#rates) |
| Network functions (`rtt`, `throughput`) | Model latency, bandwidth, transfer time | [Network Functions](/docs/user-guide/functions/#network-functions) |
| Storage functions (`read`, `seek`) | Model disk I/O by device type | [Storage Functions](/docs/user-guide/functions/#storage-functions) |

---

## Walkthrough: Sizing a Web Service

### Step 1: Traffic Assumptions

Start with what you know — expected requests per second and growth rate:

```calcmark
peak_rps = 10K/s
avg_rps = peak_rps * 0.4
daily_requests = avg_rps over 1 day
growth_rate = 25%
```

`daily_requests` uses `over` to accumulate the average rate across a full day. `growth_rate` is a percentage — CalcMark tracks the type.

### Step 2: Compute Capacity

Each server handles a known number of requests. How many servers for the peak?

```calcmark
capacity_per_server = 2K
servers_needed = peak_rps at capacity_per_server
```

The `at...per` syntax divides and rounds up — you can't run half a server.

### Step 3: Storage Budget

Estimate storage from request volume and payload size:

```calcmark
avg_payload = 4 KB
daily_storage = daily_requests * avg_payload as napkin
monthly_storage = daily_storage * 30 as napkin
```

`as napkin` rounds to 2 significant figures — perfect for planning where `~14 GB` is more useful than `13.8240 GB`.

### Step 4: Network I/O

Use the built-in `throughput` and `transfer_time` functions:

```calcmark
link_speed = 10 gigabits
burst_transfer = transfer_time 50 GB at link_speed
```

---

## What to Read Next

- **Complete example:** [System Sizing](/docs/examples/system-sizing/) — full worked document with servers, storage, and bandwidth
- **Datacenter costs:** [Datacenter Cost](/docs/examples/datacenter-cost/) — power, cooling, and rack-level modeling
- **Functions reference:** [Network Functions](/docs/user-guide/functions/#network-functions) and [Storage Functions](/docs/user-guide/functions/#storage-functions)
- **Language spec:** [Rates](/docs/language-reference/#rates), [Napkin Math](/docs/language-reference/#as-napkin)
