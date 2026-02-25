package editor

import (
	"strings"
	"testing"

	"github.com/CalcMark/go-calcmark/spec/document"
)

// TestDividerPositionConsistency verifies that the divider appears at a consistent
// column position across all lines in the View() output.
func TestDividerPositionConsistency(t *testing.T) {
	// Create document with varying content lengths
	content := "# Header\nx = 10\nvery_long_variable_name = 100\ny = 5"
	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 24
	m.previewMode = PreviewFull

	// Get pane widths
	leftWidth, rightWidth := m.GetPaneWidths(80)
	t.Logf("leftWidth=%d, rightWidth=%d, total=%d", leftWidth, rightWidth, leftWidth+rightWidth)

	// Expected divider column accounting for divider width
	// The divider is placed after leftContentWidth (leftWidth - dividerWidth)
	// With dividerWidth=1: divider is at position (leftWidth - 1) - 1 = leftWidth - 2
	const dividerWidth = 1
	leftContentWidth := leftWidth - dividerWidth
	// Divider appears at the END of left content (after leftContentWidth characters)
	expectedDividerCol := leftContentWidth

	// Render view
	view := m.View().Content
	lines := strings.Split(view, "\n")

	t.Logf("Total lines in view: %d", len(lines))

	// Track divider positions
	dividerPositions := make(map[int][]int) // map[lineIdx][]positions

	// Check each line for divider position
	for i, line := range lines {
		if len(line) == 0 {
			continue // Skip empty lines
		}

		// Strip ANSI codes to find the divider position
		plainLine := stripANSI(line)

		// Find all divider characters '│'
		var positions []int
		for pos, ch := range plainLine {
			if ch == '│' {
				positions = append(positions, pos)
			}
		}

		if len(positions) > 0 {
			dividerPositions[i] = positions
			if positions[0] != expectedDividerCol {
				t.Errorf("Line %d: divider at column %d, expected column %d\n  PlainLine: %q (len=%d)",
					i, positions[0], expectedDividerCol, plainLine, len(plainLine))
			} else {
				t.Logf("Line %d: divider at column %d ✓", i, positions[0])
			}
		}
	}

	if len(dividerPositions) == 0 {
		t.Error("No divider characters found in View() output")
	} else {
		t.Logf("Found dividers on %d lines, all at column %d", len(dividerPositions), expectedDividerCol)
	}
}
