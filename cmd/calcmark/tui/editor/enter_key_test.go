package editor

import (
	"testing"

	"github.com/CalcMark/go-calcmark/spec/document"
	tea "charm.land/bubbletea/v2"
)

// TestDocumentBlockStructure verifies how different document contents create blocks
func TestDocumentBlockStructure(t *testing.T) {
	tests := []struct {
		name            string
		content         string
		wantBlocks      int
		wantSourceLines []string
	}{
		{
			name:            "empty string",
			content:         "",
			wantBlocks:      0,
			wantSourceLines: nil,
		},
		{
			name:            "single newline",
			content:         "\n",
			wantBlocks:      1,
			wantSourceLines: []string{""},
		},
		{
			name:            "two newlines",
			content:         "\n\n",
			wantBlocks:      1,
			wantSourceLines: []string{"", ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := document.NewDocument(tt.content)
			if err != nil {
				t.Fatalf("Failed to create document: %v", err)
			}

			blocks := doc.GetBlocks()
			if len(blocks) != tt.wantBlocks {
				t.Errorf("Expected %d blocks, got %d", tt.wantBlocks, len(blocks))
			}

			if tt.wantBlocks > 0 && len(blocks) > 0 {
				sourceLines := blocks[0].Block.Source()
				t.Logf("Block 0 has %d source lines: %v", len(sourceLines), sourceLines)

				if tt.wantSourceLines != nil {
					if len(sourceLines) != len(tt.wantSourceLines) {
						t.Errorf("Expected %d source lines, got %d",
							len(tt.wantSourceLines), len(sourceLines))
					}
				}
			}
		})
	}
}

// TestEmptyDocumentStartsWithCorrectLineCount verifies empty doc doesn't have extra lines
func TestEmptyDocumentStartsWithCorrectLineCount(t *testing.T) {
	// Test what happens with truly empty document
	emptyDoc, err := document.NewDocument("")
	if err != nil {
		t.Fatalf("Failed to create empty document: %v", err)
	}

	t.Logf("Empty doc (\"\") has %d blocks", len(emptyDoc.GetBlocks()))

	// Test what happens with newline document
	newlineDoc, err := document.NewDocument("\n")
	if err != nil {
		t.Fatalf("Failed to create newline document: %v", err)
	}

	blocks := newlineDoc.GetBlocks()
	t.Logf("Newline doc (\"\\n\") has %d blocks", len(blocks))
	if len(blocks) > 0 {
		source := blocks[0].Block.Source()
		t.Logf("Block 0 source: %d lines: %v", len(source), source)
	}

	// Now create editor with empty doc
	m := New(emptyDoc)
	m.width = 80
	m.height = 24

	lines := m.TotalLines()
	t.Logf("Editor initialized with empty doc: TotalLines()=%d", lines)

	// After initialization, transitionToReady() creates doc with "\n" if empty
	// So we expect the editor to have the doc structure of "\n"
	editorBlocks := m.doc.GetBlocks()
	t.Logf("Editor doc has %d blocks after init", len(editorBlocks))
	if len(editorBlocks) > 0 {
		source := editorBlocks[0].Block.Source()
		t.Logf("Editor block 0 source: %d lines: %v", len(source), source)
	}

	// Get actual lines from editor
	docLines := m.GetLines()
	t.Logf("Editor GetLines() returned %d lines:", len(docLines))
	for i, line := range docLines {
		t.Logf("  Line %d: %q", i, line)
	}

	// THE BUG: Empty document should result in 1 line, not 2
	if lines != 1 {
		t.Errorf("Editor with empty/newline document should have 1 line, got %d lines", lines)
	}
}

// TestTypingInEmptyDocDoesNotCreateExtraLines verifies typing doesn't create phantom lines
func TestTypingInEmptyDocDoesNotCreateExtraLines(t *testing.T) {
	doc, err := document.NewDocument("")
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 24

	initialLines := m.TotalLines()
	t.Logf("Initial: %d lines", initialLines)

	// Type "1. asdf"
	for i, ch := range "1. asdf" {
		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
		m = result.(Model)

		afterLines := m.TotalLines()
		if i == 0 {
			t.Logf("After typing first char '%c': %d lines, cursorLine=%d, editBuf=%q",
				ch, afterLines, m.cursorLine, m.editBuf)
		}
	}

	finalLines := m.TotalLines()
	t.Logf("After typing '1. asdf': %d lines, cursorLine=%d, editBuf=%q",
		finalLines, m.cursorLine, m.editBuf)

	// Typing should create at most 1 line (the line we're typing on)
	// It should NOT create extra blank lines
	if finalLines > 1 {
		t.Errorf("After typing, should have 1 line, got %d lines", finalLines)

		// Show what the lines are
		docLines := m.GetLines()
		for i, line := range docLines {
			t.Logf("  Line %d: %q", i, line)
		}
	}
}

// TestEnterAfterOrderedListItem verifies that pressing ENTER after typing "1. asdf"
// creates exactly ONE new line, not TWO.
// This reproduces the bug shown in the user's screenshot.
func TestEnterAfterOrderedListItem(t *testing.T) {
	// Start with empty document
	doc, err := document.NewDocument("")
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 24

	// Type "1. asdf"
	for _, ch := range "1. asdf" {
		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
		m = result.(Model)
	}

	// Verify we have exactly 1 line
	linesBefore := m.TotalLines()
	t.Logf("After typing '1. asdf': %d lines, cursorLine=%d, editBuf=%q",
		linesBefore, m.cursorLine, m.editBuf)

	if linesBefore != 1 {
		t.Errorf("Before ENTER: should have 1 line, got %d lines", linesBefore)
		docLines := m.GetLines()
		for i, line := range docLines {
			t.Logf("  Line %d: %q", i, line)
		}
	}

	// Press ENTER to create new line
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = result.(Model)

	linesAfter := m.TotalLines()
	t.Logf("After pressing ENTER: %d lines, cursorLine=%d, editBuf=%q",
		linesAfter, m.cursorLine, m.editBuf)

	// CRITICAL: Should have added exactly 1 new line
	expectedLines := linesBefore + 1
	if linesAfter != expectedLines {
		t.Errorf("ENTER should add 1 line: expected %d lines, got %d lines (added %d lines instead of 1)",
			expectedLines, linesAfter, linesAfter-linesBefore)

		docLines := m.GetLines()
		for i, line := range docLines {
			t.Logf("  Line %d: %q", i, line)
		}
	}

	// Verify cursor moved to the next line
	expectedCursorLine := linesBefore // Should be on the newly created line
	if m.cursorLine != expectedCursorLine {
		t.Errorf("Cursor should be on line %d, got %d", expectedCursorLine, m.cursorLine)
	}

	// Verify editBuf is empty (we're on a new blank line)
	if m.editBuf != "" {
		t.Errorf("editBuf should be empty on new line, got %q", m.editBuf)
	}
}

// TestEnterInMiddleOfOrderedListItem verifies ENTER splits the line correctly
func TestEnterInMiddleOfOrderedListItem(t *testing.T) {
	// Start with "1. hello world"
	doc, err := document.NewDocument("1. hello world")
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 24

	// Move cursor to position after "hello " (before "world")
	// "1. hello world"
	//           ^ cursor here (position 9)
	m.cursorLine = 0
	m.cursorCol = 9
	m.mode = StateDefault
	m.loadCurrentLineIntoEditBuffer() // Load the line into editBuf

	linesBefore := m.TotalLines()
	t.Logf("Before ENTER: %d lines, cursorLine=%d, cursorCol=%d, editBuf=%q",
		linesBefore, m.cursorLine, m.cursorCol, m.editBuf)

	// Press ENTER to split the line
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = result.(Model)

	linesAfter := m.TotalLines()
	t.Logf("After ENTER: %d lines, cursorLine=%d, editBuf=%q",
		linesAfter, m.cursorLine, m.editBuf)

	// Should add exactly 1 line
	expectedLines := linesBefore + 1
	if linesAfter != expectedLines {
		t.Errorf("ENTER should add 1 line: expected %d lines, got %d lines (added %d lines)",
			expectedLines, linesAfter, linesAfter-linesBefore)
	}

	// First line should be "1. hello "
	lines := m.GetLines()
	if len(lines) > 0 {
		firstLine := lines[0]
		if firstLine != "1. hello " {
			t.Errorf("First line should be '1. hello ', got %q", firstLine)
		}
	}

	// Cursor should be on line 1 (the new second line)
	if m.cursorLine != 1 {
		t.Errorf("Cursor should be on line 1, got %d", m.cursorLine)
	}

	// editBuf should contain "world" (the text after cursor)
	if m.editBuf != "world" {
		t.Errorf("editBuf should be 'world', got %q", m.editBuf)
	}
}

// TestEnterAtEndOfLine verifies ENTER at end of line creates new blank line
func TestEnterAtEndOfLine(t *testing.T) {
	doc, err := document.NewDocument("line 1")
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 24

	// Position cursor at end of line 0
	m.cursorLine = 0
	m.cursorCol = len("line 1")
	m.mode = StateDefault

	// Enter edit mode
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	m = result.(Model)

	linesBefore := m.TotalLines()
	t.Logf("Before ENTER: %d lines, cursorLine=%d, cursorCol=%d",
		linesBefore, m.cursorLine, m.cursorCol)

	// Press ENTER
	result, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = result.(Model)

	linesAfter := m.TotalLines()
	t.Logf("After ENTER: %d lines, cursorLine=%d, editBuf=%q",
		linesAfter, m.cursorLine, m.editBuf)

	// Should add exactly 1 line
	expectedLines := linesBefore + 1
	if linesAfter != expectedLines {
		t.Errorf("ENTER should add 1 line: expected %d lines, got %d lines (added %d lines)",
			expectedLines, linesAfter, linesAfter-linesBefore)
	}

	// Cursor should be on new line
	if m.cursorLine != 1 {
		t.Errorf("Cursor should be on line 1, got %d", m.cursorLine)
	}

	// editBuf should be empty (blank new line)
	if m.editBuf != "" {
		t.Errorf("editBuf should be empty, got %q", m.editBuf)
	}
}

// TestMultipleEntersCreateMultipleLines verifies consecutive ENTER presses
func TestMultipleEntersCreateMultipleLines(t *testing.T) {
	doc, err := document.NewDocument("start")
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 24

	// Go to end of line and enter edit mode
	m.cursorLine = 0
	m.cursorCol = len("start")
	m.mode = StateDefault
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	m = result.(Model)

	startingLines := m.TotalLines()
	t.Logf("Starting with %d lines", startingLines)

	// Press ENTER 3 times
	for i := range 3 {
		beforeEnter := m.TotalLines()
		result, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		m = result.(Model)
		afterEnter := m.TotalLines()

		t.Logf("After ENTER %d: lines went from %d to %d (cursorLine=%d)",
			i+1, beforeEnter, afterEnter, m.cursorLine)

		// Each ENTER should add exactly 1 line
		if afterEnter != beforeEnter+1 {
			t.Errorf("ENTER %d should add 1 line, but lines went from %d to %d (change of %d)",
				i+1, beforeEnter, afterEnter, afterEnter-beforeEnter)
		}
	}

	// Should have added exactly 3 lines
	finalLines := m.TotalLines()
	expectedFinalLines := startingLines + 3
	if finalLines != expectedFinalLines {
		t.Errorf("After 3 ENTERs: expected %d lines, got %d lines",
			expectedFinalLines, finalLines)
	}
}
