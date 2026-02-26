package editor

import (
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/CalcMark/go-calcmark/cmd/calcmark/config/theme"
	"github.com/CalcMark/go-calcmark/cmd/calcmark/tui/geometry"
)

// selectionStyle returns the lipgloss style for selected text.
func selectionStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Background(theme.Selection).
		Foreground(theme.SelectionFg)
}

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

	selStyle := selectionStyle()

	runes := []rune(rawText)
	lineLen := len(runes)

	// Determine selection bounds for this line
	var selStart, selEnd int

	if lineNum == startLine && lineNum == endLine {
		selStart = startCol
		selEnd = endCol
	} else if lineNum == startLine {
		selStart = startCol
		selEnd = lineLen
	} else if lineNum == endLine {
		selStart = 0
		selEnd = endCol
	} else {
		selStart = 0
		selEnd = lineLen
	}

	// Clamp to valid range
	if selStart < 0 {
		selStart = 0
	}
	if selStart > lineLen {
		selStart = lineLen
	}
	if selEnd < 0 {
		selEnd = 0
	}
	if selEnd > lineLen {
		selEnd = lineLen
	}

	// Nothing to select on this line
	if selStart >= selEnd {
		return tintStyle.Render(rawText)
	}

	// Build the styled line: before + selected + after
	var result strings.Builder

	// Part before selection (tinted)
	if selStart > 0 {
		result.WriteString(tintStyle.Render(string(runes[:selStart])))
	}

	// Selected part with highlighting
	result.WriteString(selStyle.Render(string(runes[selStart:selEnd])))

	// Part after selection (tinted)
	if selEnd < lineLen {
		result.WriteString(tintStyle.Render(string(runes[selEnd:])))
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
	selStart, selEnd := m.editLineSelectionRange()
	// Clamp to content bounds
	if selStart < 0 {
		selStart = 0
	}
	if selEnd > contentLen {
		selEnd = contentLen
	}

	if selStart >= 0 && selStart < selEnd {
		// Batch-render contiguous style segments instead of character-by-character.
		// Content is split into up to 5 segments based on cursor and selection boundaries.
		selStyle := selectionStyle()
		cursorStyle := m.styles.Cursor.Inline(true)

		var result strings.Builder

		// Helper to render a rune slice with a given style (no-op for empty slices)
		renderSlice := func(start, end int, style lipgloss.Style) {
			if start < end && start < contentLen {
				end = min(end, contentLen)
				result.WriteString(style.Render(string(runes[start:end])))
			}
		}

		if col < selStart {
			// Cursor before selection: normal | cursor | normal | selection | normal
			renderSlice(0, col, lineStyle)
			if col < contentLen {
				result.WriteString(cursorStyle.Render(string(runes[col])))
			}
			renderSlice(col+1, selStart, lineStyle)
			renderSlice(selStart, selEnd, selStyle)
			renderSlice(selEnd, contentLen, lineStyle)
		} else if col < selEnd {
			// Cursor inside selection: normal | selection | cursor | selection | normal
			renderSlice(0, selStart, lineStyle)
			renderSlice(selStart, col, selStyle)
			if col < contentLen {
				result.WriteString(cursorStyle.Render(string(runes[col])))
			}
			renderSlice(col+1, selEnd, selStyle)
			renderSlice(selEnd, contentLen, lineStyle)
		} else {
			// Cursor after selection: normal | selection | normal | cursor | normal
			renderSlice(0, selStart, lineStyle)
			renderSlice(selStart, selEnd, selStyle)
			renderSlice(selEnd, col, lineStyle)
			if col < contentLen {
				result.WriteString(cursorStyle.Render(string(runes[col])))
			}
			renderSlice(col+1, contentLen, lineStyle)
		}

		// Cursor beyond content
		if col >= contentLen {
			result.WriteString(cursorStyle.Render(" "))
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

	var selStyle lipgloss.Style
	if hasSelectionOnLine {
		selStyle = selectionStyle()
	}

	// Track the source column offset as we iterate through wrapped segments
	segOffset := 0
	for i, seg := range wrappedContent {
		segRunes := []rune(seg)
		segLen := len(segRunes)

		var s strings.Builder

		if i == cursorLineIdx && hasSelectionOnLine {
			// Cursor line WITH selection — batch-render contiguous segments.

			if cursorColInLine < 0 {
				cursorColInLine = 0
			}
			if cursorColInLine > segLen {
				cursorColInLine = segLen
			}

			// Map source-column selection bounds to segment-local coordinates
			localSelStart := max(selStart-segOffset, 0)
			localSelEnd := min(selEnd-segOffset, segLen)

			// Helper to render a segment-local rune slice
			renderSeg := func(start, end int, style lipgloss.Style) {
				if start < end {
					s.WriteString(style.Render(string(segRunes[start:end])))
				}
			}

			if localSelStart >= localSelEnd {
				// No selection in this segment — just cursor + normal
				renderSeg(0, cursorColInLine, lineStyle)
				if cursorColInLine < segLen {
					s.WriteString(m.styles.Cursor.Render(string(segRunes[cursorColInLine])))
				}
				renderSeg(cursorColInLine+1, segLen, lineStyle)
			} else if cursorColInLine < localSelStart {
				renderSeg(0, cursorColInLine, lineStyle)
				if cursorColInLine < segLen {
					s.WriteString(m.styles.Cursor.Render(string(segRunes[cursorColInLine])))
				}
				renderSeg(cursorColInLine+1, localSelStart, lineStyle)
				renderSeg(localSelStart, localSelEnd, selStyle)
				renderSeg(localSelEnd, segLen, lineStyle)
			} else if cursorColInLine < localSelEnd {
				renderSeg(0, localSelStart, lineStyle)
				renderSeg(localSelStart, cursorColInLine, selStyle)
				if cursorColInLine < segLen {
					s.WriteString(m.styles.Cursor.Render(string(segRunes[cursorColInLine])))
				}
				renderSeg(cursorColInLine+1, localSelEnd, selStyle)
				renderSeg(localSelEnd, segLen, lineStyle)
			} else {
				renderSeg(0, localSelStart, lineStyle)
				renderSeg(localSelStart, localSelEnd, selStyle)
				renderSeg(localSelEnd, cursorColInLine, lineStyle)
				if cursorColInLine < segLen {
					s.WriteString(m.styles.Cursor.Render(string(segRunes[cursorColInLine])))
				}
				renderSeg(cursorColInLine+1, segLen, lineStyle)
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
			// Non-cursor segment WITH selection — batch-render up to 3 segments

			localSelStart := max(selStart-segOffset, 0)
			localSelEnd := min(selEnd-segOffset, segLen)

			if localSelStart >= localSelEnd {
				// No selection in this segment
				s.WriteString(lineStyle.Render(seg))
			} else {
				if localSelStart > 0 {
					s.WriteString(lineStyle.Render(string(segRunes[:localSelStart])))
				}
				s.WriteString(selStyle.Render(string(segRunes[localSelStart:localSelEnd])))
				if localSelEnd < segLen {
					s.WriteString(lineStyle.Render(string(segRunes[localSelEnd:])))
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
