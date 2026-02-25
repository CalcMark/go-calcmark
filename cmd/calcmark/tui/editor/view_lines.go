package editor

import (
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/CalcMark/go-calcmark/cmd/calcmark/config/theme"
	"github.com/CalcMark/go-calcmark/cmd/calcmark/tui/geometry"
)

// renderLineWithSelection applies selection highlighting to a non-cursor line.
// IMPORTANT: rawText must be plain text WITHOUT ANSI codes. Column positions
// are document-column (rune) indices that only work on un-styled text.
// Non-selected text is rendered with tintFg/tintBg; selected text uses theme.Selection.
func (m Model) renderLineWithSelection(lineNum int, rawText string, tintFg, tintBg color.Color) string {
	tintStyle := lipgloss.NewStyle().Foreground(tintFg).Background(tintBg)

	if !m.HasSelection() {
		return tintStyle.Render(rawText)
	}

	startLine, startCol, endLine, endCol := m.GetSelectionRange()

	// Line is outside selection range
	if lineNum < startLine || lineNum > endLine {
		return tintStyle.Render(rawText)
	}

	// Selection style - use theme colors
	selectionStyle := lipgloss.NewStyle().
		Background(theme.Selection).
		Foreground(theme.SelectionFg)

	runes := []rune(rawText)
	lineLen := len(runes)

	// Determine selection bounds for this line
	var selectStart, selectEnd int

	if lineNum == startLine && lineNum == endLine {
		selectStart = startCol
		selectEnd = endCol
	} else if lineNum == startLine {
		selectStart = startCol
		selectEnd = lineLen
	} else if lineNum == endLine {
		selectStart = 0
		selectEnd = endCol
	} else {
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
		return tintStyle.Render(rawText)
	}

	// Build the styled line: before + selected + after
	var result strings.Builder

	// Part before selection (tinted)
	if selectStart > 0 {
		result.WriteString(tintStyle.Render(string(runes[:selectStart])))
	}

	// Selected part with highlighting
	result.WriteString(selectionStyle.Render(string(runes[selectStart:selectEnd])))

	// Part after selection (tinted)
	if selectEnd < lineLen {
		result.WriteString(tintStyle.Render(string(runes[selectEnd:])))
	}

	return result.String()
}

// renderLineWithCursor renders a line with cursor at the specified column.
// content must be plain text WITHOUT ANSI codes (rune-based column arithmetic).
// When a selection is active, interleaves cursor + selection + normal styles
// character-by-character so that rune positions are never corrupted by ANSI codes.
func (m Model) renderLineWithCursor(content string, col int, width int, useEditStyle bool) string {
	// Determine which style to use (includes foreground and background)
	var lineStyle lipgloss.Style
	if useEditStyle {
		lineStyle = m.styles.EditLine.Inline(true)
	} else {
		lineStyle = m.styles.CurrentLine.Inline(true)
	}

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

	// Check for active selection on the cursor line
	hasSelection := false
	var selStart, selEnd int
	var selectionStyle lipgloss.Style

	if m.HasSelection() && m.cursorLine >= 0 {
		sLine, sCol, eLine, eCol := m.GetSelectionRange()
		if sLine <= m.cursorLine && eLine >= m.cursorLine {
			hasSelection = true
			selectionStyle = lipgloss.NewStyle().
				Background(theme.Selection).
				Foreground(theme.SelectionFg)

			if sLine == m.cursorLine {
				selStart = sCol
			}
			if eLine == m.cursorLine {
				selEnd = eCol
			} else {
				selEnd = contentLen
			}
			// Clamp
			if selStart < 0 {
				selStart = 0
			}
			if selEnd > contentLen {
				selEnd = contentLen
			}
		}
	}

	if hasSelection && selStart < selEnd {
		// Character-by-character rendering to interleave cursor + selection + normal
		var result strings.Builder
		for i, r := range runes {
			ch := string(r)
			if i == col {
				result.WriteString(m.styles.Cursor.Inline(true).Render(ch))
			} else if i >= selStart && i < selEnd {
				result.WriteString(selectionStyle.Render(ch))
			} else {
				result.WriteString(lineStyle.Render(ch))
			}
		}
		// Cursor beyond content
		if col >= contentLen {
			result.WriteString(m.styles.Cursor.Inline(true).Render(" "))
		}
		// Pad to width
		totalPadding := width - contentLen
		if col >= contentLen {
			totalPadding--
		}
		if totalPadding > 0 {
			result.WriteString(lineStyle.Render(strings.Repeat(" ", totalPadding)))
		}
		return result.String()
	}

	// No selection — batch render (original fast path)
	var cursorChar string
	if col >= contentLen {
		cursorChar = " "
	} else {
		cursorChar = string(runes[col])
	}

	totalPadding := width - contentLen
	if col >= contentLen {
		totalPadding--
	}

	var result strings.Builder

	if col > 0 {
		result.WriteString(lineStyle.Render(string(runes[:col])))
	}
	result.WriteString(m.styles.Cursor.Inline(true).Render(cursorChar))
	if col+1 <= contentLen {
		result.WriteString(lineStyle.Render(string(runes[col+1:])))
	}
	if totalPadding > 0 {
		result.WriteString(lineStyle.Render(strings.Repeat(" ", totalPadding)))
	}

	return result.String()
}

// editLineSelectionRange returns the selection column range on the edit line
// (m.cursorLine). Returns (-1, -1) if no selection overlaps the cursor line.
// The range [selectStart, selectEnd) is in source-line column (rune) coordinates.
func (m Model) editLineSelectionRange() (selectStart, selectEnd int) {
	if !m.HasSelection() {
		return -1, -1
	}
	sLine, sCol, eLine, eCol := m.GetSelectionRange()
	if sLine > m.cursorLine || eLine < m.cursorLine {
		return -1, -1
	}
	// Selection includes cursor line — determine column bounds
	if sLine == m.cursorLine {
		selectStart = sCol
	} else {
		selectStart = 0
	}
	if eLine == m.cursorLine {
		selectEnd = eCol
	} else {
		selectEnd = len([]rune(m.editBuf))
	}
	return selectStart, selectEnd
}

// renderEditLineWrapped handles rendering the current editing line when it wraps
// across multiple visual lines. Returns an array of rendered lines.
// Applies selection highlighting when an active selection overlaps the cursor line.
func (m Model) renderEditLineWrapped(width int) []string {
	wrappedContent := geometry.WrapText(m.editBuf, width)

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

	// Precompute selection range on this line (in source rune coordinates).
	// When no selection overlaps the cursor line, use the fast batch-render path.
	selStart, selEnd := m.editLineSelectionRange()
	hasSelectionOnLine := selStart >= 0

	var selectionStyle lipgloss.Style
	if hasSelectionOnLine {
		selectionStyle = lipgloss.NewStyle().
			Background(theme.Selection).
			Foreground(theme.SelectionFg)
	}

	// Track the source column offset as we iterate through wrapped segments
	segOffset := 0
	for i, seg := range wrappedContent {
		segRunes := []rune(seg)
		segLen := len(segRunes)

		var s strings.Builder

		if i == cursorLineIdx && hasSelectionOnLine {
			// Cursor line WITH selection — character-by-character rendering
			// needed to interleave cursor, selection, and normal styles.

			if cursorColInLine < 0 {
				cursorColInLine = 0
			}
			if cursorColInLine > segLen {
				cursorColInLine = segLen
			}

			for j := range segLen {
				srcCol := segOffset + j
				ch := string(segRunes[j])

				if j == cursorColInLine {
					s.WriteString(m.styles.Cursor.Render(ch))
				} else if srcCol >= selStart && srcCol < selEnd {
					s.WriteString(selectionStyle.Render(ch))
				} else {
					s.WriteString(lineStyle.Render(ch))
				}
			}

			if cursorColInLine >= segLen {
				s.WriteString(m.styles.Cursor.Render(" "))
			}

		} else if i == cursorLineIdx {
			// Cursor line WITHOUT selection — batch-render (original fast path)

			if cursorColInLine < 0 {
				cursorColInLine = 0
			}
			if cursorColInLine > segLen {
				cursorColInLine = segLen
			}

			var cursorChar string
			if cursorColInLine >= segLen {
				cursorChar = " "
			} else {
				cursorChar = string(segRunes[cursorColInLine])
			}

			if cursorColInLine > 0 {
				s.WriteString(lineStyle.Render(string(segRunes[:cursorColInLine])))
			}
			s.WriteString(m.styles.Cursor.Render(cursorChar))
			if cursorColInLine+1 < segLen {
				s.WriteString(lineStyle.Render(string(segRunes[cursorColInLine+1:])))
			}

		} else if hasSelectionOnLine {
			// Non-cursor segment WITH selection — character-by-character rendering

			for j := range segLen {
				srcCol := segOffset + j
				ch := string(segRunes[j])

				if srcCol >= selStart && srcCol < selEnd {
					s.WriteString(selectionStyle.Render(ch))
				} else {
					s.WriteString(lineStyle.Render(ch))
				}
			}

		} else {
			// Non-cursor segment WITHOUT selection — batch-render
			s.WriteString(lineStyle.Render(seg))
		}

		// Pad to full width
		currentWidth := lipgloss.Width(s.String())
		padding := width - currentWidth
		if padding > 0 {
			s.WriteString(lineStyle.Render(strings.Repeat(" ", padding)))
		}

		result = append(result, s.String())
		segOffset += segLen
	}

	return result
}
