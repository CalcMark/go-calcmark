package editor

import (
	"strings"

	"github.com/CalcMark/go-calcmark/cmd/calcmark/tui/components"
	"github.com/CalcMark/go-calcmark/cmd/calcmark/tui/geometry"
	"github.com/charmbracelet/lipgloss"
)

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

// overlayPadLine pads content to exactly targetWidth visual columns and applies
// a background style. Used by all modal overlays (help, export, command menu,
// file picker) to ensure consistent line widths inside bordered panels.
//
// Handles ANSI escape codes correctly:
//   - Uses lipgloss.Width() to measure visual width (ignores escape sequences)
//   - Strips ANSI reset codes to prevent them from clearing the applied background
//   - Wraps content + padding together so background covers the full width
func overlayPadLine(content string, targetWidth int, bg lipgloss.TerminalColor) string {
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
	// For calc results like "a -> 2", the arrow and value are usually short enough
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
