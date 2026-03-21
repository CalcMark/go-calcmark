# CalcMark Architecture

This document describes the internal architecture of go-calcmark for contributors. It covers the language specification, interpreter implementation, and all client surfaces (CLI, TUI, LSP, website).

## Overview

CalcMark is an interpreted language that blends CommonMark markdown and calculations in one document. The Go implementation is organized into four layers with strict dependency rules:

```
                    +-----------+
                    | Clients   |  CLI, TUI, LSP, Site
                    +-----+-----+
                          |
                    +-----v-----+
                    | Go Library|  calcmark.Eval(), Convert(), Session
                    +-----+-----+
                          |
              +-----------+-----------+
              |                       |
        +-----v-----+          +-----v-----+
        |   spec/   |  <----  |   impl/   |
        | Language  |         | Interpreter|
        |   Spec    |         |            |
        +-----------+         +-----+------+
                                    |
                              +-----v-----+
                              |  format/  |
                              |  Output   |
                              +-----------+
```

**Dependency rule:** `spec/` never imports `impl/`. Dependencies flow one way. This is enforced by `TestSpecNeverImportsImpl` in `spec/boundary_test.go`.

## Evaluation Pipeline

A CalcMark expression flows through five stages:

```
Source Text
    |
    v
Frontmatter Parsing -----> Exchange rates, globals, scale, convert_to
    |                       (spec/document.ParseFrontmatter)
    v
Lexer -----> []Token        70+ token types: numbers, units, currencies,
    |        (spec/lexer)   operators, keywords, dates, durations
    v
Parser -----> []ast.Node    Recursive descent with precedence climbing
    |         (spec/parser) Precedence: () > ^ > unary > */ > +- > compare
    v
Semantic -----> []Diagnostic  Type checking, variable scope, unit compatibility
    |           (spec/semantic) Typo detection, helpful suggestions
    v
Interpreter -----> []types.Type  Number, Currency, Quantity, Duration, Rate,
                   (impl/interpreter) Fraction, Percentage, Date, Boolean
```

Each stage produces a well-typed output consumed by the next. Failures at any stage produce diagnostics that propagate to the user.

## Layer 1: Language Specification (`spec/`)

The spec layer defines what CalcMark IS — its grammar, types, and validation rules. It contains zero execution logic and can be vendored or ported independently.

### Packages

| Package | Purpose | Key Types |
|---------|---------|-----------|
| `spec/ast` | AST node definitions | `Node`, `BinaryOp`, `Assignment`, `FunctionCall`, `NumberLiteral`, `QuantityLiteral`, `CurrencyLiteral`, `DirectiveRef` |
| `spec/lexer` | Hand-written tokenizer | `Token`, `TokenType` (70+ types) |
| `spec/parser` | Recursive descent parser | `RecursiveDescentParser`, `Parse()` |
| `spec/semantic` | Type checker and validator | `Checker`, `Environment`, `Diagnostic` |
| `spec/document` | Document structure model | `Document`, `CalcBlock`, `TextBlock`, `Frontmatter` |
| `spec/types` | Value type definitions | `Number`, `Currency`, `Quantity`, `Duration`, `Rate`, `Fraction`, `Percentage`, `Date`, `Boolean` |
| `spec/units` | Canonical unit mappings | `StandardUnits`, `UnitMapping` (200+ units, UCUM/NIST) |
| `spec/features` | Feature registry for IDE/help | `Registry`, `Feature`, `Suggestion`, completion functions |
| `spec/classifier` | Line classification (calc vs text) | `Classify()` |
| `spec/identifiers` | Identifier validation | `IsValid()` |
| `spec/transform` | AST transformation pipeline | Scale application, unit conversion |

### Parser Organization

The parser is split by concern across three files:

```
spec/parser/
  rdparser.go          # Infrastructure + precedence backbone (parseProgram → parseAdditive)
  multiplicative.go    # parseMultiplicative (rate syntax, per, capacity, unit conversion)
  primary.go           # parsePrimary, parseUnary, parseFunctionCall, literals
  nl_functions.go      # Natural language: "read 100 MB from ssd"
  nl_growth_functions.go  # Natural language: "compound 1000 by 5% over 10 years"
  rate_helpers.go      # Rate-specific parsing helpers
  helpers.go           # isAllUppercase, isCurrency
  adapter.go           # Public Parse() entry point
  errors.go            # ParseError, SecurityError
  limits.go            # MaxNestingDepth, MaxTokenCount
```

### Document Model

A CalcMark document is a sequence of blocks separated by empty lines:

```
---                          <-- Frontmatter (YAML)
exchange: USD_EUR = 0.92
globals:
  tax_rate: 0.08
---

# Shopping List               <-- TextBlock (Markdown)

price = $29.99                 <-- CalcBlock (calculations)
tax = price * @globals.tax_rate
total = price + tax

## Summary                     <-- TextBlock

The total is {{total}}.        <-- TextBlock with interpolation
```

- **Hard boundary**: 2+ consecutive empty lines create a new block
- **CalcBlock**: Contains parsed statements, results, variables, diagnostics
- **TextBlock**: Markdown prose, supports `{{variable}}` interpolation
- **Frontmatter**: Exchange rates, global variables, scale factors, measurement preferences

## Layer 2: Interpreter Implementation (`impl/`)

The impl layer executes AST nodes and produces typed values. It imports `spec/` freely.

### Packages

| Package | Purpose | Key Types |
|---------|---------|-----------|
| `impl/interpreter` | Expression evaluation | `Interpreter`, `Environment`, `FunctionDef` |
| `impl/document` | Document-level evaluation | `Evaluator`, `DirectiveResolver` |
| `impl/embedded` | Markdown + CalcMark scanner | `Segment`, `Scanner` |
| `impl/types` | Type conversion helpers | `NewNumber()`, `NewCurrency()` |

### Interpreter Architecture

```
impl/interpreter/
  interpreter.go       # Eval() dispatch: Node → types.Type
  operators.go          # evalBinaryOperation, evalUnaryOperation, evalComparison
  functions.go          # BuiltinFunctions registry (17 functions)
  registry.go           # Feature lookup, synonym resolution
  literals.go           # Literal evaluation (numbers, currencies, dates)
  environment.go        # Variable storage, exchange rates, constants (PI, E)

  # Domain-specific function implementations:
  growth_functions.go   # compound(), grow(), depreciate()
  network_functions.go  # rtt(), throughput(), transfer_time()
  capacity_functions.go # capacity(), capacity_at()
  storage_functions.go  # read(), seek()
  rate_functions.go     # accumulate(), convert_rate()
  availability_functions.go  # downtime()
  compression_functions.go   # compress()

  # Special evaluation:
  unit_conversion_eval.go    # "5 meters in feet"
  napkin_eval.go             # "as napkin" (human-readable rounding)
  precise_eval.go            # "as precise" (full precision)
  percentage_of_eval.go      # "20% of 500"
  fraction_ops.go            # Exact fraction arithmetic (big.Rat)
  datetime.go                # Date arithmetic, relative dates
```

### Type Dispatch

Binary operations use a two-phase pattern in `evalBinaryOperation`:

1. **Normalization**: Fraction→Number, Rate widening, Percentage widening, unitless Quantity normalization (order matters, recursive)
2. **Type dispatch**: Sequential type assertions for Number, Currency, Quantity, Duration, Date, Rate, Percentage, Boolean

### Document Evaluator

`impl/document.Evaluator` orchestrates full document evaluation:

1. Apply frontmatter (exchange rates, globals, scale, convert_to)
2. Parse and semantic-check each CalcBlock
3. Evaluate blocks in order, accumulating the Environment
4. Interpolate `{{variable}}` references in TextBlocks
5. Apply scale/convert_to transforms to results

## Layer 3: Output Formatting (`format/`)

Converts evaluated documents to various output formats.

```
format/
  formatter.go          # Formatter interface, Options
  registry.go           # Format name → Formatter lookup
  html_formatter.go     # HTML output (bluemonday sanitized)
  json_formatter.go     # Structured JSON
  markdown_formatter.go # Markdown with results annotated
  text_formatter.go     # Plain text
  calcmark_formatter.go # Source preservation

  display/
    formatter.go        # Locale-aware number formatting
    config.go           # DecimalSep, ThousandsSep per locale
    normalize.go        # Normalization utilities
    fraction_unicode.go # 1/2 → ½ (optional)
```

Supported locales: en-US, de-DE, fr-FR, es-ES, it-IT, ja-JP, zh-CN, and more.

## Layer 4: Clients

### Go Library (repo root)

The public API for Go consumers:

```go
// eval.go — Single-shot evaluation
result, err := calcmark.Eval("price = 100\ntax = price * 8%\ntotal = price + tax")
// result.Value = 108, result.AllValues = [100, 8, 108]

// session.go — Stateful evaluation (preserves variables)
s := calcmark.NewSession()
s.Eval("x = 42")
s.Eval("y = x * 2")  // y = 84

// convert.go — Format conversion
html, _ := calcmark.Convert(input, calcmark.Options{Format: "html", Mode: calcmark.Embedded})

// result.go — Result type
type Result struct {
    Value       types.Type     // Final value
    AllValues   []types.Type   // Per-line results
    Diagnostics []Diagnostic   // Errors, warnings, hints
}
```

### CLI (`cmd/calcmark/`)

Built with Cobra. Key commands:

| Command | Usage | Pipeline |
|---------|-------|----------|
| `cm eval file.cm` | Evaluate a file | `Eval()` → text output |
| `cm convert -f html file.cm` | Convert to HTML | `Convert()` → formatted output |
| `echo "1+1" \| cm` | Pipe evaluation | stdin → `Eval()` → stdout |
| `cm` or `cm edit file.cm` | TUI editor | Interactive editor |
| `cm lsp` | LSP server | Starts language server for editor integration |
| `cm watch file.cm` | File watcher | Re-evaluate on save |

### TUI Editor (`cmd/calcmark/tui/editor/`)

Built with [Bubble Tea v2](https://github.com/charmbracelet/bubbletea). Non-modal architecture (no vim-style modes — always editing).

```
+---Source Pane (60%)-----+---Preview Pane (40%)--+
| 1  price = $29.99       | price  → $29.99       |
| 2  tax = price * 8%     | tax    → $2.40        |
| 3  total = price + tax  | total  → $32.39       |
| ~                       |                       |
| ~                       |                       |
+---------+---------------+-----------------------+
| Status: [New] L2:15 | Ready                     |
| avg(value1, value2, ...) — Arithmetic mean       |
+--------------------------------------------------+
```

Key architectural decisions:
- **State machine**: `EditorState` (Ready, Editing, Processing) with explicit transitions
- **Overlays**: Help, command menu, file picker, export — managed via `InputState` and centralized `mode_transitions.go`
- **Testing**: Data-driven catwalk tests using `cockroachdb/datadriven`
- **Autocomplete**: Delegates to shared `spec/features/completion.go` functions

### LSP Server (`lsp/`)

Built with [glsp](https://github.com/tliron/glsp). Supports VS Code (extension in `editors/vscode/`).

```
               didChange
Editor  ───────────────>  Server
                            |
                     source updated immediately
                     debounce timer (150ms)
                            |
                     timer fires → evaluate async
                            |
                     snapshot updated
                            |
              <─────────────+
        diagnostics, semantic tokens
```

**DocumentSnapshot**: Immutable evaluated state. Created on each evaluation cycle. Request handlers (completion, hover, signature) read the latest snapshot without blocking.

**Completion**: Delegates to shared `spec/features/completion.go` functions, then converts `[]Suggestion` to LSP `CompletionItem` with snippets and parameter docs.

### Site (`site/`)

Hugo-based documentation site at calcmark.org. CalcMark examples in docs are evaluated at build time via `cmd/doceval`:

```
Hugo Build → doceval renders .cm blocks → HTML with computed results
```

## Performance Targets

| Operation | Target | Typical |
|-----------|--------|---------|
| Lexer + Parser | < 10us | 1-5us |
| Interpreter | < 50us | 10-50us |
| Document operations | < 100us | 50-100us |
| LSP debounce | 150ms | (intentional) |

Benchmarks in `spec/parser/benchmark_test.go` and `impl/interpreter/operator_benchmark_test.go`.

## Testing Strategy

| Layer | Strategy | Location |
|-------|----------|----------|
| Parser | Golden files (valid + invalid) | `testdata/spec/` |
| Interpreter | Table-driven + golden files | `testdata/eval/`, `impl/interpreter/*_test.go` |
| TUI | Catwalk data-driven tests | `cmd/calcmark/tui/editor/testdata/` |
| LSP | Protocol compliance tests | `lsp/*_test.go` |
| Cross-layer | Fuzz tests (lexer, parser, document) | `spec/*/fuzz_test.go` |
| Boundary | Import constraint test | `spec/boundary_test.go` |

**Quality gate**: `task quality` runs `go fmt`, `go vet`, `gopls modernize`, and `staticcheck`. Must pass before every commit.

## Adding New Features

See [CONTRIBUTING.md](CONTRIBUTING.md) for the step-by-step guide. The typical path:

1. **Lexer**: Add token type if new syntax
2. **Parser**: Add parsing in the appropriate file (`primary.go`, `multiplicative.go`, or new NL file)
3. **Semantic**: Add type checking rules
4. **Interpreter**: Implement evaluation
5. **Classifier**: Update if new line-level syntax
6. **Features**: Register in `spec/features/registry.go` for IDE/help
7. **Completion**: Handled automatically via shared `spec/features/completion.go`
8. **Tests**: Golden files + table-driven tests at each layer
