# go-calcmark

Go implementation of the CalcMark calculation language.

## Overview

This is the core CalcMark language library that provides:
- **Types** - Number, Currency, Boolean with arbitrary precision
- **Lexer** - Unicode-aware tokenization
- **Parser** - Recursive descent parser with operator precedence
- **Evaluator** - Expression evaluation with context
- **Validator** - Semantic validation with diagnostics
- **Classifier** - Line classification (calculation vs markdown)

## Installation

```bash
go get github.com/CalcMark/go-calcmark
```

## Usage

```go
import (
    "fmt"
    "github.com/CalcMark/go-calcmark/evaluator"
)

func main() {
    content := "salary = $5000\nrent = $1500\nsavings = salary - rent"

    context := evaluator.NewContext()
    results, err := evaluator.Evaluate(content, context)
    if err != nil {
        panic(err)
    }

    for _, result := range results {
        fmt.Println(result.String())
    }
}
```

## Development

### Run Tests
```bash
go test ./...
```

### Test with Coverage
```bash
go test -cover ./...
```

### Run Specific Package Tests
```bash
go test ./evaluator -v
go test ./parser -v
```

## Package Structure

```
go-calcmark/
├── types/       # Core types (Number, Currency, Boolean)
├── lexer/       # Tokenization
├── ast/         # Abstract syntax tree
├── parser/      # Recursive descent parser
├── evaluator/   # Expression evaluation
├── validator/   # Semantic validation
└── classifier/  # Line classification
```

## Test Coverage

- **types**: 14 tests
- **lexer**: 25 tests
- **parser**: 23 tests
- **evaluator**: 25 tests
- **validator**: 32 tests
- **classifier**: 27 tests

**Total: 146 tests** ✅

## Dependencies

- `github.com/shopspring/decimal` - Arbitrary precision decimals (only external dependency)
- Go standard library

## Documentation

### Language Specifications

- **[SEMANTIC_VALIDATION.md](SEMANTIC_VALIDATION.md)** - Semantic validation rules and diagnostics
- **[LINE_CLASSIFICATION_SPEC.md](LINE_CLASSIFICATION_SPEC.md)** - Line classification rules (calculation vs markdown)
- **[DOCUMENT_MODEL_SPEC.md](DOCUMENT_MODEL_SPEC.md)** - Document model and API specification

These specifications define the CalcMark language behavior and implementation details.

## License

Same as CalcMark/CalcDown project.

## Contributing

This library is used by:
- [CalcMark Server](https://github.com/CalcMark/server) - HTTP API server
- [CalcMark Web](https://github.com/CalcMark/calcmark) - Web application

When making changes, ensure all tests pass:
```bash
go test ./...
```
