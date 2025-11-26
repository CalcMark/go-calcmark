package parser

import (
	"testing"

	"github.com/CalcMark/go-calcmark/spec/ast"
)

func TestCapacityPlanningNaturalSyntax(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
		checkFunc   func(*testing.T, []ast.Node)
	}{
		{
			name:        "with keyword - basic",
			input:       "10000 req/s with 450 req/s\n",
			expectError: false,
			checkFunc: func(t *testing.T, nodes []ast.Node) {
				if len(nodes) != 1 {
					t.Fatalf("Expected 1 node, got %d", len(nodes))
				}
				fc, ok := nodes[0].(*ast.FunctionCall)
				if !ok {
					t.Fatalf("Expected FunctionCall, got %T", nodes[0])
				}
				if fc.Name != "requires" {
					t.Errorf("Expected function 'requires', got '%s'", fc.Name)
				}
				if len(fc.Arguments) != 2 {
					t.Errorf("Expected 2 arguments, got %d", len(fc.Arguments))
				}
			},
		},
		{
			name:        "with and buffer",
			input:       "10000 req/s with 450 req/s and 20%\n",
			expectError: false,
			checkFunc: func(t *testing.T, nodes []ast.Node) {
				fc, ok := nodes[0].(*ast.FunctionCall)
				if !ok {
					t.Fatalf("Expected FunctionCall, got %T", nodes[0])
				}
				if fc.Name != "requires" {
					t.Errorf("Expected 'requires', got '%s'", fc.Name)
				}
				if len(fc.Arguments) != 3 {
					t.Errorf("Expected 3 arguments, got %d", len(fc.Arguments))
				}
			},
		},
		{
			name:        "with TB capacity and buffer",
			input:       "10 TB with 2 TB and 10%\n",
			expectError: false,
			checkFunc: func(t *testing.T, nodes []ast.Node) {
				fc, ok := nodes[0].(*ast.FunctionCall)
				if !ok {
					t.Fatalf("Expected FunctionCall, got %T", nodes[0])
				}
				if fc.Name != "requires" {
					t.Errorf("Expected 'requires', got '%s'", fc.Name)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes, err := Parse(tt.input)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
				return
			}

			if !tt.expectError && err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if tt.checkFunc != nil {
				tt.checkFunc(t, nodes)
			}

			t.Logf("✓ Parsed: %+v", nodes[0])
		})
	}
}
