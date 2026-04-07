---
date: 2026-04-06
topic: relative-date-math
---

# Relative Date Math

## Problem Frame

CalcMark supports basic date keywords (`today`, `tomorrow`, `yesterday`) and date arithmetic (`Dec 25 + 2 weeks`), but lacks the relative temporal expressions people naturally use in calculations: "next Friday", "two weeks ago", "this quarter", "next April 25". Users doing financial modeling, project planning, or everyday calculations must manually compute these reference points, which defeats CalcMark's promise of natural-language computation.

The feature surface is large and semantically treacherous — "next Friday" means different things to different people, fiscal quarters vary by organization, and named months can be points or spans. Getting this wrong erodes trust in a calculation tool. This is why we must delegate calendar math to Go's `time` package (or a trusted library) and validate with a test-first approach: write expectations before implementation.

## Architecture Principle: Data vs. Presentation

Internal datetime values are **precise and unambiguous** — full `time.Time` precision, always UTC. Formatting is a **separate concern** with sensible defaults and config overrides. No formatting decision should constrain the internal representation, and no internal precision should leak into display without explicit user intent.

## Requirements

**Relative Weekday Expressions**

- R1. Support `this <weekday>`, `next <weekday>`, and `last <weekday>` where weekday is Monday–Sunday. Use **narrow-window semantics**:
  - `next <weekday>`: the soonest future occurrence of that weekday. If today is that weekday, it means next week's.
  - `last <weekday>`: the most recent past occurrence. If today is that weekday, it means last week's.
  - `this <weekday>`: the occurrence in the current calendar week (Monday–Sunday), whether past or future.
- R2. Bare `<weekday>` (e.g., `Friday`) is shorthand for `this <weekday>`. This introduces weekday names as reserved keywords. Planning must assess whether any existing .cm documents use weekday names as variable identifiers and determine lexer disambiguation rules. If backward-compatibility risk is non-trivial, R2 may be deferred or require a migration path.

**Relative Named Month Expressions**

- R3. Support `next <month>`, `last <month>`, and `this <month>` where month is January–December. Resolves to a **datetime** at midnight on the 1st of that month. `next April` = midnight April 1 of the next occurrence of April strictly after the current month. If the current month is April, `this April` = April 1 of this year; `next April` = April 1 of next year.
- R4. Support `next <month> <day>` for specific dates (e.g., `next April 25` = April 25 of the next future occurrence). `next February 29` in a non-leap year resolves to the next leap year's Feb 29 — the forward-scan must detect Go's silent date normalization (where `time.Date(2025, 2, 29, ...)` becomes March 1) and skip invalid dates.

**Duration-Relative Expressions**

- R5. Support `<N> <unit> ago` (e.g., `2 weeks ago`, `10 minutes ago`, `3 months ago`). Subtracts the duration from the current datetime.
- R6. Support `<N> <unit> from now` and `<N> <unit> from <date-expr>` (e.g., `2 weeks from now`, `3 days from next Friday`). The `from` syntax partially exists and is tested — extend to compose with all new relative date expressions.
- R7. Support `<duration> tomorrow`, `<duration> yesterday` as shorthand for `<duration> from tomorrow` (e.g., `two weeks tomorrow` = `2 weeks from tomorrow`). Planning must determine whether this creates parser ambiguity with other expression forms — if it does, defer R7 rather than introducing fragile grammar rules.

**Relative Period Expressions (Week / Month / Year)**

- R8. Complete evaluation for `this/next/last week`, `this/next/last month`, `this/next/last year`. Tokens already exist in the lexer but evaluation is stubbed. Each resolves to the **first day** of the respective period:
  - `this week` = Monday of the current week. `next week` = Monday of next week. `last week` = Monday of last week.
  - `this month` = 1st of current month. `next month` = 1st of next month. `last month` = 1st of last month.
  - `this year` = Jan 1 of current year. `next year` = Jan 1 of next year. `last year` = Jan 1 of last year.

**Calendar Quarter Expressions**

- R9. Support `this quarter`, `next quarter`, `last quarter`. Resolves to the **first day** of the respective calendar quarter (Q1=Jan, Q2=Apr, Q3=Jul, Q4=Oct). `Q1` through `Q4` are shorthand for the current year's calendar quarters. `Q1` always means calendar quarter — see R12 for fiscal.

**Fiscal Year and Quarter Expressions**

- R10. Support a **frontmatter key** to configure fiscal year start month (e.g., `fiscal_year_starts: july`). This follows the existing frontmatter pattern used by `exchange`, `scale`, `convert_to`, and `measurement` directives. When set, `this fiscal quarter`, `next fiscal quarter`, `this fiscal year`, `next fiscal year` resolve relative to the configured fiscal calendar.
- R11. Without a fiscal year frontmatter key, fiscal keywords (`FQ1`, `FY26`, `this fiscal quarter`, etc.) produce a clear diagnostic: "fiscal expressions require a 'fiscal_year_starts' frontmatter key".
- R12. **Notation conventions:**
  - `Q1`–`Q4`: Always **calendar** quarter. `Q1` = Jan–Mar of the current year.
  - `FQ1`–`FQ4`: **Fiscal** quarter. Requires fiscal_year_starts. FQ1 begins at the configured start month. E.g., with `fiscal_year_starts: july`, FQ1 = Jul–Sep, FQ2 = Oct–Dec, FQ3 = Jan–Mar, FQ4 = Apr–Jun.
  - `FY26`, `FY2026`: Fiscal year 2026. Resolves to the first day of the fiscal year (e.g., July 1, 2025 for Microsoft's FY26 with July start).
  - `CY2001`, `CY01`: Explicit calendar year. Resolves to Jan 1 of that year. Disambiguates from a bare integer.
  - Bare `2001`: Remains an integer, not a date. CalcMark does not guess whether a bare number is a year. Users must use `CY2001` or a date literal (`Jan 1 2001`) to express a year as a datetime.
- R13. `this fiscal quarter` resolves to the **first day** of the current fiscal quarter (same pattern as calendar quarters in R9). E.g., with `fiscal_year_starts: july`, on August 15 → July 1 (first day of FQ1).

**Calendar Correctness**

- R14. All date arithmetic MUST correctly handle leap years, leap seconds, month-length variations (28/29/30/31 days), and DST-adjacent edge cases. CalcMark must not implement its own calendar math — it must delegate to Go's `time` package or a trusted, well-tested library. Hand-rolling calendar arithmetic is explicitly prohibited.
- R15. Edge cases must be tested: Feb 29 + 1 year (leap → non-leap), Jan 31 + 1 month, `next February 29` in a non-leap year.
- R16. Duration arithmetic must be **unit-aware**, not seconds-based. Month and year durations must use `time.AddDate()` for calendar-correct results. Sub-day durations (hours, minutes, seconds) must use `time.Add()`. The current approach of converting all durations to seconds (where 1 month = 2,592,000 seconds = exactly 30 days) is incorrect for calendar arithmetic — `3 months ago` from April 6 must be January 6, not January 5.

**DateTime Unification**

- R17. Promote `Date` to carry full `time.Time` precision internally (defaulting to midnight UTC). All relative expressions produce a unified datetime value. The internal representation is always precise — formatting decides what to show.
- R18. Sub-day duration arithmetic applies to datetimes: `next Friday + 2 hours` = Friday at 2:00 AM. `10 minutes ago` = current datetime minus 10 minutes. This requires replacing `evalDateDurationOperation`'s current integer-days-only path with unit-aware dispatch.
- R19. The existing `Time` type (standalone `3:00 PM`) remains separate for time-of-day literals that are not anchored to a date.

**Precision**

- R20. Internal datetime precision is nanosecond (Go's `time.Time`). Display precision is **seconds** — sub-second components are not shown. Duration arithmetic preserves full internal precision but results display at second granularity. Sub-second duration units (milliseconds, microseconds, nanoseconds) remain valid in duration expressions and arithmetic — they just don't surface in datetime display.

**Display Formatting**

- R21. Smart display default: datetime values display **date-only** when the time component is exactly midnight (e.g., `next Friday` → `Fri, Apr 10, 2027`). When the time is non-midnight, display both date and time (e.g., `next Friday + 2 hours` → `Fri, Apr 10, 2027 2:00 AM`). This is the only behavior — no config knob. If a user need to force time display emerges later, add it then.
- R22. Time display format (12-hour vs 24-hour) defaults to the locale convention (e.g., `en_US` → 12-hour, `de_DE` → 24-hour) with an explicit override config key (e.g., `time_format: 24h`). Lives in the same config layer as `locale`.
- R23. Extend the existing locale-aware date formatting system to cover time formatting. Same locale key, same fallback chain. **Blocked on the locale-aware date display work** (brainstorm `2026-04-05-locale-aware-date-display`) — either that lands first, or R21–R23 ship as part of it.

**Cross-Locale Formatting Tests**

- R24. Cross-locale formatting tests must cover every locale in the existing `dateFormats` table in `format/display/date_locale.go` (currently 18+ locales), not a subset. At minimum this includes: Japanese date with ISO time, Chinese date without time, German 24-hour format, US 12-hour format, Korean format, and mixed locale edge cases (e.g., locale set but `time_format` overridden).

**Test-First Development**

- R25. Every requirement above must have failing tests written **before** implementation begins. Test expectations define the contract — implementation fulfills it. This includes:
  - Parser tests for each new expression form
  - Evaluator tests with pinned reference dates (not `time.Now()`) for deterministic assertions
  - Formatting tests across locales
  - Edge case tests for calendar correctness (R15)
  - Error diagnostic tests for fiscal expressions without configuration (R11)

## Success Criteria

- Relative date expressions compose naturally with existing duration arithmetic: `next Friday + 2 weeks - 3 days` works.
- `next Friday + 2 hours` produces a datetime with correct time component, displayed with smart omission.
- `3 months ago` from April 6 = January 6 (calendar-correct, not 90 days back).
- `Q1` = calendar quarter, `FQ1` = fiscal quarter, `FY26` = fiscal year — all resolve correctly with and without fiscal configuration.
- Bare `2001` remains an integer. `CY2001` = Jan 1 2001.
- Fiscal quarter expressions produce correct dates for any configured start month, validated against known corporate fiscal calendars (e.g., Microsoft FY starts July → `FY27` in August 2026 = July 1, 2026).
- All existing date tests continue to pass — backwards compatibility with current `today`, `tomorrow`, `yesterday`, date literals, and duration arithmetic.
- Cross-locale display tests cover all locales in the existing locale table.
- Every requirement has a corresponding test that was written before implementation.

## Scope Boundaries

- **Not in scope:** Natural language parsing beyond the defined syntax (no "the day after next Friday" or free-form English).
- **Not in scope:** Timezone-aware computation. All datetimes are UTC internally. Timezone support is a separate future feature.
- **Not in scope:** Date spans/ranges as a type. Named months and quarters resolve to points (first day), not intervals.
- **Not in scope:** Recurring events or cron-like expressions.
- **Not in scope:** Relative time-of-day expressions like "this morning" or "tonight".
- **Not in scope:** Bare integers as implicit years. `2001` is a number, not a date.

## Key Decisions

- **Narrow-window semantics for weekdays**: `next Friday` = the soonest future Friday. Matches GNU date, chrono-node, Ruby Chronic. Least surprising for a calculation tool.
- **Named months resolve to points**: `next April` = midnight April 1, not a span. Keeps the type system simple and composable.
- **Promote Date to DateTime internally**: Single type with full `time.Time` precision. Avoids a doubled operator matrix. Smart display handles presentation.
- **Data vs. presentation separation**: Internal values are always precise UTC datetimes. Formatting is a separate layer with sensible defaults and config overrides. No formatting decisions constrain internal precision.
- **Fiscal configuration via frontmatter**: Follows existing frontmatter pattern (`fiscal_year_starts: july`). Per-document, not global.
- **Q = calendar, FQ = fiscal, FY = fiscal year, CY = calendar year**: Unambiguous notation. Bare numbers are never years.
- **Unit-aware duration arithmetic**: Month/year operations use `AddDate()`, sub-day uses `Add()`. No seconds-based approximation for calendar units.
- **Locale-driven time format with override**: 12h/24h defaults from locale, explicit override via config. Extends the existing locale system.
- **Second-precision display**: Internal nanosecond precision (Go `time.Time`), display at seconds. No epoch-millisecond representation — Go's `time.Time` uses wall clock internally with no overflow concerns.
- **Test-first**: Failing tests define the contract before implementation begins.

## Dependencies / Assumptions

- The locale-aware date display work (brainstorm `2026-04-05-locale-aware-date-display`) is in progress. R21–R23 are blocked on or merged with that work.
- Go's `time.Time` handles leap years, month-length variations, and calendar arithmetic correctly via `AddDate()` and `Add()`. This is the foundation — no custom calendar math.
- The existing `from` keyword and duration arithmetic operators provide the foundation for R5–R7.
- The `RelativeDateKeywords` map in `spec/lexer/date_keywords.go` already has token slots for `this/next/last week/month/year` but evaluation is stubbed (tests skipped). These are unevaluated stubs to build on, not working features.
- The existing `evalDateDurationOperation` converts durations to integer days via `durationToDays()`, discarding sub-day precision. R16 and R18 require replacing this with unit-aware dispatch — this is an architectural change to the operator path.

## Outstanding Questions

### Resolve Before Planning

_(None — all product decisions resolved during brainstorm.)_

### Deferred to Planning

- [Affects R1, R3][Needs research] Should CalcMark adopt a Go third-party library (e.g., `jinzhu/now` for quarter calculations) or implement relative date resolution using Go's `time` package directly? The expression set is bounded enough that a hand-rolled solution atop `time` may be simpler than adapting a library, but leap year and month-boundary handling must be thoroughly tested either way.
- [Affects R17, R18][Technical] How should the `types.Date` promotion work internally? The existing `NewDateFromTime` normalizes to midnight. Options: (a) add a `hasTime bool` field so display knows whether time was explicitly set vs. defaulted, or (b) always store full precision and let display check for midnight. Option (a) preserves the "data tells you how it was constructed" principle.
- [Affects R2][Technical] Assess whether any existing .cm documents or testdata use weekday names (`Monday`–`Sunday`) as variable identifiers. If so, R2 is a breaking change and needs a migration path or must be deferred.
- [Affects R7][Technical] Determine whether `<duration> tomorrow` shorthand creates parser ambiguity. If ambiguous, defer R7.
- [Affects R16, R18][Technical] The current `evalDateDurationOperation` and `durationToDays()` must be redesigned for unit-aware dispatch. Plan the migration: which duration units route to `AddDate()` vs. `Add()`, how compound durations (e.g., `1 month and 2 hours`) are handled.
- [Affects R21–R23][Needs research] What are the correct default time formats (12h/24h, separator, AM/PM labels) for each locale in the existing `dateFormats` table?
- [Affects R1, R2][Technical] Weekday names need new lexer tokens. The existing `RelativeDateKeywords` phrase-matching handles `"next"` as a prefix — plan how `"next"` followed by a weekday vs. month vs. `"week"`/`"month"`/`"year"`/`"quarter"` is disambiguated without greedy consumption of user-defined identifiers.

## Next Steps

-> `/ce:plan` for structured implementation planning
