---
title: "Dates & Time"
weight: 3
---

CalcMark has built-in support for dates, durations, relative date expressions, calendar quarters, and fiscal periods. This page covers how dates work, what creates a date, and what does not.

### Creating Dates {#creating-dates}

There are three ways to create a date in CalcMark:

**1. Date literals** require a month name (full or abbreviated), optionally with a day and year:

```calcmark
christmas = Dec 25 2025
project_start = January 15 2026
independence_day = Jul 4
```

When the year is omitted, CalcMark uses the current year. Month names are case-insensitive and accept standard abbreviations (`Jan`, `Feb`, `Mar`, `Apr`, `Jun`, `Jul`, `Aug`, `Sep`, `Sept`, `Oct`, `Nov`, `Dec`).

**2. Date keywords** produce dates relative to today:

```calcmark
right_now = today
the_day_after = tomorrow
the_day_before = yesterday
```

**3. Relative date expressions** resolve dates from context:

```calcmark
meeting = next Friday
retro = last Tuesday
start = this Monday
month_start = this month
q_start = this quarter
eom = end of this month
```

See [Relative Dates](#relative-dates) below for the full list.

### What is NOT a Date {#not-a-date}

These common patterns do **not** create dates in CalcMark:

| Expression | What CalcMark sees | Why |
|---|---|---|
| `2019` | The number 2,019 | Bare numbers are never dates |
| `Apr 1 2019 12:30PM` | Plain text (markdown) | Date + time in a single literal is not supported |
| `10:30 AM` | Plain text (markdown) | Standalone time literals are not recognized as calculations |
| `CY2019` | Error: undefined variable | CY/FY/Q notation is planned but not yet implemented |

**To express a year as a date**, use a month name: `Jan 1 2019`.

**To express a date with a time offset**, use arithmetic: `Apr 1 2019 + 12 hours + 30 minutes`. The result carries time precision internally.

### Date Arithmetic {#date-arithmetic}

Dates compose naturally with durations:

```calcmark
launch = Jan 15 2026 + 90 days
deadline = Jun 1 2026 - 2 weeks
next_review = today + 3 months
```

CalcMark uses **calendar-correct** month and year arithmetic. Adding 1 month to January 31 gives February 28 (or 29 in a leap year) -- not March 3.

```calcmark
end_of_jan = Jan 31 2026
plus_one_month = Jan 31 2026 + 1 month
```

Date subtraction produces a duration in days:

```calcmark
project_start = Jan 1 2026
project_end = Jun 30 2026
project_length = Jun 30 2026 - Jan 1 2026
```

### Duration Units {#durations}

Supported duration units: `years` (`year`, `yr`, `yrs`), `months` (`month`), `weeks` (`week`), `days` (`day`), `hours` (`hour`, `hr`), `minutes` (`minute`, `min`), `seconds` (`second`, `sec`), `milliseconds` (`millisecond`, `ms`).

Sub-day durations (hours, minutes, seconds) add time precision to dates:

```calcmark
meeting = Apr 1 2026 + 14 hours + 30 minutes
```

### The `from` and `ago` Keywords {#from-and-ago}

Two alternative syntaxes for duration-relative dates:

```calcmark
review = 7 days from Jan 1 2026
launch = 2 weeks from today
next_step = 3 days from next Friday
budget = 90 days from now

lookback = 2 weeks ago
prior_quarter = 3 months ago
```

`from` computes a future date from a base. `ago` computes a past date from the current time.

### Relative Dates {#relative-dates}

#### Weekday Expressions

```calcmark
d = next Friday
d = last Tuesday
d = this Wednesday
d = Friday
```

`next <weekday>` resolves to the **soonest future occurrence** of that weekday. If today is that weekday, it means next week's. `last <weekday>` resolves to the most recent past occurrence. `this <weekday>` resolves to that weekday in the current calendar week (Monday through Sunday). A bare weekday name (`Friday`) is shorthand for `this Friday`.

#### Period Expressions

```calcmark
d = this week
d = next month
d = last year
```

Period expressions resolve to the **first day** of the period. `this week` = Monday of the current week. `this month` = the 1st of the current month. `this year` = January 1.

#### Named Month Expressions

```calcmark
d = next April
d = last December
d = this September
```

`next <month>` resolves to the 1st of that month in its next occurrence. If the current month is April, `next April` means April of next year.

#### Calendar Quarters

```calcmark
q_start = this quarter
planning = next quarter
report = last quarter
```

Calendar quarters: Q1 = January, Q2 = April, Q3 = July, Q4 = October. Resolves to the first day of the quarter.

### Start and End of Periods {#start-end-of}

Use `start of` and `end of` to get the first or last day of any period:

```calcmark
eom = end of this month
eoq = end of this quarter
eoy = end of this year

board_deck = end of this quarter - 2 weeks
```

`end of` resolves to the **last day** of the period. `start of` is the explicit form of the default (first day). These compose with all period types: weeks, months, years, quarters, named months, and fiscal periods.

### Fiscal Periods {#fiscal}

CalcMark supports fiscal year and quarter calculations when configured via frontmatter:

```yaml
---
fiscal_year_starts: july
---
```

You can also specify a start day for fiscal years that don't begin on the 1st:

```yaml
---
fiscal_year_starts: July 15
---
```

Only `Month` or `Month Day` are accepted — not full dates, not relative expressions, not years.

With fiscal configuration:

```calcmark
fq_start = this fiscal quarter
next_fq = next fiscal quarter
fy_start = this fiscal year
fy_end = end of this fiscal year
```

> **Note:** Without the `fiscal_year_starts` frontmatter key, fiscal expressions produce an error: *"fiscal expressions require a 'fiscal_year_starts' frontmatter key"*.

Fiscal quarter numbering: FQ1 begins at the configured start month. With `fiscal_year_starts: july`: FQ1 = Jul-Sep, FQ2 = Oct-Dec, FQ3 = Jan-Mar, FQ4 = Apr-Jun.

### Quarter and Year Shorthand {#notation}

CalcMark supports compact notation for quarters and years:

```calcmark
q1_start = Q1
q4_start = Q4
eoq2 = end of Q2
```

`Q1` through `Q4` are **always calendar quarters**: Q1 = Jan, Q2 = Apr, Q3 = Jul, Q4 = Oct of the current year.

With fiscal configuration, fiscal notation is available:

```calcmark
fq1_start = FQ1
fq3_start = FQ3
fy_start = FY2027
cy_start = CY2026
```

- `FQ1`-`FQ4` — fiscal quarters (requires `fiscal_year_starts`)
- `FY27` or `FY2027` — first day of fiscal year 2027. **FY is labeled by the year it ends in** (Microsoft/Australian convention). With `fiscal_year_starts: july`, FY2027 = July 1, 2026 through June 30, 2027
- `CY26` or `CY2026` — January 1 of calendar year 2026. Disambiguates a year from a bare number (`2026` is the number 2,026; `CY2026` is January 1, 2026)

All notation is case-insensitive: `q1`, `fq3`, `fy27`, `cy2026` all work.

### Leap Year Handling {#leap-years}

CalcMark delegates all calendar math to Go's `time` package for correct leap year handling:

```calcmark
leap = Feb 29 2024
after = Feb 29 2024 + 1 year
```

`Feb 29 2024 + 1 year` = February 28, 2025 (clipped to the last day of February in a non-leap year).

### Year Range {#year-range}

CalcMark validates years to the range **1900-2100**. Dates outside this range produce an error. BCE dates are not supported. CalcMark is designed for technical, business, and personal planning -- not historical research.
