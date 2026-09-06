---
title: "feat: Named Tables and Array Type (refreshed)"
type: feat
status: active
date: 2026-09-05
supersedes: docs/plans/2026-04-06-003-feat-named-tables-and-arrays-plan.md (branch feat/named-tables-and-arrays, docs-only, never implemented)
origin: docs/brainstorms/2026-04-06-named-tables-and-arrays-requirements.md (see its 2026-09-05 addendum)
issue: https://github.com/CalcMark/go-calcmark/issues/118
---

# feat: Named Tables and Array Type

## Overview

A markdown table preceded by `<!-- table: name (col1, col2, …) -->` becomes a
named data source. Calc blocks read its columns as arrays via dot access
(`rates.rate`), combine them element-wise (`rates.rate * rates.hc`), aggregate
them (`sum(...)`, `avg`, `min`, `max`, `count`), and display per-row results
back into the table through `{{array_var}}` interpolation. Requirements R1–R15
are unchanged from April; R16 (LSP column completion) and R17 (cmw consumer
parity) are added by the addendum.

## What changed since the April plan

| April assumption | Today (v2.2.12 + PRs #166–#168) | Effect on the plan |
|---|---|---|
| Flat `.cm` sources only | Embedded mode: markdown with ```` ```cm ```` fences (`NewDocumentEmbedded`) | Tables live in TextBlocks in both modes; extraction is TextBlock-based and needs no dispatch. Golden tests cover both. |
| Most AST nodes had no `Range` | Every primary expression is ranged; runtime errors are positioned | `MemberAccess` needs no special range code; table-extraction diagnostics must set `Column`/`EndColumn`. |
| Fraction was the type exemplar | Period (v2.0, PR #145) is the latest full-stack type | Follow Period's layer touches; the Fraction checklist stays the layer list. |
| Semantic errors aborted the block | Clean statements still run (#113) | A bad `rates.nope` reference no longer hides the rest of the block. |
| LSP completion out of scope | LSP is the editor's source of truth (cmw #35) | R16: column completion after `table.` |
| No consumer of per-row values | cmw renders tables as widgets with cell interpolation | R17: R15 must flow through `InterpolatedSource` so cmw needs no new projection. |

## Key technical decisions (confirmed)

- **`Table` and `Array` are `types.Type` values in the environment.** `env["rates"]` is a `*Table`; `MemberAccess` on it yields the column `*Array`. No parallel storage.
- **Text cells** become a minimal `*types.Text` (display-only, no arithmetic). It exists so a text column can be counted and interpolated; it is not a lexer/parser-level type.
- **`MemberAccess{Object, Field}`** is distinct from `DirectiveRef`. Nested access is rejected at parse time.
- **DOT after identifiers** is emitted by the lexer only when the next character starts an identifier, so `1.5` and `@globals.x` are unaffected.
- **Extraction runs during evaluation** in all three entry points (`Evaluate`, `EvaluateBlock`, `EvaluateAffectedBlocks`) so TUI/LSP incremental passes never hold stale table data.
- **Element-wise dispatch** is one normalization block at the top of `evalBinaryOperation`; scalar paths are untouched.
- **Aggregates** dispatch on the first argument's type: `*Array` → aggregate path; anything else → the existing variadic path. `sum(array)` is legal, so the parser's "sum needs 2 args" rule is relaxed to ≥1 and the arity check moves to the interpreter where the type is known.
- **Array-in-cell interpolation** indexes by data-row position within the *same* markdown table; length mismatch leaves the tag unresolved and adds a diagnostic on the table's first data row.

## Implementation units

Each unit is test-first and lands green on `task test`. Order matters: types → AST → lexer → parser → extraction → checker → interpreter → aggregates → display/interpolation → integration.

1. **Types** — `spec/types/array.go`, `table.go`, `text.go`; `typeName` cases. Tests: homogeneity, empty arrays, header-only tables, column lookup.
2. **AST** — `MemberAccess` node + `ContainsScaleRef` + `SetRangeIfMissing` case.
3. **Lexer** — DOT after identifier when followed by identifier-start; regression for decimals and `@globals.x`.
4. **Parser** — `IDENT DOT IDENT` → `MemberAccess`; nested access rejected; `sum()` arity relaxed; `min`/`max`/`count` are plain builtins (no new tokens needed — functional calls go through `parseFunctionCall` by name).
5. **Extraction** — `impl/document/table_extraction.go`: directive regex, name normalization, column-count check, cell lexing via the parser, homogeneity, duplicate-name and collision diagnostics (with positions); wired into all three evaluator entry points.
6. **Checker** — `MemberAccess` case: the object must be a known name; cannot validate columns at check time (tables register at eval time) so column errors are runtime, positioned at the field.
7. **Interpreter** — `evalMemberAccess`; element-wise + broadcast in `array_ops.go`; length-mismatch and non-table errors positioned.
8. **Aggregates** — array paths for `sum`/`avg`; new `min`/`max`/`count`; registry + `FunctionSpecs` entries.
9. **Display + interpolation** — `Format` case for `*Array` (`[a, b]`) and `*Text`; JSON `populateResult` for arrays (`type: "array"`, `elements: [...]`); dotted refs and per-row array interpolation in `interpolation.go`.
10. **Integration** — classifier/detector recognise `ident.ident`; feature registry entries; golden files (flat + Embedded + errors); `agent-integration.md` and language reference; LSP column completion (R16).

Downstream (separate repos, after the go-calcmark release): cmw named-table badge + column completion pass-through (C8), calcmark-lark prebaked example + dependency bump (C9).

## Risks

| Risk | Mitigation |
|---|---|
| DOT breaks decimal or directive lexing | Emit only after IDENTIFIER and only when followed by identifier-start; lexer regression table. |
| Classifier misses `rates.rate * 2` lines (most-missed layer) | Unit 10 has an explicit classifier test; `task test` after each unit. |
| Stale tables in incremental TUI/LSP evaluation | All three entry points extract; catwalk test edits a table cell and asserts the dependent calc updates. |
| Array in operator dispatch widens blast radius | Single normalization block; scalar tests unchanged. |
| Interpolation length mismatch silently misaligns rows | Mismatch leaves tags unresolved + diagnostic; never partial substitution. |
