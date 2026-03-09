---
status: pending
priority: p2
issue_id: "010"
tags: [code-review, testing, json-formatter]
dependencies: []
---

# Missing JSON Formatter Test for Percentage Type

## Problem Statement

Every other type (number, currency, quantity, rate, duration, boolean) has a dedicated test in `format/json_formatter_test.go`. Percentage is missing. A regression could silently break percentage JSON output — the primary interface for agent/programmatic consumers.

## Findings

- Agent-native-reviewer identified this gap
- The JSON formatter code at `format/json_formatter.go:121-124` handles Percentage correctly
- But there's no test to guard against regressions

## Technical Details

**Affected files:**
- `format/json_formatter_test.go` — add `TestJSONFormatterPercentageType`

**Fix:** Add test verifying `type == "percentage"`, `numeric_value` is populated (fractional form), `value` shows display form (e.g., "20%"), and `unit` is empty.

## Acceptance Criteria

- [ ] `TestJSONFormatterPercentageType` exists and passes
- [ ] Covers percentage literal, widened result, and Percentage+Percentage
