package editor

import (
	"fmt"
	"strings"

	"github.com/CalcMark/go-calcmark/cmd/calcmark/config/theme"
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

	srcBg := m.sourcePaneBg()

	// Compute frontmatter line count for block-level syntax highlighting.
	// Lines 0..fmCount-1 are frontmatter, then calc/markdown follows.
	fmCount := 0
	if fm := m.doc.GetFrontmatter(); fm != nil {
		serialized := fm.Serialize()
		if serialized != "" {
			fmCount = len(strings.Split(strings.TrimRight(serialized, "\n"), "\n"))
		}
	}

	linesWritten := 0
	for i := start; i < end && linesWritten < visibleLines; i++ {
		if i >= len(sourceLines) {
			break
		}
		sl := sourceLines[i]

		// In edit mode, skip pre-computed wrapped lines for the cursor line
		// since we'll render the edit buffer with its own wrapping
		if m.editBufLoaded && sl.isWrapped && sl.sourceLineIdx == m.cursorLine {
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
		if m.editBufLoaded && sl.isCursorLine {
			// Show edit buffer with cursor - handle wrapping
			editLines := m.renderEditLineWrapped(contentWidth)
			for j, editLine := range editLines {
				var completeLine string
				if j > 0 {
					// Continuation lines have no line number
					emptyLineNum := m.styles.LineNumber.Width(lineNumWidth).Render("")
					gutterStyle := lipgloss.NewStyle().Background(srcBg)
					completeLine = emptyLineNum + gutterStyle.Render(" ") + editLine
				} else {
					gutterStyle := lipgloss.NewStyle().Background(srcBg)
					completeLine = lineNum + gutterStyle.Render(" ") + editLine
				}
				// Ensure complete line is exactly width
				completeLine = ensureFullWidth(completeLine, width, srcBg)
				renderedLines = append(renderedLines, completeLine)
				linesWritten++
			}
			continue
		} else if sl.isCursorLine {
			// Cursor line when NOT typing - show cursor at current column
			content = m.renderLineWithCursor(sl.content, m.cursorCol, contentWidth, false)
		} else if sl.isPadding {
			// Padding line - blank (for alignment with preview wrapping)
			content = padToWidth("", contentWidth, srcBg)
		} else if sl.isWrapped {
			// Wrapped continuation line - apply block-level tint then selection highlighting
			tinted := applyBlockTint(sl.content, sl.sourceLineIdx, fmCount, sl.isCalc, srcBg)
			lineWithSelection := m.renderLineWithSelection(sl.sourceLineIdx, tinted)
			content = padToWidth(lineWithSelection, contentWidth, srcBg)
		} else {
			// Normal source line - apply block-level tint then selection highlighting
			tinted := applyBlockTint(sl.content, sl.sourceLineIdx, fmCount, sl.isCalc, srcBg)
			lineWithSelection := m.renderLineWithSelection(sl.lineNum-1, tinted)
			content = padToWidth(lineWithSelection, contentWidth, srcBg)
		}

		// Assemble complete line: lineNum + gutter + content
		gutterStyle := lipgloss.NewStyle().Background(srcBg)
		completeLine := lineNum + gutterStyle.Render(" ") + content
		// Ensure complete line is exactly width
		completeLine = ensureFullWidth(completeLine, width, srcBg)
		renderedLines = append(renderedLines, completeLine)
		linesWritten++
	}

	// Fill remaining space with tilde indicators
	for i := linesWritten; i < visibleLines; i++ {
		tildeLine := m.styles.LineNumber.Render("~")
		// Pad tilde line to full width
		tildeLine = ensureFullWidth(tildeLine, width, srcBg)
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
		return theme.SourcePaneBg // Fallback to palette
	}
	return color
}

// previewPaneBg returns the background color for the preview pane.
func (m Model) previewPaneBg() lipgloss.TerminalColor {
	color := m.styles.PreviewPane.GetBackground()
	if _, ok := color.(lipgloss.NoColor); ok {
		return theme.PreviewPaneBg // Fallback to palette
	}
	return color
}

// renderPreviewPaneAligned renders the preview pane using pre-computed aligned lines.
// This avoids recomputing alignment which could cause cycles.
//
// Two rendering modes based on frontmatter presence:
//   - With frontmatter: Globals panel content is rendered inline within frontmatter
//     preview lines (blockID == "") so it appears vertically adjacent to the YAML.
//   - Without frontmatter: Globals panel is rendered as a fixed header at the top.
func (m Model) renderPreviewPaneAligned(width, height int, aligned alignedPanes) string {
	previewLines := aligned.previewLines

	// Pre-compute Globals panel lines (used in both modes).
	globalsPanel := m.renderGlobalsPanel(width)
	globalsPanelLines := strings.Split(globalsPanel, "\n")

	pvBg := m.previewPaneBg()

	// Add separator after globals content
	separatorStyle := lipgloss.NewStyle().
		Foreground(theme.DividerFg).
		Background(pvBg)
	separatorLine := separatorStyle.Render(strings.Repeat("─", width))
	separatorLine = ensureFullWidth(separatorLine, width, pvBg)
	globalsPanelLines = append(globalsPanelLines, separatorLine)

	hasFrontmatter := m.frontmatterLineCount() > 0
	resultsHeight := height

	// Build complete lines to avoid bare newlines
	var allLines []string

	if !hasFrontmatter {
		// No frontmatter: render globals as fixed header at top (original behavior)
		allLines = append(allLines, globalsPanelLines...)
		resultsHeight = max(height-len(globalsPanelLines), 1)
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
	if m.editBufLoaded {
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

	// Track which globals panel line to render next for frontmatter positions.
	// Frontmatter preview lines (blockID == "") get globals content instead of blanks.
	globalsPanelIdx := 0

	linesWritten := 0
	cursorLineProcessed := false
	for j := start; j < end && linesWritten < resultsHeight; j++ {
		if j >= len(previewLines) {
			break
		}
		pl := previewLines[j]

		// Frontmatter lines: always render globals panel content inline,
		// regardless of cursor position. The globals panel is an overlay that
		// replaces the empty preview content of frontmatter lines — the editBuf
		// cursor-line path must not interfere with this substitution.
		if pl.isFrontmatter {
			var completeLine string
			if globalsPanelIdx < len(globalsPanelLines) {
				completeLine = ensureFullWidth(globalsPanelLines[globalsPanelIdx], width, pvBg)
				globalsPanelIdx++
			} else {
				// More frontmatter lines than globals content — fill with background
				completeLine = ensureFullWidth("", width, pvBg)
			}
			allLines = append(allLines, completeLine)
			linesWritten++
			continue
		}

		// In edit mode, handle cursor line specially to match source pane's edit rendering.
		// This only applies to non-frontmatter lines (calc blocks, markdown).
		if m.editBufLoaded && pl.sourceLineNum == m.cursorLine {
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
						completeLine = ensureFullWidth(cursorPreviewLines[k].content, width, pvBg)
					} else {
						// Empty line with background
						completeLine = ensureFullWidth("", width, pvBg)
					}
					allLines = append(allLines, completeLine)
					linesWritten++
				}
				cursorLineProcessed = true
			}
			// Skip all pre-computed lines for cursor (we've already output editLineCount lines)
			continue
		}

		paddedContent := ensureFullWidth(pl.content, width, pvBg)
		allLines = append(allLines, paddedContent)
		linesWritten++
	}

	// Fill remaining space to maintain consistent height
	for i := linesWritten; i < resultsHeight; i++ {
		emptyLine := ensureFullWidth("", width, pvBg)
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

	pvBg := m.previewPaneBg()

	if r.Error != "" && isActuallyCalc {
		// Show full error message - per CONTEXT.md decision
		// "Show full error message in preview (not abbreviated)"
		errStyle := lipgloss.NewStyle().
			Foreground(theme.CalcErrorFg).
			Background(pvBg)

		// Check if this is a blocked (cascading) error
		if r.IsBlocked {
			// Blocked errors show brief indicator - root cause shown elsewhere
			blockedStyle := lipgloss.NewStyle().
				Foreground(theme.CalcBlockedFg).
				Background(pvBg)
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

	// Space with background to prevent terminal bleed-through
	sp := lipgloss.NewStyle().Background(pvBg).Render(" ")

	// Changed indicator: asterisk in yellow for values that were recomputed
	changedMarker := ""
	if r.WasChanged {
		valueStyle = m.styles.Changed.Background(pvBg)
		changedMarker = m.styles.Changed.Background(pvBg).Bold(true).Render("* ")
	}

	switch m.previewMode {
	case PreviewFull:
		// Full mode: "varName → value" for assignments, "→ value" for anonymous calcs
		if r.VarName != "" {
			return changedMarker + m.styles.CalcVarName.Render(r.VarName) + sp + m.styles.CalcArrow.Render("→") + sp + valueStyle.Render(r.Value)
		}
		// Anonymous calculation (no variable assignment) - show arrow without placeholder
		return changedMarker + m.styles.CalcArrow.Render("→") + sp + valueStyle.Render(r.Value)

	case PreviewMinimal:
		// Minimal mode: left-aligned "→ value" (with * if changed)
		arrow := "→ "
		return changedMarker + valueStyle.Render(arrow+r.Value)
	}

	return ""
}

// applyBlockTint applies a subtle foreground color tint to source line text
// based on the block type: frontmatter (muted gray), calc (subtle blue),
// or markdown (default text color — no tint applied).
func applyBlockTint(content string, sourceLineIdx, fmCount int, isCalc bool, bg lipgloss.TerminalColor) string {
	if content == "" {
		return content
	}

	var fg lipgloss.TerminalColor
	switch {
	case sourceLineIdx < fmCount:
		fg = theme.SourceFrontmatter
	case isCalc:
		fg = theme.SourceCalc
	default:
		// Markdown lines — apply background to prevent terminal bleed-through
		return lipgloss.NewStyle().
			Foreground(theme.SourceMarkdown).
			Background(bg).
			Render(content)
	}

	return lipgloss.NewStyle().Foreground(fg).Background(bg).Render(content)
}
