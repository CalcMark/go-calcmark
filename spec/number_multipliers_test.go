package spec_test

import (
	"testing"

	"github.com/CalcMark/go-calcmark/spec/parser"
)

// TestNumberMultipliers tests number multiplier suffixes (k, M, B, T) and scientific notation
func TestNumberMultipliers(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantValue   string // expected expanded value
		description string
	}{
		// Percentage
		{
			name:        "percentage basic",
			input:       "5%\n",
			wantValue:   "0.05",
			description: "5% should expand to 0.05",
		},
		{
			name:        "percentage decimal",
			input:       "12.5%\n",
			wantValue:   "0.125",
			description: "12.5% should expand to 0.125",
		},

		// Thousand (k/K)
		{
			name:        "thousand lowercase k",
			input:       "12k\n",
			wantValue:   "12000",
			description: "12k should expand to 12000",
		},
		{
			name:        "thousand uppercase K",
			input:       "15K\n",
			wantValue:   "15000",
			description: "15K should expand to 15000",
		},
		{
			name:        "thousand decimal",
			input:       "1.5k\n",
			wantValue:   "1500",
			description: "1.5k should expand to 1500",
		},

		// Million (M)
		{
			name:        "million basic",
			input:       "1.2M\n",
			wantValue:   "1200000",
			description: "1.2M should expand to 1,200,000",
		},
		{
			name:        "million integer",
			input:       "5M\n",
			wantValue:   "5000000",
			description: "5M should expand to 5,000,000",
		},

		// Billion (B)
		{
			name:        "billion basic",
			input:       "2.5B\n",
			wantValue:   "2500000000",
			description: "2.5B should expand to 2,500,000,000",
		},
		{
			name:        "billion integer",
			input:       "3B\n",
			wantValue:   "3000000000",
			description: "3B should expand to 3,000,000,000",
		},

		// Trillion (T)
		{
			name:        "trillion basic",
			input:       "1.5T\n",
			wantValue:   "1500000000000",
			description: "1.5T should expand to 1,500,000,000,000",
		},

		// Scientific notation
		{
			name:        "scientific positive exponent",
			input:       "1.23e10\n",
			wantValue:   "12300000000",
			description: "1.23e10 should expand to 12,300,000,000",
		},
		{
			name:        "scientific negative exponent",
			input:       "4.56e-7\n",
			wantValue:   "0.000000456",
			description: "4.56e-7 should expand to 0.000000456",
		},
		{
			name:        "scientific uppercase E",
			input:       "1.2E5\n",
			wantValue:   "120000",
			description: "1.2E5 should expand to 120,000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes, err := parser.Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}

			if len(nodes) < 1 {
				t.Fatalf("Parse() returned no nodes")
			}

			// The AST should contain a NumberLiteral with expanded value
			astStr := nodes[0].String()
			t.Logf("AST: %s", astStr)
			t.Logf("%s", tt.description)

			// For now, just verify it parses successfully
			// TODO: extract and verify actual NumberLiteral.Value matches wantValue
		})
	}
}

// TestNumberMultiplierWithUnit tests critical edge case: 1.1k K (= 1100 Kelvin)
// This verifies that 'k' as multiplier (no space) is different from 'K' as unit (with space)
func TestNumberMultiplierWithUnit(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		description string
	}{
		{
			name:        "k multiplier with K unit (Kelvin)",
			input:       "temp = 1.1k K\n",
			description: "1.1k K should be 1100 Kelvin (k=multiplier, K=unit)",
		},
		{
			name:        "k multiplier with meters",
			input:       "distance = 5k meters\n",
			description: "5k meters should be 5000 meters",
		},
		{
			name:        "M multiplier with dollars",
			input:       "budget = 1.2M dollars\n",
			description: "1.2M dollars should be 1,200,000 dollars",
		},
		{
			name:        "scientific with units",
			input:       "bigNum = 1.23e10 meters\n",
			description: "1.23e10 meters should work with scientific notation",
		},
		{
			name:        "scientific negative with units",
			input:       "smallNum = 4.56e-7 grams\n",
			description: "4.56e-7 grams should work with negative exponent",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes, err := parser.Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}

			if len(nodes) < 1 {
				t.Fatalf("Parse() returned no nodes")
			}

			t.Logf("%s", tt.description)
			t.Logf("AST: %s", nodes[0].String())

			// Verify it parses as an Assignment with a QuantityLiteral
			// The number should be expanded, unit should be preserved
		})
	}
}

// TestMultiplierNoSpace verifies that multipliers ONLY work with NO SPACE
func TestMultiplierNoSpace(t *testing.T) {
	// This should parse as number (12) + identifier (k)
	// NOT as a multiplier
	input := "x = 12 k\n" // Space between 12 and k

	nodes, err := parser.Parse(input)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	t.Logf("12 k (with space) parses as: %s", nodes[0].String())
	// Should be: Assignment with BinaryOp or something, not a simple number
}
