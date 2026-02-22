---
title: Frontmatter Variable Ordering Non-Determinism
category: logic-errors
tags: [go-maps, non-determinism, frontmatter, flaky-tests, yaml-v3, ordering, catwalk]
module: spec/document, eval, format/*, cmd/calcmark/tui/editor
symptom: Catwalk tests for frontmatter editing fail intermittently; variables processed in random order across runs
root_cause: Go maps have non-deterministic iteration order; Frontmatter struct stored variables in maps without preserving insertion order
date_resolved: 2026-02-22
severity: critical
---

# Frontmatter Variable Ordering Non-Determinism

## Problem

CalcMark frontmatter variables (globals and exchange rates) were stored in Go maps (`map[string]string` and `map[string]decimal.Decimal`). Go maps have intentionally randomized iteration order. Every site that iterated frontmatter variables -- the evaluator, all three formatters (text, HTML, JSON), and the TUI globals panel -- produced different orderings per run.

**Symptoms:**
- Catwalk tests for frontmatter editing passed sometimes, failed sometimes
- Non-reproducible output formatting across runs
- Variables potentially evaluated in wrong order (violating frontmatter semantics)

**User impact:** "The correct behavior is for each variable defined in front matter to be processed *in order* and *before* the rest of the document. That's why it's *front* matter."

## Investigation

1. Identified flaky `TestEditorCatwalkFrontmatterEditing` test
2. Running the test 5 times showed intermittent pass/fail
3. Traced to the frontmatter observer iterating `range fm.Globals` (Go map)
4. Launched codebase-wide audit: found **18 iteration sites** across 15 files that used `range fm.Globals` or `range fm.Exchange`

## Root Cause

The `Frontmatter` struct relied entirely on Go maps:

```go
type Frontmatter struct {
    Exchange map[string]decimal.Decimal
    Globals  map[string]string
}
```

Go maps randomize iteration order for security. Without explicit key ordering, every consumer produced non-deterministic results.

## Solution

### 1. Added Ordered Key Slices to Data Model

```go
type Frontmatter struct {
    Exchange     map[string]decimal.Decimal
    Globals      map[string]string
    exchangeKeys []string  // preserves insertion order
    globalKeys   []string  // preserves insertion order
    rawSource    string
}
```

### 2. Accessor Methods with Sorted Fallback

For backward compatibility with struct literals (which bypass the parser):

```go
func (f *Frontmatter) GlobalKeys() []string {
    if f == nil { return nil }
    if len(f.globalKeys) > 0 { return f.globalKeys }
    // Fallback: sorted order for determinism
    keys := make([]string, 0, len(f.Globals))
    for k := range f.Globals { keys = append(keys, k) }
    sort.Strings(keys)
    return keys
}
```

### 3. YAML Document Order via yaml.v3 Node API

Standard `yaml.Unmarshal` into a map loses document order. The yaml.v3 `Node` API preserves it:

```go
func extractYAMLKeyOrder(yamlContent string, topKey string) []string {
    var root yaml.Node
    if err := yaml.Unmarshal([]byte(yamlContent), &root); err != nil { return nil }
    if root.Kind != yaml.DocumentNode || len(root.Content) == 0 { return nil }
    mapping := root.Content[0]
    if mapping.Kind != yaml.MappingNode { return nil }
    for i := 0; i < len(mapping.Content)-1; i += 2 {
        keyNode := mapping.Content[i]
        valueNode := mapping.Content[i+1]
        if keyNode.Value == topKey && valueNode.Kind == yaml.MappingNode {
            var keys []string
            for j := 0; j < len(valueNode.Content)-1; j += 2 {
                keys = append(keys, valueNode.Content[j].Value)
            }
            return keys
        }
    }
    return nil
}
```

### 4. Updated All 18 Iteration Sites

Every consumer changed from map iteration to ordered method calls:

```go
// Before (non-deterministic)
for key := range fm.Exchange { ... }
for name := range fm.Globals { ... }

// After (deterministic, document order)
for _, key := range fm.ExchangeKeys() { ... }
for _, name := range fm.GlobalKeys() { ... }
```

**Files updated:** `eval.go`, `format/text_formatter.go`, `format/html_formatter.go`, `format/json_formatter.go`, `cmd/calcmark/tui/editor/view_state.go`, `cmd/calcmark/tui/editor/frontmatter_test.go`

### 5. Lifecycle Methods Maintain Order

- `SetGlobal()`: Appends new keys to `globalKeys`, invalidates `rawSource`
- `SetExchangeRate()`: Appends new keys to `exchangeKeys`, invalidates `rawSource`
- `ParseFrontmatter()`: Populates ordered keys via `extractYAMLKeyOrder()`
- `Serialize()`: Uses `GlobalKeys()` and `ExchangeKeys()` for output

## Verification

100-iteration determinism test (`TestFrontmatter_OrderPreservation`):

```go
// YAML with intentionally non-alphabetical order
source := `---
globals:
  zebra: 1
  alpha: 2
  middle: 3
---`

for i := range 100 {
    fm, _, _ := ParseFrontmatter(source)
    keys := fm.GlobalKeys()
    // Must always be ["zebra", "alpha", "middle"], never sorted
    assert.Equal(t, []string{"zebra", "alpha", "middle"}, keys)
}
```

Previously flaky `TestEditorCatwalkFrontmatterEditing` now passes deterministically across 5+ consecutive runs.

## Prevention Strategies

### 1. Never Iterate Go Maps When Order Matters

Always maintain a parallel ordered slice. The map provides O(1) lookup; the slice preserves insertion order.

### 2. Use yaml.v3 Node API for Document Order

Parse twice when order matters: `yaml.Node` for order, `yaml.Unmarshal` for typed data.

### 3. Sorted Fallback for Backward Compatibility

When adding order to existing structs, provide a deterministic fallback (sorted keys) for code that constructs via struct literals.

### 4. 100+ Iteration Determinism Tests

Non-deterministic bugs are intermittent. Expose them reliably:

```go
for i := range 100 {
    result := functionUnderTest()
    if result != expected {
        t.Fatalf("Non-deterministic on iteration %d", i)
    }
}
```

### 5. Codebase-Wide Audit on Map Ordering Fixes

When fixing a map ordering bug, `grep` for ALL iteration sites. This fix required updating 18 sites across 15 files.

## Related Files

- `spec/document/frontmatter.go` -- Core data model with ordered keys
- `spec/document/frontmatter_test.go` -- Order preservation tests
- `spec/document/globals.go` -- `ParseGlobalsOrdered()` for ordered evaluation
- `cmd/calcmark/tui/editor/TESTING.md` -- Catwalk test framework docs
- `docs/solutions/code-organization/split-view-go-into-cohesive-modules.md` -- Related TUI architecture

## References

- PR: https://github.com/CalcMark/go-calcmark/pull/4
- Commit: `90f664d` (feat(tui,spec): align Globals panel with frontmatter and fix ordering bug)
- Go spec on map iteration: "The iteration order over maps is not specified and is not guaranteed to be the same from one iteration to the next"
- yaml.v3 Node API: https://pkg.go.dev/gopkg.in/yaml.v3#Node
