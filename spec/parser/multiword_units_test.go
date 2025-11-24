package parser

import (
	"testing"

	"github.com/CalcMark/go-calcmark/spec/ast"
)

// TestMultiWordUnitParsing tests parsing of multi-word units like "nautical mile" and "metric ton"
func TestMultiWordUnitParsing(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		expectedUnit string
		shouldParse  bool
	}{
		// Multi-word units that should work
		{"nautical mile", "1 nautical mile\n", "nautical mile", true},
		{"nautical miles plural", "5 nautical miles\n", "nautical miles", true},
		{"metric ton", "2 metric ton\n", "metric ton", true},
		{"metric tons plural", "10 metric tons\n", "metric tons", true},

		// Single-word aliases still work
		{"nmi alias", "1 nmi\n", "nmi", true},
		{"t alias", "5 t\n", "t", true},
		{"tonne alias", "5 tonne\n", "tonne", true},

		// Conversion with multi-word units
		{"convert to nautical miles", "10 km in nautical miles\n", "nautical miles", true},
		{"convert to metric tons", "100 pounds in metric tons\n", "metric tons", true},

		// Invalid multi-word combinations should NOT be recognized as units
		// They'll be parsed as number + identifier + identifier (second will cause error)
		{"invalid two words", "1 apple banana\n", "apple", false}, // Will parse as "1 apple" then error on "banana"
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewRecursiveDescentParser(tt.input)
			nodes, err := parser.Parse()

			if !tt.shouldParse {
				// We expect this to fail at some point
				if err == nil {
					t.Errorf("Expected parse error for invalid multi-word unit, but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}

			if len(nodes) == 0 {
				t.Fatal("No nodes parsed")
			}

			// Check the unit in the parsed node
			switch node := nodes[0].(type) {
			case *ast.QuantityLiteral:
				if node.Unit != tt.expectedUnit {
					t.Errorf("Expected unit %q, got %q", tt.expectedUnit, node.Unit)
				}
			case *ast.UnitConversion:
				if node.TargetUnit != tt.expectedUnit {
					t.Errorf("Expected target unit %q, got %q", tt.expectedUnit, node.TargetUnit)
				}
			default:
				t.Errorf("Expected QuantityLiteral or UnitConversion, got %T", node)
			}
		})
	}
}
