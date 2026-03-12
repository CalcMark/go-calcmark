---
title: "feat: Expandable Diagnostic Footer"
type: feat
status: completed
date: 2026-03-12
origin: docs/brainstorms/2026-03-12-expandable-diagnostic-footer-brainstorm.md
---

# feat: Expandable Diagnostic Footer

## Overview

Evolve the TUI context footer from a fixed 2-line component to a dynamically-sized component (2-4 lines) that shows structured diagnostic content when the cursor is on an error line. Currently, frontmatter errors like `invalid unit category "Weight"` are truncated in the narrow preview pane, and frontmatter errors are completely invisible in the context footer due to an `IsCalc` gate.

## Problem Statement

Three interrelated problems:

1. **Frontmatter errors invisible in footer** — `view_footer.go:23` gates errors with `currentResult.IsCalc`, so frontmatter errors (`IsFrontmatter=true`) never appear in the context footer. The user's only feedback is a truncated `⚠` message in the preview pane.

2. **2 lines too short for diagnostics** — Complex errors (invalid unit category with list of valid options, incompatible units with type details) get truncated to a single line. The most actionable part of the error (e.g., the valid categories list) is cut off.

3. **Double "frontmatter:" prefix** — Error messages show `frontmatter: frontmatter: invalid unit category...` due to `ParseFrontmatter` wrapping with "frontmatter:" and `NewDocument` wrapping again.

(see brainstorm: docs/brainstorms/2026-03-12-expandable-diagnostic-footer-brainstorm.md)

## Proposed Solution

### Approach: Evolve the Context Footer

Make the context footer height dynamic — 2 lines by default, expanding to up to 4 lines when the cursor is on a line with an error. The expansion is content-driven (uses only the lines needed, up to 4 max).

### Key Design Decisions (from brainstorm)

1. **Evolve context footer** — not a separate component
2. **Max 4 lines** — bounded, predictable height impact
3. **Cursor on error line triggers expansion** — same UX as current footer
4. **Structured content** — error + hint + context, not raw word-wrap
5. **Defer help links** — ship structured diagnostics first
6. **Fix double "frontmatter:" prefix** — as part of this work

### Diagnostic Layout

```
⚠ Invalid unit category "Weight"
💡 Valid categories: All, Area, Currency,
  Custom, DataSize, Energy, Length, Mass,
  Number, Power, Speed, Temperature, Volume
```

Line 1: Error icon + cleaned message. Lines 2-4: Actionable hint + context, word-wrapped.

Note: "Did you mean X?" fuzzy matching is deferred (see brainstorm scope). v1 shows the valid options list extracted from the error message.

## Technical Approach

### Implementation Phases

#### Phase 1: Foundation — Ungate Frontmatter Errors + Fix Double Prefix

Quick wins that deliver value immediately and can be tested independently.

- [x] **Fix `IsCalc` gate** in `view_footer.go:23` — Change condition from `currentResult.IsCalc && currentResult.Error != ""` to `currentResult.Error != ""` so errors on any line type (calc, frontmatter, text) flow into the footer
  - File: `cmd/calcmark/tui/editor/view_footer.go:23`

- [x] **Add frontmatter `Diagnostic`** — When attaching `frontmatterErr` to the closing `---` `LineResult`, also create a synthetic `Diagnostic{Code: "frontmatter_validation", Message: ...}` so the hint system can match on it
  - File: `cmd/calcmark/tui/editor/results.go:55-57`

- [x] **Fix double "frontmatter:" prefix** — Add deduplication to `CleanErrorMessage()` that strips repeated `"frontmatter: "` prefixes
  - File: `cmd/calcmark/tui/components/errors.go:113`

- [x] **Add frontmatter hint** to `GetHintForDiagnostic()` — New case for `"frontmatter_validation"` code that extracts the valid options list from the error message and formats it as a hint (e.g., "Valid categories: All, Area, ..."). No fuzzy "did you mean" matching — just show the valid options.
  - File: `cmd/calcmark/tui/components/errors.go:19`

- [ ] **Catwalk test** — Cursor on frontmatter error line shows error + hint in context footer (even if still 2 lines)
  - File: `cmd/calcmark/tui/editor/testdata/diagnostic_frontmatter_footer`

**Success criteria:** Frontmatter errors appear in the context footer. Hint shows valid categories. No double prefix. All existing tests pass.

#### Phase 2: Dynamic Footer Height

Make the footer height respond to content needs.

- [x] **Add `contextFooterHeight()` helper** — A pure function that takes the `[]LineResult` slice and cursor line index, returns 2 (no error) or the needed height up to 4 (error with hint). Called in `View()` after `GetLineResults()` but before computing `contentHeight`. Must also check that autocomplete/function-help aren't active (those cap footer at 2).
  - File: `cmd/calcmark/tui/editor/view.go` (new helper)

- [x] **Replace magic number `4`** in `contentHeight` formula — Change `max(totalHeight-components.StatusBarHeight-4, 5)` to `max(totalHeight-components.StatusBarHeight-footerHeight-2, 5)` where `footerHeight` is the dynamic value and `2` accounts for separator + empty line
  - File: `cmd/calcmark/tui/editor/view.go:93-94`

- [x] **Change `RenderContextFooter` signature** — Add `maxHeight int` parameter. Replace `ContextFooterHeight` constant usage inside `padToHeight` closure with this parameter. Function always returns exactly `maxHeight` lines.
  - File: `cmd/calcmark/tui/components/contextfooter.go:57`
  - Keep `ContextFooterHeight = 2` as the **default** height constant (used when no error)

- [x] **Word-wrap hint text** — When `maxHeight > 2`, the error display branch should word-wrap the hint across lines 2-N instead of truncating. Use `geometry.WrapText` for consistent wrapping logic.
  - File: `cmd/calcmark/tui/components/contextfooter.go:159-202`

- [x] **Audit `calculatePopupScreenPosition`** — Verify it uses the same `contentHeight` computed with the dynamic footer height, so autocomplete popups position correctly
  - File: `cmd/calcmark/tui/editor/view_overlays.go:89`

- [x] **Behavioral tests** (in `dynamic_footer_height_test.go`):
  - Footer shows full hint on error line
  - Hint not truncated at narrow width
  - View has correct line count at various terminal sizes (including small + error)
  - Cursor visible when footer expands
  - Footer does not expand on non-error line
  - Footer transitions correctly on cursor movement
  - Autocomplete suppresses footer expansion

**Success criteria:** Footer dynamically sizes between 2-4 lines. Pane heights recalculate correctly. No rendering artifacts. Autocomplete popup positions correctly.

#### Phase 3: Polish

- [x] **Blocked error handling** — When `IsBlocked=true`, show a brief hint ("Caused by error above — fix it first") instead of the full diagnostic. Keep footer at 2 lines for blocked errors since the user should fix the root cause.
  - File: `cmd/calcmark/tui/editor/view_footer.go`, `cmd/calcmark/tui/editor/view.go`
  - Note: Current evaluator handles cascading errors gracefully (downstream lines don't error), so `IsBlocked` is a defensive code path.

- [x] **Priority interaction** — Autocomplete (P0) keeps footer at 2 lines via `contextFooterHeight` check. Function help (P0.5) renders 2 lines of content; extra footer height from error is harmlessly filled by `padToHeight` with styled background.
  - File: `cmd/calcmark/tui/editor/view.go:319-324`

- [x] **Background color discipline** — Verified: all error display styles (`iconStyle`, `msgStyle`, `hintStyle`, `bgText`) use `.Background(bg)`. `padToHeight` fills empty lines and partial widths with `StyledPadding(width, bg)`.
  - File: `cmd/calcmark/tui/components/contextfooter.go:162-215`

## Acceptance Criteria

- [x] Frontmatter errors appear in the context footer when cursor is on the closing `---` line
- [x] Error messages show structured content: icon + message on line 1, hint on lines 2-4
- [x] Footer expands from 2 to up to 4 lines when cursor is on an error line
- [x] Footer shrinks back to 2 lines when cursor moves off the error line
- [x] Pane heights recalculate correctly when footer size changes
- [x] Double "frontmatter:" prefix is cleaned up
- [x] `GetHintForDiagnostic` handles `frontmatter_validation` errors with valid options list
- [x] Autocomplete popup positions correctly when footer is expanded
- [x] All existing catwalk tests pass
- [x] New behavioral tests cover: hint visibility, line count correctness, cursor visibility, footer transitions, autocomplete interaction

## Dependencies & Risks

**Dependencies:**
- Existing `ParseErrorForDisplay()` and `GetHintForDiagnostic()` infrastructure
- `geometry.WrapText` for word-wrapping hint content
- Recent fix in `results.go` that attaches `frontmatterErr` to closing `---` line

**Risks:**
- **Bubbletea rendering artifacts** — Height must be deterministic per frame. Mitigated by computing height from state before rendering, never lazily during render.
- **Visual jitter on rapid cursor movement** — Holding down-arrow through interleaved error/non-error lines causes 2-row layout shifts. Acceptable for v1; debouncing deferred.
- **Autocomplete popup positioning** — Must audit `calculatePopupScreenPosition` to use the dynamic `contentHeight`.

**Institutional learnings to follow:**
- Every styled element needs explicit `Background()` (lipgloss-background-bleed-through)
- Statement index drift when mapping lines to results (context-footer-statement-index-drift)
- Test parsers must not rely on `│` delimiters that appear in multiple contexts (preview-pane-jump-frontmatter-and-context-footer-false-positive)

## Sources & References

### Origin

- **Brainstorm document:** [docs/brainstorms/2026-03-12-expandable-diagnostic-footer-brainstorm.md](docs/brainstorms/2026-03-12-expandable-diagnostic-footer-brainstorm.md) — Key decisions carried forward: evolve context footer (not separate component), max 4 lines, defer help links.

### Internal References

- Context footer constant: `cmd/calcmark/tui/components/contextfooter.go:15`
- Context footer render: `cmd/calcmark/tui/components/contextfooter.go:57`
- Height formula: `cmd/calcmark/tui/editor/view.go:93-94`
- IsCalc gate: `cmd/calcmark/tui/editor/view_footer.go:23`
- Frontmatter error attachment: `cmd/calcmark/tui/editor/results.go:55-57`
- Hint system: `cmd/calcmark/tui/components/errors.go:19-97`
- Popup positioning: `cmd/calcmark/tui/editor/view_overlays.go:89`
- Double prefix source: `spec/document/frontmatter.go:527` + `spec/document/document.go:42`
