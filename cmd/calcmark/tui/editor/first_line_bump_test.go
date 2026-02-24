package editor

import (
	"testing"

	"github.com/CalcMark/go-calcmark/spec/document"
	tea "charm.land/bubbletea/v2"
)

// TestFirstLineBumpWhenTyping reproduces the bug where the first line
// visually shifts down when you start typing in a fresh editor.
//
// Expected: Line stays at visual line 0 when transitioning to edit mode
// Actual: Line bumps down (visual alignment changes)
func TestFirstLineBumpWhenTyping(t *testing.T) {
	// Create editor with single line
	doc, err := document.NewDocument("k")
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 24

	// Get initial visual state before typing
	initialView := m.View()
	t.Logf("Initial view (before typing):\n%s", initialView)

	// Get visual line count before typing
	initialVisualLines := countVisualLines(m)
	t.Logf("Initial visual lines: %d", initialVisualLines)

	// Simulate typing a character (should trigger transitionToEditing)
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	m = result.(Model)

	// Get visual state after typing
	afterView := m.View()
	t.Logf("After typing 'k':\n%s", afterView)

	// Get visual line count after typing
	afterVisualLines := countVisualLines(m)
	t.Logf("After visual lines: %d", afterVisualLines)

	// BUG: Visual line count should remain the same
	// The line content changes from "k" to "kk", but the number of
	// visual lines in the editor should not change just because we
	// started editing.
	if initialVisualLines != afterVisualLines {
		t.Errorf("Visual line count changed from %d to %d when typing started - this causes visual 'bump'",
			initialVisualLines, afterVisualLines)
	}

	// Additional check: cursor visual line should stay at 0
	if m.cursorLine != 0 {
		t.Errorf("Cursor line moved from 0 to %d", m.cursorLine)
	}
}

// countVisualLines counts the number of visual lines in the editor pane.
// This is used to detect when lines visually shift.
func countVisualLines(m Model) int {
	// Use the aligned computation to get actual visual line count
	leftWidth, rightWidth := m.GetPaneWidths(m.width)
	aligned := m.computeAlignedPanes(leftWidth, rightWidth)
	return len(aligned.sourceLines)
}

// TestEmptyEditorFirstCharacter tests typing the very first character
// in a completely empty editor (fresh start).
func TestEmptyEditorFirstCharacter(t *testing.T) {
	// Create editor with empty document (single newline creates 1 empty line)
	doc, err := document.NewDocument("\n")
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 24

	// Verify we start with 1 line
	if m.TotalLines() != 1 {
		t.Fatalf("Expected 1 line, got %d", m.TotalLines())
	}

	initialVisualLines := countVisualLines(m)
	t.Logf("Empty editor initial visual lines: %d", initialVisualLines)
	t.Logf("Initial view:\n%s", m.View())

	// Type first character
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	m = result.(Model)

	afterVisualLines := countVisualLines(m)
	t.Logf("After typing first char, visual lines: %d", afterVisualLines)
	t.Logf("After typing view:\n%s", m.View())

	// Visual line count should stay the same (1 line before, 1 line after)
	if initialVisualLines != afterVisualLines {
		t.Errorf("Visual line count changed from %d to %d when typing first character",
			initialVisualLines, afterVisualLines)
	}

	// Check editBuf (which holds the typed content before debounce saves it)
	if m.editBuf != "x" {
		t.Errorf("Expected editBuf 'x', got %q", m.editBuf)
	}

	// Note: The document lines won't update until debounce timer fires
	// For immediate verification, check editBuf instead
}
