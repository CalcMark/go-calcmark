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
	case "cmd+a":
		msg = tea.KeyPressMsg{Code: 'a', Mod: tea.ModSuper}
	case "shift+up":
		msg = tea.KeyPressMsg{Code: tea.KeyUp, Mod: tea.ModShift}
	case "shift+down":
		msg = tea.KeyPressMsg{Code: tea.KeyDown, Mod: tea.ModShift}
	case "shift+left":
		msg = tea.KeyPressMsg{Code: tea.KeyLeft, Mod: tea.ModShift}
	case "shift+right":
		msg = tea.KeyPressMsg{Code: tea.KeyRight, Mod: tea.ModShift}
	case "shift+home":
		msg = tea.KeyPressMsg{Code: tea.KeyHome, Mod: tea.ModShift}
	case "shift+end":
		msg = tea.KeyPressMsg{Code: tea.KeyEnd, Mod: tea.ModShift}
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
	if ed.frontmatterLineCount() != 10 {
		t.Fatalf("Expected 10 frontmatter lines, got %d", ed.frontmatterLineCount())
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
	content := "---\nexchange:\n  USD_EUR: 0.92\nglobals:\n  my_var: 42\n---\nx = @globals.my_var + 1"
	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("NewDocument: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 24

	var model tea.Model = m

	// Navigate to body line (line 6: "x = @globals.my_var + 1")
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
	content := "---\nexchange:\n  USD_EUR: 0.92\nglobals:\n  my_var: 42\n---\nx = @globals.my_var + 1"
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

// TestUndoFrontmatter_BackspaceDeleteFMLinesAndUndo tests that deleting
// frontmatter line content via backspace and then undoing restores all lines.
//
// Bug: Ctrl+F → navigate to first line → backspace to delete "---" →
// navigate down (commits + structural change) → join with previous →
// navigate down (commits). Pressing Ctrl+Z twice should restore the
// post-Ctrl+F state. Pressing Ctrl+Z a third time should remove frontmatter.
//
// Root cause: editBuf was loaded before redetectBlockTypes() in handleUndo,
// so when the document rebuild changed line numbering (splitting embedded
// newlines), the stale editBuf was flushed to the wrong line on the next undo.
func TestUndoFrontmatter_BackspaceDeleteFMLinesAndUndo(t *testing.T) {
	content := "x = 10"
	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("NewDocument: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 24

	var model tea.Model = m
	originalLines := m.GetLines()

	// Step 1: Insert frontmatter via Ctrl+F
	model = sendKey(t, model, "ctrl+f")
	ed := model.(Model)
	if ed.frontmatterLineCount() != 10 {
		t.Fatalf("Expected 10 FM lines after Ctrl+F, got %d", ed.frontmatterLineCount())
	}
	afterInsertLines := ed.GetLines()
	t.Logf("After Ctrl+F: %q", afterInsertLines)

	// Step 2: Navigate to line 0 ("---"), end of line
	model = sendKey(t, model, "up")  // line 1
	model = sendKey(t, model, "up")  // line 0
	model = sendKey(t, model, "end") // col 3

	// Step 3: Backspace 3 times to delete "---"
	model = sendKey(t, model, "backspace")
	model = sendKey(t, model, "backspace")
	model = sendKey(t, model, "backspace")
	ed = model.(Model)
	if ed.editBuf != "" {
		t.Fatalf("Expected empty editBuf after deleting '---', got %q", ed.editBuf)
	}

	// Step 4: Navigate down to commit + trigger structural change
	// (editBuf "" saved to frontmatter line 0 → removes opening "---" → no frontmatter)
	model = sendKey(t, model, "down")

	// Step 5: Backspace at BOL to join with previous (empty) line
	model = sendKey(t, model, "home")
	model = sendKey(t, model, "backspace")

	// Step 6: Navigate down to commit the join
	model = sendKey(t, model, "down")

	ed = model.(Model)
	t.Logf("After all edits: lines=%q", ed.GetLines())

	// Undo stack (3 batches, from top):
	//   1. [OpReplace(join)]  — the line join
	//   2. [OpDelete×3]       — deleting "---"
	//   3. [OpDocReplace]     — Ctrl+F insertion

	// Undo 1: reverse the join
	model = sendKey(t, model, "ctrl+z")
	ed = model.(Model)
	t.Logf("After undo 1 (join): lines=%q", ed.GetLines())

	// Undo 2: restore "---" → should fully restore post-Ctrl+F state
	model = sendKey(t, model, "ctrl+z")
	ed = model.(Model)
	t.Logf("After undo 2 (---): lines=%q", ed.GetLines())

	restoredLines := ed.GetLines()
	if len(restoredLines) != len(afterInsertLines) {
		t.Errorf("After 2 undos: line count %d, want %d", len(restoredLines), len(afterInsertLines))
	}
	for i := range min(len(restoredLines), len(afterInsertLines)) {
		if restoredLines[i] != afterInsertLines[i] {
			t.Errorf("After 2 undos: line %d = %q, want %q", i, restoredLines[i], afterInsertLines[i])
		}
	}

	// Undo 3: reverse Ctrl+F → back to original (no frontmatter)
	model = sendKey(t, model, "ctrl+z")
	ed = model.(Model)
	t.Logf("After undo 3 (Ctrl+F): lines=%q", ed.GetLines())

	finalLines := ed.GetLines()
	if len(finalLines) != len(originalLines) {
		t.Errorf("After 3 undos: line count %d, want %d", len(finalLines), len(originalLines))
	}
	for i := range min(len(finalLines), len(originalLines)) {
		if finalLines[i] != originalLines[i] {
			t.Errorf("After 3 undos: line %d = %q, want %q", i, finalLines[i], originalLines[i])
		}
	}

	// Undo 4: should be "Nothing to undo"
	model = sendKey(t, model, "ctrl+z")
	ed = model.(Model)
	if ed.statusMsg != "Nothing to undo" {
		t.Errorf("Expected 'Nothing to undo' after 4 undos, got %q", ed.statusMsg)
	}
}

// TestUndoFrontmatter_BackspaceWithinFMLineAndUndo tests the simpler case:
// delete characters within a single frontmatter line via backspace, then undo.
// The edit is committed via navigation, then undo should restore.
func TestUndoFrontmatter_BackspaceWithinFMLineAndUndo(t *testing.T) {
	content := "x = 10"
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

	// Navigate to line 0 (---), go to end
	model = sendKey(t, model, "up")
	model = sendKey(t, model, "up")
	model = sendKey(t, model, "end")

	// Delete "---" via backspace
	model = sendKey(t, model, "backspace")
	model = sendKey(t, model, "backspace")
	model = sendKey(t, model, "backspace")

	// Navigate to commit the edit (saveCurrentLineAndMoveTo)
	model = sendKey(t, model, "down")

	ed := model.(Model)
	lines := ed.GetLines()
	t.Logf("After deleting '---' and commit: lines=%q fmCount=%d",
		lines, ed.frontmatterLineCount())

	// Undo — should restore "---"
	model = sendKey(t, model, "ctrl+z")
	ed = model.(Model)
	lines = ed.GetLines()
	t.Logf("After undo: lines=%q statusMsg=%q fmCount=%d",
		lines, ed.statusMsg, ed.frontmatterLineCount())

	if ed.statusMsg == "Nothing to undo" {
		t.Error("Expected undo to work, got 'Nothing to undo'")
	}
	if len(lines) == 0 || lines[0] != "---" {
		t.Errorf("Line 0 after undo: got %q, want %q", lines[0], "---")
	}
}

// TestUndoFrontmatter_CtrlFThenCtrlZRemovesFrontmatter tests that inserting
// frontmatter via Ctrl+F is undoable via Ctrl+Z.
//
// Bug: Open empty editor → Ctrl+F to insert frontmatter → Ctrl+Z shows
// "Nothing to undo" and the frontmatter remains. Expected: frontmatter is removed.
func TestUndoFrontmatter_CtrlFThenCtrlZRemovesFrontmatter(t *testing.T) {
	content := "x = 10\ny = 20\nz = 30"
	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("NewDocument: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 24

	var model tea.Model = m

	// Verify: no frontmatter initially
	ed := model.(Model)
	if ed.doc.GetFrontmatter() != nil {
		t.Fatal("Expected no frontmatter initially")
	}
	originalLines := ed.GetLines()
	t.Logf("Before Ctrl+F: lines=%q", originalLines)

	// Insert frontmatter via Ctrl+F
	model = sendKey(t, model, "ctrl+f")
	ed = model.(Model)
	if ed.doc.GetFrontmatter() == nil {
		t.Fatal("Expected frontmatter after Ctrl+F")
	}
	if ed.statusMsg != "Frontmatter inserted" {
		t.Fatalf("Expected status %q, got %q", "Frontmatter inserted", ed.statusMsg)
	}
	t.Logf("After Ctrl+F: totalLines=%d fmCount=%d", ed.TotalLines(), ed.frontmatterLineCount())

	// Undo the frontmatter insertion via Ctrl+Z
	model = sendKey(t, model, "ctrl+z")
	ed = model.(Model)
	t.Logf("After Ctrl+Z: totalLines=%d fmCount=%d statusMsg=%q", ed.TotalLines(), ed.frontmatterLineCount(), ed.statusMsg)

	// The undo should NOT say "Nothing to undo"
	if ed.statusMsg == "Nothing to undo" {
		t.Error("Ctrl+Z after Ctrl+F said 'Nothing to undo' — frontmatter insertion was not recorded in undo history")
	}

	// After undo, frontmatter should be removed
	if ed.doc.GetFrontmatter() != nil && ed.frontmatterLineCount() > 0 {
		t.Error("Expected frontmatter to be removed after Ctrl+Z")
	}

	// Lines should match the original document
	afterLines := ed.GetLines()
	if len(afterLines) != len(originalLines) {
		t.Errorf("Line count mismatch: got %d, want %d", len(afterLines), len(originalLines))
		t.Logf("After undo lines: %q", afterLines)
	}
	for i := range min(len(afterLines), len(originalLines)) {
		if afterLines[i] != originalLines[i] {
			t.Errorf("Line %d: got %q, want %q", i, afterLines[i], originalLines[i])
		}
	}
}

// TestUndoFrontmatter_CtrlFThenCtrlZRestoresCursor tests that cursor position
// is restored after undoing frontmatter insertion.
func TestUndoFrontmatter_CtrlFThenCtrlZRestoresCursor(t *testing.T) {
	content := "x = 10\ny = 20"
	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("NewDocument: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 24

	var model tea.Model = m

	// Remember initial cursor position
	ed := model.(Model)
	initialLine, initialCol := ed.cursorLine, ed.cursorCol

	// Insert frontmatter
	model = sendKey(t, model, "ctrl+f")

	// Undo
	model = sendKey(t, model, "ctrl+z")
	ed = model.(Model)

	// Cursor should be restored to initial position
	if ed.cursorLine != initialLine || ed.cursorCol != initialCol {
		t.Errorf("Cursor after undo: got (%d,%d), want (%d,%d)",
			ed.cursorLine, ed.cursorCol, initialLine, initialCol)
	}
}

// TestUndoFrontmatter_CtrlFThenCtrlZThenCtrlYRedoes tests that Ctrl+Y redoes
// the frontmatter insertion after undoing it.
func TestUndoFrontmatter_CtrlFThenCtrlZThenCtrlYRedoes(t *testing.T) {
	content := "x = 10"
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
	ed := model.(Model)
	fmLineCount := ed.frontmatterLineCount()
	if fmLineCount == 0 {
		t.Fatal("Expected frontmatter after Ctrl+F")
	}

	// Undo
	model = sendKey(t, model, "ctrl+z")
	ed = model.(Model)
	if ed.frontmatterLineCount() != 0 {
		t.Error("Expected no frontmatter after undo")
	}

	// Redo
	model = sendKey(t, model, "ctrl+y")
	ed = model.(Model)
	t.Logf("After redo: totalLines=%d fmCount=%d statusMsg=%q",
		ed.TotalLines(), ed.frontmatterLineCount(), ed.statusMsg)

	if ed.statusMsg == "Nothing to redo" {
		t.Error("Ctrl+Y after Ctrl+Z said 'Nothing to redo'")
	}
	if ed.frontmatterLineCount() != fmLineCount {
		t.Errorf("After redo: fmLineCount=%d, want %d", ed.frontmatterLineCount(), fmLineCount)
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
