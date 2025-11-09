# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

CalcMark is a calculation language that blends calculations with markdown. This is the core Go library implementing the language parser, evaluator, and validator.

**Core Philosophy**: "Calculation by Exclusion" - lines are greedily interpreted as calculations whenever possible, falling back to markdown only when parsing fails.

## Development Commands

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run specific package tests
go test ./evaluator -v
go test ./parser -v
go test ./lexer -v
go test ./validator -v
go test ./classifier -v
go test ./types -v

# Run a single test function
go test ./evaluator -run TestSimpleAssignment -v
```

### Building

```bash
# This is a library - no build command needed
# Import with: go get github.com/CalcMark/go-calcmark
```

## Architecture

### Package Structure & Dataflow

The codebase follows a clean pipeline architecture:

```
Input (string) → Lexer → Tokens → Parser → AST → Evaluator → Result
                                        ↓
                                   Validator (parallel)
                                        ↓
                                   Diagnostics
```

**Key packages** (in execution order):

1. **lexer/** - Tokenizes CalcMark source into tokens
   - Unicode-aware identifier recognition
   - Handles numbers, currency ($€£¥), booleans, operators
   - Entry: `lexer.Tokenize(string) ([]Token, error)`

2. **parser/** - Recursive descent parser building AST
   - Precedence climbing for operators: `()` > `^` > `*/%` > `+-` > comparisons
   - Validates syntax, handles parentheses and unary operators
   - Entry: `parser.Parse(string) ([]ast.Node, error)`

3. **ast/** - Abstract syntax tree node definitions
   - All nodes have `Range` field for error positioning
   - Node types: Assignment, Expression, BinaryOp, ComparisonOp, UnaryOp, Identifier, Literal

4. **evaluator/** - Executes AST and computes results
   - Maintains `Context` with variable bindings
   - Evaluates line-by-line (forward references are errors)
   - Entry: `evaluator.Evaluate(string, *Context) ([]types.Type, error)`

5. **validator/** - Semantic validation without evaluation
   - Detects undefined variables by walking AST
   - Checks blank line isolation (style hint)
   - Returns diagnostics with severity levels (ERROR/WARNING/HINT)
   - Entry: `validator.ValidateDocument(string, *Context) *ValidationResult`

6. **classifier/** - Determines if a line is CALCULATION, MARKDOWN, or BLANK
   - Context-aware: `x` is calculation only if `x` is defined
   - Used for syntax highlighting and rendering decisions
   - Entry: `classifier.ClassifyLine(string, *Context) LineType`

7. **types/** - Core value types (Number, Currency, Boolean)
   - Uses `github.com/shopspring/decimal` for arbitrary precision
   - Currency preserves symbol ($€£¥) through calculations

### Critical Design Decisions

**Line-by-line evaluation**: Variables must be defined before use. No forward references allowed.
```go
// VALID
x = 5
y = x + 2  // x is already defined

// INVALID
a = b + 2  // ERROR: b is undefined
b = 5
```

**Context-aware classification**: Whether a line is a calculation depends on variable context.
```go
// With empty context: "total" → MARKDOWN (undefined variable)
// With total defined: "total" → CALCULATION (valid reference)
```

**Minimal type coercion**: Operations must make semantic sense.
```go
$100 + $50    // ✓ → $150
$100 * 2      // ✓ → $200
$100 + 50     // ✗ ERROR: type mismatch (currency + number)
$100 * $50    // ✗ ERROR: currency * currency nonsensical
```

**Markdown bullet detection**: Lines starting with `- ` (dash + space) are markdown, but `-50` is a negative number.
```go
// Implementation in classifier/classifier.go:29-32
if (firstChar == '-' || firstChar == '*') && stripped[1:2] == constants.Space {
    return Markdown
}
```

## Common Patterns

### Adding a New Operator

1. Add token type to `lexer/token.go`
2. Update `lexer/lexer.go` tokenization logic
3. Update parser precedence in `parser/parser.go`
4. Implement evaluation in `evaluator/evaluator.go`
5. Add tests to `parser/parser_test.go` and `evaluator/evaluator_test.go`
6. Update `SYNTAX_SPEC.md`

### Adding a New Diagnostic

1. Add diagnostic code to `validator/diagnostics.go` (DiagnosticCode constants)
2. Create diagnostic in validation logic
3. Choose severity: ERROR (blocks), WARNING (semantic), HINT (style)
4. Add test to `validator/validator_test.go`
5. Update `DIAGNOSTIC_LEVELS.md`

### Testing Strategy

All packages have comprehensive test coverage. When adding features:

1. Write test first (TDD approach)
2. Test both positive and negative cases
3. Include edge cases (empty input, Unicode, large numbers)
4. Verify error messages are helpful
5. Check position information in diagnostics

## Special Implementation Details

### Operator Precedence (parser/parser.go)

Implemented using **precedence climbing** algorithm:
- Highest: `()` parentheses
- `^` exponentiation (right-associative)
- Unary `-`, `+` (prefix)
- `*`, `/`, `%` (left-associative)
- `+`, `-` (left-associative)
- Lowest: `>`, `<`, `>=`, `<=`, `==`, `!=` (non-associative)

### Boolean Keywords

Case-insensitive support for: `true/false`, `yes/no`, `t/f`, `y/n`

Implementation in `evaluator/evaluator.go:60-73` - Context.Get() resolves boolean keywords dynamically.

### Unicode Identifiers

Fully supported: `給料 = $5000` is valid. Lexer uses Unicode-aware character detection.

### Diagnostic Severity Levels

- **ERROR**: Syntax errors that prevent parsing (e.g., `x * `)
- **WARNING**: Semantic errors in valid syntax (e.g., undefined variables)
- **HINT**: Style suggestions for valid code (e.g., blank line isolation)

See `DIAGNOSTIC_LEVELS.md` for complete decision tree.

## Testing Requirements

When modifying code:

1. All tests must pass: `go test ./...`
2. Maintain or improve coverage
3. Add tests for new error cases
4. Verify position information in error messages
5. Update test count in README.md if adding tests

## Documentation Files

- **SYNTAX_SPEC.md** - Grammar, operators, precedence rules (the source of truth)
- **LINE_CLASSIFICATION_SPEC.md** - How lines are classified as calculation vs markdown
- **DIAGNOSTIC_LEVELS.md** - Error severity levels and diagnostic codes
- **DECISIONS.md** - Technical decisions with rationale and test coverage tracking
- **DOCUMENT_MODEL_SPEC.md** - Document model API specification
- **SEMANTIC_VALIDATION_GO.md** - Validation API for Go (use instead of SEMANTIC_VALIDATION.md)

When changing behavior, update the relevant spec file.

## Integration Points

This library is used by:
- **CalcMark Server** (github.com/CalcMark/server) - HTTP API
- **CalcMark Web** (github.com/CalcMark/calcmark) - Web application

Changes here affect all downstream consumers. Maintain backward compatibility.

## Common Gotchas

1. **Trailing tokens fail parsing**: `$100 budget` has trailing text → classified as MARKDOWN
2. **Parentheses required for grouping**: Use `constants/strings.go` constants instead of magic strings like `"\n"`
3. **Context must be threaded through**: Validator and classifier need context for undefined variable detection
4. **Position information is critical**: All AST nodes and diagnostics need Range for editor integration
5. **Blank line isolation is a HINT not ERROR**: Valid calculations work without blank lines, but hints suggest adding them for readability

# Development rules

- Strictly adhere to semantic versioning for this core library
- Avoid writing duplicate documentation.
- The SYNTAX_SPEC.md is the source of truth for syntax rules.
- Document key design choices and rationale succinctly in DECISIONS.md.
- Write purely functional code with no side effects wherever possible.
- Always keep the @SYNTAX_HIGHLIGHTER_SPEC.json up to date with the syntax of CalcMark so that simple editor implementations can understand what tokens and keywords - Reserved keywords list
  - Multi-token function patterns
  - Token patterns (numbers, currency, booleans)
  - Identifier rules
  - Breaking changes