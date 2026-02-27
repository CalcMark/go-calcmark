package editor

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/CalcMark/go-calcmark/cmd/calcmark/config/theme"
	"github.com/CalcMark/go-calcmark/cmd/calcmark/tui/components"
	"github.com/CalcMark/go-calcmark/cmd/calcmark/tui/geometry"
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

	// Reserve space: status bar + context footer (2) + separator (1) + empty line (1)
	contentHeight := max(totalHeight-components.StatusBarHeight-4, 5)

	// Calculate pane widths based on preview mode using centralized configuration
	leftWidth, rightWidth := m.GetPaneWidths(totalWidth)

	// Account for divider between panes (1 character)
	// The divider is visually part of the separation, so we subtract it from left pane
	const dividerWidth = 1
	leftContentWidth := leftWidth - dividerWidth // Content area width (excludes divider)

	// Pane content height (minus header row)
	paneContentHeight := max(contentHeight-1, 3)

	// CRITICAL: Compute aligned line structure ONCE to avoid cycles.
	// Use leftContentWidth (not leftWidth) because divider takes 1 char from left pane
	// Both widths are fixed, and we compute wrapping/padding based on them.
	// This prevents: preview reflows → padding changes → width changes → reflow...
	aligned := m.computeAlignedPanes(leftContentWidth, rightWidth)

	// Render source pane with header
	sourceHeader := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.TextBright).
		Background(m.sourcePaneBg()).
		Padding(0, 1).
		Width(leftContentWidth).
		Render("Source")
	// Ensure header is exactly leftContentWidth
	sourceHeader = ensureFullWidth(sourceHeader, leftContentWidth, m.sourcePaneBg())

	// No source padding needed — globals/exchange rates are only defined in frontmatter,
	// and frontmatter lines provide natural alignment in the preview pane.
	sourceContentHeight := max(paneContentHeight, 1)

	sourceContent := m.renderSourcePaneAligned(leftContentWidth, sourceContentHeight, aligned)

	// Assemble source pane with header
	var sourcePaneLines []string
	sourcePaneLines = append(sourcePaneLines, sourceHeader)
	sourcePaneLines = append(sourcePaneLines, strings.Split(sourceContent, "\n")...)
	sourcePane := strings.Join(sourcePaneLines, "\n")

	// Render preview pane (if visible)
	var previewPane string
	if m.previewMode != PreviewHidden {
		previewHeader := lipgloss.NewStyle().
			Bold(true).
			Foreground(theme.TextBright).
			Background(m.previewPaneBg()).
			Padding(0, 1).
			Width(rightWidth).
			Render("Results")
		// Ensure header is exactly rightWidth
		previewHeader = ensureFullWidth(previewHeader, rightWidth, m.previewPaneBg())

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
		sbs := NewSideBySide(leftContentWidth, rightWidth, m.sourcePaneBg(), m.previewPaneBg())
		panesOutput := sbs.Render(sourcePane, previewPane)
		allUILines = append(allUILines, strings.Split(panesOutput, "\n")...)
	} else {
		// Single pane - use full left width (no divider in single-pane mode)
		sourcePane = ensureLinesAreFullWidth(sourcePane, leftWidth, m.sourcePaneBg())
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
	sourceContentWidth := max(sourceWidth-lineNumWidth-2, 10)

	input := AlignedModelInput{
		Lines:              m.GetLines(),
		Results:            m.GetLineResults(),
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
	isCalc        bool   // True if this line belongs to a CalcBlock
}

// previewLine represents a line in the preview pane with its source mapping.
type previewLine struct {
	content       string // Rendered content for this preview line
	sourceLineNum int    // Which source line this corresponds to (-1 if spanning multiple)
	blockID       string // Block this line belongs to
	isCalc        bool   // Whether this is from a CalcBlock
	isFrontmatter bool   // Whether this is from the YAML frontmatter block
}
