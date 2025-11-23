# CalcMark Architecture

This document provides a detailed technical architecture overview for language developers.

## System Overview

CalcMark is implemented as a pipeline of transformations:

```
Source Text → Tokens → AST → Validation → Results
```

Each stage is independent and can be used separately.

## Component Architecture

### 1. Lexer (`spec/lexer`)

**Purpose**: Convert source text into tokens

**Implementation**: Hand-written scanner with lookahead

**Key Features**:
- Unicode-aware character handling
- Number format detection (multipliers, percentages, separators)
- Currency symbol and ISO code recognition
- Reserved keyword detection
- Position tracking for error messages

**Entry Point**:
```go
lexer := lexer.NewLexer(sourceCode)
tokens, err := lexer.Tokenize()
```

**Performance**: 600ns - 1μs per expression

### 2. Parser (`spec/parser`)

**Purpose**: Convert tokens into Abstract Syntax Tree (AST)

**Implementation**: Recursive descent with precedence climbing

**Operator Precedence** (highest to lowest):
1. Parentheses `()`
2. Exponentiation `^` (right-associative)
3. Unary `-, +` (prefix)
4. Multiplicative `*, /, %` (left-associative)
5. Additive `+, -` (left-associative)
6. Comparison `>, <, >=, <=, ==, !=` (non-associative)

**Entry Point**:
```go
nodes, err := parser.Parse(sourceCode)
```

**Performance**: 1-5μs per expression

### 3. Semantic Checker (`spec/semantic`)

**Purpose**: Validate AST for semantic correctness

**Checks**:
- Undefined variables
- Type compatibility
- Unit compatibility (e.g., kg + meters → error)
- Currency code validation

**Entry Point**:
```go
checker := semantic.NewChecker()
diagnostics := checker.Check(astNodes)
```

**Output**: Structured diagnostics with severity levels

### 4. Interpreter (`impl/interpreter`)

**Purpose**: Execute AST and compute results

**Implementation**: Tree-walking interpreter

**Environment**: Variable bindings persist across evaluations

**Entry Point**:
```go
env := interpreter.NewEnvironment()
interp := interpreter.NewInterpreterWithEnv(env)
results, err := interp.Eval(astNodes)
```

**Performance**: <50μs for typical multi-line programs

### 5. Type System (`spec/types`)

**Purpose**: Represent CalcMark values

**Types**:
- `Number`: Arbitrary-precision decimals (shopspring/decimal)
- `Currency`: Number + symbol/code
- `Quantity`: Number + unit (kg, meters, etc.)
- `Date`: Calendar dates
- `Time`: Time of day with timezone
- `Duration`: Time intervals
- `Boolean`: True/false

**Interface**:
```go
type Type interface {
    String() string
}
```

### 6. Document Model (`spec/document`)

**Purpose**: Manage mixed markdown + calculation documents

**Block Types**:
- `CalcBlock`: Calculation code
- `TextBlock`: Markdown/prose

**Features**:
- Dependency tracking
- Incremental evaluation
- Block-level error isolation

## Data Flow

### Full Pipeline Example

Input:
```
x = 5
y = x + 3
```

**1. Lexer Output** (Tokens):
```
[IDENTIFIER:"x", ASSIGN:"=", NUMBER:"5", NEWLINE, 
 IDENTIFIER:"y", ASSIGN:"=", IDENTIFIER:"x", PLUS:"+", NUMBER:"3", NEWLINE, EOF]
```

**2. Parser Output** (AST):
```
[
  Assignment{Variable: "x", Value: Literal{5}},
  Assignment{Variable: "y", Value: BinaryOp{"+", Identifier{"x"}, Literal{3}}}
]
```

**3. Semantic Check**:
```
Diagnostics: [] (no errors)
```

**4. Interpreter**:
```
Environment after line 1: {x: 5}
Environment after line 2: {x: 5, y: 8}
Results: [Number(5), Number(8)]
```

## Unit Handling

### First-Unit-Wins Rule

Binary operations preserve the first operand's unit:

```
5 kg + 10 lb  → ~9.54 kg    (converts lb to kg)
$100 + €50    → ~$155      (converts EUR to USD)
10 m + 5 ft   → ~11.52 m   (converts ft to m)
```

### Function Unit Handling

Same units preserved, mixed units dropped:

```
avg($100, $200)    → $150.00   (same unit)
avg($100, €200)    → 150       (mixed → dimensionless)
avg(5 kg, 10 kg)   → 7.5 kg    (same unit)
avg(5 kg, 10 lb)   → error     (incompatible)
```

## Error Handling

### Error Philosophy

- **Fail fast**: Invalid code rejected at earliest stage
- **Specific messages**: Include context and position
- **Structured diagnostics**: Machine-readable error codes

### Error Flow

```
Lexer errors    → LexerError with line/column
Parser errors   → Parse error with position
Semantic errors → Diagnostic with code/severity
Runtime errors  → Go error with context
```

## Performance Considerations

### Design Decisions for Speed

1. **Single-pass lexing**: No backtracking
2. **Efficient token representation**: Reuse strings where possible
3. **Lazy evaluation**: Only compute what's needed
4. **Environment optimization**: HashMap for O(1) variable lookup

### Bottlenecks to Avoid

- String concatenation in loops (use `strings.Builder`)
- Reflection for type checking (use type switches)
- Unnecessary allocations (reuse buffers)

## Extension Points

### Adding Support for New Syntax

1. **Lexer**: Recognize new token patterns
2. **Parser**: Parse into AST nodes
3. **Semantic**: Add validation rules
4. **Interpreter**: Implement evaluation
5. **Types**: Add new type if needed

### Plugin Architecture

Not currently supported - go-calcmark uses static linking for performance

## Thread Safety

- **Lexer**: Safe (no shared state)
- **Parser**: Safe (no shared state)
- **Semantic**: Safe (no shared state)
- **Interpreter**: **NOT SAFE** - Environment is mutable
  - Use separate environments per goroutine
  - Or synchronize access with mutexes

## Memory Usage

### Typical Memory Footprint

- Small program (10 lines): ~10KB
- Medium program (100 lines): ~100KB
- Large program (1000 lines): ~1MB

Memory is dominated by:
- AST nodes
- Environment variable storage
- arbitary-precision decimals

## Testing Strategy

### Test Pyramid

```
      /\
     /  \  Unit Tests     (90%)
    /____\
   /      \  Integration  (8%)
  /________\
 /__________\ E2E         (2%)
```

### Coverage Targets

- Lexer: 95%+
- Parser: 95%+
- Semantic: 90%+
- Interpreter: 90%+
- Types: 95%+

## References

- [LANGUAGE_SPEC.md](spec/LANGUAGE_SPEC.md): Language specification
- [CONTRIBUTING.md](CONTRIBUTING.md): Development guide
- Source code: Inline godoc comments
