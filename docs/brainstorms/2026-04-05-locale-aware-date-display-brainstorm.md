---
date: 2026-04-05
topic: locale-aware-date-display
---

# Locale-Aware Short Date Display

## Problem Frame

Date results display as `"Monday, January 2, 2006"` — verbose English regardless of the user's locale setting. This is inconsistent with the rest of the display system, where numbers and currencies already respect the `locale` config. Non-English users see untranslated day/month names, and the long format wastes space in the editor preview pane.

## Requirements

**Display Format**
- R1. Default date display format changes from full (`Monday, January 2, 2006`) to short (`Wed, Jan 12, 2025`)
- R2. Day-of-week and month names are localized using the user's `locale` setting from `config.toml`
- R3. Date element ordering follows locale convention (e.g., `Wed, Jan 12, 2025` for en-US vs `mer., 12 janv. 2025` for fr-FR)

**Integration**
- R4. Date formatting flows through the display `Formatter`, consistent with how Currency, Rate, Quantity, and Duration are formatted
- R5. The existing `locale` config key in `config.toml` drives date formatting — no new config keys needed

**Locale Data**
- R6. Locale data is a compact internal lookup table of abbreviated weekday and month names, not an external dependency
- R7. The table covers the same set of locales that `DisplayConfig` already supports for numeric formatting

## Success Criteria

- Date results in the TUI, JSON `value` field, and markdown export use the short localized format
- `locale = "fr-FR"` in config produces French abbreviated day/month names with day-before-month ordering
- No new external dependencies added

## Scope Boundaries

- Full `Date.String()` in the spec/types layer is unchanged (it's the precise/machine form)
- Time-of-day formatting is not in scope
- User-customizable date format strings (beyond locale) are not in scope
- Adding new locales beyond what `DisplayConfig` already handles is not in scope

## Key Decisions

- **Internal table over external library**: CalcMark's locale list is small and stable. A ~50-line lookup table avoids dependency risk and gives full control over output.
- **Short format with weekday**: `Wed, Jan 12, 2025` balances compactness with context. The weekday helps users verify date arithmetic results at a glance.
- **Format changes display layer only**: `Date.String()` stays as-is for backwards compatibility. Only the display formatter output changes.

## Dependencies / Assumptions

- The `DisplayConfig.Tag` field (BCP 47 language tag) is already parsed and available in the `Formatter` — verified in `format/display/config.go`

## Outstanding Questions

### Deferred to Planning
- [Affects R3][Needs research] What is the conventional short date format for each supported locale? (day-month vs month-day ordering, punctuation, abbreviation style)
- [Affects R7][Technical] Which locales does `DisplayConfig` currently handle? The planner should enumerate the full list from the existing separator extraction logic.

## Next Steps

`-> /ce:plan` for structured implementation planning
