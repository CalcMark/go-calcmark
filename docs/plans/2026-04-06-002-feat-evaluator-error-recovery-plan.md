---
title: "feat: error recovery in evaluator for multi-diagnostic support"
type: feat
status: completed
pr: https://github.com/CalcMark/go-calcmark/pull/112
date: 2026-04-06
origin: https://github.com/CalcMark/go-calcmark/issues/73
---

# feat: error recovery in evaluator for multi-diagnostic support

## Overview

The CalcMark evaluator currently stops at the first error, meaning the LSP, TUI, and CLI can only surface one diagnostic at a time. This plan adds error recovery so the evaluator continues past failed statements and blocks, tracking errored variables out-of-band and distinguishing cascading errors (hint severity) from root-cause errors. The result: users see all diagnostics simultaneously, with successful computations still showing values.

## Problem Frame

A 50-line CalcMark document with an error on line 3 hides all feedback for the remaining 47 lines. Users must fix errors sequentially instead of seeing the full picture. This was discovered during LSP development (#72) and affects the TUI editor equally. The limitation is in the evaluator, not the downstream consumers.

## Requirements Trace

- R1. Multiple errors in a document are all visible simultaneously
- R2. Variables with successful evaluation show their values even when other lines have errors
- R3. Cascading errors (from errored dependencies) are visually distinct from root-cause errors
- R4. Works in LSP (`cm lsp`), TUI editor, and CLI
- R5. No performance regression for valid documents
- R6. Existing tests continue to pass

## Scope Boundaries

- **In scope:** `Evaluator.Evaluate()`, `EvaluateBlock()`, `EvaluateAffectedBlocks()`, and their consumers (TUI results, CLI eval, formatters)
- **In scope:** `cmd/doceval` (Lark website doc evaluator) and `convert.go` (Go library API) -- both call `Evaluate()` and must handle `ErrPartialEvaluation`
- **Out of scope:** `calcmark.Eval()` public Go API (separate evaluation pipeline that bypasses `Evaluator`, file follow-up issue)
- **Out of scope:** Parse error recovery within the parser itself (if parsing fails, the whole block is errored)
- **Out of scope:** Semantic error recovery within a block (semantic errors still mark the whole block)

## Context & Research

### Relevant Code and Patterns

- `impl/document/evaluator.go` -- Three entry points (`Evaluate`, `EvaluateBlock`, `EvaluateAffectedBlocks`) all share the same stop-on-first-error pattern via `evaluateCalcBlockWithDoc`
- `impl/interpreter/environment.go` -- Simple `map[string]types.Type` for variable bindings. No sentinel/error concept yet
- `spec/document/document.go` -- `Diagnostic` struct already supports `"hint"` severity
- `cmd/calcmark/tui/editor/results.go` -- Already tracks `blockedVars` and `IsBlocked` for cascading errors within a block via string-matching on error messages
- `lsp/server.go:251` -- Already ignores `Evaluate()` error return and collects all block diagnostics
- `lsp/diagnostics.go` -- Already maps `"hint"` severity to `DiagnosticSeverityHint`

### Institutional Learnings

- **Statement index drift** (`docs/solutions/ui-bugs/context-footer-statement-index-drift.md`): 4 prior incidents where source lines and result indices get misaligned. Prevention: use nil placeholders in results slice so `len(results) == len(statements)`. Use separate `resultIdx` counters when iterating source lines vs results
- **Diagnostic pipeline** (`docs/solutions/code-organization/diagnostic-detailed-field-pipeline.md`): Every diagnostic must carry `Detailed` field through the full pipeline (semantic -> document -> calcmark). Three carry sites in the evaluator
- **DocLine line numbers** (`docs/solutions/code-organization/docline-diagnostic-line-numbers.md`): Use `blockLineOffset()` for document-absolute line numbers, `node.GetRange()` for block-relative. Never type-assert to specific AST node types
- **Defense-in-depth** (`docs/solutions/security-issues/nan-inf-panic-yaml-frontmatter-scale.md`): Error recovery means more code paths reach the interpreter. Sentinel detection must happen before crash-prone operations. Fuzz test invariant ("never panic on any input") must hold

## Key Technical Decisions

- **Out-of-band error tracking:** Add `erroredVars map[string]error` to `Environment` rather than a sentinel `types.Type`. This minimizes blast radius -- no changes to the type system, formatters, or transforms. The interpreter checks `env.GetError(name)` in `evalIdentifier` before the main variable lookup
- **Statement-level + block-level recovery:** Continue past failed statements within a block AND continue to next blocks. Use nil placeholders in the results slice to maintain 1:1 alignment with statements (prevents statement index drift)
- **Sentinel return value:** `Evaluate()` returns a new `ErrPartialEvaluation` when some blocks had errors but evaluation continued. Callers use `errors.Is()` to detect this. CLI treats it as exit 1 but still formats output. LSP already ignores the error
- **Cascading error detection:** Use a structured `cascading_error` diagnostic code with `Hint` severity. The `Detailed` field names the root-cause variable. The TUI's `results.go` checks for this code alongside the existing `undefined_variable` string matching
- **Block dirty state:** A block with any errors is NOT marked clean (`SetDirty(false)` only on full success). Partial results are stored but the block remains dirty for re-evaluation
- **`@global`/`@exchange` directives:** If a block has directives that succeed but later statements fail, the directive mutations persist. This is correct -- the directive executed successfully

## Open Questions

### Resolved During Planning

- **Should parse errors allow partial block evaluation?** No. Parse errors mean no valid AST exists. The block gets a diagnostic but no statements can run. Other blocks continue
- **Should semantic errors allow partial interpretation?** No. Semantic errors (redefined variable, type mismatch) currently return before interpretation. Keeping this behavior avoids complexity. Other blocks continue
- **What should `{{var}}` render for errored variables?** Leave as `{{var}}` (unresolved). The variable is not in the environment's `vars` map, so the existing unresolved-variable handling applies naturally
- **Should errored variables count as "defined" for redefinition checks?** Yes. An errored variable was assigned; a later block redefining it is still a redefinition error

### Deferred to Implementation

- Exact error message format for cascading errors (must work with existing `isUndefinedVarError` string matching in TUI, or TUI detection updated simultaneously)
- Performance impact of nil-placeholder results on formatter alignment code (note: `format/align.go:67` already nil-guards results)

## High-Level Technical Design

> *This illustrates the intended approach and is directional guidance for review, not implementation specification. The implementing agent should treat it as context, not code to reproduce.*

```
Evaluate(doc) flow with error recovery:
                                                                    
  ApplyFrontmatter ──error?──> return err (fatal, no recovery)     
        │                                                          
        ▼                                                          
  for each block:                                                  
    ├── TextBlock ──> checkTextBlockForLikelyCalculations (unchanged)
    │                                                              
    └── CalcBlock ──> evaluateCalcBlockWithDoc                     
          │                                                        
          ├── Parse ──error?──> add diagnostic, block has no AST   
          │                     (can't identify vars to mark)      
          │                     continue to next block             
          │                                                        
          ├── Semantic check ──error?──> add diagnostic,           
          │                              mark defined vars errored 
          │                              (extracted from AST)      
          │                              continue to next block    
          │                                                        
          └── Interpret stmt-by-stmt:                              
                │                                                  
                ├── interp.Eval returns CascadingError?            
                │   (detected in evalIdentifier via env.GetError)  
                │   ──> cascading_error diag (hint severity)       
                │       nil placeholder in results                 
                │       mark assigned var errored                  
                │       continue to next stmt    
                │                                                  
                ├── eval error (div/0, type mismatch)?             
                │   ──> eval_error diag                            
                │       nil placeholder in results                 
                │       mark assigned var errored                  
                │       continue to next stmt                      
                │                                                  
                └── success ──> append result                      
                                set var in env                     
                                                                   
  applyTransforms (skip blocks with errors)                        
  interpolateTextBlocks (errored vars stay as {{var}})             
                                                                   
  return ErrPartialEvaluation if any block had errors, else nil    
```

**Environment error tracking:**
```
Environment {
    vars:        map[string]types.Type     // successful values
    erroredVars: map[string]error          // errored variable tracking
    ...
}

SetError(name, err)   -- marks variable as errored
GetError(name) -> (error, bool)  -- checks if variable is errored
HasError(name) -> bool           -- quick check
ClearErrors()                    -- reset for fresh evaluation
```

## Implementation Units

- [ ] **Unit 1: Add error tracking to Environment**

**Goal:** Extend `Environment` with out-of-band error tracking for variables that failed evaluation.

**Requirements:** R1, R2, R3

**Dependencies:** None

**Files:**
- Modify: `impl/interpreter/environment.go`
- Test: `impl/interpreter/environment_test.go`

**Approach:**
- Add `erroredVars map[string]error` field to `Environment`
- Add methods: `SetError(name string, err error)`, `GetError(name string) (error, bool)`, `ClearError(name string)`, `ClearErrors()`
- `ClearError(name)` removes a single variable's error state (needed for incremental re-evaluation via `EvaluateAffectedBlocks` when a previously errored variable is fixed)
- Initialize map in `NewEnvironment()`
- `Clone()` must copy the `erroredVars` map
- `GetAllVariables()` is unchanged -- it returns only successful vars

**Patterns to follow:**
- Existing `vars`/`exchangeRates` map patterns in `environment.go`
- `Clone()` uses `maps.Copy` for both existing maps

**Test scenarios:**
- Happy path: `SetError("x", err)` then `GetError("x")` returns the error and true
- Happy path: `GetError("x")` returns nil, false for non-errored variables
- Happy path: `ClearError("x")` removes a single variable's error; `GetError("x")` returns nil, false after
- Happy path: `ClearErrors()` removes all tracked errors
- Edge case: `SetError` then `Set` (successful value) -- both coexist, `GetError` still returns true (variable is both errored and has a prior value from a different context)
- Edge case: `Clone()` includes errored vars -- mutating clone does not affect original
- Edge case: `GetError` on unset variable returns nil, false

**Verification:**
- All existing environment tests pass unchanged
- New error tracking methods work independently from the vars map

---

- [ ] **Unit 2: Add ErrPartialEvaluation sentinel and cascading_error diagnostic code**

**Goal:** Define the sentinel error type returned by `Evaluate()` and the diagnostic code used for cascading errors.

**Requirements:** R1, R3, R4

**Dependencies:** None (can be done in parallel with Unit 1)

**Files:**
- Modify: `impl/document/evaluator.go` (add sentinel error and diagnostic code constants near top of file)

**Approach:**
- Define `var ErrPartialEvaluation = errors.New("partial evaluation: one or more blocks had errors")` in evaluator.go
- Document `cascading_error` as the diagnostic code for errors caused by referencing an errored variable
- The `Detailed` field on cascading error diagnostics should name the root-cause variable for user-facing display

**Patterns to follow:**
- Existing error patterns in evaluator.go
- Existing diagnostic codes like `"parse_error"`, `"eval_error"` in `evaluateCalcBlockWithDoc`
- `Hint` severity already used for currency disambiguation in semantic checker

**Test scenarios:**
- Happy path: `errors.Is(ErrPartialEvaluation, ErrPartialEvaluation)` returns true
- Happy path: `errors.Is(fmt.Errorf("wrapped: %w", ErrPartialEvaluation), ErrPartialEvaluation)` returns true

**Verification:**
- Sentinel error is importable and works with `errors.Is()`

---

- [ ] **Unit 3: Statement-level error recovery in evaluateCalcBlockWithDoc**

**Goal:** Make the core per-block evaluation continue past failed statements, producing nil placeholders for failed statements and marking errored variables in the environment.

**Requirements:** R1, R2, R3

**Dependencies:** Unit 1, Unit 2

**Files:**
- Modify: `impl/document/evaluator.go` (the `evaluateCalcBlockWithDoc` method, lines 476-637)
- Test: `impl/document/evaluator_test.go`

**Approach:**
- In the statement evaluation loop (lines 585-595), instead of `break` on error:
  - Create a diagnostic (eval_error or cascading_error with appropriate severity)
  - Append nil to results slice as placeholder (maintains 1:1 alignment with statements)
  - If the statement is an assignment, call `env.SetError(varName, err)` to mark the variable as errored
  - Continue to next statement
- Before evaluating each statement, check if any referenced variables are errored (via `env.GetError`). If so, produce a `cascading_error` diagnostic with `Hint` severity, append nil placeholder, mark assigned var as errored, continue
- After the loop, if any statement failed, do NOT call `SetDirty(false)` -- block stays dirty
- Still set `block.SetError()` with the first error for legacy compatibility
- Still store partial results via `block.SetResults(results)` -- results slice now has nil entries for failed statements
- The method keeps its `error` return signature unchanged. It still returns an error for the first failure (used by `block.SetError()` for legacy consumers). Callers in Unit 4 check `block.Error() != nil` after the call to determine if the block had errors, track `hasErrors`, and continue to the next block. No signature change needed
- `SetLastValue` must use the last *non-nil* result, or be skipped entirely when all results are nil, to avoid downstream nil dereferences

**Key concern -- statement index drift:** The results slice MUST have exactly `len(nodes)` entries. Each successful statement appends its result. Each failed statement appends nil. This prevents the 5th instance of statement index drift

**Key concern -- cascading detection in interpreter:** When `interp.Eval` is called for a statement that references an errored variable, the interpreter's `evalIdentifier` needs to check `env.GetError(name)` and return a distinguishable error. This happens naturally if the variable is not in `env.vars` (undefined variable error), but we need the cascading_error distinction. Two options: (a) pre-scan the statement's AST for references to errored vars before calling interp.Eval, or (b) have the interpreter return a typed error that the evaluator can detect. Option (b) is cleaner

**Patterns to follow:**
- Existing diagnostic creation pattern at lines 604-631 (use `node.GetRange()`, set `Line`, `DocLine` via `blockLineOffset`)
- `Detailed` field must be populated per the diagnostic pipeline learning

**Test scenarios:**
- Happy path: Block with `a = 1 / 0; b = 10` -- both get results (nil for a, 10 for b)
- Happy path: Block with `a = 1 / 0; c = a * 2` -- a gets eval_error, c gets cascading_error with hint severity
- Happy path: Block with all successful statements -- unchanged behavior, no regressions
- Edge case: Block where every statement fails -- all nil results, all diagnostics, block stays dirty
- Edge case: Statement referencing multiple errored variables -- one cascading_error diagnostic naming one root cause
- Edge case: Whitespace/empty lines between statements -- nil placeholder not inserted for empty lines (they aren't in the nodes slice)
- Edge case: `@global x = 10` succeeds, next statement fails -- x is still set in frontmatter
- Integration: Diagnostics have correct `Line`, `DocLine`, `Column` values using `node.GetRange()` and `blockLineOffset()`
- Integration: `Detailed` field populated on cascading_error diagnostics

**Verification:**
- `len(block.Results()) == len(block.Statements())` invariant holds for all test cases
- All existing evaluator tests pass (no behavioral change for single-error documents)
- Block with errors has `block.Error() != nil` and `block.IsDirty() == true`

---

- [ ] **Unit 4: Block-level error recovery in Evaluate, EvaluateBlock, EvaluateAffectedBlocks**

**Goal:** Make all three evaluation entry points continue to the next block when a block has errors, collecting all diagnostics across the document.

**Requirements:** R1, R2, R4

**Dependencies:** Unit 3

**Files:**
- Modify: `impl/document/evaluator.go` (`Evaluate` lines 93-137, `EvaluateBlock` lines 193-301, `EvaluateAffectedBlocks` lines 314-337, `evaluateCalcBlockSelective` lines 342-466)
- Test: `impl/document/evaluator_test.go`

**Approach:**
- **`Evaluate()`:** Change the CalcBlock case (lines 118-123) to NOT return on error. Instead, track `hasErrors bool`. After all blocks, return `ErrPartialEvaluation` if `hasErrors` is true, nil otherwise. `applyTransforms` and `interpolateTextBlocks` still run -- transforms already skip errored blocks (`cb.Error() != nil`), and interpolation leaves `{{var}}` unresolved for missing vars. Callers check `block.Error() != nil` after `evaluateCalcBlockWithDoc` returns to determine if the block had errors
- **`EvaluateBlock()`:** Pass 1 (lines 200-260) has two early-return paths that must become non-fatal: parse errors (lines 215-218) and variable redefinition errors (lines 243-247). Both should add diagnostics to the block and continue to the next block rather than returning. Errored variables should still be tracked in `allDefinedVars` (they count as "defined" for redefinition checks, and `cb.Variables()` from dependency analysis is already populated even if evaluation fails)
- **`EvaluateBlock()` pass 2:** `evaluateCalcBlockSelective` (lines 342-466) needs its own statement-level recovery logic parallel to Unit 3. Key difference: it uses a *cloned* environment (`evalEnv`) and selectively writes back only "authoritative" assignments. When a variable is errored, the error must propagate back to the shared `reactiveEnv` via `env.SetError(varName, err)` in the selective writeback loop (lines 454-462), so downstream blocks see the error. On successful re-evaluation of a previously errored variable, call `env.ClearError(varName)` to remove stale error state
- **`EvaluateAffectedBlocks()`:** Continue past block errors (lines 321-326). Track `hasErrors` and return `ErrPartialEvaluation` if any block failed. Since the environment is NOT reset, use `env.ClearError(varName)` when a statement succeeds that was previously errored, to support incremental fixing

**Patterns to follow:**
- The existing block iteration loop structure
- `applyTransforms` already guards on `cb.Error() != nil`

**Test scenarios:**
- Happy path: Document with 3 blocks, block 1 has error, blocks 2-3 are independent -- blocks 2-3 evaluate successfully, all 3 blocks have diagnostics/results
- Happy path: Document with no errors -- returns nil (unchanged behavior)
- Happy path: `Evaluate` returns `ErrPartialEvaluation` when any block has errors
- Edge case: Block 2 references errored variable from block 1 -- block 2 gets cascading_error diagnostics
- Edge case: All blocks have errors -- `ErrPartialEvaluation` returned, all blocks have diagnostics
- Edge case: `EvaluateBlock` pass 1 encounters error in block 1, pass 2 re-evaluates block 2 which depends on block 1's errored variable
- Edge case: `EvaluateAffectedBlocks` with one errored block and one clean block
- Integration: `applyTransforms` skips errored blocks but transforms successful ones
- Integration: `interpolateTextBlocks` resolves `{{var}}` for successful vars, leaves unresolved for errored vars

**Verification:**
- `errors.Is(err, ErrPartialEvaluation)` for documents with errors
- `err == nil` for documents without errors
- All blocks have diagnostics populated regardless of which block failed first
- Existing tests continue to pass

---

- [ ] **Unit 5: Interpreter cascading error detection**

**Goal:** Make the interpreter produce a distinguishable error when a statement references an errored variable, so the evaluator can classify cascading vs root-cause diagnostics.

**Requirements:** R3

**Dependencies:** Unit 1

**Files:**
- Modify: `impl/interpreter/variables.go` (`evalIdentifier`)
- Modify: `impl/interpreter/errors.go` (new file, ~10 lines for `CascadingError` type)
- Test: `impl/interpreter/interpreter_test.go`

**Approach:**
- In `evalIdentifier` (variables.go:22), add a check for `env.GetError(name)` *before* `env.Get(name)`. If errored, return a `CascadingError{VarName: name, Cause: err}` instead of the generic "undefined variable" error
- `CascadingError` is a minimal struct (~5 lines): `VarName string`, `Cause error`, implements `error` interface and `Unwrap() error`
- The evaluator in Unit 3 uses `errors.As(err, &CascadingError{})` to choose between `cascading_error` (Hint) vs `eval_error` (Error) diagnostic codes
- **Alternative considered:** Pre-scanning each statement's AST against `env.erroredVars` using the existing `extractIdentifiers` pattern from `spec/document/deps.go`. Rejected because: (a) `extractIdentifiers` is unexported in the spec package, (b) duplicating AST walking logic adds more code than a 5-line typed error, (c) the interpreter is the natural place to detect "I tried to use a variable and it was errored"

**Patterns to follow:**
- Existing error handling in `evalIdentifier` for undefined variables (variables.go:35)
- The check order matters: errored vars checked first, then `env.Get`, then boolean keywords, then "undefined variable"

**Test scenarios:**
- Happy path: `env.SetError("x", someErr)` then evaluating expression `x + 1` returns `CascadingError` with VarName "x"
- Happy path: Normal variable lookup (not errored) works unchanged
- Edge case: Variable errored but also has a value in vars -- `GetError` takes precedence (errored check is first)
- Edge case: Expression `a + b` where `a` is errored and `b` is valid -- first `CascadingError` (for `a`) surfaces

**Verification:**
- `errors.As(err, &CascadingError{})` correctly identifies cascading errors
- `CascadingError.Unwrap()` returns the original cause
- Non-cascading interpreter errors are unaffected

---

- [ ] **Unit 6: Update TUI cascading error detection in results.go**

**Goal:** Make the TUI's `GetLineResults` detect cascading errors from the new `cascading_error` diagnostic code, not just string-matching on "undefined variable".

**Requirements:** R3, R4

**Dependencies:** Unit 3 (needs cascading_error diagnostics to exist)

**Files:**
- Modify: `cmd/calcmark/tui/editor/results.go`
- Test: `cmd/calcmark/tui/editor/results_test.go` (or catwalk tests in `testdata/`)

**Approach:**
- In the diagnostic-checking code (around lines 126-141, 161-176), add a check for `diag.Code == "cascading_error"` alongside the existing `isUndefinedVarError` check
- When `cascading_error` is detected, set `lr.IsBlocked = true`
- The `blockedVars` map should be seeded from all blocks' errored variables (cross-block), not just tracked incrementally within the current iteration
- Handle nil placeholders in the results slice -- when `stmtResults[stmtIdx]` is nil, treat it as an error line

**Key concern -- statement index drift:** The results slice now has nil entries for failed statements. The existing `stmtIdx < len(stmtResults)` check handles this, but the code that reads `stmtResults[stmtIdx]` must handle nil. Verify that `results.go` checks for nil before calling `.String()` or `.TypeName()` on results

**Patterns to follow:**
- Existing `blockedVars` tracking pattern in results.go
- Existing `diagByLine` map pattern for per-line diagnostics

**Test scenarios:**
- Happy path: Block 1 has root-cause error, block 2 has cascading error -- block 2's error shows as `IsBlocked = true`
- Happy path: Block with `a = 1/0; b = a*2` -- line for `b` shows as blocked
- Edge case: Block with nil result placeholder -- displays error, not a blank value
- Edge case: Multiple cascading errors in different blocks all show as blocked
- Integration: TUI renders cascading errors with muted/hint styling vs prominent error styling for root causes

**Verification:**
- `IsBlocked` is true for cascading errors, false for root-cause errors
- No statement index drift (values align with correct source lines)

---

- [ ] **Unit 7: Update CLI and formatter error handling**

**Goal:** Make `cm eval` handle `ErrPartialEvaluation` by formatting output AND exiting with code 1.

**Requirements:** R4

**Dependencies:** Unit 4

**Files:**
- Modify: `cmd/calcmark/cmd/eval.go`
- Modify: `convert.go` (if applicable)
- Test: `cmd/calcmark/cmd/eval_test.go` (or integration test)

**Approach:**
- In the CLI eval command, check `errors.Is(err, ErrPartialEvaluation)`. If true, format the output normally (partial results + diagnostics are on the blocks), then exit with code 1
- For non-partial errors (frontmatter failure, etc.), keep current behavior (error message, exit 1, no output)
- **Audit all formatters for nil-safety:** `json_formatter.go`, `text_formatter.go`, `html_formatter.go`, `markdown_formatter.go` all iterate block results. With nil placeholders for failed statements, every `results[i].String()` and `results[i].TypeName()` call must guard against nil. Add nil guards where missing
- JSON formatter: blocks with errors should include `diagnostics` in the JSON output. Check existing `block.Error()` handling -- it may already work since blocks now have both results and errors
- `convert.go`: if `ErrPartialEvaluation`, still return the formatted output (with error markers), or return error. Depends on use case -- the Hugo doc evaluator should probably still fail

**Patterns to follow:**
- Existing error handling in `cmd/calcmark/cmd/eval.go`
- JSON formatter's existing `block.Error()` checks

**Test scenarios:**
- Happy path: `cm eval` on document with errors produces formatted output AND non-zero exit
- Happy path: `cm eval --format json` on document with errors produces JSON with both results and diagnostics
- Happy path: `cm eval` on valid document -- unchanged behavior (exit 0, full output)
- Edge case: All blocks errored -- output shows all diagnostics, exit 1
- Error path: Frontmatter error -- still fatal, no output, exit 1

**Verification:**
- CLI exit code is 1 when `ErrPartialEvaluation`
- JSON output includes both results and diagnostics for mixed documents
- Valid documents are completely unaffected

---

- [ ] **Unit 8: Update doceval (Lark website) and convert.go for error recovery**

**Goal:** Make the Lark documentation site evaluator and the Go library `Convert()` API handle `ErrPartialEvaluation` correctly. The Lark site is the primary user-facing surface for CalcMark and a case study for Go API consumers.

**Requirements:** R4

**Dependencies:** Unit 4

**Files:**
- Modify: `cmd/doceval/main.go` (3 call sites: `evaluateFullDoc` line 184, `evaluateBlockProgressive` line 301, `evaluateBlock` line 360)
- Modify: `convert.go` (2 call sites: `convertCM` line 115, `convertEmbeddedBlock` line 219)
- Test: `cmd/doceval/main_test.go` (if exists), `convert_test.go`

**Approach:**
- **doceval:** The three `Evaluate()` call sites each handle errors differently:
  - `evaluateFullDoc` (line 184): Generates full Markdown pages from .cm files. On `ErrPartialEvaluation`, should still format output with partial results -- the doc page should render with visible error markers rather than failing entirely. Log a warning with the filename and error count
  - `evaluateBlockProgressive` (line 301): Progressive evaluation of ```calcmark blocks. On `ErrPartialEvaluation`, return partial results (successful lines get values, errored lines get error text). This maintains the per-line `LineResult` structure
  - `evaluateBlock` (line 355): Standalone block evaluation. On `ErrPartialEvaluation`, return `BlockResult` with both `Lines` (partial results) and per-line errors
- **convert.go `convertCM`** (line 115): On `ErrPartialEvaluation`, still format the output (blocks have partial results) but also return the error to the caller. `calcmark.Convert()` callers need to know evaluation was partial. This is a behavior change: previously it returned `("", error)`, now it returns `(formatted_output, ErrPartialEvaluation)`. Callers can check `errors.Is` to decide whether to use the partial output
- **convert.go `convertEmbeddedBlock`** (line 219): On `ErrPartialEvaluation`, render the block with error markers using the existing `formatEmbeddedBlockError` pattern. The embedded block should show which lines succeeded and which failed
- Both doceval and convert.go currently iterate block results after evaluation -- nil placeholders from Unit 3 must be handled (nil results -> error display or empty)

**Patterns to follow:**
- Existing `BlockResult.Error` field in doceval already supports per-block error reporting
- `formatEmbeddedBlockError` in convert.go already renders error markers in embedded blocks

**Test scenarios:**
- Happy path: doceval processes a page with one errored block and one valid block -- valid block renders correctly, errored block shows diagnostics
- Happy path: convert.go produces HTML with both results and error markers for mixed documents
- Edge case: doceval progressive mode with errored variable carrying across blocks
- Edge case: convert.go embedded mode with ```calcmark block that has cascading errors
- Integration: `task generate-docs` still succeeds with the existing Lark site content (no regressions in docs)

**Verification:**
- Lark site documentation pages render correctly with `task generate-docs`
- No panics from nil result values in doceval's `LineResult` building
- convert.go produces meaningful output (not empty) for documents with partial errors

---

- [ ] **Unit 9: Golden test files and integration validation**

**Goal:** Add comprehensive golden test files covering multi-error scenarios and validate full-stack behavior.

**Requirements:** R1, R2, R3, R5, R6

**Dependencies:** Units 3-8

**Files:**
- Create: `testdata/eval/errors/multi_error_recovery.cm`
- Create: `testdata/eval/errors/cascading_errors.cm`
- Create: `testdata/eval/errors/mixed_success_and_errors.cm`
- Test: `impl/document/evaluator_test.go` (integration tests using golden files)

**Approach:**
- Write golden test files exercising the key scenarios from the issue's acceptance criteria
- Multi-block document: block 1 error, block 2 references block 1, block 3 independent
- Within-block: `a = 1/0; b = 10; c = a * 2; total = a + b + c`
- All-error document: every block has errors
- Run `task test` to validate all existing tests pass
- Run `task quality` for full quality gate
- Benchmark valid document evaluation to confirm no performance regression

**Patterns to follow:**
- Existing golden files in `testdata/eval/errors/` and `testdata/eval/success/`
- Test helper patterns in `evaluator_test.go`

**Test scenarios:**
- Happy path: Multi-block error recovery golden file matches expected diagnostics
- Happy path: Cascading errors golden file shows hint-severity diagnostics for dependent variables
- Happy path: Mixed success/error golden file shows values for successful lines alongside errors
- Integration: All existing golden tests pass unchanged
- Integration: `task test` passes with zero failures
- Integration: `task quality` passes

**Verification:**
- `task test` passes
- `task quality` passes
- No performance regression for valid documents (evaluator still returns nil, no extra work)

## System-Wide Impact

- **Interaction graph:** `evaluateCalcBlockWithDoc` -> `Environment.SetError` -> `evalIdentifier` checks error -> cascading diagnostic -> `results.go` `IsBlocked` detection -> TUI view renders hint-style. LSP path: `Evaluate` -> block diagnostics -> `publishDiagnostics` (no changes needed)
- **Error propagation:** Root-cause errors stay as `error` severity. Cascading errors propagate as `hint` severity with `cascading_error` code. `ErrPartialEvaluation` wraps the summary for callers
- **State lifecycle risks:** The `erroredVars` map on Environment persists across blocks within one `Evaluate()` call. It is cleared in `Evaluate()` when `env` is reset. For `EvaluateBlock`'s two-pass approach, errored vars from pass 1 must be available in pass 2
- **API surface parity:** CLI, TUI, LSP, doceval (Lark site), and `convert.go` all consume the same evaluator. JSON formatter needs to handle blocks with both results and errors. doceval has 3 call sites with different error semantics. `calcmark.Eval()` public API is out of scope (follow-up issue)
- **Integration coverage:** Unit tests alone won't prove the full diagnostic pipeline. Golden tests (Unit 8) validate end-to-end
- **Unchanged invariants:** The `types.Type` interface is NOT modified. The `Diagnostic` struct is NOT modified (uses existing fields). Semantic checker behavior is unchanged. Parser behavior is unchanged

## Risks & Dependencies

| Risk | Mitigation |
|------|------------|
| Statement index drift (5th instance) | Nil placeholders in results slice maintain 1:1 with statements. Explicit test for `len(results) == len(statements)` invariant |
| Panic on nil result in formatters/TUI | Audit every `results[i].String()` and `results[i].TypeName()` call. Add nil guard tests |
| `EvaluateBlock` two-pass complexity | Test both passes independently. Errored vars from pass 1 carry into pass 2 via shared Environment |
| Backwards compatibility for CLI scripts | `ErrPartialEvaluation` ensures non-zero exit code. Scripts checking `$?` see failure as before |
| Lark site doc generation breaks | `task generate-docs` validation in Unit 9. doceval call sites updated in Unit 8 to handle partial results |
| Performance regression from continued evaluation | CalcMark docs are small (<1000 lines). Even evaluating all blocks after errors adds <5ms. Benchmark in Unit 8 confirms |

## Sources & References

- Related issue: #73 (error recovery in evaluator for multi-diagnostic support)
- Related issue: #72 (LSP development, where this limitation was discovered)
- Institutional learning: `docs/solutions/ui-bugs/context-footer-statement-index-drift.md` (4 prior incidents)
- Institutional learning: `docs/solutions/code-organization/diagnostic-detailed-field-pipeline.md`
- Institutional learning: `docs/solutions/code-organization/docline-diagnostic-line-numbers.md`
- Institutional learning: `docs/solutions/security-issues/nan-inf-panic-yaml-frontmatter-scale.md`
