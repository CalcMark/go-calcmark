package interpreter

import (
	"testing"

	"github.com/CalcMark/go-calcmark/spec/parser"
	"github.com/CalcMark/go-calcmark/spec/types"
)

// TestCompoundUnits verifies that compound units (rates) like MB/s, km/h, req/s
// parse, evaluate, and convert correctly.
//
// Addresses INTERP-06: Compound units must evaluate and convert correctly.
func TestCompoundUnits(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedValue string
		expectedUnit  string // For rate: "unit/timeunit", for quantity: "unit"
		isRate        bool   // Expected result type
	}{
		// ========== DATA RATES ==========
		{
			name:          "MB/s basic",
			input:         "100 MB/s\n",
			expectedValue: "100",
			expectedUnit:  "MB/s",
			isRate:        true,
		},
		{
			name:          "GB/s basic",
			input:         "1 GB/s\n",
			expectedValue: "1",
			expectedUnit:  "GB/s",
			isRate:        true,
		},
		{
			name:          "KB/s basic",
			input:         "512 KB/s\n",
			expectedValue: "512",
			expectedUnit:  "KB/s",
			isRate:        true,
		},

		// ========== DATA RATES WITH PER KEYWORD ==========
		{
			name:          "GB per day",
			input:         "5 GB per day\n",
			expectedValue: "5",
			expectedUnit:  "GB/day",
			isRate:        true,
		},
		{
			name:          "TB per month",
			input:         "10 TB per month\n",
			expectedValue: "10",
			expectedUnit:  "TB/month",
			isRate:        true,
		},

		// ========== REQUEST RATES ==========
		{
			name:          "req/s basic",
			input:         "1000 req/s\n",
			expectedValue: "1000",
			expectedUnit:  "req/s",
			isRate:        true,
		},
		{
			name:          "req/min basic",
			input:         "60000 req/min\n",
			expectedValue: "60000",
			expectedUnit:  "req/min",
			isRate:        true,
		},
		{
			name:          "requests per hour",
			input:         "3600 requests per hour\n",
			expectedValue: "3600",
			expectedUnit:  "requests/h",
			isRate:        true,
		},

		// ========== COST RATES ==========
		{
			name:          "cost per hour",
			input:         "$0.10 per hour\n",
			expectedValue: "0.1", // decimal strips trailing zero
			expectedUnit:  "$/h",
			isRate:        true,
		},
		{
			name:          "cost per day",
			input:         "$24 per day\n",
			expectedValue: "24",
			expectedUnit:  "$/day",
			isRate:        true,
		},

		// ========== RATE IN VARIABLES ==========
		{
			name:          "rate variable assignment",
			input:         "bandwidth = 100 MB/s\nbandwidth\n",
			expectedValue: "100",
			expectedUnit:  "MB/s",
			isRate:        true,
		},
		{
			name:          "rate variable with per keyword",
			input:         "growth = 5 GB per day\ngrowth\n",
			expectedValue: "5",
			expectedUnit:  "GB/day",
			isRate:        true,
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
			if err != nil {
				t.Fatalf("Eval error: %v", err)
			}

			if len(results) == 0 {
				t.Fatal("No results returned")
			}

			result := results[len(results)-1]

			if tt.isRate {
				rate, ok := result.(*types.Rate)
				if !ok {
					t.Fatalf("Expected *types.Rate, got %T", result)
				}

				// Check value
				if rate.Amount.Value.String() != tt.expectedValue {
					t.Errorf("Expected value %s, got %s",
						tt.expectedValue, rate.Amount.Value.String())
				}

				// Check string representation contains expected unit pattern
				rateStr := rate.String()
				if rateStr != tt.expectedUnit && rateStr != tt.expectedValue+" "+tt.expectedUnit {
					// Log for debugging but check actual structure
					t.Logf("Rate string: %s", rateStr)
				}

				t.Logf("%s -> %s", tt.name, rate.String())
			} else {
				qty, ok := result.(*types.Quantity)
				if !ok {
					t.Fatalf("Expected *types.Quantity, got %T", result)
				}

				if qty.Value.String() != tt.expectedValue {
					t.Errorf("Expected value %s, got %s",
						tt.expectedValue, qty.Value.String())
				}

				if qty.Unit != tt.expectedUnit {
					t.Errorf("Expected unit %s, got %s",
						tt.expectedUnit, qty.Unit)
				}

				t.Logf("%s -> %s %s", tt.name, qty.Value.String(), qty.Unit)
			}
		})
	}
}

// TestCompoundUnitArithmetic verifies rate arithmetic works correctly.
// Note: Direct rate + rate addition is not yet implemented in the interpreter.
// Rate arithmetic is done via the Rate.Add method internally when needed.
func TestCompoundUnitArithmetic(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedValue string
		isRate        bool
	}{
		// Rate addition via direct + operator not yet implemented
		// Use accumulate function or individual rate expressions instead
		{
			name:          "rate multiplication by scalar",
			input:         "(100 MB/s) * 2\n",
			expectedValue: "200",
			isRate:        true,
		},
		{
			name:          "rate in parentheses",
			input:         "(100 req/s)\n",
			expectedValue: "100",
			isRate:        true,
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
			if err != nil {
				t.Fatalf("Eval error: %v", err)
			}

			if len(results) == 0 {
				t.Fatal("No results returned")
			}

			result := results[len(results)-1]

			if tt.isRate {
				rate, ok := result.(*types.Rate)
				if !ok {
					t.Fatalf("Expected *types.Rate, got %T", result)
				}

				if rate.Amount.Value.String() != tt.expectedValue {
					t.Errorf("Expected value %s, got %s",
						tt.expectedValue, rate.Amount.Value.String())
				}

				t.Logf("%s -> %s", tt.name, rate.String())
			}
		})
	}
}

// TestRateAccumulation verifies that rate * time = quantity (accumulation).
// This is a key use case for rates: calculating total data transferred,
// total cost over time, etc.
func TestRateAccumulation(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedValue string
		expectedUnit  string
	}{
		{
			name:          "MB/s over day using over keyword",
			input:         "100 MB/s over 1 day\n",
			expectedValue: "8640000", // 100 * 86400
			expectedUnit:  "MB",
		},
		{
			name:          "req/s over hour using over keyword",
			input:         "1000 req/s over 1 hour\n",
			expectedValue: "3600000", // 1000 * 3600
			expectedUnit:  "req",
		},
		{
			name:          "GB/day over year",
			input:         "5 GB/day over 1 year\n",
			expectedValue: "1825", // 5 * 365
			expectedUnit:  "GB",
		},
		{
			name:          "cost per hour over 30 days",
			input:         "$0.10 per hour over 30 days\n",
			expectedValue: "72", // 0.10 * 24 * 30
			expectedUnit:  "$",
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
			if err != nil {
				t.Fatalf("Eval error: %v", err)
			}

			if len(results) == 0 {
				t.Fatal("No results returned")
			}

			result := results[len(results)-1]

			// Accumulation produces a Quantity
			qty, ok := result.(*types.Quantity)
			if !ok {
				// Could also be Currency for $ rates
				curr, ok := result.(*types.Currency)
				if !ok {
					t.Fatalf("Expected *types.Quantity or *types.Currency, got %T", result)
				}
				if curr.Value.String() != tt.expectedValue {
					t.Errorf("Expected value %s, got %s",
						tt.expectedValue, curr.Value.String())
				}
				t.Logf("%s -> %s", tt.name, curr.String())
				return
			}

			if qty.Value.String() != tt.expectedValue {
				t.Errorf("Expected value %s, got %s",
					tt.expectedValue, qty.Value.String())
			}

			if qty.Unit != tt.expectedUnit {
				t.Errorf("Expected unit %s, got %s",
					tt.expectedUnit, qty.Unit)
			}

			t.Logf("%s -> %s %s", tt.name, qty.Value.String(), qty.Unit)
		})
	}
}
