# Architecture

**Analysis Date:** 2026-02-01

## Pattern Overview

**Overall:** Layered pipeline with strict separation between language specification (spec) and implementation (impl), connected by a clean public API.

**Key Characteristics:**
- Unidirectional dependency flow: Implementation uses Spec, never the reverse
- Multi-stage processing pipeline: Frontmatter → Parser → Semantic Analysis → Interpreter
- Block-based document model for incremental evaluation
- Formatted output layer decoupled from core evaluation
- TUI built as presentation layer over core evaluation engine

## Layers

**Spec (Language Specification):**
- Purpose: Defines CalcMark language semantics, types, and validation rules. No dependencies on implementation.
- Location: `spec/`
- Contains: AST definitions, lexer, parser, semantic checker, type system, units registry
- Depends on: External libraries only (shopspring/decimal, gomarkdown, golang.org/x)
- Used by: Implementation layer, public API

**Implementation (Interpreter):**
- Purpose: Executes validated AST nodes with stateful environment, implements type operations
- Location: `impl/`
- Contains: Interpreter runtime, built-in functions, operation handlers, type implementations
- Depends on: Spec layer
- Used by: Public API layer, CLI commands

**Public API:**
- Purpose: Clean Go package API for external consumers of CalcMark
- Location: Root-level files (`eval.go`, `result.go`, `session.go`, `version.go`)
- Contains: Session management, stateful evaluation, result types, diagnostic conversion
- Depends on: Spec and Implementation layers
- Used by: CLI, WASM bindings, external applications

**Format Layer:**
- Purpose: Converts evaluated documents to user-facing output formats (text, JSON, HTML, CalcMark)
- Location: `format/`
- Contains: Formatter registry, format-specific implementations, display normalization
- Depends on: Spec documents, public result types
- Used by: CLI commands (convert, eval), not part of core evaluation path

**CLI Commands:**
- Purpose: User-facing command handlers (eval, convert, edit, REPL, TUI)
- Location: `cmd/calcmark/cmd/`
- Contains: Cobra command definitions, file I/O, user interaction
- Depends on: Public API, Format layer, TUI components
- Used by: Main entry point

**TUI (Terminal User Interface):**
- Purpose: Interactive editor and REPL for users to write and execute CalcMark documents
- Location: `cmd/calcmark/tui/`
- Contains: App controller, editor (ModelV2), REPL model, shared components, theming
- Depends on: Public API, charmbracelet libraries (bubbletea, lipgloss, glamour)
- Used by: CLI commands (edit, REPL)

## Data Flow

**Evaluation Pipeline:**

1. **Input Stage**: Raw CalcMark source (string)
2. **Frontmatter Parsing**: Extract metadata (exchange rates, globals)
3. **Block Detection**: Identify CalcBlocks (code) vs TextBlocks (markdown)
4. **Lexical Analysis** (`spec/lexer/`): Tokenize source into semantic tokens
5. **Parsing** (`spec/parser/rdparser.go`): Recursive descent parser builds AST
6. **Semantic Validation** (`spec/semantic/checker.go`): Check types, units, variable definitions
7. **Interpretation** (`impl/interpreter/interpreter.go`): Execute AST with stateful environment
8. **Result Building**: Convert typed values to Result objects
9. **Output Formatting** (`format/`): Render results in requested format

**Stateful Evaluation (Sessions):**

- `Session` (root API) maintains `Environment` across multiple `Eval()` calls
- Variables defined in one evaluation persist to the next
- Each `Eval()` call reuses the same interpreter environment
- Allows interactive REPL: user types "x = 10", next eval can reference x

**Document-Based Evaluation (Incremental):**

- `Document` (`spec/document/document.go`) manages blocks and dependencies
- On source change, identifies affected blocks using dependency graph
- Re-evaluates only modified blocks and their dependents
- Used by TUI editor for live feedback without full re-evaluation

## Key Abstractions

**AST (Abstract Syntax Tree):**
- Purpose: Intermediate representation of parsed source
- Examples: `spec/ast/nodes.go` defines NumberLiteral, Assignment, BinaryOp, FunctionCall
- Pattern: Each node type implements `Node` interface with `String()` and `GetRange()`

**Type System:**
- Purpose: Represents runtime values with domain-specific types
- Examples: `spec/types/` defines Number, Currency, Quantity, Rate, Duration, Date, Boolean
- Pattern: All types implement `Type` interface; `impl/types/` provides additional methods

**Environment:**
- Purpose: Symbol table maintaining variable bindings across evaluation
- Examples: `impl/interpreter/environment.go` stores variables; `spec/semantic/environment.go` for validation
- Pattern: Two environments: semantic (validation phase) and interpreter (execution phase)

**Block Model:**
- Purpose: Document structure with incremental update support
- Examples: `CalcBlock` (calculation code), `TextBlock` (markdown prose)
- Pattern: Blocks wrap with UUID; dependency graph (`varToBlocks` map) tracks variable usage

**Formatter Registry:**
- Purpose: Pluggable output format system
- Examples: TextFormatter, JSONFormatter, HTMLFormatter, CalcMarkFormatter
- Pattern: Registry pattern; Formatter interface for extensibility

## Entry Points

**Library Usage (eval.go):**
- Location: `/Users/bitsbyme/projects/go-calcmark/eval.go`
- Triggers: When external code imports `calcmark.Eval()` or `calcmark.NewSession()`
- Responsibilities: Parse frontmatter → Semantic validation → Interpret → Return typed result

**CLI: eval command:**
- Location: `cmd/calcmark/cmd/eval.go`
- Triggers: `cm eval file.cm` or `cm eval < input.cm`
- Responsibilities: Load file → evaluate → format output → write to stdout

**CLI: convert command:**
- Location: `cmd/calcmark/cmd/convert.go`
- Triggers: `cm convert file.cm --to=html`
- Responsibilities: Load file → evaluate → apply format → write result

**CLI: edit command (TUI Editor):**
- Location: `cmd/calcmark/cmd/edit.go`
- Triggers: `cm file.cm` or `cm file.cm edit`
- Responsibilities: Create Document, launch TUI editor, handle user interactions

**CLI: REPL:**
- Location: `cmd/calcmark/cmd/root.go` (runREPL function)
- Triggers: `cm` (no arguments)
- Responsibilities: Create session, read user input, evaluate, display results interactively

**TUI App:**
- Location: `cmd/calcmark/tui/app.go`
- Triggers: When editor/REPL is activated
- Responsibilities: Initialize lipgloss, delegate to Editor or REPL model, manage mode switching

## Error Handling

**Strategy:** Three-phase error accumulation with non-blocking warnings

**Phases:**
1. **Parse errors**: Return immediately (unrecoverable)
2. **Semantic errors**: Collect as diagnostics, prevent interpretation
3. **Runtime errors**: Return with evaluation error, allow recovery in session

**Patterns:**
- `spec/semantic/diagnostic.go` defines Diagnostic type with Severity (Error/Warning/Hint)
- Semantic checker accumulates all diagnostics, checks for blocking errors
- If semantic errors exist, skip interpretation; return diagnostics only
- Runtime errors use Go's standard error pattern but preserve environment state

**Conversion Pipeline:**
- Spec diagnostics → Public API diagnostics (convertDiagnostics in eval.go)
- TUI displays diagnostics with line numbers and severity coloring

## Cross-Cutting Concerns

**Logging:** Implicit via debug AST dumps and test output; no logging framework used

**Validation:**
- **Lexer**: Detects and reports invalid tokens (reservations, emoji modifiers)
- **Parser**: Validates token sequences, nesting depth (security limits)
- **Semantic**: Type checking, unit compatibility, variable redefinition, date ranges
- **Interpreter**: Runtime errors (division by zero, invalid conversions)

**Security:**
- Token count limit (MaxTokenCount in parser)
- Nesting depth limit (MaxNestingDepth in recursive descent parser)
- See `cmd/calcmark/security.go` for additional CLI-level checks

**Authentication:** None (single-user REPL, no network operations)

**Unit Handling:**
- `spec/units/canonical.go` defines canonical unit registry
- Semantic layer validates unit compatibility before interpretation
- Interpreter converts between compatible units using defined rates
- Exchange rates configurable via frontmatter or session environment

**Performance:**
- Lazy block re-evaluation in Document (only affected blocks)
- Parser uses lookahead for efficient decision-making
- Decimal arithmetic for precision (not float64)
- Time complexity tracked in interpreter functions (see ml_functions.go for optimizations)

---

*Architecture analysis: 2026-02-01*
