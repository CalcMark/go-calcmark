---
title: Pure Functional Layout Calculations for Dynamic UI Components
category: code-organization
date: 2026-03-12
tags: [tui, layout, bubbletea, height-calculation, scroll, pure-functions]
components: [editor/view.go, editor/navigation.go, components/contextfooter.go]
---

# Pure Functional Layout Calculations for Dynamic UI Components

## Problem

When adding a dynamically-sized TUI component (a context footer that expands from 2 to 4 lines on error lines), the height calculation must be **identical** everywhere it's used — `View()` for rendering and `getVisibleHeight()` for scroll/cursor positioning. If these disagree, the cursor can scroll off-screen, the view produces wrong line counts, or bubbletea renders ghost lines.

Early attempts introduced bugs:
- A hardcoded pessimistic constant (`8`) in `getVisibleHeight()` broke scroll tests (expected `visibleHeight=10`, got `8`).
- Subtracting a header row in `getVisibleHeight()` but not in `View()` caused an off-by-one (`visibleHeight=9` vs `10`).

## Root Cause

Height calculations lived in two places with subtly different formulas. Any divergence — even by one line — produces visible rendering artifacts in bubbletea's full-repaint model.

## Solution

Extract a **single pure helper** that both `View()` and `getVisibleHeight()` call:

```go
// contextFooterHeight computes the dynamic footer height from state.
// Pure: reads only cursorLine, mode, autocompleteState, and line results.
func (m Model) contextFooterHeight(results []LineResult) int {
    if m.mode == StateAutocomplete && m.autocompleteState.Visible {
        return components.ContextFooterHeight // 2
    }
    if m.cursorLine < len(results) {
        r := results[m.cursorLine]
        if r.Error != "" && !r.IsBlocked {
            hint := /* ... get hint from diagnostic or parsed error ... */
            if hint != "" {
                return min(4, 2+countWrappedLines(hint, m.width-4))
            }
        }
    }
    return components.ContextFooterHeight // 2
}
```

Both consumers use the **same formula**:

```go
// In View():
results := m.GetLineResults()
footerHeight := m.contextFooterHeight(results)
contentHeight := max(totalHeight-components.StatusBarHeight-footerHeight-2, 5)

// In getVisibleHeight() (used by scroll system):
func (m *Model) getVisibleHeight() int {
    results := m.GetLineResults()
    footerHeight := m.contextFooterHeight(results)
    return max(m.height-components.StatusBarHeight-footerHeight-2, 5)
}
```

### Key Principles

1. **Single source of truth** — One helper, called by both renderer and scroll system. Never duplicate the formula.

2. **Compute before render** — Height is determined from state *before* any rendering begins. Never lazily compute height inside a render function.

3. **State flows down** — `GetLineResults()` is computed once per frame in `View()` and passed to all sub-renderers. The height helper takes results as a parameter rather than re-computing them.

4. **Constants are named** — `components.ContextFooterHeight = 2` is the default. The `2` in the formula accounts for separator + empty line. The `5` is the minimum content height.

5. **Pure renderers accept maxHeight** — `RenderContextFooter(state, width, bg, maxHeight)` always returns exactly `maxHeight` lines via `padToHeight`. This makes the output deterministic regardless of content.

## Prevention

- When adding any component that affects layout height, ensure the height calculation is a **single extracted helper** called by both the renderer and the scroll system.
- Never hardcode height constants that should be derived from state. If a component's height varies, compute it — don't approximate.
- The formula `max(totalHeight - statusBar - footer - separators, minContent)` should appear in exactly ONE place. Everything else calls that place.
- Test by rendering `View()` and counting output lines (see `TestViewHasCorrectLineCountAlways`) — this catches formula divergence that unit tests on individual helpers would miss.
