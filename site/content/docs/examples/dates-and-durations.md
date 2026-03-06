---
title: "Dates & Durations"
summary: "Date keywords, literals, arithmetic, and duration calculations."
weight: 45
calcmark_build: progressive
---

From {{< repo-file path="testdata/eval/success/features/dates.cm" >}}.

## Date Keywords

```calcmark
d1 = today
d2 = tomorrow
d3 = yesterday
```

## Date Literals (Month Day)

```calcmark
d4 = Dec 25
d5 = January 15
d6 = Jul 4
```

## Date Literals (Month Day Year)

```calcmark
d7 = Dec 25 2025
d8 = January 1 2026
d9 = Jul 4 2024
```

## Date Arithmetic

```calcmark
d10 = today + 2 days
d11 = tomorrow + 1 week
d12 = Dec 25 2025 + 7 days
d13 = today - 3 days
d14 = yesterday - 1 week
```

## Named Date References

```calcmark
christmas = Dec 25 2025
new_year = christmas + 7 days
week_before = Dec 25 2025 - 1 week
```

## Duration Literals

```calcmark
dur1 = 2 days
dur2 = 3 weeks
dur3 = 1 hour
dur4 = 30 minutes
dur5 = 1 year
```

## Duration Arithmetic

```calcmark
total_time = 2 weeks + 3 days
```

## "X from Y" Syntax

```calcmark
d15 = 2 days from today
d16 = 3 weeks from tomorrow
d17 = 1 week from yesterday
d18 = 5 days from Dec 25 2025

d19 = 1 month from today
d20 = 2 years from Jan 1 2025
```

### What This Demonstrates

- Built-in date keywords: `today`, `tomorrow`, `yesterday`
- Date literals with month names (abbreviated or full)
- Date + duration arithmetic
- Named date variables in expressions
- Duration literals with various time units
- `X from Y` syntax as an alternative to `Y + X`
