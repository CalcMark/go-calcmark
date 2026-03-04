package parser

import (
	"testing"

	"github.com/CalcMark/go-calcmark/spec/ast"
)

// TestGrowthFunctionBasicParsing tests that growth functions parse as standard function calls.
func TestGrowthFunctionBasicParsing(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		funcName string
		wantArgs int
	}{
		{"compound 3 args", "compound(1000, 5%, 10)", "compound", 3},
		{"compound with currency", "compound($1000, 5%, 10)", "compound", 3},
		{"compound with duration", "compound(1000, 5%, 10 years)", "compound", 3},
		{"compound with 4 args period", "compound(1000, 5%, 12, monthly)", "compound", 4},
		{"grow 3 args", "grow(100, 20, 5)", "grow", 3},
		{"grow with quantity", "grow(10 GB, 2 GB, 5)", "grow", 3},
		{"depreciate 3 args", "depreciate(10000, 15%, 5)", "depreciate", 3},
		{"depreciate with salvage", "depreciate(10000, 15%, 5, 1000)", "depreciate", 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes, err := Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse(%q) error = %v", tt.input, err)
			}

			if len(nodes) != 1 {
				t.Fatalf("Parse(%q) returned %d nodes, want 1", tt.input, len(nodes))
			}

			funcCall, ok := nodes[0].(*ast.FunctionCall)
			if !ok {
				t.Fatalf("Parse(%q) returned %T, want *ast.FunctionCall", tt.input, nodes[0])
			}

			if funcCall.Name != tt.funcName {
				t.Errorf("Parse(%q) function name = %q, want %q", tt.input, funcCall.Name, tt.funcName)
			}

			if len(funcCall.Arguments) != tt.wantArgs {
				t.Errorf("Parse(%q) got %d arguments, want %d", tt.input, len(funcCall.Arguments), tt.wantArgs)
			}
		})
	}
}

// TestGrowthFunctionCompoundedModifier tests parsing of the "compounded" modifier
// in functional syntax: compound(1000, 5%, 10, compounded monthly)
func TestGrowthFunctionCompoundedModifier(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantArgs int
	}{
		{
			name:     "compounded monthly",
			input:    "compound(1000, 12%, 10, compounded monthly)",
			wantArgs: 4,
		},
		{
			name:     "compounded quarterly",
			input:    "compound(1000, 8%, 5, compounded quarterly)",
			wantArgs: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes, err := Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse(%q) error = %v", tt.input, err)
			}

			if len(nodes) != 1 {
				t.Fatalf("Parse(%q) returned %d nodes, want 1", tt.input, len(nodes))
			}

			funcCall, ok := nodes[0].(*ast.FunctionCall)
			if !ok {
				t.Fatalf("Parse(%q) returned %T, want *ast.FunctionCall", tt.input, nodes[0])
			}

			if funcCall.Name != "compound" {
				t.Errorf("function name = %q, want %q", funcCall.Name, "compound")
			}

			if len(funcCall.Arguments) != tt.wantArgs {
				t.Errorf("got %d arguments, want %d", len(funcCall.Arguments), tt.wantArgs)
			}

			// 4th arg should be an Identifier with "compounded:" prefix
			if len(funcCall.Arguments) >= 4 {
				ident, ok := funcCall.Arguments[3].(*ast.Identifier)
				if !ok {
					t.Fatalf("4th arg is %T, want *ast.Identifier", funcCall.Arguments[3])
				}
				if tt.name == "compounded monthly" && ident.Name != "compounded:monthly" {
					t.Errorf("4th arg name = %q, want %q", ident.Name, "compounded:monthly")
				}
				if tt.name == "compounded quarterly" && ident.Name != "compounded:quarterly" {
					t.Errorf("4th arg name = %q, want %q", ident.Name, "compounded:quarterly")
				}
			}
		})
	}
}

// TestGrowthNLCompound tests natural language compound syntax.
func TestGrowthNLCompound(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantArgs int
	}{
		{
			name:     "basic compound NL",
			input:    "compound $1000 by 5% over 10 years",
			wantArgs: 3,
		},
		{
			name:     "compound NL per month",
			input:    "compound $1000 by 5% per month over 12 months",
			wantArgs: 4,
		},
		{
			name:     "compound NL compounded monthly",
			input:    "compound $1000 by 12% compounded monthly over 10 years",
			wantArgs: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes, err := Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse(%q) error = %v", tt.input, err)
			}

			if len(nodes) != 1 {
				t.Fatalf("Parse(%q) returned %d nodes, want 1", tt.input, len(nodes))
			}

			funcCall, ok := nodes[0].(*ast.FunctionCall)
			if !ok {
				t.Fatalf("Parse(%q) returned %T, want *ast.FunctionCall", tt.input, nodes[0])
			}

			if funcCall.Name != "compound" {
				t.Errorf("function name = %q, want %q", funcCall.Name, "compound")
			}

			if len(funcCall.Arguments) != tt.wantArgs {
				t.Errorf("got %d arguments, want %d", len(funcCall.Arguments), tt.wantArgs)
			}

			// Check modifier type for 4-arg forms
			if tt.wantArgs == 4 && len(funcCall.Arguments) >= 4 {
				ident, ok := funcCall.Arguments[3].(*ast.Identifier)
				if !ok {
					t.Fatalf("4th arg is %T, want *ast.Identifier", funcCall.Arguments[3])
				}
				if tt.name == "compound NL per month" && ident.Name != "month" {
					t.Errorf("4th arg = %q, want %q", ident.Name, "month")
				}
				if tt.name == "compound NL compounded monthly" && ident.Name != "compounded:monthly" {
					t.Errorf("4th arg = %q, want %q", ident.Name, "compounded:monthly")
				}
			}
		})
	}
}

// TestGrowthNLGrow tests natural language grow syntax.
func TestGrowthNLGrow(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantArgs int
	}{
		{
			name:     "basic grow NL",
			input:    "grow 100 by 20 over 5 months",
			wantArgs: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes, err := Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse(%q) error = %v", tt.input, err)
			}

			if len(nodes) != 1 {
				t.Fatalf("Parse(%q) returned %d nodes, want 1", tt.input, len(nodes))
			}

			funcCall, ok := nodes[0].(*ast.FunctionCall)
			if !ok {
				t.Fatalf("Parse(%q) returned %T, want *ast.FunctionCall", tt.input, nodes[0])
			}

			if funcCall.Name != "grow" {
				t.Errorf("function name = %q, want %q", funcCall.Name, "grow")
			}

			if len(funcCall.Arguments) != tt.wantArgs {
				t.Errorf("got %d arguments, want %d", len(funcCall.Arguments), tt.wantArgs)
			}
		})
	}
}

// TestGrowthNLDepreciate tests natural language depreciate syntax.
func TestGrowthNLDepreciate(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantArgs int
	}{
		{
			name:     "basic depreciate NL",
			input:    "depreciate $50000 by 15% over 5 years",
			wantArgs: 3,
		},
		{
			name:     "depreciate NL with salvage",
			input:    "depreciate $50000 by 15% over 5 years to $5000",
			wantArgs: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes, err := Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse(%q) error = %v", tt.input, err)
			}

			if len(nodes) != 1 {
				t.Fatalf("Parse(%q) returned %d nodes, want 1", tt.input, len(nodes))
			}

			funcCall, ok := nodes[0].(*ast.FunctionCall)
			if !ok {
				t.Fatalf("Parse(%q) returned %T, want *ast.FunctionCall", tt.input, nodes[0])
			}

			if funcCall.Name != "depreciate" {
				t.Errorf("function name = %q, want %q", funcCall.Name, "depreciate")
			}

			if len(funcCall.Arguments) != tt.wantArgs {
				t.Errorf("got %d arguments, want %d", len(funcCall.Arguments), tt.wantArgs)
			}
		})
	}
}

// TestGrowthNLFunctionalParity verifies NL and functional forms produce same-shaped ASTs.
func TestGrowthNLFunctionalParity(t *testing.T) {
	tests := []struct {
		name string
		nl   string
		fn   string
	}{
		{
			name: "compound basic",
			nl:   "compound $1000 by 5% over 10 years",
			fn:   "compound($1000, 5%, 10 years)",
		},
		{
			name: "grow basic",
			nl:   "grow 100 by 20 over 5 months",
			fn:   "grow(100, 20, 5 months)",
		},
		{
			name: "depreciate basic",
			nl:   "depreciate $50000 by 15% over 5 years",
			fn:   "depreciate($50000, 15%, 5 years)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nlNodes, nlErr := Parse(tt.nl)
			fnNodes, fnErr := Parse(tt.fn)

			if nlErr != nil {
				t.Fatalf("NL Parse(%q) error = %v", tt.nl, nlErr)
			}
			if fnErr != nil {
				t.Fatalf("FN Parse(%q) error = %v", tt.fn, fnErr)
			}

			nlFunc, ok := nlNodes[0].(*ast.FunctionCall)
			if !ok {
				t.Fatalf("NL returned %T, want *ast.FunctionCall", nlNodes[0])
			}
			fnFunc, ok := fnNodes[0].(*ast.FunctionCall)
			if !ok {
				t.Fatalf("FN returned %T, want *ast.FunctionCall", fnNodes[0])
			}

			if nlFunc.Name != fnFunc.Name {
				t.Errorf("function names differ: NL=%q, FN=%q", nlFunc.Name, fnFunc.Name)
			}
			if len(nlFunc.Arguments) != len(fnFunc.Arguments) {
				t.Errorf("arg counts differ: NL=%d, FN=%d", len(nlFunc.Arguments), len(fnFunc.Arguments))
			}
		})
	}
}

// TestGrowthFunctionInAssignment tests growth functions in assignment context.
func TestGrowthFunctionInAssignment(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		varName  string
		funcName string
	}{
		{"compound assignment", "result = compound(1000, 5%, 10)", "result", "compound"},
		{"grow assignment", "total = grow(100, 20, 5)", "total", "grow"},
		{"depreciate assignment", "value = depreciate(50000, 15%, 7)", "value", "depreciate"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes, err := Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse(%q) error = %v", tt.input, err)
			}

			if len(nodes) != 1 {
				t.Fatalf("Parse(%q) returned %d nodes, want 1", tt.input, len(nodes))
			}

			assign, ok := nodes[0].(*ast.Assignment)
			if !ok {
				t.Fatalf("Parse(%q) returned %T, want *ast.Assignment", tt.input, nodes[0])
			}

			if assign.Name != tt.varName {
				t.Errorf("variable name = %q, want %q", assign.Name, tt.varName)
			}

			funcCall, ok := assign.Value.(*ast.FunctionCall)
			if !ok {
				t.Fatalf("assignment value is %T, want *ast.FunctionCall", assign.Value)
			}

			if funcCall.Name != tt.funcName {
				t.Errorf("function name = %q, want %q", funcCall.Name, tt.funcName)
			}
		})
	}
}
