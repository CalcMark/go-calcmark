package editor

// cut_paste_test.go — Tests for cut, paste, and selection-delete operations,
// with a focus on interactions around YAML frontmatter boundaries.
//
// Architecture note: the editor uses lazy persistence. After editing operations,
// editBuf holds the live state but the document is NOT updated until the user
// navigates (triggering saveCurrentLine). Tests that call the underlying API
// directly must call commitEditBuf() to persist. Tests using key simulation
// use navigation steps (down/up) to trigger the natural persistence path.

import (
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/CalcMark/go-calcmark/spec/document"
)

// --- helpers ----------------------------------------------------------------

// newEditorWithContent creates a Model from raw text, ready for testing.
func newEditorWithContent(t *testing.T, content string) Model {
	t.Helper()
	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("NewDocument: %v", err)
	}
	m := New(doc)
	m.width = 80
	m.height = 24
	m.previewMode = PreviewFull
	return m
}

// commitEditBuf persists the editBuf to the document and re-evaluates.
// Required after direct API calls (DeleteSelection, simulatePaste) since the
// editor normally persists lazily on navigation.
func commitEditBuf(m *Model) {
	if m.editBufLoaded {
		m.updateCurrentLine(m.editBuf)
		m.redetectBlockTypes()
		m.reEvaluate()
	}
}

// linesOf returns the document lines as a slice.
func linesOf(m Model) []string {
	return m.GetLines()
}

// assertLines checks that the document contains exactly the expected lines.
func assertLines(t *testing.T, m Model, expected []string) {
	t.Helper()
	got := linesOf(m)
	if len(got) != len(expected) {
		t.Errorf("line count: got %d, want %d\n  got:  %q\n  want: %q",
			len(got), len(expected), got, expected)
		return
	}
	for i := range got {
		if got[i] != expected[i] {
			t.Errorf("line %d: got %q, want %q", i, got[i], expected[i])
		}
	}
}

// assertNoDuplicateLines verifies no adjacent non-empty lines are identical.
func assertNoDuplicateLines(t *testing.T, m Model) {
	t.Helper()
	lines := linesOf(m)
	for i := 1; i < len(lines); i++ {
		if lines[i] == lines[i-1] && lines[i] != "" {
			t.Errorf("duplicate adjacent lines at %d/%d: %q\nall lines: %q",
				i-1, i, lines[i], lines)
		}
	}
}

// assertCursor verifies cursor position.
func assertCursor(t *testing.T, m Model, line, col int) {
	t.Helper()
	if m.cursorLine != line || m.cursorCol != col {
		t.Errorf("cursor: got (%d,%d), want (%d,%d)",
			m.cursorLine, m.cursorCol, line, col)
	}
}

// selectRange sets up a selection from (anchorLine, anchorCol) to (cursorLine, cursorCol).
func selectRange(m *Model, anchorLine, anchorCol, cursorLine, cursorCol int) {
	m.selectionAnchorLine = anchorLine
	m.selectionAnchorCol = anchorCol
	m.cursorLine = cursorLine
	m.cursorCol = cursorCol
	m.editBuf = m.GetLines()[m.cursorLine]
	m.editBufLoaded = true
}

// simulatePaste simulates a paste of text at the current cursor position,
// bypassing the system clipboard. Mirrors handlePaste logic then persists.
func simulatePaste(m *Model, text string) {
	if m.HasSelection() {
		m.DeleteSelection()
		commitEditBuf(m)
	}
	m.undoManager.ForceBoundary()
	lines := strings.Split(text, "\n")
	if len(lines) == 1 {
		m.insertTextAtCursor(lines[0])
	} else {
		m.insertMultiLineText(lines)
	}
	m.undoManager.ForceBoundary()
	m.modified = true
	commitEditBuf(m)
}

// commitViaNavigation triggers document persistence by pressing Enter then Undo.
// Enter always persists the edit buffer (it calls updateCurrentLine directly),
// and Undo reverses the line split. This works regardless of document size,
// unlike down/up which are no-ops on single-line documents.
func commitViaNavigation(t *testing.T, model tea.Model) tea.Model {
	t.Helper()
	ed := model.(Model)
	totalBefore := ed.TotalLines()

	// Enter always commits + splits the line
	model = sendKey(t, model, "enter")

	ed = model.(Model)
	totalAfter := ed.TotalLines()

	// Undo the Enter to restore original line count
	if totalAfter > totalBefore {
		model = sendKey(t, model, "ctrl+z")
	}
	return model
}

// --- Same-line selection and deletion ---------------------------------------

func TestCutPaste_SameLineSelect(t *testing.T) {
	m := newEditorWithContent(t, "hello world\nsecond line")

	// Select "hello" (cols 0–5 on line 0)
	selectRange(&m, 0, 0, 0, 5)

	text := m.GetSelectedText()
	if text != "hello" {
		t.Fatalf("selected text: got %q, want %q", text, "hello")
	}

	deleted, _ := m.DeleteSelection()
	if deleted != "hello" {
		t.Fatalf("deleted text: got %q, want %q", deleted, "hello")
	}
	commitEditBuf(&m)

	assertCursor(t, m, 0, 0)
	assertLines(t, m, []string{" world", "second line"})
}

func TestCutPaste_SameLinePasteBack(t *testing.T) {
	m := newEditorWithContent(t, "hello world\nsecond line")

	// Select and delete "world" (cols 6–11)
	selectRange(&m, 0, 6, 0, 11)
	deleted, _ := m.DeleteSelection()
	commitEditBuf(&m)
	assertLines(t, m, []string{"hello ", "second line"})

	// Paste it back at cursor (col 6)
	assertCursor(t, m, 0, 6)
	simulatePaste(&m, deleted)

	assertLines(t, m, []string{"hello world", "second line"})
	assertCursor(t, m, 0, 11)
}

// --- Multi-line selection and deletion --------------------------------------

func TestCutPaste_MultiLineSelect(t *testing.T) {
	m := newEditorWithContent(t, "line one\nline two\nline three")

	// Select from line 0 col 5 to line 2 col 4 → "one\nline two\nline"
	selectRange(&m, 0, 5, 2, 4)

	text := m.GetSelectedText()
	if text != "one\nline two\nline" {
		t.Fatalf("selected text: got %q, want %q", text, "one\nline two\nline")
	}

	deleted, _ := m.DeleteSelection()
	if deleted != "one\nline two\nline" {
		t.Fatalf("deleted text: got %q, want %q", deleted, "one\nline two\nline")
	}
	commitEditBuf(&m)

	// After deletion: "line " + " three" merged into one line
	assertLines(t, m, []string{"line  three"})
	assertCursor(t, m, 0, 5)
}

func TestCutPaste_MultiLinePasteBack(t *testing.T) {
	m := newEditorWithContent(t, "aaa\nbbb\nccc")

	// Select from end of line 0 to end of line 1: "\nbbb"
	selectRange(&m, 0, 3, 1, 3)

	deleted, _ := m.DeleteSelection()
	if deleted != "\nbbb" {
		t.Fatalf("deleted text: got %q, want %q", deleted, "\nbbb")
	}
	commitEditBuf(&m)

	// "aaa" + "ccc" should be merged
	assertLines(t, m, []string{"aaa", "ccc"})

	// Paste back at end of line 0
	m.cursorLine = 0
	m.cursorCol = 3
	m.editBuf = m.GetLines()[0]
	m.editBufLoaded = true

	simulatePaste(&m, deleted)
	assertLines(t, m, []string{"aaa", "bbb", "ccc"})
}

func TestCutPaste_SelectAllDeleteAndPasteBack(t *testing.T) {
	original := "x = 10\ny = 20\nz = 30"
	m := newEditorWithContent(t, original)

	m.SelectAll()
	if !m.HasSelection() {
		t.Fatal("expected selection after SelectAll")
	}

	text := m.GetSelectedText()
	if text != original {
		t.Fatalf("SelectAll text: got %q, want %q", text, original)
	}

	m.DeleteSelection()
	commitEditBuf(&m)

	// Paste back
	simulatePaste(&m, text)
	assertLines(t, m, []string{"x = 10", "y = 20", "z = 30"})
	assertNoDuplicateLines(t, m)
}

// --- Frontmatter: selection spanning FM boundary ----------------------------

func TestCutPaste_SelectAcrossFrontmatterBoundary(t *testing.T) {
	content := "---\nglobals:\n  my_var: 42\n---\nx = @globals.my_var + 1"
	m := newEditorWithContent(t, content)

	fmCount := m.frontmatterLineCount()
	if fmCount != 4 {
		t.Fatalf("frontmatter line count: got %d, want 4", fmCount)
	}

	// Select from inside frontmatter (line 2 col 2) to body (line 4 col 5)
	selectRange(&m, 2, 2, 4, 5)

	text := m.GetSelectedText()
	expected := "my_var: 42\n---\nx = @"
	if text != expected {
		t.Fatalf("cross-boundary text: got %q, want %q", text, expected)
	}

	deleted, _ := m.DeleteSelection()
	if deleted != expected {
		t.Fatalf("deleted text: got %q, want %q", deleted, expected)
	}
	commitEditBuf(&m)

	assertNoDuplicateLines(t, m)
}

func TestCutPaste_CutEntireFrontmatter(t *testing.T) {
	content := "---\nglobals:\n  my_var: 42\n---\nx = 10"
	m := newEditorWithContent(t, content)

	// Select all frontmatter lines (0–3): "---\nglobals:\n  my_var: 42\n---"
	selectRange(&m, 0, 0, 3, 3)

	text := m.GetSelectedText()
	expected := "---\nglobals:\n  my_var: 42\n---"
	if text != expected {
		t.Fatalf("text: got %q, want %q", text, expected)
	}

	m.DeleteSelection()
	commitEditBuf(&m)
	assertNoDuplicateLines(t, m)

	lines := linesOf(m)
	t.Logf("lines after cut: %q", lines)
	if len(lines) == 0 {
		t.Fatal("document is empty after cutting frontmatter")
	}
}

// --- Frontmatter: paste into frontmatter ------------------------------------

func TestCutPaste_PasteSingleLineIntoFrontmatter(t *testing.T) {
	content := "---\nglobals:\n  my_var: 42\n---\nx = @globals.my_var + 1"
	m := newEditorWithContent(t, content)

	// Position cursor at end of "  my_var: 42" (line 2)
	m.cursorLine = 2
	line := m.GetLines()[2]
	m.cursorCol = runeLen(line)
	m.editBuf = line
	m.editBufLoaded = true

	simulatePaste(&m, "3")

	lines := linesOf(m)
	assertNoDuplicateLines(t, m)
	t.Logf("after paste into FM: %q", lines)

	// Line 2 should end with "423"
	if !strings.Contains(lines[2], "423") {
		t.Errorf("paste into frontmatter: line 2 = %q, expected '423'", lines[2])
	}
}

func TestCutPaste_PasteMultiLineIntoFrontmatter(t *testing.T) {
	content := "---\nglobals:\n  my_var: 42\n---\nx = 10"
	m := newEditorWithContent(t, content)

	// Position cursor at end of "globals:" (line 1, col 8)
	m.cursorLine = 1
	m.cursorCol = 8
	m.editBuf = m.GetLines()[1]
	m.editBufLoaded = true

	simulatePaste(&m, "\n  rate: 0.5\n  tax: 10")

	lines := linesOf(m)
	assertNoDuplicateLines(t, m)
	t.Logf("after paste: %q", lines)

	found := false
	for _, l := range lines {
		if strings.Contains(l, "rate: 0.5") {
			found = true
			break
		}
	}
	if !found {
		t.Error("pasted line 'rate: 0.5' not found in document")
	}
}

// --- Frontmatter: cut from body, paste into FM and vice versa ---------------

func TestCutPaste_CutBodyPasteIntoFrontmatter(t *testing.T) {
	content := "---\nglobals:\n  my_var: 42\n---\nhello world"
	m := newEditorWithContent(t, content)

	// Select "hello" from the body line (line 4, cols 0–5)
	selectRange(&m, 4, 0, 4, 5)
	deleted, _ := m.DeleteSelection()
	if deleted != "hello" {
		t.Fatalf("deleted: got %q, want %q", deleted, "hello")
	}
	commitEditBuf(&m)

	// Paste "hello" at beginning of my_var line (line 2)
	lines := linesOf(m)
	m.cursorLine = 2
	m.cursorCol = 0
	m.editBuf = lines[2]
	m.editBufLoaded = true

	simulatePaste(&m, deleted)

	lines = linesOf(m)
	assertNoDuplicateLines(t, m)
	t.Logf("after paste into FM: %q", lines)

	if !strings.HasPrefix(lines[2], "hello") {
		t.Errorf("line 2: got %q, expected prefix 'hello'", lines[2])
	}
}

func TestCutPaste_CutFrontmatterPasteIntoBody(t *testing.T) {
	content := "---\nglobals:\n  my_var: 42\n---\nx = 10"
	m := newEditorWithContent(t, content)

	// Select "my_var: 42" from frontmatter (line 2, cols 2–12)
	selectRange(&m, 2, 2, 2, 12)
	deleted, _ := m.DeleteSelection()
	if deleted != "my_var: 42" {
		t.Fatalf("deleted: got %q, want %q", deleted, "my_var: 42")
	}
	commitEditBuf(&m)

	// Find the body line and paste at its end
	lines := linesOf(m)
	bodyIdx := -1
	for i, l := range lines {
		if strings.Contains(l, "x = 10") {
			bodyIdx = i
			break
		}
	}
	if bodyIdx < 0 {
		t.Fatalf("body line 'x = 10' not found; lines: %q", lines)
	}

	m.cursorLine = bodyIdx
	m.cursorCol = runeLen(lines[bodyIdx])
	m.editBuf = lines[bodyIdx]
	m.editBufLoaded = true

	simulatePaste(&m, deleted)

	lines = linesOf(m)
	assertNoDuplicateLines(t, m)
	t.Logf("after paste into body: %q", lines)

	found := false
	for _, l := range lines {
		if strings.Contains(l, "my_var: 42") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("pasted text not found in body; lines: %q", lines)
	}
}

// --- Selection via key simulation -------------------------------------------

func TestCutPaste_ShiftSelectThenDelete(t *testing.T) {
	m := newEditorWithContent(t, "abcdef\nghijkl")

	var model tea.Model = m

	// Shift+Right 3 times to select "abc"
	for range 3 {
		model = sendKey(t, model, "shift+right")
	}

	ed := model.(Model)
	if !ed.HasSelection() {
		t.Fatal("expected selection after shift+right")
	}

	text := ed.GetSelectedText()
	if text != "abc" {
		t.Fatalf("selected: got %q, want %q", text, "abc")
	}

	// Press backspace to delete selection, then navigate to persist
	model = sendKey(t, model, "backspace")
	model = commitViaNavigation(t, model)
	ed = model.(Model)

	lines := linesOf(ed)
	if lines[0] != "def" {
		t.Errorf("line 0 after delete: got %q, want %q", lines[0], "def")
	}
	assertNoDuplicateLines(t, ed)
}

func TestCutPaste_ShiftSelectMultiLineThenDelete(t *testing.T) {
	m := newEditorWithContent(t, "aaa\nbbb\nccc")

	var model tea.Model = m

	// Move to col 1, then shift+down to select across lines
	model = sendKey(t, model, "right")
	model = sendKey(t, model, "shift+down")

	ed := model.(Model)
	if !ed.HasSelection() {
		t.Fatal("expected selection")
	}

	text := ed.GetSelectedText()
	if text != "aa\nb" {
		t.Fatalf("selected: got %q, want %q", text, "aa\nb")
	}

	// Delete and persist
	model = sendKey(t, model, "backspace")
	model = commitViaNavigation(t, model)
	ed = model.(Model)

	lines := linesOf(ed)
	if lines[0] != "abb" {
		t.Errorf("line 0: got %q, want %q", lines[0], "abb")
	}
	assertNoDuplicateLines(t, ed)
}

func TestCutPaste_SelectAllThenType(t *testing.T) {
	m := newEditorWithContent(t, "x = 10\ny = 20")

	var model tea.Model = m

	// Ctrl+A then type replacement
	model = sendKey(t, model, "ctrl+a")
	model = typeText(t, model, "z = 30")

	// Verify editBuf has the typed text (immediate state)
	ed := model.(Model)
	if ed.editBuf != "z = 30" {
		t.Errorf("editBuf: got %q, want %q", ed.editBuf, "z = 30")
	}

	// Navigate to persist editBuf → document
	model = commitViaNavigation(t, model)
	ed = model.(Model)

	assertNoDuplicateLines(t, ed)

	// After persistence, document should reflect the typed text
	lines := linesOf(ed)
	t.Logf("after commit: editBuf=%q lines=%q", ed.editBuf, lines)
	if !slices.Contains(lines, "z = 30") {
		t.Errorf("replacement text not found; lines: %q", lines)
	}
}

// --- Frontmatter: key-simulated selection/deletion around FM ----------------

func TestCutPaste_SelectAllWithFrontmatter(t *testing.T) {
	content := "---\nglobals:\n  my_var: 42\n---\nx = 10"
	m := newEditorWithContent(t, content)

	var model tea.Model = m

	model = sendKey(t, model, "ctrl+a")
	ed := model.(Model)
	if !ed.HasSelection() {
		t.Fatal("expected selection after ctrl+a")
	}

	text := ed.GetSelectedText()
	if text != content {
		t.Fatalf("SelectAll text: got %q, want %q", text, content)
	}

	// Delete all and persist
	model = sendKey(t, model, "backspace")
	model = commitViaNavigation(t, model)
	ed = model.(Model)

	assertNoDuplicateLines(t, ed)
	lines := linesOf(ed)
	if len(lines) > 1 {
		t.Errorf("expected ≤1 line after delete-all, got %d: %q", len(lines), lines)
	}
}

func TestCutPaste_ShiftSelectInFrontmatter(t *testing.T) {
	content := "---\nglobals:\n  my_var: 42\n---\nx = 10"
	m := newEditorWithContent(t, content)

	var model tea.Model = m

	// Navigate to line 2 (  my_var: 42)
	model = sendKey(t, model, "down")
	model = sendKey(t, model, "down")

	// Move to col 2, then select "my_var" with 6 shift+rights
	model = sendKey(t, model, "right")
	model = sendKey(t, model, "right")
	for range 6 {
		model = sendKey(t, model, "shift+right")
	}

	ed := model.(Model)
	text := ed.GetSelectedText()
	if text != "my_var" {
		t.Fatalf("selected: got %q, want %q", text, "my_var")
	}

	// Type replacement and navigate to persist
	model = typeText(t, model, "tax_rate")
	model = commitViaNavigation(t, model)
	ed = model.(Model)

	lines := linesOf(ed)
	assertNoDuplicateLines(t, ed)

	found := false
	for _, l := range lines {
		if strings.Contains(l, "tax_rate: 42") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("replacement not found; lines: %q", lines)
	}
}

func TestCutPaste_ShiftSelectSpanningFrontmatterClosingDelimiter(t *testing.T) {
	content := "---\nglobals:\n  my_var: 42\n---\nx = 10"
	m := newEditorWithContent(t, content)

	var model tea.Model = m

	// Navigate to end of line 2 (my_var: 42)
	model = sendKey(t, model, "down")
	model = sendKey(t, model, "down")
	model = sendKey(t, model, "end")

	// Shift+Down twice to span across --- and into body
	model = sendKey(t, model, "shift+down")
	model = sendKey(t, model, "shift+down")

	ed := model.(Model)
	if !ed.HasSelection() {
		t.Fatal("expected selection spanning FM delimiter")
	}

	text := ed.GetSelectedText()
	t.Logf("selected across delimiter: %q", text)
	if !strings.Contains(text, "---") {
		t.Errorf("selection should span closing ---; got %q", text)
	}

	// Delete and persist
	model = sendKey(t, model, "backspace")
	model = commitViaNavigation(t, model)
	ed = model.(Model)

	assertNoDuplicateLines(t, ed)
	t.Logf("lines after delete across delimiter: %q", linesOf(ed))
}

// --- Paste into empty and boundary positions --------------------------------

func TestCutPaste_PasteIntoEmptyDocument(t *testing.T) {
	m := newEditorWithContent(t, "\n")

	m.cursorLine = 0
	m.cursorCol = 0
	m.editBuf = ""
	m.editBufLoaded = true

	simulatePaste(&m, "hello\nworld")

	lines := linesOf(m)
	assertNoDuplicateLines(t, m)
	t.Logf("after paste into empty: %q", lines)

	if !slices.Contains(lines, "hello") {
		t.Errorf("'hello' not found in lines: %q", lines)
	}
}

func TestCutPaste_PasteMultiLineAtMiddleOfLine(t *testing.T) {
	m := newEditorWithContent(t, "abcdef")

	m.cursorLine = 0
	m.cursorCol = 3
	m.editBuf = "abcdef"
	m.editBufLoaded = true

	simulatePaste(&m, "X\nY\nZ")

	assertNoDuplicateLines(t, m)
	assertLines(t, m, []string{"abcX", "Y", "Zdef"})
	assertCursor(t, m, 2, 1)
}

func TestCutPaste_PasteAtDocumentEnd(t *testing.T) {
	m := newEditorWithContent(t, "first\nsecond")

	m.cursorLine = 1
	m.cursorCol = 6
	m.editBuf = "second"
	m.editBufLoaded = true

	simulatePaste(&m, "\nthird")

	lines := linesOf(m)
	assertNoDuplicateLines(t, m)
	if lines[len(lines)-1] != "third" {
		t.Errorf("last line: got %q, want %q", lines[len(lines)-1], "third")
	}
}

func TestCutPaste_PasteAtDocumentStart(t *testing.T) {
	m := newEditorWithContent(t, "existing")

	m.cursorLine = 0
	m.cursorCol = 0
	m.editBuf = "existing"
	m.editBufLoaded = true

	simulatePaste(&m, "new\n")

	lines := linesOf(m)
	assertNoDuplicateLines(t, m)

	if lines[0] != "new" {
		t.Errorf("first line: got %q, want %q", lines[0], "new")
	}
	if !strings.HasPrefix(lines[1], "existing") {
		t.Errorf("second line: got %q, expected prefix 'existing'", lines[1])
	}
}

// --- Paste replaces selection -----------------------------------------------

func TestCutPaste_PasteReplacesSelection(t *testing.T) {
	m := newEditorWithContent(t, "hello world")

	selectRange(&m, 0, 6, 0, 11)
	simulatePaste(&m, "universe")

	assertLines(t, m, []string{"hello universe"})
	assertNoDuplicateLines(t, m)
}

func TestCutPaste_PasteMultiLineReplacesSelection(t *testing.T) {
	m := newEditorWithContent(t, "aaa bbb ccc")

	selectRange(&m, 0, 4, 0, 7)
	simulatePaste(&m, "X\nY\nZ")

	assertNoDuplicateLines(t, m)
	assertLines(t, m, []string{"aaa X", "Y", "Z ccc"})
}

// --- Cursor position after paste --------------------------------------------

func TestCutPaste_CursorAfterSingleLinePaste(t *testing.T) {
	m := newEditorWithContent(t, "ab")

	m.cursorLine = 0
	m.cursorCol = 1
	m.editBuf = "ab"
	m.editBufLoaded = true

	simulatePaste(&m, "XY")

	assertLines(t, m, []string{"aXYb"})
	assertCursor(t, m, 0, 3)
}

func TestCutPaste_CursorAfterMultiLinePaste(t *testing.T) {
	m := newEditorWithContent(t, "ab")

	m.cursorLine = 0
	m.cursorCol = 1
	m.editBuf = "ab"
	m.editBufLoaded = true

	simulatePaste(&m, "X\nY")

	assertLines(t, m, []string{"aX", "Yb"})
	assertCursor(t, m, 1, 1)
}

// --- Undo after cut/paste ---------------------------------------------------

func TestCutPaste_UndoAfterDelete(t *testing.T) {
	m := newEditorWithContent(t, "aaa\nbbb\nccc")

	var model tea.Model = m

	// Navigate to line 1, select "bbb"
	model = sendKey(t, model, "down")
	model = sendKey(t, model, "home")
	for range 3 {
		model = sendKey(t, model, "shift+right")
	}

	ed := model.(Model)
	if ed.GetSelectedText() != "bbb" {
		t.Fatalf("selected: got %q, want %q", ed.GetSelectedText(), "bbb")
	}

	// Delete and persist
	model = sendKey(t, model, "backspace")
	model = commitViaNavigation(t, model)

	// Undo and persist
	model = sendKey(t, model, "ctrl+z")
	model = commitViaNavigation(t, model)
	ed = model.(Model)

	assertNoDuplicateLines(t, ed)
	if !slices.Contains(linesOf(ed), "bbb") {
		t.Errorf("undo did not restore 'bbb'; lines: %q", linesOf(ed))
	}
}

func TestCutPaste_UndoAfterPaste(t *testing.T) {
	m := newEditorWithContent(t, "hello")

	m.cursorLine = 0
	m.cursorCol = 5
	m.editBuf = "hello"
	m.editBufLoaded = true

	simulatePaste(&m, " world")
	assertLines(t, m, []string{"hello world"})

	// Undo
	var model tea.Model = m
	model = sendKey(t, model, "ctrl+z")
	model = commitViaNavigation(t, model)
	ed := model.(Model)

	lines := linesOf(ed)
	assertNoDuplicateLines(t, ed)
	t.Logf("after undo paste: %q", lines)
}

func TestCutPaste_UndoAfterMultiLinePaste(t *testing.T) {
	m := newEditorWithContent(t, "abc")

	m.cursorLine = 0
	m.cursorCol = 3
	m.editBuf = "abc"
	m.editBufLoaded = true

	simulatePaste(&m, "\nline2\nline3")

	lines := linesOf(m)
	assertNoDuplicateLines(t, m)
	t.Logf("after multi-line paste: %q", lines)

	// Undo
	var model tea.Model = m
	model = sendKey(t, model, "ctrl+z")
	model = commitViaNavigation(t, model)
	ed := model.(Model)

	assertNoDuplicateLines(t, ed)
	t.Logf("after undo multi-line paste: %q", linesOf(ed))
}

// --- Frontmatter-specific undo of cut/paste ---------------------------------

func TestCutPaste_UndoCutInFrontmatter(t *testing.T) {
	content := "---\nglobals:\n  my_var: 42\n---\nx = 10"
	m := newEditorWithContent(t, content)

	var model tea.Model = m

	// Navigate to line 2, go to end, select "42" via shift+left twice
	model = sendKey(t, model, "down")
	model = sendKey(t, model, "down")
	model = sendKey(t, model, "end")
	model = sendKey(t, model, "shift+left")
	model = sendKey(t, model, "shift+left")

	ed := model.(Model)
	text := ed.GetSelectedText()
	if text != "42" {
		t.Fatalf("selected: got %q, want %q", text, "42")
	}

	// Delete and persist
	model = sendKey(t, model, "backspace")
	model = commitViaNavigation(t, model)

	// Undo and persist
	model = sendKey(t, model, "ctrl+z")
	model = commitViaNavigation(t, model)
	ed = model.(Model)

	assertNoDuplicateLines(t, ed)
	found := false
	for _, l := range linesOf(ed) {
		if strings.Contains(l, "42") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("undo did not restore '42'; lines: %q", linesOf(ed))
	}
}

func TestCutPaste_RepeatedCutPasteInFrontmatter(t *testing.T) {
	content := "---\nglobals:\n  rate: 0.5\n  tax: 10\n---\nx = @globals.rate + @globals.tax"
	m := newEditorWithContent(t, content)

	fmCount := m.frontmatterLineCount()
	if fmCount < 5 {
		t.Fatalf("frontmatter lines: got %d, want >= 5", fmCount)
	}

	// Cut "0.5" from the rate line (line 2, cols 8–11)
	selectRange(&m, 2, 8, 2, 11)
	deleted, _ := m.DeleteSelection()
	if deleted != "0.5" {
		t.Fatalf("deleted: got %q, want %q", deleted, "0.5")
	}
	commitEditBuf(&m)
	assertNoDuplicateLines(t, m)

	// Find the tax line and paste at the end
	lines := linesOf(m)
	taxIdx := -1
	for i, l := range lines {
		if strings.Contains(l, "tax:") {
			taxIdx = i
			break
		}
	}
	if taxIdx < 0 {
		t.Fatalf("'tax:' line not found; lines: %q", lines)
	}

	m.cursorLine = taxIdx
	taxLine := lines[taxIdx]
	m.cursorCol = runeLen(taxLine)
	m.editBuf = taxLine
	m.editBufLoaded = true

	simulatePaste(&m, deleted)
	assertNoDuplicateLines(t, m)

	lines = linesOf(m)
	t.Logf("after cut-paste within FM: %q", lines)

	found := false
	for _, l := range lines {
		if strings.Contains(l, "tax:") && strings.Contains(l, "0.5") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("pasted value not found in tax line; lines: %q", lines)
	}
}

// --- Edge cases -------------------------------------------------------------

func TestCutPaste_EmptySelectionDeleteIsNoop(t *testing.T) {
	m := newEditorWithContent(t, "hello")

	// Set anchor == cursor (zero-width selection)
	m.selectionAnchorLine = 0
	m.selectionAnchorCol = 3
	m.cursorLine = 0
	m.cursorCol = 3

	if m.HasSelection() {
		t.Fatal("anchor == cursor should not be a selection")
	}

	deleted, _ := m.DeleteSelection()
	if deleted != "" {
		t.Fatalf("expected empty deleted text, got %q", deleted)
	}

	assertLines(t, m, []string{"hello"})
}

func TestCutPaste_DeleteSelectionClearsCursorCorrectly(t *testing.T) {
	m := newEditorWithContent(t, "abcdef")

	// Select "cd" (cols 2–4)
	selectRange(&m, 0, 2, 0, 4)
	m.DeleteSelection()
	commitEditBuf(&m)

	// Cursor should be at the start of the selection
	assertCursor(t, m, 0, 2)
	// Selection should be cleared
	if m.HasSelection() {
		t.Error("selection should be cleared after DeleteSelection")
	}
	assertLines(t, m, []string{"abef"})
}

func TestCutPaste_MultiLineDeleteCursorPosition(t *testing.T) {
	m := newEditorWithContent(t, "first\nsecond\nthird")

	// Select from (0, 3) to (2, 2): "st\nsecond\nth"
	selectRange(&m, 0, 3, 2, 2)
	deleted, _ := m.DeleteSelection()
	if deleted != "st\nsecond\nth" {
		t.Fatalf("deleted: got %q, want %q", deleted, "st\nsecond\nth")
	}
	commitEditBuf(&m)

	assertCursor(t, m, 0, 3)
	assertLines(t, m, []string{"firird"})
	assertNoDuplicateLines(t, m)
}

// --- GetSelectedText correctness for frontmatter ----------------------------

func TestCutPaste_GetSelectedTextInFrontmatter(t *testing.T) {
	content := "---\nglobals:\n  my_var: 42\n---\nx = 10"
	m := newEditorWithContent(t, content)

	// Select the entire "globals:" line
	selectRange(&m, 1, 0, 1, 8)
	text := m.GetSelectedText()
	if text != "globals:" {
		t.Errorf("got %q, want %q", text, "globals:")
	}

	// Select across frontmatter boundary
	selectRange(&m, 3, 0, 4, 1)
	text = m.GetSelectedText()
	if text != "---\nx" {
		t.Errorf("cross-boundary: got %q, want %q", text, "---\nx")
	}
}

func TestCutPaste_GetSelectedTextFullDocument(t *testing.T) {
	content := "---\nglobals:\n  my_var: 42\n---\nresult = @globals.my_var * 2"
	m := newEditorWithContent(t, content)

	m.SelectAll()
	text := m.GetSelectedText()
	if text != content {
		t.Errorf("SelectAll: got %q, want %q", text, content)
	}
}

// --- Frontmatter: cut inner content preserves delimiters --------------------

func TestCutPaste_CutInnerFrontmatterPreservesDelimiters(t *testing.T) {
	// Bug: selecting all content between --- delimiters and cutting
	// removed the entire frontmatter block instead of leaving ---\n---
	content := "---\nglobals:\n  my_var: 42\n---\nx = 10"
	m := newEditorWithContent(t, content)

	// Frontmatter lines: 0=---, 1=globals:, 2=  my_var: 42, 3=---
	// Select lines 1-2 (all content between delimiters)
	selectRange(&m, 1, 0, 2, 12)

	got := m.GetSelectedText()
	if got != "globals:\n  my_var: 42" {
		t.Fatalf("selected: got %q, want %q", got, "globals:\n  my_var: 42")
	}

	deleted, _ := m.DeleteSelection()
	if deleted != "globals:\n  my_var: 42" {
		t.Fatalf("deleted: got %q, want %q", deleted, "globals:\n  my_var: 42")
	}

	// After cutting, the frontmatter delimiters should remain: ---\n---
	// with one empty merged line between them, plus the body content.
	lines := linesOf(m)
	t.Logf("lines after cut: %q", lines)

	// First line must be opening delimiter
	if len(lines) < 3 {
		t.Fatalf("too few lines after cut: got %d, want >= 3", len(lines))
	}
	if lines[0] != "---" {
		t.Errorf("line 0: got %q, want %q", lines[0], "---")
	}
	// Merged empty line where content was
	if lines[1] != "" {
		t.Errorf("line 1: got %q, want %q (empty merged line)", lines[1], "")
	}
	// Closing delimiter
	if lines[2] != "---" {
		t.Errorf("line 2: got %q, want %q", lines[2], "---")
	}
	// Body content preserved
	if len(lines) > 3 && lines[3] != "x = 10" {
		t.Errorf("line 3: got %q, want %q", lines[3], "x = 10")
	}

	assertNoDuplicateLines(t, m)
}

func TestCutPaste_CutSingleLineInFrontmatter(t *testing.T) {
	// Verify single-line frontmatter deletion also works atomically
	content := "---\nglobals:\n  my_var: 42\n---\nx = 10"
	m := newEditorWithContent(t, content)

	// Select "globals" on line 1 (cols 0-7)
	selectRange(&m, 1, 0, 1, 7)
	deleted, _ := m.DeleteSelection()
	if deleted != "globals" {
		t.Fatalf("deleted: got %q, want %q", deleted, "globals")
	}

	lines := linesOf(m)
	t.Logf("lines after single-line cut: %q", lines)

	// Line 0 should still be ---
	if lines[0] != "---" {
		t.Errorf("line 0: got %q, want %q", lines[0], "---")
	}
	// Line 1 should be ":" (remainder after removing "globals")
	if lines[1] != ":" {
		t.Errorf("line 1: got %q, want %q", lines[1], ":")
	}
}

func TestCutPaste_DeleteSelectionViaKeyInFrontmatter(t *testing.T) {
	// Test the full key-simulation path: select inner FM content, press backspace
	content := "---\nglobals:\n  my_var: 42\n---\nx = 10"
	m := newEditorWithContent(t, content)
	var model tea.Model = m

	// Navigate to line 1 col 0
	model = sendKey(t, model, "down")
	model = sendKey(t, model, "home")

	// Select down to end of line 2
	model = sendKey(t, model, "shift+down")
	model = sendKey(t, model, "shift+end")

	ed := model.(Model)
	selected := ed.GetSelectedText()
	if selected != "globals:\n  my_var: 42" {
		t.Fatalf("selected: got %q, want %q", selected, "globals:\n  my_var: 42")
	}

	// Delete selection via backspace
	model = sendKey(t, model, "backspace")
	model = commitViaNavigation(t, model)
	ed = model.(Model)

	lines := linesOf(ed)
	t.Logf("lines after key-driven delete: %q", lines)

	// Delimiters must survive
	if len(lines) < 2 {
		t.Fatalf("too few lines: got %d", len(lines))
	}
	if lines[0] != "---" {
		t.Errorf("line 0: got %q, want %q", lines[0], "---")
	}

	// Find closing delimiter
	foundClosing := false
	for _, l := range lines {
		if l == "---" {
			foundClosing = true
		}
	}
	// There should be at least 2 "---" lines (opening and closing)
	count := 0
	for _, l := range lines {
		if l == "---" {
			count++
		}
	}
	if count < 2 {
		t.Errorf("expected 2 '---' delimiters, found %d; lines: %q", count, lines)
	}
	_ = foundClosing

	assertNoDuplicateLines(t, ed)
}
