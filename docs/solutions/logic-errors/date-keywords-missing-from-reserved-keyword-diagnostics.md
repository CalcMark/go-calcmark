---
title: Date keywords missing from reserved keyword diagnostics
date: 2026-04-06
category: logic-errors
module: calcmark-diagnostics
problem_type: logic_error
component: tooling
symptoms:
  - "Writing 'today = Mar 3 2021' silently became a text block with no diagnostic warning"
  - "Date keywords (today, tomorrow, yesterday) bypassed reservedKeywordDiagnostic() entirely"
  - "IsReservedKeywordToken() only checked ReservedKeywords map, not DateKeywords map"
root_cause: logic_error
resolution_type: code_fix
severity: medium
tags:
  - reserved-keywords
  - date-keywords
  - diagnostics
  - lexer
  - token
  - classifier
  - calcmark
---

# Date keywords missing from reserved keyword diagnostics

## Problem

Date keywords (`today`, `tomorrow`, `yesterday`) used as variable names (e.g., `today = Mar 3 2021`) silently became text blocks with no diagnostic warning. Regular reserved keywords like `end` correctly warned users, but date keywords were invisible to the entire diagnostic chain.

## Symptoms

- `end = 5` correctly emits: `"end" is a reserved keyword and cannot be used as a variable name`
- `today = Mar 3 2021` silently becomes a text block — no warning, no error
- Same for `tomorrow` and `yesterday`
- Users get no guidance that date keywords cannot be used as variable names

## What Didn't Work

The original reserved keyword warning implementation (for `end`, `if`, etc.) only registered `ReservedKeywords` in the `reservedKeywordTokens` set during `init()`. When `DateKeywords` were added to the lexer as a separate category in `spec/lexer/date_keywords.go`, they were never included in `IsReservedKeywordToken()`. This meant the diagnostic pipeline had a blind spot for an entire keyword category.

The failure was silent because the diagnostic pipeline has two gates that both failed:

1. **Calculation indicator gate** (`impl/document/diagnostic.go`): The `reserved_keyword_assignment` indicator checked `IsReservedKeywordToken(tokens[0].Type)` — returned false for `DATE_TODAY`
2. **Assignment indicator gate**: The `assignment` indicator checked `tokens[0].Type == IDENTIFIER` — `DATE_TODAY` is not `IDENTIFIER`

With neither indicator matching, `looksLikeFailedCalculation()` returned false and no diagnostic was generated.

## Solution

**Change 1** — Extend `init()` in `spec/lexer/token.go` to include `DateKeywords`:

```go
// Before: only ReservedKeywords
func init() {
    reservedKeywordTokens = make(map[TokenType]bool, len(ReservedKeywords))
    for _, tt := range ReservedKeywords {
        reservedKeywordTokens[tt] = true
    }
}

// After: includes DateKeywords
func init() {
    reservedKeywordTokens = make(map[TokenType]bool, len(ReservedKeywords)+len(DateKeywords))
    for _, tt := range ReservedKeywords {
        reservedKeywordTokens[tt] = true
    }
    for _, tt := range DateKeywords {
        reservedKeywordTokens[tt] = true
    }
}
```

**Change 2** — Add context-appropriate hint suggestions in `impl/document/diagnostic.go`:

```go
var dateKeywordSuggestions = map[string]string{
    "today":     "start_date",
    "tomorrow":  "next_day",
    "yesterday": "prev_day",
}
```

Used in `reservedKeywordDiagnostic()` to replace the generic `_val` suffix:

```go
suggestion := dateKeywordSuggestions[strings.ToLower(keyword)]
if suggestion == "" {
    suggestion = keyword + "_val"
}
```

## Why This Works

The diagnostic pipeline already had the correct logic in its `reserved_keyword_assignment` calculation indicator — it checked `IsReservedKeywordToken(tokens[0].Type) && tokens[1].Type == ASSIGN`. The only issue was that `IsReservedKeywordToken()` was built from `ReservedKeywords` alone, excluding `DateKeywords` tokens (`DATE_TODAY`, `DATE_TOMORROW`, `DATE_YESTERDAY`).

Adding one loop to `init()` populates the lookup map with date keyword tokens, which unblocks the entire existing diagnostic chain: indicator matching → failed-parse detection → reserved keyword warning generation. No new indicator or diagnostic path was needed.

## Prevention

- **Keyword registration discipline**: When adding new keyword categories to the lexer (like `DateKeywords` was), they must be included in `reservedKeywordTokens` if they should not be used as variable names. This is the same "incomplete keyword set" anti-pattern documented in `compound-bare-frequency-modifier-silently-ignored.md`.
- **Exhaustive token assertions**: The `TestIsReservedKeywordToken` test now iterates over both `ReservedKeywords` and `DateKeywords`, so missing registrations cause test failures immediately.
- **Cross-layer awareness**: The diagnostic pipeline spans three layers: lexer (tokenization) → classifier (calc vs text) → evaluator (diagnostic generation). A keyword category invisible to the middle layer's indicators silently drops through all downstream diagnostics. When adding keywords, trace the full pipeline.

### Diagnostic pipeline flow reference

```
Line input
  → Lexer tokenizes (spec/lexer/)
  → Classifier determines calc vs text block (spec/document/detector.go)
  → For text blocks: evaluator scans for likely failed calculations (impl/document/evaluator.go)
    → calculationIndicators check token patterns (impl/document/diagnostic.go)
    → If indicator matches AND parse fails → looksLikeFailedCalculation returns true
    → reservedKeywordDiagnostic() generates specific warning message
    → Diagnostic attached to TextBlock for TUI/CLI display
```

## Related Issues

- GitHub issue: #109
- Related pattern: `docs/solutions/logic-errors/compound-bare-frequency-modifier-silently-ignored.md` — same "incomplete keyword set" anti-pattern affecting a different guard function (`isNaturalSyntaxKeyword`)
- Related architecture: `docs/solutions/language-features/directive-as-value-cross-layer-learnings.md` — documents the classifier as "the most commonly missed layer" when adding language features
- Related checklist: `docs/solutions/integration-issues/nl-functional-syntax-parity-and-doc-staleness.md` — keyword-guard checklist should be extended to include `IsReservedKeywordToken`
