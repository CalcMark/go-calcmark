package editor

import (
	"fmt"
	"strings"

	"github.com/CalcMark/go-calcmark/cmd/calcmark/tui/components"
	"github.com/CalcMark/go-calcmark/cmd/calcmark/tui/geometry"
	"github.com/CalcMark/go-calcmark/spec/document"
	"github.com/charmbracelet/lipgloss"
)

// renderSourcePaneAligned renders the source pane using pre-computed aligned lines.
// This avoids recomputing alignment which could cause cycles.
func (m Model) renderSourcePaneAligned(width, height int, aligned alignedPanes) string {
	sourceLines := aligned.sourceLines
	visibleLines := height

	// Convert cursor's source line to visual line index for proper scrolling
	cursorVisualLine := 0
	if visualIdx, ok := aligned.sourceToVisual[m.cursorLine]; ok {
		cursorVisualLine = visualIdx
	}

	// Convert m.scrollOffset from source-line space to visual-line space
	// m.scrollOffset is stored as a source line index, but we need visual line index
	visualScrollOffset := 0
	if visualIdx, ok := aligned.sourceToVisual[m.scrollOffset]; ok {
		visualScrollOffset = visualIdx
	}

	// Ensure cursor is visible by adjusting scroll based on visual position
	if cursorVisualLine < visualScrollOffset {
		visualScrollOffset = cursorVisualLine
	}
	if cursorVisualLine >= visualScrollOffset+visibleLines {
		visualScrollOffset = cursorVisualLine - visibleLines + 1
	}

	// Calculate visible range
	start := visualScrollOffset
	end := min(start+visibleLines, len(sourceLines))

	lineNumWidth := 4
	contentWidth := width - lineNumWidth - 2

	// Build complete lines first, then join them
	// This ensures NO bare newlines that could show terminal default color
	var renderedLines []string

	linesWritten := 0
	for i := start; i < end && linesWritten < visibleLines; i++ {
		if i >= len(sourceLines) {
			break
		}
		sl := sourceLines[i]

		// In edit mode, skip pre-computed wrapped lines for the cursor line
		// since we'll render the edit buffer with its own wrapping
		if m.editBuf != "" && sl.isWrapped && sl.sourceLineIdx == m.cursorLine {
			continue
		}

		var lineNum string
		if sl.isPadding || sl.isWrapped {
			// Padding or wrapped continuation line - no line number
			lineNum = m.styles.LineNumber.
				Width(lineNumWidth).
				Render("")
		} else {
			// Regular line - show line number
			lineNum = m.styles.LineNumber.
				Width(lineNumWidth).
				Align(lipgloss.Right).
				Render(fmt.Sprintf("%d", sl.lineNum))
		}

		var content string
		if m.editBuf != "" && sl.isCursorLine {
			// Show edit buffer with cursor - handle wrapping
			editLines := m.renderEditLineWrapped(contentWidth)
			for j, editLine := range editLines {
				var completeLine string
				if j > 0 {
					// Continuation lines have no line number
					emptyLineNum := m.styles.LineNumber.Width(lineNumWidth).Render("")
					gutterStyle := lipgloss.NewStyle().Background(lipgloss.Color("236"))
					completeLine = emptyLineNum + gutterStyle.Render(" ") + editLine
				} else {
					gutterStyle := lipgloss.NewStyle().Background(lipgloss.Color("236"))
					completeLine = lineNum + gutterStyle.Render(" ") + editLine
				}
				// Ensure complete line is exactly width
				completeLine = ensureFullWidth(completeLine, width, lipgloss.Color("236"))
				renderedLines = append(renderedLines, completeLine)
				linesWritten++
			}
			continue
		} else if sl.isCursorLine {
			// Cursor line when NOT typing - show cursor at current column
			content = m.renderLineWithCursor(sl.content, m.cursorCol, contentWidth, false)
		} else if sl.isPadding {
			// Padding line - blank (for alignment with preview wrapping)
			content = padToWidth("", contentWidth, lipgloss.Color("236"))
		} else if sl.isWrapped {
			// Wrapped continuation line - apply selection highlighting then source text style
			lineWithSelection := m.renderLineWithSelection(sl.sourceLineIdx, sl.content)
			content = padToWidth(lineWithSelection, contentWidth, lipgloss.Color("236"))
		} else {
			// Normal source line - apply selection highlighting then source text style
			lineWithSelection := m.renderLineWithSelection(sl.lineNum-1, sl.content)
			content = padToWidth(lineWithSelection, contentWidth, lipgloss.Color("236"))
		}

		// Assemble complete line: lineNum + gutter + content
		gutterStyle := lipgloss.NewStyle().Background(lipgloss.Color("236"))
		completeLine := lineNum + gutterStyle.Render(" ") + content
		// Ensure complete line is exactly width
		completeLine = ensureFullWidth(completeLine, width, lipgloss.Color("236"))
		renderedLines = append(renderedLines, completeLine)
		linesWritten++
	}

	// Fill remaining space with tilde indicators
	for i := linesWritten; i < visibleLines; i++ {
		tildeLine := m.styles.LineNumber.Render("~")
		// Pad tilde line to full width
		tildeLine = ensureFullWidth(tildeLine, width, lipgloss.Color("236"))
		renderedLines = append(renderedLines, tildeLine)
	}

	// Join all lines - the newline separators are now between fully-styled lines
	// so terminal default cannot bleed through
	return strings.Join(renderedLines, "\n")
}

// sourcePaneBg returns the background color for the source pane.
func (m Model) sourcePaneBg() lipgloss.TerminalColor {
	color := m.styles.SourcePane.GetBackground()
	if _, ok := color.(lipgloss.NoColor); ok {
		return lipgloss.Color("236") // Fallback
	}
	return color
}

// previewPaneBg returns the background color for the preview pane.
func (m Model) previewPaneBg() lipgloss.TerminalColor {
	color := m.styles.PreviewPane.GetBackground()
	if _, ok := color.(lipgloss.NoColor); ok {
		return lipgloss.Color("236") // Fallback
	}
	return color
}

// renderPreviewPaneAligned renders the preview pane using pre-computed aligned lines.
// This avoids recomputing alignment which could cause cycles.
func (m Model) renderPreviewPaneAligned(width, height int, aligned alignedPanes) string {
	previewLines := aligned.previewLines

	// Globals panel at top (collapsible) - per spec lines 334-339
	globalsPanel := m.renderGlobalsPanel(width)
	globalsHeight := strings.Count(globalsPanel, "\n") + 1

	// Build complete lines to avoid bare newlines
	var allLines []string
	allLines = append(allLines, strings.Split(globalsPanel, "\n")...)

	// Separator after globals
	separatorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Background(lipgloss.Color("236"))
	separatorLine := separatorStyle.Render(strings.Repeat("─", width))
	// Ensure separator is exactly width
	separatorLine = ensureFullWidth(separatorLine, width, lipgloss.Color("236"))
	allLines = append(allLines, separatorLine)

	// Adjust height for globals panel and separator
	resultsHeight := height - globalsHeight - 1 // -1 for separator
	if resultsHeight < 1 {
		resultsHeight = 1
	}

	// Convert cursor's source line to visual line index for proper scrolling
	// Must use same scroll offset as source pane to keep them aligned
	cursorVisualLine := 0
	if visualIdx, ok := aligned.sourceToVisual[m.cursorLine]; ok {
		cursorVisualLine = visualIdx
	}

	// Convert m.scrollOffset from source-line space to visual-line space
	// m.scrollOffset is stored as a source line index, but we need visual line index
	visualScrollOffset := 0
	if visualIdx, ok := aligned.sourceToVisual[m.scrollOffset]; ok {
		visualScrollOffset = visualIdx
	}

	// Ensure cursor is visible by adjusting scroll based on visual position
	if cursorVisualLine < visualScrollOffset {
		visualScrollOffset = cursorVisualLine
	}
	if cursorVisualLine >= visualScrollOffset+resultsHeight {
		visualScrollOffset = cursorVisualLine - resultsHeight + 1
	}

	// Apply scroll offset and render visible lines
	// Note: wrapping is already done in computeAlignedPanes to ensure proper alignment
	start := visualScrollOffset
	end := min(start+resultsHeight, len(previewLines))

	// In edit mode, source pane may render different number of lines for cursor line
	// because it renders the live edit buffer. We need to:
	// 1. Count how many pre-computed lines exist for cursor's source line
	// 2. Count how many lines the edit buffer would render
	// 3. Adjust by skipping pre-computed wrapped lines or adding empty lines
	var editLineCount int
	var preComputedCursorLineCount int
	if m.editBuf != "" {
		// Count how many lines the edit buffer would produce
		contentWidth := width // approximate
		editLines := geometry.WrapText(m.editBuf, contentWidth)
		editLineCount = len(editLines)
		if editLineCount == 0 {
			editLineCount = 1
		}

		// Count how many pre-computed visual lines exist for cursor's source line
		for _, pl := range previewLines {
			if pl.sourceLineNum == m.cursorLine {
				preComputedCursorLineCount++
			}
		}
	}

	linesWritten := 0
	cursorLineProcessed := false
	for j := start; j < end && linesWritten < resultsHeight; j++ {
		if j >= len(previewLines) {
			break
		}
		pl := previewLines[j]

		// In edit mode, handle cursor line specially to match source pane's edit rendering
		if m.editBuf != "" && pl.sourceLineNum == m.cursorLine {
			if !cursorLineProcessed {
				// First occurrence of cursor line - output editLineCount lines
				// to match the source pane's edit buffer rendering.
				// Show the actual preview content (computed result) rather than blank.
				cursorPreviewLines := []previewLine{}
				for _, cpl := range previewLines {
					if cpl.sourceLineNum == m.cursorLine {
						cursorPreviewLines = append(cursorPreviewLines, cpl)
					}
				}
				for k := 0; k < editLineCount && linesWritten < resultsHeight; k++ {
					// Show preview content if available, otherwise empty
					var completeLine string
					if k < len(cursorPreviewLines) {
						completeLine = ensureFullWidth(cursorPreviewLines[k].content, width, lipgloss.Color("236"))
					} else {
						// Empty line with background
						completeLine = ensureFullWidth("", width, lipgloss.Color("236"))
					}
					allLines = append(allLines, completeLine)
					linesWritten++
				}
				cursorLineProcessed = true
			}
			// Skip all pre-computed lines for cursor (we've already output editLineCount lines)
			continue
		}

		paddedContent := ensureFullWidth(pl.content, width, lipgloss.Color("236"))
		allLines = append(allLines, paddedContent)
		linesWritten++
	}

	// Fill remaining space to maintain consistent height
	for i := linesWritten; i < resultsHeight; i++ {
		emptyLine := ensureFullWidth("", width, lipgloss.Color("236"))
		allLines = append(allLines, emptyLine)
	}

	// Join all lines - newlines between fully-styled lines prevent bleed-through
	return strings.Join(allLines, "\n")
}

// renderCalcLine renders a single calculation line result.
func (m Model) renderCalcLine(r LineResult, width int) string {
	// Use the detector to check if this line is actually a calculation
	detector := document.NewDetector()
	isActuallyCalc, _ := detector.IsCalculation(r.Source)

	if r.Error != "" && isActuallyCalc {
		// Show full error message - per CONTEXT.md decision
		// "Show full error message in preview (not abbreviated)"
		errStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("208")) // amber

		// Check if this is a blocked (cascading) error
		if r.IsBlocked {
			// Blocked errors show brief indicator - root cause shown elsewhere
			blockedStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("244")) // gray (less prominent)
			return blockedStyle.Render("⊘ blocked")
		}

		// Root cause error - show cleaned message without error code prefix
		// CleanErrorMessage removes prefixes like "undefined_variable: "
		errorText := components.CleanErrorMessage(r.Error)
		maxLen := width - 4 // room for "⚠ " prefix
		if maxLen > 0 && lipgloss.Width(errorText) > maxLen {
			// Truncate very long messages - full details in context footer
			errorText = components.TruncateWithEllipsis(errorText, maxLen)
		}
		return errStyle.Render("⚠ " + errorText)
	}

	if !isActuallyCalc {
		// Render as markdown even if in a CalcBlock
		mdRenderer, err := NewMarkdownRenderer(width)
		if err != nil {
			return r.Source
		}
		lines := mdRenderer.RenderLine(r.Source)
		if len(lines) > 0 {
			return lines[0] // Return first line; wrapping handled by caller
		}
		return ""
	}

	if r.Value == "" {
		return ""
	}

	// Use themed colors for calculation results
	valueStyle := m.styles.CalcValue

	// Changed indicator: asterisk in yellow for values that were recomputed
	changedMarker := ""
	if r.WasChanged {
		valueStyle = m.styles.Changed
		changedMarker = m.styles.Changed.Bold(true).Render("* ")
	}

	switch m.previewMode {
	case PreviewFull:
		// Full mode: "varName → value" for assignments, "→ value" for anonymous calcs
		if r.VarName != "" {
			return changedMarker + m.styles.CalcVarName.Render(r.VarName) + " " + m.styles.CalcArrow.Render("→") + " " + valueStyle.Render(r.Value)
		}
		// Anonymous calculation (no variable assignment) - show arrow without placeholder
		return changedMarker + m.styles.CalcArrow.Render("→") + " " + valueStyle.Render(r.Value)

	case PreviewMinimal:
		// Minimal mode: left-aligned "→ value" (with * if changed)
		arrow := "→ "
		return changedMarker + valueStyle.Render(arrow+r.Value)
	}

	return ""
}
