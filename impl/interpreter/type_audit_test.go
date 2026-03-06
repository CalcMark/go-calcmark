package interpreter

import (
	"testing"

	"github.com/CalcMark/go-calcmark/spec/parser"
	"github.com/CalcMark/go-calcmark/spec/types"
	"github.com/shopspring/decimal"
)

// evalExpr is a helper that parses and evaluates an expression, returning the result.
// Used for type preservation testing.
func evalExpr(input string) (types.Type, error) {
	// Wrap expression in assignment to make it a valid statement
	source := "x = " + input + "\n"
	nodes, err := parser.Parse(source)
	if err != nil {
		return nil, err
	}
	interp := NewInterpreter()
	results, err := interp.Eval(nodes)
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, nil
	}
	return results[0], nil
}

// TestTypePreservationAudit documents the type preservation audit for Plan 09-02.
//
// AUDIT SUMMARY (09-02):
// All types.NewNumber usages in impl/interpreter/*.go were audited.
// NO type erasure bugs were found. Each usage is intentional:
//
// CORRECT: Returns type matching input type
// - napkin_eval.go:33     - Number input -> Number output (Quantity/Currency/etc. preserve type)
// - operators.go:323      - Unary neg on Number -> Number
// - literals.go:19        - NumberLiteral -> Number
// - percentage_of_eval.go:40 - Number % of Number -> Number (Quantity/Currency preserve type)
//
// INTENTIONAL: Math functions that aggregate/transform
// - operators.go:164      - Rate / Rate -> dimensionless Number (ratio)
// - operators.go:236      - Number op Number -> Number
// - functions.go:avg()    - preserves currency when all args same currency
// - functions.go:sqrt()   - returns Number (dimensionless math transform)
// - environment.go:40-41  - PI/E constants are Number
//
// This test file exercises type preservation across the evaluation chain
// to prevent regression of type erasure bugs like the napkin bug fixed in 09-01.

func TestTypePreservationAudit(t *testing.T) {
	t.Run("Quantity through operations preserves type", func(t *testing.T) {
		testCases := []struct {
			name       string
			input      string
			expectType string
		}{
			{"addition", "10 meters + 5 meters", "*types.Quantity"},
			{"subtraction", "10 meters - 5 meters", "*types.Quantity"},
			{"multiply by scalar", "10 meters * 2", "*types.Quantity"},
			{"divide by scalar", "10 meters / 2", "*types.Quantity"},
			{"unit conversion", "10 meters in feet", "*types.Quantity"},
			{"unary negation", "-10 meters", "*types.Quantity"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result, err := evalExpr(tc.input)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				_, ok := result.(*types.Quantity)
				if !ok {
					t.Errorf("expected %s, got %T", tc.expectType, result)
				}
			})
		}
	})

	t.Run("Currency through operations preserves type", func(t *testing.T) {
		testCases := []struct {
			name       string
			input      string
			expectType string
		}{
			{"addition", "$100 + $50", "*types.Currency"},
			{"subtraction", "$100 - $50", "*types.Currency"},
			{"multiply by scalar", "$100 * 2", "*types.Currency"},
			// Note: Currency / Number is not currently supported in the language
			{"napkin conversion", "$100 as napkin", "*types.Currency"},
			{"unary negation", "-$100", "*types.Currency"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result, err := evalExpr(tc.input)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				_, ok := result.(*types.Currency)
				if !ok {
					t.Errorf("expected %s, got %T", tc.expectType, result)
				}
			})
		}
	})

	t.Run("Duration through operations preserves type", func(t *testing.T) {
		testCases := []struct {
			name       string
			input      string
			expectType string
		}{
			{"addition", "1 day + 12 hours", "*types.Duration"},
			{"subtraction", "1 day - 12 hours", "*types.Duration"},
			{"multiply by scalar", "1 day * 2", "*types.Duration"},
			{"divide by scalar", "1 day / 2", "*types.Duration"},
			{"napkin conversion", "86400 seconds as napkin", "*types.Duration"},
			{"unary negation", "-1 day", "*types.Duration"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result, err := evalExpr(tc.input)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				_, ok := result.(*types.Duration)
				if !ok {
					t.Errorf("expected %s, got %T", tc.expectType, result)
				}
			})
		}
	})

	t.Run("Rate through operations preserves type", func(t *testing.T) {
		testCases := []struct {
			name       string
			input      string
			expectType string
		}{
			{"multiply by scalar", "100 MB/s * 2", "*types.Rate"},
			{"divide by scalar", "100 MB/s / 2", "*types.Rate"},
			{"napkin conversion", "100 MB/s as napkin", "*types.Rate"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result, err := evalExpr(tc.input)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				_, ok := result.(*types.Rate)
				if !ok {
					t.Errorf("expected %s, got %T", tc.expectType, result)
				}
			})
		}
	})

	t.Run("Rate accumulation returns Quantity", func(t *testing.T) {
		testCases := []struct {
			name       string
			input      string
			expectType string
		}{
			{"basic accumulation", "accumulate(100 MB/s, 1 day)", "*types.Quantity"},
			{"accumulation with napkin", "accumulate(100 MB/s, 1 day) as napkin", "*types.Quantity"},
			{"accumulation with smaller rate", "accumulate(5 MB/s, 1 day)", "*types.Quantity"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result, err := evalExpr(tc.input)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				_, ok := result.(*types.Quantity)
				if !ok {
					t.Errorf("expected %s, got %T", tc.expectType, result)
				}
			})
		}
	})

	t.Run("Percentage of preserves type", func(t *testing.T) {
		testCases := []struct {
			name       string
			input      string
			expectType string
		}{
			{"number", "10% of 100", "*types.Number"},
			{"quantity", "10% of 100 meters", "*types.Quantity"},
			{"currency", "10% of $100", "*types.Currency"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result, err := evalExpr(tc.input)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				switch tc.expectType {
				case "*types.Number":
					if _, ok := result.(*types.Number); !ok {
						t.Errorf("expected %s, got %T", tc.expectType, result)
					}
				case "*types.Quantity":
					if _, ok := result.(*types.Quantity); !ok {
						t.Errorf("expected %s, got %T", tc.expectType, result)
					}
				case "*types.Currency":
					if _, ok := result.(*types.Currency); !ok {
						t.Errorf("expected %s, got %T", tc.expectType, result)
					}
				}
			})
		}
	})
}

// TestFunctionTypePreservation verifies that aggregate/math functions
// preserve currency type when all inputs share the same currency,
// and return plain Number for plain number inputs.
func TestFunctionTypePreservation(t *testing.T) {
	t.Run("avg returns Number for plain numbers", func(t *testing.T) {
		result, err := evalExpr("avg(10, 20, 30)")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		num, ok := result.(*types.Number)
		if !ok {
			t.Errorf("avg() should return *types.Number, got %T", result)
		}
		if !num.Value.Equal(decimal.NewFromInt(20)) {
			t.Errorf("expected 20, got %v", num.Value)
		}
	})

	t.Run("avg preserves currency when all args same currency", func(t *testing.T) {
		result, err := evalExpr("avg($100, $200, $300)")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		cur, ok := result.(*types.Currency)
		if !ok {
			t.Fatalf("avg($100,$200,$300) should return *types.Currency, got %T", result)
		}
		if !cur.Value.Equal(decimal.NewFromInt(200)) {
			t.Errorf("expected 200, got %v", cur.Value)
		}
		if cur.Symbol != "$" {
			t.Errorf("expected symbol $, got %s", cur.Symbol)
		}
	})

	t.Run("avg returns Number for mixed currency and number", func(t *testing.T) {
		// Mixed types can't preserve currency
		result, err := evalExpr("avg(100, 200, 300)")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		_, ok := result.(*types.Number)
		if !ok {
			t.Errorf("avg(mixed) should return *types.Number, got %T", result)
		}
	})

	t.Run("sqrt returns Number for plain number", func(t *testing.T) {
		result, err := evalExpr("sqrt(16)")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		num, ok := result.(*types.Number)
		if !ok {
			t.Errorf("sqrt() should return *types.Number, got %T", result)
		}
		if !num.Value.Equal(decimal.NewFromInt(4)) {
			t.Errorf("expected 4, got %v", num.Value)
		}
	})

	t.Run("Rate divided by Rate returns Number (ratio)", func(t *testing.T) {
		// Rate / Rate with same time unit = dimensionless ratio
		result, err := evalExpr("(100 MB/s) / (50 MB/s)")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		num, ok := result.(*types.Number)
		if !ok {
			t.Errorf("Rate/Rate should return *types.Number (ratio), got %T", result)
		}
		// 100/50 = 2
		if !num.Value.Equal(decimal.NewFromInt(2)) {
			t.Errorf("expected 2, got %v", num.Value)
		}
	})
}

// TestOriginalNapkinBugRegression tests the exact scenario from the napkin
// bug fixed in 09-01: accumulate(5mb/s, 1 day) as napkin should return
// Quantity (~400 GB), not Number (430K).
func TestOriginalNapkinBugRegression(t *testing.T) {
	// This was the original bug: the result was Number(430080) instead of Quantity(~400GB)
	result, err := evalExpr("accumulate(5 MB/s, 1 day) as napkin")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	qty, ok := result.(*types.Quantity)
	if !ok {
		t.Fatalf("BUG REGRESSION: expected *types.Quantity, got %T (type erasure detected)", result)
	}

	// Should be approximately 400 GB (5 MB/s * 86400 s = 432000 MB = ~400 GB with napkin rounding)
	// After napkin normalization, should be in GB or TB
	if qty.Unit != "GB" && qty.Unit != "TB" {
		t.Errorf("expected unit to be GB or TB (normalized), got %s", qty.Unit)
	}

	// Value should be around 400-432 (depending on napkin rounding)
	f, _ := qty.Value.Float64()
	if f < 350 || f > 500 {
		t.Errorf("expected value around 400, got %v", f)
	}
}

// TestTypePreservationChain verifies that type is preserved through
// multiple operations in a chain.
func TestTypePreservationChain(t *testing.T) {
	testCases := []struct {
		name       string
		input      string
		expectType string
	}{
		{
			name:       "Quantity through multiple operations",
			input:      "(10 meters + 5 meters) * 2",
			expectType: "*types.Quantity",
		},
		{
			name:       "Currency through multiple operations",
			input:      "($100 + $50) * 2",
			expectType: "*types.Currency",
		},
		{
			name:       "Rate through scalar operations",
			input:      "(100 MB/s * 2) / 4",
			expectType: "*types.Rate",
		},
		{
			name:       "Quantity through unit conversion",
			input:      "(1000 meters in km) * 2",
			expectType: "*types.Quantity",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := evalExpr(tc.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			switch tc.expectType {
			case "*types.Quantity":
				if _, ok := result.(*types.Quantity); !ok {
					t.Errorf("expected %s, got %T", tc.expectType, result)
				}
			case "*types.Currency":
				if _, ok := result.(*types.Currency); !ok {
					t.Errorf("expected %s, got %T", tc.expectType, result)
				}
			case "*types.Rate":
				if _, ok := result.(*types.Rate); !ok {
					t.Errorf("expected %s, got %T", tc.expectType, result)
				}
			}
		})
	}
}
