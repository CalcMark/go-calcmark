package interpreter

import (
	"testing"

	"github.com/CalcMark/go-calcmark/spec/ast"
	"github.com/CalcMark/go-calcmark/spec/parser"
)

func TestRatesWithArbitraryUnits(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"apples per second", "20 apples/sec\n"},
		{"widgets per minute", "100 widgets per minute\n"},
		{"cars per day", "1k cars/day\n"},
		{"items per hour", "500 items/h\n"},
		{"transactions per second", "10k transactions/s\n"},
		{"users per year", "1M users/year\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that arbitrary units PARSE correctly
			nodes, err := parser.Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}

			if len(nodes) == 0 {
				t.Fatal("No nodes")
			}

			rate, ok := nodes[0].(*ast.RateLiteral)
			if !ok {
				t.Fatalf("Expected RateLiteral, got %T", nodes[0])
			}

			t.Logf("✓ Arbitrary unit parses: %s", rate.String())
		})
	}
}
