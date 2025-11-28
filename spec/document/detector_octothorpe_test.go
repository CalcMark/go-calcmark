package document

import (
	"testing"
)

// TestDetectorOctothorpeHandling verifies that lines with inline octothorpe
// are treated as text (not calculations) since they fail to tokenize.
//
// Design decision: Invalid calculation syntax is treated as markdown text,
// not as an error. This follows the principle: "if a line is NOT syntactically
// valid calculation then it's markdown."
func TestDetectorOctothorpeHandling(t *testing.T) {
	tests := []struct {
		name       string
		source     string
		expectText bool // true if line should be treated as text
	}{
		{
			name:       "inline octothorpe after assignment - treated as text",
			source:     "x = 10 # this is invalid",
			expectText: true,
		},
		{
			name:       "inline octothorpe with result annotation - treated as text",
			source:     "result = rtt(local) # → 0.5 ms",
			expectText: true,
		},
		{
			name:       "octothorpe in expression - treated as text",
			source:     "y = 5 + # incomplete",
			expectText: true,
		},
		{
			name:       "valid heading at line start - treated as text",
			source:     "# This is a valid markdown heading",
			expectText: true,
		},
		{
			name:       "valid calculation without octothorpe - is calculation",
			source:     "x = 10 + 20",
			expectText: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detector := NewDetector()
			blocks, err := detector.DetectBlocks(tt.source)

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if len(blocks) != 1 {
				t.Errorf("Expected 1 block, got %d", len(blocks))
				return
			}

			isText := blocks[0].Type() == BlockText
			if isText != tt.expectText {
				expectedType := "calculation"
				if tt.expectText {
					expectedType = "text"
				}
				t.Errorf("Expected %s block, got %v", expectedType, blocks[0].Type())
			}
		})
	}
}
