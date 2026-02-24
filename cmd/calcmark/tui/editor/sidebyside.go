package editor

import (
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/CalcMark/go-calcmark/cmd/calcmark/config/theme"
)

// SideBySide renders two panes side-by-side with a vertical divider and guaranteed
// full-width backgrounds. The divider occupies 1 character between the panes.
//
// Key guarantees:
// - Every output line is exactly (leftWidth + 1 + rightWidth) characters wide
// - All characters have background styling (no unstyled gaps)
// - Line counts are balanced (both panes have same number of lines)
type SideBySide struct {
	leftWidth  int
	rightWidth int
	leftBg     color.Color
	rightBg    color.Color
}

// NewSideBySide creates a new side-by-side renderer.
func NewSideBySide(leftWidth, rightWidth int, leftBg, rightBg color.Color) *SideBySide {
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
	maxLines := max(len(leftLines), len(rightLines))

	// Pad to match line counts
	for len(leftLines) < maxLines {
		leftLines = append(leftLines, "")
	}
	for len(rightLines) < maxLines {
		rightLines = append(rightLines, "")
	}

	// Style for the vertical divider between panes
	dividerStyle := lipgloss.NewStyle().
		Foreground(theme.DividerFg).
		Background(s.leftBg)

	// Build output line by line
	var result strings.Builder
	for i := range maxLines {
		if i > 0 {
			result.WriteString("\n")
		}

		// Pad left line to exact width with left background
		leftPadded := s.padLine(leftLines[i], s.leftWidth, s.leftBg)
		result.WriteString(leftPadded)

		// Render vertical divider between panes
		result.WriteString(dividerStyle.Render("│"))

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
func (s *SideBySide) padLine(line string, width int, bg color.Color) string {
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

// TotalWidth returns the total width of the rendered output (left + divider + right).
func (s *SideBySide) TotalWidth() int {
	return s.leftWidth + 1 + s.rightWidth
}
