# Testing Patterns

**Analysis Date:** 2026-02-01

## Test Framework

**Runner:**
- Go's built-in `testing` package (no external test framework)
- Config: `Taskfile.yml` defines test targets
- Test files use `*_test.go` suffix with `_test` package name

**Assertion Library:**
- None; uses manual assertions with `if got != want` patterns
- No assertion library dependencies
- Error messages constructed with formatted strings

**Run Commands:**
```bash
task test              # Run all tests (excludes WASM - use test:wasm for those)
task test:short       # Run tests in short mode
task test:coverage    # Run tests with coverage report
task test:lexer       # Run lexer tests only
task test:parser      # Run parser tests only
task test:semantic    # Run semantic validation tests
task test:interpreter # Run interpreter tests
task test:integration # Run integration tests
task test:e2e         # Run golden file e2e tests (validates testdata/*.cm files)
task test:wasm        # Run WASM tests (requires wasm build environment)
```

## Test File Organization

**Location:**
- Co-located with implementation: `{module}_test.go` in same directory as `{module}.go`
- Example: `/Users/bitsbyme/projects/go-calcmark/spec/parser/quantity_test.go` tests parser functions
- Test packages use `{package}_test` naming convention (separate from implementation package)

**Naming:**
- Test function names: `Test{Feature}` (e.g., `TestQuantityParsing`, `TestComprehensiveFeatures`)
- Sub-tests using `t.Run()` with descriptive names (e.g., `"simple addition"`, `"meter short"`)

**Structure:**
```
impl/interpreter/           # Implementation code
  ├── interpreter.go
  ├── functions.go
  ├── operators.go
  └── interpreter_test.go    # Co-located tests
  └── functions_test.go

spec/parser/
  ├── rdparser.go
  ├── quantity_test.go       # Tests parser functionality
  └── golden_test.go         # Golden file tests
```

## Test Structure

**Suite Organization:**
Uses Go's standard `func Test{Name}(t *testing.T)` pattern with sub-tests via `t.Run()`.

**Example Structure:**
```go
func TestFunctionEvaluation(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"avg with 3 args", "avg(1, 2, 3)\n", "2"},
		{"sqrt of 9", "sqrt(9)\n", "3"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes, err := parser.Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}

			interp := interpreter.NewInterpreter()
			results, err := interp.Eval(nodes)
			if err != nil {
				t.Fatalf("Eval error: %v", err)
			}

			if len(results) == 0 {
				t.Fatal("No results returned")
			}

			actual := results[0].String()
			if actual != tt.expected {
				t.Errorf("Result = %s, expected %s", actual, tt.expected)
			}
		})
	}
}
```

**Patterns:**
- Table-driven tests (struct slice with test cases)
- Each test case includes descriptive name, input, and expected output
- `t.Run()` for sub-tests with consistent naming
- Early return on setup errors (`t.Fatalf` for parse/eval failures)
- Type assertions where needed: `if len(results) == 0 { t.Fatal(...) }`
- Final assertions use `t.Errorf` (non-fatal for comparison)

**Setup/Teardown:**
- Minimal setup: constructors called per test case (e.g., `interpreter.NewInterpreter()`)
- No shared state between tests (session reset or new instances created)
- Session tests explicitly test persistence: set variable, use variable, reset

**Error Testing:**
- Tests check both positive cases (success expected) and negative cases (error expected)
- Boolean field `shouldError` in struct tags controls expected behavior
- Error message validation: `strings.Contains(err.Error(), tt.errorMsg)`

## Mocking

**Framework:**
- No mocking framework used (net dependency on testify, GoMock, etc.)
- Mocks created via explicit test doubles or nil checks

**Patterns:**
- Environment isolation: each test gets fresh `interpreter.NewInterpreter()`
- Session tests create new `calcmark.NewSession()` per test
- No global state manipulation

**What to Mock:**
- Nothing explicitly mocked; instead tests use real implementations with controlled inputs
- Environment setup via direct method calls (e.g., `env.Set(name, value)`)

**What NOT to Mock:**
- Don't mock parser (test actual parsing)
- Don't mock interpreter (test actual evaluation)
- Don't mock AST construction (test actual node creation)
- These core components are fast and reliable

## Fixtures and Factories

**Test Data:**
- Golden test files in `./testdata/` directory
- Expression examples in `testdata/spec/valid/expressions/`
- Feature examples in `testdata/spec/valid/features/` (e.g., `arbitrary_units.cm`)
- Error cases in `testdata/spec/invalid/`

**Location:**
- `./testdata/spec/` - Specification/grammar test files
- `./testdata/eval/` - Evaluation behavior test files (success and errors)
- `./testdata/vhs_tapes/` - VHS recordings for TUI/visual tests

**Pattern - Golden File Test:**
```go
func testValidSpecFiles(t *testing.T, baseDir string) {
	validDir := filepath.Join(baseDir, "valid")

	t.Run("documents", func(t *testing.T) {
		docDir := filepath.Join(validDir, "documents")
		testDocumentGoldenFiles(t, docDir)
	})

	t.Run("expressions", func(t *testing.T) {
		exprDir := filepath.Join(validDir, "expressions")
		testExpressionGoldenFiles(t, exprDir)
	})
}
```

**Factory Pattern:**
- Constructors used as factories: `NewInterpreter()`, `NewLexer()`, `NewSession()`
- No separate factory functions; constructors are sufficient

## Coverage

**Requirements:**
- Not enforced by CI (no minimum coverage gate in task definitions)
- Coverage report available via `task test:coverage`
- Generated to `coverage.html` for inspection

**View Coverage:**
```bash
task test:coverage    # Generates coverage.html
go tool cover -html=coverage.out -o coverage.html  # Manual generation
```

## Test Types

**Unit Tests:**
- **Lexer tests**: `spec/lexer/*_test.go` test tokenization
  - Example: `lexer_multipliers_test.go`, `lexer_percentage_test.go`
  - Input: source strings with various formats
  - Output: token sequences

- **Parser tests**: `spec/parser/*_test.go` test AST construction
  - Example: `quantity_test.go`, `date_test.go`, `parser_function_test.go`
  - Input: expressions as strings
  - Output: parsed AST nodes (verified via node count/type)

- **Interpreter tests**: `impl/interpreter/*_test.go` test evaluation
  - Example: `functions_test.go`, `logical_test.go`, `rate_functions_test.go`
  - Input: AST nodes or parsed expressions
  - Output: typed values (Number, Quantity, Duration, Rate, etc.)

- **Type tests**: `spec/types/*_test.go` test type system
  - Example: `types_test.go`, `rate_test.go`
  - Input: type constructors and operations
  - Output: validated type values

**Integration Tests:**
- **Document tests**: `spec/document/*_test.go` test full document parsing
  - Example: `markdown_test.go`, `eval_result_test.go`
  - Input: multi-line CalcMark documents
  - Output: document blocks and evaluation results

- **Comprehensive tests**: `impl/interpreter/comprehensive_test.go` test feature interaction
  - Example: multipliers with units, functions with multipliers
  - All features tested together

- **Golden eval tests**: `impl/interpreter/golden_eval_test.go` test against testdata files
  - Input: CalcMark files from `testdata/eval/`
  - Output: verified against file-based expectations

**E2E Tests:**
- **Golden file tests**: `cmd/calcmark/golden_e2e_test.go` test entire CLI pipeline
  - Location: `testdata/spec/valid/` and `testdata/spec/invalid/`
  - Run with `task test:e2e` (uses `-run "Golden"` filter)
  - Tests parse success, parse failure, and eval behavior
  - **Critical**: Prevents regression in specification compliance

- **VHS tests**: `testdata/vhs_tapes/*.tape` for TUI visual testing
  - Uses catwalk-based data-driven testing
  - Reference: `cmd/calcmark/tui/editor/TESTING.md`
  - Simulates key sequences and validates TUI behavior

## Common Patterns

**Async Testing:**
- Go's testing package handles goroutines naturally
- No async patterns observed in test suite
- All tests are synchronous

**Error Testing:**
```go
func TestFunctionErrors(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"avg no args", "avg()\n"},
		{"sqrt negative", "sqrt(-1)\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes, err := parser.Parse(tt.input)
			if err != nil {
				return // Parse error is acceptable
			}

			interp := interpreter.NewInterpreter()
			_, err = interp.Eval(nodes)
			if err == nil {
				t.Errorf("Expected error for %q but got none", tt.input)
			}
		})
	}
}
```

**Prefix Matching (for floating point):**
```go
if !strings.HasPrefix(actual, tt.expected) {
	t.Errorf("Result = %s, expected to start with %s", actual, tt.expected)
}
```
Used for decimal precision comparisons where exact match may vary.

**Table-Driven with Error Context:**
```go
tests := []struct {
	name        string
	input       string
	shouldError bool
	errorMsg    string
}{
	{"apples vs oranges", "5 apples + 3 oranges\n", true, "incompatible units"},
}

// In test loop:
if tt.shouldError {
	if err == nil {
		t.Errorf("Expected error but got none")
	} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
		t.Errorf("Expected error containing %q, got: %v", tt.errorMsg, err)
	}
}
```

## Benchmarks

**Support:**
- Available via `task bench`, `task bench:lexer`, `task bench:parser`
- Run with `go test ./... -bench=. -benchmem`
- Benchmarks in `*_test.go` files: `func Benchmark{Name}(b *testing.B)`
- Example: `unit_benchmark_test.go` for interpreter performance

**Focus:**
- Parser and lexer performance (parsing is hot path for REPL)
- Unit conversion performance (used frequently)
- Benchmark memory allocations with `-benchmem`

---

*Testing analysis: 2026-02-01*
