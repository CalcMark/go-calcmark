# Phase 12: Undo/Redo - Context

## Phase Goal

Full history with cursor position restoration.

---

## Decisions

*Locked choices from /gsd:discuss-phase - Claude MUST honor these exactly.*

### History Granularity

| Decision | Choice |
|----------|--------|
| What counts as one undo step | Natural boundaries (group until pause/navigation) |
| Pause duration for boundary | 1-2 seconds |
| Navigation creates boundary | Claude's discretion |
| Delete/Backspace with selection | Depends on selection (with selection = 1 step) |
| Enter (new line) | Always creates undo boundary |
| Line joins (Delete at EOL/Backspace at BOL) | Separate step |
| Paste operations | One step regardless of content size |
| Cut (Ctrl+X) restoration | Claude's discretion |

### State Capture

| Decision | Choice |
|----------|--------|
| What to restore | Claude's discretion |
| Storage approach | Diffs/operations (not full snapshots) |
| Redo stack on new edit | Claude's discretion |
| Undo across saves | Claude's discretion |

### Undo Boundaries & Limits

| Decision | Choice |
|----------|--------|
| Storage mechanism | Memory-only (no swap files) |
| Max history size | Soft limit ~500-1000 states |
| At limit behavior | Drop oldest states silently |
| On file open | Empty history (fresh start) |
| Document size limit | 1-5MB max |
| Memory budget | Max 2x file size for state management |

### Security Considerations

Cross-cutting concerns captured during discussion (may warrant separate phase):

- **Fuzzing tests** - Add fuzz testing for interpreter input validation
- **Magic bytes verification** - Validate UTF-8 text file, reject binary/non-text
- **Early exit for large files** - Check size limit before full load
- **Unknown file size handling** - Handle network streams, pipes without known size

---

## Claude's Discretion

*Areas where Claude makes implementation choices.*

- Navigation creating undo boundary (recommendation: yes for arrow keys, no for scroll)
- Cut restoration approach (recommendation: treat like delete, restore content)
- What state to restore beyond cursor (recommendation: cursor position + scroll offset)
- Redo stack behavior on new edit (recommendation: clear redo stack, standard behavior)
- Undo across saves (recommendation: preserve history, save doesn't affect undo)
- Exact grouping algorithm for natural boundaries
- Internal data structures for diff storage

---

## Deferred Ideas

*Out of scope for Phase 12 - do NOT include.*

- File-based undo persistence (swap files)
- Undo history surviving app restart
- Collaborative editing undo semantics
- Visual undo history browser
- Fuzzing/security hardening (separate phase if needed)

---

## Technical Notes

### Memory Budget Constraint

User specified: File up to 5MB, memory use no more than 2x (10MB total) for state management.

This strongly favors diff-based storage over snapshots:
- Each undo state stores only the delta (operation type, position, old/new content)
- Typical diff: ~100 bytes for small edits, ~10KB for large paste
- 500-1000 states at average 1KB = 0.5-1MB overhead
- Well within 2x budget even for large files

### Implementation Approach

1. **Operation-based diffs**: Store (type, line, col, deleted_text, inserted_text)
2. **Batch grouping**: Use timer + navigation detection
3. **Circular buffer**: Fixed-size ring for history, oldest dropped automatically
4. **Cursor restoration**: Store (line, col) with each undo state
