package editor

import (
	tea "charm.land/bubbletea/v2"
	"testing"
)

// TestCtrlArrowEmptyEditor tests CTRL+Arrow navigation after typing in an empty editor
// Bug repro: Open empty editor, type "test this", CTRL+Arrow doesn't work
func TestCtrlArrowEmptyEditor(t *testing.T) {
	// Start with a completely empty editor
	m := New(nil)

	// Type "test this"
	text := "test this"
	for _, ch := range text {
		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
		m = newModel.(Model)
	}

	// Cursor should be at position 9 (end of "test this")
	if m.cursorCol != 9 {
		t.Errorf("After typing 'test this', cursor at col %d, want 9", m.cursorCol)
	}

	// Debug: Check what's in editBuf
	t.Logf("Before CTRL+Left: editBuf=%q, cursorCol=%d", m.editBuf, m.cursorCol)

	// Test CTRL+Left - should move to start of "this" (col 5)
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlLeft})
	m = newModel.(Model)

	t.Logf("After CTRL+Left: editBuf=%q, cursorCol=%d", m.editBuf, m.cursorCol)

	if m.cursorCol != 5 {
		t.Errorf("After CTRL+Left, cursor at col %d, want 5 (start of 'this')", m.cursorCol)
	}

	// Test CTRL+Left again - should move to start of line (col 0)
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlLeft})
	m = newModel.(Model)

	if m.cursorCol != 0 {
		t.Errorf("After second CTRL+Left, cursor at col %d, want 0", m.cursorCol)
	}

	// Test CTRL+Right - most editors move to START of next word (col 5)
	// But some move to END of current word (col 4)
	// Let's check what actually happens
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlRight})
	m = newModel.(Model)

	t.Logf("After CTRL+Right from col 0: cursorCol=%d", m.cursorCol)

	// The standard behavior is to move to START of next word
	if m.cursorCol != 5 {
		t.Errorf("After CTRL+Right, cursor at col %d, want 5 (start of 'this')", m.cursorCol)
	}

	// Test CTRL+Right again - should move to end of line (col 9)
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlRight})
	m = newModel.(Model)

	if m.cursorCol != 9 {
		t.Errorf("After second CTRL+Right, cursor at col %d, want 9", m.cursorCol)
	}
}
