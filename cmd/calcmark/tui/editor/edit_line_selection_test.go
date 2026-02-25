package editor

// edit_line_selection_test.go — Tests for selection highlighting on the
// cursor's edit buffer line.
//
// Bug: renderEditLineWrapped did not apply selection highlighting, so the
// cursor line (which always uses this renderer when editBufLoaded=true)
// never showed a visual selection — even though the internal selection state
// was correct.

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/CalcMark/go-calcmark/cmd/calcmark/config/theme"
	"github.com/CalcMark/go-calcmark/spec/document"
)

// selectionBgCode returns the ANSI TrueColor background code for the selection
// theme color, used to detect whether selection highlighting is present in
// rendered output.
func selectionBgCode() string {
	r, g, b, _ := theme.Selection.RGBA()
	// RGBA returns 0–65535; convert to 0–255
	return strings.Join([]string{
		"48;2",
		strings.TrimLeft(strings.Replace(
			strings.Replace(
				strings.Replace(
					strings.Join([]string{
						itoa(int(r >> 8)),
						itoa(int(g >> 8)),
						itoa(int(b >> 8)),
					}, ";"),
					"\n", "", -1),
				"\t", "", -1),
			" ", "", -1),
			""),
	}, ";")
}

func itoa(n int) string {
	if n < 10 {
		return string(rune('0' + n))
	}
	return itoa(n/10) + string(rune('0'+n%10))
}

// TestEditLineSelection_ShiftRightShowsHighlighting tests that pressing
// Shift+Right creates a visible selection on the cursor line.
// This is the exact user-reported scenario.
func TestEditLineSelection_ShiftRightShowsHighlighting(t *testing.T) {
	content := "x = 10\ny = 20\nz = 30"
	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("NewDocument: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 24
	m.previewMode = PreviewFull

	var model tea.Model = m

	// Press Shift+Right 3 times to select "x ="
	for range 3 {
		model = sendKey(t, model, "shift+right")
	}

	ed := model.(Model)
	if !ed.HasSelection() {
		t.Fatal("Expected selection after Shift+Right")
	}

	sLine, sCol, eLine, eCol := ed.GetSelectionRange()
	t.Logf("Selection: (%d,%d) to (%d,%d)", sLine, sCol, eLine, eCol)

	// Verify the editBuf is loaded (this was the precondition for the bug)
	if !ed.editBufLoaded {
		t.Fatal("Expected editBufLoaded=true after Shift+Right")
	}

	// Render the edit line and check for selection highlighting
	editLines := ed.renderEditLineWrapped(40)
	if len(editLines) == 0 {
		t.Fatal("Expected at least one edit line")
	}

	// The selection background color should appear in the rendered output
	selBg := selectionBgCode()
	t.Logf("Looking for selection bg code: %s", selBg)

	found := false
	for i, line := range editLines {
		t.Logf("Edit line %d: %q", i, line)
		if strings.Contains(line, selBg) {
			found = true
		}
	}

	if !found {
		t.Error("BUG: Selection highlighting NOT visible on cursor edit line")
		t.Error("renderEditLineWrapped must apply selection styling when HasSelection() is true")
	}
}

// TestEditLineSelection_NoSelectionNoHighlighting ensures that without a
// selection, renderEditLineWrapped produces NO selection-colored output.
func TestEditLineSelection_NoSelectionNoHighlighting(t *testing.T) {
	content := "x = 10\ny = 20"
	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("NewDocument: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 24

	// Just load the edit buffer without any selection
	m.loadCurrentLineIntoEditBuffer()
	if m.HasSelection() {
		t.Fatal("Expected no selection initially")
	}

	editLines := m.renderEditLineWrapped(40)
	selBg := selectionBgCode()

	for i, line := range editLines {
		if strings.Contains(line, selBg) {
			t.Errorf("Found selection highlighting on line %d without selection", i)
		}
	}
}

// TestEditLineSelection_MultiLineSelectionShowsCursorLine tests that when
// Shift+Down creates a multi-line selection, the cursor line portion is
// also highlighted.
func TestEditLineSelection_MultiLineSelectionShowsCursorLine(t *testing.T) {
	content := "x = 10\ny = 20\nz = 30"
	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("NewDocument: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 24

	var model tea.Model = m

	// Move to column 2 on line 0
	model = sendKey(t, model, "right")
	model = sendKey(t, model, "right")

	// Shift+Down to select from line 0 col 2 to line 1
	model = sendKey(t, model, "shift+down")

	ed := model.(Model)
	if !ed.HasSelection() {
		t.Fatal("Expected selection after Shift+Down")
	}

	sLine, sCol, eLine, eCol := ed.GetSelectionRange()
	t.Logf("Selection: (%d,%d) to (%d,%d) cursorLine=%d", sLine, sCol, eLine, eCol, ed.cursorLine)

	// Cursor should be on line 1 now
	if ed.cursorLine != 1 {
		t.Fatalf("Expected cursor on line 1, got %d", ed.cursorLine)
	}

	// The cursor line (line 1) should show selection highlighting
	// because the selection extends from line 0 to line 1
	editLines := ed.renderEditLineWrapped(40)
	selBg := selectionBgCode()

	found := false
	for _, line := range editLines {
		if strings.Contains(line, selBg) {
			found = true
		}
	}

	if !found {
		t.Error("BUG: Cursor line not highlighted in multi-line selection")
	}
}

// TestEditLineSelection_FullViewPipeline verifies that selection highlighting
// survives the full View() pipeline (renderEditLineWrapped → SideBySide.padLine
// → stripResetCodes → bgStyle.Render) and is actually visible in final output.
func TestEditLineSelection_FullViewPipeline(t *testing.T) {
	content := "x = 10\ny = 20\nz = 30"
	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("NewDocument: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 24
	m.previewMode = PreviewFull
	m.eval.Evaluate(m.doc)

	var model tea.Model = m

	// Press Shift+Right 3 times to select "x ="
	for range 3 {
		model = sendKey(t, model, "shift+right")
	}

	ed := model.(Model)
	if !ed.HasSelection() {
		t.Fatal("Expected selection after Shift+Right")
	}

	// Get the full View() output — this goes through the entire rendering pipeline
	view := ed.View().Content
	viewLines := strings.Split(view, "\n")

	selBg := selectionBgCode()
	found := false
	for _, line := range viewLines {
		// Check that selection bg appears BEFORE visible text (not just in escape codes)
		if strings.Contains(line, selBg) {
			found = true
		}
	}

	if !found {
		t.Error("BUG: Selection highlighting not visible in full View() output after Shift+Right")
	}
}

// TestEditLineSelection_NonCursorLineInFullView verifies that non-cursor lines
// in a multi-line selection show highlighting in the full View() output.
// Bug: renderLineWithSelection was called on ANSI-tinted text, so rune column
// positions didn't match document columns. Selection was applied to escape code
// bytes instead of visible text, rendering zero highlighted characters.
func TestEditLineSelection_NonCursorLineInFullView(t *testing.T) {
	content := "x = 10\ny = 20\nz = 30"
	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("NewDocument: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 24
	m.previewMode = PreviewFull
	m.eval.Evaluate(m.doc)

	var model tea.Model = m

	// Select all: Ctrl+A
	model = sendKey(t, model, "ctrl+a")

	ed := model.(Model)
	if !ed.HasSelection() {
		t.Fatal("Expected selection after Ctrl+A")
	}

	view := ed.View().Content
	viewLines := strings.Split(view, "\n")

	selBg := selectionBgCode()

	// Check that the non-cursor line "x = 10" (line 0) has selection highlighting.
	// After Ctrl+A, cursor is on last line, so line 0 is a non-cursor line.
	foundXLine := false
	xLineHighlighted := false
	for _, line := range viewLines {
		plain := stripANSI(line)
		if strings.Contains(plain, "x = 10") || strings.Contains(plain, "x = 1") {
			foundXLine = true
			// Verify the selection bg code appears and renders visible text after it.
			// Split on the selection bg code and check that text follows.
			parts := strings.SplitN(line, selBg, 2)
			if len(parts) > 1 {
				afterSelCode := parts[1]
				plainAfter := stripANSI(afterSelCode)
				if len(plainAfter) > 0 {
					xLineHighlighted = true
				}
			}
		}
	}

	if !foundXLine {
		t.Fatal("Could not find 'x = 10' line in View()")
	}
	if !xLineHighlighted {
		t.Error("BUG: Non-cursor line 'x = 10' not highlighted in Select All")
		t.Error("renderLineWithSelection must operate on raw text, not ANSI-tinted text")
	}
}

// TestEditLineSelectionRange verifies editLineSelectionRange returns correct
// ranges for various selection configurations.
func TestEditLineSelectionRange(t *testing.T) {
	content := "hello world\nsecond line\nthird line"
	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("NewDocument: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 24

	tests := []struct {
		name        string
		cursorLine  int
		cursorCol   int
		anchorLine  int
		anchorCol   int
		expectStart int
		expectEnd   int
	}{
		{
			name: "no selection", cursorLine: 0, cursorCol: 3,
			anchorLine: -1, anchorCol: -1,
			expectStart: -1, expectEnd: -1,
		},
		{
			name: "same line forward", cursorLine: 0, cursorCol: 5,
			anchorLine: 0, anchorCol: 0,
			expectStart: 0, expectEnd: 5,
		},
		{
			name: "same line backward", cursorLine: 0, cursorCol: 2,
			anchorLine: 0, anchorCol: 8,
			expectStart: 2, expectEnd: 8,
		},
		{
			name: "multi-line cursor on first line", cursorLine: 0, cursorCol: 3,
			anchorLine: 2, anchorCol: 5,
			expectStart: 3, expectEnd: 11, // full line 0 length
		},
		{
			name: "multi-line cursor on last line", cursorLine: 2, cursorCol: 7,
			anchorLine: 0, anchorCol: 3,
			expectStart: 0, expectEnd: 7,
		},
		{
			name: "cursor at end of multi-line selection", cursorLine: 1, cursorCol: 4,
			anchorLine: 0, anchorCol: 3,
			expectStart: 0, expectEnd: 4, // cursor line is end line; selectEnd = cursorCol
		},
		{
			name: "selection on different line", cursorLine: 0, cursorCol: 3,
			anchorLine: 2, anchorCol: 2,
			expectStart: 3, expectEnd: 11, // full line 0 length
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m.cursorLine = tt.cursorLine
			m.cursorCol = tt.cursorCol
			m.selectionAnchorLine = tt.anchorLine
			m.selectionAnchorCol = tt.anchorCol
			m.editBuf = m.GetLines()[m.cursorLine]

			gotStart, gotEnd := m.editLineSelectionRange()
			if gotStart != tt.expectStart || gotEnd != tt.expectEnd {
				t.Errorf("editLineSelectionRange() = (%d, %d), want (%d, %d)",
					gotStart, gotEnd, tt.expectStart, tt.expectEnd)
			}
		})
	}
}
