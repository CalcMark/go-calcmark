---
title: "refactor: unified feature registry — single source of truth for all function/unit metadata"
type: refactor
status: active
date: 2026-03-17
origin: docs/brainstorms/2026-03-17-unified-feature-registry-brainstorm.md
---

# Unified Feature Registry

## Overview

Consolidate three overlapping function metadata registries into one canonical `Feature` struct in `spec/features/`. Move `ParamSpec`/`ArgType` to `spec/types/`. Slim `impl/interpreter.FunctionDef` to `Name + Eval` only. Delete `spec/semantic/function_types.go`. Every consumer — LSP, TUI, help, docgen, semantic checker — imports one package for metadata.

Nobody is using the Zed/VS Code integrations yet, so radical changes to the LSP layer are fine. Primary goal: consistency, correctness, simplicity in the spec/impl foundation.

(see brainstorm: `docs/brainstorms/2026-03-17-unified-feature-registry-brainstorm.md`)

## Problem Statement / Motivation

Building a single LSP completion item for `accumulate` requires importing three packages:

| Package | Import | What it provides |
|---------|--------|-----------------|
| `impl/interpreter` | `BuiltinFunctions` | Name, signature, description, synonyms |
| `spec/features` | `Registry` | Aliases, NL examples, category |
| `spec/semantic` | `FunctionSpecs` | Param types, valid values, optional/variadic |

Adding a new function means updating all three. The semantic checker in `spec/semantic/checker.go` also **hardcodes function names** (lines 337-389) instead of deriving behavior from `FunctionSpecs` — a fourth drift point.

Seven tests in `function_consistency_test.go` exist solely to catch drift between registries. A single source of truth eliminates the need for all of them.

## Proposed Solution

### Architecture After Refactoring

```
spec/types/
  param_types.go          ← NEW: ParamSpec, ArgType, FunctionSpec (moved from semantic)

spec/units/canonical.go   (unchanged — single source for units)

spec/features/
  registry.go             ← ENRICHED: Feature gains Params, Synonyms
  suggestions.go          (unchanged — SuggestionSource interface)

spec/semantic/
  checker.go              ← UPDATED: imports spec/types for ParamSpec
  diagnostics.go          (unchanged)
  function_types.go       ← DELETED

impl/interpreter/
  functions.go            ← SLIMMED: FunctionDef = Name + Eval only
  registry.go             ← THIN ADAPTER: joins BuiltinFunctions with features.Registry

lsp/                      ← SIMPLIFIED: single import for all metadata
cmd/calcmark/tui/         ← SIMPLIFIED: SuggestionSource reads from Feature
cmd/calcmark/cmd/help.go  ← SIMPLIFIED: single import
```

### Dependency Flow

```
spec/types/param_types.go     (leaf — no internal deps)
       ↑              ↑
spec/features/    spec/semantic/checker.go
       ↑
lsp/  TUI/  help/  docgen/
       ↑
impl/interpreter/functions.go  (Name + Eval only, joins with features for metadata)
```

## Technical Approach

### Phase 1: Move ParamSpec/ArgType to spec/types

**Files changed:**
- `spec/types/param_types.go` — NEW: move `ArgType`, `ParamSpec`, `FunctionSpec`, `ArgTypeExamples`, `GetFunctionSpec()`, `GetExamplesForType()` from `spec/semantic/function_types.go`
- `spec/semantic/function_types.go` — DELETED
- `spec/semantic/checker.go` — update import from `spec/semantic` internal to `spec/types`

**Consumers to update (import path change):**
- `lsp/completion.go` — `semantic.GetFunctionSpec` → `types.GetFunctionSpec`
- `lsp/signature.go` — `semantic.GetFunctionSpec` → `types.GetFunctionSpec`
- `cmd/calcmark/tui/editor/cursor_context.go` — `semantic.GetFunctionSpec`, `semantic.ParamSpec`, `semantic.GetExamplesForType` → `types.*`
- `cmd/calcmark/tui/editor/cursor_context_test.go` — `semantic.ArgType*` → `types.ArgType*`
- `cmd/calcmark/tui/editor/view_footer.go` — `semantic.GetFunctionSpec`, `semantic.GetExamplesForType` → `types.*`
- `impl/interpreter/function_consistency_test.go` — `semantic.GetFunctionSpec`, `semantic.FunctionSpecs` → `types.*`

**Test:** `go test ./...` passes. All existing consistency tests still work (they just import from a different package).

### Phase 2: Enrich Feature with Params and Synonyms

**Files changed:**
- `spec/features/registry.go`:
  - Add `Params []types.ParamSpec` field to `Feature`
  - Add `Synonyms []string` field to `Feature` (runtime-dispatchable names)
  - Populate `Params` in `getFunctions()` and `getGrowthFeatures()` from the data currently in `FunctionSpecs`
  - **This is now THE single definition of each function's metadata**

**Key data merge — for each of the 17 functions, combine:**

| From `BuiltinFunctions` | From `FunctionSpecs` | From current `Feature` | → Unified `Feature` |
|--------------------------|---------------------|------------------------|---------------------|
| `Name` | `Name` | `Name` | `Name` |
| `Description` | — | `Description` | `Description` |
| `Signature` | — | `Syntax` | `Syntax` (keep Feature's field name) |
| `Synonyms` | — | — | `Synonyms` (new field) |
| `Category` | — | `Category` | `Category` |
| — | `Params` | — | `Params` (new field) |
| — | — | `Aliases` | `Aliases` |
| — | — | `NLExample` | `NLExample` |
| — | — | `Example` | `Example` |

**Decision from brainstorm:** Synonyms merge into Aliases long-term (a synonym is just an alias with `Parseable=true`). But for this refactor, keep `Synonyms []string` as a separate field on Feature to minimize interpreter changes. The interpreter dispatches on exact string match; Alias has extra fields. Merging is a follow-up.

**Test:** `TestEveryFunctionHasParams` — every Feature in CategoryFunction has non-empty Params.

### Phase 3: Slim FunctionDef to Name + Eval

**Files changed:**
- `impl/interpreter/functions.go`:
  - Remove `Description`, `Signature`, `Synonyms`, `Category` fields from `FunctionDef`
  - `FunctionDef` becomes: `Name string` + `Eval EvalFunc`
  - `BuiltinFunctions` entries shrink to two fields each
  - **Synonyms for dispatch:** The interpreter's `evalFunctionCall` needs synonym lookup. Add a computed `synonymMap` built at init from `features.Registry` that maps synonym → canonical name.

- `impl/interpreter/registry.go`:
  - `FunctionInfo` struct stays (it's the public API for `GetAllFunctions()` etc.)
  - But it's now built by joining `BuiltinFunctions[i].Name` with `features.Registry.GetByName(name)`
  - Add `GetByName(name string) *Feature` method to `features.Registry`

- `impl/interpreter/function_consistency_test.go`:
  - Delete `TestEveryBuiltinFunctionHasFunctionSpec` (no more FunctionSpecs)
  - Delete `TestEveryFunctionSpecHasBuiltinFunction` (no more FunctionSpecs)
  - Delete `TestArgCountConsistency` (single source)
  - Delete `TestSignatureParamCountMatchesFunctionSpec` (single source)
  - Delete `TestFunctionSpecOptionalFlagMatchesSignature` (single source)
  - **Keep** `TestEveryBuiltinFunctionHasEval` (still validates impl has Eval)
  - **Add** `TestEveryBuiltinFunctionHasFeature` — every Name in BuiltinFunctions has a matching Feature in the registry
  - **Add** `TestEveryFunctionFeatureHasBuiltinFunction` — every CategoryFunction Feature has a matching BuiltinFunctions entry

### Phase 4: Update All Consumers

**LSP (3 files):**
- `lsp/completion.go`:
  - Remove `interpreter.BuiltinFunctions` iteration → use `features.Registry.ByCategory(CategoryFunction)`
  - Remove `semantic.GetFunctionSpec` → use `feature.Params` directly
  - `buildFunctionSnippet` reads from `feature.Params`
  - `buildFunctionDoc` reads from `feature.Params`
  - Single import: `spec/features` (+ `spec/types` for ParamSpec if needed)
- `lsp/hover.go`:
  - Remove `interpreter.BuiltinFunctions` iteration → use `features.Registry`
  - Remove `features.NewRegistry()` call for NL examples (already on Feature)
- `lsp/signature.go`:
  - Remove `interpreter.BuiltinFunctions` iteration → use `features.Registry.GetByName()`
  - Remove `semantic.GetFunctionSpec` → use `feature.Params`

**TUI (3 files):**
- `cmd/calcmark/tui/editor/autocomplete.go`:
  - `FunctionSuggestionSource`: remove `interpreter.BuiltinFunctions` iteration
  - Build suggestions from `features.Registry.ByCategory(CategoryFunction)` only
  - Synonyms come from `feature.Synonyms`, NL examples from `feature.Aliases`
- `cmd/calcmark/tui/editor/cursor_context.go`:
  - `types.GetFunctionSpec()` → `features.Registry.GetByName(name).Params`
  - Or keep using `types.GetFunctionSpec()` (it's in spec/types now, clean dependency)
- `cmd/calcmark/tui/editor/view_footer.go`:
  - Same as cursor_context — use `types.GetFunctionSpec()` or features

**Help (1 file):**
- `cmd/calcmark/cmd/help.go`:
  - Remove `interpreter.GetFunctionsByCategory()` → use `features.Registry.ByCategory()`
  - Category ordering comes from features registry

**Docgen (1 file):**
- `cmd/docgen/main.go` — already uses features registry, minor cleanup

### Phase 5: Address Semantic Checker Hardcoding

**Optional but valuable.** `spec/semantic/checker.go` lines 337-389 hardcode function names to determine which arguments are identifiers vs expressions:

```go
case "rtt", "throughput", "seek":
    // These functions take ONLY identifier arguments — skip all validation
    return
case "convert_rate", "downtime":
    // First argument is an expression, second is an identifier
```

This could be derived from `ParamSpec.Type == ArgTypeString` (identifier args) vs other types (expression args). But this is a separate concern from the registry consolidation and can be a follow-up.

## System-Wide Impact

### Interaction Graph

The registry is read-only metadata. No callbacks, middleware, or event handlers fire. The refactoring changes import paths and data access patterns but not runtime behavior. The interpreter's `evalFunctionCall` dispatch is the only code path where the change has semantic impact (synonym lookup moves from inline array to features-derived map).

### Error Propagation

No change — errors flow through the same diagnostic pipeline. The LSP's diagnostic handling is unaffected.

### State Lifecycle Risks

None — registry data is immutable after init. No persistence, no cleanup needed.

### API Surface Parity

After consolidation, the LSP, TUI, help, and docgen all read the same `Feature` struct. Parity is guaranteed by construction, not by testing.

## Acceptance Criteria

### Functional Requirements

- [ ] `spec/types/param_types.go` contains `ParamSpec`, `ArgType`, `FunctionSpec`, all helper functions
- [ ] `spec/semantic/function_types.go` is deleted
- [ ] `spec/features/Feature` struct has `Params []types.ParamSpec` and `Synonyms []string` fields
- [ ] Every `CategoryFunction` Feature has non-empty `Params` (test enforced)
- [ ] `impl/interpreter/FunctionDef` has only `Name` and `Eval` fields
- [ ] `impl/interpreter/registry.go` builds `FunctionInfo` by joining with features registry
- [ ] LSP completion, hover, signature help import only `spec/features` (+ `spec/types` for type constants)
- [ ] `lsp/completion.go` no longer imports `impl/interpreter`
- [ ] `lsp/hover.go` no longer imports `impl/interpreter`
- [ ] `lsp/signature.go` no longer imports `impl/interpreter`
- [ ] TUI autocomplete builds suggestions from `spec/features` only
- [ ] `cm help` works correctly with consolidated data

### Quality Gates

- [ ] `task test` passes (all 30 packages)
- [ ] `task quality` passes (lint + modernize + staticcheck)
- [ ] No `impl/interpreter` imports from `lsp/` package
- [ ] `TestEveryBuiltinFunctionHasFeature` passes
- [ ] `TestEveryFunctionFeatureHasBuiltinFunction` passes
- [ ] `TestEveryFunctionHasParams` passes
- [ ] Existing golden test files unchanged (no behavior change)

## Dependencies & Prerequisites

None — this is a pure refactoring. No new dependencies, no new packages.

## Risk Analysis & Mitigation

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Synonym dispatch breaks | Medium | High | `TestEveryBuiltinFunctionHasFeature` catches missing entries. Run `task test` after each phase. |
| TUI autocomplete behavior changes | Low | Medium | Golden catwalk tests catch regressions. |
| LSP completion content changes | Low | Low | Nobody is using the extensions yet. |
| Help output changes | Low | Medium | `TestHelpOutput` golden test catches regressions. |

## Sources & References

### Origin

- **Brainstorm document:** [docs/brainstorms/2026-03-17-unified-feature-registry-brainstorm.md](docs/brainstorms/2026-03-17-unified-feature-registry-brainstorm.md) — Key decisions: spec-first single Feature struct, absorb ParamSpec, synonyms merge with aliases, TUI SuggestionSource stays as adapter, ParamSpec/ArgType go to spec/types

### Internal References (Learnings Applied)

- 8-layer checklist: `docs/solutions/logic-errors/adding-new-type-fraction-cross-layer-checklist.md` — registry is layer 9, never optional
- NL/functional parity: `docs/solutions/integration-issues/nl-functional-syntax-parity-and-doc-staleness.md` — parallel code paths drift, consolidation prevents it
- Diagnostic pipeline: `docs/solutions/code-organization/diagnostic-detailed-field-pipeline.md` — metadata flows one-way from source to consumers
- Module cohesion: `docs/solutions/code-organization/split-view-go-into-cohesive-modules.md` — group by semantic purpose

### Internal References (Code)

- Feature struct: `spec/features/registry.go:39`
- FunctionDef struct: `impl/interpreter/functions.go:15`
- FunctionSpec: `spec/semantic/function_types.go:41`
- Consistency tests: `impl/interpreter/function_consistency_test.go`
- Semantic checker hardcoding: `spec/semantic/checker.go:337-389`
- LSP completion (3 imports): `lsp/completion.go:8-12`
- TUI autocomplete (2 imports): `cmd/calcmark/tui/editor/autocomplete.go`

### Related Work

- Issue: #74 (consolidate function metadata)
