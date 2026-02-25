package editor

import (
	"fmt"
	"strings"
	"testing"

	"github.com/CalcMark/go-calcmark/spec/document"
)

// TestViewportDoesNotExceedHeight verifies that the View() output
// doesn't exceed the specified terminal height, which would cause
// terminal content to bleed through at the bottom.
//
// REGRESSION: Terminal build output appearing at bottom of screen
func TestViewportDoesNotExceedHeight(t *testing.T) {
	doc, err := document.NewDocument("x = 10\n")
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 24 // Standard terminal height

	view := m.View().Content
	lines := strings.Split(view, "\n")

	t.Logf("View has %d lines for height %d", len(lines), m.height)
	t.Logf("View byte count: %d", len(view))
	t.Logf("First 5 lines:\n%s", strings.Join(lines[:min(5, len(lines))], "\n"))
	t.Logf("Last 5 lines:\n%s", strings.Join(lines[max(0, len(lines)-5):], "\n"))

	// Check for trailing newlines beyond the last line
	trailingNewlines := 0
	for i := len(view) - 1; i >= 0 && view[i] == '\n'; i-- {
		trailingNewlines++
	}
	if trailingNewlines > 1 {
		t.Logf("WARNING: View has %d trailing newlines", trailingNewlines)
	}

	// The View should not exceed the terminal height
	// If it does, content will wrap and terminal history will show through
	if len(lines) > m.height {
		t.Errorf("View has %d lines but terminal height is %d - this causes terminal bleed-through!",
			len(lines), m.height)
		t.Errorf("Excess lines: %d", len(lines)-m.height)
	}

	// Check if last line is empty (which would cause an extra visual line)
	if len(lines) > 0 && lines[len(lines)-1] != "" {
		t.Logf("Last line is not empty: %q", lines[len(lines)-1])
	}
}

// TestViewportHeightWithLargeContent tests with more content
func TestViewportHeightWithLargeContent(t *testing.T) {
	// Create document with multiple lines
	var b strings.Builder
	for i := 1; i <= 50; i++ {
		fmt.Fprintf(&b, "x%c = %c\n", rune('0'+i%10), rune('0'+i%10))
	}
	content := b.String()

	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 24

	view := m.View().Content
	lines := strings.Split(view, "\n")

	t.Logf("View has %d lines for height %d (with scrolling)", len(lines), m.height)

	// Even with scrolling, the rendered view should not exceed terminal height
	if len(lines) > m.height {
		t.Errorf("View has %d lines but terminal height is %d",
			len(lines), m.height)
		t.Errorf("This causes terminal content to bleed through at the bottom")
	}
}
