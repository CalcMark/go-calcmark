---
title: "refactor: Architectural Review — Readability, Performance, and Feature Velocity"
type: refactor
status: completed
date: 2026-03-20
origin: docs/brainstorms/2026-03-20-architectural-review-requirements.md
deepened: 2026-03-20
---

# Architectural Review: Readability, Performance, and Feature Velocity

## Enhancement Summary

**Deepened on:** 2026-03-20
**Research agents used:** architecture-strategist, performance-oracle, pattern-recognition-specialist, code-simplicity-reviewer, best-practices-researcher, git-history-analyzer

### Key Changes from Deepening

1. **Reduced from 8 requirements across 4 phases to 5 focused PRs** — simplicity review identified R2 (operator flattening), R3 (function registry), R4 sub-structs, R5 audit document, and R7 audit document as over-engineered for their actual problems
2. **R8 package location corrected** — `spec/completion/` violates the spec/impl boundary because TUI completion imports `impl/interpreter.BuiltinFunctions`. Changed to use `spec/features` as data source exclusively
3. **Critical benchmark gaps identified** — zero benchmarks for `evalBinaryOperation` and zero for rate/capacity parser paths. Must add before any refactoring
4. **Pre-existing bugs found** — dead code in parser (duplicate `QUANTITY`/`CURRENCY_SYM` branches), debug leftover, `autocomplete_handler.go` bypasses mode transitions, `getDurationFactorDecimal` allocates a map per call on the hot path
5. **Confirmed no actual import cycle exists** in `functions.go` — the init() pattern is preventative, not reactive. Direct assignment works.

### Findings That Changed the Plan

| Agent | Key Finding | Impact |
|-------|------------|--------|
| Simplicity | `evalBinaryOperation` max nesting is 4 levels, not 10+ | R2 downgraded from standalone PR to optional cleanup |
| Simplicity | R4 sub-structs add verbosity to 25+ files for no demonstrated bug | Dropped sub-structs, kept catwalk splitting |
| Architecture | `spec/completion/` violates boundary — TUI imports `impl/interpreter` | Changed R8 data source strategy |
| Architecture | `parseUnary` + `parseFunctionCall` should move with `parsePrimary` | R1 extraction scope expanded |
| Architecture | Dead code: duplicate QUANTITY/CURRENCY_SYM branches in parsePrimary | Cleanup added to R1 |
| Performance | Zero benchmarks for evalBinaryOperation — cannot detect regressions | Benchmark PR added as prerequisite |
| Performance | `getDurationFactorDecimal` allocates 14-entry map per call | Quick fix added |
| Patterns | No actual import cycle — init() pattern is unnecessary | R3 simplified to one-liner |
| Patterns | `autocomplete_handler.go` bypasses mode transitions 3 times | Pre-existing bug flagged |
| Patterns | `strings.HasPrefix(strings.ToLower(...))` repeated 26 times | Helper extraction added to R8 |

---

## Overview

Systematic refactoring of go-calcmark's critical-path code to improve feature velocity and maintenance health. Five focused PRs target the files that every new language feature touches.

**Hard constraint: zero regression.** No behavioral changes to the interpreter, TUI, LSP, Go library, or site. No release required. Every PR must leave `task test` and `task quality` passing with identical golden file output.

**Primary pain point: TUI testability.** Catwalk tests are valuable but overly complex. The target is one focused expectation per test file.

(see origin: docs/brainstorms/2026-03-20-architectural-review-requirements.md)

## Problem Statement / Motivation

The codebase is well-structured with clean spec/impl separation and 272 test files. However:

1. `spec/parser/rdparser.go` (1,592 lines) is on the critical path for every new feature. Its size slows development and review.
2. TUI autocomplete and LSP completion duplicate ~450 lines of logic with a known bug (LSP variable filtering).
3. TUI catwalk tests are too complex — multiple unrelated expectations per file.
4. Several small performance and hygiene issues accumulate: uncached registries, per-call map allocations, dead code, mode transition bypasses.

## Proposed Solution

Five independent PRs, each a pure refactoring with zero behavioral changes. No phasing required — they are independent and can be done in any order, though benchmark and hygiene work should land first.

---

## Technical Approach

### PR 0: Benchmark Coverage + Quick Fixes (Prerequisite)

**Goal:** Close critical benchmark gaps and fix pre-existing bugs before any structural refactoring.

**Benchmark gaps (from performance oracle):**

The parser has 9 benchmarks but zero coverage for rate syntax, capacity syntax, unit conversion, or `as` keyword — the most complex paths in `parseMultiplicative` (the R1 extraction target). The interpreter has 3 unit-conversion benchmarks but zero for `evalBinaryOperation` (the hottest path).

**New parser benchmarks to add** (`spec/parser/benchmark_test.go`):

```go
func BenchmarkParseRateSlash(b *testing.B) {
    input := "100 MB/s\n"
    for b.Loop() { _, _ = parser.Parse(input) }
}

func BenchmarkParseRatePer(b *testing.B) {
    input := "5 GB per day\n"
    for b.Loop() { _, _ = parser.Parse(input) }
}

func BenchmarkParseUnitConversion(b *testing.B) {
    input := "10 meters in feet\n"
    for b.Loop() { _, _ = parser.Parse(input) }
}

func BenchmarkParseCapacity(b *testing.B) {
    input := "10 TB at 2 TB per disk\n"
    for b.Loop() { _, _ = parser.Parse(input) }
}
```

**New interpreter benchmarks to add** (new file `impl/interpreter/operator_benchmark_test.go`):

```go
func BenchmarkEvalBinaryOp_NumberNumber(b *testing.B) { /* 42 + 17 */ }
func BenchmarkEvalBinaryOp_CurrencyNumber(b *testing.B) { /* $99.99 * 1.08 */ }
func BenchmarkEvalBinaryOp_PercentageWidening(b *testing.B) { /* 1000 + 20% */ }
func BenchmarkEvalBinaryOp_QuantityQuantity(b *testing.B) { /* 100m + 50m */ }
```

**Quick fixes (5 minutes each):**

1. **Hoist `getDurationFactorDecimal` map to package-level var** (`impl/interpreter/operators.go:697`):
   ```go
   // Before: allocates 14-entry map on every duration operation
   func getDurationFactorDecimal(unit string) decimal.Decimal {
       factors := map[string]decimal.Decimal{...}

   // After: single allocation at package init
   var durationFactors = map[string]decimal.Decimal{...}
   func getDurationFactorDecimal(unit string) decimal.Decimal {
       if f, ok := durationFactors[unit]; ok { return f }
   ```

2. **Cache `features.NewRegistry()` as singleton** — 16 non-test call sites recreate the registry (immutable data) on every request. The LSP path calls it 3 times per keystroke (~10-20KB allocation per keystroke for GC):
   ```go
   // spec/features/registry.go
   var (
       defaultRegistry     *Registry
       defaultRegistryOnce sync.Once
   )
   func DefaultRegistry() *Registry {
       defaultRegistryOnce.Do(func() { defaultRegistry = NewRegistry() })
       return defaultRegistry
   }
   ```
   Keep `NewRegistry()` public for tests. Migrate call sites to `DefaultRegistry()`.

3. **Simplify `functions.go` init() to direct assignment** — pattern-recognition confirmed no actual import cycle exists. The three-structure indirection (`BuiltinFunctions` + `functionEvalMap` + `init()`) can be replaced with direct assignment:
   ```go
   var BuiltinFunctions = []FunctionDef{
       {Name: "avg", Eval: evalAvgFunc},
       {Name: "sum", Eval: evalSumFunc},
       // ...
   }
   ```
   Delete `functionEvalMap` and `init()`.

4. **Add import constraint test** — permanent protection that `spec/` never imports `impl/`:
   ```go
   func TestSpecNeverImportsImpl(t *testing.T) {
       // go list -json ./spec/... | verify no impl/ imports
   }
   ```

**Scope:** Small. Single PR. ~200 lines of new benchmarks + ~50 lines of fixes.

**Verification:** `task test`, `task quality`, `task bench` (capture baseline).

---

### PR 1: Parser Decomposition

**Goal:** Split `spec/parser/rdparser.go` (1,592 lines) so adding a new NL function or expression form requires reading only the relevant module.

**Decomposition (refined by architecture review):**

| New file | Content | Lines |
|----------|---------|-------|
| `multiplicative.go` | `parseMultiplicative` and its sub-parsers (rate, capacity, unit conversion) | ~344 |
| `primary.go` | `parsePrimary`, `parseUnary`, `parseFunctionCall`, `maybeCompoundModifier`, `parseFromTarget` | ~580 |
| `rdparser.go` (remaining) | Infrastructure, precedence climbing backbone (`parseProgram` → `parseComparison`), statement parsing | ~670 |

**Changes from original plan:**

- **`parseUnary` moves with `parsePrimary`** (architecture review finding) — it chains directly into `parsePrimary` and its postfix logic (`as napkin`, `as <unit>`) is primary-adjacent. Leaving it in the backbone would orphan domain logic.
- **`parseFunctionCall`, `maybeCompoundModifier`, `parseFromTarget` move with `parsePrimary`** — called exclusively from within it.
- **File naming follows existing convention** without `rdparser_` prefix — consistent with existing `nl_functions.go`, `nl_growth_functions.go`, `rate_helpers.go`.

**Dead code to remove during extraction:**

- Duplicate `QUANTITY` match block at lines 1157-1185 (unreachable — first match at line 1110 consumes the token)
- Duplicate `CURRENCY_SYM` match block at lines 1141-1152 (unreachable — first match at line 1090 consumes the token)
- Debug leftover at lines 533-538 (`if nextIdent == "downtime" { _ = nextIdent }`)

**Research insight:** Go compiles all files in a package into a single compilation unit. The generated machine code is identical regardless of source file boundaries. The parser package's compiled code is far smaller than L1 instruction cache (192KB on Apple M-series). Zero performance risk from file splitting.

**Scope:** Medium. Single PR. Move methods between files within the same package — no API changes.

**Verification:**
- `task test` — all 272 test files pass
- Golden file byte-for-byte match (do NOT regenerate — per learning `unified-feature-registry-three-to-one.md`)
- `task bench` — parser benchmarks show no regression (run with `-count=6` for `benchstat` reliability)
- Fuzz tests pass: `go test ./spec/parser/ -fuzz=. -fuzztime=30s`

---

### PR 2: Catwalk Test Splitting + TUI Bug Fixes

**Goal:** Make catwalk tests focused (one expectation per file) and fix pre-existing TUI bugs.

This replaces the original R4 (TUI model sub-struct decomposition) and R5 (test audit). The simplicity review found that sub-structs would add verbosity to 25+ files for no demonstrated bug, and a full test audit produces a stale spreadsheet. What actually matters: focused tests and fixing real bugs.

**Tasks:**

1. **Split complex catwalk test files** into focused single-expectation files:
   - Each file tests ONE user action → ONE observable outcome
   - Name files after the user expectation: `autocomplete_shows_functions_on_prefix.txtar`
   - Use the simplest observer that proves the expectation (`results` > `debug` > `view`)
   - `view` observer assertions are acceptable when testing rendering behavior, but avoid them for logic tests

2. **Fix `autocomplete_handler.go` mode transition bypass** (P0 from pattern recognition):
   - Lines 116-160 directly set `m.mode = StateDefault` (3 times) and `m.mode = StateAutocomplete` (1 time)
   - Should use centralized `exitAutocomplete()` / enter methods from `mode_transitions.go`
   - Direct `m.mode = StateDefault` skips the `m.autocompleteState` reset that `exitAutocomplete()` performs

3. **Document test rubric in CONTRIBUTING.md** (replaces the R5 audit):
   - **Behavioral test:** Would still catch a bug if production code were rewritten differently
   - **Implementation-coupled test:** Would break on a correct refactoring
   - Prefer `results` observer for catwalk tests (tests user-visible output, not internal state)
   - Apply rubric going forward — no retroactive audit needed

**Scope:** Medium. Single PR.

**Verification:**
- Snapshot all catwalk expected output before starting. After splitting, diff must be empty.
- `task test`, `task quality`

---

### PR 3: Idiomatic Go + Operator Cleanup

**Goal:** Apply safe `gopls modernize` suggestions and clean up operator code that materially hurts readability.

This merges the original R2 (operator flattening) and R6 (idiomatic Go). The simplicity review found that `evalBinaryOperation`'s actual max nesting is 4 levels (not 10+), so a full extract-and-dispatch refactoring is overkill. However, there are genuine improvements:

**Tasks:**

1. **Run `gopls modernize` in check-only mode** — apply safe subset:
   - `slices.Contains` instead of manual loops
   - Range-over-int where applicable
   - **Skip** `maps.Keys`/`maps.Values` — per learning `go-maps-non-deterministic-ordering-frontmatter.md`, 18 iteration sites are order-sensitive
   - **Skip** files touched by PR 1 to avoid merge conflicts

2. **Extract `evalUnaryOperation` helper** (P1 from pattern recognition):
   - 6 nearly identical type-assertion blocks (~80 lines) follow the same pattern
   - Extract to ~20 lines with a generic helper or per-type functions matching existing `evalNumberOperation` convention

3. **Hoist any remaining per-call allocations** found during `gopls` sweep

4. **Assess `napkin_eval.go` decimal→float64→decimal round-trip** — is it necessary or can decimal rounding replace it? Document finding either way.

**What this PR does NOT do (simplicity review rationale):**

The original R2 proposed extracting `evalBinaryOperation` into 8-10 single-call-site helper functions (`evalBooleanOp`, `evalNumberBinaryOp`, `evalCurrencyOp`, etc.). The simplicity reviewer read the actual code and found:
- The type dispatch (lines 164-378) is a flat sequence of `if leftX, ok := left.(*types.X)` blocks at nesting level 2-3 for most cases
- Deepest nesting is 4 levels (Currency-Currency special case)
- Extracting 8-10 functions that each get called from exactly one place would increase total LOC and decrease locality
- The code is already well-commented with clear phase separation

If after the benchmark PR you find `evalBinaryOperation` is a performance bottleneck or readability blocker for a specific feature, revisit with data. The architecture reviewer's extract-and-dispatch approach is sound *if* the nesting is actually a problem — the evidence says it isn't.

**Key constraint:** If extracting helpers, normalization-phase recursive calls must stay as `evalBinaryOperation(...)` calls, NOT calls to extracted type-specific helpers. They re-enter the full normalization pipeline intentionally.

**Scope:** Medium. Single PR. Exclude files touched by PR 1.

**Verification:** `task quality`, `task test`, `task bench` (compare against PR 0 baseline).

---

### PR 4: Completion Logic Unification

**Goal:** Eliminate ~450 lines of duplicated completion logic between TUI and LSP. Fix the LSP variable filtering bug.

**Design (revised per architecture review):**

The original plan proposed `spec/completion/` as the package location. The architecture reviewer identified that this **violates the spec/impl boundary** — the TUI's current `FunctionSuggestionSource` imports `impl/interpreter.BuiltinFunctions`. To place the shared code in `spec/`, the unified implementation must use only `spec/features.Registry` as its data source (which is the LSP's current approach and the architecturally correct one).

**Revised design — plain functions, no interfaces (per simplicity review):**

Create shared completion functions in `spec/features/` (extending the existing `suggestions.go` that already defines `Suggestion` and `SuggestionSource`):

```go
// spec/features/completion.go — new file
func FunctionSuggestions(registry *Registry, prefix string) []Suggestion
func UnitSuggestions(prefix string) []Suggestion
func VariableSuggestions(vars map[string]VariableInfo, prefix string, cursorLine int) []Suggestion
func DirectiveSuggestions(frontmatter FrontmatterProvider, prefix string) []Suggestion
func ExtractPrefix(line string, col int) string
```

**Why `spec/features/` instead of `spec/completion/`:**
- `spec/features/suggestions.go` already defines the `Suggestion` type
- The data source is `spec/features.Registry` (no `impl/` imports needed)
- Adding a new package for 5 functions is over-engineering — extend the existing package
- The LSP already uses `features.NewRegistry()` as its sole data source; the TUI must switch from `interpreter.BuiltinFunctions` to `features.DefaultRegistry()` (from PR 0)

**Additional cleanup:**
- Extract `matchesPrefix(s, lowerPrefix string) bool` helper — `strings.HasPrefix(strings.ToLower(...))` repeated 26 times across TUI and LSP completion (pattern recognition finding)
- Fix LSP `variableCompletionItems()` — `cursorLine` parameter is accepted but never used for filtering. Port the TUI's correct filtering logic.

**TUI adapter changes:**
- Replace `FunctionSuggestionSource`, `UnitSuggestionSource`, `VariableSuggestionSource`, `DirectiveSuggestionSource` with calls to shared functions
- Keep `CombinedSuggestionSource` for TUI-specific sorting and context suppression

**LSP adapter changes:**
- Replace `functionCompletionItems()`, `unitCompletionItems()`, `variableCompletionItems()`, `directiveCompletionItems()` with shared function calls + conversion to `protocol.CompletionItem`
- Keep LSP-specific enrichment (snippets, parameter docs, markdown documentation) in the adapter

**Scope:** Medium. Single PR.

**Verification:**
- Side-by-side test: for a fixed set of prefixes, old TUI and old LSP produce the same items as unified version
- LSP `consistency_test.go` passes
- TUI autocomplete catwalk tests pass
- `task test`, `task quality`

---

## System-Wide Impact

### Interaction Graph

All changes are internal refactoring. No public API changes, no new features.

- **Interpreter output:** Identical for all inputs
- **Parser AST:** Identical for all inputs (dead code removal has no behavioral effect)
- **TUI rendering:** Identical for all inputs
- **LSP protocol:** Identical except R8 fixes the variable filtering bug (intentional improvement)
- **Go library API:** `calcmark.Eval()`, `calcmark.Convert()`, `calcmark.Session` — no changes

### State Lifecycle Risks

PR 2 (catwalk splitting + mode transition fix) is the only PR that touches TUI state management. The mode transition fix is a correctness improvement, not a refactoring — it routes through existing centralized transition methods instead of bypassing them.

### API Surface Parity

No API surface changes. The only behavioral change is fixing the LSP variable completion bug.

## Acceptance Criteria

### PR 0 (Benchmarks + Quick Fixes)
- [ ] Parser benchmarks added for rate, capacity, unit conversion paths
- [ ] Interpreter benchmarks added for `evalBinaryOperation` type combinations
- [ ] `getDurationFactorDecimal` map hoisted to package-level var
- [ ] `features.DefaultRegistry()` singleton added, call sites migrated
- [ ] `functions.go` init() replaced with direct assignment
- [ ] Import constraint test added for spec/impl boundary
- [ ] `task test`, `task quality`, `task bench` pass

### PR 1 (Parser Decomposition)
- [ ] `rdparser.go` reduced to ~670 lines
- [ ] `primary.go` contains `parsePrimary`, `parseUnary`, `parseFunctionCall`, `maybeCompoundModifier`, `parseFromTarget`
- [ ] `multiplicative.go` contains `parseMultiplicative`
- [ ] Dead code removed (duplicate QUANTITY/CURRENCY_SYM branches, debug leftover)
- [ ] All parser golden tests pass byte-for-byte
- [ ] Parser benchmarks show no regression (benchstat, -count=6)

### PR 2 (Catwalk Tests + TUI Bug Fixes)
- [ ] Complex catwalk files split into single-expectation files
- [ ] `autocomplete_handler.go` mode transitions routed through `mode_transitions.go`
- [ ] Test rubric documented in CONTRIBUTING.md
- [ ] All catwalk expected output unchanged (except mode transition fix if it changes `debug` observer output)

### PR 3 (Idiomatic Go + Operator Cleanup)
- [ ] Safe `gopls modernize` suggestions applied
- [ ] `evalUnaryOperation` duplication reduced
- [ ] `task quality`, `task test`, `task bench` pass with no regression

### PR 4 (Completion Unification)
- [ ] Shared completion functions in `spec/features/completion.go`
- [ ] LSP variable filtering bug fixed
- [ ] `matchesPrefix` helper extracted (replaces 26 call sites)
- [ ] TUI and LSP produce identical suggestions for identical inputs

### Quality Gates (every PR)
- [ ] `task test` passes
- [ ] `task quality` passes
- [ ] Golden files unchanged (byte-for-byte) — exception: dead code removal in PR 1
- [ ] No new exported symbols unless justified
- [ ] PR is reviewable in one sitting (< 500 lines preferred, < 1000 max)
- [ ] Benchmarks run with `-count=6` for reliable `benchstat` comparison

## Success Metrics

- **Feature velocity:** After PR 1, adding a new parser feature requires reading <700 lines of context instead of 1,592
- **TUI test clarity:** After PR 2, every catwalk test file has a name describing the single expectation it tests
- **Completion parity:** After PR 4, new completable features are added in one place, not two
- **Zero regression:** `task test`, `task quality`, and `task bench` show no degradation across all PRs

## Dependencies & Prerequisites

- `task test` and `task quality` must pass on main before starting
- Go 1.25.1 (confirmed) — all modern idiom suggestions apply
- PR 0 (benchmarks) must land before PR 1 and PR 3 to establish baselines

## Risk Analysis & Mitigation

| Risk | Impact | Likelihood | Mitigation |
|------|--------|------------|------------|
| PR 1 dead code removal changes behavior | Subtle parsing bugs | Very Low | Dead branches are unreachable (token consumed by first match) — verify with golden tests |
| PR 2 mode transition fix changes TUI behavior | Autocomplete state inconsistency | Low | The fix routes through existing methods — behavior should improve, not change |
| PR 3 modernize changes behavior | Subtle semantic changes | Low | Apply only non-behavioral changes; skip maps.Keys; review each diff |
| PR 4 data source switch (BuiltinFunctions → Registry) | Missing functions in TUI completion | Medium | Side-by-side test comparing old and new suggestion lists for a set of prefixes |

## What Was Dropped from the Original Plan (and Why)

| Original | Decision | Rationale |
|----------|----------|-----------|
| R2: Full evalBinaryOperation extract-and-dispatch | Downgraded to optional cleanup in PR 3 | Simplicity review: actual max nesting is 4 levels, not 10+. Extract-method would create 8-10 single-call-site functions, increasing LOC and decreasing locality. |
| R3: Function registry rewrite | Absorbed into PR 0 as a one-liner | Pattern recognition confirmed no actual import cycle. Direct assignment works. 8 lines of standard Go init() is not a contributor onboarding problem. |
| R4: TUI Model sub-struct decomposition | Dropped entirely | Sub-structs would add `m.cursor.line` verbosity to 25+ files. Model fields are already well-grouped with section comments. No bug demonstrates missing invariant enforcement. |
| R5: Full test audit spreadsheet | Replaced with test rubric in CONTRIBUTING.md | One-time audit document goes stale on next commit. The rubric is the valuable part — apply it going forward. |
| R7: Written findings document with dependency diagram | Replaced with permanent import constraint test | A static analysis document provides no ongoing protection. A test that fails on violation does. |
| 4-phase execution ordering | Dropped | The surviving PRs are independent. No dependency graph needed. |

## Sources & References

### Origin

- **Origin document:** [docs/brainstorms/2026-03-20-architectural-review-requirements.md](docs/brainstorms/2026-03-20-architectural-review-requirements.md) — Key decisions: incremental over big-bang; feature velocity as north star; zero regression constraint.

### Agent Review Findings

- **Architecture strategist:** R8 package location violates spec/impl boundary; parseUnary should move with parsePrimary; dead code in parsePrimary; R2 recursion safety caveat
- **Performance oracle:** Critical benchmark gaps; getDurationFactorDecimal hot-path allocation; registry caching highest-ROI; use plain functions not closures for R2 helpers; value-embedded sub-structs
- **Pattern recognition:** No actual import cycle in functions.go; autocomplete_handler bypasses mode transitions; evalUnaryOperation duplication; strings.HasPrefix(strings.ToLower()) repeated 26 times; duplicate QUANTITY/CURRENCY_SYM dead code confirmed
- **Code simplicity:** R2/R3/R4-substruct/R5-audit/R7-audit are over-engineered; actual eval nesting is 4 levels not 10+; plan reduced from 8 requirements to 5 PRs

### Institutional Learnings

- `docs/solutions/code-organization/unified-feature-registry-three-to-one.md` — Single source of truth by construction; never regenerate golden files
- `docs/solutions/code-organization/split-view-go-into-cohesive-modules.md` — "One rectangular region = one file"
- `docs/solutions/test-failures/test-behavior-not-implementation.md` — Test rubric: "If I rewrote the production code, would this test still catch a bug?"
- `docs/solutions/logic-errors/go-maps-non-deterministic-ordering-frontmatter.md` — 18 order-sensitive map iteration sites; skip maps.Keys
- `docs/solutions/feature-gaps/rate-type-arithmetic-widening.md` — Rate widening is intentionally asymmetric; guard recursion
- `docs/solutions/language-features/directive-as-value-cross-layer-learnings.md` — Classifier is #1 missed layer
- `docs/solutions/ui-bugs/tui-mode-transitions-formatter-indexing-and-bracketed-paste-fixes.md` — Mode transitions centralized in mode_transitions.go

### Internal References

- `spec/parser/rdparser.go` — Parser decomposition target (PR 1)
- `impl/interpreter/operators.go` — Operator cleanup target (PR 3)
- `impl/interpreter/functions.go` — init() simplification (PR 0)
- `spec/features/suggestions.go` — Existing Suggestion type (PR 4 builds on this)
- `cmd/calcmark/tui/editor/autocomplete.go` — TUI autocomplete (PR 4)
- `lsp/completion.go` — LSP completion (PR 4)
- `cmd/calcmark/tui/editor/autocomplete_handler.go` — Mode transition bypass (PR 2)
- `cmd/calcmark/tui/editor/TESTING.md` — Catwalk test documentation
- `CONTRIBUTING.md` — Development workflow, performance targets
