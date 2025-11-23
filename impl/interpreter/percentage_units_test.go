package interpreter_test

import (
	"testing"

	"github.com/CalcMark/go-calcmark/impl/interpreter"
	"github.com/CalcMark/go-calcmark/spec/parser"
)

// TestPercentageWithUnits tests if percentages work with units
func TestPercentageWithUnits(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		expected   string
		shouldWork bool
	}{
		// Quantity * percentage should preserve unit
		{"1 lb * 20%", "1 lb * 20%\n", "0.2 lb", true},
		{"100 meters * 50%", "100 meters * 50%\n", "50 meters", true},
		{"10 apples * 20%", "10 apples * 20%\n", "2 apples", true},

		// Discount: quantity * (1 - percent)
		{"20% off 100 meters", "100 meters * (1 - 20%)\n", "80 meters", true},
		{"50% off 10 apples", "10 apples * (1 - 50%)\n", "5 apples", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes, err := parser.Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}

			interp := interpreter.NewInterpreter()
			results, err := interp.Eval(nodes)

			if tt.shouldWork {
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
			} else {
				if err == nil {
					t.Error("Expected error but got none")
				}
			}
		})
	}
}
