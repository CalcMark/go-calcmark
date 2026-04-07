---
date: 2026-04-06
topic: named-tables-and-arrays
---

# Named Tables and Array Type

## Problem Frame

CalcMark excels at linear calculations but can't work with tabular data natively. Users building living documents — consulting proposals, budget plans, project cost models — must duplicate data between markdown tables (for presentation) and calc blocks (for computation). There's no way to say "sum this column" or "multiply these two columns together."

The canonical use cases:
- **Consulting SOW**: A table of roles, rates, and headcount. CalcMark computes line totals and grand totals from the table data, with per-row computed values displayed in the table and totals referenced elsewhere via interpolation.
- **Project budget**: Regional spend by quarter in a table. CalcMark sums columns and rows, renders totals via interpolation.
- **Published living documents**: A git-tracked `.cm` file rendered to HTML (like evidence.dev), where changing one input (e.g., an exchange rate) recomputes everything downstream.

## Requirements

**Named Tables**

- R1. A markdown table becomes a named data source when preceded by a directive comment: `<!-- table: name (col1, col2, col3) -->`. The directive declares both the table name and the column names used in calc block references. The markdown header row is display-only.
- R2. Names in the directive undergo simple normalization: lowercased, unicode whitespace replaced with underscores. The resulting names must be valid CalcMark identifiers. No complex normalization (stripping emoji, transliterating accented chars, etc.) — if the normalized name isn't a valid identifier, it's an error.
- R3. The number of column names in the directive must match the number of columns in the markdown table. A mismatch produces a diagnostic.
- R4. Column values are parsed as CalcMark literal types (currencies, quantities, numbers, etc.) using existing parsing rules. Table data cells must contain literal values only — no interpolation tags or expressions.
- R5. Named tables are registered in the environment where they appear in the document. Calc blocks below can reference them; calc blocks above cannot (preserves top-to-bottom execution).
- R5a. Referencing a nonexistent table or column produces a diagnostic error (e.g., "column 'revenue' not found in table 'sales'"). Same treatment as referencing an undefined variable.
- R5b. Tables without a directive are inert markdown — no name, no column extraction, no impact on the environment. This preserves backward compatibility.

**Array Type**

- R6. A new `Array` type joins the CalcMark type system. An array is an ordered collection of values. All elements in a column-derived array must be the same CalcMark type (e.g., all Currency, all Number). Columns with mixed types produce a diagnostic at table registration time.
- R7. Table columns are arrays. `sales.q1` evaluates to the array of values in the `q1` column of the `sales` table. This requires a new dot-access syntax (`identifier.identifier`) in the lexer, parser, and AST — distinct from the existing `@globals.field` directive syntax.

**Element-wise Arithmetic**

- R8. Binary operations (`+`, `-`, `*`, `/`) between two arrays of the same length produce a new array (element-wise). Mismatched lengths produce a type error diagnostic referencing both operands. E.g., `rates.rate * rates.hc` → array of per-role costs.
- R9. Binary operations between an array and a scalar apply the scalar to every element. E.g., `rates.rate * 1.1` → array of increased rates.

**Aggregate Functions**

- R10. `sum()`, `avg()`, `min()`, `max()`, `count()` accept a single array argument and return a scalar. Note: `min()`, `max()`, and `count()` are entirely new functions that do not exist in the codebase today.
- R11. These functions also retain their existing variadic scalar behavior: `sum(a, b, c)` still works. Mixed array+scalar arguments are a type error — `sum(array)` uses the aggregate path; `sum(scalar, scalar, ...)` requires 2+ arguments (existing behavior). This subsumes the existing `sum()` brainstorm.
- R12. Aggregates compose with element-wise arithmetic: `sum(rates.rate * rates.hc)` computes element-wise product then sums.

**Display and Output**

- R13. Scalar computed results are displayable via existing `{{interpolation}}` syntax.
- R14. Arrays display as comma-separated values in square brackets when shown directly (e.g., in JSON output or when assigned and displayed).
- R15. When `{{array_var}}` appears in a table cell and the variable is an Array type, each data row receives the corresponding array element (row 0 gets element 0, etc.). Array length must match the number of data rows; a mismatch produces a diagnostic.

## Success Criteria

- A consulting SOW with a rates table can compute total cost via `sum(rates.rate * rates.hc)` without duplicating any data between the table and calc blocks.
- Per-row computed values (e.g., `line_cost = rates.rate * rates.hc`) can be displayed in the table via `{{line_cost}}` array interpolation, with each row showing its corresponding value.
- Computed totals can be referenced elsewhere in the document via `{{total_cost}}` interpolation.
- Changing one value in a table (e.g., a rate) recomputes all downstream results correctly.
- Existing CalcMark documents with markdown tables continue to work unchanged (tables without names are inert).
- `sum()` works on both arrays and variadic scalars.

## Scope Boundaries

- **No array literals** — arrays come from tables only. Literal syntax (`[1, 2, 3]`) is a future addition if needed.
- **No indexing or row access** — `arr[0]`, `sales.row("East")`, filtering, slicing are all out of scope.
- **No cell-level formulas** — table cells contain literal data only, not expressions. Computation happens in calc blocks.
- **No CSV/file loading** — data must be inline in the document. File loading (`table sales from "data.csv"`) is a natural future extension but out of scope.
- **No dependency graph execution** — execution remains top-to-bottom. Tables are data declarations, not bidirectional spreadsheet cells.
- **No DuckDB or external query engine** — arrays are native CalcMark values, evaluated by the existing interpreter.

## Key Decisions

- **Directive-based naming**: Table and column names are declared explicitly via `<!-- table: name (col1, col2) -->`. Heading text and header row text are display-only. This eliminates normalization ambiguity — you know exactly what identifier to use in calc blocks because you wrote it. Renaming a display heading or header never breaks references; changing the directive name does, and produces diagnostics on broken references.
- **First-class array type**: Arrays are real values in the type system, not opaque references. This enables element-wise arithmetic and composability with aggregates, which is essential for the consulting SOW case.
- **Strict column typing**: All values in a column must be the same CalcMark type. Mixed-type columns produce a diagnostic at registration. This avoids the "homogeneous-ish" ambiguity and keeps the type system predictable.
- **Tables only (no array literals)**: Keeps the feature grounded in the "document that computes" identity. Arrays serve tables, not general-purpose programming.
- **Top-to-bottom preserved**: No execution model change. Tables register data where they appear; calc blocks reference it below. Interpolation handles display (including existing forward-reference support for scalar results).
- **Element-wise arithmetic in v1**: Without it, users must pre-compute line totals in the table itself, defeating the purpose. `sum(rates.rate * rates.hc)` is the killer expression.
- **Array interpolation in table cells**: When `{{array_var}}` appears in a table column, each row receives the corresponding element. This enables the per-row computed column use case without introducing cell-level formulas or indexing.
- **Broken reference diagnostics**: Renaming headings, columns, or tables that breaks downstream references produces diagnostics, consistent with how undefined variables are handled today.

## Dependencies / Assumptions

- The existing `sum()` brainstorm (`docs/brainstorms/2026-03-09-sum-function-brainstorm.md`) is subsumed by this work. `sum()` should be implemented as part of this feature with both variadic scalar and array support.
- Interpolation (`{{var}}`) already supports forward references for scalar values. Array interpolation in table cells (R15) is a new behavior that extends the existing interpolation system.
- Column value parsing reuses the existing lexer/parser for individual CalcMark literal values.
- Table data cells contain literals only — no `{{var}}` tags in data cells (avoids chicken-and-egg with forward references during table registration).
- The evaluator must be modified to inspect TextBlocks for named table syntax during the evaluation pass, extracting column data and registering it in the environment. This is a change to the current block-type separation where TextBlocks are only processed for interpolation after evaluation.

## Outstanding Questions

### Deferred to Planning

- [Affects R1][Technical] How should collisions be handled when two directives declare the same table name?
- [Affects R5][Technical] Should tables in the same calc block section (between `---` separators) be accessible within that section, or only in subsequent sections?
- [Affects R5][Technical] What happens when a table name collides with an existing variable name — error, shadow, or undefined?
- [Affects R7][Technical] Dot-access syntax requires a new AST node (e.g., `MemberAccess`) distinct from `DirectiveRef`. How should this interact with the existing `@globals.field` syntax?
- [Affects R8][Technical] What happens when element-wise operations produce incompatible types (e.g., `currency_array / currency_array` → plain numbers)? Follow existing CalcMark type arithmetic rules.
- [Affects R6][Needs research] What is the right representation for the Array type — `[]types.Type` or a dedicated struct with metadata (source table, column name)?
- [Affects R5][Needs research] Should table registration happen during document parsing (new block type) or during evaluation? This affects the spec/ vs impl/ dependency boundary.
- [Affects R15][Technical] How does array interpolation in table cells interact with the existing interpolation pass? The interpolation system needs to be table-structure-aware to map array elements to rows.

## Future Considerations

**Computed rows and columns**: v1 requires users to define calculations in calc blocks and use `{{interpolation}}` to display results. A future version could automatically bolt computed values onto tables — if a calc block clearly references a table (e.g., `sum(data.b)`), CalcMark could inject a summary row into the rendered output without the user adding any extra markup. The pre-processing pipeline already transforms TextBlocks before markdown-to-HTML rendering (interpolation does this today), so the architecture supports it. The key principle: avoid inventing CalcMark-specific magic syntax. If the computation is obvious from context, just do it.

**NL syntax for array functions**: `sum of rates.rate` reads naturally, but `sum of rates.rate * rates.hc` is ambiguous. NL syntax for array-accepting functions is deferred until the interaction with element-wise expressions is better understood.

## Next Steps

-> `/ce:plan` for structured implementation planning
