package editor

// undo_frontmatter_test.go — Tests for undo/redo with frontmatter documents.
// These tests check for the duplicate-lines-on-undo bug where pressing Ctrl+Z
// repeatedly causes frontmatter lines to appear doubled in the editor.

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/CalcMark/go-calcmark/spec/document"
)

// sendKey simulates a key press and returns the updated model.
func sendKey(t *testing.T, m tea.Model, keyName string) tea.Model {
	t.Helper()
	var msg tea.KeyPressMsg
	switch keyName {
	case "up":
		msg = tea.KeyPressMsg{Code: tea.KeyUp}
	case "down":
		msg = tea.KeyPressMsg{Code: tea.KeyDown}
	case "left":
		msg = tea.KeyPressMsg{Code: tea.KeyLeft}
	case "right":
		msg = tea.KeyPressMsg{Code: tea.KeyRight}
	case "enter":
		msg = tea.KeyPressMsg{Code: tea.KeyEnter}
	case "backspace":
		msg = tea.KeyPressMsg{Code: tea.KeyBackspace}
	case "home":
		msg = tea.KeyPressMsg{Code: tea.KeyHome}
	case "end":
		msg = tea.KeyPressMsg{Code: tea.KeyEnd}
	case "ctrl+z":
		msg = tea.KeyPressMsg{Code: 'z', Mod: tea.ModCtrl}
	case "ctrl+y":
		msg = tea.KeyPressMsg{Code: 'y', Mod: tea.ModCtrl}
	case "ctrl+f":
		msg = tea.KeyPressMsg{Code: 'f', Mod: tea.ModCtrl}
	case "shift+up":
		msg = tea.KeyPressMsg{Code: tea.KeyUp, Mod: tea.ModShift}
	case "shift+down":
		msg = tea.KeyPressMsg{Code: tea.KeyDown, Mod: tea.ModShift}
	case "shift+left":
		msg = tea.KeyPressMsg{Code: tea.KeyLeft, Mod: tea.ModShift}
	case "shift+right":
		msg = tea.KeyPressMsg{Code: tea.KeyRight, Mod: tea.ModShift}
	case "ctrl+a":
		msg = tea.KeyPressMsg{Code: 'a', Mod: tea.ModCtrl}
	default:
		t.Fatalf("sendKey: unrecognized key %q", keyName)
	}
	newM, _ := m.Update(msg)
	return newM
}

// typeText simulates typing each character.
func typeText(t *testing.T, m tea.Model, text string) tea.Model {
	t.Helper()
	for _, r := range text {
		msg := tea.KeyPressMsg{Code: r, Text: string(r)}
		newM, _ := m.Update(msg)
		m = newM
	}
	return m
}

// checkNoDuplicateLines verifies that no adjacent lines in GetLines() are identical.
// Returns the duplicated line content and true if duplication was found.
func checkNoDuplicateLines(m Model) (string, bool) {
	lines := m.GetLines()
	for i := 1; i < len(lines); i++ {
		if lines[i] == lines[i-1] && lines[i] != "" {
			return lines[i], true
		}
	}
	return "", false
}

// dumpLines returns a formatted string of all lines for debugging.
func dumpLines(m Model) string {
	lines := m.GetLines()
	var sb strings.Builder
	for i, line := range lines {
		sb.WriteString(strings.Repeat(" ", 2))
		if i < 9 {
			sb.WriteString(" ")
		}
		sb.WriteString(line)
		sb.WriteString("\n")
	}
	return sb.String()
}

// TestUndoFrontmatterNoDuplication_InsertThenUndo tests that inserting frontmatter
// and then undoing a previous edit does not produce duplicated lines.
//
// Scenario: User types on line 0, commits, inserts frontmatter (Ctrl+F),
// then presses Ctrl+Z. The undo operations reference pre-frontmatter line numbers,
// which now point to frontmatter lines instead of body lines.
func TestUndoFrontmatterNoDuplication_InsertThenUndo(t *testing.T) {
	content := "x = 10\ny = 20\nz = 30"
	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("NewDocument: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 24

	var model tea.Model = m

	// Step 1: Type "abc" on line 0
	model = typeText(t, model, "abc")

	// Step 2: Navigate down to commit the edit
	model = sendKey(t, model, "down")

	// Verify the edit was applied
	ed := model.(Model)
	lines := ed.GetLines()
	t.Logf("After typing+nav, lines[0]=%q totalLines=%d", lines[0], len(lines))
	if lines[0] != "abcx = 10" {
		t.Fatalf("Expected line 0 to be %q, got %q", "abcx = 10", lines[0])
	}

	// Step 3: Insert frontmatter (Ctrl+F)
	model = sendKey(t, model, "ctrl+f")

	ed = model.(Model)
	t.Logf("After Ctrl+F: totalLines=%d fmCount=%d", ed.TotalLines(), ed.frontmatterLineCount())

	// Verify frontmatter was inserted
	if ed.doc.GetFrontmatter() == nil {
		t.Fatal("Expected frontmatter to be present")
	}
	if ed.frontmatterLineCount() != 6 {
		t.Fatalf("Expected 6 frontmatter lines, got %d", ed.frontmatterLineCount())
	}

	// Step 4: Press Ctrl+Z to undo the previous typing
	model = sendKey(t, model, "ctrl+z")

	ed = model.(Model)
	t.Logf("After Ctrl+Z: totalLines=%d fmCount=%d", ed.TotalLines(), ed.frontmatterLineCount())

	// Check for duplicate lines
	if dupLine, hasDup := checkNoDuplicateLines(ed); hasDup {
		t.Errorf("DUPLICATE LINE FOUND after Ctrl+Z: %q", dupLine)
		t.Errorf("Full lines:\n%s", dumpLines(ed))
	}
}

// TestUndoFrontmatterNoDuplication_EditBodyThenUndo tests editing a body line
// in a frontmatter document and undoing.
func TestUndoFrontmatterNoDuplication_EditBodyThenUndo(t *testing.T) {
	content := "---\nexchange:\n  USD_EUR: 0.92\nglobals:\n  my_var: 42\n---\nx = my_var + 1"
	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("NewDocument: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 24

	var model tea.Model = m

	// Navigate to body line (line 6: "x = my_var + 1")
	for range 6 {
		model = sendKey(t, model, "down")
	}

	// Type "abc"
	model = typeText(t, model, "abc")

	// Navigate to commit
	model = sendKey(t, model, "down")

	// Undo
	model = sendKey(t, model, "ctrl+z")

	ed := model.(Model)
	t.Logf("After undo: totalLines=%d fmCount=%d", ed.TotalLines(), ed.frontmatterLineCount())

	if dupLine, hasDup := checkNoDuplicateLines(ed); hasDup {
		t.Errorf("DUPLICATE LINE FOUND: %q", dupLine)
	}
}

// TestUndoFrontmatterNoDuplication_EditFMLineThenUndo tests editing a frontmatter
// line and undoing.
func TestUndoFrontmatterNoDuplication_EditFMLineThenUndo(t *testing.T) {
	content := "---\nexchange:\n  USD_EUR: 0.92\nglobals:\n  my_var: 42\n---\nx = my_var + 1"
	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("NewDocument: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 24

	var model tea.Model = m

	// Navigate to frontmatter line 4 ("  my_var: 42")
	for range 4 {
		model = sendKey(t, model, "down")
	}

	// Go to end of line, type "3" (changes "42" area)
	model = sendKey(t, model, "end")
	model = typeText(t, model, "3")

	// Navigate to commit
	model = sendKey(t, model, "down")

	// Undo
	model = sendKey(t, model, "ctrl+z")

	ed := model.(Model)
	t.Logf("After undo: totalLines=%d fmCount=%d", ed.TotalLines(), ed.frontmatterLineCount())

	if dupLine, hasDup := checkNoDuplicateLines(ed); hasDup {
		t.Errorf("DUPLICATE LINE FOUND: %q", dupLine)
	}
}

// TestUndoFrontmatterNoDuplication_RepeatedUndo tests pressing Ctrl+Z multiple
// times on a frontmatter document.
func TestUndoFrontmatterNoDuplication_RepeatedUndo(t *testing.T) {
	content := "---\nexchange:\n  USD_EUR: 0.92\nglobals:\n  my_var: 42\n---\nx = 10"
	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("NewDocument: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 24

	var model tea.Model = m

	// Navigate to body line, type text, navigate to commit
	for range 6 {
		model = sendKey(t, model, "down")
	}
	model = typeText(t, model, "a")
	model = sendKey(t, model, "down")

	// Navigate to FM line, type text, navigate to commit
	for range 7 {
		model = sendKey(t, model, "up")
	}
	// Now on line 0 ("---")
	model = sendKey(t, model, "down") // line 1 "exchange:"
	model = sendKey(t, model, "down") // line 2 "  USD_EUR: 0.92"
	model = sendKey(t, model, "end")
	model = typeText(t, model, "1")
	model = sendKey(t, model, "down") // commit

	// Undo multiple times
	for i := range 5 {
		model = sendKey(t, model, "ctrl+z")

		ed := model.(Model)
		t.Logf("After undo %d: totalLines=%d fmCount=%d", i+1, ed.TotalLines(), ed.frontmatterLineCount())

		if dupLine, hasDup := checkNoDuplicateLines(ed); hasDup {
			t.Errorf("DUPLICATE LINE FOUND after undo %d: %q", i+1, dupLine)
			t.Errorf("Full lines:\n%s", dumpLines(ed))
		}
	}
}

// TestUndoFrontmatterNoDuplication_EnterOnFMLineThenUndo tests pressing Enter on
// frontmatter lines to create new lines, then undoing those Enter presses.
// This tests the OpInsertLine undo path for frontmatter lines and was the primary
// reproduction case for the duplicate-lines-on-undo bug.
func TestUndoFrontmatterNoDuplication_EnterOnFMLineThenUndo(t *testing.T) {
	content := "---\nexchange:\n  USD_EUR: 0.92\nglobals:\n  my_var: 42\n---\nx = 10"
	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("NewDocument: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 24

	var model tea.Model = m

	// Navigate to line 1 ("exchange:"), go to end, press Enter
	model = sendKey(t, model, "down")
	model = sendKey(t, model, "end")
	model = sendKey(t, model, "enter")

	// Navigate to "  USD_EUR: 0.92" line, go to end, press Enter
	model = sendKey(t, model, "down") // USD_EUR line
	model = sendKey(t, model, "end")
	model = sendKey(t, model, "enter")

	// Now undo both Enter presses
	for i := range 4 {
		model = sendKey(t, model, "ctrl+z")

		ed := model.(Model)
		t.Logf("After undo %d: totalLines=%d fmCount=%d statusMsg=%q",
			i+1, ed.TotalLines(), ed.frontmatterLineCount(), ed.statusMsg)

		if dupLine, hasDup := checkNoDuplicateLines(ed); hasDup {
			t.Errorf("DUPLICATE LINE FOUND after undo %d: %q", i+1, dupLine)
			break
		}
	}
}

// TestUndoFrontmatterNoDuplication_InsertFMEditFMThenUndo tests inserting frontmatter
// via Ctrl+F, editing frontmatter lines, then undoing.
func TestUndoFrontmatterNoDuplication_InsertFMEditFMThenUndo(t *testing.T) {
	content := "x = 10\ny = 20\nz = 30"
	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("NewDocument: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 24

	var model tea.Model = m

	// Insert frontmatter
	model = sendKey(t, model, "ctrl+f")

	// Now edit a frontmatter line (cursor is at line 2: "  USD_EUR: 0.92")
	model = sendKey(t, model, "end")
	model = typeText(t, model, "x")
	model = sendKey(t, model, "down") // commit

	// Edit another FM line
	model = sendKey(t, model, "down") // line 4: "  my_var: 42"
	model = sendKey(t, model, "end")
	model = typeText(t, model, "y")
	model = sendKey(t, model, "down") // commit

	// Undo multiple times
	for i := range 6 {
		model = sendKey(t, model, "ctrl+z")

		ed := model.(Model)
		t.Logf("After undo %d: totalLines=%d fmCount=%d statusMsg=%q",
			i+1, ed.TotalLines(), ed.frontmatterLineCount(), ed.statusMsg)

		if dupLine, hasDup := checkNoDuplicateLines(ed); hasDup {
			t.Errorf("DUPLICATE LINE FOUND after undo %d: %q", i+1, dupLine)
			break
		}
	}
}

// TestUndoFrontmatterNoDuplication_InsertFMAndMultipleUndo tests inserting frontmatter
// after body edits and pressing Ctrl+Z multiple times.
func TestUndoFrontmatterNoDuplication_InsertFMAndMultipleUndo(t *testing.T) {
	content := "x = 10\ny = 20\nz = 30"
	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("NewDocument: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 24

	var model tea.Model = m

	// Type on each line to create multiple undo batches
	model = typeText(t, model, "a")
	model = sendKey(t, model, "down") // commit + move to line 1
	model = typeText(t, model, "b")
	model = sendKey(t, model, "down") // commit + move to line 2
	model = typeText(t, model, "c")
	model = sendKey(t, model, "down") // commit

	// Insert frontmatter
	model = sendKey(t, model, "ctrl+f")

	// Press Ctrl+Z multiple times
	for i := range 5 {
		model = sendKey(t, model, "ctrl+z")

		ed := model.(Model)
		t.Logf("After undo %d: totalLines=%d fmCount=%d statusMsg=%q",
			i+1, ed.TotalLines(), ed.frontmatterLineCount(), ed.statusMsg)

		if dupLine, hasDup := checkNoDuplicateLines(ed); hasDup {
			t.Errorf("DUPLICATE LINE FOUND after undo %d: %q", i+1, dupLine)
			t.Errorf("Full lines:\n%s", dumpLines(ed))
			break
		}
	}
}
