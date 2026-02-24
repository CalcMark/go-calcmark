package editor

import (
	"testing"

	"github.com/CalcMark/go-calcmark/spec/document"
	tea "charm.land/bubbletea/v2"
)

// TestDeleteLastCharThenType reproduces the exact bug:
// 1. Type "abc"
// 2. Press backspace 3 times (deleting c, b, then a)
// 3. Type 'b'
// Expected: line contains just "b"
// Bug: line contains "ba" because empty editBuf wasn't saved to document
func TestDeleteLastCharThenType(t *testing.T) {
	doc, err := document.NewDocument("")
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 24

	// Type "abc"
	for _, r := range "abc" {
		m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = m2.(Model)
	}

	t.Logf("After 'abc': cursorCol=%d, editBuf=%q", m.cursorCol, m.editBuf)

	if m.editBuf != "abc" {
		t.Fatalf("Expected editBuf='abc', got %q", m.editBuf)
	}

	// Simulate debounce firing after typing (as happens in real app)
	m.transitionToProcessing()
	t.Logf("After debounce (saves 'abc'): doc line=%q", m.GetLines()[0])

	// Delete 'c'
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	m = m2.(Model)
	t.Logf("After delete 'c': cursorCol=%d, editBuf=%q", m.cursorCol, m.editBuf)

	// Simulate debounce firing
	m.transitionToProcessing()
	t.Logf("After debounce (saves 'ab'): doc line=%q", m.GetLines()[0])

	// Delete 'b'
	m3, _ := m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	m = m3.(Model)
	t.Logf("After delete 'b': cursorCol=%d, editBuf=%q", m.cursorCol, m.editBuf)

	// Simulate debounce firing
	m.transitionToProcessing()
	t.Logf("After debounce (saves 'a'): doc line=%q", m.GetLines()[0])

	// Delete 'a' - this is where the bug manifests
	m4, _ := m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	m = m4.(Model)
	t.Logf("After delete 'a': cursorCol=%d, editBuf=%q", m.cursorCol, m.editBuf)

	if m.editBuf != "" {
		t.Errorf("Expected editBuf='', got %q", m.editBuf)
	}

	// Simulate debounce firing - BUG: empty editBuf not saved!
	m.transitionToProcessing()
	docLine := m.GetLines()[0]
	t.Logf("After debounce (should save ''): doc line=%q", docLine)

	// Now type 'b' - this should result in just "b", not "ba"
	m5, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
	m = m5.(Model)
	t.Logf("After type 'b': cursorCol=%d, editBuf=%q", m.cursorCol, m.editBuf)

	// The bug: editBuf becomes "ba" instead of "b"
	if m.editBuf != "b" {
		t.Errorf("Expected editBuf='b', got %q (BUG: empty line wasn't saved to document)", m.editBuf)
	}
}
