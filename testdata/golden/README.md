# CalcMark Golden Test Files

This directory contains **golden files** - definitive examples of valid and invalid CalcMark documents.

## Purpose

Golden files serve as:
1. **Living specification** - Anyone can read these to understand CalcMark
2. **Regression tests** - Ensure parser changes don't break existing features
3. **Documentation** - Real-world examples of CalcMark usage
4. **Integration tests** - Test full documents, not just isolated expressions

## Structure

```
golden/
├── valid/              # Documents that MUST parse successfully
│   ├── documents/      # Document-level tests (TextBlock + CalcBlock)
│   ├── expressions/    # Expression-level tests
│   └── features/       # Feature-specific tests
└── invalid/            # Documents that MUST fail to parse
    ├── syntax/         # Syntax errors
    └── semantic/       # Semantic errors (parse OK, semantic fail)
```

## Valid Documents

### `valid/documents/`
Tests for **document-level structure** - ensures TextBlock and CalcBlock detection works correctly.

- `mixed_content.cm` - Text and calculations interleaved
- `markdown_heavy.cm` - Primarily markdown with some calculations
- `calc_heavy.cm` - Primarily calculations with some text
- `empty_lines.cm` - Various empty line scenarios
- `block_boundaries.cm` - Two empty lines create block boundaries

### `valid/expressions/`
Tests for **calculation expressions** - ensures all calculation syntax parses.

- `arithmetic.cm` - All arithmetic operators
- `comparisons.cm` - All comparison operators
- `assignments.cm` - Variable assignments
- `precedence.cm` - Operator precedence

### `valid/features/`
Tests for **specific CalcMark features**.

- `functions.cm` - All function syntaxes (avg, sqrt, average of, etc.)
- `currency.cm` - Currency literals and arithmetic
- `quantities.cm` - Quantities with units
- `dates.cm` - All date formats
- `durations.cm` - Duration expressions
- `date_arithmetic.cm` - Date + duration operations
- `natural_language.cm` - Natural language syntax

## Invalid Documents

### `invalid/syntax/`
Documents with **syntax errors** that should fail to parse.

- `incomplete_expressions.cm` - "1 +", "+ 2", etc.
- `invalid_functions.cm` - avg(), sqrt(1,2), etc.
- `mismatched_parens.cm` - Unclosed parentheses

### `invalid/semantic/`
Documents that parse but have **semantic errors**.

- `undefined_variables.cm` - Using undefined variables
- `type_errors.cm` - Incompatible operations
- `invalid_dates.cm` - Feb 30, Dec 32, etc.

## File Format

Each `.cm` file contains:
1. **Comments** (markdown) explaining what it tests
2. **Test cases** - actual CalcMark code
3. **Expected behavior** - comments indicating expected results

Example:
```markdown
# Function Syntax Test

This tests all valid function call syntaxes.

## Traditional Syntax

avg(10, 20, 30)
sqrt(16)

## Natural Language Syntax

average of 10, 20, 30
square root of 16
```

## Running Tests

```bash
# Run all golden file tests
go test ./spec/parser -run Golden

# Run specific category
go test ./spec/parser -run Golden/valid
go test ./spec/parser -run Golden/invalid

# Verbose output
go test ./spec/parser -v -run Golden
```

## Adding New Golden Files

1. Create `.cm` file in appropriate directory
2. Add markdown comments explaining the test
3. Include actual CalcMark code
4. Update test in `spec/parser/golden_test.go` if needed

## Maintenance

- **When adding features:** Add golden file examples
- **When fixing bugs:** Add regression test golden file
- **When changing behavior:** Update existing golden files and document changes
