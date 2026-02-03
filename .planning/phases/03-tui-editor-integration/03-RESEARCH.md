# Phase 3: TUI Editor Integration - Research

**Researched:** 2026-02-03
**Domain:** Interactive TUI editor with cursor tracking, scrolling, debounced evaluation, and model unification
**Confidence:** HIGH

## Summary

Phase 3 transforms the Phase 2 layout infrastructure into a fully interactive editor. The current codebase has solid foundations: Bubble Tea for the TUI framework, an existing debounce pattern using `tea.Tick`, and an AlignedModel computation for line-by-line source/preview synchronization. The critical work is unifying the two model implementations (Model in `model.go` vs ModelV2 in `model_v2.go`), implementing the user-specified cursor behavior (logical-line navigation, left/right wrap to adjacent lines), and adding viewport scrolling that keeps both panes synchronized.

The existing `Model` in `model.go` already has most features needed: cursor position tracking (`cursorLine`, `cursorCol`), edit buffer management (`editBuf`), debounced evaluation with `evalDebounceMsg`, dirty state tracking (`modified`, `savedContent`), and quit confirmation (`StateSavePrompt`). The existing `ModelV2` in `model_v2.go` uses `textarea.Model` from bubbles but is incomplete and should be deprecated per user decision. The unification strategy is clear: rename `Model` to keep it as the primary, migrate any unique features from `ModelV2` if needed, then delete `ModelV2`.

**Primary recommendation:** Focus on cursor/viewport behavior hardening and the model unification. The debounce infrastructure exists; the geometry infrastructure exists from Phase 2. The work is: (1) implement the specified cursor movement behaviors (logical-line arrows, left/right wrapping), (2) add proper viewport scrolling with configurable margin, (3) verify evaluation pipeline with catwalk tests, (4) execute model unification by renaming and cleanup.

## Standard Stack

The established libraries/tools for this domain:

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| bubbletea | v1.3.10 | TUI framework (Model-Update-View, tea.Tick for debounce) | Already used. Provides tea.Cmd, tea.WindowSizeMsg, tea.KeyMsg. |
| bubbles/textarea | v0.21.0 | Optional: Provides LineInfo, cursor tracking | Used by ModelV2 but being deprecated. Useful for reference only. |
| lipgloss | v1.1.1 | Terminal styling, width measurement | Already used in view.go for all rendering. |
| geometry package | local | Pure text wrapping (WrapText, StringWidth) | Phase 1 foundation. Used by aligned.go for wrapping calculations. |
| catwalk | v0.1.4 | Data-driven TUI testing | Already used. Essential for testing cursor behavior, navigation, evaluation. |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| datadriven | v1.0.2 | Test framework for catwalk | Underlying runner for catwalk tests. |
| runewidth | v0.0.16 | Unicode width (CJK, emoji) | Used by geometry.WrapText for accurate cursor positioning. |
| termenv | v0.16.0 | Terminal capabilities | Used for color profile detection in tests. |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Custom cursor tracking | bubbles/textarea | textarea handles cursor in wrapped text but ModelV2 approach abandoned per user decision. Keep custom for precise control. |
| tea.Every for debounce | tea.Tick | tea.Every repeats; tea.Tick is one-shot. Debounce needs one-shot with snapshot comparison. Keep tea.Tick. |

**Installation:**
```bash
# No new dependencies needed. Phase 3 uses only existing libraries.
```

## Architecture Patterns

### Recommended Project Structure (unchanged from Phase 2)
```
cmd/calcmark/tui/
├── geometry/             # Pure layout math - NO CHANGES
│   └── geometry.go       # WrapText, CalculateRowGeometry, StringWidth
├── editor/               # All Phase 3 changes here
│   ├── model.go          # RENAME: Model (currently) -> keep as primary
│   ├── model_v2.go       # DELETE at end of Phase 3
│   ├── state.go          # EditorState machine (StateReady, StateEditing, StateProcessing)
│   ├── aligned.go        # AlignedModel computation (unchanged)
│   ├── view.go           # View() render pipeline (minor cursor adjustments)
│   ├── results.go        # LineResult bridge (unchanged)
│   └── testdata/         # Catwalk tests (ADD cursor, scrolling, eval tests)
└── shared/               # Shared types (keys.go has KeyMap)
```

### Pattern 1: Elm-Style State Machine (Existing)
**What:** The Model uses explicit state transitions via `transitionToReady()`, `transitionToEditing()`, `transitionToProcessing()`.
**When to use:** All state changes. Prevents invalid state combinations.
**Existing code (state.go):**
```go
// transitionToEditing moves the editor to StateEditing.
func (m *Model) transitionToEditing() {
    if m.state == StateEditing {
        return
    }
    if m.editBuf == "" {
        m.loadCurrentLineIntoEditBuffer()
    }
    m.userIsTyping = true
    m.state = StateEditing
}
```
**Phase 3 impact:** No changes needed. The state machine is already correct.

### Pattern 2: Debounce with Snapshot Comparison (Existing)
**What:** The existing debounce pattern uses `evalDebounceMsg` with a snapshot of `editBuf`. When the message fires, it only evaluates if the snapshot matches current `editBuf`.
**When to use:** After every keystroke that modifies content.
**Existing code (model.go):**
```go
type evalDebounceMsg struct {
    editBufSnapshot string
}

func (m Model) debounceUpdate() (tea.Model, tea.Cmd) {
    snapshot := m.editBuf
    return m, tea.Tick(evalDebounceDelay, func(t time.Time) tea.Msg {
        return evalDebounceMsg{editBufSnapshot: snapshot}
    })
}

// In Update():
case evalDebounceMsg:
    if m.editBuf == msg.editBufSnapshot {
        m.transitionToProcessing()  // Updates line, re-evaluates, transitions to ready
    }
```
**Phase 3 impact:** Change `evalDebounceDelay` from 50ms to ~100ms per user decision. Make configurable as constant.

### Pattern 3: Logical-Line Cursor Navigation (NEW)
**What:** Arrow keys move by logical lines, not visual lines within wrapped text.
**When to use:** Up/Down arrow key handling.
**Implementation:**
```go
func (m Model) handleUpKey() (tea.Model, tea.Cmd) {
    m.loadCurrentLineIntoEditBuffer()
    if m.cursorLine > 0 {
        m.saveCurrentLineAndMoveTo(m.cursorLine - 1)
        // cursorCol is preserved (clamped to line length in saveCurrentLineAndMoveTo)
    }
    return m, nil
}
```
**Key insight:** The existing code already does this. Verify with catwalk tests.

### Pattern 4: Left/Right Wrap to Adjacent Lines (NEW)
**What:** Left arrow at column 0 moves to end of previous line. Right arrow at end of line moves to start of next line.
**When to use:** Left/Right arrow key handling.
**Existing code (model.go) - already implements this:**
```go
func (m Model) handleLeftKey() (tea.Model, tea.Cmd) {
    m.loadCurrentLineIntoEditBuffer()
    if m.cursorCol > 0 {
        m.cursorCol--
    } else if m.cursorLine > 0 {
        // At start of line - move to end of previous line
        m.saveCurrentLineAndMoveTo(m.cursorLine - 1)
        m.cursorCol = len(m.editBuf)
    }
    return m, nil
}
```
**Phase 3 impact:** Already implemented. Verify with catwalk tests.

### Pattern 5: Viewport Scrolling with Margin (ENHANCE)
**What:** When cursor goes off-screen, adjust scroll offset to keep cursor visible with a margin.
**When to use:** After any cursor movement.
**Current code (model.go moveCursor):**
```go
visibleHeight := m.height - 6
if m.cursorLine < m.scrollOffset {
    m.scrollOffset = m.cursorLine
}
if m.cursorLine >= m.scrollOffset+visibleHeight {
    m.scrollOffset = m.cursorLine - visibleHeight + 1
}
```
**Enhancement needed:** Convert to visual-line space (use `sourceToVisual` mapping) and add configurable margin.
**Recommended implementation:**
```go
const scrollMargin = 3  // Keep cursor 3 lines from edge (Claude's discretion)

func (m *Model) adjustScrollForCursor(aligned *AlignedModel) {
    cursorVisual := aligned.CursorVisualLine(m.cursorLine)
    if cursorVisual < 0 {
        return
    }

    visibleHeight := m.getVisibleHeight()

    // Scroll up if cursor too close to top
    if cursorVisual < m.scrollOffset + scrollMargin {
        m.scrollOffset = max(0, cursorVisual - scrollMargin)
    }

    // Scroll down if cursor too close to bottom
    if cursorVisual >= m.scrollOffset + visibleHeight - scrollMargin {
        m.scrollOffset = cursorVisual - visibleHeight + scrollMargin + 1
    }
}
```

### Pattern 6: Line Split on Enter (Existing)
**What:** Enter key splits the current line at cursor position.
**Existing code (model.go handleEnterKey):**
```go
func (m Model) handleEnterKey() (tea.Model, tea.Cmd) {
    m.loadCurrentLineIntoEditBuffer()
    textBefore := m.editBuf[:m.cursorCol]
    textAfter := m.editBuf[m.cursorCol:]

    m.editBuf = textBefore
    m.updateCurrentLine(m.editBuf)
    m.insertLineBelow()
    m.editBuf = textAfter
    m.cursorCol = 0

    m.redetectBlockTypes()
    m.reEvaluate()
    m.pushUndoState()
    m.modified = true
    m.userIsTyping = false
    return m, nil
}
```
**Phase 3 impact:** Already implemented correctly. Verify with catwalk tests.

### Anti-Patterns to Avoid
- **Visual-line cursor movement:** Do NOT implement cursor moving through visual wrapped lines. The user decided: down arrow jumps to next logical line.
- **Mutable state access:** The user wants methods-only state access. Never access `m.cursorLine` directly in new code; use/create getter methods.
- **Auto-save:** User specified manual save only (Ctrl+S). No auto-save implementation.
- **Esc for quit:** User specified Ctrl+Q for quit, NOT Esc. Esc does nothing in normal mode.

## Don't Hand-Roll

Problems that look simple but have existing solutions:

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Debounce timer | Manual goroutine/channel | `tea.Tick` with snapshot pattern | Already implemented in model.go. Race-free, integrates with Elm architecture. |
| Unicode cursor positioning | `len(string)` | `runewidth.StringWidth` | CJK double-width, emoji, combining chars. Already used in geometry package. |
| Line wrapping | Custom word-break | `geometry.WrapText` | Handles all unicode edge cases. Phase 1 foundation. |
| Visual-to-source mapping | Custom tracking | `AlignedModel.SourceToVisual` | Already computed in aligned.go. |
| Styled text width | `len(string)` | `lipgloss.Width` | Accounts for ANSI escape codes. |
| Key handling | Manual key parsing | `tea.KeyMsg.Type` and `tea.KeyMsg.String()` | Bubble Tea handles terminal escape sequences. |

**Key insight:** Phase 3 builds on extensive existing infrastructure. The work is integration and hardening, not building from scratch.

## Common Pitfalls

### Pitfall 1: Source-Line vs Visual-Line Scroll Mismatch
**What goes wrong:** Scroll offset is in source-line space but viewport operates in visual-line space. Wrapped lines cause misalignment.
**Why it happens:** The existing code stores `m.scrollOffset` as a source line index. When source line N wraps to 3 visual lines, scrolling by 1 source line can jump 3 visual lines.
**How to avoid:** Always convert scroll offset to visual-line space using `aligned.sourceToVisual[m.scrollOffset]` before applying to viewport. The existing `renderSourcePaneAligned` and `renderPreviewPaneAligned` already do this.
**Warning signs:** After scrolling, cursor appears at unexpected position or off-screen.

### Pitfall 2: EditBuf Not Synced During Navigation
**What goes wrong:** User types "abc", presses Down without triggering debounce, and "abc" is lost.
**Why it happens:** The debounce timer hasn't fired yet, so `editBuf` content isn't saved to document.
**How to avoid:** The existing `saveCurrentLineAndMoveTo` function explicitly saves `editBuf` before moving. This is correct. Verify with catwalk test.
**Warning signs:** Content disappears after rapid navigation during typing.

### Pitfall 3: Cursor Column Out of Bounds After Line Change
**What goes wrong:** Cursor at column 50 on a long line moves to a short line (10 chars). Cursor column stays at 50, causing rendering issues.
**Why it happens:** Column isn't clamped to new line length.
**How to avoid:** The existing `saveCurrentLineAndMoveTo` clamps column to line length:
```go
if savedCol > len(m.editBuf) {
    m.cursorCol = len(m.editBuf)
} else {
    m.cursorCol = savedCol
}
```
**Warning signs:** Cursor renders past end of line, or causes index-out-of-bounds.

### Pitfall 4: Preview Not Updating After Evaluation
**What goes wrong:** User types `x = 10`, debounce fires, but preview still shows old value.
**Why it happens:** Aligned model cache not invalidated after re-evaluation.
**How to avoid:** The existing `transitionToProcessing` calls `redetectBlockTypes()` and `reEvaluate()`, but doesn't explicitly invalidate cache. Add `m.InvalidateAlignedCache()` in `transitionToProcessing`.
**Warning signs:** Preview shows stale values until window resize.

### Pitfall 5: Quit Without Save Prompt Bypassed
**What goes wrong:** User with unsaved changes presses Ctrl+C and loses work.
**Why it happens:** Ctrl+C is hardcoded to quit immediately (Unix signal behavior).
**How to avoid:** This is intentional per existing code - Ctrl+C is emergency exit. Ctrl+Q shows save prompt. Document this behavior clearly.
**Warning signs:** User surprise at lost work (but this is standard Unix behavior).

### Pitfall 6: Delete Key Joining Lines Unexpectedly
**What goes wrong:** Delete at end of line joins with next line when user expected no action.
**Why it happens:** Delete key behavior not clearly specified.
**How to avoid:** User left this to Claude's discretion. Recommendation: Delete at end of non-empty line does nothing. Delete on empty line removes the line.
**Warning signs:** Unexpected line merging.

## Code Examples

Verified patterns from the existing codebase:

### Existing Debounce Implementation
```go
// Source: cmd/calcmark/tui/editor/model.go
const evalDebounceDelay = 50 * time.Millisecond  // CHANGE TO 100ms

type evalDebounceMsg struct {
    editBufSnapshot string
}

func (m Model) debounceUpdate() (tea.Model, tea.Cmd) {
    snapshot := m.editBuf
    return m, tea.Tick(evalDebounceDelay, func(t time.Time) tea.Msg {
        return evalDebounceMsg{editBufSnapshot: snapshot}
    })
}

// In Update():
case evalDebounceMsg:
    if m.editBuf == msg.editBufSnapshot {
        m.transitionToProcessing()
    }
```

### Existing Save Prompt Implementation
```go
// Source: cmd/calcmark/tui/editor/model.go
case tea.KeyCtrlQ:
    if m.hasUnsavedChanges() {
        m.mode = StateSavePrompt
        m.statusMsg = "Unsaved changes! Save before quit? (y/n/c)"
        return m, nil
    }
    m.quitting = true
    return m, tea.Quit
```

### Existing Dirty State Detection
```go
// Source: cmd/calcmark/tui/editor/model.go
func (m *Model) hasUnsavedChanges() bool {
    currentContent := m.getDocumentContent()
    return currentContent != m.savedContent
}
```

### Catwalk Test Pattern for Cursor Navigation
```
# Source: testdata pattern
# Test: Down arrow moves to next logical line
run observe=debug
key down
----
-- debug:
mode=0 cursorLine=1 cursorCol=0 ...
```

### Catwalk Test Pattern for Evaluation
```
# Test: Typing triggers debounced evaluation
run observe=debug
type x = 5
----
-- debug:
mode=0 cursorLine=0 cursorCol=5 editBuf="x = 5" ...

# Wait for debounce (use separate run block)
run observe=results
----
-- results:
Line 0 (x = 5): value=5, error=""
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| ModelV2 with textarea | Model with custom cursor | Phase 3 decision | Simpler, more control, better testing |
| 50ms debounce | 100ms debounce | Phase 3 decision | More conservative, configurable |
| Esc to quit | Ctrl+Q to quit | Phase 3 decision | Standard TUI pattern, save prompt |
| No dirty tracking | savedContent comparison | Already in model.go | Reliable unsaved detection |

**Deprecated/outdated:**
- `model_v2.go`: To be deleted at end of Phase 3. Do not add features to it.
- `TWO_PANE_DESIGN.md`: Historical textarea approach. Informational only.
- `TEXTAREA_INTEGRATION*.md`: Historical. The textarea approach is abandoned.

## Open Questions

Things that couldn't be fully resolved:

1. **Word boundary definition for Ctrl+Arrow**
   - What we know: User decided standard word boundaries (whitespace/punctuation as separators)
   - What's unclear: Exact punctuation set. CalcMark has `=`, `+`, `-`, `*`, `/`, `%`, etc. Should these all be word boundaries?
   - Recommendation: Use Go's `unicode.IsSpace` and `unicode.IsPunct` for boundaries. Test with catwalk.

2. **Page Up/Down implementation**
   - What we know: User left to Claude's discretion. Current code has basic implementation in `handlePageUpKey`/`handlePageDownKey`.
   - What's unclear: Should it move by visual lines or source lines? How much margin?
   - Recommendation: Move by `height - 4` source lines (current implementation). Keep cursor visible with margin.

3. **Delete key behavior at end of line**
   - What we know: User left to Claude's discretion.
   - What's unclear: Should Delete at end of line join with next line (like some editors)?
   - Recommendation: Delete at end of non-empty line does nothing. Delete on empty line removes the line. This matches the "empty line backspace joins" behavior symmetrically.

4. **Word deletion (Ctrl+Backspace/Delete)**
   - What we know: User left to Claude's discretion given complexity.
   - What's unclear: CalcMark expressions like `tax * price` have semantic word boundaries (variable names) vs syntactic (whitespace/punct).
   - Recommendation: Implement standard word deletion (stop at whitespace/punctuation). Don't try to be CalcMark-aware. Keep simple for Phase 3.

## Sources

### Primary (HIGH confidence)
- `cmd/calcmark/tui/editor/model.go` -- Primary editor model (2003 lines, all cursor/edit/save/quit logic)
- `cmd/calcmark/tui/editor/model_v2.go` -- Secondary model to be deprecated (665 lines)
- `cmd/calcmark/tui/editor/state.go` -- EditorState machine (140 lines)
- `cmd/calcmark/tui/editor/view.go` -- View render pipeline (1006 lines)
- `cmd/calcmark/tui/editor/aligned.go` -- AlignedModel computation (519 lines)
- `cmd/calcmark/tui/editor/results.go` -- LineResult bridge (211 lines)
- `cmd/calcmark/tui/editor/TESTING.md` -- Catwalk testing documentation
- `cmd/calcmark/tui/geometry/geometry.go` -- Pure geometry functions (121 lines)
- `go.mod` -- Dependency versions (bubbletea v1.3.10, bubbles v0.21.0, etc.)
- `.planning/phases/03-tui-editor-integration/03-CONTEXT.md` -- User decisions

### Secondary (MEDIUM confidence)
- [Bubble Tea GitHub](https://github.com/charmbracelet/bubbletea) -- Elm architecture patterns
- [Bubble Tea v2 Cursor PR](https://github.com/charmbracelet/bubbletea/pull/1303) -- Future cursor API (not in v1.3.10)
- [bubbles/textarea docs](https://pkg.go.dev/github.com/charmbracelet/bubbles/textarea) -- LineInfo struct reference

### Tertiary (LOW confidence)
- None. All findings verified from codebase analysis.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- all libraries already in go.mod and actively used
- Architecture: HIGH -- based on direct reading of model.go, view.go, state.go, aligned.go
- Pitfalls: HIGH -- based on existing bug fix patterns and codebase analysis
- Cursor/scroll behavior: HIGH -- existing implementations found and analyzed
- Debounce pattern: HIGH -- existing implementation verified in model.go

**Research date:** 2026-02-03
**Valid until:** 2026-03-05 (30 days -- stable domain, no external API changes expected)
