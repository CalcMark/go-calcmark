package editor

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/CalcMark/go-calcmark/v2/spec/document"
)

// TestEditOperation_Reverse tests that operations reverse correctly.
func TestEditOperation_Reverse(t *testing.T) {
	tests := []struct {
		name     string
		op       EditOperation
		wantType OpType
		wantOld  string
		wantNew  string
	}{
		{
			name: "Insert becomes Delete",
			op: EditOperation{
				Type:         OpInsert,
				Line:         5,
				Col:          10,
				OldText:      "",
				NewText:      "hello",
				CursorLine:   5,
				CursorCol:    10,
				ScrollOffset: 0,
			},
			wantType: OpDelete,
			wantOld:  "hello", // The inserted text becomes old text to delete
			wantNew:  "",
		},
		{
			name: "Delete becomes Insert",
			op: EditOperation{
				Type:         OpDelete,
				Line:         3,
				Col:          5,
				OldText:      "world",
				NewText:      "",
				CursorLine:   3,
				CursorCol:    5,
				ScrollOffset: 2,
			},
			wantType: OpInsert,
			wantOld:  "",
			wantNew:  "world", // The deleted text becomes new text to insert
		},
		{
			name: "Replace swaps Old and New",
			op: EditOperation{
				Type:         OpReplace,
				Line:         1,
				Col:          0,
				OldText:      "foo",
				NewText:      "bar",
				CursorLine:   1,
				CursorCol:    0,
				ScrollOffset: 0,
			},
			wantType: OpReplace,
			wantOld:  "bar", // Swapped
			wantNew:  "foo", // Swapped
		},
		{
			name: "InsertLine becomes DeleteLine",
			op: EditOperation{
				Type:         OpInsertLine,
				Line:         2,
				Col:          5,
				OldText:      "hello world",
				NewText:      " world",
				CursorLine:   2,
				CursorCol:    5,
				ScrollOffset: 0,
			},
			wantType: OpDeleteLine,
			wantOld:  "hello world",
			wantNew:  " world",
		},
		{
			name: "DeleteLine becomes InsertLine",
			op: EditOperation{
				Type:         OpDeleteLine,
				Line:         3,
				Col:          0,
				OldText:      "deleted line content",
				NewText:      "",
				CursorLine:   3,
				CursorCol:    0,
				ScrollOffset: 1,
			},
			wantType: OpInsertLine,
			wantOld:  "deleted line content",
			wantNew:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reversed := tt.op.Reverse()

			if reversed.Type != tt.wantType {
				t.Errorf("Type = %v, want %v", reversed.Type, tt.wantType)
			}
			if reversed.OldText != tt.wantOld {
				t.Errorf("OldText = %q, want %q", reversed.OldText, tt.wantOld)
			}
			if reversed.NewText != tt.wantNew {
				t.Errorf("NewText = %q, want %q", reversed.NewText, tt.wantNew)
			}

			// Position should be preserved
			if reversed.Line != tt.op.Line {
				t.Errorf("Line = %d, want %d", reversed.Line, tt.op.Line)
			}
			if reversed.Col != tt.op.Col {
				t.Errorf("Col = %d, want %d", reversed.Col, tt.op.Col)
			}

			// Cursor and scroll should be preserved
			if reversed.CursorLine != tt.op.CursorLine {
				t.Errorf("CursorLine = %d, want %d", reversed.CursorLine, tt.op.CursorLine)
			}
			if reversed.CursorCol != tt.op.CursorCol {
				t.Errorf("CursorCol = %d, want %d", reversed.CursorCol, tt.op.CursorCol)
			}
			if reversed.ScrollOffset != tt.op.ScrollOffset {
				t.Errorf("ScrollOffset = %d, want %d", reversed.ScrollOffset, tt.op.ScrollOffset)
			}
		})
	}
}

// TestUndoBatch_Reverse tests that batches reverse in correct order.
func TestUndoBatch_Reverse(t *testing.T) {
	batch := UndoBatch{
		Operations: []EditOperation{
			{Type: OpInsert, NewText: "a"},
			{Type: OpInsert, NewText: "b"},
			{Type: OpInsert, NewText: "c"},
		},
	}

	reversed := batch.Reverse()

	// Should be 3 operations
	if len(reversed) != 3 {
		t.Fatalf("Reverse() returned %d operations, want 3", len(reversed))
	}

	// Should be in reverse order: c, b, a (and each should be reversed to Delete)
	expected := []string{"c", "b", "a"}
	for i, op := range reversed {
		if op.Type != OpDelete {
			t.Errorf("reversed[%d].Type = %v, want OpDelete", i, op.Type)
		}
		if op.OldText != expected[i] {
			t.Errorf("reversed[%d].OldText = %q, want %q", i, op.OldText, expected[i])
		}
	}
}

// TestUndoManager_AddAndCommit tests adding operations and committing batches.
func TestUndoManager_AddAndCommit(t *testing.T) {
	m := NewUndoManager(100)

	// Initially empty
	if m.HasUndo() {
		t.Error("HasUndo() = true on new manager, want false")
	}
	if m.CurrentBatchSize() != 0 {
		t.Errorf("CurrentBatchSize() = %d, want 0", m.CurrentBatchSize())
	}

	// Add operation
	m.AddOperation(EditOperation{Type: OpInsert, NewText: "hello"})
	if m.CurrentBatchSize() != 1 {
		t.Errorf("CurrentBatchSize() = %d after add, want 1", m.CurrentBatchSize())
	}
	// Not committed yet, so no undo available
	if m.HasUndo() {
		t.Error("HasUndo() = true before commit, want false")
	}

	// Add another operation
	m.AddOperation(EditOperation{Type: OpInsert, NewText: " world"})
	if m.CurrentBatchSize() != 2 {
		t.Errorf("CurrentBatchSize() = %d after second add, want 2", m.CurrentBatchSize())
	}

	// Commit
	m.CommitBatch()
	if m.CurrentBatchSize() != 0 {
		t.Errorf("CurrentBatchSize() = %d after commit, want 0", m.CurrentBatchSize())
	}
	if !m.HasUndo() {
		t.Error("HasUndo() = false after commit, want true")
	}
	if m.HistorySize() != 1 {
		t.Errorf("HistorySize() = %d, want 1", m.HistorySize())
	}

	// Empty commit is no-op
	m.CommitBatch()
	if m.HistorySize() != 1 {
		t.Errorf("HistorySize() = %d after empty commit, want 1", m.HistorySize())
	}
}

// TestUndoManager_UndoRedo tests undo and redo operations.
func TestUndoManager_UndoRedo(t *testing.T) {
	m := NewUndoManager(100)

	// Undo on empty returns false
	_, ok := m.Undo()
	if ok {
		t.Error("Undo() returned true on empty manager")
	}

	// Redo on empty returns false
	_, ok = m.Redo()
	if ok {
		t.Error("Redo() returned true on empty manager")
	}

	// Add and commit batch 1
	m.AddOperation(EditOperation{Type: OpInsert, NewText: "first"})
	m.CommitBatch()

	// Add and commit batch 2
	m.AddOperation(EditOperation{Type: OpInsert, NewText: "second"})
	m.CommitBatch()

	if m.HistorySize() != 2 {
		t.Errorf("HistorySize() = %d, want 2", m.HistorySize())
	}

	// Undo returns last batch
	batch, ok := m.Undo()
	if !ok {
		t.Fatal("Undo() returned false, want true")
	}
	if len(batch.Operations) != 1 || batch.Operations[0].NewText != "second" {
		t.Errorf("Undo() returned batch with NewText = %q, want 'second'", batch.Operations[0].NewText)
	}
	if m.HistorySize() != 1 {
		t.Errorf("HistorySize() = %d after undo, want 1", m.HistorySize())
	}
	if !m.HasRedo() {
		t.Error("HasRedo() = false after undo, want true")
	}

	// Redo returns the undone batch
	batch, ok = m.Redo()
	if !ok {
		t.Fatal("Redo() returned false, want true")
	}
	if len(batch.Operations) != 1 || batch.Operations[0].NewText != "second" {
		t.Errorf("Redo() returned batch with NewText = %q, want 'second'", batch.Operations[0].NewText)
	}
	if m.HistorySize() != 2 {
		t.Errorf("HistorySize() = %d after redo, want 2", m.HistorySize())
	}
	if m.HasRedo() {
		t.Error("HasRedo() = true after redo, want false")
	}

	// Undo moves batch to redo stack
	m.Undo()
	if !m.HasRedo() {
		t.Error("HasRedo() = false after second undo, want true")
	}

	// Undo again
	batch, ok = m.Undo()
	if !ok {
		t.Fatal("Second Undo() returned false")
	}
	if batch.Operations[0].NewText != "first" {
		t.Errorf("Second Undo() returned batch with NewText = %q, want 'first'", batch.Operations[0].NewText)
	}

	// Now redo should go back in order
	batch, _ = m.Redo()
	if batch.Operations[0].NewText != "first" {
		t.Errorf("Redo after two undos returned %q, want 'first'", batch.Operations[0].NewText)
	}
	batch, _ = m.Redo()
	if batch.Operations[0].NewText != "second" {
		t.Errorf("Second Redo returned %q, want 'second'", batch.Operations[0].NewText)
	}
}

// TestUndoManager_ClearRedoOnNewEdit tests that new edits clear the redo stack.
func TestUndoManager_ClearRedoOnNewEdit(t *testing.T) {
	m := NewUndoManager(100)

	// Create some history
	m.AddOperation(EditOperation{Type: OpInsert, NewText: "one"})
	m.CommitBatch()
	m.AddOperation(EditOperation{Type: OpInsert, NewText: "two"})
	m.CommitBatch()

	// Undo to build redo stack
	m.Undo()
	if !m.HasRedo() {
		t.Fatal("HasRedo() = false after undo")
	}

	// New edit should clear redo
	m.AddOperation(EditOperation{Type: OpInsert, NewText: "new"})
	if m.HasRedo() {
		t.Error("HasRedo() = true after new edit, want false")
	}

	// Redo should fail
	_, ok := m.Redo()
	if ok {
		t.Error("Redo() succeeded after new edit, want failure")
	}
}

// TestUndoManager_CircularBuffer tests the circular buffer behavior.
func TestUndoManager_CircularBuffer(t *testing.T) {
	maxHistory := 5
	m := NewUndoManager(maxHistory)

	// Commit more batches than maxHistory
	for i := 0; i < maxHistory+3; i++ {
		m.AddOperation(EditOperation{
			Type:    OpInsert,
			NewText: string(rune('a' + i)),
		})
		m.CommitBatch()
	}

	// Should only have maxHistory batches
	if m.HistorySize() != maxHistory {
		t.Errorf("HistorySize() = %d, want %d", m.HistorySize(), maxHistory)
	}

	// Undo should return batches in reverse order, starting from most recent
	// We added a, b, c, d, e, f, g, h (8 batches)
	// With max 5, we should have d, e, f, g, h (indices 3-7)
	expected := []string{"h", "g", "f", "e", "d"}
	for i, exp := range expected {
		batch, ok := m.Undo()
		if !ok {
			t.Fatalf("Undo() %d failed", i)
		}
		got := batch.Operations[0].NewText
		if got != exp {
			t.Errorf("Undo() %d returned %q, want %q", i, got, exp)
		}
	}

	// One more undo should fail (all undone)
	_, ok := m.Undo()
	if ok {
		t.Error("Undo() succeeded when history exhausted")
	}
}

// TestUndoManager_CircularBufferOverwrite tests that oldest batches are silently dropped.
func TestUndoManager_CircularBufferOverwrite(t *testing.T) {
	m := NewUndoManager(3)

	// Add 3 batches
	for i := range 3 {
		m.AddOperation(EditOperation{Type: OpInsert, NewText: string(rune('a' + i))})
		m.CommitBatch()
	}

	// Undo all 3
	batches := make([]string, 0, 3)
	for m.HasUndo() {
		batch, _ := m.Undo()
		batches = append(batches, batch.Operations[0].NewText)
	}
	if len(batches) != 3 {
		t.Fatalf("Got %d batches, want 3", len(batches))
	}

	// Redo all 3
	for m.HasRedo() {
		m.Redo()
	}

	// Add 2 more batches (should overwrite 'a' and 'b')
	m.AddOperation(EditOperation{Type: OpInsert, NewText: "x"})
	m.CommitBatch()
	m.AddOperation(EditOperation{Type: OpInsert, NewText: "y"})
	m.CommitBatch()

	// Should have 3 batches: c, x, y
	if m.HistorySize() != 3 {
		t.Errorf("HistorySize() = %d, want 3", m.HistorySize())
	}

	// Undo should return y, x, c
	expected := []string{"y", "x", "c"}
	for i, exp := range expected {
		batch, ok := m.Undo()
		if !ok {
			t.Fatalf("Undo() %d failed", i)
		}
		got := batch.Operations[0].NewText
		if got != exp {
			t.Errorf("Undo() %d returned %q, want %q", i, got, exp)
		}
	}
}

// TestUndoManager_Clear tests the Clear method.
func TestUndoManager_Clear(t *testing.T) {
	m := NewUndoManager(100)

	// Build up some state
	m.AddOperation(EditOperation{Type: OpInsert, NewText: "test"})
	m.CommitBatch()
	m.AddOperation(EditOperation{Type: OpInsert, NewText: "more"})
	m.CommitBatch()
	m.Undo()

	// Verify state exists
	if !m.HasUndo() {
		t.Fatal("HasUndo() = false before clear")
	}
	if !m.HasRedo() {
		t.Fatal("HasRedo() = false before clear")
	}

	// Add uncommitted operation
	m.AddOperation(EditOperation{Type: OpInsert, NewText: "uncommitted"})
	if m.CurrentBatchSize() != 1 {
		t.Fatalf("CurrentBatchSize() = %d before clear, want 1", m.CurrentBatchSize())
	}

	// Clear
	m.Clear()

	// All state should be gone
	if m.HasUndo() {
		t.Error("HasUndo() = true after Clear()")
	}
	if m.HasRedo() {
		t.Error("HasRedo() = true after Clear()")
	}
	if m.CurrentBatchSize() != 0 {
		t.Errorf("CurrentBatchSize() = %d after Clear(), want 0", m.CurrentBatchSize())
	}
	if m.HistorySize() != 0 {
		t.Errorf("HistorySize() = %d after Clear(), want 0", m.HistorySize())
	}
}

// TestNewUndoManager_MinHistory tests that maxHistory has a minimum of 1.
func TestNewUndoManager_MinHistory(t *testing.T) {
	// Zero should become 1
	m := NewUndoManager(0)
	if m.maxHistory != 1 {
		t.Errorf("maxHistory = %d for input 0, want 1", m.maxHistory)
	}

	// Negative should become 1
	m = NewUndoManager(-5)
	if m.maxHistory != 1 {
		t.Errorf("maxHistory = %d for input -5, want 1", m.maxHistory)
	}
}

// TestOpType_String tests OpType string representation.
func TestOpType_String(t *testing.T) {
	tests := []struct {
		op   OpType
		want string
	}{
		{OpInsert, "Insert"},
		{OpDelete, "Delete"},
		{OpReplace, "Replace"},
		{OpInsertLine, "InsertLine"},
		{OpDeleteLine, "DeleteLine"},
		{OpType(99), "Unknown"},
	}

	for _, tt := range tests {
		got := tt.op.String()
		if got != tt.want {
			t.Errorf("OpType(%d).String() = %q, want %q", tt.op, got, tt.want)
		}
	}
}

// TestUndoManager_GroupIDIncrement tests that groupID increments correctly.
func TestUndoManager_GroupIDIncrement(t *testing.T) {
	m := NewUndoManager(100)

	// Initial groupID should be 0
	if m.GetGroupID() != 0 {
		t.Errorf("Initial GetGroupID() = %d, want 0", m.GetGroupID())
	}

	// Each AddOperation increments groupID
	m.AddOperation(EditOperation{Type: OpInsert, NewText: "a"})
	if m.GetGroupID() != 1 {
		t.Errorf("GetGroupID() after first add = %d, want 1", m.GetGroupID())
	}

	m.AddOperation(EditOperation{Type: OpInsert, NewText: "b"})
	if m.GetGroupID() != 2 {
		t.Errorf("GetGroupID() after second add = %d, want 2", m.GetGroupID())
	}

	m.AddOperation(EditOperation{Type: OpInsert, NewText: "c"})
	if m.GetGroupID() != 3 {
		t.Errorf("GetGroupID() after third add = %d, want 3", m.GetGroupID())
	}

	// CommitBatch does NOT reset groupID (stale timers remain invalidated)
	m.CommitBatch()
	if m.GetGroupID() != 3 {
		t.Errorf("GetGroupID() after CommitBatch = %d, want 3 (unchanged)", m.GetGroupID())
	}

	// Adding more operations continues incrementing
	m.AddOperation(EditOperation{Type: OpInsert, NewText: "d"})
	if m.GetGroupID() != 4 {
		t.Errorf("GetGroupID() after fourth add = %d, want 4", m.GetGroupID())
	}
}

// TestUndoManager_CommitCurrentBatch tests the CommitCurrentBatch alias.
func TestUndoManager_CommitCurrentBatch(t *testing.T) {
	m := NewUndoManager(100)

	// Multiple AddOperation calls before CommitCurrentBatch creates single batch
	m.AddOperation(EditOperation{Type: OpInsert, NewText: "h"})
	m.AddOperation(EditOperation{Type: OpInsert, NewText: "e"})
	m.AddOperation(EditOperation{Type: OpInsert, NewText: "l"})
	m.AddOperation(EditOperation{Type: OpInsert, NewText: "l"})
	m.AddOperation(EditOperation{Type: OpInsert, NewText: "o"})

	if m.CurrentBatchSize() != 5 {
		t.Errorf("CurrentBatchSize() = %d, want 5", m.CurrentBatchSize())
	}

	// CommitCurrentBatch groups them
	m.CommitCurrentBatch()

	if m.CurrentBatchSize() != 0 {
		t.Errorf("CurrentBatchSize() after commit = %d, want 0", m.CurrentBatchSize())
	}
	if m.HistorySize() != 1 {
		t.Errorf("HistorySize() = %d, want 1", m.HistorySize())
	}

	// Undo returns all operations together
	batch, ok := m.Undo()
	if !ok {
		t.Fatal("Undo() failed")
	}
	if len(batch.Operations) != 5 {
		t.Errorf("Batch has %d operations, want 5", len(batch.Operations))
	}

	// Verify the operations are in order
	expected := []string{"h", "e", "l", "l", "o"}
	for i, exp := range expected {
		if batch.Operations[i].NewText != exp {
			t.Errorf("Operation[%d].NewText = %q, want %q", i, batch.Operations[i].NewText, exp)
		}
	}

	// CommitCurrentBatch with no current ops is no-op
	m.Redo() // Put the batch back
	if m.HistorySize() != 1 {
		t.Fatalf("HistorySize() = %d after redo, want 1", m.HistorySize())
	}

	m.CommitCurrentBatch() // No-op, no uncommitted operations
	if m.HistorySize() != 1 {
		t.Errorf("HistorySize() = %d after empty commit, want 1", m.HistorySize())
	}
}

// TestUndoManager_ForceBoundary tests the ForceBoundary method.
func TestUndoManager_ForceBoundary(t *testing.T) {
	m := NewUndoManager(100)

	// Add some operations
	m.AddOperation(EditOperation{Type: OpInsert, NewText: "h"})
	m.AddOperation(EditOperation{Type: OpInsert, NewText: "i"})

	// ForceBoundary commits current batch (simulates Enter key)
	m.ForceBoundary()

	if m.HistorySize() != 1 {
		t.Errorf("HistorySize() after ForceBoundary = %d, want 1", m.HistorySize())
	}
	if m.CurrentBatchSize() != 0 {
		t.Errorf("CurrentBatchSize() after ForceBoundary = %d, want 0", m.CurrentBatchSize())
	}

	// Subsequent AddOperation starts new batch
	m.AddOperation(EditOperation{Type: OpInsert, NewText: "t"})
	m.AddOperation(EditOperation{Type: OpInsert, NewText: "h"})
	m.AddOperation(EditOperation{Type: OpInsert, NewText: "e"})
	m.AddOperation(EditOperation{Type: OpInsert, NewText: "r"})
	m.AddOperation(EditOperation{Type: OpInsert, NewText: "e"})

	m.ForceBoundary()

	if m.HistorySize() != 2 {
		t.Errorf("HistorySize() after second ForceBoundary = %d, want 2", m.HistorySize())
	}

	// Undo should return "there" first, then "hi"
	batch1, _ := m.Undo()
	if len(batch1.Operations) != 5 {
		t.Errorf("First undo batch has %d ops, want 5", len(batch1.Operations))
	}

	batch2, _ := m.Undo()
	if len(batch2.Operations) != 2 {
		t.Errorf("Second undo batch has %d ops, want 2", len(batch2.Operations))
	}
}

// TestUndoManager_GroupingScenarios tests realistic grouping scenarios.
func TestUndoManager_GroupingScenarios(t *testing.T) {
	t.Run("Scenario 1: Type 'hello' rapidly -> 1 undo step", func(t *testing.T) {
		m := NewUndoManager(100)

		// Rapid typing (5 ops), then commit (simulating timer expiry)
		for _, ch := range "hello" {
			m.AddOperation(EditOperation{Type: OpInsert, NewText: string(ch)})
		}
		m.CommitCurrentBatch()

		if m.HistorySize() != 1 {
			t.Errorf("HistorySize() = %d, want 1", m.HistorySize())
		}

		batch, _ := m.Undo()
		if len(batch.Operations) != 5 {
			t.Errorf("Batch has %d operations, want 5", len(batch.Operations))
		}
	})

	t.Run("Scenario 2: Type 'hi', boundary, type 'there' -> 2 undo steps", func(t *testing.T) {
		m := NewUndoManager(100)

		// Type "hi"
		for _, ch := range "hi" {
			m.AddOperation(EditOperation{Type: OpInsert, NewText: string(ch)})
		}
		m.ForceBoundary() // Simulates navigation or Enter

		// Type "there"
		for _, ch := range "there" {
			m.AddOperation(EditOperation{Type: OpInsert, NewText: string(ch)})
		}
		m.CommitCurrentBatch()

		if m.HistorySize() != 2 {
			t.Errorf("HistorySize() = %d, want 2", m.HistorySize())
		}

		// First undo: "there"
		batch1, _ := m.Undo()
		if len(batch1.Operations) != 5 {
			t.Errorf("First batch has %d ops, want 5", len(batch1.Operations))
		}

		// Second undo: "hi"
		batch2, _ := m.Undo()
		if len(batch2.Operations) != 2 {
			t.Errorf("Second batch has %d ops, want 2", len(batch2.Operations))
		}
	})

	t.Run("Scenario 3: Type 'a', delete 'a' in same batch -> 1 undo step", func(t *testing.T) {
		m := NewUndoManager(100)

		// Type "a"
		m.AddOperation(EditOperation{
			Type:       OpInsert,
			NewText:    "a",
			Line:       0,
			Col:        0,
			CursorLine: 0,
			CursorCol:  0,
		})

		// Delete "a" (within same batch - rapid typing/deleting)
		m.AddOperation(EditOperation{
			Type:       OpDelete,
			OldText:    "a",
			Line:       0,
			Col:        0,
			CursorLine: 0,
			CursorCol:  1,
		})

		m.CommitCurrentBatch()

		if m.HistorySize() != 1 {
			t.Errorf("HistorySize() = %d, want 1", m.HistorySize())
		}

		// Single undo restores original (both ops undone)
		batch, _ := m.Undo()
		if len(batch.Operations) != 2 {
			t.Errorf("Batch has %d operations, want 2", len(batch.Operations))
		}

		// Verify reversal: should be Insert (undo delete), then Delete (undo insert)
		reversed := batch.Reverse()
		if len(reversed) != 2 {
			t.Fatalf("Reversed batch has %d ops, want 2", len(reversed))
		}
		// First reversed: undo the delete -> insert "a"
		if reversed[0].Type != OpInsert || reversed[0].NewText != "a" {
			t.Errorf("reversed[0] = {%v, %q}, want {Insert, \"a\"}", reversed[0].Type, reversed[0].NewText)
		}
		// Second reversed: undo the insert -> delete "a"
		if reversed[1].Type != OpDelete || reversed[1].OldText != "a" {
			t.Errorf("reversed[1] = {%v, old=%q}, want {Delete, old=\"a\"}", reversed[1].Type, reversed[1].OldText)
		}
	})
}

// TestUndoGroupMsg_StaleTimer tests the stale timer detection pattern.
func TestUndoGroupMsg_StaleTimer(t *testing.T) {
	m := NewUndoManager(100)

	// Type something
	m.AddOperation(EditOperation{Type: OpInsert, NewText: "a"})
	batchID := m.GetGroupID() // Capture current groupID (1)

	if batchID != 1 {
		t.Errorf("batchID = %d, want 1", batchID)
	}

	// Create a message with this batchID (simulates timer being started)
	msg := undoGroupMsg{batchID: batchID}

	// User keeps typing (new operations)
	m.AddOperation(EditOperation{Type: OpInsert, NewText: "b"})
	m.AddOperation(EditOperation{Type: OpInsert, NewText: "c"})

	// groupID has changed, timer is now stale
	currentGroupID := m.GetGroupID()
	if currentGroupID != 3 {
		t.Errorf("currentGroupID = %d, want 3", currentGroupID)
	}

	// Verify timer is stale
	if msg.batchID == m.GetGroupID() {
		t.Error("Timer should be stale: msg.batchID == GetGroupID()")
	}

	// This is how model.go will check: if msg.batchID != m.undo.GetGroupID(), ignore
	isStale := msg.batchID != m.GetGroupID()
	if !isStale {
		t.Error("isStale should be true")
	}

	// If timer fires now, it should be ignored (the calling code checks this)
	// We can simulate "fresh" timer by matching groupID
	freshMsg := undoGroupMsg{batchID: m.GetGroupID()}
	isFresh := freshMsg.batchID == m.GetGroupID()
	if !isFresh {
		t.Error("Fresh timer should match current groupID")
	}
}

// TestUndoManager_CreateGroupCmd tests that CreateGroupCmd returns a valid tea.Cmd.
func TestUndoManager_CreateGroupCmd(t *testing.T) {
	m := NewUndoManager(100)

	// Add an operation to have a non-zero groupID
	m.AddOperation(EditOperation{Type: OpInsert, NewText: "test"})

	// CreateGroupCmd should return a non-nil tea.Cmd
	cmd := m.CreateGroupCmd()
	if cmd == nil {
		t.Error("CreateGroupCmd() returned nil")
	}

	// We can't easily test the tea.Tick behavior without running the bubbletea
	// runtime, but we can verify the command is created and the groupID is captured
	// The actual timer firing and message handling is tested in integration tests
}

// TestUndoResetsEditBufLoaded verifies that undo resets editBufLoaded
// so the cursor line's content is freshly loaded from the document.
//
// Bug scenario (P0-01 from architecture review):
//  1. Type "abc" on line 0 → editBuf="abc", editBufLoaded=true
//  2. Debounce saves to document
//  3. Undo → cursor restored to line 0, document reverts to ""
//  4. BUT editBufLoaded is still true and editBuf has stale content
//     from applyOperationReverse (which operated on a different line
//     or left editBuf with the reversed content).
//  5. View renders stale editBuf instead of loading fresh from document.
func TestUndoResetsEditBufLoaded(t *testing.T) {
	doc, err := document.NewDocument("\n")
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 24

	// Type "abc" on line 0
	for _, r := range "abc" {
		result, _ := m.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
		m = result.(Model)
	}

	if m.editBuf != "abc" {
		t.Fatalf("After typing: editBuf=%q, want 'abc'", m.editBuf)
	}
	if !m.editBufLoaded {
		t.Fatal("After typing: editBufLoaded should be true")
	}

	// Simulate debounce saving to document
	m.transitionToProcessing()

	// Verify document has "abc"
	lines := m.GetLines()
	if len(lines) == 0 || lines[0] != "abc" {
		t.Fatalf("After debounce: document line 0=%q, want 'abc'", lines[0])
	}

	// Now undo — should revert document to "" and reset editBufLoaded
	result, _ := m.Update(tea.KeyPressMsg{Code: 'z', Mod: tea.ModCtrl})
	m = result.(Model)

	// After undo, editBufLoaded MUST be true — undo eagerly reloads editBuf
	// from the document for the restored cursor line.
	if !m.editBufLoaded {
		t.Error("After undo: editBufLoaded must be true (eagerly loaded from document)")
	}

	// The document should have reverted
	lines = m.GetLines()
	if len(lines) > 0 && lines[0] != "" {
		t.Errorf("After undo: document line 0=%q, want empty", lines[0])
	}

	// editBuf should match the document's content for the cursor line
	if m.editBuf != "" {
		t.Errorf("After undo: editBuf=%q, want empty (matching document)", m.editBuf)
	}
}

// --- Tests moved from model_test.go ---

func TestUndoManager(t *testing.T) {
	doc, _ := document.NewDocument("x = 10\n")
	m := New(doc)

	// Initial undo stack should be empty (no operations recorded yet)
	if m.undoManager.HistorySize() != 0 {
		t.Errorf("Expected 0 undo entries, got %d", m.undoManager.HistorySize())
	}

	// UndoManager is initialized
	if m.undoManager == nil {
		t.Error("UndoManager should be initialized")
	}

	// Test that HasUndo returns false when empty
	if m.undoManager.HasUndo() {
		t.Error("Should not have undo available when history is empty")
	}
}

// TestUndoRedoIntegration tests the FULL undo/redo flow through Update().
// This test explicitly verifies that Ctrl+Z actually modifies the document,
// catching value/pointer receiver bugs that would cause changes to be lost.
func TestUndoRedoIntegration(t *testing.T) {
	// Use multi-line document so navigation commits edits
	doc, _ := document.NewDocument("# Header\nx = 10\n")
	m := New(doc)

	// Get initial content
	initialLines := m.GetLines()
	if len(initialLines) == 0 || initialLines[0] != "# Header" {
		t.Fatalf("Expected initial line '# Header', got %v", initialLines)
	}

	// Step 1: Type "abc" by sending KeyPressMsg through Update()
	// This simulates actual user typing (one character at a time in v2)
	var newModel tea.Model
	for _, ch := range "abc" {
		newModel, _ = m.Update(tea.KeyPressMsg{Code: ch, Text: string(ch)})
		m = newModel.(Model)
	}

	// Verify typing was recorded in editBuf
	if !strings.HasPrefix(m.editBuf, "abc") {
		t.Errorf("Expected editBuf to start with 'abc', got %q", m.editBuf)
	}

	// Step 2: Force a boundary by navigating down (commits the edit)
	downMsg := tea.KeyPressMsg{Code: tea.KeyDown}
	newModel, _ = m.Update(downMsg)
	m = newModel.(Model)

	// Verify the document now contains the typed text
	linesAfterType := m.GetLines()
	if len(linesAfterType) == 0 || !strings.HasPrefix(linesAfterType[0], "abc") {
		t.Errorf("Expected first line to start with 'abc' after typing, got %v", linesAfterType)
	}

	// Step 3: Press Ctrl+Z to undo
	undoMsg := tea.KeyPressMsg{Code: 'z', Mod: tea.ModCtrl}
	newModel, _ = m.Update(undoMsg)
	m = newModel.(Model)

	// CRITICAL ASSERTION: Document should be restored to original state
	linesAfterUndo := m.GetLines()
	if len(linesAfterUndo) == 0 {
		t.Fatal("Expected lines after undo, got empty")
	}
	if linesAfterUndo[0] != "# Header" {
		t.Errorf("UNDO FAILED: Expected first line to be '# Header' after undo, got %q", linesAfterUndo[0])
	}

	// Step 4: Press Ctrl+Y to redo
	redoMsg := tea.KeyPressMsg{Code: 'y', Mod: tea.ModCtrl}
	newModel, _ = m.Update(redoMsg)
	m = newModel.(Model)

	// CRITICAL ASSERTION: Document should have the typed text again
	linesAfterRedo := m.GetLines()
	if len(linesAfterRedo) == 0 {
		t.Fatal("Expected lines after redo, got empty")
	}
	if !strings.HasPrefix(linesAfterRedo[0], "abc") {
		t.Errorf("REDO FAILED: Expected first line to start with 'abc' after redo, got %q", linesAfterRedo[0])
	}

	// Step 5: Verify undo again works
	newModel, _ = m.Update(undoMsg)
	m = newModel.(Model)
	linesAfterUndo2 := m.GetLines()
	if linesAfterUndo2[0] != "# Header" {
		t.Errorf("UNDO (2nd) FAILED: Expected '# Header', got %q", linesAfterUndo2[0])
	}
}

// TestUndoStatusMessage verifies undo sets the status message.
// This catches the case where undo appears to work but doesn't update UI feedback.
func TestUndoStatusMessage(t *testing.T) {
	doc, _ := document.NewDocument("test\n")
	m := New(doc)

	// Type something
	newModel, _ := m.Update(tea.KeyPressMsg{Code: 'x', Text: "x"})
	m = newModel.(Model)

	// Navigate to commit
	downMsg := tea.KeyPressMsg{Code: tea.KeyDown}
	newModel, _ = m.Update(downMsg)
	m = newModel.(Model)

	// Undo
	undoMsg := tea.KeyPressMsg{Code: 'z', Mod: tea.ModCtrl}
	newModel, _ = m.Update(undoMsg)
	m = newModel.(Model)

	// Check status message
	if m.statusMsg != "Undo" {
		t.Errorf("Expected status message 'Undo', got %q", m.statusMsg)
	}
}

// TestUndoNothingToUndo verifies correct behavior when undo stack is empty.
func TestUndoNothingToUndo(t *testing.T) {
	doc, _ := document.NewDocument("test\n")
	m := New(doc)

	// Immediately try to undo without any edits
	undoMsg := tea.KeyPressMsg{Code: 'z', Mod: tea.ModCtrl}
	newModel, _ := m.Update(undoMsg)
	m = newModel.(Model)

	// Should show "Nothing to undo"
	if m.statusMsg != "Nothing to undo" {
		t.Errorf("Expected status 'Nothing to undo', got %q", m.statusMsg)
	}
}

// TestUndoAfterTypeAndEnter reproduces the bug where:
// 1. User types "hello" then Enter
// 2. User presses Ctrl+Z
// 3. BUG: Content becomes "hellohello" instead of being undone
// This test MUST FAIL until the bug is fixed.
func TestUndoAfterTypeAndEnter(t *testing.T) {
	// Start with empty document
	doc, _ := document.NewDocument("")
	m := New(doc)

	initialLines := m.GetLines()
	t.Logf("Initial lines: %v", initialLines)

	// Step 1: Type "hello"
	for _, ch := range "hello" {
		typeMsg := tea.KeyPressMsg{Code: ch, Text: string(ch)}
		newModel, _ := m.Update(typeMsg)
		m = newModel.(Model)
	}

	linesAfterHello := m.GetLines()
	t.Logf("After typing 'hello': %v, editBuf=%q", linesAfterHello, m.editBuf)

	// Step 2: Press Enter (this should commit "hello" and create newline)
	enterMsg := tea.KeyPressMsg{Code: tea.KeyEnter}
	newModel, _ := m.Update(enterMsg)
	m = newModel.(Model)

	linesAfterEnter := m.GetLines()
	t.Logf("After Enter: %v (total=%d)", linesAfterEnter, len(linesAfterEnter))

	// Verify state before undo
	if len(linesAfterEnter) < 2 {
		t.Fatalf("Expected at least 2 lines after Enter, got %d: %v", len(linesAfterEnter), linesAfterEnter)
	}
	if linesAfterEnter[0] != "hello" {
		t.Errorf("Expected first line to be 'hello', got %q", linesAfterEnter[0])
	}

	// Step 3: Press Ctrl+Z to undo
	undoMsg := tea.KeyPressMsg{Code: 'z', Mod: tea.ModCtrl}
	newModel, _ = m.Update(undoMsg)
	m = newModel.(Model)

	linesAfterUndo := m.GetLines()
	t.Logf("After Ctrl+Z: %v", linesAfterUndo)

	// CRITICAL BUG CHECK: Content should NOT be "hellohello"
	if len(linesAfterUndo) > 0 && strings.Contains(linesAfterUndo[0], "hellohello") {
		t.Errorf("BUG REPRODUCED: Undo duplicated content! Got %q", linesAfterUndo[0])
	}

	// After undo, we should be back to empty or have "hello" without newline
	// (depending on what the last committed batch was)
	// The key assertion is that content should NEVER be duplicated
	totalContent := strings.Join(linesAfterUndo, "\n")
	helloCount := strings.Count(totalContent, "hello")
	if helloCount > 1 {
		t.Errorf("BUG: 'hello' appears %d times after undo (should be 0 or 1). Content: %q", helloCount, totalContent)
	}
}

// TestUTF8TypeAndUndo verifies undo works correctly with multi-byte UTF-8 characters.
// This tests that cursorCol is handled as rune position, not byte position.
func TestUTF8TypeAndUndo(t *testing.T) {
	doc, _ := document.NewDocument("")
	m := New(doc)

	// Type "世界" (two 3-byte UTF-8 characters)
	for _, ch := range "世界" {
		typeMsg := tea.KeyPressMsg{Code: ch, Text: string(ch)}
		newModel, _ := m.Update(typeMsg)
		m = newModel.(Model)
	}

	t.Logf("After typing '世界': editBuf=%q, cursorCol=%d", m.editBuf, m.cursorCol)

	// editBuf should be "世界" (6 bytes, 2 runes)
	if m.editBuf != "世界" {
		t.Errorf("Expected editBuf='世界', got %q", m.editBuf)
	}

	// cursorCol should be 2 (2 characters), not 6 (6 bytes)
	if m.cursorCol != 2 {
		t.Errorf("Expected cursorCol=2 (rune position), got %d", m.cursorCol)
	}

	// Press Enter to commit and create undo point
	enterMsg := tea.KeyPressMsg{Code: tea.KeyEnter}
	newModel, _ := m.Update(enterMsg)
	m = newModel.(Model)

	linesAfterEnter := m.GetLines()
	t.Logf("After Enter: lines=%v", linesAfterEnter)

	if len(linesAfterEnter) < 1 || linesAfterEnter[0] != "世界" {
		t.Errorf("Expected first line='世界', got %v", linesAfterEnter)
	}

	// Undo the Enter
	undoMsg := tea.KeyPressMsg{Code: 'z', Mod: tea.ModCtrl}
	newModel, _ = m.Update(undoMsg)
	m = newModel.(Model)

	linesAfterUndo := m.GetLines()
	t.Logf("After Ctrl+Z: lines=%v", linesAfterUndo)

	// Should be back to "世界" on one line, not corrupted
	if len(linesAfterUndo) != 1 {
		t.Errorf("Expected 1 line after undo, got %d", len(linesAfterUndo))
	}
	if len(linesAfterUndo) > 0 && linesAfterUndo[0] != "世界" {
		t.Errorf("UTF-8 CORRUPTION: Expected '世界', got %q", linesAfterUndo[0])
	}
}

// TestRepeatedUndoRedoNoPanic verifies that pressing Ctrl+Z or Ctrl+Y repeatedly
// doesn't panic when the undo/redo stack is exhausted.
func TestRepeatedUndoRedoNoPanic(t *testing.T) {
	doc, _ := document.NewDocument("")
	m := New(doc)

	// Type something
	for _, ch := range "hello" {
		typeMsg := tea.KeyPressMsg{Code: ch, Text: string(ch)}
		newModel, _ := m.Update(typeMsg)
		m = newModel.(Model)
	}

	// Press Enter to commit
	enterMsg := tea.KeyPressMsg{Code: tea.KeyEnter}
	newModel, _ := m.Update(enterMsg)
	m = newModel.(Model)

	// Press Ctrl+Z many times (should not panic)
	undoMsg := tea.KeyPressMsg{Code: 'z', Mod: tea.ModCtrl}
	for range 20 {
		newModel, _ = m.Update(undoMsg)
		m = newModel.(Model)
	}

	// The undo stack should be exhausted, status should say "Nothing to undo"
	t.Logf("After 20 undos: statusMsg=%q", m.statusMsg)

	// Press Ctrl+Y many times (should not panic)
	redoMsg := tea.KeyPressMsg{Code: 'y', Mod: tea.ModCtrl}
	for range 20 {
		newModel, _ = m.Update(redoMsg)
		m = newModel.(Model)
	}

	// The redo stack should be exhausted, status should say "Nothing to redo"
	t.Logf("After 20 redos: statusMsg=%q", m.statusMsg)
}

// TestUndoRedoAlternating tests alternating undo/redo operations.
// This can trigger edge cases where cursor positions become invalid.
func TestUndoRedoAlternating(t *testing.T) {
	doc, _ := document.NewDocument("")
	m := New(doc)

	undoMsg := tea.KeyPressMsg{Code: 'z', Mod: tea.ModCtrl}
	redoMsg := tea.KeyPressMsg{Code: 'y', Mod: tea.ModCtrl}
	enterMsg := tea.KeyPressMsg{Code: tea.KeyEnter}

	// Type and enter multiple times
	for range 3 {
		for _, ch := range "hello" {
			typeMsg := tea.KeyPressMsg{Code: ch, Text: string(ch)}
			newModel, _ := m.Update(typeMsg)
			m = newModel.(Model)
		}
		newModel, _ := m.Update(enterMsg)
		m = newModel.(Model)
	}

	t.Logf("After typing: lines=%v", m.GetLines())

	// Undo, redo, undo, redo pattern
	for i := range 10 {
		newModel, _ := m.Update(undoMsg)
		m = newModel.(Model)
		t.Logf("After undo %d: lines=%d, status=%q", i, len(m.GetLines()), m.statusMsg)

		newModel, _ = m.Update(redoMsg)
		m = newModel.(Model)
		t.Logf("After redo %d: lines=%d, status=%q", i, len(m.GetLines()), m.statusMsg)
	}
}

// TestUTF8CursorMovement verifies cursor movement is rune-based, not byte-based.
