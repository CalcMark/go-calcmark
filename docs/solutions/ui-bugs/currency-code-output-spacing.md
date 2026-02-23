---
title: Currency code spacing in formatted output
category: ui-bugs
tags:
  - formatting
  - currency
  - internationalization
  - utf8
module: format/display
symptom: |
  3-letter ISO currency codes displayed directly adjacent to numeric amounts
  with no space (CNY1,000.00, INR11.55K), reducing readability. Symbol
  currencies appeared correctly spaced due to single-character visual
  separation.
root_cause: |
  FormatCurrency() concatenated symbol + numStr with no separator. The naive
  fix len(symbol) > 1 fails because € is 3 bytes, £ is 2 bytes, and ¥ is
  2 bytes in UTF-8. The correct heuristic is symbol == c.Code.
date: 2026-02-23
severity: medium
---

# Currency Code Spacing in Formatted Output

## Problem

`FormatCurrency()` produced output like `CNY1,000.00` and `INR11.55K` for
ISO 3-letter code currencies. Symbol currencies rendered correctly as
`$1,000.00` because a single glyph naturally separates from digits visually.
Three-letter codes jammed against digits are not readable.

## Root Cause

The formatter concatenated `symbol + numStr` with no separator. The obvious
fix — checking `len(symbol) > 1` to detect multi-character codes — is
incorrect because UTF-8 encoding makes byte length unreliable for this:

- `€` (EURO SIGN) is 3 bytes
- `£` (POUND SIGN) is 2 bytes
- `¥` (YEN SIGN) is 2 bytes

All three would incorrectly satisfy `len(symbol) > 1`, gaining an unwanted
space. The correct observation is that `GetCurrencySymbol()` returns the ISO
code unchanged when no symbol mapping exists, so `symbol == c.Code` is true
if and only if the code itself is being used as the display string.

## Solution

Add a `sep` variable in `FormatCurrency()` set to `" "` only when no symbol
mapping exists:

```go
sep := ""
if symbol == c.Code {
    sep = " "
}

if isNegative {
    return "-" + symbol + sep + numStr
}
return symbol + sep + numStr
```

`Currency.String()` intentionally does NOT add this space — it is the
machine-readable representation used by the JSON formatter.

## Verification

12 new test cases in `TestUnifiedCurrencyFormat`:
- Code currencies across all magnitude bands (small, mid, large K/M)
- Zero amount, negative small, negative large
- Zero-decimal currencies (VND, KRW)
- Regression guards: JPY/EUR/GBP must NOT gain a space

## Prevention

### UTF-8 String Length Is Not Character Count

In Go, `len(s)` returns byte count, not rune count. For any comparison
involving Unicode strings where you care about "number of characters":

- Use `utf8.RuneCountInString(s)` for rune count
- Or use semantic equality (`symbol == c.Code`) when the question is
  really "is this a code or a symbol?"

The semantic check is preferred — it expresses intent directly rather than
relying on an assumption about character counts.

### Document Intentional Divergence

When two representations of the same value intentionally differ (display
`CNY 1,000.00` vs machine `CNY1000.00`), add comments on both sides
explaining the divergence. Future developers will otherwise "fix" the
inconsistency.

## Related

- [exchange-rate-frontmatter-validation.md](../logic-errors/exchange-rate-frontmatter-validation.md) — Currency validation patterns
- `format/display/display.go` — `FormatCurrency()` (the changed function)
- `spec/types/currency.go` — `GetCurrencySymbol()`, `CodeToSymbol` map
- [brainstorm](../../brainstorms/2026-02-23-currency-code-output-spacing-brainstorm.md) — Design decisions and approach exploration
