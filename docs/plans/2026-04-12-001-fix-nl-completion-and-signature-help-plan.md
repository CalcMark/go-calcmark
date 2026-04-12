---
title: "fix: NL completion items carry wrong functionName + signature help missing for NL syntax"
type: fix
status: active
date: 2026-04-12
origin: https://github.com/CalcMark/go-calcmark/issues/133
---

# fix: NL completion items carry wrong functionName + signature help missing for NL syntax

## Overview

Two LSP bugs prevent clients from providing placeholder suggestions and signature help for natural-language (NL) function syntax. Paren-form (`compound(1000, 5%, 10, monthly)`) works correctly, but NL-form (`compound $1000 by 5% monthly over 10 years`) is broken on both fronts. This fix adds a `FunctionName` field to the `Suggestion` struct and an NL-aware fallback to `extractArgumentContext`.

## Problem Frame

After #131 shipped structured parameter types in LSP completion and signatureHelp, `calcmark-web` deleted its client-side function registry in favor of reading `data.params` from the LSP. This works for paren-form but is broken for NL-form:

1. **Bug 1:** NL completion items have `data.functionName` set to the alias display name (e.g., `"grow by over"`) instead of the canonical name (`"grow"`), and `data.params` is null because `GetFunctionSpec("grow by over")` returns nil.

2. **Bug 2:** `textDocument/signatureHelp` returns null for NL-form syntax because `extractArgumentContext` only scans for parentheses, which NL syntax doesn't use.

## Requirements Trace

- **R1.** Every NL-example completion item carries `data.functionName` == canonical function name (e.g., `"grow"`, not `"grow by over"`)
- **R2.** Every NL-example completion item carries `data.params` populated from the function spec
- **R3.** Functions with synonyms (e.g., `avg`/`average`/`mean`) also carry correct `data.functionName` in signature-form items (latent bug: `s.Name` includes synonym suffix)
- **R4.** `textDocument/signatureHelp` returns a valid `SignatureHelp` response when cursor is inside an NL function call
- **R5.** `activeParameter` correctly tracks which NL argument the cursor is in
- **R6.** Existing paren-form behavior is unchanged for both completion and signatureHelp

## Scope Boundaries

- NL signatureHelp uses a line-text heuristic approach matching against known NL function keywords — not full AST-based resolution
- No changes to the NL parser or evaluator
- No changes to hover behavior (hover already works via AST node ranges)
- No changes to TUI autocomplete (it consumes `Suggestion.Name` for display, which remains correct)

## Context & Research

### Relevant Code and Patterns

- `spec/features/suggestions.go:7-14` — `Suggestion` struct, shared between TUI and LSP. `SortCategory` field is precedent for adding NL-specific fields
- `spec/features/completion.go:89-107` — NL rows set `Name: nl.aliasName` and `InsertText: nl.example`; paren rows set `Name: displayName` and `InsertText: f.Name`
- `lsp/completion.go:96-106` — `functionCompletionItems` uses `s.Name` as canonical, which is wrong for both NL rows and synonym-decorated paren rows
- `lsp/argctx.go:31-95` — `extractArgumentContext` only tracks `(` `)` delimiters
- `lsp/signature.go:60-82` — `signatureHelpHandle` returns nil when `ctx.funcName == ""`
- `lsp/completion_test.go:243-286` — `TestFunctionCompletions_DataFunctionName` tests `"gro"` prefix (no synonyms), masking the bug for functions with synonyms
- `spec/features/registry.go` — NL aliases carry `Parseable: true` and `Example` strings; `NLExample` field as fallback

### Institutional Learnings

- **NL/functional parity** (`docs/solutions/integration-issues/nl-functional-syntax-parity-and-doc-staleness.md`): Every fix is two fixes — NL and functional paths are separate code paths that must both be covered in tests
- **Unified feature registry** (`docs/solutions/code-organization/unified-feature-registry-three-to-one.md`): All metadata must come from `spec/features.Feature`, never from `impl/interpreter`
- **LSP debounce** (`docs/solutions/integration-issues/lsp-debounce-staleness-read-requests.md`): Use `ds.getSource()` for text operations, not debounced snapshot — already correct in current handlers

## Key Technical Decisions

- **Add `FunctionName` field to `Suggestion`** (Option 1 from issue): Safer than repurposing `Name` because `Name` is used for display/filtering in TUI. `InsertText` carries `f.Name` for paren rows but carries the full example for NL rows, so it can't be reused either. The new field is set to `f.Name` for all function suggestion rows.
- **NL signatureHelp via line-text heuristic, not AST**: The signatureHelp handler already operates on line text (via `extractArgumentContext`). Adding a fallback that recognizes NL function keywords from the registry keeps the architecture consistent. AST-based resolution would be more robust but requires threading the evaluated document into the signature handler, which is a larger change.
- **NL `activeParameter` by numeric literal index**: Count numeric literals (matching `(\$?)(\d+(?:\.\d+)?)`) on the line from left to right, and map cursor position to the index of the nearest literal. This mirrors the approach `nlExampleToSnippet` uses for tab stops.

## Open Questions

### Resolved During Planning

- **Should `InsertText` be reused for canonical name?** No — NL rows use `InsertText` for the full example string (e.g., `"grow 100 by 20 over 5 months"`). A dedicated field is needed.
- **Should we fix the paren-form synonym bug too?** Yes — `s.Name` for paren-form includes synonyms (e.g., `"avg (average, mean)"`), making `data.functionName` wrong. The new `FunctionName` field fixes both.

### Deferred to Implementation

- Exact regex pattern for NL numeric literal detection may need tuning for currency prefixes (`$`, `€`) and percentage suffixes (`%`)
- Whether the NL keyword lookup should be cached or recomputed per request — profile if needed

## Implementation Units

- [x] **Unit 1: Add `FunctionName` field to `Suggestion` and populate it**

**Goal:** Carry the canonical function name through the `Suggestion` struct so the LSP can resolve it without guessing.

**Requirements:** R1, R2, R3

**Dependencies:** None

**Files:**
- Modify: `spec/features/suggestions.go`
- Modify: `spec/features/completion.go`
- Test: `spec/features/completion_test.go`

**Approach:**
- Add `FunctionName string` to the `Suggestion` struct with a comment explaining it carries the canonical function name for LSP lookup
- In `FunctionSuggestions`, set `FunctionName: f.Name` on both paren-form rows (line 89) and NL-example rows (line 100)
- No changes needed to other `*Suggestions` functions (units, variables, directives, dates) — they don't carry function metadata

**Patterns to follow:**
- `SortCategory` field on `Suggestion` — precedent for an NL-specific field that only some rows populate

**Test scenarios:**
- Happy path: `FunctionSuggestions("gro", nil)` returns items where every item with `Category == "example"` has `FunctionName == "grow"`
- Happy path: `FunctionSuggestions("av", nil)` returns items where the paren-form item has `FunctionName == "avg"` (not `"avg (average, mean)"`)
- Happy path: `FunctionSuggestions("sum", nil)` returns items where NL-example item has `FunctionName == "sum"`
- Edge case: `FunctionSuggestions("xyz", nil)` returns empty slice — no crash

**Verification:**
- All existing `spec/features/completion_test.go` tests pass
- New tests confirm `FunctionName` is set correctly for both paren and NL rows

- [x] **Unit 2: Use `FunctionName` in `functionCompletionItems`**

**Goal:** Fix the LSP completion handler to resolve canonical name from the new field instead of `s.Name`.

**Requirements:** R1, R2, R3, R6

**Dependencies:** Unit 1

**Files:**
- Modify: `lsp/completion.go`
- Test: `lsp/completion_test.go`

**Approach:**
- In `functionCompletionItems` (line 98), replace `canonical := s.Name` with `canonical := s.FunctionName` when non-empty, falling back to `s.Name` for non-function suggestions
- The rest of the function already uses `canonical` to call `GetFunctionSpec` and populate `data.FunctionName` and `data.Params` — no other changes needed

**Patterns to follow:**
- Existing `TestFunctionCompletions_DataFunctionName` test structure

**Test scenarios:**
- Happy path: `functionCompletionItems("gro")` — NL-example item for grow has `data.functionName == "grow"` and `data.params` with 3 entries
- Happy path: `functionCompletionItems("av")` — paren-form item for avg has `data.functionName == "avg"` and `data.params` populated (variadic `values` param)
- Happy path: NL-example item for avg has `data.functionName == "avg"` and same `data.params`
- Integration: `functionCompletionItems("compou")` — NL items for compound carry `data.functionName == "compound"` and params with 4 entries (principal, rate, periods, period?)
- Edge case: paren-form item for throughput still has `data.params[0].enumValues` populated (regression guard)

**Verification:**
- Existing `TestFunctionCompletions_DataFunctionName` still passes
- New tests cover synonym-decorated functions (avg) and multi-alias NL functions (compound)

- [x] **Unit 3: Add NL-aware fallback to `extractArgumentContext`**

**Goal:** Detect NL function calls on a line when the paren scanner finds nothing, enabling signatureHelp for NL syntax.

**Requirements:** R4, R5, R6

**Dependencies:** None (parallel with Units 1-2)

**Files:**
- Modify: `lsp/argctx.go`
- Test: `lsp/argctx_test.go`

**Approach:**
- Add a new function `extractNLArgumentContext(lineText string, col int) argumentContext` that:
  1. Strips optional assignment prefix (`name = ` or `name=`)
  2. Extracts the first identifier on the expression portion
  3. Looks it up in `types.FunctionSpecs` — if found, it's a potential NL call
  4. Counts numeric literals (regex: numbers with optional `$`/`€` prefix and optional `%` suffix) from left to right in the expression
  5. Maps cursor position to the index of the enclosing or nearest-preceding literal
  6. Returns `argumentContext{funcName: canonicalName, paramIdx: literalIndex}`
- Modify `extractArgumentContext` to try the NL fallback when the paren stack is empty and returns `funcName == ""`
- The NL fallback is conservative: if the first identifier isn't a known function name, return the empty context

**Patterns to follow:**
- `identifierEndingAt` helper in `argctx.go` — rune-aware identifier extraction
- `extractArgumentContext` test table in `argctx_test.go`

**Test scenarios:**
- Happy path: `"grow 100 by 20 over 5 months"` with cursor on `100` → `{funcName: "grow", paramIdx: 0}`
- Happy path: `"grow 100 by 20 over 5 months"` with cursor on `20` → `{funcName: "grow", paramIdx: 1}`
- Happy path: `"grow 100 by 20 over 5 months"` with cursor on `5` → `{funcName: "grow", paramIdx: 2}`
- Happy path: `"goal = compound $1000 by 5% monthly over 10 years"` with cursor on `1000` → `{funcName: "compound", paramIdx: 0}`
- Happy path: `"goal = compound $1000 by 5% monthly over 10 years"` with cursor on `5` (the 5%) → `{funcName: "compound", paramIdx: 1}`
- Happy path: `"goal = compound $1000 by 5% monthly over 10 years"` with cursor on `10` → `{funcName: "compound", paramIdx: 2}`
- Happy path: `"average of 1, 2, 3"` with cursor on `2` → `{funcName: "avg", paramIdx: 1}` (alias lookup needed — `average` is a synonym of `avg`)
- Edge case: `"x = 100 + 200"` — not an NL function call, returns empty context
- Edge case: `"grow(100, 20, 5)"` — paren scanner handles this, NL fallback not invoked
- Edge case: assignment prefix stripped: `"result = grow 100 by 20 over 5 months"` → `{funcName: "grow", paramIdx: 0}` when cursor on `100`
- Error path: unknown identifier: `"foobar 100 by 200"` → empty context

**Verification:**
- All existing `extractArgumentContext` tests pass unchanged
- New NL test cases pass for grow, compound, avg

- [x] **Unit 4: Wire NL signatureHelp through the handler**

**Goal:** Ensure `signatureHelpHandle` returns valid signature help for NL function calls.

**Requirements:** R4, R5, R6

**Dependencies:** Unit 3

**Files:**
- Modify: `lsp/signature.go` (likely no changes needed if Unit 3 is correct — `signatureHelpHandle` already delegates to `extractArgumentContext`)
- Test: `lsp/signature_test.go`

**Approach:**
- If Unit 3 correctly extends `extractArgumentContext` with the NL fallback, `signatureHelpHandle` should work without modification — it already calls `extractArgumentContext` and passes the result to `signatureHelpForFunction`
- This unit is focused on integration testing to confirm the full path works

**Test scenarios:**
- Happy path: `signatureHelpForFunction("grow", 0)` returns help with `activeParameter == 0` and label containing `"grow(amount, increment, periods)"` (already works, regression guard)
- Integration: Mock a full signatureHelp request with NL line `"grow 100 by 20 over 5 months"` and cursor on `20` → response has `activeParameter == 1`
- Integration: NL line `"goal = compound $1000 by 5% monthly over 10 years"` with cursor on `1000` → response has `activeParameter == 0` and label `"compound(principal, rate, periods, period?)"`
- Edge case: NL line with cursor between numeric literals (on keyword `"by"`) → `activeParameter` should be the index of the preceding literal

**Verification:**
- `signatureHelpHandle` returns non-nil for NL function calls
- `activeParameter` correctly tracks position within NL arguments

- [x] **Unit 5: Acceptance tests for NL completion and signatureHelp**

**Goal:** End-to-end tests confirming the full LSP handler chain works for NL syntax.

**Requirements:** R1, R2, R4, R5

**Dependencies:** Units 2, 4

**Files:**
- Test: `lsp/acceptance_test.go`

**Approach:**
- Add acceptance tests using `prepareServerDoc` + `completionAt` and signatureHelp request helpers
- Test the scenarios from the issue's "Expected test scenario" sections

**Patterns to follow:**
- Existing acceptance tests in `lsp/acceptance_test.go`

**Test scenarios:**
- Happy path: Document with `grow 100 by 20 over 5 months`, completion at prefix `"gro"` — NL item has `data.functionName == "grow"` and `data.params` with 3 entries
- Happy path: Document with `average of 1, 2, 3`, signatureHelp with cursor on `2` — returns signature with `activeParameter == 1`
- Integration: Document with `goal = compound $1000 by 5% monthly over 10 years`, signatureHelp with cursor on `1000` — returns `compound` signature with `activeParameter == 0`

**Verification:**
- Acceptance tests pass, confirming full handler chain works for NL syntax

## System-Wide Impact

- **Interaction graph:** `Suggestion` struct is shared between TUI editor and LSP. Adding `FunctionName` is additive — TUI doesn't read it, LSP starts reading it. No breakage.
- **Error propagation:** Both bugs produce nil/empty responses (graceful degradation). The fix adds data where nil existed — no new error paths.
- **State lifecycle risks:** None — all changes are stateless request handlers.
- **API surface parity:** Completion and signatureHelp are the two affected surfaces. Hover is not affected (uses AST nodes, not line-text heuristics).
- **Unchanged invariants:** TUI autocomplete behavior is unchanged. Paren-form completion and signatureHelp behavior is unchanged (regression tested).

## Risks & Dependencies

| Risk | Mitigation |
|------|------------|
| NL numeric literal regex doesn't handle all currency/percentage patterns | Start with common patterns (`$`, `€`, `%`), expand if edge cases appear in testing |
| NL function keyword lookup is O(n) per registry scan | Functions registry is small (~20 entries); cache if profiling shows need |
| Synonym-to-canonical mapping not available in `extractNLArgumentContext` | Use `types.FunctionSpecs` for canonical lookup; use registry `Synonyms` for reverse mapping |

## Sources & References

- Related issue: #133
- Related PR/issue: #131 (shipped `wireParamData` infrastructure)
- Related plan: `docs/plans/2026-04-10-001-feat-lsp-structured-param-types-plan.md`
- Institutional learning: `docs/solutions/integration-issues/nl-functional-syntax-parity-and-doc-staleness.md`
- Institutional learning: `docs/solutions/code-organization/unified-feature-registry-three-to-one.md`
