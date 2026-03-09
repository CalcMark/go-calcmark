---
title: "Strict Allowlist Rejected Unknown YAML Frontmatter Keys"
category: logic-errors
tags: [yaml, frontmatter, parsing, validation, allowlist, markdown-compatibility]
module: spec/document
symptom: "Error loading file: parse document: frontmatter: unknown frontmatter key 'title'; user variables must go under 'globals:'"
root_cause: "reservedKeys allowlist validation rejected any top-level YAML key not in the four CalcMark-specific keys"
date_solved: 2026-03-09
severity: medium
related_issue: 44
related_pr: 45
---

# Strict Allowlist Rejected Unknown YAML Frontmatter Keys

## Problem

CalcMark rejected any `.cm` file containing standard YAML frontmatter keys like `title`, `date`, `tags`, or `description`. This made CalcMark incompatible with files created by Hugo, Jekyll, Obsidian, and other Markdown-based tools that use frontmatter as standard practice.

**Error message:**
```
Error loading file: parse document: frontmatter: unknown frontmatter key 'title'; user variables must go under 'globals:'
```

**Reproduction:**
```bash
cm saas-services-pl.cm
# File contained: title: "SaaS Services P&L" alongside CalcMark globals/exchange
```

## Root Cause

`spec/document/frontmatter.go` had a two-phase frontmatter parse:

1. **Phase 1 (generic map):** Unmarshaled YAML into `map[string]any`, then iterated over keys checking against a `reservedKeys` allowlist (`exchange`, `globals`, `scale`, `convert_to`). Any key not in the allowlist produced an error.
2. **Phase 2 (typed struct):** Unmarshaled YAML into `frontmatterYAML` struct with explicit YAML struct tags for the four known fields.

Phase 1 was redundant — `yaml.v3` struct unmarshaling in Phase 2 already silently ignores unknown keys. The generic-map parse existed solely to feed the validation loop, and both could be eliminated.

```go
// DELETED: reservedKeys map (was lines 82-89)
var reservedKeys = map[string]bool{
    "exchange":   true,
    "globals":    true,
    "scale":      true,
    "convert_to": true,
}

// DELETED: generic-map parse + validation loop (was lines 458-469)
var rawMap map[string]any
if err := yaml.Unmarshal([]byte(yamlContent), &rawMap); err != nil {
    return nil, "", formatYAMLError(err)
}
for key := range rawMap {
    if !reservedKeys[key] {
        return nil, "", fmt.Errorf("unknown frontmatter key '%s'; user variables must go under 'globals:'", key)
    }
}
```

## Solution

Removed the entire first-phase generic-map parse and its supporting `reservedKeys` map (~20 lines). The typed `frontmatterYAML` struct handles everything:

- Unknown keys are silently discarded by `yaml.Unmarshal` into the typed struct
- CalcMark-specific keys continue to parse via struct tags
- Invalid CalcMark key *values* still produce errors (e.g., malformed exchange rates)
- `rawSource` preserves ALL original text (including unknown keys) for TUI round-trip fidelity

### Files Changed

| File | Change |
|------|--------|
| `spec/document/frontmatter.go` | Removed `reservedKeys` map and generic-map parse + validation loop |
| `spec/document/frontmatter_test.go` | Replaced error-expecting test with success-expecting table-driven tests |
| `eval_test.go` | Updated integration test to expect success with unknown keys |
| `cmd/calcmark/tui/editor/file_operations_test.go` | Changed parse-error test to use genuinely invalid CalcMark value |
| `cmd/calcmark/tui/editor/frontmatter_test.go` | Added TUI lifecycle test including save-path preservation |
| `testdata/eval/success/features/frontmatter_unknown_keys.cm` | New golden test with mixed frontmatter |

### Performance

Reduced `yaml.Unmarshal` calls from 4 to 3 per frontmatter parse by eliminating the redundant generic-map unmarshal.

### Security

The fix actually *reduces* attack surface — unknown keys are no longer parsed into `map[string]any` at all. The only parse is directly into the typed struct, which has explicit tags for four known fields. Existing NaN/Inf guards on exchange rate floats remain unaffected.

## Key Insight

The `rawSource` mechanism is the safety net that makes this change safe across all code paths:

- **Save path:** `getDocumentContent()` → `fm.Serialize()` → returns `rawSource` (preserves unknown keys)
- **Redetect path:** `redetectBlockTypes()` → `getDocumentContent()` → `NewDocument(content)` → new `rawSource` from literal text
- **Formatters:** CalcMark/Markdown formatters use `Serialize()` (preserves unknown keys). JSON/HTML formatters reconstruct from struct fields (unknown keys excluded by design — these are structured output formats).

`SetGlobal()` and `SetExchangeRate()` clear `rawSource`, but neither has production callers.

## Prevention

1. **Prefer typed struct parsing over generic maps** — Go's `yaml.v3` struct tags provide natural allowlisting without explicit validation code.
2. **Test round-trip fidelity** — when accepting new input variants, verify they survive the complete lifecycle (parse → edit → redetect → save).
3. **Audit all consumers** — trace every code path that touches the modified struct to confirm no destructive behavior.

## Related

- [Exchange Rate Frontmatter Validation](./exchange-rate-frontmatter-validation.md)
- [Go Maps Non-Deterministic Ordering Frontmatter](./go-maps-non-deterministic-ordering-frontmatter.md)
- [NaN/Inf Panic YAML Frontmatter Scale](../security-issues/nan-inf-panic-yaml-frontmatter-scale.md)
- [Frontmatter Editing Keyboard Dispatch Fixes](../ui-bugs/frontmatter-editing-keyboard-dispatch-fixes.md)
- GitHub Issue: [#44](https://github.com/CalcMark/go-calcmark/issues/44)
- GitHub PR: [#45](https://github.com/CalcMark/go-calcmark/pull/45)
