package editor

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/CalcMark/go-calcmark/spec/document"
)

// TestInteractive_OrderedListThenBlankThenHeading simulates the user's reported bug:
// 1. Start with ordered list
// 2. Type blank line after it
// 3. Type a heading
// This should maintain proper alignment at every step.
func TestInteractive_OrderedListThenBlankThenHeading(t *testing.T) {
	// Start with an ordered list
	initialSource := "1. First\n2. Second\n3. Third"
	doc, err := document.NewDocument(initialSource)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 24

	// Verify initial state
	sourceLines := m.GetLines()
	if len(sourceLines) != 3 {
		t.Fatalf("Initial source should have 3 lines, got %d", len(sourceLines))
	}

	aligned := m.GetAlignedModel(40, 40)
	if len(aligned.PreviewLines) != 3 {
		t.Errorf("Initial preview should have 3 lines, got %d", len(aligned.PreviewLines))
	}

	// Navigate to end of last line
	m.cursorLine = 2
	m.cursorCol = len(sourceLines[2])

	// Press ENTER to create new line
	newModel, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = newModel.(Model)

	t.Logf("After first ENTER: cursorLine=%d, TotalLines=%d", m.cursorLine, m.TotalLines())

	// Check alignment after ENTER
	sourceLines = m.GetLines()
	aligned = m.GetAlignedModel(40, 40)
	if len(aligned.PreviewLines) != len(sourceLines) {
		t.Errorf("After ENTER: Preview has %d lines but source has %d lines",
			len(aligned.PreviewLines), len(sourceLines))
	}

	// Press ENTER again to create blank line
	newModel, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = newModel.(Model)

	t.Logf("After second ENTER (blank line): cursorLine=%d, TotalLines=%d", m.cursorLine, m.TotalLines())

	// Check alignment after blank line
	sourceLines = m.GetLines()
	aligned = m.GetAlignedModel(40, 40)
	if len(aligned.PreviewLines) != len(sourceLines) {
		t.Errorf("After blank line: Preview has %d lines but source has %d lines",
			len(aligned.PreviewLines), len(sourceLines))
	}

	// Type a heading: "# Title"
	for _, ch := range "# Title" {
		newModel, _ = m.Update(tea.KeyPressMsg{Code: ch, Text: string(ch)})
		m = newModel.(Model)
	}

	t.Logf("After typing '# Title': cursorLine=%d, editBuf=%q", m.cursorLine, m.editBuf)

	// CRITICAL: Check alignment while still typing (editBuf not yet saved)
	sourceLines = m.GetLines()
	aligned = m.GetAlignedModel(40, 40)

	t.Logf("While typing:")
	t.Logf("  Source lines: %d", len(sourceLines))
	t.Logf("  Preview lines: %d", len(aligned.PreviewLines))
	t.Logf("  EditBuf: %q", m.editBuf)
	t.Logf("  UserIsTyping: %v", m.userIsTyping)

	// The aligned model should account for the editBuf
	// When user is typing, the preview should still align with what they see in source
	if len(aligned.PreviewLines) != len(sourceLines) {
		t.Errorf("ALIGNMENT BUG WHILE TYPING: Preview has %d lines but source has %d lines",
			len(aligned.PreviewLines), len(sourceLines))
		t.Logf("This is likely the bug the user reported!")
		for i, line := range aligned.PreviewLines {
			t.Logf("  Preview[%d]: SourceLine=%d Content=%q",
				i, line.SourceLineIdx, line.Content)
		}
	}

	// Press ENTER to finish typing the heading
	newModel, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = newModel.(Model)

	t.Logf("After ENTER (finish heading): cursorLine=%d", m.cursorLine)

	// Final check
	sourceLines = m.GetLines()
	aligned = m.GetAlignedModel(40, 40)
	if len(aligned.PreviewLines) != len(sourceLines) {
		t.Errorf("Final state: Preview has %d lines but source has %d lines",
			len(aligned.PreviewLines), len(sourceLines))
	}

	// Final document should be:
	// 1. First
	// 2. Second
	// 3. Third
	// (blank)
	// # Title
	// (new blank line from final ENTER)
	expectedLines := 6
	if len(sourceLines) != expectedLines {
		t.Errorf("Final document should have %d lines, got %d: %v",
			expectedLines, len(sourceLines), sourceLines)
	}
}

// TestInteractive_HeadingVisibilityWhileTyping tests if heading preview appears
// while the user is still typing (before pressing ENTER/ESC).
func TestInteractive_HeadingVisibilityWhileTyping(t *testing.T) {
	// Start with ordered list
	initialSource := "1. First\n2. Second"
	doc, err := document.NewDocument(initialSource)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 24

	// Navigate to end
	m.cursorLine = 1
	m.cursorCol = len(m.GetLines()[1])

	// Press ENTER twice to create blank line
	newModel, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = newModel.(Model)
	newModel, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = newModel.(Model)

	// Start typing "# Title"
	for _, ch := range "# Title" {
		newModel, _ = m.Update(tea.KeyPressMsg{Code: ch, Text: string(ch)})
		m = newModel.(Model)
	}

	// Check if the heading is visible in preview WHILE TYPING
	sourceLines := m.GetLines()
	results := m.GetLineResults()

	t.Logf("While typing heading:")
	t.Logf("  EditBuf: %q", m.editBuf)
	t.Logf("  CursorLine: %d", m.cursorLine)
	t.Logf("  Source lines: %d", len(sourceLines))
	for i, line := range sourceLines {
		t.Logf("    Source[%d]: %q", i, line)
	}
	t.Logf("  Line results: %d", len(results))
	for i, result := range results {
		t.Logf("    Result[%d]: LineNum=%d BlockID=%s Source=%q",
			i, result.LineNum, result.BlockID, result.Source)
	}

	// Test what glamour would render for the complete block including editBuf
	testBlock := "1. First\n2. Second\n\n# Title"
	mdRenderer, _ := NewMarkdownRenderer(40)
	if mdRenderer != nil {
		testRendered := mdRenderer.RenderLine(testBlock)
		t.Logf("  Test glamour render of complete block:")
		for i, line := range testRendered {
			t.Logf("    Rendered[%d]: %q", i, line)
		}
	}

	aligned := m.GetAlignedModel(40, 40)
	t.Logf("  Preview lines: %d", len(aligned.PreviewLines))

	// Look for the heading content in preview
	foundHeading := false
	for i, line := range aligned.PreviewLines {
		t.Logf("  Preview[%d]: SourceLine=%d Content=%q",
			i, line.SourceLineIdx, line.Content)
		if line.SourceLineIdx == m.cursorLine {
			// This preview line corresponds to the line being edited
			// It should show the heading being typed
			if line.Content != "" {
				foundHeading = true
				t.Logf("    ^ This is the preview of the line being edited")
			}
		}
	}

	if !foundHeading {
		t.Error("BUG: Heading preview not visible while typing - editBuf not being used in preview")
	} else {
		t.Log("SUCCESS: Heading preview is visible while typing!")
	}
}
