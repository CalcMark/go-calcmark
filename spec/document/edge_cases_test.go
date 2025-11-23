package document

import (
	"strings"
	"testing"
)

// TestEdgeCases explores surprising/complex block detection scenarios
func TestEdgeCases(t *testing.T) {
	detector := NewDetector()

	tests := []struct {
		name          string
		source        string
		expectedCount int
		expectedTypes []BlockType
		description   string
	}{
		{
			name: "two markdown blocks with 2 empty lines",
			source: `# Header 1


# Header 2`,
			expectedCount: 2,
			expectedTypes: []BlockType{BlockText, BlockText},
			description:   "Removing 1 empty line would MERGE these into 1 block",
		},
		{
			name: "two markdown blocks with 1 empty line",
			source: `# Header 1

# Header 2`,
			expectedCount: 1,
			expectedTypes: []BlockType{BlockText},
			description:   "Already merged - single text block",
		},
		{
			name: "markdown with empty line between calc blocks",
			source: `x = 10

# Comment

y = 20`,
			expectedCount: 3,
			expectedTypes: []BlockType{BlockCalculation, BlockText, BlockCalculation},
			description:   "Empty lines don't cross type boundaries",
		},
		{
			name: "calc block with empty line inside",
			source: `x = 10

y = 20`,
			expectedCount: 1,
			expectedTypes: []BlockType{BlockCalculation},
			description:   "Single empty line keeps calc block together",
		},
		{
			name: "trailing empty lines",
			source: `x = 10


`,
			expectedCount: 1,
			expectedTypes: []BlockType{BlockCalculation},
			description:   "Trailing empty lines don't create new blocks",
		},
		{
			name: "leading empty lines",
			source: `

x = 10`,
			expectedCount: 1,
			expectedTypes: []BlockType{BlockCalculation},
			description:   "Leading empty lines are ignored or part of first block",
		},
		{
			name: "only empty lines",
			source: `


`,
			expectedCount: 0,
			description:   "Document of only empty lines produces no blocks",
		},
		{
			name: "alternating calc and text with single empty lines",
			source: `x = 10

Some text

y = 20

More text`,
			expectedCount: 4,
			expectedTypes: []BlockType{BlockCalculation, BlockText, BlockCalculation, BlockText},
			description:   "Type changes always create new blocks regardless of empty lines",
		},
		{
			name: "three empty lines (extra boundary)",
			source: `x = 10



y = 20`,
			expectedCount: 2,
			expectedTypes: []BlockType{BlockCalculation, BlockCalculation},
			description:   "3 empty lines = definite boundary",
		},
		{
			name: "markdown list that looks like calc",
			source: `- item 1
- item 2
x = 10`,
			expectedCount: 2,
			expectedTypes: []BlockType{BlockText, BlockCalculation},
			description:   "Markdown list stays as text, then calc",
		},
		{
			name: "single word on line (ambiguous)",
			source: `hello
x = 10`,
			expectedCount: 2,
			expectedTypes: []BlockType{BlockText, BlockCalculation},
			description:   "Single word defaults to text in current heuristic",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blocks, err := detector.DetectBlocks(tt.source)
			if err != nil {
				t.Fatalf("DetectBlocks() error = %v", err)
			}

			if len(blocks) != tt.expectedCount {
				t.Errorf("Expected %d blocks, got %d\nDescription: %s\nSource:\n%s",
					tt.expectedCount, len(blocks), tt.description, tt.source)

				// Show what we got
				for i, block := range blocks {
					t.Logf("  Block %d: %v, source: %q", i, block.Type(), strings.Join(block.Source(), "\\n"))
				}
			}

			for i, expectedType := range tt.expectedTypes {
				if i >= len(blocks) {
					break
				}
				if blocks[i].Type() != expectedType {
					t.Errorf("Block %d: expected %v, got %v", i, expectedType, blocks[i].Type())
				}
			}
		})
	}
}

// TestBlockMerging tests the user's scenario: can blocks be merged by removing empty lines?
func TestBlockMerging(t *testing.T) {
	detector := NewDetector()

	// Scenario 1: Two markdown blocks
	twoBlocks := `# Header 1


# Header 2`

	oneBlock := `# Header 1

# Header 2`

	blocks1, _ := detector.DetectBlocks(twoBlocks)
	blocks2, _ := detector.DetectBlocks(oneBlock)

	if len(blocks1) != 2 {
		t.Errorf("Expected 2 blocks with double empty line, got %d", len(blocks1))
	}

	if len(blocks2) != 1 {
		t.Errorf("Expected 1 merged block with single empty line, got %d", len(blocks2))
	}

	t.Log("✅ Confirmed: Removing 1 empty line from 2-empty-line boundary MERGES blocks")
}

// TestBlockSplitting tests the inverse: can a block be split by adding empty lines?
func TestBlockSplitting(t *testing.T) {
	detector := NewDetector()

	// Start with single calc block
	oneBlock := `x = 10

y = 20`

	// Add empty line to split
	twoBlocks := `x = 10


y = 20`

	blocks1, _ := detector.DetectBlocks(oneBlock)
	blocks2, _ := detector.DetectBlocks(twoBlocks)

	if len(blocks1) != 1 {
		t.Errorf("Expected 1 block with single empty line, got %d", len(blocks1))
	}

	if len(blocks2) != 2 {
		t.Errorf("Expected 2 blocks with double empty line, got %d", len(blocks2))
	}

	t.Log("✅ Confirmed: Adding empty line (1→2) SPLITS a block")
}
