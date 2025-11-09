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

### Basic Evaluation

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

### Validation with Diagnostics

```go
import (
    "fmt"
    "github.com/CalcMark/go-calcmark/evaluator"
    "github.com/CalcMark/go-calcmark/validator"
)

func main() {
    content := `x = 5
y = z + 2
total = x + y`

    context := evaluator.NewContext()
    result := validator.ValidateDocument(content, context)

    // Check validation status
    if result.IsValid() {
        fmt.Println("Document is valid!")
    } else {
        fmt.Printf("Found %d errors\n", len(result.Errors()))
    }

    // Process all diagnostics
    for _, diagnostic := range result.Diagnostics {
        fmt.Printf("[%s] %s: %s\n",
            diagnostic.Severity,
            diagnostic.Code,
            diagnostic.Message)

        if diagnostic.Range != nil {
            fmt.Printf("  at line %d, column %d\n",
                diagnostic.Range.Start.Line,
                diagnostic.Range.Start.Column)
        }
    }

    // Filter by severity
    if result.HasErrors() {
        fmt.Println("\nErrors:")
        for _, err := range result.Errors() {
            fmt.Printf("  - %s\n", err.Message)
        }
    }

    if result.HasWarnings() {
        fmt.Println("\nWarnings:")
        for _, warn := range result.Warnings() {
            fmt.Printf("  - %s\n", warn.Message)
        }
    }

    if result.HasHints() {
        fmt.Println("\nHints:")
        for _, hint := range result.Hints() {
            fmt.Printf("  - %s\n", hint.Message)
        }
    }
}
```

### Line Classification

```go
import (
    "fmt"
    "github.com/CalcMark/go-calcmark/classifier"
    "github.com/CalcMark/go-calcmark/evaluator"
)

func main() {
    context := evaluator.NewContext()

    lines := []string{
        "x = 5",
        "This is markdown text",
        "x + 2",
        "",
        "- bullet point",
    }

    for _, line := range lines {
        lineType := classifier.ClassifyLine(line, context)
        fmt.Printf("%s: %s\n", lineType, line)
    }
}
```

### Working with Diagnostic Codes

```go
import (
    "encoding/json"
    "fmt"
    "github.com/CalcMark/go-calcmark/validator"
)

func main() {
    content := "result = undefined_var * 2"

    result := validator.ValidateDocument(content, nil)

    for _, diagnostic := range result.Diagnostics {
        // Access structured data
        fmt.Printf("Code: %s\n", diagnostic.Code)        // "undefined_variable"
        fmt.Printf("Severity: %s\n", diagnostic.Severity) // "error"

        // Convert to map for JSON serialization
        diagMap := diagnostic.ToMap()
        jsonData, _ := json.MarshalIndent(diagMap, "", "  ")
        fmt.Println(string(jsonData))

        // Access variable name for undefined variable errors
        if diagnostic.Code == validator.UndefinedVariable {
            fmt.Printf("Undefined variable: %s\n", diagnostic.VariableName)
        }
    }
}
```

### Diagnostic Severity Levels

- **ERROR**: Invalid syntax that prevents parsing (e.g., `x * `)
  - Code: `syntax_error`
- **WARNING**: Valid syntax but evaluation failure (e.g., undefined variables)
  - Codes: `undefined_variable`, `division_by_zero`, `type_mismatch`
- **HINT**: Style suggestions for valid code (e.g., blank line isolation)
  - Code: `blank_line_isolation`

See [DIAGNOSTIC_LEVELS.md](DIAGNOSTIC_LEVELS.md) for complete details.

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

## Architecture

CalcMark processes text through a series of stages, each handled by a specialized package. Here's how the components work together:

```
┌─────────────────────────────────────────────────────────────────┐
│ Input: "sales_tax = 0.08\nsales = 1000 * sales_tax"             │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             ▼
                    ┌────────────────┐
                    │  1. LEXER      │  Breaks text into tokens
                    │  lexer/        │  "sales_tax" "=" "0.08" "\n" "sales" ...
                    └────────┬───────┘
                             │
                             ▼
                    ┌────────────────┐
                    │  2. PARSER     │  Builds Abstract Syntax Tree (AST)
                    │  parser/       │  Assignment(sales_tax, Literal(0.08))
                    │  ast/          │  Assignment(sales, BinaryOp(1000, *, sales_tax))
                    └────────┬───────┘
                             │
                ┌────────────┴────────────┐
                │                         │
                ▼                         ▼
       ┌────────────────┐        ┌────────────────┐
       │  3a. VALIDATOR │        │ 3b. EVALUATOR  │
       │  validator/    │        │  evaluator/    │
       │                │        │                │
       │ Checks AST for │        │ Executes AST   │
       │ semantic errors│        │ with Context   │
       │ (undefined vars)│        │                │
       │                │        │ Line 1:        │
       │ Returns:       │        │ sales_tax=0.08 │
       │ Diagnostics    │        │ Context: {     │
       │                │        │   sales_tax: 0.08 }
       └────────────────┘        │                │
                                 │ Line 2:        │
                                 │ 1000*0.08=80   │
                                 │ Context: {     │
                                 │   sales_tax: 0.08,
                                 │   sales: 80 }  │
                                 └────────┬───────┘
                                          │
                                          ▼
                                 ┌────────────────┐
                                 │  4. TYPES      │
                                 │  types/        │
                                 │                │
                                 │ Results stored │
                                 │ as typed values│
                                 │ Number, Currency,
                                 │ Boolean        │
                                 └────────────────┘

Additional Component:

┌────────────────┐
│  CLASSIFIER    │  Determines if a line is CALCULATION, MARKDOWN, or BLANK
│  classifier/   │  Uses parser + context to classify each line
└────────────────┘  (Used for syntax highlighting and rendering)
```

### Component Responsibilities

#### 1. **lexer/** - Tokenization
**Purpose**: Converts raw text into a stream of tokens (lexemes).

**Example**: `sales = 1000 * sales_tax` becomes:
```
[IDENTIFIER:"sales", ASSIGN:"=", NUMBER:"1000", MULTIPLY:"*", IDENTIFIER:"sales_tax", EOF]
```

**Key Features**:
- Unicode-aware (supports international characters, emojis)
- Recognizes numbers, currency ($€£¥), booleans, operators
- Tracks line/column positions for error reporting

**Entry Point**: `lexer.Tokenize(string) ([]Token, error)`

#### 2. **parser/** - Syntax Analysis
**Purpose**: Converts tokens into an Abstract Syntax Tree (AST) that represents the program structure.

**Example**: Tokens become:
```
Assignment {
  Variable: "sales"
  Value: BinaryOp {
    Left: Literal(1000)
    Op: "*"
    Right: Identifier("sales_tax")
  }
}
```

**Key Features**:
- Recursive descent parsing with precedence climbing
- Operator precedence: `()` > `^` > `*/%` > `+-` > comparisons
- Validates syntax (parentheses matching, valid expressions)
- Builds AST nodes with position information

**Entry Point**: `parser.Parse(string) ([]ast.Node, error)`

#### 3a. **validator/** - Semantic Analysis
**Purpose**: Checks if code is semantically valid WITHOUT executing it.

**What it checks**:
- **Undefined variables**: References to variables not yet defined
- **Semantic errors**: Issues that would prevent evaluation
- **Style hints**: Suggestions like blank line isolation

**Example**:
```go
sales = 1000 * sales_tax  // ERROR: sales_tax undefined
```

**Returns**: Diagnostics with severity levels:
- **ERROR**: Syntax errors (parsing failed)
- **WARNING**: Semantic errors (undefined variables)
- **HINT**: Style suggestions (missing blank lines)

**Entry Point**: `validator.ValidateDocument(string, *Context) *ValidationResult`

#### 3b. **evaluator/** - Execution
**Purpose**: Executes the AST and computes actual values.

**How it works**:
1. Processes lines sequentially
2. Maintains a **Context** (variable storage)
3. Evaluates each expression using the context
4. Updates context with assignment results

**Example Execution**:
```
Line 1: sales_tax = 0.08
  → Evaluates: 0.08
  → Context: {sales_tax: 0.08}
  → Returns: 0.08

Line 2: sales = 1000 * sales_tax
  → Evaluates: 1000 * 0.08
  → Context: {sales_tax: 0.08, sales: 80}
  → Returns: 80
```

**Key Rules**:
- Variables must be defined before use (no forward references)
- Context flows between lines
- Type checking (can't add currency + number)

**Entry Point**: `evaluator.Evaluate(string, *Context) ([]types.Type, error)`

#### 4. **types/** - Value System
**Purpose**: Represents CalcMark values with proper types.

**Types**:
- **Number**: Arbitrary precision decimals (using `shopspring/decimal`)
- **Currency**: Number + symbol ($, €, £, ¥)
- **Boolean**: true/false (keywords: true, false, yes, no, t, f, y, n)

**Example**:
```go
sales_tax := types.NewNumber(0.08)     // Number
sales := types.NewCurrency(80, "$")    // Currency
```

#### 5. **classifier/** - Line Classification
**Purpose**: Determines if a line is a calculation or markdown text.

**Classification Logic**:
1. Try to parse the line
2. Check if all variables are defined in context
3. Return: `CALCULATION`, `MARKDOWN`, or `BLANK`

**Context-Aware Example**:
```
// Empty context:
"sales_tax"  → MARKDOWN (undefined variable)

// With sales_tax defined:
"sales_tax"  → CALCULATION (valid reference)
```

**Entry Point**: `classifier.ClassifyLine(string, *Context) LineType`

#### 6. **ast/** - Abstract Syntax Tree
**Purpose**: Defines the node types that represent parsed code structure.

**Node Types**:
- `Assignment`: Variable assignment (e.g., `x = 5`)
- `BinaryOp`: Arithmetic operations (e.g., `+`, `-`, `*`, `/`)
- `ComparisonOp`: Comparisons (e.g., `>`, `<`, `==`)
- `UnaryOp`: Unary minus/plus (e.g., `-5`)
- `Identifier`: Variable reference
- `Literal`: Number, currency, or boolean value

**All nodes include**:
- `Range`: Position information (line, column) for error messages

### How Data Flows Through the System

Let's trace `sales = 1000 * sales_tax` through the entire system:

**1. Lexer Input**: `"sales = 1000 * sales_tax"`

**2. Lexer Output** (Tokens):
```
[IDENTIFIER:"sales", ASSIGN:"=", NUMBER:"1000", MULTIPLY:"*", IDENTIFIER:"sales_tax"]
```

**3. Parser Output** (AST):
```go
Assignment{
  Variable: Identifier{Name: "sales", Range: ...}
  Value: BinaryOp{
    Op: "*"
    Left: Literal{Value: Number(1000), Range: ...}
    Right: Identifier{Name: "sales_tax", Range: ...}
  }
}
```

**4a. Validator** (if sales_tax undefined):
```go
ValidationResult{
  Diagnostics: [
    {
      Severity: Warning,
      Code: "undefined_variable",
      Message: "Undefined variable: sales_tax",
      Range: {Line: 1, Column: 14},
      VariableName: "sales_tax"
    }
  ]
}
```

**4b. Evaluator** (if sales_tax = 0.08):
```go
// Looks up sales_tax in context → 0.08
// Evaluates: 1000 * 0.08 = 80
// Stores: context.Set("sales", 80)
// Returns: Number(80)
```

**5. Result**:
```
sales = 80
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

- **[SYNTAX_SPEC.md](SYNTAX_SPEC.md)** - Grammar, operators, precedence, parsing rules
- **[LINE_CLASSIFICATION_SPEC.md](LINE_CLASSIFICATION_SPEC.md)** - Line classification rules (calculation vs markdown)
- **[SEMANTIC_VALIDATION_GO.md](SEMANTIC_VALIDATION_GO.md)** - Semantic validation API and usage (Go-specific)
- **[DIAGNOSTIC_LEVELS.md](DIAGNOSTIC_LEVELS.md)** - Diagnostic severity levels (ERROR/WARNING/HINT)
- **[DOCUMENT_MODEL_SPEC.md](DOCUMENT_MODEL_SPEC.md)** - Document model and API specification
- **[CLAUDE.md](CLAUDE.md)** - Technical decisions and test coverage tracking

### Legacy Documentation

- **[SEMANTIC_VALIDATION.md](SEMANTIC_VALIDATION.md)** - Original Python-based semantic validation doc (use SEMANTIC_VALIDATION_GO.md instead)

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
