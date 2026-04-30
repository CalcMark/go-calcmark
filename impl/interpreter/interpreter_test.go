package interpreter

import (
	"errors"
	"fmt"
	"testing"

	"github.com/CalcMark/go-calcmark/v2/spec/ast"
	"github.com/CalcMark/go-calcmark/v2/spec/types"
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

// TestBuiltinConstants verifies PI and E are pre-defined.
func TestBuiltinConstants(t *testing.T) {
	interp := NewInterpreter()

	tests := []struct {
		name   string
		prefix string // Expected prefix of the value
	}{
		{"PI", "3.14159265358979"},
		{"E", "2.71828182845904"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id := &ast.Identifier{Name: tt.name}
			result, err := interp.evalIdentifier(id)
			if err != nil {
				t.Fatalf("evalIdentifier(%q) error = %v", tt.name, err)
			}

			num, ok := result.(*types.Number)
			if !ok {
				t.Fatalf("Expected *types.Number, got %T", result)
			}

			str := num.String()
			if len(str) < len(tt.prefix) || str[:len(tt.prefix)] != tt.prefix {
				t.Errorf("%s = %v, want prefix %v", tt.name, str, tt.prefix)
			}
		})
	}
}

// TestBuiltinConstantsUsage verifies PI and E can be used in expressions.
func TestBuiltinConstantsUsage(t *testing.T) {
	interp := NewInterpreter()

	// Test: 2 * PI (should be ~6.28...)
	node := &ast.BinaryOp{
		Operator: "*",
		Left:     &ast.NumberLiteral{Value: "2"},
		Right:    &ast.Identifier{Name: "PI"},
	}

	result, err := interp.evalBinaryOp(node)
	if err != nil {
		t.Fatalf("evalBinaryOp error = %v", err)
	}

	num, ok := result.(*types.Number)
	if !ok {
		t.Fatalf("Expected *types.Number, got %T", result)
	}

	// 2*PI should be approximately 6.28
	val := num.Value.InexactFloat64()
	if val < 6.28 || val > 6.29 {
		t.Errorf("2 * PI = %v, want ~6.28", val)
	}
}

// Test CascadingError type

func TestCascadingError_ErrorMessage(t *testing.T) {
	cause := fmt.Errorf("division by zero")
	ce := &CascadingError{VarName: "x", Cause: cause}

	want := `depends on errored variable "x": division by zero`
	if got := ce.Error(); got != want {
		t.Errorf("CascadingError.Error() = %q, want %q", got, want)
	}
}

func TestCascadingError_Unwrap(t *testing.T) {
	cause := fmt.Errorf("division by zero")
	ce := &CascadingError{VarName: "x", Cause: cause}

	if unwrapped := ce.Unwrap(); unwrapped != cause {
		t.Errorf("CascadingError.Unwrap() = %v, want %v", unwrapped, cause)
	}
}

func TestCascadingError_ErrorsAs(t *testing.T) {
	cause := fmt.Errorf("division by zero")
	var err error = &CascadingError{VarName: "x", Cause: cause}

	var ce *CascadingError
	if !errors.As(err, &ce) {
		t.Fatal("errors.As failed to match CascadingError")
	}
	if ce.VarName != "x" {
		t.Errorf("VarName = %q, want %q", ce.VarName, "x")
	}
}

// Test evalIdentifier with errored variables

func TestEvalIdentifier_ErroredVariable(t *testing.T) {
	interp := NewInterpreter()
	cause := fmt.Errorf("division by zero")
	interp.env.SetError("x", cause)

	id := &ast.Identifier{Name: "x"}
	_, err := interp.evalIdentifier(id)
	if err == nil {
		t.Fatal("expected error for errored variable, got nil")
	}

	var ce *CascadingError
	if !errors.As(err, &ce) {
		t.Fatalf("expected CascadingError, got %T: %v", err, err)
	}
	if ce.VarName != "x" {
		t.Errorf("VarName = %q, want %q", ce.VarName, "x")
	}
	if ce.Cause != cause {
		t.Errorf("Cause = %v, want %v", ce.Cause, cause)
	}
}

func TestEvalIdentifier_NormalVariable_Unchanged(t *testing.T) {
	interp := NewInterpreter()
	interp.env.Set("y", types.NewNumber(decimal.NewFromInt(42)))

	id := &ast.Identifier{Name: "y"}
	result, err := interp.evalIdentifier(id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.String() != "42" {
		t.Errorf("result = %v, want 42", result.String())
	}
}

func TestEvalIdentifier_ErroredTakesPrecedenceOverValue(t *testing.T) {
	interp := NewInterpreter()
	cause := fmt.Errorf("some error")
	// Variable has both a value and an error; error should take precedence
	interp.env.Set("x", types.NewNumber(decimal.NewFromInt(10)))
	interp.env.SetError("x", cause)

	id := &ast.Identifier{Name: "x"}
	_, err := interp.evalIdentifier(id)
	if err == nil {
		t.Fatal("expected CascadingError when variable is errored, even if value exists")
	}

	var ce *CascadingError
	if !errors.As(err, &ce) {
		t.Fatalf("expected CascadingError, got %T: %v", err, err)
	}
}

func TestEvalIdentifier_CascadingErrorInBinaryOp(t *testing.T) {
	interp := NewInterpreter()
	cause := fmt.Errorf("division by zero")
	interp.env.SetError("a", cause)
	interp.env.Set("b", types.NewNumber(decimal.NewFromInt(5)))

	// Expression: a + b (where a is errored)
	node := &ast.BinaryOp{
		Operator: "+",
		Left:     &ast.Identifier{Name: "a"},
		Right:    &ast.Identifier{Name: "b"},
	}

	_, err := interp.evalBinaryOp(node)
	if err == nil {
		t.Fatal("expected error from binary op with errored variable")
	}

	var ce *CascadingError
	if !errors.As(err, &ce) {
		t.Fatalf("expected CascadingError, got %T: %v", err, err)
	}
	if ce.VarName != "a" {
		t.Errorf("VarName = %q, want %q", ce.VarName, "a")
	}
}

func TestEvalIdentifier_UndefinedVariable_NotCascading(t *testing.T) {
	interp := NewInterpreter()

	id := &ast.Identifier{Name: "nonexistent"}
	_, err := interp.evalIdentifier(id)
	if err == nil {
		t.Fatal("expected error for undefined variable")
	}

	var ce *CascadingError
	if errors.As(err, &ce) {
		t.Error("undefined variable error should NOT be a CascadingError")
	}
}
