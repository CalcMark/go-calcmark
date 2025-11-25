package interpreter

import (
	"testing"

	"github.com/CalcMark/go-calcmark/spec/parser"
	"github.com/CalcMark/go-calcmark/spec/types"
)

func TestNaturalSyntaxInterpreter(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
		checkResult func(*testing.T, types.Type)
	}{
		{
			name:        "over keyword - accumulate",
			input:       "100 MB/s over 1 day\n",
			expectError: false,
			checkResult: func(t *testing.T, result types.Type) {
				qty, ok := result.(*types.Quantity)
				if !ok {
					t.Fatalf("Expected Quantity, got %T", result)
				}
				// 100 MB/s * 86400 s = 8,640,000 MB = 8640 GB = 8.64 TB
				if qty.Unit != "MB" {
					t.Errorf("Expected unit MB, got %s", qty.Unit)
				}
				t.Logf("✓ Result: %s", result.String())
			},
		},
		{
			name:        "cost rate over time",
			input:       "$0.10/hour over 30 days\n",
			expectError: false,
			checkResult: func(t *testing.T, result types.Type) {
				qty, ok := result.(*types.Quantity)
				if !ok {
					t.Fatalf("Expected Quantity, got %T", result)
				}
				// $0.10/hour * (30 days * 24 hours) = $72
				if qty.Unit != "USD" {
					t.Errorf("Expected unit USD, got %s", qty.Unit)
				}
				t.Logf("✓ Result: %s", result.String())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes, err := parser.Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}

			interp := NewInterpreter()
			results, err := interp.Eval(nodes)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
				return
			}

			if !tt.expectError && err != nil {
				t.Fatalf("Eval error: %v", err)
			}

			if len(results) == 0 {
				t.Fatal("No results")
			}

			if tt.checkResult != nil {
				tt.checkResult(t, results[0])
			}
		})
	}
}
