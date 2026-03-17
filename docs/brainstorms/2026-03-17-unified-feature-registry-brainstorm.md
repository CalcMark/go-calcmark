---
title: "refactor: unified feature registry — single source of truth for all function/unit metadata"
type: refactor
status: brainstorm
date: 2026-03-17
origin: github.com/CalcMark/go-calcmark/issues/74
---

# Unified Feature Registry

## What We're Building

Consolidate three overlapping metadata registries into one canonical `Feature` struct in `spec/features/`. Every consumer — TUI autosuggest, LSP completions/hover/signature help, `cm help`, documentation generation, and the semantic checker — imports a single package for all function, unit, and keyword metadata.

## Why This Approach

Today, building a single LSP completion item for `accumulate` requires importing three packages:

| Package | What it provides | Unique data |
|---------|-----------------|-------------|
| `impl/interpreter.BuiltinFunctions` | Name, signature, description, synonyms | `Eval` func pointer |
| `spec/features.Registry` | Aliases, NL examples, category | Parseable flag, search aliases |
| `spec/semantic.FunctionSpecs` | Param types, examples, optional/variadic | `ParamSpec` with `ArgType` |

These overlap on name, description, and examples. Adding a new function means updating all three. They **will** drift.

The `Eval` function pointer is the only thing that can't move to `spec/` (it's implementation). Everything else is metadata about the language, which belongs in the spec layer.

## Key Decisions

1. **`spec/features/Feature` becomes the single enriched struct.** It absorbs:
   - `ParamSpec` / `FunctionSpec` fields (from `spec/semantic/function_types.go`)
   - Signature, synonyms (from `impl/interpreter.BuiltinFunctions`)
   - Everything it already has (aliases, NL examples, category, syntax)

2. **`impl/interpreter.BuiltinFunctions` becomes minimal.** Each entry is:
   ```go
   type FunctionDef struct {
       Name string           // Matches Feature.Name — the join key
       Eval EvalFunc         // The implementation
   }
   ```
   Synonyms, description, signature, category — all come from `spec/features`.

3. **`spec/semantic/function_types.go` is deleted.** `ParamSpec`, `ArgType`, `FunctionSpec` move into `spec/features/`. The semantic checker imports `spec/features` instead.

4. **`spec/units/canonical.go` stays as-is.** It's already the single source of truth for units. `spec/features/registry.go` already reads from it to populate unit features. No change needed.

5. **`impl/interpreter/registry.go` becomes a thin adapter.** `GetAllFunctions()` and `GetFunctionByName()` join `BuiltinFunctions` with `features.Registry` at runtime.

## Migration Path

### Phase 1: Enrich Feature struct
- Add `Params []ParamSpec` to `Feature`
- Move `ParamSpec`, `ArgType` types to `spec/features/`
- Populate params in `getFunctions()` from the current `FunctionSpecs` data
- Delete `spec/semantic/function_types.go`
- Update `spec/semantic/checker.go` to import `spec/features`

### Phase 2: Slim down BuiltinFunctions
- Remove `Description`, `Signature`, `Synonyms`, `Category` from `FunctionDef`
- Add `features.Registry.GetByName(fn.Name)` lookups where needed
- Update `impl/interpreter/registry.go` to join with features

### Phase 3: Update all consumers
- `lsp/completion.go` → single `features.Registry` import
- `lsp/hover.go` → single import
- `lsp/signature.go` → single import
- `cmd/calcmark/cmd/help.go` → single import
- `cmd/calcmark/tui/editor/autocomplete.go` → single import
- TUI cursor context, error display → single import

### Phase 4: Add drift guard test
- `TestEveryBuiltinFunctionHasFeature` — every name in `BuiltinFunctions` must have a matching `Feature` with non-empty `Params`
- This is the permanent safety net even after consolidation

## Dependency Direction

```
spec/units/canonical.go          (foundation — unchanged)
    ↓
spec/features/registry.go       (THE source of truth for all metadata)
    ↓                    ↓
spec/semantic/checker.go    lsp/*         cmd/*/help    TUI autosuggest
    ↓
impl/interpreter/functions.go   (name + Eval only, joins with features)
```

## What Changes for Each Consumer

| Consumer | Before | After |
|----------|--------|-------|
| LSP completion | 3 imports | 1 import (`spec/features`) |
| LSP hover | 2 imports | 1 import |
| LSP signature | 1 import (`spec/semantic`) | 1 import (`spec/features`) |
| Semantic checker | own `FunctionSpecs` map | `features.Registry.GetByName()` |
| TUI autosuggest | 2 imports | 1 import |
| `cm help` | 2 imports | 1 import |
| Interpreter | `FunctionDef` has everything | `FunctionDef` has name + Eval; joins for metadata |

## Resolved Questions

1. **`Feature.Params` reuses the exact `ParamSpec` type** — `ArgType`, `Optional`, `Variadic`, `Examples` all move as-is. Both semantic validation and LSP presentation need every field.

2. **Interpreter synonyms merge into `Feature.Aliases`** — a synonym is a parseable alias. The interpreter looks up `Aliases` where `Parseable=true` for dispatch. One concept, one field. `FunctionDef.Synonyms` is deleted.

3. **TUI `SuggestionSource` stays as adapter** — lowest risk. It reads from `Feature` internally but the TUI rendering code doesn't change. The TUI works fine; only the data source changes.

4. **`ParamSpec`/`ArgType` go to `spec/types`, not `spec/features`** — avoids the semantic checker depending on a "features" package. Both `spec/features` and `spec/semantic` import `spec/types` for the shared param types. Clean dependency direction.

## Not Changing

- `spec/units/canonical.go` — already the single source of truth
- `spec/identifiers/` — canonical identifier lists (NetworkScopes, StorageTypes, etc.)
- The `Eval` function pointer stays in `impl/interpreter`
