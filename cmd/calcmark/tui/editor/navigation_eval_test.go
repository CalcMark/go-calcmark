package editor

import (
	"strings"
	"testing"

	"github.com/CalcMark/go-calcmark/spec/document"
	tea "github.com/charmbracelet/bubbletea"
)

// TestArrowUpEvaluatesLine tests that pressing arrow up saves and evaluates the current line.
// User expectation: Type "a = 2 * 3", press UP, should see "a -> 6" in preview.
func TestArrowUpEvaluatesLine(t *testing.T) {
	// Start with empty document
	doc, err := document.NewDocument("")
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 24

	// Press ENTER to create a new line (cursor moves to line 1)
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newModel.(Model)

	t.Logf("After ENTER: cursorLine=%d", m.cursorLine)

	// Type "a = 2 * 3"
	for _, ch := range "a = 2 * 3" {
		newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
		m = newModel.(Model)
	}

	t.Logf("After typing: cursorLine=%d, editBuf=%q", m.cursorLine, m.editBuf)

	// Press arrow UP - should save and evaluate the line
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = newModel.(Model)

	t.Logf("After UP: cursorLine=%d, userIsTyping=%v", m.cursorLine, m.userIsTyping)

	// Check that the line was saved
	lines := m.GetLines()
	t.Logf("Lines: %v", lines)

	if len(lines) < 2 {
		t.Fatal("Expected at least 2 lines after typing and UP")
	}

	if lines[1] != "a = 2 * 3" {
		t.Errorf("Expected line 1 to be 'a = 2 * 3', got %q", lines[1])
	}

	// Check that the line was evaluated
	results := m.GetLineResults()
	t.Logf("Line results: %d", len(results))
	for i, result := range results {
		t.Logf("  Result[%d]: LineNum=%d IsCalc=%v Value=%q Error=%v",
			i, result.LineNum, result.IsCalc, result.Value, result.Error)
	}

	// Find the result for line 1
	var foundCalc bool
	var calcValue string
	for _, result := range results {
		if result.LineNum == 1 && result.IsCalc {
			foundCalc = true
			calcValue = result.Value
			break
		}
	}

	if !foundCalc {
		t.Error("BUG: Line was not evaluated as calc after arrow UP")
	}

	if calcValue != "6" {
		t.Errorf("Expected calc value '6', got %q", calcValue)
	}

	// Check preview shows the result
	aligned := m.GetAlignedModel(40, 40)
	if len(aligned.PreviewLines) > 1 {
		preview1 := aligned.PreviewLines[1].Content
		t.Logf("Preview line 1: %q", preview1)
		if !strings.Contains(preview1, "6") {
			t.Error("BUG: Preview does not show calculated value '6'")
		}
	}
}

// TestArrowDownEvaluatesLine tests that pressing arrow down saves and evaluates the current line.
func TestArrowDownEvaluatesLine(t *testing.T) {
	// Start with empty document
	doc, err := document.NewDocument("")
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 24

	// Type "b = 5 + 5"
	for _, ch := range "b = 5 + 5" {
		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
		m = newModel.(Model)
	}

	// Press ENTER to create a new line
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newModel.(Model)

	t.Logf("Before UP: cursorLine=%d", m.cursorLine)

	// Press arrow UP to go back
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = newModel.(Model)

	t.Logf("After UP: cursorLine=%d", m.cursorLine)

	// Now press arrow DOWN - should save and evaluate line 0
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = newModel.(Model)

	t.Logf("After DOWN: cursorLine=%d, userIsTyping=%v", m.cursorLine, m.userIsTyping)

	// Check that line 0 was evaluated
	results := m.GetLineResults()
	var foundCalc bool
	var calcValue string
	for _, result := range results {
		t.Logf("Result[%d]: LineNum=%d IsCalc=%v Value=%q", 0, result.LineNum, result.IsCalc, result.Value)
		if result.LineNum == 0 && result.IsCalc {
			foundCalc = true
			calcValue = result.Value
			break
		}
	}

	if !foundCalc {
		t.Error("BUG: Line 0 was not evaluated as calc after arrow DOWN")
	}

	if calcValue != "10" {
		t.Errorf("Expected calc value '10', got %q", calcValue)
	}
}

// TestEnterOnEmptyLineCursorPosition tests that pressing ENTER on an empty line
// creates a new line and positions cursor correctly.
// User expectation: Empty line above "some content", press ENTER, cursor should be on the NEW empty line.
func TestEnterOnEmptyLineCursorPosition(t *testing.T) {
	// Start with empty line, then content
	source := "\nsome content"
	doc, err := document.NewDocument(source)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 24
	m.cursorLine = 0 // Cursor on the empty line
	m.cursorCol = 0

	t.Logf("Before ENTER: cursorLine=%d, cursorCol=%d, TotalLines=%d", m.cursorLine, m.cursorCol, m.TotalLines())
	linesBefore := m.GetLines()
	t.Logf("Lines before: %v", linesBefore)
	for i, line := range linesBefore {
		t.Logf("  [%d] %q", i, line)
	}
	t.Logf("EditBuf before ENTER: %q", m.editBuf)

	// Press ENTER on the empty line
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newModel.(Model)

	t.Logf("After ENTER: cursorLine=%d, cursorCol=%d, TotalLines=%d", m.cursorLine, m.cursorCol, m.TotalLines())
	t.Logf("EditBuf after ENTER: %q", m.editBuf)
	linesAfter := m.GetLines()
	t.Logf("Lines after: %v", linesAfter)
	for i, line := range linesAfter {
		t.Logf("  [%d] %q", i, line)
	}

	// Expected behavior:
	// - Line 0 remains empty
	// - New empty line created at line 1
	// - Cursor should be at line 1, col 0
	// - "some content" moves to line 2

	if m.cursorLine != 1 {
		t.Errorf("BUG: After ENTER on empty line, cursor should be at line 1, got line %d", m.cursorLine)
	}

	if m.cursorCol != 0 {
		t.Errorf("BUG: After ENTER, cursor should be at col 0, got col %d", m.cursorCol)
	}

	lines := m.GetLines()
	if len(lines) < 3 {
		t.Fatalf("Expected at least 3 lines after ENTER, got %d", len(lines))
	}

	if lines[0] != "" {
		t.Errorf("Line 0 should be empty, got %q", lines[0])
	}

	if lines[1] != "" {
		t.Errorf("Line 1 (new line) should be empty, got %q", lines[1])
	}

	if lines[2] != "some content" {
		t.Errorf("Line 2 should be 'some content', got %q", lines[2])
	}
}

// TestHeadingAlignmentAfterEmptyLine tests that heading preview appears on correct line.
// User scenario: empty line, then "# test", then empty line, then "a = 1"
// The heading preview should be on line 2, not line 1.
func TestHeadingAlignmentAfterEmptyLine(t *testing.T) {
	source := "\n# test\n\na = 1\n"
	doc, err := document.NewDocument(source)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 24

	lines := m.GetLines()
	t.Logf("Source lines: %d", len(lines))
	for i, line := range lines {
		t.Logf("  [%d] %q", i, line)
	}

	// Test what glamour renders for this block
	testBlock := "\n# test\n"
	mdRenderer, _ := NewMarkdownRenderer(40)
	if mdRenderer != nil {
		testRendered := mdRenderer.RenderLine(testBlock)
		t.Logf("Glamour input: %q", testBlock)
		t.Logf("Glamour output (%d lines):", len(testRendered))
		for i, line := range testRendered {
			t.Logf("  [%d] %q", i, line)
		}
	}

	// Get aligned model
	aligned := m.GetAlignedModel(40, 40)
	t.Logf("Preview lines: %d", len(aligned.PreviewLines))
	for i, line := range aligned.PreviewLines {
		t.Logf("  [%d] SourceLine=%d Content=%q", i, line.SourceLineIdx, line.Content)
	}

	// Verify alignment: each source line should have exactly one preview line
	if len(aligned.PreviewLines) != len(lines) {
		t.Errorf("ALIGNMENT BUG: Preview has %d lines but source has %d lines",
			len(aligned.PreviewLines), len(lines))
	}

	// Line 0: empty -> preview should be empty
	if aligned.PreviewLines[0].SourceLineIdx != 0 {
		t.Errorf("Preview line 0 should map to source line 0, got %d", aligned.PreviewLines[0].SourceLineIdx)
	}

	// Line 1: "# test" -> preview should show heading
	if aligned.PreviewLines[1].SourceLineIdx != 1 {
		t.Errorf("Preview line 1 should map to source line 1, got %d", aligned.PreviewLines[1].SourceLineIdx)
	}
	if aligned.PreviewLines[1].Content == "" {
		t.Errorf("BUG: Preview line 1 (heading) is empty, should show 'test'")
	}

	// Line 2: empty -> preview should be empty
	if aligned.PreviewLines[2].SourceLineIdx != 2 {
		t.Errorf("Preview line 2 should map to source line 2, got %d", aligned.PreviewLines[2].SourceLineIdx)
	}

	// Line 3: "a = 1" -> preview should show result
	if aligned.PreviewLines[3].SourceLineIdx != 3 {
		t.Errorf("Preview line 3 should map to source line 3, got %d", aligned.PreviewLines[3].SourceLineIdx)
	}
}

// TestEscKeyDoesNothingInNormalMode tests that ESC is a no-op in normal editing mode.
// ESC is only for canceling special modes (globals, export, etc.), not for creating lines.
func TestEscKeyDoesNothingInNormalMode(t *testing.T) {
	doc, err := document.NewDocument("a = 1\n")
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 24
	m.cursorLine = 0
	m.cursorCol = 5 // End of "a = 1"
	m.loadCurrentLineIntoEditBuffer()

	t.Logf("Before ESC: cursorLine=%d, TotalLines=%d", m.cursorLine, m.TotalLines())

	// Press ESC - should do nothing in normal mode
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = newModel.(Model)

	t.Logf("After ESC: cursorLine=%d, TotalLines=%d", m.cursorLine, m.TotalLines())

	// Should still have 1 line (no change)
	if m.TotalLines() != 1 {
		t.Errorf("Expected 1 line after ESC (no change), got %d", m.TotalLines())
	}

	// Cursor should still be at line 0 (no change)
	if m.cursorLine != 0 {
		t.Errorf("Expected cursor at line 0 after ESC (no change), got %d", m.cursorLine)
	}

	// Press ESC again - should still do nothing
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = newModel.(Model)

	t.Logf("After 2nd ESC: cursorLine=%d, TotalLines=%d", m.cursorLine, m.TotalLines())

	// Should still have 1 line
	if m.TotalLines() != 1 {
		t.Errorf("Expected 1 line after 2nd ESC (no change), got %d", m.TotalLines())
	}

	// Cursor should still be at line 0
	if m.cursorLine != 0 {
		t.Errorf("Expected cursor at line 0 after 2nd ESC (no change), got %d", m.cursorLine)
	}
}

// TestDocumentPreservesEmptyLines tests that the document parser correctly handles multiple empty lines.
func TestDocumentPreservesEmptyLines(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "Two empty lines then content",
			input:    "\n\nsome content\n",
			expected: []string{"", "", "some content"},
		},
		{
			name:     "Empty then content",
			input:    "\nsome content\n",
			expected: []string{"", "some content"},
		},
		{
			name:     "Content then empty",
			input:    "some content\n\n",
			expected: []string{"some content", ""},
		},
		{
			name:     "Three empty lines",
			input:    "\n\n\n",
			expected: []string{"", "", ""},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			doc, err := document.NewDocument(tc.input)
			if err != nil {
				t.Fatalf("Failed to create document: %v", err)
			}

			m := New(doc)
			lines := m.GetLines()

			t.Logf("Input: %q", tc.input)
			t.Logf("Expected %d lines: %v", len(tc.expected), tc.expected)
			t.Logf("Got %d lines: %v", len(lines), lines)

			if len(lines) != len(tc.expected) {
				t.Errorf("Expected %d lines, got %d", len(tc.expected), len(lines))
			}

			for i := range tc.expected {
				if i >= len(lines) {
					break
				}
				if lines[i] != tc.expected[i] {
					t.Errorf("Line %d: expected %q, got %q", i, tc.expected[i], lines[i])
				}
			}
		})
	}
}
