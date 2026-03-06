---
title: "Language Reference"
summary: "Formal specification for the CalcMark language."
weight: 30
---

**Version:** 1.0.0

This is the complete and authoritative specification for the CalcMark language.

---

## Overview

CalcMark is a calculation language that blends seamlessly with markdown. It allows calculations to live naturally within prose documents.

### Design Goals

- **Familiar**: Syntax feels like calculator/spreadsheet usage
- **Minimal**: Only essential features, no unnecessary complexity
- **Unambiguous**: One way to do things, clear error messages
- **Unicode-aware**: Full international character support
- **Markdown-compatible**: Works within existing markdown documents

### Key Characteristics

- **Line-based**: Each line is classified independently
- **Context-aware**: Variables must be defined before use
- **Strongly typed**: Minimal type coercion, clear type errors
- **Arbitrary precision**: Decimal math for exact results

---

## Philosophy

### Calculation by Exclusion

CalcMark uses "calculation by exclusion" - if a line looks like markdown, it's markdown. Only unambiguous calculations are treated as calculations.

```cm
# My Budget          -> MARKDOWN (header prefix)
salary = $5000       -> CALCULATION (assignment)
This is text         -> MARKDOWN (natural language)
5 + 3                -> CALCULATION (arithmetic)
- List item          -> MARKDOWN (bullet prefix)
-5 + 3               -> CALCULATION (negative number)
$100 budget          -> MARKDOWN (trailing text)
```

### Explicit Over Implicit

- Spaces NOT allowed in identifiers (`my_budget`, not `my budget`)
- Forward references not allowed (define before use)
- Type mismatches are errors (no silent coercion)
- Reserved keywords cannot be variable names

---

## Document Model

A CalcMark document is a sequence of **lines**. Each line is independently:

1. **Classified** as BLANK, MARKDOWN, or CALCULATION
2. **Parsed** (if CALCULATION)
3. **Validated** (optional, produces diagnostics)
4. **Evaluated** (if CALCULATION and valid)

### Three Line Types

| Type | Description | Examples |
|------|-------------|----------|
| **BLANK** | Empty or whitespace-only | `""`, `"   "`, `"\t"` |
| **MARKDOWN** | Prose, headers, lists, or invalid calculations | `"# Header"`, `"Some text"`, `"- Item"` |
| **CALCULATION** | Valid CalcMark expression or assignment | `"x = 5"`, `"10 + 20"`, `"salary * 12"` |

---

## Line Classification

### Classification Rules

Lines are classified in this order:

1. **BLANK** — Empty or only whitespace
2. **INDENTED CODE** → MARKDOWN — Line starts with 4+ spaces or a tab
3. **FENCED CODE BLOCK** → MARKDOWN — Lines between `` ``` `` or `~~~` fences (stateful; all content inside is markdown regardless of what it looks like)
4. **MARKDOWN pattern** — Matches a known CommonMark construct:
   - Block-level: `#` (ATX heading), `>` (blockquote), `- ` / `* ` / `+ ` (unordered list), `digit.` (ordered list), `---` / `***` / `___` (horizontal rule), `===` / `---` (setext heading underline), `` ``` `` / `~~~` (fenced code fence)
   - Inline-level at start of line: `![` (image), `[text](url)` (inline link), `[id]: url` (link definition), `**text**` (bold formatting)
5. **CALCULATION** — Attempt to parse and validate:
   - Starts with a literal (number, currency, boolean)
   - Contains assignment (`=`)
   - Is a valid expression
   - All variables are defined (context-aware)
6. **MARKDOWN** (fallback) — Anything else

### Context-Aware Classification

```cm
x = 5               -> CALCULATION (assignment)
y = x + 10          -> CALCULATION (x is defined)
z = unknown * 2     -> MARKDOWN (unknown is undefined)
```

### Edge Cases

| Input | Classification | Reason |
|-------|----------------|--------|
| `$100 budget` | MARKDOWN | Trailing text after valid token |
| `-5 + 3` | CALCULATION | Negative number (no space after `-`) |
| `- 5` | MARKDOWN | Bullet list (space after `-`) |
| `+ item` | MARKDOWN | Unordered list (`+` marker with space) |
| `    x = 10` | MARKDOWN | Indented code block (4-space prefix) |
| `![alt](img.png)` | MARKDOWN | Image syntax |
| `[ref]: https://…` | MARKDOWN | Link definition |
| `x *` | MARKDOWN | Incomplete expression |
| `average` | MARKDOWN | Not reserved, not in context |
| `avg` | MARKDOWN | Reserved keyword alone (not a valid expression) |

---

## Syntax & Grammar

### EBNF Grammar

```
Statement       ::= Assignment | Expression
Assignment      ::= IDENTIFIER "=" Expression
Expression      ::= Comparison
Comparison      ::= Additive (ComparisonOp Additive)?
ComparisonOp    ::= ">" | "<" | ">=" | "<=" | "==" | "!="
Additive        ::= Multiplicative (("+"|"-") Multiplicative)*
Multiplicative  ::= Exponent (("*"|"/"|"%") Exponent)*
Exponent        ::= Unary ("^" Unary)*
Unary           ::= ("-"|"+")? Primary
Primary         ::= Number | Currency | Boolean | Identifier | "(" Expression ")"
```

### Operator Precedence

From **highest** to **lowest**:

1. Parentheses `()`
2. Exponentiation `^` (right-associative)
3. Unary `-`, `+` (prefix)
4. Multiplicative `*`, `/`, `%` (left-associative)
5. Additive `+`, `-` (left-associative)
6. Comparison `>`, `<`, `>=`, `<=`, `==`, `!=` (non-associative)

---

## Type System {#type-system}

### Data Types

| Type | Example | Internal |
|------|---------|----------|
| **Number** | `42`, `3.14`, `1,000` | Arbitrary-precision decimal |
| **Currency** | `$100`, `€50.99` | Symbol + decimal |
| **Boolean** | `true`, `false`, `yes`, `no` | Boolean |
| **Quantity** | `10 meters`, `5 kg`, `100 MB` | Value + unit |
| **Duration** | `5 days`, `2 weeks`, `1 year` | Value + time unit |
| **Rate** | `100 MB/s`, `$50/hour`, `1000 req/s` | Numerator / time unit |
| **Date** | `Jan 15 2025`, `today` | Calendar date |

### Type Compatibility

**Binary operations (preserve units):**

```cm
Number + Number -> Number
Currency + Number -> Currency  (unit preserved)
Number + Currency -> Currency  (unit preserved)
Currency + Currency (same symbol) -> Currency
Currency + Currency (different symbols) -> Number  (units dropped)
Quantity + Quantity (same unit) -> Quantity
Date + Duration -> Date
Date - Date -> Duration
Rate * Duration -> Quantity  (via "over" keyword)
```

**Functions (drop units when mixed):**

```cm
avg($100, $200) -> $150.00  (same unit preserved)
avg($100, €200) -> 150  (Number, mixed units)
sqrt($100) -> $10.00  (single unit preserved)
```

**Type errors:**

```cm
Boolean + Number -> ERROR (no boolean arithmetic)
Quantity + Currency -> ERROR (incompatible types)
```

### Literals

#### Numbers

```
42              Valid integer
3.14            Valid decimal
1,000           Thousands separator (comma)
1_000_000       Thousands separator (underscore)
0.5             Leading zero
.5              Invalid (must have leading zero)
1.2.3           Invalid (multiple decimal points)
```

#### Multiplier Suffixes

```cm
10K             -> 10000
5M              -> 5000000
2B              -> 2000000000
1.5K            -> 1500
```

#### Currency

```
$100            USD
$1,000.50       With separators
€50             EUR
£25.99          GBP
¥1000           JPY
100$            Invalid (symbol must be prefix)
$ 100           Invalid (no space between symbol and number)
```

**Supported symbols:** `$`, `€`, `£`, `¥`

#### Percentages

```cm
50%             -> 0.5 (Number)
8.25%           -> 0.0825
15% of 200      -> 30
```

#### Booleans

Case-insensitive keywords:

```
true, false     Standard
yes, no         Natural language
t, f            Single letter
y, n            Single letter
True, FALSE     Any case
```

#### Quantities

```cm
10 meters       Quantity: 10 in meters
5 kg            Quantity: 5 in kilograms
100 MB          Quantity: 100 in megabytes
```

#### Rates

```cm
100 MB/s        Rate: 100 megabytes per second
$50/hour        Rate: $50 per hour
1000 req/s      Rate: 1000 requests per second
$120000/year    Rate: $120,000 per year
```

#### Dates

```cm
Jan 15 2025     Date literal
Dec 25 2025     Date literal
today           Current date
tomorrow        Tomorrow's date
yesterday       Yesterday's date
```

#### Durations

```cm
5 days          Duration
2 weeks         Duration
3 months        Duration
1 year          Duration
```

#### Identifiers

- Must start with letter, underscore, or Unicode character (not digit)
- Can contain letters, digits, underscores, Unicode, emoji
- Cannot be reserved keywords or constants
- Spaces NOT allowed (use underscores)

```
x               Valid
salary          Valid
tax_rate        Valid (use underscores, not spaces)
_private        Valid (underscore prefix)
123abc          Invalid (cannot start with digit)
my budget       Invalid (spaces not allowed, use my_budget)
avg             Invalid (reserved keyword)
PI              Invalid (reserved constant)
```

### Mathematical Constants

Built-in constants (read-only, case-insensitive):

| Constant | Value |
|----------|-------|
| `PI`, `pi` | `3.141592653589793` |
| `E`, `e` | `2.718281828459045` |

Constants cannot be assigned:

```cm
PI = 3          ERROR: Cannot assign to constant 'PI'
radius = 5
area = PI * radius ^ 2
```

---

## Operators

### Arithmetic

| Operator | Name | Example | Result | Associativity |
|----------|------|---------|--------|---------------|
| `^` | Exponent | `2 ^ 3` | `8` | Right |
| `*` | Multiply | `3 * 4` | `12` | Left |
| `/` | Divide | `10 / 2` | `5` | Left |
| `%` | Modulus | `10 % 3` | `1` | Left |
| `+` | Add | `5 + 3` | `8` | Left |
| `-` | Subtract | `5 - 3` | `2` | Left |

**Multiply aliases:** `*`, `x`, `X` (when following a number)

### Comparison

| Operator | Name | Example | Result |
|----------|------|---------|--------|
| `>` | Greater than | `5 > 3` | `true` |
| `<` | Less than | `5 < 3` | `false` |
| `>=` | Greater or equal | `5 >= 5` | `true` |
| `<=` | Less or equal | `5 <= 3` | `false` |
| `==` | Equal | `5 == 5` | `true` |
| `!=` | Not equal | `5 != 3` | `true` |

### Unary

| Operator | Name | Example | Result |
|----------|------|---------|--------|
| `-` | Negation | `-5` | `-5` |
| `+` | Plus | `+5` | `5` |

### Assignment

| Operator | Name | Example | Effect |
|----------|------|---------|--------|
| `=` | Assign | `x = 5` | Stores 5 in variable x |

---

## Reserved Keywords

These words **cannot** be used as variable names.

### Logical Operators

```
and, or, not
```

Case-insensitive: `AND`, `and`, `And` all work.

### Control Flow (Reserved for Future)

```
if, then, else, elif, end
for, in, while
return, break, continue
let, const
```

### Function Names

All built-in function names are reserved:

```
avg, sqrt, accumulate, convert_rate, capacity,
downtime, rtt, throughput, transfer_time,
read, seek, compress, compound, grow, depreciate
```

### Language Keywords

```
in, as, of, per, over, at, from, with, napkin, precise
```

---

## Functions {#functions}

{{< feature-table category="function" >}}

For detailed examples of every function, including natural language syntax forms, see the [User Guide: Function Reference](/docs/user-guide/#function-reference).

### Unit Handling in Functions

**Same units are preserved:**

```cm
avg($100, $200, $300) -> $200.00
sqrt($100) -> $10.00
```

**Mixed units are dropped:**

```cm
avg($100, €200) -> 150  (no units)
average of $50, €100, £150 -> 100  (no units)
```

---

## Natural Language Syntax {#natural-language-syntax}

CalcMark supports natural language forms for many functions. These are equivalent to the function-call syntax.

| Pattern | Equivalent |
|---------|------------|
| `average of X, Y, Z` | `avg(X, Y, Z)` |
| `square root of X` | `sqrt(X)` |
| `read X from Y` | `read(X, Y)` |
| `compress X using Y` | `compress(X, Y)` |
| `transfer X across Y Z` | `transfer_time(X, Y, Z)` |
| `compound X by Y% over Z` | `compound(X, Y%, Z)` |
| `compound X by Y% per P over Z` | `compound(X, Y%, Z, P)` |
| `compound X by Y% compounded F over Z` | `compound(X, Y%, Z, compounded F)` |
| `grow X by Y over Z` | `grow(X, Y, Z)` |
| `depreciate X by Y% over Z` | `depreciate(X, Y%, Z)` |
| `depreciate X by Y% over Z to W` | `depreciate(X, Y%, Z, W)` |

See the [User Guide: Natural Language Syntax](/docs/user-guide/#natural-language-syntax) for the complete reference table with examples.

---

## Napkin Math {#as-napkin}

The `as napkin` modifier rounds results to 2 significant figures and normalizes units. See the [User Guide: Napkin Math](/docs/user-guide/#napkin-math) for usage examples.

**Syntax:** `expression as napkin`

**Works with:** Number, Quantity, Currency, Duration, Rate

---

## Precise Display {#as-precise}

The `as precise` modifier shows full float precision, skipping all display rounding. This is the opposite of `as napkin` and is useful when you need exact values from unit conversions.

**Syntax:** `expression as precise`

Can be chained after a unit conversion: `10 meters as feet as precise`

**Works with:** Number, Quantity, Currency, Duration, Rate

---

## Rates {#rates}

Rates are defined using slash syntax (e.g., `100 MB/s`, `$50/hour`). See the [User Guide: Rates](/docs/user-guide/#rates) for rate accumulation with `over` and rate conversion.

---

## Date Arithmetic {#dates}

CalcMark supports date literals (`Jan 15 2025`, `today`), duration arithmetic, and the `from` keyword. See the [User Guide: Date Arithmetic](/docs/user-guide/#date-arithmetic) for details.

---

## Network Functions {#network}

CalcMark provides `rtt`, `throughput`, `transfer_time`, and `downtime` for network planning. See the [User Guide: Network Functions](/docs/user-guide/#network-functions) for scope tables and examples.

---

## Storage Functions {#storage}

CalcMark provides `read`, `seek`, and `compress` for storage planning. See the [User Guide: Storage Functions](/docs/user-guide/#storage-functions) for device type tables and examples.

---

## Growth Functions {#growth}

CalcMark provides `compound`, `grow`, and `depreciate` for modeling growth and depreciation over time.

### Compound Growth

```cm
compound($1000, 5%, 10)                              -> $1628.89
compound(500 customers, 20%, 12)                     -> 4458.05 customers
compound($1000, 5%, 10 years, compounded monthly)    -> $1647.01
```

**Natural language forms:**

```cm
compound $1000 by 5% over 10 years
compound $1000 by 5% per month over 12 months
compound $1000 by 12% compounded monthly over 10 years
```

### Linear Growth

```cm
grow($500, $100, 36)               -> $4100.00
grow 100 by 20 over 5 months       -> 200  (NL form)
```

### Depreciation

```cm
depreciate($50000, 15%, 5)                -> $22185.27
depreciate($50000, 15%, 20, $5000)        -> $5000.00  (salvage floor)
depreciate $50000 by 15% over 5 years     -> (NL form)
depreciate $50000 by 15% over 5 years to $5000
```

See the [User Guide: Growth Functions](/docs/user-guide/#growth-functions) for the full argument reference.

---

## Validation & Diagnostics

### Diagnostic Codes

| Code | Severity | Description |
|------|----------|-------------|
| `SyntaxError` | Error | Invalid syntax |
| `UndefinedVariable` | Warning | Variable used before definition |
| `TypeMismatch` | Error | Incompatible types in operation |
| `DivisionByZero` | Error | Division or modulus by zero |

### Diagnostic Levels

| Severity | Meaning |
|----------|---------|
| **Error** | Prevents evaluation, line becomes MARKDOWN |
| **Warning** | Line stays CALCULATION but evaluation may fail |
| **Info** | Informational, doesn't affect classification |
| **Hint** | Suggestions for improvement |
