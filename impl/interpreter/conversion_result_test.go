package interpreter_test

import (
	"testing"

	"github.com/CalcMark/go-calcmark/impl/interpreter"
	"github.com/CalcMark/go-calcmark/spec/parser"
)

// TestConversionExpressionReturnsResult verifies that conversion expressions return results
func TestConversionExpressionReturnsResult(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		expectResult bool
	}{
		{"simple conversion", "10 meters in feet", true},
		{"quantity literal", "10 meters", true},
		{"assignment", "x = 10 meters", true},
		{"arithmetic", "5 + 3", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes, err := parser.Parse(tt.input + "\n")
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}

			interp := interpreter.NewInterpreter()
			results, err := interp.Eval(nodes)

			if err != nil {
				t.Fatalf("Eval error: %v", err)
			}

			if tt.expectResult && len(results) == 0 {
				t.Errorf("Expected result but got none")
			}

			if len(results) > 0 {
				t.Logf("✓ Result: %v", results[len(results)-1])
			} else {
				t.Logf("✗ No result returned")
			}
		})
	}
}
