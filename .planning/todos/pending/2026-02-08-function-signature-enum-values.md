---
created: 2026-02-08T06:15
title: Function signature help shows incomplete enum values
area: tui
files:
  - cmd/calcmark/tui/components/suggest.go
  - spec/semantic/function_types.go
---

## Problem

The `transfer_time` function signature help shows only a subset of network types (`gigabit, ten_gig`) when `wifi` is also a valid option.

For function signature items that have a strict enumeration of possible values, we should show ALL valid values, not just a subset.

This affects any function parameter that has enumerated valid values - users can't discover all valid options from the help text.

## Solution

TBD - Check how function signature help is built in suggest.go. The function definitions in function_types.go should contain the complete list of valid enum values. Ensure the help rendering pulls from the canonical source and displays all options.
