package interpreter_test

import (
	"testing"

	"github.com/CalcMark/go-calcmark/impl/interpreter"
	"github.com/CalcMark/go-calcmark/spec/parser"
)

// TestAllInterpreterFeatures provides comprehensive coverage of all interpreter capabilities
func TestAllInterpreterFeatures(t *testing.T) {
	tests := []struct {
		category string
		name     string
		input    string
		expected string
	}{
		// Basic arithmetic
		{"arithmetic", "addition", "1 + 2\n", "3"},
		{"arithmetic", "subtraction", "10 - 5\n", "5"},
		{"arithmetic", "multiplication", "3 * 4\n", "12"},
		{"arithmetic", "division", "20 / 4\n", "5"},
		{"arithmetic", "modulus", "10 % 3\n", "1"},
		{"arithmetic", "exponent", "2 ^ 3\n", "8"},

		// Number multipliers
		{"multipliers", "k", "1k\n", "1000"},
		{"multipliers", "M", "1M\n", "1000000"},
		{"multipliers", "B", "1B\n", "1000000000"},
		{"multipliers", "arithmetic with k", "5k + 2k\n", "7000"},

		// Percentages
		{"percentages", "literal", "20%\n", "0.2"},
		{"percentages", "multiplication", "100 * 20%\n", "20"},
		{"percentages", "discount", "100 * (1 - 20%)\n", "80"},
		{"percentages", "markup", "100 * (1 + 20%)\n", "120"},

		// Functions
		{"functions", "avg", "avg(1, 2, 3)\n", "2"},
		{"functions", "sqrt", "sqrt(25)\n", "5"},
		{"functions", "natural avg", "average of 10, 20, 30\n", "20"},
		{"functions", "natural sqrt", "square root of 100\n", "10"},

		// Arbitrary units
		{"units", "apples", "5 apples + 3 apples\n", "8 apples"},
		{"units", "widgets multiply", "10 widgets * 2\n", "20 widgets"},
		{"units", "number + unit", "1 + 1 dogs\n", "2 dogs"},

		// Known units
		{"units", "meters", "10 meters + 5 meters\n", "15 meters"},
		{"units", "unit conversion", "10 meters + 5 feet\n", "11.524 meters"},
		{"units", "kilograms", "5 kg + 3 kg\n", "8 kg"},

		// Boolean
		{"boolean", "true", "true\n", "true"},
		{"boolean", "false", "false\n", "false"},

		// Variables
		{"variables", "assignment", "x = 42\nx\n", "42"},
		{"variables", "expression", "y = 10 + 20\ny\n", "30"},
	}

	for _, tt := range tests {
		t.Run(tt.category+"/"+tt.name, func(t *testing.T) {
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

			// Get last result (for variable tests)
			actual := results[len(results)-1].String()
			if actual != tt.expected {
				t.Errorf("Result = %s, expected %s", actual, tt.expected)
			}
		})
	}
}
