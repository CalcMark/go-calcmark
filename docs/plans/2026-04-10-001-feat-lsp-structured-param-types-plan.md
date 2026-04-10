---
title: "feat: expose structured parameter types in LSP completion, signatureHelp, and hover"
type: feat
status: active
date: 2026-04-10
origin: https://github.com/CalcMark/go-calcmark/issues/131
---

# feat: expose structured parameter types in LSP completion, signatureHelp, and hover

## Overview

Teach the LSP to ship the language's parameter type information to clients as structured data, so `calcmark-web` (and future editors) can render contextual placeholder suggestions, enum dropdowns, variables-in-scope by type, and rich hover without duplicating calcmark knowledge on the client.

Four surfaces change:

1. `textDocument/completion` is made **argument-context-aware** — when the cursor is inside a function call argument, completions are filtered to what is valid at that position (enum values inside a string-typed arg; type-compatible variables inside a typed arg; example values when nothing better applies).
2. Every function completion item carries `data.functionName` and `data.params` so clients never reverse-engineer the label.
3. `textDocument/signatureHelp` attaches structured `data.type` / `data.examples` on each `ParameterInformation`.
4. `textDocument/hover` returns rich markdown for functions, parameters, and variables — including inferred runtime type for variables.

No custom LSP methods. No protocol extensions beyond `CompletionItem.data` (already in the LSP spec) and a pragmatic `ParameterInformation.data` extension (non-standard but documented in the issue as intentional).

## Problem Frame

`calcmark-web` currently duplicates large chunks of calcmark language knowledge on the client — a hardcoded function registry (`function-registry.ts`), enum lists (`ENUM_VALUES` in `placeholder-suggestions.ts`), a type compatibility table (`TYPE_COMPATIBILITY`), a regex-based variable extractor (`extract-variables.ts`), and a label-parsing helper (`function-name-from-label.ts`). This:

- Drifts from `spec/types/param_types.go` silently
- Can't distinguish `percentage` from `number` via regex
- Can't reason about scope or evaluator-assigned types
- Has to guess the canonical function name for NL-example snippets like `grow 100 by 20 over 5 months`

The LSP already has the authoritative data — the `types.FunctionSpecs` map, `identifiers.NetworkScopes`/`NetworkTypes`/`StorageTypes`/`CompressionTypes`, the evaluator `Environment` returning `map[string]types.Type`. It just doesn't expose it through the wire. This plan closes that gap so calcmark-web can delete its shims in a follow-up cleanup PR.

## Requirements Trace

From the issue's "Acceptance criteria / definition of done":

- **R1.** Completion inside `throughput("|")` returns `gigabit`, `ten_gig`, `wifi`, … as completion items — not functions.
- **R2.** Completion inside `accumulate(|, 1 hour)` returns in-scope `rate`-typed variables (not `number` or `duration`).
- **R3.** Every completion item for `grow` — signature form *and* any NL example form — has `data.functionName == "grow"` and `data.params` populated.
- **R4.** SignatureHelp for `grow(100, |, 5)` returns `activeParameter: 1` and `parameters[1].data.type` set to the `grow` spec's rate-like arg type (the issue writes `number_or_quantity`; calcmark's spec uses `any` for `grow`'s `increment` — the plan ships whatever the spec says, see decisions).
- **R5.** Hover on `grow` in `goal = grow(100, 20, 5)` returns markdown with signature, description, and at least one example.
- **R6.** Hover on `price` in a doc with `price = 100` returns markdown containing `number` and `100`.

Derived from the final contract comment:

- **R7.** Completion items carry `data.kind` ∈ `{"function", "variable", "enum_value", "keyword"}`.
- **R8.** Enum values inside a string-typed arg are returned as regular `CompletionItem`s (no side-channel `data.enumValues`).
- **R9.** No new `calcmark/documentVariables` method; variables flow through `textDocument/completion`.
- **R10.** No `TYPE_COMPATIBILITY` table shipped to the client; server filters.
- **R11.** Union types are not discriminated in the wire format; the plan ships the existing `ArgType` strings verbatim.

## Scope Boundaries

**In scope:**

- LSP completion, signatureHelp, hover for the four surfaces above
- Server-side argument-context detection (which arg am I in; what type does it expect; am I inside a string literal)
- Server-side filtering of variables by type
- Enum value completions derived from `spec/identifiers`
- Structured `data` on completion items and parameter information
- Rich hover with runtime type inference for variables
- Full LSP-layer test coverage for acceptance criteria R1–R6

**Out of scope:**

- Calcmark-web client changes (handled in a follow-up cleanup PR)
- New `calcmark/documentVariables` or other custom methods
- Union-type discriminated wire format
- Shipping `TYPE_COMPATIBILITY` to the client
- Changes to the evaluator's variable storage shape beyond what's already exposed
- Incremental parsing or snapshot caching work
- TUI editor changes (calcmark CLI TUI); this is LSP-only
- Changes to `spec/features/registry.go`'s `Feature` shape beyond optionally mirroring new fields

## Context & Research

### Relevant Code and Patterns

**Primary touch points:**

- `lsp/completion.go` — current completion handler; `functionCompletionItems` around line 68; `variableCompletionItems` at 346; `classifyCompletionContext` at 121 which today only distinguishes `General` / `AfterUnitKeyword` / `Markdown`
- `lsp/signature.go` — `extractFunctionContext` at 111 walks backward counting commas to find `(funcName, argIndex)`; `signatureHelpForFunction` at 37 builds the current response
- `lsp/hover.go` — `textDocumentHover` at 15 supports variables (name + value only), functions, and units; no parameter or runtime-type information
- `lsp/server.go` — `DocumentSnapshot` holds `Evaluator`; read at 213
- `spec/types/param_types.go` — `ArgType` enum (lines 14–26), `ParamSpec` (29–35), `FunctionSpec` (38–41), `FunctionSpecs` map (56–177), `GetFunctionSpec`/`GetParamAtIndex` (181–202)
- `spec/identifiers/identifiers.go` — `NetworkScopes`, `NetworkTypes`, `StorageTypes`, `StorageAliases`, `CompressionTypes`
- `spec/features/registry.go` — `Feature` struct with `Params`, `Syntax`, `Description`, `NLExample`, `Synonyms`; `getFunctions()` mirrors `types.FunctionSpecs.Params` into features (lines 354–357)
- `impl/interpreter/environment.go` — `Environment.GetAllVariables() map[string]types.Type` at line 79; the evaluator already exposes typed values
- `spec/types/` — concrete types (`Number`, `Quantity`, `Rate`, `Duration`, `Percentage`, `Currency`, `Date`, `Fraction`, `Boolean`, …) implementing a shared `Type` interface in `spec/types/types.go`
- `lsp/server_test.go` — table-driven tests for helpers (`TestExtractFunctionContext`, `TestSignatureHelpForFunction`, `TestCompletionItems_FunctionsIncluded`) — established pattern for new tests

**Protocol types used:**

- `github.com/tliron/glsp/protocol_3_16.CompletionItem` — already has `Data any` ✓
- `github.com/tliron/glsp/protocol_3_16.ParameterInformation` — **does NOT have a `Data` field**. The LSP 3.16 spec doesn't define one. See decision below.

### Institutional Learnings

- `docs/solutions/best-practices/unified-feature-registry-three-to-one.md` — `ParamSpec`/`FunctionSpec` consolidation in `spec/types` is the single source of truth; new fields should land there and mirror into `features.Feature` via the existing population loop, not be added in a parallel registry.
- `docs/solutions/bugs/lsp-debounce-staleness-read-requests.md` — completion/signature/hover handlers must read source text from the immediate `ds.getSource()` but semantic state (evaluator, snapshot) from the debounced `ds.getSnapshot()`. This plan inherits that discipline unchanged.
- `docs/solutions/best-practices/directive-as-value-cross-layer-learnings.md` — LSP prefix/context logic is duplicated in TUI autocomplete. Argument-context-awareness added here should stay inside `lsp/` and not spread to the TUI unless explicitly scoped; the TUI has its own completion path and is out of scope for this plan.
- `docs/solutions/bugs/nl-function-missing-ast-range.md` — NL function call AST nodes now carry `Range`, meaning hover/go-to-definition work uniformly across signature and NL forms. Hover enrichment in Unit 7 can lean on this and does not need a parallel fallback.

### External References

Not needed. The LSP spec definition of `CompletionItem.data` is well known; the decision to extend `ParameterInformation` with a non-standard `data` is documented in the issue body itself as an intentional, pragmatic choice.

## Key Technical Decisions

**D1. Ship exactly the `ArgType` strings the spec already uses.** No union-type discrimination, no string rewriting. `grow`'s `increment` param is `ArgTypeAny` in the current spec — the plan ships `"any"`, not `"number_or_quantity"`. Rationale: the issue's final contract explicitly retracts union-type discrimination ("Stringly-typed … is an internal calcmark concern — ship whatever the spec package already uses"). If the spec later introduces unions, that's a separate change.

**D2. Add `EnumValues []string` to `ParamSpec` in `spec/types/param_types.go`.** Populate it for every string-typed param that has a finite identifier list (`rtt.scope`, `throughput.network_type`, `transfer_time.scope` / `network_type`, `read.storage_type`, `seek.storage_type`, `compress.compression_type`). The `Examples` field keeps its quoted-form values for display continuity. `EnumValues` holds unquoted canonical names. Rationale: the alternatives — client-side introspection of `Examples` (brittle), a parallel map in `lsp/` (drift), or stashing the list in the registry (wrong layer) — all duplicate the knowledge that already lives in `spec/identifiers`. A new field on `ParamSpec` is one line plus populated literals and keeps the single source of truth intact.

**D3. Custom `lspParameterInformation` wrapper struct for signatureHelp.** Since `protocol.ParameterInformation` has no `Data` field, define a local struct in `lsp/signature.go` that mirrors its JSON shape plus `Data any \`json:"data,omitempty"\``, and change the signatureHelp handler's return to build a `SignatureHelp`-shaped local struct instead of `protocol.SignatureHelp`. The tliron/glsp router accepts `any` return values and JSON-marshals them. Rationale: forking `tliron/glsp` is excessive; embedding the glsp struct doesn't add the field to the wire because glsp's `MarshalJSON` controls the shape. A thin local wrapper is the smallest viable change and keeps everything else unchanged.

**D4. Server-side variable type filtering uses the evaluator's live `types.Type` map.** `Environment.GetAllVariables()` already returns `map[string]types.Type`. A new `lsp/typemap.go` helper maps each concrete runtime type to its `ArgType` string (`*types.Number` → `"number"`, `*types.Percentage` → `"percentage"`, `*types.Rate` → `"rate"`, etc.). Variables whose runtime type matches the active parameter's `ArgType` are preferred; exact-match candidates are returned first, `ArgTypeAny` matches everything. Rationale: the client cannot see runtime types and shouldn't. The evaluator already has them. Mapping lives in `lsp/` because it's LSP-wire concerns, not spec concerns.

**D5. Argument context detection extends `extractFunctionContext` rather than replacing `classifyCompletionContext`.** A new helper `extractArgumentContext` in `lsp/signature.go` (or a new `lsp/argctx.go`) returns `(funcName, paramIdx, insideStringLiteral, stringQuoteChar)` from backward-walking the rune slice. `classifyCompletionContext` stays for its general/unit/markdown role and the new detection is layered on top inside `textDocumentCompletion`. Rationale: keeps the context classifier simple and single-purpose; reuses the existing battle-tested paren/comma walker; lets signatureHelp and completion share the same primitive.

**D6. Both signature-form and NL-example completion items carry the same `functionName` in `data`.** The `features.FunctionSuggestions` path returns two item kinds — `Category == "example"` and the regular function row. Both paths attach `data.functionName = <canonical name>`. For NL rows, the canonical name comes from the feature's `Name` field (already the canonical), not parsed from the label. Rationale: explicit acceptance criterion R3.

**D7. Structured `data` shape is additive and JSON-marshaled as-is.** Use a plain Go struct `completionItemData` with `Kind`, `FunctionName`, `Params` fields (omitempty where appropriate). Clients that don't know about the new field ignore it; glsp marshals any Go value assigned to `CompletionItem.Data any`. Rationale: no special serializer work; round-trip through the glsp library is already proven for `CompletionItem.Data`.

**D8. Pipeline posture is test-first for LSP handlers.** New tests go into `lsp/server_test.go` (or new files `lsp/completion_test.go`, `lsp/signature_test.go`, `lsp/hover_test.go`) following the established table-driven pattern. Each acceptance criterion R1–R6 has a dedicated test. Rationale: the acceptance criteria are specific enough to convert directly into assertions; test-first keeps the wire shape honest while refactoring.

## Open Questions

### Resolved During Planning

- **Q.** Where does `EnumValues` live — `ParamSpec`, the Feature, or a side-table in `lsp/`? → **`ParamSpec` in `spec/types/param_types.go`** (D2).
- **Q.** How do we attach a `Data` field to `ParameterInformation` when the glsp type doesn't have one? → **Local wrapper struct** (D3).
- **Q.** Does the evaluator expose variable types for filtering? → **Yes**, `Environment.GetAllVariables() map[string]types.Type` at `impl/interpreter/environment.go:79`. No evaluator changes needed.
- **Q.** Do we need to track "inside string literal" to trigger enum completions? → **Yes**, extend the backward walker to recognize `"` and (optionally) `'` quoting while scanning for the enclosing `(`.
- **Q.** Does `grow`'s `increment` param ship as `"any"` or `"number_or_quantity"`? → **`"any"`**, matching `spec/types/param_types.go:155`. The issue's JSON examples used `number_or_quantity` illustratively; R4's test asserts whatever the spec says, not the literal string in the issue body. Cross-checked against D1.

### Deferred to Implementation

- Exact method signatures for new helpers (`extractArgumentContext`, type mapper) — let TDD shape them.
- Whether to split `lsp/completion.go` into smaller files — defer unless it grows past ~600 lines during implementation.
- Whether `ArgTypeAny` variables should be surface-ordered before or after type-specific matches when both apply — iterate on this against the tests once they exist; default is "exact-type matches first, then any".
- Precise markdown layout for hover on functions vs. variables — iterate to match the examples in acceptance criteria, no pre-specification needed.
- Whether to mirror the new `EnumValues` field into `features.Feature.Params` via the existing population loop — cheap to do; decide during Unit 2 based on whether any code reads `Feature.Params[i].EnumValues`.

## High-Level Technical Design

> *This illustrates the intended approach and is directional guidance for review, not implementation specification. The implementing agent should treat it as context, not code to reproduce.*

**Data flow for a single `textDocument/completion` request inside `throughput("g|")`:**

```
Client cursor at `throughput("g|")`
        │
        ▼
┌────────────────────────────────┐
│ textDocumentCompletion         │
│  - read source (immediate)     │
│  - read snapshot (debounced)   │
└──────────────┬─────────────────┘
               │
               ▼
┌────────────────────────────────┐
│ extractArgumentContext(line,   │
│   col) → {                     │
│     funcName: "throughput",    │
│     paramIdx: 0,               │
│     insideString: true,        │
│     stringPrefix: "g"          │
│   }                            │
└──────────────┬─────────────────┘
               │
               ▼
┌────────────────────────────────┐
│ Look up FunctionSpec and       │
│ active ParamSpec:              │
│   { Name: "network_type",      │
│     Type: ArgTypeString,       │
│     EnumValues: [...]          │
│   }                            │
└──────────────┬─────────────────┘
               │
               ▼
┌────────────────────────────────┐
│ Branch: insideString &&        │
│   EnumValues != nil            │
│                                │
│ → enumValueCompletionItems(    │
│     EnumValues, prefix="g")    │
│   filtered to {"gigabit"}      │
└──────────────┬─────────────────┘
               │
               ▼
   []CompletionItem with
   data.kind == "enum_value"
```

**Data flow for `accumulate(|, 1 hour)`:**

```
extractArgumentContext →
  { funcName: "accumulate",
    paramIdx: 0,
    insideString: false }

activeParam = accumulate.Params[0] → ArgTypeRate

variableCompletionItems(snap, prefix="", line,
                        requiredType=ArgTypeRate)
  ↓
  env.GetAllVariables() → map[string]types.Type
  for each (name, val):
    argType = runtimeTypeToArgType(val)
    if argType == ArgTypeRate OR requiredType == ArgTypeAny:
      include with data.kind = "variable", data.type = argType

Also include:
  exampleValueCompletionItems(spec.Params[0].Examples)
    → "10 MB/s", "100 requests/second", "5 GB/day"
       as data.kind = "example_value"
```

**SignatureHelp wire shape change (conceptually):**

```
// Before (glsp ParameterInformation, no data):
{ "label": "rate", "documentation": { ... } }

// After (lspParameterInformation, local wrapper):
{ "label": "rate",
  "documentation": { ... },
  "data": {
    "type": "rate",
    "examples": ["10 MB/s", "100 requests/second", "5 GB/day"]
  } }
```

## Implementation Units

- [ ] **Unit 1: Add `EnumValues` to `ParamSpec` and populate enum-typed params**

  **Goal:** Give every string-typed parameter with a finite identifier list a structured `EnumValues` field so downstream code (completion + future consumers) has one canonical place to read the valid values.

  **Requirements:** R1, R8, R11

  **Dependencies:** None.

  **Files:**
  - Modify: `spec/types/param_types.go` (add field; populate `rtt`, `throughput`, `transfer_time`, `read`, `seek`, `compress` entries)
  - Test: `spec/types/param_types_test.go` (new table-driven test)

  **Approach:**
  - Add `EnumValues []string` to the `ParamSpec` struct next to `Examples`.
  - Populate each string-typed param that has a finite identifier list by reading directly from `identifiers.NetworkScopes` / `identifiers.NetworkTypes` / `identifiers.StorageTypes` (include aliases via `identifiers.AllStorageNames()` if that helper already covers the intent; otherwise use the primary slice) / `identifiers.CompressionTypes`.
  - `Examples` stays as the quoted display form — do not replace or re-derive it.
  - Non-enum string params (e.g., `convert_rate.time_unit`, `capacity.unit`, `compound.period`) leave `EnumValues` nil.

  **Execution note:** Test-first — write the enum-values table test first so that "every identifier-backed string param has the full enum list" is a durable assertion.

  **Patterns to follow:**
  - Identifier references via `identifiers.NetworkScopes` etc. exactly like `Examples: quotedNames(identifiers.NetworkScopes)` at `spec/types/param_types.go:105,111,118,119,126,132,139`.

  **Test scenarios:**
  - Happy path: `GetFunctionSpec("throughput").Params[0].EnumValues` equals `identifiers.NetworkTypes` in order.
  - Happy path: `GetFunctionSpec("rtt").Params[0].EnumValues` equals `identifiers.NetworkScopes`.
  - Happy path: `GetFunctionSpec("transfer_time").Params[1].EnumValues` equals `identifiers.NetworkScopes` and `Params[2].EnumValues` equals `identifiers.NetworkTypes`.
  - Happy path: `GetFunctionSpec("read").Params[1].EnumValues` covers every entry in `identifiers.StorageTypes`.
  - Happy path: `GetFunctionSpec("compress").Params[1].EnumValues` equals `identifiers.CompressionTypes`.
  - Edge case: non-enum string params (`convert_rate.time_unit`, `compound.period`, `capacity.unit`) have `EnumValues == nil`.
  - Edge case: non-string params (`accumulate.rate`, `grow.periods`) have `EnumValues == nil`.

  **Verification:**
  - `go test ./spec/types/...` passes with the new assertions.
  - Existing `TestCompoundFunctionSpec_HasPeriodParam` still passes.

- [ ] **Unit 2: Runtime-type to ArgType mapper**

  **Goal:** Produce a single helper that maps an evaluator `types.Type` value to its corresponding `ArgType` string so the LSP can filter variables by parameter type.

  **Requirements:** R2, R6

  **Dependencies:** None.

  **Files:**
  - Create: `lsp/typemap.go`
  - Test: `lsp/typemap_test.go`

  **Approach:**
  - Define `runtimeTypeToArgType(v types.Type) types.ArgType` using a type switch on concrete types (`*types.Number`, `*types.Quantity`, `*types.Rate`, `*types.Duration`, `*types.Percentage`, `*types.Currency`, `*types.String`, `*types.Boolean`, `*types.Date`, `*types.Fraction`, others).
  - Currency maps to `ArgTypeQuantity` (currency is a specialized quantity for filtering purposes) — document this in an inline comment; single-line.
  - Unknown types fall through to `ArgTypeAny`.
  - Also provide `argTypesCompatible(actual, required ArgType) bool` where `required == ArgTypeAny` is always true, otherwise exact match.

  **Execution note:** Test-first.

  **Patterns to follow:**
  - Type-switch pattern used in `impl/interpreter/*.go` when branching on concrete runtime types (the existing interpreter already does this extensively).

  **Test scenarios:**
  - Happy path: `*types.Number{}` → `ArgTypeNumber`.
  - Happy path: `*types.Percentage{}` → `ArgTypePercentage`.
  - Happy path: `*types.Quantity{}` → `ArgTypeQuantity`.
  - Happy path: `*types.Rate{}` → `ArgTypeRate`.
  - Happy path: `*types.Duration{}` → `ArgTypeDuration`.
  - Happy path: `*types.Currency{}` → `ArgTypeQuantity`.
  - Edge case: `nil` input → `ArgTypeAny`.
  - Edge case: an unrecognized type → `ArgTypeAny`.
  - Happy path: `argTypesCompatible(ArgTypeRate, ArgTypeRate)` true; `(ArgTypeNumber, ArgTypeRate)` false; `(ArgTypeNumber, ArgTypeAny)` true.

  **Verification:**
  - `go test ./lsp/` passes including the new typemap test file.

- [ ] **Unit 3: Argument context analyzer**

  **Goal:** Produce a helper that reports, for a given `(lineText, col)`, whether the cursor is inside a function call's Nth argument and whether that position is inside a string literal. Reuses the existing backward paren walker.

  **Requirements:** R1, R2, R3, R4

  **Dependencies:** None (can land alongside Unit 1; no dependency on EnumValues).

  **Files:**
  - Modify: `lsp/signature.go` (add `extractArgumentContext` next to `extractFunctionContext`, keeping the original for backward compat) OR create `lsp/argctx.go`
  - Test: `lsp/argctx_test.go` (or extend `lsp/server_test.go`'s `TestExtractFunctionContext`)

  **Approach:**
  - Extend the backward walker to also track when the scanner is inside a string literal (toggle on unescaped `"`). When the cursor position is inside an unterminated string literal that lives inside the active argument, report `insideString: true`.
  - Return struct: `argumentContext { funcName string; paramIdx int; insideString bool; stringPrefix string }` where `stringPrefix` is the characters already typed inside the string (for filtering enum value matches).
  - `extractFunctionContext` remains as a thin adapter that returns `(funcName, paramIdx)` by delegating to the new helper, so existing signatureHelp code doesn't need a parallel implementation.
  - Must handle nested parens (`accumulate(convert_rate(10 MB/s, |)`) via the existing depth counter.
  - Must handle escaped quotes inside strings (`"fo\"o"`).

  **Execution note:** Test-first — the backward walker is the kind of helper where a failing-first test drives the edge cases out.

  **Patterns to follow:**
  - `extractFunctionContext` at `lsp/signature.go:111` — rune-aware, backward, depth-counted.

  **Test scenarios:**
  - Happy path: `throughput("` with cursor at end → `{throughput, 0, insideString: true, stringPrefix: ""}`.
  - Happy path: `throughput("gig` with cursor at end → `{throughput, 0, insideString: true, stringPrefix: "gig"}`.
  - Happy path: `accumulate(` with cursor at end → `{accumulate, 0, insideString: false}`.
  - Happy path: `accumulate(10 MB/s, ` with cursor at end → `{accumulate, 1, insideString: false}`.
  - Happy path: `grow(100, ` → `{grow, 1, insideString: false}`.
  - Edge case: nested call `accumulate(convert_rate(10 MB/s, "` → `{convert_rate, 1, insideString: true}` (inner call wins).
  - Edge case: cursor outside any call → `{"", -1, false}`.
  - Edge case: unmatched close paren — cursor after `foo())` → `{"", -1, false}`.
  - Edge case: escaped quote inside string `throughput("a\"b` → treat as still inside the string, `stringPrefix == "a\"b"`.
  - Edge case: cursor on a non-function call like `x = 1 + ` → `{"", -1, false}`.

  **Verification:**
  - `go test ./lsp/` passes; existing `TestExtractFunctionContext` still passes (backward-compat adapter).

- [ ] **Unit 4: Enum-value completions inside string-typed args**

  **Goal:** When the cursor is inside a string-typed arg that has `EnumValues`, return those values as ordinary `CompletionItem`s filtered by the typed prefix.

  **Requirements:** R1, R7, R8

  **Dependencies:** Unit 1 (EnumValues), Unit 3 (argumentContext).

  **Files:**
  - Modify: `lsp/completion.go` (thread argument context through `textDocumentCompletion`; add `enumValueCompletionItems` helper)
  - Test: `lsp/completion_test.go` (new file) or extend `lsp/server_test.go`

  **Approach:**
  - In `textDocumentCompletion`, after computing `classifyCompletionContext`, call `extractArgumentContext` on the same line/col.
  - If `argCtx.funcName != "" && argCtx.insideString && activeParam.EnumValues != nil`, return **only** `enumValueCompletionItems(activeParam.EnumValues, argCtx.stringPrefix)`. Do not mix in functions/units/variables — the cursor is inside a string literal.
  - `enumValueCompletionItems` builds items with:
    - `Label` = the enum value (unquoted)
    - `Kind` = `CompletionItemKindEnumMember` (or `CompletionItemKindValue` if EnumMember is unavailable)
    - `InsertText` = the unquoted value (the cursor is already between the `""`)
    - `Data` = `completionItemData{Kind: "enum_value"}`
    - `Detail` = a short label like `"network_type"` derived from the parameter name (optional, nice-to-have)
  - Filter by `strings.HasPrefix(value, argCtx.stringPrefix)` (case-insensitive is a judgement call — default to case-sensitive to match calcmark's identifier casing).

  **Patterns to follow:**
  - `unitCompletionItems` at `lsp/completion.go:292` for the shape of a specialized item builder.

  **Test scenarios:**
  - Happy path (R1): cursor at `throughput("|")` returns items labeled `gigabit`, `ten_gig`, `hundred_gig`, `wifi`, `four_g`, `five_g` and no function items.
  - Happy path: cursor at `throughput("g|")` returns only items starting with `g` (`gigabit`).
  - Happy path: cursor at `rtt("re|")` returns `regional`.
  - Happy path: cursor at `read(1 GB, "ss|")` returns storage-type items starting with `ss`.
  - Happy path: cursor at `compress(1 GB, "gz|")` returns `gzip`.
  - Edge case: cursor at `convert_rate(10 MB/s, "` (string arg but no `EnumValues` — `time_unit` is free-form) → returns no enum items, falls through to general completions.
  - Edge case: cursor on an unknown function `frobnicate("` → falls through to general completions (no spec to drive filtering).
  - Integration: every returned enum item has `data.kind == "enum_value"`.

  **Verification:**
  - `go test ./lsp/` passes with R1 assertions.

- [ ] **Unit 5: Type-filtered variable completions inside typed args**

  **Goal:** When the cursor is inside a typed (non-string) function argument, prefer in-scope variables whose runtime type matches the active parameter's `ArgType`.

  **Requirements:** R2, R7

  **Dependencies:** Unit 2 (typemap), Unit 3 (argumentContext).

  **Files:**
  - Modify: `lsp/completion.go` (`variableCompletionItems` signature gains an optional `requiredType` parameter; calling site passes it when `argCtx` identifies an active param)
  - Test: extend `lsp/completion_test.go`

  **Approach:**
  - Extend `variableCompletionItems` to accept an optional `requiredType types.ArgType` (zero value = "no filter").
  - Inside the loop, call `runtimeTypeToArgType(val)` to get each variable's `ArgType`. If `requiredType != ""` and `!argTypesCompatible(varType, requiredType)`, skip it.
  - Attach `data.kind = "variable"` and `data.type = varType` to each returned item.
  - When inside an argument context, variable completions are mixed with example-value items (see Unit 6) — they're complementary, not exclusive.
  - Do not filter when `argCtx` is empty (bare expression context); preserve current behavior.
  - Preserve the existing limitation (no line-based scope filtering) — that's a separate bug.

  **Patterns to follow:**
  - Existing `variableCompletionItems` at `lsp/completion.go:346` — add filtering inside the loop.

  **Test scenarios:**
  - Happy path (R2): a document with `bandwidth = 10 MB/s` and `delay = 1 hour`, cursor at `accumulate(|, 1 hour)` → returns `bandwidth` and not `delay`.
  - Happy path: same doc, cursor at `accumulate(bandwidth, |)` → returns `delay` (duration-typed) and not `bandwidth` (rate-typed).
  - Happy path: a doc with `count = 5` (`number`) and `price = 100` (`number`), cursor at `sqrt(|)` → returns both.
  - Happy path: cursor at `sum(|)` (variadic `any`) → returns all variables regardless of type.
  - Edge case: no variables in scope → empty list, no error.
  - Edge case: cursor outside any function call → returns all variables (current behavior preserved).
  - Integration: every variable item has `data.kind == "variable"` and `data.type` set to the evaluator-assigned type string.

  **Verification:**
  - `go test ./lsp/` passes with R2 assertions.

- [ ] **Unit 6: Structured `data` on every function completion item**

  **Goal:** Attach `data.kind`, `data.functionName`, and `data.params` to every function completion item — signature-form AND NL-example form — so clients never parse labels.

  **Requirements:** R3, R7

  **Dependencies:** Unit 1 (for `EnumValues` to flow through to params when clients want them — optional here, but the struct should reserve space for it).

  **Files:**
  - Modify: `lsp/completion.go` (`functionCompletionItems`, `buildFunctionDoc` unchanged, new `buildCompletionItemData`)
  - Create inline or co-located: `completionItemData`, `completionItemParamData` Go structs
  - Test: extend `lsp/completion_test.go`

  **Approach:**
  - Define `completionItemData` struct:
    - `Kind string` — "function" | "variable" | "enum_value" | "keyword"
    - `FunctionName string` (omitempty)
    - `Params []completionItemParamData` (omitempty)
  - Define `completionItemParamData` struct:
    - `Name string`
    - `Type types.ArgType`
    - `Examples []string` (omitempty)
    - `Optional bool` (omitempty)
    - `Variadic bool` (omitempty)
    - `EnumValues []string` (omitempty) — carries the same info that enum-value completions use, useful for clients that want to render all valid values without a separate request
  - In `functionCompletionItems`, look up `types.GetFunctionSpec(s.Name)` (or the registry `Feature.Name`) and build a `completionItemData` for both the signature row and the NL-example row. For NL-example rows, use `suggestion.Name` (the canonical) as the `FunctionName`, not the label.
  - Assign the result to `CompletionItem.Data`.

  **Patterns to follow:**
  - `buildFunctionSnippet` at `lsp/completion.go:241` — spec lookup by name pattern.
  - Feature registry `features.FunctionSuggestions` already returns items with `Name` set to the canonical; no reverse lookup needed.

  **Test scenarios:**
  - Happy path (R3): completing `gro` returns items for `grow` — both the signature-form row and every NL-example row all have `data.functionName == "grow"` and `data.params[0].name == "amount"`.
  - Happy path: `data.params[i].type` mirrors `types.FunctionSpecs["grow"].Params[i].Type` verbatim.
  - Happy path: for `throughput`, `data.params[0].enumValues` is non-empty and matches `identifiers.NetworkTypes`.
  - Happy path: for `accumulate`, `data.params[0].enumValues` is empty/nil (no enum).
  - Edge case: completion on a prefix matching zero functions returns zero items, no panic, no nil-pointer.
  - Integration: every function item has `data.kind == "function"`.

  **Verification:**
  - `go test ./lsp/` passes; R3 assertion holds for both signature-form and NL-example rows.

- [ ] **Unit 7: Structured `data` on signatureHelp ParameterInformation**

  **Goal:** Return `data.type` and `data.examples` on each `ParameterInformation` in signatureHelp so the gutter signature panel can render typed hints without parsing the documentation markdown.

  **Requirements:** R4

  **Dependencies:** Unit 1 (param EnumValues not strictly required here, but the wrapper struct should include them so a future client can read them without a second round-trip).

  **Files:**
  - Modify: `lsp/signature.go` (introduce local wrapper types; change handler return to `any`)
  - Test: extend `lsp/server_test.go` `TestSignatureHelpForFunction` and add new cases

  **Approach:**
  - Define local structs at the bottom of `lsp/signature.go`:
    - `lspParameterInformation { Label any; Documentation any; Data any }` with JSON tags matching `protocol.ParameterInformation` field names plus `data`
    - `lspSignatureInformation { Label, Documentation, Parameters, ActiveParameter }` matching protocol names
    - `lspSignatureHelp { Signatures []lspSignatureInformation; ActiveSignature, ActiveParameter }`
  - Change `signatureHelpForFunction` return type to `*lspSignatureHelp` and update `textDocumentSignatureHelp` to return `any` (the glsp router accepts this).
  - Populate each `lspParameterInformation.Data` with a `signatureParamData{Type types.ArgType; Examples []string; EnumValues []string \`json:"enumValues,omitempty"\`}` struct.
  - Keep the existing markdown `Documentation` intact for backward compat with clients that don't know about `data`.

  **Execution note:** Carefully verify the JSON shape round-trips through the glsp router (no wrapping; no field renaming). Add one end-to-end JSON-marshal assertion in the test.

  **Patterns to follow:**
  - `signatureHelpForFunction` at `lsp/signature.go:37` — mirror its structure.
  - `protocol.ParameterInformation` field names and JSON tags must match exactly (`label`, `documentation`) so clients that read the standard shape still work.

  **Test scenarios:**
  - Happy path (R4): signatureHelp for `grow(100, |, 5)` returns `activeParameter == 1`. `signatures[0].parameters[1].data.type` matches `types.FunctionSpecs["grow"].Params[1].Type` (i.e. `"any"`).
  - Happy path: signatureHelp for `accumulate(|, 1 hour)` returns `parameters[0].data.type == "rate"` and `parameters[0].data.examples` non-empty.
  - Happy path: signatureHelp for `throughput("|")` returns `parameters[0].data.type == "string"` and `parameters[0].data.enumValues == identifiers.NetworkTypes`.
  - Happy path: signatureHelp for an unknown function returns `nil`.
  - Integration: marshal the response via `encoding/json` and assert the resulting shape matches the contract in the issue (top-level keys `signatures`, `activeSignature`, `activeParameter`; `parameters[*]` has `label`, `documentation`, `data`).
  - Edge case: legacy clients reading `documentation` still see the existing markdown.

  **Verification:**
  - `go test ./lsp/` passes; the marshaled JSON contains the new `data` field and does not break existing fields.

- [ ] **Unit 8: Rich hover with parameter types and runtime variable types**

  **Goal:** Return markdown hover content that includes the signature + description + an example for functions, and the inferred runtime type + value for variables.

  **Requirements:** R5, R6

  **Dependencies:** Unit 2 (typemap) for variable type inference.

  **Files:**
  - Modify: `lsp/hover.go` (`textDocumentHover` — extend variable branch with type; extend function branch with parameter table and example)
  - Test: extend `lsp/server_test.go` or add `lsp/hover_test.go`

  **Approach:**
  - **Variable branch:** when a variable `word` matches, call `runtimeTypeToArgType(val)` to get the type string. Format as `**name**: `number` = `100`` (or similar). Keep backward-compat content available.
  - **Function branch:** after the existing signature + description, append a parameter list drawn from `types.FunctionSpecs[fn.Name].Params` — one bullet per param with name, type, and the first example. Append a fenced "Example" block using `fn.Example` if non-empty, otherwise synthesize one from param examples.
  - Unknown words: unchanged fallthrough behavior.

  **Patterns to follow:**
  - `lsp/hover.go:38` (variable branch) and `:55` (function branch).

  **Test scenarios:**
  - Happy path (R5): hover on `grow` in `goal = grow(100, 20, 5)` returns markdown containing the string `grow(`, a description sentence, and at least one example.
  - Happy path (R6): hover on `price` in a doc with `price = 100` returns markdown containing `number` and `100`.
  - Happy path: hover on `tax_rate` in a doc with `tax_rate = 8%` returns markdown containing `percentage` and `8%`.
  - Happy path: hover on `bandwidth` in a doc with `bandwidth = 10 MB/s` returns markdown containing `rate` and `10 MB/s`.
  - Happy path: hover on a function synonym still returns the canonical function's hover content.
  - Edge case: hover on an unknown word returns `nil` — not an error.
  - Edge case: hover on a variable whose evaluator assignment failed (not in `GetAllVariables`) falls through to function/unit matching.

  **Verification:**
  - `go test ./lsp/` passes with R5 and R6 assertions.

- [ ] **Unit 9: Full-suite regression pass and acceptance-criteria JSON fixture**

  **Goal:** Prove the whole contract holds by running the full test suite and landing one end-to-end test that constructs a small document, drives the LSP handlers as a client would, and asserts on the marshaled JSON for each of R1–R6.

  **Requirements:** R1–R6 (integration)

  **Dependencies:** Units 1–8.

  **Files:**
  - Create: `lsp/acceptance_test.go`

  **Approach:**
  - One test file with a table of cases where each row is an acceptance criterion: `{doc string, position (line,col), handler "completion|signatureHelp|hover", expected contains []string, expected absent []string}`.
  - Each case constructs a minimal calcmark document, spins up a `Server`, calls `textDocumentDidOpen` to prime the snapshot, waits for (or synchronously triggers) evaluation, then calls the handler directly.
  - Assert on the marshaled JSON response using a contains/absent check rather than strict equality (wire shape stability across cosmetic changes).
  - No shell commands; plain Go unit tests.

  **Execution note:** Test-last (regression). Every acceptance criterion must have a matching row in this table.

  **Patterns to follow:**
  - `lsp/server_test.go` — table-driven with direct handler invocation; no JSON-RPC framing needed.
  - Use `encoding/json.Marshal` to produce the wire bytes and assert substrings.

  **Test scenarios:**
  - R1: doc `x = throughput("")` with cursor inside the `""` → completion contains `gigabit`, absent `accumulate`.
  - R2: doc `bandwidth = 10 MB/s\nresult = accumulate(, 1 hour)` with cursor at the first arg → completion contains `bandwidth`, absent `accumulate`.
  - R3: doc `x = gro` → completion contains `"functionName":"grow"`, and for the NL-example form `"functionName":"grow"` also present.
  - R4: doc `goal = grow(100, , 5)` with cursor at the second arg → signatureHelp marshaled JSON contains `"activeParameter":1` and `"type":"any"` on parameter index 1.
  - R5: doc `goal = grow(100, 20, 5)` → hover on `grow` contains `grow(`, a description, and at least one example.
  - R6: doc `price = 100` → hover on `price` contains `number` and `100`.
  - Regression: running `task test` passes (no unrelated breakage).

  **Verification:**
  - `task test` passes.
  - `task quality` passes.
  - Every R1–R6 row in the acceptance table passes.

## System-Wide Impact

- **Interaction graph:**
  - `spec/types/param_types.go` is read by `lsp/completion.go`, `lsp/signature.go`, `lsp/hover.go`, `spec/semantic/*.go`, `spec/features/registry.go`, and `impl/interpreter/*` (via shared imports). Adding a new field on `ParamSpec` is additive; existing consumers ignore it automatically.
  - `features.Feature.Params` mirrors `types.ParamSpec` via `spec/features/registry.go:354-357`. Go's value-copy means the new `EnumValues` field will flow through automatically — no changes needed there, but the copy does need verification. Add one regression test that asserts `features.DefaultRegistry().ByCategory(CategoryFunction)` reads the new field for at least one function.
- **Error propagation:**
  - All new LSP branches return `nil, nil` on failure (consistent with current handlers) rather than protocol errors, keeping the client-facing contract unchanged.
  - The argument-context analyzer must never panic on malformed input — if the walker confuses itself, fall back to general completions. A fuzz test on `extractArgumentContext` in Unit 3 covers this.
- **State lifecycle risks:**
  - The LSP reads source text from `ds.getSource()` (immediate) and semantic state from `ds.getSnapshot()` (debounced). New code must honor this split — variable-in-scope reads go through the snapshot's evaluator; position-based reads go through the fresh source. Flagged explicitly because R2's variable-filtering behavior depends on the snapshot being fresh enough. This is the same staleness concern as `lsp-debounce-staleness-read-requests.md`; the debounce window can mean a just-typed variable isn't offered until evaluation completes. Acceptable and documented, not a new bug.
- **API surface parity:**
  - TUI completion and autosuggest are **not** updated. The TUI has its own completion path (`cmd/calcmark/tui/editor/*`) and is explicitly out of scope. A docs/solutions follow-up may note that the TUI can benefit from the same `EnumValues` flow once a separate issue is filed.
  - Calcmark-web is where the deletion happens — covered by the follow-up cleanup PR.
- **Integration coverage:**
  - Unit 9's acceptance table drives each R-criterion through the handler end-to-end, not just helpers in isolation — the typical failure mode where unit tests pass but the wire shape doesn't match.
  - Enum completion triggered by "inside string literal" depends on the backward walker correctly classifying the cursor; a dedicated fuzz/random case set on `extractArgumentContext` hardens this.
- **Unchanged invariants:**
  - `protocol.SignatureHelp` shape: the wrapper introduced in Unit 7 is a **superset** of the existing `protocol.SignatureHelp` shape — every field and JSON tag from the standard type is preserved, and `data` is added. Existing clients that don't know about `data` continue to work unchanged.
  - `textDocumentCompletion` return value for unchanged contexts (bare expression, after `in` / `as`, markdown) is byte-identical; only argument-context and function items are enriched.
  - Hover for units and unknown words is unchanged.

## Risks & Dependencies

| Risk | Mitigation |
|------|------------|
| Adding `data` to `ParameterInformation` is non-standard; some editors may warn on the extra field. | The LSP 3.16 spec treats unknown fields in notifications/responses as permitted. tliron/glsp forwards Go struct JSON verbatim. Acceptance-criterion R4's JSON-fixture test verifies the wire shape. |
| Backward walker misclassifies `"` inside a string, breaking enum triggering. | Extensive Unit 3 test table covering nested calls, escaped quotes, multi-arg strings. A fuzz test against random input ensures no panics. |
| `runtimeTypeToArgType` drifts when the evaluator adds a new runtime type. | Central mapper in `lsp/typemap.go` with a default-case `ArgTypeAny`; a cross-package test in Unit 2 asserts mapping for every known concrete type in `spec/types/`. |
| Argument-context detection breaks existing signatureHelp by returning the wrong param index. | Unit 3 retains `extractFunctionContext` as an adapter over the new helper. The existing `TestExtractFunctionContext` table becomes the regression gate. |
| Debounced evaluator staleness means a just-assigned variable isn't offered in completion. | Pre-existing LSP constraint documented in `docs/solutions/bugs/lsp-debounce-staleness-read-requests.md`. Not new to this plan; no mitigation beyond the existing 150ms debounce. Note in the PR description. |
| `features.Feature.Params` mirror might not pick up the new `EnumValues` field if the copy path loses it. | Go struct copy via `features[i].Params = spec.Params` (`spec/features/registry.go:354-357`) copies all fields including the new slice header. Add a one-line regression test in Unit 1 that asserts `DefaultRegistry().ByCategory(CategoryFunction)[...].Params[...].EnumValues` is non-empty for at least one function. |
| `CompletionItemKindEnumMember` may not be exported by the vendored glsp version. | Fall back to `CompletionItemKindValue` or `CompletionItemKindConstant`. Unit 4's test only asserts the `data.kind == "enum_value"` field, not the protocol Kind enum value, so the visual icon choice is independent of the contract. |

## Documentation / Operational Notes

- Update `docs/solutions/` with a new best-practices note once the feature is shipped — the argument-context detection pattern and the `ParameterInformation.data` wrapper are the kind of cross-layer learnings that belong there.
- No migration for existing users — the LSP wire changes are additive; clients that don't know about `data` keep working.
- No rollout concerns; LSP changes land with the next `cm` release.
- Calcmark-web follow-up PR (separate repo, separate issue) deletes the client-side shims per the issue body's migration plan. Coordinate timing: this LSP change must ship first and be released as a binary calcmark-web can pin to.

## Sources & References

- **Origin:** [GH issue #131](https://github.com/CalcMark/go-calcmark/issues/131) — "feat: expose structured parameter types in LSP completion and signature help"
- **Related code:**
  - `lsp/completion.go`, `lsp/signature.go`, `lsp/hover.go`, `lsp/server.go`
  - `spec/types/param_types.go`
  - `spec/identifiers/identifiers.go`
  - `spec/features/registry.go`
  - `impl/interpreter/environment.go`
- **Related solutions:**
  - `docs/solutions/best-practices/unified-feature-registry-three-to-one.md`
  - `docs/solutions/bugs/lsp-debounce-staleness-read-requests.md`
  - `docs/solutions/best-practices/directive-as-value-cross-layer-learnings.md`
  - `docs/solutions/bugs/nl-function-missing-ast-range.md`
- **Related plans:**
  - `docs/plans/2026-03-16-002-feat-ide-extensions-and-lsp-support-plan.md` (Phase 1 LSP foundation)
  - `docs/plans/2026-03-17-002-feat-ide-integration-completion-plan.md` (preview + rendering bridge)
