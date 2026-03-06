package editor

import (
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/CalcMark/go-calcmark/cmd/calcmark/tui/components"
	"github.com/CalcMark/go-calcmark/cmd/calcmark/tui/geometry"
)

// padToWidth pads a string to exactly width visual columns (no truncation).
// Uses lipgloss.Width for correct unicode handling.
// Pads with styled spaces using the given background color to prevent terminal bleed-through.
func padToWidth(s string, width int, bg color.Color) string {
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
func ensureFullWidth(line string, width int, bg color.Color) string {
	currentWidth := lipgloss.Width(line)
	if currentWidth >= width {
		return line
	}
	padding := width - currentWidth
	// Use centralized StyledPadding utility
	return line + components.StyledPadding(padding, bg)
}

// ensureLinesAreFullWidth ensures every line in a multi-line string is exactly the target width.
func ensureLinesAreFullWidth(content string, width int, bg color.Color) string {
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		lines[i] = ensureFullWidth(line, width, bg)
	}
	return strings.Join(lines, "\n")
}

// overlayPadLine pads content to exactly targetWidth visual columns and applies
// a background style. Used by all modal overlays (help, export, command menu,
// file picker) to ensure consistent line widths inside bordered panels.
//
// Handles ANSI escape codes correctly:
//   - Uses lipgloss.Width() to measure visual width (ignores escape sequences)
//   - Strips ANSI reset codes to prevent them from clearing the applied background
//   - Wraps content + padding together so background covers the full width
func overlayPadLine(content string, targetWidth int, bg color.Color) string {
	// Strip reset codes so they don't clear our background
	content = stripResetCodes(content)

	visualWidth := lipgloss.Width(content)
	if visualWidth < targetWidth {
		content += strings.Repeat(" ", targetWidth-visualWidth)
	}

	return lipgloss.NewStyle().Background(bg).Render(content)
}

// wrapStyledLine wraps a line containing ANSI escape codes using visual width.
// This is needed for styled content where len(string) != visual width.
//
// ANSI escape codes are preserved across wrap boundaries: each continuation
// line is prepended with the accumulated ANSI state (non-reset codes) from
// prior segments, matching the overlayStringAt pattern for state tracking.
func wrapStyledLine(line string, maxWidth int) []string {
	if maxWidth <= 0 {
		return []string{line}
	}

	visualWidth := lipgloss.Width(line)
	if visualWidth <= maxWidth {
		return []string{line}
	}

	// Determine wrap points from plain text.
	plainText := stripANSI(line)
	wrappedPlain := geometry.WrapText(plainText, maxWidth)
	if len(wrappedPlain) <= 1 {
		return []string{line}
	}

	// Walk the styled string rune-by-rune, splitting at the same boundaries
	// that WrapText computed. Track ANSI state so each continuation line
	// inherits the active foreground/background codes.
	styledRunes := []rune(line)
	styledIdx := 0

	// ansiState accumulates non-reset ANSI codes; cleared on reset.
	var ansiState []rune

	result := make([]string, 0, len(wrappedPlain))

	for _, segment := range wrappedPlain {
		segmentRunes := []rune(segment)
		var buf strings.Builder

		// Replay accumulated ANSI state on continuation lines.
		if len(result) > 0 && len(ansiState) > 0 {
			buf.WriteString(string(ansiState))
		}

		// Consume styled runes matching this plain-text segment.
		plainIdx := 0
		for styledIdx < len(styledRunes) && plainIdx < len(segmentRunes) {
			r := styledRunes[styledIdx]

			if r == '\x1b' {
				// Collect the full ANSI escape sequence.
				esc := collectEscape(styledRunes, &styledIdx)
				buf.WriteString(string(esc))
				trackANSIState(&ansiState, esc)
			} else {
				buf.WriteRune(r)
				plainIdx++
				styledIdx++
			}
		}

		// Consume trailing ANSI codes between segments (e.g., reset after
		// the last visible char before the wrap boundary).
		for styledIdx < len(styledRunes) && styledRunes[styledIdx] == '\x1b' {
			esc := collectEscape(styledRunes, &styledIdx)
			buf.WriteString(string(esc))
			trackANSIState(&ansiState, esc)
		}

		result = append(result, buf.String())
	}

	return result
}

// collectEscape reads a complete ANSI escape sequence starting at runes[*idx]
// (which must be '\x1b') and advances *idx past the terminal 'm'.
func collectEscape(runes []rune, idx *int) []rune {
	var esc []rune
	esc = append(esc, runes[*idx])
	*idx++
	for *idx < len(runes) {
		esc = append(esc, runes[*idx])
		if runes[*idx] == 'm' {
			*idx++
			break
		}
		*idx++
	}
	return esc
}

// trackANSIState updates the accumulated ANSI state slice. Reset codes
// (\x1b[0m, \x1b[m) clear the state; all other SGR codes accumulate.
func trackANSIState(state *[]rune, esc []rune) {
	s := string(esc)
	if s == "\x1b[0m" || s == "\x1b[m" {
		*state = (*state)[:0]
	} else {
		*state = append(*state, esc...)
	}
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
// Preserves ANSI state from the base line so that text after the overlay
// retains its original styling (backgrounds, colors).
//
// The base line may use either multi-segment ANSI (separate style per component)
// or single-envelope ANSI (one background wrapping the entire line, as produced
// by SideBySide.padLine()). Both cases are handled correctly by tracking ALL
// ANSI sequences seen while scanning the base, then replaying them after the
// overlay's reset to re-establish the base line's styling context.
func overlayStringAt(base, overlay string, col int) string {
	baseRunes := []rune(base)
	overlayRunes := []rune(overlay)

	// Collect all non-reset ANSI escape sequences from the base as we scan.
	// After the overlay, we replay these to restore the base's styling context.
	// This handles the single-envelope case where stripResetCodes() removed all
	// internal resets, leaving only the opening \x1b[48;2;R;G;Bm at the start.
	var baseANSIState []rune

	var result []rune

	// Copy base characters up to col, tracking ANSI state
	visualCol := 0
	baseIdx := 0
	for baseIdx < len(baseRunes) && visualCol < col {
		r := baseRunes[baseIdx]
		result = append(result, r)
		if r == '\x1b' {
			// Collect this escape sequence for state tracking
			var esc []rune
			esc = append(esc, r)
			for baseIdx < len(baseRunes)-1 && baseRunes[baseIdx] != 'm' {
				baseIdx++
				result = append(result, baseRunes[baseIdx])
				esc = append(esc, baseRunes[baseIdx])
			}
			escStr := string(esc)
			if escStr != "\x1b[0m" && escStr != "\x1b[m" {
				baseANSIState = append(baseANSIState, esc...)
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

	// Reset after overlay to prevent overlay styles from bleeding into base text
	result = append(result, []rune("\x1b[0m")...)

	// Skip the overlaid portion of base using VISUAL width of overlay.
	// Continue collecting ANSI state from the skipped region.
	overlayVisualWidth := lipgloss.Width(overlay)
	for baseIdx < len(baseRunes) && overlayVisualWidth > 0 {
		r := baseRunes[baseIdx]
		if r == '\x1b' {
			var esc []rune
			for baseIdx < len(baseRunes) {
				esc = append(esc, baseRunes[baseIdx])
				if baseRunes[baseIdx] == 'm' {
					baseIdx++
					break
				}
				baseIdx++
			}
			escStr := string(esc)
			if escStr == "\x1b[0m" || escStr == "\x1b[m" {
				// Reset clears all state
				baseANSIState = baseANSIState[:0]
			} else {
				baseANSIState = append(baseANSIState, esc...)
			}
		} else {
			overlayVisualWidth--
			baseIdx++
		}
	}

	// Replay accumulated ANSI state to restore the base line's styling.
	// This covers both cases:
	// - Single-envelope: the opening \x1b[48;2;...m was captured in the first loop
	// - Multi-segment: all non-reset codes from both loops are accumulated
	result = append(result, baseANSIState...)

	// Append rest of base
	if baseIdx < len(baseRunes) {
		result = append(result, baseRunes[baseIdx:]...)
	}

	return string(result)
}
