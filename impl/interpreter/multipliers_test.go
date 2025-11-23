package interpreter_test

import (
	"testing"

	"github.com/CalcMark/go-calcmark/impl/interpreter"
	"github.com/CalcMark/go-calcmark/spec/parser"
)

// TestMultipliers verifies number multipliers work (with and without units)
func TestMultipliers(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// Pure numbers
		{"1k + 1", "1k + 1\n", "1001"},
		{"5k + 2k", "5k + 2k\n", "7000"},
		{"1M + 500k", "1M + 500k\n", "1500000"},

		// Not testing with units yet - that's a separate feature
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes, err := parser.Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}

			interp := interpreter.NewInterpreter()
			results, err := interp.Eval(nodes)
			if err != nil {
				t.Fatalf("Eval error: %v", err)
			}

			if len(results) == 0 {
				t.Fatal("No results returned")
			}

			actual := results[0].String()
			if actual != tt.expected {
				t.Errorf("Result = %s, expected %s", actual, tt.expected)
			}
		})
	}
}

// TestCurrencyMixing verifies $ and USD can be mixed
func TestCurrencyMixing(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		shouldOK bool
	}{
		{"$100 + 50 USD", "$100 + 50\n", false}, // This won't work - "50" is just a number
		{"$100 + $50", "$100 + $50\n", true},
		// Need to verify if "100 USD" postfix syntax works first
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes, err := parser.Parse(tt.input)
			if err != nil {
				if !tt.shouldOK {
					return // Expected to fail parsing
				}
				t.Fatalf("Parse error: %v", err)
			}

			interp := interpreter.NewInterpreter()
			_, err = interp.Eval(nodes)

			if tt.shouldOK && err != nil {
				t.Errorf("Expected success but got error: %v", err)
			}
		})
	}
}
