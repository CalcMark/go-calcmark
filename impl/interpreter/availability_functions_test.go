package interpreter

import (
	"testing"

	"github.com/CalcMark/go-calcmark/spec/types"
	"github.com/shopspring/decimal"
)

func TestCalculateDowntime(t *testing.T) {
	tests := []struct {
		name          string
		availability  decimal.Decimal
		timePeriod    types.Type
		expectedValue string
		expectedUnit  string
		expectError   bool
	}{
		{
			name:          "99.9% per month",
			availability:  decimal.NewFromFloat(0.999), // 99.9%
			timePeriod:    mustDuration("1", "month"),
			expectedValue: "43.2", // 30 days * 24 hr * 60 min * 0.001 = 43.2 min
			expectedUnit:  "minute",
		},
		{
			name:          "99.99% per year",
			availability:  decimal.NewFromFloat(0.9999), // 99.99%
			timePeriod:    mustDuration("1", "year"),
			expectedValue: "52.56", // 365 days * 24 hr * 60 min * 0.0001 = 52.56 min
			expectedUnit:  "minute",
		},
		{
			name:          "99.999% per day",
			availability:  decimal.NewFromFloat(0.99999), // 99.999%
			timePeriod:    mustDuration("1", "day"),
			expectedValue: "0.864", // 24 * 60 * 60 * 0.00001 = 0.864 sec
			expectedUnit:  "second",
		},
		{
			name:          "99% per week",
			availability:  decimal.NewFromFloat(0.99), // 99%
			timePeriod:    mustDuration("1", "week"),
			expectedValue: "1.68", // 7 days * 24 hr * 0.01 = 1.68 hr
			expectedUnit:  "hour",
		},
		{
			name:          "100% availability",
			availability:  decimal.NewFromInt(1), // 100%
			timePeriod:    mustDuration("1", "month"),
			expectedValue: "0", // No downtime
			expectedUnit:  "second",
		},
		{
			name:          "0% availability",
			availability:  decimal.Zero, // 0%
			timePeriod:    mustDuration("1", "day"),
			expectedValue: "24", // Full day down = 24 hours
			expectedUnit:  "hour",
		},
		{
			name:         "invalid: > 100%",
			availability: decimal.NewFromFloat(1.5), // 150%
			timePeriod:   mustDuration("1", "month"),
			expectError:  true,
		},
		{
			name:         "invalid: negative",
			availability: decimal.NewFromInt(-1),
			timePeriod:   mustDuration("1", "month"),
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			avail := types.NewNumber(tt.availability)

			result, err := calculateDowntime(avail, tt.timePeriod)

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
				t.Errorf("Expected unit %s, got %s", tt.expectedUnit, result.Unit)
			}

			if result.Value.String() != tt.expectedValue {
				t.Errorf("Expected value %s, got %s", tt.expectedValue, result.Value.String())
			}

			t.Logf("✓ %s = %s %s", tt.name, result.Value.String(), result.Unit)
		})
	}
}

func mustDuration(value, unit string) *types.Duration {
	d, err := types.NewDurationFromString(value, unit)
	if err != nil {
		panic(err)
	}
	return d
}
