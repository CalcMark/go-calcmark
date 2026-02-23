package editor

import (
	"testing"

	"github.com/CalcMark/go-calcmark/spec/document"
	tea "github.com/charmbracelet/bubbletea"
)

func TestEditModeEnterCreatesNewLine(t *testing.T) {
	// Create a document with 2 lines
	content := `a = 1
b = 2`
	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 24

	// Position cursor at end of first line
	m.cursorLine = 0
	lines := m.GetLines()
	m.cursorCol = len(lines[0]) // End of line
	m.loadCurrentLineIntoEditBuffer()

	t.Logf("Before Enter: cursorLine=%d, cursorCol=%d, editBuf=%q, totalLines=%d",
		m.cursorLine, m.cursorCol, m.editBuf, m.TotalLines())

	// Press Enter at end of line - should create new line below
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = result.(Model)

	t.Logf("After Enter: cursorLine=%d, cursorCol=%d, editBuf=%q, totalLines=%d, mode=%v",
		m.cursorLine, m.cursorCol, m.editBuf, m.TotalLines(), m.mode)

	// Should have moved to line 1, still in edit mode
	if m.mode != StateDefault {
		t.Errorf("Expected StateDefault after Enter, got %v", m.mode)
	}
	if m.cursorLine != 1 {
		t.Errorf("Expected cursorLine 1, got %d", m.cursorLine)
	}
	// New line should be empty (cursor was at end)
	if m.editBuf != "" {
		t.Errorf("Expected empty editBuf for new line, got %q", m.editBuf)
	}
	// Total lines should have increased
	if m.TotalLines() != 3 {
		t.Errorf("Expected 3 total lines, got %d", m.TotalLines())
	}
}

func TestEditModeNavigationAndEnterTogether(t *testing.T) {
	content := `a = 1
b = 2`
	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 24

	// Type a character that won't trigger autocomplete (punctuation)
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'#'}})
	m = result.(Model)

	// Navigate down to line 1 (ensure we're in StateDefault for navigation)
	if m.mode != StateDefault {
		// Dismiss any autocomplete that may have been triggered
		result, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
		m = result.(Model)
	}
	result, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = result.(Model)

	if m.cursorLine != 1 || m.mode != StateDefault {
		t.Fatalf("Expected line 1 in default mode, got line %d mode %v", m.cursorLine, m.mode)
	}

	t.Logf("On line 1: editBuf=%q", m.editBuf)

	// Press Enter to add new line after line 1
	result, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = result.(Model)

	t.Logf("After Enter: cursorLine=%d, editBuf=%q, totalLines=%d", m.cursorLine, m.editBuf, m.TotalLines())

	if m.mode != StateDefault {
		t.Errorf("Expected StateDefault, got %v", m.mode)
	}
	if m.cursorLine != 2 {
		t.Errorf("Expected cursorLine 2, got %d", m.cursorLine)
	}
	if m.TotalLines() != 3 {
		t.Errorf("Expected 3 total lines, got %d", m.TotalLines())
	}

	// Navigate back up
	result, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = result.(Model)

	if m.cursorLine != 1 || m.mode != StateDefault {
		t.Errorf("Expected line 1 in edit mode after Up, got line %d mode %v", m.cursorLine, m.mode)
	}
}
