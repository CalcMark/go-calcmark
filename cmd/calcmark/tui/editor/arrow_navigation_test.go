package editor

import (
	"testing"

	"github.com/CalcMark/go-calcmark/spec/document"
	tea "github.com/charmbracelet/bubbletea"
)

// TestArrowKeyNavigationAcrossLines tests that left/right arrow keys
// can move between lines when at the start/end of a line.
func TestArrowKeyNavigationAcrossLines(t *testing.T) {
	source := "line 1\n\nline 3"
	doc, _ := document.NewDocument(source)
	m := New(doc)
	m.width = 80
	m.height = 24

	// Start on line 1 (middle line, which is empty)
	m.cursorLine = 1
	m.loadCurrentLineIntoEditBuffer()

	t.Logf("Starting: cursorLine=%d, cursorCol=%d, editBuf=%q", m.cursorLine, m.cursorCol, m.editBuf)

	// Line 1 is empty, so editBuf should be ""
	if m.editBuf != "" {
		t.Errorf("Expected empty editBuf, got %q", m.editBuf)
	}

	// Test: Left arrow from empty line should move to end of previous line
	t.Log("\n=== Testing Left Arrow from empty line ===")
	updatedModel, _ := m.handleLeftKey()
	m = updatedModel.(Model)
	t.Logf("After Left: cursorLine=%d, cursorCol=%d, editBuf=%q", m.cursorLine, m.cursorCol, m.editBuf)

	if m.cursorLine != 0 {
		t.Errorf("Left from line 1 should move to line 0, got line %d", m.cursorLine)
	}
	if m.editBuf != "line 1" {
		t.Errorf("Expected editBuf 'line 1', got %q", m.editBuf)
	}
	if m.cursorCol != len("line 1") {
		t.Errorf("Expected cursorCol at end (%d), got %d", len("line 1"), m.cursorCol)
	}

	// Test: Right arrow from end of line should move to start of next line
	t.Log("\n=== Testing Right Arrow from end of line ===")
	updatedModel, _ = m.handleRightKey()
	m = updatedModel.(Model)
	t.Logf("After Right: cursorLine=%d, cursorCol=%d, editBuf=%q", m.cursorLine, m.cursorCol, m.editBuf)

	if m.cursorLine != 1 {
		t.Errorf("Right from end of line 0 should move to line 1, got line %d", m.cursorLine)
	}
	if m.cursorCol != 0 {
		t.Errorf("Expected cursorCol at start (0), got %d", m.cursorCol)
	}

	// Test: Right arrow from start of empty line should move to next line
	t.Log("\n=== Testing Right Arrow from empty line ===")
	updatedModel, _ = m.handleRightKey()
	m = updatedModel.(Model)
	t.Logf("After Right: cursorLine=%d, cursorCol=%d, editBuf=%q", m.cursorLine, m.cursorCol, m.editBuf)

	if m.cursorLine != 2 {
		t.Errorf("Right from empty line 1 should move to line 2, got line %d", m.cursorLine)
	}
	if m.editBuf != "line 3" {
		t.Errorf("Expected editBuf 'line 3', got %q", m.editBuf)
	}
	if m.cursorCol != 0 {
		t.Errorf("Expected cursorCol at start (0), got %d", m.cursorCol)
	}

	t.Log("\n✓ Arrow key navigation across lines works correctly")
}

// TestArrowKeyNavigationWithKeyMsg tests arrow keys using actual KeyMsg events.
func TestArrowKeyNavigationWithKeyMsg(t *testing.T) {
	source := "abc\n\ndef"
	doc, _ := document.NewDocument(source)
	m := New(doc)
	m.width = 80
	m.height = 24

	// Start on line 0, at end of "abc"
	m.cursorLine = 0
	m.loadCurrentLineIntoEditBuffer()
	m.cursorCol = len("abc")

	t.Logf("Starting: cursorLine=%d, cursorCol=%d, editBuf=%q", m.cursorLine, m.cursorCol, m.editBuf)

	// Send Right arrow key
	msg := tea.KeyMsg{Type: tea.KeyRight}
	updatedModel, _ := m.Update(msg)
	m = updatedModel.(Model)

	t.Logf("After Right: cursorLine=%d, cursorCol=%d, editBuf=%q", m.cursorLine, m.cursorCol, m.editBuf)

	if m.cursorLine != 1 {
		t.Errorf("Expected to move to line 1, got line %d", m.cursorLine)
	}
	if m.cursorCol != 0 {
		t.Errorf("Expected cursorCol=0, got %d", m.cursorCol)
	}

	// Send Left arrow key (should go back)
	msg = tea.KeyMsg{Type: tea.KeyLeft}
	updatedModel, _ = m.Update(msg)
	m = updatedModel.(Model)

	t.Logf("After Left: cursorLine=%d, cursorCol=%d, editBuf=%q", m.cursorLine, m.cursorCol, m.editBuf)

	if m.cursorLine != 0 {
		t.Errorf("Expected to move back to line 0, got line %d", m.cursorLine)
	}
	if m.cursorCol != len("abc") {
		t.Errorf("Expected cursorCol=%d, got %d", len("abc"), m.cursorCol)
	}

	t.Log("✓ Arrow keys work correctly with KeyMsg events")
}
