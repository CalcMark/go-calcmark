---
title: "feat: Period-to-duration conversion via in/as"
type: feat
status: active
date: 2026-04-07
issue: 125
---

# feat: Period-to-duration conversion via `in`/`as`

## Overview

Enable `FQ1 in days`, `this month in weeks`, `April in days` — period expressions implicitly convert to their duration when used with `in`/`as` unit conversion. No new span type. The evaluator computes the period length internally by resolving start and end dates from the keyword.

## Problem Frame

Period expressions (Q1, FQ1, this month, next April, this quarter) resolve to points (first day). Users need the *length* of those periods for budget calculations, capacity planning, and scheduling. Currently `FQ1 in days` fails with "requires a quantity or duration, got *types.Date".

## Requirements Trace

- R1. `Q1 in days` → 90 (Jan 1 - Mar 31 = 90 days)
- R2. `FQ1 in days` → 92 (Jul 1 - Sep 30, with fiscal_year_starts: july)
- R3. `this month in days` → varies (28-31 depending on month/year)
- R4. `February 2024 in days` → 29 (leap year), `February 2025 in days` → 28
- R5. `this quarter in weeks` → approximate (days / 7)
- R6. Works with `end of` periods: `end of Q2 in days` should error or be meaningless (end-of is a point, not a period)
- R7. Non-period dates error clearly: `today in days` → "cannot convert date to duration — 'today' is a point, not a period"

## Scope Boundaries

- No new "span" or "range" type exposed to users
- No `as fiscal` / `as calendar` label conversion (separate feature)
- Only the `in` conversion path — not `as` (which is for display formatting like napkin)

## Key Technical Decisions

- **Implicit conversion in evalUnitConversion**: When the evaluated Quantity node is a `*types.Date`, check if the original AST node is a period expression. If so, compute duration = end - start + 1 day, then proceed with normal duration `in` conversion
- **Period detection via AST node**: The `UnitConversion.Quantity` field holds the original AST node. Check if it's a `RelativeDateLiteral` with a period keyword. Non-period dates (today, tomorrow, specific date literals) produce a clear error
- **Duration computation**: Reuse `evalEndOf` logic internally — period duration = `endOf(keyword) - startOf(keyword) + 1 day`
- **No changes to the type system**: The Date type doesn't gain a "period" flag. The conversion is purely evaluator-level, keyed on the AST node type

## Context & Research

### Relevant Code

- `impl/interpreter/unit_conversion_eval.go:88-92` — the error point where Date fails the `in` conversion
- `impl/interpreter/datetime.go` — `evalEndOf` already computes period end dates for every period type
- `impl/interpreter/datetime.go` — `evalRelativeDateLiteral` resolves period starts
- `spec/ast/nodes.go:88-93` — `UnitConversion` struct, `Quantity` is a `Node`

### Pattern

The existing Duration `in` conversion at line 64-70 of `unit_conversion_eval.go` is the model. Once we have a Duration from the period, the same `duration.Convert(targetUnit)` handles the rest.

## Implementation Units

- [ ] **Unit 1: Period-to-duration in evalUnitConversion**

**Goal:** When `in` conversion receives a Date from a period expression, compute the period's duration and convert it.

**Requirements:** R1-R5, R7

**Dependencies:** None

**Files:**
- Modify: `impl/interpreter/unit_conversion_eval.go`
- Modify: `impl/interpreter/datetime.go` (extract period duration helper)
- Test: `impl/interpreter/date_test.go`

**Approach:**
- In `evalUnitConversion`, after evaluating the Quantity node, check if result is `*types.Date`
- If so, inspect the original AST node (`u.Quantity`). If it's a `RelativeDateLiteral`, extract the keyword
- Call a new `periodDuration(keyword, now, fiscalConfig)` function that returns `*types.Duration` or error
- The function resolves start (existing eval) and end (existing evalEndOf) for the keyword, computes `end - start + 1 day`, returns Duration in days
- If the AST node is NOT a period expression (e.g., a DateLiteral like `Jan 1 2025`, or `today`), return a clear error: "cannot convert date to duration — use a period expression like 'this month', 'Q1', or 'FQ1'"
- Once we have a Duration, fall through to the existing duration conversion path (line 64-70)

**Execution note:** Test-first. Write failing tests for each period type before implementing.

**Patterns to follow:**
- Duration conversion at `unit_conversion_eval.go:64-70`
- Period end computation in `evalEndOf`

**Test scenarios:**
- Happy path: `Q1 in days` → 90, `Q2 in days` → 91, `Q3 in days` → 92, `Q4 in days` → 92
- Happy path: `FQ1 in days` with July start → 92
- Happy path: `this month in days` in April → 30, in February 2024 → 29
- Happy path: `this quarter in weeks` → days/7 (approximate)
- Happy path: `next April in days` → 30
- Edge case: `February 2024 in days` → 29 (leap), `February 2025 in days` → 28
- Edge case: `this year in days` → 365 or 366
- Error path: `today in days` → clear error (point, not period)
- Error path: `Jan 1 2025 in days` → clear error
- Error path: `FQ1 in days` without fiscal config → fiscal_year_starts error

**Verification:**
- All period types produce correct day counts
- Non-period dates produce clear errors
- Existing `in` conversions (quantities, durations, currencies) unchanged

- [ ] **Unit 2: Detect period vs. point expressions**

**Goal:** Classify which RelativeDateLiteral keywords are periods (have a span) vs. points.

**Requirements:** R6, R7

**Dependencies:** Unit 1

**Files:**
- Modify: `impl/interpreter/datetime.go`
- Test: `impl/interpreter/date_test.go`

**Approach:**
- Create `isPeriodExpression(keyword) bool` that returns true for keywords with a known start+end: this/next/last week/month/year/quarter, this/next/last fiscal quarter/year, named months (this April, next Dec), notation (Q1-Q4, FQ1-FQ4)
- Returns false for: today, tomorrow, yesterday, now, weekday expressions (next Friday), `end of` prefixed expressions, date literals, FY/CY notation (these are points, not periods)

**Test scenarios:**
- Happy path: "this month", "next quarter", "Q1", "FQ3", "this April" → true
- Happy path: "today", "now", "next friday", "end of this month" → false
- Edge case: "this fiscal quarter" → true, "FY2026" → false (FY is a year-long period... but FY notation gives you the start point, not the span — implementation decision needed)

**Verification:**
- Period classification is complete for all known keyword patterns

## Risks & Dependencies

| Risk | Mitigation |
|------|------------|
| AST node not available in evalUnitConversion | The `UnitConversion.Quantity` field IS the AST node — verified in research |
| Period duration off-by-one | Use `end - start` in days (not +1) since DaysBetween already gives the correct count. Verify with known values: Q1 = Jan 1 to Mar 31 = 89 days via DaysBetween, but Q1 has 90 days. Need to verify whether +1 is needed |

## Sources & References

- Related issue: #125
- Related PR: #119 (relative date math)
- Key file: `impl/interpreter/unit_conversion_eval.go`
