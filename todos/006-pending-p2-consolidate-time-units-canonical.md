---
status: pending
priority: p2
issue_id: "006"
tags: [code-review, architecture, spec]
dependencies: []
---

# Consolidate Time Unit Definitions into spec/units/canonical.go

## Problem Statement

Time unit definitions are fragmented across 4 maps in 3 files, violating Single Source of Truth. Adding or modifying any time unit requires coordinated changes across multiple files with no compile-time guarantee of consistency.

This fragmentation caused issue #31 (`ms` not recognized as millisecond) and will cause similar bugs for any new time unit (microsecond, nanosecond, etc.).

## Findings

Current locations:
1. `spec/lexer/date_keywords.go` — `TimeUnits` map (lexer tokenization aliases, includes `secs`, `mins`, `hrs`)
2. `spec/types/rate.go` — `timeUnitAliases` package var (parser/interpreter normalization, includes `daily`, `weekly`, `quarterly`)
3. `spec/types/duration.go` — `durationToSecondsDecimal` (conversion factors, canonical + plural forms)
4. `spec/types/duration.go` — `DurationToSeconds` (int64 version, excludes millisecond since 0.001 can't be int64)

These maps have overlapping but not identical entries:
- `TimeUnits` has `"secs"` but `timeUnitAliases` does not
- `timeUnitAliases` has `"daily"`, `"weekly"`, `"quarterly"` but `TimeUnits` does not
- `DurationToSeconds` (int64) excludes `"millisecond"` but `durationToSecondsDecimal` includes it

## Proposed Solutions

### Option A: Extend canonical.go with Time/Duration section

Extend `spec/units/canonical.go` with a Time/Duration quantity section using the existing `UnitMapping` struct pattern. Add conversion factors (as `decimal.Decimal`) either to `UnitMapping` or a companion map keyed by canonical name.

Then derive or replace the 4 scattered maps from this single source:
- `NormalizeTimeUnit()` reads from `canonical.go` aliases
- `IsValidDurationUnit()` checks `canonical.go` entries
- `TimeUnits` in the lexer reads from `canonical.go` aliases
- `durationToSecondsDecimal` derives from `canonical.go` conversion factors

**Pros:** Single source of truth, consistent with existing physical unit pattern, compile-time safe.
**Cons:** Larger refactor, `UnitMapping` may need a conversion factor field.
**Effort:** Medium

### Option B: Create spec/units/time.go as dedicated time unit registry

New file alongside `canonical.go` specifically for time units, with its own struct that includes conversion factors. Other files derive from it.

**Pros:** Separation of concerns (time units have different needs than physical units), smaller change.
**Cons:** Two unit registries instead of one.
**Effort:** Small-Medium

## Technical Details

**Affected files:**
- `spec/units/canonical.go`
- `spec/types/rate.go`
- `spec/types/duration.go`
- `spec/lexer/date_keywords.go`

## Acceptance Criteria

- [ ] All time unit aliases defined in exactly one place
- [ ] `NormalizeTimeUnit()`, `IsValidDurationUnit()`, `TimeUnits`, and `durationToSecondsDecimal` all derive from the single source
- [ ] Adding a new time unit (e.g., microsecond) requires changes in only one file
- [ ] All existing tests pass without modification

## Work Log

| Date | Action | Learnings |
|------|--------|-----------|
| 2026-03-05 | Created during issue #31 review | Fragmentation caused #31; maintainer wants canonical.go as single source |

## Resources

- Issue #31: https://github.com/CalcMark/go-calcmark/issues/31
- `spec/units/canonical.go` — existing physical unit registry pattern
