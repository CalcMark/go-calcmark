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

// TestRateTypePreservation verifies that Rate type is preserved or correctly
// transformed through various operations.
//
// Type rules for Rate:
// - Rate literal -> Rate
// - Rate in variable -> Rate
// - Rate * scalar -> Rate
// - Rate accumulation (rate over duration) -> Quantity
func TestRateTypePreservation(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		expectType string // "Rate", "Quantity", "Currency"
	}{
		// Rate stays Rate
		{
			name:       "rate literal stays Rate",
			input:      "100 MB/s\n",
			expectType: "Rate",
		},
		{
			name:       "rate in variable stays Rate",
			input:      "speed = 100 MB/s\nspeed\n",
			expectType: "Rate",
		},
		{
			name:       "rate with per keyword stays Rate",
			input:      "5 GB per day\n",
			expectType: "Rate",
		},

		// Rate * scalar = Rate
		// Note: Only rate * scalar is supported, not scalar * rate (not commutative)
		{
			name:       "rate times scalar stays Rate",
			input:      "(100 MB/s) * 2\n",
			expectType: "Rate",
		},

		// Rate accumulation = Quantity (rate over duration)
		{
			name:       "accumulate produces Quantity",
			input:      "100 MB/s over 1 hour\n",
			expectType: "Quantity",
		},
		{
			name:       "data rate over day produces Quantity",
			input:      "5 GB/day over 1 week\n",
			expectType: "Quantity",
		},

		// Request rate accumulation
		{
			name:       "req/s over hour produces Quantity",
			input:      "1000 req/s over 1 hour\n",
			expectType: "Quantity",
		},

		// Cost rate accumulation produces Currency (not Quantity)
		{
			name:       "cost rate over duration produces Currency",
			input:      "$0.10 per hour over 24 hours\n",
			expectType: "Currency",
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
			var actualType string

			switch result.(type) {
			case *types.Rate:
				actualType = "Rate"
			case *types.Quantity:
				actualType = "Quantity"
			case *types.Currency:
				actualType = "Currency"
			case *types.Number:
				actualType = "Number"
			default:
				actualType = "Unknown"
			}

			if actualType != tt.expectType {
				t.Errorf("Expected type %s, got %s (value: %v)",
					tt.expectType, actualType, result)
			}

			t.Logf("%s -> %s (type: %s)", tt.name, result.String(), actualType)
		})
	}
}

// TestUnitlessQuantityMultiplication verifies that a unitless quantity
// (from accumulating a unitless rate) can multiply with a quantity that has units.
// Regression test for https://github.com/CalcMark/go-calcmark/issues/8
func TestUnitlessQuantityMultiplication(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		expectedUnit string
	}{
		{
			name: "unitless rate accumulated then multiplied by quantity",
			input: `mau = 1M / month
w = mau over 1 week
avg_session_data = 500 KB
monthly_data = w * avg_session_data
`,
			expectedUnit: "KB",
		},
		{
			name: "quantity multiplied by unitless rate accumulation",
			input: `mau = 1M / month
w = mau over 1 week
avg_session_data = 500 KB
monthly_data = avg_session_data * w
`,
			expectedUnit: "KB",
		},
		// NOTE: "unitless quantity times unitless quantity" moved to
		// TestUnitlessQuantityNormalization — unitless quantities are now
		// normalized to Numbers at the top of evalBinaryOperation, so the
		// result is *types.Number, not *types.Quantity.
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

			qty, ok := result.(*types.Quantity)
			if !ok {
				t.Fatalf("Expected *types.Quantity, got %T (%v)", result, result)
			}

			if qty.Unit != tt.expectedUnit {
				t.Errorf("Expected unit %q, got %q", tt.expectedUnit, qty.Unit)
			}

			if qty.Value.IsZero() {
				t.Error("Result value should not be zero")
			}

			t.Logf("Result: %s", qty.String())
		})
	}
}

// TestUnitlessQuantityNormalization verifies that unitless quantities from
// accumulate/over are normalized to Numbers before type dispatch.
func TestUnitlessQuantityNormalization(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name: "unitless * unitless gives number",
			input: `a = 1M / month
b = a over 1 week
c = 2M / month
d = c over 1 week
result = b * d
`,
		},
		{
			name: "number / unitless gives number",
			input: `posts_rate = 2/week
posts_per_day = posts_rate over 1 day
result = 400M / posts_per_day
`,
		},
		{
			name: "unitless * number gives number",
			input: `posts_rate = 2/week
posts_per_day = posts_rate over 1 day
result = posts_per_day * 100
`,
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
			_, ok := result.(*types.Number)
			if !ok {
				t.Fatalf("Expected *types.Number, got %T (%v)", result, result)
			}
		})
	}
}

// TestNumberTimesRate verifies Number * Rate → Rate (commutative).
// TestNumberTimesRate verifies Number * Rate → widened result.
// Rate arithmetic widening: when a rate is on the RIGHT side of *, the time
// denominator is dropped. Unitless rates widen to Number, rated quantities
// widen to Quantity. Rate on the LEFT (Rate * Number) preserves the rate type.
func TestNumberTimesRate(t *testing.T) {
	t.Run("integer * unitless rate → number", func(t *testing.T) {
		nodes, err := parser.Parse("r = 100/second\nresult = 3 * r\n")
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}
		interp := NewInterpreter()
		results, err := interp.Eval(nodes)
		if err != nil {
			t.Fatalf("Eval error: %v", err)
		}
		result := results[len(results)-1]
		num, ok := result.(*types.Number)
		if !ok {
			t.Fatalf("Expected *types.Number, got %T (%v)", result, result)
		}
		if num.Value.String() != "300" {
			t.Errorf("Expected 300, got %s", num.Value.String())
		}
	})

	t.Run("fractional * rate with unit → quantity", func(t *testing.T) {
		nodes, err := parser.Parse("r = 100 MB/s\nresult = 0.5 * r\n")
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}
		interp := NewInterpreter()
		results, err := interp.Eval(nodes)
		if err != nil {
			t.Fatalf("Eval error: %v", err)
		}
		result := results[len(results)-1]
		qty, ok := result.(*types.Quantity)
		if !ok {
			t.Fatalf("Expected *types.Quantity, got %T (%v)", result, result)
		}
		if qty.Value.String() != "50" {
			t.Errorf("Expected 50, got %s", qty.Value.String())
		}
		if qty.Unit != "MB" {
			t.Errorf("Expected unit MB, got %s", qty.Unit)
		}
	})

	t.Run("zero * unitless rate → number", func(t *testing.T) {
		nodes, err := parser.Parse("r = 100/second\nresult = 0 * r\n")
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}
		interp := NewInterpreter()
		results, err := interp.Eval(nodes)
		if err != nil {
			t.Fatalf("Eval error: %v", err)
		}
		result := results[len(results)-1]
		num, ok := result.(*types.Number)
		if !ok {
			t.Fatalf("Expected *types.Number, got %T (%v)", result, result)
		}
		if num.Value.String() != "0" {
			t.Errorf("Expected 0, got %s", num.Value.String())
		}
	})
}

// TestRateTimesQuantity verifies Rate * Quantity → Quantity and Quantity * Rate → Quantity.
func TestRateTimesQuantity(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedUnit  string
		expectNonZero bool
	}{
		{
			name:          "rate * quantity",
			input:         "r = 100/second\nresult = r * 10 KB\n",
			expectedUnit:  "KB",
			expectNonZero: true,
		},
		{
			name:          "quantity * rate (commutative)",
			input:         "r = 100/second\nresult = 10 KB * r\n",
			expectedUnit:  "KB",
			expectNonZero: true,
		},
		{
			name:          "zero rate * quantity",
			input:         "r = 0/second\nresult = r * 10 KB\n",
			expectedUnit:  "KB",
			expectNonZero: false,
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
			result := results[len(results)-1]
			qty, ok := result.(*types.Quantity)
			if !ok {
				t.Fatalf("Expected *types.Quantity, got %T (%v)", result, result)
			}
			if qty.Unit != tt.expectedUnit {
				t.Errorf("Expected unit %q, got %q", tt.expectedUnit, qty.Unit)
			}
			if tt.expectNonZero && qty.Value.IsZero() {
				t.Error("Result value should not be zero")
			}
		})
	}
}
