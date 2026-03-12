---
title: "feat: Full Preview Rendering — Transform Indicators & Frontmatter Display"
type: feat
status: completed
date: 2026-03-12
origin: docs/brainstorms/2026-03-12-transform-result-indicators-brainstorm.md
---

# Full Preview Rendering — Transform Indicators & Frontmatter Display

## Overview

Two related improvements to the TUI preview pane that make it a complete, informative companion to the source pane:

1. **Transform result indicators** — Visual symbols showing which transforms (scale, convert_to) were applied to each result
2. **Frontmatter preview rendering** — Show frontmatter YAML as styled text instead of mostly-blank rows

Both features share the preview pane rendering pipeline and are scoped to make the preview pane truly useful at a glance.

## Part A: Transform Result Indicators

### Problem

Currently only scaling shows a `*` suffix in orange/amber. There is no indicator for `convert_to` transforms. The `*` is also ambiguous — it could mean "changed" (the `changedMarker` prefix also uses `*`). Users cannot tell at a glance which transforms affected a result.

### Solution

(see brainstorm: `docs/brainstorms/2026-03-12-transform-result-indicators-brainstorm.md`)

| Transform | Symbol | Color |
|-----------|--------|-------|
| Scaled only | `×` (U+00D7) | Orange/amber (existing `ScaleIndicator`) |
| Converted only | `•` (U+2022) | New `ConvertIndicator` theme color |
| Both | `×•` (concatenated) | Each in its own color |
| Neither | (none) | — |

### Critical Design Decision: How to Detect Conversion

**`WouldConvert()` cannot work like `WouldScale()`.**

`WouldScale()` works on post-transform results because scaling changes the value but not the unit — category matching still works. Conversion changes the unit itself, so calling `WouldConvert()` on an already-converted result would check the *new* unit against the target system and always return `false` (it's already there).

**Approach: Evaluator-cached conversion flags.**

The evaluator at `impl/document/evaluator.go` already has access to pre-transform results and computes `scaleExempt []bool`. We add parallel tracking:

1. In `applyBlockTransform`, compare each result's unit before and after `transform.Apply`
2. Store `convertApplied []bool` on `CalcBlock` (parallel to `ScaleExempt`)
3. `results.go` reads the cached flag instead of calling a predicate

This is the only reliable approach — it handles all edge cases: `IsExplicit` quantities, custom units, already-in-target-system, conversion failures, rates.

### Implementation Tasks

#### A1. Add `WouldConvert()` to `spec/transform/transform.go`

- [x] Implement `WouldConvert(result types.Type, cfg *document.ConvertToConfig) bool`
- [x] Must check: non-nil result/config, type is `*types.Quantity` or `*types.Rate`, not `IsExplicit`, not custom unit, category matches, not already in target system, target unit exists
- [x] Add unit tests in `spec/transform/transform_test.go`
- [x] This is for non-TUI consumers (CLI JSON, REPL). The TUI uses evaluator-cached flags.

#### A2. Add conversion tracking to `CalcBlock`

- [x] Add `convertApplied []bool` field to `CalcBlock` in `spec/document/block.go`
- [x] Add `ConvertApplied() []bool` and `SetConvertApplied([]bool)` methods
- [x] Parallel to existing `ScaleExempt()` / `SetScaleExempt()` pattern

#### A3. Compute conversion flags in evaluator

- [x] In `impl/document/evaluator.go`, within the transform application section (~line 666-677)
- [x] Before `transform.Apply`, snapshot each result's unit string
- [x] After `transform.Apply`, compare units — if different, mark `convertApplied[i] = true`
- [x] Call `cb.SetConvertApplied(convertApplied)` after the loop
- [x] Add tests in `impl/document/evaluator_test.go` or `diagnostic_test.go`

#### A4. Add `IsConverted` to `LineResult` in `results.go`

- [x] Add `IsConverted bool` field to `LineResult` struct (`cmd/calcmark/tui/editor/results.go:26`)
- [x] In `GetLineResults()` (~line 188-196), read `cb.ConvertApplied()` and set `IsConverted`
- [x] Pattern follows existing `IsScaled` / `WouldScale` logic but uses cached flags

#### A5. Add `ConvertIndicator` theme color to `palette.go`

- [x] Add `ConvertIndicator` color to `cmd/calcmark/config/theme/palette.go`
- [x] Light: `#0891B2` (teal-600), Dark: `#22D3EE` (teal-300) — distinct from amber scale, green results, red errors
- [x] Must be distinguishable from `ScaleIndicator` (#D97706/#F59E0B) when side-by-side
- [x] Verify contrast on both `PreviewPaneBg` variants

#### A6. Update indicator rendering in `view_panes.go`

- [x] Replace `*` scale indicator with `×` (U+00D7) at `view_panes.go:723-730`
- [x] Add `•` (U+2022) convert indicator with `ConvertIndicator` color
- [x] Compose `×•` when both `IsScaled` and `IsConverted` are true
- [x] Both symbols rendered with their own styled color, concatenated
- [x] Update the `scaleSuffix` variable name to `transformSuffix` or similar

#### A7. Add catwalk tests

- [x] Test all four indicator states: none, scale only, convert only, both
- [x] Use the `results` observer to verify `IsScaled` and `IsConverted` flags
- [x] Exercise: custom units (scale only), volume with SI convert (both), booleans (none), rates (convert only if applicable)

#### A8. Handle `@scale`-exempt + converted interaction

- [x] When a statement references `@scale`, it gets `nil` scale config but still gets `convert_to` applied
- [x] Verify `IsScaled=false, IsConverted=true` for these cases
- [x] Add test case

### Edge Cases

- **`IsExplicit` quantities** (`weight = 280 grams in grams`): `WouldConvert` returns false, no `•` indicator
- **Already in target system** (`10 mL` with `convert_to: si`): no conversion happens, no `•`
- **Napkin estimates**: `~946.35 mL×•` — the `~` prefix and indicator suffix coexist naturally
- **Error results**: no indicators (no transform applied)
- **Conversion failure after scale**: `IsScaled=true`, `IsConverted=false` (unit comparison catches this)

---

## Part B: Frontmatter Preview Rendering

### Problem

The preview pane renders most frontmatter lines as blank rows. Only `exchange:` and `globals:` value-lines show formatted values. Everything else — `hello: world`, `scale:`, `factor: 2`, `convert_to:`, `unit_categories:`, list items — falls through to empty rows. This wastes vertical space and looks broken (see screenshot).

The `---` YAML delimiters must NOT be rendered as markdown horizontal rules.

### Solution

Render frontmatter as styled YAML text in the preview pane across all preview modes. No glamour markdown rendering — just syntax-colored YAML.

The user has explicitly said they're willing to NOT render evaluated values for globals in this case, just the text. This simplifies the approach significantly: every frontmatter line gets rendered as styled YAML text, period.

### Design Decisions

1. **No glamour for frontmatter** — Glamour would interpret `---` as `<hr>`. Frontmatter is YAML, not markdown.
2. **Plain YAML text rendering** — Each frontmatter line rendered with a YAML-appropriate style (keys in one color, values in another, or a single `SourceFrontmatter` tint).
3. **Drop the value-map approach** — The current `buildFrontmatterValueMap()` / `extractFrontmatterKey()` / `renderFrontmatterValueLine()` system becomes unnecessary. Every frontmatter line is rendered as text.
4. **Preserve 1:1 alignment** — Each frontmatter source line still maps to exactly one preview line. No wrapping needed for typical YAML.
5. **All preview modes** — Full, Minimal, Rendered, Reading all render frontmatter text.

### Implementation Tasks

#### B1. Simplify frontmatter preview rendering in `view_panes.go`

- [x] Replace the structural/value branching logic (lines 324-356) with simple YAML text rendering
- [x] Each frontmatter line: render the source text with a frontmatter-appropriate style
- [x] Keep the error rendering for the closing `---` when `frontmatterErr != nil` — that's still useful
- [x] `---` delimiters render as styled `---` text (not blank, not `<hr>`)

#### B2. Style the YAML text

- [x] Use existing `SourceFrontmatter` color or a new preview-specific frontmatter style
- [x] Keys could be slightly brighter/bolder than values for readability
- [x] Or keep it simple: one consistent style for all frontmatter text in the preview
- [x] The style must work on `PreviewPaneBg` (not `SourcePaneBg`)

#### B3. Update Reading mode frontmatter rendering

- [x] The reading mode path at `view_panes.go:460` and `view_panes.go:574` also handles frontmatter
- [x] Apply the same YAML text rendering there
- [x] Verify frontmatter appears in reading mode (currently may be filtered out)

#### B4. Remove or simplify helper functions

- [x] `buildFrontmatterValueMap()` in `view_state.go` — may become unused
- [x] `extractFrontmatterKey()` — may become unused
- [x] `renderFrontmatterValueLine()` — replaced by simple text rendering
- [x] `isFrontmatterStructuralLine()` — may still be useful for error line detection, but the blank-row behavior changes
- [x] Clean up only what's truly unused; don't remove prematurely

#### B5. Add catwalk tests for frontmatter preview

- [x] Test that frontmatter YAML text appears in the preview pane (was previously blank)
- [x] Test that `---` renders as text, not as a horizontal rule
- [x] Test all preview modes show frontmatter text
- [x] Test frontmatter error still displays on closing `---`

#### B6. Verify existing frontmatter tests still pass

- [x] `frontmatter_test.go` — 14 test functions
- [x] `frontmatter_preview_jump_test.go` — cursor stability
- [x] `frontmatter_preview_stale_test.go` — globals positioning
- [x] `frontmatter_delimiter_test.go` — delimiter editing
- [x] Many catwalk golden files reference frontmatter — may need regeneration

### Edge Cases

- **Long YAML values** — Truncate with ellipsis if wider than preview pane
- **Frontmatter parse errors** — Still show `⚠` error on closing `---` (existing behavior preserved)
- **Empty frontmatter** (`---\n---`) — Two `---` lines rendered as styled text
- **YAML comments** (`# comment`) — Render as text in a dimmer style

---

## Acceptance Criteria

### Transform Indicators (Part A)
- [x] `×` (U+00D7) replaces `*` for scaled results
- [x] `•` (U+2022) appears for converted results in distinct color
- [x] `×•` appears when both transforms apply
- [x] No indicator for booleans, strings, errors, unaffected results
- [x] `IsExplicit` quantities show no convert indicator
- [x] `@scale`-exempt statements can still show convert indicator
- [x] All tests pass (`task test`)

### Frontmatter Preview (Part B)
- [x] All frontmatter lines render as styled YAML text in preview pane
- [x] `---` delimiters render as text, not horizontal rules
- [x] Works in all preview modes (Full, Minimal, Rendered, Reading)
- [x] Frontmatter parse errors still display on closing `---`
- [x] 1:1 source-preview alignment preserved
- [x] All existing frontmatter tests pass or are updated
- [x] All tests pass (`task test`)

## Key Files

| File | Change |
|------|--------|
| `spec/transform/transform.go` | Add `WouldConvert()` |
| `spec/document/block.go` | Add `convertApplied` field and methods |
| `impl/document/evaluator.go` | Compute conversion flags in transform section |
| `cmd/calcmark/tui/editor/results.go` | Add `IsConverted` to `LineResult` |
| `cmd/calcmark/config/theme/palette.go` | Add `ConvertIndicator` color |
| `cmd/calcmark/tui/editor/view_panes.go` | Replace `*` with `×`/`•`, render frontmatter as YAML text |
| `cmd/calcmark/tui/editor/view_state.go` | Simplify/remove frontmatter value helpers |

## Sources

- **Origin brainstorm:** [docs/brainstorms/2026-03-12-transform-result-indicators-brainstorm.md](docs/brainstorms/2026-03-12-transform-result-indicators-brainstorm.md) — symbol choices (`×`, `•`, `×•`), color strategy, composability
- **DocLine learning:** [docs/solutions/code-organization/docline-diagnostic-line-numbers.md](docs/solutions/code-organization/docline-diagnostic-line-numbers.md) — TUI two-layer architecture context
- **SpecFlow analysis:** Identified critical pre/post-transform detection issue (Gap 16) — `WouldConvert()` cannot work on post-transform results, must use evaluator-cached flags
