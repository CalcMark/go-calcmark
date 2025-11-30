package editor

import (
	"fmt"
	"strings"

	"github.com/CalcMark/go-calcmark/cmd/calcmark/tui/components"
	"github.com/CalcMark/go-calcmark/spec/document"
	"github.com/charmbracelet/lipgloss"
)

// alignedPanes holds pre-computed line structures for both panes.
// This is computed ONCE per render to avoid cycles between pane widths and content.
type alignedPanes struct {
	sourceLines  []sourceLine
	previewLines []previewLine
	// sourceToVisual maps source line index to the first visual line index for that source line
	sourceToVisual map[int]int
}

// View implements tea.Model.
// The Document Editor is a split-pane TUI for working with CalcMark documents.
// Left pane: editable source, Right pane: computed results.
// CRITICAL: Both panes must maintain exact 1:1 vertical line alignment.
func (m Model) View() string {
	if m.quitting {
		return "Goodbye!\n"
	}

	var b strings.Builder

	// Calculate layout
	totalWidth := m.width
	totalHeight := m.height

	// Reserve space: status bar (2) + context footer (2) + separator (1)
	contentHeight := totalHeight - 5
	if contentHeight < 5 {
		contentHeight = 5
	}

	// Calculate pane widths based on preview mode using centralized configuration
	leftWidth, rightWidth := m.GetPaneWidths(totalWidth)

	// Pane content height (minus header row)
	paneContentHeight := contentHeight - 1
	if paneContentHeight < 3 {
		paneContentHeight = 3
	}

	// Calculate globals panel height for alignment
	// (collapsed = 1 line, expanded = 1 + number of globals)
	globalsHeight := 1 // collapsed state
	if m.globalsExpanded {
		globalsHeight = 1 + m.getGlobalsCount()
		if m.getGlobalsCount() == 0 {
			globalsHeight = 2 // "(no globals defined)" message
		}
	}
	globalsHeight++ // +1 for separator line

	// CRITICAL: Compute aligned line structure ONCE to avoid cycles.
	// Both widths are fixed, and we compute wrapping/padding based on them.
	// This prevents: preview reflows → padding changes → width changes → reflow...
	aligned := m.computeAlignedPanes(leftWidth, rightWidth)

	// Render source pane with header
	sourceHeader := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("252")).
		Background(lipgloss.Color("236")).
		Padding(0, 1).
		Width(leftWidth).
		Render("Source")

	// Source pane needs padding at top to match globals panel height in preview
	var sourcePadding string
	if m.previewMode != PreviewHidden {
		for i := 0; i < globalsHeight; i++ {
			sourcePadding += "\n"
		}
	}

	sourceContentHeight := paneContentHeight
	if m.previewMode != PreviewHidden {
		sourceContentHeight = paneContentHeight - globalsHeight
	}
	if sourceContentHeight < 1 {
		sourceContentHeight = 1
	}

	sourceContent := m.renderSourcePaneAligned(leftWidth, sourceContentHeight, aligned)
	sourcePane := lipgloss.JoinVertical(lipgloss.Left, sourceHeader, sourcePadding+sourceContent)

	// Render preview pane (if visible)
	var previewPane string
	if m.previewMode != PreviewHidden {
		previewHeader := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("252")).
			Background(lipgloss.Color("236")).
			Padding(0, 1).
			Width(rightWidth).
			Render("Preview")

		previewContent := m.renderPreviewPaneAligned(rightWidth, paneContentHeight, aligned)
		previewPane = lipgloss.JoinVertical(lipgloss.Left, previewHeader, previewContent)
	}

	// Join panes horizontally - MUST align at top
	if m.previewMode != PreviewHidden {
		b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, sourcePane, previewPane))
	} else {
		b.WriteString(sourcePane)
	}
	b.WriteString("\n")

	// Render context footer (variables referenced in current line)
	contextFooter := m.renderContextFooter(totalWidth)
	b.WriteString(contextFooter)
	b.WriteString("\n")

	// Separator
	separatorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	b.WriteString(separatorStyle.Render(strings.Repeat("─", totalWidth)))
	b.WriteString("\n")

	// Render status bar
	statusBarState := m.GetStatusBarState()
	statusBarStyle := components.DefaultStatusBarStyle()
	statusBar := components.RenderStatusBar(statusBarState, totalWidth, statusBarStyle)
	b.WriteString(statusBar)

	// Render command line if in command mode (overlay)
	if m.mode == ModeCommand {
		cmdLine := lipgloss.NewStyle().
			Foreground(lipgloss.Color("6")).
			Bold(true).
			Render("/" + m.cmdInput + "█")
		b.WriteString("\n")
		b.WriteString(cmdLine)
	}

	return b.String()
}

// computeAlignedPanes computes both pane line structures once with fixed widths.
// This is the single source of truth for alignment, preventing reflow cycles.
// It uses the cached AlignedModel and converts to the legacy format for rendering.
func (m Model) computeAlignedPanes(sourceWidth, previewWidth int) alignedPanes {
	// Use the cached AlignedModel - this is the canonical computation
	// Note: We need a mutable reference to update the cache, but View() receives
	// a value copy. For now, we recompute each time in View() but the AlignedModel
	// computation is still the single source of truth.
	aligned := m.computeAlignedModelFresh(sourceWidth, previewWidth)

	// Convert AlignedModel to legacy alignedPanes format
	sourceLines := make([]sourceLine, len(aligned.SourceLines))
	for i, al := range aligned.SourceLines {
		sourceLines[i] = sourceLine{
			content:       al.Content,
			lineNum:       al.LineNum,
			isPadding:     al.Kind == AlignedLinePadding,
			isWrapped:     al.Kind == AlignedLineWrapped || al.Kind == AlignedLineCursorWrapped,
			isCursorLine:  al.Kind == AlignedLineCursor,
			sourceLineIdx: al.SourceLineIdx,
		}
	}

	previewLines := make([]previewLine, len(aligned.PreviewLines))
	for i, al := range aligned.PreviewLines {
		previewLines[i] = previewLine{
			content:       al.Content,
			sourceLineNum: al.SourceLineIdx,
			blockID:       al.BlockID,
			isCalc:        al.IsCalc,
		}
	}

	return alignedPanes{
		sourceLines:    sourceLines,
		previewLines:   previewLines,
		sourceToVisual: aligned.SourceToVisual,
	}
}

// computeAlignedModelFresh computes a fresh AlignedModel without caching.
// Used by computeAlignedPanes since View() receives a value copy of Model.
func (m Model) computeAlignedModelFresh(sourceWidth, previewWidth int) AlignedModel {
	// Calculate content width for source pane (accounting for line numbers)
	lineNumWidth := 4
	sourceContentWidth := sourceWidth - lineNumWidth - 2
	if sourceContentWidth < 10 {
		sourceContentWidth = 10
	}

	input := AlignedModelInput{
		Lines:              m.GetLines(),
		Results:            m.GetLineResults(),
		SourceContentWidth: sourceContentWidth,
		PreviewWidth:       previewWidth,
		CursorLine:         m.cursorLine,
		PreviewMode:        m.previewMode,
	}

	// Compute with render functions that match view.go behavior
	return ComputeAlignedModel(input, m.renderCalcLine, func(line string, width int) []string {
		mdRenderer, _ := NewMarkdownRenderer(width)
		if mdRenderer != nil {
			return mdRenderer.RenderLine(line)
		}
		return WrapText(line, width)
	})
}

// sourceLine represents a line in the source pane (may be padding or wrapped).
type sourceLine struct {
	content       string // Source text (empty for padding)
	lineNum       int    // Line number (0 = padding/wrapped line, no line number shown)
	isPadding     bool   // True if this is a padding line for preview alignment
	isWrapped     bool   // True if this is a continuation of a wrapped line
	isCursorLine  bool   // True if this is the cursor line
	sourceLineIdx int    // Original source line index (for cursor tracking on wrapped lines)
}

// renderSourcePaneAligned renders the source pane using pre-computed aligned lines.
// This avoids recomputing alignment which could cause cycles.
func (m Model) renderSourcePaneAligned(width, height int, aligned alignedPanes) string {
	var b strings.Builder
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

	linesWritten := 0
	for i := start; i < end && linesWritten < visibleLines; i++ {
		if i >= len(sourceLines) {
			break
		}
		sl := sourceLines[i]

		// In edit mode, skip pre-computed wrapped lines for the cursor line
		// since we'll render the edit buffer with its own wrapping
		if m.mode == ModeEditing && sl.isWrapped && sl.sourceLineIdx == m.cursorLine {
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
		if m.mode == ModeEditing && sl.isCursorLine {
			// Show edit buffer with cursor - handle wrapping
			editLines := m.renderEditLineWrapped(contentWidth)
			for j, editLine := range editLines {
				if j > 0 {
					b.WriteString("\n")
					// Continuation lines have no line number
					b.WriteString(m.styles.LineNumber.Width(lineNumWidth).Render(""))
					b.WriteString(" ")
					linesWritten++
				} else {
					b.WriteString(lineNum)
					b.WriteString(" ")
				}
				b.WriteString(editLine)
			}
			linesWritten++
			if i < end-1 {
				b.WriteString("\n")
			}
			continue
		} else if sl.isCursorLine {
			// Highlight current line
			content = m.styles.CurrentLine.
				Width(contentWidth).
				Render(padToWidth(sl.content, contentWidth))
		} else if sl.isPadding {
			// Padding line - blank (for alignment with preview wrapping)
			content = ""
		} else if sl.isWrapped {
			// Wrapped continuation line - no extra indent (line number space provides visual separation)
			content = padToWidth(sl.content, contentWidth)
		} else {
			// Normal source line - already fits within width
			content = padToWidth(sl.content, contentWidth)
		}

		b.WriteString(lineNum)
		b.WriteString(" ")
		b.WriteString(content)
		linesWritten++

		if i < end-1 {
			b.WriteString("\n")
		}
	}

	// Fill remaining space with tilde indicators
	for i := linesWritten; i < visibleLines; i++ {
		b.WriteString("\n")
		b.WriteString(m.styles.LineNumber.Render("~"))
	}

	return b.String()
}

// padToWidth pads a string to exactly width visual columns (no truncation).
// Uses lipgloss.Width for correct unicode handling.
func padToWidth(s string, width int) string {
	visualWidth := lipgloss.Width(s)
	if visualWidth >= width {
		return s
	}
	return s + strings.Repeat(" ", width-visualWidth)
}

// wrapStyledLine wraps a line containing ANSI escape codes using visual width.
// This is needed for styled content where len(string) != visual width.
func wrapStyledLine(line string, maxWidth int) []string {
	if maxWidth <= 0 {
		return []string{line}
	}

	visualWidth := lipgloss.Width(line)
	if visualWidth <= maxWidth {
		return []string{line}
	}

	// For styled content, we can't easily split mid-string without breaking ANSI codes.
	// Best approach: don't wrap styled content, let terminal handle overflow.
	// This is acceptable because calc results are typically short.
	// If we need wrapping, we'd need to strip styles, wrap, then re-apply.
	return []string{line}
}

// renderEditLine renders the line being edited with cursor (single line, no wrapping).
func (m Model) renderEditLine(width int) string {
	var line string

	// Text style for non-cursor parts (uses EditLine foreground color)
	textStyle := m.styles.EditLine.UnsetBackground().UnsetWidth()

	if m.cursorCol >= len(m.editBuf) {
		// Cursor at end - show text followed by cursor
		line = textStyle.Render(m.editBuf) + m.styles.Cursor.Render(" ")
	} else {
		// Cursor in middle - highlight the character under cursor
		before := m.editBuf[:m.cursorCol]
		charAtCursor := string(m.editBuf[m.cursorCol])
		after := m.editBuf[m.cursorCol+1:]

		line = textStyle.Render(before) + m.styles.Cursor.Render(charAtCursor) + textStyle.Render(after)
	}

	// Use configured edit line background for the full line
	return m.styles.EditLine.
		Width(width).
		Render(line)
}

// renderEditLineWrapped renders the edit buffer with wrapping support.
// Returns multiple lines if the content exceeds width.
func (m Model) renderEditLineWrapped(width int) []string {
	if len(m.editBuf) <= width {
		// Fits on one line
		return []string{m.renderEditLine(width)}
	}

	// Wrap the edit buffer content
	wrappedContent := WrapText(m.editBuf, width)
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

	textStyle := m.styles.EditLine.UnsetBackground().UnsetWidth()

	for i, seg := range wrappedContent {
		var line string
		if i == cursorLineIdx {
			// This line has the cursor
			if cursorColInLine >= len(seg) {
				line = textStyle.Render(seg) + m.styles.Cursor.Render(" ")
			} else {
				before := seg[:cursorColInLine]
				charAtCursor := string(seg[cursorColInLine])
				after := seg[cursorColInLine+1:]
				line = textStyle.Render(before) + m.styles.Cursor.Render(charAtCursor) + textStyle.Render(after)
			}
		} else {
			line = textStyle.Render(seg)
		}

		// Apply edit line background
		rendered := m.styles.EditLine.Width(width).Render(line)
		result = append(result, rendered)
	}

	return result
}

// previewLine represents a line in the preview pane with its source mapping.
type previewLine struct {
	content       string // Rendered content for this preview line
	sourceLineNum int    // Which source line this corresponds to (-1 if spanning multiple)
	blockID       string // Block this line belongs to
	isCalc        bool   // Whether this is from a CalcBlock
}

// renderPreviewPaneAligned renders the preview pane using pre-computed aligned lines.
// This avoids recomputing alignment which could cause cycles.
func (m Model) renderPreviewPaneAligned(width, height int, aligned alignedPanes) string {
	var b strings.Builder
	previewLines := aligned.previewLines

	// Globals panel at top (collapsible) - per spec lines 334-339
	globalsPanel := m.renderGlobalsPanel(width)
	globalsHeight := strings.Count(globalsPanel, "\n") + 1

	b.WriteString(globalsPanel)
	b.WriteString("\n")

	// Separator after globals
	separatorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	b.WriteString(separatorStyle.Render(strings.Repeat("─", width)))
	b.WriteString("\n")

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
	if m.mode == ModeEditing {
		// Count how many lines the edit buffer would produce
		contentWidth := width // approximate
		editLines := WrapText(m.editBuf, contentWidth)
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
		if m.mode == ModeEditing && pl.sourceLineNum == m.cursorLine {
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
					if k > 0 || linesWritten > 0 {
						b.WriteString("\n")
					}
					// Show preview content if available, otherwise empty
					if k < len(cursorPreviewLines) {
						b.WriteString(cursorPreviewLines[k].content)
					}
					linesWritten++
				}
				cursorLineProcessed = true
			}
			// Skip all pre-computed lines for cursor (we've already output editLineCount lines)
			continue
		}

		if linesWritten > 0 {
			b.WriteString("\n")
		}
		b.WriteString(pl.content)
		linesWritten++
	}

	// Fill remaining space to maintain consistent height
	for i := linesWritten; i < resultsHeight; i++ {
		b.WriteString("\n")
	}

	return b.String()
}

// renderCalcLine renders a single calculation line result.
func (m Model) renderCalcLine(r LineResult, width int) string {
	// Use the detector to check if this line is actually a calculation
	detector := document.NewDetector()
	isActuallyCalc, _ := detector.IsCalculation(r.Source)

	if r.Error != "" && isActuallyCalc {
		// Show brief error indicator inline - detailed error shown in context footer
		errStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("208")) // amber

		// Just show a brief indicator - the context footer has the details
		return errStyle.Render("⚠ error")
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

	valueStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("6"))

	// Changed indicator: asterisk in yellow for values that were recomputed
	changedMarker := ""
	if r.WasChanged {
		valueStyle = valueStyle.Foreground(lipgloss.Color("3"))
		changedMarker = lipgloss.NewStyle().
			Foreground(lipgloss.Color("3")).
			Bold(true).
			Render("* ")
	}

	switch m.previewMode {
	case PreviewFull:
		// Full mode: left-aligned "varName → value" (with * if changed)
		varStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))
		arrowStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))
		return changedMarker + varStyle.Render(r.VarName) + " " + arrowStyle.Render("→") + " " + valueStyle.Render(r.Value)

	case PreviewMinimal:
		// Minimal mode: left-aligned "→ value" (with * if changed)
		arrow := "→ "
		return changedMarker + valueStyle.Render(arrow + r.Value)
	}

	return ""
}

// renderGlobalsPanel renders the collapsible globals panel.
func (m Model) renderGlobalsPanel(width int) string {
	state := m.GetGlobalsPanelState()
	globalsCount := len(state.Globals)

	if !state.Expanded {
		// Collapsed: just show count
		indicator := "▸"
		text := fmt.Sprintf(" Globals (%d)", globalsCount)
		hint := "[g]"

		left := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("252")).
			Render(indicator + text)

		right := lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Render(hint)

		// Space between left and right
		space := width - lipgloss.Width(left) - lipgloss.Width(right) - 2
		if space < 0 {
			space = 0
		}

		return left + strings.Repeat(" ", space) + right
	}

	// Expanded: show all globals
	var b strings.Builder

	indicator := "▾"
	text := " Globals"
	hint := "[g]"

	left := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("252")).
		Render(indicator + text)

	right := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Render(hint)

	space := width - lipgloss.Width(left) - lipgloss.Width(right) - 2
	if space < 0 {
		space = 0
	}

	b.WriteString(left)
	b.WriteString(strings.Repeat(" ", space))
	b.WriteString(right)

	if globalsCount == 0 {
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Italic(true).
			Render("  (no globals defined)"))
		return b.String()
	}

	for i, g := range state.Globals {
		b.WriteString("\n")

		prefix := "  "
		if state.Focused && i == state.FocusIndex {
			prefix = "> "
		}

		nameStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))
		valueStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("6"))

		if g.IsExchange {
			nameStyle = nameStyle.Foreground(lipgloss.Color("5"))
		}

		// Format: "  name          value"
		name := fmt.Sprintf("%-18s", g.Name)
		b.WriteString(prefix)
		b.WriteString(nameStyle.Render(name))
		b.WriteString(valueStyle.Render(g.Value))
	}

	return b.String()
}

// renderContextFooter renders the context footer showing errors or referenced variables.
// Delegates to components.RenderContextFooter with prepared state.
func (m Model) renderContextFooter(width int) string {
	results := m.GetLineResults()

	// Build state for the pure render function
	state := components.ContextFooterState{}

	// Check bounds
	if m.cursorLine < len(results) {
		currentResult := results[m.cursorLine]
		state.IsCalcLine = currentResult.IsCalc

		if currentResult.IsCalc && currentResult.Error != "" {
			state.HasError = true
			state.Diagnostic = currentResult.Diagnostic

			// If no structured diagnostic, parse the error string for display
			if state.Diagnostic == nil {
				errInfo := components.ParseErrorForDisplay(currentResult.Error)
				state.ErrorMessage = errInfo.ShortMessage
				state.ErrorHint = errInfo.Hint
			}
		}

		// Get variable references if no error
		if !state.HasError && state.IsCalcLine {
			state.References = m.getLineReferences(m.cursorLine)
		}
	}

	return components.RenderContextFooter(state, width)
}

// getLineReferences returns variables referenced in the given line.
// Delegates to components.FindLineReferences with model's known variables.
func (m Model) getLineReferences(lineNum int) []components.VarReference {
	lines := m.GetLines()
	if lineNum >= len(lines) {
		return nil
	}

	line := lines[lineNum]

	// Build map of known variables from environment
	env := m.eval.GetEnvironment()
	allVars := env.GetAllVariables()

	knownVars := make(map[string]string)
	for varName, val := range allVars {
		knownVars[varName] = fmt.Sprintf("%v", val)
	}

	return components.FindLineReferences(line, knownVars, 4)
}
