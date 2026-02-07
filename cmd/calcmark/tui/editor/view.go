package editor

import (
	"fmt"
	"strings"

	"github.com/CalcMark/go-calcmark/cmd/calcmark/tui/components"
	"github.com/CalcMark/go-calcmark/cmd/calcmark/tui/geometry"
	"github.com/CalcMark/go-calcmark/spec/document"
	"github.com/CalcMark/go-calcmark/spec/semantic"
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

	// If help mode is active, render help overlay on top
	if m.mode == StateHelp {
		helpView := m.renderHelpOverlay()
		// Center the help overlay on screen with consistent background
		return lipgloss.Place(m.width, m.height,
			lipgloss.Center, lipgloss.Center,
			helpView,
			lipgloss.WithWhitespaceChars(" "),
			lipgloss.WithWhitespaceForeground(lipgloss.Color("237")),
		)
	}

	// If command menu is active, render it as full-screen modal (like help)
	if m.mode == StateCommandMenu {
		menuPopup := m.renderCommandMenuPopup()
		// Center the command menu on screen with consistent background
		return lipgloss.Place(m.width, m.height,
			lipgloss.Center, lipgloss.Center,
			menuPopup,
			lipgloss.WithWhitespaceChars(" "),
			lipgloss.WithWhitespaceForeground(lipgloss.Color("237")),
		)
	}

	// If file picker is active, render it as full-screen modal
	if m.mode == StateFilePicker {
		pickerOverlay := m.renderFilePickerOverlay()
		return lipgloss.Place(m.width, m.height,
			lipgloss.Center, lipgloss.Center,
			pickerOverlay,
			lipgloss.WithWhitespaceChars(" "),
			lipgloss.WithWhitespaceForeground(lipgloss.Color("237")),
		)
	}

	// Calculate layout
	totalWidth := m.width
	totalHeight := m.height

	// Reserve space: status bar (2) + context footer (2) + separator (1) + empty line (1)
	contentHeight := max(totalHeight-6, 5)

	// Calculate pane widths based on preview mode using centralized configuration
	leftWidth, rightWidth := m.GetPaneWidths(totalWidth)

	// Account for divider between panes (1 character)
	// The divider is visually part of the separation, so we subtract it from left pane
	const dividerWidth = 1
	leftContentWidth := leftWidth - dividerWidth // Content area width (excludes divider)

	// Pane content height (minus header row)
	paneContentHeight := max(contentHeight-1, 3)

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
	// Use leftContentWidth (not leftWidth) because divider takes 1 char from left pane
	// Both widths are fixed, and we compute wrapping/padding based on them.
	// This prevents: preview reflows → padding changes → width changes → reflow...
	aligned := m.computeAlignedPanes(leftContentWidth, rightWidth)

	// Render source pane with header
	sourceHeader := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("252")).
		Background(lipgloss.Color("236")).
		Padding(0, 1).
		Width(leftContentWidth).
		Render("Source")
	// Ensure header is exactly leftContentWidth
	sourceHeader = ensureFullWidth(sourceHeader, leftContentWidth, lipgloss.Color("236"))

	// Source pane needs padding at top to match globals panel height in preview
	var sourcePaddingLines []string
	if m.previewMode != PreviewHidden {
		for i := 0; i < globalsHeight; i++ {
			// Each padding line must be full width with background
			sourcePaddingLines = append(sourcePaddingLines, ensureFullWidth("", leftContentWidth, lipgloss.Color("236")))
		}
	}

	sourceContentHeight := paneContentHeight
	if m.previewMode != PreviewHidden {
		sourceContentHeight = paneContentHeight - globalsHeight
	}
	if sourceContentHeight < 1 {
		sourceContentHeight = 1
	}

	sourceContent := m.renderSourcePaneAligned(leftContentWidth, sourceContentHeight, aligned)

	// Assemble source pane with header and padding
	var sourcePaneLines []string
	sourcePaneLines = append(sourcePaneLines, sourceHeader)
	sourcePaneLines = append(sourcePaneLines, sourcePaddingLines...)
	sourcePaneLines = append(sourcePaneLines, strings.Split(sourceContent, "\n")...)
	sourcePane := strings.Join(sourcePaneLines, "\n")

	// Render preview pane (if visible)
	var previewPane string
	if m.previewMode != PreviewHidden {
		previewHeader := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("252")).
			Background(lipgloss.Color("236")).
			Padding(0, 1).
			Width(rightWidth).
			Render("Results")
		// Ensure header is exactly rightWidth
		previewHeader = ensureFullWidth(previewHeader, rightWidth, lipgloss.Color("236"))

		previewContent := m.renderPreviewPaneAligned(rightWidth, paneContentHeight, aligned)

		// Assemble without bare newlines
		var previewPaneLines []string
		previewPaneLines = append(previewPaneLines, previewHeader)
		previewPaneLines = append(previewPaneLines, strings.Split(previewContent, "\n")...)
		previewPane = strings.Join(previewPaneLines, "\n")
	}

	// Build complete UI as array of lines to avoid bare newlines
	var allUILines []string

	// Add panes (source and preview)
	if m.previewMode != PreviewHidden {
		// SideBySide expects content widths and adds divider
		// leftContentWidth already accounts for divider (leftWidth - dividerWidth)
		sbs := NewSideBySide(leftContentWidth, rightWidth, lipgloss.Color("236"), lipgloss.Color("236"))
		panesOutput := sbs.Render(sourcePane, previewPane)
		allUILines = append(allUILines, strings.Split(panesOutput, "\n")...)
	} else {
		// Single pane - use full left width (no divider in single-pane mode)
		sourcePane = ensureLinesAreFullWidth(sourcePane, leftWidth, lipgloss.Color("236"))
		allUILines = append(allUILines, strings.Split(sourcePane, "\n")...)
	}

	// Get context footer background once for all footer-related elements
	contextFooterBg := m.styles.ContextFooter.GetBackground()
	if _, ok := contextFooterBg.(lipgloss.NoColor); ok {
		contextFooterBg = m.sourcePaneBg() // Fallback
	}

	// NOTE: Autocomplete popup is rendered as an overlay after all lines are built

	// Empty line with background (use context footer background for transition)
	emptyLine := components.StyledPadding(totalWidth, contextFooterBg)
	allUILines = append(allUILines, emptyLine)

	// Context footer (variables referenced in current line)
	contextFooter := m.renderContextFooter(totalWidth)
	// The context footer is already multiple lines, so we need to handle each line
	contextFooterLines := strings.Split(contextFooter, "\n")
	for i, line := range contextFooterLines {
		contextFooterLines[i] = ensureFullWidth(line, totalWidth, contextFooterBg)
	}
	allUILines = append(allUILines, contextFooterLines...)

	// Separator - use context footer background for consistency
	separatorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Background(contextFooterBg)
	separatorLine := separatorStyle.Render(strings.Repeat("─", totalWidth))
	separatorLine = ensureFullWidth(separatorLine, totalWidth, contextFooterBg)
	allUILines = append(allUILines, separatorLine)

	// Status bar
	statusBarState := m.GetStatusBarState()
	statusBarStyle := components.DefaultStatusBarStyle()
	// Use themed status bar background - apply to ALL sub-components to prevent terminal bleed
	statusBarBg := m.styles.StatusBar.GetBackground()
	if _, ok := statusBarBg.(lipgloss.NoColor); ok {
		statusBarBg = m.sourcePaneBg() // Fallback to source pane background
	}
	statusBarStyle.Bar = statusBarStyle.Bar.Background(statusBarBg)
	statusBarStyle.Filename = statusBarStyle.Filename.Background(statusBarBg)
	statusBarStyle.Modified = statusBarStyle.Modified.Background(statusBarBg)
	statusBarStyle.Position = statusBarStyle.Position.Background(statusBarBg)
	statusBarStyle.Mode = statusBarStyle.Mode.Background(statusBarBg)
	statusBarStyle.Hints = statusBarStyle.Hints.Background(statusBarBg)
	statusBarStyle.StatusOK = statusBarStyle.StatusOK.Background(statusBarBg)
	statusBarStyle.StatusErr = statusBarStyle.StatusErr.Background(statusBarBg)
	statusBar := components.RenderStatusBar(statusBarState, totalWidth, statusBarStyle)
	// Ensure status bar lines have backgrounds
	statusBarLines := strings.Split(statusBar, "\n")
	for i, line := range statusBarLines {
		statusBarLines[i] = ensureFullWidth(line, totalWidth, statusBarBg)
	}
	allUILines = append(allUILines, statusBarLines...)

	// Overlay autocomplete popup if active
	if m.mode == StateAutocomplete && m.autocompleteState.Visible {
		popup := m.renderAutocompletePopup()
		if popup != "" {
			row, col := m.calculatePopupScreenPosition(contentHeight)
			allUILines = overlayPopupOnLines(allUILines, popup, row, col)
		}
	}

	// Join all lines - no bare newlines, all fully styled
	return strings.Join(allUILines, "\n")
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
		return geometry.WrapText(line, width)
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
			// Wrapped continuation line - apply source text color and background
			styledContent := m.styles.SourceText.Render(sl.content)
			content = padToWidth(styledContent, contentWidth, lipgloss.Color("236"))
		} else {
			// Normal source line - apply source text color and background
			styledContent := m.styles.SourceText.Render(sl.content)
			content = padToWidth(styledContent, contentWidth, lipgloss.Color("236"))
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

// padToWidth pads a string to exactly width visual columns (no truncation).
// Uses lipgloss.Width for correct unicode handling.
// Pads with styled spaces using the given background color to prevent terminal bleed-through.
func padToWidth(s string, width int, bg lipgloss.TerminalColor) string {
	visualWidth := lipgloss.Width(s)
	if visualWidth >= width {
		return s
	}
	padding := width - visualWidth
	// Use centralized StyledPadding utility
	return s + components.StyledPadding(padding, bg)
}

// ensureFullWidth ensures a complete line (with all components) is exactly the target width.
// This should be called on the FINAL assembled line, not on individual components.
func ensureFullWidth(line string, width int, bg lipgloss.TerminalColor) string {
	currentWidth := lipgloss.Width(line)
	if currentWidth >= width {
		return line
	}
	padding := width - currentWidth
	// Use centralized StyledPadding utility
	return line + components.StyledPadding(padding, bg)
}

// ensureLinesAreFullWidth ensures every line in a multi-line string is exactly the target width.
func ensureLinesAreFullWidth(content string, width int, bg lipgloss.TerminalColor) string {
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		lines[i] = ensureFullWidth(line, width, bg)
	}
	return strings.Join(lines, "\n")
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

	// For styled content that exceeds maxWidth, we need to wrap it properly.
	// Strategy: Use lipgloss to extract plain text, wrap it, then let lipgloss handle rendering.
	// This preserves styles while ensuring proper wrapping.

	// Extract plain text (removes ANSI codes)
	plainText := stripANSI(line)

	// Wrap the plain text
	wrappedPlainLines := geometry.WrapText(plainText, maxWidth)

	// Return wrapped lines (styles will be handled by caller if needed)
	// For calc results like "a → 2", the arrow and value are usually short enough
	// that wrapping preserves the basic format
	return wrappedPlainLines
}

// stripANSI removes ANSI escape codes from a string, returning plain text.
// This is needed to calculate actual text length for wrapping.
func stripANSI(s string) string {
	// Strip ANSI escape sequences matching pattern: \x1b\[[0-9;]*m
	var result strings.Builder
	result.Grow(len(s))
	inEscape := false
	for i := 0; i < len(s); i++ {
		if s[i] == '\x1b' && i+1 < len(s) && s[i+1] == '[' {
			inEscape = true
			i++ // skip '['
			continue
		}
		if inEscape {
			if s[i] == 'm' {
				inEscape = false
			}
			continue
		}
		result.WriteByte(s[i])
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
		lineStyle = m.styles.EditLine.Copy().ColorWhitespace(true).Inline(true)
	} else {
		// When not typing (showing document content with cursor)
		lineStyle = m.styles.CurrentLine.Copy().ColorWhitespace(true).Inline(true)
	}

	// CRITICAL INSIGHT: The issue is that concatenating multiple .Render() calls
	// creates separate ANSI blocks. We need ONE continuous background.
	// Solution: Build the ENTIRE content first, THEN render once with padding included

	contentLen := len(content)

	// Determine cursor character
	var cursorChar string
	if col >= contentLen {
		cursorChar = " "
	} else {
		cursorChar = string(content[col])
	}

	// Calculate total padding needed
	totalPadding := width - contentLen
	if col >= contentLen {
		totalPadding -= 1 // Account for cursor space
	}

	// Build result by rendering segments with inline styles
	var result strings.Builder

	// Before cursor
	if col > 0 {
		result.WriteString(lineStyle.Render(content[:col]))
	}

	// Cursor
	result.WriteString(m.styles.Cursor.Inline(true).Render(cursorChar))

	// After cursor
	if col+1 < contentLen {
		result.WriteString(lineStyle.Render(content[col+1:]))
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

		// Space between left and right with background
		space := width - lipgloss.Width(left) - lipgloss.Width(right)
		if space < 0 {
			space = 0
		}

		// Use centralized StyledPadding utility
		header := left + components.StyledPadding(space, lipgloss.Color("236")) + right
		return ensureFullWidth(header, width, lipgloss.Color("236"))
	}

	// Expanded: show all globals
	var allLines []string

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

	space := width - lipgloss.Width(left) - lipgloss.Width(right)
	if space < 0 {
		space = 0
	}

	// Use centralized StyledPadding utility
	headerLine := left + components.StyledPadding(space, lipgloss.Color("236")) + right
	headerLine = ensureFullWidth(headerLine, width, lipgloss.Color("236"))
	allLines = append(allLines, headerLine)

	if globalsCount == 0 {
		noGlobalsLine := lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Italic(true).
			Render("  (no globals defined)")
		noGlobalsLine = ensureFullWidth(noGlobalsLine, width, lipgloss.Color("236"))
		allLines = append(allLines, noGlobalsLine)
		return strings.Join(allLines, "\n")
	}

	for i, g := range state.Globals {
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
		globalLine := prefix + nameStyle.Render(name) + valueStyle.Render(g.Value)
		globalLine = ensureFullWidth(globalLine, width, lipgloss.Color("236"))
		allLines = append(allLines, globalLine)
	}

	return strings.Join(allLines, "\n")
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

	// Add autocomplete details when popup is active
	if m.mode == StateAutocomplete && m.autocompleteState.Visible {
		if len(m.autocompleteState.Suggestions) > 0 {
			selected := m.autocompleteState.Suggestions[m.autocompleteState.Selected]
			state.AutocompleteActive = true
			state.AutocompleteName = selected.InsertText
			if state.AutocompleteName == "" {
				state.AutocompleteName = selected.Name
			}
			state.AutocompleteSyntax = selected.Syntax

			// For functions, show parameter examples instead of/in addition to description
			// This helps users understand what format to use for each parameter
			funcName := selected.InsertText
			if funcName == "" {
				funcName = selected.Name
			}
			paramHint := formatFunctionParamHint(funcName)
			if paramHint != "" {
				state.AutocompleteDesc = paramHint
			} else {
				state.AutocompleteDesc = selected.Description
			}
		}
	}

	// Check for function argument context (when typing inside function call)
	// NOTE: We check this even when there's an error because incomplete function
	// calls (like "accumulate(") will have parse errors but should still show
	// parameter help. The function context takes priority over error display
	// in RenderContextFooter (Priority 0.5 vs Priority 1).
	if !state.AutocompleteActive {
		m.loadCurrentLineIntoEditBuffer()
		cursorCtx := GetCursorContext(m.editBuf, m.cursorCol)
		if cursorCtx.InFunctionCall && cursorCtx.ParamSpec != nil {
			state.InFunctionCall = true
			state.FunctionName = cursorCtx.FunctionName
			state.ParamName = cursorCtx.ParamSpec.Name
			state.ParamExamples = FormatParamHelp(cursorCtx.ParamSpec)
			state.ArgIndex = cursorCtx.ArgIndex
		}
	}

	// Get themed context footer background
	contextFooterBg := m.styles.ContextFooter.GetBackground()
	if _, ok := contextFooterBg.(lipgloss.NoColor); ok {
		contextFooterBg = m.sourcePaneBg() // Fallback to source pane background
	}

	return components.RenderContextFooter(state, width, contextFooterBg)
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

// renderAutocompletePopup renders the autocomplete popup box.
// Returns the popup as a styled string with border.
func (m Model) renderAutocompletePopup() string {
	if !m.autocompleteState.Visible || len(m.autocompleteState.Suggestions) == 0 {
		return ""
	}

	style := components.DefaultPopupStyle()
	return components.RenderPopupBox(m.autocompleteState, style)
}

// calculatePopupScreenPosition computes where to place the popup on screen.
// Returns (row, col) as screen coordinates.
func (m Model) calculatePopupScreenPosition(contentHeight int) (row, col int) {
	// The cursor visual position in the content area
	visualCursorRow := m.cursorLine - m.scrollOffset

	// Account for headers: source header (1) + globals padding if preview visible
	headerRows := 1
	if m.previewMode != PreviewHidden {
		globalsHeight := 1
		if m.globalsExpanded {
			globalsHeight = 1 + m.getGlobalsCount()
			if m.getGlobalsCount() == 0 {
				globalsHeight = 2
			}
		}
		globalsHeight++ // separator
		headerRows += globalsHeight
	}

	// Screen row for popup (below cursor)
	row = headerRows + visualCursorRow + 1

	// Ensure popup fits on screen
	popupHeight := m.autocompleteState.PopupHeight + 2 // +2 for hint and border
	if row+popupHeight > contentHeight {
		// Place above cursor instead
		row = headerRows + visualCursorRow - popupHeight
		if row < headerRows {
			row = headerRows
		}
	}

	// Column: align with cursor, adjusted for line number gutter
	gutterWidth := 5 // "  N→" format
	col = gutterWidth + m.cursorCol

	// Ensure popup doesn't go off right edge
	leftWidth, _ := m.GetPaneWidths(m.width)
	if col+m.autocompleteState.PopupWidth > leftWidth {
		col = leftWidth - m.autocompleteState.PopupWidth
	}
	if col < gutterWidth {
		col = gutterWidth
	}

	return row, col
}

// overlayPopupOnLines overlays the popup at the given position on the UI lines.
// This is a pure function that composites the popup onto the rendered output.
func overlayPopupOnLines(lines []string, popup string, row, col int) []string {
	if popup == "" {
		return lines
	}

	popupLines := strings.Split(popup, "\n")
	result := make([]string, len(lines))
	copy(result, lines)

	for i, popupLine := range popupLines {
		targetRow := row + i
		if targetRow < 0 || targetRow >= len(result) {
			continue
		}

		// Get the base line and overlay the popup
		baseLine := result[targetRow]
		result[targetRow] = overlayStringAt(baseLine, popupLine, col)
	}

	return result
}

// overlayStringAt overlays overlay on base starting at column col.
// Handles ANSI escape codes properly using lipgloss.Width for visual width.
func overlayStringAt(base, overlay string, col int) string {
	// Convert to runes for proper unicode handling
	baseRunes := []rune(base)
	overlayRunes := []rune(overlay)

	// Build result: base up to col, then overlay, then rest of base
	var result []rune

	// Copy base characters up to col
	visualCol := 0
	baseIdx := 0
	for baseIdx < len(baseRunes) && visualCol < col {
		r := baseRunes[baseIdx]
		result = append(result, r)
		// Skip ANSI escape sequences in width calculation
		if r == '\x1b' {
			// Find end of escape sequence
			for baseIdx < len(baseRunes)-1 && baseRunes[baseIdx] != 'm' {
				baseIdx++
				result = append(result, baseRunes[baseIdx])
			}
		} else {
			visualCol++
		}
		baseIdx++
	}

	// Pad with spaces if base is shorter than col
	for visualCol < col {
		result = append(result, ' ')
		visualCol++
	}

	// Append the overlay
	result = append(result, overlayRunes...)

	// CRITICAL: Add explicit ANSI reset after overlay to prevent background bleeding.
	// Lipgloss may set background colors that would otherwise affect subsequent text.
	result = append(result, []rune("\x1b[0m")...)

	// Skip the overlaid portion of base using VISUAL width of overlay
	// CRITICAL: Use lipgloss.Width() to get visual width, not len(overlayRunes)
	// which includes ANSI escape codes and would skip too many characters.
	overlayVisualWidth := lipgloss.Width(overlay)
	for baseIdx < len(baseRunes) && overlayVisualWidth > 0 {
		r := baseRunes[baseIdx]
		if r == '\x1b' {
			// Keep escape sequences (they have zero visual width)
			for baseIdx < len(baseRunes) && baseRunes[baseIdx] != 'm' {
				baseIdx++
			}
			baseIdx++
		} else {
			overlayVisualWidth--
			baseIdx++
		}
	}

	// Append rest of base
	if baseIdx < len(baseRunes) {
		result = append(result, baseRunes[baseIdx:]...)
	}

	return string(result)
}

// extractErrorHint extracts a brief hint from an error message for inline display.
// Returns something like `"var"?` for undefined variables, or a short description.
func extractErrorHint(errMsg string, maxWidth int) string {
	// Try to extract a quoted identifier from the error message
	// Common patterns: `undefined variable "foo"`, `cannot reassign 'bar'`
	for _, quote := range []string{`"`, `'`} {
		start := strings.Index(errMsg, quote)
		if start >= 0 {
			end := strings.Index(errMsg[start+1:], quote)
			if end > 0 && end < 30 {
				identifier := errMsg[start : start+end+2]
				hint := identifier + "?"
				if len(hint) <= maxWidth {
					return hint
				}
			}
		}
	}

	// Fallback: extract error type from code prefix
	if idx := strings.Index(errMsg, ": "); idx > 0 && idx < 25 {
		code := errMsg[:idx]
		// Convert snake_case to short form
		code = strings.ReplaceAll(code, "_", " ")
		if len(code) <= maxWidth {
			return code
		}
	}

	// Last resort: just say "error"
	return "error"
}

// renderCommandMenuPopup renders the command menu as a popup overlay.
func (m Model) renderCommandMenuPopup() string {
	commands := EditorCommands
	selected := m.commandMenuState.Selected

	// Calculate popup dimensions
	innerWidth := 40 // Wide enough for "Ctrl+Shift+S  Save As..." format

	// Border characters (rounded style matching autocomplete)
	borderFg := lipgloss.Color("#5C5C5C")
	borderStyle := lipgloss.NewStyle().Foreground(borderFg)

	topBorder := borderStyle.Render("╭" + strings.Repeat("─", innerWidth) + "╮")
	bottomBorder := borderStyle.Render("╰" + strings.Repeat("─", innerWidth) + "╯")
	leftBorder := borderStyle.Render("│")
	rightBorder := borderStyle.Render("│")

	var lines []string
	lines = append(lines, topBorder)

	// Title row
	title := " Commands"
	for len(title) < innerWidth {
		title += " "
	}
	titleStyle := lipgloss.NewStyle().Bold(true).Background(lipgloss.Color("#1E1E1E"))
	lines = append(lines, leftBorder+titleStyle.Render(title)+rightBorder)

	// Separator
	sepLine := strings.Repeat("─", innerWidth)
	lines = append(lines, leftBorder+borderStyle.Render(sepLine)+rightBorder)

	// Command items - show accelerator and name
	itemBg := lipgloss.Color("#1E1E1E")
	selectedBg := lipgloss.Color("#4A90D9")

	for i, cmd := range commands {
		line := fmt.Sprintf(" %-12s %s", cmd.Accelerator, cmd.Name)
		for len(line) < innerWidth {
			line += " "
		}

		var styledLine string
		if i == selected {
			selStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(selectedBg).
				Bold(true)
			styledLine = selStyle.Render(line)
		} else {
			itemStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("#CCCCCC")).
				Background(itemBg)
			styledLine = itemStyle.Render(line)
		}
		lines = append(lines, leftBorder+styledLine+rightBorder)
	}

	// Hint row
	hint := " Enter:select  Esc:close"
	for len(hint) < innerWidth {
		hint += " "
	}
	hintStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#666666")).
		Background(itemBg).
		Italic(true)
	lines = append(lines, leftBorder+hintStyle.Render(hint)+rightBorder)

	lines = append(lines, bottomBorder)
	return strings.Join(lines, "\n")
}

// renderFilePickerOverlay renders the file picker as a modal overlay.
func (m Model) renderFilePickerOverlay() string {
	var lines []string

	// Header with current directory
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("252")).
		Background(lipgloss.Color("236")).
		Padding(0, 1)

	header := headerStyle.Render(fmt.Sprintf(" Save to: %s ", m.filePicker.CurrentDirectory))
	lines = append(lines, header)
	lines = append(lines, "") // spacing

	// File picker view
	pickerView := m.filePicker.View()
	lines = append(lines, strings.Split(pickerView, "\n")...)

	// Footer with hints based on mode
	var hint string
	if m.filePickerMode == ModePickerNewFile {
		// Show filename being typed with cursor
		hint = fmt.Sprintf("  Filename: %s|   [Enter] save  [Esc] back", m.newFileName)
	} else {
		hint = "  [up/down] navigate  [Enter] open/select  [n] new file  [Esc] cancel"
	}
	footerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Italic(true)
	lines = append(lines, "")
	lines = append(lines, footerStyle.Render(hint))

	content := strings.Join(lines, "\n")

	// Wrap in a box
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Background(lipgloss.Color("235")).
		Padding(1, 2)

	return boxStyle.Render(content)
}

// formatFunctionParamHint looks up a function's parameter specs and formats
// a helpful hint showing examples for each parameter type.
// Returns empty string if the function has no parameter specs.
func formatFunctionParamHint(funcName string) string {
	spec := semantic.GetFunctionSpec(funcName)
	if spec == nil || len(spec.Params) == 0 {
		return ""
	}

	// Format: "param1: example | param2: example"
	var parts []string
	for _, param := range spec.Params {
		example := ""
		if len(param.Examples) > 0 {
			// Show first example as it's usually most representative
			example = param.Examples[0]
		} else {
			// Fall back to type examples
			typeExamples := semantic.GetExamplesForType(param.Type)
			if len(typeExamples) > 0 {
				example = typeExamples[0]
			}
		}

		if example != "" {
			parts = append(parts, param.Name+": "+example)
		}
	}

	if len(parts) == 0 {
		return ""
	}

	return strings.Join(parts, " | ")
}
