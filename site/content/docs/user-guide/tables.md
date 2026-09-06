---
title: "Named Tables & Arrays"
weight: 5
---

A markdown table is data. Give it a name and CalcMark can compute with its columns — sum a column, multiply two columns row by row, and write per-row results back into a table — without copying numbers into a calc block.

## Naming a table {#naming}

Put a `table:` directive on the line before a markdown table. The directive is an HTML comment, so it is invisible in any markdown viewer; only the table shows.

```output
<!-- table: rates (role, rate, hc) -->
| Role   | Hourly rate | Headcount |
|--------|-------------|-----------|
| Senior | $250        | 2         |
| Junior | $150        | 5         |
```

- The **table name** (`rates`) and the **column names** (`role`, `rate`, `hc`) come from the directive, not from the header row. Header text is free to say whatever reads best.
- Names are lowercased and inner spaces become underscores: `<!-- table: Q1 Sales (Region, Total Rev) -->` gives `q1_sales.total_rev`.
- The directive must declare exactly as many columns as the table has. CalcMark reports a mismatch on the directive line.
- A table with no directive is ordinary markdown. Nothing changes for existing documents.

## Reading columns {#columns}

Each column is an **array** — an ordered list of values of one kind. Read a column with a dot:

```output
senior_rate = rates.rate          → [$250.00, $150.00]
people = rates.hc                 → [2, 5]
```

Cells are parsed like any CalcMark literal: `$250` is currency, `2 weeks` is a duration, `10 GB/s` is a rate, `3` is a number. A cell that is not a literal (`Senior`, `n/a`) is **text**: it can be counted and shown, but not used in arithmetic. Every cell in a column must be the same kind of value — a column mixing `$100` and `50%` is reported at the offending cell.

## Arithmetic on columns {#arithmetic}

Arithmetic between two columns works row by row; a scalar applies to every row:

```output
line_costs = rates.rate * rates.hc   → [$500.00, $750.00]
uplifted = rates.rate * 1.1          → [$275.00, $165.00]
```

Both arrays must have the same number of rows. All the usual rules apply per row — units convert, currencies must match, and a text column cannot take part.

## Aggregates {#aggregates}

`sum`, `avg`, `min`, `max`, and `count` accept a single array:

```output
total = sum(rates.rate * rates.hc)   → $1,250.00
average_rate = avg(rates.rate)       → $200.00
cheapest = min(rates.rate)           → $150.00
largest_team = max(rates.hc)         → 5
roles = count(rates.role)            → 2
```

The scalar forms still work (`sum(a, b, c)`, `max(4, 9, 1)`). A call takes either one array or several scalars, never a mix.

## Showing results in a table {#interpolation}

Use `{{name}}` inside a markdown table row and each row receives its own element:

```output
| Role   | Line cost      |
|--------|----------------|
| Senior | {{line_costs}} |
| Junior | {{line_costs}} |
```

renders as

```output
| Role   | Line cost |
|--------|-----------|
| Senior | $500.00   |
| Junior | $750.00   |
```

The array must have exactly one value per data row; otherwise the tags are left unresolved and CalcMark reports the mismatch on the table. Outside a table, `{{line_costs}}` and `{{rates.rate}}` render as a list: `[$500.00, $750.00]`.

## Order matters {#order}

Tables register where they appear. Calc blocks **below** a table can read it; blocks above cannot. This is the same top-to-bottom rule as variables.

## What is not supported {#limits}

- No array literals (`[1, 2, 3]`) — arrays only come from tables.
- No indexing or row lookup (`rates.rate[0]`, filtering).
- No formulas inside cells — cells are literals; computation happens in calc blocks.
- No loading from files.

## In Embedded mode {#embedded}

In markdown-with-fences documents the directive and table live in the prose, and the fenced ```` ```cm ```` blocks below read them:

````output
<!-- table: rates (rate, hc) -->
| Rate | HC |
|------|----|
| $250 | 2  |

```cm
total = sum(rates.rate * rates.hc)
```
````
