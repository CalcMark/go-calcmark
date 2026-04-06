---
title: "feat: Locale-aware short date display"
type: feat
status: active
date: 2026-04-05
origin: docs/brainstorms/2026-04-05-locale-aware-date-display-brainstorm.md
---

# feat: Locale-aware short date display

## Overview

Replace the verbose English-only date display format (`Monday, January 2, 2006`) with a short locale-aware format (`Wed, Jan 12, 2025` for en-US, `mer., 12 janv. 2025` for fr-FR). This brings dates in line with the rest of the display system, where numbers, currencies, and rates already respect the user's `locale` config setting.

## Problem Frame

Date results display as full English day-of-week and month names regardless of locale. This is inconsistent with the locale-aware number and currency formatting, wastes space in the TUI preview pane, and shows untranslated text to non-English users. (see origin: `docs/brainstorms/2026-04-05-locale-aware-date-display-brainstorm.md`)

## Requirements Trace

- R1. Default date display changes from `Monday, January 2, 2006` to `Wed, Jan 12, 2025`
- R2. Day-of-week and month names are localized using the user's `locale` setting
- R3. Date element ordering follows locale convention (month-day vs day-month)
- R4. Date formatting flows through the display `Formatter`, consistent with all other types
- R5. The existing `locale` config key drives date formatting — no new config keys
- R6. Locale data is a compact internal lookup table, not an external dependency
- R7. The table covers at least the three tested locales: en-US, de-DE, fr-FR

## Scope Boundaries

- `Date.String()` in `spec/types/date.go` is **unchanged** — it is the machine/precise form
- JSON output `date_value` field (ISO 8601) is **unchanged** — handled by `format/json_formatter.go`
- Time-of-day formatting is not in scope
- User-customizable date format strings are not in scope
- Adding locales beyond the three tested ones is not in scope (but the table structure should make additions trivial)

## Context & Research

### Relevant Code and Patterns

- `format/display/formatter.go` — All `Format*` methods live here on value receiver `(f Formatter)`. The `Format()` type switch dispatches to them. `Date` is the only non-trivial type that lacks a dedicated method (line 84-85 calls `v.String()` directly).
- `format/display/display.go` — Package-level free functions delegate to `defaultFormatter`. Each `Format*` method has a matching free function.
- `format/display/config.go` — `DisplayConfig` holds `Tag language.Tag` already parsed from BCP 47. Currently only used for separator extraction. `FormatDate` can read `cfg.Tag.Base()` to select locale data.
- `format/display/locale_test.go` — Table-driven locale tests for en-US, de-DE, fr-FR. Pattern: `NewConfig(locale)` → `NewFormatter(cfg)` → call method → assert output.
- `cmd/calcmark/cmd/locale_test.go` — End-to-end locale test evaluating CalcMark source through the full pipeline.
- `cmd/calcmark/tui/editor/model.go:452-457` — TUI `displayFormat()` calls `m.formatter.Format(t)`, which will automatically pick up `FormatDate`.
- `impl/document/interpolation.go:88` — `{{variable}}` interpolation calls `df.Format(value)`, also auto-picks-up.

### Institutional Learnings

- **Locale bypass risk** (`docs/solutions/ui-bugs/locale-formatting-bypass-in-tui.md`): Every new display path needs locale wiring. Since `FormatDate` integrates into the existing `Formatter.Format()` dispatch, TUI and interpolation paths inherit it automatically. No separate wiring needed.
- **Display/model divergence** (`docs/solutions/ui-bugs/currency-code-output-spacing.md`): `String()` (machine-readable) and `Format*()` (display) intentionally diverge. Document this on both sides.
- **Map iteration non-determinism** (`docs/solutions/logic-errors/go-maps-non-deterministic-ordering-frontmatter.md`): Locale lookup tables use maps for O(1) lookup by key, but iteration order doesn't matter here — we always look up by specific key, never iterate.

### External References

Skipped — the codebase has strong local patterns for display formatting and locale handling. No external research needed.

## Key Technical Decisions

- **Internal lookup table keyed by `language.Base`**: Use `cfg.Tag.Base()` (the language subtag, e.g., `en`, `de`, `fr`) as the lookup key. This means `en-US` and `en-GB` share the same English day/month names, which is correct — date *element ordering* varies by region but abbreviation names don't within a language.
- **Locale-specific format pattern**: Each locale entry includes not just day/month names but also an ordering pattern (month-day-year vs day-month-year). This satisfies R3.
- **Fallback to English**: Unknown locales fall back to en-US data, matching the existing `DefaultConfig()` behavior.
- **Separate file for locale data**: The lookup table goes in a new `date_locale.go` file rather than crowding `formatter.go` or `config.go`. The `FormatDate` method stays in `formatter.go` with the other `Format*` methods.

## Open Questions

### Resolved During Planning

- **Where does the lookup table live?** New file `format/display/date_locale.go`. The table is ~40 lines of data, distinct from formatting logic.
- **How to handle `language.Base` matching?** `cfg.Tag.Base().String()` returns `"en"`, `"de"`, `"fr"`. Use this as the map key. Fallback: if key not found, use `"en"` entry.
- **Does this change break golden tests?** No — repo research confirmed no golden files assert on display-formatted date output. The JSON `date_value` field (ISO 8601) is untouched.

### Deferred to Implementation

- **Exact abbreviated month/day names for de-DE and fr-FR**: Research the conventional CLDR short forms during implementation. Examples: fr-FR uses `lun.`, `mar.`, `mer.` for weekdays and `janv.`, `févr.`, `mars` for months.

## High-Level Technical Design

> *This illustrates the intended approach and is directional guidance for review, not implementation specification. The implementing agent should treat it as context, not code to reproduce.*

```
dateLocaleData: map[string]dateLocale
  key: language base string ("en", "de", "fr")
  value: {
    shortDays:   [7]string     // Mon..Sun abbreviated
    shortMonths: [12]string    // Jan..Dec abbreviated
    orderFunc:   func(weekday, month, day, year) string
  }

FormatDate(d *Date) -> string:
  1. Look up locale data by f.cfg.Tag.Base().String()
  2. Fall back to "en" if not found
  3. Get abbreviated weekday and month from lookup table
  4. Apply orderFunc to produce final string
```

The `orderFunc` pattern handles divergent ordering:
- en: `Wed, Jan 12, 2025`
- de: `Mi., 12. Jan. 2025`
- fr: `mer., 12 janv. 2025`

## Implementation Units

- [ ] **Unit 1: Locale data table**

  **Goal:** Define the locale data structure and lookup table with entries for en, de, fr.

  **Requirements:** R2, R3, R6, R7

  **Dependencies:** None

  **Files:**
  - Create: `format/display/date_locale.go`
  - Test: `format/display/date_locale_test.go`

  **Approach:**
  - Define a `dateLocale` struct with short weekday names, short month names, and a format function
  - Store in a package-level `var dateLocales map[string]dateLocale`
  - Provide a `getDateLocale(tag language.Tag) dateLocale` helper that does the Base() lookup with en fallback
  - The format function for each locale encodes that locale's conventional date element ordering and punctuation

  **Patterns to follow:**
  - `format/display/config.go` `extractSeparators()` — locale-specific data extraction pattern
  - `format/display/display.go` `timeUnitAbbreviations` map — static lookup table pattern

  **Test scenarios:**
  - Happy path: `getDateLocale` with `language.AmericanEnglish` returns English data with 7 weekdays and 12 months
  - Happy path: `getDateLocale` with `language.German` returns German abbreviated names
  - Happy path: `getDateLocale` with `language.French` returns French abbreviated names
  - Edge case: `getDateLocale` with an unsupported locale (e.g., `language.Japanese`) falls back to English data
  - Happy path: Each locale's format function produces the correct element ordering — en: `Wed, Jan 12, 2025`, de: `Mi., 12. Jan. 2025`, fr: `mer., 12 janv. 2025`

  **Verification:**
  - All three locale entries produce correctly ordered, abbreviated date strings for a known reference date

- [ ] **Unit 2: FormatDate method and dispatch wiring**

  **Goal:** Add `FormatDate(*types.Date) string` to the `Formatter` type, wire it into the `Format()` dispatch, and add the package-level free function.

  **Requirements:** R1, R4, R5

  **Dependencies:** Unit 1

  **Files:**
  - Modify: `format/display/formatter.go`
  - Modify: `format/display/display.go`
  - Test: `format/display/display_test.go`

  **Approach:**
  - Add `FormatDate` method on `(f Formatter)` following the same nil-guard + lookup + format pattern as `FormatCurrency`
  - In `Format()` type switch, replace `case *types.Date: return v.String()` with `case *types.Date: return f.FormatDate(v)`
  - Add `FormatDate(d *types.Date) string` free function in `display.go` delegating to `defaultFormatter`

  **Patterns to follow:**
  - `format/display/formatter.go` `FormatCurrency()` — nil guard, read from `f.cfg`, transform, return
  - `format/display/display.go` `FormatCurrency()` — free function delegation

  **Test scenarios:**
  - Happy path: `FormatDate` with en-US config and a known date produces `Wed, Jan 12, 2025` format
  - Happy path: Package-level `FormatDate()` free function returns en-US formatted date (default formatter)
  - Edge case: `FormatDate` with nil `*types.Date` returns empty string
  - Happy path: `Format()` dispatch for a `*types.Date` value returns the short localized string, not the old verbose format

  **Verification:**
  - The `Format()` method produces short dates for Date values instead of `"Monday, January 2, 2006"` style
  - All existing tests still pass (no golden file breakage)

- [ ] **Unit 3: Locale-specific formatting tests**

  **Goal:** Add locale-specific tests for date formatting across en-US, de-DE, fr-FR, following the established locale test patterns.

  **Requirements:** R2, R3, R7

  **Dependencies:** Unit 2

  **Files:**
  - Modify: `format/display/locale_test.go`
  - Modify: `cmd/calcmark/cmd/locale_test.go`

  **Approach:**
  - Add `TestFormatterLocaleDate` in `locale_test.go` following the pattern of `TestFormatterLocaleNumbers` and `TestFormatterLocaleCurrency`
  - Add a date case to the end-to-end `TestEvalWithLocale_EndToEnd` in `cmd/calcmark/cmd/locale_test.go`

  **Patterns to follow:**
  - `format/display/locale_test.go` `TestFormatterLocaleNumbers` — table-driven, NewConfig → NewFormatter → call method → assert
  - `cmd/calcmark/cmd/locale_test.go` — end-to-end locale pipeline test

  **Test scenarios:**
  - Happy path: en-US formatter formats `Dec 25, 2025` as `Thu, Dec 25, 2025`
  - Happy path: de-DE formatter formats `Dec 25, 2025` as `Do., 25. Dez. 2025`
  - Happy path: fr-FR formatter formats `Dec 25, 2025` as `jeu., 25 déc. 2025`
  - Integration: End-to-end test evaluates `d = Dec 25 2025` with de-DE locale and asserts the display output contains German day/month abbreviations
  - Edge case: Date with `today` keyword formats correctly through the locale pipeline (not hardcoded to English)

  **Verification:**
  - `task test` passes with no regressions
  - Locale-specific date tests cover all three tested locales with correct element ordering

- [ ] **Unit 4: Documentation update**

  **Goal:** Update the language reference to document the new short date format and locale behavior.

  **Requirements:** R1, R2

  **Dependencies:** Unit 3

  **Files:**
  - Modify: `site/content/docs/language-reference.md`
  - Modify: `site/content/docs/user-guide/formatting.md` (if date display is documented there)

  **Approach:**
  - Update any documentation that shows or describes date output format
  - Note the locale-aware behavior in the formatting section

  **Test expectation:** none — documentation-only change

  **Verification:**
  - `task quality` passes
  - Documentation accurately describes the new short date format

## System-Wide Impact

- **Interaction graph:** `FormatDate` integrates into the existing `Formatter.Format()` dispatch. All consumers — TUI (`displayFormat()`), text/markdown/HTML formatters, and `{{variable}}` interpolation — automatically inherit the change with zero additional wiring.
- **Error propagation:** No new error paths. `FormatDate` is a pure formatting function that falls back to English on unknown locales.
- **State lifecycle risks:** None. The locale lookup table is read-only package-level data. No concurrency or mutation concerns.
- **API surface parity:** The JSON formatter's `date_value` field (ISO 8601) is **unchanged**. Only the `value` field in JSON output and all human-facing display paths are affected.
- **Unchanged invariants:** `Date.String()` in `spec/types/date.go` stays `"Monday, January 2, 2006"` — this is the model layer's precise representation. The JSON `date_value` stays ISO 8601. Only the display layer changes.

## Risks & Dependencies

| Risk | Mitigation |
|------|------------|
| Abbreviated names differ from CLDR standard for some locale | Use CLDR short forms as reference. The table is small and easy to correct. |
| New locales added later need manual table entries | Table structure makes additions trivial (one struct per locale). Document how to add. |
| Users expect the old verbose format | This is a display improvement, not a breaking API change. The JSON `date_value` is stable. |

## Sources & References

- **Origin document:** [docs/brainstorms/2026-04-05-locale-aware-date-display-brainstorm.md](docs/brainstorms/2026-04-05-locale-aware-date-display-brainstorm.md)
- Related code: `format/display/formatter.go`, `format/display/config.go`, `format/display/display.go`
- Related learnings: `docs/solutions/ui-bugs/locale-formatting-bypass-in-tui.md`, `docs/solutions/ui-bugs/currency-code-output-spacing.md`
