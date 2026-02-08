---
created: 2026-02-08T05:45
title: Fix autosuggest duplicative function names
area: tui
files:
  - cmd/calcmark/tui/editor/autocomplete.go
  - cmd/calcmark/tui/components/suggest.go
---

## Problem

The autosuggest popup shows duplicative text for function suggestions, displaying the function name twice:

```
throughput throughput(network_type)
transfer_time transfer_time(size, scope, network_ty...
```

Expected format should be either:
- `throughput(network_type)` (just the signature)
- `throughput` with signature shown elsewhere

The duplication makes the popup harder to read and wastes horizontal space.

## Solution

TBD - Check how FunctionSuggestionSource builds suggestion labels. The label likely includes both the match text and the full signature when it should only include one.
