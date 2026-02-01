# Codebase Structure

**Analysis Date:** 2026-02-01

## Directory Layout

```
go-calcmark/
├── spec/                    # Language specification (NOT implementation)
│   ├── ast/                 # AST node definitions
│   ├── lexer/               # Tokenization
│   ├── parser/              # Recursive descent parser
│   ├── semantic/            # Type checking & validation
│   ├── types/               # Type definitions
│   ├── units/               # Unit system & conversions
│   ├── document/            # Block model & incremental evaluation
│   ├── features/            # Feature registry
│   ├── classifier/          # Token classification
│
├── impl/                    # Implementation (Go-specific interpreter)
│   ├── interpreter/         # Runtime execution engine & built-in functions
│   ├── document/            # Document evaluator (uses spec/document)
│   ├── types/               # Runtime type implementations
│   ├── wasm/                # WebAssembly bindings
│   ├── cmd/                 # Internal utilities (not user-facing)
│
├── cmd/calcmark/            # CLI & TUI entry point
│   ├── main.go              # Version injection, delegates to cmd package
│   ├── cmd/                 # Cobra command handlers
│   │   ├── root.go          # Root command (REPL/editor)
│   │   ├── eval.go          # eval subcommand
│   │   ├── convert.go       # convert subcommand
│   │   ├── edit.go          # edit subcommand
│   │   ├── tui.go           # TUI launch handler
│   │   ├── security.go      # Security checks
│   │   └── version.go       # Version display
│   ├── config/              # Configuration (theme, color mode)
│   ├── tui/                 # Terminal UI components
│   │   ├── app.go           # Main app controller
│   │   ├── components/      # Reusable UI components
│   │   ├── editor/          # Editor UI (ModelV2)
│   │   ├── repl/            # REPL UI
│   │   └── shared/          # Shared types & utilities
│
├── format/                  # Output formatters
│   ├── formatter.go         # Formatter interface & registry
│   ├── registry.go          # Format lookup & registration
│   ├── text_formatter.go    # Plain text output
│   ├── json_formatter.go    # JSON output
│   ├── html_formatter.go    # HTML output
│   ├── calcmark_formatter.go # CalcMark output
│   ├── markdown_formatter.go # Markdown rendering
│   ├── display/             # Display utilities (normalize, wrapping)
│   └── templates/           # HTML templates
│
├── eval.go                  # Public library API (Eval function)
├── session.go               # Session management (stateful eval)
├── result.go                # Result types for public API
├── version.go               # Version constant
├── eval_test.go             # Public API integration tests
│
├── constants/               # Global constants
│   └── strings.go           # String constants
│
├── testdata/                # Test fixtures
│   ├── eval/                # Evaluation golden files
│   ├── spec/                # Parser/semantic golden files
│   └── vhs_tapes/           # TUI test recordings (catwalk)
│
├── docs/                    # User documentation
│   ├── examples/            # Example .cm files
│   └── README.md            # Overview
│
├── .github/                 # GitHub workflows
│   └── workflows/           # CI/CD automation
│
├── dist/                    # Build artifacts (binaries)
│
├── .claude/                 # AI agent configuration
│   ├── agents/              # Agent definitions
│   ├── commands/            # GSD command definitions
│   └── hooks/               # Git hooks
│
└── .planning/               # Planning documents
    └── codebase/            # Architecture analysis docs
```

## Directory Purposes

**spec/**
- Purpose: Language specification independent of any implementation
- Contains: Lexer, parser, semantic rules, type system, unit registry
- Key files: `spec/parser/rdparser.go` (main parser), `spec/semantic/checker.go` (validation)
- Generated files: None; all hand-written
- Philosophy: No dependencies on impl/ directory (one-way dependency only)

**impl/interpreter/**
- Purpose: Stateful execution engine with built-in functions
- Contains: Interpreter struct, ~20 function files (rate_functions.go, availability_functions.go, etc.)
- Key files: `interpreter.go` (main eval loop), `environment.go` (symbol table)
- Generated files: None; hand-written with comprehensive _test.go pairs
- Organization: Functions grouped by domain (rate, availability, capacity, storage, network)

**impl/document/**
- Purpose: Application-level document evaluator (higher than interpreter)
- Contains: Diagnostic types, evaluation flow
- Key files: `diagnostic.go`, `evaluator.go`
- Depends on: `spec/document/Document` for block model

**impl/types/**
- Purpose: Runtime type implementations with extended methods
- Contains: Number, Currency, Quantity, Rate, Date, Duration with operations
- Depends on: shopspring/decimal for arbitrary precision

**cmd/calcmark/cmd/**
- Purpose: Cobra subcommand handlers connecting CLI args to library calls
- Files: root.go (REPL/default), eval.go, convert.go, edit.go, tui.go, security.go
- Pattern: Each command loads file → calls library functions → handles output

**cmd/calcmark/tui/**
- Purpose: Interactive terminal interface using bubbletea (Elm-style TUI framework)
- Key files: `app.go` (mode controller), `editor/model_v2.go` (main editor), `repl/` (REPL model)
- Architecture: Non-modal editor (no vim-style modes); uses catwalk for data-driven testing
- Testing: Editor tests in `editor/testdata/` use VHS tape format for interaction recording

**format/**
- Purpose: Pluggable output formatters (text, JSON, HTML, CalcMark, Markdown)
- Pattern: Registry pattern; new formatters register in `registry.go`
- Used by: CLI commands (eval, convert); NOT used in core evaluation path
- Note: Only output formatting matters (per CLAUDE.md); evaluation is deterministic

**testdata/**
- Purpose: Golden files for parser, semantic, and evaluation tests
- Structure: Mirrors test types (eval/ for evaluation, spec/ for parser/semantic)
- Pattern: Tests compare actual output against golden files; regenerate with `-update` flag
- Usage: Both positive (valid) and negative (invalid) examples

## Key File Locations

**Entry Points:**
- `cmd/calcmark/main.go`: CLI entry point (version injection, delegates to cmd.Execute)
- `eval.go`: Library entry point (Eval, NewSession functions)
- `cmd/calcmark/cmd/root.go`: Cobra root command (REPL/editor default)
- `cmd/calcmark/tui/app.go`: TUI application root

**Configuration:**
- `cmd/calcmark/config/`: Theme and color mode configuration
- `go.mod`: Dependency manifests (Go 1.24.4, key deps: shopspring/decimal, charmbracelet/*, gopls)
- `Taskfile.yml`: Build tasks (test, build, quality checks)

**Core Logic:**
- `spec/parser/rdparser.go`: Recursive descent parser (1000+ lines, security checks)
- `spec/semantic/checker.go`: Semantic validation pipeline
- `impl/interpreter/interpreter.go`: Main interpretation loop with evalNode dispatch
- `spec/units/canonical.go`: Canonical unit definitions (central knowledge per CLAUDE.md)

**Type System:**
- `spec/types/types.go`: Type interface and basic definitions
- `spec/types/number.go`, `currency.go`, `quantity.go`, `rate.go`, `date.go`, `duration.go`: Individual types
- `impl/types/types.go`: Extended runtime implementations

**Testing:**
- `impl/interpreter/all_features_test.go`: Comprehensive feature coverage
- `spec/parser/golden_test.go`: Parser golden file tests
- `cmd/calcmark/tui/editor/catwalk_test.go`: TUI data-driven tests

**Document Model:**
- `spec/document/document.go`: Block-based document with incremental evaluation
- `spec/document/detector.go`: Block detection from source
- `spec/document/frontmatter.go`: Metadata parsing

## Naming Conventions

**Files:**
- `_test.go`: Unit tests for the package
- `_benchmark_test.go`: Performance benchmarks
- `_test.yaml` or in `testdata/`: Golden file tests
- Action verbs for helpers: `detector.go`, `evaluator.go`, `classifier.go`
- Type names match exported structs: `interpreter.go` for Interpreter, `environment.go` for Environment

**Directories:**
- Lowercase, single words or short compounds: `lexer`, `parser`, `semantic`, `interpreter`
- Domain-specific grouping: `tui/editor`, `tui/repl`, `tui/components`
- Test data co-located: `editor/testdata/` for editor tests

**Functions & Methods:**
- Exported: PascalCase (Eval, NewSession, Parse, Check)
- Unexported: camelCase (evaluate, evalNode, parseExpression, checkIdentifier)
- Receivers short and conventional: `interp` for Interpreter, `c` for Checker, `p` for Parser

**Variables:**
- Single-letter for loop indices: `i`, `j`, `k`
- Meaningful names for state: `env` (environment), `diags` (diagnostics), `nodes` (AST nodes)
- Context abbreviations: `fm` (frontmatter), `sem` (semantic)

**Types:**
- PascalCase: `NumberLiteral`, `BinaryOp`, `Checker`, `Formatter`
- Interfaces: End with suffix or meaningful name: `Type`, `Node`, `Formatter`
- Enum-like constants: ALL_CAPS: `MaxNestingDepth`, `MaxTokenCount`

## Where to Add New Code

**New Language Feature (e.g., new operator or function):**
- Spec: Add token type to `spec/lexer/token.go`, lexer pattern to `spec/lexer/lexer.go`
- Spec: Add AST node to `spec/ast/nodes.go` if new syntax
- Spec: Add parser rule to `spec/parser/rdparser.go`
- Spec: Add semantic checks to `spec/semantic/checker.go`
- Impl: Add interpreter eval method to `impl/interpreter/interpreter.go` (evalXXX function)
- Impl: Add built-in function file if function-based (e.g., `impl/interpreter/my_functions.go`)
- Tests: Add test cases to `testdata/eval/` golden files and `impl/interpreter/*_test.go` unit tests

**New Built-in Function:**
- Implementation: `impl/interpreter/domain_functions.go` (e.g., `capacity_functions.go`)
- Registration: Add to `functions.go` registry
- Semantic: If new type involved, add unit/currency validation to `spec/semantic/`
- Tests: Golden file in `testdata/eval/`, unit test in `*_functions_test.go`

**New Output Format:**
- Implementation: `format/myformat_formatter.go` implementing Formatter interface
- Registration: Add to `format/registry.go` registerFormatters function
- Tests: `format/myformat_formatter_test.go` with golden files

**New TUI Component:**
- Location: `cmd/calcmark/tui/components/mycomponent.go` (if generic) or `cmd/calcmark/tui/editor/mycomponent.go` (if editor-specific)
- Pattern: Implement bubbletea Model interface (Init, Update, View)
- Testing: Add catwalk test in `cmd/calcmark/tui/editor/testdata/`

**New CLI Command:**
- Location: `cmd/calcmark/cmd/mycommand.go` with Cobra &cobra.Command
- Registration: Add to root command as subcommand or standalone
- Pattern: Load config → call library function → format output → write

**Utilities & Helpers:**
- Shared across packages: Create in `spec/utils/` or `impl/utils/`
- TUI-specific: `cmd/calcmark/tui/shared/`
- Global constants: `constants/strings.go`

## Special Directories

**testdata/eval/**
- Purpose: Golden file evaluation tests
- Generated: NO (manually curated)
- Committed: YES (critical for backwards compatibility)
- Format: Calcmark source files with expected output
- Updated: `go test -update` regenerates expectations

**testdata/spec/**
- Purpose: Parser and semantic analysis golden files
- Used: `spec/parser/golden_test.go`, semantic tests
- Generated: NO
- Committed: YES
- Coverage: Valid and invalid syntax examples

**testdata/vhs_tapes/**
- Purpose: TUI interaction recordings for catwalk testing
- Format: VHS tape YAML format (key sequences + observations)
- Generated: `go test -update` in editor tests
- Committed: YES (defines expected TUI behavior)
- Observers: debug, results, lines, view (see TESTING.md for details)

**dist/**
- Purpose: Build artifacts
- Generated: YES (by `task build`)
- Committed: NO (`.gitignore` includes)
- Platforms: darwin-amd64, darwin-arm64, linux-amd64, linux-arm64, windows-amd64

**.planning/codebase/**
- Purpose: Architecture documentation (this file)
- Maintained by: GSD mapper commands
- Read by: GSD planner and executor commands

---

*Structure analysis: 2026-02-01*
