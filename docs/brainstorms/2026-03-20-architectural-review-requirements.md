---
date: 2026-03-20
topic: architectural-review
---

# Architectural Review: Readability, Performance, and Feature Velocity

## Problem Frame

go-calcmark is a well-structured interpreted language with clean spec/impl separation, strong test coverage (272 test files), and thoughtful interface design. However, several files on the critical path for new language features have grown large enough that their cognitive load slows both feature development and maintenance. The goal is a systematic review that identifies concrete, scoped refactoring opportunities — prioritized by impact on feature velocity and maintenance health equally.

## Requirements

### R1. Parser Decomposition Review

Assess `spec/parser/rdparser.go` (1,592 lines) for decomposition into focused modules. The file currently combines three distinct parsing modes — standard expressions, natural language functions, and growth functions — in a single recursive descent parser. Every new language feature requires understanding the entire file.

**Success looks like:** A clear recommendation for how to split the parser so that adding a new NL function or expression form requires reading only the relevant module, without breaking the recursive descent flow or introducing performance overhead.

**Review scope:**
- Map which functions belong to which parsing mode
- Identify shared state vs mode-specific state
- Assess whether `nl_functions.go` (177 lines) and `nl_growth_functions.go` (222 lines) already represent partial extraction and whether rdparser.go should follow the same pattern
- Evaluate the adapter pattern in `adapter.go` — is it pulling its weight?
- Check operator precedence climbing for unnecessary complexity
- Flag any cleverness in the parser that exists for its own sake

### R2. Operator Evaluation Flattening Review

Assess `impl/interpreter/operators.go` (789 lines), specifically `evalBinaryOperation` (~330 lines, 10+ nesting levels). This is the hottest path in the interpreter and the most complex single function. Every type combination (Number, Currency, Quantity, Fraction, Rate, Percentage, Duration, Date) adds another branch.

**Success looks like:** A recommendation for reducing nesting depth to ~5 levels maximum while preserving O(1) performance. Prefer Martin Fowler-style "Replace Conditional with Polymorphism" or dispatch-table approaches over clever abstractions.

**Review scope:**
- Map the type-pair matrix (which combinations are handled, which error)
- Identify repeated patterns across type pairs (e.g., "extract numeric value, operate, re-wrap")
- Assess whether a dispatch table `map[typePair]evaluator` would be clearer without performance regression
- Check for percentage widening and rate widening logic — is it in the right place or should it live in a pre-processing step?
- Benchmark the current hot path as a baseline

### R3. Function Registry Simplification Review

Assess `impl/interpreter/functions.go` (672 lines) and its init() cycle-avoidance pattern. The current design uses a `BuiltinFunctions` array with deferred Eval field population to avoid import cycles.

**Success looks like:** A recommendation for a simpler registry pattern that a new contributor can understand in under 2 minutes, without reintroducing import cycles.

**Review scope:**
- Map the actual import cycle that init() avoidance prevents
- Assess whether the cycle can be broken structurally (interface, separate package, dependency inversion)
- Check if `getSynonymMap()` belongs in the registry or in spec/features
- Evaluate whether the function dispatch pattern aligns with how spec/features/registry.go already catalogs functions

### R4. TUI Model State Decomposition Review

Assess `cmd/calcmark/tui/editor/model.go` (821 lines) and its ~25+ fields. State mutations are scattered across editing.go (949 lines), navigation.go (507 lines), key_dispatch.go, and others. Invariants are enforced by comments, not types.

**Success looks like:** A recommendation for grouping related state into sub-structs (CursorState, OverlayState, UndoState) and identifying which invariants can be enforced by Go's type system rather than documentation.

**Review scope:**
- Catalog all fields in Model and group by concern
- Map which files mutate which fields
- Identify implicit invariants (e.g., "cursor must be at valid position") and assess which can become type constraints
- Assess the state machine (EditorState transitions) — is it sound, or are there illegal transitions that compile?
- Check whether key_dispatch.go's switch-based routing would benefit from a key-binding registry
- Do NOT recommend rewriting the TUI — focus on incremental decomposition

### R5. Test Value Audit

Audit test files across all layers for tests that assert implementation details vs. tests that assert expectations.

**Success looks like:** A categorized list of test files with a clear assessment: "high-value behavioral test" vs. "implementation-coupled, consider rewriting." Concrete examples of both patterns from the actual codebase.

**Review scope:**
- Check whether white-box tests (same package, internal state access) in spec/lexer are justified or could be black-box
- Assess `impl/interpreter/all_features_test.go` — is the mega-table pattern helping or hurting?
- Evaluate catwalk test coverage: are catwalk tests testing user-visible behavior or internal model state?
- Check golden test maintenance: are testdata/ files serving as specification documents or just regression locks?
- Identify any tests that would break on a correct refactoring (the definition of implementation-coupled)
- Flag tests with weak assertions (e.g., checking string contains instead of exact match)

### R6. Idiomatic Go Patterns Audit

Sweep the codebase for non-idiomatic Go patterns that hurt readability or deviate from modern Go conventions.

**Success looks like:** A prioritized list of patterns to modernize, with before/after examples, focused on changes that improve readability rather than style for its own sake.

**Review scope:**
- Check for pre-Go-1.21 patterns that have cleaner modern equivalents (per golang.org/x/tools/gopls/internal/analysis/modernize)
- Assess error handling consistency: are errors wrapped with context? Is `fmt.Errorf("...: %w", err)` used consistently?
- Check for unnecessary interface{}/any usage where generics would be clearer
- Assess whether the `shopspring/decimal` usage is consistent or if some code paths use float64 unnecessarily (e.g., napkin_eval.go's decimal->float64->decimal round-trip)
- Check for exported symbols that should be unexported
- Flag any global mutable state

### R7. Separation of Concerns Audit

Verify that the spec/impl boundary is clean and that no spec package has leaked implementation concerns or vice versa.

**Success looks like:** Confirmation that the boundary is sound, OR identification of specific violations with recommended fixes.

**Review scope:**
- Verify no impl package imports spec internals beyond public interfaces
- Check whether `spec/document/frontmatter.go` (889 lines) is doing work that belongs in impl
- Assess whether `spec/document/literal_eval.go` (298 lines) crosses into evaluation territory
- Check the DirectiveResolver interface — is it the right abstraction or a leaky one?
- Verify the format/ package doesn't depend on impl details it shouldn't

### R8. Completion Logic Unification

The TUI autocomplete (`cmd/calcmark/tui/editor/autocomplete.go`, 399 lines + `autocomplete_handler.go`, 340 lines) and LSP completion (`lsp/completion.go`, 523 lines) duplicate ~450 lines of equivalent logic across five areas: function suggestions, unit suggestions, variable suggestions, directive suggestions, and prefix extraction. They use the same data sources but implement the pipeline independently.

**Known bug:** The LSP's `variableCompletionItems()` accepts a `cursorLine` parameter but never uses it — it suggests all variables regardless of position. The TUI correctly filters variables defined after the cursor.

**Success looks like:** A shared `spec/completion/` package that both TUI and LSP consume, with each adapting output to its format. New completable features only need to be added once. The LSP variable filtering bug is fixed by sharing the TUI's correct implementation.

**Review scope:**
- Map the exact duplication: which functions in TUI correspond to which in LSP
- Assess whether `spec/features/suggestions.go` (the shared `Suggestion` type) is the right foundation to build on
- Design the extraction boundary: what goes in spec/completion vs stays in each consumer
- Address the registry caching problem (both call `features.NewRegistry()` on every request)
- Determine whether the LSP's richer output (snippets, parameter docs) should live in the shared layer or the LSP adapter
- Verify the TUI's context-aware suppression (inside function calls) and LSP's lexer-based context classification can coexist

## Success Criteria

- Each requirement produces a concrete, actionable finding (not just "this could be better")
- Findings are prioritized P0/P1/P2 by impact on feature velocity and maintenance health
- Every P0 finding includes a specific refactoring recommendation with estimated scope (small/medium/large)
- Hot-path recommendations include benchmark evidence that performance is preserved or improved
- The review can be executed incrementally — each R can be done independently

## Scope Boundaries

- **Not a rewrite.** Every recommendation must be achievable as an incremental refactoring, mergeable in a single PR
- **Not style policing.** Only flag idiomatic issues that materially affect readability or correctness
- **Not the TUI rewrite conversation.** R4 is about state decomposition, not a new architecture
- **Not documentation.** Don't recommend adding comments to code that should be made self-documenting through refactoring
- **Not new features.** This review produces refactoring opportunities, not feature work
- **Performance baseline required.** Before any hot-path refactoring, benchmark the current state

## Key Decisions

- **Incremental over big-bang:** Each finding should be a standalone PR, not a multi-week refactoring branch
- **Feature velocity is the north star metric:** If a refactoring doesn't make the next language feature easier to add, it's lower priority
- **Tests are part of the architecture:** R5 treats test quality as a first-class concern, not an afterthought

## Dependencies / Assumptions

- Assumes `task test` and `task quality` pass on main before the review begins
- Assumes Go 1.22+ for modern idiom recommendations
- Benchmark baselines should be captured before any refactoring work begins

## Outstanding Questions

### Deferred to Planning

- [Affects R1][Needs research] What is the actual performance cost of splitting the parser into multiple files? Measure with parser benchmarks
- [Affects R2][Needs research] Is a dispatch-table approach for operator evaluation faster, slower, or equivalent to the current nested-if approach? Needs benchmark comparison
- [Affects R3][Technical] Can the function registry import cycle be broken by moving function implementations to a separate package, or does that create worse coupling?
- [Affects R4][Technical] Which catwalk observers (debug vs view) best validate that TUI state decomposition doesn't regress behavior?
- [Affects R6][Needs research] What Go version is the project currently targeting? This affects which modernize suggestions apply
- [Affects R8][Technical] Should the shared completion package return the existing `Suggestion` type or a richer intermediate type that the LSP can convert without losing snippet/parameter information?

## Next Steps

-> `/ce:plan` for structured implementation planning
