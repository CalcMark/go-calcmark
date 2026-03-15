# Brainstorm: Idiomatic CalcMark Guide

**Date:** 2026-03-15
**Status:** Draft

---

## What We're Building

A new guide that helps people transition from "calculator thinking" (manual arithmetic, `sum(a,b,c)`, `x + x * 0.15`) to idiomatic CalcMark (`sum of a, b, c`; `grow`; NL functions; `as napkin`). The guide follows a "moving to a new city" scenario — comparing costs, adjusting budgets, and eventually saving for a house — using gradual immersion: each section starts with working-but-clunky code and refactors it into cleaner CalcMark.

**The core problem:** People coming from calculators and spreadsheets naturally write CalcMark like a programming language. They don't know about NL syntax, `sum of`, percentage operators, rate accumulation, or growth functions. The language has expressive idioms, but nothing teaches *when and why* to use them over raw arithmetic.

---

## Why This Approach

### Story + Quick Reference (Approach C)

- **Scenario:** Moving from Austin to Denver. Universal life event — nearly everyone has done "can I afford to live there?" math on a napkin or spreadsheet.
- **Structure:** Linear story arc with gradual immersion. Each section starts with calculator-style code that works, then refactors it step by step into idiomatic CalcMark. The reader watches their code evolve.
- **Quick reference card:** A compact summary table at the end mapping every calculator-style pattern to its CalcMark idiom. Scannable reference after the story.
- **Discoverability:** `cm help` commands woven in as callouts when new idioms appear — not a lecture, just "here's how you'd find this yourself."
- **Engineering functions:** Light touch — one example of `read X from Y` as a taste, with a link to the system-sizing guide.

### Why "moving to a new city"?

- Nearly universal experience
- Naturally requires progressively more sophisticated math: simple arithmetic → percentages → rates → accumulation → growth/compound
- Emotionally engaging — the reader has real stakes in these numbers
- Ends with "saving for a house" which perfectly introduces `compound` and `grow`

---

## Key Decisions

1. **Placement:** New guide at `site/content/guides/idiomatic-calcmark/_index.md`, added to sidebar.
2. **Narrative arc:** Gradual immersion — each section starts with calculator-style code, then refactors to idiomatic CalcMark. Story: Austin → Denver move.
3. **Contrast style:** NOT side-by-side before/after. Instead, the code evolves within each section. Reader sees their own instinct, then learns the better way.
4. **Discoverability:** `cm help` callouts woven in when new idioms first appear. No dedicated section.
5. **Engineering functions:** Light touch — one example, link to system-sizing guide.
6. **Quick reference:** Summary table at the end mapping calculator-style → CalcMark for every idiom covered.
7. **Tabs:** Reuse the tab component from the Understanding Measurements guide (CalcMark/Editor/JSON).

---

## Proposed Section Outline

### 1. Introduction — "You Don't Need a Spreadsheet"

- You're moving from Austin to Denver. Time to crunch some numbers.
- "You could open a spreadsheet. Or you could write it down the way you think about it."
- First example: `rent_austin = $1450` / `rent_denver = $1750` / `difference = rent_denver - rent_austin` — this is already CalcMark! Simple arithmetic works.

### 2. Adding Things Up

- Calculator instinct: `total = rent + groceries + transport + utilities`
- Still works! But when the list grows: `total = sum of rent, groceries, transport, utilities`
- Teaching point: `sum of` reads like English and handles any number of items.
- Callout: *Run `cm help sum` to see both syntaxes.*

### 3. Percentages Without the Math

- Calculator instinct: `adjusted = salary * 1.08` or `adjusted = salary + salary * 0.08`
- CalcMark way: `adjusted = salary + 8%` — CalcMark understands percentage context
- Reverse: `after_tax = salary - 24%`
- Teaching point: Percentages work as operators, not just numbers. No need to convert to decimals.

### 4. Rates and Accumulation

- Commute cost: calculator instinct: `monthly_commute = 2.75 * 2 * 22`
- CalcMark way: `fare = $2.75 per ride` / `daily = fare * 2` / `monthly = daily * 22`
- Or with accumulation: `$5.50/day over 1 month`
- Teaching point: Rates are first-class — `per` creates them, `over` accumulates them.
- Callout: *Run `cm help keywords` to see `per`, `over`, `at`, and other contextual keywords.*

### 5. Monthly Budget — Putting It Together

- Full budget document combining everything learned so far.
- Show the "calculator version" (raw arithmetic, manual totals) then refactor:
  - Raw addition → `sum of`
  - Manual percentage → `salary - 24%`
  - Computed commute → rate with `per` and `over`
- Show how readable the final version is compared to where we started.

### 6. Saving for a House — Growth and Compound Interest

- "You're settled in Denver. Time to start saving."
- Calculator instinct: manual compound interest formula `P * (1 + r/n)^(nt)`
- CalcMark way: `compound $500 by 4.5% monthly over 5 years`
- Or simpler: `grow $500 by $500 over 60 months` for fixed contributions
- Depreciation mention: `depreciate $35000 by 15% over 5 years` for a car losing value
- Teaching point: Growth functions read like sentences. No formula memorization.
- Callout: *Run `cm help compound` to see all growth function variants.*

### 7. Beyond the Calculator — A Taste of Domain Functions

- Light touch: "CalcMark also has domain-specific functions that go beyond what any calculator offers."
- One example: `read 100 MB from ssd` — reads like English, computes storage read time
- Link to system-sizing guide for the full story.

### 8. Quick Reference Card

Summary table:

| Calculator Style | Idiomatic CalcMark | Why |
|------------------|--------------------|-----|
| `a + b + c + d` | `sum of a, b, c, d` | Reads like English, scales to any length |
| `x * 1.08` | `x + 8%` | Percentage as operator, not decimal math |
| `x - x * 0.24` | `x - 24%` | Same pattern for deductions |
| `2.75 * 2 * 22` | `$5.50/day over 1 month` | Rate accumulation handles the multiplication |
| `P*(1+r/n)^(nt)` | `compound P by r% monthly over t years` | Reads like a sentence |
| `sum(a,b,c)` | `sum of a, b, c` | NL form preferred in CalcMark |
| `avg(a,b,c)` | `average of a, b, c` | Same pattern |

### 9. What to Read Next

- Links to: Understanding Measurements (for units/conversions), system-sizing (for engineering functions), language reference (full NL syntax list)

---

## Resolved Questions

1. **Scenario:** Moving from Austin to Denver — universal, emotionally engaging, naturally progressive.
2. **Contrast style:** Gradual immersion — code evolves within each section, not side-by-side.
3. **Discoverability:** `cm help` callouts woven in at first appearance of each new idiom.
4. **Engineering functions:** Light touch — one `read X from Y` example, link to system-sizing.
5. **Saving for a house:** Natural story endpoint introducing `compound` and `grow`.
6. **Quick reference:** Summary table at the end mapping calculator → CalcMark patterns.

## Resolved (cont.)

7. **Guide URL:** `guides/thinking-in-calcmark/` — signals a mindset shift, not just syntax.
8. **Sidebar weight:** 24, before Understanding Measurements (26). This guide is more foundational — basic CalcMark idioms before units and conversions.
9. **NL limitations:** Brief practical note when it naturally comes up ("If a NL function doesn't accept a variable, use the functional form instead"). Honest without being discouraging.

## Open Questions

None remaining.
