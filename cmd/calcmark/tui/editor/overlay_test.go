package editor

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
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

// TestOverlayStringAt_MultiSegmentBase verifies that when an overlay spans
// across multiple ANSI-styled segments in the base line, the remaining text
// after the overlay retains background styling.
// This simulates real TUI lines where gutter, content, and padding are
// separate lipgloss.Render() calls concatenated together.
func TestOverlayStringAt_MultiSegmentBase(t *testing.T) {
	// Build a base line like real TUI output: multiple styled segments
	gutterStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#888888")).
		Background(lipgloss.Color("#1A1A1A"))
	contentStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#2A2520"))
	paddingStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("#2A2520"))

	// "  1 " (4 chars) + "some text" (9 chars) + "     ...     " (27 chars) = 40 chars total
	baseLine := gutterStyle.Render("  1 ") + contentStyle.Render("some text") + paddingStyle.Render(strings.Repeat(" ", 27))

	baseVisualWidth := lipgloss.Width(baseLine)
	if baseVisualWidth != 40 {
		t.Fatalf("Expected base visual width 40, got %d", baseVisualWidth)
	}

	// Overlay a popup border line at column 4 (over the content area), width 30
	overlayStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#AAAAAA")).
		Background(lipgloss.Color("#333333"))
	overlay := overlayStyle.Render(strings.Repeat("─", 30))

	result := overlayStringAt(baseLine, overlay, 4)
	resultVisualWidth := lipgloss.Width(result)

	// Visual width must be preserved
	if resultVisualWidth != baseVisualWidth {
		t.Errorf("Visual width changed: base=%d, result=%d", baseVisualWidth, resultVisualWidth)
	}

	// The remaining characters after the overlay (columns 34-39) must have
	// ANSI background styling, not be raw unstyled text.
	// Find the end of the overlay content in the result
	plain := stripANSI(result)
	// Plain text should be: "  1 " + 30 "─" chars + 6 spaces = 40 visual columns
	plainWidth := lipgloss.Width(plain)
	if plainWidth != 40 {
		t.Errorf("Plain text visual width: got %d, want 40. Plain: %q", plainWidth, plain)
	}

	// The 6 trailing spaces (after the overlay) must be styled, not raw.
	// If they're unstyled, the terminal will show default background (white bleed).
	// Check: the result string after the overlay should contain ANSI codes
	// before any trailing visible characters.

	// Find position of last "─" in the result
	lastDash := strings.LastIndex(result, "─")
	if lastDash < 0 {
		t.Fatal("Could not find overlay content in result")
	}
	afterOverlayInResult := result[lastDash+len("─"):]
	plainAfter := stripANSI(afterOverlayInResult)

	if len(plainAfter) > 0 && !strings.Contains(afterOverlayInResult, "\x1b[") {
		t.Errorf("Text after overlay has no ANSI styling (terminal will show default bg):\nAfter overlay: %q\nPlain: %q",
			afterOverlayInResult, plainAfter)
	}
}

// TestOverlayStringAt_SingleEnvelopeBase verifies that overlayStringAt correctly
// handles the real SideBySide output format where each pane is wrapped in a SINGLE
// ANSI background envelope (all internal resets stripped by stripResetCodes()).
//
// This test uses raw ANSI codes instead of lipgloss.Render() because lipgloss
// strips all ANSI in test environments (no terminal detected). The real terminal
// behavior is what we need to test.
func TestOverlayStringAt_SingleEnvelopeBase(t *testing.T) {
	// Simulate SideBySide.padLine() output: one bg envelope for the whole pane.
	// Format: \x1b[48;2;R;G;Bm (set bg) + content + \x1b[0m (reset)
	// This is what stripResetCodes + bgStyle.Render() produces in a real terminal.
	leftBg := "\x1b[48;2;42;37;32m"  // brown background
	rightBg := "\x1b[48;2;26;26;26m" // dark background
	dividerFg := "\x1b[38;2;85;85;85m"
	reset := "\x1b[0m"

	// Left pane: 40 chars in a single bg envelope
	leftPadded := leftBg + "  1 some text" + strings.Repeat(" ", 27) + reset

	// Divider: 1 char with fg+bg
	divider := dividerFg + leftBg + "│" + reset

	// Right pane: 30 chars in a single bg envelope
	rightPadded := rightBg + "→ 10" + strings.Repeat(" ", 26) + reset

	// Full SideBySide line: left + divider + right
	baseLine := leftPadded + divider + rightPadded
	baseVisualWidth := lipgloss.Width(baseLine)
	expectedWidth := 40 + 1 + 30
	if baseVisualWidth != expectedWidth {
		t.Fatalf("Expected base visual width %d, got %d", expectedWidth, baseVisualWidth)
	}

	// Overlay popup at col 5, width 30 (within left pane, leaves 5 chars + divider + right pane)
	overlayBg := "\x1b[48;2;51;51;51m" // overlay background
	overlayFg := "\x1b[38;2;170;170;170m"
	overlayContent := overlayFg + overlayBg + strings.Repeat("─", 30) + reset

	result := overlayStringAt(baseLine, overlayContent, 5)

	// Visual width must be preserved
	resultVisualWidth := lipgloss.Width(result)
	if resultVisualWidth != baseVisualWidth {
		t.Errorf("Visual width changed: base=%d, result=%d", baseVisualWidth, resultVisualWidth)
	}

	t.Logf("Raw result: %q", result)

	// CRITICAL CHECK: After the overlay (col 5 + width 30 = col 35), the remaining
	// 5 chars of left pane + divider + right pane must have ANSI background styling.

	// Find the divider in the result using strings.Cut
	afterOverlayRegion, afterDivider, found := strings.Cut(result, "│")
	if !found {
		t.Fatal("Could not find divider '│' in result")
	}

	// The region between the overlay's reset and the divider should have a background.
	lastReset := strings.LastIndex(afterOverlayRegion, "\x1b[0m")
	if lastReset >= 0 {
		betweenResetAndDivider := afterOverlayRegion[lastReset+4:]
		plainBetween := stripANSI(betweenResetAndDivider)
		if len(plainBetween) > 0 && !strings.Contains(betweenResetAndDivider, "\x1b[") {
			t.Errorf("BLEED: %d chars between overlay reset and divider have NO ANSI styling.\n"+
				"These chars will show terminal default bg (white bleed).\n"+
				"Raw: %q\nPlain: %q",
				lipgloss.Width(plainBetween), betweenResetAndDivider, plainBetween)
		}
	}

	// Also check that the right pane (after divider) has ANSI styling
	if len(afterDivider) > 0 && !strings.Contains(afterDivider[:min(50, len(afterDivider))], "\x1b[") {
		t.Errorf("Right pane after divider has no ANSI styling.\nRaw: %q", afterDivider[:min(50, len(afterDivider))])
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
