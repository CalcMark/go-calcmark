---
title: "Fix missing autosuggest help in status bar for optional args and incorrect category tag"
date: 2026-03-05
category: ui-bugs
tags:
  - autocomplete
  - status-bar
  - function-spec
  - tui
  - growth-functions
severity: minor
components:
  - cmd/calcmark/tui/editor/view_overlays.go
  - spec/semantic/function_types.go
symptoms:
  - Autocomplete popup displayed "gro" instead of "fn" for Growth-category functions
  - Status bar showed no parameter hint for compound()'s optional 4th argument
root_cause:
  - suggestionTag() switch statement missing "Growth" category
  - FunctionSpecs["compound"] only declared 3 of 4 parameters
github_issue: 30
---

# Fix Missing Autosuggest Help and Incorrect Category Tag

## Problem

GitHub Issue #30 reported two related bugs in the TUI autocomplete and status bar help system:

1. **Incorrect category tag**: Growth-category functions (`compound`, `grow`, `depreciate`) displayed a "gro" tag in the autocomplete popup instead of "fn".

2. **Missing parameter hint**: When typing `compound(1000, 5%, 10, ` the status bar showed no hint for the optional `period?` parameter.

## Root Cause

### Bug 1: "gro" tag

`suggestionTag()` in `cmd/calcmark/tui/editor/view_overlays.go` maps function categories to short tags for popup display. The switch statement handled "Math", "Conversion", "Network", "Storage", and "Capacity" but not "Growth". The default branch truncated to 3 lowercase chars: `strings.ToLower(category[:3])` producing "gro".

### Bug 2: Missing compound period hint

`FunctionSpecs["compound"]` in `spec/semantic/function_types.go` only declared 3 parameters (principal, rate, periods). The interpreter (`evalCompoundFunc`) accepts 3-4 arguments where the 4th is an optional period modifier. The function signature string already advertised `compound(principal, rate, duration, period?)`, but the FunctionSpec was incomplete. `GetParamAtIndex(3)` returned nil, so the status bar rendered nothing.

## Solution

### Bug 1 fix

Added "Growth" to the existing switch case in `suggestionTag()`:

```go
// Before
case "Math", "Conversion", "Network", "Storage", "Capacity":
    return "fn"

// After
case "Math", "Conversion", "Network", "Storage", "Capacity", "Growth":
    return "fn"
```

### Bug 2 fix

Added the missing optional parameter to `FunctionSpecs["compound"]`, following the pattern used by `depreciate` (salvage?) and `capacity` (buffer?):

```go
{Name: "period", Type: ArgTypeString, Optional: true, Examples: []string{"month", "compounded:monthly", "compounded:quarterly"}},
```

## Prevention

These bugs represent two general classes of drift:

### Incomplete switch on extensible value sets

When a new function category is added, the compiler gives no warning that `suggestionTag()` is missing a case. **Prevention**: Write a test that iterates over all known categories and asserts `suggestionTag()` returns a non-default result for each. This test fails automatically when a new category is added without updating the switch.

### Multiple representations of the same truth diverging

The compound function's parameter shape is defined in three places: the interpreter (runtime), the signature string (docs), and FunctionSpec (autocomplete). **Prevention**: Write a cross-layer consistency test (in `impl` or an integration test package, where both `spec` and `impl` are importable) that verifies FunctionSpec parameter counts match the interpreter's accepted argument range for every function.

## Related Documentation

- `docs/solutions/ui-bugs/overlay-compositing-ansi-state-bleed-through.md` -- autocomplete popup styling
- `docs/solutions/ui-bugs/tui-editor-rendering-divider-status-bar-error-line.md` -- status bar rendering
- `docs/solutions/ui-bugs/context-footer-statement-index-drift.md` -- context footer display drift
