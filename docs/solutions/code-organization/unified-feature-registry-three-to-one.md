---
title: "Unified feature registry: consolidate three metadata sources into one"
category: code-organization
date: 2026-03-17
severity: P1
module: spec/features, spec/types, impl/interpreter, lsp
tags: [registry, drift, single-source-of-truth, refactoring, lsp]
---

## Problem

Building a single LSP completion item for a function like `accumulate` required importing three separate packages, each with overlapping but non-identical metadata:

- `impl/interpreter.BuiltinFunctions` — name, signature, description, synonyms, Eval func
- `spec/features.Registry` — aliases, NL examples, category
- `spec/semantic.FunctionSpecs` — parameter types, valid values, optional/variadic

Adding a new function meant updating all three. Seven consistency tests existed solely to catch drift between them. The semantic checker also hardcoded function names in `checkFunctionCall()` — a fourth drift point.

## Root Cause

The three registries grew independently as new consumers (TUI help, autocomplete, LSP) needed different views of the same function metadata. No one consolidated because each package had a legitimate reason to exist. The drift risk was managed with tests rather than eliminated by design.

## Solution

1. **Move `ParamSpec`/`ArgType`/`FunctionSpec` to `spec/types/param_types.go`** — neutral shared package that both `spec/features` and `spec/semantic` can import without circular dependencies.

2. **Enrich `Feature` struct** with `Params []types.ParamSpec`, `Synonyms []string`, and `Subcategory string`. Params are populated from `types.FunctionSpecs` in `getFunctions()` — not duplicated.

3. **Slim `FunctionDef` to `Name + Eval`** — all metadata (description, signature, synonyms, category) removed. The interpreter's `registry.go` joins `BuiltinFunctions` with `features.Registry` at runtime.

4. **Update all consumers** — LSP, TUI autocomplete, `cm help`, docgen all read from `spec/features.Feature` only. The LSP no longer imports `impl/interpreter`.

5. **Delete 5 of 7 consistency tests** — drift is impossible by construction. Only `TestEveryBuiltinFunctionHasEval` and `TestEveryBuiltinFunctionHasFeature` remain.

Net result: -200 lines, 3 imports → 1 for metadata consumers.

## Prevention

- **Golden test regression**: When refactoring data sources, never regenerate golden files to match new output. The golden files define expected behavior — fix the code to match them. In this case, `Feature.Syntax` had different param names than the old `FunctionDef.Signature`, causing TUI autocomplete visual regressions.

- **Single source of truth by construction**: Instead of testing that N registries stay in sync, make it structurally impossible for them to diverge. One struct, one definition site, consumers join at runtime.

- **Subcategory for help grouping**: The old `FunctionDef.Category` ("Math", "Network") was a help-display concern, not a type concern. Adding `Feature.Subcategory` preserved TUI behavior without conflating it with `Feature.Category` (which is "function", "unit", "keyword").
