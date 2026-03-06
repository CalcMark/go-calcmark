---
title: "Functions & Natural Language"
summary: "All function call styles: traditional, natural language, nested, and mixed."
weight: 10
calcmark_build: progressive
---

From {{< repo-file path="testdata/eval/success/features/functions.cm" >}},
{{< repo-file path="testdata/eval/success/features/natural_language.cm" >}},
and {{< repo-file path="testdata/eval/success/features/growth_functions.cm" >}}.

## Traditional Function Syntax

```calcmark
avg(10, 20, 30)
avg(1, 2, 3, 4, 5)
sqrt(16)
sqrt(2)
```

## Natural Language Function Syntax

```calcmark
average of 10, 20, 30
average of 1, 2, 3, 4, 5
square root of 16
square root of 2
```

## read...from / compress...using / transfer...across

```calcmark
read 100 MB from ssd
read 1 GB from nvme
read 500 MB from hdd
read 10 GB from pcie_ssd

compress 1 GB using gzip
compress 500 MB using lz4
compress 2 GB using zstd
compress 100 MB using snappy

transfer 1 GB across regional gigabit
transfer 500 MB across global wifi
transfer 100 MB across local ten_gig
transfer 10 GB across continental hundred_gig
```

## Growth Functions: compound...by...over / grow...by...over / depreciate...by...over

```calcmark
# Compound growth - functional
compound($1000, 5%, 10)
compound(500 customers, 20%, 12)
compound($1000, 5%, 10 years, compounded monthly)
compound($1000, 5%, 10 years, compounded quarterly)

# Compound growth - natural language
compound $1000 by 5% over 10 years
compound $1000 by 12% compounded monthly over 10 years
compound $1000 by 5% per month over 12 months

# Linear growth - functional
grow($500, $100, 36)
grow(100, 20, 5)

# Linear growth - natural language
grow 100 by 20 over 5 months

# Depreciation - functional
depreciate($50000, 15%, 5)
depreciate($50000, 15%, 20, $5000)

# Depreciation - natural language
depreciate $50000 by 15% over 5 years
depreciate $50000 by 15% over 5 years to $5000
```

## Functions with Expressions

```calcmark
x = 10
y = 20
z = 30
a = 5
b = 4

avg(x, y, z)
avg(10 + 5, 20 * 2, 30 - 10)
sqrt(x + y)
square root of (a + b)
```

## Nested Functions

```calcmark
avg(sqrt(16), sqrt(25))
sqrt(avg(1, 2, 3))
average of square root of 4, square root of 9
```

## Functions in Assignments

```calcmark
mean = avg(10, 20, 30)
root = sqrt(16)
calculated = average of 100, 200, 300
side = square root of 25
```

## Mixed Syntax in Same Document

```calcmark
Traditional syntax:
total1 = avg(1, 2, 3)

Natural language syntax:
total2 = average of 1, 2, 3

Both should produce same result:
same = total1 == total2
```

### What This Demonstrates

- Traditional `fn(args)` and natural language `name of args` syntax
- Multi-argument `read...from`, `compress...using`, `transfer...across` patterns
- Growth functions: `compound...by...over`, `grow...by...over`, `depreciate...by...over`
- Expressions and variables as function arguments
- Nested function calls
- Assignment from function results
- Both syntaxes produce identical results
