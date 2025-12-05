package editor

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

func TestSideBySide_BasicRendering(t *testing.T) {
	lipgloss.SetColorProfile(termenv.ANSI256)

	sbs := NewSideBySide(40, 40, lipgloss.Color("236"), lipgloss.Color("235"))

	left := "line 1\nline 2"
	right := "preview 1\npreview 2"

	output := sbs.Render(left, right)
	lines := strings.Split(output, "\n")

	// Should have 2 lines (matching input)
	if len(lines) != 2 {
		t.Errorf("Expected 2 lines, got %d", len(lines))
	}

	// Each line should be exactly 80 characters (40 + 40)
	for i, line := range lines {
		width := lipgloss.Width(line)
		if width != 80 {
			t.Errorf("Line %d has width %d, expected 80", i, width)
		}
	}

	// Should contain background styling
	if !strings.Contains(output, "\x1b[48;5;236m") {
		t.Error("Left pane missing background styling (236)")
	}
	if !strings.Contains(output, "\x1b[48;5;235m") {
		t.Error("Right pane missing background styling (235)")
	}
}

func TestSideBySide_UnevenLineCounts(t *testing.T) {
	lipgloss.SetColorProfile(termenv.ANSI256)

	sbs := NewSideBySide(40, 40, lipgloss.Color("236"), lipgloss.Color("235"))

	left := "line 1\nline 2\nline 3"
	right := "preview 1"

	output := sbs.Render(left, right)
	lines := strings.Split(output, "\n")

	// Should have 3 lines (matching longer input)
	if len(lines) != 3 {
		t.Errorf("Expected 3 lines, got %d", len(lines))
	}

	// All lines should be 80 characters
	for i, line := range lines {
		width := lipgloss.Width(line)
		if width != 80 {
			t.Errorf("Line %d has width %d, expected 80", i, width)
		}
	}

	// First line should contain both
	if !strings.Contains(lines[0], "line 1") {
		t.Error("First line missing left content")
	}
	if !strings.Contains(lines[0], "preview 1") {
		t.Error("First line missing right content")
	}

	// Lines 2 and 3 should have left content but right should be empty (padded)
	if !strings.Contains(lines[1], "line 2") {
		t.Error("Second line missing left content")
	}
	if !strings.Contains(lines[2], "line 3") {
		t.Error("Third line missing left content")
	}
}

func TestSideBySide_EmptyInput(t *testing.T) {
	lipgloss.SetColorProfile(termenv.ANSI256)

	sbs := NewSideBySide(40, 40, lipgloss.Color("236"), lipgloss.Color("235"))

	output := sbs.Render("", "")
	lines := strings.Split(output, "\n")

	// Should have 1 line (empty inputs produce 1 empty line each)
	if len(lines) != 1 {
		t.Errorf("Expected 1 line, got %d", len(lines))
	}

	// Line should be exactly 80 characters of styled spaces
	width := lipgloss.Width(lines[0])
	if width != 80 {
		t.Errorf("Line has width %d, expected 80", width)
	}

	// Should have background styling
	if !strings.Contains(lines[0], "\x1b[48;5;") {
		t.Error("Empty line missing background styling")
	}
}

func TestSideBySide_StyledContent(t *testing.T) {
	lipgloss.SetColorProfile(termenv.ANSI256)

	sbs := NewSideBySide(40, 40, lipgloss.Color("236"), lipgloss.Color("235"))

	// Create styled content
	leftStyled := lipgloss.NewStyle().Foreground(lipgloss.Color("200")).Render("styled left")
	rightStyled := lipgloss.NewStyle().Foreground(lipgloss.Color("100")).Render("styled right")

	output := sbs.Render(leftStyled, rightStyled)
	lines := strings.Split(output, "\n")

	// Should preserve content styling
	if !strings.Contains(output, "styled left") {
		t.Error("Left styled content missing")
	}
	if !strings.Contains(output, "styled right") {
		t.Error("Right styled content missing")
	}

	// Should have background styling
	if !strings.Contains(output, "\x1b[48;5;236m") {
		t.Error("Left background missing")
	}
	if !strings.Contains(output, "\x1b[48;5;235m") {
		t.Error("Right background missing")
	}

	// Width should still be exact
	width := lipgloss.Width(lines[0])
	if width != 80 {
		t.Errorf("Line has width %d, expected 80", width)
	}
}

func TestSideBySide_NoGaps(t *testing.T) {
	lipgloss.SetColorProfile(termenv.ANSI256)

	sbs := NewSideBySide(40, 40, lipgloss.Color("236"), lipgloss.Color("235"))

	left := "short"
	right := "short"

	output := sbs.Render(left, right)
	lines := strings.Split(output, "\n")

	// Line should be exactly 80 characters
	width := lipgloss.Width(lines[0])
	if width != 80 {
		t.Errorf("Line has width %d, expected 80", width)
	}

	// The raw string should contain ONLY styled spaces after content
	// (no unstyled spaces that would show terminal default)
	raw := lines[0]

	// Count how many characters are in the styled portions
	// This is a sanity check that we're not leaving unstyled gaps
	if !strings.Contains(raw, "\x1b[48;5;236m") {
		t.Error("Missing left pane background styling")
	}
	if !strings.Contains(raw, "\x1b[48;5;235m") {
		t.Error("Missing right pane background styling")
	}

	// The padding spaces should have background codes
	// We expect the pattern: content + \x1b[48;5;XXXm + spaces + \x1b[0m
	if !strings.Contains(raw, "\x1b[48;5;236m") || !strings.Contains(raw, "\x1b[48;5;235m") {
		t.Error("Padding spaces missing background styling")
	}
}

func TestSideBySide_TotalWidth(t *testing.T) {
	sbs := NewSideBySide(30, 50, lipgloss.Color("236"), lipgloss.Color("235"))

	if sbs.TotalWidth() != 80 {
		t.Errorf("TotalWidth() = %d, expected 80", sbs.TotalWidth())
	}
}
