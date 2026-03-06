---
status: pending
priority: p2
issue_id: "008"
tags: [docs, site]
dependencies: []
---

# Write dedicated compound function documentation page

## Problem Statement

The `compound()` function has three distinct modes of operation but the language reference only shows examples without explaining the underlying theory or semantic differences between modes.

The current docs at `site/content/docs/language-reference.md:491-509` show three example blocks but never explain:
- What the mathematical formula is for each mode
- That `periods` changes meaning between mode 1/2 and mode 3
- That `rate` changes from per-period to annual in mode 3
- When to use which mode
- Real-world scenarios (savings account, investment, customer growth)

### The three modes

1. **Simple compound growth:** `compound(P, r, n)` produces P * (1+r)^n
   - `r` is rate per period, `n` is number of periods
2. **Named period:** `compound(P, r, n, unit)` produces same formula, period label for clarity
3. **Financial compounding:** `compound(P, r, t, compounded freq)` produces P * (1+r/n)^(n*t)
   - `r` becomes **annual** rate, `t` becomes **years**, `n` = periods-per-year from freq

## Proposed Solutions

Create a dedicated page at `site/content/docs/examples/compound-growth.md` that:

1. Explains the theory of compound interest / compound growth
2. Shows each mode with its formula, parameter meanings, and worked examples
3. Explicitly calls out the semantic shift in mode 3 (financial compounding)
4. Provides real-world scenarios: savings accounts, investment portfolios, customer/user growth
5. Links to the NL syntax alternatives (`compound $1000 by 5% over 10 years`)
6. Cross-references from the language reference page

## Technical Details

**Affected files:**
- `site/content/docs/language-reference.md:491-509` — add cross-reference
- `site/content/docs/examples/investment-growth.md` — existing example to extend or link
- `impl/interpreter/growth_functions.go:118-212` — implementation reference
- `spec/semantic/function_types.go:143-150` — parameter spec reference

## Acceptance Criteria

- [ ] Dedicated page explains all three modes with formulas
- [ ] Worked examples with actual numbers showing step-by-step calculation
- [ ] Clear callout of the semantic shift in financial compounding mode
- [ ] Language reference page links to the dedicated page
- [ ] NL syntax alternatives shown alongside fn syntax

## Work Log

| Date | Action | Learnings |
|------|--------|-----------|
| 2026-03-05 | Created during autocomplete bug fix | 3rd arg semantics confusing; existing docs insufficient |
