package editor

import (
	"fmt"
	"image/color"
	"slices"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/CalcMark/go-calcmark/v2/cmd/calcmark/config/theme"
	"github.com/CalcMark/go-calcmark/v2/cmd/calcmark/tui/components"
	"github.com/CalcMark/go-calcmark/v2/spec/document"
)

// readingMarginWidth is the left gutter width in Reading mode for visual breathing room.
const readingMarginWidth = 2

// readingNavState holds the set of source lines visible in Reading mode.
// Updated during renderReadingPane (View) and consulted by navigation (Update)
// to skip invisible lines. Shared via pointer for Bubble Tea value-copy safety.
type readingNavState struct {
	visibleLines []int // sorted source line indices that have visual representation
}

// nearestVisible returns the closest visible source line to target.
// Used by the renderer to project the cursor highlight.
func (s *readingNavState) nearestVisible(target int) int {
	if len(s.visibleLines) == 0 {
		return target
	}
	best := s.visibleLines[0]
	bestDist := best - target
	if bestDist < 0 {
		bestDist = -bestDist
	}
	for _, v := range s.visibleLines[1:] {
		d := v - target
		if d < 0 {
			d = -d
		}
		if d < bestDist {
			best = v
			bestDist = d
		}
	}
	return best
}

// nextVisible returns the first visible source line strictly after current.
// Returns current if already at or past the last visible line.
func (s *readingNavState) nextVisible(current int) int {
	for _, v := range s.visibleLines {
		if v > current {
			return v
		}
	}
	return current
}

// prevVisible returns the last visible source line strictly before current.
// Returns current if already at or before the first visible line.
func (s *readingNavState) prevVisible(current int) int {
	prev := current
	for _, v := range s.visibleLines {
		if v >= current {
			break
		}
		prev = v
	}
	return prev
}

// visualScrollState holds the persisted visual scroll offset between frames.
// Shared via pointer so Bubble Tea's value-copied Model in View() can read/write
// the same state as Update(). This is safe because Bubble Tea guarantees
// single-goroutine execution for Update() and View().
type visualScrollState struct {
	offset int // last visual scroll offset used
}

// resolveVisualLine looks up a source line in sourceToVisual.
// If the exact line isn't mapped (common with block-level alignment),
// it searches nearby lines (±distance) to find the nearest mapped entry.
// Returns 0 only if nothing within ±50 lines is mapped.
func resolveVisualLine(sourceToVisual map[int]int, sourceLine int) int {
	if v, ok := sourceToVisual[sourceLine]; ok {
		return v
	}
	// Search outward from the target line for the nearest mapped entry.
	for delta := 1; delta <= 50; delta++ {
		if v, ok := sourceToVisual[sourceLine-delta]; ok {
			return v
		}
		if v, ok := sourceToVisual[sourceLine+delta]; ok {
			return v
		}
	}
	return 0
}

// computeVisualScroll returns a stable visual scroll offset, only adjusting
// when the cursor would be outside the viewport. The offset is persisted
// between frames via m.visualScroll to prevent viewport jumps.
func (m Model) computeVisualScroll(aligned alignedPanes, visibleLines int) int {
	cursorVisualLine := resolveVisualLine(aligned.sourceToVisual, m.cursorLine)

	// Use persisted offset as starting point.
	offset := m.visualScroll.offset

	// Clamp to valid range.
	maxOffset := max(0, len(aligned.sourceLines)-visibleLines)
	offset = min(offset, maxOffset)

	// Only adjust if cursor is outside the current viewport.
	offset = min(offset, cursorVisualLine)
	if cursorVisualLine >= offset+visibleLines {
		offset = cursorVisualLine - visibleLines + 1
	}

	// Persist for next frame.
	m.visualScroll.offset = offset
	return offset
}

// renderSourcePaneAligned renders the source pane using pre-computed aligned lines.
// This avoids recomputing alignment which could cause cycles.
func (m Model) renderSourcePaneAligned(width, height int, aligned alignedPanes) string {
	sourceLines := aligned.sourceLines
	visibleLines := height

	visualScrollOffset := m.computeVisualScroll(aligned, visibleLines)

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
	fmCount := m.frontmatterLineCount()

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
			// Wrapped continuation line — pass raw text + tint colors to selection renderer.
			// Selection must operate on un-styled text so rune column positions are correct.
			fg, bg := blockTintColors(sl.sourceLineIdx, fmCount, sl.isCalc, srcBg)
			lineWithSelection := m.renderLineWithSelection(sl.sourceLineIdx, sl.content, fg, bg)
			content = padToWidth(lineWithSelection, contentWidth, srcBg)
		} else {
			// Normal source line — pass raw text + tint colors to selection renderer.
			fg, bg := blockTintColors(sl.sourceLineIdx, fmCount, sl.isCalc, srcBg)
			lineWithSelection := m.renderLineWithSelection(sl.lineNum-1, sl.content, fg, bg)
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
func (m Model) sourcePaneBg() color.Color {
	color := m.styles.SourcePane.GetBackground()
	if _, ok := color.(lipgloss.NoColor); ok {
		return theme.SourcePaneBg // Fallback to palette
	}
	return color
}

// previewPaneBg returns the background color for the preview pane.
func (m Model) previewPaneBg() color.Color {
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
	// Reading mode has its own rendering path with independent scroll.
	if m.previewMode == PreviewReading {
		return m.renderReadingPane(width, height, aligned)
	}

	previewLines := aligned.previewLines

	pvBg := m.previewPaneBg()

	resultsHeight := height

	// Build complete lines to avoid bare newlines
	var allLines []string

	// Use the same visual scroll offset as source pane to keep them aligned.
	// computeVisualScroll was already called by renderSourcePaneAligned,
	// so m.visualScroll.offset is up to date.
	visualScrollOffset := m.computeVisualScroll(aligned, resultsHeight)

	// Apply scroll offset and render visible lines
	start := visualScrollOffset
	end := min(start+resultsHeight, len(previewLines))

	// In edit mode, source pane may render different number of lines for cursor line
	// because it renders the live edit buffer. We need to match the pre-computed
	// aligned count so both panes emit the same number of visual lines.
	var preComputedCursorLineCount int
	if m.editBufLoaded {
		// Count how many pre-computed visual lines exist for cursor's source line.
		// This matches the source pane's numAligned count from alignment computation.
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

		// Frontmatter lines: render as styled YAML text in the preview pane.
		// This shows the full frontmatter content (not just globals values)
		// so the preview pane is never blank for frontmatter lines.
		if pl.isFrontmatter {
			var completeLine string
			fmCount := m.frontmatterLineCount()
			sourceLines := m.GetLines()
			if pl.sourceLineNum < len(sourceLines) {
				srcLine := sourceLines[pl.sourceLineNum]

				// Show frontmatter error on the closing --- line
				if m.frontmatterErr != nil && pl.sourceLineNum == fmCount-1 {
					errStyle := lipgloss.NewStyle().
						Foreground(theme.CalcErrorFg).
						Background(pvBg)
					errorText := components.CleanErrorMessage(m.frontmatterErr.Error())
					maxLen := width - 4
					if maxLen > 0 && lipgloss.Width(errorText) > maxLen {
						errorText = components.TruncateWithEllipsis(errorText, maxLen)
					}
					completeLine = ensureFullWidth(errStyle.Render("⚠ "+errorText), width, pvBg)
				} else {
					// Render YAML text with frontmatter styling
					fmStyle := lipgloss.NewStyle().
						Foreground(theme.SourceFrontmatter).
						Background(pvBg)
					completeLine = ensureFullWidth(fmStyle.Render(srcLine), width, pvBg)
				}
			} else {
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
				cursorPreviewLines := []previewLine{}
				for _, cpl := range previewLines {
					if cpl.sourceLineNum == m.cursorLine && !cpl.isPadding {
						cursorPreviewLines = append(cursorPreviewLines, cpl)
					}
				}
				for k := 0; k < preComputedCursorLineCount && linesWritten < resultsHeight; k++ {
					var completeLine string
					if k < len(cursorPreviewLines) {
						completeLine = ensureFullWidth(cursorPreviewLines[k].content, width, pvBg)
					} else {
						completeLine = ensureFullWidth("", width, pvBg)
					}
					allLines = append(allLines, completeLine)
					linesWritten++
				}
				cursorLineProcessed = true
			}
			continue
		}

		content := pl.content
		paddedContent := ensureFullWidth(content, width, pvBg)
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

// readingVisualLine is a pre-processed line for Reading mode rendering.
// Built in a first pass to enable correct scrolling within the filtered line set.
type readingVisualLine struct {
	previewLine        // embedded original line (may be zero-value for separators)
	isSeparator   bool // inserted block-transition separator
	sourceLineNum int  // source line this maps to (-1 for separators)
}

// renderReadingPane renders the preview pane in Reading mode with independent scrolling.
// Reading mode skips alignment padding and empty filler, adds block separators,
// and scrolls within the filtered line set rather than the raw aligned lines.
func (m Model) renderReadingPane(width, height int, aligned alignedPanes) string {
	previewLines := aligned.previewLines
	pvBg := m.previewPaneBg()
	sourceLines := m.GetLines()

	// First pass: build filtered reading lines and source→visual mapping.
	// Rules:
	//  1. Skip alignment padding (isPadding).
	//  2. Skip inter-block blank lines (block separator handles spacing).
	//  3. Keep at most ONE consecutive blank line within a block (collapse duplicates).
	//  4. Insert a block separator on block transitions.
	var filtered []readingVisualLine
	sourceToReading := make(map[int]int) // source line → first reading visual line
	prevBlockID := ""
	lastNonEmptyBlockID := ""
	lastFilteredWasBlank := false // for collapsing consecutive blanks
	for _, pl := range previewLines {
		// Skip alignment padding.
		if pl.isPadding {
			continue
		}
		// Handle empty preview lines.
		if pl.content == "" && !pl.isCalc && !pl.isFrontmatter {
			// Skip inter-block blanks (separator handles spacing).
			if pl.blockID == "" || pl.blockID != lastNonEmptyBlockID {
				continue
			}
			// Collapse consecutive blanks to at most one.
			if lastFilteredWasBlank {
				continue
			}
			lastFilteredWasBlank = true
		} else {
			lastFilteredWasBlank = false
			if pl.blockID != "" {
				lastNonEmptyBlockID = pl.blockID
			}
		}
		// All frontmatter lines are now rendered as YAML text (no skipping).

		// Insert block separator when block changes.
		if pl.blockID != "" && prevBlockID != "" && pl.blockID != prevBlockID && len(filtered) > 0 {
			filtered = append(filtered, readingVisualLine{
				isSeparator:   true,
				sourceLineNum: -1,
			})
			lastFilteredWasBlank = false
		}
		if pl.blockID != "" {
			prevBlockID = pl.blockID
		}

		// Record first visual line for this source line.
		if _, exists := sourceToReading[pl.sourceLineNum]; !exists {
			sourceToReading[pl.sourceLineNum] = len(filtered)
		}

		filtered = append(filtered, readingVisualLine{
			previewLine:   pl,
			sourceLineNum: pl.sourceLineNum,
		})
	}

	// Build the set of "navigable" source lines for Reading mode.
	// Calc lines and frontmatter values are individually navigable.
	// Within a markdown block, blank lines act as section boundaries
	// (heading→paragraph, paragraph→list). Each section's first source
	// line is navigable, so Up/Down jumps per visual section, not per
	// entire multi-section block.
	navigable := make(map[int]bool)
	seenSection := make(map[string]bool) // blockID → already recorded first line in current section
	for _, rl := range filtered {
		if rl.isSeparator {
			continue
		}
		pl := rl.previewLine
		if pl.isCalc || pl.isFrontmatter {
			navigable[pl.sourceLineNum] = true
			continue
		}
		key := pl.blockID
		if key == "" {
			navigable[pl.sourceLineNum] = true
			continue
		}
		// Blank line resets the section boundary — next content line
		// becomes navigable (new heading, paragraph, or list).
		if pl.content == "" {
			delete(seenSection, key)
			continue
		}
		if !seenSection[key] {
			seenSection[key] = true
			navigable[pl.sourceLineNum] = true
		}
	}
	visible := make([]int, 0, len(navigable))
	for srcLine := range navigable {
		visible = append(visible, srcLine)
	}
	slices.Sort(visible)
	m.readingNav.visibleLines = visible

	// Project the cursor onto the nearest visible source line for highlighting.
	// The actual m.cursorLine stays accurate (for status bar, evaluation),
	// but the visual highlight lands on the closest rendered line.
	highlightSourceLine := m.readingNav.nearestVisible(m.cursorLine)

	// Compute scroll offset within the filtered list.
	cursorVisual := resolveVisualLine(sourceToReading, highlightSourceLine)
	offset := m.visualScroll.offset
	maxOffset := max(0, len(filtered)-height)
	offset = min(offset, maxOffset)
	if cursorVisual < offset+scrollMargin {
		offset = max(0, cursorVisual-scrollMargin)
	}
	if cursorVisual >= offset+height-scrollMargin {
		offset = cursorVisual - height + scrollMargin + 1
	}
	offset = min(offset, maxOffset)
	m.visualScroll.offset = offset

	// Second pass: render visible window from filtered list.
	start := offset
	end := min(start+height, len(filtered))
	var allLines []string
	linesWritten := 0

	for i := start; i < end && linesWritten < height; i++ {
		rl := filtered[i]

		if rl.isSeparator {
			allLines = append(allLines, ensureFullWidth("", width, pvBg))
			linesWritten++
			continue
		}

		pl := rl.previewLine
		isCursorLine := pl.sourceLineNum == highlightSourceLine
		lineBg := pvBg
		if isCursorLine {
			lineBg = theme.ReadingCursorBg
		}

		// Frontmatter lines: render as styled YAML text.
		if pl.isFrontmatter {
			var completeLine string
			if pl.sourceLineNum < len(sourceLines) {
				srcLine := sourceLines[pl.sourceLineNum]
				fmStyle := lipgloss.NewStyle().
					Foreground(theme.SourceFrontmatter).
					Background(lineBg)
				completeLine = ensureFullWidth(fmStyle.Render(srcLine), width, lineBg)
			} else {
				completeLine = ensureFullWidth("", width, lineBg)
			}
			allLines = append(allLines, completeLine)
			linesWritten++
			continue
		}

		// Apply background: highlight for cursor, maintain for non-cursor.
		content := pl.content
		if content != "" {
			if isCursorLine {
				content = replaceBackground(content, pvBg, lineBg)
			} else {
				content = maintainBackground(content, lineBg)
			}
		}
		allLines = append(allLines, ensureFullWidth(content, width, lineBg))
		linesWritten++
	}

	// Fill remaining space.
	for i := linesWritten; i < height; i++ {
		allLines = append(allLines, ensureFullWidth("", width, pvBg))
	}

	// Prepend left margin gutter.
	gutter := lipgloss.NewStyle().Background(pvBg).Render(strings.Repeat(" ", readingMarginWidth))
	for i, line := range allLines {
		allLines[i] = gutter + line
	}

	return strings.Join(allLines, "\n")
}

// renderCalcLine renders a single calculation line result.
func (m Model) renderCalcLine(r LineResult, width int) string {
	// Use the detector to check if this line is actually a calculation
	detector := document.NewDetector()
	isActuallyCalc, _ := detector.IsCalculation(r.Source)

	pvBg := m.previewPaneBg()

	if r.Error != "" && (isActuallyCalc || r.Diagnostic != nil) {
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

	// Transform indicators: symbols showing which transforms affected this result.
	// × (U+00D7 MULTIPLICATION SIGN) for scale, • (U+2022 BULLET) for convert_to.
	// Both can appear together as ×• when both transforms applied.
	transformSuffix := ""
	if r.IsScaled {
		scaleStyle := lipgloss.NewStyle().
			Foreground(theme.ScaleIndicator).
			Background(pvBg)
		transformSuffix += scaleStyle.Render("\u00D7") // ×
	}
	if r.IsConverted {
		convertStyle := lipgloss.NewStyle().
			Foreground(theme.ConvertIndicator).
			Background(pvBg)
		transformSuffix += convertStyle.Render("\u2022") // •
	}

	switch m.previewMode {
	case PreviewFull, PreviewRendered, PreviewReading:
		// Full/Rendered mode: "varName → value" for assignments, "→ value" for anonymous calcs
		if r.VarName != "" {
			return changedMarker + m.styles.CalcVarName.Render(r.VarName) + sp + m.styles.CalcArrow.Render("→") + sp + valueStyle.Render(r.Value) + transformSuffix
		}
		// Anonymous calculation (no variable assignment) - show arrow without placeholder
		return changedMarker + m.styles.CalcArrow.Render("→") + sp + valueStyle.Render(r.Value) + transformSuffix

	case PreviewMinimal:
		// Minimal mode: left-aligned "→ value" (with * if changed)
		arrow := "→ "
		return changedMarker + valueStyle.Render(arrow+r.Value) + transformSuffix
	}

	return ""
}

// blockTintColors returns the foreground and background colors for a source line
// based on the block type: frontmatter, calc, or markdown.
func blockTintColors(sourceLineIdx, fmCount int, isCalc bool, bg color.Color) (fg, bgOut color.Color) {
	switch {
	case sourceLineIdx < fmCount:
		return theme.SourceFrontmatter, bg
	case isCalc:
		return theme.SourceCalc, bg
	default:
		return theme.SourceMarkdown, bg
	}
}
