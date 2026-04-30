package editor

// navigation.go — Cursor movement, scrolling, search, and goto.

import (
	"unicode"

	tea "charm.land/bubbletea/v2"
	"github.com/CalcMark/go-calcmark/v2/cmd/calcmark/tui/components"
)

// ========================================
// Arrow key navigation
// ========================================

// prepareNavigation is the common preamble for all navigation handlers.
// When extendSelection is true (Shift held), the selection anchor is set/preserved;
// otherwise any active selection is cleared.
func (m *Model) prepareNavigation(extendSelection bool) {
	if extendSelection {
		m.ensureSelectionAnchor()
	} else {
		m.ClearSelection()
	}
	m.undoManager.ForceBoundary()
	m.loadCurrentLineIntoEditBuffer()
}

func (m Model) handleUpKey() (tea.Model, tea.Cmd) {
	if m.HasSelection() {
		startLine, startCol, _, _ := m.GetSelectionRange()
		m.collapseSelectionTo(startLine, startCol)
		return m, nil
	}
	m.prepareNavigation(false)
	if m.cursorLine > 0 {
		// In Reading mode, jump to the previous visible source line.
		// A multi-line paragraph renders as one visual block; skip past it in one press.
		if m.previewMode == PreviewReading && len(m.readingNav.visibleLines) > 0 {
			prev := m.readingNav.prevVisible(m.cursorLine)
			if prev < m.cursorLine {
				m.saveCurrentLineAndMoveTo(prev)
			}
		} else {
			m.saveCurrentLineAndMoveTo(m.cursorLine - 1)
		}
	}
	return m, nil
}

func (m Model) handleDownKey() (tea.Model, tea.Cmd) {
	if m.HasSelection() {
		_, _, endLine, endCol := m.GetSelectionRange()
		m.collapseSelectionTo(endLine, endCol)
		return m, nil
	}
	m.prepareNavigation(false)
	if m.cursorLine < m.TotalLines()-1 {
		// In Reading mode, jump to the next visible source line.
		if m.previewMode == PreviewReading && len(m.readingNav.visibleLines) > 0 {
			next := m.readingNav.nextVisible(m.cursorLine)
			if next > m.cursorLine {
				m.saveCurrentLineAndMoveTo(next)
			}
		} else {
			m.saveCurrentLineAndMoveTo(m.cursorLine + 1)
		}
	}
	return m, nil
}

func (m Model) handleLeftKey() (tea.Model, tea.Cmd) {
	if m.HasSelection() {
		startLine, startCol, _, _ := m.GetSelectionRange()
		m.collapseSelectionTo(startLine, startCol)
		return m, nil
	}
	m.prepareNavigation(false)
	if m.cursorCol > 0 {
		m.cursorCol--
	} else if m.cursorLine > 0 {
		m.saveCurrentLineAndMoveTo(m.cursorLine - 1)
		m.cursorCol = runeLen(m.editBuf)
	}
	return m, nil
}

func (m Model) handleRightKey() (tea.Model, tea.Cmd) {
	if m.HasSelection() {
		_, _, endLine, endCol := m.GetSelectionRange()
		m.collapseSelectionTo(endLine, endCol)
		return m, nil
	}
	m.prepareNavigation(false)
	if m.cursorCol < runeLen(m.editBuf) {
		m.cursorCol++
	} else if m.cursorLine < m.TotalLines()-1 {
		m.saveCurrentLineAndMoveTo(m.cursorLine + 1)
		m.cursorCol = 0
	}
	return m, nil
}

// ========================================
// Mouse scrolling
// ========================================

// mouseScrollLines is the number of lines to scroll per mouse wheel tick.
const mouseScrollLines = 3

func (m Model) handleMouseWheel(msg tea.MouseWheelMsg) (tea.Model, tea.Cmd) {
	var delta int
	switch msg.Button {
	case tea.MouseWheelUp:
		delta = -mouseScrollLines
	case tea.MouseWheelDown:
		delta = mouseScrollLines
	default:
		return m, nil
	}

	m.prepareNavigation(false)
	total := m.TotalLines()
	if total == 0 {
		return m, nil
	}

	// In Reading mode, jump by visible lines instead of source lines.
	if m.previewMode == PreviewReading && len(m.readingNav.visibleLines) > 0 {
		target := m.cursorLine
		steps := delta
		if steps < 0 {
			steps = -steps
		}
		for i := 0; i < steps; i++ {
			if delta > 0 {
				next := m.readingNav.nextVisible(target)
				if next > target {
					target = next
				}
			} else {
				prev := m.readingNav.prevVisible(target)
				if prev < target {
					target = prev
				}
			}
		}
		if target != m.cursorLine {
			m.saveCurrentLineAndMoveTo(target)
		}
		return m, nil
	}

	targetLine := m.cursorLine + delta
	targetLine = max(0, min(targetLine, total-1))
	if targetLine != m.cursorLine {
		m.saveCurrentLineAndMoveTo(targetLine)
	}
	return m, nil
}

// ========================================
// Page and boundary navigation
// ========================================

func (m Model) handlePageUpKey() (tea.Model, tea.Cmd) {
	m.prepareNavigation(false)
	m.moveCursor(-(m.height - 4), 0)
	return m, nil
}

func (m Model) handlePageDownKey() (tea.Model, tea.Cmd) {
	m.prepareNavigation(false)
	m.moveCursor(m.height-4, 0)
	return m, nil
}

func (m Model) handleHomeKey() (tea.Model, tea.Cmd) {
	m.prepareNavigation(false)
	m.cursorCol = 0
	return m, nil
}

func (m Model) handleEndKey() (tea.Model, tea.Cmd) {
	m.prepareNavigation(false)
	m.cursorCol = runeLen(m.editBuf)
	return m, nil
}

// handleCtrlHomeKey moves cursor to document start (line 0, column 0).
func (m Model) handleCtrlHomeKey() (tea.Model, tea.Cmd) {
	m.prepareNavigation(false)
	m.saveCurrentLineAndMoveTo(0)
	m.cursorCol = 0
	return m, nil
}

// handleCtrlEndKey moves cursor to document end (last line, end of line).
func (m Model) handleCtrlEndKey() (tea.Model, tea.Cmd) {
	m.prepareNavigation(false)
	lastLine := max(m.TotalLines()-1, 0)
	m.saveCurrentLineAndMoveTo(lastLine)
	m.cursorCol = runeLen(m.editBuf)
	return m, nil
}

// ========================================
// Word navigation (Ctrl+Arrow / Alt+B/F)
// ========================================

// wordBoundaryLeft finds the previous word boundary from col in runes.
// Word boundaries are determined by unicode.IsSpace and unicode.IsPunct.
func wordBoundaryLeft(runes []rune, col int) int {
	col = min(col, len(runes))

	startCol := col

	// Skip whitespace backwards
	for col > 0 && unicode.IsSpace(runes[col-1]) {
		col--
	}

	// Skip word characters backwards (non-space, non-punct)
	for col > 0 && !unicode.IsSpace(runes[col-1]) && !unicode.IsPunct(runes[col-1]) {
		col--
	}

	// If we only skipped punctuation, skip it too
	if col == startCol || col == len(runes) {
		for col > 0 && unicode.IsPunct(runes[col-1]) {
			col--
		}
	}

	return col
}

// wordBoundaryRight finds the next word boundary from col in runes.
// Word boundaries are determined by unicode.IsSpace and unicode.IsPunct.
func wordBoundaryRight(runes []rune, col int) int {
	lineLen := len(runes)
	if col > lineLen {
		col = lineLen
	}

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

	return col
}

// handleCtrlLeftKey moves cursor left to previous word boundary.
// If at column 0, wraps to end of previous line first (like handleLeftKey).
func (m Model) handleCtrlLeftKey() (tea.Model, tea.Cmd) {
	m.prepareNavigation(false)

	// If at start of line, move to end of previous line first
	if m.cursorCol == 0 {
		if m.cursorLine > 0 {
			m.saveCurrentLineAndMoveTo(m.cursorLine - 1)
			m.cursorCol = runeLen(m.editBuf)
		}
		return m, nil
	}

	m.cursorCol = wordBoundaryLeft([]rune(m.editBuf), m.cursorCol)
	return m, nil
}

// handleCtrlRightKey moves cursor right to next word boundary.
// If at end of line, wraps to start of next line first (like handleRightKey).
func (m Model) handleCtrlRightKey() (tea.Model, tea.Cmd) {
	m.prepareNavigation(false)

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

	m.cursorCol = wordBoundaryRight(runes, m.cursorCol)
	return m, nil
}

// ========================================
// Shift+Arrow selection navigation
// ========================================
// These handlers extend the selection while moving the cursor.
// They mirror the normal navigation handlers but call ensureSelectionAnchor()
// instead of ClearSelection().

func (m Model) handleShiftUpKey() (tea.Model, tea.Cmd) {
	m.prepareNavigation(true)
	if m.cursorLine > 0 {
		m.saveCurrentLineAndMoveTo(m.cursorLine - 1)
	}
	return m, nil
}

func (m Model) handleShiftDownKey() (tea.Model, tea.Cmd) {
	m.prepareNavigation(true)
	if m.cursorLine < m.TotalLines()-1 {
		m.saveCurrentLineAndMoveTo(m.cursorLine + 1)
	}
	return m, nil
}

func (m Model) handleShiftLeftKey() (tea.Model, tea.Cmd) {
	m.prepareNavigation(true)
	if m.cursorCol > 0 {
		m.cursorCol--
	} else if m.cursorLine > 0 {
		m.saveCurrentLineAndMoveTo(m.cursorLine - 1)
		m.cursorCol = runeLen(m.editBuf)
	}
	return m, nil
}

func (m Model) handleShiftRightKey() (tea.Model, tea.Cmd) {
	m.prepareNavigation(true)
	if m.cursorCol < runeLen(m.editBuf) {
		m.cursorCol++
	} else if m.cursorLine < m.TotalLines()-1 {
		m.saveCurrentLineAndMoveTo(m.cursorLine + 1)
		m.cursorCol = 0
	}
	return m, nil
}

func (m Model) handleShiftHomeKey() (tea.Model, tea.Cmd) {
	m.prepareNavigation(true)
	m.cursorCol = 0
	return m, nil
}

func (m Model) handleShiftEndKey() (tea.Model, tea.Cmd) {
	m.prepareNavigation(true)
	m.cursorCol = runeLen(m.editBuf)
	return m, nil
}

func (m Model) handleShiftPageUpKey() (tea.Model, tea.Cmd) {
	m.prepareNavigation(true)
	m.moveCursor(-(m.height - 4), 0)
	return m, nil
}

func (m Model) handleShiftPageDownKey() (tea.Model, tea.Cmd) {
	m.prepareNavigation(true)
	m.moveCursor(m.height-4, 0)
	return m, nil
}

func (m Model) handleShiftCtrlHomeKey() (tea.Model, tea.Cmd) {
	m.prepareNavigation(true)
	m.saveCurrentLineAndMoveTo(0)
	m.cursorCol = 0
	return m, nil
}

func (m Model) handleShiftCtrlEndKey() (tea.Model, tea.Cmd) {
	m.prepareNavigation(true)
	lastLine := max(m.TotalLines()-1, 0)
	m.saveCurrentLineAndMoveTo(lastLine)
	m.cursorCol = runeLen(m.editBuf)
	return m, nil
}

func (m Model) handleShiftCtrlLeftKey() (tea.Model, tea.Cmd) {
	m.prepareNavigation(true)

	if m.cursorCol == 0 {
		if m.cursorLine > 0 {
			m.saveCurrentLineAndMoveTo(m.cursorLine - 1)
			m.cursorCol = runeLen(m.editBuf)
		}
		return m, nil
	}

	m.cursorCol = wordBoundaryLeft([]rune(m.editBuf), m.cursorCol)
	return m, nil
}

func (m Model) handleShiftCtrlRightKey() (tea.Model, tea.Cmd) {
	m.prepareNavigation(true)

	runes := []rune(m.editBuf)
	lineLen := len(runes)

	if m.cursorCol > lineLen {
		m.cursorCol = lineLen
	}

	if m.cursorCol >= lineLen {
		if m.cursorLine < m.TotalLines()-1 {
			m.saveCurrentLineAndMoveTo(m.cursorLine + 1)
			m.cursorCol = 0
		}
		return m, nil
	}

	m.cursorCol = wordBoundaryRight(runes, m.cursorCol)
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

	// In Reading mode, snap to nearest visible line after page jump.
	if m.previewMode == PreviewReading && len(m.readingNav.visibleLines) > 0 {
		m.cursorLine = m.readingNav.nearestVisible(m.cursorLine)
	}

	// Move column
	lines := m.GetLines()
	if m.cursorLine < len(lines) {
		lineLen := runeLen(lines[m.cursorLine])
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
// Uses the same dynamic footer height formula as View() to ensure the scroll
// system and renderer agree on available space. Returns contentHeight (which
// includes the header row) since the scroll system counts source lines only.
//
// Performance: calls GetLineResults() + contextFooterHeight() to compute the
// exact same value as View(). GetLineResults iterates blocks but does not
// re-evaluate — this is O(lines), not O(eval).
func (m *Model) getVisibleHeight() int {
	results := m.GetLineResults()
	footerHeight := m.contextFooterHeight(results)
	// Identical to View(): contentHeight = max(totalHeight - StatusBarHeight - footerHeight - 2, 5)
	return max(m.height-components.StatusBarHeight-footerHeight-2, 5)
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
