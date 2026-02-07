---
created: 2026-02-07T09:06
title: convert_rate does not show correct preview result
area: interpreter
files:
  - impl/interpreter/convert_eval.go
  - format/display/display.go
---

## Problem

The `convert_rate` function does not display a correct preview result in the TUI editor.

Example that fails:
```
convert_rate(10 mb/s, per hour)
```

The calculation should convert 10 MB/s to the equivalent rate per hour (36 GB/hour), but the preview shows an incorrect or unexpected result.

This may be related to:
- Rate type formatting in `display.go`
- Unit conversion in `convert_eval.go`
- How the "per hour" target is parsed and applied

## Solution

TBD - Needs investigation:
1. Check what `convert_rate` returns for this input
2. Verify Rate type formatting in FormatRate()
3. Check if "per hour" is correctly parsed as a time unit conversion target
