package interpreter_test

import (
	"testing"

	"github.com/CalcMark/go-calcmark/impl/interpreter"
	"github.com/CalcMark/go-calcmark/spec/parser"
)

// TestQuantityNumberOperations verifies number * quantity operations
func TestQuantityNumberOperations(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// Number + Quantity should work if only one operand has a unit
		{"1 + 1 dogs", "1 + 1 dogs\n", "2 dogs"},
		{"1 dogs + 1", "1 dogs + 1\n", "2 dogs"},

		// Multiplication with units
		{"quantity * number", "10 dogs * 2\n", "20 dogs"},
		{"number * quantity", "2 * 10 dogs\n", "20 dogs"},
		{"fractional", "10 dogs * 0.5\n", "5 dogs"},

		// Division with units
		{"quantity / number", "10 dogs / 2\n", "5 dogs"},
		{"half a dog", "1 dogs * 0.5\n", "0.5 dogs"}, // Nonsensical but allowed
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
