# CalcMark Language Specification

**Version:** 1.0.0
**Status:** Draft - Implementation in Progress

This is the **complete and authoritative** specification for the CalcMark language.

---

## Table of Contents

1. [Overview](#overview)
2. [Philosophy](#philosophy)
3. [Document Model](#document-model)
4. [Line Classification](#line-classification)
5. [Syntax & Grammar](#syntax--grammar)
6. [Type System](#type-system)
7. [Operators](#operators)
8. [Reserved Keywords](#reserved-keywords)
9. [Functions](#functions)
10. [Validation & Diagnostics](#validation--diagnostics)
11. [Examples](#examples)

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
- **Arbitrary precision**: Decimal math using `github.com/shopspring/decimal`

---

## Philosophy

### Calculation by Exclusion

CalcMark uses **"calculation by exclusion"** - if a line looks like markdown, it's markdown. Only unambiguous calculations are treated as calculations.

**Examples:**
```
# My Budget          → MARKDOWN (header prefix)
salary = $5000       → CALCULATION (assignment)
This is text         → MARKDOWN (natural language)
5 + 3                → CALCULATION (arithmetic)
- List item          → MARKDOWN (bullet prefix)
-5 + 3               → CALCULATION (negative number)
$100 budget          → MARKDOWN (trailing text)
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

### Processing Model

```
Input Line
    ↓
Classify (classifier/classifier.go)
    ↓
BLANK → Skip
MARKDOWN → Preserve
CALCULATION → Tokenize → Parse → Validate → Evaluate
    ↓
Result + Diagnostics
```

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

```
x = 5               → CALCULATION (assignment)
y = x + 10          → CALCULATION (x is defined)
z = unknown * 2     → MARKDOWN (unknown is undefined)
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

**Implementation:** `classifier/classifier.go`

---

## Syntax & Grammar

### EBNF Grammar

```ebnf
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

| Type | Example | Internal Representation |
|------|---------|------------------------|
| **Number** | `42`, `3.14`, `1,000` | `decimal.Decimal` |
| **Currency** | `$100`, `€50.99` | `Currency{Symbol, decimal.Decimal}` |
| **Boolean** | `true`, `false`, `yes`, `no` | `bool` |

### Type Compatibility

**Allowed operations:**

```
Number + Number → Number
Number * Number → Number
Currency + Currency (same symbol) → Currency
Currency * Number → Currency
Number * Currency → Currency
Boolean == Boolean → Boolean
Number == Number → Boolean
Currency == Currency (same symbol) → Boolean
```

**Type errors:**

```
Currency + Number → ERROR (type mismatch)
Currency * Currency → ERROR (nonsensical)
$100 + €50 → ERROR (currency mismatch)
Boolean + Number → ERROR (no boolean arithmetic)
```

### Literals

#### Numbers

```
42              ✓ Integer
3.14            ✓ Decimal
1,000           ✓ Thousands separator (comma)
1_000_000       ✓ Thousands separator (underscore)
0.5             ✓ Leading zero
.5              ✗ Must have leading zero
1.2.3           ✗ Multiple decimal points
```

#### Currency

```
$100            ✓ USD
$1,000.50       ✓ With separators
€50             ✓ EUR
£25.99          ✓ GBP
¥1000           ✓ JPY
100$            ✗ Symbol must be prefix
$ 100           ✗ No space between symbol and number
```

**Note:** Currently only `$` is fully implemented and tested. Other symbols are tokenized but may not be fully supported.

#### Booleans

Case-insensitive keywords:

```
true, false     ✓ Standard
yes, no         ✓ Natural language
t, f            ✓ Single letter
y, n            ✓ Single letter
True, FALSE     ✓ Any case
```

#### Identifiers

**Rules:**
- **BREAKING CHANGE (v1.0.0):** Spaces NOT allowed in identifiers
- Must start with letter, underscore, or Unicode character (not digit)
- Can contain letters, digits, underscores, Unicode, emoji
- Cannot be reserved keywords

```
x               ✓
salary          ✓
tax_rate        ✓ Use underscores (not spaces)
給料            ✓ Unicode (Japanese)
💰              ✓ Emoji
_private        ✓ Underscore prefix
my_var_123      ✓ Mixed
123abc          ✗ Cannot start with digit
my budget       ✗ Spaces not allowed (use my_budget)
avg             ✗ Reserved keyword
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

**Multiply aliases:** `*`, `×`, `x`, `X` (when following a number)

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

**IMPORTANT:** These words **cannot** be used as variable names.

### Logical Operators (Implemented in Phase 1)

```
and, or, not
```

Case-insensitive: `AND`, `and`, `And` all work.

**Note:** Tokens exist but parser/evaluator support coming in Phase 3.

### Control Flow (Reserved for Future)

```
if, then, else, elif, end
for, in, while
return, break, continue
let, const
```

**Status:** Reserved but not yet implemented.

### Function Names (Implemented in Phase 1)

```
avg, sqrt
```

**Status:** Tokens exist, implementation coming in Phases 6-7.

### Multi-Token Function Keywords

These sequences are combined during tokenization:

```
average of      → FUNC_AVERAGE_OF (maps to "avg")
square root of  → FUNC_SQUARE_ROOT_OF (maps to "sqrt")
```

**Examples:**
```
avg(1, 2, 3)            → Function call (Phase 7)
average of 1, 2, 3      → Same as avg (natural syntax)
sqrt(16)                → Function call (Phase 6)
square root of 16       → Same as sqrt (natural syntax)
```

**Note:** The words `average`, `square`, `root`, `of` are **not** individually reserved. Only the multi-token sequences are special.

---

## Functions

**Status:** Function infrastructure not yet implemented (Phases 5-7).

### Planned Functions

| Function | Aliases | Signature | Description |
|----------|---------|-----------|-------------|
| `avg()` | `average of` | `avg(x, y, ...)` | Average of numbers |
| `sqrt()` | `square root of` | `sqrt(x)` | Square root |

### Function Syntax

**Traditional (parentheses):**
```
avg(1, 2, 3, 4, 5)
sqrt(16)
```

**Natural language:**
```
average of 1, 2, 3, 4, 5
square root of 16
```

Both forms produce the same AST node and behavior.

### Unit Preservation

Functions preserve units:

```
sqrt($100) → $10 (not just 10)
avg($100, $200, $300) → $200
```

---

## Validation & Diagnostics

### Diagnostic Codes

| Code | Severity | Description |
|------|----------|-------------|
| `SyntaxError` | Error | Invalid syntax (lexer/parser error) |
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

### Example Diagnostics

```go
{
    Code: "UndefinedVariable",
    Severity: "warning",
    Message: "Variable 'x' is not defined",
    Line: 5,
    Column: 10,
    Suggestion: "Define 'x' before using it"
}
```

**Implementation:** `validator/diagnostics.go`

---

## Examples

### Basic Calculations

```calcmark
# Simple Math

5 + 3
10 - 2
4 * 5
20 / 4
2 ^ 3
10 % 3
```

### Variables

```calcmark
# Budget

salary = $5000
bonus = $500
expenses = $3000

net = salary + bonus - expenses
```

### Comparisons

```calcmark
# Checks

salary = $50000
threshold = $60000

is_high_earner = salary > threshold
needs_raise = salary < $40000
meets_target = salary >= $50000
```

### Complex Expressions

```calcmark
# Mortgage

principal = $200000
rate = 0.04
years = 30
months = years * 12

monthly_rate = rate / 12
payment = principal * (monthly_rate * (1 + monthly_rate) ^ months) / ((1 + monthly_rate) ^ months - 1)
```

### Mixed Markdown

```calcmark
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

---

## Implementation Status

| Feature | Status | Phase |
|---------|--------|-------|
| Basic arithmetic | ✅ Complete | - |
| Variables & assignment | ✅ Complete | - |
| Comparison operators | ✅ Complete | - |
| Unary operators | ✅ Complete | - |
| Parentheses | ✅ Complete | - |
| Currency type | ✅ Complete | - |
| Boolean type | ✅ Complete | - |
| Unicode identifiers | ✅ Complete | - |
| Reserved keywords | ✅ Tokens only | Phase 1 ✅ |
| Multi-token functions | ✅ Tokens only | Phase 1 ✅ |
| Statement/Expression distinction | ⏳ Planned | Phase 2 |
| Logical operators (and/or/not) | ⏳ Planned | Phase 3 |
| Type system enhancements | ⏳ Planned | Phase 4 |
| Function infrastructure | ⏳ Planned | Phase 5 |
| sqrt() function | ⏳ Planned | Phase 6 |
| avg() function | ⏳ Planned | Phase 7 |

---

## Breaking Changes

### Version 1.0.0

**Spaces removed from identifiers**

- **Before:** `my budget = 1000` ✓
- **After:** `my_budget = 1000` ✓
- **Rationale:** Required for multi-token function support
- **Migration:** Replace spaces with underscores in all identifier names

---

## References

- Implementation: `github.com/CalcMark/go-calcmark`
- Syntax Highlighter Spec: `SYNTAX_HIGHLIGHTER_SPEC.json`
- Test Suites: `*_test.go` files in each package

---

**End of Language Specification**
