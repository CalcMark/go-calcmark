package interpreter_test

import (
	"testing"

	"github.com/CalcMark/go-calcmark/impl/interpreter"
	"github.com/CalcMark/go-calcmark/spec/parser"
	"github.com/CalcMark/go-calcmark/spec/types"
)

// TestLogicalOperators tests AND, OR, NOT operators
func TestLogicalOperators(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		// AND operator
		{"true and true", "true and true\n", true},
		{"true and false", "true and false\n", false},
		{"false and true", "false and true\n", false},
		{"false and false", "false and false\n", false},

		// OR operator
		{"true or true", "true or true\n", true},
		{"true or false", "true or false\n", true},
		{"false or true", "false or true\n", true},
		{"false or false", "false or false\n", false},

		// NOT operator
		{"not true", "not true\n", false},
		{"not false", "not false\n", true},

		// Combined expressions
		{"not true or false", "not true or false\n", false},
		{"not false and true", "not false and true\n", true},
		{"true or false and false", "true or false and false\n", true}, // AND has higher precedence
		{"(true or false) and false", "(true or false) and false\n", false},

		// Comparison results with logical ops
		{"1 > 0 and 2 > 1", "1 > 0 and 2 > 1\n", true},
		{"1 > 0 or 0 > 1", "1 > 0 or 0 > 1\n", true},
		{"not (1 > 2)", "not (1 > 2)\n", true},

		// Complex expressions
		{"5 > 3 and 10 < 20 and not false", "5 > 3 and 10 < 20 and not false\n", true},
		{"5 < 3 or 10 > 20 or true", "5 < 3 or 10 > 20 or true\n", true},
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

			boolResult, ok := results[0].(*types.Boolean)
			if !ok {
				t.Fatalf("expected Boolean, got %T (%v)", results[0], results[0])
			}

			if boolResult.Value != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, boolResult.Value)
			}
		})
	}
}

// TestLogicalOperatorErrors tests error cases for logical operators
func TestLogicalOperatorErrors(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"and with number left", "5 and true\n"},
		{"and with number right", "true and 5\n"},
		{"or with number left", "5 or false\n"},
		{"or with number right", "false or 5\n"},
		{"not with number", "not 5\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes, err := parser.Parse(tt.input)
			if err != nil {
				// Parse error is also acceptable for type errors
				return
			}

			interp := interpreter.NewInterpreter()
			_, err = interp.Eval(nodes)
			if err == nil {
				t.Error("expected error but got none")
			}
		})
	}
}

// TestLogicalOperatorPrecedence tests operator precedence (NOT > AND > OR)
func TestLogicalOperatorPrecedence(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		// NOT has highest precedence
		{"not binds tighter than and", "not false and true\n", true}, // (not false) and true = true and true = true
		{"not binds tighter than or", "not true or false\n", false},  // (not true) or false = false or false = false

		// AND has higher precedence than OR
		{"and binds tighter than or", "true or false and false\n", true},   // true or (false and false) = true or false = true
		{"and binds tighter than or 2", "false or true and true\n", true},  // false or (true and true) = false or true = true
		{"and binds tighter than or 3", "false and false or true\n", true}, // (false and false) or true = false or true = true

		// Parentheses override precedence
		{"parens override and", "(true or false) and false\n", false},
		{"parens override not", "not (true and true)\n", false},
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

			boolResult, ok := results[0].(*types.Boolean)
			if !ok {
				t.Fatalf("expected Boolean, got %T", results[0])
			}

			if boolResult.Value != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, boolResult.Value)
			}
		})
	}
}

// TestVariablesWithLogicalOps tests variables in logical expressions
func TestVariablesWithLogicalOps(t *testing.T) {
	interp := interpreter.NewInterpreter()

	// Set up variables
	setupCode := "a = true\nb = false\n"
	nodes, err := parser.Parse(setupCode)
	if err != nil {
		t.Fatalf("failed to parse setup: %v", err)
	}
	_, err = interp.Eval(nodes)
	if err != nil {
		t.Fatalf("failed to eval setup: %v", err)
	}

	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"a and b", "a and b\n", false},
		{"a or b", "a or b\n", true},
		{"not a", "not a\n", false},
		{"not b", "not b\n", true},
		{"a and not b", "a and not b\n", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes, err := parser.Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}

			results, err := interp.Eval(nodes)
			if err != nil {
				t.Fatalf("Eval error: %v", err)
			}

			if len(results) == 0 {
				t.Fatal("No results returned")
			}

			boolResult, ok := results[0].(*types.Boolean)
			if !ok {
				t.Fatalf("expected Boolean, got %T", results[0])
			}

			if boolResult.Value != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, boolResult.Value)
			}
		})
	}
}
