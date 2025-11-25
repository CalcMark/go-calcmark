package parser

import (
	"testing"

	"github.com/CalcMark/go-calcmark/spec/ast"
)

func TestRateWithMultipliers(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
		description string
	}{
		{
			name:        "1k with rate slash",
			input:       "1k m/s\n",
			expectError: false,
			description: "1k meters per second",
		},
		{
			name:        "1M with rate slash",
			input:       "1M req/s\n",
			expectError: false,
			description: "1 million requests per second",
		},
		{
			name:        "100k with per keyword",
			input:       "100k requests per minute\n",
			expectError: false,
			description: "100k requests per minute",
		},
		{
			name:        "1.5M with rate",
			input:       "1.5M GB/day\n",
			expectError: false,
			description: "1.5 million GB per day",
		},
		{
			name:        "scientific notation with rate",
			input:       "1.2e6 bytes/s\n",
			expectError: false,
			description: "1.2e6 bytes per second",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes, err := Parse(tt.input)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
				return
			}

			if !tt.expectError && err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if !tt.expectError {
				if len(nodes) == 0 {
					t.Fatal("No nodes returned")
				}

				// Should be a RateLiteral
				rate, ok := nodes[0].(*ast.RateLiteral)
				if !ok {
					t.Fatalf("Expected RateLiteral, got %T", nodes[0])
				}

				t.Logf("✓ %s: %s", tt.description, rate.String())
			}
		})
	}
}
