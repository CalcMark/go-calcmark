---
title: "feat: Relative Date Math"
type: feat
status: active
date: 2026-04-06
origin: docs/brainstorms/2026-04-06-relative-date-math-requirements.md
issue: 119
---

# feat: Relative Date Math

## Overview

Add relative temporal expressions to CalcMark: weekday references (`next Friday`, `last Tuesday`), named month references (`next April`, `next April 25`), duration-relative expressions (`2 weeks ago`, `10 minutes ago`), calendar and fiscal quarters (`Q1`, `FQ1`, `FY26`, `CY2001`), and complete evaluation of existing `this/next/last week/month/year` stubs. Includes DateTime unification, unit-aware duration arithmetic, smart display formatting, and full surface integration (TUI autosuggest, LSP completions, classifier).

## Problem Frame

CalcMark supports basic date keywords and date arithmetic but lacks the relative temporal expressions people naturally use. Users doing financial modeling, project planning, or everyday calculations must manually compute reference points like "next Friday" or "this quarter." The feature must work correctly across leap years, month boundaries, and fiscal calendars — delegating all calendar math to Go's `time` package. (see origin: `docs/brainstorms/2026-04-06-relative-date-math-requirements.md`)

## Requirements Trace

All 25 requirements (R1–R25) from the origin document are addressed. Key groupings:

- **R1–R2**: Relative weekday expressions (narrow-window semantics)
- **R3–R4**: Relative named month expressions (point resolution)
- **R5–R6**: Duration-relative (`ago`, `from`)
- **R7**: Deferred — `<duration> tomorrow` shorthand creates parse error (see Resolved During Planning)
- **R8**: Period expressions (`this/next/last week/month/year`)
- **R9–R13**: Calendar and fiscal quarters, FY/CY notation, frontmatter
- **R14–R16**: Calendar correctness, unit-aware arithmetic
- **R17–R19**: DateTime unification
- **R20**: Second-precision display
- **R21–R23**: Smart display, locale time formatting
- **R24**: Cross-locale formatting tests
- **R25**: Test-first development

## Scope Boundaries

- No natural language parsing beyond defined syntax (see origin)
- No timezone-aware computation — all datetimes UTC
- No date spans/ranges as a type
- No recurring events or cron-like expressions
- No relative time-of-day expressions ("this morning", "tonight")
- Bare integers are never years — `2001` stays a number
- R7 deferred — `from` keyword required (see Resolved During Planning)

## Context & Research

### Relevant Code and Patterns

**Lexer layer:**
- `spec/lexer/date_keywords.go` — `DateKeywords` (single-word), `RelativeDateKeywords` (two-word phrases), `MonthNames`, `TimeUnits` maps
- `spec/lexer/date_tokenizer.go` — `tryReadDateKeyword()` dispatches via `peekWord()` then `peekTwoWords()`. Two-word limit requires extension to three-word for fiscal expressions
- `spec/lexer/token.go:100-118` — existing DATE_* token types. New tokens follow this pattern

**Parser layer:**
- `spec/parser/primary.go:328-337` — all date tokens matched in single `p.match()` call → `RelativeDateLiteral`
- `spec/parser/primary.go:358-392` — `FROM` expression: `DURATION_LITERAL` + `FROM` + target → `BinaryOp`
- `spec/parser/primary.go:584-624` — `parseFromTarget()` accepts today/tomorrow/yesterday/date-literal

**AST layer:**
- `spec/ast/nodes.go` — `RelativeDateLiteral{Keyword, SourceText}` is already general enough for new keywords
- New node types needed: `CalendarQuarterLiteral`, `FiscalQuarterLiteral`, `FiscalYearLiteral`, `CalendarYearLiteral`

**Evaluator layer:**
- `impl/interpreter/datetime.go:75-89` — `evalRelativeDateLiteral` handles today/tomorrow/yesterday only. Calls `time.Now()` directly (no injection)
- `impl/interpreter/operators.go:467-479` — `evalDateDurationOperation` converts to integer days via `durationToDays()`. Discards sub-day precision
- `impl/interpreter/operators.go:702-720` — `durationToDays()` uses seconds conversion. Month = 30 days, year = 365 days (approximate)

**Types layer:**
- `spec/types/date.go:33-37` — `NewDateFromTime` normalizes to midnight UTC, discarding time
- `spec/types/duration.go:29-33` — month = 2,592,000s, year = 31,536,000s (fixed, not calendar-correct)

**Frontmatter:**
- `spec/document/frontmatter.go` — pattern: add field to `Frontmatter` struct, field to `frontmatterYAML`, parse function, add to `knownKeys` map, add serialization

**Display:**
- `format/display/date_locale.go` — 18+ locale entries with `monday` library for i18n
- `format/display/formatter.go:280-286` — `FormatDate` calls `getDateFormat` then `formatDate`

**Surface layers (gaps identified):**
- `spec/classifier/classifier.go` — **missing** date token checks entirely
- `cmd/calcmark/tui/editor/autocomplete.go` — **missing** date keyword suggestions
- `lsp/completion.go` — **missing** date completion items
- `spec/document/detector.go:376-388` — `isDateToken()` properly handles all 15 date token types (OK)

### Institutional Learnings

1. **12-layer checklist** (`docs/solutions/language-features/directive-as-value-cross-layer-learnings.md`): Classifier is the #1 most commonly missed layer. Full checklist: lexer → parser → AST → semantic checker → **classifier** → document detector → interpreter → scale exemption → interpolation → TUI autocomplete → TUI side-by-side → LSP completions.

2. **Keyword registration discipline** (`docs/solutions/logic-errors/date-keywords-missing-from-reserved-keyword-diagnostics.md`): New date keywords must be registered in ALL relevant sets: `reservedKeywordTokens`, `isNaturalSyntaxKeyword()`, and guard functions. The diagnostic pipeline spans lexer → classifier → evaluator.

3. **Rate arithmetic widening** (`docs/solutions/feature-gaps/rate-type-arithmetic-widening.md`): Type dispatch template for operators. Add normalization block early in `evalBinaryOperation()`. Test full type dispatch matrix.

4. **Display formatting bypass** (`docs/solutions/ui-bugs/locale-formatting-bypass-in-tui.md`): Always use `m.displayFormat(val)`, never `fmt.Sprintf("%v", val)`. Three TUI paths (export, footer, autocomplete) were missed previously — same paths need datetime formatting.

5. **User config leaks** (`docs/solutions/test-failures/user-config-leaks-into-tests.md`): Use `TestMain` with isolated HOME. For date tests: `t.Setenv("TZ", "UTC")` or clock injection.

6. **Feature registry** (`docs/solutions/code-organization/unified-feature-registry-three-to-one.md`): Register all feature metadata in `spec/features/registry.go` only. Interpreter's `FunctionDef` should contain only Name + Eval.

## Key Technical Decisions

- **Clock injection**: Add `TimeFunc func() time.Time` field to the Interpreter struct. All date evaluation uses this instead of `time.Now()`. Defaults to `time.Now` in production. Tests inject a pinned clock. This is the prerequisite for all deterministic date testing.

- **Date promotion via `HasTime` flag**: Add `HasTime bool` to `types.Date`. `NewDateFromTime` continues to normalize to midnight with `HasTime: false`. New `NewDateTime(t time.Time)` preserves full precision with `HasTime: true`. Display layer checks `HasTime` for smart omission. This preserves the "data tells you how it was constructed" principle (see origin).

- **Unit-aware duration dispatch**: Duration arithmetic routes by unit category:
  - Year/month → `time.AddDate(years, months, 0)` — calendar-correct
  - Week/day → `time.AddDate(0, 0, days)` — exact
  - Hour/minute/second/sub-second → `time.Add(duration)` — nanosecond-precise
  - Compound durations (e.g., `1 month and 2 hours`) → apply each component sequentially in descending unit order

- **Tokenization strategy**: Extend existing patterns:
  - Bare weekdays → `DateKeywords` map (single word) with new `DATE_WEEKDAY` token
  - `this/next/last <weekday>` → `RelativeDateKeywords` map (21 entries) with 3 new tokens: `DATE_THIS_WEEKDAY`, `DATE_NEXT_WEEKDAY`, `DATE_LAST_WEEKDAY`. Weekday name carried in token value
  - `this/next/last <month>` → `RelativeDateKeywords` map (36 entries) with 3 new tokens: `DATE_THIS_MONTH_NAME`, `DATE_NEXT_MONTH_NAME`, `DATE_LAST_MONTH_NAME`
  - `this/next/last quarter` → `RelativeDateKeywords` map (3 entries) with 3 new tokens
  - `this/next/last fiscal quarter/year` → add `peekThreeWords()` for 6 entries with 6 new tokens
  - `Q1`–`Q4`, `FQ1`–`FQ4`, `FY26`, `CY2001` → new tokenizer function for prefix+number patterns
  - `ago` → new `AGO` token type in `DateKeywords`

- **Fiscal via frontmatter**: `fiscal_year_starts: july` following the `measurement` config parsing pattern. Stored as `*FiscalYearConfig` on `Frontmatter` struct. Threaded to evaluator via document context.

- **R7 deferred**: Parser research confirmed `2 weeks tomorrow` is a parse error — the parser requires a newline or `from` keyword after a duration. The existing `2 weeks from tomorrow` works correctly. No fragile grammar changes.

## Open Questions

### Resolved During Planning

- **R7 `<duration> tomorrow` shorthand**: Creates parse error — parser expects newline after `2 weeks`. The `from` keyword is already required and works (`2 weeks from tomorrow`). **Decision: Defer R7.** Document the `from` syntax as the canonical form.

- **R2 backward compatibility**: Zero weekday names used as identifiers in any testdata `.cm` file. **Decision: Safe to add as reserved keywords.** No migration path needed.

- **Fiscal tokenization**: **Decision: `peekThreeWords` extending the existing pattern.** Three-word phrases like `this fiscal quarter` become single tokens, consistent with how `this week` is a single token. Six bounded combos (this/next/last × fiscal quarter/year).

- **Third-party library**: **Decision: Use Go's `time` package directly.** The expression set is bounded. `time.AddDate()` and `time.Add()` handle all calendar math correctly. Adding a library like `jinzhu/now` would introduce a dependency for functionality CalcMark can build with ~200 lines of evaluator code atop `time`. Quarter calculations are simple arithmetic.

- **Fiscal directive syntax**: **Decision: Frontmatter key** (`fiscal_year_starts: july`). Every existing directive uses frontmatter. No reason to invent a new mechanism.

### Deferred to Implementation

- Exact `peekThreeWords` implementation — whether to generalize `peekTwoWords` or add a separate function
- How `parseFromTarget` is extended for new relative date expressions as from-targets
- Whether `ContainsScaleRef` in AST needs updates for new node types
- Exact locale → 12h/24h mapping table (research needed during Unit 8)

## High-Level Technical Design

> *This illustrates the intended approach and is directional guidance for review, not implementation specification. The implementing agent should treat it as context, not code to reproduce.*

### Expression Grammar (Pseudo-BNF)

```
relative-date   := weekday-expr | month-expr | period-expr | quarter-expr
                 | fiscal-expr | ago-expr | from-expr | notation-expr

weekday-expr    := ("this" | "next" | "last") WEEKDAY
                 | WEEKDAY                          -- shorthand for "this WEEKDAY"

month-expr      := ("this" | "next" | "last") MONTH
                 | ("next" | "last") MONTH DAY      -- "next April 25"

period-expr     := ("this" | "next" | "last") ("week" | "month" | "year")

quarter-expr    := ("this" | "next" | "last") "quarter"

fiscal-expr     := ("this" | "next" | "last") "fiscal" ("quarter" | "year")

ago-expr        := DURATION "ago"                   -- "2 weeks ago"

from-expr       := DURATION "from" date-expr        -- "3 days from next Friday"

notation-expr   := "Q" [1-4]                        -- calendar quarter
                 | "FQ" [1-4]                       -- fiscal quarter
                 | "FY" DIGITS                      -- fiscal year
                 | "CY" DIGITS                      -- calendar year

WEEKDAY         := "Monday" | "Tuesday" | ... | "Sunday"
MONTH           := "January" | "Jan" | ... | "December" | "Dec"
DURATION        := NUMBER UNIT ("and" NUMBER UNIT)*
```

### Data Flow

```
                    ┌─────────┐
                    │  Lexer  │  peekWord / peekTwoWords / peekThreeWords
                    └────┬────┘
                         │ Tokens (DATE_NEXT_WEEKDAY, AGO, Q_LITERAL, etc.)
                    ┌────▼────┐
                    │ Parser  │  parsePrimary → RelativeDateLiteral / new AST nodes
                    └────┬────┘
                         │ AST nodes
                    ┌────▼────────────┐
                    │   Evaluator     │  evalRelativeDateLiteral + new eval functions
                    │  (TimeFunc)     │  Uses time.AddDate / time.Add
                    └────┬────────────┘
                         │ types.Date (with HasTime flag)
              ┌──────────┼──────────┐
              │          │          │
         ┌────▼───┐ ┌───▼────┐ ┌──▼───┐
         │  TUI   │ │  HTML  │ │  LSP │
         │Display │ │Format  │ │      │
         └────────┘ └────────┘ └──────┘
              │          │          │
              ▼          ▼          ▼
         FormatDateTime (smart omission, locale-aware 12h/24h)
```

## Implementation Units

### Phase 1: Foundation

- [x] **Unit 1: Clock injection for deterministic testing**

**Goal:** Enable pinned-time testing for all date evaluation. This is the prerequisite for every subsequent unit.

**Requirements:** R25 (test-first), R14 (calendar correctness testing)

**Dependencies:** None

**Files:**
- Modify: `impl/interpreter/interpreter.go` (add `TimeFunc` field)
- Modify: `impl/interpreter/datetime.go` (use `TimeFunc` instead of `time.Now()`)
- Modify: `impl/interpreter/date_test.go` (inject pinned clocks, unskip extended tests)
- Test: `impl/interpreter/date_test.go`

**Approach:**
- Add `TimeFunc func() time.Time` to the `Interpreter` struct, defaulting to `time.Now` in the constructor
- Replace `time.Now()` call in `evalRelativeDateLiteral` with `i.TimeFunc()`
- **Also address `types.NewDate` clock leak:** `spec/types/date.go:18` calls `time.Now().Year()` when year is 0 (for date literals like `Dec 25` without a year). Since `spec` cannot depend on `impl`, add a `NewDateWithYear(month, day, defaultYear)` variant or pass the year explicitly from `evalDateLiteral` using `i.TimeFunc().Year()`. This ensures year-less date literals also respect the pinned clock
- Add a test helper `newTestInterpreterWithClock(t time.Time)` that returns an interpreter with a fixed clock
- Convert existing date tests to use pinned clocks where they currently use `time.Now()` at test start
- Unskip the 9 `TestExtendedRelativeDates` test cases — they will fail with "not implemented" errors, confirming the test infrastructure works. Unit 5 will implement the evaluation to make them pass

**Execution note:** Start with the test helper, then modify the interpreter. Existing tests must continue passing.

**Patterns to follow:**
- The `Interpreter` struct already holds document-level config. Adding `TimeFunc` follows the same pattern
- Test helpers in `impl/interpreter/` use table-driven tests with `parser.Parse` → `interp.Eval`

**Test scenarios:**
- Happy path: `newTestInterpreterWithClock(fixedTime)` evaluates `today` → returns fixedTime's date, not actual today
- Happy path: `tomorrow` with pinned clock 2026-12-31 → returns 2027-01-01 (year boundary)
- Happy path: `yesterday` with pinned clock 2026-01-01 → returns 2025-12-31
- Edge case: Default `TimeFunc` (nil/unset) behaves identically to current `time.Now()` behavior
- Integration: Existing `TestDateKeywords`, `TestDateArithmeticEval`, `TestDateDifference` all still pass

**Verification:**
- All existing date tests pass with no behavior change
- New test helper works with arbitrary pinned dates
- The 9 extended relative date tests are unskipped and fail with "not implemented" (expected — Unit 5 implements them)

---

- [x] **Unit 2: Date type promotion with HasTime flag**

**Goal:** Enable `types.Date` to carry full time precision while preserving backward compatibility for date-only values.

**Requirements:** R17, R20 (datetime unification, nanosecond internal precision)

**Dependencies:** None (can run parallel with Unit 1)

**Files:**
- Modify: `spec/types/date.go` (add `HasTime`, new factory, updated methods)
- Test: `spec/types/date_test.go`

**Approach:**
- Add `HasTime bool` field to `Date` struct
- `NewDateFromTime(t)` continues normalizing to midnight with `HasTime: false` — all existing callers unchanged
- Add `NewDateTime(t time.Time) *Date` — preserves full precision, sets `HasTime: true`
- `AddDays` returns `HasTime: false` (day-level operation). New `AddDuration` returns `HasTime: true` when sub-day components present
- `String()` behavior: if `HasTime && time is not midnight`, include time in output. Otherwise date-only. This is the model's machine representation; display formatting is separate
- `DaysBetween` continues working (both normalized and non-normalized dates)

**Execution note:** Test-first. Write tests for the new factory and display behavior before implementing.

**Patterns to follow:**
- The `IsExplicit` bool pattern on `types.Quantity` (per institutional learnings). `HasTime` follows the same principle — a metadata flag that affects display but not arithmetic semantics

**Test scenarios:**
- Happy path: `NewDateFromTime(someTime)` → midnight, `HasTime: false`
- Happy path: `NewDateTime(time.Date(2026, 4, 10, 14, 30, 0, 0, time.UTC))` → preserves 14:30, `HasTime: true`
- Happy path: `NewDateTime` at exactly midnight → `HasTime: true` (explicitly constructed with time)
- Edge case: `DaysBetween` works between a HasTime=true and HasTime=false date
- Edge case: `String()` on HasTime=false → date only. `String()` on HasTime=true with non-midnight → includes time. `String()` on HasTime=true at midnight → includes time (explicitly set)

**Verification:**
- All existing `types.Date` tests pass unchanged
- New factory produces dates with time precision
- `HasTime` flag is preserved through methods

---

- [x] **Unit 3: Unit-aware duration arithmetic**

**Goal:** Replace the seconds-based duration-to-days conversion with calendar-correct unit-aware dispatch. Fixes the "1 month = 30 days" bug.

**Requirements:** R16 (unit-aware arithmetic), R14 (calendar correctness), R18 (sub-day arithmetic)

**Dependencies:** Unit 1 (clock injection for testing), Unit 2 (HasTime for sub-day results)

**Files:**
- Modify: `impl/interpreter/operators.go` (replace `evalDateDurationOperation`, `durationToDays`)
- Test: `impl/interpreter/date_test.go`

**Approach:**
- Replace `evalDateDurationOperation` with unit-aware dispatch:
  - `year` / `years` → `time.AddDate(N, 0, 0)`, result `HasTime: false`
  - `month` / `months` → `time.AddDate(0, N, 0)`, result `HasTime: false`
  - `week` / `weeks` → `time.AddDate(0, 0, N*7)`, result `HasTime: false`
  - `day` / `days` → `time.AddDate(0, 0, N)`, result `HasTime: false`
  - `hour` / `hours`, `minute` / `minutes`, `second` / `seconds`, sub-second → `time.Add(duration)`, result `HasTime: true`
- For compound durations: the lexer already tokenizes compound durations (e.g., `"3:week:4:day"`) but the parser currently discards all but the first component — `DurationLiteral` has single `Value`/`Unit` fields. **Extend `DurationLiteral` to carry multiple components** (e.g., `Components []DurationComponent` where each has Value+Unit), or introduce a `CompoundDurationLiteral` node. Apply components sequentially in descending unit order (year → month → week → day → hour → minute → second). `HasTime: true` if any sub-day component is present
- Preserve `durationToDays` for the `Date - Date` path (still returns days) but mark it as legacy
- Keep `evalDateDateOperation` unchanged — date subtraction still returns Duration in days

**Execution note:** Test-first. Write calendar-correctness tests with pinned clocks before modifying operators.

**Patterns to follow:**
- Rate arithmetic widening pattern in `evalBinaryOperation()` (per institutional learnings)
- Normalization block at top of operation, test full type dispatch matrix

**Test scenarios:**
- Happy path: `Jan 31 + 1 month` = Feb 28 (non-leap) or Feb 29 (leap) — via `AddDate(0,1,0)`
- Happy path: `Apr 6 - 3 months` = Jan 6 (calendar-correct, NOT Jan 5)
- Happy path: `next Friday + 2 hours` = Friday at 2:00 AM (requires Unit 2 HasTime)
- Happy path: `today + 1 year` across leap year boundary
- Edge case: `Feb 29 2024 + 1 year` = Feb 28 2025 (Go's `AddDate` behavior)
- Edge case: `today + 0 hours` → HasTime: true (sub-day unit present, even though zero)
- Edge case: `today + 1 month and 2 hours` → compound: AddDate then Add, HasTime: true
- Error path: Duration with unknown unit → existing error handling preserved
- Integration: `$100/month * 3 months` rate calculation still works (existing test)

**Verification:**
- `3 months ago from Apr 6` = Jan 6 (not Jan 5)
- `Jan 31 + 1 month` = Feb 28/29 (not Mar 2/3)
- All existing duration arithmetic tests pass
- Sub-day arithmetic produces dates with HasTime: true

---

### Phase 2: Expressions

- [x] **Unit 4: Weekday expressions (R1, R2)**

**Goal:** Add `this/next/last <weekday>` and bare `<weekday>` expressions to lexer, parser, and evaluator.

**Requirements:** R1, R2

**Dependencies:** Unit 1 (clock injection), Unit 2 (Date type with HasTime)

**Files:**
- Modify: `spec/lexer/token.go` (new token types)
- Modify: `spec/lexer/date_keywords.go` (weekday entries in DateKeywords + RelativeDateKeywords)
- Modify: `spec/parser/primary.go` (match new tokens)
- Modify: `impl/interpreter/datetime.go` (evaluate weekday expressions)
- Modify: `spec/lexer/date_tokenizer_test.go`
- Test: `spec/parser/date_test.go`
- Test: `impl/interpreter/date_test.go`
- Create: `testdata/spec/valid/features/relative_dates.cm`

**Approach:**
- Add 4 new token types: `DATE_WEEKDAY`, `DATE_THIS_WEEKDAY`, `DATE_NEXT_WEEKDAY`, `DATE_LAST_WEEKDAY`
- Add `Monday`–`Sunday` (case-insensitive) to `DateKeywords` map → `DATE_WEEKDAY`
- Add 21 entries to `RelativeDateKeywords`: `this/next/last` × 7 weekdays → appropriate token type. Token value carries full phrase (e.g., `"next friday"`)
- Parser: add new token types to the `p.match()` call in primary.go:328. Creates `RelativeDateLiteral{Keyword: value}`
- Evaluator: parse weekday name from keyword string, compute target date using `time.Weekday()`:
  - `next <weekday>`: find soonest future occurrence (if today is that day, skip to next week)
  - `last <weekday>`: find most recent past occurrence (if today is that day, go to last week)
  - `this <weekday>`: find occurrence in current calendar week (Mon–Sun)
  - Bare `<weekday>`: alias for `this <weekday>`

**Execution note:** Test-first. Write evaluator tests with pinned clocks for every day-of-week scenario before implementing.

**Patterns to follow:**
- Existing `DateKeywords`/`RelativeDateKeywords` map pattern
- `evalRelativeDateLiteral` switch statement pattern

**Test scenarios:**
- Happy path: On Wednesday, `next Friday` = this week's Friday (2 days ahead)
- Happy path: On Saturday, `next Friday` = next week's Friday (6 days ahead)
- Happy path: On Friday, `next Friday` = next week's Friday (7 days ahead — skip today)
- Happy path: On Wednesday, `last Monday` = this week's Monday (2 days ago)
- Happy path: On Monday, `last Monday` = last week's Monday (7 days ago — skip today)
- Happy path: On Wednesday, `this Friday` = this week's Friday. `this Monday` = this week's Monday (past)
- Happy path: Bare `Friday` = same as `this Friday`
- Edge case: `next Monday` on Sunday (last day of week) = tomorrow (Monday)
- Edge case: `last Sunday` on Monday (first day of week) = yesterday
- Edge case: Year boundary — `next Monday` on Dec 31 (a Wednesday) = Jan 5, next year
- Integration: `next Friday + 2 weeks` composes with existing duration arithmetic
- Integration: `3 days from next Friday` composes with existing `from` syntax

**Verification:**
- All 7 weekdays work with all 3 modifiers + bare form
- Narrow-window semantics verified for every day-of-week as reference
- Composition with duration arithmetic works

---

- [x] **Unit 5: Period evaluation + relative month expressions (R3, R4, R8)**

**Goal:** Implement evaluation for `this/next/last week/month/year` (unstub), and add `this/next/last <month>` and `next <month> <day>` expressions.

**Requirements:** R3, R4, R8

**Dependencies:** Unit 1 (clock injection), Unit 4 (weekday pattern established)

**Files:**
- Modify: `spec/lexer/token.go` (new month-relative token types)
- Modify: `spec/lexer/date_keywords.go` (month entries in RelativeDateKeywords)
- Modify: `spec/parser/primary.go` (match new tokens, handle `next <month> <day>` lookahead)
- Modify: `impl/interpreter/datetime.go` (evaluate period and month expressions)
- Test: `spec/parser/date_test.go`
- Test: `impl/interpreter/date_test.go`
- Create: `testdata/spec/valid/features/relative_periods.cm`

**Approach:**
- **Periods (R8):** The 9 tokens (`DATE_THIS_WEEK` etc.) already exist. Add evaluation cases to `evalRelativeDateLiteral`:
  - `this week` → Monday of current week. `next week` → Monday + 7 days. `last week` → Monday - 7 days
  - `this month` → 1st of current month. `next month` → 1st of next month via `AddDate(0,1,0)`. `last month` → 1st of previous month via `AddDate(0,-1,0)`
  - `this year` → Jan 1. `next year` → Jan 1 + 1 year. `last year` → Jan 1 - 1 year
- **Months (R3):** Add 3 new token types: `DATE_THIS_MONTH_NAME`, `DATE_NEXT_MONTH_NAME`, `DATE_LAST_MONTH_NAME`. Add 36 entries to `RelativeDateKeywords` (this/next/last × 12 months + abbreviations). Evaluate: `next April` = April 1 of next occurrence strictly after current month. `this April` in April = April 1 this year. `last April` = April 1 of most recent past occurrence
- **Month+Day (R4):** After parsing `DATE_NEXT_MONTH_NAME`, lookahead for a number literal. If present, use that as the day. `next April 25` → April 25 of next occurrence. `next February 29` → forward-scan to next leap year: use `time.Date(year, 2, 29, ...)`, check if result month is still February (Go normalizes invalid dates). **No year component in relative month+day expressions** — `next April 25 2028` is NOT valid. If the user wants a specific year, they should use the existing date literal syntax `April 25 2028`. The relative modifier (`next/last/this`) implies forward/backward scanning, which is incompatible with a fixed year

**Execution note:** Test-first. The 9 `TestExtendedRelativeDates` cases were unskipped in Unit 1 (currently failing). Implement evaluation to make them pass, then add month-relative tests.

**Patterns to follow:**
- Period evaluation follows the existing `evalRelativeDateLiteral` switch pattern
- Month parsing follows `readDateLiteral` for month name recognition

**Test scenarios:**
- Happy path: `this week` on Wednesday Apr 8 = Monday Apr 6
- Happy path: `next month` on Jan 31 = Feb 1 (not Feb 28)
- Happy path: `last year` on Mar 15 2026 = Jan 1 2025
- Happy path: `next April` on Mar 15 = April 1 this year. `next April` on April 15 = April 1 next year
- Happy path: `this April` on April 15 = April 1 this year
- Happy path: `last April` on March 15 = April 1 of previous year
- Happy path: `next April 25` on March 1 = April 25 this year
- Edge case: `next February 29` on March 1, 2025 (non-leap) = Feb 29, 2028 (next leap year)
- Edge case: `this week` on Monday = same Monday. `this week` on Sunday = Monday of that week (start of week)
- Edge case: `next month` on Dec 15 = Jan 1 of next year (year boundary)
- Integration: `next April + 10 days` = April 11
- Integration: `2 weeks from next month` composes with from syntax

**Verification:**
- All 9 period expression tests pass (previously skipped)
- Month expressions resolve to correct dates across year boundaries
- `next February 29` correctly skips non-leap years

---

- [x] **Unit 6: Duration-relative expressions — AGO and extended FROM (R5, R6)**

**Goal:** Add `<duration> ago` syntax and extend `from` to accept all new relative date expressions as targets.

**Requirements:** R5, R6

**Dependencies:** Unit 1 (clock injection), Unit 3 (unit-aware arithmetic), Unit 4 & 5 (relative dates as from-targets)

**Files:**
- Modify: `spec/lexer/date_keywords.go` (add `ago` to DateKeywords)
- Modify: `spec/lexer/token.go` (add `AGO` token type)
- Modify: `spec/parser/primary.go` (check for AGO after DURATION_LITERAL; extend `parseFromTarget`)
- Modify: `impl/interpreter/datetime.go` or `operators.go` (AGO evaluation)
- Test: `spec/parser/date_test.go`
- Test: `impl/interpreter/date_test.go`
- Create: `testdata/spec/valid/features/ago_expressions.cm`

**Approach:**
- **AGO (R5):** Add `"ago"` → `AGO` token to `DateKeywords`. In parser, after matching `DURATION_LITERAL`, check for `AGO` token (similar to `FROM` check). Desugar `2 weeks ago` into `BinaryOp("-", now, DurationLiteral(2, "week"))` where `now` is a **synthetic** `RelativeDateLiteral("now")` created by the parser (not from a token). The evaluator already handles `"now"` in its switch — this is parser-internal desugaring, not a new token type
- **Extended FROM (R6):** Expand `parseFromTarget()` to accept all new relative date token types from Units 4–5 (DATE_WEEKDAY, DATE_THIS_WEEKDAY, DATE_NEXT_WEEKDAY, DATE_LAST_WEEKDAY, DATE_THIS_MONTH_NAME, DATE_NEXT_MONTH_NAME, DATE_LAST_MONTH_NAME, and the 9 existing DATE_THIS/NEXT/LAST_WEEK/MONTH/YEAR) in addition to today/tomorrow/yesterday/date-literal. Quarter tokens from Unit 7 can be added to `parseFromTarget` when Unit 7 lands — Unit 6 does not depend on Unit 7
- Register `ago` in `isNaturalSyntaxKeyword()` to prevent it being consumed as a unit name

**Patterns to follow:**
- The existing `FROM` check at primary.go:358-375
- The existing `parseFromTarget` function at primary.go:584-624

**Test scenarios:**
- Happy path: `2 weeks ago` from pinned date Apr 6 = Mar 23
- Happy path: `10 minutes ago` from pinned time 14:30 = 14:20 (HasTime: true)
- Happy path: `3 months ago` from Apr 6 = Jan 6 (calendar-correct via unit-aware arithmetic)
- Happy path: `3 days from next Friday` — FROM with new relative date target
- Happy path: `1 month from this quarter` — FROM composes with quarter expressions (after Unit 7)
- Edge case: `1 year ago` from Mar 1 2025 (non-leap) when origin was leap year Feb 29 scenario — depends on pinned date
- Error path: `ago` without preceding duration → parse error
- Integration: `2 weeks ago + 3 days` — ago result composes with further arithmetic

**Verification:**
- `ago` produces correct past dates with unit-aware arithmetic
- `from` accepts all new relative date expression types
- `ago` registered in natural syntax keyword guard

---

- [x] **Unit 7: Calendar quarters, fiscal system, FY/CY notation (R9–R13)**

**Goal:** Add calendar quarter expressions, fiscal year frontmatter, fiscal quarter/year expressions, and Q/FQ/FY/CY notation.

**Requirements:** R9, R10, R11, R12, R13

**Dependencies:** Unit 1 (clock injection), Unit 5 (month expressions pattern)

**Files:**
- Modify: `spec/lexer/token.go` (quarter, fiscal, notation tokens)
- Modify: `spec/lexer/date_keywords.go` (quarter entries in RelativeDateKeywords)
- Modify: `spec/lexer/date_tokenizer.go` (add `peekThreeWords`, add notation tokenizer for Q/FQ/FY/CY)
- Modify: `spec/parser/primary.go` (match new tokens, new AST nodes)
- Modify: `spec/ast/nodes.go` (extend `RelativeDateLiteral` with optional `Value` field for notation like Q1/FQ1/FY26/CY2001, or add dedicated node types if the value semantics diverge enough)
- Modify: `spec/document/frontmatter.go` (add `fiscal_year_starts` key)
- Modify: `impl/interpreter/datetime.go` (evaluate quarter and fiscal expressions)
- Test: `spec/lexer/date_tokenizer_test.go`
- Test: `spec/parser/date_test.go`
- Test: `impl/interpreter/date_test.go`
- Test: `spec/document/frontmatter_test.go`
- Create: `testdata/spec/valid/features/quarters.cm`
- Create: `testdata/spec/valid/features/fiscal.cm`

**Approach:**
- **Calendar quarters (R9):** Add `this/next/last quarter` to `RelativeDateKeywords` with 3 new tokens. Evaluate: `this quarter` → first day of current Q (Jan/Apr/Jul/Oct). Quarter = `(month - 1) / 3`
- **Q1–Q4 notation (R12):** New tokenizer function `tryReadQuarterNotation()`: recognizes `Q` followed by `1`-`4`. Produces `CALENDAR_QUARTER_LITERAL` token with value `"1"`–`"4"`. Resolves to first day of that quarter in current year
- **Fiscal frontmatter (R10):** Add `FiscalYearStarts *time.Month` to `Frontmatter` struct. Parse in `ParseFrontmatter` following `parseMeasurementConfig` pattern. Validate: must be a valid month name. **Thread to evaluator:** add `FiscalYearStarts *time.Month` field to the `Interpreter` struct (same pattern as `measurement` config). Set during document initialization alongside measurement config. Fiscal expression evaluation reads this field directly
- **Fiscal quarters (R12):** `FQ1`–`FQ4` tokenized as `FISCAL_QUARTER_LITERAL`. Evaluation reads `fiscal_year_starts` from document frontmatter. FQ1 starts at configured month. Error diagnostic (R11) if frontmatter not set
- **FY notation (R12):** `FY26`/`FY2026` tokenized as `FISCAL_YEAR_LITERAL`. Resolves to first day of fiscal year (e.g., FY27 with July start = July 1 2026)
- **CY notation (R12):** `CY2001`/`CY01` tokenized as `CALENDAR_YEAR_LITERAL`. Resolves to Jan 1 of that year. Two-digit years: `CY01` = 2001 (21st century default)
- **Three-word tokenization:** Add `peekThreeWords()` to date_tokenizer.go. Add 6 entries to a new `ThreeWordDateKeywords` map: `this/next/last fiscal quarter/year`
- **`this fiscal quarter` evaluation (R13):** same pattern as calendar quarter but offset by fiscal start month

**Execution note:** Test-first. Write fiscal diagnostic tests (missing frontmatter → error) before implementing the happy path.

**Patterns to follow:**
- `readDateLiteral` for prefix+number tokenization
- `parseMeasurementConfig` for frontmatter parsing
- `evalRelativeDateLiteral` switch for quarter evaluation

**Test scenarios:**
- Happy path: `this quarter` on Feb 15 = Jan 1. On Apr 1 = Apr 1. On Jul 15 = Jul 1. On Oct 31 = Oct 1
- Happy path: `next quarter` on Nov 15 = Jan 1 next year (year boundary)
- Happy path: `Q1` = Jan 1 current year. `Q4` = Oct 1 current year
- Happy path: `FQ1` with `fiscal_year_starts: july` = Jul 1 of current fiscal year
- Happy path: `FY27` with `fiscal_year_starts: july` = Jul 1, 2026
- Happy path: `CY2001` = Jan 1, 2001. `CY01` = Jan 1, 2001
- Happy path: `this fiscal quarter` on Aug 15 with July start = Jul 1 (FQ1)
- Happy path: `next fiscal quarter` on Aug 15 with July start = Oct 1 (FQ2)
- Edge case: `FQ1` without fiscal frontmatter → diagnostic error "fiscal expressions require a 'fiscal_year_starts' frontmatter key"
- Edge case: `fiscal_year_starts: january` → fiscal = calendar (FQ1 = Q1)
- Edge case: Invalid frontmatter `fiscal_year_starts: banana` → parse error
- Integration: `FQ1 + 30 days` = Jul 31 (with July start) — composes with arithmetic
- Integration: `Q1 - Q4` = Duration (date subtraction)

**Verification:**
- All 4 calendar quarters resolve correctly for every month
- Fiscal expressions work with configurable start month
- Missing frontmatter produces clear diagnostic
- FY/CY notation resolves correctly
- `peekThreeWords` works without breaking existing two-word tokenization

---

### Phase 3: Formatting

- [ ] **Unit 8: Smart datetime display and locale time formatting (R20–R24)**

**Goal:** Implement smart display (date-only when midnight, date+time when non-midnight), locale-aware 12h/24h formatting, and cross-locale tests.

**Requirements:** R20, R21, R22, R23, R24

**Dependencies:** Unit 2 (HasTime flag), Unit 3 (sub-day arithmetic produces HasTime dates). Blocked on or merges with locale-aware date display work (brainstorm `2026-04-05`)

**Files:**
- Modify: `format/display/date_locale.go` (add time format per locale, datetime layout strings)
- Modify: `format/display/formatter.go` (add `FormatDateTime` or extend `FormatDate`)
- Modify: `format/display/config.go` (add `TimeFormat` override field)
- Modify: `format/display/display.go` (route Date with HasTime through datetime formatting)
- Test: `format/display/date_locale_test.go`
- Create: `format/display/datetime_format_test.go`

**Approach:**
- Add `timeFormats` map parallel to `dateFormats`: locale → 12h/24h layout string and separator
- Add `TimeFormat string` to `DisplayConfig` (values: `""` for locale default, `"12h"`, `"24h"`)
- Extend `FormatDate` (or add `FormatDateTime`): check `HasTime` on the Date value. If `HasTime` and time is non-midnight → format with date + time layout. If `HasTime` and time is midnight → date + time (explicitly set). If not `HasTime` → date only
- Display precision: seconds. Format strings use `15:04:05` (24h) or `3:04:05 PM` (12h), omitting seconds when zero
- Locale defaults: US/UK → 12h, most European/CJK → 24h. Research exact mapping during implementation

**Patterns to follow:**
- Existing `dateFormats` map and `getDateFormat` function
- `monday.Format` for locale-aware output
- `IsExplicit` flag pattern for display decisions

**Test scenarios:**
- Happy path: Date with `HasTime: false` → date only ("Wed, Apr 10, 2026")
- Happy path: Date with `HasTime: true`, time 14:30 → date + time ("Wed, Apr 10, 2026 2:30 PM" in en_US)
- Happy path: Same datetime in de_DE → "Mi, 10. Apr. 2026 14:30"
- Happy path: `time_format: 24h` override in en_US → "Wed, Apr 10, 2026 14:30"
- Edge case: `HasTime: true` at midnight → shows time ("12:00 AM" / "00:00")
- Cross-locale: ja_JP date with time → correct Japanese date format + 24h time
- Cross-locale: zh_CN date without time → Chinese date format only
- Cross-locale: ko_KR with time → Korean format + 24h
- Cross-locale: All 18+ locales in `dateFormats` table produce valid output for both date-only and datetime
- Cross-locale: Locale set + `time_format` overridden → override takes precedence

**Verification:**
- Smart omission works: midnight = date-only, non-midnight = date+time
- All 18+ locales produce valid formatted output
- `time_format` override works
- Second-precision display (no sub-second shown)

---

### Phase 4: Surface Integration

- [x] **Unit 9: 12-layer integration — classifier, feature registry, autosuggest, LSP**

**Goal:** Ensure all new date expressions work across every CalcMark surface: classifier, document detector, feature registry, TUI autosuggest, LSP completions, interpolation, scale exemption.

**Requirements:** R25 (comprehensive test coverage), institutional learning (12-layer checklist)

**Dependencies:** Units 4–7 (all expression types exist)

**Files:**
- Modify: `spec/classifier/classifier.go` (add `containsDateKeywords()`)
- Modify: `spec/features/registry.go` (register new date features)
- Modify: `cmd/calcmark/tui/editor/autocomplete.go` (add date suggestions)
- Modify: `lsp/completion.go` (add date completion items)
- Modify: `spec/document/detector.go` (add new token types to `isDateToken`)
- Modify: `spec/ast/nodes.go` (`ContainsScaleRef` default case for new nodes)
- Modify: `spec/lexer/token.go` (`IsReservedKeywordToken` for new tokens)
- Test: `spec/classifier/classifier_test.go`
- Test: `spec/features/registry_test.go`
- Test: `cmd/calcmark/tui/editor/autocomplete_test.go`
- Test: `lsp/completion_test.go`

**Approach:**
- **Classifier:** Add `containsDateKeywords()` function checking for all DATE_* tokens. Call it in `ClassifyLine()` before the directive check. Lines with date tokens should classify as calculations
- **Feature registry:** Add new features to `getDateFeatures()`: `next_weekday`, `last_weekday`, `this_weekday`, `this_quarter`, `fiscal_quarter`, `ago`, `calendar_year`, `fiscal_year`. Include `Syntax`, `Description`, `Example`, `NLExample` fields
- **TUI autosuggest:** Add date feature suggestions to `GetSuggestions()` merge block. Surface keywords like "next Friday", "this quarter", "FQ1" with descriptions
- **LSP completions:** Add `dateCompletionItems()` function modeled on `unitCompletionItems()`. Convert feature suggestions to `protocol.CompletionItem`
- **Document detector:** Add all new token types to `isDateToken()`. Currently handles 15 types; add the new ~20 tokens
- **Reserved keywords:** Add all new date tokens to `IsReservedKeywordToken()` and `isNaturalSyntaxKeyword()` where appropriate
- **Scale exemption:** Add new AST node types to `ContainsScaleRef` default case

**Execution note:** Follow the 12-layer checklist systematically. Test each layer independently.

**Patterns to follow:**
- `containsOperators()` pattern for classifier
- `unitCompletionItems()` pattern for LSP
- `getDateFeatures()` pattern for registry
- `isDateToken()` pattern for detector

**Test scenarios:**
- Classifier: `next Friday + 2 weeks` classifies as calculation (not markdown)
- Classifier: `this quarter` classifies as calculation
- Classifier: `FQ1` classifies as calculation
- Feature registry: `TestEveryBuiltinFunctionHasFeature` passes (consistency check)
- Autosuggest: Typing "next" suggests "next Friday", "next week", "next quarter", etc.
- Autosuggest: Typing "FQ" suggests "FQ1", "FQ2", "FQ3", "FQ4"
- LSP: Completion for "this" includes "this week", "this month", "this quarter", "this fiscal quarter"
- Detector: Document containing only `next Friday = today + 2 days` detected as calcmark
- Reserved keywords: `next = 5` produces reserved keyword warning
- Integration: All new expressions work in TUI side-by-side preview (FormatDate path)
- Integration: All new expressions work in HTML output (Formatter path)
- Integration: Interpolation `{{next_friday}}` resolves correctly when variable holds a date

**Verification:**
- Full 12-layer checklist validated
- No surface layer silently drops date expressions
- All existing surface tests still pass

---

### Phase 5: Edge Cases and Golden Tests

- [x] **Unit 10: Calendar correctness edge cases and golden test files**

**Goal:** Comprehensive edge case testing for calendar correctness, composition, and golden test files for regression.

**Requirements:** R14, R15, R25

**Dependencies:** All previous units

**Files:**
- Create: `testdata/spec/valid/features/relative_dates.cm` (if not created in Unit 4)
- Create: `testdata/spec/valid/features/fiscal_dates.cm`
- Create: `testdata/eval/success/features/relative_dates.cm`
- Create: `testdata/eval/success/features/fiscal_dates.cm`
- Create: `testdata/spec/invalid/features/fiscal_missing_config.cm`
- Modify: `impl/interpreter/date_test.go` (edge case tests)

**Approach:**
- Write golden test files (.cm) that exercise every expression form and serve as regression tests
- Write focused edge case unit tests with pinned clocks for calendar boundary conditions
- Validate against known corporate fiscal calendars (Microsoft FY starts July, Australian FY starts July, UK FY starts April)

**Test scenarios:**
- Edge case: Feb 29 + 1 year = Feb 28 (leap → non-leap)
- Edge case: Jan 31 + 1 month = Feb 28/29 (month clipping)
- Edge case: `next February 29` from 2025 (non-leap) → Feb 29, 2028
- Edge case: `next Friday + 2 weeks - 3 days` — multi-step composition
- Edge case: `10 minutes ago + 30 minutes` — sub-day composition
- Edge case: `last year - 6 months` — negative composition
- Edge case: Dec 31 boundary — `next week`, `next month`, `next year` all cross year
- Edge case: `FY27` with July start → Jul 1, 2026 (fiscal year starts BEFORE calendar year)
- Edge case: `next fiscal quarter` on Jun 15 with July start → Jul 1 (FQ1 of next FY)
- Edge case: `Q4 - Q1` = Duration in days (quarter subtraction)
- Edge case: Large durations: `100 years ago` — should not overflow
- Golden test: Valid `.cm` file with all expression forms, expected output verified
- Golden test: Invalid `.cm` file with fiscal expressions but no frontmatter → diagnostic errors

**Verification:**
- All edge cases pass with calendar-correct results
- Golden test files serve as regression suite
- `task test` passes with full test suite

## System-Wide Impact

- **Interaction graph:** New tokens flow through lexer → parser → AST → classifier → evaluator → formatter → TUI/LSP/HTML. Frontmatter threading adds fiscal config to evaluator context. All 12 layers from the institutional checklist are touched.
- **Error propagation:** Missing fiscal frontmatter produces diagnostics at evaluation time (not parse time). Invalid frontmatter produces diagnostics at parse time. Calendar errors (invalid dates) are caught by Go's `time` package normalization checks.
- **State lifecycle risks:** `TimeFunc` on Interpreter must be set before any evaluation. Default to `time.Now` in constructor. Tests must not share Interpreter instances across parallel subtests with different clocks.
- **API surface parity:** All new expressions must work identically in: TUI editor, `cm` CLI pipe mode, LSP-connected editor, HTML output, JSON output. The Formatter is the single display path — no bypasses.
- **Integration coverage:** Unit tests alone will not prove: (a) TUI autosuggest showing date keywords, (b) LSP completions including fiscal expressions, (c) classifier correctly routing date-heavy lines. Each needs its own layer-specific test.
- **Unchanged invariants:** Existing `today`, `tomorrow`, `yesterday` behavior unchanged. Existing date literal parsing (`Dec 25 2025`) unchanged. Existing duration arithmetic unchanged except for calendar-correctness improvement (months/years). `Time` type (standalone `3:00 PM`) remains separate. Bare integers remain integers.

## Risks & Dependencies

| Risk | Mitigation |
|------|------------|
| Token type explosion (21 weekday + 36 month + 6 fiscal + notation tokens) | Token types are cheap in Go. The alternative (fewer tokens with parser disambiguation) adds parser complexity. Bounded set — will not grow |
| `peekThreeWords` breaking existing two-word matching | Three-word check runs AFTER two-word check fails. Existing behavior unaffected. Regression tests cover all existing phrases |
| Unit-aware arithmetic changing existing behavior | `3 months` currently = 90 days; after fix = calendar months. This is a **correctness improvement**, not a regression. Document in changelog. Existing tests updated to expect correct behavior |
| Locale time format research incomplete | Phase 3 (Unit 8) is blocked on locale display work already in progress. **Phases 1–2 and 4–5 can ship independently** as a first PR. Phase 3 ships as a follow-up when locale display lands or merges with it |
| Fiscal year config complexity | Fiscal is cleanly scoped to frontmatter + 6 tokens + evaluator switch. No global state. No runtime config changes |

## Future Considerations

- **Localized input names** (e.g., `Montag`, `Lundi`, `Vendredi`): The `monday` library has locale-specific weekday/month name tables that could be used to build reverse-lookup keyword maps. Architecture supports this — the keyword maps are data-driven and the locale is available via frontmatter. Follow-up feature: populate `DateKeywords` and `RelativeDateKeywords` with locale-specific entries based on the document's `locale` setting. This is consistent with CalcMark's existing pattern: locale affects *display* (numbers, dates), and this would extend it to *input*. English names would always work regardless of locale.

## Documentation / Operational Notes

- Update `site/content/` documentation for all new date expressions
- Add worked examples to the documentation site showing financial modeling with fiscal quarters
- Update `cm help` output via feature registry additions
- Changelog entry noting the calendar-correctness improvement for month/year arithmetic (behavior change)
- Changelog entry noting 7 new reserved words (Monday–Sunday) that can no longer be used as variable names

## Sources & References

- **Origin document:** [docs/brainstorms/2026-04-06-relative-date-math-requirements.md](docs/brainstorms/2026-04-06-relative-date-math-requirements.md)
- **Related issue:** #119
- **Related PR:** #121
- **Institutional learnings:**
  - `docs/solutions/language-features/directive-as-value-cross-layer-learnings.md` (12-layer checklist)
  - `docs/solutions/logic-errors/date-keywords-missing-from-reserved-keyword-diagnostics.md` (keyword registration)
  - `docs/solutions/feature-gaps/rate-type-arithmetic-widening.md` (operator type dispatch)
  - `docs/solutions/ui-bugs/locale-formatting-bypass-in-tui.md` (display formatting)
  - `docs/solutions/test-failures/user-config-leaks-into-tests.md` (test isolation)
  - `docs/solutions/code-organization/unified-feature-registry-three-to-one.md` (feature registry)
- **Go time package:** calendar arithmetic via `AddDate()` and `Add()`
- **Related brainstorm:** `docs/brainstorms/2026-04-05-locale-aware-date-display-brainstorm.md` (locale display dependency)
