---
created: 2026-02-08T18:00
title: Seconds to milliseconds conversion fails
area: interpreter
files:
  - spec/semantic/units.go
  - impl/interpreter/evaluator.go
---

## Problem

User reports: `a = 1.5 s` followed by `b = a in ms` fails with "incompatible units" error.

Seconds should be convertible to milliseconds - both are time units. The autocomplete also doesn't show `seconds` or `milliseconds` as options, suggesting these may not be registered properly in the unit system.

Need to investigate:
1. Are seconds/milliseconds in the canonical unit set?
2. Is the conversion path defined between them?
3. Why doesn't autocomplete show these units?

## Solution

Fixed in issue #31 PR. The root cause was:
1. `NormalizeTimeUnit()` in `spec/types/rate.go` was missing `ms`/`millisecond`/`milliseconds` entries.
2. The parser's `as` and `in` keyword handlers did not normalize time unit abbreviations before checking `IsValidDurationUnit()`.
3. `isTimeUnit()` in the parser was missing `"millisecond"` from its canonical forms switch.

Note: seconds and milliseconds were already in `durationToSecondsDecimal` (spec/types/duration.go) and `TimeUnits` (spec/lexer/date_keywords.go). The gap was in the parser's conversion path, not the unit definitions themselves. Time units are NOT in `spec/units/canonical.go` — they live in the types/lexer layer. Consolidating them into canonical.go is a separate architectural improvement.
