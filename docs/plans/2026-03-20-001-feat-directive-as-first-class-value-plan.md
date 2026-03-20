---
title: "feat: Directive References as First-Class Values in All Expression Positions"
type: feat
status: active
date: 2026-03-20
origin: docs/brainstorms/2026-03-20-directive-as-value-requirements.md
---

# feat: Directive References as First-Class Values in All Expression Positions

## Overview

`@scale` and `@globals.x` directive references already work in arithmetic expressions via the parser and interpreter, but two layers have gaps: the **session-level classifier** doesn't recognize `@` tokens (causing directive-only lines to be silently treated as markdown in the TUI/REPL), and the **TUI autocomplete** has no directive awareness. This plan closes those gaps so directives behave as numbers in every context.

## Problem Statement / Motivation

Users writing `@scale meters / second` or standalone `@scale` in the TUI see no result — the line is classified as markdown and silently ignored. This is confusing because `a = @scale * 3` works fine (the operator triggers a different classifier path). The inconsistency undermines trust in the directive system. (see origin: docs/brainstorms/2026-03-20-directive-as-value-requirements.md)

## Proposed Solution

Three targeted fixes across two subsystems, plus a new autocomplete source:

1. **Classifier AT_SIGN awareness** — Add directive detection to `spec/classifier/classifier.go` so lines containing `@` are classified as calculations when they parse successfully.
2. **`allIdentifiersDefined` DirectiveRef case** — Add explicit `*ast.DirectiveRef → true` case instead of relying on the fragile `default` arm.
3. **TUI DirectiveSuggestionSource** — New suggestion source offering `@scale` and `@globals.x` completions, with `getCurrentWordPrefix` updated to include a leading `@` in the prefix.

## Technical Considerations

### What Already Works (No Changes Needed)

| Layer | Status | Evidence |
|-------|--------|----------|
| Lexer | Complete | AT_SIGN token, DOT-in-context for globals |
| Parser (`parsePrimary`) | Complete | Handles AT_SIGN → DirectiveRef at `rdparser.go:1045` |
| Semantic checker | Complete | Validates @scale/@globals at `checker.go:473` |
| Interpreter (`evalDirectiveRef`) | Complete | Resolves to runtime values at `interpreter.go:139` |
| Document-level detector | Complete | Recognizes AT_SIGN at `detector.go:322` |
| Scale exemption | Complete | `ast.ContainsScaleRef()` recursive walker at `nodes.go:416` |
| NL function arguments | Complete | All NL functions call `parseExponent()` → `parsePrimary()` |

### What Needs Fixing

| Gap | File | Impact |
|-----|------|--------|
| Classifier has no AT_SIGN check | `spec/classifier/classifier.go` | Standalone directives & directive+unit lines silently become markdown in TUI/REPL |
| `allIdentifiersDefined` missing DirectiveRef | `spec/classifier/classifier.go:152` | Latent correctness risk — works by accident via `default → true` |
| `isWordRune` excludes `@` | `cmd/calcmark/tui/editor/autocomplete_handler.go:245` | Prefix extraction breaks on `@`, so `@sc` → `sc` |
| No DirectiveSuggestionSource | `cmd/calcmark/tui/editor/autocomplete.go` | No directive completions offered |

### Architecture & Performance

- All changes are additive — no existing behavior changes.
- Classifier change is O(n) token scan for AT_SIGN, same as existing operator/keyword scans.
- Autocomplete source is lazy (callback pattern), matching existing `VariableSuggestionSource`.
- spec/impl boundary preserved: classifier is in `spec/`, autocomplete is in `cmd/`.

### Security Considerations

- Per learning from `docs/solutions/security-issues/nan-inf-panic-yaml-frontmatter-scale.md`: no new `decimal.NewFromFloat()` paths are introduced. Existing NaN/Inf guards remain sufficient.
- No new frontmatter parsing paths — autocomplete reads from already-validated `Frontmatter.Globals`.

## System-Wide Impact

- **Interaction graph**: Classifier change affects TUI line evaluation and REPL. No callbacks or middleware involved — pure function.
- **Error propagation**: When `@scale` is used without frontmatter, classifier now classifies as Calculation → interpreter returns error → TUI shows error result. This is an improvement over the current silent-markdown behavior.
- **State lifecycle risks**: None. Autocomplete source uses lazy callbacks (per learning from `docs/solutions/logic-errors/go-closure-capturing-stale-value-type.md`), so frontmatter edits are reflected immediately.
- **API surface parity**: CLI (`--format json`) already works. This brings TUI/REPL to parity.

## Acceptance Criteria

### Phase 1: Classifier Fix

- [ ] `@scale` as standalone line is classified as `Calculation` (not `Markdown`) in `spec/classifier/classifier.go`
- [ ] `@globals.tax_rate` as standalone line is classified as `Calculation`
- [ ] `@scale meters / second` is classified as `Calculation`
- [ ] `x = 1 + @scale` continues to classify as `Calculation`
- [ ] `@scale` without frontmatter is classified as `Calculation` (runtime error shown, not silent)
- [ ] `allIdentifiersDefined` has explicit `*ast.DirectiveRef` case returning `true`
- [ ] Golden tests added to `spec/classifier/classifier_test.go`
- [ ] `task test` passes

### Phase 2: TUI Autocomplete

- [ ] Typing `@` in expression position shows `@scale` suggestion (when frontmatter has `scale:`)
- [ ] Typing `@g` shows `@globals` suggestion (when frontmatter has `globals:`)
- [ ] Typing `@globals.` shows available global field names from frontmatter
- [ ] `getCurrentWordPrefix` includes leading `@` in prefix (without modifying `isWordRune`)
- [ ] `DirectiveSuggestionSource` uses callback pattern for lazy frontmatter access
- [ ] Category label is `"directive"` with appropriate sort priority in `catOrder`
- [ ] `@scale` NOT offered when no `scale:` in frontmatter; `@globals` NOT offered when no `globals:` defined
- [ ] Catwalk tests cover autocomplete trigger, selection, and insertion
- [ ] `task test` passes

### Phase 3: Integration & NL Function Verification

- [ ] `echo '---\nscale:\n  factor: 3\n  unit_categories: [All]\n---\nb = @scale meters / second' | cm --format json` → calculation result `3 m/s`
- [ ] `a = @scale * 3` with factor 3 → `9` (no double-scaling, verified via existing exemption)
- [ ] Three-tier NL tests per `docs/solutions/integration-issues/nl-functional-syntax-parity-and-doc-staleness.md`:
  - Tier 1: Absolute value tests for `@scale` in NL function arguments
  - Tier 2: Intra-syntax equivalence (NL result matches programmatic result)
  - Tier 3: Cross-syntax parity (directive arg produces same result as literal arg)
- [ ] `task quality` passes

## Implementation Phases

### Phase 1: Classifier Fix (`spec/classifier/classifier.go`)

**Goal**: Lines containing directive references are classified as calculations.

**Tasks**:

1. **Add AT_SIGN detection** — After the existing special keywords check (step 5b, ~line 291), add a new check: scan `contentTokens` for `AT_SIGN`. If found, attempt to parse the line. If parsing succeeds and the AST contains a `DirectiveRef`, classify as `Calculation`.

   ```
   // spec/classifier/classifier.go — new step after special keywords
   // Step 5c: Check for directive references (@scale, @globals.x)
   for _, tok := range contentTokens {
       if tok.Type == lexer.AT_SIGN {
           // Parse and verify — directives are always calculations
           ...
       }
   }
   ```

2. **Add `*ast.DirectiveRef` to `allIdentifiersDefined`** — Explicit case returning `true`. Directives are always "defined" from the classifier's perspective; semantic validation is the interpreter's job.

3. **Write classifier tests** — Test cases:
   - `@scale` standalone → Calculation
   - `@globals.tax_rate` standalone → Calculation
   - `@scale meters / second` → Calculation
   - `x = 1 + @scale` → Calculation (already works, regression test)
   - `@scale` without frontmatter → Calculation (error shown at runtime)
   - Ensure existing markdown lines unchanged

**Files**:
- `spec/classifier/classifier.go` (modify)
- `spec/classifier/classifier_test.go` (add tests)

### Phase 2: TUI Autocomplete

**Goal**: `@scale` and `@globals.x` appear in autocomplete suggestions.

**Tasks**:

1. **Update `getCurrentWordPrefix`** in `autocomplete_handler.go` — After the backward scan completes, check if the character immediately before the prefix start is `@`. If so, extend the prefix to include `@`. Do NOT modify `isWordRune` (per SpecFlow analysis: `email@example` would break).

2. **Create `DirectiveSuggestionSource`** in `autocomplete.go`:
   - Constructor takes a callback `func() *frontmatter.Frontmatter` for lazy access
   - `GetSuggestions(prefix)`:
     - If prefix starts with `@`:
       - If frontmatter has `scale:`, offer `@scale`
       - If frontmatter has `globals:` and prefix is `@globals.`, offer field completions
       - If frontmatter has `globals:` and prefix is `@g...`, offer `@globals`
     - If prefix doesn't start with `@`, return empty
   - Category: `"directive"`
   - Use pointer indirection per learning from `docs/solutions/logic-errors/go-closure-capturing-stale-value-type.md`

3. **Register in `CombinedSuggestionSource`** — Add `DirectiveSuggestionSource` to the sources list and add `"directive"` to `catOrder`.

4. **Handle `@globals.` two-stage completion** — When `@globals` is accepted, insert `@globals.` (with trailing dot). Then `getCurrentWordPrefix` returns `@globals.` prefix, and `DirectiveSuggestionSource` matches field names against the portion after the dot.

5. **Catwalk tests** — Write data-driven tests for:
   - Typing `@` triggers directive suggestions
   - Typing `@sc` narrows to `@scale`
   - Accepting `@globals` inserts `@globals.`
   - Typing `@globals.t` narrows to matching globals

**Files**:
- `cmd/calcmark/tui/editor/autocomplete_handler.go` (modify prefix extraction)
- `cmd/calcmark/tui/editor/autocomplete.go` (add DirectiveSuggestionSource)
- `cmd/calcmark/tui/editor/model.go` (register source)
- `cmd/calcmark/tui/editor/testdata/` (catwalk tests)

### Phase 3: Integration Tests & NL Verification

**Goal**: End-to-end verification that directives work in all contexts, with three-tier NL testing.

**Tasks**:

1. **Golden eval tests** — Add to `testdata/eval/success/features/`:
   - `directive_in_unit_expression.cm`: `@scale meters / second`
   - `directive_in_nl_function.cm`: `base growing at @globals.rate for 3 years`
   - `directive_standalone.cm`: `@scale` as bare expression

2. **Three-tier NL parity tests** (per learning):
   - Tier 1: `@scale` in NL function → correct absolute value
   - Tier 2: NL result matches functional syntax result
   - Tier 3: Directive arg produces same result as equivalent literal arg

3. **Scale exemption regression** — Verify `a = @scale * 3` is NOT double-scaled (existing behavior, add explicit regression test if missing)

4. **Run `task quality`** — Full suite validation

**Files**:
- `testdata/eval/success/features/` (new golden files)
- `spec/classifier/classifier_test.go` (integration-level tests)

## Key Decisions

- **Classify as Calculation even without frontmatter** — Runtime errors are more helpful than silent markdown treatment (see origin: docs/brainstorms/2026-03-20-directive-as-value-requirements.md)
- **Don't modify `isWordRune`** — Surgical `getCurrentWordPrefix` change avoids breaking `email@example` patterns
- **Only offer directives when frontmatter defines them** — `@scale` appears only when `scale:` exists; `@globals.x` only when globals are defined. Reduces noise.
- **Explicit @scale use opts out of auto-scaling** — Already implemented via `ContainsScaleRef`. No changes needed. (see origin)
- **`allIdentifiersDefined` returns `true` for DirectiveRef unconditionally** — Classifier does syntactic classification, not semantic validation

## Dependencies & Prerequisites

- No external dependencies
- No migration or data changes
- Parser, interpreter, semantic checker, and scale exemption are all already complete

## Sources & References

### Origin

- **Origin document:** [docs/brainstorms/2026-03-20-directive-as-value-requirements.md](docs/brainstorms/2026-03-20-directive-as-value-requirements.md) — Key decisions: explicit @scale opts out of auto-scaling; @directive is a number everywhere; no new directive types

### Internal References

- Classifier: `spec/classifier/classifier.go` (lines 179-341)
- Document detector (already works): `spec/document/detector.go:322`
- Parser directive handling: `spec/parser/rdparser.go:1045-1072`
- Scale exemption: `spec/ast/nodes.go:416-453`, `impl/document/evaluator.go:676-727`
- Interpreter eval: `impl/interpreter/interpreter.go:139-166`
- Autocomplete handler: `cmd/calcmark/tui/editor/autocomplete_handler.go:218-247`
- Autocomplete sources: `cmd/calcmark/tui/editor/autocomplete.go`

### Institutional Learnings Applied

- `docs/solutions/security-issues/nan-inf-panic-yaml-frontmatter-scale.md` — No new decimal.NewFromFloat paths introduced
- `docs/solutions/logic-errors/go-closure-capturing-stale-value-type.md` — Use pointer indirection for autocomplete source
- `docs/solutions/integration-issues/nl-functional-syntax-parity-and-doc-staleness.md` — Three-tier testing for NL syntax
- `docs/solutions/logic-errors/adding-new-type-fraction-cross-layer-checklist.md` — Classifier is the most commonly missed layer
- `docs/solutions/ui-bugs/display-formatter-overrides-explicit-unit-conversion.md` — Respect IsExplicit flag (no changes needed here)
