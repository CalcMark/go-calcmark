# CalcMark Language Specification

This directory contains the **implementation-agnostic** CalcMark language specification. These packages define what CalcMark is as a language and can be used as reference for any implementation in any programming language (Python, JavaScript, Rust, etc.).

## Architecture Principle

**The spec packages should NEVER depend on implementation packages** (`/impl`).

This separation ensures that:
- The language definition remains pure and portable
- Other language implementations can use this as authoritative reference
- The grammar and semantics are clearly documented through code
- Implementation details don't leak into language design decisions

## Packages

### `/lexer` - Tokenization
Converts raw CalcMark source text into tokens. Handles:
- Numbers (with separators like `1,000`)
- Quantities (numbers with units like `$100`, `100 cm`)
- Booleans (`true`, `false`, `yes`, `no`)
- Identifiers (variable names, including Unicode and emoji)
- Operators (`+`, `-`, `*`, `/`, `%`, `^`, etc.)
- Keywords (`if`, `then`, `else`, `let`, `const`, etc.)

### `/parser` - Grammar & AST Construction
Parses token streams into Abstract Syntax Trees. Implements:
- Expression parsing with operator precedence
- Statement parsing (assignments, expressions)
- AST node construction
- Syntax error reporting with precise locations

### `/ast` - AST Node Definitions
Defines the structure of Abstract Syntax Tree nodes:
- `Assignment` - Variable assignments (`x = 5`)
- `BinaryOp` - Binary operations (`5 + 3`)
- `UnaryOp` - Unary operations (`-5`)
- `Literal` - Literal values (numbers, booleans, quantities)
- `Identifier` - Variable references

All nodes include `Range` information for error reporting.

### `/validator` - Semantic Validation
Validates AST semantics without executing code:
- Undefined variable detection
- Type checking (where applicable)
- Diagnostic generation (ERROR, WARNING, HINT)
- Blank line style hints

### `/classifier` - Line Classification
Determines whether a line is:
- `CALCULATION` - Valid CalcMark calculation
- `MARKDOWN` - Markdown text
- `BLANK` - Empty line

Context-aware: whether `total` is a calculation depends on if `total` is defined.

## Specification Documents

### `LANGUAGE_SPEC.md`
Complete language specification defining syntax, semantics, type system, and operator precedence.

### `UNITS_DESIGN.md`
Design document for units and quantities system (currency, measurements, etc.).

### `SYNTAX_HIGHLIGHTER_SPEC.json`
Machine-readable syntax specification for editor integrations.

## Generating EBNF Grammar

To generate an EBNF (Extended Backus-Naur Form) grammar from the current specification:

```bash
cmspec > calcmark.ebnf
```

The EBNF is **dynamically generated** by introspecting the actual lexer and parser implementation at runtime, ensuring it stays in sync with the code.

This allows you to:
- Diff the intended spec against the implemented spec
- Generate railroad diagrams
- Validate spec consistency
- Share the grammar with other implementers

The grammar generation logic is in `/spec/grammar/ebnf.go`, which introspects the token types and operator precedence from the actual implementation.

## Import Paths

```go
import "github.com/CalcMark/go-calcmark/spec/lexer"
import "github.com/CalcMark/go-calcmark/spec/parser"
import "github.com/CalcMark/go-calcmark/spec/ast"
import "github.com/CalcMark/go-calcmark/spec/validator"
import "github.com/CalcMark/go-calcmark/spec/classifier"
```

## For Implementation Authors

If you're implementing CalcMark in another language:

1. **Study the spec packages** - They define the authoritative behavior
2. **Run the tests** - They document expected behavior comprehensively
3. **Generate EBNF** - Use `calcmark-spec` to get the formal grammar
4. **Don't depend on `/impl`** - Those are Go-specific implementation details

The Go implementation in `/impl` is just one possible way to evaluate CalcMark. Your implementation might:
- Use different numeric types
- Have different performance characteristics
- Support additional features
- Target different platforms

As long as it conforms to the spec defined here, it's a valid CalcMark implementation.

## Relationship to Go Implementation

```
/spec/                     ← Language specification (implementation-agnostic)
  /lexer, /parser, /ast   ← Core language packages
  LANGUAGE_SPEC.md        ← What CalcMark means

/impl/                     ← Go implementation (one possible implementation)
  /evaluator              ← How we evaluate in Go
  /types                  ← Go runtime type system
  /wasm                   ← WebAssembly bindings
```

## Version

Current version: **1.0.0** (draft, implementation in progress)
