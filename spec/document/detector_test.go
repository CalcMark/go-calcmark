package document

import (
	"testing"
)

func TestBlockDetection(t *testing.T) {
	detector := NewDetector()

	tests := []struct {
		name          string
		source        string
		expectedTypes []BlockType
		expectedCount int
	}{
		{
			name:          "single calc line",
			source:        "x = 10",
			expectedTypes: []BlockType{BlockCalculation},
			expectedCount: 1,
		},
		{
			name:          "two calc lines",
			source:        "x = 10\ny = 20",
			expectedTypes: []BlockType{BlockCalculation},
			expectedCount: 1,
		},
		{
			name:          "calc then text",
			source:        "x = 10\n# Header",
			expectedTypes: []BlockType{BlockCalculation, BlockText},
			expectedCount: 2,
		},
		{
			name:          "calc with 1 empty line (stays in block)",
			source:        "x = 10\n\ny = 20",
			expectedTypes: []BlockType{BlockCalculation},
			expectedCount: 1,
		},
		{
			name:          "calc with 2 empty lines (splits blocks)",
			source:        "x = 10\n\n\ny = 20",
			expectedTypes: []BlockType{BlockCalculation, BlockCalculation},
			expectedCount: 2,
		},
		{
			name:          "text then calc then text",
			source:        "# Header\nx = 10\nMore text",
			expectedTypes: []BlockType{BlockText, BlockCalculation, BlockText},
			expectedCount: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blocks, err := detector.DetectBlocks(tt.source)
			if err != nil {
				t.Fatalf("DetectBlocks() error = %v", err)
			}

			if len(blocks) != tt.expectedCount {
				t.Errorf("Expected %d blocks, got %d", tt.expectedCount, len(blocks))
			}

			for i, expectedType := range tt.expectedTypes {
				if i >= len(blocks) {
					break
				}
				if blocks[i].Type() != expectedType {
					t.Errorf("Block %d: expected type %v, got %v", i, expectedType, blocks[i].Type())
				}
			}
		})
	}
}

func TestEmptyLineDelimiter(t *testing.T) {
	detector := NewDetector()

	// 2 consecutive empty lines = block boundary
	source := `x = 10


y = 20`

	blocks, err := detector.DetectBlocks(source)
	if err != nil {
		t.Fatalf("DetectBlocks() error = %v", err)
	}

	if len(blocks) != 2 {
		t.Errorf("Expected 2 blocks (split by 2 empty lines), got %d", len(blocks))
	}

	// Both should be calc blocks
	for i, block := range blocks {
		if block.Type() != BlockCalculation {
			t.Errorf("Block %d should be calculation, got %v", i, block.Type())
		}
	}
}
