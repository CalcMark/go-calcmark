---
title: "feat: NL growth functions accept variable arguments"
type: feat
status: completed
date: 2026-03-18
origin: docs/brainstorms/2026-03-18-nl-variable-args-requirements.md
---

# feat: NL growth functions accept variable arguments

## Overview

NL growth functions (`compound`, `grow`, `depreciate`) reject variable references as arguments — the document detector misclassifies lines like `compound price by 5% over 10 years` as markdown prose. The parser and classifier already handle these correctly; only the detector's `looksLikeCalculation()` heuristic needs updating. Additionally, the detector's NL keyword list is hard-coded and out of sync with the classifier's, creating drift risk for future NL functions.

## Problem Statement / Motivation

Users writing CalcMark documents expect to define a variable and use it in NL syntax:

```
price = $500
compound price by 5% over 10 years
```

Today, line 2 becomes markdown text. Users must fall back to functional syntax: `compound(price, 5%, 10 years)`. System NL functions (`read data from ssd`) already work with variables — only growth functions are broken.

The root cause is in `looksLikeCalculation()` (detector.go:339-349). When a line starts with IDENTIFIER + IDENTIFIER, it checks for `read`/`compress`/`transfer` patterns but not `compound`/`grow`/`depreciate`. The growth functions fall through to `return false`.

Five places currently must be updated for each NL function: parser gates, NL parse functions, detector heuristics, classifier, and LSP completions. The feature registry is the natural single source of truth but doesn't yet drive classification behavior.

(see origin: docs/brainstorms/2026-03-18-nl-variable-args-requirements.md)

## Proposed Solution

Three-phase approach following TDD:

1. **Parity test** (R3) — Write failing tests proving the detector/classifier disagree on growth NL patterns with variables
2. **Detector fix** (R1) — Add growth function patterns to `looksLikeCalculation()`, making tests pass
3. **Registry-driven refactor** (R2) — Extract NL trigger keywords from the feature registry so both detector and classifier derive from one source

## Technical Considerations

### Token analysis (verified)

`compound price by 5% over 10 years` tokenizes as:
```
[0] IDENTIFIER    "compound"
[1] IDENTIFIER    "price"
[2] IDENTIFIER    "by"       ← soft keyword, still plain IDENTIFIER
[3] NUMBER_PERCENT "5%"
[4] OVER          "over"     ← reserved keyword
[5] DURATION_LITERAL "10:year"
```

The detector pattern to match: `IDENTIFIER(trigger) + IDENTIFIER(any) + IDENTIFIER("by")`.

### Why false positives are not a concern

`IsCalculation()` first runs `parser.Parse()` (detector.go:221). Only lines that parse successfully reach `looksLikeCalculation()`. English prose like `Grow your business by focusing...` fails parsing because the NL growth parser expects a value after "by", not another prose word. The parser is the real gatekeeper; `looksLikeCalculation()` is a secondary heuristic for lines the parser accepted.

### Undefined variable behavior

When a variable used in NL syntax is not defined, the line still classifies as a calculation and produces an "undefined variable" runtime error. This matches existing behavior for system NL functions (`read data from ssd` when `data` is undefined). The detector is stateless and cannot know variable scope.

(see origin: docs/brainstorms/2026-03-18-nl-variable-args-requirements.md — Scope Boundaries)

### Registry approach: follow existing patterns

The user's architecture has a clean separation: the registry provides metadata, the LSP reads from it, and the detector/classifier have their own logic. R2 adds a `NLTriggerKeywords()` method to the registry that extracts trigger keywords from parseable `Alias` entries (the first word before `"..."`). Both detector and classifier call this instead of hard-coding keyword lists. No new struct needed — this follows the existing `ByCategory()` / `Search()` pattern on the `Registry` type.

## System-Wide Impact

- **Detector** (spec/document/detector.go): `looksLikeCalculation()` gains growth function patterns, then refactors to registry-driven
- **Classifier** (spec/classifier/classifier.go): `containsFunctions()` refactors to use registry instead of hard-coded switch. No behavioral change.
- **Registry** (spec/features/registry.go): New `NLTriggerKeywords()` method
- **LSP**: No changes — already reads from registry via `ByCategory()`
- **Parser**: No changes — already handles variables via `parseExponent()`

## Acceptance Criteria

### Phase 1: Parity test (R3)

- [ ] New test file `spec/document/detector_classifier_parity_test.go` (or extend existing test files)
- [ ] Test exercises all 6 NL functions with **literal** arguments through both `detector.IsCalculation()` and `classifier.ClassifyLine()` — assert both agree
- [ ] Test exercises all 6 NL functions with **variable** arguments through both paths — assert both agree
- [ ] Test for growth functions with variable args **fails** initially (proving the bug)
- [ ] Variable argument test lines use format: `compound price by 5% over 10 years`, `grow servers by 20 over 5 months`, `depreciate car_value by 15% over 5 years`
- [ ] Classifier tests provide an `IdentifierResolver` with the variable defined

### Phase 2: Detector fix (R1)

- [ ] `looksLikeCalculation()` in detector.go recognizes IDENTIFIER + IDENTIFIER + IDENTIFIER("by") when first token is `compound`, `grow`, or `depreciate`
- [ ] All parity tests from Phase 1 pass
- [ ] Golden test file `testdata/spec/valid/features/growth_functions.cm` updated with variable-argument examples
- [ ] Golden test file `testdata/eval/success/features/growth_functions.cm` updated with variable-argument evaluation examples
- [ ] `task test` passes — full suite, no regressions

### Phase 3: Registry-driven refactor (R2)

- [ ] `Registry.NLTriggerKeywords() []string` method added to `spec/features/registry.go`
- [ ] Method extracts trigger keywords from parseable aliases (first word of `Alias.Name` before `"..."`)
- [ ] `looksLikeCalculation()` in detector.go calls `NLTriggerKeywords()` instead of hard-coding `"compress"`, `"read"`, `"transfer"`, `"compound"`, `"grow"`, `"depreciate"`
- [ ] `containsFunctions()` in classifier.go calls `NLTriggerKeywords()` instead of hard-coding `"compound"`, `"grow"`, `"depreciate"` in the switch
- [ ] Test in `spec/features/registry_test.go` verifies `NLTriggerKeywords()` returns all 6 triggers
- [ ] Parity test continues to pass
- [ ] `task test` and `task quality` pass

## Success Metrics

- All 6 NL functions work with variable arguments in the document pipeline
- Adding a future NL function requires only: registry entry + parser — detector and classifier derive automatically
- Parity test catches any future drift between detector and classifier

## Dependencies & Risks

- **Low risk**: The parser already handles all variable cases. This is purely a classification fix.
- **Institutional learning**: "Every fix is two fixes" (from `docs/solutions/integration-issues/nl-functional-syntax-parity-and-doc-staleness.md`). This plan addresses both the detector fix AND the systemic drift prevention.
- **`frequencyAdverbs` duplication** (same learning doc): The parser and interpreter both define frequency adverbs separately. Not in scope but worth noting for future cleanup.

## Sources & References

### Origin

- **Origin document:** [docs/brainstorms/2026-03-18-nl-variable-args-requirements.md](docs/brainstorms/2026-03-18-nl-variable-args-requirements.md) — Key decisions: registry-driven approach over ad-hoc patches; no LSP changes; classifier already works correctly.

### Internal References

- Detector heuristic: `spec/document/detector.go:261` (`looksLikeCalculation()`)
- Detector NL gap: `spec/document/detector.go:339-349` (missing growth patterns)
- Classifier growth check: `spec/classifier/classifier.go:91-96` (`containsFunctions()`)
- Registry aliases: `spec/features/registry.go:661-686` (growth function alias definitions)
- Parser NL gates: `spec/parser/rdparser.go:1281-1291` (growth function lookahead)
- NL parse functions: `spec/parser/nl_growth_functions.go:12-188`
- Existing detector NL tests: `spec/document/detector_test.go:579-622` (`TestNLFunctionVariableDetection`)
- Existing classifier tests: `spec/classifier/classifier_function_fixed_test.go:144-165` (`TestGrowthFunctionClassification`)

### Institutional Learnings

- NL/functional parity: `docs/solutions/integration-issues/nl-functional-syntax-parity-and-doc-staleness.md`
- NL function AST range: `docs/solutions/logic-errors/nl-function-missing-ast-range.md`
- Cross-layer checklist: `docs/solutions/logic-errors/adding-new-type-fraction-cross-layer-checklist.md`
- Unified registry: `docs/solutions/code-organization/unified-feature-registry-three-to-one.md`
