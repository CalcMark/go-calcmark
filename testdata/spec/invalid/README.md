# Invalid CalcMark Syntax

This directory contains files that represent **syntactically invalid** CalcMark expressions. Each file contains a single expression that the parser must reject.

## Why one expression per file?

CalcMark is a mixed markdown/calculation format. The parser treats markdown headings and prose as valid non-calculation content. This means a file with markdown followed by invalid calculation code would still parse successfully (the markdown parses fine, and the invalid calculation is isolated).

To properly test parse failures, each file contains only the invalid expression itself.

## Directory structure

- `syntax/` - Basic syntax errors (missing operands, unclosed parens)
- `features/` - Feature-specific syntax errors (invalid time units, missing logical operands)

## Files

### syntax/
- `missing_right_operand.cm` - Expression without right operand: `1 +`
- `unclosed_paren.cm` - Parenthesis not closed: `(1 + 2`

### features/
- `missing_and_operand.cm` - AND without right operand: `true and`
- `missing_not_operand.cm` - NOT without operand: `not`
- `invalid_time_unit.cm` - Invalid rate time unit: `100 MB per fortnight`
