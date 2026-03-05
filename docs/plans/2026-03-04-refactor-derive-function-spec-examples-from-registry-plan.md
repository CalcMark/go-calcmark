---
title: "Derive FunctionSpec examples from feature registry identifiers"
type: refactor
status: completed
date: 2026-03-04
deepened: 2026-03-04
---

# Derive FunctionSpec Examples from Feature Registry Identifiers

## Enhancement Summary

**Deepened on:** 2026-03-04
**Research agents used:** pattern-recognition-specialist, performance-oracle, code-simplicity-reviewer, architecture-strategist, learnings-researcher, codebase-explorer

### Key Improvements from Research

1. **New `spec/identifiers/` leaf package** instead of `spec/features/identifiers.go` — avoids coupling `spec/semantic` to the UI-oriented `spec/features` catalog package (architecture-strategist)
2. **Plain `[]string` slices** instead of `Identifier` struct — the `Description` field has no consumer in the derivation chain; every target needs only names (code-simplicity-reviewer)
3. **Test-time validation** instead of `init()` panics — panics crash the binary for all users even if only one identifier drifts; tests provide the same guarantee safely (code-simplicity-reviewer, architecture-strategist)
4. **Bidirectional validation** — check both directions: every canonical name has a map entry AND every map key is canonical (pattern-recognition-specialist)
5. **Quoting logic stays in the consumer** — `QuotedPrimaryNames` mixes presentation into the data package; move quoting to `function_types.go` (pattern-recognition-specialist)
6. **Pre-allocate in `AllNames()`** — follow codebase convention of `make([]string, 0, n)` (performance-oracle)

### New Considerations Discovered

- `spec/semantic` currently has zero dependency on `spec/features` — adding one violates the layering principle (semantic analysis should not depend on a UI catalog)
- The `Identifier.Description` field is consumed by zero of the four derivation targets
- The `Aliases` field serves only 1 of ~20 identifiers — a separate `map[string]string` is simpler
- Go map non-deterministic ordering learning confirms ordered slices are the correct choice for canonical lists
- Error messages follow a consistent `"unknown X 'Y' (valid Xs: ...)"` pattern across all interpreter files

---

## Overview

Consolidate duplicated function parameter identifiers (network scopes, network types, storage types, compression types) into a single canonical source in a new `spec/identifiers/` package. Three independent sources have drifted apart, causing bugs like `throughput()` autocomplete showing wrong values and `read()`/`seek()` suggesting `"memory"` which causes a runtime error. A single source of truth eliminates this entire class of bug.

## Problem Statement

Valid identifier values are hardcoded in four places that cannot stay in sync:

| Duplication Site | File | What It Stores |
|---|---|---|
| 1. ParamSpec.Examples | `spec/semantic/function_types.go` | String lists for TUI autocomplete hints |
| 2. Feature entries | `spec/features/registry.go` | Individual Feature structs for search/autocomplete |
| 3. Runtime maps | `impl/interpreter/*_functions.go` | `map[string]float64` lookup tables |
| 4. Error messages | `impl/interpreter/*_functions.go` | Hardcoded "valid types: ..." strings in `fmt.Errorf` |

**Confirmed drift:**

| Identifier | ParamSpec.Examples | registry.go | Interpreter map |
|---|---|---|---|
| `hundred_gig` | MISSING (throughput, transfer_time) | YES | YES |
| `four_g`, `five_g` | MISSING (transfer_time) | YES | YES |
| `sata_ssd`, `nvme`, `pcie_ssd` | MISSING (read, seek) | YES | YES |
| `lz4`, `bzip2` | MISSING (compress) | YES | YES |
| `none` | MISSING (compress) | NO ENTRY | YES (ratio 1.0) |
| `memory` | YES (read, seek) | NO ENTRY | NO MAP KEY |

The `memory` bug (TUI suggests it, runtime rejects it) is out of scope for this refactoring — it will naturally be excluded from the canonical list since it has no interpreter mapping.

## Proposed Solution

### New canonical source: `spec/identifiers/identifiers.go`

A dedicated leaf package with zero internal dependencies, following the same pattern as `spec/units/canonical.go`. Uses plain `[]string` slices — the simplest data structure that every consumer actually needs.

```go
package identifiers

// NetworkScopes lists valid scope identifiers for rtt() and transfer_time().
// Ordered by most commonly used first — first element is the representative
// example shown in the TUI compact footer (view_footer.go:144 displays Examples[0]).
var NetworkScopes = []string{"local", "regional", "continental", "global"}

// NetworkTypes lists valid network type identifiers for throughput() and transfer_time().
var NetworkTypes = []string{"gigabit", "ten_gig", "hundred_gig", "wifi", "four_g", "five_g"}

// StorageTypes lists valid storage type identifiers for read() and seek().
var StorageTypes = []string{"ssd", "nvme", "pcie_ssd", "hdd"}

// StorageAliases maps alternative storage names to their canonical name.
// The interpreter accepts both the canonical name and any alias.
// Only ssd/sata_ssd needs this — sata_ssd resolves to the same behavior as ssd.
var StorageAliases = map[string]string{"sata_ssd": "ssd"}

// CompressionTypes lists valid compression type identifiers for compress().
var CompressionTypes = []string{"gzip", "zstd", "lz4", "snappy", "bzip2", "none"}
```

### Helper functions in `spec/identifiers/identifiers.go`

```go
// AllStorageNames returns all valid storage names including aliases.
// Used by interpreter validation and error messages.
func AllStorageNames() []string {
    n := len(StorageTypes) + len(StorageAliases)
    names := make([]string, 0, n)
    names = append(names, StorageTypes...)
    for alias := range StorageAliases {
        names = append(names, alias)
    }
    return names
}
```

### Quoting helper stays in the consumer

The `spec/semantic/function_types.go` file owns the quoting logic since it's a presentation concern specific to `ParamSpec.Examples`:

```go
// quotedNames wraps each name in double quotes for ParamSpec.Examples display.
// Computed once at package init via the var block — not called per-access.
func quotedNames(names []string) []string {
    quoted := make([]string, len(names))
    for i, n := range names {
        quoted[i] = `"` + n + `"`
    }
    return quoted
}
```

### Derivation chain

```
spec/identifiers/identifiers.go  (CANONICAL SOURCE — leaf package, zero imports)
    |
    +---> spec/features/registry.go        (Feature entries use identifier names, keep own descriptions)
    +---> spec/semantic/function_types.go   (ParamSpec.Examples = quotedNames(identifiers.NetworkTypes))
    +---> impl/interpreter/*_functions.go   (map keys validated against canonical names via tests)
    +---> impl/interpreter/*_functions.go   (error messages use identifiers.NetworkTypes for valid values)
```

### Research Insights

**Architecture (architecture-strategist):** Placing identifiers in `spec/features/` would force `spec/semantic` to depend on a UI catalog package, violating layering. A dedicated `spec/identifiers/` leaf package creates a clean star topology:

```
spec/identifiers/         (leaf, no spec/ imports)
    ^         ^         ^
    |         |         |
spec/features  spec/semantic  impl/interpreter
```

**Simplicity (code-simplicity-reviewer):** The `Identifier` struct's `Description` field is consumed by zero derivation targets. The registry already has its own `Feature.Description`. Every consumer needs only `[]string` of names. Plain slices eliminate ~40 lines of struct definitions and helpers.

**Performance (performance-oracle):** Package-level `var` initializers in Go execute exactly once during package init before `main()`. The `quotedNames()` calls in `FunctionSpecs` are O(k) total where k ≈ 20 — under 1 microsecond at startup. Subsequent reads of `param.Examples` are zero-cost slice access, identical to current behavior.

**Learnings (go-maps-non-deterministic-ordering):** The `docs/solutions/logic-errors/go-maps-non-deterministic-ordering-frontmatter.md` learning confirms that ordered slices are correct here. If anyone converts these to maps, the TUI would show different "first" examples per run. Error message construction must iterate the canonical slice, not the interpreter map.

## Acceptance Criteria

- [x] New package `spec/identifiers/` with canonical `[]string` slices for all 4 identifier sets
- [x] New file `spec/identifiers/identifiers_test.go` with non-empty slice tests
- [x] `spec/semantic/function_types.go` imports `spec/identifiers` and derives `ParamSpec.Examples` via local `quotedNames()` helper
- [x] `spec/features/registry.go` imports `spec/identifiers` and derives `CategoryNetwork`, `CategoryStorage`, `CategoryCompression` Feature entries from identifier slices
- [x] Bidirectional consistency test: every canonical name has a map entry AND every map key is a canonical name (or alias)
- [x] Interpreter error messages derive valid values from `identifiers.*` slices — no more hardcoded strings
- [x] `none` compression is in the canonical list and gets a Feature entry in registry.go
- [x] `memory` is NOT in the canonical list (out of scope)
- [x] All existing golden tests pass with zero changes
- [x] `task test` passes (catwalk failures are pre-existing, unrelated)
- [x] `task quality` passes (modernize issue in growth_functions.go is pre-existing)
- [x] No import cycles introduced

## Technical Considerations

### Import graph after refactoring

```
spec/identifiers/              (NEW leaf package — no imports from spec/)
    ^         ^         ^
    |         |         |
spec/features/registry.go     (NEW import: spec/identifiers)
spec/semantic/function_types.go (NEW import: spec/identifiers)
impl/interpreter/*_functions.go (NEW import: spec/identifiers)
```

**Cycle analysis:** `spec/identifiers` imports nothing from `spec/`. All three consumers add it as a leaf dependency. No cycle is possible. This preserves the existing independence between `spec/semantic` and `spec/features`.

**Contrast with original brainstorm:** The brainstorm proposed `spec/features/identifiers.go`. Research revealed this would force `spec/semantic` → `spec/features`, coupling semantic analysis to a UI catalog package. The dedicated leaf package avoids this.

### Ordering contract

Identifier slices are ordered by "most commonly used first." The first element in each slice is the representative example shown in the TUI compact footer (`view_footer.go:144` displays only `Examples[0]`). This ordering is documented in a code comment on each slice.

### Alias handling

The `ssd`/`sata_ssd` alias is the only alias case across all ~20 identifiers. A separate `StorageAliases map[string]string` handles it without burdening the other 19 identifiers with an empty `Aliases` field:

- `StorageTypes` contains `"ssd"` (what users see in autocomplete/examples)
- `StorageAliases` maps `"sata_ssd" → "ssd"` (accepted at runtime)
- `AllStorageNames()` returns both for interpreter validation and error messages
- Registry derives two Feature entries: one for `ssd` (primary), one for `sata_ssd` (with alias relationship using the existing `features.Alias` struct)
- Interpreter maps include both keys

**Note (pattern-recognition-specialist):** The existing `features.Alias` struct (with `Name`, `Parseable`, `Example` fields) serves *discovery* aliases for search/autocomplete. `StorageAliases` serves *runtime* name resolution. These are distinct concerns — document the distinction clearly.

### Test-time validation instead of init() panics

The original plan proposed `init()` functions that `panic()` on drift. Research identified this as a reliability hazard:

- A missing map entry crashes the entire `cm` binary for all users, even those not using that function
- The project's TDD philosophy (per CLAUDE.md) and `task test` requirement already enforce test-time validation
- The existing `init()` pattern in `functions.go:171` is non-panicking (populates fields, no assertions)

**Replacement: bidirectional consistency tests** in each interpreter test file:

```go
// spec/identifiers/identifiers_test.go or impl/interpreter/*_test.go

func TestNetworkThroughputCoversCanonicalTypes(t *testing.T) {
    // Forward: every canonical identifier has a map entry
    for _, name := range identifiers.NetworkTypes {
        if _, ok := networkThroughput[name]; !ok {
            t.Errorf("networkThroughput missing canonical identifier: %s", name)
        }
    }
    // Reverse: every map key is a canonical identifier
    canonical := slices.Concat(identifiers.NetworkTypes)
    for key := range networkThroughput {
        if !slices.Contains(canonical, key) {
            t.Errorf("networkThroughput has non-canonical key: %s", key)
        }
    }
}
```

**Note:** Since `spec/` cannot import `impl/`, these bidirectional tests must live in the `impl/interpreter/` test files (which can import both `spec/identifiers` and access the package-level maps). This follows the project's existing pattern where cross-layer validation happens in the consuming layer's tests.

### Performance analysis

| Concern | Assessment |
|---|---|
| `quotedNames()` in `var FunctionSpecs` | O(k) at init, k ≈ 20. Executes once. Zero per-access cost. |
| `AllStorageNames()` on error path | O(n) with one allocation. Error paths are cold — `fmt.Errorf` allocates anyway. |
| `strings.Join(identifiers.NetworkTypes, ", ")` in error messages | O(n) on cold path. Acceptable. |
| TUI `param.Examples` access | Unchanged — direct `[]string` slice read, O(1) index. |
| Startup time | ~20 extra map lookups in tests. Zero runtime overhead — no init() functions added. |

## Implementation Phases

### Phase 1: Create canonical source (`spec/identifiers/`)

**Files:**
- CREATE `spec/identifiers/identifiers.go` — 4 `[]string` slices, `StorageAliases` map, `AllStorageNames()` helper
- CREATE `spec/identifiers/identifiers_test.go` — non-empty slices, `AllStorageNames` includes aliases, ordering determinism

**Verification:** `go test ./spec/identifiers/...` passes.

### Phase 2: Derive registry Feature entries

**Files:**
- MODIFY `spec/features/registry.go` — import `spec/identifiers`, `getNetworkFeatures()`, `getStorageFeatures()`, `getCompressionFeatures()` iterate over identifier slices. Feature descriptions remain in `registry.go` (they include syntax, NL examples, and presentation details that belong to the catalog layer, not the canonical name list). Add a `none` compression Feature entry.

**Implementation detail:** The registry keeps its own per-identifier descriptions since they include rich presentation data (syntax patterns, NL examples) that the canonical name list intentionally does not carry. The loop over `identifiers.NetworkTypes` ensures the names stay in sync while the registry owns the presentation.

**Verification:** `go test ./spec/features/...` passes. Registry contains all expected Features including `none` compression.

### Phase 3: Derive ParamSpec.Examples

**Files:**
- MODIFY `spec/semantic/function_types.go` — import `spec/identifiers`, add local `quotedNames()` helper, replace hardcoded `Examples` strings with `quotedNames(identifiers.NetworkTypes)` etc.

**Example transformation:**
```go
// Before:
{Name: "network_type", Type: ArgTypeString, Examples: []string{`"gigabit"`, `"ten_gig"`, `"wifi"`, `"four_g"`, `"five_g"`}}

// After (computed once at package init — not called per-access):
{Name: "network_type", Type: ArgTypeString, Examples: quotedNames(identifiers.NetworkTypes)}
```

**Verification:** `go test ./spec/semantic/...` passes. TUI autocomplete hints now show all valid values including previously missing `hundred_gig`, `nvme`, `pcie_ssd`, `lz4`, `bzip2`, `none`.

### Phase 4: Derive interpreter error messages + add consistency tests

**Files:**
- MODIFY `impl/interpreter/network_functions.go` — import `spec/identifiers`, replace hardcoded error message strings with `strings.Join(identifiers.NetworkTypes, ", ")` etc.
- MODIFY `impl/interpreter/storage_functions.go` — same pattern, use `identifiers.AllStorageNames()` for error messages
- MODIFY `impl/interpreter/compression_functions.go` — same pattern
- MODIFY `impl/interpreter/network_functions_test.go` — add `TestNetworkMapsMatchCanonicalIdentifiers` (bidirectional)
- MODIFY `impl/interpreter/storage_functions_test.go` — add `TestStorageMapsMatchCanonicalIdentifiers` (bidirectional)
- MODIFY `impl/interpreter/compression_functions_test.go` — add `TestCompressionMapsMatchCanonicalIdentifiers` (bidirectional)

**Error message transformation:**
```go
// Before:
return 0, fmt.Errorf("unknown network type '%s' (valid types: gigabit, ten_gig, hundred_gig, wifi, four_g, five_g)", networkType)

// After:
return 0, fmt.Errorf("unknown network type '%s' (valid types: %s)", networkType, strings.Join(identifiers.NetworkTypes, ", "))
```

**Verification:** `go test ./impl/interpreter/...` passes. Bidirectional tests catch any future drift.

### Phase 5: Full validation

**Verification:**
- `task test` — all golden tests pass
- `task quality` — no lint issues
- Manual: `cm functions` output unchanged (it doesn't consume Examples)
- Manual: TUI autocomplete now shows complete identifier lists

## Out of Scope

- `memory` storage type — neither remove from old Examples nor add to interpreter. Handle separately.
- `cm functions` CLI output — does not currently show per-parameter examples. Enabling this is a separate enhancement.
- Adding new identifiers — this refactoring only consolidates existing ones.
- Performance numbers in Feature descriptions drifting from interpreter constants — accepted; descriptions are presentation concerns owned by the registry.
- `transfer_time` golden test coverage gap — `hundred_gig`, `four_g`, `five_g`, `wifi` are untested as `transfer_time` network_type parameters. Worth a follow-up but not blocking for this refactoring.

## References

- Brainstorm: `docs/brainstorms/2026-03-04-derive-function-spec-examples-from-registry-brainstorm.md`
- Related bug: `throughput()` autosuggest showed `"1gbe"`, `"10gbe"` instead of valid identifiers
- Learning: `docs/solutions/code-organization/custom-help-hardcoding-flags.md` — same "hardcoded lists drift" pattern
- Learning: `docs/solutions/code-organization/split-view-go-into-cohesive-modules.md` — shared data structures define ownership
- Learning: `docs/solutions/logic-errors/go-maps-non-deterministic-ordering-frontmatter.md` — confirms ordered slices over maps
- Open todo: `.planning/todos/pending/2026-02-08-function-signature-enum-values.md` — this refactoring resolves it
- Codebase pattern: `spec/units/canonical.go` — existing leaf package with canonical data (same architecture)
- Codebase pattern: `impl/interpreter/functions.go:171` — existing non-panicking init() for reference
