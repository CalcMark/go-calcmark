# Phase 2: TUI Geometry & Layout - Research

**Researched:** 2026-02-02
**Domain:** Two-column terminal layout rendering, text wrapping alignment, side-by-side pane composition
**Confidence:** HIGH

## Summary

Phase 2 addresses the core challenge of rendering a two-column editor (source + results) with correct text wrapping and vertical alignment under all scenarios. The foundation is already solid: Phase 1 created a pure `geometry` package with `WrapText`, `CalculateRowGeometry`, and `StringWidth` functions. The existing codebase already has a working `AlignedModel` computation (`aligned.go`) and `SideBySide` renderer (`sidebyside.go`) that compose the two panes. The existing `View()` function in `view.go` already orchestrates the full render pipeline.

The critical insight from codebase analysis is that **most of the infrastructure already exists**. The `code.sh` geometry algorithm is already implemented in `geometry.CalculateRowGeometry`. The `ComputeAlignedModel` function in `aligned.go` already processes blocks, wraps both panes, aligns heights with padding, and maintains bidirectional source-to-visual mappings. The `SideBySide` renderer in `sidebyside.go` already guarantees every output line is exactly `leftWidth + rightWidth` characters with full background coverage.

**What Phase 2 actually needs to do** is: (1) verify and harden the existing layout pipeline against the specific success criteria (overlapping text, bleed-through, resize reflow), (2) add targeted integration tests that prove the five success criteria hold, (3) fix any gaps found during hardening, and (4) ensure the `geometry.CalculateRowGeometry` function is fully integrated into the render path (currently `ComputeAlignedModel` uses `geometry.WrapText` directly but not `CalculateRowGeometry`).

**Primary recommendation:** This phase is primarily a hardening and testing phase, not a greenfield build. The infrastructure exists. The work is: write tests for each success criterion, run them, fix what fails, verify terminal resize behavior, and ensure `CalculateRowGeometry` from the geometry package is the canonical row-level computation.

## Standard Stack

The established libraries/tools for this domain:

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| geometry package | local | Pure two-column layout math (WrapText, CalculateRowGeometry, StringWidth) | Created in Phase 1. Zero TUI deps. Foundation for all layout. |
| lipgloss | v1.1.1+ | Terminal styling (backgrounds, padding, width measurement) | Already used extensively in view.go, sidebyside.go. Handles ANSI-aware width. |
| bubbletea | v1.3.10 | TUI framework (Model-Update-View, WindowSizeMsg) | Already the framework. Provides terminal resize events. |
| glamour | v0.10.0 | Markdown rendering for preview pane | Already used in markdown.go for text block previews. |
| mattn/go-runewidth | v0.0.16 | Unicode width measurement | Used by geometry package. The canonical width function for plain text. |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| catwalk | v0.1.4 | Data-driven TUI testing | For integration tests that simulate key sequences and verify View() output. |
| datadriven | v1.0.2 | Test framework for catwalk | Underlying test runner for catwalk-based tests. |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Custom SideBySide renderer | lipgloss.JoinHorizontal | JoinHorizontal does not guarantee per-line width padding with backgrounds. Custom SideBySide already handles ANSI reset stripping and full-width background guarantee. Keep custom. |
| Per-row CalculateRowGeometry calls | Current block-level ComputeAlignedModel | CalculateRowGeometry is per-row, ComputeAlignedModel is per-document. Both use geometry.WrapText. Keep ComputeAlignedModel as the document-level orchestrator; CalculateRowGeometry for targeted tests. |

**Installation:**
```bash
# No new dependencies needed. Phase 2 uses only what Phase 1 established.
```

## Architecture Patterns

### Existing Project Structure (unchanged)
```
cmd/calcmark/tui/
├── geometry/             # Pure layout math (Phase 1) - NO CHANGES NEEDED
│   ├── geometry.go       # WrapText, CalculateRowGeometry, StringWidth
│   ├── geometry_test.go  # 23 table-driven tests
│   └── doc.go
├── editor/               # TUI editor (all Phase 2 changes here)
│   ├── aligned.go        # ComputeAlignedModel - alignment orchestrator
│   ├── linemodel.go      # LineModel (simpler model, also used)
│   ├── view.go           # View() render pipeline
│   ├── sidebyside.go     # SideBySide pane compositor
│   ├── model.go          # Model + Update + WindowSizeMsg handling
│   ├── markdown.go       # Glamour renderer for preview
│   ├── results.go        # LineResult bridge
│   ├── state.go          # Editor state machine
│   └── testdata/         # Catwalk test data files
├── components/           # Shared TUI components (status bar, etc.)
├── repl/                 # REPL model
└── shared/               # Shared types
```

### Pattern 1: Compute-Once Aligned Model
**What:** The `computeAlignedPanes` function in `view.go` computes the entire visual line structure ONCE per render cycle. Both `renderSourcePaneAligned` and `renderPreviewPaneAligned` consume this pre-computed data.
**When to use:** Every render cycle. This prevents reflow cycles where preview reflows cause padding changes which cause width changes which cause reflow.
**How it works:**
```
View() call
  ├── GetPaneWidths(totalWidth) → leftWidth, rightWidth
  ├── computeAlignedPanes(leftWidth, rightWidth) → alignedPanes (COMPUTE ONCE)
  ├── renderSourcePaneAligned(leftWidth, sourceContentHeight, aligned)
  └── renderPreviewPaneAligned(rightWidth, paneContentHeight, aligned)
```
**Critical invariant:** `len(aligned.sourceLines) == len(aligned.previewLines)` -- both panes always have the same number of visual lines.

### Pattern 2: Full-Width Background Guarantee (SideBySide)
**What:** The `SideBySide` renderer ensures every output line is exactly `leftWidth + rightWidth` characters with styled backgrounds on every character.
**When to use:** Final pane composition in `View()`.
**Why it matters:** Without this, terminal default background bleeds through on short lines, creating visible gaps between panes.
**Key implementation details:**
- Strips ANSI reset codes (`\x1b[0m`) before applying background -- resets clear all styling including backgrounds
- Pads both content and gap with background-styled spaces
- Balances line counts between panes (pads shorter pane with empty lines)

### Pattern 3: Visual-Line Scroll Synchronization
**What:** Both panes use the same scroll offset computed from `sourceToVisual` mapping. The scroll offset is stored as a source line index in `m.scrollOffset`, then converted to visual line index via the `sourceToVisual` map.
**When to use:** Both `renderSourcePaneAligned` and `renderPreviewPaneAligned` use identical scroll logic.
**Critical for alignment:** If the two panes scrolled independently, row N in source could show next to row M in preview.

### Pattern 4: Height Budget Accounting
**What:** The `View()` function carefully accounts for every pixel of terminal height:
- Total height: `m.height`
- Content height: `max(totalHeight - 6, 5)` (reserves: status bar 2 + context footer 2 + separator 1 + empty line 1)
- Pane content height: `max(contentHeight - 1, 3)` (minus header row)
- Source content height: `paneContentHeight - globalsHeight` (when preview visible)
**Why it matters:** Off-by-one errors here cause the last line to overflow the terminal, creating rendering artifacts.
**Warning:** Phase 1 already fixed a viewport height off-by-one (overhead calculation from 5 to 6). This area is fragile.

### Pattern 5: WindowSizeMsg Invalidation
**What:** On `tea.WindowSizeMsg`, the model updates `m.width` and `m.height` and calls `m.InvalidateAlignedCache()`. Next `View()` call recomputes everything with new dimensions.
**How it works (model.go line 282-285):**
```go
case tea.WindowSizeMsg:
    m.width = msg.Width
    m.height = msg.Height
    m.InvalidateAlignedCache()
```

### Anti-Patterns to Avoid
- **Computing alignment more than once per render:** The `computeAlignedPanes` call is expensive. Calling it in both `renderSourcePaneAligned` and `renderPreviewPaneAligned` would double the work and risk inconsistency.
- **Using lipgloss.Width in the geometry package:** Phase 1 established that geometry uses `runewidth.StringWidth` (plain text). Styled text width measurement uses `lipgloss.Width` in the editor package.
- **Mixing plain text and ANSI text in geometry functions:** Geometry functions receive plain text, never styled text. The `wrapStyledLine` function in `view.go` handles styled content separately by stripping ANSI, wrapping the plain text, and returning plain wrapped lines.
- **Independent scroll offsets for the two panes:** Both panes MUST use the same visual scroll offset, derived from the same `sourceToVisual` mapping, to maintain row alignment.

## Don't Hand-Roll

Problems that look simple but have existing solutions:

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Text wrapping with unicode | Custom width counting | `geometry.WrapText` (uses runewidth) | CJK double-width, emoji, combining chars. Already tested with 14 edge cases. |
| Per-row alignment of two columns | Manual max-height calculation | `geometry.CalculateRowGeometry` | Already computes height, pads both sides to same length. |
| Document-level alignment with blocks | Manual iteration over lines | `ComputeAlignedModel` in `aligned.go` | Handles block boundaries, calc vs text blocks, markdown rendering, bidirectional mappings. |
| Side-by-side rendering with backgrounds | `lipgloss.JoinHorizontal` | `SideBySide` in `sidebyside.go` | Handles ANSI reset stripping, per-line padding, background guarantee. |
| Styled text width | `len(string)` | `lipgloss.Width` | Accounts for ANSI escape codes, combining characters. |
| Styled content wrapping | Custom ANSI parser | `wrapStyledLine` in `view.go` | Strips ANSI, wraps plain text via geometry.WrapText, returns plain lines. |

**Key insight:** The entire two-column layout pipeline already exists. Phase 2 is about testing, hardening, and verifying -- not building from scratch.

## Common Pitfalls

### Pitfall 1: Edit Mode Alignment Mismatch
**What goes wrong:** When the user is typing (editBuf is active), the source pane renders the live edit buffer which may wrap differently than the pre-computed aligned lines. The preview pane must match.
**Why it happens:** The `renderSourcePaneAligned` function skips pre-computed wrapped lines for the cursor line and renders the edit buffer instead. The `renderPreviewPaneAligned` must emit the same number of visual lines for that logical row.
**How to avoid:** Both render functions already have edit-mode handling (view.go lines 314-316 for source, lines 718-771 for preview). Tests must verify alignment during active editing with wrapping content.
**Warning signs:** Preview showing extra blank lines or missing lines adjacent to the cursor during typing.

### Pitfall 2: Globals Panel Height Mismatch
**What goes wrong:** The preview pane has a collapsible globals panel at the top. The source pane adds equivalent padding lines. If the heights don't match exactly, all subsequent lines are misaligned.
**Why it happens:** Globals panel height calculation is duplicated: once in View() for source padding, once implicitly in renderPreviewPaneAligned.
**How to avoid:** The current code calculates `globalsHeight` once in View() and uses it consistently. Tests should verify alignment with globals both expanded and collapsed.
**Warning signs:** Source and preview content offset by 1-2 lines when globals panel is toggled.

### Pitfall 3: Terminal Resize Causing Stale Layout
**What goes wrong:** After resize, the first render uses old dimensions because the cache wasn't properly invalidated.
**Why it happens:** `InvalidateAlignedCache()` might not clear all state that depends on width/height.
**How to avoid:** The current WindowSizeMsg handler invalidates the cache. Integration tests should simulate resize with `tea.WindowSizeMsg{Width: newW, Height: newH}` and verify the view uses new dimensions.
**Warning signs:** Content still rendered at old width for one frame after resize, then snaps to new width.

### Pitfall 4: ANSI Escape Codes Breaking Width Calculation
**What goes wrong:** A styled string that is visually 10 chars wide might be 40 bytes due to ANSI codes. Functions that use `len()` instead of `lipgloss.Width()` will over-pad or under-pad.
**Why it happens:** Calc results and markdown preview content contain ANSI styling. The `renderCalcLine` function returns styled strings.
**How to avoid:** In the editor package, always use `lipgloss.Width()` for visual width of styled content. In the geometry package, always use `runewidth.StringWidth()` for plain text. Never cross these boundaries. The `wrapStyledLine` function in `view.go` strips ANSI before wrapping -- this is correct.
**Warning signs:** Right pane content overflowing into terminal margin, or left pane content bleeding into right pane.

### Pitfall 5: Off-By-One in Height Budget
**What goes wrong:** The content area renders one line too many or too few, causing terminal bleed-through (extra line at bottom) or wasted space (missing last line of content).
**Why it happens:** Multiple height reservations compound: status bar (2), context footer (2), separator (1), empty line (1), pane header (1), globals panel (variable). Getting any of these wrong cascades.
**How to avoid:** Test with specific terminal heights and count rendered lines. The test `TestViewportDoesNotExceedHeight` already checks this. Phase 2 should add tests with various terminal sizes (e.g., 24, 40, 80 rows).
**Warning signs:** View rendering `height + 1` lines, causing the last line to wrap onto the terminal's built-in bottom row.

### Pitfall 6: Ordered List Rendering Breaking Alignment
**What goes wrong:** Ordered lists (`1. item`) are rendered as a complete block by glamour (to get sequential numbering: 1, 2, 3). But distributing the rendered output back to individual source lines can produce wrong alignment.
**Why it happens:** The current code in `aligned.go` (lines 188-253) has special handling for ordered lists vs single-line rendering. The distribution algorithm divides rendered lines across source lines, but this can produce uneven results.
**How to avoid:** The wrapping bug fix (WRAPPING_BUG.md) already addressed this partially by rendering non-ordered-list text blocks line-by-line. For ordered lists, the block-level rendering with distribution is a known trade-off. Tests should cover ordered lists specifically.
**Warning signs:** Ordered list items showing on wrong lines in preview, or extra padding between list items.

## Code Examples

Verified patterns from the existing codebase:

### Full Render Pipeline (View function)
```go
// Source: cmd/calcmark/tui/editor/view.go (View function)
// This is the EXISTING pipeline -- Phase 2 tests verify it works correctly.

// 1. Calculate pane widths
leftWidth, rightWidth := m.GetPaneWidths(totalWidth)

// 2. Compute aligned model ONCE
aligned := m.computeAlignedPanes(leftWidth, rightWidth)

// 3. Render source pane
sourceContent := m.renderSourcePaneAligned(leftWidth, sourceContentHeight, aligned)

// 4. Render preview pane
previewContent := m.renderPreviewPaneAligned(rightWidth, paneContentHeight, aligned)

// 5. Compose side-by-side
sbs := NewSideBySide(leftWidth, rightWidth, ...)
panesOutput := sbs.Render(sourcePane, previewPane)
```

### Integration Test Pattern: Verify Alignment Invariant
```go
// Source: cmd/calcmark/tui/editor/aligned_test.go (existing pattern)
// Test that source and preview have same visual line count for any content.

model := ComputeAlignedModel(input, mockRenderCalcLine, mockRenderMarkdown)

// The key invariant
if len(model.SourceLines) != len(model.PreviewLines) {
    t.Errorf("Alignment broken: source=%d, preview=%d",
        len(model.SourceLines), len(model.PreviewLines))
}

// Verify bidirectional mappings
inv := model.Invariants()
if !inv.SourcePreviewMatch { t.Error("SourcePreviewMatch failed") }
if !inv.MappingComplete    { t.Error("MappingComplete failed") }
if !inv.ReverseComplete    { t.Error("ReverseComplete failed") }
```

### Catwalk Test Pattern: Verify View Output
```go
// Source: cmd/calcmark/tui/editor/catwalk_test.go (existing pattern)
// Run test data file with alignment observer

catwalk.RunModel(t, path, m,
    catwalk.WithObserver("alignment", func(out io.Writer, m tea.Model) error {
        model := m.(Model)
        leftWidth, rightWidth := model.GetPaneWidths(model.width)
        aligned := model.computeAlignedPanes(leftWidth, rightWidth)
        // Emit line-by-line alignment data for verification
        for i := 0; i < len(aligned.sourceLines); i++ {
            fmt.Fprintf(out, "[%d] SRC(ln=%d) | PRV(ln=%d)\n",
                i, aligned.sourceLines[i].lineNum, aligned.previewLines[i].sourceLineNum)
        }
        return nil
    }),
)
```

### WindowSizeMsg Test Pattern: Verify Resize
```go
// Source: cmd/calcmark/tui/editor/model_test.go (existing pattern)
// Simulate terminal resize and verify layout recalculation

updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
m = updated.(Model)

view := m.View()
lines := strings.Split(view, "\n")

// View should use new dimensions
// Each line should be exactly 120 chars wide (leftWidth + rightWidth)
for i, line := range lines {
    w := lipgloss.Width(line)
    if w != 120 {
        t.Errorf("Line %d width = %d, want 120", i, w)
    }
}
```

### CalculateRowGeometry Integration Point
```go
// Source: cmd/calcmark/tui/geometry/geometry.go
// This is available for per-row testing. ComputeAlignedModel uses WrapText
// directly but the same algorithm. For targeted success criteria tests:

geom := geometry.CalculateRowGeometry(
    "a very long source line that wraps to three visual lines in this width",
    "short result",
    30, // leftWidth
    30, // rightWidth
)

// geom.Height == 3 (source wraps to 3 lines)
// geom.LeftLines has 3 entries (wrapped source)
// geom.RightLines has 3 entries ("short result" + 2 empty padding lines)
// Left and right ALWAYS have same length == geom.Height
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `muesli/reflow/wordwrap` (code.sh) | `geometry.WrapText` with `runewidth` | Phase 1 (2026-02-02) | Same algorithm, zero TUI deps, handles CJK/emoji |
| `lipgloss.Width` in layout code | `runewidth.StringWidth` in geometry, `lipgloss.Width` in editor | Phase 1 | Clean separation: geometry=plain text, editor=styled text |
| Line-by-line text block rendering | Block-level rendering for ordered lists, line-level for everything else | Post-WRAPPING_BUG fix | Ordered lists get correct numbering; headings wrap correctly |
| No alignment invariant checking | `AlignedModel.Invariants()` method | Pre-Phase 1 (aligned.go) | Runtime validation that source/preview line counts match |

**Deprecated/outdated:**
- `TWO_PANE_DESIGN.md` recommends using two `textarea.Model` components -- this approach was explored in `model_v2.go` but the current `Model` (not ModelV2) with `ComputeAlignedModel` is the active approach. `model_v2.go` is a work-in-progress that is NOT the priority per user direction.
- `TEXTAREA_INTEGRATION_PLAN.md` and `TEXTAREA_INTEGRATION.md` describe the textarea approach -- these are informational but the current rendering pipeline in `view.go` is the active code.

## Open Questions

Things that could not be fully resolved:

1. **Should `CalculateRowGeometry` be directly called in the render path?**
   - What we know: `ComputeAlignedModel` calls `geometry.WrapText` directly and computes alignment inline. `CalculateRowGeometry` in the geometry package does the same per-row computation. They use the same algorithm but are separate code paths.
   - What's unclear: Whether to refactor `ComputeAlignedModel` to call `CalculateRowGeometry` per row, or keep the inline computation (which is identical but allows block-level awareness like ordered list rendering).
   - Recommendation: Keep `ComputeAlignedModel` as-is for the render path (it has block-level awareness that `CalculateRowGeometry` does not). Use `CalculateRowGeometry` in targeted unit tests to verify per-row geometry. This satisfies EDITOR-01 ("code.sh geometry algorithm implemented") since the algorithm IS implemented in both places.

2. **How to test "no rendering artifacts" on terminal resize?**
   - What we know: `tea.WindowSizeMsg` triggers cache invalidation and re-render. But we cannot see actual terminal artifacts in unit tests (no real terminal).
   - What's unclear: Whether there are edge cases where the first re-render after resize has stale data.
   - Recommendation: Test by sending WindowSizeMsg and verifying: (a) view line widths match new width, (b) line count matches new height budget, (c) alignment invariant holds. For visual artifacts, rely on manual VHS tape testing or TUI snapshot tests.

3. **ModelV2 convergence**
   - What we know: `model_v2.go` exists as a work-in-progress using `textarea.Model` from bubbles. The user stated high priority on the code.sh-based approach first. The current `Model` in `model.go` is the active implementation.
   - What's unclear: Whether Phase 2 should touch `model_v2.go` at all.
   - Recommendation: Phase 2 should NOT modify `model_v2.go`. Focus entirely on the existing `Model` and its render pipeline. ModelV2 is deferred per user direction.

4. **Edit buffer wrapping vs pre-computed wrapping mismatch**
   - What we know: During editing, the source pane renders the live `editBuf` with `geometry.WrapText`. The pre-computed `alignedPanes` used `input.Lines[cursorLine]` (the saved text, not the live buffer). This means the source pane may show different wrap count than what the aligned model computed.
   - What's unclear: Whether this causes visible misalignment during typing (before debounce saves the buffer).
   - Recommendation: The existing code handles this with special-case logic in both render functions (view.go lines 314-316 for source, 718-771 for preview). Tests should verify this works by typing content that wraps.

## Sources

### Primary (HIGH confidence)
- `cmd/calcmark/tui/geometry/geometry.go` -- Phase 1 geometry package (WrapText, CalculateRowGeometry, StringWidth)
- `cmd/calcmark/tui/editor/aligned.go` -- AlignedModel computation (ComputeAlignedModel, 519 lines)
- `cmd/calcmark/tui/editor/view.go` -- View() render pipeline (997 lines)
- `cmd/calcmark/tui/editor/sidebyside.go` -- SideBySide compositor (109 lines)
- `cmd/calcmark/tui/editor/model.go` -- Model definition, WindowSizeMsg handling, GetPaneWidths
- `cmd/calcmark/tui/editor/linemodel.go` -- LineModel computation (233 lines)
- `cmd/calcmark/tui/editor/markdown.go` -- Glamour renderer for preview (236 lines)
- `code.sh` -- Original reference implementation of the geometry algorithm
- `.planning/phases/01-foundation/01-02-SUMMARY.md` -- Phase 1 geometry extraction summary
- `.planning/phases/01-foundation/01-VERIFICATION.md` -- Phase 1 verification (3/4 truths verified)
- `cmd/calcmark/tui/editor/WRAPPING_BUG.md` -- Documentation of wrapping alignment fix
- Existing test files: `aligned_test.go`, `linemodel_test.go`, `view_alignment_test.go`, `catwalk_test.go`

### Secondary (MEDIUM confidence)
- `.planning/STATE.md` -- Project state with accumulated decisions
- `cmd/calcmark/tui/editor/TWO_PANE_DESIGN.md` -- Historical design doc (textarea approach, not active)
- `cmd/calcmark/tui/editor/TESTING.md` -- Catwalk testing guide

### Tertiary (LOW confidence)
- None. All findings verified from codebase analysis.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- all libraries already in go.mod and actively used in the codebase
- Architecture: HIGH -- based on direct reading of all render pipeline files (view.go, aligned.go, sidebyside.go, model.go)
- Pitfalls: HIGH -- based on existing bug fix documentation (WRAPPING_BUG.md), existing test failures documented in Phase 1 verification, and direct code inspection of edit-mode handling
- Success criteria mapping: HIGH -- all five success criteria directly map to existing code paths and testable behaviors

**Research date:** 2026-02-02
**Valid until:** 2026-03-04 (30 days -- stable domain, no external API changes expected)
