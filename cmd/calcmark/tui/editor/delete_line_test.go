package editor

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	implDoc "github.com/CalcMark/go-calcmark/v2/impl/document"
	"github.com/CalcMark/go-calcmark/v2/spec/document"
)

// TestDeleteLineDeletesCurrentLine verifies Ctrl+K deletes the line the cursor is on.
func TestDeleteLineDeletesCurrentLine(t *testing.T) {
	doc, _ := document.NewDocument("line one\nline two\nline three\n")
	m := New(doc)

	// Cursor starts at line 0
	if m.cursorLine != 0 {
		t.Fatalf("Expected cursor at line 0, got %d", m.cursorLine)
	}

	lines := m.GetLines()
	if len(lines) != 3 {
		t.Fatalf("Expected 3 lines, got %d", len(lines))
	}

	// Delete line 0 ("line one")
	newModel, _ := m.Update(tea.KeyPressMsg{Code: 'k', Mod: tea.ModCtrl})
	m = newModel.(Model)

	lines = m.GetLines()
	if len(lines) != 2 {
		t.Fatalf("Expected 2 lines after delete, got %d: %v", len(lines), lines)
	}
	if lines[0] != "line two" {
		t.Errorf("Expected first line to be 'line two', got %q", lines[0])
	}
	if lines[1] != "line three" {
		t.Errorf("Expected second line to be 'line three', got %q", lines[1])
	}
}

// TestDeleteLineMiddle verifies deleting a middle line leaves cursor on the next line.
func TestDeleteLineMiddle(t *testing.T) {
	doc, _ := document.NewDocument("a = 1\nb = 2\nc = 3\n")
	m := New(doc)

	// Move cursor to line 1 ("b = 2")
	newModel, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	m = newModel.(Model)
	if m.cursorLine != 1 {
		t.Fatalf("Expected cursor at line 1, got %d", m.cursorLine)
	}

	// Delete line 1
	newModel, _ = m.Update(tea.KeyPressMsg{Code: 'k', Mod: tea.ModCtrl})
	m = newModel.(Model)

	lines := m.GetLines()
	if len(lines) != 2 {
		t.Fatalf("Expected 2 lines, got %d: %v", len(lines), lines)
	}
	if lines[0] != "a = 1" {
		t.Errorf("Expected first line 'a = 1', got %q", lines[0])
	}
	if lines[1] != "c = 3" {
		t.Errorf("Expected second line 'c = 3', got %q", lines[1])
	}
}

// TestDeleteLineWithPendingEdits verifies Ctrl+K deletes the correct line
// when the user has typed uncommitted text in the edit buffer.
func TestDeleteLineWithPendingEdits(t *testing.T) {
	doc, _ := document.NewDocument("original\nkeep me\n")
	m := New(doc)

	// Type something to modify the first line (uncommitted)
	for _, r := range "modified" {
		newModel, _ := m.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
		m = newModel.(Model)
	}

	// Ctrl+K should delete the current line (with modified content), not "keep me"
	newModel, _ := m.Update(tea.KeyPressMsg{Code: 'k', Mod: tea.ModCtrl})
	m = newModel.(Model)

	lines := m.GetLines()
	if len(lines) != 1 {
		t.Fatalf("Expected 1 line after delete, got %d: %v", len(lines), lines)
	}
	if lines[0] != "keep me" {
		t.Errorf("Expected remaining line 'keep me', got %q", lines[0])
	}
}

// TestDeleteLineUndo verifies Ctrl+Z restores a line deleted by Ctrl+K.
func TestDeleteLineUndo(t *testing.T) {
	doc, _ := document.NewDocument("a = 1\nb = 2\nc = 3\n")
	m := New(doc)

	// Move to line 1 ("b = 2")
	newModel, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	m = newModel.(Model)

	// Delete line 1
	newModel, _ = m.Update(tea.KeyPressMsg{Code: 'k', Mod: tea.ModCtrl})
	m = newModel.(Model)

	lines := m.GetLines()
	if len(lines) != 2 {
		t.Fatalf("Expected 2 lines after delete, got %d", len(lines))
	}

	// Undo the deletion
	newModel, _ = m.Update(tea.KeyPressMsg{Code: 'z', Mod: tea.ModCtrl})
	m = newModel.(Model)

	lines = m.GetLines()
	if len(lines) != 3 {
		t.Fatalf("Expected 3 lines after undo, got %d: %v", len(lines), lines)
	}
	if lines[0] != "a = 1" {
		t.Errorf("Expected line 0 'a = 1', got %q", lines[0])
	}
	if lines[1] != "b = 2" {
		t.Errorf("Expected line 1 'b = 2', got %q", lines[1])
	}
	if lines[2] != "c = 3" {
		t.Errorf("Expected line 2 'c = 3', got %q", lines[2])
	}
}

// TestDeleteLineReEvaluates verifies the interpreter runs immediately after Ctrl+K.
// Deleting a variable assignment should cause dependent expressions to show errors.
func TestDeleteLineReEvaluates(t *testing.T) {
	doc, _ := document.NewDocument("x = 10\ny = x * 2\n")
	m := New(doc)

	// Set up evaluator
	m.eval = implDoc.NewEvaluator()
	_ = m.eval.Evaluate(m.doc)

	// Verify x is defined before deletion
	env := m.eval.GetEnvironment()
	if _, ok := env.Get("x"); !ok {
		t.Fatal("Expected variable 'x' to be defined before deletion")
	}

	// Delete line 0 ("x = 10")
	newModel, _ := m.Update(tea.KeyPressMsg{Code: 'k', Mod: tea.ModCtrl})
	m = newModel.(Model)

	// After deletion, the document should only have "y = x * 2"
	lines := m.GetLines()
	if len(lines) != 1 {
		t.Fatalf("Expected 1 line, got %d: %v", len(lines), lines)
	}
	if lines[0] != "y = x * 2" {
		t.Errorf("Expected 'y = x * 2', got %q", lines[0])
	}

	// The evaluator should have re-run: x is no longer defined,
	// so y = x * 2 should produce an error
	env = m.eval.GetEnvironment()
	if _, ok := env.Get("x"); ok {
		t.Error("Expected variable 'x' to be undefined after deleting its line")
	}
}

// TestDeleteLastLine verifies deleting the last line moves cursor up.
func TestDeleteLastLine(t *testing.T) {
	doc, _ := document.NewDocument("first\nsecond\n")
	m := New(doc)

	// Move to last line
	newModel, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	m = newModel.(Model)
	if m.cursorLine != 1 {
		t.Fatalf("Expected cursor at line 1, got %d", m.cursorLine)
	}

	// Delete last line
	newModel, _ = m.Update(tea.KeyPressMsg{Code: 'k', Mod: tea.ModCtrl})
	m = newModel.(Model)

	lines := m.GetLines()
	if len(lines) != 1 {
		t.Fatalf("Expected 1 line, got %d: %v", len(lines), lines)
	}
	if lines[0] != "first" {
		t.Errorf("Expected 'first', got %q", lines[0])
	}
	if m.cursorLine != 0 {
		t.Errorf("Expected cursor at line 0 after deleting last line, got %d", m.cursorLine)
	}
}
