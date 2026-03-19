---
title: "Infrastructure Cost Analysis"
date: 2026-03-18
author: Larky
tags: [infrastructure, cost, analysis]
draft: false
---

# Infrastructure Cost Analysis

This document models a cloud infrastructure build-out using [CalcMark][cm]
for live calculations. It exercises advanced Markdown features alongside
embedded CalcMark blocks to ensure the preprocessor handles real-world
content[^1].

## Server Pricing

We start with base server costs. This block uses the ` ```calcmark ` fence:

```calcmark
servers = 12
cost_per_server = 450 USD
monthly_server_cost = servers * cost_per_server
```

> **Note:** Prices are based on Q1 2026 list pricing from our vendor.
> See the [pricing page](https://example.com/pricing) for current rates.

## Storage Costs

Using a ` ```cm ` fence for brevity:

```cm
storage_tb = 50
cost_per_tb = 23 USD
monthly_storage = storage_tb * cost_per_tb
```

## Network Costs (Tilde Fence)

Tilde fences work too — useful when documenting code that itself contains
backtick fences:

~~~calcmark
egress_tb = 10
cost_per_egress_tb = 87 USD
monthly_network = egress_tb * cost_per_egress_tb
~~~

## Scaled Estimate

This block uses CalcMark frontmatter to scale all values by 1,000 (modeling
a fleet at scale). The outer Hugo frontmatter is **not** affected.

```cm
---
scale:
  factor: 1000
  unit_categories: [Currency]
---
units = 5
unit_cost = 200 USD
fleet_cost = units * unit_cost
```

## Bandwidth with Attributes

The `cm` info string can carry extra attributes — the scanner should still
match it:

```cm {.highlight}
bandwidth_gbps = 40
cost_per_gbps = 15 USD
monthly_bandwidth = bandwidth_gbps * cost_per_gbps
```

## Deliberate Error Block

This block references an undefined variable. The preprocessor should emit
an inline error and continue processing the rest of the document.

```cm
broken_total = nonexistent_var + 100
```

## Markdown Feature Showcase

The following exercises Markdown features that must pass through unchanged.

### Reference Links

Read more about [CalcMark][cm] or check the [spec][spec].

[cm]: https://calcmark.com "CalcMark Homepage"
[spec]: https://calcmark.com/spec "Language Specification"

### Footnotes

Infrastructure modeling is a common use case[^1]. Cost estimates should
always include a margin of error[^2].

[^1]: CalcMark was designed for exactly this kind of napkin math.
[^2]: Industry standard is ±15% for early-stage estimates.

### Tables (GFM)

| Component | Monthly Cost | Notes            |
|-----------|-------------|------------------|
| Servers   | $5,400      | 12 × $450        |
| Storage   | $1,150      | 50 TB × $23      |
| Network   | $870        | 10 TB × $87      |

### Task Lists (GFM)

- [x] Model server costs
- [x] Model storage costs
- [x] Model network costs
- [ ] Add redundancy multiplier
- [ ] Review with finance team

### Definition Lists

CalcMark
:   An interpreted language that blends CommonMark markdown and
    calculations in one document.

Embedded Mode
:   A preprocessing mode where CalcMark evaluates only fenced code
    blocks tagged `cm` or `calcmark` inside a standard Markdown file.

### HTML Pass-Through

<div class="callout" data-type="warning">
  <p>These estimates are <strong>preliminary</strong>. Do not use for
  budgetary approval without review.</p>
</div>

### Images and Autolinks

![Architecture diagram](./images/architecture.png "System Architecture")

Visit <https://calcmark.com> for documentation.

### Nested Blockquotes

> Level 1 quote
>
> > Level 2 quote with **bold** and *italic*
> >
> > > Level 3 — deep nesting preserved

### Fenced Code (Non-CalcMark)

These fences should pass through completely unchanged:

```python
def estimate_cost(servers, price):
    return servers * price
```

```yaml
infrastructure:
  servers: 12
  storage_tb: 50
```

~~~bash
cm convert --embedded report.md -o processed.md
~~~

### Indented Code Block

    This is an indented code block.
    It should NOT be treated as a CalcMark fence,
    even if it mentions ```cm``` in its content.

### Hard Line Breaks

This line has two trailing spaces
to force a hard line break.

This line uses a backslash\
for the same effect.

---

*Document generated with CalcMark embedded mode.*
