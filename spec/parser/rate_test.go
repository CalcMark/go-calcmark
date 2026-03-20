package parser

import (
	"testing"

	"github.com/CalcMark/go-calcmark/spec/ast"
)

func TestRateParsing(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
		checkAST    func(*testing.T, []ast.Node)
	}{
		{
			name:        "rate with slash",
			input:       "100 MB/s\n",
			expectError: false,
			checkAST: func(t *testing.T, nodes []ast.Node) {
				if len(nodes) != 1 {
					t.Fatalf("Expected 1 node, got %d", len(nodes))
				}
				// RateLiteral is returned directly, not wrapped
				rate, ok := nodes[0].(*ast.RateLiteral)
				if !ok {
					t.Fatalf("Expected RateLiteral, got %T", nodes[0])
				}
				if rate.PerUnit != "s" {
					t.Errorf("Expected per unit 's', got '%s'", rate.PerUnit)
				}
			},
		},
		{
			name:        "rate with per keyword",
			input:       "5 GB per day\n",
			expectError: false,
			checkAST: func(t *testing.T, nodes []ast.Node) {
				if len(nodes) != 1 {
					t.Fatalf("Expected 1 node, got %d", len(nodes))
				}
				rate, ok := nodes[0].(*ast.RateLiteral)
				if !ok {
					t.Fatalf("Expected RateLiteral, got %T", nodes[0])
				}
				if rate.PerUnit != "day" {
					t.Errorf("Expected per unit 'day', got '%s'", rate.PerUnit)
				}
			},
		},
		{
			name:        "cost rate per hour",
			input:       "$0.10 per hour\n",
			expectError: false,
			checkAST: func(t *testing.T, nodes []ast.Node) {
				if len(nodes) != 1 {
					t.Fatalf("Expected 1 node, got %d", len(nodes))
				}
				rate, ok := nodes[0].(*ast.RateLiteral)
				if !ok {
					t.Fatalf("Expected RateLiteral, got %T", nodes[0])
				}
				if rate.PerUnit != "hour" {
					t.Errorf("Expected per unit 'hour', got '%s'", rate.PerUnit)
				}
			},
		},
		{
			name:        "regular division not a rate",
			input:       "10 / 5\n",
			expectError: false,
			checkAST: func(t *testing.T, nodes []ast.Node) {
				if len(nodes) != 1 {
					t.Fatalf("Expected 1 node, got %d", len(nodes))
				}
				// Should be BinaryOp (division), not RateLiteral
				_, ok := nodes[0].(*ast.BinaryOp)
				if !ok {
					t.Fatalf("Expected BinaryOp for division, got %T", nodes[0])
				}
			},
		},
		{
			name:        "rate with minutes",
			input:       "1000 req/min\n",
			expectError: false,
			checkAST: func(t *testing.T, nodes []ast.Node) {
				if len(nodes) != 1 {
					t.Fatalf("Expected 1 node, got %d", len(nodes))
				}
				rate, ok := nodes[0].(*ast.RateLiteral)
				if !ok {
					t.Fatalf("Expected RateLiteral, got %T", nodes[0])
				}
				if rate.PerUnit != "min" {
					t.Errorf("Expected per unit 'min', got '%s'", rate.PerUnit)
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

			if tt.checkAST != nil && !tt.expectError {
				tt.checkAST(t, nodes)
			}
		})
	}
}

// TestIdentifierPerDesugarsToConvertRate tests that "identifier per time_unit"
// desugars to convert_rate(identifier, time_unit) at parse time (issue #87).
func TestIdentifierPerDesugarsToConvertRate(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		targetUnit string
	}{
		{
			name:       "identifier per year",
			input:      "d per year\n",
			targetUnit: "year",
		},
		{
			name:       "identifier per second",
			input:      "r per second\n",
			targetUnit: "second",
		},
		{
			name:       "identifier per month",
			input:      "rate per month\n",
			targetUnit: "month",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes, err := Parse(tt.input)
			if err != nil {
				t.Fatalf("Unexpected parse error: %v", err)
			}

			if len(nodes) != 1 {
				t.Fatalf("Expected 1 node, got %d", len(nodes))
			}

			// Must be a FunctionCall to convert_rate, NOT a RateLiteral
			fc, ok := nodes[0].(*ast.FunctionCall)
			if !ok {
				t.Fatalf("Expected FunctionCall (convert_rate), got %T", nodes[0])
			}

			if fc.Name != "convert_rate" {
				t.Errorf("Expected function name 'convert_rate', got '%s'", fc.Name)
			}

			if len(fc.Arguments) != 2 {
				t.Fatalf("Expected 2 arguments, got %d", len(fc.Arguments))
			}

			// First arg should be the identifier
			_, isIdent := fc.Arguments[0].(*ast.Identifier)
			if !isIdent {
				t.Errorf("Expected first arg to be Identifier, got %T", fc.Arguments[0])
			}

			// Second arg should be the target time unit identifier
			targetIdent, ok := fc.Arguments[1].(*ast.Identifier)
			if !ok {
				t.Fatalf("Expected second arg to be Identifier, got %T", fc.Arguments[1])
			}
			if targetIdent.Name != tt.targetUnit {
				t.Errorf("Expected target unit '%s', got '%s'", tt.targetUnit, targetIdent.Name)
			}
		})
	}
}

// TestLiteralPerStillCreatesRate ensures that literal expressions with per
// still create RateLiterals (not convert_rate calls).
func TestLiteralPerStillCreatesRate(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "quantity per day",
			input: "5 GB per day\n",
		},
		{
			name:  "number per hour",
			input: "100 per hour\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes, err := Parse(tt.input)
			if err != nil {
				t.Fatalf("Unexpected parse error: %v", err)
			}
			if len(nodes) != 1 {
				t.Fatalf("Expected 1 node, got %d", len(nodes))
			}
			// Must be RateLiteral, NOT FunctionCall
			_, ok := nodes[0].(*ast.RateLiteral)
			if !ok {
				t.Fatalf("Expected RateLiteral, got %T", nodes[0])
			}
		})
	}
}

// TestRateUnitConversionParsing tests parsing of rate-to-rate unit conversion.
// Examples: "10 m/s in inch/s", "60 km/h in mph"
func TestRateUnitConversionParsing(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectError    bool
		targetUnit     string
		targetTimeUnit string
	}{
		{
			name:           "meters per second to inches per second",
			input:          "10 m/s in inch/s\n",
			expectError:    false,
			targetUnit:     "inch",
			targetTimeUnit: "s",
		},
		{
			name:           "km per hour to miles per hour",
			input:          "60 km/h in mile/h\n",
			expectError:    false,
			targetUnit:     "mile",
			targetTimeUnit: "h",
		},
		{
			name:           "rate conversion with per keyword",
			input:          "10 m/s in inch per second\n",
			expectError:    false,
			targetUnit:     "inch",
			targetTimeUnit: "second",
		},
		{
			name:           "rate conversion changing time unit",
			input:          "60 m/s in m/min\n",
			expectError:    false,
			targetUnit:     "m",
			targetTimeUnit: "min",
		},
		{
			name:        "invalid time unit in rate conversion",
			input:       "10 m/s in inch/foo\n",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes, err := Parse(tt.input)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected parse error: %v", err)
			}

			if len(nodes) != 1 {
				t.Fatalf("Expected 1 node, got %d", len(nodes))
			}

			conv, ok := nodes[0].(*ast.UnitConversion)
			if !ok {
				t.Fatalf("Expected UnitConversion, got %T", nodes[0])
			}

			if conv.TargetUnit != tt.targetUnit {
				t.Errorf("Expected target unit '%s', got '%s'", tt.targetUnit, conv.TargetUnit)
			}

			if conv.TargetTimeUnit != tt.targetTimeUnit {
				t.Errorf("Expected target time unit '%s', got '%s'", tt.targetTimeUnit, conv.TargetTimeUnit)
			}

			// Verify the source is a RateLiteral
			_, isRate := conv.Quantity.(*ast.RateLiteral)
			if !isRate {
				t.Errorf("Expected source to be RateLiteral, got %T", conv.Quantity)
			}
		})
	}
}
