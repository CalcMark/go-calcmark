---
title: "Locale Formatting Bypasses in TUI Export, Footer, and Autocomplete"
date: 2026-03-02
category: ui-bugs
tags: [locale-formatting, tui-editor, display-formatting, bug-fix, postfix-currency]
severity: P1
component: "cmd/calcmark/tui"
symptoms:
  - "TUI export writes files with en-US formatting regardless of selected locale"
  - "Variable references in context footer display en-US formatting instead of locale-aware formatting"
  - "Autocomplete suggestions show en-US formatted values instead of locale-aware formatting"
  - "Postfix-code currencies (CNY, VND, KRW) lacked locale test coverage"
related_features:
  - locale-aware-formatting
  - unified-locale-formatting
---

# Locale Formatting Bypasses in TUI Export, Footer, and Autocomplete

## Problem

After implementing locale-aware formatting (the `feat/unified-locale-formatting` branch), three code paths in the TUI editor still bypassed the locale formatter. When a user set their locale to `de-DE`, exported files, context footer variable references, and autocomplete suggestions continued showing en-US formatting (e.g., `$1,500.00` instead of `$1.500,00`).

Additionally, postfix-code currencies (CNY, VND, KRW) had zero locale test coverage — all existing tests only used prefix-symbol currencies (`$`, `¥`).

## Root Cause

The locale-aware `Formatter` was added to `format.Options` and integrated into the main rendering path, but **three code paths were missed** that still used `fmt.Sprintf("%v", val)` instead of the locale-aware `displayFormat()` method:

1. **Export path** (`file_operations.go:122`): `exportFile()` constructed `format.Options` without passing `m.formatter`
2. **Footer path** (`view_footer.go:111`): `getLineReferences()` used `fmt.Sprintf("%v", val)` which calls the type's `String()` method (always en-US)
3. **Autocomplete path** (`model.go:399`): The variable suggestion source closure used `fmt.Sprintf` instead of the locale-aware formatter

The underlying issue is **architectural asymmetry**: `fmt.Sprintf("%v", val)` on a `types.Type` calls its `String()` method which always returns en-US format, while `m.displayFormat(val)` routes through the locale-aware formatter.

## Solution

### Fix 1: Export Locale Bug

**File:** `cmd/calcmark/tui/editor/file_operations.go` (line 122)

```go
// Before
opts := format.Options{Verbose: false, IncludeErrors: true}

// After
opts := format.Options{
    Verbose:          false,
    IncludeErrors:    true,
    DisplayFormatter: m.formatter,
}
```

### Fix 2: Context Footer Locale Bypass

**File:** `cmd/calcmark/tui/editor/view_footer.go` (line 111)

```go
// Before
knownVars[varName] = fmt.Sprintf("%v", val)

// After
knownVars[varName] = m.displayFormat(val)
```

Side effect: Removed unused `fmt` import.

### Fix 3: Autocomplete Locale Bypass

**File:** `cmd/calcmark/tui/editor/model.go` (line 399)

```go
// Before (inside closure in New())
result[name] = fmt.Sprintf("%v", val)

// After
result[name] = m.displayFormat(val)
```

The closure captures `m` by closure, so `m.displayFormat()` is accessible.

### Fix 4: Test Coverage Gap

Added 14 postfix-code currency test cases to `TestFormatterLocaleCurrency` in `format/display/locale_test.go`:

- **CNY** (2-decimal postfix code): Small, mid, large values across en-US, de-DE, fr-FR
- **VND** (0-decimal): en-US and de-DE with thousand separators
- **KRW** (0-decimal): Similar coverage
- **Negative values**: CNY with proper sign handling

## Key Insight

The bug manifests as a **pattern of missed integration points** rather than a single logical error. When new code paths are added to the TUI (export, footer, autocomplete), developers naturally reach for `fmt.Sprintf("%v", val)` — the simplest Go formatting idiom. But for `types.Type` values displayed to users, this always produces en-US output.

The fix establishes a consistent rule:
- **For user-visible type values**: Always use `m.displayFormat(val)` in TUI code
- **For `format.Options` consumers**: Always pass `DisplayFormatter: m.formatter`
- **For tests/debug only**: `fmt.Sprintf("%v", val)` is acceptable

## Prevention

### Detection: Grep for the Anti-Pattern

```bash
# Find potential locale bypasses in TUI code
grep -rn 'fmt\.Sprintf.*%v' cmd/calcmark/tui/ --include="*.go" | grep -v "_test.go"
```

### Code Review Checklist

When reviewing changes that display `types.Type` values:

- [ ] Is `m.displayFormat(val)` used instead of `fmt.Sprintf("%v", val)`?
- [ ] If constructing `format.Options`, is `DisplayFormatter` set?
- [ ] Does a locale test exist for the new code path?

### Test Strategy

Every new display path should have a test that:
1. Injects a non-default formatter (e.g., de-DE)
2. Asserts locale-specific formatting appears in output
3. Example: `TestExportFileLocale`, `TestEditorCatwalkPreviewPaneLocale`, `TestREPLLocaleFormatting`

## Files Modified

| File | Change |
|------|--------|
| `format/display/locale_test.go` | +14 postfix-code currency test cases |
| `cmd/calcmark/tui/editor/file_operations.go` | Wire `m.formatter` into export Options |
| `cmd/calcmark/tui/editor/view_footer.go` | Use `m.displayFormat()` for references |
| `cmd/calcmark/tui/editor/model.go` | Use `m.displayFormat()` for autocomplete |
| `cmd/calcmark/tui/editor/file_operations_test.go` | Add `TestExportFileLocale` |
| `cmd/calcmark/tui/editor/catwalk_test.go` | Add `TestEditorCatwalkPreviewPaneLocale` |
| `cmd/calcmark/tui/editor/testdata/preview_pane_locale/de_DE_currency` | Catwalk testdata |
| `cmd/calcmark/tui/repl/model_test.go` | Add `TestREPLLocaleFormatting` |

## Related Documentation

- `site/content/docs/cli-reference.md` — `--locale` flag documentation
- `site/content/docs/configuration.md` — Display locale config section
- `site/content/docs/user-guide.md` — Locale formatting guide
- `docs/plans/2026-03-02-feat-unified-locale-aware-formatting-plan.md` — Feature specification
- `format/display/config.go` — `DisplayConfig` and `NewConfig()` implementation
- `format/display/formatter.go` — `Formatter` value type with locale-aware methods
