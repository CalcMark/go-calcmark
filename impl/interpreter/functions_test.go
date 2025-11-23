package interpreter_test

import (
	"testing"

	"github.com/CalcMark/go-calcmark/impl/interpreter"
	"github.com/CalcMark/go-calcmark/spec/parser"
)

func TestFunctionEvaluation(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"avg with 3 args", "avg(1, 2, 3)\n", "2"},
		{"avg with 5 args", "avg(10, 20, 30, 40, 50)\n", "30"},
		{"avg with 1 arg", "avg(42)\n", "42"},
		{"sqrt of 9", "sqrt(9)\n", "3"},
		{"sqrt of 16", "sqrt(16)\n", "4"},
		{"sqrt of 2.25", "sqrt(2.25)\n", "1.5"},
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

func TestFunctionErrors(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"avg no args", "avg()\n"},
		{"sqrt no args", "sqrt()\n"},
		{"sqrt too many args", "sqrt(1, 2)\n"},
		{"sqrt negative", "sqrt(-1)\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes, err := parser.Parse(tt.input)
			if err != nil {
				// Parse error is acceptable for some invalid syntax
				return
			}

			interp := interpreter.NewInterpreter()
			_, err = interp.Eval(nodes)
			if err == nil {
				t.Errorf("Expected error for %q but got none", tt.input)
			}
		})
	}
}
