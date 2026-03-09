---
title: "Accept Arbitrary YAML Frontmatter Keys"
type: feat
status: completed
date: 2026-03-09
---

# feat: Accept Arbitrary YAML Frontmatter Keys

## Enhancement Summary

**Deepened on:** 2026-03-09
**Sections enhanced:** 3 (Proposed Solution, Technical Considerations, Acceptance Criteria)
**Agents used:** security-sentinel, code-simplicity-reviewer, architecture-strategist, pattern-recognition-specialist, explore (rawMap usage)

### Key Improvements
1. Expanded scope: remove entire first-phase generic-map parse (rawMap + reservedKeys), not just the validation loop
2. Added fuzz test seed update for unknown-key frontmatter
3. Clarified security posture: unknown keys never reach typed struct, no attack surface change

## Overview

CalcMark currently rejects any top-level YAML frontmatter key that is not one of the four reserved CalcMark keys (`exchange`, `globals`, `scale`, `convert_to`). This makes it impossible to open standard Markdown files that use common frontmatter keys like `title`, `date`, `tags`, `description`, etc. — a common pattern in Hugo, Jekyll, and other static site generators.

Since YAML frontmatter is standard Markdown practice, CalcMark should silently ignore unknown keys while continuing to process CalcMark-specific ones.

## Problem Statement

**Reproduction:**
```bash
cm saas-services-pl.cm
# Error loading file: parse document: frontmatter: unknown frontmatter key 'title'; user variables must go under 'globals:'
```

Any `.cm` file with standard YAML frontmatter keys produces this error. The validation at `spec/document/frontmatter.go:464-468` explicitly rejects unknown keys:

```go
for key := range rawMap {
    if !reservedKeys[key] {
        return nil, "", fmt.Errorf("unknown frontmatter key '%s'; user variables must go under 'globals:'", key)
    }
}
```

## Proposed Solution

**Remove the entire first-phase generic-map parse and its supporting code.** The typed `frontmatterYAML` struct (line 277) already only unmarshals the four known keys via YAML struct tags — unknown keys are naturally ignored by `yaml.Unmarshal`. The generic-map parse exists solely to feed the validation loop, and both can be eliminated together.

### Implementation

1. **Delete lines 458-469** in `spec/document/frontmatter.go` — the generic-map parse (`rawMap`) and the unknown key validation loop. This is ~12 lines total:
   ```go
   // DELETE: lines 458-469
   // First, parse into a generic map to check for unknown keys
   var rawMap map[string]any
   if err := yaml.Unmarshal([]byte(yamlContent), &rawMap); err != nil {
       return nil, "", formatYAMLError(err)
   }

   // Validate that all top-level keys are reserved
   for key := range rawMap {
       if !reservedKeys[key] {
           return nil, "", fmt.Errorf("unknown frontmatter key '%s'; user variables must go under 'globals:'", key)
       }
   }
   ```
2. **Delete lines 82-89** — the `reservedKeys` map and its comment. It has no other references in the codebase.
3. **Update `TestParseFrontmatter_UnknownKey`** (frontmatter_test.go:423) — flip from expecting error to expecting success; verify unknown keys are silently ignored and CalcMark keys still parse correctly
4. **Update integration test** (eval_test.go:620) — same flip: unknown keys should not produce errors
5. **Add new test cases** for mixed frontmatter (standard markdown keys + CalcMark keys coexisting)
6. **Add golden test file** — `testdata/eval/success/features/frontmatter_unknown_keys.cm` with `title`, `date`, `tags` alongside CalcMark `globals`
7. **Update fuzz seed** — in `spec/document/fuzz_test.go:36`, the seed `"---\nunknown_key: value\n---\n"` currently expects an error; after this change it should succeed, validating no-panic behavior

### Files to Modify

| File | Change |
|------|--------|
| `spec/document/frontmatter.go:458-469` | Remove generic-map parse and validation loop |
| `spec/document/frontmatter.go:82-89` | Remove `reservedKeys` map and comment |
| `spec/document/frontmatter_test.go:423` | Update `TestParseFrontmatter_UnknownKey` to expect success |
| `eval_test.go:620` | Update integration test to expect success with unknown keys |
| `testdata/eval/success/features/frontmatter_unknown_keys.cm` | New golden test with mixed frontmatter |
| `spec/document/fuzz_test.go:36` | Fuzz seed now expects success (no behavior change needed, just documenting) |

## Technical Considerations

- **rawSource preservation**: The `rawSource` field already stores the original YAML text. Unknown keys are preserved in `rawSource` for TUI round-trip fidelity even though they aren't in the `Frontmatter` struct. No changes needed.
- **Serialization**: `Frontmatter.Serialize()` reconstructs YAML from parsed maps. Unknown keys won't appear in serialized output, but `rawSource` is used for editing. This is acceptable — CalcMark doesn't claim to preserve non-CalcMark frontmatter across edits.
- **Security**: With this change, unknown keys are never even parsed into a generic map — the only parse is directly into the typed `frontmatterYAML` struct, which has explicit YAML tags for the four known fields. Unknown keys are discarded by `yaml.Unmarshal` before any CalcMark code touches them. This is actually a *smaller* attack surface than before (where unknown keys were parsed into `map[string]any`). The existing NaN/Inf guards on exchange rate float values (per `docs/solutions/security-issues/nan-inf-panic-yaml-frontmatter-scale.md`) remain unaffected.
- **Typo detection**: Removing strict validation means a typo like `global:` (instead of `globals:`) will be silently ignored rather than flagged. This is an acceptable trade-off — CalcMark keys are few and well-documented. A future enhancement could add "did you mean?" suggestions for near-matches of reserved keys.
- **Forward compatibility**: The original comment said keys were rejected "to ensure forward compatibility." In practice, this prevented CalcMark from being used with standard Markdown tooling. Future CalcMark keys added to `frontmatterYAML` will automatically be picked up by the typed struct — no allowlist needed.
- **YAML error detection**: The generic-map parse also served as a first-pass YAML syntax check. After removal, YAML syntax errors will still be caught by the typed-struct parse on the very next line. No gap in error detection.

## Acceptance Criteria

- [ ] `cm` opens files with arbitrary YAML frontmatter keys without error
- [ ] CalcMark-specific keys (`exchange`, `globals`, `scale`, `convert_to`) still work correctly
- [ ] Mixed frontmatter (standard + CalcMark keys) works correctly
- [ ] Invalid CalcMark key *values* (e.g., bad exchange rate format) still produce errors
- [ ] TUI editor handles files with unknown frontmatter keys
- [ ] All existing tests pass (with updated expectations)
- [ ] New tests cover mixed frontmatter scenarios
- [ ] `task quality` passes

## References

- Bug location: `spec/document/frontmatter.go:464-468`
- Generic-map parse: `spec/document/frontmatter.go:458-462`
- reservedKeys map: `spec/document/frontmatter.go:82-89`
- Frontmatter struct: `spec/document/frontmatter.go:50-80`
- Typed YAML struct: `spec/document/frontmatter.go:277-282`
- Existing unknown key test: `spec/document/frontmatter_test.go:423`
- Integration test: `eval_test.go:620`
- Fuzz test seed: `spec/document/fuzz_test.go:36`
- Related plan: `docs/plans/2026-02-27-fix-status-bar-truncation-and-file-open-validation-plan.md:37`
- Learnings: `docs/solutions/security-issues/nan-inf-panic-yaml-frontmatter-scale.md`
- Learnings: `docs/solutions/logic-errors/go-maps-non-deterministic-ordering-frontmatter.md`
- Learnings: `docs/solutions/ui-bugs/frontmatter-editing-keyboard-dispatch-fixes.md`
