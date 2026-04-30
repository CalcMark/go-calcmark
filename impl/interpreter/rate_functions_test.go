package interpreter

import (
	"testing"

	"github.com/CalcMark/go-calcmark/v2/spec/types"
	"github.com/shopspring/decimal"
)

func TestAccumulateRate(t *testing.T) {
	t.Run("non-currency rates return Quantity", func(t *testing.T) {
		tests := []struct {
			name          string
			rate          *types.Rate
			timePeriod    decimal.Decimal
			periodUnit    string
			expectedValue string
			expectedUnit  string
		}{
			{
				name: "100 MB/s over 1 day",
				rate: types.NewRate(
					&types.Quantity{Value: decimal.NewFromInt(100), Unit: "MB"},
					"second",
				),
				timePeriod:    decimal.NewFromInt(1),
				periodUnit:    "day",
				expectedValue: "8640000", // 100 * 86400
				expectedUnit:  "MB",
			},
			{
				name: "5 GB/day over 1 year",
				rate: types.NewRate(
					&types.Quantity{Value: decimal.NewFromInt(5), Unit: "GB"},
					"day",
				),
				timePeriod:    decimal.NewFromInt(1),
				periodUnit:    "year",
				expectedValue: "1825", // 5 * 365
				expectedUnit:  "GB",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result, err := accumulateRate(tt.rate, tt.timePeriod, tt.periodUnit)
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}

				qty, ok := result.(*types.Quantity)
				if !ok {
					t.Fatalf("Expected *types.Quantity, got %T", result)
				}

				if qty.Unit != tt.expectedUnit {
					t.Errorf("Expected unit %q, got %q", tt.expectedUnit, qty.Unit)
				}
				if qty.Value.String() != tt.expectedValue {
					t.Errorf("Expected value %s, got %s", tt.expectedValue, qty.Value.String())
				}
				t.Logf("✓ %s = %s %s", tt.name, qty.Value.String(), qty.Unit)
			})
		}
	})

	t.Run("currency rates return Currency", func(t *testing.T) {
		tests := []struct {
			name           string
			rate           *types.Rate
			timePeriod     decimal.Decimal
			periodUnit     string
			expectedValue  string
			expectedSymbol string
			expectedCode   string
		}{
			{
				name: "$0.10/hour over 30 days",
				rate: types.NewRate(
					&types.Quantity{Value: decimal.NewFromFloat(0.10), Unit: "$"},
					"hour",
				),
				timePeriod:     decimal.NewFromInt(30),
				periodUnit:     "day",
				expectedValue:  "72",
				expectedSymbol: "$",
				expectedCode:   "USD",
			},
			{
				name: "€50/day over 1 year",
				rate: types.NewRate(
					&types.Quantity{Value: decimal.NewFromInt(50), Unit: "€"},
					"day",
				),
				timePeriod:     decimal.NewFromInt(1),
				periodUnit:     "year",
				expectedValue:  "18250",
				expectedSymbol: "€",
				expectedCode:   "EUR",
			},
			{
				name: "USD100/month over 3 years",
				rate: types.NewRate(
					&types.Quantity{Value: decimal.NewFromInt(100), Unit: "USD"},
					"month",
				),
				timePeriod:     decimal.NewFromInt(3),
				periodUnit:     "year",
				expectedValue:  "3650", // 3 years = 36.5 months (365*3/30)
				expectedSymbol: "USD",
				expectedCode:   "USD",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result, err := accumulateRate(tt.rate, tt.timePeriod, tt.periodUnit)
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}

				cur, ok := result.(*types.Currency)
				if !ok {
					t.Fatalf("Expected *types.Currency, got %T (%s)", result, result.String())
				}

				if cur.Symbol != tt.expectedSymbol {
					t.Errorf("Expected symbol %q, got %q", tt.expectedSymbol, cur.Symbol)
				}
				if cur.Code != tt.expectedCode {
					t.Errorf("Expected code %q, got %q", tt.expectedCode, cur.Code)
				}
				if cur.Value.String() != tt.expectedValue {
					t.Errorf("Expected value %s, got %s", tt.expectedValue, cur.Value.String())
				}
				t.Logf("✓ %s = %s", tt.name, cur.String())
			})
		}
	})

	t.Run("nil rate returns error", func(t *testing.T) {
		_, err := accumulateRate(nil, decimal.NewFromInt(1), "day")
		if err == nil {
			t.Error("Expected error for nil rate but got none")
		}
	})
}

func TestConvertRateTimeUnit(t *testing.T) {
	tests := []struct {
		name          string
		rate          *types.Rate
		targetUnit    string
		expectedValue decimal.Decimal
		expectedUnit  string
		expectError   bool
	}{
		{
			name: "5 million/day to per second",
			rate: types.NewRate(
				&types.Quantity{Value: decimal.NewFromInt(5000000), Unit: ""},
				"day",
			),
			targetUnit:    "second",
			expectedValue: decimal.NewFromFloat(57.87), // ~57.87
			expectedUnit:  "",
			expectError:   false,
		},
		{
			name: "1000 req/s to per hour",
			rate: types.NewRate(
				&types.Quantity{Value: decimal.NewFromInt(1000), Unit: "req"},
				"second",
			),
			targetUnit:    "hour",
			expectedValue: decimal.NewFromInt(3600000), // 1000 * 3600
			expectedUnit:  "req",
			expectError:   false,
		},
		{
			name: "same unit no change",
			rate: types.NewRate(
				&types.Quantity{Value: decimal.NewFromInt(100), Unit: "MB"},
				"second",
			),
			targetUnit:    "second",
			expectedValue: decimal.NewFromInt(100),
			expectedUnit:  "MB",
			expectError:   false,
		},
		{
			// User scenario: 10 MB/s converted to per hour should be exactly 36000 MB/hour
			// This tests the precision fix: using Mul(targetSeconds/sourceSeconds)
			// instead of Div(sourceSeconds/targetSeconds) to avoid division precision loss
			name: "10 MB/s to per hour (exact precision)",
			rate: types.NewRate(
				&types.Quantity{Value: decimal.NewFromInt(10), Unit: "MB"},
				"second",
			),
			targetUnit:    "hour",
			expectedValue: decimal.NewFromInt(36000), // 10 * 3600 = 36000 exactly
			expectedUnit:  "MB",
			expectError:   false,
		},
		{
			// Test reverse: hour to second should also be exact
			name: "3600 req/h to per second (exact precision)",
			rate: types.NewRate(
				&types.Quantity{Value: decimal.NewFromInt(3600), Unit: "req"},
				"hour",
			),
			targetUnit:    "second",
			expectedValue: decimal.NewFromInt(1), // 3600 / 3600 = 1 exactly
			expectedUnit:  "req",
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := convertRateTimeUnit(tt.rate, tt.targetUnit)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if result.Amount.Unit != tt.expectedUnit {
				t.Errorf("Expected unit %q, got %q", tt.expectedUnit, result.Amount.Unit)
			}

			// Check if value is within tolerance (0.01%)
			tolerance := decimal.NewFromFloat(0.0001)
			diff := result.Amount.Value.Sub(tt.expectedValue).Abs()
			maxDiff := tt.expectedValue.Abs().Mul(tolerance)

			if diff.GreaterThan(maxDiff) {
				t.Errorf("Expected value ~%s, got %s (diff: %s, max allowed: %s)",
					tt.expectedValue.String(), result.Amount.Value.String(),
					diff.String(), maxDiff.String())
			}

			if result.PerUnit != tt.targetUnit {
				t.Errorf("Expected time unit %q, got %q", tt.targetUnit, result.PerUnit)
			}

			t.Logf("✓ %s = %s", tt.name, result.String())
		})
	}
}

// TestConvertRateSubSecondUnits verifies that rate conversions work correctly
// with sub-second time units (millisecond, microsecond, nanosecond).
// Each expected value is hand-calculated from the conversion factor:
//
//	conversionFactor = targetSeconds / sourceSeconds
//	newAmount = amount * conversionFactor
func TestConvertRateSubSecondUnits(t *testing.T) {
	t.Run("second to sub-second (scaling down)", func(t *testing.T) {
		tests := []struct {
			name          string
			rate          *types.Rate
			targetUnit    string
			expectedExact string
			expectedPer   string
		}{
			{
				// 1000 req/s → per ms: factor = 0.001/1 = 0.001, 1000 * 0.001 = 1
				name: "1000 req/s per millisecond",
				rate: types.NewRate(
					&types.Quantity{Value: decimal.NewFromInt(1000), Unit: "req"},
					"second",
				),
				targetUnit:    "millisecond",
				expectedExact: "1",
				expectedPer:   "millisecond",
			},
			{
				// 1M req/s → per μs: factor = 0.000001/1 = 0.000001, 1000000 * 0.000001 = 1
				name: "1M req/s per microsecond",
				rate: types.NewRate(
					&types.Quantity{Value: decimal.NewFromInt(1000000), Unit: "req"},
					"second",
				),
				targetUnit:    "microsecond",
				expectedExact: "1",
				expectedPer:   "microsecond",
			},
			{
				// 1B ops/s → per ns: factor = 0.000000001/1 = 1e-9, 1e9 * 1e-9 = 1
				name: "1B ops/s per nanosecond",
				rate: types.NewRate(
					&types.Quantity{Value: decimal.NewFromInt(1000000000), Unit: "ops"},
					"second",
				),
				targetUnit:    "nanosecond",
				expectedExact: "1",
				expectedPer:   "nanosecond",
			},
			{
				// 10 req/s → per ms: factor = 0.001, 10 * 0.001 = 0.01
				name: "10 req/s per millisecond",
				rate: types.NewRate(
					&types.Quantity{Value: decimal.NewFromInt(10), Unit: "req"},
					"second",
				),
				targetUnit:    "millisecond",
				expectedExact: "0.01",
				expectedPer:   "millisecond",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result, err := convertRateTimeUnit(tt.rate, tt.targetUnit)
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}
				if result.Amount.Value.String() != tt.expectedExact {
					t.Errorf("Expected exactly %s, got %s", tt.expectedExact, result.Amount.Value.String())
				}
				if result.PerUnit != tt.expectedPer {
					t.Errorf("Expected per unit %q, got %q", tt.expectedPer, result.PerUnit)
				}
			})
		}
	})

	t.Run("sub-second to second (scaling up)", func(t *testing.T) {
		tests := []struct {
			name          string
			rate          *types.Rate
			targetUnit    string
			expectedExact string
		}{
			{
				// 1 req/ms → per second: factor = 1/0.001 = 1000, 1 * 1000 = 1000
				name: "1 req/ms per second",
				rate: types.NewRate(
					&types.Quantity{Value: decimal.NewFromInt(1), Unit: "req"},
					"millisecond",
				),
				targetUnit:    "second",
				expectedExact: "1000",
			},
			{
				// 1 req/μs → per second: factor = 1/0.000001 = 1000000
				name: "1 req/μs per second",
				rate: types.NewRate(
					&types.Quantity{Value: decimal.NewFromInt(1), Unit: "req"},
					"microsecond",
				),
				targetUnit:    "second",
				expectedExact: "1000000",
			},
			{
				// 1 op/ns → per second: factor = 1/0.000000001 = 1000000000
				name: "1 op/ns per second",
				rate: types.NewRate(
					&types.Quantity{Value: decimal.NewFromInt(1), Unit: "op"},
					"nanosecond",
				),
				targetUnit:    "second",
				expectedExact: "1000000000",
			},
			{
				// 5 req/ms → per minute: factor = 60/0.001 = 60000, 5 * 60000 = 300000
				name: "5 req/ms per minute",
				rate: types.NewRate(
					&types.Quantity{Value: decimal.NewFromInt(5), Unit: "req"},
					"millisecond",
				),
				targetUnit:    "minute",
				expectedExact: "300000",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result, err := convertRateTimeUnit(tt.rate, tt.targetUnit)
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}
				if result.Amount.Value.String() != tt.expectedExact {
					t.Errorf("Expected exactly %s, got %s", tt.expectedExact, result.Amount.Value.String())
				}
				if result.PerUnit != tt.targetUnit {
					t.Errorf("Expected per unit %q, got %q", tt.targetUnit, result.PerUnit)
				}
			})
		}
	})

	t.Run("sub-second to sub-second", func(t *testing.T) {
		tests := []struct {
			name          string
			rate          *types.Rate
			targetUnit    string
			expectedExact string
		}{
			{
				// 1 req/ms → per μs: factor = 0.000001/0.001 = 0.001, 1 * 0.001 = 0.001
				name: "1 req/ms per microsecond",
				rate: types.NewRate(
					&types.Quantity{Value: decimal.NewFromInt(1), Unit: "req"},
					"millisecond",
				),
				targetUnit:    "microsecond",
				expectedExact: "0.001",
			},
			{
				// 1000 req/μs → per ms: factor = 0.001/0.000001 = 1000, 1000 * 1000 = 1000000
				name: "1000 req/μs per millisecond",
				rate: types.NewRate(
					&types.Quantity{Value: decimal.NewFromInt(1000), Unit: "req"},
					"microsecond",
				),
				targetUnit:    "millisecond",
				expectedExact: "1000000",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result, err := convertRateTimeUnit(tt.rate, tt.targetUnit)
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}
				if result.Amount.Value.String() != tt.expectedExact {
					t.Errorf("Expected exactly %s, got %s", tt.expectedExact, result.Amount.Value.String())
				}
			})
		}
	})

	t.Run("alias forms normalize correctly", func(t *testing.T) {
		// "ms" should normalize to "millisecond", "μs" to "microsecond", "ns" to "nanosecond"
		rate := types.NewRate(
			&types.Quantity{Value: decimal.NewFromInt(1000), Unit: "req"},
			"s", // alias for "second"
		)

		for _, alias := range []string{"ms", "μs", "us", "ns"} {
			t.Run(alias, func(t *testing.T) {
				result, err := convertRateTimeUnit(rate, alias)
				if err != nil {
					t.Fatalf("convertRateTimeUnit with alias %q failed: %v", alias, err)
				}
				if result == nil {
					t.Fatalf("Expected result for alias %q, got nil", alias)
				}
			})
		}
	})
}

// TestConvertRateTimeUnitExactPrecision verifies that rate time unit conversions
// produce exact results for the common user scenario: converting rates from
// smaller to larger time units (e.g., per second to per hour).
//
// This is critical for user-facing calculations where 10 MB/s per hour should
// display as exactly 36000 MB/hour, not 35999.99... MB/hour.
//
// Note: Reverse conversions (large to small, like per hour to per second) may
// have tiny precision loss due to repeating decimals (e.g., 1/3600), but this
// is acceptable as it only affects edge cases and the error is < 1e-11.
func TestConvertRateTimeUnitExactPrecision(t *testing.T) {
	t.Run("small to large time unit conversions must be exact", func(t *testing.T) {
		tests := []struct {
			name          string
			rate          *types.Rate
			targetUnit    string
			expectedExact string
		}{
			{
				// User scenario from bug report: 10 MB/s to per hour
				name: "10 MB/s to per hour",
				rate: types.NewRate(
					&types.Quantity{Value: decimal.NewFromInt(10), Unit: "MB"},
					"second",
				),
				targetUnit:    "hour",
				expectedExact: "36000", // 10 * 3600 = 36000
			},
			{
				// Larger scale: 1000/s to per hour
				name: "1000/s to per hour",
				rate: types.NewRate(
					&types.Quantity{Value: decimal.NewFromInt(1000), Unit: ""},
					"second",
				),
				targetUnit:    "hour",
				expectedExact: "3600000", // 1000 * 3600 = 3600000
			},
			{
				// Second to day
				name: "1/s to per day",
				rate: types.NewRate(
					&types.Quantity{Value: decimal.NewFromInt(1), Unit: ""},
					"second",
				),
				targetUnit:    "day",
				expectedExact: "86400", // 1 * 86400 = 86400
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result, err := convertRateTimeUnit(tt.rate, tt.targetUnit)
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}

				actualStr := result.Amount.Value.String()
				if actualStr != tt.expectedExact {
					t.Errorf("Precision loss detected: expected exactly %q, got %q",
						tt.expectedExact, actualStr)
				}

				t.Logf("Exact: %s = %s", tt.name, result.String())
			})
		}
	})

	t.Run("large to small time unit conversions have acceptable precision", func(t *testing.T) {
		// Reverse conversions may have tiny precision loss due to repeating decimals
		// (e.g., 1/3600 = 0.000277...), but the error should be negligible.
		tests := []struct {
			name          string
			rate          *types.Rate
			targetUnit    string
			expectedValue decimal.Decimal
		}{
			{
				name: "3600 req/h to per second",
				rate: types.NewRate(
					&types.Quantity{Value: decimal.NewFromInt(3600), Unit: "req"},
					"hour",
				),
				targetUnit:    "second",
				expectedValue: decimal.NewFromInt(1),
			},
			{
				name: "86400/day to per second",
				rate: types.NewRate(
					&types.Quantity{Value: decimal.NewFromInt(86400), Unit: ""},
					"day",
				),
				targetUnit:    "second",
				expectedValue: decimal.NewFromInt(1),
			},
		}

		tolerance := decimal.NewFromFloat(1e-10) // Very small tolerance for precision loss

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result, err := convertRateTimeUnit(tt.rate, tt.targetUnit)
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}

				diff := result.Amount.Value.Sub(tt.expectedValue).Abs()
				if diff.GreaterThan(tolerance) {
					t.Errorf("Precision loss too large: expected ~%s, got %s (diff: %s)",
						tt.expectedValue.String(), result.Amount.Value.String(), diff.String())
				}

				t.Logf("Within tolerance: %s = %s (expected %s)",
					tt.name, result.String(), tt.expectedValue.String())
			})
		}
	})
}
