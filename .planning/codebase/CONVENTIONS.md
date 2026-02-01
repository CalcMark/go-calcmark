# Coding Conventions

**Analysis Date:** 2026-02-01

## Naming Patterns

**Files:**
- Lower case with underscores for word separation
- Test files use `_test.go` suffix
- Grouped by functionality: `functions.go`, `operators.go`, `literals.go`, etc.
- Eval-specific files: `{feature}_eval.go` (e.g., `unit_conversion_eval.go`, `rate_eval.go`)
- File names reflect primary responsibility (interpreter, parser, lexer, types)

**Functions:**
- Exported functions use PascalCase (e.g., `NewInterpreter`, `Parse`, `Eval`)
- Unexported functions use camelCase (e.g., `evalNode`, `parseExpression`, `readNumber`)
- Constructor functions follow `New{Type}` pattern (e.g., `NewLexer`, `NewInterpreter`, `NewEnvironment`)
- Eval methods follow `eval{Node}` pattern (e.g., `evalBinaryOp`, `evalFunctionCall`, `evalNumberLiteral`)
- Helper functions with descriptive verbs (e.g., `checkTokenLimit`, `skipWhitespace`, `peekAhead`)

**Variables:**
- Local variables use camelCase
- Receiver names use single/two letter abbreviations (e.g., `l *Lexer`, `p *RecursiveDescentParser`, `interp *Interpreter`)
- Loop variables use conventional names: `i`, `j`, `err` for errors
- Loop iterations use `file`, `unit`, `node` for clarity over single letters when meaningful

**Types:**
- Public types use PascalCase (e.g., `NumberLiteral`, `BinaryOp`, `Interpreter`)
- Interface names end in `-er` when appropriate (e.g., no explicit interfaces shown, but error follows Go convention)
- Type fields are exported (PascalCase) when part of public API
- Private struct fields use lowercase (e.g., `tokens []lexer.Token`, `depth int`)

**Constants/Maps:**
- Exported constants use PascalCase (e.g., `BooleanKeywords`, `ReservedKeywords`)
- Config values from literals (e.g., `MaxTokenCount`, `MaxNestingDepth`)
- Map keys are lowercase strings for keyword lookup

## Code Style

**Formatting:**
- `go fmt` enforced (checked in lint task)
- Line length: 140 characters (from `.golangci.yml` lll setting)
- Indentation: tabs (Go standard)

**Linting:**
- Tool: golangci-lint with comprehensive settings (`.golangci.yml`)
- Enabled linters: errcheck, gosimple, govet, staticcheck, typecheck, unused, gofmt, goimports, misspell, goconst, gocritic, gocyclo, dupl
- Cyclomatic complexity: max 15 for functions (gocyclo min-complexity)
- Duplication: max 100 lines before flag (dupl threshold)
- Const length: minimum 3 characters, 3+ occurrences to extract (goconst)
- Shadow checking enabled (govet check-shadowing)

**Code Organization:**
- Logical grouping by functionality (eval operations together, parsing together)
- Clear helper method sections with comment headers (e.g., `// ============================================================================`)
- Related functions grouped in same file when semantically connected

## Import Organization

**Order:**
1. Standard library imports (fmt, strings, unicode, testing)
2. blank line
3. External dependencies (github.com/shopspring/decimal, github.com/CharMedium/*, etc.)
4. blank line
5. Project imports using absolute path (github.com/CalcMark/go-calcmark/...)

**Path Aliases:**
- Used for disambiguation: `implDoc "github.com/CalcMark/go-calcmark/impl/document"` (when importing both spec and impl variants)
- Follows `goimports` local-prefixes setting: `github.com/CalcMark/go-calcmark`

**Examples:**
```go
import (
	"fmt"
	"strings"

	"github.com/shopspring/decimal"

	"github.com/CalcMark/go-calcmark/spec/ast"
	"github.com/CalcMark/go-calcmark/spec/types"
)
```

```go
import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	implDoc "github.com/CalcMark/go-calcmark/impl/document"
	"github.com/CalcMark/go-calcmark/spec/document"
)
```

## Error Handling

**Patterns:**
- Standard Go pattern: `if err != nil { return nil, err }`
- Error wrapping with context: `fmt.Errorf("message: %w", err)` for critical errors
- Explicit custom error types for domain-specific errors: `SecurityError`, `TypeError`, `LexerError`
- Error types implement `Error()` method returning formatted string with context
- Parse/semantic errors wrapped with "parse error:" or "evaluation error:" prefix
- Validation errors include context (line numbers, limits exceeded)

**Error Returns:**
- Functions return `(result, error)` pairs
- Multiple results: `([]nodes, error)` for slice results, `(nil, error)` for failures
- Errors propagate up immediately (no silent failures)
- Node evaluation returns `(types.Type, error)` consistently

**Error Messages:**
- Start with lowercase for inline errors
- Context-specific: include variable names, actual vs expected values
- Security violations include limit names: `"token count exceeds security limit: %d tokens (max %d)"`

## Comments

**When to Comment:**
- Exported functions and types must have comment starting with name (e.g., `// Eval executes...`)
- Complex algorithms deserve explanation (e.g., rate accumulation formula comments in `rate_functions.go`)
- Keyword definitions include purpose (e.g., `// Percentage expressions: "10% of 200"`)
- Edge cases and special handling (e.g., `// Special case: convert_rate's second argument should NOT be evaluated`)
- Non-obvious design decisions (e.g., `// NOTE: "downtime" is NOT a reserved keyword - checked contextually`)

**JSDoc/TSDoc:**
- Go uses doc comments (standard format)
- Example patterns:
  - `// CalcMark is an interpreted language...`
  - `// Interpreter executes validated AST nodes...`
  - `// Eval evaluates a CalcMark expression...`
- Package-level comments include usage examples (see `eval.go`)

**Comment Style:**
- Line comments for single-line explanation
- Block comments for complex sections
- URLs included for reference standards: `// See: https://go.dev/ref/spec#Keywords`

## Function Design

**Size:**
- Functions kept focused on single responsibility
- Large switch statements for type dispatching (e.g., `evalNode` with 20+ cases is acceptable for interpreter dispatcher)
- Helper functions extracted for repeated logic

**Parameters:**
- Receiver-based methods for owned types (e.g., `(interp *Interpreter) evalNode(node ast.Node)`)
- Minimal parameters - context passed via receiver environment when possible
- Return early on validation errors (fail fast)

**Return Values:**
- Consistent pairs: `(value, error)` across package
- Multiple values use slice types: `([]types.Type, error)`
- Named returns avoided (uses unnamed in signatures)
- Nil checks: explicitly check `if result != nil` for optional returns

## Module Design

**Exports:**
- Functions/types exported when part of public API
- Interpreter environment intentionally private (accessed via `GetEnvironment()`)
- Helper/private functions use lowercase consistently

**Barrel Files:**
- No barrel files observed; imports are direct package imports
- Each module `xxx.go` handles specific responsibility

**Initialization:**
- Constructor functions handle all setup (`NewInterpreter`, `NewLexer`, `NewParser`)
- Environment initialized with empty/default state, populated on first use
- No global variables for stateful data (prevents test interference)

## Consistency Patterns

**Type System:**
- All evaluated values implement `types.Type` interface (implicit - methods on types)
- Numbers use `github.com/shopspring/decimal` for arbitrary precision throughout
- Consistent error wrapping at package boundaries

**Testing Integration:**
- Test package names follow `{package}_test` convention (separate from implementation)
- Test setup is minimal (direct constructor calls)
- Assertions use standard `if got != want` with descriptive errors

**Dependency Direction:**
- `spec` packages never depend on `impl` (enforced by import patterns)
- `impl` packages depend on `spec` (unidirectional)
- Clear public API packages: `calcmark` (root), `impl/interpreter`, `spec/parser`

---

*Convention analysis: 2026-02-01*
