---
title: "feat: Named Tables and Array Type"
type: feat
status: active
date: 2026-04-06
origin: docs/brainstorms/2026-04-06-named-tables-and-arrays-requirements.md
---

# feat: Named Tables and Array Type

## Overview

Add named markdown tables and a first-class Array type to CalcMark, enabling tabular data to serve as a computation source. Users declare table and column names via an HTML comment directive (`<!-- table: name (col1, col2) -->`), and calc blocks reference columns as arrays via dot-access syntax (`rates.rate`). Element-wise arithmetic and aggregate functions operate on arrays, with results displayable via interpolation — including per-row array interpolation in table cells.

## Problem Frame

CalcMark users building living documents (consulting SOWs, budget plans, project cost models) must duplicate data between markdown tables and calc blocks. There is no way to say "sum this column" or "multiply these two columns." This feature makes the markdown table the single source of truth for both display and computation. (see origin: `docs/brainstorms/2026-04-06-named-tables-and-arrays-requirements.md`)

## Requirements Trace

- R1. Directive-based table naming: `<!-- table: name (col1, col2) -->`
- R2. Simple normalization of directive names (lowercase, whitespace → underscore)
- R3. Column count in directive must match table column count
- R4. Column values parsed as CalcMark literal types
- R5. Tables registered in environment top-to-bottom; R5a broken-ref diagnostics; R5b tables without directives are inert
- R6. New Array type — ordered, same-type elements, diagnostic on mixed types
- R7. Dot-access syntax: `sales.q1` → column array
- R8. Element-wise arithmetic (same-length arrays); length mismatch → diagnostic
- R9. Array-scalar broadcasting
- R10. Aggregate functions: `sum()`, `avg()`, `min()`, `max()`, `count()` on arrays
- R11. Retain variadic scalar behavior; mixed array+scalar args → error
- R12. Aggregates compose with element-wise: `sum(rates.rate * rates.hc)`
- R13. Scalar interpolation unchanged
- R14. Array display as `[val, val, ...]`
- R15. Array interpolation in table cells: each row gets corresponding element

## Scope Boundaries

- No array literals — arrays come from tables only
- No indexing or row access (`arr[0]`, filtering, slicing)
- No cell-level formulas — table data cells are literals only
- No CSV/file loading
- No dependency graph — top-to-bottom preserved
- No DuckDB or external query engine
- No NL syntax for new functions in v1 (functional syntax only for min/max/count)

## Context & Research

### Relevant Code and Patterns

- **Type addition checklist**: `docs/solutions/logic-errors/adding-new-type-fraction-cross-layer-checklist.md` — 9-layer process. Classifier is the most commonly missed layer.
- **Expression form addition**: `docs/solutions/language-features/directive-as-value-cross-layer-learnings.md` — 12-layer cookbook including interpolation, autocomplete, LSP.
- **Function addition pattern**: `spec/features/registry.go` (metadata), `spec/types/param_types.go` (FunctionSpecs), `impl/interpreter/functions.go` (BuiltinFunctions + eval). Follow unified registry pattern from `docs/solutions/code-organization/unified-feature-registry-three-to-one.md`.
- **Operator dispatch**: `impl/interpreter/operators.go` — normalization blocks at top of `evalBinaryOperation()`, then type-assertion dispatch. See `docs/solutions/feature-gaps/rate-type-arithmetic-widening.md` for the pattern.
- **Existing dot-access**: `@globals.field` uses `DirectiveRef` AST node. DOT token only emitted in `@globals` context (`spec/lexer/lexer.go:~1115`).
- **Interpolation**: `impl/document/interpolation.go` — regex `\{\{\s*(@?\w+(?:\.\w+)?)\s*\}\}` already matches dotted names but resolves via flat `env[ref]` lookup.
- **Evaluator block loop**: `impl/document/evaluator.go:129-140` — TextBlocks only get `checkTextBlockForLikelyCalculations`. Table extraction inserts here.
- **Environment**: `impl/interpreter/environment.go` — flat `map[string]types.Type`. Table stored as a new Type value.

### Institutional Learnings

- Always set `Range` on new AST nodes (from NL function Range learning)
- Classifier (`spec/classifier/classifier.go` AND `spec/document/detector.go`) is the #1 missed layer
- Defense-in-depth: validation in both semantic checker and interpreter
- `InterpolatedSource()` not `Source()` for display
- New keyword categories must be registered in `IsReservedKeywordToken()` and diagnostic pipeline
- Test full type dispatch matrix for new type combinations

## Key Technical Decisions

- **Table as a Type**: A new `*Table` type in `spec/types/` implements the `Type` interface and is stored in the environment as `vars["rates"] = &Table{...}`. `MemberAccess` on a Table returns the column's Array. This leverages the existing environment without adding a parallel storage mechanism.
- **Array as a Type**: A new `*Array` type in `spec/types/` wraps `[]types.Type` with element-type metadata. Implements `Type` interface. Stored in environment when assigned to a variable.
- **MemberAccess AST node**: New node `MemberAccess{Object Node, Field string}` in `spec/ast/nodes.go`. Distinct from `DirectiveRef`. Parsed when `IDENTIFIER DOT IDENTIFIER` is encountered (not preceded by `@`).
- **DOT token generalization**: Extend the lexer to emit DOT tokens after any IDENTIFIER when followed by `.` and another identifier-start character. The existing `@globals` DOT path continues to work via `DirectiveRef` parsing; the new MemberAccess path handles non-`@` cases.
- **Table extraction in evaluator**: The evaluator's block processing loop gains a new step for TextBlocks: scan source lines for `<!-- table: ... -->` directives, parse the subsequent markdown table, extract column data, and register in the environment. This happens during evaluation (not document parsing) so it participates in the top-to-bottom ordering.
- **Array interpolation**: The interpolation system gains table-structure awareness. When `{{var}}` resolves to an Array and the tag is inside a markdown table row, the interpolation pass maps array elements to rows by index.
- **Aggregate function dispatch**: `sum()` and `avg()` gain an array path alongside their existing variadic scalar path. Dispatch on first argument type: if Array, use aggregate path; if scalar, use variadic path. `min()`, `max()`, `count()` are new functions with array-only support initially.

## Open Questions

### Resolved During Planning

- **Where does table extraction happen?** In the evaluator (`impl/document/evaluator.go`), during the block processing loop. TextBlocks are scanned for directive comments. This keeps table registration in the evaluation pass (top-to-bottom ordering) without modifying the document parser or creating a new block type.
- **How is MemberAccess distinguished from DirectiveRef?** By the `@` prefix. `@globals.field` → DirectiveRef (existing path). `sales.q1` → MemberAccess (new path). The parser dispatches based on whether the expression starts with `AT_SIGN`.
- **How does the interpolation regex handle dotted names for arrays?** The regex already matches `\w+\.\w+`. The resolution logic changes: if `ref` contains a `.`, split into `table.column`, look up the table in env, get the column Array. For array-in-table-cell display, detect markdown table row context and index by row position.
- **Should Table implement Type?** Yes. It makes the environment's `map[string]types.Type` work without modification. `Table.String()` returns a summary like `table(3 rows, 4 cols)`.

### Deferred to Implementation

- Exact identifier validation rules for directive names (which invalid characters produce diagnostics)
- How table name collisions with existing variables are reported (likely same as variable redefinition: diagnostic error)
- Whether `NormalizeForDisplay` needs Array-specific behavior or delegates to per-element formatting
- Interaction between `@scale` directive and table cell values

## High-Level Technical Design

> *This illustrates the intended approach and is directional guidance for review, not implementation specification. The implementing agent should treat it as context, not code to reproduce.*

```
Document source:

  <!-- table: rates (role, rate, hc) -->
  | Role   | Rate    | HC |
  |--------|---------|-----|
  | Senior | $250/hr | 3   |
  | Junior | $150/hr | 5   |

  ---

  line_costs = rates.rate * rates.hc
  total = sum(line_costs)

Evaluation flow:

  1. Evaluator encounters TextBlock containing directive + table
  2. Parse directive: name="rates", columns=["role","rate","hc"]
  3. Parse markdown table rows, lex each cell as CalcMark literal
  4. Build column arrays:
       role → Array[String?] (non-numeric, accessible but errors on arithmetic)
       rate → Array[Rate: $250/hr, $150/hr]
       hc   → Array[Number: 3, 5]
  5. Create Table{columns: map[string]*Array{...}}
  6. Register: env["rates"] = &Table{...}

  7. Evaluator reaches CalcBlock
  8. Parse: line_costs = rates.rate * rates.hc
  9. Evaluate MemberAccess "rates.rate" → Array[$250/hr, $150/hr]
  10. Evaluate MemberAccess "rates.hc" → Array[3, 5]
  11. Element-wise multiply: Array[$750/hr, $750/hr]
  12. Assign: env["line_costs"] = Array[$750/hr, $750/hr]

  13. Parse: total = sum(line_costs)
  14. Evaluate sum(Array) → aggregate → $1,500/hr
  15. Assign: env["total"] = Rate($1,500/hr)

Interpolation pass (post-evaluation):
  - {{total}} in prose → "$1,500/hr" (scalar, existing behavior)
  - {{line_costs}} in table cell row 0 → "$750/hr" (array element 0)
  - {{line_costs}} in table cell row 1 → "$750/hr" (array element 1)
```

## Implementation Units

### Phase 1: Type Foundation

- [ ] **Unit 1: Array and Table Types**

**Goal:** Introduce `Array` and `Table` types implementing the `Type` interface.

**Requirements:** R6, R14

**Dependencies:** None — foundational types that everything else builds on.

**Files:**
- Create: `spec/types/array.go`
- Create: `spec/types/table.go`
- Test: `spec/types/array_test.go`
- Test: `spec/types/table_test.go`

**Approach:**
- `Array` struct: `Elements []types.Type`, `ElementType string` (for display/diagnostics). Constructor validates all elements are compatible types; returns error on mixed types. `String()` returns `[val1, val2, ...]` using element `String()`.
- `Table` struct: `Name string`, `Columns map[string]*Array`, `ColumnOrder []string` (preserves declaration order), `RowCount int`. `String()` returns summary. `Column(name string) (*Array, bool)` accessor.
- Both implement `Type` interface.

**Patterns to follow:**
- `spec/types/number.go`, `spec/types/quantity.go` for struct + constructor + String() pattern
- `spec/types/param_types.go` for ArgType constants if needed

**Test scenarios:**
- Happy path: Create Array of Numbers, verify String() output is `[1, 2, 3]`
- Happy path: Create Array of Currencies, verify String() preserves currency symbols
- Happy path: Create Table with multiple columns, verify Column() accessor returns correct arrays
- Edge case: Create Array with zero elements — should succeed (empty table column)
- Edge case: Create Table with zero rows — should succeed (header-only table)
- Error path: Create Array with mixed types (Number + Currency) — constructor returns error
- Error path: Table.Column("nonexistent") — returns nil, false

**Verification:**
- `go test ./spec/types/...` passes
- Array and Table values can be stored in and retrieved from the Environment

---

### Phase 2: Syntax (Lexer, AST, Parser)

- [ ] **Unit 2: MemberAccess AST Node**

**Goal:** Add a `MemberAccess` AST node for `table.column` dot-access expressions.

**Requirements:** R7

**Dependencies:** None

**Files:**
- Modify: `spec/ast/nodes.go` (add MemberAccess node and ContainsScaleRef case)

**Approach:**
- `MemberAccess{Object Node, Field string, Range *Range}` — Object is typically an `Identifier`, Field is the column name string.
- Add case to `ContainsScaleRef()` that recurses into `Object`.
- `String()` returns `Object.Field`.

**Patterns to follow:**
- `DirectiveRef` in `spec/ast/nodes.go` for a dot-access node pattern
- Always set `Range` from token position (learning: NL function Range)

**Test scenarios:**
- Happy path: MemberAccess node has correct String() representation
- Happy path: ContainsScaleRef correctly recurses into MemberAccess.Object
- Happy path: GetRange() returns the range set at construction

**Verification:**
- `go test ./spec/ast/...` passes
- Node can be constructed, printed, and inspected

---

- [ ] **Unit 3: Lexer DOT Token Generalization**

**Goal:** Emit DOT tokens after regular identifiers (not just `@globals`), enabling `sales.q1` to tokenize as `IDENTIFIER DOT IDENTIFIER`.

**Requirements:** R7

**Dependencies:** Unit 2 (AST node exists)

**Files:**
- Modify: `spec/lexer/lexer.go` (DOT emission logic)
- Modify: `spec/lexer/token.go` (if new token types needed for min/max/count)
- Test: `spec/lexer/lexer_test.go`

**Approach:**
- Current DOT emission is guarded by `dirToken.Value == "globals" && l.currentChar() == '.'`. Generalize: after emitting an IDENTIFIER token, if the next char is `.` and the char after that is an identifier-start character (letter or underscore), emit DOT and continue to the next IDENTIFIER.
- Must not break: decimal numbers (`1.5`), existing `@globals.field` path, range notation if any.
- Add `FUNC_MIN`, `FUNC_MAX`, `FUNC_COUNT` tokens and their `ReservedKeywords` entries.

**Patterns to follow:**
- Existing DOT handling in `spec/lexer/lexer.go:~1115`
- `FUNC_SUM`, `FUNC_AVG` token pattern for new function tokens

**Test scenarios:**
- Happy path: `sales.q1` tokenizes as `IDENTIFIER("sales") DOT IDENTIFIER("q1")`
- Happy path: `@globals.tax_rate` still tokenizes correctly (regression)
- Happy path: `1.5` still tokenizes as a number, not `NUMBER DOT NUMBER`
- Happy path: `min`, `max`, `count` tokenize as their function tokens
- Edge case: `a.b.c` tokenizes as `IDENTIFIER DOT IDENTIFIER DOT IDENTIFIER` (parser will reject nested access)
- Edge case: `a.123` does NOT emit DOT (digit after dot → not identifier start)
- Error path: reserved keywords like `sum.x` — `sum` tokenizes as FUNC_SUM, then `.x` is an error (function tokens don't get dot-access)

**Verification:**
- `go test ./spec/lexer/...` passes
- Existing lexer tests pass (no regressions in number parsing)

---

- [ ] **Unit 4: Parser — MemberAccess and New Function Tokens**

**Goal:** Parse `identifier.field` into MemberAccess nodes. Parse `min()`, `max()`, `count()` function calls.

**Requirements:** R7, R10

**Dependencies:** Unit 2 (AST node), Unit 3 (lexer tokens)

**Files:**
- Modify: `spec/parser/primary.go` (`parsePrimary` — identifier case, function call case)
- Test: `spec/parser/parser_test.go`
- Create: `testdata/spec/valid/features/named_tables.cm`
- Create: `testdata/spec/invalid/features/named_tables.cm`

**Approach:**
- In `parsePrimary()`, after matching an `IDENTIFIER` token: peek for `DOT`. If DOT follows, consume it, expect another `IDENTIFIER`, build `MemberAccess{Object: &Identifier{Name: first}, Field: second}`. Set Range from first token.
- Reject nested dot access (`a.b.c`) with a clear diagnostic: "nested dot access is not supported."
- Add `FUNC_MIN`, `FUNC_MAX`, `FUNC_COUNT` to the `p.match(...)` call in the function call section of `parsePrimary()`.
- Add arg count validation for new functions in `parseFunctionCall()`.
- **P0 fix**: Relax parser-level arg count validation for `sum()` and `avg()` from `< 2` to `< 1`. The parser currently rejects `sum(array)` at parse time (`spec/parser/primary.go:532`). With arrays, these functions accept either a single array arg or 2+ scalar args — the interpreter (Unit 8) validates based on argument type, not the parser based on count. (NL syntax for min/max/count is deferred to v2; this unit covers functional syntax only.)

**Patterns to follow:**
- DirectiveRef parsing in `parsePrimary()` for the `AT_SIGN → IDENTIFIER → DOT → IDENTIFIER` pattern
- `parseFunctionCall()` for arg count validation

**Test scenarios:**
- Happy path: `x = sales.q1` parses as Assignment with MemberAccess RHS
- Happy path: `sum(sales.q1)` parses as FunctionCall with MemberAccess argument
- Happy path: `sum(a.b * c.d)` parses with element-wise expression inside function call
- Happy path: `min(x, y)`, `max(x, y)`, `count(arr)` parse correctly
- Error path: `a.b.c` produces "nested dot access is not supported" diagnostic
- Error path: `min()` with no args produces argument count error
- Integration: Parsing still works for `@globals.field` (DirectiveRef regression)

**Verification:**
- `go test ./spec/parser/...` passes
- Golden test files parse correctly

---

### Phase 3: Document Pipeline

- [ ] **Unit 5: Table Directive Parsing and Table Extraction**

**Goal:** Detect `<!-- table: name (col1, col2) -->` directives in TextBlocks, parse the subsequent markdown table, extract column data as Arrays, and register as a Table in the environment.

**Requirements:** R1, R2, R3, R4, R5, R5a, R5b, R6

**Dependencies:** Unit 1 (Array and Table types)

**Files:**
- Create: `impl/document/table_extraction.go`
- Create: `impl/document/table_extraction_test.go`
- Modify: `impl/document/evaluator.go` (block processing loop, ~line 136)

**Approach:**
- New function `extractNamedTables(tb *document.TextBlock, env *Environment) ([]*types.Table, []document.Diagnostic)`:
  - Scan TextBlock source lines for regex matching `<!--\s*table:\s*(\w+)\s*\(([^)]+)\)\s*-->`.
  - Parse directive: extract table name + column names. Apply normalization (lowercase, whitespace → underscore). Validate names are valid identifiers.
  - Find the next markdown table in subsequent lines (detect pipe-delimited rows, skip separator row).
  - Validate column count matches directive.
  - For each data row, split by `|`, trim whitespace, lex each cell value using the existing lexer/parser to produce a `types.Type`.
  - Build column Arrays (validate same-type within each column).
  - Construct Table, register in environment via `env.Set(name, table)`.
- In the evaluator's block loop, add a TextBlock case that calls `extractNamedTables` before `checkTextBlockForLikelyCalculations`. Tables are registered in the environment immediately, making them available to subsequent CalcBlocks.
- **All three evaluator entry points must extract tables**: `Evaluate()` (~line 136), `EvaluateBlock()` (two-pass, ~line 224), and `EvaluateAffectedBlocks()` (~line 391). Currently all three only process CalcBlocks for TextBlocks. Without this, TUI incremental evaluation will hold stale table data when a TextBlock is edited.
- Diagnostics for: invalid name, column count mismatch, unparseable cell value, mixed-type column, duplicate table name.

**Patterns to follow:**
- `checkTextBlockForLikelyCalculations` for the pattern of inspecting TextBlock source lines during evaluation
- `impl/document/interpolation.go` for line-by-line TextBlock processing

**Test scenarios:**
- Happy path: Directive + table with 3 numeric columns → 3 Arrays registered
- Happy path: Directive with Currency column → Array of Currency values
- Happy path: Table with mixed column types (text + numeric) — each column gets its own type
- Happy path: Multiple tables in different TextBlocks → both registered
- Edge case: Directive with whitespace in names → normalized correctly (`Rate Per Hour` → `rate_per_hour`)
- Edge case: Table with empty cells → diagnostic on that cell
- Edge case: Directive with no subsequent table → diagnostic
- Error path: Column count mismatch (directive says 3, table has 4) → diagnostic
- Error path: Mixed types within a single column ($100 and 50%) → diagnostic
- Error path: Duplicate table name → diagnostic
- Error path: Invalid identifier after normalization → diagnostic
- Integration: TextBlock without directive is untouched (backward compatibility)

**Verification:**
- `go test ./impl/document/...` passes
- A `.cm` file with a directive+table evaluates without errors and populates the environment

---

### Phase 4: Semantic Checker and Interpreter

- [ ] **Unit 6: Semantic Checker — MemberAccess and Array Awareness**

**Goal:** Add MemberAccess to the semantic checker's node dispatch. Validate that the object part is a known variable (when possible). Track Array variables for function call validation.

**Requirements:** R5a, R7

**Dependencies:** Unit 2 (AST node), Unit 4 (parser produces MemberAccess)

**Files:**
- Modify: `spec/semantic/checker.go` (`checkNode` switch)
- Test: `spec/semantic/checker_test.go`

**Approach:**
- Add `*ast.MemberAccess` case to `checkNode()`. Mark `node.Object` (if Identifier) as a referenced variable. The semantic checker cannot fully validate column names at check time (tables are registered during evaluation, not parsing), so the primary check is variable-exists for the table name.
- Extend function call validation for `min`, `max`, `count`: accept 1+ arguments.
- Add `FunctionSpec` entries in `spec/types/param_types.go` for min, max, count.

**Patterns to follow:**
- `DirectiveRef` case in `checkNode()` for dot-access validation
- Existing `FunctionSpecs` entries for sum, avg

**Test scenarios:**
- Happy path: `sales.q1` does not produce an "undefined variable" warning when `sales` is known
- Happy path: `min(x)`, `max(x, y)`, `count(arr)` pass validation
- Error path: `unknown_table.col` produces undefined variable diagnostic for `unknown_table`
- Integration: Existing semantic checks still pass (regression)

**Verification:**
- `go test ./spec/semantic/...` passes

---

- [ ] **Unit 7: Interpreter — MemberAccess, Element-wise Ops, Array Assignment**

**Goal:** Evaluate MemberAccess nodes (table.column → Array), element-wise binary operations on Arrays, and Array-scalar broadcasting.

**Requirements:** R7, R8, R9, R12

**Dependencies:** Unit 1 (types), Unit 2 (AST), Unit 5 (tables in env)

**Files:**
- Modify: `impl/interpreter/interpreter.go` (`evalNode` switch)
- Modify: `impl/interpreter/operators.go` (`evalBinaryOperation`)
- Create: `impl/interpreter/array_ops.go`
- Test: `impl/interpreter/array_ops_test.go`
- Create: `testdata/eval/success/features/named_tables.cm`
- Create: `testdata/eval/errors/features/named_tables.cm`

**Approach:**
- `evalNode`: add `*ast.MemberAccess` case → `evalMemberAccess()`. Look up Object in env, assert `*types.Table`, call `table.Column(field)`. Return the Array. Produce diagnostic if table or column not found.
- `evalBinaryOperation`: add Array normalization block at the top (before existing normalization). If both operands are Arrays: validate same length, zip-apply the operation element-wise (recursing into `evalBinaryOperation` for each pair), return new Array. If one operand is Array and other is scalar: broadcast scalar, apply element-wise.
- New file `array_ops.go` for `evalArrayBinaryOp` and `evalArrayScalarOp` helpers.

**Patterns to follow:**
- `evalBinaryOperation` normalization blocks at top (from rate-widening learning)
- Existing `evalDirectiveRef` for the pattern of resolving a dot-access expression

**Test scenarios:**
- Happy path: `rates.rate` evaluates to Array of Rate values
- Happy path: `rates.rate * rates.hc` → element-wise Array result
- Happy path: `rates.rate * 1.1` → broadcast scalar to each element
- Happy path: `line_costs = rates.rate * rates.hc` assigns Array to variable
- Happy path: Chained: `sum(rates.rate * rates.hc)` evaluates correctly end-to-end
- Edge case: Element-wise on single-element arrays → single-element result
- Edge case: Array * Array where element types differ but are coercible (Number * Currency)
- Error path: `rates.nonexistent` → diagnostic "column 'nonexistent' not found in table 'rates'"
- Error path: `nonexistent.col` → diagnostic "undefined variable 'nonexistent'"
- Error path: `scalar_var.field` → diagnostic "dot access on non-table value"
- Error path: Arrays of different lengths → diagnostic with both lengths

**Verification:**
- `go test ./impl/interpreter/...` passes
- Golden test files in `testdata/eval/` pass

---

- [ ] **Unit 8: Aggregate Functions — sum, avg, min, max, count**

**Goal:** Implement array-accepting aggregate functions. Extend existing `sum()` and `avg()` with array dispatch. Add new `min()`, `max()`, `count()` functions.

**Requirements:** R10, R11, R12

**Dependencies:** Unit 1 (Array type), Unit 4 (parser support for min/max/count tokens), Unit 7 (element-wise ops for R12)

**Files:**
- Modify: `impl/interpreter/functions.go` (sum, avg eval functions + new min/max/count)
- Modify: `spec/types/param_types.go` (FunctionSpecs for min, max, count)
- Modify: `spec/features/registry.go` (feature entries for min, max, count + update sum)
- Test: `impl/interpreter/functions_test.go`

**Approach:**
- `evalSumFunc`: check first arg type. If `*types.Array`, iterate elements and accumulate (reuse existing addition logic). If scalar, use existing variadic path (min 2 args).
- `evalAvgFunc`: same dispatch pattern — Array path divides sum by count.
- New `evalMinFunc`, `evalMaxFunc`: accept single Array arg, iterate with comparison. Use existing `evalComparison` for element comparison.
- New `evalCountFunc`: accept single Array arg, return `types.NewNumber(len(elements))`.
- All aggregate functions: if Array is empty, return diagnostic.
- Register new functions in `BuiltinFunctions` and `functionEvalMap`.
- Add Feature entries in registry with Category `CategoryMath`.

**Patterns to follow:**
- Existing `evalAvgFunc` structure for sum/avg extension
- `BuiltinFunctions` registration pattern in `functions.go`
- Feature registration in `spec/features/registry.go` (`getFunctions()`)

**Test scenarios:**
- Happy path: `sum(array_of_numbers)` → correct sum
- Happy path: `sum(array_of_currencies)` → correct currency sum with auto-scaling
- Happy path: `avg(array_of_numbers)` → correct average
- Happy path: `min(array_of_numbers)` → smallest element
- Happy path: `max(array_of_currencies)` → largest currency
- Happy path: `count(array)` → Number equal to element count
- Happy path: `sum(a, b, c)` still works (variadic scalar, regression)
- Happy path: `sum(rates.rate * rates.hc)` → aggregate over element-wise result
- Edge case: Single-element array in sum/avg/min/max → returns that element
- Error path: `sum(array, scalar)` → mixed args diagnostic
- Error path: `sum(single_scalar)` → still requires 2+ args (existing behavior)
- Error path: `min()` with no args → argument error
- Error path: Aggregate on empty array → diagnostic

**Verification:**
- `go test ./impl/interpreter/...` passes
- Feature registry includes min, max, count entries

---

### Phase 5: Display and Interpolation

- [ ] **Unit 9: Display Formatting and Array Interpolation**

**Goal:** Format Array values for display output. Extend interpolation to map Array elements to markdown table rows.

**Requirements:** R13, R14, R15

**Dependencies:** Unit 1 (Array type), Unit 5 (table extraction), Unit 7 (MemberAccess)

**Files:**
- Modify: `format/display/formatter.go` (`Format` method — Array case)
- Modify: `format/json_formatter.go` (`populateResult` type switch — Array/Table cases)
- Modify: `impl/document/interpolation.go` (array-in-table-cell logic)
- Test: `format/display/formatter_test.go`
- Test: `format/json_formatter_test.go`
- Test: `impl/document/interpolation_test.go`

**Approach:**
- **Display**: Add `*types.Array` case to `Formatter.Format()`. Iterate elements, format each with existing per-type formatter, join with `, `, wrap in `[...]`.
- **Scalar interpolation** (`{{total}}`): No change needed — env lookup returns scalar, existing path formats it.
- **Dotted interpolation** (`{{rates.rate}}`): Extend `interpolateLine` resolution: if `ref` contains `.`, split into table+column, look up table in env, get column Array. If scalar context (not in a table row), format as `[...]`. If in a markdown table row, use row-index mapping.
- **Array-in-table interpolation** (`{{line_costs}}` in table rows): The interpolation pass must detect when a `{{var}}` resolving to an Array is inside a pipe-delimited markdown table row. When detected, determine the data row index (skip header and separator rows), and substitute the corresponding Array element. Length mismatch → leave tag unresolved and add diagnostic.
- The interpolation regex already matches dotted names — no regex change needed.

**Patterns to follow:**
- Existing `interpolateLine` for per-line resolution pattern
- STX/ETX sentinel wrapping for HTML output

**Test scenarios:**
- Happy path: Array formats as `[$250/hr, $150/hr]` in non-table context
- Happy path: `{{total}}` (scalar) in prose → formatted value (regression)
- Happy path: `{{line_costs}}` in table row 0 → first element; row 1 → second element
- Happy path: `{{rates.rate}}` dotted access in interpolation resolves correctly
- Edge case: `{{array_var}}` outside a table → formats as `[val, val]`
- Edge case: Table with `{{var}}` where var is scalar → same value in every row (existing behavior)
- Error path: Array length doesn't match table row count → tag left unresolved, diagnostic
- Happy path: JSON output (`--format json`) serializes Array as a JSON array of typed results
- Integration: Existing interpolation tests pass unchanged

**Verification:**
- `go test ./format/display/...` passes
- `go test ./impl/document/...` passes
- End-to-end: a `.cm` file with table + calc block + interpolation produces correct output

---

### Phase 6: Integration

- [ ] **Unit 10: Classifier, Feature Registry, and Golden Tests**

**Goal:** Ensure the classifier recognizes table-referencing expressions, update feature registry, add comprehensive golden tests, and update documentation.

**Requirements:** All — integration verification

**Dependencies:** All previous units

**Files:**
- Modify: `spec/classifier/classifier.go` (recognize MemberAccess patterns)
- Modify: `spec/document/detector.go` (if needed for table-referencing lines)
- Modify: `spec/features/registry.go` (Array type feature, named tables feature)
- Create: `testdata/eval/success/features/named_tables.cm` (comprehensive golden test)
- Create: `testdata/eval/errors/features/named_tables_errors.cm` (error golden test)
- Create: `testdata/spec/valid/features/named_tables.cm` (parse-only golden test)

**Execution note:** Run `task test` after each sub-change. The classifier is the most commonly missed layer (institutional learning).

**Approach:**
- **Classifier**: Lines containing `identifier.identifier` patterns (table column references) should be classified as calculation lines, not prose. Add pattern recognition for dot-access expressions.
- **Feature registry**: Add entries for "Named Tables" (keyword category) and "Array" (type category). Update sum() feature entry to mention array support.
- **Golden tests**: Comprehensive `.cm` files covering the consulting SOW use case end-to-end, including directive, table, calc block, interpolation, and error cases.

**Patterns to follow:**
- Existing classifier patterns in `spec/classifier/classifier.go`
- Feature entries in `spec/features/registry.go`
- Golden test format: `var = expression` followed by `# Expected: value`

**Test scenarios:**
- Happy path: Full consulting SOW example evaluates correctly end-to-end
- Happy path: `rates.rate` line classified as calculation, not prose
- Happy path: Feature registry includes named tables, array, min, max, count
- Error path: All error golden tests produce expected diagnostics
- Integration: `task test` passes with all existing and new tests
- Integration: `task quality` passes

**Verification:**
- `task test` passes (full suite)
- `task quality` passes
- Golden test files validate correct behavior

## System-Wide Impact

- **Interaction graph:** Table extraction inserts into the evaluator's block processing loop. Interpolation gains array awareness. The semantic checker gains MemberAccess support. The operator dispatch gains Array normalization. These are all additive — no existing paths are modified in ways that change their behavior for non-array values.
- **Error propagation:** Diagnostics from table extraction (parse errors, type mismatches) flow through the existing `document.Diagnostic` pipeline. MemberAccess errors in the interpreter use the same error recovery pattern as other eval errors.
- **State lifecycle risks:** Table data is extracted from TextBlock source on each evaluation pass. No caching across passes — consistent with how CalcBlocks are re-evaluated. In the TUI's `EvaluateAffectedBlocks`, editing a TextBlock with a table directive should trigger re-extraction and re-evaluation of dependent CalcBlocks.
- **API surface parity:** JSON output (`--format json`) needs to handle Array values. The LSP should provide completions for table names and column names (deferred — not in v1 scope but should not be blocked by the design).
- **Integration coverage:** The full pipeline (directive → extraction → env registration → MemberAccess → element-wise ops → aggregation → interpolation) must be tested end-to-end, not just per-unit.
- **Unchanged invariants:** Existing CalcMark documents with markdown tables but no directives are completely unaffected. The `Type` interface is unchanged. The `Environment` API is unchanged. All existing functions retain their current behavior for scalar arguments.

## Risks & Dependencies

| Risk | Mitigation |
|------|------------|
| DOT token generalization breaks number parsing (e.g., `1.5`) | Lexer only emits DOT after IDENTIFIER tokens, never after NUMBER. Extensive regression tests on decimal literals. |
| Table extraction performance for large documents with many TextBlocks | Extraction is O(lines) per TextBlock, only triggered when directive pattern is found. Early-exit regex check before full parsing. |
| Array type creates blast radius across operator dispatch | Array normalization is a single block at the top of `evalBinaryOperation`, before all existing type dispatch. Existing scalar paths are untouched. |
| TUI incremental evaluation may not re-extract tables on TextBlock edit | Ensure `EvaluateAffectedBlocks` marks table-containing TextBlocks as dirty when their source changes. Add catwalk test for this scenario. |
| Interpolation regex matches `table.column` but existing resolution treats it as flat key | Resolution logic is extended: if `ref` contains a `.`, split into table+column and resolve via Table.Column(). Since CalcMark identifiers cannot contain dots, no backward compatibility risk exists for dotted names. |

## Documentation / Operational Notes

- Update `site/content/` documentation with named tables feature guide
- Add a "Consulting SOW" worked example to the documentation
- Update `ARCHITECTURE.md` if the 8-layer checklist needs a "table extraction" note
- The `sum()` function documentation should mention array support

## Sources & References

- **Origin document:** [docs/brainstorms/2026-04-06-named-tables-and-arrays-requirements.md](docs/brainstorms/2026-04-06-named-tables-and-arrays-requirements.md)
- **Type addition checklist:** `docs/solutions/logic-errors/adding-new-type-fraction-cross-layer-checklist.md`
- **Expression form cookbook:** `docs/solutions/language-features/directive-as-value-cross-layer-learnings.md`
- **Unified registry pattern:** `docs/solutions/code-organization/unified-feature-registry-three-to-one.md`
- **Operator dispatch pattern:** `docs/solutions/feature-gaps/rate-type-arithmetic-widening.md`
- Related brainstorm: `docs/brainstorms/2026-03-09-sum-function-brainstorm.md` (subsumed)
