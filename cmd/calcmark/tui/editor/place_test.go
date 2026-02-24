package editor

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/muesli/termenv"
)

// TestLipglossPlaceStripsStyling checks if lipgloss.Place() strips ANSI codes
func TestLipglossPlaceStripsStyling(t *testing.T) {
	// Set color profile explicitly for this test
	lipgloss.SetColorProfile(termenv.ANSI256)
	// Create content with background styling
	bgStyle := lipgloss.NewStyle().Background(lipgloss.Color("236"))
	styledContent := bgStyle.Render("hello") + bgStyle.Render(strings.Repeat(" ", 10))

	t.Logf("Styled content (raw): %q", styledContent)
	t.Logf("Styled content (width): %d", lipgloss.Width(styledContent))

	// Apply Place()
	placed := lipgloss.Place(
		20, 3,
		lipgloss.Left, lipgloss.Top,
		styledContent,
		lipgloss.WithWhitespaceBackground(lipgloss.Color("236")),
	)

	t.Logf("After Place (raw): %q", placed)

	// Check if background codes are preserved
	expectedBg := "\x1b[48;5;236m"
	if !strings.Contains(placed, expectedBg) {
		t.Error("Place() stripped background styling!")
	}
}

// TestPadToWidthPreservesStyle checks if padToWidth adds styled padding
func TestPadToWidthPreservesStyle(t *testing.T) {
	// Set color profile explicitly for this test
	lipgloss.SetColorProfile(termenv.ANSI256)
	input := "hello"
	result := padToWidth(input, 20, lipgloss.Color("236"))

	t.Logf("Input: %q", input)
	t.Logf("Result (raw): %q", result)
	t.Logf("Result (width): %d", lipgloss.Width(result))

	// Check for background color codes
	expectedBg := "\x1b[48;5;236m"
	if !strings.Contains(result, expectedBg) {
		t.Error("padToWidth() didn't add background styling!")
	}

	// Check width
	if lipgloss.Width(result) != 20 {
		t.Errorf("Width is %d, expected 20", lipgloss.Width(result))
	}
}
