package editor

import (
	"strings"

	"github.com/CalcMark/go-calcmark/cmd/calcmark/config/theme"
	"github.com/CalcMark/go-calcmark/cmd/calcmark/tui/geometry"
	"github.com/charmbracelet/lipgloss"
)

// renderLineWithSelection applies selection highlighting to a line if needed.
// Returns the line with selection styling applied.
// Uses UTF-8 safe rune operations for column positions.
func (m Model) renderLineWithSelection(lineNum int, lineText string) string {
	if !m.HasSelection() {
		return lineText
	}

	startLine, startCol, endLine, endCol := m.GetSelectionRange()

	// Line is outside selection range
	if lineNum < startLine || lineNum > endLine {
		return lineText
	}

	// Selection style
	selectionStyle := lipgloss.NewStyle().
		Background(theme.Selection).
		Foreground(theme.SelectionFg)

	runes := []rune(lineText)
	lineLen := len(runes)

	// Determine selection bounds for this line
	var selectStart, selectEnd int

	if lineNum == startLine && lineNum == endLine {
		// Selection is within this line only
		selectStart = startCol
		selectEnd = endCol
	} else if lineNum == startLine {
		// First line of multi-line selection
		selectStart = startCol
		selectEnd = lineLen
	} else if lineNum == endLine {
		// Last line of multi-line selection
		selectStart = 0
		selectEnd = endCol
	} else {
		// Middle line - select entire line
		selectStart = 0
		selectEnd = lineLen
	}

	// Clamp to valid range
	if selectStart < 0 {
		selectStart = 0
	}
	if selectStart > lineLen {
		selectStart = lineLen
	}
	if selectEnd < 0 {
		selectEnd = 0
	}
	if selectEnd > lineLen {
		selectEnd = lineLen
	}

	// Nothing to select on this line
	if selectStart >= selectEnd {
		return lineText
	}

	// Build the styled line: before + selected + after
	var result strings.Builder

	// Part before selection
	if selectStart > 0 {
		result.WriteString(string(runes[:selectStart]))
	}

	// Selected part with highlighting
	selectedText := string(runes[selectStart:selectEnd])
	result.WriteString(selectionStyle.Render(selectedText))

	// Part after selection
	if selectEnd < lineLen {
		result.WriteString(string(runes[selectEnd:]))
	}

	return result.String()
}

// renderLineWithCursor renders a line with cursor at the specified column.
// If content is empty, uses m.GetCurrentLineText() to get the actual line content.
// This ensures the cursor is ALWAYS visible, even when not actively typing.
func (m Model) renderLineWithCursor(content string, col int, width int, useEditStyle bool) string {
	// Determine which style to use (includes foreground and background)
	var lineStyle lipgloss.Style
	if useEditStyle {
		// When typing (editBuf active)
		lineStyle = m.styles.EditLine.Inline(true)
	} else {
		// When not typing (showing document content with cursor)
		lineStyle = m.styles.CurrentLine.Inline(true)
	}

	// CRITICAL INSIGHT: The issue is that concatenating multiple .Render() calls
	// creates separate ANSI blocks. We need ONE continuous background.
	// Solution: Build the ENTIRE content first, THEN render once with padding included

	// Use runes for proper UTF-8 handling - col is a rune position
	runes := []rune(content)
	contentLen := len(runes)

	// Clamp col to valid range to prevent panic
	if col < 0 {
		col = 0
	}
	if col > contentLen {
		col = contentLen
	}

	// Determine cursor character
	var cursorChar string
	if col >= contentLen {
		cursorChar = " "
	} else {
		cursorChar = string(runes[col])
	}

	// Calculate total padding needed (use rune length for display width approximation)
	totalPadding := width - contentLen
	if col >= contentLen {
		totalPadding -= 1 // Account for cursor space
	}

	// Build result by rendering segments with inline styles
	var result strings.Builder

	// Before cursor
	if col > 0 {
		result.WriteString(lineStyle.Render(string(runes[:col])))
	}

	// Cursor
	result.WriteString(m.styles.Cursor.Inline(true).Render(cursorChar))

	// After cursor
	if col+1 < contentLen {
		result.WriteString(lineStyle.Render(string(runes[col+1:])))
	}

	// Padding - CRITICAL FIX: lipgloss strips background from trailing spaces!
	// We need to add the padding as part of the CONTENT, not trailing
	// Solution: Add padding BEFORE the cursor line ends, by using Width() on the style
	if totalPadding > 0 {
		bgColor := lineStyle.GetBackground()
		paddingStyle := lipgloss.NewStyle().Background(bgColor).Width(totalPadding)
		result.WriteString(paddingStyle.Render(""))
	}

	return result.String()
}

// renderEditLine renders the line being edited with cursor (single line, no wrapping).
func (m Model) renderEditLine(width int) string {
	return m.renderLineWithCursor(m.editBuf, m.cursorCol, width, true)
}

// renderEditLineWrapped renders the edit buffer with wrapping support.
// Returns multiple lines if the content exceeds width.
func (m Model) renderEditLineWrapped(width int) []string {
	if len(m.editBuf) <= width {
		// Fits on one line
		return []string{m.renderEditLine(width)}
	}

	// Wrap the edit buffer content
	wrappedContent := geometry.WrapText(m.editBuf, width)
	var result []string

	// Track which wrapped line contains the cursor
	charsSoFar := 0
	cursorLineIdx := 0
	cursorColInLine := m.cursorCol

	for i, seg := range wrappedContent {
		if m.cursorCol >= charsSoFar && m.cursorCol < charsSoFar+len(seg) {
			cursorLineIdx = i
			cursorColInLine = m.cursorCol - charsSoFar
			break
		}
		charsSoFar += len(seg)
		// Handle cursor at very end
		if i == len(wrappedContent)-1 && m.cursorCol >= charsSoFar {
			cursorLineIdx = i
			cursorColInLine = m.cursorCol - charsSoFar + len(seg)
		}
	}

	lineStyle := m.styles.EditLine

	for i, seg := range wrappedContent {
		var s strings.Builder

		if i == cursorLineIdx {
			// This line has the cursor
			segLen := len(seg)

			// Determine cursor character
			var cursorChar string
			if cursorColInLine >= segLen {
				cursorChar = " "
			} else {
				cursorChar = string(seg[cursorColInLine])
			}

			// Before cursor
			if cursorColInLine > 0 {
				s.WriteString(lineStyle.Render(seg[:cursorColInLine]))
			}

			// Cursor
			s.WriteString(m.styles.Cursor.Render(cursorChar))

			// After cursor
			if cursorColInLine+1 < segLen {
				s.WriteString(lineStyle.Render(seg[cursorColInLine+1:]))
			}

			// Calculate padding
			currentWidth := lipgloss.Width(s.String())
			padding := width - currentWidth
			if padding > 0 {
				s.WriteString(lineStyle.Render(strings.Repeat(" ", padding)))
			}

			result = append(result, s.String())
		} else {
			// No cursor on this line
			s.WriteString(lineStyle.Render(seg))

			// Padding
			currentWidth := lipgloss.Width(s.String())
			padding := width - currentWidth
			if padding > 0 {
				s.WriteString(lineStyle.Render(strings.Repeat(" ", padding)))
			}

			result = append(result, s.String())
		}
	}

	return result
}
