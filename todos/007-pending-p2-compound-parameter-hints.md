---
status: pending
priority: p2
issue_id: "007"
tags: [tui, ux, autosuggest]
dependencies: []
---

# Add longer human-language hints to compound parameter placeholders

## Problem Statement

The status bar parameter hints for `compound()` show terse placeholders like `period: month, compounded monthly, compounded quarterly`. This is insufficient because the `periods` (3rd arg) has three different semantic interpretations depending on the 4th arg:

1. **No 4th arg:** `periods` = number of compounding periods, rate is per-period
2. **4th arg is a unit (e.g., `month`):** `periods` = count of that unit, rate is per-period
3. **4th arg is `compounded monthly`:** `periods` silently becomes **total duration in years**, rate becomes **annual rate** divided by compounding frequency (financial formula A = P(1+r/n)^(n×t))

Users typing `compound($10K, 5%, 10, compounded monthly)` have no indication that `10` now means "10 years" instead of "10 periods", or that the rate is now treated as annual.

## Proposed Solutions

### Option A: Add Description field to ParamSpec

Extend `ParamSpec` in `spec/semantic/function_types.go` with a `Description string` field that provides a human-readable explanation. Update `FormatParamHelp` in `cursor_context.go` to display description alongside examples.

For compound specifically:
- `principal`: "starting amount to grow"
- `rate`: "growth rate per period (or annual rate with compounded modifier)"
- `periods`: "number of periods (or duration in years with compounded modifier)"
- `period`: "optional: compounding frequency (e.g., compounded monthly)"

**Effort:** Small

### Option B: Contextual hints based on sibling arguments

More ambitious: hints that change based on which arguments are already filled in. When the user starts typing the 4th arg, the hint for the 3rd arg could update to clarify "duration in years (for financial compounding)".

**Effort:** Medium-Large

## Technical Details

**Affected files:**
- `spec/semantic/function_types.go:143-150` — ParamSpec definitions
- `cmd/calcmark/tui/editor/view_footer.go:72-82` — status bar rendering
- `cmd/calcmark/tui/editor/cursor_context.go:159-182` — FormatParamHelp

## Acceptance Criteria

- [ ] Each compound parameter shows a human-readable description in the status bar
- [ ] Description is distinct from the examples list
- [ ] Other functions with complex semantics also benefit from descriptions

## Work Log

| Date | Action | Learnings |
|------|--------|-----------|
| 2026-03-05 | Created during autocomplete bug fix | User confused by `periods` arg changing meaning based on 4th arg |
