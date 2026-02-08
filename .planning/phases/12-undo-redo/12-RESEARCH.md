# Phase 12: Undo/Redo - Research

**Researched:** 2026-02-07
**Domain:** Text editor undo/redo with operation-based history and natural grouping
**Confidence:** HIGH

## Summary

This research covers implementing undo/redo functionality for the go-calcmark TUI editor with the following locked decisions from CONTEXT.md:

1. **Natural boundaries for grouping** - Group edits until pause (1-2 seconds) or navigation
2. **Operation-based diffs** - Store operations, not full document snapshots
3. **Memory-only storage** - No swap files, max 2x file size budget
4. **500-1000 state limit** - Circular buffer, drop oldest silently

The current implementation uses full document snapshots (`undoStack []string`) with a 100-state limit. This must be replaced with an operation-based system that captures individual edit operations and groups them intelligently.

**Primary recommendation:** Implement a Command pattern with edit operations (insert, delete, replace) that store only the delta, using a timer-based grouping mechanism (1000ms) that coalesces consecutive same-type operations. Use a circular buffer for history management.

## Standard Stack

This is a pure Go implementation with no external dependencies needed.

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Standard library only | Go 1.24+ | All implementation | No dependencies needed for undo/redo logic |
| time.Timer | stdlib | Debounce/grouping | Already used for eval debounce |
| container/ring | stdlib | Circular buffer option | Built-in, but custom is simpler |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| bubbletea tea.Tick | existing | Timer messages | Already in use for debounce |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Custom circular buffer | container/ring | container/ring uses Value interface{}, custom is type-safe |
| Command pattern | Memento pattern | Memento uses more memory (full snapshots), command is operation-based |
| github.com/niklabh/undo-redo | Custom | External lib adds dependency, custom fits exact needs |

**Installation:**
No new packages needed. Pure Go implementation using existing dependencies.

## Architecture Patterns

### Recommended Project Structure

```
cmd/calcmark/tui/editor/
├── model.go           # Existing - add undo keybindings
├── undo.go            # NEW - UndoManager, EditOperation types
├── undo_test.go       # NEW - Unit tests for undo logic
└── testdata/
    └── undo           # NEW - Catwalk tests for undo/redo flows
```

### Pattern 1: Operation-Based Undo (Command Pattern Variant)

**What:** Each edit is captured as an operation with position, deleted content, and inserted content. Operations can be applied forward (redo) or reversed (undo).

**When to use:** Always - this is the locked decision from CONTEXT.md.

**Example:**
```go
// EditOperation represents a single atomic edit
type EditOperation struct {
    Type       OpType    // OpInsert, OpDelete, OpReplace
    Line       int       // Line where edit occurred
    Col        int       // Column where edit occurred
    OldText    string    // Text that was removed (empty for insert)
    NewText    string    // Text that was inserted (empty for delete)
    CursorLine int       // Cursor position before edit (for restoration)
    CursorCol  int       // Cursor column before edit
}

// Apply applies the operation (for redo)
func (op *EditOperation) Apply(doc *Document) {
    // Replace OldText with NewText at position
}

// Reverse reverses the operation (for undo)
func (op *EditOperation) Reverse(doc *Document) {
    // Replace NewText with OldText at position
}
```

### Pattern 2: Timer-Based Grouping

**What:** Use a timer (1000-2000ms) to detect typing pauses. Consecutive edits within the timer window are grouped into a single undo batch.

**When to use:** For continuous typing - characters typed rapidly become one undo step.

**Example:**
```go
// UndoBatch groups operations that should undo together
type UndoBatch struct {
    Operations []EditOperation  // All operations in this batch
    Timestamp  time.Time        // When batch was finalized
}

// UndoManager manages undo/redo history with grouping
type UndoManager struct {
    history     []UndoBatch  // Circular buffer of batches
    redoStack   []UndoBatch  // Stack of undone batches
    current     []EditOperation  // Operations in progress (not yet committed)
    lastEdit    time.Time    // Time of last edit
    groupTimer  *time.Timer  // Timer for grouping
    maxHistory  int          // Max batches to keep (500-1000)
    head        int          // Ring buffer head
    size        int          // Current size
}

const (
    GroupingDelay = 1000 * time.Millisecond  // 1 second pause creates boundary
)
```

### Pattern 3: Circular Buffer for History

**What:** Fixed-size ring buffer that overwrites oldest entries when full. Provides O(1) push and automatic memory management.

**When to use:** For the main history storage to enforce the 500-1000 state limit.

**Example:**
```go
// Simple circular buffer implementation
type HistoryBuffer struct {
    items    []UndoBatch
    head     int   // Next write position
    size     int   // Current number of items
    capacity int   // Max capacity
}

func (h *HistoryBuffer) Push(batch UndoBatch) {
    h.items[h.head] = batch
    h.head = (h.head + 1) % h.capacity
    if h.size < h.capacity {
        h.size++
    }
}

func (h *HistoryBuffer) Pop() (UndoBatch, bool) {
    if h.size == 0 {
        return UndoBatch{}, false
    }
    h.size--
    idx := (h.head - 1 + h.capacity) % h.capacity
    h.head = idx
    return h.items[idx], true
}
```

### Anti-Patterns to Avoid

- **Full document snapshots:** Storing entire document state for each edit. Uses O(n*m) memory where n=doc size, m=edits. CONTEXT.md explicitly prohibits this.

- **No grouping:** Creating undo step for every keystroke. Users expect typing to undo as a unit. Makes undo unusable.

- **Sync timers:** Using time.Sleep or blocking timers. Must use async tea.Tick pattern already in codebase.

- **Unbounded history:** Growing history without limit. Must use circular buffer with 500-1000 limit per CONTEXT.md.

## Don't Hand-Roll

Problems that look simple but have existing solutions in the codebase:

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Timer messages | Manual time.After | tea.Tick pattern | Already used for evalDebounceDelay, consistent with Bubble Tea |
| Key bindings | Raw key string matching | key.Matches with KeyMap | Ctrl+Z/Y already defined in shared/keys.go |
| Document manipulation | Direct string editing | Existing model methods | updateCurrentLine, insertLine, deleteLine already work |

**Key insight:** The existing editor already has debounce infrastructure (evalDebounceMsg, tea.Tick). The undo grouping timer can follow the exact same pattern.

## Common Pitfalls

### Pitfall 1: Not Saving State Before First Edit

**What goes wrong:** First undo does nothing because there's no "before" state to restore to.

**Why it happens:** Only saving state after edits means the initial document state is lost.

**How to avoid:** Save initial document state when file is opened. Current code does this in `New()` with `m.pushUndoState()` - ensure this is preserved.

**Warning signs:** Undo immediately after opening file has no effect.

### Pitfall 2: Cursor Not Restored

**What goes wrong:** Undo restores text but cursor is in wrong position, confusing user.

**Why it happens:** Only storing document content, not cursor position with each operation.

**How to avoid:** Store (cursorLine, cursorCol) with each EditOperation. On undo, restore cursor to position BEFORE the operation was applied.

**Warning signs:** After undo, cursor is at document end or random position.

### Pitfall 3: Timer Race with Document Saves

**What goes wrong:** Grouping timer fires after user has already committed an undo batch manually (via Enter or navigation).

**Why it happens:** Timer is async, can fire after manual boundary.

**How to avoid:** Cancel pending grouping timer when manual boundary occurs (Enter, arrow keys). Clear `current` operations when committing batch.

**Warning signs:** Same edit appears in multiple undo batches.

### Pitfall 4: editBuf Not Flushed Before Undo

**What goes wrong:** User types "abc", presses undo, but "abc" is still in editBuf and gets saved.

**Why it happens:** Editor has editBuf for in-progress line edits. Must flush to document before capturing undo state.

**How to avoid:** Call `transitionToProcessing()` or similar to flush editBuf before any undo operation recording.

**Warning signs:** Undo seems to do nothing, or partially works.

### Pitfall 5: Redo Stack Not Cleared

**What goes wrong:** User undoes, types new text, then redo brings back old content unexpectedly.

**Why it happens:** New edits should clear redo stack (standard behavior), but implementation forgot.

**How to avoid:** In `RecordEdit()`, clear redoStack. Per CONTEXT.md discretion: use standard behavior of clearing redo on new edit.

**Warning signs:** Redo after new edit restores overwritten content.

## Code Examples

### Integration with Existing Debounce Pattern

```go
// Follow existing evalDebounceMsg pattern from model.go
const undoGroupingDelay = 1000 * time.Millisecond

// undoGroupMsg is sent after grouping delay to commit pending edits
type undoGroupMsg struct {
    batchID int  // Unique ID to prevent stale commits
}

// After any edit operation:
func (m *Model) recordEdit(op EditOperation) tea.Cmd {
    m.undoManager.AddOperation(op)
    m.undoGroupID++  // Increment to invalidate pending timers

    return tea.Tick(undoGroupingDelay, func(t time.Time) tea.Msg {
        return undoGroupMsg{batchID: m.undoGroupID}
    })
}

// In Update():
case undoGroupMsg:
    if msg.batchID == m.undoGroupID {
        m.undoManager.CommitCurrentBatch()
    }
    // If batchID doesn't match, timer was stale - ignore
```

### Minimal EditOperation Implementation

```go
type OpType int

const (
    OpInsert OpType = iota
    OpDelete
    OpReplace  // Sugar for delete+insert
)

type EditOperation struct {
    Type       OpType
    Line       int
    Col        int
    OldText    string  // What was there before (empty for pure insert)
    NewText    string  // What is there now (empty for pure delete)
    CursorLine int     // Cursor before operation
    CursorCol  int
    ScrollOffset int   // Scroll offset before operation
}

// Memory estimation: ~100 bytes base + len(OldText) + len(NewText)
// Typing "hello": 5 operations, ~500 bytes
// Pasting 1KB: 1 operation, ~1100 bytes
```

### Undo/Redo Handler in handleDefaultKey

```go
// In handleDefaultKey, add cases:
case tea.KeyCtrlZ:
    return m.handleUndo()
case tea.KeyCtrlY:
    return m.handleRedo()

func (m Model) handleUndo() (tea.Model, tea.Cmd) {
    // Flush any pending edits to document
    m.transitionToProcessing()

    // Commit current batch before undoing
    m.undoManager.CommitCurrentBatch()

    // Perform undo
    batch, ok := m.undoManager.Undo()
    if !ok {
        m.statusMsg = "Nothing to undo"
        return m, nil
    }

    // Apply operations in reverse order
    for i := len(batch.Operations) - 1; i >= 0; i-- {
        m.applyOperationReverse(batch.Operations[i])
    }

    // Restore cursor from first (chronologically last) operation
    if len(batch.Operations) > 0 {
        op := batch.Operations[0]
        m.cursorLine = op.CursorLine
        m.cursorCol = op.CursorCol
        m.scrollOffset = op.ScrollOffset
    }

    m.statusMsg = "Undo"
    return m, nil
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Full snapshots | Operation-based diffs | Industry standard since 1990s | 10-100x memory reduction |
| Keystroke-level undo | Semantic grouping | Standard since ~2000 | Much better UX |
| Unlimited history | Bounded circular buffer | Always for production | Predictable memory |

**Deprecated/outdated:**
- Storing full document copies: Only acceptable for tiny documents. For 5MB files with 1000 edits, this would use 5GB.

## Open Questions

Things that couldn't be fully resolved:

1. **Exact grouping timer value**
   - What we know: CONTEXT.md says "1-2 seconds"
   - What's unclear: Exact milliseconds (1000? 1500? 2000?)
   - Recommendation: Start with 1000ms, match existing debounce feel

2. **Navigation boundary behavior**
   - What we know: CONTEXT.md says "Claude's discretion"
   - What's unclear: Should cursor movement without editing create boundary?
   - Recommendation: Yes for arrow keys (user is reviewing), no for scroll (user is reading)

3. **Integration with existing undo()/redo() methods**
   - What we know: model.go has undo()/redo() methods using old snapshot approach
   - What's unclear: Replace completely vs refactor
   - Recommendation: Replace with new UndoManager, remove undoStack/redoStack fields

## Sources

### Primary (HIGH confidence)

- **Codebase analysis** - model.go, state.go, keys.go - verified current implementation
- **CONTEXT.md** - Phase decisions (locked choices)
- **REQUIREMENTS.md** - UNDO-01 through UNDO-05 requirements

### Secondary (MEDIUM confidence)

- [goki.dev/gi/v2/undo](https://pkg.go.dev/goki.dev/gi/v2/undo) - Go undo package with diff-based storage and SaveReplace for coalescing
- [JS-UndoManager](https://github.com/wvteijlingen/JS-UndoManager) - Reference implementation with grouping and coalescing
- [Text Editor Data Structures: Rethinking Undo](https://cdacamar.github.io/data%20structures/algorithms/benchmarking/text%20editors/c++/rethinking-undo/) - Myers diff for undo
- [Redux-undo](https://redux-undo.js.org/) - groupBy function for batch actions

### Tertiary (LOW confidence)

- [niklabh/undo-redo](https://github.com/niklabh/undo-redo) - Simple Go library (interface ideas)
- [Command Pattern in Go](https://refactoring.guru/design-patterns/command/go/example) - Pattern structure

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - No external dependencies needed, all Go stdlib + existing deps
- Architecture: HIGH - Command pattern well-established, circular buffer straightforward
- Pitfalls: HIGH - Common issues well-documented in text editor literature
- Grouping algorithm: MEDIUM - Timer values are tunable, exact behavior may need adjustment

**Research date:** 2026-02-07
**Valid until:** 2026-03-07 (30 days - stable domain, no rapid changes expected)
