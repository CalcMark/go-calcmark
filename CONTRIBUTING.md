# Contributing to CalcMark

This guide is for developers who want to extend the CalcMark language or contribute to the interpreter implementation.

## Table of Contents

- [Getting Started](#getting-started)
- [Development Workflow](#development-workflow)
- [Architecture Overview](#architecture-overview)
- [Adding New Features](#adding-new-features)
- [Testing Guidelines](#testing-guidelines)
- [Code Style](#code-style)

## Getting Started

### Prerequisites

- Go 1.21 or later
- [Task](https://taskfile.dev/) for running development commands
- (Optional) [golangci-lint](https://golangci-lint.run/) for strict linting
- (Optional) [staticcheck](https://staticcheck.io/) for advanced analysis

### Setup

```bash
# Clone the repository
git clone https://github.com/CalcMark/go-calcmark
cd go-calcmark

# Install dependencies
task deps

# Run tests to verify setup
task test

# Run quality checks
task quality
```

## Development Workflow

### Common Tasks

```bash
# Run tests
task test

# Run specific package tests
task test:lexer
task test:parser
task test:semantic
task test:interpreter

# Run with coverage
task test:coverage

# Quality checks
task lint            # Basic formatting and vet
task vet             # Go vet only
task staticcheck     # Advanced static analysis
task quality         # All quality checks

# Benchmarks
task bench
task bench:parser
task bench:lexer

# Build
task build           # Current platform
task build:all       # All platforms
```

### Before Committing

Always run quality checks before committing:

```bash
task quality
```

This runs:
1. `go fmt` - Format code
2. `go vet` - Static analysis
3. `staticcheck` - Advanced checks

## Architecture Overview

### Key Packages

| Package | Purpose | Entry Point |
|---------|---------|-------------|
| `spec/lexer` | Tokenization | `NewLexer(text)` |
| `spec/parser` | Parsing to AST | `Parse(text)` |
| `spec/semantic` | Semantic analysis | `NewChecker().Check(nodes)` |
| `spec/document` | Document model | `NewDocument()` |
| `impl/interpreter` | Execution | `NewInterpreter().Eval(nodes)` |
| `impl/document` | Document evaluation | `NewEvaluator().Evaluate(doc)` |
| `spec/types` | Type system | `NewNumber()`, `NewCurrency()`, etc. |

### Evaluation Flow

```
Source Text
    ↓
Lexer (spec/lexer) → Tokens
    ↓
Parser (spec/parser) → AST Nodes
    ↓
Semantic Checker (spec/semantic) → Validated AST + Diagnostics
    ↓
Interpreter (impl/interpreter) → Results + Updated Environment
```

For detailed documentation on in-memory data structures, global variable scope, and
the relationship between Document, CalcBlock, Evaluator, and Environment, see
[impl/README.md](impl/README.md).

## Adding New Features

### Adding a New Operator

1. **Lexer**: Add token type in `spec/lexer/token.go`
2. **Parser**: Add parsing logic in `spec/parser/recursive_descent.go`
3. **Interpreter**: Implement evaluation in `impl/interpreter/operators.go`
4. **Tests**: Add tests at each layer
5. **Docs**: Update `spec/LANGUAGE_SPEC.md`

### Adding a New Function

1. **Lexer**: Add function keyword if needed
2. **Parser**: Handle in function call parsing
3. **Interpreter**: Implement in `impl/interpreter/functions.go`
4. **Tests**: Comprehensive test coverage
5. **Docs**: Update `spec/LANGUAGE_SPEC.md`

## Testing Guidelines

### Performance Targets

- Lexer/Parser: < 10μs for typical expressions
- Interpreter: < 50μs for multi-line programs
- Document operations: < 100μs for incremental updates

### Test Structure

Use table-driven tests:

```go
func TestFeature(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
        wantErr  bool
    }{
        {"valid case", "input", "expected", false},
        {"error case", "bad", "", true},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := Function(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("unexpected error: %v", err)
            }
            if result != tt.expected {
                t.Errorf("got %v, want %v", result, tt.expected)
            }
        })
    }
}
```

## Code Style

- Follow [Effective Go](https://golang.org/doc/effective_go.html)
- Use `gofmt` (automated via `task lint`)
- Document exported functions with godoc comments
- Write meaningful variable names
- Include usage examples in godoc

### Error Messages

Be specific and include context:

```go
// Good
return fmt.Errorf("parse error at line %d, column %d: expected ')' after function name", line, col)

// Bad
return fmt.Errorf("syntax error")
```

## Questions?

- See the [Architecture Overview](#architecture-overview) section above for system design
- See [LANGUAGE_SPEC.md](spec/LANGUAGE_SPEC.md) for language details
- See [OUTPUT_FORMATTERS.md](OUTPUT_FORMATTERS.md) for output format details
- Review existing code for patterns
