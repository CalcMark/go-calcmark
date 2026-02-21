package editor

// navigation.go — Cursor movement, scrolling, search, and goto.

import (
	"fmt"
	"strings"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"
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
	lastLine := m.TotalLines() - 1
	if lastLine < 0 {
		lastLine = 0
	}
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
	if col > len(runes) {
		col = len(runes)
	}

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

// adjustScroll ensures cursor is visible (legacy method for compatibility).
// Use adjustScrollForCursor for new code to get scroll margin behavior.
func (m *Model) adjustScroll() {
	m.adjustScrollForCursor()
}

// ========================================
// Search and goto
// ========================================

// searchDocument searches for a term in the document and jumps to the first match.
func (m *Model) searchDocument(term string) {
	m.searchTerm = term
	m.searchMatches = nil
	m.searchIdx = -1

	lines := m.GetLines()
	for i, line := range lines {
		if strings.Contains(strings.ToLower(line), strings.ToLower(term)) {
			m.searchMatches = append(m.searchMatches, i)
		}
	}

	if len(m.searchMatches) == 0 {
		m.statusMsg = fmt.Sprintf("No matches for: %s", term)
		m.statusIsErr = true
		return
	}

	// Jump to first match at or after cursor
	for i, lineNum := range m.searchMatches {
		if lineNum >= m.cursorLine {
			m.searchIdx = i
			m.cursorLine = lineNum
			m.adjustScroll()
			break
		}
	}
	if m.searchIdx == -1 {
		// All matches are before cursor, go to first
		m.searchIdx = 0
		m.cursorLine = m.searchMatches[0]
		m.adjustScroll()
	}

	m.statusMsg = fmt.Sprintf("Match %d of %d: %s", m.searchIdx+1, len(m.searchMatches), term)
}

// nextSearchMatch jumps to the next search match.
func (m *Model) nextSearchMatch() {
	if len(m.searchMatches) == 0 {
		m.statusMsg = "No search active"
		m.statusIsErr = true
		return
	}

	m.searchIdx = (m.searchIdx + 1) % len(m.searchMatches)
	m.cursorLine = m.searchMatches[m.searchIdx]
	m.adjustScroll()
	m.statusMsg = fmt.Sprintf("Match %d of %d: %s", m.searchIdx+1, len(m.searchMatches), m.searchTerm)
}

// prevSearchMatch jumps to the previous search match.
func (m *Model) prevSearchMatch() {
	if len(m.searchMatches) == 0 {
		m.statusMsg = "No search active"
		m.statusIsErr = true
		return
	}

	m.searchIdx--
	if m.searchIdx < 0 {
		m.searchIdx = len(m.searchMatches) - 1
	}
	m.cursorLine = m.searchMatches[m.searchIdx]
	m.adjustScroll()
	m.statusMsg = fmt.Sprintf("Match %d of %d: %s", m.searchIdx+1, len(m.searchMatches), m.searchTerm)
}

// gotoLine jumps to a specific line number.
func (m *Model) gotoLine(lineStr string) {
	var lineNum int
	if _, err := fmt.Sscanf(lineStr, "%d", &lineNum); err != nil {
		m.statusMsg = fmt.Sprintf("Invalid line number: %s", lineStr)
		m.statusIsErr = true
		return
	}

	// Convert to 0-indexed
	lineNum--

	total := m.TotalLines()
	if lineNum < 0 {
		lineNum = 0
	}
	if lineNum >= total {
		lineNum = total - 1
	}

	m.cursorLine = lineNum
	m.adjustScroll()
	m.statusMsg = fmt.Sprintf("Line %d", lineNum+1)
}
