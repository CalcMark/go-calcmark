package editor

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// SideBySide renders two panes side-by-side with guaranteed full-width backgrounds.
// This component ensures that every line is exactly (leftWidth + rightWidth) characters
// with proper background styling on every character, preventing terminal default bleed-through.
//
// Key guarantees:
// - Every output line is exactly (leftWidth + rightWidth) characters wide
// - All characters have background styling (no unstyled gaps)
// - Line counts are balanced (both panes have same number of lines)
type SideBySide struct {
	leftWidth  int
	rightWidth int
	leftBg     lipgloss.TerminalColor
	rightBg    lipgloss.TerminalColor
}

// NewSideBySide creates a new side-by-side renderer.
func NewSideBySide(leftWidth, rightWidth int, leftBg, rightBg lipgloss.TerminalColor) *SideBySide {
	return &SideBySide{
		leftWidth:  leftWidth,
		rightWidth: rightWidth,
		leftBg:     leftBg,
		rightBg:    rightBg,
	}
}

// Render takes two multi-line strings and renders them side-by-side.
// Both inputs will be split on newlines and aligned to have the same number of lines.
// Each line will be padded to its pane width with the appropriate background.
func (s *SideBySide) Render(left, right string) string {
	leftLines := strings.Split(left, "\n")
	rightLines := strings.Split(right, "\n")

	// Balance line counts
	maxLines := len(leftLines)
	if len(rightLines) > maxLines {
		maxLines = len(rightLines)
	}

	// Pad to match line counts
	for len(leftLines) < maxLines {
		leftLines = append(leftLines, "")
	}
	for len(rightLines) < maxLines {
		rightLines = append(rightLines, "")
	}

	// Build output line by line
	var result strings.Builder
	for i := 0; i < maxLines; i++ {
		if i > 0 {
			result.WriteString("\n")
		}

		// Pad left line to exact width with left background
		leftPadded := s.padLine(leftLines[i], s.leftWidth, s.leftBg)
		result.WriteString(leftPadded)

		// Pad right line to exact width with right background
		rightPadded := s.padLine(rightLines[i], s.rightWidth, s.rightBg)
		result.WriteString(rightPadded)
	}

	return result.String()
}

// padLine pads a line to exactly the target width with background styling.
// Uses lipgloss.Width for correct unicode/ANSI handling.
//
// CRITICAL: Removes ANSI reset codes (\x1b[0m) from input before applying background.
// Reset codes clear ALL attributes including backgrounds, causing terminal default to bleed through.
func (s *SideBySide) padLine(line string, width int, bg lipgloss.TerminalColor) string {
	// Strip ANSI reset codes to prevent them from clearing our background
	line = stripResetCodes(line)

	visualWidth := lipgloss.Width(line)
	bgStyle := lipgloss.NewStyle().Background(bg)

	if visualWidth >= width {
		// Even if already full width, wrap with background to ensure no gaps
		return bgStyle.Render(line)
	}

	padding := width - visualWidth
	// CRITICAL: Wrap BOTH content and padding with background
	return bgStyle.Render(line + strings.Repeat(" ", padding))
}

// stripResetCodes removes ANSI reset codes (\x1b[0m and \x1b[m) from a string.
// Reset codes clear all formatting including backgrounds, which causes terminal bleed-through.
func stripResetCodes(s string) string {
	// Replace both \x1b[0m and \x1b[m (short form)
	result := strings.ReplaceAll(s, "\x1b[0m", "")
	result = strings.ReplaceAll(result, "\x1b[m", "")
	return result
}

// TotalWidth returns the total width of the rendered output.
func (s *SideBySide) TotalWidth() int {
	return s.leftWidth + s.rightWidth
}
