---
title: "fix: Date keywords should warn when used as variable names"
type: fix
status: active
date: 2026-04-06
---

# fix: Date keywords should warn when used as variable names

## Overview

`today = Mar 3 2021` silently becomes a text block with no diagnostic. It should warn the user that `today` is a reserved keyword, matching the behavior of `end = 5`.

## Problem Frame

Date keywords (`today`, `tomorrow`, `yesterday`) are tokenized as `DATE_TODAY` etc. by the lexer before the classifier sees them. The diagnostic system's `IsReservedKeywordToken()` only checks the `ReservedKeywords` map, not `DateKeywords`. The calculation indicator system also only checks `IsReservedKeywordToken`, so the line doesn't even get flagged as a "likely failed calculation."

## Requirements Trace

- R1. `today = Mar 3 2021` produces a warning: `"today" is a reserved keyword and cannot be used as a variable name`
- R2. Same behavior for `tomorrow` and `yesterday`
- R3. The hint suggests a sensible alternative variable name (e.g., `start_date` not `today_val`)

## Scope Boundaries

- Multi-word date keywords (`this week`, `next month`, `last year`) are out of scope — they contain spaces and don't look like variable assignments
- Making date keywords assignable (overriding the keyword) is explicitly not in scope — the decision is to warn

## Context & Research

### Relevant Code

- `spec/lexer/token.go:307-324` — `reservedKeywordTokens` map built from `ReservedKeywords` only; `IsReservedKeywordToken()` checks it
- `spec/lexer/date_keywords.go:5-9` — `DateKeywords` map with `today`, `tomorrow`, `yesterday`
- `impl/document/diagnostic.go:59-84` — `calculationIndicators` list: both `assignment` and `reserved_keyword_assignment` miss date keyword tokens
- `impl/document/diagnostic.go:172-196` — `reservedKeywordDiagnostic()` uses `IsReservedKeywordToken()`

### Token flow for `today = Mar 3 2021`

```
Lexer: DATE_TODAY("today"), ASSIGN("="), DATE_LITERAL("March:3:2021"), EOF
Classifier: text block (not recognized as calculation)
Diagnostic: no indicator matches → no warning
```

## Key Technical Decisions

- **Extend `IsReservedKeywordToken` to include date keywords**: This is the narrowest fix. Adding `DATE_TODAY`, `DATE_TOMORROW`, `DATE_YESTERDAY` to `reservedKeywordTokens` at init time fixes both the indicator check and the `reservedKeywordDiagnostic` check in one change.
- **Custom hint text for date keywords**: `today_val` is an awkward suggestion. Use a mapping: `today` → `start_date`, `tomorrow` → `next_day`, `yesterday` → `prev_day`.

## Implementation Units

- [ ] **Unit 1: Extend reservedKeywordTokens to include date keywords**

  **Goal:** Make `IsReservedKeywordToken` return true for `DATE_TODAY`, `DATE_TOMORROW`, `DATE_YESTERDAY`.

  **Requirements:** R1, R2

  **Dependencies:** None

  **Files:**
  - Modify: `spec/lexer/token.go`
  - Test: `spec/lexer/lexer_reserved_test.go`

  **Approach:**
  - In the `init()` function that builds `reservedKeywordTokens`, also add entries from `DateKeywords`
  - This makes both `calculationIndicators` and `reservedKeywordDiagnostic` recognize date keyword tokens

  **Patterns to follow:**
  - The existing `init()` at `spec/lexer/token.go:311-315` — same loop pattern over `DateKeywords`

  **Test scenarios:**
  - Happy path: `IsReservedKeywordToken(DATE_TODAY)` returns true
  - Happy path: `IsReservedKeywordToken(DATE_TOMORROW)` returns true
  - Happy path: `IsReservedKeywordToken(DATE_YESTERDAY)` returns true
  - Happy path: Existing reserved keywords (END, IF) still return true

  **Verification:**
  - `go test ./spec/lexer/ -run TestReservedKeyword` passes

- [ ] **Unit 2: Better hint text for date keywords**

  **Goal:** When a date keyword is used as a variable name, suggest a meaningful alternative instead of appending `_val`.

  **Requirements:** R3

  **Dependencies:** Unit 1

  **Files:**
  - Modify: `impl/document/diagnostic.go`
  - Test: `impl/document/diagnostic_test.go`

  **Approach:**
  - In `reservedKeywordDiagnostic`, after detecting a date keyword assignment, use a small map to pick a better suggestion: `today` → `start_date`, `tomorrow` → `next_day`, `yesterday` → `prev_day`
  - For non-date reserved keywords, keep the existing `keyword + "_val"` behavior

  **Patterns to follow:**
  - The existing `reservedKeywordDiagnostic` at `impl/document/diagnostic.go:172-196`

  **Test scenarios:**
  - Happy path: `today = Mar 3 2021` produces warning with hint suggesting `start_date`
  - Happy path: `tomorrow = Mar 4 2021` produces warning with hint suggesting `next_day`
  - Happy path: `yesterday = Mar 2 2021` produces warning with hint suggesting `prev_day`
  - Happy path: `end = 5` still produces hint suggesting `end_val` (regression check)
  - Integration: End-to-end test evaluating `today = Mar 3 2021` through full pipeline produces diagnostic in JSON output

  **Verification:**
  - `task test` passes with all new and existing diagnostic tests green

## Risks & Dependencies

| Risk | Mitigation |
|------|------------|
| Other code checks `IsReservedKeywordToken` and date keywords cause unexpected behavior | Grep for all call sites — verify each is compatible with date keywords being included |

## Sources & References

- Related issue: #109
- Related code: `spec/lexer/token.go`, `impl/document/diagnostic.go`
