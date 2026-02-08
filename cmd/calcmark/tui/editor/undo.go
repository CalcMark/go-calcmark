// Package editor provides the TUI editor for CalcMark documents.
package editor

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// undoGroupingDelay is the duration of typing pause that creates an undo boundary.
// Consecutive typing within this duration groups into a single undo step.
// Per CONTEXT.md: 1-2 seconds, starting with 1 second.
const undoGroupingDelay = 1000 * time.Millisecond

// undoGroupMsg is sent when the grouping timer fires to commit the current batch.
// The batchID field prevents stale timer commits using the same pattern as evalDebounceMsg.
// If batchID doesn't match the manager's current groupID, the timer is stale and ignored.
type undoGroupMsg struct {
	batchID int
}

// OpType represents the type of edit operation.
type OpType int

const (
	// OpInsert represents an insertion of text.
	OpInsert OpType = iota
	// OpDelete represents a deletion of text.
	OpDelete
	// OpReplace represents a replacement of text (delete + insert at same position).
	OpReplace
)

// String returns the string representation of OpType.
func (t OpType) String() string {
	switch t {
	case OpInsert:
		return "Insert"
	case OpDelete:
		return "Delete"
	case OpReplace:
		return "Replace"
	default:
		return "Unknown"
	}
}

// EditOperation captures a single edit operation with position and text.
// Each operation stores enough information to both undo and redo.
//
// Memory estimate: ~100 bytes base + len(OldText) + len(NewText)
type EditOperation struct {
	// Type is the kind of operation (insert, delete, replace).
	Type OpType

	// Line and Col specify where the edit occurred (0-indexed).
	Line int
	Col  int

	// OldText is the text that was removed (empty for insert).
	OldText string

	// NewText is the text that was added (empty for delete).
	NewText string

	// CursorLine and CursorCol store the cursor position BEFORE the operation.
	// This allows undo to restore the exact cursor state.
	CursorLine int
	CursorCol  int

	// ScrollOffset stores the scroll position BEFORE the operation.
	// This allows undo to restore the viewport state.
	ScrollOffset int
}

// Reverse returns an EditOperation that undoes this operation.
// Insert becomes Delete, Delete becomes Insert, Replace swaps Old/New.
// The cursor and scroll positions are preserved to restore the pre-edit state.
func (op EditOperation) Reverse() EditOperation {
	reversed := EditOperation{
		Line:         op.Line,
		Col:          op.Col,
		CursorLine:   op.CursorLine,
		CursorCol:    op.CursorCol,
		ScrollOffset: op.ScrollOffset,
	}

	switch op.Type {
	case OpInsert:
		// Undoing insert means deleting the inserted text
		reversed.Type = OpDelete
		reversed.OldText = op.NewText
		reversed.NewText = ""
	case OpDelete:
		// Undoing delete means inserting the deleted text
		reversed.Type = OpInsert
		reversed.OldText = ""
		reversed.NewText = op.OldText
	case OpReplace:
		// Undoing replace means swapping old and new
		reversed.Type = OpReplace
		reversed.OldText = op.NewText
		reversed.NewText = op.OldText
	}

	return reversed
}

// UndoBatch groups multiple operations that should be undone together.
// This enables natural undo boundaries (e.g., typing until pause becomes one undo).
type UndoBatch struct {
	// Operations is the list of operations in this batch, in execution order.
	Operations []EditOperation

	// Timestamp records when this batch was finalized.
	// Used for timer-based grouping decisions.
	Timestamp time.Time
}

// Reverse returns a slice of reversed operations in reverse order.
// This is used to undo the entire batch: operations must be reversed
// in the opposite order they were originally applied.
func (b UndoBatch) Reverse() []EditOperation {
	reversed := make([]EditOperation, len(b.Operations))
	for i, op := range b.Operations {
		// Place reversed operations in reverse order
		reversed[len(b.Operations)-1-i] = op.Reverse()
	}
	return reversed
}

// UndoManager manages the undo/redo history using a circular buffer.
// It stores operation batches rather than full document snapshots for memory efficiency.
//
// Design decisions (from CONTEXT.md):
// - Memory-only storage (no swap files)
// - Soft limit of 1000 states
// - Oldest states dropped silently when full
// - Fresh history on file open
// - Max 2x file size memory budget
//
// Grouping behavior (from CONTEXT.md):
// - Timer expiry (1 second pause) -> auto boundary
// - Enter key -> immediate boundary (always creates boundary)
// - Arrow keys -> immediate boundary (navigation creates boundary)
// - Line joins (Delete at EOL, Backspace at BOL) -> immediate boundary (separate step)
// - Paste -> immediate boundary before AND after (one step)
// - Scroll -> NO boundary (scroll doesn't create boundary)
// - Character typing: No boundary (grouped by timer)
// - Delete/Backspace within line: No boundary (grouped by timer)
type UndoManager struct {
	// history is a circular buffer of committed batches.
	// Pre-allocated to maxHistory capacity.
	history []UndoBatch

	// head points to the next write position in the circular buffer.
	head int

	// size tracks how many batches are currently in history.
	size int

	// redoStack holds batches that were undone and can be redone.
	// Cleared when new edits are made (standard undo/redo behavior).
	redoStack []UndoBatch

	// current holds uncommitted operations for the current batch.
	current []EditOperation

	// maxHistory is the maximum number of batches to keep.
	maxHistory int

	// groupID is incremented on each AddOperation to invalidate pending timers.
	// When a timer fires, if its batchID doesn't match groupID, the timer is stale.
	// This is the same pattern used by evalDebounceMsg.editBufSnapshot.
	groupID int
}

// NewUndoManager creates a new UndoManager with the specified history limit.
// The history slice is pre-allocated to avoid allocations during editing.
func NewUndoManager(maxHistory int) *UndoManager {
	if maxHistory < 1 {
		maxHistory = 1
	}
	return &UndoManager{
		history:    make([]UndoBatch, maxHistory),
		maxHistory: maxHistory,
		redoStack:  make([]UndoBatch, 0),
		current:    make([]EditOperation, 0),
	}
}

// AddOperation adds an operation to the current uncommitted batch.
// This clears the redo stack (standard undo/redo behavior per CONTEXT.md discretion).
// The groupID is incremented to invalidate any pending grouping timers.
func (m *UndoManager) AddOperation(op EditOperation) {
	m.current = append(m.current, op)
	// Clear redo stack on new edit (standard behavior)
	m.redoStack = m.redoStack[:0]
	// Increment groupID to invalidate stale timers
	m.groupID++
}

// GetGroupID returns the current groupID for timer validation.
// Timer messages compare their batchID against this value to detect staleness.
func (m *UndoManager) GetGroupID() int {
	return m.groupID
}

// CreateGroupCmd returns a tea.Cmd that fires undoGroupMsg after undoGroupingDelay.
// The message includes the current groupID so stale timers can be detected.
func (m *UndoManager) CreateGroupCmd() tea.Cmd {
	batchID := m.groupID
	return tea.Tick(undoGroupingDelay, func(t time.Time) tea.Msg {
		return undoGroupMsg{batchID: batchID}
	})
}

// CommitBatch finalizes the current operations as a batch in history.
// If there are no current operations, this is a no-op.
// The batch is timestamped with the current time.
func (m *UndoManager) CommitBatch() {
	if len(m.current) == 0 {
		return
	}

	batch := UndoBatch{
		Operations: m.current,
		Timestamp:  time.Now(),
	}

	// Push to circular buffer
	m.history[m.head] = batch
	m.head = (m.head + 1) % m.maxHistory
	if m.size < m.maxHistory {
		m.size++
	}

	// Reset current batch
	m.current = make([]EditOperation, 0)
}

// Undo pops the most recent batch from history for reversal.
// Returns the batch and true if successful, or an empty batch and false if history is empty.
// The undone batch is pushed to the redo stack.
func (m *UndoManager) Undo() (UndoBatch, bool) {
	if m.size == 0 {
		return UndoBatch{}, false
	}

	// Pop from circular buffer (head-1, wrapping)
	m.head = (m.head - 1 + m.maxHistory) % m.maxHistory
	m.size--

	batch := m.history[m.head]

	// Push to redo stack
	m.redoStack = append(m.redoStack, batch)

	return batch, true
}

// Redo pops the most recent undone batch from the redo stack.
// Returns the batch and true if successful, or an empty batch and false if redo stack is empty.
// The redone batch is pushed back to history.
func (m *UndoManager) Redo() (UndoBatch, bool) {
	if len(m.redoStack) == 0 {
		return UndoBatch{}, false
	}

	// Pop from redo stack
	batch := m.redoStack[len(m.redoStack)-1]
	m.redoStack = m.redoStack[:len(m.redoStack)-1]

	// Push to circular buffer
	m.history[m.head] = batch
	m.head = (m.head + 1) % m.maxHistory
	if m.size < m.maxHistory {
		m.size++
	}

	return batch, true
}

// HasUndo returns true if there are batches available to undo.
func (m *UndoManager) HasUndo() bool {
	return m.size > 0
}

// HasRedo returns true if there are batches available to redo.
func (m *UndoManager) HasRedo() bool {
	return len(m.redoStack) > 0
}

// Clear resets all undo/redo state.
// Called when opening a new file (per CONTEXT.md: fresh start).
func (m *UndoManager) Clear() {
	m.head = 0
	m.size = 0
	m.redoStack = m.redoStack[:0]
	m.current = m.current[:0]
}

// CurrentBatchSize returns the number of uncommitted operations.
// Useful for deciding when to commit based on operation count.
func (m *UndoManager) CurrentBatchSize() int {
	return len(m.current)
}

// HistorySize returns the number of committed batches in history.
func (m *UndoManager) HistorySize() int {
	return m.size
}

// CommitCurrentBatch commits the current uncommitted operations as a batch immediately.
// This is used when the grouping timer fires after a typing pause.
// If there are no current operations, this is a no-op.
//
// This is an alias for CommitBatch with semantic clarity for timer-based usage.
// The groupID is NOT reset, so any pending timers remain invalidated.
func (m *UndoManager) CommitCurrentBatch() {
	m.CommitBatch()
}

// ForceBoundary creates an immediate undo boundary by committing the current batch.
// This is used for operations that should always create a boundary:
//   - Enter key (new line)
//   - Arrow keys (navigation)
//   - Line joins (Delete at EOL, Backspace at BOL)
//   - Before and after paste operations
//
// Unlike timer-based commits, this is called synchronously when the boundary-creating
// action occurs. The effect is identical to CommitCurrentBatch, but the name makes
// the intent clear in calling code.
func (m *UndoManager) ForceBoundary() {
	m.CommitBatch()
}
