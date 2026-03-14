# Full Preview Pane Markdown Rendering

**Date:** 2026-03-11
**Status:** Brainstorm

## What We're Building

Render all markdown content in the TUI preview pane — not just calc results and headings, but paragraphs, lists, tables, emphasis, links, code blocks, and interpolated `{{variable}}` values. Today, non-heading TextBlock lines render as empty strings in the preview pane.

### Current State

- Preview pane renders: calc results (`varName -> value`), headings, ordered lists
- Plain text, emphasis, links, code blocks, tables → empty string in preview
- `{{variable}}` interpolation exists at the evaluator level but `LineResult.Source` uses raw source — interpolated values are invisible in the TUI
- `renderPerLinePreview` in `aligned.go` explicitly filters non-heading text to empty
- Zero caching in the render pipeline — every `View()` frame recomputes everything, including creating new glamour renderers per call

### Goals

- All markdown content renders correctly in the preview pane
- Interpolated `{{variable}}` values display with their resolved values
- Typing at 90 WPM feels instantaneous (View() < 10ms)
- No regressions in CalcBlock line-level alignment
- Purely functional, composable rendering preserved

## Why This Approach

### Key Decision: Block-Level Glamour Rendering

**Decision:** Render each TextBlock as a single glamour pass, not per-line.

**Why:** Glamour (charmbracelet's markdown renderer) processes a document in one pass. Per-line rendering breaks multi-line constructs (lists, tables, blockquotes). Block-level rendering preserves markdown semantics — list renumbering, paragraph flow, table formatting — while keeping the rendering unit manageable.

TextBlocks already group "everything between CalcBlocks" split on double-newlines. This grouping naturally captures multi-line markdown constructs. No changes to block boundaries needed.

### Key Decision: Block-Level Alignment for TextBlocks

**Decision:** TextBlocks align at block boundaries, not line-by-line.

**Why:** This was the single most important architectural decision. History shows that line-level alignment has been the most fragile part of the TUI — every change causes regressions. Glamour output has no structural relationship to input line count (a 3-line list might render to 7 terminal lines). Attempting to reverse-map glamour output to individual source lines is unreliable.

With block-level alignment:
- Source pane shows lines 5-12 (a TextBlock). Preview pane shows glamour output for that block, anchored to the same vertical start. The shorter side gets padding.
- CalcBlocks retain line-level alignment (unchanged, battle-tested behavior)
- The alignment function's job simplifies: `max(source_lines, preview_lines)` per TextBlock, pad the shorter side

### Key Decision: Cache Middleware (RenderedBlockCache)

**Decision:** A memoized middleware sits between the evaluator and `ComputeAlignedModel`.

**Why:** `ComputeAlignedModel` is a pure function and must stay pure. Caching can't live inside it. The cache middleware:
- Key: `(blockID, interpolatedSource, previewWidth)`
- Cache miss → glamour render → store `[]string`
- Cache hit → return stored `[]string`
- CalcBlocks pass through unchanged

**Performance characteristics:**
- Typing in CalcBlock (common case): all TextBlock cache hits, zero glamour calls
- Typing in TextBlock: one cache miss for edited block only
- Terminal resize: all visible blocks re-render (one-time cost)
- Only visible blocks rendered — off-screen blocks skip glamour entirely

### Key Decision: Don't Touch GetLineResults()

**Decision:** `GetLineResults()` stays unchanged. Rendered TextBlock content travels as a separate `map[blockID][]string` alongside `LineResult[]`.

**Why:** `GetLineResults()` is load-bearing and historically fragile. Line numbers, diagnostics, and decorations depend on its exact behavior. Adding rendered content as a parallel, additive data structure means:
- Zero regression risk in existing alignment code
- If the rendered map is empty/missing a block, fallback to current behavior
- Safe to roll out incrementally

### Key Decision: Pre-Rendered Input (Eager)

**Decision:** The Model pre-renders visible TextBlocks through the cache before calling `ComputeAlignedModel`. The pure function receives data, not callbacks.

**Why:** Purity is the priority. Callbacks break the data-in/data-out contract, making tests harder to write and debug snapshots impossible. The viewport filtering is the caller's responsibility — the Model already knows what's visible.

## Architecture

### Rendering Pipeline

```
Evaluator (existing, unchanged)
  | Calc eval -> Transforms (@scale, @convert_to) -> Interpolation ({{var}})
  | Output: blocks with InterpolatedSource()
  v
RenderedBlockCache (NEW middleware)
  | Per TextBlock: hash(blockID, interpolatedSource, width)
  | Cache miss -> glamour render -> store []string
  | Cache hit -> return stored []string
  | CalcBlocks: pass through unchanged
  | Only renders blocks within/near viewport
  v
ComputeAlignedModel (modified pure function)
  | Inputs: LineResult[] (unchanged) + map[blockID][]string (new)
  | TextBlocks: block-level alignment (rendered lines vs source lines)
  | CalcBlocks: line-level alignment (unchanged)
  | Output: AlignedModel
  v
View rendering (existing, minor changes)
```

### Pass Order

The evaluator already runs these passes in order:
1. **Calc evaluation** — top-down, variables accumulate
2. **Transforms** — @scale, @convert_to applied to results
3. **Interpolation** — `{{var}}` resolved against final environment, stored in `InterpolatedSource()`

The new work adds one pass after interpolation:
4. **Glamour rendering** — per visible TextBlock, using `InterpolatedSource()`

### Performance Budget

Target: **View() < 10ms** (total end-to-end ~35-45ms at 90 WPM)

| Component | Budget | Notes |
|-----------|--------|-------|
| Event handling + model update | < 1ms | Pure data |
| Cache lookup + pre-render | < 2ms | Mostly cache hits |
| ComputeAlignedModel | < 3ms | Block-level alignment is cheaper than line-level |
| View string assembly (lipgloss) | < 2ms | Unchanged |
| Headroom / GC | ~2ms | |

Cache miss path (TextBlock edit): single glamour call estimated 0.5-3ms for a typical block (10-30 lines). Must benchmark.

### What Changes vs What Doesn't

**Unchanged:**
- `GetLineResults()` — exact same behavior
- CalcBlock line-level alignment in `ComputeAlignedModel`
- CalcBlock preview rendering (`renderCalcLine`)
- Evaluator pass order and semantics
- `format/align.go` (`AlignResults`) — used by formatters, not TUI

**New:**
- `RenderedBlockCache` struct — memoized glamour renderer
- `ComputeAlignedModel` signature — additional `map[blockID][]string` input
- TextBlock handling in `ComputeAlignedModel` — block-level alignment instead of per-line
- Model pre-filters visible blocks before calling alignment

**Modified:**
- `computeAlignedModelFresh()` in `view.go` — integrates cache middleware
- `renderTextBlockPreview()` in `aligned.go` — replaced by cache lookup

## Testing Strategy

Three layers, from fast/precise to slow/comprehensive:

### Layer 1: Unit Tests on ComputeAlignedModel
Pure function with known inputs → assert exact AlignedModel output:
- Block-level padding: TextBlock with 3 source lines, 7 rendered lines → 7 aligned rows, source side padded
- CalcBlock alignment preserved: mixed doc with TextBlocks and CalcBlocks → CalcBlock lines still align 1:1
- Edge cases: empty TextBlocks, single-line blocks, blocks that wrap extensively
- Fallback behavior: missing block in rendered map → current behavior

### Layer 2: Catwalk Tests (Data-Driven TUI Tests)
End-to-end rendering verification:
- New observer for preview pane content showing rendered markdown
- Golden test files for documents with mixed calc/text/interpolation
- Regression tests for existing CalcBlock alignment behavior
- Key sequences that edit TextBlocks and verify preview updates

### Layer 3: Glamour Benchmarks
Performance regression detection:
- Benchmark glamour rendering at realistic TextBlock sizes (5, 20, 50 lines)
- Benchmark cache hit vs miss paths
- Benchmark full View() cycle with representative documents
- Track these in CI to catch performance regressions

## Resolved Questions

1. **EditBuf interaction:** Live rendering (real-time). Preview updates on every keystroke while typing in a TextBlock. The cache misses every time but it's just one block (~0.5-3ms). Consistent with how CalcBlock preview already updates as you type.

2. **Frontmatter preview:** Keep as-is. Frontmatter continues showing formatted directive values. It's YAML, not markdown — glamour wouldn't improve it.

3. **Rollout strategy:** New `PreviewMode` variant. Add `PreviewRendered` alongside existing `PreviewFull`/`PreviewMinimal`/`PreviewHidden`. Users toggle with **Ctrl+P** (existing keybinding). Ship iteratively, gather feedback, potentially make default later.

4. **Glamour performance baseline:** Prerequisite before implementation. Write and run glamour benchmarks at realistic TextBlock sizes (5, 20, 50 lines) before any architectural work begins. If glamour is too slow (>5ms for a typical block), reconsider the renderer choice before investing in the architecture. The architecture itself is renderer-agnostic — swapping glamour for a lighter renderer is possible without changing the cache/alignment design.

5. **Maximum document size:** LRU with viewport window. Keep rendered output for blocks within ~50 blocks of the viewport. Evict everything else. Re-render on scroll if needed (cache miss). 1MB theoretical max means potentially hundreds of TextBlocks, but bounded cache keeps memory predictable. Occasional ~1-frame delay on fast scrolling through large docs is acceptable.
