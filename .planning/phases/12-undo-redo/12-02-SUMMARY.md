---
phase: 12-undo-redo
plan: 02
subsystem: tui
tags: [undo-redo, timer, debounce, bubbletea, tea.Tick]

# Dependency graph
requires:
  - phase: 12-01
    provides: UndoManager core with circular buffer, operations, batches
provides:
  - undoGroupingDelay constant (1000ms)
  - undoGroupMsg type with batchID for stale timer detection
  - groupID field for timer invalidation
  - CommitCurrentBatch() for timer-based commits
  - ForceBoundary() for immediate boundaries
  - CreateGroupCmd() for tea.Tick integration
affects: [12-03, model.go integration]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Same debounce invalidation pattern as evalDebounceMsg (batchID vs groupID)"
    - "Immediate boundaries for Enter, navigation, line joins"
    - "Timer-based boundaries for typing pause (1s)"

key-files:
  created: []
  modified:
    - cmd/calcmark/tui/editor/undo.go
    - cmd/calcmark/tui/editor/undo_test.go

key-decisions:
  - "undoGroupingDelay = 1000ms (lower end of 1-2 second range per CONTEXT.md)"
  - "Follow evalDebounceMsg pattern for stale timer detection"
  - "CommitCurrentBatch and ForceBoundary are semantic aliases for CommitBatch"
  - "groupID never resets (stale timers remain invalidated across commits)"

patterns-established:
  - "Timer invalidation: batchID captured at timer creation, compared to groupID on fire"
  - "Boundary triggers: Enter, navigation, line joins call ForceBoundary() immediately"
  - "Grouping: Character typing and in-line deletes grouped by timer, not immediate"

# Metrics
duration: 3min
completed: 2026-02-08
---

# Phase 12 Plan 02: Timer-based Grouping Summary

**Undo grouping infrastructure with 1-second timer, stale detection via groupID/batchID, and immediate boundaries for Enter/navigation**

## Performance

- **Duration:** 3 min
- **Started:** 2026-02-08T16:14:07Z
- **Completed:** 2026-02-08T16:16:39Z
- **Tasks:** 3
- **Files modified:** 2

## Accomplishments
- Added undoGroupingDelay constant (1000ms) for natural typing boundaries
- Implemented undoGroupMsg type with batchID for stale timer detection
- Added groupID field that increments on each AddOperation (invalidates pending timers)
- Created CommitCurrentBatch() and ForceBoundary() methods for boundary control
- Added CreateGroupCmd() that returns tea.Tick command for bubbletea integration
- Comprehensive tests for grouping scenarios (302 new lines)

## Task Commits

Each task was committed atomically:

1. **Task 1: Add grouping constants and message type** - `fb6ecf9` (feat)
2. **Task 2: Implement boundary triggers** - `b6b4f2a` (feat)
3. **Task 3: Add unit tests for grouping behavior** - `4ec4263` (test)

## Files Created/Modified
- `cmd/calcmark/tui/editor/undo.go` - Added grouping constants, undoGroupMsg, groupID, boundary methods
- `cmd/calcmark/tui/editor/undo_test.go` - Added 302 lines of grouping tests (760 total lines)

## Decisions Made
- **undoGroupingDelay = 1000ms:** Lower end of 1-2 second range from CONTEXT.md. Can be tuned later based on user feedback.
- **Same pattern as evalDebounceMsg:** batchID vs groupID comparison for stale timer detection. Proven pattern in codebase.
- **groupID never resets:** CommitBatch doesn't reset groupID, so stale timers from before a commit remain invalidated.
- **Semantic method names:** CommitCurrentBatch for timer callbacks, ForceBoundary for immediate triggers. Both delegate to CommitBatch.

## Deviations from Plan
None - plan executed exactly as written.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Grouping infrastructure complete, ready for model.go integration
- Plan 12-03 will wire UndoManager into Model and handle undoGroupMsg
- All boundary triggers (Enter, navigation, line joins) documented in UndoManager comments

---
*Phase: 12-undo-redo*
*Completed: 2026-02-08*
