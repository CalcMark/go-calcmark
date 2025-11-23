package interpreter_test

import (
	"testing"

	"github.com/CalcMark/go-calcmark/impl/interpreter"
	"github.com/CalcMark/go-calcmark/spec/parser"
)

// TestPercentageCalculations tests percentage operations
func TestPercentageCalculations(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// Basic percentage literals
		{"20% literal", "20%\n", "0.2"},
		{"50% literal", "50%\n", "0.5"},
		{"100% literal", "100%\n", "1"},
		{"5% literal", "5%\n", "0.05"},

		// Percentage arithmetic with numbers
		// For now, percentages are just decimal numbers
		// 100 + 20% means 100 + 0.20 = 100.20 (literal interpretation)
		{"number + percent", "100 + 20%\n", "100.2"},
		{"number - percent", "100 - 20%\n", "99.8"},
		{"number * percent", "100 * 20%\n", "20"},
		{"percent * number", "20% * 100\n", "20"},

		// Complex expressions
		{"discount calc", "100 * (1 - 20%)\n", "80"},
		{"markup calc", "100 * (1 + 20%)\n", "120"},

		// With multipliers
		{"1k + 10%", "1k + 10%\n", "1000.1"},
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
