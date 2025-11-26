package interpreter

import (
	"testing"

	"github.com/CalcMark/go-calcmark/spec/ast"
	"github.com/CalcMark/go-calcmark/spec/types"
)

// TestDowntimeFunctionIntegration - tests the full downtime() function via interpreter
func TestDowntimeFunctionIntegration(t *testing.T) {
	interp := New()

	tests := []struct {
		name         string
		input        *ast.FunctionCall
		expectedUnit string
		expectedVal  string
		expectError  bool
	}{
		{
			name: "downtime(99.9%, month)",
			input: &ast.FunctionCall{
				Name: "downtime",
				Arguments: []ast.Node{
					&ast.NumberLiteral{Value: "0.999"}, // 99.9%
					&ast.Identifier{Name: "month"},
				},
			},
			expectedUnit: "minute",
			expectedVal:  "43.2",
		},
		{
			name: "downtime(99.99%, year)",
			input: &ast.FunctionCall{
				Name: "downtime",
				Arguments: []ast.Node{
					&ast.NumberLiteral{Value: "0.9999"}, // 99.99%
					&ast.Identifier{Name: "year"},
				},
			},
			expectedUnit: "minute",
			expectedVal:  "52.56",
		},
		{
			name: "downtime(99.999%, day)",
			input: &ast.FunctionCall{
				Name: "downtime",
				Arguments: []ast.Node{
					&ast.NumberLiteral{Value: "0.99999"}, // 99.999%
					&ast.Identifier{Name: "day"},
				},
			},
			expectedUnit: "second",
			expectedVal:  "0.864",
		},
		{
			name: "downtime(99%, week)",
			input: &ast.FunctionCall{
				Name: "downtime",
				Arguments: []ast.Node{
					&ast.NumberLiteral{Value: "0.99"}, // 99%
					&ast.Identifier{Name: "week"},
				},
			},
			expectedUnit: "hour",
			expectedVal:  "1.68",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := interp.evalFunctionCall(tt.input)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			duration, ok := result.(*types.Duration)
			if !ok {
				t.Fatalf("Expected *types.Duration, got %T", result)
			}

			if duration.Unit != tt.expectedUnit {
				t.Errorf("Expected unit %s, got %s", tt.expectedUnit, duration.Unit)
			}

			if duration.Value.String() != tt.expectedVal {
				t.Errorf("Expected value %s, got %s", tt.expectedVal, duration.Value.String())
			}

			t.Logf("✓ %s = %s %s", tt.name, duration.Value.String(), duration.Unit)
		})
	}
}
