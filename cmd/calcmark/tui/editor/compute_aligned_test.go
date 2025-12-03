package editor

import (
	"testing"
)

// TestComputeAligned_EditBuf tests that editBuf is correctly used in preview rendering.
func TestComputeAligned_EditBuf(t *testing.T) {
	// Simulate the state: 3 lines saved, 1 line being typed
	input := AlignedModelInput{
		Lines: []string{"1. First", "2. Second", "", ""},
		Results: []LineResult{
			{LineNum: 0, BlockID: "test-block", IsCalc: false, Source: "1. First"},
			{LineNum: 1, BlockID: "test-block", IsCalc: false, Source: "2. Second"},
			{LineNum: 2, BlockID: "test-block", IsCalc: false, Source: ""},
			{LineNum: 3, BlockID: "test-block", IsCalc: false, Source: ""},
		},
		SourceContentWidth: 40,
		PreviewWidth:       40,
		CursorLine:         3,
		PreviewMode:        PreviewFull,
		EditBuf:            "# Title", // User is typing this on line 3
		EditBufLine:        3,
	}

	// Create a simple markdown renderer
	mdRenderer, err := NewMarkdownRenderer(40)
	if err != nil {
		t.Fatalf("Failed to create markdown renderer: %v", err)
	}

	// Test what SHOULD be rendered
	expectedBlock := "1. First\n2. Second\n\n# Title"
	expectedRendered := mdRenderer.RenderLine(expectedBlock)
	t.Logf("Expected block text: %q", expectedBlock)
	t.Logf("Expected render (%d lines):", len(expectedRendered))
	for i, line := range expectedRendered {
		t.Logf("  [%d] %q", i, line)
	}

	// Compute aligned model
	aligned := ComputeAlignedModel(input, nil, func(line string, width int) []string {
		t.Logf("RenderMarkdown called with: %q", line)
		result := mdRenderer.RenderLine(line)
		t.Logf("  Returned %d lines", len(result))
		return result
	})

	// Check results
	t.Logf("Preview lines: %d", len(aligned.PreviewLines))
	for i, line := range aligned.PreviewLines {
		t.Logf("  [%d] SourceLine=%d Content=%q", i, line.SourceLineIdx, line.Content)
	}

	// ASSERTION: Preview should have 4 lines
	if len(aligned.PreviewLines) != 4 {
		t.Errorf("Expected 4 preview lines, got %d", len(aligned.PreviewLines))
	}

	// ASSERTION: Line 3 should show the heading (from editBuf)
	if len(aligned.PreviewLines) > 3 {
		preview3 := aligned.PreviewLines[3].Content
		if preview3 == "" {
			t.Errorf("BUG: Preview line 3 is empty, should show heading from editBuf")
			t.Logf("EditBuf was: %q", input.EditBuf)
		} else {
			t.Logf("SUCCESS: Preview line 3 shows: %q", preview3)
		}
	}
}
