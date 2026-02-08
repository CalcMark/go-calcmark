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

TBD - Check spec/units/canonical.go for time unit definitions and ensure seconds <-> milliseconds conversion is properly defined.
