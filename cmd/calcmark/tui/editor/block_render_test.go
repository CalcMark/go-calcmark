package editor

import (
	"strings"
	"testing"

	"github.com/CalcMark/go-calcmark/v2/spec/document"
)

// TestRenderTextBlockSimple_OrderedList verifies that ordered lists render with correct numbering
func TestRenderTextBlockSimple_OrderedList(t *testing.T) {
	// Create a document with an ordered list
	content := "1. First item\n1. Second item\n1. Third item"
	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	blocks := doc.GetBlocks()
	if len(blocks) == 0 {
		t.Fatal("Document has no blocks")
	}

	textBlock, ok := blocks[0].Block.(*document.TextBlock)
	if !ok {
		t.Fatalf("First block is not a TextBlock, got %T", blocks[0].Block)
	}

	renderer, err := NewMarkdownRenderer(80)
	if err != nil {
		t.Fatalf("Failed to create renderer: %v", err)
	}

	rendered := RenderTextBlockSimple(textBlock, renderer)

	t.Logf("Rendered %d lines:", len(rendered))
	for i, line := range rendered {
		t.Logf("  [%d] %q", i, line)
	}

	// Verify we got output
	if len(rendered) == 0 {
		t.Fatal("No rendered output")
	}

	// Join all rendered lines to search for numbering
	fullOutput := strings.Join(rendered, "\n")
	fullOutput = ansiEscapeRe.ReplaceAllString(fullOutput, "")

	// Check that the output contains proper numbering
	// Glamour renders ordered lists with format "1. ", "2. ", "3. "
	if !strings.Contains(fullOutput, "1.") {
		t.Error("Output should contain '1.' for first item")
	}
	if !strings.Contains(fullOutput, "2.") {
		t.Error("Output should contain '2.' for second item")
	}
	if !strings.Contains(fullOutput, "3.") {
		t.Error("Output should contain '3.' for third item")
	}

	// Make sure we're not just showing "1." for everything
	count1 := strings.Count(fullOutput, "1.")
	count2 := strings.Count(fullOutput, "2.")
	count3 := strings.Count(fullOutput, "3.")

	if count1 == 0 || count2 == 0 || count3 == 0 {
		t.Errorf("Expected all three numbers to appear at least once, got: 1.=%d, 2.=%d, 3.=%d",
			count1, count2, count3)
	}
}

// TestRenderTextBlockSimple_UnorderedList verifies unordered lists still work
func TestRenderTextBlockSimple_UnorderedList(t *testing.T) {
	content := "- First item\n- Second item\n- Third item"
	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	blocks := doc.GetBlocks()
	if len(blocks) == 0 {
		t.Fatal("Document has no blocks")
	}

	textBlock, ok := blocks[0].Block.(*document.TextBlock)
	if !ok {
		t.Fatalf("First block is not a TextBlock, got %T", blocks[0].Block)
	}

	renderer, err := NewMarkdownRenderer(80)
	if err != nil {
		t.Fatalf("Failed to create renderer: %v", err)
	}

	rendered := RenderTextBlockSimple(textBlock, renderer)

	t.Logf("Rendered %d lines:", len(rendered))
	for i, line := range rendered {
		t.Logf("  [%d] %q", i, line)
	}

	if len(rendered) == 0 {
		t.Fatal("No rendered output")
	}

	// Unordered lists should have bullet points (glamour uses "• ")
	fullOutput := strings.Join(rendered, "\n")
	if !strings.Contains(fullOutput, "•") && !strings.Contains(fullOutput, "-") {
		t.Error("Output should contain bullet points for unordered list")
	}
}

// TestRenderTextBlockSimple_MixedContent verifies mixed markdown content
func TestRenderTextBlockSimple_MixedContent(t *testing.T) {
	content := "# Header\n\n1. First\n1. Second\n\n- Bullet\n- Point"
	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	blocks := doc.GetBlocks()
	if len(blocks) == 0 {
		t.Fatal("Document has no blocks")
	}

	textBlock, ok := blocks[0].Block.(*document.TextBlock)
	if !ok {
		t.Fatalf("First block is not a TextBlock, got %T", blocks[0].Block)
	}

	renderer, err := NewMarkdownRenderer(80)
	if err != nil {
		t.Fatalf("Failed to create renderer: %v", err)
	}

	rendered := RenderTextBlockSimple(textBlock, renderer)

	t.Logf("Rendered %d lines:", len(rendered))
	for i, line := range rendered {
		t.Logf("  [%d] %q", i, line)
	}

	if len(rendered) == 0 {
		t.Fatal("No rendered output")
	}

	fullOutput := strings.Join(rendered, "\n")
	fullOutput = ansiEscapeRe.ReplaceAllString(fullOutput, "")

	// Should contain header, ordered list with numbering, and bullets
	if !strings.Contains(fullOutput, "Header") {
		t.Error("Output should contain header text")
	}
	if !strings.Contains(fullOutput, "1.") {
		t.Error("Output should contain '1.' for ordered list")
	}
	if !strings.Contains(fullOutput, "2.") {
		t.Error("Output should contain '2.' for second ordered item")
	}
	if !strings.Contains(fullOutput, "•") && !strings.Contains(fullOutput, "-") {
		t.Error("Output should contain bullets for unordered list")
	}
}

// TestRenderTextBlockSimple_EmptyBlock verifies empty blocks are handled
func TestRenderTextBlockSimple_EmptyBlock(t *testing.T) {
	content := ""
	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	blocks := doc.GetBlocks()
	if len(blocks) == 0 {
		// Empty document has no blocks - this is OK
		return
	}

	textBlock, ok := blocks[0].Block.(*document.TextBlock)
	if !ok {
		// Not a text block - skip
		return
	}

	renderer, err := NewMarkdownRenderer(80)
	if err != nil {
		t.Fatalf("Failed to create renderer: %v", err)
	}

	rendered := RenderTextBlockSimple(textBlock, renderer)

	// Empty block should produce empty or minimal output
	t.Logf("Empty block rendered to %d lines", len(rendered))
}

// TestRenderTextBlockAligned_BasicAlignment verifies alignment mapping works
func TestRenderTextBlockAligned_BasicAlignment(t *testing.T) {
	content := "Line 1\nLine 2\nLine 3"
	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	blocks := doc.GetBlocks()
	if len(blocks) == 0 {
		t.Fatal("Document has no blocks")
	}

	textBlock, ok := blocks[0].Block.(*document.TextBlock)
	if !ok {
		t.Fatalf("First block is not a TextBlock, got %T", blocks[0].Block)
	}

	renderer, err := NewMarkdownRenderer(80)
	if err != nil {
		t.Fatalf("Failed to create renderer: %v", err)
	}

	results := RenderTextBlockAligned(textBlock, renderer)

	t.Logf("Got %d alignment results:", len(results))
	for i, r := range results {
		t.Logf("  [%d] Source line %d -> %d preview lines: %v",
			i, r.SourceLineIndex, len(r.PreviewLines), r.PreviewLines)
	}

	// Should have one result per source line
	sourceLines := textBlock.Source()
	if len(results) != len(sourceLines) {
		t.Errorf("Expected %d results (one per source line), got %d",
			len(sourceLines), len(results))
	}

	// Each result should have correct source index
	for i, r := range results {
		if r.SourceLineIndex != i {
			t.Errorf("Result %d has wrong SourceLineIndex: expected %d, got %d",
				i, i, r.SourceLineIndex)
		}
	}
}

// TestRenderTextBlockAligned_OrderedList verifies ordered list alignment
func TestRenderTextBlockAligned_OrderedList(t *testing.T) {
	content := "1. First item\n1. Second item\n1. Third item"
	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	blocks := doc.GetBlocks()
	if len(blocks) == 0 {
		t.Fatal("Document has no blocks")
	}

	textBlock, ok := blocks[0].Block.(*document.TextBlock)
	if !ok {
		t.Fatalf("First block is not a TextBlock, got %T", blocks[0].Block)
	}

	renderer, err := NewMarkdownRenderer(80)
	if err != nil {
		t.Fatalf("Failed to create renderer: %v", err)
	}

	results := RenderTextBlockAligned(textBlock, renderer)

	t.Logf("Got %d alignment results:", len(results))
	for i, r := range results {
		t.Logf("  [%d] Source line %d -> preview: %v",
			i, r.SourceLineIndex, r.PreviewLines)
	}

	// Collect all preview lines
	var allPreview []string
	for _, r := range results {
		allPreview = append(allPreview, r.PreviewLines...)
	}

	fullOutput := strings.Join(allPreview, "\n")
	t.Logf("Full preview output:\n%s", fullOutput)
	fullOutput = ansiEscapeRe.ReplaceAllString(fullOutput, "")

	// Verify proper numbering appears
	if !strings.Contains(fullOutput, "1.") {
		t.Error("Preview should contain '1.'")
	}
	if !strings.Contains(fullOutput, "2.") {
		t.Error("Preview should contain '2.'")
	}
	if !strings.Contains(fullOutput, "3.") {
		t.Error("Preview should contain '3.'")
	}
}

// TestRenderTextBlockAligned_WithEmptyLines verifies empty line handling
func TestRenderTextBlockAligned_WithEmptyLines(t *testing.T) {
	content := "Line 1\n\nLine 3"
	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	blocks := doc.GetBlocks()
	if len(blocks) == 0 {
		t.Fatal("Document has no blocks")
	}

	textBlock, ok := blocks[0].Block.(*document.TextBlock)
	if !ok {
		t.Fatalf("First block is not a TextBlock, got %T", blocks[0].Block)
	}

	renderer, err := NewMarkdownRenderer(80)
	if err != nil {
		t.Fatalf("Failed to create renderer: %v", err)
	}

	results := RenderTextBlockAligned(textBlock, renderer)

	t.Logf("Got %d alignment results:", len(results))
	for i, r := range results {
		t.Logf("  [%d] Source line %d -> preview: %v",
			i, r.SourceLineIndex, r.PreviewLines)
	}

	// Should have 3 results (including empty line)
	if len(results) != 3 {
		t.Errorf("Expected 3 results, got %d", len(results))
	}

	// Middle result should have empty preview
	if len(results) >= 2 {
		if len(results[1].PreviewLines) != 1 || results[1].PreviewLines[0] != "" {
			t.Error("Empty source line should produce empty preview line")
		}
	}
}

// TestDeterminism verifies the function is deterministic
func TestDeterminism(t *testing.T) {
	content := "1. First\n1. Second\n1. Third"
	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	blocks := doc.GetBlocks()
	if len(blocks) == 0 {
		t.Fatal("Document has no blocks")
	}

	textBlock, ok := blocks[0].Block.(*document.TextBlock)
	if !ok {
		t.Fatalf("First block is not a TextBlock, got %T", blocks[0].Block)
	}

	renderer, err := NewMarkdownRenderer(80)
	if err != nil {
		t.Fatalf("Failed to create renderer: %v", err)
	}

	// Render multiple times
	rendered1 := RenderTextBlockSimple(textBlock, renderer)
	rendered2 := RenderTextBlockSimple(textBlock, renderer)
	rendered3 := RenderTextBlockSimple(textBlock, renderer)

	// Results should be identical
	if len(rendered1) != len(rendered2) || len(rendered1) != len(rendered3) {
		t.Errorf("Renders produced different lengths: %d, %d, %d",
			len(rendered1), len(rendered2), len(rendered3))
	}

	for i := 0; i < len(rendered1) && i < len(rendered2) && i < len(rendered3); i++ {
		if rendered1[i] != rendered2[i] || rendered1[i] != rendered3[i] {
			t.Errorf("Line %d differs between renders:\n  1: %q\n  2: %q\n  3: %q",
				i, rendered1[i], rendered2[i], rendered3[i])
		}
	}
}

// TestIntegration_FullDocument verifies the complete test-lists.cm document renders correctly
func TestIntegration_FullDocument(t *testing.T) {
	// This is the document from test-lists.cm that shows the bug
	content := `# Test

1. asdf
1. asdf

a = 2

- test
- ts

1. asdf
1. 2432`

	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	blocks := doc.GetBlocks()
	t.Logf("Document has %d blocks", len(blocks))

	renderer, err := NewMarkdownRenderer(80)
	if err != nil {
		t.Fatalf("Failed to create renderer: %v", err)
	}

	// Process each block
	for i, blockNode := range blocks {
		t.Logf("\nBlock %d: %T", i, blockNode.Block)

		if textBlock, ok := blockNode.Block.(*document.TextBlock); ok {
			rendered := RenderTextBlockSimple(textBlock, renderer)
			t.Logf("  TextBlock rendered to %d lines:", len(rendered))
			for j, line := range rendered {
				t.Logf("    [%d] %q", j, line)
			}

			// Check for proper ordered list numbering
			fullOutput := strings.Join(rendered, "\n")
			fullOutput = ansiEscapeRe.ReplaceAllString(fullOutput, "")

			// First ordered list (lines 2-3 of block 0)
			if i == 0 {
				if !strings.Contains(fullOutput, "Test") {
					t.Error("Block 0 should contain header 'Test'")
				}
				// Should have properly numbered ordered list
				if strings.Count(fullOutput, "1.") < 1 {
					t.Error("Block 0 should contain at least one '1.'")
				}
				if strings.Count(fullOutput, "2.") < 1 {
					t.Error("Block 0 should contain '2.' for second list item")
				}
			}

			// Third block has unordered list + second ordered list
			if i == 2 {
				// Should have bullets for unordered list
				if !strings.Contains(fullOutput, "•") && !strings.Contains(fullOutput, "-") {
					t.Error("Block 2 should contain bullets for unordered list")
				}
				// Should have properly numbered second ordered list
				if strings.Count(fullOutput, "1.") < 1 {
					t.Error("Block 2 should contain '1.' for ordered list")
				}
				if strings.Count(fullOutput, "2.") < 1 {
					t.Error("Block 2 should contain '2.' for second ordered item")
				}
			}
		}
	}
}

// TestIntegration_OrderedListAfterCalc verifies ordered list after calc doesn't break calc display
// This is the specific bug shown in the user's screenshot
func TestIntegration_OrderedListAfterCalc(t *testing.T) {
	content := "a = 2 * 5\n\n1. First item"

	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	blocks := doc.GetBlocks()
	t.Logf("Document has %d blocks", len(blocks))

	// Should have at least 2 blocks: calc block and text block
	if len(blocks) < 2 {
		t.Fatalf("Expected at least 2 blocks, got %d", len(blocks))
	}

	// First block should be CalcBlock
	_, isCalc := blocks[0].Block.(*document.CalcBlock)
	if !isCalc {
		t.Errorf("Block 0 should be CalcBlock, got %T", blocks[0].Block)
	}

	// Second block should be TextBlock with ordered list
	textBlock, isText := blocks[1].Block.(*document.TextBlock)
	if !isText {
		t.Fatalf("Block 1 should be TextBlock, got %T", blocks[1].Block)
	}

	renderer, err := NewMarkdownRenderer(80)
	if err != nil {
		t.Fatalf("Failed to create renderer: %v", err)
	}

	rendered := RenderTextBlockSimple(textBlock, renderer)
	t.Logf("TextBlock with ordered list rendered to %d lines:", len(rendered))
	for i, line := range rendered {
		t.Logf("  [%d] %q", i, line)
	}

	// The ordered list should start with "1."
	fullOutput := strings.Join(rendered, "\n")
	fullOutput = ansiEscapeRe.ReplaceAllString(fullOutput, "")
	if !strings.Contains(fullOutput, "1.") {
		t.Error("Ordered list should contain '1.'")
	}
	if !strings.Contains(fullOutput, "First item") {
		t.Error("Ordered list should contain item text")
	}
}

// TestRenderBlock_PreservesBlankLines verifies blank lines between headings
// and paragraphs are preserved when rendering through RenderBlock.
func TestRenderBlock_PreservesBlankLines(t *testing.T) {
	renderer, err := NewMarkdownRenderer(80)
	if err != nil {
		t.Fatalf("Failed to create renderer: %v", err)
	}

	tests := []struct {
		name      string
		input     string
		wantLines int // expected output line count
		wantBlank int // index of a blank line
	}{
		{
			name:      "heading + blank + paragraph",
			input:     "# Title\n\nSome text here",
			wantLines: 3,
			wantBlank: 1,
		},
		{
			name:      "two paragraphs with blank",
			input:     "First paragraph\n\nSecond paragraph",
			wantLines: 3,
			wantBlank: 1,
		},
		{
			name:      "heading + blank + list",
			input:     "## Items\n\n- First\n- Second",
			wantLines: 4,
			wantBlank: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := renderer.RenderBlock(tt.input)

			t.Logf("Rendered %d lines:", len(result))
			for i, line := range result {
				plain := ansiEscapeRe.ReplaceAllString(line, "")
				t.Logf("  [%d] plain=%q", i, plain)
			}

			if len(result) != tt.wantLines {
				t.Errorf("Expected %d lines, got %d", tt.wantLines, len(result))
			}

			if tt.wantBlank < len(result) && result[tt.wantBlank] != "" {
				t.Errorf("Expected blank line at index %d, got %q",
					tt.wantBlank, result[tt.wantBlank])
			}
		})
	}
}

// TestRenderBlock_LinkHidesURL verifies that link URLs are hidden in rendered output.
func TestRenderBlock_LinkHidesURL(t *testing.T) {
	renderer, err := NewMarkdownRenderer(80)
	if err != nil {
		t.Fatalf("Failed to create renderer: %v", err)
	}

	result := renderer.RenderLine("[click here](https://example.com)")
	fullOutput := ansiEscapeRe.ReplaceAllString(strings.Join(result, "\n"), "")

	t.Logf("Rendered: %q", fullOutput)

	if !strings.Contains(fullOutput, "click here") {
		t.Error("Link text should be visible")
	}
	// URL should be hidden (rendered in background color, so stripping ANSI + trimming
	// should not show the URL as visible text mixed with other content)
	if strings.Contains(fullOutput, "example.com") {
		t.Error("Link URL should be hidden (rendered in background color)")
	}
}

// TestRenderBlock_TableSeparators verifies table rendering includes separators.
func TestRenderBlock_TableSeparators(t *testing.T) {
	renderer, err := NewMarkdownRenderer(80)
	if err != nil {
		t.Fatalf("Failed to create renderer: %v", err)
	}

	input := "| Name | Value |\n|------|-------|\n| Foo  | 42    |"
	result := renderer.RenderLine(input)
	fullOutput := ansiEscapeRe.ReplaceAllString(strings.Join(result, "\n"), "")

	t.Logf("Table rendered: %q", fullOutput)

	if !strings.Contains(fullOutput, "Name") {
		t.Error("Table should contain header text")
	}
	if !strings.Contains(fullOutput, "│") {
		t.Error("Table should contain column separators (│)")
	}
	if !strings.Contains(fullOutput, "─") {
		t.Error("Table should contain row separators (─)")
	}
}
