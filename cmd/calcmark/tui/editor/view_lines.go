package editor

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/CalcMark/go-calcmark/cmd/calcmark/config/theme"
	"github.com/CalcMark/go-calcmark/cmd/calcmark/tui/geometry"
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

	// Selection style - use theme colors
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
	// Apply selection highlighting first if needed
	if m.HasSelection() && m.cursorLine >= 0 {
		content = m.renderLineWithSelection(m.cursorLine, content)
	}

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

	// After cursor (including padding to fill width)
	afterCursorStart := col + 1
	if afterCursorStart <= contentLen {
		result.WriteString(lineStyle.Render(string(runes[afterCursorStart:])))
	}

	// Add padding to fill the width
	if totalPadding > 0 {
		result.WriteString(lineStyle.Render(strings.Repeat(" ", totalPadding)))
	}

	return result.String()
}

// renderEditLineWrapped handles rendering the current editing line when it wraps
// across multiple visual lines. Returns an array of rendered lines.
func (m Model) renderEditLineWrapped(width int) []string {
	content := []rune(m.editBuf)
	wrappedContent := geometry.WrapText(string(content), width)

	// Find which wrapped line contains the cursor
	cursorLineIdx := 0
	cursorColInLine := m.cursorCol

	totalCol := 0
	for i, seg := range wrappedContent {
		segLen := len([]rune(seg))
		if m.cursorCol >= totalCol && m.cursorCol < totalCol+segLen {
			cursorLineIdx = i
			cursorColInLine = m.cursorCol - totalCol
			break
		}
		totalCol += segLen
		if m.cursorCol == totalCol && i == len(wrappedContent)-1 {
			// Cursor at the very end
			cursorLineIdx = i
			cursorColInLine = segLen
		}
	}

	// Build the result
	result := []string{}

	// Use inline styles to maintain consistent background
	if width <= 0 {
		return result
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
