# CalcMark Language Specification

**Version:** 2.0.0
**Status:** Current

This is the **complete and authoritative** specification for the CalcMark language. The implementation in `spec/` and `impl/` is the source of truth; this document describes the language as implemented.

---

## Table of Contents

1. [Overview](#overview)
2. [Document Model](#document-model)
3. [Syntax & Grammar](#syntax--grammar)
4. [Type System](#type-system)
5. [Operators](#operators)
6. [Units & Measurement](#units--measurement)
7. [Rates](#rates)
8. [Dates & Durations](#dates--durations)
9. [Functions](#functions)
10. [Keywords](#keywords)
11. [Frontmatter](#frontmatter)
12. [Display & Formatting](#display--formatting)
13. [Validation & Diagnostics](#validation--diagnostics)

---

## Overview

CalcMark is a calculation language that blends with CommonMark markdown. Calculations and prose coexist in a single document. Each line is independently classified as markdown or calculation — no mode switching, no special blocks.

### Design Goals

- **Familiar**: Syntax resembles calculator and spreadsheet usage
- **Minimal**: Only essential features, no unnecessary complexity
- **Unambiguous**: One way to do things, clear error messages
- **Unicode-aware**: Full international character support
- **Markdown-compatible**: Works within existing markdown documents

### Key Characteristics

- **Line-based**: Each line is classified independently
- **Context-aware**: Variables must be defined before use (no forward references)
- **Strongly typed**: Minimal type coercion, clear type errors
- **Arbitrary precision**: Decimal math via `github.com/shopspring/decimal`, exact fractions via `math/big.Rat`

---

## Document Model

A CalcMark document is a sequence of lines. Each line is independently:

1. **Classified** as BLANK, MARKDOWN, or CALCULATION
2. **Parsed** (if CALCULATION)
3. **Evaluated** (if CALCULATION and valid)
4. **Transformed** (if frontmatter directives apply: scale, convert_to)

### Line Classification

Lines are classified by exclusion — if a line looks like markdown, it's markdown. Only unambiguous calculations are treated as calculations.

**Classification rules (in order):**

1. **BLANK** — Empty or only whitespace
2. **MARKDOWN** — Has markdown prefix (`#`, `>`, `-`, `*`, `digit.`)
3. **CALCULATION** — Starts with a number, currency, boolean, `(`, `-number`, or assignment; all referenced variables are defined
4. **MARKDOWN** (fallback) — Anything else

```
# My Budget          -> MARKDOWN (header prefix)
salary = $5000       -> CALCULATION (assignment)
This is text         -> MARKDOWN (natural language)
5 + 3                -> CALCULATION (arithmetic)
- List item          -> MARKDOWN (bullet prefix)
-5 + 3               -> CALCULATION (negative number)
z = unknown * 2      -> MARKDOWN (unknown is undefined)
```

---

## Syntax & Grammar

### EBNF Grammar (Simplified)

```ebnf
Statement       ::= Assignment | Expression
Assignment      ::= IDENTIFIER "=" Expression
Expression      ::= LogicalOr
LogicalOr       ::= LogicalAnd ("or" LogicalAnd)*
LogicalAnd      ::= Comparison ("and" Comparison)*
Comparison      ::= Additive (ComparisonOp Additive)?
Additive        ::= Multiplicative (("+"|"-") Multiplicative)*
Multiplicative  ::= Exponent (("*"|"/"|"%") Exponent)*
Exponent        ::= Unary ("^" Unary)*
Unary           ::= ("-"|"+"|"not")? Postfix
Postfix         ::= Primary PostfixOp*
PostfixOp       ::= "in" Unit | "as" Format | "over" Duration | "per" TimeUnit
                   | "at" CapacityExpr | "%" | "of" Expression
Primary         ::= Number | Fraction | Currency | Quantity | Duration
                   | Date | Boolean | Percentage | RateLiteral
                   | FunctionCall | Identifier | "(" Expression ")"
RateLiteral     ::= Quantity "/" TimeUnit | Quantity "per" TimeUnit
FunctionCall    ::= IDENTIFIER "(" ArgList ")"
```

### Operator Precedence

From **lowest** to **highest**:

1. Assignment `=`
2. Logical OR `or`
3. Logical AND `and`
4. Comparison `==`, `!=`, `<`, `>`, `<=`, `>=`
5. Addition/Subtraction `+`, `-`
6. Multiplication/Division/Modulo `*`, `/`, `%`
7. Exponentiation `^` or `**` (right-associative)
8. Unary `+`, `-`, `not`
9. Postfix `in`, `as`, `over`, `per`, `at`, `of`
10. Primary (literals, identifiers, parentheses)

### Literals

#### Numbers

```
42              integer
3.14            decimal
1,000           thousands separator (comma)
1_000_000       thousands separator (underscore)
.5              leading zero optional
5K              5,000 (multiplier suffix)
2.5M            2,500,000
1B              1,000,000,000
3T              3,000,000,000,000
1.2e10          scientific notation
4.5e-7          negative exponent
```

Multiplier suffixes: `K`/`k` (thousand), `M` (million), `B`/`b` (billion), `T`/`t` (trillion).

#### Fractions

Write numerator/denominator **without spaces** around the `/`:

```
1/2             simple fraction
3/4             simple fraction
11 3/8          mixed number (integer + fraction)
```

Fractions are exact rational numbers (arbitrary-precision `big.Rat`). Arithmetic results are automatically simplified to lowest terms.

Fractions with denominators exceeding 10^9 or numerators exceeding 10^18 are automatically converted to decimal for safety.

#### Currency

```
$100            prefix symbol
$1,000.50       with separators
EUR100          ISO 4217 code prefix
100 USD         code suffix
```

Supported symbols: `$`, `€`, `£`, `¥`. Any 3-letter uppercase code is accepted as a currency code.

#### Percentages

```
50%             percentage literal (stored as 0.5 internally)
8.25%           decimal percentage
```

#### Booleans

Case-insensitive: `true`, `false`, `yes`, `no`, `t`, `f`, `y`, `n`.

#### Identifiers

- Must start with letter, underscore, or Unicode character (not digit)
- Can contain letters, digits, underscores, Unicode, emoji
- **No spaces** in identifiers (use underscores: `my_budget`, not `my budget`)
- Cannot shadow reserved keywords or constants

```
x               valid
tax_rate        valid
給料            valid (Japanese)
💰              valid (emoji)
123abc          invalid (starts with digit)
avg             invalid (reserved function name)
PI              invalid (reserved constant)
```

### Mathematical Constants

| Constant | Value |
|----------|-------|
| `PI`, `pi` | 3.141592653589793 |
| `E`, `e` | 2.718281828459045 |

Constants are read-only and cannot be assigned to.

---

## Type System

### Data Types

| Type | Example | Internal |
|------|---------|----------|
| **Number** | `42`, `3.14`, `5K` | `decimal.Decimal` |
| **Fraction** | `1/3`, `11 3/8` | `big.Rat` (exact) |
| **Percentage** | `50%`, `8.25%` | Fractional decimal (0.5) |
| **Currency** | `$100`, `EUR50` | Symbol + decimal |
| **Quantity** | `10 meters`, `5 kg` | Value + unit string |
| **Duration** | `5 days`, `2 weeks` | Value + time unit |
| **Rate** | `100 MB/s`, `$50/hour` | Quantity / time unit |
| **Date** | `today`, `Jan 15 2025` | Calendar date |
| **Boolean** | `true`, `false` | Boolean |

### Type Compatibility Matrix

| Left | Right | `+` / `-` | `*` | `/` | Notes |
|------|-------|-----------|-----|-----|-------|
| Number | Number | Number | Number | Number | Standard arithmetic |
| Fraction | Fraction | Fraction | Fraction | Fraction | Exact rational, auto-simplified |
| Number | Fraction | Fraction | Fraction | Fraction | Number promoted to Fraction |
| Number | Currency | Currency | Currency | — | Right symbol preserved |
| Currency | Number | Currency | Currency | Currency | Left symbol preserved |
| Currency | Currency | Currency | — | — | Same ISO code required |
| Number | Quantity | Quantity | Quantity | — | Right unit preserved |
| Quantity | Number | Quantity | Quantity | Quantity | Left unit preserved |
| Quantity | Quantity | Quantity | — | — | Same-category; left unit wins |
| Duration | Duration | Duration | — | — | Converted to left's unit |
| Duration | Number | — | Duration | Duration | Left unit preserved |
| Date | Duration | Date | — | — | Add/subtract days |
| Date | Date | Duration | — | — | Subtraction only: days between |
| Rate | Number | — | Rate | Rate | Scales the amount |
| Speed | Duration | — | Distance | — | Bridge: 60 mph * 2 hours = 120 mi |
| *any* | Percentage | Widened | Extract | Extract | See percentage rules below |
| Percentage | Percentage | Percentage | — | — | Decimal addition |
| Boolean | Boolean | — | — | — | `and`, `or` only |

### Percentage Widening

When a percentage appears as the **right** operand of `+` or `-`, it applies proportionally:

```
$100 + 10%     -> $110    (100 * 1.10)
$100 - 20%     -> $80     (100 * 0.80)
200 + 50%      -> 300     (200 * 1.50)
```

For `*` and `/`, the percentage is extracted as its decimal value:

```
$100 * 50%     -> $50     (100 * 0.50)
```

### Rate Widening

When a rate appears as the **right** operand of `*` or `/`, its time denominator is dropped:

```
3 * (2 posts/week)     -> 6 posts     (amount extracted, time dropped)
```

When a rate is the **left** operand, it stays a rate:

```
(100 MB/s) * 2         -> 200 MB/s    (rate scaled)
```

---

## Units & Measurement

### Unit Categories

CalcMark supports 14 unit categories. Units within the same category can be converted with `in`.

| Category | Units | Example Conversion |
|----------|-------|-------------------|
| **Acceleration** | standard-gravity, m/s^2, cm/s^2, ft/s^2 | `1 standard-gravity in ft/s^2` |
| **Area** | m², km², ha, ft², in², yd², mi², acre | `5 acres in hectares` |
| **DataSize** | byte, KB, MB, GB, TB, PB (+ binary: KiB, MiB, etc.) | `1.5 GB in MB` |
| **Energy** | joule, kilojoule, calorie, kilocalorie, kWh | `1 kwh in joules` |
| **Force** | newton, kilonewton, dyne, kilogram-force, pound-force, poundal | `10 newtons in pound-force` |
| **Frequency** | hertz, kilohertz, megahertz, gigahertz, terahertz | `2.4 gigahertz in megahertz` |
| **Impulse** | newton-second, pound-force-second | `110 pound-force-seconds in newton-seconds` |
| **Length** | m, cm, mm, km, in, ft, yd, mi, nmi | `26.2 miles in km` |
| **Mass** | mg, g, kg, tonne, oz, lb, troy oz, troy lb | `80 kg in pounds` |
| **Power** | watt, kilowatt, megawatt, horsepower | `300 hp in kilowatts` |
| **Pressure** | pascal, kilopascal, megapascal, bar, millibar, atmosphere, torr, psi, inch of mercury | `1 atmosphere in psi` |
| **Speed** | m/s, km/h, mph, knot | `100 kph in mph` |
| **Temperature** | celsius, fahrenheit, kelvin | `100 celsius in fahrenheit` |
| **Volume** | mL, L, tsp, tbsp, cup, fl oz, pt, qt, gal (+ imperial variants) | `2 cups in ml` |

Additionally: **Currency**, **Custom**, **Number** are virtual categories used by frontmatter directives.

Run `cm help constants` for the complete list with aliases.

### Unit Conversion

```
10 meters in feet       -> 32.81 feet
100 celsius in fahrenheit -> 212 fahrenheit
1.5 GB in MB            -> 1536 MB
```

Conversions are only valid within the same category. `10 meters in kilograms` produces an error.

### Hyphenated Units

Some units have hyphenated names. Write them with hyphens in both expressions and `in` targets:

```
50 pound-force
110 pound-force-seconds in newton-seconds
1 kilogram-force in newtons
```

### Custom Units

Any identifier can be a unit. Custom units like `cookies`, `servers`, or `widgets` work in arithmetic but cannot be converted to standard units and are unaffected by `convert_to`.

```
yield = 24 cookies
double = yield * 2      -> 48 cookies
```

### Measurement Conventions

Some unit names are ambiguous (US gallon vs UK gallon, avoirdupois vs troy ounce). Configure via the `measurement` frontmatter directive. See [Frontmatter](#frontmatter).

### Speed-Rate Bridge

Speed units (`mph`, `kph`, `mps`, `knots`) are quantities, but CalcMark bridges them to rates when needed:

```
60 mph over 2 hours     -> 120 mi       (speed -> rate -> accumulate)
60 kph * 2 hours        -> 120 km       (speed * duration -> distance)
60 kph in m/s           -> 16.67 m/s    (speed -> rate conversion)
60 km/h in mph          -> 37.28 mph    (rate -> speed conversion)
```

The bridge only fires for Speed-category units. Other quantities multiplied by durations produce an error.

---

## Rates

Rates represent a quantity per unit of time. They are first-class types.

### Rate Syntax

```
100 MB/s                slash notation
100 MB per second       word notation
$50/hour                currency rate
1000 req/s              custom unit rate
```

The denominator must be a time unit (second, minute, hour, day, week, month, quarter, year).

### Rate Operations

```
rate = 100 MB/s
doubled = rate * 2              -> 200 MB/s       (scaling)
total = rate over 1 day         -> 8,640,000 MB   (accumulation)
total = accumulate(rate, 1 day) -> 8,640,000 MB   (function form)
converted = 100 MB/s in GB/min  -> rate conversion
```

### Rate Accumulation

The `over` keyword (or `accumulate()` function) multiplies a rate by a duration to produce a total quantity:

```
100 MB/s over 1 hour    -> 360,000 MB
$0.10/hour over 30 days -> $72
5 GB/day over 1 year    -> 1,825 GB
```

---

## Dates & Durations

### Date Literals

```
today                   current date
tomorrow                next day
yesterday               previous day
Jan 15 2025             explicit date
December 25             month + day (current year)
```

### Relative Dates

```
this week               Monday of current week
next month              1st of next month
last year               Jan 1 of last year
```

### Duration Units

| Unit | Aliases |
|------|---------|
| millisecond | ms |
| second | s, sec |
| minute | m, min |
| hour | h, hr |
| day | d |
| week | w, wk |
| month | mo |
| quarter | — |
| year | y, yr |

### Date Arithmetic

```
today + 7 days          date + duration -> date
tomorrow + 1 week       date + duration -> date
today - yesterday       date - date -> duration
7 days from today       duration from date -> date
```

---

## Functions

All functions have both a traditional `fn(args)` form and a natural language form. Run `cm help functions` for the complete list.

### Math

| Function | Signature | NL Form | Description |
|----------|-----------|---------|-------------|
| `avg` | `avg(a, b, ...)` | `average of a, b, c` | Average of values |
| `sum` | `sum(a, b, ...)` | `sum of a, b, c` | Sum of values |
| `sqrt` | `sqrt(x)` | `square root of x` | Square root |
| `number` | `number(x)` | — | Strip units, return plain number |

### Rate & Capacity

| Function | Signature | NL Form | Description |
|----------|-----------|---------|-------------|
| `accumulate` | `accumulate(rate, time)` | `{rate} over {time}` | Total from rate over duration |
| `convert_rate` | `convert_rate(rate, unit)` | — | Convert rate to different time unit |
| `capacity` | `capacity(demand, cap, unit)` | `{demand} at {cap} per {unit}` | Calculate units needed for load |

### Network & Storage

| Function | Signature | NL Form | Description |
|----------|-----------|---------|-------------|
| `rtt` | `rtt(scope)` | — | Round-trip time (local, regional, continental, global) |
| `throughput` | `throughput(type)` | — | Network bandwidth |
| `transfer_time` | `transfer_time(size, scope, net)` | `transfer {size} across {scope} {net}` | Data transfer time |
| `read` | `read(size, storage)` | `read {size} from {storage}` | Storage read time (ssd, nvme, hdd) |
| `seek` | `seek(storage)` | — | Storage access latency |
| `compress` | `compress(size, algo)` | `compress {size} using {algo}` | Compressed size estimate |
| `downtime` | `downtime(avail, period)` | — | Downtime from availability percentage |

### Growth

| Function | Signature | NL Form | Description |
|----------|-----------|---------|-------------|
| `compound` | `compound(P, rate, periods)` | `compound {P} by {rate} over {periods}` | Compound growth |
| `grow` | `grow(amount, incr, periods)` | `grow {amount} by {incr} over {periods}` | Linear growth |
| `depreciate` | `depreciate(val, rate, periods)` | `depreciate {val} by {rate} over {periods}` | Declining balance depreciation |

### Unit Handling in Functions

Same units are preserved. Mixed units drop to Number:

```
avg($100, $200, $300)           -> $200.00  (same unit)
avg($100, €200)                 -> 150      (mixed -> Number)
sqrt($100)                      -> $10.00   (single unit preserved)
```

---

## Keywords

### Reserved Keywords

These cannot be used as variable names.

**Conversion & Formatting:** `in`, `as`, `napkin`, `precise`
**Rate & Capacity:** `per`, `over`, `at`, `with`, `of`
**Date:** `from`, `today`, `tomorrow`, `yesterday`
**Logical:** `and`, `or`, `not`
**Functions:** `avg`, `sum`, `sqrt`, `number` (and their NL forms)
**Future:** `if`, `then`, `else`, `elif`, `end`, `for`, `while`, `return`, `break`, `continue`, `let`, `const`

### Contextual Keywords

Not reserved (can be variable names) but consumed in natural language contexts:

`by`, `compounded`, `to`, `using`, `across`

---

## Frontmatter

YAML frontmatter between `---` delimiters at the start of a document. All directives are optional.

### exchange

Define currency conversion rates. Keys use `FROM_TO` format with 3-letter ISO 4217 codes.

```yaml
exchange:
  USD_EUR: 0.92
  GBP_USD: 1.27
```

### globals

User-defined variables available to the document body. Values are CalcMark expressions.

```yaml
globals:
  tax_rate: 0.32
  base_price: $100
```

Referenced in expressions with `@globals.name`:

```
tax = income * @globals.tax_rate
```

### scale

Multiply quantity results by a factor. Requires `unit_categories` to specify which categories are affected.

```yaml
scale:
  factor: 2
  unit_categories: [Mass, Volume]
```

A bare `scale: 2` sets the factor (accessible via `@scale`) but scales nothing unless `unit_categories` is specified.

Valid categories: `Acceleration`, `All`, `Area`, `Currency`, `Custom`, `DataSize`, `Energy`, `Force`, `Frequency`, `Impulse`, `Length`, `Mass`, `Number`, `Power`, `Pressure`, `Speed`, `Temperature`, `Volume`.

### convert_to

Convert quantity results to a target measurement system. Applied after scale.

```yaml
convert_to: si          # or imperial
```

Map form with category filtering:

```yaml
convert_to:
  system: imperial
  unit_categories: [Length, Volume]
```

Rules:
- Quantities already in the target system are unchanged
- Explicit `in` conversions override `convert_to`
- Custom units are unaffected
- Frequency is unaffected (hertz is universal)
- Rates have their amount converted, time denominator unchanged

### measurement

Configure how ambiguous unit names are interpreted. All axes are independent.

```yaml
measurement:
  volume: imperial      # us (default) or imperial
  mass: troy            # standard (default) or troy
  ton: long             # short (default), long, or metric
  strict: false         # annotate ambiguous units in output (default true)
```

### @Directive References

```
per_unit = total / @scale           # scale factor
tax = income * @globals.tax_rate    # named global
```

### Template Variables

Embed calculated values in markdown prose:

```
Total revenue: {{revenue}}
```

After evaluation, `{{variable}}` is replaced with the formatted value.

---

## Display & Formatting

### as napkin

Round to 2 significant figures and normalize to a human-friendly unit. Prefixed with `~`.

```
bulk_flour = 20 cups as napkin      -> ~1.25 gal
```

Sets `is_approximate: true` in JSON output.

### as precise

Show full decimal precision, skip display rounding.

```
third = 1 / 3 as precise           -> 0.333333333333333
```

### as % of

Calculate what percentage one value is of another:

```
$100 as % of $500                   -> 20%
```

---

## Validation & Diagnostics

### Diagnostic Codes

| Code | Severity | Description |
|------|----------|-------------|
| `syntax_error` | Error | Invalid syntax |
| `undefined_variable` | Error | Variable used before definition |
| `type_mismatch` | Error | Incompatible types in operation |
| `division_by_zero` | Error | Division or modulus by zero |
| `incompatible_units` | Error | Cannot convert between different unit categories |

### Security Limits

- Maximum identifier length: 256 characters
- Maximum nesting depth: 50 levels
- Maximum token count per line: configurable
- Fraction denominator limit: 10^9 (auto-converts to decimal)
- Fraction numerator limit: 10^18 (auto-converts to decimal)

---

## References

- Implementation: `github.com/CalcMark/go-calcmark`
- Spec directory: `spec/` (types, units, lexer, parser, features)
- Implementation: `impl/` (interpreter, formatters)
- Golden tests: `testdata/` (`.cm` files with expected behavior)
- Feature registry: `spec/features/registry.go`
- Canonical units: `spec/units/canonical.go`

---

**End of Language Specification**
