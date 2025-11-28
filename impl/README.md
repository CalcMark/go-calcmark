# Implementation Package (`impl/`)

This package contains the runtime implementation of the CalcMark interpreter. While `spec/` defines the language specification (parsing, AST, types), `impl/` executes it.

## Package Overview

| Package | Purpose |
|---------|---------|
| `interpreter/` | Core expression evaluator - executes AST nodes |
| `document/` | Document-level evaluation orchestration |
| `types/` | Runtime type operations (arithmetic, conversions) |
| `wasm/` | WebAssembly bindings for browser use |

## In-Memory Data Structures

Understanding how CalcMark represents documents and variables in memory:

```
Document (spec/document/document.go)
  └── []BlockNode                    // Ordered list of blocks
        ├── ID string                // Unique block identifier
        └── Block                    // Either CalcBlock or TextBlock
              │
              ├── CalcBlock (spec/document/block.go)
              │     ├── source []string      // Raw source lines
              │     ├── statements []Node    // Parsed AST nodes
              │     ├── variables []string   // Variable NAMES defined in this block
              │     ├── dependencies []string // Variable names referenced from other blocks
              │     ├── results []Type       // Per-statement evaluation results
              │     └── lastValue Type       // Result of last statement (not per-variable!)
              │
              └── TextBlock (spec/document/block.go)
                    └── source []string      // Markdown text lines

Evaluator (impl/document/evaluator.go)
  └── env *Environment               // THE source of truth for variable VALUES
        └── vars map[string]Type     // Global variable bindings (e.g., x → 10)
```

## Key Concepts

### Global Variable Scope

All variables in CalcMark have **global scope**. A variable defined in one block is accessible in all subsequent blocks. Reassignment in any block updates the single global binding.

```
x = 10      # Block 1: defines x
---
y = x + 5   # Block 2: references x from block 1, y = 15
---
x = 100     # Block 3: reassigns global x
```

### Understanding CalcBlock Fields

- **`CalcBlock.Variables()`** returns variable *names* defined in that block (for dependency tracking and UI display ordering), NOT values.

- **`CalcBlock.LastValue()`** returns the result of the *last statement* in the block, not individual variable values. For a block with `x = 5` followed by `y = 10`, `LastValue()` returns `10` (y's value).

- **`CalcBlock.Results()`** returns all per-statement results in order.

### The Environment is the Source of Truth

**`Environment.Get(name)`** is the single source of truth for current variable values. The Evaluator maintains one Environment across all block evaluations.

```go
eval := NewEvaluator()
eval.Evaluate(doc)

// Get current variable values from the environment
env := eval.GetEnvironment()
value, ok := env.Get("x")  // Returns current value of x
```

### DependencyAnalyzer

`DependencyAnalyzer` (`spec/document/deps.go`) is a utility that analyzes a CalcBlock to extract which variable names it defines and references. It populates `CalcBlock.variables` and `CalcBlock.dependencies` but doesn't store values.

## Evaluation Flow

### Single Expression

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

### Document-Level Evaluation

```go
// Parse source into a document with blocks
doc, err := document.NewDocument(source)
if err != nil {
    // Handle parse error
}

// Create evaluator (holds the Environment)
eval := NewEvaluator()

// Evaluate all blocks top-down
err = eval.Evaluate(doc)
if err != nil {
    // Handle evaluation error
}

// Access results
for _, node := range doc.GetBlocks() {
    if calcBlock, ok := node.Block.(*document.CalcBlock); ok {
        fmt.Println(calcBlock.LastValue())  // Last statement result
    }
}

// Access current variable values
env := eval.GetEnvironment()
if val, ok := env.Get("total"); ok {
    fmt.Printf("total = %v\n", val)
}
```

### Incremental Evaluation (REPL)

For interactive use, reuse the same Evaluator to maintain variable state:

```go
eval := NewEvaluator()

// First evaluation
doc1, _ := document.NewDocument("x = 10\n")
eval.Evaluate(doc1)

// Later: add a new block that references x
result, _ := doc1.InsertBlock(lastBlockID, document.BlockCalculation, []string{"y = x + 5"})
eval.EvaluateBlock(doc1, result.ModifiedBlockID)

// Environment now has both x=10 and y=15
env := eval.GetEnvironment()
```

## Testing

Run interpreter tests:

```bash
task test:interpreter
```

Run document evaluation tests:

```bash
go test ./impl/document/... -v
```

Key test files:
- `interpreter/interpreter_test.go` - Core evaluation tests
- `document/evaluator_test.go` - Document and global scope tests
