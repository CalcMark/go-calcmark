---
title: "feat: Frontmatter as a first-class part of the language (registry, semantic check, LSP surface)"
type: feat
status: active
date: 2026-04-14
---

# feat: Frontmatter as a first-class part of the language

## Overview

CalcMark documents already accept a YAML frontmatter block delimited by `---` lines (standard Markdown convention). The parser at `spec/document/frontmatter.go:563` (`ParseFrontmatter`) recognizes a hardcoded set of CalcMark-specific keys (`exchange`, `globals`, `scale`, `convert_to`, `measurement`, `fiscal_year_starts`) and silently captures everything else into a preserved `Extra` slice. This is the right behavior for users — non-CalcMark keys (Jekyll-style `title`, `date`, `tags`, etc.) flow through unchanged — but it means CalcMark's *own* frontmatter is parsed in detail without being treated as a first-class part of the language by the rest of the toolchain. The semantic checker only knows about `globals` and `scale` (via the minimal `FrontmatterInfo` interface at `spec/semantic/checker.go:105`); the LSP exposes nothing for frontmatter at all (no hover, completion, document symbol, or diagnostics for any frontmatter key).

This plan finishes the lift: extract the existing hardcoded allowlist into a structured registry; extend the semantic checker so registered keys with malformed values produce useful diagnostics; surface registered keys via the LSP (hover with docstring + expected type, completion at key and value positions, document symbols). Non-CalcMark keys remain ignored — they pass through to `Extra` as today, no warnings, no LSP surface, no validation. The Markdown frontmatter convention is unchanged.

This is upstream work: a separate plan in calcmark-web (or other clients) will then consume the new LSP surface to render and edit frontmatter cleanly. Until that lands, all current callers of `ParseFrontmatter` and `Checker.SetFrontmatter` continue to work unchanged — additions are additive and the existing struct shapes are preserved.

## Problem Frame

CalcMark's frontmatter is a load-bearing piece of the language:

- `globals` declares document-scoped variable bindings used throughout calc blocks.
- `convert_to` selects an output unit system (SI / Imperial) that affects every quantity rendering.
- `exchange` overrides currency rates used by every cross-currency calculation.
- `scale` multiplies every quantity in the document by a configured factor.
- `measurement` resolves ambiguous units (volume / mass / ton) for the entire document.
- `fiscal_year_starts` anchors fiscal-period date arithmetic.

Each of these affects the meaning of expressions throughout the document. Yet the toolchain treats them as opaque configuration:

- The semantic checker has no way to surface "you wrote `convert_to: xyz_not_a_currency`" or "your `exchange.USD_EUR` value isn't a number" — those errors land at runtime or silently mis-evaluate.
- Authors editing in any LSP-aware client (`go-calcmark` LSP serves VS Code, Helix, Neovim today) get no completion when typing `conv` in a frontmatter key position, no hover docs for `globals.foo`, no document outline showing the configured directives.
- `web/server.go` (calcmark-web) and the TUI both consume `ParseFrontmatter`'s output but cannot easily know "what would have been a valid value here" — there's no introspection.

The fix is the registry-of-truth pattern: name what's known, expose the metadata, let downstream consumers (semantic checker, LSP, future renderers) reflect over it. Non-CalcMark keys keep their current escape hatch (silent capture in `Extra`).

## Requirements Trace

These align with the six testable expectations captured in calcmark-web's plan `2026-04-14-003`'s "Deferred to Separate Tasks" entry, plus a registry-shape requirement.

- **R1 — Registry of CalcMark frontmatter keys.** A `spec/frontmatter` package (or `spec/document/frontmatter_registry.go`, whichever fits the repo's existing layout) exposes a typed registry: each known key has a name, an expected value type/shape, a docstring, and (where applicable) a value-domain enumerator (e.g., `convert_to` values are `si` or `imperial`). The registry is the single source of truth for "what is a CalcMark frontmatter key" — `ParseFrontmatter`, `semantic.CheckFrontmatter`, and the LSP all consult it.
- **R2 — Semantic check for registered keys.** A new `semantic.CheckFrontmatter(fm Frontmatter) []Diagnostic` validates registered keys with wrong-type values (e.g., `convert_to: xyz`, `exchange: not-a-map`, `globals: 42`). Non-registered keys produce no diagnostics. Existing semantic-checker invocations gain access to these diagnostics by calling the new function (or having it folded into `Checker.Check` if cleaner).
- **R3 — LSP hover.** `textDocument/hover` at a position inside the frontmatter region on a registered key (or a registered key's value, where the value has a known schema like `convert_to`) returns the registry's docstring + expected type. Hover on a non-registered key returns nothing (passthrough).
- **R4 — LSP completion.** `textDocument/completion` inside the frontmatter region: typing at a key position surfaces all registered keys with their docstrings; typing at a value position for a known-type key surfaces the enum values (e.g., `si` and `imperial` for `convert_to`).
- **R5 — LSP document symbol.** `textDocument/documentSymbol` includes one symbol per registered frontmatter key present in the document, so editors' outline views show the configured directives.
- **R6 — Test coverage.** Unit tests on the registry (registry shape, lookup), unit tests on `CheckFrontmatter` (each known key's failure modes), and integration tests in `lsp/acceptance_test.go` covering all three LSP surfaces (hover, completion, document symbol) for registered keys, plus negative tests confirming non-CalcMark keys produce no LSP responses.
- **R7 — Backward compatibility.** No existing `ParseFrontmatter` caller breaks. The `Frontmatter` struct shape is unchanged. The `FrontmatterInfo` interface (`spec/semantic/checker.go:105`) may grow methods but doesn't lose any. Calcmark-web's `web/server.go:309` and the TUI continue working unmodified.

## Scope Boundaries

### In scope

- New `spec/frontmatter` package OR new `spec/document/frontmatter_registry.go` with a typed registry of CalcMark-specific keys + metadata
- Refactor `ParseFrontmatter` to consult the registry instead of the hardcoded allowlist at `frontmatter.go:700-703` (no behavioral change for callers)
- New `semantic.CheckFrontmatter(fm Frontmatter) []Diagnostic` covering registered keys with malformed values
- LSP hover / completion / documentSymbol for registered keys, dispatched by detecting the cursor is inside the frontmatter region of the document
- Tests at three layers: registry unit tests, semantic-checker unit tests, LSP integration tests

### Out of scope

- Adding new CalcMark frontmatter keys beyond the existing six (this is a structural / introspection change, not a feature expansion)
- Validating non-CalcMark keys (`Extra` keys stay silently captured; this is by design — Markdown frontmatter convention preserved)
- Any frontend / editor consumer changes (calcmark-web, TUI, VS Code extension, etc.) — those are downstream plans
- Changing the YAML frontmatter delimiter convention (`---` fences stay)
- Renaming any existing `Frontmatter` struct fields (R7 backward compat)
- Editing frontmatter via LSP code actions (read-only LSP surface for now; editing happens in the editor's normal text edit path)
- The `measurement` config's user-facing integration (parsed today but not yet wired downstream — surveyed and noted; not opened here)
- Cross-document frontmatter inheritance / includes

### Deferred to Separate Tasks

- **Calcmark-web frontmatter rendering migration**: tracked in calcmark-web's plan `2026-04-14-003` Deferred entry. Becomes its own follow-up plan once this one ships and a go-calcmark release is cut.
- **TUI frontmatter editing surface**: separate plan if/when the TUI grows interactive frontmatter editing.

## Context & Research

### Relevant Code and Patterns

- `spec/document/frontmatter.go` — `ParseFrontmatter` (line 563), `Frontmatter` struct (lines 80-123), hardcoded allowlist (lines 700-703), `Extra` capture (lines 699-712). The registry replaces / wraps the allowlist; the struct stays.
- `spec/document/frontmatter.go:700-712` — the existing "silently capture unknowns into `Extra`" branch is the model for the "non-CalcMark passthrough" behavior. R7 preserves it exactly.
- `spec/semantic/checker.go:105-110` — current `FrontmatterInfo` interface (HasScale, HasGlobals, HasGlobal, GlobalKeys). May grow methods if the new `CheckFrontmatter` needs more handles, but cannot lose any.
- `spec/semantic/checker.go:147` — `SetFrontmatter(fm FrontmatterInfo)` and `Check(nodes)` (line 153). New `CheckFrontmatter` is either a separate exported function or a method on `Checker` — pick whichever reads cleaner without breaking existing callers.
- `lsp/server.go:80-124` — main server; standard LSP method dispatch lives here. Hover, completion, documentSymbol handlers are at lines 109, 111, 113.
- `lsp/handler_wrap.go` — `interceptingHandler` is where method-level intercepts already exist (used by `signatureHelp` and `hover` per the surveyed extensions). Frontmatter-region detection can hook in similarly.
- `lsp/diagnostics.go:54,103,140,148` — existing `calcmark/documentRendered` notification pattern. Custom CalcMark notifications already work; if new ones are needed for frontmatter (none expected in this plan), the pattern is established.
- `lsp/acceptance_test.go` — full notification/request integration tests. Frontmatter LSP coverage lands here.

### Institutional Learnings

- `docs/plans/2026-04-10-001-feat-lsp-structured-param-types-plan.md` — most recent LSP extension plan; structural template for LSP-touching plans in this repo.
- `docs/plans/2026-02-24-frontmatter-stability-preview-alignment-plan.md` — earlier frontmatter work; useful context on how frontmatter parsing interacts with the preview pipeline.

### External References

- CommonMark spec for frontmatter — there is no formal spec; the de-facto convention is Jekyll's YAML between `---` fences at the start of the document. CalcMark already implements this; nothing to change.
- LSP spec for `textDocument/hover`, `textDocument/completion`, `textDocument/documentSymbol` — standard. The implementations dispatch on cursor position; the only novelty here is detecting "cursor is inside the frontmatter region."

## Key Technical Decisions

- **D1. Registry shape: typed Go values, not stringly-typed map.** The registry exposes `[]RegisteredKey` where each entry has fields like `Name string`, `Type FrontmatterKeyType` (enum: `String`, `MapStringDecimal`, `MapStringString`, `EnumString`, `Struct`), `Doc string`, `EnumValues []string` (for `EnumString` / `Struct` keys with constrained value sets). Compile-time-safe, easy to iterate, easy to add new keys. *Rejected alternative: `map[string]reflect.Type` — looser, easier to write but harder to attach docstrings, harder to evolve the type taxonomy when new shapes are needed.*

- **D2. Registry lives in `spec/document/frontmatter_registry.go`, not a new package.** The registry is intimately tied to the `Frontmatter` struct's existing fields. Putting it in a new `spec/frontmatter` package would add an import cycle risk (the registry needs to know the struct shape; the parser needs to know the registry) and split related code across packages for no semantic gain. *Rejected alternative: `spec/frontmatter/registry.go` — cleaner naming but worse cohesion; defer if this grows enough to justify a new package.*

- **D3. `ParseFrontmatter` consults the registry but its public contract is unchanged.** The hardcoded allowlist at lines 700-703 becomes a `for _, k := range Registry { ... }` loop. Behavior is identical: known keys flow into typed struct fields, unknown keys flow into `Extra`. *Rejected alternative: have `ParseFrontmatter` return a `[]Diagnostic` for malformed registered values too — would expand the function's return signature and break R7. Diagnostics are the semantic checker's job (R2), not the parser's.*

- **D4. `semantic.CheckFrontmatter` is a separate exported function, not a method on `Checker`.** The semantic checker today operates on `[]ast.Node` (calc-line statements). Frontmatter validation is structural ("is this map a `map[string]decimal`?") and doesn't need the Checker's accumulated state. Keeping it separate also lets calcmark-web call it during `/api/eval` without dragging in the full Checker setup. The Checker can call it internally if the user has called `SetFrontmatter`. *Rejected alternative: add a `Checker.CheckFrontmatter()` method — couples frontmatter validation to the Checker's lifecycle for no benefit; harder to test in isolation.*

- **D5. LSP frontmatter-region detection is a pure helper.** A new `lsp/frontmatter_region.go` exports `IsInFrontmatter(source string, position protocol.Position) (region FrontmatterRegion, ok bool)` returning the parsed range plus the cursor's position within it (key vs value, which key, etc.). Hover / completion / documentSymbol all consult this helper. *Rejected alternative: inline region detection in each handler — three copies of the same parsing logic; bug-fixes have to land three times.*

- **D6. LSP completion dispatches based on cursor sub-position within the frontmatter region.** The helper from D5 returns enough info to know whether the cursor is at a key-position (start of line, before `:`) or a value-position (after `:`, on the same line or a continuation). Completion only triggers in the frontmatter region; outside the region it falls through to existing completion logic. *Rejected alternative: register a separate completion provider for the frontmatter region — LSP doesn't really support that cleanly; the same handler dispatches based on cursor context.*

- **D7. Document symbol output uses `SymbolKind.Property` for registered frontmatter keys.** Closest match in the LSP `SymbolKind` enum. Editors render it with a property icon in their outline; semantically accurate ("this is a configured property of the document"). *Rejected alternative: `SymbolKind.Variable` — would conflate frontmatter keys with calc-block variable definitions, which already use Variable.*

- **D8. No new custom `calcmark/*` LSP method.** Standard LSP methods (hover, completion, documentSymbol) cover every R1-R5 requirement. Custom methods are reserved for things outside the standard LSP vocabulary (rendering protocol, document model push). Frontmatter hover IS hover; don't invent a separate channel. *Rejected alternative: a `calcmark/frontmatterValidate` request — pointless; semantic diagnostics ride on the standard `textDocument/publishDiagnostics` notification that the existing checker already uses.*

- **D9. Registry docstrings live in code, not in a separate doc file.** Each `RegisteredKey.Doc` is a Go string literal alongside the key entry. Easier to keep in sync with the type; one place to look. If documentation grows substantial (>2-3 lines per key), revisit and consider an embedded markdown file. *Rejected alternative: load docstrings from `docs/frontmatter/*.md` files at startup — adds I/O complexity and breaks the "compile-time-safe registry" property.*

- **D10. Mind time complexity (carry-forward from calcmark-web plan 003 D9).** Every new code path in this plan is linear or constant in input size. The registry is iterated once per parse (O(|registry|) ≈ 6 today). `CheckFrontmatter` is O(|registered keys present|). Frontmatter region detection is a single pass over the source up to the closing `---` fence (O(|frontmatter source|)). Hover / completion lookups are O(1) given the cursor position. No quadratic anywhere. *Rejected alternative: lazy / cached registry that builds an index on first access — premature; a 6-entry registry doesn't need indexing, and adding caching adds invalidation complexity.*

## Open Questions

### Resolved During Planning

- **Should the registry expose `Frontmatter` struct field paths so consumers can populate the typed struct from the registry?** No. The struct already exists; the registry is metadata about keys, not a code generator. `ParseFrontmatter` keeps its switch-style assignment to typed struct fields; the registry just provides the key names + types for validation and LSP introspection.
- **Should `Extra` keys ever produce diagnostics?** No. The user explicitly wants non-CalcMark keys ignored. Capturing in `Extra` preserves them for round-trip; producing a warning would surprise users who use frontmatter for Jekyll-style metadata.
- **Should this plan touch the `measurement` config's downstream wiring?** No. The survey flagged it as parsed-but-not-fully-wired, but that's a separate concern. This plan adds LSP/semantic surface for `measurement` like any other registered key; broader integration is a different plan.
- **Should the LSP advertise frontmatter completion at document position 0 even before the user types `---`?** No. Completion only fires when the cursor is already inside an opened frontmatter region (between two `---` fences). The author has signaled intent by typing the opening fence.

### Deferred to Implementation

- **Exact `FrontmatterKeyType` enum values.** D1 names a starting set (`String`, `MapStringDecimal`, `MapStringString`, `EnumString`, `Struct`); the implementer audits the existing six keys and lands on the precise set. Likely additions: `MapStringStruct` for `measurement` (which has nested axis fields). Document choices in code.
- **How to dispatch completion at a value position when the value type is `Struct`.** For `scale: { factor: 1000 }`, value-position completion at the `factor:` key needs the registry to expose nested field metadata. May require a small recursive registry shape; defer to implementation. Acceptable POC fallback: no completion inside struct values; just at the top-level key + simple-type value positions.
- **Whether `CheckFrontmatter` is invoked automatically by `Checker.Check` when frontmatter has been set.** Convenience vs separation: probably yes, via a one-liner inside `Check` that prepends frontmatter diagnostics. Implementer decides based on whether existing `Checker` consumers expect frontmatter diagnostics in the same return value or separately.

## High-Level Technical Design

> *This illustrates the intended approach and is directional guidance for review, not implementation specification. The implementing agent should treat it as context, not code to reproduce.*

### Registry shape (directional)

```
// Lives at spec/document/frontmatter_registry.go

type FrontmatterKeyType int
const (
    KeyTypeString FrontmatterKeyType = iota
    KeyTypeMapStringDecimal      // exchange
    KeyTypeMapStringString       // globals
    KeyTypeEnumString            // convert_to (enum: si | imperial)
    KeyTypeStruct                // scale, measurement, fiscal_year_starts
)

type RegisteredKey struct {
    Name       string             // YAML key as authored
    Type       FrontmatterKeyType
    Doc        string             // shown in LSP hover
    EnumValues []string           // populated when Type == KeyTypeEnumString
    // Future: nested-field metadata for KeyTypeStruct entries
}

var Registry = []RegisteredKey{
    { Name: "exchange", Type: KeyTypeMapStringDecimal, Doc: "..." },
    { Name: "globals", Type: KeyTypeMapStringString, Doc: "..." },
    { Name: "scale", Type: KeyTypeStruct, Doc: "..." },
    { Name: "convert_to", Type: KeyTypeEnumString, Doc: "...", EnumValues: []string{"si","imperial"} },
    { Name: "measurement", Type: KeyTypeStruct, Doc: "..." },
    { Name: "fiscal_year_starts", Type: KeyTypeStruct, Doc: "..." },
}

func IsRegisteredKey(name string) bool { ... }   // O(|Registry|), trivially small
func LookupKey(name string) (RegisteredKey, bool) { ... }
```

### Semantic check (directional)

```
// Lives at spec/semantic/frontmatter_check.go

func CheckFrontmatter(fm document.Frontmatter) []Diagnostic {
    // For each populated typed-struct field on fm, validate it matches
    // the registered key's type.
    // For each Extra key: skip (passthrough).
    // For each registered key with malformed shape (e.g., the parser
    // captured raw YAML for an enum key but the value isn't in EnumValues):
    // emit a Diagnostic with severity=Error and a useful Range pointing at
    // the offending source line if available.
}
```

Note: `ParseFrontmatter` may need to surface line/column ranges per key in the `Frontmatter` struct (or via a parallel ranges-by-key map) so diagnostics can have anchors. Detail deferred to implementation; the parser today doesn't keep these but the YAML library it uses likely does.

### LSP frontmatter-region detection (directional)

```
// Lives at lsp/frontmatter_region.go

type FrontmatterRegion struct {
    StartLine int           // first '---' fence line
    EndLine   int           // closing '---' fence line
    KeyLines  map[int]string // line index -> key name (registered or not)
}

type CursorContext struct {
    InRegion bool
    Position string         // "key" | "value" | "fence" | "outside"
    Key      string         // empty if Position == "key" and user hasn't typed yet
}

func DetectRegion(source string) (FrontmatterRegion, bool) { ... }
func ClassifyCursor(region FrontmatterRegion, pos protocol.Position) CursorContext { ... }
```

These two functions are the foundation for hover / completion / documentSymbol. They're pure (source string in, structural data out) — easy to unit-test exhaustively.

## Implementation Units

- [x] **Unit 1: Registry — extract the allowlist into a typed `Registry` slice**

**Goal:** A new file `spec/document/frontmatter_registry.go` defines `RegisteredKey`, `FrontmatterKeyType`, and a populated `Registry` covering the six existing keys with docstrings and (for `convert_to`) enum values. `ParseFrontmatter` continues working unchanged.

**Requirements:** R1, R7

**Dependencies:** None

**Files:**
- Create: `spec/document/frontmatter_registry.go`
- Create: `spec/document/frontmatter_registry_test.go`

**Approach:**
- Define types per D1.
- Populate `Registry` for the six existing keys. Docstrings: write 1-2 sentences each, using existing `frontmatter.go` parse-block comments as source material.
- Helpers: `IsRegisteredKey(name string) bool`, `LookupKey(name string) (RegisteredKey, bool)`. Both linear over `Registry` (≤6 entries today; D10 keeps it that way).
- `ParseFrontmatter` is NOT modified in this unit — that's Unit 2. This unit is purely additive; existing tests stay green.

**Execution note:** Test-first. Cover registry shape (each entry has non-empty Name/Doc, EnumValues populated iff Type==EnumString), lookups (existing keys found, unknown keys not found, case-sensitivity behavior — pick and document).

**Patterns to follow:**
- Existing `spec/document/frontmatter.go` for naming conventions.
- `spec/types/` for enum-with-stringer pattern if applicable.

**Test scenarios:**
- Happy: each of the 6 registered keys returns from `LookupKey` with its expected `Type`
- Happy: `convert_to` has `EnumValues == []string{"si","imperial"}` (or whatever the parser actually accepts — verify against `frontmatter.go:330` and adjacent code)
- Happy: every entry has a non-empty `Doc`
- Edge: `IsRegisteredKey("Exchange")` (capitalized) returns false (case-sensitive, matches YAML behavior; document)
- Edge: `LookupKey("nonexistent_key")` returns `_, false`
- Regression: existing `frontmatter_test.go` continues passing (no behavioral change)

**Verification:**
- `go test ./spec/document/... -run Registry` green
- Existing `go test ./...` unchanged

---

- [x] **Unit 2: Refactor `ParseFrontmatter` to consult the registry**

**Goal:** Replace the hardcoded allowlist at `frontmatter.go:700-703` with a `Registry`-driven check. No behavioral change for any caller — known keys still populate typed `Frontmatter` struct fields; unknown keys still flow into `Extra`. The change is one of provenance: the source of truth for "which keys are CalcMark-known" moves from a hardcoded slice into the new registry.

**Requirements:** R1, R3, R7

**Dependencies:** Unit 1

**Files:**
- Modify: `spec/document/frontmatter.go` (around lines 700-712)
- Modify: `spec/document/frontmatter_test.go` if any test asserts on the old allowlist's exact shape (likely none; behavior is identical)

**Approach:**
- Replace the inline allowlist literal with `IsRegisteredKey(yamlKey)` calls.
- Verify existing tests still pass — this is a refactor.
- Preserve the existing parsing branch for each typed key (`exchange`, `globals`, etc.) — those switch arms stay; only the "is this a known CalcMark key vs goes into Extra" decision changes its source.

**Execution note:** This is a refactor; no test-first ceremony beyond ensuring the existing test suite still passes. If any new test scenarios become natural to add (e.g., "adding a key to the Registry makes it parsed by `ParseFrontmatter` without further code changes"), add them.

**Patterns to follow:**
- Existing `ParseFrontmatter` structure.

**Test scenarios:**
- Regression: every existing `frontmatter_test.go` scenario stays green
- Integration: if a new key were added to `Registry` (in a hypothetical future change), `ParseFrontmatter` would route it through the typed-struct path. This is an architectural property; document it in a code comment.

**Verification:**
- `go test ./spec/document/...` green
- `git diff` for `frontmatter.go` is small and isolated to the allowlist site

---

- [x] **Unit 3: Capture frontmatter source ranges (per-key positions)**

**Goal:** Extend `Frontmatter` (or sidecar data) with line/column ranges per parsed key, so semantic diagnostics and LSP hover/completion can anchor to source positions. Backward-compatible — additive, no field renames or removals.

**Requirements:** R2, R3, R5, R7

**Dependencies:** None (orthogonal to Units 1-2; can run in parallel)

**Files:**
- Modify: `spec/document/frontmatter.go`
- Modify: `spec/document/frontmatter_test.go`

**Approach:**
- Likely shape: a new `KeyRanges map[string]ast.Range` field on `Frontmatter` (or a separate `Ranges` struct returned alongside; pick whatever interferes least with R7's backward-compat). The map carries one entry per registered key found in the source.
- The YAML library go-calcmark uses likely exposes node positions; if not, instrument the parsing pass to record start/end lines as it walks the YAML tree.
- For the `Extra` slice, also capture ranges so future tooling can locate non-CalcMark keys (cheap addition; no extra parser passes).
- 0-based or 1-based: match the existing `ast.Range` convention in this repo (verify; calcmark-web's plan 003 D2 is 0-based for new protocol but go-calcmark may differ — document the chosen convention in a code comment).

**Execution note:** Test-first. Fixture: a multi-key frontmatter with known line numbers; assert each key's range matches.

**Patterns to follow:**
- `ast.Range` shape and existing `_test.go` fixtures using it.

**Test scenarios:**
- Happy: a frontmatter with `convert_to: si` on line 2 surfaces a range with `Start.Line == 2` (or 1, per convention)
- Happy: a multi-line value (`globals:\n  foo: ...\n  bar: ...`) on lines 3-5 surfaces a range covering the full block
- Edge: missing closing `---` fence → behavior should be the existing parser's (likely returns no frontmatter); ranges absent
- Edge: empty frontmatter (`---\n---\n`) → empty `KeyRanges` map, no error
- Regression: all existing tests stay green; the new field is additive

**Verification:**
- `go test ./spec/document/...` green
- New field documented in `Frontmatter` struct comment

---

- [x] **Unit 4: `semantic.CheckFrontmatter` — diagnostics for malformed registered keys**

**Goal:** A new exported function `semantic.CheckFrontmatter(fm document.Frontmatter) []semantic.Diagnostic` that returns diagnostics for registered keys whose values don't match their registered type. Non-registered keys (`Extra`) produce no diagnostics. Existing semantic-check callers gain access via either calling the new function directly or via `Checker.Check` automatically forwarding when frontmatter has been set (D4 + Open Questions Deferred).

**Requirements:** R2, R6

**Dependencies:** Unit 1, Unit 3

**Files:**
- Create: `spec/semantic/frontmatter_check.go`
- Create: `spec/semantic/frontmatter_check_test.go`
- Possibly modify: `spec/semantic/checker.go` if folding into `Checker.Check` ends up cleaner

**Approach:**
- For each registered key, check whether the corresponding `Frontmatter` field is populated AND whether the populated value matches the registry's `Type`. Most cases the parser already enforces — but for `EnumString` keys the parser may accept any string and the validation lives here.
- For `convert_to: xyz_not_a_value`, emit `Diagnostic{Severity: Error, Message: "..."}` with a Range from Unit 3's `KeyRanges`.
- For value-shape mismatches the parser couldn't catch (e.g., `globals: 42` — the parser fails YAML-shape; if so the parser's own error is already returned; check whether semantic gets a chance) — confirm during implementation; add tests for both parser-caught and semantic-caught cases.
- Diagnostic messages: short, actionable, including the offending value when safe.

**Execution note:** Test-first. Each registered key gets at least one happy and one failure scenario. Use `KeyRanges` to verify Range anchors.

**Patterns to follow:**
- Existing `Diagnostic` shape in `spec/semantic/`.
- Existing `Checker.Check` test fixtures.

**Test scenarios:**
- Happy: `Frontmatter{ ConvertTo: &ConvertToConfig{Value: "si"} }` → no diagnostics
- Failure: `Frontmatter{ ConvertTo: &ConvertToConfig{Value: "xyz"} }` → one diagnostic, Severity=Error, Range from KeyRanges["convert_to"]
- Failure: `Frontmatter{ Globals: map[string]string{} }` is fine (empty map is valid); but `Globals` with a key whose value is unparseable as a CalcMark expression — out of this plan's scope (calc-line semantic check covers it later)
- Failure: `Exchange: map[string]decimal.Decimal{"USD_EUR": 0}` — emit warning if zero rates are nonsensical (decide; document)
- Edge: empty `Frontmatter{}` → no diagnostics
- Edge: `Frontmatter{ Extra: []ExtraField{{Key: "title", Value: "Hello"}} }` → no diagnostics (Extra is passthrough)
- Edge: `Frontmatter{ ConvertTo: &ConvertToConfig{Value: "SI"} }` (case mismatch on enum value) → either accept (case-insensitive enums) or reject (exact match); pick and document
- Regression: existing `Checker.Check` callers see no behavioral change unless they opt into frontmatter checking

**Verification:**
- `go test ./spec/semantic/... -run Frontmatter` green
- Diagnostics carry usable Range anchors (verified via at least one assertion that Range != zero-value)

---

- [x] **Unit 5: LSP frontmatter-region detection helper**

**Goal:** Pure helpers `DetectRegion(source string) (FrontmatterRegion, bool)` and `ClassifyCursor(region, position) CursorContext` in `lsp/frontmatter_region.go`. These are the foundation for Units 6, 7, 8 (hover, completion, documentSymbol).

**Requirements:** R3, R4, R5

**Dependencies:** None (parsing is independent of registry; can run in parallel with Units 1-4)

**Files:**
- Create: `lsp/frontmatter_region.go`
- Create: `lsp/frontmatter_region_test.go`

**Approach:**
- `DetectRegion`: scan the source for the opening `---` fence (must be at line 0 with optional whitespace) and the closing `---` fence. If both found, return the region with start/end line indexes and a `KeyLines` map (`line index → key name`). If frontmatter is malformed or absent, return `_, false`.
- `ClassifyCursor`: given a region and a `protocol.Position`, classify the cursor's role: `outside` (not in region), `fence` (on a `---` line), `key` (start of a `key:` line, before the `:`), `value` (after the `:` on the same line, OR on a continuation indented block).
- Pure functions, single-pass over the source. O(|frontmatter source|) per D10.

**Execution note:** Test-first. The classifier has well-defined rules; cover each.

**Patterns to follow:**
- Existing pure helpers in `spec/document/` for source-walking style.

**Test scenarios:**
- Happy: well-formed frontmatter → DetectRegion returns the right line range; KeyLines contains every top-level key
- Happy: ClassifyCursor at line 1 col 0 (just after the opening fence) → `key` position
- Happy: ClassifyCursor at line 1 col 12 of `convert_to: si` → `value` position
- Edge: missing closing fence → `_, false`
- Edge: only opening fence → `_, false`
- Edge: cursor outside the region → `outside`
- Edge: cursor on the `---` line itself → `fence`
- Edge: cursor on a continuation line (e.g., `globals:\n  foo: ...` with cursor on the `foo:` line) → `value` (or document otherwise)
- Edge: fenced frontmatter with empty body (`---\n---\n`) → DetectRegion returns region with empty KeyLines; cursor on line 1 (between fences) → `key` (typing-position)

**Verification:**
- `go test ./lsp/... -run FrontmatterRegion` green
- Helpers are pure; no LSP dependencies

---

- [x] **Unit 6: LSP hover for registered frontmatter keys**

**Goal:** `textDocument/hover` at a position inside the frontmatter region, on a registered key (or a registered enum value), returns the registry's docstring + expected type formatted as Markdown. Hover on a non-registered key returns nothing (passthrough — null hover response).

**Requirements:** R3, R6

**Dependencies:** Unit 1, Unit 5

**Files:**
- Modify: `lsp/server.go` (the existing hover handler at line 111)
- Possibly modify: `lsp/handler_wrap.go` if the existing intercepting layer is the right injection point for frontmatter-aware hover
- Modify: `lsp/server_test.go` and `lsp/acceptance_test.go`

**Approach:**
- In the hover handler, call `DetectRegion(source)` first. If outside the region, fall through to existing hover logic.
- If inside the region, call `ClassifyCursor` to identify the key under the cursor (or the key whose value contains the cursor).
- Look up via `LookupKey`. If found, format hover content as Markdown with key name, type, and docstring. If not found, return null hover.
- Hover on an enum value (e.g., cursor on `si` in `convert_to: si`) returns the registry's docstring + a note that this is a valid value (or just the parent key's hover).

**Execution note:** Integration-test in `acceptance_test.go` — open a doc, request hover at a frontmatter position, assert the response.

**Patterns to follow:**
- Existing hover handler (line 111 of `server.go`) and any existing acceptance tests for hover.

**Test scenarios:**
- Happy: hover on `convert_to` key returns docstring and `Type: enum (si | imperial)`
- Happy: hover on `globals` key returns docstring
- Happy: hover on `si` value of `convert_to: si` returns docstring (acceptable to delegate to the parent key's hover)
- Edge: hover on an `Extra` key like `title: Hello` returns null (no hover response)
- Edge: hover outside the frontmatter region falls through to existing hover (regression: existing hover tests stay green)
- Edge: hover on the `---` fence returns null

**Verification:**
- `go test ./lsp/... -run Hover` green
- `lsp/acceptance_test.go` includes the hover scenarios above

---

- [x] **Unit 7: LSP completion for registered keys + enum values**

**Goal:** `textDocument/completion` inside the frontmatter region surfaces:
- At a `key` position: all registered keys with their docstrings as completion items
- At a `value` position for a `EnumString`-typed key: the enum values

Outside the region, existing completion logic runs unchanged.

**Requirements:** R4, R6

**Dependencies:** Unit 1, Unit 5

**Files:**
- Modify: `lsp/server.go` (the existing completion handler at line 109)
- Modify: `lsp/server_test.go` and `lsp/acceptance_test.go`

**Approach:**
- In the completion handler, call `DetectRegion`. Outside region → fall through.
- Inside region, `ClassifyCursor` to determine key vs value position.
- Key position: build CompletionItems from `Registry`, each with kind `Property`, label = key name, documentation = docstring, sort-key alphabetic.
- Value position for enum-typed key: build CompletionItems from `EnumValues`, each with kind `EnumMember`.
- Value position for non-enum-typed key: no completions for this plan; punt to a future iteration.

**Execution note:** Test-first.

**Patterns to follow:**
- Existing completion handler.

**Test scenarios:**
- Happy: typing `con` at a key position surfaces `convert_to` in completions
- Happy: typing at a fresh key line (empty input) surfaces all 6 registered keys
- Happy: typing at the value position of `convert_to: ` (cursor after the colon+space) surfaces `si` and `imperial`
- Edge: completion at value position of `globals: ` returns no completions (non-enum type; not in scope)
- Edge: completion outside the frontmatter region returns existing completions (regression)
- Edge: completion on the `---` fence returns no completions

**Verification:**
- `go test ./lsp/... -run Completion` green
- Acceptance tests cover the scenarios above

---

- [x] **Unit 8: LSP documentSymbol for registered frontmatter keys**

**Goal:** `textDocument/documentSymbol` includes one symbol per registered frontmatter key present in the document, with `SymbolKind.Property`. The symbols sit at the top of the symbol list (before calc-block variables), so editor outlines show frontmatter at the top.

**Requirements:** R5, R6

**Dependencies:** Unit 1, Unit 3, Unit 5

**Files:**
- Modify: `lsp/server.go` (existing documentSymbol handler at line 113)
- Modify: `lsp/server_test.go` and `lsp/acceptance_test.go`

**Approach:**
- Detect the region; if absent, call existing logic.
- For each `KeyLines` entry whose key is registered, emit a `DocumentSymbol{Name: key, Kind: Property, Range: KeyRanges[key], SelectionRange: ...}`.
- Append existing calc-block symbols.

**Execution note:** Test-first.

**Patterns to follow:**
- Existing documentSymbol handler.

**Test scenarios:**
- Happy: doc with `convert_to: si` produces one symbol with name `convert_to`, kind `Property`
- Happy: doc with multiple registered keys produces multiple symbols in source order
- Edge: doc with `Extra` keys produces no symbols for them
- Edge: doc without frontmatter produces no frontmatter symbols (regression: calc-block symbols unaffected)

**Verification:**
- `go test ./lsp/... -run DocumentSymbol` green
- Acceptance tests cover the scenarios above

---

- [x] **Unit 9: Acceptance tests + close the loop**

**Goal:** End-to-end acceptance tests in `lsp/acceptance_test.go` verifying all six R-numbered behaviors from a real LSP client perspective. Update relevant docs.

**Requirements:** R6, plus durable knowledge capture

**Dependencies:** Units 1-8

**Files:**
- Modify: `lsp/acceptance_test.go` — add a frontmatter-focused scenario block
- Modify: `docs/plans/2026-04-14-001-feat-frontmatter-first-class-plan.md` — mark all unit checkboxes complete
- Possibly create: a short solutions doc in `docs/solutions/` or equivalent if the repo uses one
- Modify: any user-facing doc that lists supported frontmatter keys (e.g., README, language reference) to point at `Registry` as the source of truth

**Approach:**
- Acceptance scenario: open a doc with mixed registered + Extra frontmatter keys; verify hover, completion, documentSymbol, and diagnostics behave per R1-R6.
- Negative scenarios: verify Extra keys produce no LSP responses (no hover, no completion, no symbol, no diagnostic).
- Regression: existing acceptance tests for non-frontmatter LSP behavior continue passing.

**Execution note:** Pure verification + docs.

**Test scenarios:**
- Integration: full LSP session opens a fixture doc; one pass verifies R1-R6 against the live LSP server
- Integration: a fixture with only `Extra` keys (Jekyll-style) → LSP responses for frontmatter are empty across all four LSP methods
- Regression: existing `acceptance_test.go` tests pass

**Verification:**
- `task test` (or `go test ./...`) green
- Plan checkboxes all `[x]`

## System-Wide Impact

- **Interaction graph:** `ParseFrontmatter` consults `Registry` (Unit 2); `semantic.CheckFrontmatter` consults `Registry` and `KeyRanges` (Unit 4); LSP hover/completion/documentSymbol consult `Registry` + region helpers (Units 6/7/8). The registry is the new fan-in point — adding a new CalcMark key in the future means: (a) add to `Registry`, (b) extend the parser switch in `ParseFrontmatter`, (c) every LSP surface picks it up automatically.
- **Error propagation:** Diagnostics from `CheckFrontmatter` flow into the standard `[]Diagnostic` return; consumers that already render diagnostics (calcmark-web, TUI, LSP `publishDiagnostics`) get them for free once they call the new function.
- **State lifecycle risks:** None — `Registry` is a package-level constant slice, no mutable state. Region detection is per-call.
- **API surface parity:** Public APIs grow additively (new exported types in `spec/document` and `spec/semantic`, new helpers in `lsp/`). No existing public function signature changes (R7).
- **Integration coverage:** Three layers — registry unit tests (Unit 1), semantic-check unit tests (Unit 4), LSP acceptance tests (Units 6/7/8/9). Each layer catches a different class of regression.
- **Unchanged invariants:**
  - `Frontmatter` struct field names and types
  - `ParseFrontmatter` signature and behavior for valid input
  - Non-CalcMark keys flow into `Extra` and produce no diagnostics / no LSP responses (the user's "ignore everything except calcmark-specific" requirement)
  - YAML frontmatter delimiter convention (`---` fences at document start)
  - `FrontmatterInfo` interface keeps its existing methods (may grow new ones)

## Risks & Dependencies

| Risk | Mitigation |
|---|---|
| The YAML library go-calcmark uses doesn't expose per-key positions (Unit 3 blocker) | Implementer audits the library; if positions aren't exposed, instrument the parsing pass to record line numbers as it walks. The cost is small (one int per key) and the value (LSP anchoring) is high. |
| `CheckFrontmatter` diagnostics mostly duplicate parser errors (e.g., the parser already rejects malformed `globals`) | Verify during Unit 4. If the overlap is significant, scope `CheckFrontmatter` to only the cases the parser CAN'T catch (enum value validation, cross-key consistency, etc.). Adjust R2 in the plan if so. |
| Adding completion / hover changes the LSP behavior for users who currently get nothing for frontmatter — could surface as unexpected popups | All new LSP behavior is opt-in by cursor position; falls through silently outside the frontmatter region. Risk is low; surface in any release notes. |
| `SymbolKind.Property` icon doesn't render meaningfully in some editor (VS Code, Helix, Neovim differ) | Acceptable cosmetic difference; SymbolKind.Property is the most semantically accurate. Document in Unit 8 if a specific editor breaks. |
| The registry grows enough to need indexing (D10's "small enough" assumption fails) | The current 6 entries grow slowly; adding indexing is trivial when it matters. Defer until the registry has >20 entries or hot-path profiling shows the iteration cost. |
| Backward compat regression — calcmark-web's `web/server.go` or the TUI breaks because of a subtle parser change | R7 is the gate. Run the full `go test ./...` after every unit; verify the calcmark-web test suite (separate repo, but `go-calcmark` is its dep) still works after each go-calcmark release tag. |
| `convert_to` enum values not actually `si`/`imperial` (registry shape wrong from the start) | Unit 1's implementer confirms the actual accepted values by reading `frontmatter.go:330` and adjacent parsing code; updates the registry to match reality. |

## Documentation / Operational Notes

- New exported types: `RegisteredKey`, `FrontmatterKeyType`, `Registry`, `IsRegisteredKey`, `LookupKey`, `CheckFrontmatter`, `DetectRegion`, `ClassifyCursor`. Public API documentation comments are part of each unit's deliverable.
- After Unit 9 lands and the next go-calcmark release is cut, calcmark-web's plan `2026-04-14-003`'s Deferred entry "First-class frontmatter handling in go-calcmark" can be marked unblocked, and a follow-up calcmark-web plan for frontend frontmatter rendering can be written.
- No deploy-time changes; this is a library release.
- Release notes (in the next go-calcmark version): "Frontmatter is now a first-class part of the language. Editors with LSP support get hover docs, completion, and document outline for CalcMark-specific frontmatter keys (`exchange`, `globals`, `scale`, `convert_to`, `measurement`, `fiscal_year_starts`). Non-CalcMark keys (Jekyll-style `title`, `date`, etc.) continue to pass through unmodified."

## Autonomous Execution Guidance

- Each unit is independently committable. Phases 1-2 (Units 1-4) can run sequentially or with Units 1+3 parallelized.
- Test-first is the default execution posture. Each unit's first commit is a failing test for the new behavior.
- Mind D10 (time complexity): every new code path is linear or constant in input size. The `Registry` is iterated linearly; that's fine at 6 entries. Region detection is single-pass. If the implementer reaches for a nested loop or repeated re-scan, stop and look for a one-pass alternative.
- If Unit 3 (per-key ranges) reveals that the YAML library doesn't expose positions, the implementer scopes the work to manual instrumentation in the parsing pass — don't switch YAML libraries for this.
- If `CheckFrontmatter` (Unit 4) finds that most of its target diagnostics are already produced by the parser, scope it to the residual cases and update the plan's R2 inline rather than producing duplicate diagnostics.
- If any unit fails its verification, stop and commit what works as a documentation-only escalation; do not push through a broken state.

## Resolution

Shipped: CalcMark frontmatter is now a first-class language concept with a typed `Registry`, per-key source ranges, standalone semantic validation (`semantic.CheckFrontmatter`), and LSP hover, completion, and documentSymbol coverage for registered keys — while non-CalcMark (Extra) keys continue to pass through untouched. Downstream consumers (`calcmark-web`, TUI, any LSP client) now have a single source of truth for CalcMark-specific frontmatter semantics.

| Unit | Commit |
|---|---|
| 1 — Registry | `b412a9b` |
| 2 — ParseFrontmatter consults Registry | `76473cf` |
| 3 — KeyRanges capture | `f646cad` |
| 4 — semantic.CheckFrontmatter | `d96fae6` |
| 5 — LSP region detection helpers | `9ad815a` |
| 6 — LSP hover | `34f604b` |
| 7 — LSP completion | `a67f7dd` |
| 8 — LSP documentSymbol | `a4859f7` |
| 9 — Consolidated acceptance + close the loop | (this commit) |

Follow-up handoff: calcmark-web plan `2026-04-14-003`'s Deferred entry "First-class frontmatter handling in go-calcmark" is now unblocked. A new calcmark-web plan for frontend frontmatter rendering can be written against the next go-calcmark release tag.

## Sources & References

- Survey of current state: `spec/document/frontmatter.go` (line refs above), `spec/semantic/checker.go`, `lsp/server.go`, `lsp/diagnostics.go`, `lsp/acceptance_test.go`
- Calcmark-web plan that asks for this work: `../calcmark-web/.claude/worktrees/plan-interpreter-driven-editor/docs/plans/2026-04-14-003-feat-v2-text-frontmatter-retire-legacy-plan.md` (Deferred to Separate Tasks > "First-class frontmatter handling in go-calcmark")
- Recent precedent for LSP extension: `docs/plans/2026-04-10-001-feat-lsp-structured-param-types-plan.md`
- Frontmatter-related earlier work: `docs/plans/2026-02-24-frontmatter-stability-preview-alignment-plan.md`
- Markdown frontmatter convention: de-facto Jekyll YAML between `---` fences (no formal spec)
- LSP spec: `textDocument/hover`, `textDocument/completion`, `textDocument/documentSymbol`, `SymbolKind.Property`
