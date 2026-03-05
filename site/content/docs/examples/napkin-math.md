---
title: "Napkin Math"
summary: "Quick estimation with human-readable rounding using 'as napkin'."
weight: 40
---

From {{< repo-file path="testdata/eval/success/features/napkin.cm" >}}.

## Basic Numbers

`as napkin` rounds to 2 significant figures and adds human-readable suffixes.

```cm
small_num = 47 as napkin
medium_num = 8734 as napkin
```

## Thousands

```cm
thousands = 347234 as napkin
twelve_k = 12500 as napkin
```

## Millions

```cm
million = 1234567 as napkin
two_million = 2347000 as napkin
```

## Billions

```cm
billion = 1500000000 as napkin
five_b = 5000000000 as napkin
```

## Trillions

```cm
trillion = 1234000000000 as napkin
```

## Negative Numbers

```cm
neg_million = -1234567 as napkin
neg_thousand = -8734 as napkin
```

## With Quantities

```cm
bandwidth = (100 MB/s * 3600) as napkin
storage = (10 TB + 5 TB) as napkin
```

## In Calculations

```cm
load = 10000 req/s as napkin
capacity = 450 req/s as napkin
```

### What This Demonstrates

- `as napkin` rounds to 2 significant figures
- Human-readable suffixes: K, M, B, T
- Works with negative numbers
- Works with unit quantities and rates
- Useful for quick estimation and back-of-envelope calculations
