---
title: Currency Code Output Spacing
date: 2026-02-23
status: ready-for-plan
category: output-formatting
---

# Currency Code Output Spacing

## What We're Building

Add a space between 3-letter ISO currency codes and the numeric amount in
CalcMark's output formatter. Symbol currencies ($, €, £, ¥) keep their
existing no-space format.

**Before:** `CNY1,000.00`, `INR11.55K`, `KRW81K`
**After:** `CNY 1,000.00`, `INR 11.55K`, `KRW 81K`

Symbol currencies are unchanged: `$100.00`, `€184.00`, `£4,300.00`

## Why This Approach

- **ISO 4217 convention**: Code-prefixed amounts use a space; symbol-prefixed
  do not. This is widely adopted in banking and finance.
- **Readability**: `CNY1,000.00` visually merges the code and digits.
  `CNY 1,000.00` is immediately parseable.
- **Minimal change**: A single conditional in `FormatCurrency()`. No parser
  changes, no internal representation changes.
- This output-only change is compatible with any future locale-aware formatting.

## Key Decisions

1. **Space for code currencies only** — Symbol currencies ($, €, £, ¥) keep
   no-space prefix. Code currencies (CNY, INR, KRW, etc.) get a space.
2. **Negative format: `-CNY 50.00`** — Sign before code, consistent with
   `-$50.00`. Locale-sensitive alternatives deferred.
3. **Output only** — This is purely a display formatting change. The parser
   and internal `Currency` type are untouched.
4. **Detection heuristic: `symbol == c.Code`** — After calling
   `GetCurrencySymbol(c.Code)`, if the returned symbol equals the code,
   it's a code currency that needs a space. Do NOT use `len(symbol) > 1`
   because `€` is 3 bytes, `£` is 2 bytes, and `¥` is 2 bytes in UTF-8 —
   byte length would incorrectly add spaces after those symbols.

## Scope

### In scope
- Modify `FormatCurrency()` in `format/display/display.go`
- Add new test cases in `format/display/display_test.go` for code currencies
  (CNY small/mid/large, KRW zero-decimal, negative code currency)
- Update golden testdata files that show code-currency output

### Out of scope
- Parser changes (input format stays the same)
- Locale/region frontmatter (`locale: en_US`)
- Postfix formatting (`1,000.00 CNY`)
- Custom per-currency formatting rules

## Affected Code

- `format/display/display.go` — `FormatCurrency()` (lines ~141-173)
  - After `symbol := types.GetCurrencySymbol(c.Code)`, add:
    ```go
    sep := ""
    if symbol == c.Code {
        sep = " "
    }
    ```
    Then use `sep` between symbol and numStr in the return statements.
- `format/display/display_test.go` — Add test cases for code currencies:
  - `CNY 100.00` (small), `CNY 1,000.00` (mid), `CNY 15K` (large)
  - `KRW 81000` (zero-decimal currency)
  - `-CNY 50.00` (negative code currency)
  - No existing code-currency test cases exist today; these are all new.
- `spec/types/currency.go` — No changes needed. `GetCurrencySymbol()`
  already returns the code string for unknown currencies.
