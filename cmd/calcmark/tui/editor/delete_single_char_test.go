package editor

import (
	"testing"

	"github.com/CalcMark/go-calcmark/spec/document"
	tea "github.com/charmbracelet/bubbletea"
)

// TestDeleteSingleChar_TypeThenDelete verifies that DELETE key can delete
// the last remaining single character on a line after typing and moving left.
// Regression test for: "I still cannot delete the last remaining single character"
func TestDeleteSingleChar_TypeThenDelete(t *testing.T) {
	doc, err := document.NewDocument("")
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 24
	m.previewMode = PreviewFull

	// Type a single character
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m = m2.(Model)

	if m.cursorCol != 1 || m.editBuf != "a" {
		t.Fatalf("After typing 'a': expected cursorCol=1, editBuf='a', got cursorCol=%d, editBuf=%q",
			m.cursorCol, m.editBuf)
	}

	// Move cursor left to position on the character
	m3, _ := m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	m = m3.(Model)

	if m.cursorCol != 0 {
		t.Fatalf("After LEFT: expected cursorCol=0, got %d", m.cursorCol)
	}

	// Press DELETE - should delete the 'a'
	m4, _ := m.Update(tea.KeyMsg{Type: tea.KeyDelete})
	m = m4.(Model)

	if m.editBuf != "" {
		t.Errorf("Expected editBuf='' after DELETE, got %q", m.editBuf)
	}
}

// TestDeleteSingleChar_AtEndJoinsLines verifies that DELETE at end of line
// joins with the next line (standard editor behavior).
func TestDeleteSingleChar_AtEndJoinsLines(t *testing.T) {
	doc, err := document.NewDocument("a\nline2")
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 24
	m.previewMode = PreviewFull

	// Move to end of line (after 'a')
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnd})
	m = m2.(Model)

	if m.cursorCol != 1 {
		t.Fatalf("Expected cursorCol=1 at end of 'a', got %d", m.cursorCol)
	}

	// Press DELETE - this should join lines
	m3, _ := m.Update(tea.KeyMsg{Type: tea.KeyDelete})
	m = m3.(Model)

	if m.editBuf != "aline2" {
		t.Errorf("Expected editBuf='aline2' after DELETE at end, got %q", m.editBuf)
	}
}

// TestDeleteSingleChar_InitialCursor verifies DELETE works when cursor starts
// at position 0 (no prior typing). This was the core bug where transitionToEditing
// would reload editBuf after the deletion.
func TestDeleteSingleChar_InitialCursor(t *testing.T) {
	doc, err := document.NewDocument("a")
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 24
	m.previewMode = PreviewFull

	// Cursor starts at position 0, on the 'a' - no typing happened
	if m.cursorCol != 0 {
		t.Fatalf("Expected cursorCol=0 initially, got %d", m.cursorCol)
	}

	// Press DELETE - should delete 'a' at position 0
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyDelete})
	m = m2.(Model)

	if m.editBuf != "" {
		t.Errorf("Expected editBuf='' after DELETE at position 0, got %q", m.editBuf)
	}
}

// TestDeleteSingleChar_AfterLineJoin reproduces the exact user scenario:
// 1. Line ends with text, next line has single char 'a'
// 2. DELETE at end of first line joins lines
// 3. DELETE again should delete the 'a' that was joined
// User report: "the 'a' was previously on the next line, then I pressed DELETE a couple of times"
func TestDeleteSingleChar_AfterLineJoin(t *testing.T) {
	// Setup: "## Same Type, One Empty Line\na\nx = 10"
	doc, err := document.NewDocument("## Same Type, One Empty Line\na\nx = 10")
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 24
	m.previewMode = PreviewFull

	t.Logf("Initial lines: %v", m.GetLines())

	// Move to end of first line
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnd})
	m = m2.(Model)

	t.Logf("After END: cursorLine=%d, cursorCol=%d, editBuf=%q",
		m.cursorLine, m.cursorCol, m.editBuf)

	// First DELETE - joins with next line (the "a" line)
	m3, _ := m.Update(tea.KeyMsg{Type: tea.KeyDelete})
	m = m3.(Model)

	t.Logf("After first DELETE (join): cursorCol=%d, editBuf=%q, lines=%v",
		m.cursorCol, m.editBuf, m.GetLines())

	// editBuf should now be "## Same Type, One Empty Linea"
	expectedAfterJoin := "## Same Type, One Empty Linea"
	if m.editBuf != expectedAfterJoin {
		t.Fatalf("After first DELETE: expected editBuf=%q, got %q", expectedAfterJoin, m.editBuf)
	}

	// Cursor should be at the position where join happened (on the 'a')
	expectedCursorCol := len("## Same Type, One Empty Line")
	if m.cursorCol != expectedCursorCol {
		t.Fatalf("After join: expected cursorCol=%d, got %d", expectedCursorCol, m.cursorCol)
	}

	// Second DELETE - should delete the 'a'
	m4, _ := m.Update(tea.KeyMsg{Type: tea.KeyDelete})
	m = m4.(Model)

	t.Logf("After second DELETE: cursorCol=%d, editBuf=%q, lines=%v",
		m.cursorCol, m.editBuf, m.GetLines())

	// editBuf should now be "## Same Type, One Empty Line" (without the 'a')
	expectedAfterDelete := "## Same Type, One Empty Line"
	if m.editBuf != expectedAfterDelete {
		t.Errorf("After second DELETE: expected editBuf=%q, got %q", expectedAfterDelete, m.editBuf)
	}
}

// TestDeleteSingleChar_AfterLineJoinWithDebounce simulates what happens in the real TUI:
// The debounce timer fires between DELETE operations, which may reset state.
func TestDeleteSingleChar_AfterLineJoinWithDebounce(t *testing.T) {
	doc, err := document.NewDocument("## Same Type, One Empty Line\na\nx = 10")
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 24
	m.previewMode = PreviewFull

	// Move to end of first line
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnd})
	m = m2.(Model)

	// First DELETE - joins with next line
	m3, _ := m.Update(tea.KeyMsg{Type: tea.KeyDelete})
	m = m3.(Model)

	t.Logf("After first DELETE: cursorCol=%d, editBuf=%q", m.cursorCol, m.editBuf)

	// Simulate debounce timer firing (this happens in real TUI after 100ms)
	// The debounce message triggers transitionToProcessing which saves and re-evaluates
	snapshot := m.editBuf
	m4, _ := m.Update(evalDebounceMsg{editBufSnapshot: snapshot})
	m = m4.(Model)

	t.Logf("After debounce: cursorCol=%d, editBuf=%q, lines=%v",
		m.cursorCol, m.editBuf, m.GetLines())

	// NOW try to delete the 'a' - this is where the bug likely occurs
	m5, _ := m.Update(tea.KeyMsg{Type: tea.KeyDelete})
	m = m5.(Model)

	t.Logf("After second DELETE: cursorCol=%d, editBuf=%q, lines=%v",
		m.cursorCol, m.editBuf, m.GetLines())

	// editBuf should now be "## Same Type, One Empty Line" (without the 'a')
	expectedAfterDelete := "## Same Type, One Empty Line"
	if m.editBuf != expectedAfterDelete {
		t.Errorf("After second DELETE: expected editBuf=%q, got %q", expectedAfterDelete, m.editBuf)
	}
}

// TestDeleteSingleChar_WithClearedEditBuf tests the scenario where editBuf has been
// cleared (e.g., by some state transition) before DELETE is pressed.
// This might be the actual bug scenario.
func TestDeleteSingleChar_WithClearedEditBuf(t *testing.T) {
	// Create document with the joined line already present
	doc, err := document.NewDocument("## Same Type, One Empty Linea\nx = 10")
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 24
	m.previewMode = PreviewFull

	// Position cursor at the 'a' (position 28)
	m.cursorCol = len("## Same Type, One Empty Line")

	// IMPORTANT: editBuf is empty at this point (no typing has occurred)
	t.Logf("Initial: cursorCol=%d, editBuf=%q, lines=%v",
		m.cursorCol, m.editBuf, m.GetLines())

	if m.editBuf != "" {
		t.Fatalf("Expected editBuf to be empty initially, got %q", m.editBuf)
	}

	// Now press DELETE - should delete the 'a' at position 28
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyDelete})
	m = m2.(Model)

	t.Logf("After DELETE: cursorCol=%d, editBuf=%q, lines=%v",
		m.cursorCol, m.editBuf, m.GetLines())

	// editBuf should now be "## Same Type, One Empty Line" (without the 'a')
	expectedAfterDelete := "## Same Type, One Empty Line"
	if m.editBuf != expectedAfterDelete {
		t.Errorf("After DELETE: expected editBuf=%q, got %q", expectedAfterDelete, m.editBuf)
	}
}

// TestDeleteSingleChar_CursorAtEnd tests DELETE when cursor is at the END of the line
// (after the 'a', not ON the 'a'). This triggers line joining, not character deletion.
func TestDeleteSingleChar_CursorAtEnd(t *testing.T) {
	// Create document with the joined line
	doc, err := document.NewDocument("## Same Type, One Empty Linea\nx = 10")
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 24
	m.previewMode = PreviewFull

	// Position cursor at END of line (position 29, AFTER the 'a')
	m.cursorCol = len("## Same Type, One Empty Linea") // 29

	t.Logf("Initial: cursorCol=%d, lineLen=%d, editBuf=%q",
		m.cursorCol, len(m.GetLines()[0]), m.editBuf)

	// Press DELETE - this will try to JOIN with next line, not delete the 'a'
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyDelete})
	m = m2.(Model)

	t.Logf("After DELETE: cursorCol=%d, editBuf=%q, lines=%v",
		m.cursorCol, m.editBuf, m.GetLines())
}

// TestDeleteSingleChar_FullUserScenario reproduces the EXACT user scenario:
// 1. Line "## Same Type, One Empty Line"
// 2. Next line has just "a"
// 3. User presses END to go to end of first line
// 4. User presses DELETE to join (makes "## Same Type, One Empty Linea")
// 5. User waits (debounce fires)
// 6. User presses DELETE again - should delete the 'a' but doesn't
func TestDeleteSingleChar_FullUserScenario(t *testing.T) {
	doc, err := document.NewDocument("## Same Type, One Empty Line\na\nx = 10")
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 24
	m.previewMode = PreviewFull

	t.Logf("Step 0 - Initial: cursorLine=%d, cursorCol=%d, lines=%v",
		m.cursorLine, m.cursorCol, m.GetLines())

	// Step 1: Press END to go to end of first line
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnd})
	m = m2.(Model)
	t.Logf("Step 1 - After END: cursorLine=%d, cursorCol=%d, editBuf=%q",
		m.cursorLine, m.cursorCol, m.editBuf)

	// Step 2: Press DELETE to join with "a" line
	m3, _ := m.Update(tea.KeyMsg{Type: tea.KeyDelete})
	m = m3.(Model)
	t.Logf("Step 2 - After DELETE (join): cursorLine=%d, cursorCol=%d, editBuf=%q, lines=%v",
		m.cursorLine, m.cursorCol, m.editBuf, m.GetLines())

	// Step 3: Debounce fires (simulate 100ms wait)
	snapshot := m.editBuf
	m4, _ := m.Update(evalDebounceMsg{editBufSnapshot: snapshot})
	m = m4.(Model)
	t.Logf("Step 3 - After debounce: cursorLine=%d, cursorCol=%d, editBuf=%q, lines=%v",
		m.cursorLine, m.cursorCol, m.editBuf, m.GetLines())

	// CRITICAL CHECK: After debounce, what's the state?
	// - cursorCol should still be 28 (on the 'a')
	// - editBuf should contain the joined line
	// - OR editBuf might be cleared, requiring reload

	// Step 4: Press DELETE again - THIS IS WHERE THE BUG MIGHT BE
	m5, _ := m.Update(tea.KeyMsg{Type: tea.KeyDelete})
	m = m5.(Model)
	t.Logf("Step 4 - After second DELETE: cursorLine=%d, cursorCol=%d, editBuf=%q, lines=%v",
		m.cursorLine, m.cursorCol, m.editBuf, m.GetLines())

	// The 'a' should be deleted
	expectedLine := "## Same Type, One Empty Line"
	if m.editBuf != expectedLine {
		t.Errorf("Expected editBuf=%q, got %q", expectedLine, m.editBuf)
	}
}
