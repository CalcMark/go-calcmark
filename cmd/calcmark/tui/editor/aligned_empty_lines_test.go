package editor

import (
	"strings"
	"testing"

	"github.com/CalcMark/go-calcmark/spec/document"
)

// TestAlignedModelWithEmptyLineAfterHeading tests that ComputeAlignedModel
// correctly handles empty lines in text blocks.
func TestAlignedModelWithEmptyLineAfterHeading(t *testing.T) {
	// The exact case from the user's screenshot:
	// # Test
	// <empty>
	// <empty>
	// 1. Boop
	// 1. Bop
	source := "# Test\n\n\n1. Boop\n1. Bop"

	doc, err := document.NewDocument(source)
	if err != nil {
		t.Fatalf("NewDocument failed: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 24

	// Get line results
	results := m.GetLineResults()
	t.Logf("Got %d line results", len(results))
	for i, r := range results {
		t.Logf("  [%d] %q", i, r.Source)
	}

	// Compute aligned model
	sourceWidth := 40
	previewWidth := 40

	input := AlignedModelInput{
		Lines:              m.GetLines(),
		Results:            results,
		SourceContentWidth: sourceWidth,
		PreviewWidth:       previewWidth,
		CursorLine:         0,
		PreviewMode:        PreviewFull,
	}

	aligned := ComputeAlignedModel(input, m.renderCalcLine, func(line string, width int) []string {
		mdRenderer, _ := NewMarkdownRenderer(width)
		if mdRenderer != nil {
			return mdRenderer.RenderLine(line)
		}
		return WrapText(line, width)
	})

	t.Logf("\nComputed aligned model:")
	t.Logf("Total visual lines: %d", aligned.TotalVisualLines)
	t.Logf("\nSource lines:")
	for i, sl := range aligned.SourceLines {
		t.Logf("  [%d] (src=%d) %q (kind=%d)", i, sl.SourceLineIdx, sl.Content, sl.Kind)
	}
	t.Logf("\nPreview lines:")
	for i, pl := range aligned.PreviewLines {
		t.Logf("  [%d] (src=%d) %q (kind=%d)", i, pl.SourceLineIdx, pl.Content, pl.Kind)
	}

	// The critical check: do we have the right number of visual lines?
	// We should have 5 source lines, so we need 5+ visual lines (accounting for wrapping)
	if aligned.TotalVisualLines < 5 {
		t.Errorf("Expected at least 5 visual lines, got %d", aligned.TotalVisualLines)
	}

	// Check that preview lines 1 and 2 (corresponding to source lines 1 and 2) are empty
	// Find visual lines corresponding to source lines 1 and 2
	var visualLine1, visualLine2 *AlignedLine
	for i := range aligned.PreviewLines {
		if aligned.PreviewLines[i].SourceLineIdx == 1 && visualLine1 == nil {
			visualLine1 = &aligned.PreviewLines[i]
		}
		if aligned.PreviewLines[i].SourceLineIdx == 2 && visualLine2 == nil {
			visualLine2 = &aligned.PreviewLines[i]
		}
	}

	if visualLine1 == nil {
		t.Error("Could not find preview line for source line 1")
	} else if strings.TrimSpace(visualLine1.Content) != "" {
		t.Errorf("Preview for source line 1 should be empty, got %q", visualLine1.Content)
	}

	if visualLine2 == nil {
		t.Error("Could not find preview line for source line 2")
	} else if strings.TrimSpace(visualLine2.Content) != "" {
		t.Errorf("Preview for source line 2 should be empty, got %q", visualLine2.Content)
	}
}

// TestGlamourStripsEmptyLines tests the markdown renderer behavior directly.
func TestGlamourStripsEmptyLines(t *testing.T) {
	// Test case: heading with empty lines after it
	source := "# Test\n\n\n1. Boop"

	mdRenderer, err := NewMarkdownRenderer(40)
	if err != nil {
		t.Fatalf("NewMarkdownRenderer failed: %v", err)
	}

	// Render the entire block
	rendered := mdRenderer.RenderLine(source)

	t.Logf("Input: %q", source)
	t.Logf("Input line count: %d", strings.Count(source, "\n")+1)
	t.Logf("Rendered line count: %d", len(rendered))
	for i, line := range rendered {
		t.Logf("  [%d] %q", i, line)
	}

	// The issue: glamour might strip the empty lines
	inputLineCount := strings.Count(source, "\n") + 1 // 4 lines: "# Test", "", "", "1. Boop"
	if len(rendered) < inputLineCount {
		t.Logf("WARNING: Glamour stripped empty lines! Input had %d lines, output has %d",
			inputLineCount, len(rendered))
	}
}
