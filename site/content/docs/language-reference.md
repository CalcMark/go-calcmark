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

1. **BLANK** - Empty or only whitespace
2. **MARKDOWN** - Has markdown prefix (`#`, `>`, `-`, `*`, `digit.`)
3. **CALCULATION** - Attempt to parse/validate:
   - Starts with literal (number, currency, boolean)
   - Contains assignment (`=`)
   - Is valid expression
   - All variables are defined (context-aware)
4. **MARKDOWN** (fallback) - Anything else

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

## Type System

### Data Types

| Type | Example | Internal |
|------|---------|----------|
| **Number** | `42`, `3.14`, `1,000` | Arbitrary-precision decimal |
| **Currency** | `$100`, `€50.99` | Symbol + decimal |
| **Boolean** | `true`, `false`, `yes`, `no` | Boolean |

### Type Compatibility

**Binary operations (preserve units):**

```cm
Number + Number -> Number
Currency + Number -> Currency  (unit preserved)
Number + Currency -> Currency  (unit preserved)
Currency + Currency (same symbol) -> Currency
Currency + Currency (different symbols) -> Number  (units dropped)
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

#### Booleans

Case-insensitive keywords:

```
true, false     Standard
yes, no         Natural language
t, f            Single letter
y, n            Single letter
True, FALSE     Any case
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

```
avg, sqrt
```

### Multi-Token Function Keywords

These sequences are combined during tokenization:

```
average of      -> avg
square root of  -> sqrt
```

Both forms produce the same result:

```cm
avg(1, 2, 3)
average of 1, 2, 3
```

---

## Functions

### Implemented Functions

| Function | Aliases | Signature | Description |
|----------|---------|-----------|-------------|
| `avg()` | `average of` | `avg(x, y, ...)` | Average of numbers (variadic) |
| `sqrt()` | `square root of` | `sqrt(x)` | Square root (single argument) |

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

---

## Examples

### Basic Calculations

```cm
# Simple Math

5 + 3
10 - 2
4 * 5
20 / 4
2 ^ 3
10 % 3
```

### Variables

```cm
# Budget

salary = $5000
bonus = $500
expenses = $3000

net = salary + bonus - expenses
```

### Comparisons

```cm
# Checks

salary = $50000
threshold = $60000

is_high_earner = salary > threshold
needs_raise = salary < $40000
meets_target = salary >= $50000
```

### Complex Expressions

```cm
# Mortgage

principal = $200000
rate = 0.04
years = 30
months = years * 12

monthly_rate = rate / 12
payment = principal * (monthly_rate * (1 + monthly_rate) ^ months) / ((1 + monthly_rate) ^ months - 1)
```

### Mixed Markdown

```cm
# My Monthly Budget

I earn a salary and get bonuses.

## Income

monthly_salary = $5000
annual_bonus = $3000
monthly_bonus = annual_bonus / 12

Total monthly income:
total_income = monthly_salary + monthly_bonus

## Expenses

- Rent: $1500
- Food: $600
- Utilities: $200

rent = $1500
food = $600
utilities = $200
total_expenses = rent + food + utilities

## Summary

Monthly surplus:
surplus = total_income - total_expenses

Can I save 20%?
savings_goal = total_income * 0.20
can_save = surplus >= savings_goal
```
