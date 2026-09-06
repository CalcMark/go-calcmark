# Changelog

All notable changes to `go-calcmark` are documented in this file. The format
follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and the
project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

The v1.x history lives in the git log (`git log v1.0.0..v1.14.0`) and was
not previously captured here. This file starts at the v2.0 cycle and will
track every release going forward.

## [Unreleased]

### Added

- **Named tables and arrays** (#118). A markdown table preceded by
  `<!-- table: name (col1, col2, …) -->` registers a table whose columns
  are arrays. Calc blocks read columns with dot access (`rates.rate`),
  combine them element-wise with scalar broadcasting
  (`rates.rate * rates.hc`, `rates.rate * 1.1`), and reduce them with
  `sum`, `avg`, and the new `min`, `max`, and `count` builtins.
  `{{array_var}}` inside a markdown table row substitutes that row's
  element; `{{table.column}}` and arrays in prose render as `[a, b, c]`.
  JSON output decomposes array results as `type: "array"` with an
  `elements` list. Text cells (`Senior`) are display-only `text` values:
  they can be counted and interpolated but not used in arithmetic. The
  LSP completes column names after `table.`. Diagnostics for a malformed
  directive, a column-count mismatch, a mixed-type column (positioned at
  the cell), a missing table, a duplicate or colliding name, and an
  array whose length does not match the rows it is interpolated into.
- `sum(x)` with a single argument now parses (it may be an array); the
  scalar form's 2-argument minimum is enforced at evaluation time.

### Changed

- **`Convert(Mode: Embedded)` evaluates the host document as one unit.**
  Each ```` ```cm ```` fence used to be evaluated in isolation, so a
  variable from one fence was undefined in the next and `{{var}}` in
  prose never resolved. Rendering now goes through `NewDocumentEmbedded`
  (the API cmw already uses), which is also what lets a named table in
  the markdown feed the fences below it.

### Fixed

- **Blocks with semantic errors still evaluate their clean statements**
  (#113). The evaluator used to stop at the first checker error, so a
  block containing one redefinition showed no values at all and never
  discovered runtime errors on the other lines. Checker-rejected
  statements are now skipped (their variable is marked errored so later
  references cascade) while every other statement runs. JSON per-result
  `error` is now the diagnostic reported against that statement — a
  clean line no longer inherits a neighbor's redefinition message.
- **Runtime diagnostics now carry a column span** (#164). Argument-type
  rejections from `grow`, `compound`, `depreciate`, and `capacity` point
  at the offending argument; operator errors (currency ÷ currency,
  division by zero) point at the failing sub-expression; cascading
  errors point at the errored reference. `Diagnostic.Column` /
  `EndColumn` are populated for `eval_error` diagnostics, so LSP and
  editor consumers can underline the exact token the way they already
  do for semantic-checker diagnostics.
- **Lexer token columns no longer drift after numbers.** Lookaheads that
  rewound `pos` but not `column` shifted every later token on the line
  by the number's width; line/column are now derived from position.
- **`FY<year> FQ<n>` now honors `calendar_year_offset`** (#162). The
  explicit-year quarter form built its fiscal year from the typed label
  directly, so under the default end-year labeling `FY2027 FQ1` started a
  year after `FY2027` did. Both forms now resolve the label through the
  same convention, so `start of FY2027` and `FY2027 FQ1` agree.


### Added

- **`Rate × Duration`, `Rate × Quantity`, and `Rate × Rate` arithmetic
  with same-dimension unit cancellation across every category.** A
  long-standing gap closed: rates can now be multiplied by durations,
  cancellable quantities, and other rates, and the dimension cancels
  rather than erroring or silently coercing. The user-facing
  principle: *CalcMark works when a normal human reading the
  expression aloud would expect it to work — and refuses with a
  clear message when they wouldn't.* Concretely:
  - `cost = $100/hour; work = 3 weeks; fee = cost * work` → `$50,400`
    (the duration is auto-converted to the rate's `PerUnit`).
  - `$100/hour * 40 hours/week * 3 weeks` → `$12,000` (chained
    cancellation through Rate × Rate).
  - `100 MB/s * 10 seconds` → `1000 MB` (data-rate × time → data).
  - `60 mph * 2 hours` → `120 miles` (Speed-quantity widens to its
    underlying rate, then the cancellation engine handles it).
  - Rate literals can carry a Duration numerator: `40 hours / week`.
  - Currency identity is preserved through the pipeline: `100 USD/hour
    × 3 weeks → 50,400 USD` (Code preserved); `€50/hour × 8 hours →
    €400` (Symbol/Code both populated from `NewCurrency`).

### Changed

- **Retired the long-standing too-permissive `Rate × Quantity →
  Quantity` rule** (`100/second * 10 KB → 1000 KB`, ignoring the
  rate's `PerUnit`). It silently produced dimensionally-wrong results.
  Replaced with same-category cancellation: `100 MB/s * 10 seconds →
  1000 MB` cancels cleanly, but `100/second * 10 KB` (cross-category)
  now errors with a friendly message naming both operands and
  explaining the non-cancellation. Documents that exercised the old
  rule will see new errors and need updating; the [migration note in
  rates.cm](testdata/eval/success/features/rates.cm) shows the
  pattern.

### Internal

- New helpers in `impl/interpreter/rate_arithmetic.go`:
  `categoryOf(unit)`, `convertWithinCategory(value, from, to)`,
  `tryRateCancellation(rate, value, unit)`, `cancellable(a, b)`,
  `coerceSpeedQuantityToRate(qty)`, `rateMismatchError(left, right)`,
  `categoryPhrase(unit, category)`. Foundation generalised from
  PR #148's time-only first slice.

## [2.0.0] — 2026-04-30

Promoted from 2.0.0-rc.1; same content. Tagged stable after the soak
window closed without surfacing issues. See the 2.0.0-rc.1 section
below for full release notes, breaking changes, and migration guidance.

## [2.0.0-rc.1] — 2026-04-30

The 2.0 cycle promotes calendar/fiscal periods to a first-class type and
makes period-bearing language (Q1, FQ1, `this fiscal year`, `between A and
B`, `length of Q3`) behave consistently across the parser, interpreter,
formatters, and editor surfaces. The work spans 26 commits since v1.14.0.

### Breaking

- **Module path is now `github.com/CalcMark/go-calcmark/v2`.** Per Go's
  [module versioning rules](https://go.dev/ref/mod#major-version-suffixes),
  any major version ≥ 2 must encode the major version in the module
  path. Go consumers update their imports:
  ```diff
  - import "github.com/CalcMark/go-calcmark"
  - import "github.com/CalcMark/go-calcmark/spec/types"
  + import "github.com/CalcMark/go-calcmark/v2"
  + import "github.com/CalcMark/go-calcmark/v2/spec/types"
  ```
  And bump the `require` line: `github.com/CalcMark/go-calcmark/v2 v2.0.0`.
  Mechanical: every existing import gets a `/v2` segment after
  `go-calcmark`. Non-Go consumers (the CLI binary, the lark playground,
  the calcmark-web embedded server) are unaffected — only direct Go
  module consumers need to change anything.
- **Period-bearing keywords now evaluate to `*types.Period`, not `Date`.**
  `Q1`, `FQ2`, `CY2026`, `FY2027`, `this month`, `this fiscal quarter`, bare
  month names like `April`, etc. now produce a `Period` value instead of
  the period's start `Date`. Code that consumed the result as a `Date` must
  read `.Start` (the start day) or use the explicit `start of <period>`
  operator. The `End` field is now also populated for every period kind.
- **`Period + Duration` arithmetic extends from the period's END, not its
  start.** Pre-v2: `Q1 + 30 days` = Jan 1 + 30 = Jan 31. v2: `Q1 + 30 days`
  = Mar 31 + 30 = Apr 30. Documents that want the old "from the start"
  behavior now write `start of Q1 + 30 days` explicitly. The asymmetric
  rule matches how people describe ranges in conversation ("a week after
  Q1 ends") and removes the silent ambiguity of pre-v2 arithmetic.
- **`grow` / `compound` / `depreciate` 3-arg form: the periods argument
  must be a plain Number.** v1 silently coerced a Duration: `compound
  $1000 by 5% over 3 years` interpreted `3 years` as 3 iterations. v2
  rejects the Duration with a clear diagnostic and asks for the count
  without a unit (`over 3`). The 4-arg form with `compounded monthly`
  still accepts a Duration as the time argument because that form has
  separate iteration semantics. Fixes the only remaining unit→count
  coercion in the language.
- **`between` is now a reserved keyword.** Documents that previously used
  `between` as an identifier (variable name, function-call positional)
  must rename it.
- **Spaces are no longer permitted in identifiers.** Identifiers like
  `tax rate` must use `tax_rate`. (This change shipped progressively
  across the v1 series; v2.0 removes the last remaining quirks.)

### Added

- **First-class `Period` type** (`spec/types/period.go`). Concrete kinds:
  `PeriodCalendarQuarter`, `PeriodCalendarYear`, `PeriodCalendarMonth`,
  `PeriodFiscalQuarter`, `PeriodFiscalYear`, `PeriodRelativeQuarter`,
  `PeriodRelativeFiscalQuarter`, `PeriodRelativeYear`,
  `PeriodRelativeFiscalYear`, `PeriodCustom`. Every Period carries both
  `Start` and `End` (closed interval, day precision).
- **First-class `end of <period>` and `start of <period>` operators.**
  `end of Q1` = Mar 31; `start of FQ2` = the first day of fiscal quarter 2.
  Compose with every period kind.
- **Custom date ranges** via `between A and B` and the equivalent
  `from A to B`. Produces a `PeriodCustom` spanning the two dates
  inclusive. Backed by `ast.BetweenExpr` and `types.NewCustomPeriod`.
- **Period→Duration conversion** via `length of <period>` and
  `days in <period>`. `length of Q1` returns the Duration covering the
  quarter; `days in Q1` returns the day count as a Number.
- **Period basis conversion** via `<period> as fiscal` and
  `<period> as calendar`. Year-grain conversions match by label
  (`CY2026 as fiscal` → `FY2026`); quarter-grain conversions preserve the
  underlying date range and pick the FQ whose dates contain the input
  midpoint.
- **Bare month names parse as period literals.** `April` (with no year)
  resolves to April of the current year as a `PeriodCalendarMonth`.
  `April 2027` resolves to April 2027.
- **Directed notation** for relative periods: `next FQ1` / `last CY26` /
  `this Q1`. Combines a direction prefix with year-bearing notation.
- **Bare `fiscal quarter` / `fiscal year` aliases for `this`.** Documents
  can write `next fiscal quarter` and `last fiscal year` without the
  redundant `this` prefix.
- **`calendar_year_offset` frontmatter key** (`before` / `after`) selects
  whether an FY label refers to the calendar year the FY ends in
  (default — Australian government year, US tax year, most publicly
  traded companies) or the calendar year the FY starts in (some
  companies). Has no effect when `fiscal_year_starts: january`. Backed by
  `types.FYLabelMode`, `NewFiscalYearWithMode`, `NewFiscalQuarterWithMode`,
  and `Interpreter.SetFiscalCalendar`.
- **Configurable date format** via `cm config date_format` and
  `cm config period_date_format`. Locale-aware DSL (`MON dd, YYYY`,
  `dd-MON-YYYY`, etc.) backed by the `goodsign/monday` library so weekday
  and month names render in the user's locale.
- **LSP completion + documentation for period vocabulary.** Period
  literals (`Q1`, `FY2027`), the `end of` / `start of` operators, the
  `between` / `from-to` / `length of` / `days in` keywords, and the
  `as fiscal/calendar` conversion all surface in completion popups with
  hover documentation. New `keywordPhrases` LSP request lets editors
  highlight multi-word keyword phrases as a single chip.
- **Documentation:** new sections in
  [`docs/user-guide/dates.md`](site/content/docs/user-guide/dates.md)
  (Fiscal Periods → `calendar_year_offset`) and
  [`docs/user-guide/frontmatter.md`](site/content/docs/user-guide/frontmatter.md)
  (Fiscal Calendar) covering both keys with worked examples.

### Changed

- **Period display is locale-aware and compact.** A Period now renders as
  a `dd-MON-YYYY → dd-MON-YYYY` range (single day collapses to one
  date) using the configured date format and the user's locale, instead
  of the kind-bare label that pre-v2 relative kinds produced.
- **Period stores `End` natively.** Pre-v2 code that recomputed the end
  from `Start` + kind-specific math can now read `period.End` directly.

### Fixed

- `NewFiscalQuarter` correctly labels the FY by the year it ends in (the
  default labeling). Previously the label drifted by one in some Jan-start
  configurations.
- Period TUI output renders the date range for relative kinds (was: bare
  label for `this fiscal quarter` etc.).
- Doc examples (`testdata/examples/datacenter-cost.cm`,
  `embedded-datacenter-cost.md`, `investment-growth.cm`) updated to the
  v2 plain-Number iteration count for `grow` / `compound` / `depreciate`.

### Internal

- `spec/features/registry.go` now catalogs the v2.0 period operators
  (`between`, `length of`, `days in`, `as fiscal/calendar`,
  `start of`/`end of`) plus the previously-missing `fiscal_year_starts`
  and the new `calendar_year_offset` frontmatter entries.
- Semantic checker arms collapsed and rewritten for the new period AST
  nodes; transitional shim removed.
- Interpreter `evalNode` rewritten so all period keywords return
  `*types.Period` directly instead of going through a date-coercion
  shim.
- Classifier recognises bare-line `end of Q1` as a calculation rather
  than prose (was: misclassified before U2.5).

### Migration notes

If your documents do any of the following, update them before bumping
to v2.0.0:

1. Use `period + duration` arithmetic and rely on extending from the
   start. Replace `Q1 + 30 days` with `start of Q1 + 30 days` to keep
   the v1 behavior, or accept the v2 end-extension semantics if that's
   what you actually meant.
2. Pass a Duration as the period count to `grow`/`compound`/`depreciate`
   in the 3-arg form. Replace `over 5 years` with `over 5`.
3. Use a variable named `between`. Rename it.
4. Have a multi-word identifier with spaces. Rename to use underscores.

If your **Go code** imports go-calcmark as a library, also:

5. Add `/v2` to every import path:
   `github.com/CalcMark/go-calcmark/...` → `github.com/CalcMark/go-calcmark/v2/...`.
   Run `gofmt -w -r '"github.com/CalcMark/go-calcmark" -> "github.com/CalcMark/go-calcmark/v2"'`
   plus a sed pass for the sub-package paths, then `go mod tidy` against
   `github.com/CalcMark/go-calcmark/v2 v2.0.0`.

Documents that don't touch any of the above need no changes — period
literals (`Q1`, `FY2027`) read identically; `start of <period>` was
always the explicit form; the new `between` / `length of` / `days in`
keywords are net-new vocabulary.

[Unreleased]: https://github.com/CalcMark/go-calcmark/compare/v2.0.0...HEAD
[2.0.0]: https://github.com/CalcMark/go-calcmark/compare/v1.14.0...v2.0.0
[2.0.0-rc.1]: https://github.com/CalcMark/go-calcmark/compare/v1.14.0...v2.0.0-rc.1
