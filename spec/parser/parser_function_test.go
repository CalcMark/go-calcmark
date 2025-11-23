package parser

import (
	"testing"

	"github.com/CalcMark/go-calcmark/spec/ast"
)

// TestParseFunctionCallsStandalone tests that standalone function calls parse correctly
func TestParseAvgFunction(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantArgs int
	}{
		{"avg with 3 args", "avg(1, 2, 3)", 3},
		{"avg with 5 args", "avg(1, 2, 3, 4, 5)", 5},
		{"avg with 1 arg", "avg(42)", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes, err := Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse(%q) error = %v, want nil", tt.input, err)
			}

			if len(nodes) != 1 {
				t.Fatalf("Parse(%q) returned %d nodes, want 1", tt.input, len(nodes))
			}

			funcCall, ok := nodes[0].(*ast.FunctionCall)
			if !ok {
				t.Fatalf("Parse(%q) returned %T, want *ast.FunctionCall", tt.input, nodes[0])
			}

			if funcCall.Name != "avg" {
				t.Errorf("Parse(%q) function name = %q, want \"avg\"", tt.input, funcCall.Name)
			}

			if len(funcCall.Arguments) != tt.wantArgs {
				t.Errorf("Parse(%q) got %d arguments, want %d", tt.input, len(funcCall.Arguments), tt.wantArgs)
			}
		})
	}
}

func TestParseAverageOfFunction(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantArgs int
	}{
		{"average of 3 args", "average of 1, 2, 3", 3},
		{"average of 5 args", "average of 1, 3, 5, 7, 9", 5},
		{"average of 2 args", "average of 10, 20", 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes, err := Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse(%q) error = %v, want nil", tt.input, err)
			}

			if len(nodes) != 1 {
				t.Fatalf("Parse(%q) returned %d nodes, want 1", tt.input, len(nodes))
			}

			funcCall, ok := nodes[0].(*ast.FunctionCall)
			if !ok {
				t.Fatalf("Parse(%q) returned %T, want *ast.FunctionCall", tt.input, nodes[0])
			}

			// "average of" should be normalized to "avg"
			if funcCall.Name != "avg" {
				t.Errorf("Parse(%q) function name = %q, want \"avg\"", tt.input, funcCall.Name)
			}

			if len(funcCall.Arguments) != tt.wantArgs {
				t.Errorf("Parse(%q) got %d arguments, want %d", tt.input, len(funcCall.Arguments), tt.wantArgs)
			}
		})
	}
}

func TestParseSqrtFunction(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"sqrt integer", "sqrt(16)"},
		{"sqrt decimal", "sqrt(2.5)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes, err := Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse(%q) error = %v, want nil", tt.input, err)
			}

			if len(nodes) != 1 {
				t.Fatalf("Parse(%q) returned %d nodes, want 1", tt.input, len(nodes))
			}

			funcCall, ok := nodes[0].(*ast.FunctionCall)
			if !ok {
				t.Fatalf("Parse(%q) returned %T, want *ast.FunctionCall", tt.input, nodes[0])
			}

			if funcCall.Name != "sqrt" {
				t.Errorf("Parse(%q) function name = %q, want \"sqrt\"", tt.input, funcCall.Name)
			}

			if len(funcCall.Arguments) != 1 {
				t.Errorf("Parse(%q) got %d arguments, want 1", tt.input, len(funcCall.Arguments))
			}
		})
	}
}

func TestParseSquareRootOfFunction(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"square root of integer", "square root of 25"},
		{"square root of decimal", "square root of 100"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes, err := Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse(%q) error = %v, want nil", tt.input, err)
			}

			if len(nodes) != 1 {
				t.Fatalf("Parse(%q) returned %d nodes, want 1", tt.input, len(nodes))
			}

			// The original code had an `expr, ok := nodes[0].(*ast.Expression)` check here,
			// but the `Code Edit` implies that `nodes[0]` should directly be a `*ast.FunctionCall`.
			// This is a correction to align with the expected AST structure for a standalone function call.
			funcCall, ok := nodes[0].(*ast.FunctionCall)
			if !ok {
				t.Fatalf("Parse(%q) returned %T, want *ast.FunctionCall", tt.input, nodes[0])
			}

			// "square root of" should be normalized to "sqrt"
			if funcCall.Name != "sqrt" {
				t.Errorf("Parse(%q) function name = %q, want \"sqrt\"", tt.input, funcCall.Name)
			}

			if len(funcCall.Arguments) != 1 {
				t.Errorf("Parse(%q) got %d arguments, want 1", tt.input, len(funcCall.Arguments))
			}
		})
	}
}

func TestParseFunctionInAssignment(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		varName  string
		funcName string
	}{
		{"avg in assignment", "result = avg(1, 2, 3)", "result", "avg"},
		{"average of in assignment", "mean = average of 10, 20, 30", "mean", "avg"},
		{"sqrt in assignment", "root = sqrt(16)", "root", "sqrt"},
		{"square root of in assignment", "val = square root of 25", "val", "sqrt"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes, err := Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse(%q) error = %v, want nil", tt.input, err)
			}

			if len(nodes) != 1 {
				t.Fatalf("Parse(%q) returned %d nodes, want 1", tt.input, len(nodes))
			}

			assign, ok := nodes[0].(*ast.Assignment)
			if !ok {
				t.Fatalf("Parse(%q) returned %T, want *ast.Assignment", tt.input, nodes[0])
			}

			if assign.Name != tt.varName {
				t.Errorf("Parse(%q) variable name = %q, want %q", tt.input, assign.Name, tt.varName)
			}

			funcCall, ok := assign.Value.(*ast.FunctionCall)
			if !ok {
				t.Fatalf("Parse(%q) assignment value is %T, want *ast.FunctionCall", tt.input, assign.Value)
			}

			if funcCall.Name != tt.funcName {
				t.Errorf("Parse(%q) function name = %q, want %q", tt.input, funcCall.Name, tt.funcName)
			}
		})
	}
}
