package interpreter_test

import (
	"testing"

	"github.com/CalcMark/go-calcmark/impl/interpreter"
	"github.com/CalcMark/go-calcmark/spec/parser"
)

// TestArbitraryUnits verifies arbitrary user-defined units work
func TestArbitraryUnits(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"apples", "5 apples + 3 apples\n", "8 apples"},
		{"widgets", "10 widgets - 4 widgets\n", "6 widgets"},
		{"items", "100 items + 50 items\n", "150 items"},
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

// TestArbitraryUnits_Mismatch verifies mismatched units produce errors
func TestArbitraryUnits_Mismatch(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"apples vs oranges", "5 apples + 3 oranges\n"},
		{"widgets vs items", "10 widgets + 5 items\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes, err := parser.Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}

			interp := interpreter.NewInterpreter()
			_, err = interp.Eval(nodes)
			if err == nil {
				t.Errorf("Expected error for mismatched units but got none")
			}
		})
	}
}
