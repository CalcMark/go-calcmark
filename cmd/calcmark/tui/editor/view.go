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

	// Reserve space: status bar (1) + context footer (1) + separator (1)
	contentHeight := totalHeight - 3
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
// Alignment is done BLOCK-BY-BLOCK: for each block, we compare source vs preview
// line counts and pad whichever is shorter.
func (m Model) computeAlignedPanes(sourceWidth, previewWidth int) alignedPanes {
	results := m.GetLineResults()
	lines := m.GetLines()

	// Calculate content width for source pane (accounting for line numbers)
	lineNumWidth := 4
	sourceContentWidth := sourceWidth - lineNumWidth - 2
	if sourceContentWidth < 10 {
		sourceContentWidth = 10
	}

	var sourceLines []sourceLine
	var previewLines []previewLine
	sourceToVisual := make(map[int]int)

	// Process results block by block
	i := 0
	for i < len(results) {
		blockID := results[i].BlockID
		isCalcBlock := results[i].IsCalc

		// Collect all results in this block
		var blockResults []LineResult
		for i < len(results) && results[i].BlockID == blockID {
			blockResults = append(blockResults, results[i])
			i++
		}

		// Build source and preview lines with LINE-BY-LINE alignment
		// Algorithm: For each source line, wrap both sides independently,
		// take max visual lines, pad shorter side to match.
		var blockSourceLines []sourceLine
		var blockPreviewLines []previewLine
		mdRenderer, _ := NewMarkdownRenderer(previewWidth)

		for _, r := range blockResults {
			if r.LineNum >= len(lines) {
				continue
			}
			line := lines[r.LineNum]
			isCursor := r.LineNum == m.cursorLine

			// Wrap source content
			wrappedSource := wrapLine(line, sourceContentWidth)

			// Render and wrap preview content
			var wrappedPreview []string
			if isCalcBlock {
				previewContent := m.renderCalcLine(r, previewWidth)
				// For styled content (ANSI codes), use visual width wrapping
				wrappedPreview = wrapStyledLine(previewContent, previewWidth)
			} else if mdRenderer != nil {
				// Glamour already wraps to width, returns multiple lines
				wrappedPreview = mdRenderer.RenderLine(r.Source)
			} else {
				wrappedPreview = wrapLine(r.Source, previewWidth)
			}

			// Determine max visual lines needed
			sourceCount := len(wrappedSource)
			previewCount := len(wrappedPreview)
			maxLines := sourceCount
			if previewCount > maxLines {
				maxLines = previewCount
			}

			// Record mapping: source line -> first visual line index
			// This is recorded BEFORE adding to blockSourceLines so we get the correct offset
			visualIdx := len(sourceLines) + len(blockSourceLines)
			if _, exists := sourceToVisual[r.LineNum]; !exists {
				sourceToVisual[r.LineNum] = visualIdx
			}

			// Emit source visual lines
			for j := 0; j < maxLines; j++ {
				var sl sourceLine
				if j < sourceCount {
					sl = sourceLine{
						content:       wrappedSource[j],
						sourceLineIdx: r.LineNum,
						isCursorLine:  isCursor && j == 0,
					}
					if j == 0 {
						sl.lineNum = r.LineNum + 1
						sl.isWrapped = false
					} else {
						sl.lineNum = 0
						sl.isWrapped = true
					}
				} else {
					// Padding line for source (preview wrapped more)
					sl = sourceLine{
						content:       "",
						sourceLineIdx: r.LineNum,
						lineNum:       0,
						isPadding:     true,
						isWrapped:     false,
						isCursorLine:  false,
					}
				}
				blockSourceLines = append(blockSourceLines, sl)
			}

			// Emit preview visual lines
			for j := 0; j < maxLines; j++ {
				var pl previewLine
				if j < previewCount {
					pl = previewLine{
						content:       wrappedPreview[j],
						sourceLineNum: r.LineNum,
						blockID:       blockID,
						isCalc:        isCalcBlock,
					}
				} else {
					// Padding line for preview (source wrapped more)
					pl = previewLine{
						content:       "",
						sourceLineNum: r.LineNum,
						blockID:       blockID,
						isCalc:        isCalcBlock,
					}
				}
				blockPreviewLines = append(blockPreviewLines, pl)
			}
		}

		// Append block lines to final result
		sourceLines = append(sourceLines, blockSourceLines...)
		previewLines = append(previewLines, blockPreviewLines...)
	}

	return alignedPanes{
		sourceLines:    sourceLines,
		previewLines:   previewLines,
		sourceToVisual: sourceToVisual,
	}
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

	// Calculate scroll offset in visual line space
	// Ensure cursor is visible by adjusting scroll based on visual position
	visualScrollOffset := m.scrollOffset
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

// wrapLine wraps a single line of text to fit within maxWidth, preferring word boundaries.
// Returns a slice of strings, each fitting within maxWidth.
// Uses visual width (lipgloss.Width) to correctly handle unicode and double-width characters.
func wrapLine(line string, maxWidth int) []string {
	if maxWidth <= 0 {
		return []string{line}
	}

	if lipgloss.Width(line) <= maxWidth {
		return []string{line}
	}

	var result []string
	runes := []rune(line)
	start := 0

	for start < len(runes) {
		// Find how many runes fit within maxWidth
		end := start
		currentWidth := 0
		lastSpaceIdx := -1

		for end < len(runes) {
			rw := lipgloss.Width(string(runes[end]))
			if currentWidth+rw > maxWidth {
				break
			}
			if runes[end] == ' ' {
				lastSpaceIdx = end
			}
			currentWidth += rw
			end++
		}

		// If we've consumed all remaining runes, we're done
		if end >= len(runes) {
			result = append(result, string(runes[start:]))
			break
		}

		// Prefer breaking at word boundary
		if lastSpaceIdx > start {
			// Break after the space
			result = append(result, string(runes[start:lastSpaceIdx+1]))
			start = lastSpaceIdx + 1
		} else if end > start {
			// No space found, hard break
			result = append(result, string(runes[start:end]))
			start = end
		} else {
			// Single character wider than maxWidth - include it anyway to avoid infinite loop
			result = append(result, string(runes[start:start+1]))
			start++
		}
	}

	if len(result) == 0 {
		return []string{line}
	}

	return result
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
	wrappedContent := wrapLine(m.editBuf, width)
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

	// Calculate scroll offset in visual line space
	visualScrollOffset := m.scrollOffset
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

	for j := start; j < end; j++ {
		if j >= len(previewLines) {
			break
		}
		pl := previewLines[j]
		b.WriteString(pl.content)
		if j < end-1 {
			b.WriteString("\n")
		}
	}

	// Fill remaining space to maintain consistent height
	for i := end - start; i < resultsHeight; i++ {
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
		// Error display (amber) - only for actual calculation errors
		// Errors wrap like other content - no truncation
		errText := "⚠ " + r.Error
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("3")).
			Render(errText)
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

	if r.WasChanged {
		valueStyle = valueStyle.Foreground(lipgloss.Color("3"))
	}

	switch m.previewMode {
	case PreviewFull:
		// Full mode: left-aligned "varName  value"
		varStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))
		// Left-justify: name followed by spaces, then value
		return varStyle.Render(r.VarName) + "  " + valueStyle.Render(r.Value)

	case PreviewMinimal:
		// Minimal mode: left-aligned "→ value"
		arrow := "→ "
		return valueStyle.Render(arrow + r.Value)
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

// renderContextFooter renders the context footer showing referenced variables.
func (m Model) renderContextFooter(width int) string {
	results := m.GetLineResults()

	if m.cursorLine >= len(results) {
		return ""
	}

	currentResult := results[m.cursorLine]
	if !currentResult.IsCalc {
		return ""
	}

	// Get referenced variables for the current line
	refs := m.getLineReferences(m.cursorLine)
	if len(refs) == 0 {
		return ""
	}

	// Format as: "var1 = value │ var2 = value │ ..."
	var parts []string
	for _, ref := range refs {
		parts = append(parts, fmt.Sprintf("%s = %s", ref.Name, ref.Value))
	}

	content := strings.Join(parts, " │ ")

	// Let content wrap naturally - no truncation
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Width(width).
		Render(content)
}

// VarReference represents a referenced variable and its value.
type VarReference struct {
	Name  string
	Value string
}

// getLineReferences returns variables referenced in the given line.
func (m Model) getLineReferences(lineNum int) []VarReference {
	lines := m.GetLines()
	if lineNum >= len(lines) {
		return nil
	}

	line := lines[lineNum]

	// Simple variable reference extraction
	// Look for identifier patterns that match known variables
	env := m.eval.GetEnvironment()
	allVars := env.GetAllVariables()

	var refs []VarReference
	seen := make(map[string]bool)

	for varName, val := range allVars {
		// Check if this variable is referenced in the line
		// Skip if it's being defined on this line (left of =)
		if strings.Contains(line, varName) && !strings.HasPrefix(strings.TrimSpace(line), varName+" =") {
			if !seen[varName] {
				seen[varName] = true
				refs = append(refs, VarReference{
					Name:  varName,
					Value: fmt.Sprintf("%v", val),
				})
			}
		}
	}

	// Limit to 4 references to fit in footer
	if len(refs) > 4 {
		refs = refs[:4]
	}

	return refs
}

