---
title: "feat: Thinking in CalcMark guide — idiomatic expressions"
type: feat
status: completed
date: 2026-03-15
origin: docs/brainstorms/2026-03-15-idiomatic-calcmark-guide-brainstorm.md
---

# feat: Thinking in CalcMark Guide

## Overview

Create a new guide at `site/content/guides/thinking-in-calcmark/_index.md` that teaches people how to transition from "calculator thinking" to idiomatic CalcMark. The guide follows a "moving from Austin to Denver" scenario using gradual immersion: each section starts with working-but-clunky calculator-style code and refactors it into cleaner, more natural CalcMark expressions. Ends with a quick reference card mapping every calculator pattern to its CalcMark idiom.

(see brainstorm: `docs/brainstorms/2026-03-15-idiomatic-calcmark-guide-brainstorm.md`)

## Problem Statement / Motivation

People coming from calculators and spreadsheets naturally write CalcMark like a programming language: `total = a + b + c + d`, `adjusted = salary * 1.08`, `sum(a, b, c)`. CalcMark has expressive idioms — `sum of`, percentage operators, rate accumulation with `over`, NL growth functions — but nothing teaches *when and why* to use them. The language reference documents the syntax; this guide teaches the mindset.

## Proposed Solution

A single-page guide with TOC and section anchors, following the established guide structure pattern. The "Austin → Denver" scenario threads through all sections. Tabbed examples (CalcMark/Editor/JSON) reuse the tab component built for the Understanding Measurements guide.

### Differentiation from existing guides

| Guide | Purpose | Teaches |
|-------|---------|---------|
| **Thinking in CalcMark** (new) | Mindset shift from calculator → idiomatic | `sum of`, `%` operators, rates with `per`/`over`, `compound`/`grow`, `cm help` |
| Understanding Measurements | Progressive tutorial of the measurement type system | Quantities, `in`, fractions, `convert_to`, `as napkin`, conventions |
| System Sizing | Domain recipe: infrastructure | Rates, `at...per`, capacity planning, napkin math |

## Technical Approach

### Phase 1: Guide Content

Create `site/content/guides/thinking-in-calcmark/_index.md` following the established guide structure.

**Weight:** 24 (before Understanding Measurements at 26)

**Section outline with verified examples:**

#### Section 1: "You Don't Need a Spreadsheet" (Introduction)

- You're moving from Austin to Denver. Time to crunch numbers.
- First example: `rent_austin = $1450` / `rent_denver = $1750` / `difference = rent_denver - rent_austin`
- Teaching point: simple arithmetic already works. CalcMark understands currency.
- No refactoring needed here — establish that calculator-style code IS valid CalcMark.

#### Section 2: Adding Things Up (`sum of`)

- Calculator instinct: `total = rent + groceries + transport + utilities`
- Still works! But introduce: `total = sum of rent, groceries, transport, utilities`
- Verified output: `sum of` with currency variables → `$2,435.00`
- Teaching point: `sum of` reads like English, scales to any number of items, is the idiomatic way.
- Callout: *Run `cm help sum` to see both syntaxes.*

#### Section 3: Percentages Without the Math

- Calculator instinct: `adjusted = salary * 1.08` or `adjusted = salary + salary * 0.08`
- CalcMark way: `adjusted = salary + 8%`
- Reverse: `after_tax = adjusted - 24%`
- Verified: `$85K + 8%` → `$91.8K`; `$91.8K - 24%` → `$69.77K`
- Teaching point: `%` is an operator, not a number. No manual decimal conversion.

#### Section 4: Rates and Accumulation (`per` / `over`)

- Commute cost: calculator instinct: `monthly_commute = 2.75 * 2 * 22`
- Step 1: Use variables — `cost_per_ride = $2.75` / `daily_commute = cost_per_ride * 2`
- Step 2: Introduce rates — `fare = $5.50/day`
- Step 3: Accumulate — `monthly_commute = fare over 1 month` → `$165.00`
- Teaching point: rates are first-class (`$5.50/day` is a Rate type). `over` accumulates them over a time period.
- Callout: *Run `cm help keywords` to see `per`, `over`, `at`, and other contextual keywords.*

#### Section 5: Monthly Budget — Putting It Together

- Full budget document combining everything learned so far.
- Show "v1" (calculator-style): raw addition, manual percentages, computed commute.
- Refactor to "v2" (idiomatic): `sum of`, `salary - 24%`, rate with `over`.
- Side-by-side in tabs: v1 CalcMark tab, v2 CalcMark tab (reuse tab component with `group="version"`).
- Teaching point: the idiomatic version is shorter AND more readable.

#### Section 6: Saving for a House (`compound` / `grow`)

- "You're settled in Denver. Time to start saving."
- Calculator instinct: `future = 500 * (1 + 0.045/12) ^ (12 * 5)` — the compound interest formula
- CalcMark way: `compound $500 by 4.5% monthly over 5 years` → `$625.90`
- Also: `depreciate $35000 by 15% over 5 years` → `$15.53K` (car losing value)
- Teaching point: growth functions read like sentences. No formula memorization.
- Callout: *Run `cm help compound` to see all growth function variants.*

#### Section 7: A Taste of Domain Functions (Light Touch)

- "CalcMark also has domain-specific functions for engineering and infrastructure."
- One example: `read 100 MB from ssd` → `0.1818 second`
- Brief: "These read like English and compute real results. See the [System Sizing guide](/guides/system-sizing/) for the full story."

#### Section 8: NL Syntax with Variables

- Show that NL functions accept both literals and variable references in any argument position.
- Example: `compress yearly_storage using gzip` works just like `compress 796 GB using gzip`.
- Teaching point: "Define your values once, then use them by name in NL syntax."

#### Section 9: Quick Reference Card

| Calculator Style | Idiomatic CalcMark | Why |
|------------------|--------------------|-----|
| `a + b + c + d` | `sum of a, b, c, d` | Reads like English, scales to any length |
| `x * 1.08` | `x + 8%` | Percentage as operator, not decimal math |
| `x - x * 0.24` | `x - 24%` | Same pattern for deductions |
| `2.75 * 2 * 22` | `$5.50/day over 1 month` | Rate accumulation handles the multiplication |
| `P*(1+r/n)^(nt)` | `compound P by r% monthly over t years` | Reads like a sentence |
| `sum(a,b,c)` | `sum of a, b, c` | NL form preferred |
| `avg(a,b,c)` | `average of a, b, c` | Same pattern |
| `sqrt(16)` | `square root of 16` | Same pattern |

#### Section 10: What to Read Next

Links to: Understanding Measurements (units/conversions), system-sizing (engineering functions), language reference (full NL syntax list), functions user guide.

### Phase 2: Sidebar and Cross-Linking

1. **Update `site/hugo.yaml`**: Add "Thinking in CalcMark" sidebar entry at weight 24 (before Understanding Measurements at 26)
2. **Update `site/content/guides/_index.md`**: Add hand-written entry before Understanding Measurements
3. **Cross-link from system-sizing guide**: Add "See also" pointing here for NL function basics

## Acceptance Criteria

- [x] Guide follows established structure: frontmatter (title, summary, weight: 24), intro, feature table, walkthrough, "What to Read Next"
- [x] Each section starts with calculator-style code, then gradually refactors to idiomatic CalcMark
- [x] Tabbed examples (CalcMark/Editor/JSON) using existing tab component
- [x] All CalcMark examples produce correct output (verified via `cm --format json`)
- [x] `cm help` callouts woven in at first appearance of each new idiom
- [x] NL limitations mentioned with practical workaround
- [x] Quick reference card at the end mapping calculator → CalcMark patterns
- [x] Sidebar entry added at weight 24
- [x] Guides index page updated
- [x] `task quality` passes
- [x] Hugo builds cleanly

## Dependencies & Risks

**Dependencies:**
- Tab component already exists (built in the Understanding Measurements guide)
- No new infrastructure needed

**Risks:**
- **NL syntax edge cases:** Some NL functions have quirks with variable references. The guide addresses this with a practical note in Section 8.
- **`$X per Y` syntax:** `$2.75 per ride` parsed as text, not a rate. The guide uses `$5.50/day` (slash syntax) instead, which works. The "per" keyword works for capacity planning (`at X per Y`) but not standalone rate creation with currency.

## Sources & References

### Origin

- **Brainstorm document:** [docs/brainstorms/2026-03-15-idiomatic-calcmark-guide-brainstorm.md](docs/brainstorms/2026-03-15-idiomatic-calcmark-guide-brainstorm.md) — Key decisions: gradual immersion, Austin→Denver scenario, `cm help` callouts, light-touch engineering functions, quick reference card.

### Internal References

- Tab component: `site/layouts/shortcodes/tabs.html`, `tab.html`
- Understanding Measurements guide (pattern to follow): `site/content/guides/understanding-measurements/_index.md`
- NL function registry: `spec/features/registry.go`
- NL syntax limitations: memory file `nl-syntax-limitations.md`
- Progressive guide authoring learnings: `docs/solutions/infrastructure/progressive-guide-authoring-pattern.md`

### Verified CalcMark Output

All examples verified against `cm v1.8.6`:

```
echo 'salary = $85000\nadjusted = salary + 8%\nafter_tax = adjusted - 24%' | ./cm --format json
echo 'fare = $5.50/day\nmonthly = fare over 1 month' | ./cm --format json
echo 'compound $500 by 4.5% monthly over 5 years' | ./cm --format json
echo 'sum of rent, groceries, transport, utilities' | ./cm --format json
echo 'depreciate $35000 by 15% over 5 years' | ./cm --format json
echo 'read 100 MB from ssd' | ./cm --format json
```

### Key Learnings Applied (from Understanding Measurements guide)

- Remove `# Title` from content — Hugo template renders it from frontmatter
- No `---` horizontal rules between sections — H2s already have border-bottom
- Verify every example with `cm --format json` before writing content
- Add sidebar menu entry in `hugo.yaml` for discoverability
- Larky uses they/them pronouns
