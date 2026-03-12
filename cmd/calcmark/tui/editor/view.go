package editor

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/CalcMark/go-calcmark/cmd/calcmark/config/theme"
	"github.com/CalcMark/go-calcmark/cmd/calcmark/tui/components"
	"github.com/CalcMark/go-calcmark/cmd/calcmark/tui/geometry"
	"github.com/CalcMark/go-calcmark/spec/document"
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
func (m Model) View() tea.View {
	if m.quitting {
		return tea.NewView("Goodbye!\n")
	}

	// Modal overlays - render centered on screen with consistent background
	switch m.mode {
	case StateHelp:
		helpView := m.renderHelpOverlay()
		return tea.NewView(lipgloss.Place(m.width, m.height,
			lipgloss.Center, lipgloss.Center,
			helpView,
			lipgloss.WithWhitespaceChars(" "),
			lipgloss.WithWhitespaceStyle(lipgloss.NewStyle().Foreground(theme.OverlayWhitespaceFg).Background(theme.OverlayWhitespaceFg)),
		))

	case StateCommandMenu:
		menuPopup := m.renderCommandMenuPopup()
		return tea.NewView(lipgloss.Place(m.width, m.height,
			lipgloss.Center, lipgloss.Center,
			menuPopup,
			lipgloss.WithWhitespaceChars(" "),
			lipgloss.WithWhitespaceStyle(lipgloss.NewStyle().Foreground(theme.OverlayWhitespaceFg).Background(theme.OverlayWhitespaceFg)),
		))

	case StateFilePicker:
		pickerOverlay := m.renderFilePickerOverlay()
		return tea.NewView(lipgloss.Place(m.width, m.height,
			lipgloss.Center, lipgloss.Center,
			pickerOverlay,
			lipgloss.WithWhitespaceChars(" "),
			lipgloss.WithWhitespaceStyle(lipgloss.NewStyle().Foreground(theme.OverlayWhitespaceFg).Background(theme.OverlayWhitespaceFg)),
		))

	case StateExport:
		exportOverlay := m.renderExportOverlay()
		return tea.NewView(lipgloss.Place(m.width, m.height,
			lipgloss.Center, lipgloss.Center,
			exportOverlay,
			lipgloss.WithWhitespaceChars(" "),
			lipgloss.WithWhitespaceStyle(lipgloss.NewStyle().Foreground(theme.OverlayWhitespaceFg).Background(theme.OverlayWhitespaceFg)),
		))

	case StateShareTo:
		shareOverlay := m.renderShareOverlay()
		return tea.NewView(lipgloss.Place(m.width, m.height,
			lipgloss.Center, lipgloss.Center,
			shareOverlay,
			lipgloss.WithWhitespaceChars(" "),
			lipgloss.WithWhitespaceStyle(lipgloss.NewStyle().Foreground(theme.OverlayWhitespaceFg).Background(theme.OverlayWhitespaceFg)),
		))

	case StateOpenFrom:
		openFromOverlay := m.renderOpenFromOverlay()
		return tea.NewView(lipgloss.Place(m.width, m.height,
			lipgloss.Center, lipgloss.Center,
			openFromOverlay,
			lipgloss.WithWhitespaceChars(" "),
			lipgloss.WithWhitespaceStyle(lipgloss.NewStyle().Foreground(theme.OverlayWhitespaceFg).Background(theme.OverlayWhitespaceFg)),
		))
	}

	// Calculate layout
	totalWidth := m.width
	totalHeight := m.height

	// Compute line results ONCE per frame and pass to all sub-renderers.
	// Must happen before height calculation because footer height depends on
	// whether the cursor line has an error.
	results := m.GetLineResults()

	// Determine context footer height: 2 lines normally, up to 4 on error lines.
	// Must be computed before contentHeight since it affects the height budget.
	footerHeight := m.contextFooterHeight(results)

	// Reserve space: status bar + context footer + separator (1) + empty line (1)
	contentHeight := max(totalHeight-components.StatusBarHeight-footerHeight-2, 5)

	// Calculate pane widths based on preview mode using centralized configuration
	leftWidth, rightWidth := m.GetPaneWidths(totalWidth)

	// Account for divider between panes (1 character) — only when both panes are visible
	const dividerWidth = 1
	showSource := m.ShowSource()
	showPreview := m.previewMode != PreviewHidden
	hasDivider := showSource && showPreview
	leftContentWidth := leftWidth
	if hasDivider {
		leftContentWidth = leftWidth - dividerWidth // Content area width (excludes divider)
	}

	// In Reading mode, reserve a small left margin for visual breathing room.
	// The margin is added as a gutter prefix during rendering in renderPreviewPaneAligned.
	if m.previewMode == PreviewReading {
		rightWidth -= readingMarginWidth
	}

	// Pane content height (minus header row)
	paneContentHeight := max(contentHeight-1, 3)

	// CRITICAL: Compute aligned line structure ONCE to avoid cycles.
	// Both widths are fixed, and we compute wrapping/padding based on them.
	// This prevents: preview reflows → padding changes → width changes → reflow...
	aligned := m.computeAlignedPanes(leftContentWidth, rightWidth, results)

	// Render source pane with header (if visible)
	var sourcePane string
	if showSource {
		sourceHeader := lipgloss.NewStyle().
			Bold(true).
			Foreground(theme.TextBright).
			Background(m.sourcePaneBg()).
			Padding(0, 1).
			Width(leftContentWidth).
			Render("Source")
		sourceHeader = ensureFullWidth(sourceHeader, leftContentWidth, m.sourcePaneBg())

		sourceContentHeight := max(paneContentHeight, 1)
		sourceContent := m.renderSourcePaneAligned(leftContentWidth, sourceContentHeight, aligned)

		var sourcePaneLines []string
		sourcePaneLines = append(sourcePaneLines, sourceHeader)
		sourcePaneLines = append(sourcePaneLines, strings.Split(sourceContent, "\n")...)
		sourcePane = strings.Join(sourcePaneLines, "\n")
	}

	// Render preview pane (if visible)
	var previewPane string
	if showPreview {
		headerTitle := "Results"
		switch m.previewMode {
		case PreviewRendered:
			headerTitle = "Side-by-Side"
		case PreviewReading:
			headerTitle = "Reading Mode"
		}
		previewHeader := lipgloss.NewStyle().
			Bold(true).
			Foreground(theme.TextBright).
			Background(m.previewPaneBg()).
			Padding(0, 1).
			Width(rightWidth).
			Render(headerTitle)
		previewHeader = ensureFullWidth(previewHeader, rightWidth, m.previewPaneBg())

		// In Reading mode, prepend left margin gutter to the header too.
		if m.previewMode == PreviewReading {
			gutter := lipgloss.NewStyle().Background(m.previewPaneBg()).Render(strings.Repeat(" ", readingMarginWidth))
			previewHeader = gutter + previewHeader
		}

		previewContent := m.renderPreviewPaneAligned(rightWidth, paneContentHeight, aligned)

		var previewPaneLines []string
		previewPaneLines = append(previewPaneLines, previewHeader)
		previewPaneLines = append(previewPaneLines, strings.Split(previewContent, "\n")...)
		previewPane = strings.Join(previewPaneLines, "\n")
	}

	// Build complete UI as array of lines to avoid bare newlines
	var allUILines []string

	// Add panes (source and/or preview)
	if showSource && showPreview {
		// Side-by-side: source + divider + preview
		sbs := NewSideBySide(leftContentWidth, rightWidth, m.sourcePaneBg(), m.previewPaneBg())
		panesOutput := sbs.Render(sourcePane, previewPane)
		allUILines = append(allUILines, strings.Split(panesOutput, "\n")...)
	} else if showSource {
		// Source only (PreviewHidden)
		sourcePane = ensureLinesAreFullWidth(sourcePane, leftWidth, m.sourcePaneBg())
		allUILines = append(allUILines, strings.Split(sourcePane, "\n")...)
	} else {
		// Preview only (PreviewReading)
		previewPane = ensureLinesAreFullWidth(previewPane, rightWidth, m.previewPaneBg())
		allUILines = append(allUILines, strings.Split(previewPane, "\n")...)
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
	contextFooter := m.renderContextFooter(totalWidth, results, footerHeight)
	// The context footer is already multiple lines, so we need to handle each line
	contextFooterLines := strings.Split(contextFooter, "\n")
	for i, line := range contextFooterLines {
		contextFooterLines[i] = ensureFullWidth(line, totalWidth, contextFooterBg)
	}
	allUILines = append(allUILines, contextFooterLines...)

	// Separator - use context footer background for consistency
	separatorStyle := lipgloss.NewStyle().
		Foreground(theme.DividerFg).
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
	return tea.NewView(strings.Join(allUILines, "\n"))
}

// computeAlignedPanes computes both pane line structures once with fixed widths.
// This is the single source of truth for alignment, preventing reflow cycles.
// It uses the cached AlignedModel and converts to the legacy format for rendering.
func (m Model) computeAlignedPanes(sourceWidth, previewWidth int, results []LineResult) alignedPanes {
	// Use the cached AlignedModel - this is the canonical computation
	// Note: We need a mutable reference to update the cache, but View() receives
	// a value copy. For now, we recompute each time in View() but the AlignedModel
	// computation is still the single source of truth.
	aligned := m.computeAlignedModelFresh(sourceWidth, previewWidth, results)

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
			isCalc:        al.IsCalc,
		}
	}

	previewLines := make([]previewLine, len(aligned.PreviewLines))
	for i, al := range aligned.PreviewLines {
		previewLines[i] = previewLine{
			content:       al.Content,
			sourceLineNum: al.SourceLineIdx,
			blockID:       al.BlockID,
			isCalc:        al.IsCalc,
			isFrontmatter: al.BlockID == "" && !al.IsCalc,
			isPadding:     al.Kind == AlignedLinePadding,
		}
	}

	return alignedPanes{
		sourceLines:    sourceLines,
		previewLines:   previewLines,
		sourceToVisual: aligned.SourceToVisual,
	}
}

// contextFooterHeight determines the height of the context footer based on
// whether the cursor line has an error. Returns ContextFooterHeight (2) by
// default, expanding up to 4 when the cursor is on an error line with a hint.
// Autocomplete and function-help override errors (lower priority number wins),
// so the footer stays at default height when those are active.
func (m Model) contextFooterHeight(results []LineResult) int {
	// Autocomplete active → keep 2-line footer to maximize popup space.
	// Function help (P0.5) also renders 2 lines of content; if the footer
	// happens to be taller (error on same line), padToHeight fills the extra
	// rows with background — no visual artifacts. Checking function context
	// here would require editBuf to be loaded, adding coupling for marginal gain.
	if m.mode == StateAutocomplete && m.autocompleteState.Visible {
		return components.ContextFooterHeight
	}

	// Check if cursor line has an error (including frontmatter block errors)
	if r := m.effectiveErrorForLine(results); r != nil {
		// Blocked errors stay compact — the user should fix the root cause.
		if r.IsBlocked {
			return components.ContextFooterHeight
		}
		// Compute how many lines the hint needs
		hint := ""
		if r.Diagnostic != nil {
			hint = components.GetHintForDiagnostic(r.Diagnostic)
		} else {
			errInfo := components.ParseErrorForDisplay(r.Error)
			hint = errInfo.Hint
		}
		if hint != "" {
			// Line 1: error message, Lines 2+: hint (word-wrapped).
			// Cap at 4 lines total.
			return min(4, 2+countWrappedLines(hint, m.width-4))
		}
		// Error with no hint: stay at 2
	}
	return components.ContextFooterHeight
}

// effectiveErrorForLine returns the LineResult carrying the error that applies
// to the cursor line. For most lines this is the line itself. For frontmatter
// lines, the error is only attached to the closing --- delimiter, but it applies
// to the whole block — so when the cursor is on any frontmatter line, we find
// and return the closing line's result. Returns nil if there is no error.
func (m Model) effectiveErrorForLine(results []LineResult) *LineResult {
	if m.cursorLine >= len(results) {
		return nil
	}
	r := results[m.cursorLine]
	if r.Error != "" {
		return &r
	}
	// For frontmatter lines without a direct error, look for the block error
	// on the closing --- line.
	if r.IsFrontmatter && m.frontmatterErr != nil {
		for i := len(results) - 1; i >= 0; i-- {
			if results[i].IsFrontmatter && results[i].Error != "" {
				return &results[i]
			}
		}
	}
	return nil
}

// countWrappedLines returns how many visual lines a string takes at the given width.
func countWrappedLines(s string, width int) int {
	if width <= 0 || s == "" {
		return 1
	}
	lines := 1
	col := 0
	for _, r := range s {
		if r == '\n' {
			lines++
			col = 0
			continue
		}
		col++
		if col >= width {
			lines++
			col = 0
		}
	}
	return lines
}

// computeAlignedModelFresh computes a fresh AlignedModel without caching.
// Used by computeAlignedPanes since View() receives a value copy of Model.
func (m Model) computeAlignedModelFresh(sourceWidth, previewWidth int, results []LineResult) AlignedModel {
	// Calculate content width for source pane (accounting for line numbers).
	// In Reading mode (source hidden), use preview width so alignment doesn't
	// create gaps from narrow source wrapping vs wide preview content.
	lineNumWidth := 4
	var sourceContentWidth int
	if m.previewMode == PreviewReading {
		sourceContentWidth = max(previewWidth-2, 10)
	} else {
		sourceContentWidth = max(sourceWidth-lineNumWidth-2, 10)
	}

	input := AlignedModelInput{
		Lines:              m.GetLines(),
		Results:            results,
		SourceContentWidth: sourceContentWidth,
		PreviewWidth:       previewWidth,
		CursorLine:         m.cursorLine,
		PreviewMode:        m.previewMode,
	}

	// When the user is actively typing, the edit buffer may differ from the
	// committed document line. Feed it into alignment so pre-computed wrapping
	// uses the live text, matching what the source pane renders.
	if m.editBufLoaded {
		input.EditBuf = m.editBuf
		input.EditBufLine = m.cursorLine
	}

	// In PreviewRendered/PreviewReading mode, pre-render visible TextBlocks through the cache.
	// The rendered content is passed as a parallel data structure alongside LineResults,
	// keeping GetLineResults() completely untouched.
	if (m.previewMode == PreviewRendered || m.previewMode == PreviewReading) && m.renderCache != nil && m.doc != nil {
		input.RenderedTextBlocks = m.renderTextBlocks(previewWidth)
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

// renderTextBlocks builds a map of pre-rendered TextBlock content from the cache.
// Iterates blocks in document order to compute line offsets (same approach as
// GetLineResults). Only TextBlocks are rendered; CalcBlocks are skipped.
func (m Model) renderTextBlocks(previewWidth int) map[string][]string {
	rendered := make(map[string][]string)
	lineNum := m.frontmatterLineCount()

	for _, node := range m.doc.GetBlocks() {
		switch tb := node.Block.(type) {
		case *document.TextBlock:
			interpolated := tb.InterpolatedSource()
			blockStart := lineNum

			// If the user is editing a line in this TextBlock, splice the editBuf
			// so the preview updates live. Skip in Reading mode — the rendered
			// view should always show fully interpolated content, never raw
			// {{variable}} template syntax from the edit buffer.
			if m.editBufLoaded && m.previewMode != PreviewReading {
				blockEnd := blockStart + len(interpolated)
				if m.cursorLine >= blockStart && m.cursorLine < blockEnd {
					spliced := make([]string, len(interpolated))
					copy(spliced, interpolated)
					spliced[m.cursorLine-blockStart] = m.editBuf
					interpolated = spliced
				}
			}

			rendered[node.ID] = m.renderCache.Render(node.ID, interpolated, previewWidth)
			lineNum += len(tb.Source())

		case *document.CalcBlock:
			lineNum += len(tb.Source())
		}
	}
	return rendered
}

// sourceLine represents a line in the source pane (may be padding or wrapped).
type sourceLine struct {
	content       string // Source text (empty for padding)
	lineNum       int    // Line number (0 = padding/wrapped line, no line number shown)
	isPadding     bool   // True if this is a padding line for preview alignment
	isWrapped     bool   // True if this is a continuation of a wrapped line
	isCursorLine  bool   // True if this is the cursor line
	sourceLineIdx int    // Original source line index (for cursor tracking on wrapped lines)
	isCalc        bool   // True if this line belongs to a CalcBlock
}

// previewLine represents a line in the preview pane with its source mapping.
type previewLine struct {
	content       string // Rendered content for this preview line
	sourceLineNum int    // Which source line this corresponds to (-1 if spanning multiple)
	blockID       string // Block this line belongs to
	isCalc        bool   // Whether this is from a CalcBlock
	isFrontmatter bool   // Whether this is from the YAML frontmatter block
	isPadding     bool   // Whether this is alignment padding (skip in Reading mode)
}
