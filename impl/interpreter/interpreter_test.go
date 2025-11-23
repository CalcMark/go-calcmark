package interpreter

import (
	"testing"

	"github.com/CalcMark/go-calcmark/spec/ast"
	"github.com/CalcMark/go-calcmark/spec/types"
	"github.com/shopspring/decimal"
)

// Test pure helper functions

func TestExpandNumberLiteral(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"42", "42"},
		{"1k", "1000"},
		{"1.2k", "1200"},
		{"1M", "1000000"},
		{"1.5M", "1500000"},
		{"1B", "1000000000"},
		{"2.5B", "2500000000"},
		{"1e3", "1000"},
		{"1.2e6", "1200000"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := expandNumberLiteral(tt.input)
			if err != nil {
				t.Fatalf("expandNumberLiteral(%q) error = %v", tt.input, err)
			}
			if got.String() != tt.want {
				t.Errorf("expandNumberLiteral(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseBooleanValue(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"true", true},
		{"TRUE", true},
		{"false", false},
		{"FALSE", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := parseBooleanValue(tt.input)
			if err != nil {
				t.Fatalf("parseBooleanValue(%q) error = %v", tt.input, err)
			}
			if got != tt.want {
				t.Errorf("parseBooleanValue(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

// TestParseBooleanValue_InvalidInputs verifies that y/n/t/yes/no are NOT parsed as booleans
func TestParseBooleanValue_InvalidInputs(t *testing.T) {
	invalidInputs := []string{"y", "n", "t", "yes", "no", "YES", "NO", "1", "0"}

	for _, input := range invalidInputs {
		t.Run(input, func(t *testing.T) {
			_, err := parseBooleanValue(input)
			if err == nil {
				t.Errorf("parseBooleanValue(%q) should return error but didn't", input)
			}
		})
	}
}

func TestParseMonth(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"Jan", 1},
		{"January", 1},
		{"Feb", 2},
		{"December", 12},
		{"dec", 12},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := parseMonth(tt.input)
			if err != nil {
				t.Fatalf("parseMonth(%q) error = %v", tt.input, err)
			}
			if got != tt.want {
				t.Errorf("parseMonth(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

// Test literal evaluation

func TestEvalNumberLiteral(t *testing.T) {
	interp := NewInterpreter()

	node := &ast.NumberLiteral{Value: "1.2k"}
	result, err := interp.evalNumberLiteral(node)
	if err != nil {
		t.Fatalf("evalNumberLiteral error = %v", err)
	}

	if result.String() != "1200" {
		t.Errorf("Result = %v, want 1200", result.String())
	}
}

func TestEvalCurrencyLiteral(t *testing.T) {
	interp := NewInterpreter()

	node := &ast.CurrencyLiteral{
		Value:  "100",
		Symbol: "$",
	}

	result, err := interp.evalCurrencyLiteral(node)
	if err != nil {
		t.Fatalf("evalCurrencyLiteral error = %v", err)
	}

	if result.String() != "$100.00" {
		t.Errorf("Result = %v, want $100.00", result.String())
	}
}

// Test binary operations

func TestEvalNumberOperation(t *testing.T) {
	tests := []struct {
		name     string
		left     string
		right    string
		operator string
		want     string
	}{
		{"addition", "10", "5", "+", "15"},
		{"subtraction", "10", "5", "-", "5"},
		{"multiplication", "10", "5", "*", "50"},
		{"division", "10", "5", "/", "2"},
		{"modulus", "10", "3", "%", "1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			left, _ := decimal.NewFromString(tt.left)
			right, _ := decimal.NewFromString(tt.right)

			leftNum := &types.Number{Value: left}
			rightNum := &types.Number{Value: right}

			result, err := evalNumberOperation(leftNum, rightNum, tt.operator)
			if err != nil {
				t.Fatalf("evalNumberOperation error = %v", err)
			}

			if result.String() != tt.want {
				t.Errorf("Result = %v, want %v", result.String(), tt.want)
			}
		})
	}
}

// Test environment

func TestEnvironment(t *testing.T) {
	env := NewEnvironment()

	// Test setting and getting
	env.Set("x", nil)
	if !env.Has("x") {
		t.Error("Expected variable 'x' to be defined")
	}

	// Test undefined variable
	if env.Has("y") {
		t.Error("Expected variable 'y' to be undefined")
	}
}

// Test variable assignment and lookup

func TestAssignmentAndLookup(t *testing.T) {
	interp := NewInterpreter()

	// x = 42
	assignment := &ast.Assignment{
		Name:  "x",
		Value: &ast.NumberLiteral{Value: "42"},
	}

	_, err := interp.evalAssignment(assignment)
	if err != nil {
		t.Fatalf("evalAssignment error = %v", err)
	}

	// Look up x
	id := &ast.Identifier{Name: "x"}
	result, err := interp.evalIdentifier(id)
	if err != nil {
		t.Fatalf("evalIdentifier error = %v", err)
	}

	if result.String() != "42" {
		t.Errorf("Result = %v, want 42", result.String())
	}
}
