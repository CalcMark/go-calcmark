package editor

import (
	"strings"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"
)

// HasSelection returns true if there is an active text selection.
// A selection exists when the anchor is set (>= 0) and differs from cursor.
func (m *Model) HasSelection() bool {
	if m.selectionAnchorLine < 0 {
		return false
	}
	// Selection exists if anchor differs from cursor
	return m.selectionAnchorLine != m.cursorLine || m.selectionAnchorCol != m.cursorCol
}

// SetSelectionAnchor sets the selection anchor to the current cursor position.
// Called when starting a selection (e.g., when Shift is held).
func (m *Model) SetSelectionAnchor() {
	m.selectionAnchorLine = m.cursorLine
	m.selectionAnchorCol = m.cursorCol
}

// ensureSelectionAnchor sets the selection anchor at the current cursor position
// only if no selection is currently active. This is called before shift+navigation
// to start a new selection or extend an existing one.
func (m *Model) ensureSelectionAnchor() {
	if m.selectionAnchorLine < 0 {
		m.SetSelectionAnchor()
	}
}

// ClearSelection clears the current selection by resetting the anchor.
func (m *Model) ClearSelection() {
	m.selectionAnchorLine = -1
	m.selectionAnchorCol = -1
}

// GetSelectionRange returns the normalized selection range where start <= end.
// Returns (startLine, startCol, endLine, endCol).
// Returns (-1, -1, -1, -1) if no selection.
func (m *Model) GetSelectionRange() (startLine, startCol, endLine, endCol int) {
	if !m.HasSelection() {
		return -1, -1, -1, -1
	}

	anchorLine := m.selectionAnchorLine
	anchorCol := m.selectionAnchorCol
	curLine := m.cursorLine
	curCol := m.cursorCol

	// Normalize so start <= end
	// Compare lines first, then columns within same line
	if anchorLine < curLine || (anchorLine == curLine && anchorCol <= curCol) {
		// Anchor is before or at cursor
		return anchorLine, anchorCol, curLine, curCol
	}
	// Cursor is before anchor
	return curLine, curCol, anchorLine, anchorCol
}

// selectionRuneCount returns the number of runes in the current selection
// without materializing the full selected text string. Used by the status bar
// to avoid the overhead of GetSelectedText() on every render frame.
func (m *Model) selectionRuneCount() int {
	startLine, startCol, endLine, endCol := m.GetSelectionRange()
	if startLine < 0 {
		return 0
	}

	lines := m.GetLines()
	if len(lines) == 0 {
		return 0
	}

	// Clamp to valid line range
	if startLine >= len(lines) {
		startLine = len(lines) - 1
	}
	if endLine >= len(lines) {
		endLine = len(lines) - 1
	}

	if startLine == endLine {
		line := lines[startLine]
		lineRuneCount := utf8.RuneCountInString(line)
		sc := min(max(startCol, 0), lineRuneCount)
		ec := min(max(endCol, 0), lineRuneCount)
		return ec - sc
	}

	// Multi-line: first partial + middle full lines + last partial + newlines
	count := 0

	// First line: from startCol to end
	firstRuneCount := utf8.RuneCountInString(lines[startLine])
	sc := min(max(startCol, 0), firstRuneCount)
	count += firstRuneCount - sc

	// Middle lines: full lines + newline separators
	for i := startLine + 1; i < endLine; i++ {
		count += 1 // newline
		count += utf8.RuneCountInString(lines[i])
	}

	// Last line: from start to endCol + newline before it
	count += 1 // newline
	lastRuneCount := utf8.RuneCountInString(lines[endLine])
	ec := min(max(endCol, 0), lastRuneCount)
	count += ec

	return count
}

// GetSelectedText returns the selected text as a string.
// For multi-line selections, lines are joined with "\n".
// Returns empty string if no selection.
func (m *Model) GetSelectedText() string {
	startLine, startCol, endLine, endCol := m.GetSelectionRange()
	if startLine < 0 {
		return ""
	}

	lines := m.GetLines()
	if len(lines) == 0 {
		return ""
	}

	// Clamp to valid line range
	if startLine >= len(lines) {
		startLine = len(lines) - 1
	}
	if endLine >= len(lines) {
		endLine = len(lines) - 1
	}

	if startLine == endLine {
		// Same-line selection
		line := lines[startLine]
		runes := []rune(line)

		// Clamp columns to valid range
		if startCol < 0 {
			startCol = 0
		}
		if startCol > len(runes) {
			startCol = len(runes)
		}
		if endCol < 0 {
			endCol = 0
		}
		if endCol > len(runes) {
			endCol = len(runes)
		}

		return string(runes[startCol:endCol])
	}

	// Multi-line selection
	var result []string

	// First line: from startCol to end
	firstLine := lines[startLine]
	firstRunes := []rune(firstLine)
	if startCol < 0 {
		startCol = 0
	}
	if startCol > len(firstRunes) {
		startCol = len(firstRunes)
	}
	result = append(result, string(firstRunes[startCol:]))

	// Middle lines: full lines
	for i := startLine + 1; i < endLine; i++ {
		result = append(result, lines[i])
	}

	// Last line: from start to endCol
	lastLine := lines[endLine]
	lastRunes := []rune(lastLine)
	if endCol < 0 {
		endCol = 0
	}
	if endCol > len(lastRunes) {
		endCol = len(lastRunes)
	}
	result = append(result, string(lastRunes[:endCol]))

	return strings.Join(result, "\n")
}

// DeleteSelection deletes the selected text from the document and returns the deleted text.
// This integrates with the undo system via recordEdit().
// Returns empty string if no selection.
// After deletion, the cursor is placed at the start of the selection and selection is cleared.
func (m *Model) DeleteSelection() (deletedText string, cmd tea.Cmd) {
	startLine, startCol, endLine, endCol := m.GetSelectionRange()
	if startLine < 0 {
		return "", nil
	}

	// Get the selected text before deletion
	deletedText = m.GetSelectedText()
	if deletedText == "" {
		m.ClearSelection()
		return "", nil
	}

	// Capture state BEFORE the edit for undo
	beforeLine := m.cursorLine
	beforeCol := m.cursorCol
	beforeScroll := m.scrollOffset

	// Force boundary for multi-line or significant deletions
	if startLine != endLine {
		m.undoManager.ForceBoundary()
	}

	lines := m.GetLines()

	if startLine == endLine {
		// Same-line deletion
		line := lines[startLine]
		runes := []rune(line)

		// Delete the selected portion
		newLine := string(runes[:startCol]) + string(runes[endCol:])

		// Update the document
		m.cursorLine = startLine
		m.cursorCol = startCol
		m.transitionToEditing()
		m.editBuf = newLine

		// Record the delete operation
		op := EditOperation{
			Type:         OpDelete,
			Line:         startLine,
			Col:          startCol,
			OldText:      deletedText,
			NewText:      "",
			CursorLine:   beforeLine,
			CursorCol:    beforeCol,
			ScrollOffset: beforeScroll,
		}
		cmd = m.recordEdit(op)
	} else {
		// Multi-line deletion
		// Get the part before selection on first line
		firstLine := lines[startLine]
		firstRunes := []rune(firstLine)
		beforePart := string(firstRunes[:startCol])

		// Get the part after selection on last line
		lastLine := lines[endLine]
		lastRunes := []rune(lastLine)
		afterPart := string(lastRunes[endCol:])

		// The new merged line
		mergedLine := beforePart + afterPart

		// Record as Replace operation
		op := EditOperation{
			Type:         OpReplace,
			Line:         startLine,
			Col:          startCol,
			OldText:      deletedText,
			NewText:      "",
			CursorLine:   beforeLine,
			CursorCol:    beforeCol,
			ScrollOffset: beforeScroll,
		}

		// Delete lines from end to start+1 (preserve first line for merging)
		// Work backwards to maintain correct indices
		for lineNum := endLine; lineNum > startLine; lineNum-- {
			m.cursorLine = lineNum
			m.deleteLine()
		}

		// Update the first line with merged content
		m.cursorLine = startLine
		m.cursorCol = startCol
		m.transitionToEditing()
		m.editBuf = mergedLine

		cmd = m.recordEdit(op)
	}

	// Clear selection
	m.ClearSelection()

	// Position cursor at start of where selection was
	m.cursorLine = startLine
	m.cursorCol = startCol

	return deletedText, cmd
}

// SelectAll selects all text in the document.
// Sets anchor to 0,0 and moves cursor to end of last line.
func (m *Model) SelectAll() {
	lines := m.GetLines()
	if len(lines) == 0 {
		// Empty document - nothing to select
		return
	}

	// Set anchor at document start
	m.selectionAnchorLine = 0
	m.selectionAnchorCol = 0

	// Move cursor to end of last line
	m.cursorLine = len(lines) - 1
	lastLine := lines[m.cursorLine]
	m.cursorCol = runeLen(lastLine)
}
