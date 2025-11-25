package interpreter

import (
	"testing"

	"github.com/CalcMark/go-calcmark/spec/parser"
	"github.com/CalcMark/go-calcmark/spec/types"
)

func TestRateEvaluation(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
		checkResult func(*testing.T, types.Type)
	}{
		{
			name:        "bandwidth rate with slash",
			input:       "100 MB/s\n",
			expectError: false,
			checkResult: func(t *testing.T, result types.Type) {
				rate, ok := result.(*types.Rate)
				if !ok {
					t.Fatalf("Expected *types.Rate, got %T", result)
				}
				if rate.PerUnit != "second" {
					t.Errorf("Expected per unit 'second', got '%s'", rate.PerUnit)
				}
				if rate.Amount.Unit != "MB" {
					t.Errorf("Expected amount unit 'MB', got '%s'", rate.Amount.Unit)
				}
			},
		},
		{
			name:        "data rate with per keyword",
			input:       "5 GB per day\n",
			expectError: false,
			checkResult: func(t *testing.T, result types.Type) {
				rate, ok := result.(*types.Rate)
				if !ok {
					t.Fatalf("Expected *types.Rate, got %T", result)
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
			checkResult: func(t *testing.T, result types.Type) {
				rate, ok := result.(*types.Rate)
				if !ok {
					t.Fatalf("Expected *types.Rate, got %T", result)
				}
				if rate.PerUnit != "hour" {
					t.Errorf("Expected per unit 'hour', got '%s'", rate.PerUnit)
				}
			},
		},
		{
			name:        "requests per minute",
			input:       "1000 req/min\n",
			expectError: false,
			checkResult: func(t *testing.T, result types.Type) {
				rate, ok := result.(*types.Rate)
				if !ok {
					t.Fatalf("Expected *types.Rate, got %T", result)
				}
				if rate.PerUnit != "minute" {
					t.Errorf("Expected per unit 'minute', got '%s'", rate.PerUnit)
				}
			},
		},
		{
			name:        "assigned rate",
			input:       "bandwidth = 10 GB/s\n",
			expectError: false,
			checkResult: func(t *testing.T, result types.Type) {
				rate, ok := result.(*types.Rate)
				if !ok {
					t.Fatalf("Expected *types.Rate, got %T", result)
				}
				if rate.PerUnit != "second" {
					t.Errorf("Expected per unit 'second', got '%s'", rate.PerUnit)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the input
			nodes, err := parser.Parse(tt.input)
			if err != nil {
				if tt.expectError {
					return // Expected parse error
				}
				t.Fatalf("Parse error: %v", err)
			}

			// Evaluate with interpreter
			interp := NewInterpreter()
			results, err := interp.Eval(nodes)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Eval error: %v", err)
			}

			if len(results) == 0 {
				t.Fatal("No results returned")
			}

			if tt.checkResult != nil {
				tt.checkResult(t, results[0])
			}
		})
	}
}
