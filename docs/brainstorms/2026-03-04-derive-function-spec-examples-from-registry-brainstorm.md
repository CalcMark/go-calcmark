---
title: "Derive FunctionSpec examples from feature registry identifiers"
date: 2026-03-04
type: refactor
status: active
---

# Derive FunctionSpec Examples from Feature Registry Identifiers

## What We're Building

A single source of truth for function parameter identifiers (network scopes, network types, storage types, compression types) in a new `spec/features/identifiers.go` file. Currently these valid values are duplicated in three places:

1. `spec/semantic/function_types.go` — hardcoded `ParamSpec.Examples`
2. `spec/features/registry.go` — handcrafted `Feature` entries per identifier
3. `impl/interpreter/*_functions.go` — runtime lookup maps with numeric values

The identifier structs in `spec/features/identifiers.go` become the canonical source. All three consumers derive from it.

## Why This Approach

The bug that motivated this: `throughput()` autosuggest showed `"1gbe"`, `"10gbe"` instead of the actual valid identifiers `gigabit`, `ten_gig`, etc. Three copies drifted. A single source prevents this class of bug entirely.

**Chosen architecture:**

- **Typed identifier structs** (Name + Description) in `spec/features/identifiers.go`
- **Feature registry** derives its `CategoryNetwork`, `CategoryStorage`, `CategoryCompression` entries from these structs
- **FunctionSpecs** derives `ParamSpec.Examples` from identifier names
- **Interpreter** imports identifier names as map keys (dependency direction: impl → spec, which is correct)

## Key Decisions

1. **Location**: New `spec/features/identifiers.go` — separate from registry to keep concerns clear
2. **Data shape**: Typed structs with `Name` and `Description` fields, grouped into ordered slices per identifier set
3. **Derivation direction**: identifiers.go → registry.go (Feature entries), identifiers.go → function_types.go (Examples), identifiers.go → interpreter maps (keys)
4. **Interpreter coupling**: Interpreter imports `spec/features` for identifier names and iterates over identifier slices to build its `map[string]float64` at init. Since keys are derived, no sync test is needed — the compiler enforces it
5. **Ordered slices, not maps**: Use `[]Identifier` to preserve display order (maps are unordered in Go)

## Identifier Sets

| Set | Identifiers | Used by |
|-----|------------|---------|
| NetworkScopes | local, regional, continental, global | rtt(), transfer_time() |
| NetworkTypes | gigabit, ten_gig, hundred_gig, wifi, four_g, five_g | throughput(), transfer_time() |
| StorageTypes | ssd, sata_ssd, nvme, pcie_ssd, hdd | read(), seek() |
| CompressionTypes | gzip, lz4, zstd, bzip2, snappy, none | compress() |

## Scope

- Eliminates duplication across spec and impl layers
- Does NOT change any user-facing behavior
- Does NOT change the CalcMark language grammar or semantics
- Does NOT add new identifiers (just consolidates existing ones)

## Assumptions

- `spec/semantic` can import `spec/features` (intra-spec dependency, same direction as `spec/features` importing `spec/units`)
- The `Identifier` struct carries only Name + Description; derivation functions in registry.go add Syntax, Example, and other Feature fields
