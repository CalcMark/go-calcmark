package interpreter_test

import (
	"testing"

	"github.com/CalcMark/go-calcmark/impl/interpreter"
	"github.com/CalcMark/go-calcmark/spec/parser"
)

// TestNaturalLanguageFunctions verifies "average of" and "square root of" syntax
func TestNaturalLanguageFunctions(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"average of 3 nums", "average of 1, 2, 3\n", "2"},
		{"average of 5 nums", "average of 10, 20, 30, 40, 50\n", "30"},
		{"square root of 25", "square root of 25\n", "5"},
		{"square root of 100", "square root of 100\n", "10"},
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
