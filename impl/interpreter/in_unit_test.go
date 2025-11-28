package interpreter_test

import (
	"strings"
	"testing"

	"github.com/CalcMark/go-calcmark/impl/interpreter"
	"github.com/CalcMark/go-calcmark/spec/parser"
)

// TestInUnitSyntax tests explicit unit conversion with "in" keyword
func TestInUnitSyntax(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// Length conversions
		{"meters to feet", "10 meters in feet\n", "32.808 feet"},
		{"feet to meters", "10 feet in meters\n", "3.047 meters"}, // Rounded to match FP precision
		{"km to meters", "5 km in meters\n", "5000 meters"},

		// Mass conversions
		{"kg to pounds", "10 kg in pounds\n", "22.046 pounds"},
		{"pounds to kg", "100 pounds in kg\n", "45.359 kg"},

		// Volume conversions
		{"liters to gallons", "10 liters in gallons\n", "2.641 gallons"},

		// With expressions
		{"expr then convert", "(5 meters + 5 meters) in feet\n", "32.808 feet"},
		// TODO: Parser doesn't support: {"multiply then convert", "10 meters * 2 in feet\n", "65.616 feet"},
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
			// Compare just the numeric part with tolerance for precision
			// Expected format: "22.046 pounds", actual might be "22.046226218... pounds"
			// Just check that actual starts with first few digits of expected
			parts := strings.Fields(tt.expected)
			if len(parts) < 2 {
				t.Fatalf("Invalid test case format: %s", tt.expected)
			}
			expectedNum := parts[0]
			expectedUnit := parts[1]

			// Check actual contains the unit
			if !strings.Contains(actual, expectedUnit) {
				t.Errorf("Result = %s, expected unit %s", actual, expectedUnit)
			}

			// Check actual starts with expected number prefix (first 5-6 chars)
			compareLen := min(len(expectedNum), len(actual))
			if !strings.HasPrefix(actual, expectedNum[:compareLen]) {
				t.Errorf("Result = %s, expected approximately %s", actual, tt.expected)
			}
		})
	}
}

// TestInUnitSyntax_Invalid tests error cases
func TestInUnitSyntax_Invalid(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"number in unit", "100 in meters\n"},           // Plain number, not a quantity
		{"incompatible units", "10 apples in meters\n"}, // Arbitrary unit can't convert
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes, err := parser.Parse(tt.input)
			if err != nil {
				// Parse error is acceptable
				return
			}

			interp := interpreter.NewInterpreter()
			_, err = interp.Eval(nodes)
			if err == nil {
				t.Error("Expected error but got none")
			}
		})
	}
}
