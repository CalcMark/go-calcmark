---
created: 2026-03-05T17:00
title: Consolidate time unit definitions into spec/units/canonical.go
area: spec
files:
  - spec/units/canonical.go
  - spec/types/rate.go
  - spec/types/duration.go
  - spec/lexer/date_keywords.go
---

## Problem

Time unit definitions are fragmented across 4 maps in 3 files, violating Single Source of Truth. Adding or modifying any time unit requires coordinated changes across multiple files with no compile-time guarantee of consistency.

Current locations:
1. `spec/lexer/date_keywords.go` — `TimeUnits` map (lexer tokenization aliases, includes `secs`, `mins`, `hrs`)
2. `spec/types/rate.go` — `timeUnitAliases` package var (parser/interpreter normalization, includes `daily`, `weekly`, `quarterly`)
3. `spec/types/duration.go` — `durationToSecondsDecimal` (conversion factors, canonical + plural forms)
4. `spec/types/duration.go` — `DurationToSeconds` (int64 version, excludes millisecond since 0.001 can't be int64)

These maps have overlapping but not identical entries. For example:
- `TimeUnits` has `"secs"` but `timeUnitAliases` does not
- `timeUnitAliases` has `"daily"`, `"weekly"`, `"quarterly"` but `TimeUnits` does not
- `DurationToSeconds` (int64) excludes `"millisecond"` but `durationToSecondsDecimal` includes it

This fragmentation caused issue #31 (`ms` not recognized as millisecond) and will cause similar bugs for any new time unit (microsecond, nanosecond, etc.).

## Proposed approach

Extend `spec/units/canonical.go` with a Time/Duration quantity section using the existing `UnitMapping` struct pattern. Add conversion factors (as `decimal.Decimal`) either to `UnitMapping` or a companion map keyed by canonical name.

Then derive or replace the 4 scattered maps from this single source:
- `NormalizeTimeUnit()` reads from `canonical.go` aliases
- `IsValidDurationUnit()` checks `canonical.go` entries
- `TimeUnits` in the lexer reads from `canonical.go` aliases
- `durationToSecondsDecimal` derives from `canonical.go` conversion factors

## Context

Identified during code review of the issue #31 fix. The maintainer flagged that `canonical.go` should be the single source of truth for all units including time.
