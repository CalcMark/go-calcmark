package editor

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/atotto/clipboard"
)

// handleCut cuts selected text to system clipboard (Ctrl+X).
// Returns the model and any command to execute.
// If no selection exists, does nothing.
// The cut operation is recorded for undo.
func (m Model) handleCut() (tea.Model, tea.Cmd) {
	if !m.HasSelection() {
		m.statusMsg = "Nothing selected"
		return m, nil
	}

	// DeleteSelection returns the deleted text and records undo
	deletedText, cmd := m.DeleteSelection()

	// Write to system clipboard
	if err := clipboard.WriteAll(deletedText); err != nil {
		m.statusMsg = "Clipboard error"
		m.statusIsErr = true
		return m, cmd
	}

	m.statusMsg = "Cut to clipboard"
	m.modified = true
	m.reEvaluate()

	return m, cmd
}

// handleCopy copies selected text to system clipboard (Ctrl+C).
// If no selection exists, returns false to indicate quit should happen.
// This preserves the Unix interrupt behavior when no text is selected.
func (m Model) handleCopy() (tea.Model, tea.Cmd, bool) {
	if !m.HasSelection() {
		// No selection - return false to indicate caller should handle as quit
		return m, nil, false
	}

	// Get selected text without modifying document
	text := m.GetSelectedText()

	// Write to system clipboard
	if err := clipboard.WriteAll(text); err != nil {
		m.statusMsg = "Clipboard error"
		m.statusIsErr = true
		return m, nil, true
	}

	m.statusMsg = "Copied to clipboard"
	return m, nil, true
}

// handlePaste pastes text from system clipboard at cursor position (Ctrl+V).
// Multi-line paste is supported - lines are inserted properly.
// The paste operation is recorded for undo.
func (m Model) handlePaste() (tea.Model, tea.Cmd) {
	text, err := clipboard.ReadAll()
	if err != nil {
		m.statusMsg = "Clipboard error"
		m.statusIsErr = true
		return m, nil
	}

	if text == "" {
		m.statusMsg = "Clipboard empty"
		return m, nil
	}

	// If there's a selection, delete it first
	var cmd tea.Cmd
	if m.HasSelection() {
		_, cmd = m.DeleteSelection()
	}

	// Force undo boundary before paste (per RESEARCH.md)
	m.undoManager.ForceBoundary()

	// Insert the pasted text
	// For multi-line paste, we need to handle line breaks
	lines := strings.Split(text, "\n")

	if len(lines) == 1 {
		// Single line paste - simple insert at cursor
		m.insertTextAtCursor(lines[0])
	} else {
		// Multi-line paste - more complex
		m.insertMultiLineText(lines)
	}

	// Force undo boundary after paste (per RESEARCH.md)
	m.undoManager.ForceBoundary()

	m.statusMsg = "Pasted from clipboard"
	m.modified = true
	m.reEvaluate()

	return m, cmd
}

// insertTextAtCursor inserts text at the current cursor position within the current line.
// Records the operation for undo.
func (m *Model) insertTextAtCursor(text string) {
	if text == "" {
		return
	}

	// Capture state BEFORE the edit for undo
	beforeLine := m.cursorLine
	beforeCol := m.cursorCol
	beforeScroll := m.scrollOffset

	// Insert text at cursor position
	m.transitionToEditing()
	m.editBuf = runeInsert(m.editBuf, m.cursorCol, text)
	m.cursorCol += runeLen(text)

	// Record the insert operation
	op := EditOperation{
		Type:         OpInsert,
		Line:         beforeLine,
		Col:          beforeCol,
		OldText:      "",
		NewText:      text,
		CursorLine:   beforeLine,
		CursorCol:    beforeCol,
		ScrollOffset: beforeScroll,
	}
	m.undoManager.AddOperation(op)
}

// insertMultiLineText handles pasting multiple lines.
// Splits current line at cursor, inserts new lines, joins appropriately.
// Records operations for undo.
func (m *Model) insertMultiLineText(lines []string) {
	if len(lines) == 0 {
		return
	}

	// Capture state BEFORE the edit for undo
	beforeLine := m.cursorLine
	beforeCol := m.cursorCol
	beforeScroll := m.scrollOffset

	m.loadCurrentLineIntoEditBuffer()

	// Split current line at cursor
	textBefore, textAfter := runeSlice(m.editBuf, m.cursorCol)

	// First pasted line appends to text before cursor
	firstLine := textBefore + lines[0]

	// Update current line with first pasted line
	m.editBuf = firstLine
	m.updateCurrentLine(m.editBuf)

	// Insert middle lines (if any) as new complete lines
	for i := 1; i < len(lines)-1; i++ {
		m.insertLineBelow()
		m.editBuf = lines[i]
		m.updateCurrentLine(m.editBuf)
	}

	// Last pasted line prepends to text after cursor
	if len(lines) > 1 {
		m.insertLineBelow()
		lastLine := lines[len(lines)-1] + textAfter
		m.editBuf = lastLine
		m.updateCurrentLine(m.editBuf)

		// Position cursor at end of pasted content (before textAfter)
		m.cursorCol = runeLen(lines[len(lines)-1])
	} else {
		// Only one line - cursor already positioned correctly
		m.cursorCol = runeLen(firstLine)
	}

	// Record as a multi-line insert operation (Replace type for multi-line)
	pastedText := strings.Join(lines, "\n")
	op := EditOperation{
		Type:         OpReplace,
		Line:         beforeLine,
		Col:          beforeCol,
		OldText:      "",
		NewText:      pastedText,
		CursorLine:   beforeLine,
		CursorCol:    beforeCol,
		ScrollOffset: beforeScroll,
	}
	m.undoManager.AddOperation(op)
}
