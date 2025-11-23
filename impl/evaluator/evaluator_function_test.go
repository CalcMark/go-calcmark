package evaluator

import (
	"strings"
	"testing"

	"github.com/CalcMark/go-calcmark/impl/types"
	"github.com/shopspring/decimal"
)

// TestEvaluateAvgFunction tests avg() function evaluation
func TestEvaluateAvgFunction(t *testing.T) {
	ctx := NewContext()

	tests := []struct {
		name     string
		input    string
		want     string
		wantType string
	}{
		{"avg basic", "avg(1, 2, 3)", "2", "Number"},
		{"avg five", "avg(1, 2, 3, 4, 5)", "3", "Number"},
		{"avg two", "avg(10, 20)", "15", "Number"},
		{"avg decimals", "avg(1.5, 2.5)", "2", "Number"},
		{"avg single", "avg(42)", "42", "Number"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := Evaluate(tt.input, ctx)
			if err != nil {
				t.Fatalf("Evaluate(%q) error = %v, want nil", tt.input, err)
			}

			if len(results) != 1 {
				t.Fatalf("Evaluate(%q) returned %d results, want 1", tt.input, len(results))
			}

			result := results[0]
			if result.TypeName() != tt.wantType {
				t.Errorf("Evaluate(%q) type = %s, want %s", tt.input, result.TypeName(), tt.wantType)
			}

			if result.String() != tt.want {
				t.Errorf("Evaluate(%q) = %s, want %s", tt.input, result.String(), tt.want)
			}
		})
	}
}

// TestEvaluateAverageOfFunction tests "average of" multi-token function
func TestEvaluateAverageOfFunction(t *testing.T) {
	ctx := NewContext()

	tests := []struct {
		name     string
		input    string
		want     string
		wantType string
	}{
		{"average of basic", "average of 1, 2, 3", "2", "Number"},
		{"average of many", "average of 1, 3, 5, 7, 9", "5", "Number"},
		{"average of two", "average of 10, 20", "15", "Number"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := Evaluate(tt.input, ctx)
			if err != nil {
				t.Fatalf("Evaluate(%q) error = %v, want nil", tt.input, err)
			}

			if len(results) != 1 {
				t.Fatalf("Evaluate(%q) returned %d results, want 1", tt.input, len(results))
			}

			result := results[0]
			if result.TypeName() != tt.wantType {
				t.Errorf("Evaluate(%q) type = %s, want %s", tt.input, result.TypeName(), tt.wantType)
			}

			if result.String() != tt.want {
				t.Errorf("Evaluate(%q) = %s, want %s", tt.input, result.String(), tt.want)
			}
		})
	}
}

// TestEvaluateSqrtFunction tests sqrt() function evaluation
func TestEvaluateSqrtFunction(t *testing.T) {
	ctx := NewContext()

	tests := []struct {
		name     string
		input    string
		want     string
		wantType string
	}{
		{"sqrt of 16", "sqrt(16)", "4", "Number"},
		{"sqrt of 25", "sqrt(25)", "5", "Number"},
		{"sqrt of 4", "sqrt(4)", "2", "Number"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := Evaluate(tt.input, ctx)
			if err != nil {
				t.Fatalf("Evaluate(%q) error = %v, want nil", tt.input, err)
			}

			if len(results) != 1 {
				t.Fatalf("Evaluate(%q) returned %d results, want 1", tt.input, len(results))
			}

			result := results[0]
			if result.TypeName() != tt.wantType {
				t.Errorf("Evaluate(%q) type = %s, want %s", tt.input, result.TypeName(), tt.wantType)
			}

			if result.String() != tt.want {
				t.Errorf("Evaluate(%q) = %s, want %s", tt.input, result.String(), tt.want)
			}
		})
	}
}

// TestEvaluateSquareRootOfFunction tests "square root of" multi-token function
func TestEvaluateSquareRootOfFunction(t *testing.T) {
	ctx := NewContext()

	tests := []struct {
		name     string
		input    string
		want     string
		wantType string
	}{
		{"square root of 25", "square root of 25", "5", "Number"},
		{"square root of 100", "square root of 100", "10", "Number"},
		{"square root of 9", "square root of 9", "3", "Number"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := Evaluate(tt.input, ctx)
			if err != nil {
				t.Fatalf("Evaluate(%q) error = %v, want nil", tt.input, err)
			}

			if len(results) != 1 {
				t.Fatalf("Evaluate(%q) returned %d results, want 1", tt.input, len(results))
			}

			result := results[0]
			if result.TypeName() != tt.wantType {
				t.Errorf("Evaluate(%q) type = %s, want %s", tt.input, result.TypeName(), tt.wantType)
			}

			if result.String() != tt.want {
				t.Errorf("Evaluate(%q) = %s, want %s", tt.input, result.String(), tt.want)
			}
		})
	}
}

// TestEvaluateFunctionInAssignment tests functions in assignments
func TestEvaluateFunctionInAssignment(t *testing.T) {
	ctx := NewContext()

	tests := []struct {
		name     string
		input    string
		varName  string
		want     string
		wantType string
	}{
		{"avg in assignment", "result = avg(1, 2, 3)", "result", "2", "Number"},
		{"average of in assignment", "mean = average of 10, 20, 30", "mean", "20", "Number"},
		{"sqrt in assignment", "root = sqrt(16)", "root", "4", "Number"},
		{"square root of in assignment", "val = square root of 25", "val", "5", "Number"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := Evaluate(tt.input, ctx)
			if err != nil {
				t.Fatalf("Evaluate(%q) error = %v, want nil", tt.input, err)
			}

			if len(results) != 1 {
				t.Fatalf("Evaluate(%q) returned %d results, want 1", tt.input, len(results))
			}

			// Check that variable was set
			varValue, err := ctx.Get(tt.varName)
			if err != nil {
				t.Fatalf("Variable %q not set: %v", tt.varName, err)
			}

			if varValue.TypeName() != tt.wantType {
				t.Errorf("Variable %q type = %s, want %s", tt.varName, varValue.TypeName(), tt.wantType)
			}

			if varValue.String() != tt.want {
				t.Errorf("Variable %q = %s, want %s", tt.varName, varValue.String(), tt.want)
			}
		})
	}
}

// TestEvaluateFunctionWithVariables tests functions with variable arguments
func TestEvaluateFunctionWithVariables(t *testing.T) {
	ctx := NewContext()
	// Set up variables
	_, err := Evaluate("x = 5\ny = 10\nz = 15", ctx)
	if err != nil {
		t.Fatalf("Setup error: %v", err)
	}

	tests := []struct {
		name     string
		input    string
		want     string
		wantType string
	}{
		{"avg with variables", "avg(x, y, z)", "10", "Number"},
		{"average of with variables", "average of x, y, z", "10", "Number"},
		{"sqrt with variable", "sqrt(y)", "3.1622776601683795", "Number"}, // sqrt(10)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := Evaluate(tt.input, ctx)
			if err != nil {
				t.Fatalf("Evaluate(%q) error = %v, want nil", tt.input, err)
			}

			if len(results) != 1 {
				t.Fatalf("Evaluate(%q) returned %d results, want 1", tt.input, len(results))
			}

			result := results[0]
			if result.TypeName() != tt.wantType {
				t.Errorf("Evaluate(%q) type = %s, want %s", tt.input, result.TypeName(), tt.wantType)
			}

			if result.String() != tt.want {
				t.Errorf("Evaluate(%q) = %s, want %s", tt.input, result.String(), tt.want)
			}
		})
	}
}

// TestEvaluateFunctionWithCurrency tests functions preserve currency type
func TestEvaluateFunctionWithCurrency(t *testing.T) {
	ctx := NewContext()

	tests := []struct {
		name     string
		input    string
		want     string
		wantType string
	}{
		{"avg with currency", "avg($100, $200, $300)", "$200.00", "Currency"},
		{"sqrt with currency", "sqrt($16)", "$4.00", "Currency"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := Evaluate(tt.input, ctx)
			if err != nil {
				t.Fatalf("Evaluate(%q) error = %v, want nil", tt.input, err)
			}

			if len(results) != 1 {
				t.Fatalf("Evaluate(%q) returned %d results, want 1", tt.input, len(results))
			}

			result := results[0]
			if result.TypeName() != tt.wantType {
				t.Errorf("Evaluate(%q) type = %s, want %s", tt.input, result.TypeName(), tt.wantType)
			}

			if result.String() != tt.want {
				t.Errorf("Evaluate(%q) = %s, want %s", tt.input, result.String(), tt.want)
			}
		})
	}
}

// TestEvaluateFunctionErrors tests error cases
func TestEvaluateFunctionErrors(t *testing.T) {
	ctx := NewContext()

	tests := []struct {
		name      string
		input     string
		wantError string
	}{
		{"sqrt negative", "sqrt(-4)", "sqrt() of negative number is not supported"},
		{"sqrt too many args", "sqrt(1, 2)", "sqrt() requires exactly one argument"},
		{"avg no args", "avg()", "avg() requires at least one argument"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Evaluate(tt.input, ctx)
			if err == nil {
				t.Fatalf("Evaluate(%q) expected error, got nil", tt.input)
			}

			// Accept either ParseError (from parser validation) or EvaluationError (from evaluator)
			// Parser now validates argument counts, so sqrt(1,2) and avg() return ParseError
			errMsg := err.Error()
			if !strings.Contains(errMsg, tt.wantError) && !strings.Contains(errMsg, "argument") {
				t.Errorf("Evaluate(%q) error = %q, want to contain %q", tt.input, errMsg, tt.wantError)
			}
		})
	}
}

// TestEvaluateFunctionPrecision tests that decimal precision is maintained
func TestEvaluateFunctionPrecision(t *testing.T) {
	ctx := NewContext()

	// Test that avg maintains precision
	input := "avg(1, 2)"
	results, err := Evaluate(input, ctx)
	if err != nil {
		t.Fatalf("Evaluate(%q) error = %v, want nil", input, err)
	}

	if len(results) != 1 {
		t.Fatalf("Evaluate(%q) returned %d results, want 1", input, len(results))
	}

	num := results[0].(*types.Number)
	expected := decimal.NewFromFloat(1.5)
	if !num.Value.Equal(expected) {
		t.Errorf("avg(1, 2) = %s, want %s", num.Value, expected)
	}
}
