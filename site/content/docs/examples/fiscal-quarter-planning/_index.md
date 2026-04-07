---
title: "Fiscal Quarter Planning"
summary: "Budget allocation across fiscal quarters with FQ/FY notation, leap year impact, and daily run rates."
weight: 56
calcmark_source: testdata/examples/fiscal-quarter-planning.cm
---

How does a flat quarterly budget translate to different daily costs when fiscal quarters have different lengths? This example models an Australian SaaS company's $4.8M engineering budget across fiscal quarters (July fiscal year start), showing how FQ notation, calendar-correct arithmetic, and the leap year impact combine.

The complete CalcMark file is available at {{< repo-file path="testdata/examples/fiscal-quarter-planning.cm" >}}.

---

## Frontmatter

The `fiscal_year_starts` key enables FQ and FY notation:

```yaml
---
fiscal_year_starts: july
---
```

You can also specify a start day: `fiscal_year_starts: July 15`.

**CalcMark features:** `fiscal_year_starts` frontmatter key.

---

## FQ Notation

`FQ1` through `FQ4` resolve to the first day of each fiscal quarter. `end of FQ1` gives the last day.

```calcmark
fq1_start = FQ1
fq1_end = end of FQ1
fq2_start = FQ2
fq2_end = end of FQ2
```

**CalcMark features:** `FQ1`-`FQ4` notation; `end of` modifier.

---

## Calendar vs. Fiscal

Calendar Q3 (July) is Fiscal Q1 when the fiscal year starts in July:

```calcmark
cal_q3 = Q3
fiscal_q1 = FQ1
```

**CalcMark features:** `Q1`-`Q4` (always calendar); `FQ1`-`FQ4` (fiscal, requires frontmatter).

---

## Daily Run Rates

Same $1.2M per quarter, different daily costs because quarters have 90-92 days:

```calcmark
quarterly_budget = $4,800,000 / 4
fq1_daily = quarterly_budget / 92
fq3_daily = quarterly_budget / 90
```

---

## Leap Year Impact

FQ3 spans January-March. In a leap year, that's 91 days instead of 90:

```calcmark
fy24_fq3 = Apr 1 2024 - Jan 1 2024
fy25_fq3 = Apr 1 2025 - Jan 1 2025
leap_difference = Apr 1 2024 - Jan 1 2024 - (Apr 1 2025 - Jan 1 2025)
```

**CalcMark features:** Calendar-correct date subtraction; leap year handling.

---

## Key Dates

FQ notation composes with arithmetic:

```calcmark
board_review = FQ3 + 15 days
budget_lock = end of this fiscal year - 2 weeks
```

**CalcMark features:** FQ + duration arithmetic; `end of this fiscal year`.
