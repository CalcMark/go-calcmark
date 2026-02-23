---
title: "feat: Add space between ISO currency codes and amounts in output"
type: feat
status: completed
date: 2026-02-23
brainstorm: docs/brainstorms/2026-02-23-currency-code-output-spacing-brainstorm.md
---

# feat: Add space between ISO currency codes and amounts in output

## Overview

Add a space between 3-letter ISO currency codes and the numeric amount in
CalcMark's display formatter. Symbol currencies ($, €, £, ¥) are unchanged.

| Before | After |
|---|---|
| `CNY1,000.00` | `CNY 1,000.00` |
| `INR11.55K` | `INR 11.55K` |
| `-CNY50.00` | `-CNY 50.00` |
| `$100.00` | `$100.00` (unchanged) |
| `¥5000` | `¥5000` (unchanged) |

## Acceptance Criteria

- [x] `FormatCurrency()` inserts a space for code currencies using `symbol == c.Code`
- [x] Symbol currencies ($, €, £, ¥) have NO space — regressions tested
- [x] Code currencies (CNY, INR, KRW, VND, etc.) have a space in all magnitude bands
- [x] Negative code currencies format as `-CNY 50.00` (sign before code)
- [x] Zero-decimal code currencies format correctly (e.g., `KRW 5,000`, `VND 5,000`)
- [x] All existing tests pass unchanged
- [x] New unit tests cover code-currency formatting across all paths
- [x] `task test` and `task quality` pass

## Implementation

### 1. Modify `FormatCurrency()` — `format/display/display.go`

After line 147 (`symbol := types.GetCurrencySymbol(c.Code)`), add:

```go
// Code currencies (CNY, INR) get a space; symbol currencies ($, €) do not.
sep := ""
if symbol == c.Code {
    sep = " "
}
```

Change the two return statements (lines 170, 172):

```go
// Before:
return "-" + symbol + numStr       // line 170
return symbol + numStr             // line 172

// After:
return "-" + symbol + sep + numStr // line 170
return symbol + sep + numStr       // line 172
```

**Do NOT use `len(symbol) > 1`** — `€` is 3 bytes, `£` and `¥` are 2 bytes in UTF-8.

### 2. Add unit tests — `format/display/display_test.go`

Add these cases to `TestUnifiedCurrencyFormat`:

```go
// ISO code currencies — space between code and amount
{"ISO code small", "42.50", "CNY", "CNY 42.50"},
{"ISO code mid-range", "1500", "CNY", "CNY 1,500.00"},
{"ISO code large K", "15000", "CNY", "CNY 15K"},
{"ISO code large M", "1500000", "CNY", "CNY 1.5M"},
{"ISO code zero", "0", "CNY", "CNY 0.00"},
{"ISO code negative small", "-50.00", "CNY", "-CNY 50.00"},
{"ISO code negative large", "-15000", "CNY", "-CNY 15K"},
{"VND zero-decimal", "5000", "VND", "VND 5000"},
{"KRW zero-decimal", "5000", "KRW", "KRW 5000"},

// Regression: symbol currencies must NOT gain a space
{"JPY symbol unaffected", "5000", "JPY", "¥5000"},
{"EUR symbol unaffected", "100", "EUR", "€100.00"},
{"GBP symbol unaffected", "100", "GBP", "£100.00"},
```

### 3. Update `FormatCurrency` doc comment

Add an ISO-code example to the function comment showing the space behavior.

### 4. Add comment to `Currency.String()` — `spec/types/currency.go`

Document that `String()` does NOT add a space — it is the precise/machine
representation. `FormatCurrency()` is the display representation. This makes
the intentional divergence from the JSON formatter explicit.

## Not Changed

- **Parser** — input format is unchanged
- **`Currency.String()`** — machine-readable representation, no space (JSON formatter uses this)
- **`spec/types/currency.go`** — no structural changes
- **Golden eval testdata** — `TestEvalFilesEvaluate` only checks that evaluation succeeds, not output format

## Notes

- **WASM**: Shares the same `format/display` package. No separate fix needed.
- **TUI**: Preview pane, REPL, and pinned variables all go through `display.Format()` — they inherit the change automatically.
- **Backward compatibility**: The `cm eval` text output format changes for code currencies. Any external script parsing `CNY1000` from stdout will need updating. This is an acceptable output format improvement.

## References

- Brainstorm: [docs/brainstorms/2026-02-23-currency-code-output-spacing-brainstorm.md](../brainstorms/2026-02-23-currency-code-output-spacing-brainstorm.md)
- `format/display/display.go:141-173` — `FormatCurrency()`
- `spec/types/currency.go:106-111` — `GetCurrencySymbol()`
- `spec/types/currency.go:74-79` — `CodeToSymbol` (only USD, EUR, GBP, JPY)
- Learnings: `docs/solutions/logic-errors/exchange-rate-frontmatter-validation.md` — validate at every entry point
- Learnings: `docs/solutions/ui-bugs/lipgloss-background-bleed-through.md` — TUI styling gaps
