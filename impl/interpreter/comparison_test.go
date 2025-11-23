package interpreter_test

import (
	"testing"

	"github.com/CalcMark/go-calcmark/impl/interpreter"
	"github.com/CalcMark/go-calcmark/spec/parser"
)

// TestComparisonOperations tests if comparison operators work
func TestComparisonOperations(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"equal true", "5 == 5\n", "true"},
		{"equal false", "5 == 3\n", "false"},
		{"not equal true", "5 != 3\n", "true"},
		{"not equal false", "5 != 5\n", "false"},
		{"greater than true", "10 > 5\n", "true"},
		{"greater than false", "5 > 10\n", "false"},
		{"less than true", "3 < 7\n", "true"},
		{"less than false", "7 < 3\n", "false"},
		{"gte true", "5 >= 5\n", "true"},
		{"lte true", "5 <= 5\n", "true"},
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
