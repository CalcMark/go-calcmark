package editor

import (
	"strings"
	"testing"

	"github.com/CalcMark/go-calcmark/spec/document"
	tea "charm.land/bubbletea/v2"
)

// TestVisualStateAfterTypingAndEnter verifies that after typing and pressing ENTER,
// the document is still visible (not empty) and cursor is positioned correctly.
//
// Bug report: Type "1. asdf" then ENTER makes document appear empty.
func TestVisualStateAfterTypingAndEnter(t *testing.T) {
	// Start with empty document
	doc, err := document.NewDocument("")
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 24

	t.Logf("=== INITIAL STATE ===")
	t.Logf("TotalLines: %d", m.TotalLines())
	t.Logf("GetLines: %v", m.GetLines())
	t.Logf("cursorLine: %d, cursorCol: %d", m.cursorLine, m.cursorCol)
	t.Logf("editBuf: %q", m.editBuf)

	// Type "1. asdf"
	for _, ch := range "1. asdf" {
		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
		m = result.(Model)
	}

	t.Logf("\n=== AFTER TYPING '1. asdf' ===")
	t.Logf("TotalLines: %d", m.TotalLines())
	docLines := m.GetLines()
	t.Logf("GetLines (%d lines):", len(docLines))
	for i, line := range docLines {
		t.Logf("  [%d] %q", i, line)
	}
	t.Logf("cursorLine: %d, cursorCol: %d", m.cursorLine, m.cursorCol)
	t.Logf("editBuf: %q", m.editBuf)
	t.Logf("userIsTyping: %v", m.userIsTyping)

	// Document should have content (either in lines or editBuf)
	if len(docLines) == 0 && m.editBuf == "" {
		t.Fatal("After typing, document should have content (either in GetLines or editBuf)")
	}

	// Press ENTER
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = result.(Model)

	t.Logf("\n=== AFTER PRESSING ENTER ===")
	t.Logf("TotalLines: %d", m.TotalLines())
	docLinesAfter := m.GetLines()
	t.Logf("GetLines (%d lines):", len(docLinesAfter))
	for i, line := range docLinesAfter {
		t.Logf("  [%d] %q", i, line)
	}
	t.Logf("cursorLine: %d, cursorCol: %d", m.cursorLine, m.cursorCol)
	t.Logf("editBuf: %q", m.editBuf)
	t.Logf("userIsTyping: %v", m.userIsTyping)

	// CRITICAL: Document should NOT be empty after ENTER
	if len(docLinesAfter) == 0 {
		t.Fatal("BUG: Document appears empty after pressing ENTER (GetLines returned 0 lines)")
	}

	// The text we typed should still be visible somewhere
	foundTypedText := false
	for _, line := range docLinesAfter {
		if strings.Contains(line, "1. asdf") {
			foundTypedText = true
			break
		}
	}
	if !foundTypedText && m.editBuf != "1. asdf" {
		t.Error("BUG: Typed text '1. asdf' has disappeared from both document lines and editBuf")
	}

	// Cursor should be at valid position
	if m.cursorLine < 0 {
		t.Errorf("BUG: Cursor line is negative: %d", m.cursorLine)
	}
	if m.cursorLine >= m.TotalLines() {
		t.Errorf("BUG: Cursor line %d is out of bounds (total lines: %d)",
			m.cursorLine, m.TotalLines())
	}
}

// TestVisualStateView verifies that View() produces non-empty output when document has content
func TestVisualStateView(t *testing.T) {
	doc, err := document.NewDocument("")
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 24

	// Type some content
	for _, ch := range "hello" {
		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
		m = result.(Model)
	}

	// Get the visual output
	view := m.View()

	t.Logf("View output length: %d bytes", len(view))
	t.Logf("View contains lines: %d", strings.Count(view, "\n"))

	// View should not be empty
	if view == "" {
		t.Fatal("BUG: View() returned empty string")
	}

	// View should contain the typed text (either in source or preview)
	if !strings.Contains(view, "hello") && !strings.Contains(view, "editBuf") {
		t.Error("BUG: View() does not show the typed text 'hello'")
	}
}

// TestDocumentConsistency verifies that GetLines() matches the underlying document
func TestDocumentConsistency(t *testing.T) {
	doc, _ := document.NewDocument("line1\nline2")
	m := New(doc)

	// GetLines should match document structure
	docLines := m.GetLines()
	t.Logf("GetLines returned: %v", docLines)

	// Count lines in document
	blocks := m.doc.GetBlocks()
	totalSourceLines := 0
	for _, block := range blocks {
		source := block.Block.Source()
		totalSourceLines += len(source)
		t.Logf("Block has %d source lines: %v", len(source), source)
	}

	t.Logf("Total source lines in document: %d", totalSourceLines)
	t.Logf("GetLines() returned: %d lines", len(docLines))

	// They should match (unless editBuf is active)
	if m.editBuf == "" && len(docLines) != totalSourceLines {
		t.Errorf("Inconsistency: document has %d source lines but GetLines() returned %d",
			totalSourceLines, len(docLines))
	}
}
