package editor

import (
	"testing"

	"github.com/CalcMark/go-calcmark/spec/document"
)

func TestHasSelection(t *testing.T) {
	doc, err := document.NewDocument("line one\nline two\nline three")
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}
	m := New(doc)

	// Initially no selection
	if m.HasSelection() {
		t.Error("HasSelection() = true, want false (initial state)")
	}

	// Set anchor - cursor and anchor are same, so no effective selection
	m.SetSelectionAnchor()
	if m.HasSelection() {
		t.Error("HasSelection() = true, want false (anchor == cursor)")
	}

	// Move cursor to create selection
	m.cursorCol = 5
	if !m.HasSelection() {
		t.Error("HasSelection() = false, want true (anchor != cursor)")
	}

	// Clear selection
	m.ClearSelection()
	if m.HasSelection() {
		t.Error("HasSelection() = true after ClearSelection(), want false")
	}
}

func TestGetSelectionRange_Normalization(t *testing.T) {
	doc, err := document.NewDocument("line one\nline two\nline three")
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}
	m := New(doc)

	tests := []struct {
		name                  string
		anchorLine, anchorCol int
		cursorLine, cursorCol int
		wantStart, wantSCol   int
		wantEnd, wantECol     int
	}{
		{
			name:       "forward same line",
			anchorLine: 0, anchorCol: 5,
			cursorLine: 0, cursorCol: 10,
			wantStart: 0, wantSCol: 5,
			wantEnd: 0, wantECol: 10,
		},
		{
			name:       "backward same line",
			anchorLine: 0, anchorCol: 10,
			cursorLine: 0, cursorCol: 5,
			wantStart: 0, wantSCol: 5,
			wantEnd: 0, wantECol: 10,
		},
		{
			name:       "cursor before anchor (different lines)",
			anchorLine: 2, anchorCol: 0,
			cursorLine: 0, cursorCol: 5,
			wantStart: 0, wantSCol: 5,
			wantEnd: 2, wantECol: 0,
		},
		{
			name:       "anchor before cursor (different lines)",
			anchorLine: 0, anchorCol: 5,
			cursorLine: 2, cursorCol: 0,
			wantStart: 0, wantSCol: 5,
			wantEnd: 2, wantECol: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m.selectionAnchorLine = tt.anchorLine
			m.selectionAnchorCol = tt.anchorCol
			m.cursorLine = tt.cursorLine
			m.cursorCol = tt.cursorCol

			startLine, startCol, endLine, endCol := m.GetSelectionRange()

			if startLine != tt.wantStart || startCol != tt.wantSCol {
				t.Errorf("start = (%d, %d), want (%d, %d)",
					startLine, startCol, tt.wantStart, tt.wantSCol)
			}
			if endLine != tt.wantEnd || endCol != tt.wantECol {
				t.Errorf("end = (%d, %d), want (%d, %d)",
					endLine, endCol, tt.wantEnd, tt.wantECol)
			}
		})
	}
}

func TestGetSelectionRange_NoSelection(t *testing.T) {
	doc, err := document.NewDocument("test")
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}
	m := New(doc)

	// No selection set
	startLine, startCol, endLine, endCol := m.GetSelectionRange()
	if startLine != -1 || startCol != -1 || endLine != -1 || endCol != -1 {
		t.Errorf("GetSelectionRange() = (%d, %d, %d, %d), want (-1, -1, -1, -1)",
			startLine, startCol, endLine, endCol)
	}
}

func TestGetSelectedText_SingleLine(t *testing.T) {
	doc, err := document.NewDocument("hello world")
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}
	m := New(doc)

	// Select "llo wo" (positions 2-8)
	m.selectionAnchorLine = 0
	m.selectionAnchorCol = 2
	m.cursorLine = 0
	m.cursorCol = 8

	got := m.GetSelectedText()
	want := "llo wo"
	if got != want {
		t.Errorf("GetSelectedText() = %q, want %q", got, want)
	}
}

func TestGetSelectedText_SingleLine_Unicode(t *testing.T) {
	doc, err := document.NewDocument("hello \u4e16\u754c world") // "hello \u4e16\u754c world" (Chinese characters)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}
	m := New(doc)

	// Select "\u4e16\u754c wo" (rune positions 6-11)
	m.selectionAnchorLine = 0
	m.selectionAnchorCol = 6
	m.cursorLine = 0
	m.cursorCol = 11

	got := m.GetSelectedText()
	want := "\u4e16\u754c wo"
	if got != want {
		t.Errorf("GetSelectedText() = %q, want %q", got, want)
	}
}

func TestGetSelectedText_MultiLine(t *testing.T) {
	doc, err := document.NewDocument("first line\nsecond line\nthird line")
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}
	m := New(doc)

	// Select from "rst li" (line 0, col 2) to "third" (line 2, col 5)
	m.selectionAnchorLine = 0
	m.selectionAnchorCol = 2
	m.cursorLine = 2
	m.cursorCol = 5

	got := m.GetSelectedText()
	want := "rst line\nsecond line\nthird"
	if got != want {
		t.Errorf("GetSelectedText() = %q, want %q", got, want)
	}
}

func TestGetSelectedText_NoSelection(t *testing.T) {
	doc, err := document.NewDocument("test")
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}
	m := New(doc)

	got := m.GetSelectedText()
	if got != "" {
		t.Errorf("GetSelectedText() = %q, want empty string", got)
	}
}

func TestDeleteSelection_SingleLine(t *testing.T) {
	doc, err := document.NewDocument("hello world")
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}
	m := New(doc)
	m.width = 80
	m.height = 24

	// Select "llo wo" (positions 2-8)
	m.selectionAnchorLine = 0
	m.selectionAnchorCol = 2
	m.cursorLine = 0
	m.cursorCol = 8

	deletedText, cmd := m.DeleteSelection()

	// Verify deleted text returned
	wantDeleted := "llo wo"
	if deletedText != wantDeleted {
		t.Errorf("DeleteSelection() returned %q, want %q", deletedText, wantDeleted)
	}

	// Verify command returned (for undo)
	if cmd == nil {
		t.Error("DeleteSelection() returned nil cmd, want non-nil for undo")
	}

	// Verify selection cleared
	if m.HasSelection() {
		t.Error("Selection not cleared after DeleteSelection()")
	}

	// Verify cursor at start of deleted range
	if m.cursorLine != 0 || m.cursorCol != 2 {
		t.Errorf("cursor = (%d, %d), want (0, 2)", m.cursorLine, m.cursorCol)
	}

	// Verify document updated - need to transition to get the editBuf applied
	m.transitionToProcessing()
	lines := m.GetLines()
	if len(lines) != 1 {
		t.Fatalf("lines = %d, want 1", len(lines))
	}
	wantLine := "herld"
	if lines[0] != wantLine {
		t.Errorf("line[0] = %q, want %q", lines[0], wantLine)
	}

	// Verify undo manager has operation recorded
	// The operation is in current batch or committed history
	if !m.undoManager.HasUndo() && m.undoManager.CurrentBatchSize() == 0 {
		t.Error("undo history empty, want operation recorded")
	}
}

func TestDeleteSelection_MultiLine(t *testing.T) {
	doc, err := document.NewDocument("first line\nsecond line\nthird line")
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}
	m := New(doc)
	m.width = 80
	m.height = 24

	// Select from position 6 on line 0 to position 5 on line 2
	// This selects "line\nsecond line\nthird"
	m.selectionAnchorLine = 0
	m.selectionAnchorCol = 6
	m.cursorLine = 2
	m.cursorCol = 5

	deletedText, _ := m.DeleteSelection()

	// Verify deleted text
	wantDeleted := "line\nsecond line\nthird"
	if deletedText != wantDeleted {
		t.Errorf("DeleteSelection() returned %q, want %q", deletedText, wantDeleted)
	}

	// Verify selection cleared
	if m.HasSelection() {
		t.Error("Selection not cleared after DeleteSelection()")
	}

	// Verify cursor position
	if m.cursorLine != 0 || m.cursorCol != 6 {
		t.Errorf("cursor = (%d, %d), want (0, 6)", m.cursorLine, m.cursorCol)
	}

	// Verify document updated - transition to apply editBuf
	m.transitionToProcessing()
	lines := m.GetLines()
	// Should be single line: "first  line" (first 6 chars + last 5 chars)
	if len(lines) != 1 {
		t.Fatalf("lines = %d, want 1, got %v", len(lines), lines)
	}
	wantLine := "first  line"
	if lines[0] != wantLine {
		t.Errorf("line[0] = %q, want %q", lines[0], wantLine)
	}
}

func TestDeleteSelection_NoSelection(t *testing.T) {
	doc, err := document.NewDocument("test")
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}
	m := New(doc)

	deletedText, cmd := m.DeleteSelection()

	if deletedText != "" {
		t.Errorf("DeleteSelection() returned %q, want empty string", deletedText)
	}
	if cmd != nil {
		t.Error("DeleteSelection() returned non-nil cmd for no selection")
	}
}

func TestSelectAll(t *testing.T) {
	doc, err := document.NewDocument("first line\nsecond line\nthird line")
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}
	m := New(doc)

	m.SelectAll()

	// Verify anchor at 0,0
	if m.selectionAnchorLine != 0 || m.selectionAnchorCol != 0 {
		t.Errorf("anchor = (%d, %d), want (0, 0)",
			m.selectionAnchorLine, m.selectionAnchorCol)
	}

	// Verify cursor at end of last line
	// "third line" has 10 characters
	if m.cursorLine != 2 || m.cursorCol != 10 {
		t.Errorf("cursor = (%d, %d), want (2, 10)", m.cursorLine, m.cursorCol)
	}

	// Verify HasSelection returns true
	if !m.HasSelection() {
		t.Error("HasSelection() = false after SelectAll(), want true")
	}

	// Verify GetSelectedText returns all content
	got := m.GetSelectedText()
	want := "first line\nsecond line\nthird line"
	if got != want {
		t.Errorf("GetSelectedText() = %q, want %q", got, want)
	}
}

func TestSelectAll_EmptyDocument(t *testing.T) {
	doc, err := document.NewDocument("")
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}
	m := New(doc)

	m.SelectAll()

	// Empty document still has one empty line (line 0)
	// SelectAll sets anchor at 0,0 and cursor at end of last line (0,0)
	// But since anchor == cursor, HasSelection returns false
	if m.selectionAnchorLine != 0 || m.selectionAnchorCol != 0 {
		t.Errorf("anchor = (%d, %d), want (0, 0)",
			m.selectionAnchorLine, m.selectionAnchorCol)
	}
	if m.cursorLine != 0 || m.cursorCol != 0 {
		t.Errorf("cursor = (%d, %d), want (0, 0)", m.cursorLine, m.cursorCol)
	}

	// Empty selection (anchor == cursor)
	if m.HasSelection() {
		t.Error("HasSelection() = true for empty document, want false (anchor == cursor)")
	}

	// GetSelectedText returns empty string for empty document
	got := m.GetSelectedText()
	if got != "" {
		t.Errorf("GetSelectedText() = %q, want empty string", got)
	}
}

func TestSelectAll_SingleLine(t *testing.T) {
	doc, err := document.NewDocument("single line content")
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}
	m := New(doc)

	m.SelectAll()

	// Verify anchor at 0,0
	if m.selectionAnchorLine != 0 || m.selectionAnchorCol != 0 {
		t.Errorf("anchor = (%d, %d), want (0, 0)",
			m.selectionAnchorLine, m.selectionAnchorCol)
	}

	// "single line content" has 19 characters
	if m.cursorLine != 0 || m.cursorCol != 19 {
		t.Errorf("cursor = (%d, %d), want (0, 19)", m.cursorLine, m.cursorCol)
	}

	got := m.GetSelectedText()
	want := "single line content"
	if got != want {
		t.Errorf("GetSelectedText() = %q, want %q", got, want)
	}
}
