package editor

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/CalcMark/go-calcmark/spec/document"
)

func TestEditModeUpDownNavigation(t *testing.T) {
	// Create a document with 3 lines
	content := `a = 1
b = 2
c = 3`
	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 24

	// Verify we start in normal mode on line 0
	if m.mode != ModeNormal {
		t.Errorf("Expected ModeNormal, got %v", m.mode)
	}
	if m.cursorLine != 0 {
		t.Errorf("Expected cursorLine 0, got %d", m.cursorLine)
	}

	// Enter edit mode (press 'e')
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	m = result.(Model)

	t.Logf("After 'e': mode=%v, cursorLine=%d, editBuf=%q", m.mode, m.cursorLine, m.editBuf)

	if m.mode != ModeEditing {
		t.Fatalf("Expected ModeEditing after 'e', got %v", m.mode)
	}
	if m.cursorLine != 0 {
		t.Errorf("Expected cursorLine 0, got %d", m.cursorLine)
	}
	if m.editBuf != "a = 1" {
		t.Errorf("Expected editBuf 'a = 1', got %q", m.editBuf)
	}

	// Press Down arrow - should move to line 1 while staying in edit mode
	result, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = result.(Model)

	t.Logf("After Down: mode=%v, cursorLine=%d, editBuf=%q", m.mode, m.cursorLine, m.editBuf)

	if m.mode != ModeEditing {
		t.Errorf("Expected ModeEditing after Down, got %v", m.mode)
	}
	if m.cursorLine != 1 {
		t.Errorf("Expected cursorLine 1 after Down, got %d", m.cursorLine)
	}
	if m.editBuf != "b = 2" {
		t.Errorf("Expected editBuf 'b = 2', got %q", m.editBuf)
	}

	// Press Down arrow again - should move to line 2 while staying in edit mode
	result, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = result.(Model)

	t.Logf("After Down 2: mode=%v, cursorLine=%d, editBuf=%q", m.mode, m.cursorLine, m.editBuf)

	if m.mode != ModeEditing {
		t.Errorf("Expected ModeEditing after second Down, got %v", m.mode)
	}
	if m.cursorLine != 2 {
		t.Errorf("Expected cursorLine 2 after second Down, got %d", m.cursorLine)
	}
	if m.editBuf != "c = 3" {
		t.Errorf("Expected editBuf 'c = 3', got %q", m.editBuf)
	}

	// Press Up arrow - should move back to line 1 while staying in edit mode
	result, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = result.(Model)

	t.Logf("After Up: mode=%v, cursorLine=%d, editBuf=%q", m.mode, m.cursorLine, m.editBuf)

	if m.mode != ModeEditing {
		t.Errorf("Expected ModeEditing after Up, got %v", m.mode)
	}
	if m.cursorLine != 1 {
		t.Errorf("Expected cursorLine 1 after Up, got %d", m.cursorLine)
	}
	if m.editBuf != "b = 2" {
		t.Errorf("Expected editBuf 'b = 2', got %q", m.editBuf)
	}
}

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

	// Enter edit mode on first line
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	m = result.(Model)

	if m.mode != ModeEditing {
		t.Fatalf("Expected ModeEditing, got %v", m.mode)
	}

	t.Logf("Before Enter: cursorLine=%d, cursorCol=%d, editBuf=%q, totalLines=%d",
		m.cursorLine, m.cursorCol, m.editBuf, m.TotalLines())

	// Press Enter at end of line - should create new line below
	result, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = result.(Model)

	t.Logf("After Enter: cursorLine=%d, cursorCol=%d, editBuf=%q, totalLines=%d, mode=%v",
		m.cursorLine, m.cursorCol, m.editBuf, m.TotalLines(), m.mode)

	// Should have moved to line 1, still in edit mode
	if m.mode != ModeEditing {
		t.Errorf("Expected ModeEditing after Enter, got %v", m.mode)
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

func TestEditModeEnterSplitsLine(t *testing.T) {
	content := `hello world`
	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 24

	// Enter edit mode
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	m = result.(Model)

	// Move cursor to middle (after "hello ")
	m.cursorCol = 6

	t.Logf("Before Enter: cursorLine=%d, cursorCol=%d, editBuf=%q",
		m.cursorLine, m.cursorCol, m.editBuf)

	// Press Enter - should split line
	result, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = result.(Model)

	t.Logf("After Enter: cursorLine=%d, cursorCol=%d, editBuf=%q, totalLines=%d",
		m.cursorLine, m.cursorCol, m.editBuf, m.TotalLines())

	// Should be on line 1 with "world"
	if m.cursorLine != 1 {
		t.Errorf("Expected cursorLine 1, got %d", m.cursorLine)
	}
	if m.editBuf != "world" {
		t.Errorf("Expected editBuf 'world', got %q", m.editBuf)
	}
	if m.cursorCol != 0 {
		t.Errorf("Expected cursorCol 0, got %d", m.cursorCol)
	}
	if m.TotalLines() != 2 {
		t.Errorf("Expected 2 total lines, got %d", m.TotalLines())
	}

	// Line 0 should now be "hello "
	lines := m.GetLines()
	if lines[0] != "hello " {
		t.Errorf("Expected first line 'hello ', got %q", lines[0])
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

	// Enter edit mode on line 0
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	m = result.(Model)

	// Navigate down to line 1
	result, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = result.(Model)

	if m.cursorLine != 1 || m.mode != ModeEditing {
		t.Fatalf("Expected line 1 in edit mode, got line %d mode %v", m.cursorLine, m.mode)
	}

	t.Logf("On line 1: editBuf=%q", m.editBuf)

	// Press Enter to add new line after line 1
	result, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = result.(Model)

	t.Logf("After Enter: cursorLine=%d, editBuf=%q, totalLines=%d", m.cursorLine, m.editBuf, m.TotalLines())

	if m.mode != ModeEditing {
		t.Errorf("Expected ModeEditing, got %v", m.mode)
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

	if m.cursorLine != 1 || m.mode != ModeEditing {
		t.Errorf("Expected line 1 in edit mode after Up, got line %d mode %v", m.cursorLine, m.mode)
	}
}
