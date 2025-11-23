package interpreter_test

import (
	"strings"
	"testing"

	"github.com/CalcMark/go-calcmark/impl/interpreter"
	"github.com/CalcMark/go-calcmark/spec/parser"
)

// TestComprehensiveFeatures tests all implemented features with valid inputs
func TestComprehensiveFeatures(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// Natural language functions
		{"average of", "average of 1, 2, 3\n", "2"},
		{"average of decimals", "average of 10.5, 20.5\n", "15.5"},
		{"square root of", "square root of 25\n", "5"},
		{"square root of decimal", "square root of 2.25\n", "1.5"},

		// Multipliers
		{"1k", "1k\n", "1000"},
		{"1k + 1", "1k + 1\n", "1001"},
		{"5k + 2k", "5k + 2k\n", "7000"},
		{"1M", "1M\n", "1000000"},
		{"1M + 500k", "1M + 500k\n", "1500000"},
		{"1B", "1B\n", "1000000000"},

		// Arbitrary units
		{"apples", "5 apples + 3 apples\n", "8 apples"},
		{"widgets", "100 widgets - 25 widgets\n", "75 widgets"},
		{"items", "10 items * 5\n", "50 items"},

		// Known units (regression)
		{"meters", "10 meters + 5 meters\n", "15 meters"},
		{"kilograms", "5 kg + 3 kg\n", "8 kg"},
		{"mixed known units", "10 meters + 5 feet\n", "11.524 meters"},

		// Functions with multipliers
		{"avg with k", "avg(1k, 2k, 3k)\n", "2000"},
		{"sqrt of k", "sqrt(1M)\n", "1000"},
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
			if !strings.HasPrefix(actual, tt.expected) {
				t.Errorf("Result = %s, expected to start with %s", actual, tt.expected)
			}
		})
	}
}

// TestRegressionInvalidInputs tests that invalid inputs properly fail
func TestRegressionInvalidInputs(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		shouldError bool
		errorMsg    string
	}{
		// Mismatched arbitrary units
		{"apples vs oranges", "5 apples + 3 oranges\n", true, "incompatible units"},
		{"widgets vs items", "10 widgets + 5 items\n", true, "incompatible units"},

		// Mismatched known units (different dimensions)
		{"meters vs kg", "10 meters + 5 kg\n", true, "incompatible"},

		// Function errors
		{"avg no args", "avg()\n", true, "at least one argument"},
		{"sqrt no args", "sqrt()\n", true, "exactly one argument"},
		{"sqrt too many", "sqrt(1, 2)\n", true, "exactly one argument"},
		{"sqrt negative", "sqrt(-1)\n", true, "negative"},

		// Invalid natural language
		{"average without of", "average 1, 2, 3\n", true, ""},

		// Type mismatches
		{"unit times unit", "5 meters * 3 meters\n", true, "unsupported"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes, err := parser.Parse(tt.input)
			if err != nil {
				if tt.shouldError {
					return // Parse error expected
				}
				t.Fatalf("Unexpected parse error: %v", err)
			}

			interp := interpreter.NewInterpreter()
			_, err = interp.Eval(nodes)

			if tt.shouldError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing %q, got: %v", tt.errorMsg, err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

// TestEdgeCases tests edge cases and boundary conditions
func TestEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		shouldError bool
	}{
		// Zero values
		{"zero apples", "0 apples + 5 apples\n", false},
		{"zero meters", "0 meters + 10 meters\n", false},

		// Large values
		{"large multiply", "1M * 1000\n", false},

		// Decimal arbitrary units
		{"decimal apples", "5.5 apples + 2.5 apples\n", false},

		// Single letter units (edge case)
		{"x unit", "10 x + 5 x\n", false},
		{"y unit", "100 y - 50 y\n", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes, err := parser.Parse(tt.input)
			if err != nil {
				if !tt.shouldError {
					t.Fatalf("Parse error: %v", err)
				}
				return
			}

			interp := interpreter.NewInterpreter()
			_, err = interp.Eval(nodes)

			if tt.shouldError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.shouldError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}
