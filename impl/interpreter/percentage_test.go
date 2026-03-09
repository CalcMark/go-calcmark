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
		// Basic percentage literals — display as percentage
		{"20% literal", "20%\n", "20%"},
		{"50% literal", "50%\n", "50%"},
		{"100% literal", "100%\n", "100%"},
		{"5% literal", "5%\n", "5%"},

		// Percentage widening: value + pct = value * (1 + pct)
		{"number + percent", "100 + 20%\n", "120"},
		{"number - percent", "100 - 20%\n", "80"},
		// Percentage normalizes to decimal for * and /
		{"number * percent", "100 * 20%\n", "20"},
		{"percent * number", "20% * 100\n", "20"},

		// Complex expressions
		{"discount calc", "100 * (1 - 20%)\n", "80"},
		{"markup calc", "100 * (1 + 20%)\n", "120"},

		// With multipliers — widening applies
		{"1k + 10%", "1k + 10%\n", "1100"},

		// Percentage of expressions (natural syntax)
		{"10% of 200", "10% of 200\n", "20"},
		{"50% of 100", "50% of 100\n", "50"},
		{"25% of 80", "25% of 80\n", "20"},
		{"100% of 50", "100% of 50\n", "50"},
		{"1% of 1000", "1% of 1000\n", "10"},
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
