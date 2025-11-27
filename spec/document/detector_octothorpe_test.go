package document

import (
	"strings"
	"testing"
)

// TestDetectorOctothorpeError verifies that lines with inline octothorpe
// are properly rejected during block detection.
// TDD: This test should FAIL initially, then PASS after fix.
func TestDetectorOctothorpeError(t *testing.T) {
	tests := []struct {
		name        string
		source      string
		shouldError bool
		errorMsg    string
	}{
		{
			name:        "inline octothorpe after assignment",
			source:      "x = 10 # this is invalid",
			shouldError: true,
			errorMsg:    "octothorpe",
		},
		{
			name:        "inline octothorpe with result annotation",
			source:      "result = rtt(local) # → 0.5 ms",
			shouldError: true,
			errorMsg:    "octothorpe",
		},
		{
			name:        "octothorpe in expression",
			source:      "y = 5 + # incomplete",
			shouldError: true,
			errorMsg:    "octothorpe",
		},
		{
			name:        "valid heading at line start",
			source:      "# This is a valid markdown heading",
			shouldError: false,
		},
		{
			name:        "valid calculation without octothorpe",
			source:      "x = 10 + 20",
			shouldError: false,
		},
		{
			name:        "multiple lines with octothorpe error",
			source:      "x = 10\ny = 20 # bad comment\nz = 30",
			shouldError: true,
			errorMsg:    "octothorpe",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detector := NewDetector()
			blocks, err := detector.DetectBlocks(tt.source)

			if tt.shouldError {
				if err == nil {
					t.Errorf("Expected error for %q, but got none. Blocks: %v", tt.source, blocks)
					return
				}
				if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing %q, got: %v", tt.errorMsg, err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for %q: %v", tt.source, err)
				}
			}
		})
	}
}
