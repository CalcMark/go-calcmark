---
title: "Language Reference"
summary: "Formal specification for the CalcMark language."
weight: 30
---

**Version:** 1.0.0

This is the complete and authoritative specification for the CalcMark language.

---

- [Overview](#overview)
- [Philosophy](#philosophy)
- [Document Model](#document-model)
- [Frontmatter](#frontmatter) — [@Directive References](#directive-references) — [Template Interpolation](#template-interpolation)
- [Line Classification](#line-classification)
- [Syntax & Grammar](#syntax--grammar)
- [Type System](#type-system)
- [Operators](#operators)
- [Reserved Keywords](#reserved-keywords)
- [Functions](#functions)
- [Natural Language Syntax](#natural-language-syntax)
- [Napkin Math](#as-napkin)
- [Precise Display](#as-precise)
- [Rates](#rates) — [Rate Arithmetic Widening](#rate-arithmetic-widening)
- [Date Arithmetic](#dates)
- [Network Functions](#network)
- [Storage Functions](#storage)
- [Growth Functions](#growth)
- [Validation & Diagnostics](#validation--diagnostics)

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

```calcmark
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

## Frontmatter

A CalcMark document can begin with a YAML frontmatter block delimited by `---`. Frontmatter defines document-level configuration that is available to all calculations.

{{< feature-table category="frontmatter" >}}

### Exchange Rates

Define currency conversion rates using `FROM_TO: rate` format (underscore separator):

```yaml
---
exchange:
  USD_EUR: 0.92
  EUR_GBP: 0.86
  USD_GBP: 0.79
  GBP_USD: 1.27
---
```

Rates are not automatically reversed. If you define `USD_EUR`, you must also define `EUR_USD` to convert in the other direction.

### Global Variables

Define values available throughout the document:

```yaml
---
globals:
  tax_rate: 0.32
  base_price: $100
  start_date: Jan 15 2025
  bandwidth: 100 MB/s
---
```

Globals support all CalcMark literal types (numbers, currencies, quantities, dates, durations, rates, booleans, percentages). Expressions like `1 + 1` are not allowed -- only literal values.

### Scale

Multiply all quantity results by a factor. Applied after evaluation, before display.

```yaml
---
scale: 2
---
```

Scale accepts a number or a map with `factor` and optional `unit_categories`:

```yaml
---
scale:
  factor: 4
  unit_categories: [Length, Mass]
---
```

**Rules:**

- Scaling is **explicit**: you must specify `unit_categories` for any scaling to occur. A bare `scale: 2` sets the factor but scales nothing.
- **Quantities** in listed categories are multiplied by the factor
- **Currency** scales only when `Currency` is listed in `unit_categories`
- **Number** (unitless values) scales only when `Number` is listed in `unit_categories`
- **Boolean**, **Date**, **Duration**, and **Rate** are always immune to scale
- The special keyword `All` matches every category: `unit_categories: [All]`
- Expressions containing `@scale` are exempt from scaling to prevent double-scaling

Valid categories: `All`, `Area`, `Currency`, `Custom`, `DataSize`, `Energy`, `Length`, `Mass`, `Number`, `Power`, `Speed`, `Temperature`, `Volume`.

### @Directive References {#directive-references}

Use `@scale` and `@globals.name` to reference frontmatter values in expressions:

```text
per_unit = total_cost / @scale
tax = income * @globals.tax_rate
```

**`@scale`** resolves to the numeric scale factor from frontmatter. Requires `scale:` to be defined.

**`@globals.name`** resolves to the typed value of a named global. Requires `globals:` with that key.

**Validation rules:**

| Reference | Valid when | Error otherwise |
|-----------|-----------|-----------------|
| `@scale` | `scale:` defined in frontmatter | `@scale requires 'scale:' in frontmatter` |
| `@globals.name` | `globals:` has `name` key | `undefined global 'name'; defined globals: ...` |
| `@globals` (no field) | Never | Parser error: `@globals requires a field name` |
| `@globals.a.b` | Never | Parser error: nested dots not supported |
| `@exchange`, `@convert_to`, `@foo` | Never | `not a supported directive; use @scale or @globals.name` |

`@scale` always resolves to a `Number`. `@globals.name` resolves to whatever type the global is (Number, Currency, Quantity, etc.).

### Convert To

Convert quantity results to a target measurement system. Applied after scale.

```yaml
---
convert_to: si
---
```

Valid systems: `si` (metric) and `imperial` (US customary). Accepts a string or a map with `system` and optional `unit_categories`:

```yaml
---
convert_to:
  system: imperial
  unit_categories: [Length, Volume]
---
```

**Rules:**

- Quantities already in the target system are unchanged
- Explicit `in` conversions override `convert_to` (the user chose the unit)
- Custom units (e.g., `eggs`, `servers`) have no system mapping and are not converted
- Currency, numbers, and other non-quantity types are unaffected
- Rates have their amount converted, leaving the time denominator unchanged

Valid categories: `All`, `Area`, `Currency`, `Custom`, `DataSize`, `Energy`, `Length`, `Mass`, `Number`, `Power`, `Speed`, `Temperature`, `Volume`.

### Measurement Conventions

Some unit names are ambiguous — a "gallon" in the US (3.785 L) is different from a "gallon" in the UK (4.546 L). Similarly, an "ounce" of gold (troy, 31.10g) differs from an "ounce" of flour (standard, 28.35g).

CalcMark defaults to US Customary definitions for backwards compatibility. Use `measurement:` to declare which conventions your document uses:

```yaml
measurement:
  volume: imperial    # gallon, pint, fl oz → Imperial (UK) definitions
  mass: troy          # ounce, pound → troy (precious metals) definitions
  ton: long           # ton → Imperial long ton (2240 lb)
```

Each axis is independent — only specify axes that differ from the US defaults.

| Axis | Options | Default | Affected Units |
|------|---------|---------|---------------|
| `volume` | `us`, `imperial` | `us` | gallon, quart, pint, fl oz, cup |
| `mass` | `standard`, `troy` | `standard` | ounce, pound |
| `ton` | `short`, `long`, `metric` | `short` | ton |

**What does "standard" mass mean?** Standard is the avoirdupois system — the everyday weight you encounter at the grocery store and on bathroom scales (1 oz = 28.35g, 1 lb = 16 oz). Troy weight is used for precious metals and gemstones (1 troy oz = 31.10g, 1 troy lb = 12 troy oz).

#### Inline Qualifiers

Override the document convention for a single expression using explicit prefixes:

```
gold = 10 troy oz          # Always troy, regardless of frontmatter
milk = 2 imp pt            # Always imperial pint
shipping = 5 short ton     # Always US short ton
```

Available prefixes: `us`, `imp`/`imperial`, `troy`, `short`, `long`, `metric`.

#### Strict Mode

By default, when `measurement:` is present, the formatter annotates bare ambiguous units in output so readers always know which definition is active:

| Source | Frontmatter | Output |
|--------|-------------|--------|
| `2 oz` | (none) | `2 us oz` |
| `2 oz` | `mass: troy` | `2 troy ounce` |
| `2 troy oz` | (any) | `2 troy oz` (already explicit) |

Set `strict: false` to suppress annotation:

```yaml
measurement:
  volume: imperial
  strict: false       # bare units display without prefix
```

#### Precedence

`config.toml` (global default) < frontmatter (per-document) < inline qualifier (per-expression).

#### Interaction with Other Directives

- **`convert_to`**: Independent. `measurement` controls input interpretation (which gallon definition is used). `convert_to` controls output system (convert everything to SI or Imperial). They compose cleanly.
- **`scale`**: Unaffected. Scale multiplies values; measurement resolves unit identities.
- **`locale`**: Independent. Locale controls number formatting (decimal/thousand separators), not unit definitions.

All three directives compose. Here's how `1 gallon` flows through each combination:

| Directives | Result | What happened |
|-----------|--------|---------------|
| (none) | `1 gal` | US gallon, default display |
| `measurement: { volume: imperial }` | `1 imperial gallon` | Resolved as imperial |
| `convert_to: si` | `3.79 l` | US gallon → liters |
| `measurement: { volume: imperial }` + `convert_to: si` | `4.55 l` | Imperial gallon → liters |
| `measurement: { volume: imperial }` + `scale: 3` + `convert_to: si` | `13.6 l` | Imperial gallon × 3 → liters |
| `1 us gal` (inline) + `measurement: { volume: imperial }` + `convert_to: si` | `3.79 l` | Inline qualifier overrides convention |
| `1 gallon in ml` (explicit) + `convert_to: si` | `3,785 ml` | Explicit `in` skips `convert_to` |

### Transform Order

When frontmatter directives are present, they apply in this order:

1. **Resolve** measurement conventions (bare unit names mapped to qualified forms)
2. **Evaluate** all expressions
3. **Scale** quantity results
4. **Convert** to target measurement system
5. **Annotate** ambiguous units in output (strict mode)

See the [Recipe Scaling](/docs/examples/recipe-scaling/) example for a complete walkthrough, and the [User Guide — Frontmatter](/docs/user-guide/#frontmatter) for a gentler introduction.

### Template Interpolation {#template-interpolation}

Template variables embed calculated values into prose. After all calculations are evaluated, `{{variable_name}}` tags in text blocks are replaced with display-formatted results. Resolved values render **bold** in markdown and are wrapped in `<span class="cm-interpolated">` in HTML.

#### Forward References

The primary use case is forward references — a summary at the top of a document that pulls in values computed below:

```text
## Executive Summary

| Metric | Value |
|--------|-------|
| Revenue | {{total_rev}} |
| Gross Margin | {{gross_margin}} |
| Team | {{team_hc}} |


total_rev = $4.2M
gross_margin = 28%
team_hc = 14 people
```

The summary table renders with **$4.2M**, **28%**, and **14 people** — even though the calculations appear below the text.

#### Inline Formatting

Combine `{{var}}` with markdown formatting for emphasis, headings, and lists:

```text
The grand total is {{total_cost}}.

_Team: {{headcount}}_

## Budget: {{annual_budget}}

- Revenue: {{revenue}}
- Costs: {{costs}}
- Margin: {{margin}}
```

Backticks around tags are consumed — `` `{{var}}` `` renders as **value**, not `value`. This prevents interpolated values from appearing as inline code.

#### Tables

Interpolation works naturally in markdown tables. Use it for dashboard-style summaries:

```text
| Scenario | Revenue | Margin |
|----------|---------|--------|
| Baseline | {{base_rev}} | {{base_margin}} |
| Optimistic | {{opt_rev}} | {{opt_margin}} |
| Stressed | {{stress_rev}} | {{stress_margin}} |
```

#### Syntax

`{{variable_name}}` — variable names use word characters (letters, digits, underscore). Whitespace inside braces is tolerated: `{{ total_rev }}` resolves the same as `{{total_rev}}`.

#### Rules

| Behavior | Detail |
|----------|--------|
| Resolved values | Display-formatted with locale, currency symbols, units, K/M/B suffixes |
| Markdown rendering | Resolved values are **bold**; backticks around tags are stripped |
| HTML rendering | Resolved values wrapped in `<span class="cm-interpolated">` |
| Scale / convert_to | Applied before formatting — interpolated values match CalcBlock display |
| Missing variables | `{{unknown}}` is left as-is in the output |
| Scope | Text blocks only — `{{var}}` inside a CalcBlock is a syntax error, not interpolation |
| Round-trip safety | Saving a `.cm` file preserves raw `{{var}}` tags (interpolation is render-time only) |
| Empty / expression tags | `{{}}` and `{{a + b}}` are not matched |
| Directives | `{{@scale}}` and `{{@globals.name}}` are not currently supported — use a variable alias |

**JSON output:** The JSON formatter includes both `source` (raw text with `{{var}}` tags) and `interpolated_source` (resolved text) for programmatic consumers.

See the [Services P&L](/docs/examples/services-pl/) example for a production use of interpolated summary tables, and the [Household Budget](/docs/examples/household-budget/) for inline results in prose.

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

```calcmark
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

The grammar below covers the core expression hierarchy. Natural language function syntax (described in [Natural Language Syntax](#natural-language-syntax)) is parsed at the Primary level but omitted here for clarity.

```
Statement       ::= Assignment | Expression
Assignment      ::= IDENTIFIER "=" Expression
Expression      ::= Or
Or              ::= And ("or" And)*
And             ::= Comparison ("and" Comparison)*
Comparison      ::= Additive (ComparisonOp Additive)*
ComparisonOp    ::= ">" | "<" | ">=" | "<=" | "==" | "!="
Additive        ::= Multiplicative (("+"|"-") Multiplicative)*
                     ("as" ConversionTarget)?
ConversionTarget ::= "napkin" | "precise" | UnitName
Multiplicative  ::= Exponent (("*"|"/"|"%") Exponent)*
                     ("in" UnitName)?
                     ("per" TimeUnit)?
                     ("over" Expression)?
                     ("at" Expression "per" Expression ("with" Expression)?)?
Exponent        ::= Unary ("^" Unary)*
Unary           ::= ("not" | "-" | "+")* Postfix
Postfix         ::= Primary ("%" ("of" Expression)?)?
Primary         ::= Number | Currency | Quantity | Percentage
                   | Boolean | Date | Duration | Rate
                   | DirectiveRef | Identifier
                   | FunctionCall | "(" Expression ")"
FunctionCall    ::= FunctionName "(" (Expression ("," Expression)*)? ")"
DirectiveRef    ::= "@scale" | "@globals." IDENTIFIER
```

### Operator Precedence

From **highest** to **lowest**:

1. Parentheses `()`
2. Exponentiation `^` (right-associative)
3. Unary `not`, `-`, `+` (prefix)
4. Postfix `%`, `% of`
5. Multiplicative `*`, `/`, `%` (left-associative); postfix `in`, `per`, `over`, `at`
6. Additive `+`, `-` (left-associative); postfix `as`
7. Comparison `>`, `<`, `>=`, `<=`, `==`, `!=`
8. Logical AND `and` (left-associative)
9. Logical OR `or` (left-associative)

---

## Type System {#type-system}

### Data Types

| Type | Example | Internal |
|------|---------|----------|
| **Number** | `42`, `3.14`, `1,000` | Arbitrary-precision decimal |
| **Percentage** | `50%`, `8.25%` | Fractional decimal (0.5, 0.0825) |
| **Currency** | `$100`, `€50.99` | Symbol + decimal |
| **Boolean** | `true`, `false`, `yes`, `no` | Boolean |
| **Quantity** | `10 meters`, `5 kg`, `100 MB` | Value + unit |
| **Duration** | `5 days`, `2 weeks`, `1 year` | Value + time unit |
| **Rate** | `100 MB/s`, `$50/hour`, `1000 req/s` | Numerator / time unit |
| **Date** | `Jan 15 2025`, `today` | Calendar date |

### Type Compatibility

**Binary operations (preserve units):**

```calcmark
Number + Number -> Number
Currency + Number -> Currency  (unit preserved)
Number + Currency -> Currency  (unit preserved)
Currency + Currency (same symbol) -> Currency
Currency + Currency (different symbols) -> Number  (units dropped)
Quantity + Quantity (same unit) -> Quantity
Date + Duration -> Date
Date - Date -> Duration
Rate * Duration -> Quantity  (via "over" keyword)
Number * Rate -> Rate        (scaling: 3 * 10 MB/s = 30 MB/s)
Rate * Number -> Rate        (commutative)
Rate * Quantity -> Quantity   (e.g., 10 MB/s * 500 MB = 5000 MB)
Quantity * Rate -> Quantity   (commutative)
```

**Percentage widening:**

When a percentage appears in addition or subtraction with another type, it widens to a proportion of that value:

```calcmark
$100 + 15% -> $115.00        (same as $100 * 1.15)
$100 - 15% -> $85.00         (same as $100 * 0.85)
10 kg + 50% -> 15 kg         (same as 10 kg * 1.5)
15% of 200 -> 30             ("of" syntax)
```

In multiplication or standalone use, percentages behave as their decimal value (`50%` = `0.5`).

**Functions (drop units when mixed):**

```calcmark
avg($100, $200) -> $150.00  (same unit preserved)
avg($100, €200) -> 150  (Number, mixed units)
sqrt($100) -> $10.00  (single unit preserved)
```

**Type errors:**

```calcmark
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

```calcmark
10K             -> 10000
5M              -> 5000000
2B              -> 2000000000
1.5T            -> 1500000000000
1.5K            -> 1500
```

#### Scientific Notation

```calcmark
1.2e10          -> 12000000000 (displayed as 12B)
5e3             -> 5000
2.5e-2          -> 0.025
```

#### Currency

Currency literals use either a prefix symbol or a postfix ISO 4217 code.

**Symbol syntax** (prefix only):

```
$100            USD
$1,000.50       With separators
€50             EUR
£25.99          GBP
¥1000           JPY
100$            Invalid (symbol must be prefix)
$ 100           Invalid (no space between symbol and number)
```

**Supported symbols:** `$` (USD), `€` (EUR), `£` (GBP), `¥` (JPY)

**ISO code syntax** (postfix):

Any 3-letter uppercase code works as a postfix currency identifier. Codes with a corresponding symbol display with that symbol; others display with the ISO code:

```calcmark
100 USD         -> $100.00
50 EUR          -> €50.00
25 GBP          -> £25.00
1000 JPY        -> ¥1,000
100 CHF         -> 100 CHF
50 CAD          -> 50 CAD
```

Semantic validation checks whether the code is a known ISO 4217 currency. Unknown codes produce an `invalid_currency_code` diagnostic. Currency conversion between different codes requires `exchange:` rates defined in [Frontmatter](#frontmatter).

#### Percentages

Percentages are their own type, stored as a decimal fraction internally:

```calcmark
50%             -> Percentage (0.5 internally)
8.25%           -> Percentage (0.0825 internally)
15% of 200      -> 30  ("of" syntax)
```

See [Type Compatibility](#type-compatibility) for percentage widening rules.

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

```calcmark
10 meters       Quantity: 10 in meters
5 kg            Quantity: 5 in kilograms
100 MB          Quantity: 100 in megabytes
```

#### Custom Units

Any identifier following a number becomes a unit. CalcMark does not require units to be predefined — these are called custom units:

```calcmark
5 apples        Quantity: 5 apples
1000 req/s      Rate: 1000 requests per second
10 servers      Quantity: 10 servers
```

Arithmetic with matching custom units preserves the unit. Mismatched custom units produce an error:

```calcmark
5 apples + 3 apples    -> 8 apples
10 servers * 2         -> 20 servers
5 apples + 3 oranges   -> ERROR (incompatible units)
```

#### Rates

```calcmark
100 MB/s        Rate: 100 megabytes per second
$50/hour        Rate: $50 per hour
1000 req/s      Rate: 1000 requests per second
$120000/year    Rate: $120,000 per year
```

#### Dates

```calcmark
Jan 15 2025     Date literal
Dec 25 2025     Date literal
today           Current date
tomorrow        Tomorrow's date
yesterday       Yesterday's date
```

#### Durations

```calcmark
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

```calcmark
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


### Comparison

| Operator | Name | Example | Result |
|----------|------|---------|--------|
| `>` | Greater than | `5 > 3` | `true` |
| `<` | Less than | `5 < 3` | `false` |
| `>=` | Greater or equal | `5 >= 5` | `true` |
| `<=` | Less or equal | `5 <= 3` | `false` |
| `==` | Equal | `5 == 5` | `true` |
| `!=` | Not equal | `5 != 3` | `true` |

### Logical

| Operator | Name | Example | Result |
|----------|------|---------|--------|
| `and` | Logical AND | `true and false` | `false` |
| `or` | Logical OR | `true or false` | `true` |
| `not` | Logical NOT | `not true` | `false` |

Case-insensitive: `AND`, `and`, `And` all work.

### Unary

| Operator | Name | Example | Result |
|----------|------|---------|--------|
| `-` | Negation | `-5` | `-5` |
| `+` | Plus | `+5` | `5` |
| `not` | Logical NOT | `not true` | `false` |

### Conversion and Postfix

| Operator | Name | Example | Result |
|----------|------|---------|--------|
| `in` | Unit conversion | `10 meters in feet` | `32.81 feet` |
| `as` | Display modifier | `1234567 as napkin` | `~1.2M` |
| `per` | Rate denominator | `100 MB per day` | `100 MB/day` |
| `over` | Accumulation | `100 MB/s over 1 day` | `~8.64 TB` |
| `at...per` | Capacity | `10 TB at 2 TB per disk` | `5 disk` |
| `% of` | Percentage of | `15% of 200` | `30` |
| `as % of` | Ratio as percentage | `$100 as % of $500` | `20%` |
| `from` | Date offset | `2 days from today` | *(date)* |

### Assignment

| Operator | Name | Example | Effect |
|----------|------|---------|--------|
| `=` | Assign | `x = 5` | Stores 5 in variable x |

### Type Arithmetic Rules {#type-arithmetic-rules}

Not every combination of types can be multiplied or divided. CalcMark
follows **dimensional analysis**: the result of an operation must have a
meaningful type.

| Expression | Result | Why |
|-----------|--------|-----|
| `$100 + $50` | `$150` | Adding prices is natural |
| `$100 - $50` | `$50` | Subtracting prices is natural |
| `$100 * 5` | `$500` | Scaling a price by a number |
| `$100 / 5` | `$20` | Splitting a price evenly |
| `$100 * $50` | **error** | "Square dollars" isn't a real unit |
| `$100 / $50` | **error** | Dollars divided by dollars isn't dollars |
| `10 kg + 5 kg` | `15 kg` | Adding quantities is natural |
| `10 kg * 3` | `30 kg` | Scaling a quantity by a number |
| `10 kg * 5 kg` | **error** | "Square kilograms" isn't a real unit |
| `10 kg / 5 kg` | **error** | kg divided by kg isn't kg |

**To get a percentage**, use `as % of`:

```
profit_margin = operating_income as % of total_revenue
```

This returns a `Percentage` type (e.g., `41.01%`). Both values must be
the same type — you can't compute `$100 as % of 50 kg`.

For a raw number ratio, use `number()` to strip the type from both sides:

```
ratio = number(operating_income) / number(total_revenue)
```

**Which side you wrap matters.** `number()` strips the unit or currency, returning a plain number. The arithmetic rules then determine the result type:

| Expression | Result | Why |
|-----------|--------|-----|
| `$100 / number($50)` | `$2.00` | currency / number = currency |
| `number($100) / number($50)` | `2` | number / number = plain number |
| `number($100) / $50` | error | number / currency is not defined |

**Don't over-wrap.** If a value is already a plain number (from a previous `number()` call, or because it was never typed), wrapping it again is harmless but noisy. Only wrap typed values (currency, quantity) that need their type stripped.

The same rule applies to fractions with units:

| Expression | Result | Why |
|-----------|--------|-----|
| `2/3 cup + 1/4 cup` | `11/12 cup` | Adding quantities |
| `2/3 cup * 3` | `2 cup` | Scaling by a number |
| `2/3 cup * 1/3 cup` | **error** | Square cups isn't real |
| `1/3 * $200` | `$66.67` | Dimensionless fraction scales currency |

#### Fractions vs Division

CalcMark uses whitespace to disambiguate fractions from division:

- `1/3` — fraction (no spaces around `/`)
- `1 / 3` — division (spaces around `/`)
- `a/b` where `a` and `b` are variables — always division

Only literal integers without spaces produce fractions.

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
for, while
return, break, continue
let, const
```

### Function Names

All built-in function names are reserved:

```
avg, sum, sqrt, number, accumulate, convert_rate, capacity,
downtime, rtt, throughput, transfer_time,
read, seek, compress, compound, grow, depreciate
```

### Language Keywords

```
in, as, of, per, over, at, from, with, napkin, precise
```

### Contextual Keywords

These words have special meaning in specific syntactic positions but are not reserved as variable names:

```
by, compounded, to, monthly, quarterly, weekly, daily, yearly
```

```calcmark
compound $1000 by 5% monthly over 10 years         (by, monthly)
compound $1000 by 5% compounded monthly over 10    (by, compounded)
depreciate $50000 by 15% over 5 to $5000           (to)
```

Bare frequency adverbs (`monthly`, `quarterly`, `weekly`, `daily`, `yearly`) are shorthand for `compounded monthly`, `compounded quarterly`, etc.

---

## Functions {#functions}

{{< feature-table category="function" >}}

For detailed examples of every function, including natural language syntax forms, see the [User Guide: Function Reference](/docs/user-guide/#function-reference).

### Unit Handling in Functions

**Same units are preserved:**

```calcmark
avg($100, $200, $300) -> $200.00
sqrt($100) -> $10.00
```

**Mixed units are dropped:**

```calcmark
avg($100, €200) -> 150  (no units)
average of $50, €100, £150 -> 100  (no units)
```

---

## Natural Language Syntax {#natural-language-syntax}

CalcMark supports natural language forms for many functions. These are equivalent to the function-call syntax. Arguments can be literal values (`100 MB`) or variable references (`data`).

| Pattern | Equivalent |
|---------|------------|
| `average of X, Y, Z` | `avg(X, Y, Z)` |
| `sum of X, Y, Z` | `sum(X, Y, Z)` |
| `square root of X` | `sqrt(X)` |
| `X over Y` | `accumulate(X, Y)` |
| `X at Y per Z` | `capacity(X, Y, Z)` |
| `X at Y per Z with W% buffer` | `capacity(X, Y, Z, W%)` |
| `read X from Y` | `read(X, Y)` |
| `compress X using Y` | `compress(X, Y)` |
| `transfer X across Y Z` | `transfer_time(X, Y, Z)` |
| `compound X by Y% over Z` | `compound(X, Y%, Z)` |
| `compound X by Y% monthly over Z` | `compound(X, Y%, Z, monthly)` |
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

### Rate Arithmetic Widening

When a rate appears on the **right side** of `*` or `/`, its time denominator is dropped and the rate's amount is used instead. This is called **widening** — the rate widens into its underlying quantity.

When a rate appears on the **left side**, it stays a rate. This lets you scale rates naturally.

**Operand order determines the result type:**

| Expression | Left | Right | Result | Why |
|---|---|---|---|---|
| `rate * 3` | Rate | Number | **Rate** | Rate on left → stays rate (scaling) |
| `3 * rate` | Number | Rate | **Quantity** | Rate on right → widened |
| `rate / 2` | Rate | Number | **Rate** | Rate on left → stays rate |
| `100 / rate` | Number | Rate | **Number** | Rate on right → widened |
| `rate * qty` | Rate | Quantity | **Quantity** | Cross-type, extracts amount |
| `qty * rate` | Quantity | Rate | **Quantity** | Rate on right → widened |
| `rate / rate` | Rate | Rate | **Number** | Same-unit ratio (no widening) |

```text
posts_rate = 2 posts/week
scaled = posts_rate * 3           -> 6 posts/week  (Rate — rate on left)
total  = 3 * posts_rate           -> 6 posts       (Quantity — rate on right)
half   = posts_rate / 2           -> 1 posts/week  (Rate — rate on left)
```

This rule is **asymmetric by design**. The operand on the left is the "subject" of the expression:

- `read_rate * peak_multiplier` — you are scaling a rate, so the result is a rate.
- `daily_users * posts_per_user_per_week` — you are scaling a count by a rate, so the result is a quantity.

Rate widening only applies to binary `*` and `/`. It does not affect functions like `accumulate()`, `over`, or `per`.

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

**Simple compound growth** -- `A = P(1+r)^t`, compounding once per year. The 3rd argument is years:

```calcmark
compound($1000, 5%, 10)                              -> $1628.89
compound(500 customers, 20%, 12)                     -> 4458.05 customers
```

**Financial compounding** -- `A = P(1+r/n)^(nt)`, compounding multiple times per year. Triggered by a frequency adverb (`monthly`, `quarterly`, `weekly`, `daily`, `yearly`):

```calcmark
compound($1000, 5%, 10, monthly)                     -> $1647.01
compound($1000, 5%, 10, quarterly)                   -> $1643.62
compound($1000, 5%, 24 months, monthly)              -> $1103.43
```

**Natural language forms:**

```calcmark
compound $1000 by 5% over 10 years
compound $1000 by 5% monthly over 10 years
compound $1000 by 5% per month over 12 months
```

The frequency adverb controls how many times per year the rate is applied. Without it, the rate compounds once per year. With `monthly`, the annual rate is split into 12 applications per year.

### Linear Growth

```calcmark
grow($500, $100, 36)               -> $4100.00
grow 100 by 20 over 5 months       -> 200  (NL form)
```

### Depreciation

```calcmark
depreciate($50000, 15%, 5)                -> $22185.27
depreciate($50000, 15%, 20, $5000)        -> $5000.00  (salvage floor)
depreciate $50000 by 15% over 5 years     -> (NL form)
depreciate $50000 by 15% over 5 years to $5000
```

See the [User Guide: Growth Functions](/docs/user-guide/#growth-functions) for the full argument reference.

---

## Validation & Diagnostics

### Diagnostic Levels

| Severity | Meaning |
|----------|---------|
| **Error** | Prevents evaluation, line becomes MARKDOWN |
| **Warning** | Line evaluates but issue is noted |
| **Hint** | Suggestion or style recommendation |

### Diagnostic Codes

| Code | Severity | Description |
|------|----------|-------------|
| `type_mismatch` | Error | Incompatible types in operation |
| `division_by_zero` | Error | Division or modulus by zero |
| `invalid_currency_code` | Error | Unsupported currency symbol |
| `incompatible_currencies` | Error | Mixed currency codes without exchange rate |
| `incompatible_units` | Error | Unit mismatch in operation |
| `unsupported_unit` | Error | Unit not recognized for conversion |
| `invalid_date` | Error | Invalid date literal |
| `invalid_month` | Error | Month name or number out of range |
| `invalid_day` | Error | Day out of range for the given month |
| `invalid_year` | Error | Invalid year value |
| `invalid_leap_year` | Error | Feb 29 in a non-leap year |
| `invalid_date_operation` | Error | Invalid operation on date types |
| `invalid_directive` | Error | Invalid `@` directive reference |
| `undefined_global` | Error | `@globals.name` references undefined global |
| `missing_frontmatter` | Warning | Directive requires frontmatter section |
| `undefined_variable` | Warning | Variable used before definition |
| `variable_redefinition` | Warning | Variable assigned more than once |
| `mixed_base_units` | Hint | Mixing binary (KiB) and decimal (KB) data units |
