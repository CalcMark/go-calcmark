# CalcMark Go Implementation

This directory contains the **Go-specific implementation** of CalcMark. This is one possible way to evaluate CalcMark code - other implementations in other languages are equally valid as long as they conform to `/spec`.

## Architecture Principle

**Implementation packages MAY depend on spec packages** (`/spec`) but never the reverse.

```
/impl/evaluator  →  imports  →  /spec/parser, /spec/ast, /spec/lexer
/spec/parser     →  NEVER imports  →  /impl/evaluator
```

This ensures the language specification remains implementation-agnostic.

## Packages

### `/evaluator` - Execution Engine
Executes CalcMark AST nodes and computes results. Features:
- Line-by-line evaluation with context accumulation
- Variable binding and lookup
- Type-safe operations
- Error reporting with source locations
- Boolean keyword resolution (`true`, `yes`, `t`, `y`)

### `/types` - Runtime Type System
Go-specific type system for CalcMark values:
- `Number` - Arbitrary precision decimal numbers
- `Currency` - Currency values with symbols ($, €, £, ¥)
- `Boolean` - Boolean values
- Type conversion and arithmetic operations
- Source format preservation (displays as user typed)

Uses `github.com/shopspring/decimal` for precise decimal arithmetic.

### `/wasm` - WebAssembly Bindings
WebAssembly wrapper exposing CalcMark functionality to JavaScript:
- `tokenize(source)` - Tokenization
- `parse(source)` - Parsing to AST
- `evaluate(source, preserveContext)` - Evaluation
- `validate(source)` - Semantic validation
- `classify(source)` - Line classification

### `/cmd/calcmark` - CLI Tool
Command-line interface for CalcMark:

```bash
# Build and output WASM + JS glue code
calcmark wasm [output-dir]

# Future: Interactive REPL, file evaluation, etc.
```

## Import Paths

```go
import "github.com/CalcMark/go-calcmark/impl/evaluator"
import "github.com/CalcMark/go-calcmark/impl/types"
```

## Using the Go Implementation

### As a Library

```go
import (
    "github.com/CalcMark/go-calcmark/impl/evaluator"
    "github.com/CalcMark/go-calcmark/spec/parser"
)

// Parse CalcMark source
nodes, err := parser.Parse("x = 100\ny = x * 2")
if err != nil {
    // Handle parse error
}

// Evaluate
ctx := evaluator.NewContext()
results, err := evaluator.EvaluateNodes(nodes, ctx)
if err != nil {
    // Handle evaluation error
}

// results[0] is x = 100
// results[1] is y = 200
```

### Building WASM

```bash
cd impl/wasm
GOOS=js GOARCH=wasm go build -o calcmark.wasm
```

Or use the CLI:

```bash
calcmark wasm ./output
```

This outputs:
- `calcmark-{version}.wasm` - Compiled WebAssembly module (e.g., `calcmark-0.1.1.wasm`)
- `wasm_exec.js` - Go's WASM JavaScript glue code

### Running Tests

```bash
# Test implementation packages
go test ./impl/...

# Test everything
go test ./...
```

## Design Decisions

### Why Arbitrary Precision?
Uses `decimal.Decimal` instead of `float64` to avoid floating-point errors in financial calculations:
- `0.1 + 0.2 == 0.3` ✓ (not `0.30000000000000004`)
- Preserves decimal places as entered by user

### Why Separate Type System?
The `/impl/types` package is Go-specific. Other implementations might:
- Use native BigDecimal (Java)
- Use Decimal (Python)
- Use different numeric representations entirely

The spec doesn't mandate a specific numeric implementation.

### Why Context Accumulation?
Evaluation is line-by-line with forward-only references:
```
x = 5      # x now in context
y = x + 2  # ✓ valid, x is defined
z = w + 1  # ✗ error, w is undefined
```

This matches the "document" metaphor where you read top-to-bottom.

## Relationship to Spec

```
/spec/                  ← Defines WHAT CalcMark is
  /lexer, /parser, /ast

/impl/                  ← Defines HOW we implement it in Go
  /evaluator, /types

The spec is the contract.
The impl is one implementation of that contract.
```

Other languages can provide their own `/impl` that conforms to `/spec`.
