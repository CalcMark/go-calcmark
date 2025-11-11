package evaluator

import (
	"testing"

	"github.com/CalcMark/go-calcmark/impl/types"
)

// TestFunctionArgumentVariations tests various combinations of numbers, variables, and currencies
// in function arguments with different comma/spacing styles
func TestFunctionArgumentVariations(t *testing.T) {
	tests := []struct {
		name     string
		setup    string // Variables to set up
		input    string
		want     string
		wantType string
	}{
		// Basic comma variations
		{
			name:     "comma-space separated",
			input:    "avg(1, 2, 3)",
			want:     "2",
			wantType: "Number",
		},
		{
			name:     "comma-no-space separated",
			input:    "avg(1,2,3)",
			want:     "2",
			wantType: "Number",
		},
		{
			name:     "mixed comma spacing",
			input:    "avg(1, 2,3, 4)",
			want:     "2.5",
			wantType: "Number",
		},

		// Numbers with thousands separators
		{
			name:     "thousands in args",
			input:    "avg(1,000, 2,000, 3,000)",
			want:     "2000",
			wantType: "Number",
		},
		{
			name:     "thousands no space after comma",
			input:    "avg(1,000,2,000,3,000)",
			want:     "2000",
			wantType: "Number",
		},
		{
			name:     "millions in args",
			input:    "avg(1,000,000, 2,000,000)",
			want:     "1500000",
			wantType: "Number",
		},

		// Mixed numbers and variables
		{
			name:     "number and variable",
			setup:    "x = 10",
			input:    "avg(5, x, 15)",
			want:     "10",
			wantType: "Number",
		},
		{
			name:     "thousands and variable",
			setup:    "max_price = 5000",
			input:    "avg(1,000, max_price, 9,000)",
			want:     "5000",
			wantType: "Number",
		},
		{
			name:     "multiple variables",
			setup:    "a = 100\nb = 200\nc = 300",
			input:    "avg(a, b, c)",
			want:     "200",
			wantType: "Number",
		},
		{
			name:     "variables and literals mixed",
			setup:    "price = 2500",
			input:    "avg(1,000, price, 3.14, 500)",
			want:     "1000.785",
			wantType: "Number",
		},

		// Currency values (functions ignore currency units)
		{
			name:     "currency basic",
			input:    "avg($100, $200, $300)",
			want:     "$200.00",
			wantType: "Currency",
		},
		{
			name:     "currency with thousands",
			input:    "avg($1,000, $2,000, $3,000)",
			want:     "$2000.00",
			wantType: "Currency",
		},
		{
			name:     "currency no space",
			input:    "avg($100,$200,$300)",
			want:     "$200.00",
			wantType: "Currency",
		},
		{
			name:     "euro currency",
			input:    "avg(€100, €200)",
			want:     "€150.00",
			wantType: "Currency",
		},

		// Mixed currency and numbers (returns Number - no units)
		{
			name:     "currency and number",
			input:    "avg($100, 200, 300)",
			want:     "200",
			wantType: "Number",
		},
		{
			name:     "number and currency",
			input:    "avg(100, $200, 300)",
			want:     "200",
			wantType: "Number",
		},

		// Decimals
		{
			name:     "decimals basic",
			input:    "avg(1.5, 2.5, 3.5)",
			want:     "2.5",
			wantType: "Number",
		},
		{
			name:     "decimals with thousands",
			input:    "avg(1,234.56, 2,345.67, 3,456.78)",
			want:     "2345.67",
			wantType: "Number",
		},

		// Expressions as arguments
		{
			name:     "expressions in args",
			input:    "avg(1 + 2, 3 + 4, 5 + 6)",
			want:     "7",
			wantType: "Number",
		},
		{
			name:     "complex expressions",
			setup:    "x = 10",
			input:    "avg(x * 2, x * 3, x * 4)",
			want:     "30",
			wantType: "Number",
		},

		// Multi-token function format
		{
			name:     "average of basic",
			input:    "average of 1, 2, 3",
			want:     "2",
			wantType: "Number",
		},
		{
			name:     "average of no space",
			input:    "average of 1,2,3",
			want:     "2",
			wantType: "Number",
		},
		{
			name:     "average of thousands",
			input:    "average of 1,000, 2,000, 3,000",
			want:     "2000",
			wantType: "Number",
		},
		{
			name:     "average of mixed",
			setup:    "price = 2500",
			input:    "average of 1,000, price, 4,000",
			want:     "2500",
			wantType: "Number",
		},

		// Square root variations
		{
			name:     "sqrt basic",
			input:    "sqrt(16)",
			want:     "4",
			wantType: "Number",
		},
		{
			name:     "sqrt thousands",
			input:    "sqrt(10,000)",
			want:     "100",
			wantType: "Number",
		},
		{
			name:     "sqrt variable",
			setup:    "x = 25",
			input:    "sqrt(x)",
			want:     "5",
			wantType: "Number",
		},
		{
			name:     "sqrt currency",
			input:    "sqrt($16)",
			want:     "$4.00",
			wantType: "Currency",
		},
		{
			name:     "square root of basic",
			input:    "square root of 100",
			want:     "10",
			wantType: "Number",
		},
		{
			name:     "square root of thousands",
			input:    "square root of 1,000,000",
			want:     "1000",
			wantType: "Number",
		},

		// Edge cases
		{
			name:     "single arg",
			input:    "avg(42)",
			want:     "42",
			wantType: "Number",
		},
		{
			name:     "single arg thousands",
			input:    "avg(1,000)",
			want:     "1000",
			wantType: "Number",
		},
		{
			name:     "many args",
			input:    "avg(1, 2, 3, 4, 5, 6, 7, 8, 9, 10)",
			want:     "5.5",
			wantType: "Number",
		},
		{
			name:     "nested function",
			input:    "avg(sqrt(4), sqrt(9), sqrt(16))",
			want:     "3",
			wantType: "Number",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := NewContext()

			// Run setup if provided
			if tt.setup != "" {
				_, err := Evaluate(tt.setup, ctx)
				if err != nil {
					t.Fatalf("Setup failed: %v", err)
				}
			}

			results, err := Evaluate(tt.input, ctx)
			if err != nil {
				t.Fatalf("Evaluate(%q) error = %v, want nil", tt.input, err)
			}

			if len(results) != 1 {
				t.Fatalf("Evaluate(%q) returned %d results, want 1", tt.input, len(results))
			}

			result := results[0]
			if result.TypeName() != tt.wantType {
				t.Errorf("Evaluate(%q) type = %s, want %s", tt.input, result.TypeName(), tt.wantType)
			}

			if result.String() != tt.want {
				t.Errorf("Evaluate(%q) = %s, want %s", tt.input, result.String(), tt.want)
			}
		})
	}
}

// TestFunctionArgumentsIgnoreUnits tests that numerical functions ignore currency units
// and operate purely on the numeric values
func TestFunctionArgumentsIgnoreUnits(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantValue   string
		wantSymbol  string
		description string
	}{
		{
			name:        "avg preserves first currency symbol",
			input:       "avg($100, $200)",
			wantValue:   "150",
			wantSymbol:  "$",
			description: "Average of $100 and $200 is $150, units preserved",
		},
		{
			name:        "avg treats currency as numbers",
			input:       "avg($10, $20, $30)",
			wantValue:   "20",
			wantSymbol:  "$",
			description: "Function operates on numeric values 10, 20, 30",
		},
		{
			name:        "sqrt preserves currency",
			input:       "sqrt($100)",
			wantValue:   "10",
			wantSymbol:  "$",
			description: "Square root of 100 (numeric value) with $ preserved",
		},
		{
			name:        "avg with large currency amounts",
			input:       "avg($1,000, $2,000, $3,000)",
			wantValue:   "2000",
			wantSymbol:  "$",
			description: "Thousands separator correctly handled, $ preserved",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Log(tt.description)

			ctx := NewContext()
			results, err := Evaluate(tt.input, ctx)
			if err != nil {
				t.Fatalf("Evaluate(%q) error = %v, want nil", tt.input, err)
			}

			if len(results) != 1 {
				t.Fatalf("Evaluate(%q) returned %d results, want 1", tt.input, len(results))
			}

			currency, ok := results[0].(*types.Currency)
			if !ok {
				t.Fatalf("Evaluate(%q) returned %T, want *types.Currency", tt.input, results[0])
			}

			if currency.Symbol != tt.wantSymbol {
				t.Errorf("Evaluate(%q) symbol = %s, want %s", tt.input, currency.Symbol, tt.wantSymbol)
			}

			if currency.Value.String() != tt.wantValue {
				t.Errorf("Evaluate(%q) value = %s, want %s", tt.input, currency.Value.String(), tt.wantValue)
			}
		})
	}
}

// TestFunctionArgumentEdgeCases tests edge cases and error conditions
func TestFunctionArgumentEdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantError bool
		errorMsg  string
	}{
		{
			name:      "avg empty args",
			input:     "avg()",
			wantError: true,
			errorMsg:  "avg() requires at least one argument",
		},
		{
			name:      "sqrt too many args",
			input:     "sqrt(4, 9)",
			wantError: true,
			errorMsg:  "sqrt() requires exactly one argument",
		},
		{
			name:      "sqrt negative",
			input:     "sqrt(-4)",
			wantError: true,
			errorMsg:  "sqrt() of negative number is not supported",
		},
		{
			name:      "undefined variable in args",
			input:     "avg(1, undefined_var, 3)",
			wantError: true,
			errorMsg:  "Undefined variable 'undefined_var'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := NewContext()
			_, err := Evaluate(tt.input, ctx)

			if tt.wantError {
				if err == nil {
					t.Fatalf("Evaluate(%q) expected error, got nil", tt.input)
				}
				if evalErr, ok := err.(*EvaluationError); ok {
					if evalErr.Message != tt.errorMsg {
						t.Errorf("Evaluate(%q) error = %q, want %q", tt.input, evalErr.Message, tt.errorMsg)
					}
				} else {
					// Could also be a parse error for undefined variable
					if err.Error() != tt.errorMsg && !contains(err.Error(), tt.errorMsg) {
						t.Errorf("Evaluate(%q) error = %q, want to contain %q", tt.input, err.Error(), tt.errorMsg)
					}
				}
			} else {
				if err != nil {
					t.Fatalf("Evaluate(%q) unexpected error: %v", tt.input, err)
				}
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || findInString(s, substr)))
}

func findInString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
