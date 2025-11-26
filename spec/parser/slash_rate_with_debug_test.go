package parser

import (
	"testing"
)

// TestSlashRateWithKeywordDebug - systematic debugging of the parse error
func TestSlashRateWithKeywordDebug(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
		description string
	}{
		{
			name:        "baseline: plain quantities",
			input:       "10000 req with 450 req\n",
			expectError: false,
			description: "This works - establishes baseline",
		},
		{
			name:        "baseline: TB quantities",
			input:       "10 TB with 2 TB\n",
			expectError: false,
			description: "This works - another baseline",
		},
		{
			name:        "test: single slash-rate",
			input:       "10000 req/s\n",
			expectError: false,
			description: "Just the slash-rate alone - should work",
		},
		{
			name:        "test: slash-rate WITH plain quantity",
			input:       "10000 req/s with 450 req\n",
			expectError: false,
			description: "Slash-rate followed by WITH and plain quantity",
		},
		{
			name:        "PROBLEM: slash-rate WITH slash-rate",
			input:       "10000 req/s with 450 req/s\n",
			expectError: false, // NOW FIXED!
			description: "This was the failing case - now fixed!",
		},
		{
			name:        "test: per-rate WITH per-rate",
			input:       "10000 req per s with 450 req per s\n",
			expectError: false, // NOW FIXED!
			description: "Per-style rates also work with WITH now",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Testing: %s", tt.description)
			t.Logf("Input: %q", tt.input)

			nodes, err := Parse(tt.input)

			if tt.expectError {
				if err == nil {
					t.Logf("⚠️  Expected error but parse succeeded")
					t.Logf("    Result: %+v", nodes)
				} else {
					t.Logf("✓ Got expected error: %v", err)
				}
			} else {
				if err != nil {
					t.Errorf("❌ Unexpected error: %v", err)
				} else {
					t.Logf("✓ Parse succeeded: %T", nodes[0])
				}
			}
		})
	}
}
