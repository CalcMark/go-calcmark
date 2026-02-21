package editor

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

// TestOverlayStringAt_VisualWidth verifies that overlayStringAt uses visual width,
// not byte/rune length, when determining how much of the base to skip.
// This is critical for overlays with ANSI escape codes (colors, backgrounds).
func TestOverlayStringAt_VisualWidth(t *testing.T) {
	// Create a styled overlay with explicit ANSI codes
	// Visual width: 10, but rune count is much higher due to ANSI codes
	// Format: \x1b[38;2;r;g;bm (foreground) + text + \x1b[0m (reset)
	styledOverlay := "\x1b[38;2;255;255;255m\x1b[48;2;30;30;30m0123456789\x1b[0m"

	// Verify the styled overlay has ANSI codes (more runes than visual width)
	if len(styledOverlay) <= 10 {
		t.Fatalf("Expected styled overlay to have ANSI codes, got len=%d", len(styledOverlay))
	}
	visualWidth := lipgloss.Width(styledOverlay)
	if visualWidth != 10 {
		t.Fatalf("Expected visual width 10, got %d", visualWidth)
	}

	// Create a base line: "AAAAAAAAAA____BBBBBBBBBB" (10 A's, 4 underscores, 10 B's)
	baseLine := strings.Repeat("A", 10) + "____" + strings.Repeat("B", 10)

	// Overlay the styled text at column 10 (replacing the "____BBBBBB" part)
	result := overlayStringAt(baseLine, styledOverlay, 10)

	// The result should have:
	// - First 10 A's unchanged
	// - The styled overlay (with ANSI codes)
	// - The last 4 B's after the overlay

	// Key test: The B's should still be present after the overlay
	// If we incorrectly use rune count instead of visual width, we'd skip too many chars
	// and lose all the B's
	if !strings.Contains(result, "BBBB") {
		t.Errorf("Expected trailing 'BBBB' to remain after overlay\nBase: %q\nResult: %q", baseLine, result)
	}

	// Verify the A's are at the start
	if !strings.HasPrefix(result, "AAAAAAAAAA") {
		t.Errorf("Expected leading 'AAAAAAAAAA' to remain\nResult: %q", result)
	}
}

// TestOverlayStringAt_NoAnsi verifies basic overlay behavior without ANSI codes.
func TestOverlayStringAt_NoAnsi(t *testing.T) {
	base := "Hello, World!"
	overlay := "XXX"

	result := overlayStringAt(base, overlay, 7)

	// Should be "Hello, XXX" + ANSI reset + "ld!"
	// The ANSI reset (\x1b[0m) is added to prevent background bleeding
	expected := "Hello, XXX\x1b[0mld!"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}

	// Also verify visual width is correct
	if lipgloss.Width(result) != lipgloss.Width(base) {
		t.Errorf("Visual width changed: base=%d, result=%d",
			lipgloss.Width(base), lipgloss.Width(result))
	}
}

// TestOverlayStringAt_AtStart verifies overlay at column 0.
func TestOverlayStringAt_AtStart(t *testing.T) {
	base := "Original Text"
	overlay := "NEW"

	result := overlayStringAt(base, overlay, 0)

	// Should be "NEW" + ANSI reset + "ginal Text"
	expected := "NEW\x1b[0mginal Text"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

// TestOverlayStringAt_BeyondBase verifies overlay when col exceeds base length.
func TestOverlayStringAt_BeyondBase(t *testing.T) {
	base := "Short"
	overlay := "XXX"

	result := overlayStringAt(base, overlay, 10)

	// Should pad with spaces to reach col 10, then add overlay + ANSI reset
	expected := "Short     XXX\x1b[0m"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

// TestOverlayPopupOnLines verifies that popup overlay on multiple lines works correctly.
func TestOverlayPopupOnLines(t *testing.T) {
	// Create some base lines
	lines := []string{
		"Line 0: Header",
		"Line 1: Content here",
		"Line 2: More content",
		"Line 3: Even more",
		"Line 4: Footer",
	}

	// Create a small styled popup (2 lines)
	style := lipgloss.NewStyle().Background(lipgloss.Color("#333"))
	popup := style.Render("Popup1") + "\n" + style.Render("Popup2")

	// Overlay at row 2, col 5
	result := overlayPopupOnLines(lines, popup, 2, 5)

	// Original lines 0, 1, 4 should be unchanged
	if result[0] != lines[0] {
		t.Errorf("Line 0 should be unchanged")
	}
	if result[1] != lines[1] {
		t.Errorf("Line 1 should be unchanged")
	}
	if result[4] != lines[4] {
		t.Errorf("Line 4 should be unchanged")
	}

	// Lines 2 and 3 should have overlays
	if result[2] == lines[2] {
		t.Errorf("Line 2 should have popup overlay")
	}
	if result[3] == lines[3] {
		t.Errorf("Line 3 should have popup overlay")
	}

	// The prefix "Line " (5 chars) should still be at the start of overlaid lines
	if !strings.HasPrefix(result[2], "Line ") {
		t.Errorf("Expected line 2 to start with 'Line ', got %q", result[2])
	}
}
