---
title: "Embedded Datacenter Cost"
summary: "CalcMark's embedded mode: live calculations inside a standard Markdown file, designed for Hugo and static-site pipelines."
weight: 95
---

This example demonstrates **embedded mode** — CalcMark blocks inside a standard Markdown file. Instead of writing a `.cm` document, you write normal Markdown and tag calculation blocks with ` ```cm ` or ` ```calcmark `. Each block is evaluated independently, and the results replace the original block. Everything else passes through unchanged.

The complete Markdown file is available at {{< repo-file path="testdata/examples/embedded-datacenter-cost.md" >}}.

---

## How It Works

Run `cm convert --embedded` on any `.md` file:

```bash
cm convert report.md --embedded            # stdout
cm convert report.md --embedded -o out.md  # file
```

CalcMark scans for ` ```cm ` and ` ```calcmark ` fenced code blocks, evaluates each one as a self-contained CalcMark document, and replaces it with the evaluated Markdown output. Hugo frontmatter, footnotes, tables, HTML — everything else is byte-for-byte unchanged.

---

## Block Independence

Each block is its own CalcMark document. Variables don't carry between blocks. This means you can reuse variable names and each block can have its own frontmatter:

````text
```cm
servers = 12
cost_per_server = 450 USD
monthly_server_cost = servers * cost_per_server
```

Some prose in between...

```cm
---
exchange:
  EUR_USD: 1.09
---
eu_cooling = €45000
cooling_usd = eu_cooling in USD
```
````

The first block knows nothing about `eu_cooling`, and the second block knows nothing about `servers`. Each stands alone.

---

## Block Frontmatter vs. Document Frontmatter

The file's YAML frontmatter (title, date, tags) belongs to Hugo — CalcMark never touches it. CalcMark frontmatter (`exchange`, `scale`, `convert_to`) goes *inside* each block:

````text
---
title: "My Blog Post"      ← Hugo's — passed through unchanged
date: 2026-03-18
---

# My Post

```cm
---
scale:                      ← CalcMark's — applies to this block only
  factor: 1000
  unit_categories: [Currency]
---
capex = $875000
opex = $75000
build_tco = capex + (opex * 3)
```
````

With `scale` applied, the block above shows `$875M` capex and `$1.1B` TCO — modeling a fleet of 1,000 facilities.

---

## Error Handling

If a block has an error (undefined variable, syntax error), CalcMark replaces it with an inline error blockquote and continues processing the rest of the document:

```text
> **CalcMark Error:** line 1: undefined_variable: undefined variable "x" (line 42)
```

The exit code is 1 if any block had errors, but output is always produced. This means your build pipeline can detect problems without blocking the entire site build.

---

## Fence Variants

CalcMark recognizes all CommonMark fence styles:

| Fence | Accepted? |
|-------|-----------|
| ` ```cm ` | Yes |
| ` ```calcmark ` | Yes |
| `~~~cm` | Yes |
| `~~~calcmark` | Yes |
| ` ```cm {.highlight} ` | Yes (extra attributes OK) |
| ` ```python ` | No — passed through unchanged |
| ` ```cmake ` | No — `cmake` is not `cm` |

---

## Pipeline Integration

Embedded mode is designed for static-site build pipelines — like [D2](https://d2lang.com) for diagrams, but for calculations. Wire it into your Hugo build:

```bash
# Process all .md files with embedded CalcMark blocks
for f in content/posts/*.md; do
  cm convert "$f" --embedded -o "processed/$(basename "$f")"
done

# Or use it as a single-file preprocessor
cm convert content/cost-analysis.md --embedded -o content/cost-analysis.md
```

---

## Features Demonstrated

- **Embedded mode** — ` ```cm ` and ` ```calcmark ` blocks inside standard Markdown
- **Block independence** — no shared state between blocks
- **Block frontmatter** — exchange rates and scale directives per-block
- **Currency conversion** — EUR→USD, GBP→USD via `in` keyword
- **Growth functions** — `compound`, `depreciate`, `grow`
- **Scale directive** — fleet-level modeling with `unit_categories`
- **Error tolerance** — inline error blockquotes, non-zero exit code

---

## Try It

{{< repo-file path="testdata/examples/embedded-datacenter-cost.md" >}}

```bash
cm convert testdata/examples/embedded-datacenter-cost.md --embedded
```
