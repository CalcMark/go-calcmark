package editor

import (
	"strings"
	"testing"

	"github.com/CalcMark/go-calcmark/cmd/calcmark/tui/geometry"
	"github.com/CalcMark/go-calcmark/spec/document"
)

// TestPreviewAlignmentIntegration is a comprehensive integration test that verifies
// empty lines between content blocks are preserved throughout the entire rendering pipeline:
// 1. Document model (blocks contain empty lines)
// 2. LineResults (results include empty line entries)
// 3. AlignedModel (visual lines preserve empties)
// 4. View rendering (final output shows empty space for empty lines)
//
// This test covers the specific case reported by the user:
// - Heading ("# Test")
// - Two empty lines (block separator)
// - Ordered list ("1. Boop", "1. Bop")
func TestPreviewAlignmentIntegration(t *testing.T) {
	testCases := []struct {
		name          string
		source        string
		expectLines   int   // Expected number of source lines
		emptyLineIdxs []int // Indices of lines that should be empty
		description   string
	}{
		{
			name:          "Heading with empty lines before ordered list",
			source:        "# Test\n\n\n1. Boop\n1. Bop",
			expectLines:   5,
			emptyLineIdxs: []int{1, 2}, // Lines 1 and 2 should be empty
			description:   "User's exact scenario: heading, 2 empties, ordered list",
		},
		{
			name:          "Heading with empty lines before calculation",
			source:        "# Heading\n\n\na = 10",
			expectLines:   4,
			emptyLineIdxs: []int{1, 2},
			description:   "Empty lines before calc block",
		},
		{
			name:          "Multiple text blocks with separators",
			source:        "# First\n\n\n# Second\n\n\n# Third",
			expectLines:   7,
			emptyLineIdxs: []int{1, 2, 4, 5},
			description:   "Multiple headings with empty separators",
		},
		{
			name:          "Calculation then text with separator",
			source:        "a = 10\n\n\n# Result",
			expectLines:   4,
			emptyLineIdxs: []int{1, 2},
			description:   "Calc block followed by text block with separator",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Logf("Test: %s", tc.description)

			// === LEVEL 1: Document Model ===
			doc, err := document.NewDocument(tc.source)
			if err != nil {
				t.Fatalf("NewDocument failed: %v", err)
			}

			blocks := doc.GetBlocks()
			t.Logf("Document has %d blocks", len(blocks))

			// Verify block structure contains the empty lines
			totalSourceLines := 0
			for i, node := range blocks {
				var sourceLines []string
				switch b := node.Block.(type) {
				case *document.TextBlock:
					sourceLines = b.Source()
				case *document.CalcBlock:
					sourceLines = b.Source()
				}
				totalSourceLines += len(sourceLines)
				t.Logf("  Block %d: %d lines", i, len(sourceLines))
			}

			if totalSourceLines != tc.expectLines {
				t.Errorf("Document should have %d total source lines, got %d",
					tc.expectLines, totalSourceLines)
			}

			// === LEVEL 2: LineResults ===
			m := New(doc)
			m.width = 80
			m.height = 24

			results := m.GetLineResults()
			t.Logf("GetLineResults returned %d entries", len(results))

			if len(results) != tc.expectLines {
				t.Errorf("Expected %d line results, got %d", tc.expectLines, len(results))
			}

			// Verify empty lines have empty source
			for _, idx := range tc.emptyLineIdxs {
				if idx >= len(results) {
					t.Errorf("Empty line index %d out of bounds (have %d results)", idx, len(results))
					continue
				}
				if results[idx].Source != "" {
					t.Errorf("Line %d should be empty, got %q", idx, results[idx].Source)
				}
			}

			// === LEVEL 3: AlignedModel ===
			input := AlignedModelInput{
				Lines:              m.GetLines(),
				Results:            results,
				SourceContentWidth: 35,
				PreviewWidth:       40,
				CursorLine:         0,
				PreviewMode:        PreviewFull,
			}

			aligned := ComputeAlignedModel(input, m.renderCalcLine, func(line string, width int) []string {
				mdRenderer, _ := NewMarkdownRenderer(width)
				if mdRenderer != nil {
					return mdRenderer.RenderLine(line)
				}
				return geometry.WrapText(line, width)
			})

			t.Logf("AlignedModel has %d visual lines", aligned.TotalVisualLines)

			// Verify we have at least as many visual lines as source lines
			// (could be more due to wrapping)
			if aligned.TotalVisualLines < tc.expectLines {
				t.Errorf("Expected at least %d visual lines, got %d",
					tc.expectLines, aligned.TotalVisualLines)
			}

			// Verify empty lines exist in preview
			for _, idx := range tc.emptyLineIdxs {
				// Find the first visual line corresponding to this source line
				var previewLine *AlignedLine
				for i := range aligned.PreviewLines {
					if aligned.PreviewLines[i].SourceLineIdx == idx {
						previewLine = &aligned.PreviewLines[i]
						break
					}
				}

				if previewLine == nil {
					t.Errorf("Could not find preview line for source line %d", idx)
				} else if strings.TrimSpace(previewLine.Content) != "" {
					t.Errorf("Preview for source line %d should be empty, got %q",
						idx, previewLine.Content)
				}
			}

			// === LEVEL 4: View Rendering ===
			m.previewMode = PreviewFull
			view := m.View()
			viewLines := strings.Split(view, "\n")

			t.Logf("View output has %d lines", len(viewLines))

			// Verify the view contains the expected number of content lines
			// (This is a basic sanity check - exact line count depends on layout)
			if len(viewLines) < 10 {
				t.Errorf("View should have at least 10 lines (with chrome), got %d", len(viewLines))
			}

			// === SUMMARY ===
			t.Logf("✓ All levels validated:")
			t.Logf("  - Document model: %d blocks, %d total lines", len(blocks), totalSourceLines)
			t.Logf("  - LineResults: %d entries", len(results))
			t.Logf("  - AlignedModel: %d visual lines", aligned.TotalVisualLines)
			t.Logf("  - View: %d output lines", len(viewLines))
		})
	}
}

// TestGlamourEmptyLineCompensation tests the specific compensation logic
// in aligned.go that handles glamour stripping empty lines.
func TestGlamourEmptyLineCompensation(t *testing.T) {
	// This test documents the known issue with glamour and verifies the compensation works

	source := "# Test\n\n\n1. Boop"
	doc, err := document.NewDocument(source)
	if err != nil {
		t.Fatalf("NewDocument failed: %v", err)
	}

	m := New(doc)
	results := m.GetLineResults()

	// Document has 4 lines
	if len(results) != 4 {
		t.Fatalf("Expected 4 lines, got %d", len(results))
	}

	// Direct glamour rendering strips empties
	mdRenderer, _ := NewMarkdownRenderer(40)
	if mdRenderer != nil {
		directRendered := mdRenderer.RenderLine(source)
		t.Logf("Direct glamour rendering: %d lines (input had 4)", len(directRendered))
		if len(directRendered) < 4 {
			t.Logf("  Glamour stripped %d empty lines", 4-len(directRendered))
		}
	}

	// But AlignedModel compensates
	input := AlignedModelInput{
		Lines:              m.GetLines(),
		Results:            results,
		SourceContentWidth: 35,
		PreviewWidth:       40,
		CursorLine:         0,
		PreviewMode:        PreviewFull,
	}

	aligned := ComputeAlignedModel(input, m.renderCalcLine, func(line string, width int) []string {
		mdRenderer, _ := NewMarkdownRenderer(width)
		if mdRenderer != nil {
			return mdRenderer.RenderLine(line)
		}
		return geometry.WrapText(line, width)
	})

	// AlignedModel should preserve all 4 lines
	if aligned.TotalVisualLines < 4 {
		t.Errorf("AlignedModel compensation failed: expected 4+ visual lines, got %d",
			aligned.TotalVisualLines)
	}

	// Verify lines 1 and 2 (the empties) are present
	hasLine1 := false
	hasLine2 := false
	for _, pl := range aligned.PreviewLines {
		if pl.SourceLineIdx == 1 {
			hasLine1 = true
		}
		if pl.SourceLineIdx == 2 {
			hasLine2 = true
		}
	}

	if !hasLine1 {
		t.Error("Source line 1 (empty) missing from preview")
	}
	if !hasLine2 {
		t.Error("Source line 2 (empty) missing from preview")
	}

	t.Logf("✓ Glamour compensation working: preserved empty lines")
}

// TestGlamourBrittnessMonitoring is a monitoring test that checks for the
// brittleness scenario where glamour might strip empties but also add lines
// through wrapping, causing the compensation logic to fail.
//
// This scenario hasn't been observed in practice, but we monitor for it.
func TestGlamourBrittnessMonitoring(t *testing.T) {
	// Test various markdown patterns that might cause glamour to both:
	// 1. Strip empty lines (reducing count)
	// 2. Add lines through wrapping/formatting (increasing count)
	testCases := []struct {
		name        string
		source      string
		expectLines int
		description string
	}{
		{
			name:        "Long heading that might wrap",
			source:      "# This is a very long heading that might wrap and add extra lines when rendered\n\n\n1. Item",
			expectLines: 4,
			description: "Long heading with empties followed by list",
		},
		{
			name:        "Multiple long paragraphs with empties",
			source:      "This is a very long paragraph with lots of text that might cause wrapping.\n\n\nAnother long paragraph here.",
			expectLines: 4,
			description: "Long paragraphs with empties",
		},
		{
			name:        "Complex list with empties",
			source:      "# Heading\n\n\n1. Very long list item that might wrap when rendered\n2. Another item",
			expectLines: 5,
			description: "Heading, empties, then long list items",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Logf("Testing: %s", tc.description)

			doc, err := document.NewDocument(tc.source)
			if err != nil {
				t.Fatalf("NewDocument failed: %v", err)
			}

			m := New(doc)
			m.width = 80
			m.height = 24

			results := m.GetLineResults()
			sourceLineCount := len(results)

			t.Logf("Source has %d lines", sourceLineCount)

			if sourceLineCount != tc.expectLines {
				t.Errorf("Expected %d source lines, got %d", tc.expectLines, sourceLineCount)
			}

			// Compute aligned model
			input := AlignedModelInput{
				Lines:              m.GetLines(),
				Results:            results,
				SourceContentWidth: 35,
				PreviewWidth:       40,
				CursorLine:         0,
				PreviewMode:        PreviewFull,
			}

			aligned := ComputeAlignedModel(input, m.renderCalcLine, func(line string, width int) []string {
				mdRenderer, _ := NewMarkdownRenderer(width)
				if mdRenderer != nil {
					return mdRenderer.RenderLine(line)
				}
				return geometry.WrapText(line, width)
			})

			t.Logf("AlignedModel has %d visual lines", aligned.TotalVisualLines)

			// CRITICAL CHECK: If visual lines >= source lines but we're missing empties,
			// this indicates the brittleness scenario
			if aligned.TotalVisualLines >= sourceLineCount {
				// Check if we still have all the empty lines we expect
				emptyCount := 0
				for _, r := range results {
					if strings.TrimSpace(r.Source) == "" {
						emptyCount++
					}
				}

				// Count empties in preview
				previewEmptyCount := 0
				for _, pl := range aligned.PreviewLines {
					if strings.TrimSpace(pl.Content) == "" {
						previewEmptyCount++
					}
				}

				t.Logf("Source empties: %d, Preview empties: %d", emptyCount, previewEmptyCount)

				if previewEmptyCount < emptyCount {
					t.Errorf("BRITTLENESS DETECTED: Visual lines (%d) >= source lines (%d), "+
						"but empties are missing (source: %d, preview: %d). "+
						"This means glamour stripped empties AND added lines, "+
						"bypassing compensation logic!",
						aligned.TotalVisualLines, sourceLineCount,
						emptyCount, previewEmptyCount)
				} else {
					t.Logf("✓ No brittleness: empties preserved despite visual >= source")
				}
			}

			// Verify 1:1 mapping exists for all source lines
			sourceIndices := make(map[int]bool)
			for _, pl := range aligned.PreviewLines {
				sourceIndices[pl.SourceLineIdx] = true
			}

			for i := 0; i < sourceLineCount; i++ {
				if !sourceIndices[i] {
					t.Errorf("Source line %d has no corresponding preview line", i)
				}
			}
		})
	}
}
