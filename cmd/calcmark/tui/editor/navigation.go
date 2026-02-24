package editor

// navigation.go — Cursor movement, scrolling, search, and goto.

import (
	"unicode"

	tea "charm.land/bubbletea/v2"
)

// ========================================
// Arrow key navigation
// ========================================

func (m Model) handleUpKey() (tea.Model, tea.Cmd) {
	m.ClearSelection()
	m.undoManager.ForceBoundary()
	m.loadCurrentLineIntoEditBuffer()
	if m.cursorLine > 0 {
		m.saveCurrentLineAndMoveTo(m.cursorLine - 1)
	}
	return m, nil
}

func (m Model) handleDownKey() (tea.Model, tea.Cmd) {
	m.ClearSelection()
	m.undoManager.ForceBoundary()
	m.loadCurrentLineIntoEditBuffer()
	if m.cursorLine < m.TotalLines()-1 {
		m.saveCurrentLineAndMoveTo(m.cursorLine + 1)
	}
	return m, nil
}

func (m Model) handleLeftKey() (tea.Model, tea.Cmd) {
	m.ClearSelection()
	m.undoManager.ForceBoundary()
	m.loadCurrentLineIntoEditBuffer()
	if m.cursorCol > 0 {
		m.cursorCol--
	} else if m.cursorLine > 0 {
		// At start of line - move to end of previous line
		m.saveCurrentLineAndMoveTo(m.cursorLine - 1)
		m.cursorCol = runeLen(m.editBuf)
	}
	return m, nil
}

func (m Model) handleRightKey() (tea.Model, tea.Cmd) {
	m.ClearSelection()
	m.undoManager.ForceBoundary()
	m.loadCurrentLineIntoEditBuffer()
	if m.cursorCol < runeLen(m.editBuf) {
		m.cursorCol++
	} else if m.cursorLine < m.TotalLines()-1 {
		// At end of line - move to start of next line
		m.saveCurrentLineAndMoveTo(m.cursorLine + 1)
		m.cursorCol = 0
	}
	return m, nil
}

// ========================================
// Page and boundary navigation
// ========================================

func (m Model) handlePageUpKey() (tea.Model, tea.Cmd) {
	m.ClearSelection()
	m.undoManager.ForceBoundary()
	m.loadCurrentLineIntoEditBuffer()
	m.moveCursor(-(m.height - 4), 0)
	return m, nil
}

func (m Model) handlePageDownKey() (tea.Model, tea.Cmd) {
	m.ClearSelection()
	m.undoManager.ForceBoundary()
	m.loadCurrentLineIntoEditBuffer()
	m.moveCursor(m.height-4, 0)
	return m, nil
}

func (m Model) handleHomeKey() (tea.Model, tea.Cmd) {
	m.ClearSelection()
	m.undoManager.ForceBoundary()
	m.loadCurrentLineIntoEditBuffer()
	m.cursorCol = 0
	return m, nil
}

func (m Model) handleEndKey() (tea.Model, tea.Cmd) {
	m.ClearSelection()
	m.undoManager.ForceBoundary()
	m.loadCurrentLineIntoEditBuffer()
	m.cursorCol = runeLen(m.editBuf)
	return m, nil
}

// handleCtrlHomeKey moves cursor to document start (line 0, column 0).
func (m Model) handleCtrlHomeKey() (tea.Model, tea.Cmd) {
	m.ClearSelection()
	m.undoManager.ForceBoundary()
	m.loadCurrentLineIntoEditBuffer()
	m.saveCurrentLineAndMoveTo(0)
	m.cursorCol = 0
	return m, nil
}

// handleCtrlEndKey moves cursor to document end (last line, end of line).
func (m Model) handleCtrlEndKey() (tea.Model, tea.Cmd) {
	m.ClearSelection()
	m.undoManager.ForceBoundary()
	m.loadCurrentLineIntoEditBuffer()
	lastLine := max(m.TotalLines()-1, 0)
	m.saveCurrentLineAndMoveTo(lastLine)
	m.cursorCol = runeLen(m.editBuf)
	return m, nil
}

// ========================================
// Word navigation (Ctrl+Arrow / Alt+B/F)
// ========================================

// handleCtrlLeftKey moves cursor left to previous word boundary.
// Word boundaries are determined by unicode.IsSpace and unicode.IsPunct.
// If at column 0, wraps to end of previous line first (like handleLeftKey).
func (m Model) handleCtrlLeftKey() (tea.Model, tea.Cmd) {
	m.ClearSelection()
	m.undoManager.ForceBoundary()
	m.loadCurrentLineIntoEditBuffer()

	// If at start of line, move to end of previous line first
	if m.cursorCol == 0 {
		if m.cursorLine > 0 {
			m.saveCurrentLineAndMoveTo(m.cursorLine - 1)
			m.cursorCol = runeLen(m.editBuf)
		}
		return m, nil
	}

	// Move backwards to find word boundary
	runes := []rune(m.editBuf)
	col := m.cursorCol

	// Clamp col to valid range to prevent index out of bounds
	col = min(col, len(runes))

	// Skip whitespace backwards
	for col > 0 && unicode.IsSpace(runes[col-1]) {
		col--
	}

	// Skip word characters backwards (non-space, non-punct)
	for col > 0 && !unicode.IsSpace(runes[col-1]) && !unicode.IsPunct(runes[col-1]) {
		col--
	}

	// If we only skipped punctuation, skip it too
	if col == m.cursorCol || col == len(runes) {
		for col > 0 && unicode.IsPunct(runes[col-1]) {
			col--
		}
	}

	m.cursorCol = col
	return m, nil
}

// handleCtrlRightKey moves cursor right to next word boundary.
// Word boundaries are determined by unicode.IsSpace and unicode.IsPunct.
// If at end of line, wraps to start of next line first (like handleRightKey).
func (m Model) handleCtrlRightKey() (tea.Model, tea.Cmd) {
	m.ClearSelection()
	m.undoManager.ForceBoundary()
	m.loadCurrentLineIntoEditBuffer()

	runes := []rune(m.editBuf)
	lineLen := len(runes)

	// Clamp cursorCol to valid range
	if m.cursorCol > lineLen {
		m.cursorCol = lineLen
	}

	// If at end of line, move to start of next line
	if m.cursorCol >= lineLen {
		if m.cursorLine < m.TotalLines()-1 {
			m.saveCurrentLineAndMoveTo(m.cursorLine + 1)
			m.cursorCol = 0
		}
		return m, nil
	}

	col := m.cursorCol

	// Skip current word characters forward (non-space, non-punct)
	for col < lineLen && !unicode.IsSpace(runes[col]) && !unicode.IsPunct(runes[col]) {
		col++
	}

	// Skip punctuation forward
	for col < lineLen && unicode.IsPunct(runes[col]) {
		col++
	}

	// Skip whitespace forward
	for col < lineLen && unicode.IsSpace(runes[col]) {
		col++
	}

	m.cursorCol = col
	return m, nil
}

// ========================================
// Cursor movement and scrolling
// ========================================

// moveCursor moves the cursor by delta lines and columns.
func (m *Model) moveCursor(dLine, dCol int) {
	total := m.TotalLines()
	if total == 0 {
		return
	}

	// Move line
	m.cursorLine += dLine
	if m.cursorLine < 0 {
		m.cursorLine = 0
	}
	if m.cursorLine >= total {
		m.cursorLine = total - 1
	}

	// Move column
	lines := m.GetLines()
	if m.cursorLine < len(lines) {
		lineLen := len(lines[m.cursorLine])
		m.cursorCol += dCol
		if m.cursorCol < 0 {
			m.cursorCol = 0
		}
		if m.cursorCol > lineLen {
			m.cursorCol = lineLen
		}
	}

	// Adjust scroll with margin
	m.adjustScrollForCursor()
}

// getVisibleHeight returns the number of visible content lines in the viewport.
// This accounts for the 6-line overhead from status bar, title, etc.
func (m *Model) getVisibleHeight() int {
	return m.height - 6
}

// adjustScrollForCursor ensures the cursor is visible within the viewport,
// maintaining scrollMargin lines of context above/below when possible.
func (m *Model) adjustScrollForCursor() {
	visibleHeight := m.getVisibleHeight()
	if visibleHeight <= 0 {
		return
	}

	// Ensure cursor has at least scrollMargin lines above it
	if m.cursorLine < m.scrollOffset+scrollMargin {
		m.scrollOffset = max(0, m.cursorLine-scrollMargin)
	}

	// Ensure cursor has at least scrollMargin lines below it
	if m.cursorLine >= m.scrollOffset+visibleHeight-scrollMargin {
		m.scrollOffset = m.cursorLine - visibleHeight + scrollMargin + 1
	}

	// Clamp scroll offset to valid range
	maxScroll := max(0, m.TotalLines()-visibleHeight)
	m.scrollOffset = min(m.scrollOffset, maxScroll)
}
