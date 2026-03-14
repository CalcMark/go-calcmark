---
title: "feat: Full Preview Pane Markdown Rendering"
type: feat
status: active
date: 2026-03-11
origin: docs/brainstorms/2026-03-11-full-preview-pane-rendering-brainstorm.md
---

# Full Preview Pane Markdown Rendering

## Overview

Render all markdown content in the TUI preview pane — paragraphs, lists, tables, emphasis, links, code blocks, and interpolated `{{variable}}` values. Today, non-heading TextBlock lines render as empty strings. This feature adds a new `PreviewRendered` mode (toggled via Ctrl+P) that shows full glamour-rendered markdown for TextBlocks while preserving existing calc result rendering for CalcBlocks.

The architecture introduces a memoized `RenderedBlockCache` middleware between the evaluator and alignment layer, block-level alignment for TextBlocks (not line-level), and viewport-scoped rendering — all while keeping `ComputeAlignedModel` a pure function and `GetLineResults()` completely untouched.

## Problem Statement

The preview pane currently renders calc results well but treats most markdown as invisible:
- `renderPerLinePreview` in `aligned.go:281` explicitly returns `""` for non-heading, non-list TextBlock lines
- `LineResult.Source` contains raw `{{variable}}` templates, not interpolated values (`results.go:203`)
- No caching exists — every `View()` frame creates new glamour renderers and recomputes everything

Users see a preview pane that's half-empty for markdown-heavy documents, despite the evaluator already computing interpolated source via `TextBlock.InterpolatedSource()`.

## Proposed Solution

Add a `PreviewRendered` mode that uses block-level glamour rendering for TextBlocks with a memoized cache middleware. The pipeline becomes:

```
Evaluator (unchanged)
  | Calc eval -> Transforms -> Interpolation -> InterpolatedSource()
  v
RenderedBlockCache (NEW)
  | Per TextBlock: key=(blockID, interpolatedSource, width)
  | Cache hit -> return []string
  | Cache miss -> glamour render -> store + return []string
  | Only renders blocks near viewport (LRU, capacity ~128)
  v
ComputeAlignedModel (modified signature)
  | New field: AlignedModelInput.RenderedTextBlocks map[string][]string
  | TextBlocks in PreviewRendered: block-level alignment
  | CalcBlocks: line-level alignment (unchanged)
  | PreviewFull/PreviewMinimal: existing behavior (unchanged)
  v
View rendering (minor: respects PreviewRendered mode)
```

## Technical Approach

### Architecture

#### RenderedBlockCache

A standalone, testable struct with a simple interface:

```go
// rendered_block_cache.go (new file in editor package)
type RenderedBlockCache struct {
    entries map[cacheKey][]string
    lru     // LRU eviction, capacity ~128 entries
    renderer *MarkdownRenderer // reused glamour instance
}

type cacheKey struct {
    blockID            string
    interpolatedSource string // hash or full string
    width              int
}

func (c *RenderedBlockCache) Render(blockID string, interpolatedLines []string, width int) []string
func (c *RenderedBlockCache) Invalidate(blockID string)
func (c *RenderedBlockCache) Clear()
```

The cache lives as a `*RenderedBlockCache` pointer field on `Model` (shared across `View()` value copies via pointer indirection — same pattern needed because Bubble Tea's `View()` receives a value copy, see `view.go:246-249`).

#### Data Flow: InterpolatedSource to Cache

The brainstorm decided **not to modify `GetLineResults()`** (see brainstorm: decision "Don't Touch GetLineResults"). Instead, `computeAlignedModelFresh()` in `view.go` builds the rendered TextBlock map by iterating `doc.GetBlocks()` directly:

```go
// In computeAlignedModelFresh() — only when PreviewRendered mode
renderedBlocks := make(map[string][]string)
if m.previewMode == PreviewRendered {
    for _, bn := range m.doc.GetBlocks() {
        tb, ok := bn.Block.(*document.TextBlock)
        if !ok { continue }
        blockID := bn.Block.ID()
        if !m.isBlockNearViewport(blockID) { continue }
        renderedBlocks[blockID] = m.renderCache.Render(blockID, tb.InterpolatedSource(), previewWidth)
    }
}
input.RenderedTextBlocks = renderedBlocks
```

This uses the existing `doc.GetBlocks()` → type-assert pattern (same as `results.go:201-202`).

#### Block-Level Alignment for TextBlocks

When `input.RenderedTextBlocks[blockID]` is present, `ComputeAlignedModel` treats the entire TextBlock as a single alignment unit:

1. Count source visual lines for the block — each source line may wrap to multiple visual lines via `geometry.WrapText`, so sum wrapped line counts across all source lines in the block.
2. Count rendered lines from `RenderedTextBlocks[blockID]`.
3. `numAligned = max(totalSourceVisualLines, renderedCount)`.
4. Source side: each source line's wrapped visual lines are emitted in order (with `AlignedLineNormal`/`AlignedLineWrapped` kinds as today). If total source visual lines < `numAligned`, append padding lines.
5. Preview side: rendered lines fill top-down. If rendered count < `numAligned`, append padding lines.
6. `SourceToVisual` and `VisualToSource` mappings populated per visual line, same as today — each visual line maps back to its source line index.

This is structurally similar to the existing `renderOrderedListBlock` distribution (see `aligned.go:229-264`), but simpler because we don't need proportional distribution — preview content is opaque, just displayed top-to-bottom.

**Fallback:** If `RenderedTextBlocks` is `nil` or missing a block, existing per-line behavior is used. This makes the change fully backwards-compatible and allows incremental rollout.

#### EditBuf Interaction During Live TextBlock Editing

When typing in a TextBlock (EditBuf active), the preview must show live content. The approach:

1. `computeAlignedModelFresh()` splices `editBuf` into the TextBlock's interpolated source before passing to the cache
2. The cache key includes the spliced content, so it misses on every keystroke (expected — one block, ~0.5-3ms)
3. For `{{variable}}` templates in the EditBuf line: use the last-known interpolated values from `InterpolatedSource()` for non-cursor lines; for the cursor line, show the raw EditBuf text. This accepts a brief flicker of raw templates on the cursor line until debounce fires and re-interpolation runs.
4. Partial templates like `{{vari` pass through glamour as literal text (glamour treats unknown `{{` as plain text, not errors).

This follows the documented learning: "EditBuf live text must feed into alignment computation" (see `docs/solutions/ui-bugs/tui-mode-transitions-formatter-indexing-and-bracketed-paste-fixes.md`).

#### PreviewRendered Mode Integration

```go
// model.go
const (
    PreviewFull    PreviewMode = iota
    PreviewMinimal
    PreviewRendered  // NEW
    PreviewHidden
)

var DefaultPaneWidths = map[PreviewMode][2]int{
    PreviewFull:     {60, 40},
    PreviewMinimal:  {75, 25},
    PreviewRendered: {50, 50},  // Equal split — markdown benefits from width
    PreviewHidden:   {100, 0},
}
```

`cyclePreviewMode()` in `file_operations.go:262` updated: Full → Minimal → Rendered → Hidden → Full.

CalcBlock rendering in `PreviewRendered` mode: same as `PreviewFull` (shows `varName → value`). Only TextBlock rendering changes.

### Implementation Phases

#### Phase 0: Glamour Performance Benchmark (Prerequisite)

**Goal:** Establish baseline glamour rendering cost. Gate subsequent phases on results.

**Tasks:**
- Write `benchmark_glamour_test.go` in the editor package
- Benchmark `NewMarkdownRenderer(width)` creation cost
- Benchmark `RenderLine()` for representative inputs at 5, 20, 50 lines
- Benchmark with headings, lists, tables, emphasis, code blocks, mixed content
- Measure GC pressure (allocations per render)

**Success criteria:**
- Single block render (20 lines) < 3ms
- Renderer creation < 0.5ms
- If results exceed these thresholds, document findings and reassess approach before proceeding

**Estimated effort:** Small

#### Phase 1: RenderedBlockCache (Standalone)

**Goal:** Implement and test the cache middleware in isolation.

**Tasks:**
- Create `rendered_block_cache.go` with `RenderedBlockCache` struct
- Implement LRU eviction (capacity configurable, default 128)
- Reuse a single `MarkdownRenderer` instance per width (avoid per-call creation)
- Write comprehensive unit tests:
  - Cache hit returns stored content
  - Cache miss invokes glamour and stores result
  - Width change causes cache miss
  - Content change causes cache miss
  - LRU eviction under capacity pressure
  - Concurrent safety (single goroutine, but test for value semantics)
  - Empty block input
  - Glamour render failure falls back to raw source

**Success criteria:**
- Cache hit: < 0.01ms (map lookup)
- Cache miss: glamour render + store
- All unit tests pass
- No dependency on `ComputeAlignedModel` or `GetLineResults`

**Estimated effort:** Medium

#### Phase 2: PreviewRendered Mode Skeleton

**Goal:** Add the mode to the type system without changing rendering behavior.

**Tasks:**
- Add `PreviewRendered` constant to `PreviewMode` enum (`model.go:214`)
- Add entry to `DefaultPaneWidths` (`model.go:228`)
- Update `cyclePreviewMode()` (`file_operations.go:262`)
- Add `*RenderedBlockCache` field to `Model` struct, initialize in `New()`
- Add cache clear to `resetForNewDocument()` in `mode_transitions.go`
- Add `isBlockNearViewport()` helper on Model — maps blockID to source line range via LineResults, checks against `scrollOffset + viewportHeight`
- Write catwalk test: Ctrl+P cycles through all four modes
- Write catwalk test: PreviewRendered mode shows correct pane width ratio

**Success criteria:**
- Ctrl+P cycles Full → Minimal → Rendered → Hidden → Full
- In PreviewRendered mode, preview pane appears at 50/50 split
- Cache cleared on new file open (verified by test)
- Preview content is still empty strings for TextBlocks (rendering wired in Phase 3a)
- Zero regressions in existing modes

**Estimated effort:** Small

#### Phase 3a: Wire Cache Into View Pipeline (Data Plumbing)

**Goal:** Connect the cache to `ComputeAlignedModel` as a new data input, without changing alignment logic. TextBlocks still use existing per-line rendering but sourced from the cache.

**Tasks:**
- Add `RenderedTextBlocks map[string][]string` field to `AlignedModelInput` (`aligned.go:88`)
- In `computeAlignedModelFresh()` (`view.go:286`): when `PreviewRendered`, iterate `doc.GetBlocks()`, call cache for visible TextBlocks, populate `input.RenderedTextBlocks`
- Handle EditBuf splice: when EditBuf is on a TextBlock line, splice it into the block's interpolated source before cache lookup
- In `ComputeAlignedModel`: when `RenderedTextBlocks[blockID]` exists, use cached content as the `renderMarkdown` input for per-line preview (pass-through — existing per-line alignment logic unchanged)
- Verify `AlignedModel.Invariants()` still passes

**Unit tests:**
- `RenderedTextBlocks` populated → cache content used for preview
- Empty `RenderedTextBlocks` map → fallback to existing callback behavior
- Missing block in map → fallback for that block only
- EditBuf on TextBlock line → spliced content reaches cache
- CalcBlock alignment identical with and without `RenderedTextBlocks`

**Catwalk tests:**
- Document with `{{variable}}` interpolation: verify resolved values appear in preview (previously invisible)
- Regression: all existing catwalk tests pass unchanged

**Success criteria:**
- Interpolated content visible in TextBlock preview (headings + lists, per existing filter)
- Per-line alignment behavior unchanged (no block-level shift yet)
- `AlignedModel.Invariants()` passes
- All existing tests pass

**Estimated effort:** Medium

#### Phase 3b: Block-Level TextBlock Alignment

**Goal:** Switch TextBlock preview alignment from per-line to block-level. This is the alignment algorithm change — isolated for safe testing and regression bisection.

**Tasks:**
- In `ComputeAlignedModel`: when `RenderedTextBlocks[blockID]` exists, replace per-line preview resolution with block-level alignment:
  - Collect all source lines for the block, compute total visual lines (accounting for per-line wrapping)
  - Rendered lines fill preview side top-down as an opaque block
  - `numAligned = max(totalSourceVisualLines, renderedCount)`, pad the shorter side
- Ensure `SourceToVisual` and `VisualToSource` mappings are correct for the new alignment
- Ensure `AlignedModel.Invariants()` passes for all new inputs

**Unit tests on ComputeAlignedModel (pure function):**
- TextBlock with 3 source lines (no wrapping), 7 rendered lines → 7 visual rows, source padded with 4 padding lines
- TextBlock with 5 source lines (no wrapping), 2 rendered lines → 5 visual rows, preview padded with 3 padding lines
- TextBlock with source line wrapping: 2 source lines where line 1 wraps to 3 visual lines → 4 source visual lines vs N preview lines
- Mixed document: TextBlock (block-aligned) + CalcBlock (line-aligned) + TextBlock — verify CalcBlock alignment is byte-identical
- Empty TextBlock (blank lines only) → no rendered content, padding only
- Single-line TextBlock → trivial alignment

**Catwalk tests:**
- Document with heading + paragraph + calc block: verify preview shows full rendered markdown in PreviewRendered, empty in PreviewFull
- Document with ordered list in TextBlock: verify list renders with correct numbering
- Document with table in TextBlock: rendered table takes more lines than source → source side padded
- Regression: all existing catwalk tests pass unchanged

**Success criteria:**
- Full markdown visible in preview pane in PreviewRendered mode
- `{{variable}}` values display correctly in rendered blocks
- CalcBlock preview unchanged across all modes
- `AlignedModel.Invariants()` passes for all test inputs
- Scroll sync works correctly with block-level aligned TextBlocks

**Estimated effort:** Large

#### Phase 4: Performance Validation and Polish

**Goal:** Verify the full pipeline meets the <10ms View() budget under realistic conditions.

**Tasks:**
- Write `benchmark_view_test.go`: full View() cycle with representative documents
- Measure cache hit path (typing in CalcBlock)
- Measure cache miss path (typing in TextBlock)
- Measure resize path (all visible blocks re-render)
- Measure large document (100+ blocks, only viewport rendered)
- Profile GC pressure under sustained 90 WPM typing simulation
- If budget exceeded: optimize glamour renderer reuse, reduce allocations, consider debounced preview for TextBlock edits
- Test on both dark and light terminals (per documented ANSI state learnings)

**Success criteria:**
- Cache hit path: < 5ms total View()
- Cache miss path (single TextBlock): < 10ms total View()
- Resize path: < 15ms (acceptable one-time cost)
- No ANSI background bleed-through in rendered markdown
- No visual glitches during mode cycling

**Estimated effort:** Medium

### Alternative Approaches Considered

**Per-line custom renderer (rejected):** Replace glamour with a lightweight custom ANSI renderer. Full control and no document-level parser overhead, but we'd own all rendering bugs for headings, lists, tables, emphasis, links, code blocks. Glamour already handles these correctly. (See brainstorm: "Key Decision: Block-Level Glamour Rendering")

**Line-level alignment for TextBlocks (rejected):** Reverse-map glamour output back to individual source lines using markers or proportional distribution. History shows line-level alignment is the single most fragile part of the TUI. Block-level alignment avoids this entirely. (See brainstorm: "Key Decision: Block-Level Alignment for TextBlocks")

**Cache inside ComputeAlignedModel (rejected):** Would break the pure function contract. Testing becomes harder, debugging snapshots impossible. (See brainstorm: "Key Decision: Pre-Rendered Input (Eager)")

**Modify GetLineResults (rejected):** Adding interpolated source to `LineResult` risks regressions in line numbers, diagnostics, and decorations. Parallel data structure is safer. (See brainstorm: "Key Decision: Don't Touch GetLineResults()")

## System-Wide Impact

### Interaction Graph

`View()` → `computeAlignedModelFresh()` → NEW: iterates `doc.GetBlocks()` for TextBlocks → `RenderedBlockCache.Render()` → glamour (on miss) → populates `AlignedModelInput.RenderedTextBlocks` → `ComputeAlignedModel()` (block-level alignment for TextBlocks) → `renderPreviewPaneAligned()` (displays rendered content).

Cache invalidation chain: Keystroke → `Update()` → evaluator runs → `interpolateTextBlocks()` updates `InterpolatedSource()` → next `View()` frame → cache key mismatch → cache miss → re-render.

### Error Propagation

- Glamour render failure: `RenderedBlockCache.Render()` falls back to returning raw `InterpolatedSource()` lines (graceful degradation, not crash).
- `doc.GetBlocks()` returns empty: `RenderedTextBlocks` map is empty → fallback to existing per-line behavior.
- Cache capacity exceeded: LRU evicts oldest entries → next access re-renders (slight latency, not error).

### State Lifecycle Risks

- **Cache stale after file open:** `resetForNewDocument()` must call `renderCache.Clear()`. Without this, stale entries from the previous document could match by blockID collision.
- **Cache stale after undo:** Undo restores previous document state but doesn't invalidate cache. The cache key includes `interpolatedSource`, so if undo changes content, the key misses naturally. Safe.
- **Pointer sharing:** `*RenderedBlockCache` is a pointer on `Model`. Bubble Tea's value-copy `View()` shares the pointer. Writes during `View()` are visible to other copies. Safety guarantees: (1) the cache is append-only on miss (never deletes during `View()`), (2) Bubble Tea guarantees single-goroutine execution for `Update()` and `View()`, so no concurrent access. Document both guarantees as comments on the `*RenderedBlockCache` field.

### API Surface Parity

- `PreviewMode` enum gains `PreviewRendered` — affects `cyclePreviewMode()`, `DefaultPaneWidths`, `GetPaneWidths()`, and any switch statements on `PreviewMode`.
- `AlignedModelInput` gains `RenderedTextBlocks` field — affects all test callers (~12 call sites, but nil zero-value means no breakage).
- `ComputeAlignedModel` function signature unchanged (inputs via struct field).

### Integration Test Scenarios

1. **Edit calc variable, verify TextBlock interpolation updates in preview:** Type `price = 100` → TextBlock shows "Total: **100**" in preview → Change to `price = 200` → preview updates to "Total: **200**".
2. **Ctrl+P cycling preserves content:** Open document → verify PreviewFull shows calc results → Ctrl+P to Minimal → Ctrl+P to Rendered (markdown appears) → Ctrl+P to Hidden → Ctrl+P back to Full (calc results still correct).
3. **Large document scroll performance:** Open 500-line document → scroll rapidly → verify no frame drops or visual artifacts → only viewport blocks rendered.
4. **Terminal resize during PreviewRendered:** Resize terminal → all visible blocks re-render at new width → no stale narrow/wide content visible.
5. **EditBuf in TextBlock with interpolation:** Type in TextBlock containing `{{var}}` → preview shows rendered markdown with raw `{{var}}` on cursor line → move cursor away → debounce fires → preview shows interpolated value.

## Acceptance Criteria

### Functional Requirements

- [ ] New `PreviewRendered` mode renders full markdown for TextBlocks via glamour
- [ ] CalcBlock preview rendering unchanged in all modes (line-level alignment preserved)
- [ ] `{{variable}}` interpolation displays resolved values in TextBlock preview
- [ ] Ctrl+P cycles: Full → Minimal → Rendered → Hidden → Full
- [ ] PreviewRendered uses 50/50 pane split
- [ ] Preview updates live during TextBlock editing (per-keystroke, no debounce)
- [ ] Existing PreviewFull and PreviewMinimal behavior unchanged

### Non-Functional Requirements

- [ ] View() < 10ms for cache-hit path (typing in CalcBlock)
- [ ] View() < 10ms for cache-miss path (typing in single TextBlock, ~20 lines)
- [ ] Cache capacity bounded at ~128 entries with LRU eviction
- [ ] No ANSI rendering artifacts on dark or light terminals
- [ ] `GetLineResults()` completely unmodified

### Quality Gates

- [ ] Glamour benchmark establishes baseline before implementation
- [ ] Unit tests on `RenderedBlockCache` (cache hit/miss/eviction/fallback)
- [ ] Unit tests on `ComputeAlignedModel` (block-level alignment, fallback, mixed blocks)
- [ ] Catwalk tests for PreviewRendered mode (rendering, mode cycling, interpolation)
- [ ] All existing tests pass unchanged (zero regressions)
- [ ] `AlignedModel.Invariants()` passes for all new test inputs
- [ ] Performance benchmarks for full View() cycle tracked in CI

## Dependencies & Prerequisites

- Glamour benchmark results (Phase 0) gate all subsequent phases
- `TextBlock.InterpolatedSource()` already implemented and populated by evaluator
- Existing `MarkdownRenderer` in `markdown.go` already has comprehensive style config for all markdown elements
- `doc.GetBlocks()` and TextBlock type assertion pattern already established in `results.go` and `block_render.go`

## Risk Analysis & Mitigation

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Glamour too slow (>5ms per block) | Medium | High | Phase 0 benchmark gates implementation. Fallback: lighter renderer or debounced preview. |
| Block-level alignment breaks scroll sync | Medium | High | Isolated in Phase 3b for safe bisection. Extensive unit tests on `SourceToVisual`/`VisualToSource` mappings. Invariants check. |
| ANSI state bleed in rendered markdown | Medium | Medium | Test on dark + light terminals. Apply documented ANSI envelope learnings. |
| Cache pointer sharing causes subtle bugs | Low | High | Append-only on miss + Bubble Tea single-goroutine guarantee. Document both as comments on field. |
| EditBuf/interpolation flicker | High | Low | Accepted trade-off: raw templates visible briefly on cursor line during fast typing. |
| Regression in existing preview modes | Low | High | Nil `RenderedTextBlocks` triggers fallback. All existing tests run unchanged. |

## Documentation Plan

- Update `TESTING.md` with new catwalk observer patterns for PreviewRendered mode
- Add inline comments on `RenderedBlockCache` explaining the pointer-sharing assumption
- Update help overlay text to mention PreviewRendered mode in Ctrl+P description

## Sources & References

### Origin

- **Brainstorm document:** [docs/brainstorms/2026-03-11-full-preview-pane-rendering-brainstorm.md](docs/brainstorms/2026-03-11-full-preview-pane-rendering-brainstorm.md) — Key decisions carried forward: block-level glamour rendering, block-level TextBlock alignment, cache middleware architecture, don't touch GetLineResults(), pre-rendered eager input.

### Internal References

- Alignment core: `cmd/calcmark/tui/editor/aligned.go:112` — `ComputeAlignedModel`
- Preview rendering: `cmd/calcmark/tui/editor/view_panes.go:172` — `renderPreviewPaneAligned`
- View orchestration: `cmd/calcmark/tui/editor/view.go:286` — `computeAlignedModelFresh`
- Markdown renderer: `cmd/calcmark/tui/editor/markdown.go:10` — `MarkdownRenderer`
- PreviewMode: `cmd/calcmark/tui/editor/model.go:211` — enum definition
- Mode cycling: `cmd/calcmark/tui/editor/file_operations.go:262` — `cyclePreviewMode`
- Interpolation: `impl/document/interpolation.go` — `interpolateTextBlocks`
- TextBlock model: `spec/document/block.go:240` — `InterpolatedSource()`
- GetLineResults: `cmd/calcmark/tui/editor/results.go:32` — unchanged, raw source
- Mode transitions: `cmd/calcmark/tui/editor/mode_transitions.go:85` — `resetForNewDocument`

### Institutional Learnings Applied

- Formatter indexing: separate `resultIdx` for non-blank lines (`docs/solutions/ui-bugs/tui-mode-transitions-formatter-indexing-and-bracketed-paste-fixes.md`)
- EditBuf must feed alignment: pass live editBuf into alignment computation (same solution doc)
- ANSI state bleed: explicit backgrounds on all styles (`docs/solutions/ui-bugs/overlay-compositing-ansi-state-bleed-through.md`, `docs/solutions/ui-bugs/lipgloss-background-bleed-through.md`)
- Centralized mode transitions: all state resets in `mode_transitions.go` (`docs/solutions/ui-bugs/tui-mode-transitions-formatter-indexing-and-bracketed-paste-fixes.md`)
- View module organization: pane-specific rendering in `view_panes.go` (`docs/solutions/code-organization/split-view-go-into-cohesive-modules.md`)
