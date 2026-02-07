package interpreter

import (
	"testing"

	"github.com/CalcMark/go-calcmark/spec/types"
	"github.com/shopspring/decimal"
)

func TestAccumulateRate(t *testing.T) {
	tests := []struct {
		name          string
		rate          *types.Rate
		timePeriod    decimal.Decimal
		periodUnit    string
		expectedValue string
		expectedUnit  string
		expectError   bool
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
			expectError:   false,
		},
		{
			name: "$0.10/hour over 30 days",
			rate: types.NewRate(
				&types.Quantity{Value: decimal.NewFromFloat(0.10), Unit: "USD"},
				"hour",
			),
			timePeriod:    decimal.NewFromInt(30),
			periodUnit:    "day",
			expectedValue: "72", // 0.10 * 24 * 30
			expectedUnit:  "USD",
			expectError:   false,
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
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := accumulateRate(tt.rate, tt.timePeriod, tt.periodUnit)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if result.Unit != tt.expectedUnit {
				t.Errorf("Expected unit %q, got %q", tt.expectedUnit, result.Unit)
			}

			if result.Value.String() != tt.expectedValue {
				t.Errorf("Expected value %s, got %s", tt.expectedValue, result.Value.String())
			}

			t.Logf("✓ %s = %s %s", tt.name, result.Value.String(), result.Unit)
		})
	}
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

// TestConvertRateTimeUnitExactPrecision verifies that rate time unit conversions
// produce exact results without floating-point precision loss.
// This is critical for user-facing calculations where 10 MB/s per hour should
// display as exactly 36000 MB/hour (or 35.16 GB/hour), not 35999.99... MB/hour.
func TestConvertRateTimeUnitExactPrecision(t *testing.T) {
	tests := []struct {
		name          string
		rate          *types.Rate
		targetUnit    string
		expectedExact string // Expected exact string representation
	}{
		{
			// User scenario from bug report: 10 MB/s to per hour
			name: "10 MB/s to per hour must be exactly 36000",
			rate: types.NewRate(
				&types.Quantity{Value: decimal.NewFromInt(10), Unit: "MB"},
				"second",
			),
			targetUnit:    "hour",
			expectedExact: "36000", // 10 * 3600 = 36000
		},
		{
			// Reverse conversion: hour to second
			name: "3600 req/h to per second must be exactly 1",
			rate: types.NewRate(
				&types.Quantity{Value: decimal.NewFromInt(3600), Unit: "req"},
				"hour",
			),
			targetUnit:    "second",
			expectedExact: "1", // 3600 / 3600 = 1
		},
		{
			// Larger scale: 1000/s to per hour
			name: "1000/s to per hour must be exactly 3600000",
			rate: types.NewRate(
				&types.Quantity{Value: decimal.NewFromInt(1000), Unit: ""},
				"second",
			),
			targetUnit:    "hour",
			expectedExact: "3600000", // 1000 * 3600 = 3600000
		},
		{
			// Day to second conversion
			name: "86400/day to per second must be exactly 1",
			rate: types.NewRate(
				&types.Quantity{Value: decimal.NewFromInt(86400), Unit: ""},
				"day",
			),
			targetUnit:    "second",
			expectedExact: "1", // 86400 / 86400 = 1
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

			t.Logf("Exact result: %s = %s", tt.name, result.String())
		})
	}
}
